package proofs_history_reorg

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
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	// l2BlockTimeSeconds keeps the runs short: the blocks sequenced by op-node before the fork
	// point and after the reorg take a block time each.
	l2BlockTimeSeconds = 1
	// memoryBlockBufferTarget is reth's --engine.memory-block-buffer-target, the number of newest
	// blocks reth keeps in memory after persisting. It is passed explicitly, like the persistence
	// threshold, because the reorg depths are chosen relative to it.
	memoryBlockBufferTarget = 5
	// postReorgTrafficBlocks is how far past the sequencer's head the chain is driven once the
	// follower has adopted the replacement chain, so the sweep also covers blocks the follower
	// executed after the reorg.
	postReorgTrafficBlocks = 10
	// referenceEthProofWindow lets the sequencer answer eth_getProof for every block of the run
	// from reth's own historical state; reth's default window of zero rejects them all.
	referenceEthProofWindow = 1_000_000

	// blockAttempts bounds, in 2s polls, the wait for a node to reach a block the test sequencer
	// just committed.
	blockAttempts = 30
	// reorgAttempts bounds, in 2s polls, how long the follower may take to adopt the replacement
	// chain. op-node drops gossiped blocks that do not extend its unsafe chain; once a gossiped
	// block above its head arrives, its unsafe-gap request hands that block to reth, which
	// fetches the missing replacement blocks from its EL peer and switches once they connect.
	reorgAttempts = 90
	// storeCatchUpTimeout bounds the wait for the follower's proofs store to flush the chain
	// head once sequencing stopped; the store flushes buffered blocks after 10s idle.
	storeCatchUpTimeout = 60 * time.Second
	// syncAttempts bounds, in 2s polls, the wait for the follower to catch up with the sequencer.
	syncAttempts = 60
)

var (
	recorderNumberSlot    = common.HexToHash("0x0")
	recorderTimestampSlot = common.HexToHash("0x1")
)

// reorgWorkload says what the replaced and the replacement blocks change.
type reorgWorkload string

const (
	// deployedContract: the first replaced block deploys a contract and the replaced blocks
	// after it call it. The replacement block re-includes the deployment, so the contract exists
	// at the same address on the replacement chain, and calls it right away; the traffic after
	// the reorg keeps calling it. The follower must execute calls into code that only a
	// replacement block created.
	deployedContract reorgWorkload = "deployed_contract"
	// plainTransfers: every block moves ether between accounts, nothing else.
	plainTransfers reorgWorkload = "plain_transfers"
)

// reorgKind is how reth resolves the reorg, decided by where the fork point sits relative to
// reth's persisted tip.
type reorgKind string

const (
	// inMemoryReorg: the fork point is a block reth still holds in memory or the block it
	// persisted last, so only reth's in-memory chain is rewound.
	inMemoryReorg reorgKind = "in_memory"
	// onDiskReorg: the fork point lies below reth's persisted tip, so reth unwinds its database
	// before it can extend the fork point.
	onDiskReorg reorgKind = "on_disk"
)

// reorgCase is one cell of the matrix the follower runs under.
type reorgCase struct {
	// depth is the number of blocks the reorg replaces.
	depth                uint64
	kind                 reorgKind
	persistenceThreshold uint64
	stateMaskingBlocks   uint64
	workload             reorgWorkload
}

func (c reorgCase) name() string {
	return fmt.Sprintf("threshold_%d-masking_%d-%s-depth_%d_%s",
		c.persistenceThreshold, c.stateMaskingBlocks, c.workload, c.depth, c.kind)
}

// engineArgs pin reth's persistence schedule on both nodes, which decides whether a fork point of
// the cell's depth is in memory or on disk.
func (c reorgCase) engineArgs() []string {
	return []string{
		fmt.Sprintf("--engine.memory-block-buffer-target=%d", memoryBlockBufferTarget),
		fmt.Sprintf("--engine.persistence-threshold=%d", c.persistenceThreshold),
	}
}

// followerArgs put the follower on the re-execution path for every block.
func (c reorgCase) followerArgs() []string {
	args := append(c.engineArgs(), "--proofs-history.verification-interval=1")
	if c.stateMaskingBlocks != 0 {
		args = append(args, fmt.Sprintf("--engine.num-state-masking-blocks=%d", c.stateMaskingBlocks))
	}
	return args
}

// blocksInRethMemory models how many of the newest blocks reth holds in memory at height h:
// every block until the count first exceeds the persistence threshold, then the buffer target
// right after each persistence run, growing by one per block until the next run.
func (c reorgCase) blocksInRethMemory(h uint64) uint64 {
	firstPersistence := c.persistenceThreshold + 1
	if h < firstPersistence {
		return h
	}
	blocksBetweenRuns := firstPersistence - memoryBlockBufferTarget
	return memoryBlockBufferTarget + (h-firstPersistence)%blocksBetweenRuns
}

// forkPointInMemory says whether reth, by the model above, still holds the fork point of a
// reorg of the cell's depth in memory when the chain head is at height head.
func (c reorgCase) forkPointInMemory(head uint64) bool {
	return c.blocksInRethMemory(head) >= c.depth
}

// reorgMatrix crosses the reth settings and the workloads with three reorg depths per threshold:
// shallower than the memory block buffer target, so the fork point is among the blocks reth
// never persisted; deeper than it, so the fork point has been eligible for persistence; and
// deeper than reth ever holds in memory, which reth resolves as an on-disk reorg. reth holds
// between the buffer target and the threshold of blocks in memory, so the on-disk depths exceed
// the threshold with a margin.
func reorgMatrix() []reorgCase {
	type depthCell struct {
		depth uint64
		kind  reorgKind
	}
	depthsByThreshold := []struct {
		persistenceThreshold uint64
		depths               []depthCell
	}{
		{persistenceThreshold: 7, depths: []depthCell{{2, inMemoryReorg}, {6, inMemoryReorg}, {12, onDiskReorg}}},
		{persistenceThreshold: 50, depths: []depthCell{{2, inMemoryReorg}, {6, inMemoryReorg}, {55, onDiskReorg}}},
	}
	var cases []reorgCase
	for _, threshold := range depthsByThreshold {
		for _, stateMaskingBlocks := range []uint64{0, 30} {
			for _, workload := range []reorgWorkload{deployedContract, plainTransfers} {
				for _, cell := range threshold.depths {
					cases = append(cases, reorgCase{
						depth:                cell.depth,
						kind:                 cell.kind,
						persistenceThreshold: threshold.persistenceThreshold,
						stateMaskingBlocks:   stateMaskingBlocks,
						workload:             workload,
					})
				}
			}
		}
	}
	return cases
}

// skipUnsupportedCell skips the cells the configured clients cannot run.
func skipUnsupportedCell(t devtest.T, c reorgCase) {
	sysgo.SkipOnOpGeth(t, "proofs history is an op-reth feature")
	sysgo.SkipOnKonaNode(t, "the test sequencer commits blocks through op-node's opstack_commitBlockV1 and opstack_publishBlockV1 RPCs, which kona-node does not serve")
	if c.stateMaskingBlocks != 0 {
		t.Skip("state masking is compiled out of reth 2.5.2 on develop and #23046 rejects a nonzero --engine.num-state-masking-blocks; re-enable with ethereum-optimism/protocol-team#457")
	}
}

type reorgSystem struct {
	Batcher       *dsl.L2Batcher
	Follower      *dsl.L2ELNode
	FunderL2      *dsl.FunderEOA
	L2Network     *dsl.L2Network
	Sequencer     *dsl.L2ELNode
	SequencerCL   *dsl.L2CLNode
	TestSequencer *dsl.TestSequencer
}

// newReorgSystem starts a sequencer without proofs history, which serves as the independent
// eth_getProof reference and whose op-node the test sequencer drives around the reorg, and a
// follower re-executing every block into its proofs store under the cell's reth settings.
func newReorgSystem(t devtest.T, c reorgCase) *reorgSystem {
	clKind := sysgo.ResolveMixedL2CLKind()
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
					sysgo.OpRethWithoutProofsHistory(),
					sysgo.OpRethWithEthProofWindow(referenceEthProofWindow),
					sysgo.OpRethWithExtraArgs(c.engineArgs()...),
				},
			},
			{
				CLKey:      "follower",
				CLKind:     clKind,
				ELKey:      "proofs-follower",
				ELKind:     sysgo.MixedL2ELOpReth,
				OpRethOpts: []sysgo.OpRethOption{sysgo.OpRethWithExtraArgs(c.followerArgs()...)},
			},
		},
		TestSequencerName: "test-sequencer",
		WithTestSequencer: true,
	})
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	t.Require().Len(frontends.Nodes, 2, "expected the sequencer and the proofs follower")
	t.Require().NotNil(frontends.TestSequencer, "missing the test sequencer")

	sys := &reorgSystem{
		Batcher:       frontends.L2Batcher,
		FunderL2:      frontends.FunderL2,
		L2Network:     frontends.L2Network,
		TestSequencer: frontends.TestSequencer,
	}
	for _, node := range frontends.Nodes {
		if node.Spec.IsSequencer {
			sys.Sequencer, sys.SequencerCL = node.EL, node.CL
		} else {
			sys.Follower = node.EL
		}
	}
	t.Require().NotNil(sys.Follower, "missing the proofs follower")
	return sys
}

// followerLogs are the follower's op-reth log lines the test reads after the reorg.
type followerLogs struct {
	// leftChain is the ExEx handing a block that left reth's chain mid-execution to its sync
	// loop; the line exists from the reth 2.6 bump (#23046) on.
	leftChain *logpipe.LogRecord
	// onDiskReorg is reth resolving a fork point below its persisted tip.
	onDiskReorg *logpipe.LogRecord
	// panicked is a reth critical task, such as the ExEx, crashing the node.
	panicked *logpipe.LogRecord
	// parentStateMissing is the ExEx skipping a block whose parent state reth no longer holds,
	// leaving it to a later notification.
	parentStateMissing *logpipe.LogRecord
	// reorgHandled is the proofs engine unwinding blocks it stored and buffering the replacement
	// path, as opposed to ignoring a reorg above its stored tip.
	reorgHandled *logpipe.LogRecord
	// reorgNotified is the ExEx receiving reth's reorg notification.
	reorgNotified *logpipe.LogRecord
}

func recordFollowerLogs(follower *dsl.L2ELNode) followerLogs {
	return followerLogs{
		leftChain:          follower.RecordLogLines("Block left reth's chain during proofs execution"),
		onDiskReorg:        follower.RecordLogLines("on-disk reorg detected"),
		panicked:           follower.RecordLogLines("panicked"),
		parentStateMissing: follower.RecordLogLines("Parent state not available in reth; skipping execute_block"),
		reorgHandled:       follower.RecordLogLines("Handling reorg: unwinding and buffering new path"),
		reorgNotified:      follower.RecordLogLines("ChainReorged notification received"),
	}
}

// verify requires the follower to have handled the reorg the way the cell expects, and reports
// the lines whose presence depends on timing or on the op-reth version under test.
func (l followerLogs) verify(t devtest.T, c reorgCase) {
	t.Require().NotZerof(l.reorgNotified.Count(), "the follower's ExEx never received a reorg notification")
	t.Require().NotZerof(l.reorgHandled.Count(), "the follower's proofs engine never unwound the replaced blocks: the reorg did not reach blocks it had stored")
	if c.kind == onDiskReorg {
		t.Require().NotZerof(l.onDiskReorg.Count(), "expected a fork point below reth's persisted tip, but reth never reported an on-disk reorg; the depth %d is not deeper than what reth held in memory", c.depth)
	} else {
		t.Require().Zerof(l.onDiskReorg.Count(), "expected the fork point in reth's memory, but reth reported an on-disk reorg: %v", messagesOf(l.onDiskReorg))
	}
	t.Logger().Info("Follower reorg handling log lines",
		"reorg_notified", l.reorgNotified.Count(),
		"reorg_handled", l.reorgHandled.Count(),
		"on_disk_reorg", l.onDiskReorg.Count(),
		"parent_state_missing_skips", l.parentStateMissing.Count(),
		"left_chain_handoffs", l.leftChain.Count())
}

func messagesOf(record *logpipe.LogRecord) []string {
	var messages []string
	for _, entry := range record.Entries() {
		messages = append(messages, fmt.Sprintf("%s %v", entry.LogMessage(), entry.LogFields()))
	}
	return messages
}

// workload is the accounts the blocks around the reorg change. alice and bob transact before
// the fork point, in the replaced blocks and after the reorg; carol only ever transacts on the
// replacement chain, so the nonces her replacement transactions need are the same on both sides
// of the fork; the deployer's only transaction is the recorder deployment, which the first
// replaced block carries and the replacement block carries again.
type workload struct {
	alice    *dsl.EOA
	bob      *dsl.EOA
	carol    *dsl.EOA
	deployer *dsl.EOA
	kind     reorgWorkload
	t        devtest.T

	deployment signedTx
	recorder   common.Address
	// setupBlock is the block that completed the funding; the fork point must be above it.
	setupBlock uint64
}

func newWorkload(t devtest.T, funder *dsl.FunderEOA, kind reorgWorkload) *workload {
	w := &workload{
		alice:    funder.NewFundedEOA(eth.OneEther),
		bob:      funder.NewFundedEOA(eth.OneEther),
		carol:    funder.NewFundedEOA(eth.OneEther),
		deployer: funder.NewFundedEOA(eth.OneEther),
		kind:     kind,
		t:        t,
	}
	w.setupBlock = includedBlock(t, w.alice.Transfer(w.bob.Address(), eth.OneGWei))
	if kind == deployedContract {
		w.deployment = signTx(t, w.deployer.PlanBlockRecorderDeployment())
		w.recorder = crypto.CreateAddress(w.deployer.Address(), w.deployment.nonce)
		t.Logger().Info("Signed the recorder deployment", "address", w.recorder, "tx", w.deployment.hash)
	}
	return w
}

// accounts are the accounts the proofs sweep proves at every block.
func (w *workload) accounts() []dsl.ProofAccount {
	accounts := []dsl.ProofAccount{
		{Address: w.alice.Address()},
		{Address: w.bob.Address()},
		{Address: w.carol.Address()},
		{Address: w.deployer.Address()},
	}
	if w.kind == deployedContract {
		accounts = append(accounts, dsl.ProofAccount{Address: w.recorder, StorageKeys: []common.Hash{recorderNumberSlot, recorderTimestampSlot}})
	}
	return accounts
}

// driveUntilBlock sends transactions one after another, each after the previous one was
// included, until one lands in a block at or past target, and returns that block.
func (w *workload) driveUntilBlock(target uint64) uint64 {
	for {
		if block := includedBlock(w.t, w.alice.Transfer(w.bob.Address(), eth.OneHundredthEther)); block >= target {
			return block
		}
		if block := includedBlock(w.t, w.bob.Transact(w.bobsReply())); block >= target {
			return block
		}
	}
}

// bobsReply is bob's part of the traffic: a call into the recorder for the deployed-contract
// workload, which before the deployment is a plain transaction to an address without code, a
// transfer back to alice otherwise.
func (w *workload) bobsReply() txplan.Option {
	if w.kind == deployedContract {
		return txplan.Combine(w.bob.Plan(), txplan.WithTo(&w.recorder))
	}
	return w.bob.PlanTransfer(w.alice.Address(), eth.OneHundredthEther)
}

// replacedBlockTxs are the transactions of the n-th replaced block, 1-based: the recorder
// deployment first and calls into it after that for the deployed-contract workload, transfers
// otherwise.
func (w *workload) replacedBlockTxs(n int) []signedTx {
	switch {
	case w.kind == deployedContract && n == 1:
		return []signedTx{w.deployment, signTx(w.t, txplan.NewPlannedTx(w.alice.PlanTransfer(w.bob.Address(), eth.OneHundredthEther)))}
	case w.kind == deployedContract:
		return []signedTx{signTx(w.t, txplan.NewPlannedTx(w.alice.Plan(), txplan.WithTo(&w.recorder)))}
	default:
		return []signedTx{signTx(w.t, txplan.NewPlannedTx(w.alice.PlanTransfer(w.bob.Address(), eth.OneHundredthEther)))}
	}
}

// replacementBlockTxs are the transactions of the replacement block: a transfer from carol, and
// for the deployed-contract workload the same recorder deployment as on the replaced chain
// followed by carol's call into it. carol's nonces are assigned up front, since both of her
// transactions are signed before either is included.
func (w *workload) replacementBlockTxs(int) []signedTx {
	nonce := w.carol.PendingNonce()
	transfer := signTx(w.t, txplan.NewPlannedTx(w.carol.PlanTransfer(w.bob.Address(), eth.OneHundredthEther), txplan.WithStaticNonce(nonce)))
	if w.kind != deployedContract {
		return []signedTx{transfer}
	}
	call := signTx(w.t, txplan.NewPlannedTx(w.carol.Plan(), txplan.WithTo(&w.recorder), txplan.WithStaticNonce(nonce+1)))
	return []signedTx{w.deployment, transfer, call}
}

type signedTx struct {
	hash  common.Hash
	nonce uint64
	raw   []byte
}

func signTx(t devtest.T, tx *txplan.PlannedTx) signedTx {
	signed, err := tx.Signed.Eval(t.Ctx())
	t.Require().NoError(err, "transaction must sign")
	raw, err := signed.MarshalBinary()
	t.Require().NoError(err, "transaction must encode")
	return signedTx{hash: signed.Hash(), nonce: signed.Nonce(), raw: raw}
}

func rawTxs(txs []signedTx) [][]byte {
	raws := make([][]byte, len(txs))
	for i, tx := range txs {
		raws[i] = tx.raw
	}
	return raws
}

func includedBlock(t devtest.T, tx *txplan.PlannedTx) uint64 {
	receipt, err := tx.Included.Eval(t.Ctx())
	t.Require().NoError(err, "transaction must be included")
	return bigs.Uint64Strict(receipt.BlockNumber)
}

// sequencedChain is a run of consecutive blocks the test sequencer committed, lowest first, with
// the transactions each of them carries.
type sequencedChain struct {
	blocks []eth.L2BlockRef
	txs    [][]signedTx
}

func (s sequencedChain) head() eth.L2BlockRef {
	return s.blocks[len(s.blocks)-1]
}

// verifyServedBy requires node to serve every block of the chain with its transactions.
func (s sequencedChain) verifyServedBy(t devtest.T, node *dsl.L2ELNode) {
	for i, block := range s.blocks {
		t.Require().Equalf(block.Hash, node.BlockRefByNumber(block.Number).Hash, "%s must serve block %d of the chain", node, block.Number)
		for _, tx := range s.txs[i] {
			node.AssertTxInBlock(block.Number, tx.hash)
		}
	}
}

// sequenceChain commits count blocks on parent with the test sequencer, each carrying the
// transactions blockTxs returns for it and none ahead of wall-clock time, and returns them once
// the sequencer's EL serves each.
func sequenceChain(t devtest.T, sys *reorgSystem, parent eth.L2BlockRef, count uint64, blockTxs func(n int) []signedTx) sequencedChain {
	chain := sequencedChain{}
	for i := range int(count) {
		txs := blockTxs(i + 1)
		t.Logger().Info("Sequencing a block with the test sequencer", "number", parent.Number+1, "parent", parent, "txs", len(txs))
		sys.TestSequencer.SequenceBlockWithTxsWhenDue(t, sys.L2Network, parent, rawTxs(txs))
		block := sys.Sequencer.BlockRefByNumber(parent.Number + 1)
		t.Require().Equalf(parent.Hash, block.ParentHash, "block %d must extend %s", block.Number, parent)
		chain.blocks = append(chain.blocks, block)
		chain.txs = append(chain.txs, txs)
		parent = block
	}
	return chain
}

// stopSequencing drives traffic past the workload's setup and stops op-node's sequencer, which
// makes its head the fork point of the reorg. For an in-memory cell the chain is then padded
// with empty blocks until the replaced blocks will leave the fork point in reth's memory by the
// persistence model, so the fork point does not depend on where the stop lands.
func stopSequencing(t devtest.T, sys *reorgSystem, w *workload, c reorgCase) eth.L2BlockRef {
	w.driveUntilBlock(w.setupBlock + 1)
	forkPoint := sys.Sequencer.BlockRefByHash(sys.SequencerCL.StopSequencer())
	padding := 0
	for c.kind == inMemoryReorg && !c.forkPointInMemory(forkPoint.Number+c.depth) {
		t.Require().Lessf(padding, int(c.persistenceThreshold),
			"no height keeps a fork point %d blocks deep in reth's memory at threshold %d", c.depth, c.persistenceThreshold)
		forkPoint = sequenceChain(t, sys, forkPoint, 1, func(int) []signedTx { return nil }).head()
		padding++
	}
	t.Logger().Info("Sequencing stopped at the fork point", "fork_point", forkPoint, "depth", c.depth, "kind", c.kind, "padding_blocks", padding)
	return forkPoint
}

// sequenceReplacedChain builds the blocks the reorg will replace on the fork point and waits
// until the follower has imported them and its proofs store has flushed them, so every replaced
// block was re-executed into the store before it is replaced.
func sequenceReplacedChain(t devtest.T, sys *reorgSystem, w *workload, forkPoint eth.L2BlockRef, c reorgCase) sequencedChain {
	replaced := sequenceChain(t, sys, forkPoint, c.depth, w.replacedBlockTxs)
	sys.Follower.Reached(eth.Unsafe, replaced.head().Number, blockAttempts)
	replaced.verifyServedBy(t, sys.Follower)
	sys.Follower.WaitForProofsStoreBlock(replaced.head().Number, storeCatchUpTimeout)
	return replaced
}

// sequenceReplacementChain builds the first replacement block on the fork point, which reorgs
// the sequencer's EL, and waits until the sequencer's op-node is on it so it resumes sequencing
// from there. One block is all the test sequencer can build on a deep fork: its timestamp is
// the fork point's plus a block time, which op-node's gossip validation rejects as too old once
// the replaced chain took longer than the gossip age threshold to build, and a block that is
// not published leaves the test sequencer's job sealed. op-node builds the remaining
// replacement blocks, as it does after a real sequencer failover.
func sequenceReplacementChain(t devtest.T, sys *reorgSystem, w *workload, forkPoint eth.L2BlockRef, replaced sequencedChain) sequencedChain {
	replacement := sequenceChain(t, sys, forkPoint, 1, w.replacementBlockTxs)
	t.Require().NotEqualf(replaced.blocks[0].Hash, replacement.head().Hash, "the replacement block must differ from the replaced block at height %d", replacement.head().Number)
	sys.SequencerCL.MatchedUnsafe(sys.Sequencer, syncAttempts)
	return replacement
}

// verifyRecorderCalled requires the recorder's storage on node to hold the number of a block
// after the fork point at block, which only a call on the replacement chain can have written, so
// the sweep's matching proofs of that slot were not vacuous.
func verifyRecorderCalled(t devtest.T, node *dsl.L2ELNode, w *workload, forkPoint eth.L2BlockRef, block uint64) {
	ref := node.BlockRefByNumber(block)
	value, err := node.EthClient().ReadStorageAt(t.Ctx(), w.recorder, recorderNumberSlot, ref.Hash)
	t.Require().NoError(err, "must read the recorder's storage at block %d", block)
	t.Require().Greaterf(bigs.Uint64Strict(value.Big()), forkPoint.Number,
		"the recorder at block %d must have been called on the replacement chain, but holds block number %s", block, value.Big())
}

// TestProofsHistoryFollowerMatchesTheReplacementChainAfterASequencerReorg has a follower that
// re-executes every block into its proofs store watch the sequencer replace blocks the store
// already holds: op-node stops sequencing at the fork point, the test sequencer builds the
// blocks to be replaced on top and the follower executes them, then the test sequencer builds a
// conflicting block on the fork point and op-node resumes on top of it. The follower must adopt
// the replacement chain, keep executing, and serve proofs from the fork point onwards that match
// the sequencer's own historical state.
func TestProofsHistoryFollowerMatchesTheReplacementChainAfterASequencerReorg(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range reorgMatrix() {
		t.Run(c.name(), func(t devtest.T) {
			t.Parallel()
			skipUnsupportedCell(t, c)
			sys := newReorgSystem(t, c)
			// Blocks that reach L1 are derived back as they were, so the batcher stays off until
			// the replacement chain exists and is the one it submits.
			sys.Batcher.Stop()
			logs := recordFollowerLogs(sys.Follower)
			w := newWorkload(t, sys.FunderL2, c.workload)

			forkPoint := stopSequencing(t, sys, w, c)
			replaced := sequenceReplacedChain(t, sys, w, forkPoint, c)
			replacement := sequenceReplacementChain(t, sys, w, forkPoint, replaced)
			sys.Batcher.Start()
			sys.SequencerCL.StartSequencer()

			sys.Follower.ReorgExact(replaced.blocks[0], reorgAttempts)
			t.Require().Emptyf(messagesOf(logs.panicked), "the follower's op-reth must survive the reorg")
			sys.Follower.InSync(sys.Sequencer, safety.LocalUnsafe, syncAttempts)
			replacement.verifyServedBy(t, sys.Follower)
			lastBlock := w.driveUntilBlock(sys.Sequencer.BlockRefByLabel(eth.Unsafe).Number + postReorgTrafficBlocks)

			sys.Follower.VerifyHistoricalProofsMatch(sys.Sequencer, func(opts *dsl.HistoricalProofsOpts) {
				opts.Accounts = w.accounts()
				opts.FirstBlock = forkPoint.Number
				opts.LastBlock = lastBlock
			})
			if c.workload == deployedContract {
				verifyRecorderCalled(t, sys.Follower, w, forkPoint, lastBlock)
			}
			logs.verify(t, c)
		})
	}
}
