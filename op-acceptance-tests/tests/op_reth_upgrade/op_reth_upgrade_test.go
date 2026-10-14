package op_reth_upgrade

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
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

func (c switchCase) name() string {
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
	sysgo.SkipOnOpGeth(t, "the binary switch is an op-reth datadir upgrade")
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

type upgradeSystem struct {
	Follower  upgradeNode
	FunderL2  *dsl.FunderEOA
	Sequencer upgradeNode
}

// newUpgradeSystem starts a sequencer and a proofs-history follower, both on previousOpReth. The
// sequencer runs without proofs history and serves eth_getProof from its own historical state,
// so it is the independent reference for the follower's proofs.
func newUpgradeSystem(t devtest.T) *upgradeSystem {
	clKind := sysgo.ResolveMixedL2CLKind()
	// Resolved before the chain starts so a RUST_JIT_BUILD build cannot run inside a switch.
	sysgo.ResolveOpRethBinary(t, currentOpReth)
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

	sys := &upgradeSystem{FunderL2: frontends.FunderL2}
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

// workload changes state in nearly every block: alice and bob transfer back and forth and bob
// writes to a contract, so the blocks each binary writes carry balance and storage changes the
// proofs sweep can tell apart.
type workload struct {
	alice    *dsl.EOA
	bob      *dsl.EOA
	recorder common.Address
	t        devtest.T
}

func newWorkload(t devtest.T, funder *dsl.FunderEOA) *workload {
	w := &workload{
		alice: funder.NewFundedEOA(eth.OneEther),
		bob:   funder.NewFundedEOA(eth.OneEther),
		t:     t,
	}
	w.recorder = w.alice.DeployBlockRecorder()
	return w
}

// accounts are the accounts the proofs sweep proves at every block.
func (w *workload) accounts() []dsl.ProofAccount {
	return []dsl.ProofAccount{
		{Address: w.alice.Address()},
		{Address: w.bob.Address()},
		{Address: w.recorder, StorageKeys: []common.Hash{common.HexToHash("0x0"), common.HexToHash("0x1")}},
	}
}

// driveUntilBlock sends transactions one after another until one lands in a block at or past
// target, and returns that block.
func (w *workload) driveUntilBlock(target uint64) uint64 {
	for {
		w.alice.Transfer(w.bob.Address(), eth.OneHundredthEther)
		w.bob.Transfer(w.alice.Address(), eth.OneHundredthEther)
		block := includedBlock(w.t, w.bob.Transact(w.bob.Plan(), txplan.WithTo(&w.recorder)))
		if block >= target {
			return block
		}
	}
}

func includedBlock(t devtest.T, tx *txplan.PlannedTx) uint64 {
	receipt, err := tx.Included.Eval(t.Ctx())
	t.Require().NoError(err, "transaction must be included")
	return bigs.Uint64Strict(receipt.BlockNumber)
}

// runPhase drives the chain blocksPerPhase blocks further and waits for the follower to import
// and index the result, so the binaries running now have persisted part of the phase and hold
// the rest in memory. It returns the last block of the phase.
func (sys *upgradeSystem) runPhase(w *workload) uint64 {
	target := sys.Sequencer.EL.BlockRefByLabel(eth.Unsafe).Number + blocksPerPhase
	last := w.driveUntilBlock(target)
	sys.Follower.EL.Reached(eth.Unsafe, last, resyncAttempts)
	sys.Follower.EL.WaitForProofsStoreBlock(last, storeCatchUpTimeout)
	return last
}

// restartedNode is what a binary switch left behind: the chain head the node had when it was
// stopped and the recorder of the errors the new binary logged since it started.
type restartedNode struct {
	errorLogs  *dsl.LogRecorder
	headAtStop eth.L2BlockRef
}

// switchBinary stops the node's consensus client, then its op-reth the case's way, switches it
// to binary and brings it back on its datadir, re-peered with its peers, then restarts the
// consensus client. Nothing feeds op-reth once its head is read, so what the datadir holds when
// the node is back is exactly what the stop left behind. phaseStart is the chain head when the
// binary being replaced started running.
func switchBinary(t devtest.T, n upgradeNode, binary string, how shutdown, phaseStart uint64) restartedNode {
	n.CL.Stop()
	headAtStop := n.EL.BlockRefByLabel(eth.Unsafe)
	t.Logger().Info("Switching node binary", "name", n.EL, "binary", binary, "shutdown", how, "head", headAtStop)
	if how == killShutdown {
		n.EL.StopUngracefully()
	} else {
		n.EL.Stop()
	}

	n.EL.SwitchBinary(binary)
	errorLogs := n.EL.RecordLogs(log.LevelError)
	n.EL.Start()
	n.EL.WaitForOnline()
	persistedTip := n.EL.BlockRefByLabel(eth.Unsafe)
	t.Logger().Info("Node is back on its datadir", "name", n.EL, "binary", binary,
		"persisted_tip", persistedTip, "head_at_stop", headAtStop, "phase_start", phaseStart)
	if how == cleanShutdown {
		// op-reth writes every block it holds in memory to the datadir before a clean exit.
		t.Require().Equalf(headAtStop.ID(), persistedTip.ID(),
			"%s resumed at %s, not at the head %s it had when it was stopped cleanly", n.EL, persistedTip, headAtStop)
	} else {
		t.Require().LessOrEqualf(persistedTip.Number, headAtStop.Number,
			"%s resumed at block %d, above the head %d it had when it was killed", n.EL, persistedTip.Number, headAtStop.Number)
		t.Require().Greaterf(persistedTip.Number, phaseStart,
			"%s resumed at block %d, at or below the block %d the killed binary started at: it never persisted", n.EL, persistedTip.Number, phaseStart)
		t.Require().Truef(n.PeerEL.IsCanonical(persistedTip.ID()),
			"%s resumed at %s, which is not on the chain its peer %s follows", n.EL, persistedTip, n.PeerEL)
	}

	n.EL.PeerWith(n.PeerEL)
	n.CL.Start()
	n.CL.ConnectPeer(n.PeerCL)
	return restartedNode{errorLogs: errorLogs, headAtStop: headAtStop}
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

// switchUnderTest performs the case's switch of the node and waits for it to resume: an upgrade
// from previousOpReth, or a clean upgrade followed by a phase on currentOpReth and then the
// rollback under test.
func switchUnderTest(t devtest.T, sys *upgradeSystem, n upgradeNode, w *workload, c switchCase, resumed func(restartedNode)) restartedNode {
	if c.direction == upgrade {
		restarted := switchBinary(t, n, currentOpReth, c.shutdown, 0)
		resumed(restarted)
		return restarted
	}
	upgraded := switchBinary(t, n, currentOpReth, cleanShutdown, 0)
	resumed(upgraded)
	sys.runPhase(w)
	upgraded.errorLogs.VerifyEmpty("running on " + currentOpReth + " after the clean upgrade")
	restarted := switchBinary(t, n, previousOpReth, c.shutdown, upgraded.headAtStop.Number)
	resumed(restarted)
	return restarted
}

// TestOpRethFollowerServesProofsAcrossBinarySwitch runs a proofs-history follower on the previous
// op-reth release, stops it cleanly or by SIGKILL, restarts it on this repo's op-reth over the
// same datadir and proofs store (and, for the rollback cells, back again), and requires it to
// catch up with the sequencer, keep up with new blocks, serve proofs for the blocks every binary
// wrote that match the sequencer's, and log no error while doing so.
func TestOpRethFollowerServesProofsAcrossBinarySwitch(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range switchMatrix() {
		t.Run(c.name(), func(t devtest.T) {
			t.Parallel()
			skipUnsupported(t)
			sys := newUpgradeSystem(t)
			w := newWorkload(t, sys.FunderL2)
			sys.runPhase(w)

			restarted := switchUnderTest(t, sys, sys.Follower, w, c, sys.followerResumed)

			lastBlock := w.driveUntilBlock(sys.Sequencer.EL.BlockRefByLabel(eth.Unsafe).Number + postSwitchTrafficBlocks)
			sys.Follower.EL.VerifyHistoricalProofsMatch(sys.Sequencer.EL, func(opts *dsl.HistoricalProofsOpts) {
				opts.Accounts = w.accounts()
				opts.LastBlock = lastBlock
			})
			restarted.errorLogs.VerifyEmpty(fmt.Sprintf("following the chain after the %s", c.direction))
		})
	}
}

// TestOpRethSequencerResumesSequencingAcrossBinarySwitch runs a sequencer on the previous op-reth
// release, stops it cleanly or by SIGKILL, restarts it on this repo's op-reth over the same
// datadir (and, for the rollback cells, back again), and requires it to resume sequencing on the
// chain its datadir holds, keep the follower in sync, serve the historical state the follower's
// proofs are checked against, and log no error while doing so.
func TestOpRethSequencerResumesSequencingAcrossBinarySwitch(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range switchMatrix() {
		t.Run(c.name(), func(t devtest.T) {
			t.Parallel()
			skipUnsupported(t)
			sys := newUpgradeSystem(t)
			w := newWorkload(t, sys.FunderL2)
			sys.runPhase(w)

			restarted := switchUnderTest(t, sys, sys.Sequencer, w, c, sys.sequencerResumed)

			lastBlock := w.driveUntilBlock(sys.Sequencer.EL.BlockRefByLabel(eth.Unsafe).Number + postSwitchTrafficBlocks)
			sys.Follower.EL.InSync(sys.Sequencer.EL, safety.LocalUnsafe, resyncAttempts)
			sys.Follower.EL.VerifyHistoricalProofsMatch(sys.Sequencer.EL, func(opts *dsl.HistoricalProofsOpts) {
				opts.Accounts = w.accounts()
				opts.LastBlock = lastBlock
			})
			restarted.errorLogs.VerifyEmpty(fmt.Sprintf("sequencing after the %s", c.direction))
		})
	}
}
