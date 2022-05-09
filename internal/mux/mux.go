package mux

import (
	"crypto/tls"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/soheilhy/cmux"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"net"
	"net/http"
)

func Serve(grpcServer *grpc.Server, appHandler http.Handler, metricsHandler http.Handler, config *config.Config) error {
	appL, err := appListener(config)
	if err != nil {
		return err
	}

	metricsL, err := metricsListener(config)
	if err != nil {
		return err
	}

	mux := cmux.New(appL)
	grpcL := mux.MatchWithWriters(
		cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"),
		cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc+proto"),
	)
	httpL := mux.Match(cmux.Any())

	g := new(errgroup.Group)

	g.Go(func() error { return grpcServer.Serve(grpcL) })
	g.Go(func() error { return http.Serve(httpL, appHandler) })
	g.Go(func() error { return http.Serve(metricsL, metricsHandler) })
	g.Go(func() error { return mux.Serve() })

	return g.Wait()
}

func metricsListener(config *config.Config) (net.Listener, error) {
	return net.Listen("tcp", config.Metrics.ListenAddr)
}

func appListener(config *config.Config) (net.Listener, error) {
	if config.Tls.Disable {
		return net.Listen("tcp", config.ListenAddr)
	} else {
		cer, err := tls.LoadX509KeyPair(config.Tls.CertFile, config.Tls.KeyFile)
		if err != nil {
			return nil, err
		}

		tlsConfig := &tls.Config{Certificates: []tls.Certificate{cer}}

		return tls.Listen("tcp", config.ListenAddr, tlsConfig)
	}
}
