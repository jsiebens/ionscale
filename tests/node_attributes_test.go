package tests

import (
	"github.com/jsiebens/ionscale/pkg/client/ionscale"
	"github.com/jsiebens/ionscale/pkg/defaults"
	"github.com/jsiebens/ionscale/tests/sc"
	"github.com/jsiebens/ionscale/tests/tsn"
	"github.com/stretchr/testify/require"
	"tailscale.com/tailcfg"
	"testing"
)

func TestNodeAttrs(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		tailnet := s.CreateTailnet()
		key := s.CreateAuthKey(tailnet.Id, true)

		nodeA := s.NewTailscaleNode()
		require.NoError(t, nodeA.Up(key))

		policy := defaults.DefaultACLPolicy()
		policy.NodeAttrs = []ionscale.ACLNodeAttrGrant{
			{
				Target: []string{"tag:test"},
				Attr:   []string{"ionscale:test"},
			},
		}

		s.SetACLPolicy(tailnet.Id, policy)

		require.NoError(t, nodeA.WaitFor(tsn.HasCapability("ionscale:test")))
	})
}

func TestNodeAttrs_IgnoreFunnelAttr(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		tailnet := s.CreateTailnet()
		key := s.CreateAuthKey(tailnet.Id, true)

		nodeA := s.NewTailscaleNode()
		require.NoError(t, nodeA.Up(key))

		policy := defaults.DefaultACLPolicy()
		policy.NodeAttrs = []ionscale.ACLNodeAttrGrant{
			{
				Target: []string{"tag:test"},
				Attr:   []string{"ionscale:test"},
			},
		}

		s.SetACLPolicy(tailnet.Id, policy)

		require.NoError(t, nodeA.WaitFor(tsn.HasCapability("ionscale:test")))
		require.NoError(t, nodeA.WaitFor(tsn.IsMissingCapability(tailcfg.NodeAttrFunnel)))
	})
}
