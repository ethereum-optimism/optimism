package conductor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

func TestPersistSequencerStateWhenChanged(t *testing.T) {
	op_e2e.InitParallel(t)
	ctx := context.Background()
	dir := t.TempDir()
	stateFile := dir + "/state.json"

	cfg := e2esys.DefaultSystemConfig(t)
	// We don't need a verifier - just the sequencer is enough
	delete(cfg.Nodes, "verifier")
	cfg.Nodes["sequencer"].ConfigPersistence = config.NewConfigPersistence(stateFile)

	sys, err := cfg.Start(t)
	require.NoError(t, err)

	assertPersistedSequencerState(t, stateFile, config.StateStarted)

	rollupRPCClient, err := rpc.DialContext(ctx, sys.RollupNodes["sequencer"].UserRPC().RPC())
	require.Nil(t, err)
	rollupClient := sources.NewRollupClient(client.NewBaseRPCClient(rollupRPCClient))

	err = rollupClient.StartSequencer(ctx, common.Hash{0xaa})
	require.ErrorContains(t, err, "sequencer already running")

	head, err := rollupClient.StopSequencer(ctx)
	require.NoError(t, err)
	require.NotEqual(t, common.Hash{}, head)
	assertPersistedSequencerState(t, stateFile, config.StateStopped)
}

func TestLoadSequencerStateOnStarted_Stopped(t *testing.T) {
	op_e2e.InitParallel(t)
	ctx := context.Background()
	dir := t.TempDir()
	stateFile := dir + "/state.json"

	// Prepare the persisted state file with sequencer stopped
	configReader := config.NewConfigPersistence(stateFile)
	require.NoError(t, configReader.SequencerStopped())

	cfg := e2esys.DefaultSystemConfig(t)
	// We don't need a verifier - just the sequencer is enough
	delete(cfg.Nodes, "verifier")
	seqCfg := cfg.Nodes["sequencer"]
	seqCfg.ConfigPersistence = config.NewConfigPersistence(stateFile)

	sys, err := cfg.Start(t)
	require.NoError(t, err)

	rollupRPCClient, err := rpc.DialContext(ctx, sys.RollupNodes["sequencer"].UserRPC().RPC())
	require.Nil(t, err)
	rollupClient := sources.NewRollupClient(client.NewBaseRPCClient(rollupRPCClient))

	// Still persisted as stopped after startup
	assertPersistedSequencerState(t, stateFile, config.StateStopped)

	// Sequencer is really stopped
	_, err = rollupClient.StopSequencer(ctx)
	require.ErrorContains(t, err, "sequencer not running")
	assertPersistedSequencerState(t, stateFile, config.StateStopped)
}

func TestLoadSequencerStateOnStarted_Started(t *testing.T) {
	op_e2e.InitParallel(t)
	ctx := context.Background()
	dir := t.TempDir()
	stateFile := dir + "/state.json"

	// Prepare the persisted state file with sequencer stopped
	configReader := config.NewConfigPersistence(stateFile)
	require.NoError(t, configReader.SequencerStarted())

	cfg := e2esys.DefaultSystemConfig(t)
	// We don't need a verifier - just the sequencer is enough
	delete(cfg.Nodes, "verifier")
	seqCfg := cfg.Nodes["sequencer"]
	seqCfg.Driver.SequencerStopped = true
	seqCfg.ConfigPersistence = config.NewConfigPersistence(stateFile)

	sys, err := cfg.Start(t)
	require.NoError(t, err)

	rollupRPCClient, err := rpc.DialContext(ctx, sys.RollupNodes["sequencer"].UserRPC().RPC())
	require.Nil(t, err)
	rollupClient := sources.NewRollupClient(client.NewBaseRPCClient(rollupRPCClient))

	// Still persisted as stopped after startup
	assertPersistedSequencerState(t, stateFile, config.StateStarted)

	// Sequencer is really stopped
	err = rollupClient.StartSequencer(ctx, common.Hash{})
	require.ErrorContains(t, err, "sequencer already running")
	assertPersistedSequencerState(t, stateFile, config.StateStarted)
}

func assertPersistedSequencerState(t *testing.T, stateFile string, expected config.RunningState) {
	configReader := config.NewConfigPersistence(stateFile)
	state, err := configReader.SequencerState()
	require.NoError(t, err)
	require.Equalf(t, expected, state, "expected sequencer state %v but was %v", expected, state)
}
