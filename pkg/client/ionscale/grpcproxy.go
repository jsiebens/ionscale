package ionscale

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"github.com/jsiebens/ionscale/internal/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Conn struct {
	*grpc.ClientConn
	cls io.Closer
}

func (c *Conn) Close() error {
	_ = c.ClientConn.Close()
	_ = c.cls.Close()
	return nil
}

func NewGrpcWebProxy(serverUrl url.URL, insecureSkipVerify bool) *grpcWebProxy {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
	}

	httpClient := &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}

	return &grpcWebProxy{
		serverUrl:  serverUrl,
		proxyMutex: &sync.Mutex{},
		httpClient: httpClient,
	}
}

type grpcWebProxy struct {
	serverUrl       url.URL
	proxyMutex      *sync.Mutex
	proxyListener   net.Listener
	proxyServer     *grpc.Server
	proxyUsersCount int
	httpClient      *http.Client
}

const (
	frameHeaderLength = 5
	endOfStreamFlag   = 128
)

type noopCodec struct{}

func (noopCodec) Marshal(v interface{}) ([]byte, error) {
	return v.([]byte), nil
}

func (noopCodec) Unmarshal(data []byte, v interface{}) error {
	pointer := v.(*[]byte)
	*pointer = data
	return nil
}

func (noopCodec) Name() string {
	return "proto"
}

func toFrame(msg []byte) []byte {
	frame := append([]byte{0, 0, 0, 0}, msg...)
	binary.BigEndian.PutUint32(frame, uint32(len(msg)))
	frame = append([]byte{0}, frame...)
	return frame
}

func (c *grpcWebProxy) Dial(opts ...grpc.DialOption) (*Conn, error) {
	addr, i, err := c.useGRPCProxy()
	if err != nil {
		return nil, err
	}

	dialer := func(ctx context.Context, address string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, addr.Network(), address)
	}

	opts = append(opts,
		grpc.WithBlock(),
		grpc.FailOnNonTempDialError(true),
		grpc.WithContextDialer(dialer),
		grpc.WithInsecure(), // we are handling TLS, so tell grpc not to
		grpc.WithKeepaliveParams(keepalive.ClientParameters{Time: 10 * time.Second}),
	)

	conn, err := grpc.DialContext(context.Background(), addr.String(), opts...)

	return &Conn{ClientConn: conn, cls: i}, err
}

func (c *grpcWebProxy) executeRequest(fullMethodName string, msg []byte, md metadata.MD) (*http.Response, error) {
	requestURL := fmt.Sprintf("%s%s", c.serverUrl.String(), fullMethodName)
	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewReader(toFrame(msg)))

	if err != nil {
		return nil, err
	}
	for k, v := range md {
		if strings.HasPrefix(k, ":") {
			continue
		}
		for i := range v {
			req.Header.Set(k, v[i])
		}
	}
	req.Header.Set("content-type", "application/grpc-web+proto")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s %s failed with status code %d", req.Method, req.URL, resp.StatusCode)
	}
	var code codes.Code
	if statusStr := resp.Header.Get("Grpc-Status"); statusStr != "" {
		statusInt, err := strconv.ParseUint(statusStr, 10, 32)
		if err != nil {
			code = codes.Unknown
		} else {
			code = codes.Code(statusInt)
		}
		if code != codes.OK {
			return nil, status.Error(code, resp.Header.Get("Grpc-Message"))
		}
	}
	return resp, nil
}

func (c *grpcWebProxy) startGRPCProxy() (*grpc.Server, net.Listener, error) {
	serverAddr := fmt.Sprintf("%s/ionscale-%s.sock", os.TempDir(), util.RandStringBytes(8))
	ln, err := net.Listen("unix", serverAddr)

	if err != nil {
		return nil, nil, err
	}
	proxySrv := grpc.NewServer(
		grpc.ForceServerCodec(&noopCodec{}),
		grpc.UnknownServiceHandler(func(srv interface{}, stream grpc.ServerStream) error {
			fullMethodName, ok := grpc.MethodFromServerStream(stream)
			if !ok {
				return fmt.Errorf("Unable to get method name from stream context.")
			}
			msg := make([]byte, 0)
			err := stream.RecvMsg(&msg)
			if err != nil {
				return err
			}

			md, _ := metadata.FromIncomingContext(stream.Context())

			resp, err := c.executeRequest(fullMethodName, msg, md)
			if err != nil {
				return err
			}

			go func() {
				<-stream.Context().Done()
				safeClose(resp.Body)
			}()
			defer safeClose(resp.Body)
			c.httpClient.CloseIdleConnections()

			for {
				header := make([]byte, frameHeaderLength)
				if _, err := io.ReadAtLeast(resp.Body, header, frameHeaderLength); err != nil {
					if err == io.EOF {
						err = io.ErrUnexpectedEOF
					}
					return err
				}

				if header[0] == endOfStreamFlag {
					return nil
				}
				length := int(binary.BigEndian.Uint32(header[1:frameHeaderLength]))
				data := make([]byte, length)

				if read, err := io.ReadAtLeast(resp.Body, data, length); err != nil {
					if err != io.EOF {
						return err
					} else if read < length {
						return io.ErrUnexpectedEOF
					} else {
						return nil
					}
				}

				if err := stream.SendMsg(data); err != nil {
					return err
				}

			}
		}))
	go func() {
		_ = proxySrv.Serve(ln)
	}()
	return proxySrv, ln, nil
}

func (c *grpcWebProxy) useGRPCProxy() (net.Addr, io.Closer, error) {
	c.proxyMutex.Lock()
	defer c.proxyMutex.Unlock()

	if c.proxyListener == nil {
		var err error
		c.proxyServer, c.proxyListener, err = c.startGRPCProxy()
		if err != nil {
			return nil, nil, err
		}
	}
	c.proxyUsersCount = c.proxyUsersCount + 1

	return c.proxyListener.Addr(), NewCloser(func() error {
		c.proxyMutex.Lock()
		defer c.proxyMutex.Unlock()
		c.proxyUsersCount = c.proxyUsersCount - 1
		if c.proxyUsersCount == 0 {
			c.proxyServer.Stop()
			c.proxyListener = nil
			c.proxyServer = nil
			return nil
		}
		return nil
	}), nil
}

type Closer interface {
	Close() error
}

type inlineCloser struct {
	close func() error
}

func (c *inlineCloser) Close() error {
	return c.close()
}

func NewCloser(close func() error) Closer {
	return &inlineCloser{close: close}
}

func safeClose(c Closer) {
	if c != nil {
		_ = c.Close()
	}
}
