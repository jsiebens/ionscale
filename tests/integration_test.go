package tests

import (
	"github.com/jsiebens/ionscale/tests/sc"
	"github.com/jsiebens/ionscale/tests/tsn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetIPs(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		tailnet := s.CreateTailnet()
		authKey := s.CreateAuthKey(tailnet.Id, false)

		tsNode := s.NewTailscaleNode()

		require.NoError(t, tsNode.Up(authKey))

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

func TestPing(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		tailnet := s.CreateTailnet()
		key := s.CreateAuthKey(tailnet.Id, true)

		nodeA := s.NewTailscaleNode()
		nodeB := s.NewTailscaleNode()

		require.NoError(t, nodeA.Up(key))
		require.NoError(t, nodeB.Up(key))

		require.NoError(t, nodeA.WaitFor(tsn.PeerCount(1)))
		require.NoError(t, nodeA.Ping(nodeB.Hostname()))
		require.NoError(t, nodeA.Ping(nodeB.IPv4()))
		require.NoError(t, nodeA.Ping(nodeB.IPv6()))
	})
}

func TestNodeWithSameHostname(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		tailnet := s.CreateTailnet()
		authKey := s.CreateAuthKey(tailnet.Id, false)

		tsNode := s.NewTailscaleNode(sc.WithName("test"))

		require.NoError(t, tsNode.Up(authKey))

		for i := 0; i < 5; i++ {
			tc := s.NewTailscaleNode(sc.WithName("test"))
			require.NoError(t, tc.Up(authKey))
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

func TestNodeShouldSeeAssignedTags(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		tailnet := s.CreateTailnet()
		key := s.CreateAuthKey(tailnet.Id, true, "tag:server")

		nodeA := s.NewTailscaleNode()

		require.NoError(t, nodeA.Up(key, tsn.WithAdvertiseTags("tag:test")))
		require.NoError(t, nodeA.WaitFor(tsn.HasTag("tag:server")))
		require.NoError(t, nodeA.WaitFor(tsn.HasTag("tag:test")))
	})
}
