package tests

import (
	"github.com/jsiebens/ionscale/tests/sc"
	"github.com/jsiebens/ionscale/tests/tsn"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExpiredPeersShouldBeListed(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		tailnet := s.CreateTailnet()
		key := s.CreateAuthKey(tailnet.Id, true)

		nodeA := s.NewTailscaleNode()

		require.NoError(t, nodeA.Up(key))

		s.ExpireMachines(tailnet.Id)

		nodeB := s.NewTailscaleNode()
		require.NoError(t, nodeB.Up(key))
		require.NoError(t, nodeB.Check(tsn.HasExpiredPeer(nodeA.Hostname())))
	})
}
