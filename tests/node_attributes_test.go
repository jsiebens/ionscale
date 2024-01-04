package tests

import (
	"github.com/jsiebens/ionscale/pkg/defaults"
	ionscalev1 "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/jsiebens/ionscale/tests/sc"
	"tailscale.com/tailcfg"
	"testing"
)

func TestNodeAttrs(t *testing.T) {
	sc.Run(t, func(s sc.Scenario) {
		tailnet := s.CreateTailnet("nodeattrs")
		key := s.CreateAuthKey(tailnet.Id, true)

		nodeA := s.NewTailscaleNode("test-a")
		nodeA.Up(key)

		policy := defaults.DefaultACLPolicy()
		policy.Nodeattrs = []*ionscalev1.NodeAttr{
			{
				Target: []string{"tag:test"},
				Attr:   []string{"ionscale:test"},
			},
		}

		s.SetAclPolicy(tailnet.Id, policy)

		nodeA.WaitFor(sc.HasCapability("ionscale:test"))
	})
}

func TestNodeAttrs_IgnoreFunnelAttr(t *testing.T) {
	sc.Run(t, func(s sc.Scenario) {
		tailnet := s.CreateTailnet("nodeattrs")
		key := s.CreateAuthKey(tailnet.Id, true)

		nodeA := s.NewTailscaleNode("test-a")
		nodeA.Up(key)

		policy := defaults.DefaultACLPolicy()
		policy.Nodeattrs = []*ionscalev1.NodeAttr{
			{
				Target: []string{"tag:test"},
				Attr:   []string{"ionscale:test", string(tailcfg.NodeAttrFunnel)},
			},
		}

		s.SetAclPolicy(tailnet.Id, policy)

		nodeA.WaitFor(sc.HasCapability("ionscale:test"))
		nodeA.WaitFor(sc.IsMissingCapability(tailcfg.NodeAttrFunnel))
	})
}
