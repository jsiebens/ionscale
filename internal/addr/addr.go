package addr

import (
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/jsiebens/ionscale/internal/util"
	"math/big"
	"net"
	"net/netip"
	"tailscale.com/net/tsaddr"
)

var (
	ipv4Range *net.IPNet
	ipv4Count uint64
)

func init() {
	ipv4Range, ipv4Count = prepareIP4Range()
}

func prepareIP4Range() (*net.IPNet, uint64) {
	cgnatRange := tsaddr.CGNATRange()
	_, ipNet, err := net.ParseCIDR(cgnatRange.String())
	if err != nil {
		panic(err)
	}
	return ipNet, cidr.AddressCount(ipNet)
}

type Predicate func(netip.Addr) (bool, error)

func SelectIP(predicate Predicate) (*netip.Addr, *netip.Addr, error) {
	ip4, err := selectIP(predicate)
	if err != nil {
		return nil, nil, err
	}
	ip6 := tsaddr.Tailscale4To6(*ip4)
	return ip4, &ip6, err
}

func selectIP(predicate Predicate) (*netip.Addr, error) {
	var n = util.RandUint64(ipv4Count)

	for {
		stdIP, err := cidr.HostBig(ipv4Range, big.NewInt(int64(n)))
		if err != nil {
			return nil, err
		}

		ip, _ := netip.AddrFromSlice(stdIP)
		ok, err := validateIP(ip, predicate)
		if err != nil {
			return nil, err
		}
		if ok {
			return &ip, nil
		}
		n = (n + 1) % ipv4Count
	}
}

func validateIP(ip netip.Addr, p Predicate) (bool, error) {
	if tsaddr.IsTailscaleIP(ip) {
		if p != nil {
			return p(ip)
		} else {
			return true, nil
		}
	}
	return false, nil
}
