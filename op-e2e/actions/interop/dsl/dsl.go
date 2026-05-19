package dsl

import (
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/stretchr/testify/require"
)

type ChainOpts struct {
	Chains []*Chain
}

func (c *ChainOpts) SetChains(chains ...*Chain) {
	c.Chains = chains
}

// InteropDSL provides a high-level API to drive interop action tests so that the actual test reads more declaratively
// and is separated from the details of how each action is actually executed.
type InteropDSL struct {
	t      helpers.Testing
	Actors *InteropActors
	setup  *InteropSetup

	allChains []*Chain
}

func NewInteropDSL(t helpers.Testing, opts ...setupOption) *InteropDSL {
	setup := SetupInterop(t, opts...)
	actors := setup.CreateActors()
	actors.PrepareAndVerifyInitialState(t)
	t.Logf("ChainA: %v, ChainB: %v", actors.ChainA.ChainID, actors.ChainB.ChainID)

	return &InteropDSL{
		t:         t,
		Actors:    actors,
		setup:     setup,
		allChains: []*Chain{actors.ChainA, actors.ChainB},
	}
}

func (d *InteropDSL) defaultChainOpts() ChainOpts {
	return ChainOpts{
		// Defensive copy to make sure the original slice isn't modified
		Chains: append([]*Chain{}, d.allChains...),
	}
}

// AddL2Block adds a new unsafe block to the specified chain and fully processes it in the supervisor.
func (d *InteropDSL) AddL2Block(chain *Chain) {
	priorSyncStatus := chain.Sequencer.SyncStatus()
	chain.Sequencer.ActL2StartBlock(d.t)
	chain.Sequencer.ActL2EndBlock(d.t)
	chain.Sequencer.SyncSupervisor(d.t)
	d.Actors.Supervisor.ProcessFull(d.t)
	chain.Sequencer.ActL2PipelineFull(d.t)

	status := chain.Sequencer.SyncStatus()
	expectedBlockNum := priorSyncStatus.UnsafeL2.Number + 1
	require.Equal(d.t, expectedBlockNum, status.UnsafeL2.Number, "Unsafe head did not advance")
	require.Equal(d.t, expectedBlockNum, status.CrossUnsafeL2.Number, "CrossUnsafe head did not advance")
}

type SubmitBatchDataOpts struct {
	ChainOpts
}

// SubmitBatchData submits batch data to L1 and processes the new L1 blocks, advancing the safe heads.
// By default, submits all batch data for all chains.
func (d *InteropDSL) SubmitBatchData(optionalArgs ...func(*SubmitBatchDataOpts)) {
	opts := SubmitBatchDataOpts{
		ChainOpts: d.defaultChainOpts(),
	}
	for _, arg := range optionalArgs {
		arg(&opts)
	}
	txInclusion := make([]helpers.Action, 0, len(opts.Chains))
	for _, chain := range opts.Chains {
		chain.Batcher.ActSubmitAll(d.t)
		txInclusion = append(txInclusion, d.Actors.L1Miner.ActL1IncludeTx(chain.BatcherAddr))
	}
	// Always sync the new L1 block to all chains, even if only a subset submitted batches.
	d.advanceL1(d.allChains, txInclusion)

	// Verify the local safe head advanced on each chain
	for _, chain := range opts.Chains {
		status := chain.Sequencer.SyncStatus()
		require.Equalf(d.t, status.UnsafeL2, status.LocalSafeL2, "Chain %v did not fully advance local safe head", chain.ChainID)

		// Ingest the new local-safe event
		chain.Sequencer.SyncSupervisor(d.t)
	}

	d.processCrossSafe(opts.Chains)
}

// processCrossSafe processes events in the supervisor and nodes to ensure the cross-safe head is fully updated.
func (d *InteropDSL) processCrossSafe(chains []*Chain) {
	d.Actors.Supervisor.ProcessFull(d.t)

	for _, chain := range chains {
		chain.Sequencer.ActL2PipelineFull(d.t)
		chain.Sequencer.SyncSupervisor(d.t)
	}
	d.Actors.Supervisor.ProcessFull(d.t)
	// Re-run in case there was an invalid block that was replaced so it can now be considered safe
	for _, chain := range chains {
		chain.Sequencer.ActL2PipelineFull(d.t)
		chain.Sequencer.SyncSupervisor(d.t)
	}
	d.Actors.Supervisor.ProcessFull(d.t)
	for _, chain := range chains {
		status := chain.Sequencer.SyncStatus()
		require.Equalf(d.t, status.UnsafeL2, status.SafeL2, "Chain %v did not fully advance cross safe head", chain.ChainID)
	}
}

// advanceL1 mines a new L1 block including the provided transactions and ensures it is processed by the specified chains and the supervisor.
func (d *InteropDSL) advanceL1(chains []*Chain, txInclusion []helpers.Action) {
	expectedL1BlockNum := bigs.Uint64Strict(d.Actors.L1Miner.L1Chain().CurrentBlock().Number) + 1
	d.Actors.L1Miner.ActL1StartBlock(12)(d.t)
	for _, includeTx := range txInclusion {
		includeTx(d.t)
	}
	d.Actors.L1Miner.ActL1EndBlock(d.t)
	newBlock := eth.InfoToL1BlockRef(eth.HeaderBlockInfo(d.Actors.L1Miner.L1Chain().CurrentBlock()))
	require.Equal(d.t, expectedL1BlockNum, newBlock.Number, "L1 head did not advance")
	d.Actors.Supervisor.SignalLatestL1(d.t)

	// The node will exhaust L1 data, it needs the supervisor to see the L1 block first, and provide it to the node.
	for _, chain := range chains {
		chain.Sequencer.ActL2EventsUntil(d.t, event.Is[derive.ExhaustedL1Event], 100, false)
		chain.Sequencer.SyncSupervisor(d.t)
		chain.Sequencer.ActL2PipelineFull(d.t)
		chain.Sequencer.ActL1HeadSignal(d.t)
	}

	for _, chain := range chains {
		status := chain.Sequencer.SyncStatus()
		require.Equalf(d.t, newBlock, status.HeadL1, "Chain %v did not detect new L1 head", chain.ChainID)
		require.Equalf(d.t, newBlock, status.CurrentL1, "Chain %v did not process new L1 head", chain.ChainID)
	}
}

func (d *InteropDSL) FinalizeL1() {
	actors := d.Actors
	preStatus, err := actors.Supervisor.SyncStatus(d.t.Ctx())
	require.NoError(d.t, err)
	actors.L1Miner.ActL1SafeNext(d.t)
	actors.L1Miner.ActL1FinalizeNext(d.t)
	actors.Supervisor.SignalFinalizedL1(d.t)
	actors.Supervisor.ProcessFull(d.t)
	for _, chain := range d.allChains {
		chain.Sequencer.ActL2PipelineFull(d.t)
	}

	postStatus, err := actors.Supervisor.SyncStatus(d.t.Ctx())
	require.NoError(d.t, err)
	require.Greater(d.t, postStatus.FinalizedTimestamp, preStatus.FinalizedTimestamp)
}
