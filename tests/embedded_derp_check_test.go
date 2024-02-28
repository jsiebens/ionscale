package tests

import (
	"github.com/jsiebens/ionscale/tests/sc"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNetCheckWithEmbeddedDERP(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		tailnet := s.CreateTailnet()
		authKey := s.CreateAuthKey(tailnet.Id, false)

		tsNode := s.NewTailscaleNode()

		require.NoError(t, tsNode.Up(authKey))

		report, err := tsNode.NetCheck()
		require.NoError(t, err)
		require.Equal(t, 1000, report.PreferredDERP)
	})
}
