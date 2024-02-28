package stunserver

import (
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"io"
	"net"
	"net/netip"
	"time"

	"tailscale.com/net/stun"
)

var (
	stunRequests = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "ionscale",
		Name:      "stun_requests",
	}, []string{"disposition"})

	stunAddrFamily = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "ionscale",
		Name:      "stun_addr_family",
	}, []string{"family"})

	stunReadError  = stunRequests.WithLabelValues("read_error")
	stunNotSTUN    = stunRequests.WithLabelValues("not_stun")
	stunWriteError = stunRequests.WithLabelValues("write_error")
	stunSuccess    = stunRequests.WithLabelValues("success")

	stunIPv4 = stunAddrFamily.WithLabelValues("ipv4")
	stunIPv6 = stunAddrFamily.WithLabelValues("ipv6")
)

type STUNServer struct {
	pc *net.UDPConn
}

func New(pc *net.UDPConn) *STUNServer {
	return &STUNServer{pc: pc}
}

func (s *STUNServer) Serve() error {
	if s.pc == nil {
		return nil
	}

	var buf [64 << 10]byte
	var (
		n   int
		ua  *net.UDPAddr
		err error
	)
	for {
		n, ua, err = s.pc.ReadFromUDP(buf[:])
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return nil
			}
			time.Sleep(time.Second)
			stunReadError.Inc()
			continue
		}
		pkt := buf[:n]
		if !stun.Is(pkt) {
			stunNotSTUN.Inc()
			continue
		}
		txid, err := stun.ParseBindingRequest(pkt)
		if err != nil {
			stunNotSTUN.Inc()
			continue
		}
		if ua.IP.To4() != nil {
			stunIPv4.Inc()
		} else {
			stunIPv6.Inc()
		}
		addr, _ := netip.AddrFromSlice(ua.IP)
		res := stun.Response(txid, netip.AddrPortFrom(addr, uint16(ua.Port)))
		_, err = s.pc.WriteTo(res, ua)
		if err != nil {
			stunWriteError.Inc()
		} else {
			stunSuccess.Inc()
		}
	}
}

func (s *STUNServer) Shutdown() error {
	if s.pc == nil {
		return nil
	}
	return s.pc.Close()
}
