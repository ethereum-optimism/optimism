package op_e2e

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestStopStartSequencer(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)
	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l2Seq := sys.Clients["sequencer"]
	rollupNode := sys.RollupNodes["sequencer"]

	nodeRPC, err := rpc.DialContext(context.Background(), rollupNode.HTTPEndpoint())
	require.Nil(t, err, "Error dialing node")

	blockBefore := latestBlock(t, l2Seq)
	time.Sleep(time.Duration(cfg.DeployConfig.L2BlockTime+1) * time.Second)
	blockAfter := latestBlock(t, l2Seq)
	require.Greaterf(t, blockAfter, blockBefore, "Chain did not advance")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	blockHash := common.Hash{}
	err = nodeRPC.CallContext(ctx, &blockHash, "admin_stopSequencer")
	require.Nil(t, err, "Error stopping sequencer")

	blockBefore = latestBlock(t, l2Seq)
	time.Sleep(time.Duration(cfg.DeployConfig.L2BlockTime+1) * time.Second)
	blockAfter = latestBlock(t, l2Seq)
	require.Equal(t, blockAfter, blockBefore, "Chain advanced after stopping sequencer")

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = nodeRPC.CallContext(ctx, nil, "admin_startSequencer", blockHash)
	require.Nil(t, err, "Error starting sequencer")

	blockBefore = latestBlock(t, l2Seq)
	time.Sleep(time.Duration(cfg.DeployConfig.L2BlockTime+1) * time.Second)
	blockAfter = latestBlock(t, l2Seq)
	require.Greater(t, blockAfter, blockBefore, "Chain did not advance after starting sequencer")
}

func TestPersistSequencerStateWhenChanged(t *testing.T) {
	InitParallel(t)
	ctx := context.Background()
	dir := t.TempDir()
	stateFile := dir + "/state.json"

	cfg := DefaultSystemConfig(t)
	// We don't need a verifier - just the sequencer is enough
	delete(cfg.Nodes, "verifier")
	cfg.Nodes["sequencer"].ConfigPersistence = node.NewConfigPersistence(stateFile)

	sys, err := cfg.Start()
	require.NoError(t, err)
	defer sys.Close()

	assertPersistedSequencerState(t, stateFile, node.StateStarted)

	rollupRPCClient, err := rpc.DialContext(ctx, sys.RollupNodes["sequencer"].HTTPEndpoint())
	require.Nil(t, err)
	rollupClient := sources.NewRollupClient(client.NewBaseRPCClient(rollupRPCClient))

	err = rollupClient.StartSequencer(ctx, common.Hash{0xaa})
	require.ErrorContains(t, err, "sequencer already running")

	head, err := rollupClient.StopSequencer(ctx)
	require.NoError(t, err)
	require.NotEqual(t, common.Hash{}, head)
	assertPersistedSequencerState(t, stateFile, node.StateStopped)
}

func TestLoadSequencerStateOnStarted_Stopped(t *testing.T) {
	InitParallel(t)
	ctx := context.Background()
	dir := t.TempDir()
	stateFile := dir + "/state.json"

	// Prepare the persisted state file with sequencer stopped
	configReader := node.NewConfigPersistence(stateFile)
	require.NoError(t, configReader.SequencerStopped())

	cfg := DefaultSystemConfig(t)
	// We don't need a verifier - just the sequencer is enough
	delete(cfg.Nodes, "verifier")
	seqCfg := cfg.Nodes["sequencer"]
	seqCfg.ConfigPersistence = node.NewConfigPersistence(stateFile)

	sys, err := cfg.Start()
	require.NoError(t, err)
	defer sys.Close()

	rollupRPCClient, err := rpc.DialContext(ctx, sys.RollupNodes["sequencer"].HTTPEndpoint())
	require.Nil(t, err)
	rollupClient := sources.NewRollupClient(client.NewBaseRPCClient(rollupRPCClient))

	// Still persisted as stopped after startup
	assertPersistedSequencerState(t, stateFile, node.StateStopped)

	// Sequencer is really stopped
	_, err = rollupClient.StopSequencer(ctx)
	require.ErrorContains(t, err, "sequencer not running")
	assertPersistedSequencerState(t, stateFile, node.StateStopped)
}

func TestLoadSequencerStateOnStarted_Started(t *testing.T) {
	InitParallel(t)
	ctx := context.Background()
	dir := t.TempDir()
	stateFile := dir + "/state.json"

	// Prepare the persisted state file with sequencer stopped
	configReader := node.NewConfigPersistence(stateFile)
	require.NoError(t, configReader.SequencerStarted())

	cfg := DefaultSystemConfig(t)
	// We don't need a verifier - just the sequencer is enough
	delete(cfg.Nodes, "verifier")
	seqCfg := cfg.Nodes["sequencer"]
	seqCfg.Driver.SequencerStopped = true
	seqCfg.ConfigPersistence = node.NewConfigPersistence(stateFile)

	sys, err := cfg.Start()
	require.NoError(t, err)
	defer sys.Close()

	rollupRPCClient, err := rpc.DialContext(ctx, sys.RollupNodes["sequencer"].HTTPEndpoint())
	require.Nil(t, err)
	rollupClient := sources.NewRollupClient(client.NewBaseRPCClient(rollupRPCClient))

	// Still persisted as stopped after startup
	assertPersistedSequencerState(t, stateFile, node.StateStarted)

	// Sequencer is really stopped
	err = rollupClient.StartSequencer(ctx, common.Hash{})
	require.ErrorContains(t, err, "sequencer already running")
	assertPersistedSequencerState(t, stateFile, node.StateStarted)
}

func assertPersistedSequencerState(t *testing.T, stateFile string, expected node.RunningState) {
	configReader := node.NewConfigPersistence(stateFile)
	state, err := configReader.SequencerState()
	require.NoError(t, err)
	require.Equalf(t, expected, state, "expected sequencer state %v but was %v", expected, state)
}
