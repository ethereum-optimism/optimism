package op_e2e

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

func TestStopStartSequencer(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)
	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l2Seq := sys.Clients["sequencer"]
	rollupNode := sys.RollupNodes["sequencer"]

	nodeRPC, err := rpc.DialContext(context.Background(), rollupNode.HTTPEndpoint())
	require.Nil(t, err, "Error dialing node")
	rollupClient := sources.NewRollupClient(client.NewBaseRPCClient(nodeRPC))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	active, err := rollupClient.SequencerActive(ctx)
	require.NoError(t, err)
	require.True(t, active, "sequencer should be active")

	require.NoError(
		t,
		wait.ForNextBlock(ctx, l2Seq),
		"Chain did not advance after starting sequencer",
	)

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	blockHash, err := rollupClient.StopSequencer(ctx)
	require.Nil(t, err, "Error stopping sequencer")

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	active, err = rollupClient.SequencerActive(ctx)
	require.NoError(t, err)
	require.False(t, active, "sequencer should be inactive")

	blockBefore := latestBlock(t, l2Seq)
	time.Sleep(time.Duration(cfg.DeployConfig.L2BlockTime+1) * time.Second)
	blockAfter := latestBlock(t, l2Seq)
	require.Equal(t, blockAfter, blockBefore, "Chain advanced after stopping sequencer")

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = rollupClient.StartSequencer(ctx, blockHash)
	require.Nil(t, err, "Error starting sequencer")

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	active, err = rollupClient.SequencerActive(ctx)
	require.NoError(t, err)
	require.True(t, active, "sequencer should be active again")

	require.NoError(
		t,
		wait.ForNextBlock(ctx, l2Seq),
		"Chain did not advance after starting sequencer",
	)
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

	sys, err := cfg.Start(t)
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

	sys, err := cfg.Start(t)
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

	sys, err := cfg.Start(t)
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

func TestPostUnsafePayload(t *testing.T) {
	InitParallel(t)
	ctx := context.Background()

	cfg := DefaultSystemConfig(t)
	cfg.Nodes["verifier"].RPC.EnableAdmin = true
	cfg.DisableBatcher = true

	sys, err := cfg.Start(t)
	require.NoError(t, err)
	defer sys.Close()

	l2Seq := sys.Clients["sequencer"]
	l2Ver := sys.Clients["verifier"]
	rollupClient := sys.RollupClient("verifier")

	require.NoError(t, wait.ForBlock(ctx, l2Seq, 2), "Chain did not advance after starting sequencer")
	verBlock, err := l2Ver.BlockByNumber(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(0), verBlock.NumberU64(), "Verifier should not have advanced any blocks since p2p & batcher are not enabled")

	blockNumberOne, err := l2Seq.BlockByNumber(ctx, big.NewInt(1))
	require.NoError(t, err)
	payload, err := eth.BlockAsPayload(blockNumberOne, sys.RollupConfig.CanyonTime)
	require.NoError(t, err)
	err = rollupClient.PostUnsafePayload(ctx, &eth.ExecutionPayloadEnvelope{ExecutionPayload: payload})
	require.NoError(t, err)
	require.NoError(t, wait.ForUnsafeBlock(ctx, rollupClient, 1), "Chain did not advance after posting payload")

	// Test validation
	blockNumberTwo, err := l2Seq.BlockByNumber(ctx, big.NewInt(2))
	require.NoError(t, err)
	payload, err = eth.BlockAsPayload(blockNumberTwo, sys.RollupConfig.CanyonTime)
	require.NoError(t, err)
	payload.BlockHash = common.Hash{0xaa}
	err = rollupClient.PostUnsafePayload(ctx, &eth.ExecutionPayloadEnvelope{ExecutionPayload: payload})
	require.ErrorContains(t, err, "payload has bad block hash")
}

func assertPersistedSequencerState(t *testing.T, stateFile string, expected node.RunningState) {
	configReader := node.NewConfigPersistence(stateFile)
	state, err := configReader.SequencerState()
	require.NoError(t, err)
	require.Equalf(t, expected, state, "expected sequencer state %v but was %v", expected, state)
}
