package tests

import (
	"github.com/jsiebens/ionscale/tests/sc"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPing(t *testing.T) {
	sc.Run(t, func(s sc.Scenario) {
		tailnet := s.CreateTailnet("pingtest")
		key := s.CreateAuthKey(tailnet.Id, true)

		nodeA := s.NewTailscaleNode("pingtest-a")
		nodeB := s.NewTailscaleNode("pingtest-b")

		nodeA.Up(key)
		nodeB.Up(key)

		nodeA.WaitFor(sc.PeerCount(1))
		nodeA.Ping("pingtest-b")
		nodeA.Ping(nodeB.IPv4())
		nodeA.Ping(nodeB.IPv6())
	})
}

func TestGetIPs(t *testing.T) {
	sc.Run(t, func(s sc.Scenario) {
		tailnet := s.CreateTailnet("tailnet01")
		authKey := s.CreateAuthKey(tailnet.Id, false)

		tsNode := s.NewTailscaleNode("testip")

		tsNode.Up(authKey)

		ip4 := tsNode.IPv4()
		ip6 := tsNode.IPv6()

		var found = false
		machines := s.ListMachines(tailnet.Id)
		for _, m := range machines {
			if m.Name == tsNode.Hostname() {
				found = true
				assert.Equal(t, m.Ipv4, ip4)
				assert.Equal(t, m.Ipv6, ip6)
			}
		}
		assert.True(t, found)
	})
}

func TestNodeWithSameHostname(t *testing.T) {
	sc.Run(t, func(s sc.Scenario) {
		tailnet := s.CreateTailnet("tailnet01")
		authKey := s.CreateAuthKey(tailnet.Id, false)

		tsNode := s.NewTailscaleNode("test")

		tsNode.Up(authKey)

		for i := 0; i < 5; i++ {
			tc := s.NewTailscaleNode("test")
			tc.Up(authKey)
		}

		machines := make(map[string]bool)
		for _, m := range s.ListMachines(tailnet.Id) {
			machines[m.Name] = true
		}

		assert.Equal(t, map[string]bool{
			"test":   true,
			"test-1": true,
			"test-2": true,
			"test-3": true,
			"test-4": true,
			"test-5": true,
		}, machines)
	})
}
