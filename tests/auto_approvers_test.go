package tests

import (
	"github.com/jsiebens/ionscale/pkg/client/ionscale"
	"github.com/jsiebens/ionscale/pkg/defaults"
	"github.com/jsiebens/ionscale/tests/sc"
	"github.com/jsiebens/ionscale/tests/tsn"
	"github.com/stretchr/testify/require"
	"net/netip"
	"testing"
)

func TestAdvertiseRoutesAutoApprovedOnNewNode(t *testing.T) {
	route1 := netip.MustParsePrefix("10.1.0.0/24")
	route2 := netip.MustParsePrefix("10.2.0.0/24")

	sc.Run(t, func(s *sc.Scenario) {
		aclPolicy := defaults.DefaultACLPolicy()
		aclPolicy.AutoApprovers = &ionscale.ACLAutoApprovers{
			Routes: map[string][]string{
				route1.String(): {"tag:test-route"},
			},
		}

		tailnet := s.CreateTailnet()
		s.SetACLPolicy(tailnet.Id, aclPolicy)

		testNode := s.NewTailscaleNode()
		require.NoError(t, testNode.Up(
			s.CreateAuthKey(tailnet.Id, true, "tag:test-route"),
			tsn.WithAdvertiseTags("tag:test-route"),
			tsn.WithAdvertiseRoutes([]string{
				route1.String(),
				route2.String()},
			),
		))

		require.NoError(t, testNode.WaitFor(tsn.HasTailnet(tailnet.Name)))

		mid, err := s.FindMachine(tailnet.Id, testNode.Hostname())
		require.NoError(t, err)

		machineRoutes := s.GetMachineRoutes(mid)
		require.NoError(t, err)

		require.Equal(t, []string{route1.String(), route2.String()}, machineRoutes.AdvertisedRoutes)
		require.Equal(t, []string{route1.String()}, machineRoutes.EnabledRoutes)

		require.NoError(t, testNode.Check(tsn.HasAllowedIP(route1)))
		require.NoError(t, testNode.Check(tsn.IsMissingAllowedIP(route2)))
	})
}

func TestAdvertiseRoutesAutoApprovedOnExistingNode(t *testing.T) {
	route1 := netip.MustParsePrefix("10.1.0.0/24")
	route2 := netip.MustParsePrefix("10.2.0.0/24")
	route3 := netip.MustParsePrefix("10.3.0.0/24")

	sc.Run(t, func(s *sc.Scenario) {
		aclPolicy := defaults.DefaultACLPolicy()
		aclPolicy.AutoApprovers = &ionscale.ACLAutoApprovers{
			Routes: map[string][]string{
				route1.String(): {"tag:test-route"},
				route3.String(): {"tag:test-route"},
			},
		}

		tailnet := s.CreateTailnet()
		s.SetACLPolicy(tailnet.Id, aclPolicy)

		testNode := s.NewTailscaleNode()
		require.NoError(t, testNode.Up(
			s.CreateAuthKey(tailnet.Id, true, "tag:test-route"),
			tsn.WithAdvertiseTags("tag:test-route"),
		))

		require.NoError(t, testNode.Check(tsn.HasTailnet(tailnet.Name)))

		testNode.Set(tsn.WithAdvertiseRoutes([]string{
			route3.String(),
			route1.String(),
			route2.String()},
		))

		require.NoError(t, testNode.WaitFor(tsn.HasAllowedIP(route1)))
		require.NoError(t, testNode.WaitFor(tsn.HasAllowedIP(route3)))
		require.NoError(t, testNode.WaitFor(tsn.IsMissingAllowedIP(route2)))

		mid, err := s.FindMachine(tailnet.Id, testNode.Hostname())
		require.NoError(t, err)

		machineRoutes := s.GetMachineRoutes(mid)
		require.NoError(t, err)

		require.Equal(t, []string{route1.String(), route2.String(), route3.String()}, machineRoutes.AdvertisedRoutes)
		require.Equal(t, []string{route1.String(), route3.String()}, machineRoutes.EnabledRoutes)
	})
}

func TestAdvertiseRemoveRoutesAutoApprovedOnExistingNode(t *testing.T) {
	route1 := netip.MustParsePrefix("10.1.0.0/24")
	route2 := netip.MustParsePrefix("10.2.0.0/24")

	sc.Run(t, func(s *sc.Scenario) {
		aclPolicy := defaults.DefaultACLPolicy()
		aclPolicy.AutoApprovers = &ionscale.ACLAutoApprovers{
			Routes: map[string][]string{
				route1.String(): {"tag:test-route"},
				route2.String(): {"tag:test-route"},
			},
		}

		tailnet := s.CreateTailnet()
		s.SetACLPolicy(tailnet.Id, aclPolicy)

		testNode := s.NewTailscaleNode()
		require.NoError(t, testNode.Up(
			s.CreateAuthKey(tailnet.Id, true, "tag:test-route"),
			tsn.WithAdvertiseTags("tag:test-route"),
			tsn.WithAdvertiseRoutes([]string{
				route1.String(),
				route2.String()},
			),
		))

		require.NoError(t, testNode.WaitFor(tsn.HasTailnet(tailnet.Name)))
		require.NoError(t, testNode.Check(tsn.HasAllowedIP(route1)))
		require.NoError(t, testNode.Check(tsn.HasAllowedIP(route2)))

		testNode.Set(tsn.WithAdvertiseRoutes([]string{
			route1.String(),
		}))

		require.NoError(t, testNode.WaitFor(tsn.IsMissingAllowedIP(route2)))
	})
}
