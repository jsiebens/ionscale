package tests

import (
	"github.com/jsiebens/ionscale/tests/sc"
	"github.com/jsiebens/ionscale/tests/tsn"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewHostnameShouldPropagateToPeersWhenSet(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		tailnet := s.CreateTailnet()
		key := s.CreateAuthKey(tailnet.Id, true)

		initialName := sc.RandomName()

		nodeA := s.NewTailscaleNode()
		nodeB := s.NewTailscaleNode(sc.WithName(initialName))

		require.NoError(t, nodeA.Up(key))
		require.NoError(t, nodeB.Up(key))

		require.NoError(t, nodeA.WaitFor(tsn.HasPeer(initialName)))

		newName := sc.RandomName()

		require.NoError(t, nodeB.SetHostname(newName))

		require.NoError(t, nodeA.WaitFor(tsn.HasPeer(newName)))
	})
}

func TestSetHostname(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		tailnet := s.CreateTailnet()
		key := s.CreateAuthKey(tailnet.Id, true)

		initialName := sc.RandomName()

		nodeA := s.NewTailscaleNode()
		nodeB := s.NewTailscaleNode(sc.WithName(initialName))

		require.NoError(t, nodeA.Up(key))
		require.NoError(t, nodeB.Up(key))

		require.NoError(t, nodeA.WaitFor(tsn.HasPeer(initialName)))

		mid, err := s.FindMachine(tailnet.Id, initialName)
		require.NoError(t, err)

		newName := sc.RandomName()

		require.NoError(t, s.SetMachineName(mid, false, newName))
		require.NoError(t, nodeA.WaitFor(tsn.HasPeer(newName)))

		require.NoError(t, s.SetMachineName(mid, true, ""))
		require.NoError(t, nodeA.WaitFor(tsn.HasPeer(initialName)))
	})
}

func TestSetHostnameWhenNameAlreadyInUse(t *testing.T) {
	sc.Run(t, func(s *sc.Scenario) {
		tailnet := s.CreateTailnet()
		key := s.CreateAuthKey(tailnet.Id, true)

		nodeA := s.NewTailscaleNode(sc.WithName("node-a"))
		nodeB := s.NewTailscaleNode(sc.WithName("node-b"))

		require.NoError(t, nodeA.Up(key))
		require.NoError(t, nodeB.Up(key))

		require.NoError(t, nodeA.WaitFor(tsn.PeerCount(1)))
		require.NoError(t, nodeB.WaitFor(tsn.PeerCount(1)))

		mida, err := s.FindMachine(tailnet.Id, "node-a")
		require.NoError(t, err)

		midb, err := s.FindMachine(tailnet.Id, "node-b")
		require.NoError(t, err)

		newName := sc.RandomName()

		require.NoError(t, s.SetMachineName(mida, false, newName))
		require.NoError(t, nodeB.WaitFor(tsn.HasPeer(newName)))

		err = s.SetMachineName(midb, false, newName)
		require.ErrorContains(t, err, "machine name already in use")
	})
}
