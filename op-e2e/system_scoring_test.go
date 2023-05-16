package op_e2e

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/p2pstub"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestBanPeerSendingInvalidGossip(t *testing.T) {
	InitParallel(t)
	ctx := context.Background()
	cfg := DefaultSystemConfig(t)
	delete(cfg.Nodes, "verifier") // Not needed in this test
	cfg.P2PPeerScoring = true
	// TODO: e2e setup needs to create the connction gater and connection manager
	// prepared will need to implement p2p.ExtraHostFeatures
	cfg.P2pNodes["nastyDude"] = &p2pstub.Config{}
	cfg.P2PTopology = map[string][]string{
		"nastyDude": {"sequencer"},
	}
	cfg.Loggers["nastyDude"] = testlog.Logger(t, log.LvlInfo).New("role", "nastyDude")

	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	nastyDude := sys.P2PNodes["nastyDude"]
	require.NoError(t, nastyDude.WaitForPeerCount(1), "should have connected to sequencer")

	require.NoError(t, nastyDude.JoinTopic(p2pstub.BlockTopic))
	require.NoError(t, nastyDude.WaitForPeerCountOnTopic(p2pstub.BlockTopic, 1))

	require.NoError(t, nastyDude.PublishGossip(ctx, p2pstub.BlockTopic, []byte{1}))
	require.NoError(t, nastyDude.PublishGossip(ctx, p2pstub.BlockTopic, []byte{2}))
	require.NoError(t, nastyDude.PublishGossip(ctx, p2pstub.BlockTopic, []byte{3}))

	// TODO: Make this work
	// require.NoError(t, nastyDude.WaitForPeerCount(0), "should be disconnected for bad behaviour")

	sequencerId := sys.RollupNodes["sequencer"].P2P().Host().ID()
	require.NoError(t, nastyDude.DisconnectPeer(sequencerId))
	require.NoError(t, nastyDude.WaitForPeerCount(0), "should be disconnected for bad behaviour")
	require.Error(t, nastyDude.ConnectPeer(ctx, sequencerId), "should not be able to reconnect")
}
