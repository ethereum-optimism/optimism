package op_reth_upgrade

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/log"
)

// The test switches every node between two op-reth binaries over its live datadir: the release an
// operator runs today, which the environment provides under this name, and this repo's op-reth.
const (
	// currentOpReth is this repo's op-reth, resolved like every other devstack op-reth node.
	currentOpReth = "op-reth"
	// previousOpReth is the release the upgrade starts from and the rollback returns to. No
	// package in this repo builds it, so the environment must provide it through
	// RUST_BINARY_PATH_OP_RETH_PREVIOUS (see sysgo.OpRethWithBinary).
	previousOpReth = "op-reth-previous"
)

const (
	// blocksPerPhase is how far the chain is driven on each binary before the next switch. It
	// covers reth's largest default persistence threshold (50) plus twice its memory block buffer
	// target (5), so every binary has persisted part of its phase to the datadir and holds the
	// rest in memory when it is stopped.
	blocksPerPhase = 60
	// l2BlockTimeSeconds keeps the phases short.
	l2BlockTimeSeconds = 1
	// postSwitchTrafficBlocks is how far the chain is driven once a node is back, so the proofs
	// sweep also covers blocks the switched binary produced or indexed itself.
	postSwitchTrafficBlocks = 10
	// preSwitchSampledBlocks bounds how many of the blocks below the switched node's persisted tip
	// the proofs sweep compares. Each binary wrote its share of them the same way, so a sample
	// covers them; every block from the persisted tip on is compared.
	preSwitchSampledBlocks = 12
	// referenceEthProofWindow lets the sequencer answer eth_getProof for every block of the run
	// from reth's own historical state; reth's default window of zero rejects them all.
	referenceEthProofWindow = 1_000_000
	// resyncAttempts bounds, in 2s polls, how long a restarted node may take to catch up with
	// its peer, and how long a restarted sequencer may take to produce blocks again.
	resyncAttempts = 60
	// storeCatchUpTimeout bounds the wait for the follower's proofs store to index a block.
	storeCatchUpTimeout = 60 * time.Second
)

// shutdown says how the node's op-reth is stopped right before its binary is switched.
type shutdown string

const (
	// cleanShutdown interrupts op-reth and waits for it to exit on its own.
	cleanShutdown shutdown = "clean"
	// killShutdown ends op-reth with SIGKILL, so the blocks it held only in memory are lost.
	killShutdown shutdown = "kill"
)

// direction says which binary the switch under test moves the node to.
type direction string

const (
	// upgrade moves the node from previousOpReth to currentOpReth.
	upgrade direction = "upgrade"
	// rollback moves the node back to previousOpReth after a clean upgrade to currentOpReth, so
	// the datadir and proofs store it reopens hold blocks written by both binaries.
	rollback direction = "rollback"
)

// switchCase is one cell of the matrix a node's binary switch runs under.
type switchCase struct {
	direction direction
	shutdown  shutdown
}

func (c switchCase) String() string {
	return fmt.Sprintf("%s-%s", c.shutdown, c.direction)
}

func switchMatrix() []switchCase {
	var cases []switchCase
	for _, direction := range []direction{upgrade, rollback} {
		for _, shutdown := range []shutdown{cleanShutdown, killShutdown} {
			cases = append(cases, switchCase{direction: direction, shutdown: shutdown})
		}
	}
	return cases
}

func skipUnsupported(t devtest.T) {
	sysgo.SkipWithoutRustBinaryPath(t, previousOpReth,
		"the upgrade starts from the previous op-reth release, which CI does not provide yet (#23395)")
}

// upgradeNode is one op-reth node of the system with the consensus client that drives it and the
// peers it must be reconnected to after a restart.
type upgradeNode struct {
	CL *dsl.L2CLNode
	EL *dsl.L2ELNode

	PeerCL *dsl.L2CLNode
	PeerEL *dsl.L2ELNode
}

// opRethBinary is an op-reth binary a node can be switched to. Its path is resolved before the
// chain starts, so a RUST_JIT_BUILD build never runs while a node is stopped for a switch.
type opRethBinary struct {
	name string
	path string
}

func resolveOpRethBinary(t devtest.T, name string) opRethBinary {
	return opRethBinary{name: name, path: sysgo.ResolveOpRethBinary(t, name)}
}

type upgradeSystem struct {
	Batcher        *dsl.L2Batcher
	CurrentOpReth  opRethBinary
	Follower       upgradeNode
	FunderL2       *dsl.FunderEOA
	PreviousOpReth opRethBinary
	Sequencer      upgradeNode
}

// newUpgradeSystem starts a sequencer and a proofs-history follower, both on previousOpReth. The
// sequencer runs without proofs history and serves eth_getProof from its own historical state,
// so it is the independent reference for the follower's proofs.
func newUpgradeSystem(t devtest.T) *upgradeSystem {
	clKind := sysgo.ResolveMixedL2CLKind()
	currentBinary := resolveOpRethBinary(t, currentOpReth)
	previousBinary := resolveOpRethBinary(t, previousOpReth)
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		DeployerOptions: []sysgo.DeployerOption{sysgo.WithUniformL2BlockTimes(l2BlockTimeSeconds)},
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				CLKey:       "sequencer",
				CLKind:      clKind,
				ELKey:       "sequencer",
				ELKind:      sysgo.MixedL2ELOpReth,
				IsSequencer: true,
				OpRethOpts: []sysgo.OpRethOption{
					sysgo.OpRethWithBinary(previousOpReth),
					sysgo.OpRethWithoutProofsHistory(),
					sysgo.OpRethWithEthProofWindow(referenceEthProofWindow),
				},
			},
			{
				CLKey:      "follower",
				CLKind:     clKind,
				ELKey:      "proofs-follower",
				ELKind:     sysgo.MixedL2ELOpReth,
				OpRethOpts: []sysgo.OpRethOption{sysgo.OpRethWithBinary(previousOpReth)},
			},
		},
	})
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	t.Require().Len(frontends.Nodes, 2, "expected the sequencer and the proofs follower")

	sys := &upgradeSystem{
		Batcher:        frontends.L2Batcher,
		CurrentOpReth:  currentBinary,
		FunderL2:       frontends.FunderL2,
		PreviousOpReth: previousBinary,
	}
	for _, node := range frontends.Nodes {
		if node.Spec.IsSequencer {
			sys.Sequencer.EL, sys.Sequencer.CL = node.EL, node.CL
		} else {
			sys.Follower.EL, sys.Follower.CL = node.EL, node.CL
		}
	}
	t.Require().NotNil(sys.Follower.EL, "missing the proofs follower")
	sys.Sequencer.PeerEL, sys.Sequencer.PeerCL = sys.Follower.EL, sys.Follower.CL
	sys.Follower.PeerEL, sys.Follower.PeerCL = sys.Sequencer.EL, sys.Sequencer.CL
	return sys
}

// runPhase drives the chain blocksPerPhase blocks further and waits for the follower to import
// and index the result, so the binaries running now have persisted part of the phase and hold
// the rest in memory. It returns the last block of the phase.
func (sys *upgradeSystem) runPhase(load *dsl.BlockRecorderWorkload) uint64 {
	target := sys.Sequencer.EL.BlockRefByLabel(eth.Unsafe).Number + blocksPerPhase
	last := load.DriveUntilBlock(target)
	sys.Follower.EL.Reached(eth.Unsafe, last, resyncAttempts)
	sys.Follower.EL.WaitForProofsStoreBlock(last, storeCatchUpTimeout)
	return last
}

// restartedNode is what a binary switch left behind: the binary the node runs now, the chain head
// the node had when it was stopped, the block the new binary resumed at and the recorder of the
// errors the new binary logged since it started.
type restartedNode struct {
	binary       opRethBinary
	errorLogs    *dsl.LogRecorder
	headAtStop   eth.L2BlockRef
	persistedTip eth.L2BlockRef
}

// switchBinary stops the node's consensus client, then its op-reth the given way, switches it to
// binary and brings it back on its datadir, re-peered with its peers, then restarts the consensus
// client. phaseStart is the block the binary being replaced took over at. replaced is what the
// switch that started that binary left behind, nil for the binary the system started with; its
// errors are verified once that binary has exited, so they include its shutdown.
func switchBinary(t devtest.T, node upgradeNode, binary opRethBinary, stopMode shutdown, phaseStart uint64, replaced *restartedNode) restartedNode {
	node.CL.Stop()
	headAtStop := node.EL.BlockRefByLabel(eth.Unsafe)
	t.Logger().Info("Switching node binary", "name", node.EL, "binary", binary.name, "shutdown", stopMode, "head", headAtStop)
	if stopMode == killShutdown {
		node.EL.StopUngracefully()
	} else {
		node.EL.Stop()
	}
	if replaced != nil {
		replaced.errorLogs.Stop()
		replaced.errorLogs.VerifyEmpty(fmt.Sprintf("running on %s until its %s shutdown", replaced.binary.name, stopMode))
	}

	node.EL.SwitchBinary(binary.path)
	errorLogs := node.EL.RecordLogs(log.LevelError)
	node.EL.Start()
	node.EL.WaitForOnline()
	persistedTip := node.EL.BlockRefByLabel(eth.Unsafe)
	t.Logger().Info("Node is back on its datadir", "name", node.EL, "binary", binary.name,
		"persisted_tip", persistedTip, "head_at_stop", headAtStop, "phase_start", phaseStart)
	if stopMode == cleanShutdown {
		// op-reth writes every block it holds in memory to the datadir before a clean exit. The
		// tip may lie past headAtStop when a late engine call from the stopped consensus client
		// extended the chain, so the check is that headAtStop survived, not that it is the tip.
		t.Require().GreaterOrEqualf(persistedTip.Number, headAtStop.Number,
			"%s resumed at %s, below the head %s it had when it was stopped cleanly", node.EL, persistedTip, headAtStop)
		t.Require().Truef(node.EL.IsCanonical(headAtStop.ID()),
			"%s resumed at %s on a chain without the head %s it had when it was stopped cleanly", node.EL, persistedTip, headAtStop)
	} else {
		t.Require().Greaterf(persistedTip.Number, phaseStart,
			"%s resumed at block %d, at or below the block %d the killed binary took over at: it never persisted its phase", node.EL, persistedTip.Number, phaseStart)
		node.PeerEL.Reached(eth.Unsafe, persistedTip.Number, resyncAttempts)
		t.Require().Truef(node.PeerEL.IsCanonical(persistedTip.ID()),
			"%s resumed at %s, which is not on the chain its peer %s follows", node.EL, persistedTip, node.PeerEL)
	}

	node.EL.PeerWith(node.PeerEL)
	node.CL.Start()
	node.CL.ConnectPeer(node.PeerCL)
	return restartedNode{binary: binary, errorLogs: errorLogs, headAtStop: headAtStop, persistedTip: persistedTip}
}

// followerResumed waits for the restarted follower to catch up with the sequencer.
func (sys *upgradeSystem) followerResumed(restartedNode) {
	sys.Follower.EL.InSync(sys.Sequencer.EL, safety.LocalUnsafe, resyncAttempts)
}

// sequencerResumed waits for the restarted sequencer to produce blocks past the head it had when
// it was stopped, which a sequencer that lost blocks to a kill must first rebuild.
func (sys *upgradeSystem) sequencerResumed(restarted restartedNode) {
	sys.Sequencer.EL.Reached(eth.Unsafe, restarted.headAtStop.Number+postSwitchTrafficBlocks, resyncAttempts)
}

// followerAdoptsSequencerChain moves the follower's op-reth onto the chain the sequencer rebuilt
// after a kill, through the engine API calls op-node makes when derivation replaces its unsafe
// chain. It requires a stopped batcher, which leaves nothing to derive, and op-node drops
// gossiped blocks at or below its unsafe head or off its unsafe chain, so nothing else moves the
// follower there. Its consensus client is stopped meanwhile and loads the new heads from op-reth
// when it restarts.
func (sys *upgradeSystem) followerAdoptsSequencerChain(t devtest.T) {
	sys.Follower.CL.Stop()
	target := sys.Sequencer.EL.BlockRefByLabel(eth.Unsafe)
	forkPoint := lastSharedBlock(t, sys.Follower.EL, sys.Sequencer.EL, target.Number)
	safe := sys.Follower.EL.BlockRefByLabel(eth.Safe)
	finalized := sys.Follower.EL.BlockRefByLabel(eth.Finalized)
	t.Logger().Info("Moving the follower onto the sequencer's chain", "fork_point", forkPoint, "target", target,
		"safe", safe, "finalized", finalized)
	t.Require().LessOrEqualf(safe.Number, forkPoint.Number,
		"the follower's safe head %s lies above the last block %s it shares with the sequencer", safe, forkPoint)

	for number := forkPoint.Number + 1; number <= target.Number; number++ {
		inserted := sys.Follower.EL.NewPayload(sys.Sequencer.EL, number)
		t.Require().NoErrorf(inserted.Err, "%s failed to take the sequencer's block %d", sys.Follower.EL, number)
		t.Require().NotNilf(inserted.Status, "%s returned no status for the sequencer's block %d", sys.Follower.EL, number)
		if inserted.Status.Status == eth.ExecutionInvalid {
			validationError := "no validation error"
			if inserted.Status.ValidationError != nil {
				validationError = *inserted.Status.ValidationError
			}
			t.Require().Failf("follower rejected the sequencer's block",
				"%s rejected the sequencer's block %d: %s", sys.Follower.EL, number, validationError)
		}
	}
	sys.Follower.EL.ForkchoiceUpdateRaw(target.Hash, safe.Hash, finalized.Hash, nil).WaitUntilValid(resyncAttempts)
	t.Logger().Info("Follower holds the sequencer's chain", "head", sys.Follower.EL.BlockRefByLabel(eth.Unsafe))

	sys.Follower.CL.Start()
	sys.Follower.CL.ConnectPeer(sys.Sequencer.CL)
}

// lastSharedBlock returns the highest block at or below ceiling that node holds as reference does.
func lastSharedBlock(t devtest.T, node, reference *dsl.L2ELNode, ceiling uint64) eth.L2BlockRef {
	for number := min(node.BlockRefByLabel(eth.Unsafe).Number, ceiling); ; number-- {
		ref := reference.BlockRefByNumber(number)
		if node.IsCanonical(ref.ID()) {
			return ref
		}
		t.Require().NotZerof(number, "%s and %s share no block, not even genesis", node, reference)
	}
}

// switchUnderTest performs the cell's switch of the node and waits for it to resume: an upgrade
// from previousOpReth, or a clean upgrade followed by a phase on currentOpReth and then the
// rollback under test. firstPreSwitchBlock is the block the phase on previousOpReth began at.
func switchUnderTest(t devtest.T, sys *upgradeSystem, node upgradeNode, load *dsl.BlockRecorderWorkload, cell switchCase, firstPreSwitchBlock uint64,
	resumed func(restartedNode)) restartedNode {
	if cell.direction == upgrade {
		restarted := switchBinary(t, node, sys.CurrentOpReth, cell.shutdown, firstPreSwitchBlock, nil)
		resumed(restarted)
		return restarted
	}
	upgraded := switchBinary(t, node, sys.CurrentOpReth, cleanShutdown, firstPreSwitchBlock, nil)
	resumed(upgraded)
	sys.runPhase(load)
	restarted := switchBinary(t, node, sys.PreviousOpReth, cell.shutdown, upgraded.persistedTip.Number, &upgraded)
	resumed(restarted)
	return restarted
}

// verifyFollowerProofs requires the follower's proofs of accounts to match the sequencer's from
// firstPreSwitchBlock to lastBlock: at a sample of the blocks below the switched node's persisted
// tip and at every block from it on, where the blocks a switch lost, rebuilt or reorged lie.
func (sys *upgradeSystem) verifyFollowerProofs(accounts []dsl.ProofAccount, firstPreSwitchBlock uint64, restarted restartedNode, lastBlock uint64) {
	sys.Follower.EL.VerifyHistoricalProofsMatch(sys.Sequencer.EL, func(opts *dsl.HistoricalProofsOpts) {
		opts.Accounts = accounts
		opts.FirstBlock = firstPreSwitchBlock
		opts.LastBlock = restarted.persistedTip.Number
		opts.MaxSampledBlocks = preSwitchSampledBlocks
	})
	sys.Follower.EL.VerifyHistoricalProofsMatch(sys.Sequencer.EL, func(opts *dsl.HistoricalProofsOpts) {
		opts.Accounts = accounts
		opts.FirstBlock = restarted.persistedTip.Number
		opts.LastBlock = lastBlock
	})
}

// TestOpRethFollowerServesProofsAcrossBinarySwitch runs a proofs-history follower on the previous
// op-reth release, stops it cleanly or by SIGKILL, restarts it on this repo's op-reth over the
// same datadir and proofs store (and, for the rollback cells, back again), and requires it to
// catch up with the sequencer, keep up with new blocks, serve proofs that match the sequencer's
// for a sample of the blocks from the first pre-switch phase on and for every block from the one
// it resumed at, and log no error while doing so.
func TestOpRethFollowerServesProofsAcrossBinarySwitch(gt *testing.T) {
	t := devtest.ParallelT(gt)
	devtest.RunParallel(t, switchMatrix(), func(t devtest.T, cell switchCase) {
		skipUnsupported(t)
		if cell.direction == rollback {
			t.Skip("fails on develop (ethereum-optimism/protocol-team#482): the previous op-reth aborts at startup " +
				"decoding the ExEx WAL the reth 2.6 binary wrote (`wrong msgpack marker FixArray(14)`); " +
				"the rollback runbook deletes `exex/wal` first")
		}
		sys := newUpgradeSystem(t)
		load := dsl.NewBlockRecorderWorkload(t, sys.FunderL2)
		firstPreSwitchBlock := sys.Sequencer.EL.BlockRefByLabel(eth.Unsafe).Number
		sys.runPhase(load)

		restarted := switchUnderTest(t, sys, sys.Follower, load, cell, firstPreSwitchBlock, sys.followerResumed)

		lastBlock := load.DriveUntilBlock(sys.Sequencer.EL.BlockRefByLabel(eth.Unsafe).Number + postSwitchTrafficBlocks)
		sys.verifyFollowerProofs(load.Accounts(), firstPreSwitchBlock, restarted, lastBlock)
		restarted.errorLogs.VerifyEmpty(fmt.Sprintf("following the chain after the %s", cell.direction))
	})
}

// TestOpRethSequencerResumesSequencingAcrossBinarySwitch runs a sequencer on the previous op-reth
// release, stops it cleanly or by SIGKILL, restarts it on this repo's op-reth over the same
// datadir (and, for the rollback cells, back again), and requires it to resume sequencing on the
// chain its datadir holds, keep the follower in sync, serve the historical state the follower's
// proofs are checked against from the first pre-switch phase on, and log no error while doing so.
// The batcher is stopped for the whole run, so no block reaches L1 and the sequencer's datadir
// alone decides the chain it resumes on. After a kill the follower still holds the blocks the
// sequencer lost, so it is moved onto the rebuilt chain before it is required to keep up.
func TestOpRethSequencerResumesSequencingAcrossBinarySwitch(gt *testing.T) {
	t := devtest.ParallelT(gt)
	devtest.RunParallel(t, switchMatrix(), func(t devtest.T, cell switchCase) {
		skipUnsupported(t)
		sys := newUpgradeSystem(t)
		sys.Batcher.Stop()
		load := dsl.NewBlockRecorderWorkload(t, sys.FunderL2)
		firstPreSwitchBlock := sys.Sequencer.EL.BlockRefByLabel(eth.Unsafe).Number
		sys.runPhase(load)

		restarted := switchUnderTest(t, sys, sys.Sequencer, load, cell, firstPreSwitchBlock, sys.sequencerResumed)
		if cell.shutdown == killShutdown {
			sys.followerAdoptsSequencerChain(t)
		}

		postSwitchWorkload := load.WithFreshSenders(sys.FunderL2)
		lastBlock := postSwitchWorkload.DriveUntilBlock(sys.Sequencer.EL.BlockRefByLabel(eth.Unsafe).Number + postSwitchTrafficBlocks)
		sys.Follower.EL.InSync(sys.Sequencer.EL, safety.LocalUnsafe, resyncAttempts)
		sys.verifyFollowerProofs(append(load.Accounts(), postSwitchWorkload.SenderAccounts()...), firstPreSwitchBlock, restarted, lastBlock)
		restarted.errorLogs.VerifyEmpty(fmt.Sprintf("sequencing after the %s", cell.direction))
	})
}
