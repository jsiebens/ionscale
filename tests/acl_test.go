package tests

import (
	"github.com/jsiebens/ionscale/pkg/defaults"
	ionscalev1 "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/jsiebens/ionscale/tests/sc"
	"testing"
)

func TestACL_PeersShouldBeRemovedWhenNoMatchingACLRuleIsAvailable(t *testing.T) {
	sc.Run(t, func(s sc.Scenario) {
		tailnet := s.CreateTailnet("acltest")
		clientKey := s.CreateAuthKey(tailnet.Id, true, "tag:client")
		serverKey := s.CreateAuthKey(tailnet.Id, true, "tag:server")

		client1 := s.NewTailscaleNode("client1")
		client2 := s.NewTailscaleNode("client2")
		server := s.NewTailscaleNode("server")

		client1.Up(clientKey)
		client2.Up(clientKey)
		server.Up(serverKey)

		server.WaitFor(sc.PeerCount(2))

		policy := defaults.DefaultACLPolicy()
		policy.Acls = []*ionscalev1.ACL{
			{
				Action: "accept",
				Src:    []string{"tag:server"},
				Dst:    []string{"tag:server:*"},
			},
		}

		s.SetAclPolicy(tailnet.Id, policy)

		server.WaitFor(sc.PeerCount(0))
	})
}
