package tests

import (
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/jsiebens/ionscale/tests/sc"
	"github.com/jsiebens/ionscale/tests/tsn"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestSwitchAccounts(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		s.PushOIDCUser("123", "john@localtest.me", "john")
		s.PushOIDCUser("124", "jane@localtest.me", "jane")

		tailnet := s.CreateTailnet()
		s.SetIAMPolicy(tailnet.Id, &api.IAMPolicy{Filters: []string{"domain == localtest.me"}})

		node := s.NewTailscaleNode(sc.WithName("switch"))

		code, err := node.LoginWithOidc()
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, code)

		require.NoError(t, node.WaitFor(tsn.Connected()))
		require.NoError(t, node.Check(tsn.HasUser("john@localtest.me")))
		require.NoError(t, node.Check(tsn.HasName("switch")))

		code, err = node.LoginWithOidc()
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, code)

		require.NoError(t, node.WaitFor(tsn.Connected()))
		require.NoError(t, node.Check(tsn.HasUser("jane@localtest.me")))
		require.NoError(t, node.Check(tsn.HasName("switch-1")))

		machines := s.ListMachines(tailnet.Id)
		require.Equal(t, 2, len(machines))
		require.Equal(t, "switch", machines[0].Name)
		require.Equal(t, "switch-1", machines[1].Name)
	})
}
