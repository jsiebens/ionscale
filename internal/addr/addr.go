package addr

import (
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/jsiebens/ionscale/internal/util"
	"inet.af/netaddr"
	"math/big"
	"net"
	"tailscale.com/net/tsaddr"
)

var ipv4Range = tsaddr.CGNATRange().IPNet()

type Predicate func(netaddr.IP) (bool, error)

func SelectIP(predicate Predicate) (*netaddr.IP, *netaddr.IP, error) {
	ip4, err := selectIP(ipv4Range, predicate)
	if err != nil {
		return nil, nil, err
	}
	ip6 := tsaddr.Tailscale4To6(*ip4)
	return ip4, &ip6, err
}

func selectIP(c *net.IPNet, predicate Predicate) (*netaddr.IP, error) {
	count := cidr.AddressCount(c)
	var n = util.RandUint64(count)

	for {
		stdIP, err := cidr.HostBig(c, big.NewInt(int64(n)))
		if err != nil {
			return nil, err
		}
		ip, _ := netaddr.FromStdIP(stdIP)
		ok, err := validateIP(ip, predicate)
		if err != nil {
			return nil, err
		}
		if ok {
			return &ip, nil
		}
		n = (n + 1) % count
	}
}

func validateIP(ip netaddr.IP, p Predicate) (bool, error) {
	if tsaddr.IsTailscaleIP(ip) {
		if p != nil {
			return p(ip)
		} else {
			return true, nil
		}
	}
	return false, nil
}
