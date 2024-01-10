package tests

import (
	"github.com/jsiebens/ionscale/pkg/defaults"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/jsiebens/ionscale/tests/sc"
	"github.com/jsiebens/ionscale/tests/tsn"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"net/http"
	"tailscale.com/tailcfg"
	"testing"
)

func newTailscaleNodeAndLoginWithOIDC(t *testing.T, s *sc.Scenario, expectedLoginName string, flags ...tsn.UpFlag) *tsn.TailscaleNode {
	node := s.NewTailscaleNode()
	code, err := node.LoginWithOidc(flags...)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, code)

	require.NoError(t, node.WaitFor(tsn.Connected()))
	require.NoError(t, node.WaitFor(tsn.HasUser(expectedLoginName)))

	return node
}

func TestWebLoginWithDomainFilterInIAMPolicy(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		s.PushOIDCUser("123", "john@localtest.me", "john")
		s.PushOIDCUser("124", "jane@localtest.me", "jane")

		tailnet := s.CreateTailnet()
		s.SetIAMPolicy(tailnet.Id, &api.IAMPolicy{Filters: []string{"domain == localtest.me"}})

		john := newTailscaleNodeAndLoginWithOIDC(t, s, "john@localtest.me")
		jane := newTailscaleNodeAndLoginWithOIDC(t, s, "jane@localtest.me")

		require.NoError(t, john.Check(tsn.HasTailnet(tailnet.Name)))
		require.NoError(t, jane.Check(tsn.HasTailnet(tailnet.Name)))

		require.NoError(t, john.WaitFor(tsn.PeerCount(1)))
		require.NoError(t, jane.WaitFor(tsn.PeerCount(1)))
	})
}

func TestWebLoginWithSubsAndEmailsInIAMPolicy(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		s.PushOIDCUser("123", "john@localtest.me", "john")
		s.PushOIDCUser("124", "jane@localtest.me", "jane")

		tailnet := s.CreateTailnet()
		s.SetIAMPolicy(tailnet.Id, &api.IAMPolicy{Subs: []string{"123"}, Emails: []string{"jane@localtest.me"}})

		john := newTailscaleNodeAndLoginWithOIDC(t, s, "john@localtest.me")
		jane := newTailscaleNodeAndLoginWithOIDC(t, s, "jane@localtest.me")

		require.NoError(t, john.WaitFor(tsn.PeerCount(1)))
		require.NoError(t, jane.WaitFor(tsn.PeerCount(1)))
	})
}

func TestWebLoginWithUserAsTailnetAdmin(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		s.PushOIDCUser("123", "john@localtest.me", "john")
		s.PushOIDCUser("124", "jane@localtest.me", "jane")

		tailnet := s.CreateTailnet()
		s.SetIAMPolicy(tailnet.Id, &api.IAMPolicy{
			Filters: []string{"domain == localtest.me"},
			Roles:   map[string]string{"john@localtest.me": "admin"},
		})

		john := newTailscaleNodeAndLoginWithOIDC(t, s, "john@localtest.me")
		jane := newTailscaleNodeAndLoginWithOIDC(t, s, "jane@localtest.me")

		require.NoError(t, john.Check(tsn.HasCapability(tailcfg.CapabilityAdmin)))
		require.NoError(t, jane.Check(tsn.IsMissingCapability(tailcfg.CapabilityAdmin)))
	})
}

func TestWebLoginWhenNotAuthorizedForAnyTailnet(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		s.PushOIDCUser("124", "jane@localtest.me", "jane")

		tailnet := s.CreateTailnet()
		s.SetIAMPolicy(tailnet.Id, &api.IAMPolicy{
			Subs: []string{"123"},
		})

		jane := s.NewTailscaleNode()
		code, err := jane.LoginWithOidc()
		require.NoError(t, err)
		require.Equal(t, http.StatusForbidden, code)
	})
}

func TestWebLoginWhenInvalidTagOwner(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		s.PushOIDCUser("124", "jane@localtest.me", "jane")

		tailnet := s.CreateTailnet()
		s.SetIAMPolicy(tailnet.Id, &api.IAMPolicy{
			Subs: []string{"124"},
		})

		jane := s.NewTailscaleNode()
		code, err := jane.LoginWithOidc(tsn.WithAdvertiseTags("tag:test"))
		require.NoError(t, err)
		require.Equal(t, http.StatusForbidden, code)
	})
}

func TestWebLoginAsTagOwner(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		s.PushOIDCUser("124", "jane@localtest.me", "jane")

		owners, err := structpb.NewList([]interface{}{"jane@localtest.me"})
		require.NoError(t, err)

		aclPolicy := defaults.DefaultACLPolicy()
		aclPolicy.Tagowners = map[string]*structpb.ListValue{
			"tag:localtest": owners,
		}

		tailnet := s.CreateTailnet()
		s.SetACLPolicy(tailnet.Id, aclPolicy)
		s.SetIAMPolicy(tailnet.Id, &api.IAMPolicy{
			Subs: []string{"124"},
		})

		newTailscaleNodeAndLoginWithOIDC(t, s, "tagged-devices", tsn.WithAdvertiseTags("tag:localtest"))
	})
}

func TestWebLoginWithMachineAuthorizationRequired(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		s.PushOIDCUser("123", "john@localtest.me", "john")

		tailnet := s.CreateTailnet()
		s.SetIAMPolicy(tailnet.Id, &api.IAMPolicy{Filters: []string{"domain == localtest.me"}})
		s.EnableMachineAutorization(tailnet.Id)

		node := newTailscaleNodeAndLoginWithOIDC(t, s, "john@localtest.me")

		require.NoError(t, node.Check(tsn.NeedsMachineAuth()))

		s.AuthorizeMachines(tailnet.Id)

		require.NoError(t, node.WaitFor(tsn.IsRunning()))
	})
}
