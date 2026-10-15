package core

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/rust/op-reth/tests/proofs/utils"
	"golang.org/x/sync/errgroup"
)

const (
	// proofsHistoryWindow is the follower's window: small enough that pruning moves it during the
	// test, so the earliest block is one pruning left behind and not the genesis anchor.
	proofsHistoryWindow = 20
	// tracingPermits caps the follower's concurrent tracing requests.
	tracingPermits = 2
	// concurrentWitnessRequests is well above tracingPermits, so requests have to queue for a
	// permit.
	concurrentWitnessRequests = 8

	// latencySamples is how many eth_getProof calls per block the latency check times.
	latencySamples = 10
	// latencyFactor bounds the median latency at the earliest block relative to the latest, so
	// a lookup whose cost grows with the depth in the window shows up.
	latencyFactor = 10
	// latencyFloor keeps the latency check from failing on sub-millisecond noise.
	latencyFloor = 250 * time.Millisecond

	// reorgAttempts bounds, in 2s polls, how long the follower may take to adopt the replacement
	// block, which it only learns of once a block above it is gossiped.
	reorgAttempts = 90
)

// windowCase is one cell of the matrix the proofs follower runs under.
type windowCase struct {
	persistenceThreshold uint64
	verificationInterval uint64
}

func (c windowCase) proofsPath() string {
	if c.verificationInterval != 0 {
		return "execute"
	}
	return "index"
}

func (c windowCase) name() string {
	return fmt.Sprintf("threshold_%d-%s", c.persistenceThreshold, c.proofsPath())
}

func (c windowCase) followerArgs() []string {
	return append(engineArgs(c.persistenceThreshold),
		fmt.Sprintf("--proofs-history.verification-interval=%d", c.verificationInterval),
		fmt.Sprintf("--rpc.max-tracing-requests=%d", tracingPermits),
	)
}

func windowMatrix() []windowCase {
	var cases []windowCase
	for _, persistenceThreshold := range persistenceThresholds {
		for _, verificationInterval := range []uint64{0, 1} {
			cases = append(cases, windowCase{persistenceThreshold: persistenceThreshold, verificationInterval: verificationInterval})
		}
	}
	return cases
}

func newWindowSystem(t devtest.T, c windowCase) *utils.MixedOpProofPreset {
	return newProofsFollowerSystem(t, engineArgs(c.persistenceThreshold),
		sysgo.OpRethWithProofsHistoryWindow(proofsHistoryWindow),
		sysgo.OpRethWithExtraArgs(c.followerArgs()...),
	)
}

// blockTags are the two ways a client names a canonical block: by number and by hash.
func blockTags(node *dsl.L2ELNode, number uint64) []string {
	return []string{blockNumberTag(number), node.BlockRefByNumber(number).Hash.String()}
}

// requireServedAndMatching requires eth_getProof at block on follower to succeed by number and by
// hash with proofs that verify, and the proofs of every workload account to equal reference's.
func requireServedAndMatching(t devtest.T, follower, reference *dsl.L2ELNode, w *workload, block uint64) {
	follower.VerifyProofsMatchAt(reference, block, w.accounts()...)
	for _, tag := range blockTags(follower, block) {
		proof, err := normalizedProofAt(t, follower, w.accounts()[2], tag)
		t.Require().NoErrorf(err, "eth_getProof at block %d (%s) must succeed", block, tag)
		requireProofVerifies(t, follower, proof, block)
	}
}

// medianProofLatency times latencySamples eth_getProof calls for account at block on node and
// returns their median.
func medianProofLatency(t devtest.T, node *dsl.L2ELNode, account dsl.ProofAccount, block uint64) time.Duration {
	samples := make([]time.Duration, 0, latencySamples)
	for range latencySamples {
		started := time.Now()
		_, err := normalizedProofAt(t, node, account, blockNumberTag(block))
		t.Require().NoErrorf(err, "eth_getProof at block %d must succeed while its latency is measured", block)
		samples = append(samples, time.Since(started))
	}
	slices.Sort(samples)
	return samples[len(samples)/2]
}

// sequenceBlockOn has the test sequencer build one block on parent carrying rawTxs and returns it
// once the sequencer's EL serves it.
func sequenceBlockOn(t devtest.T, sys *utils.MixedOpProofPreset, parent eth.L2BlockRef, rawTxs ...[]byte) eth.L2BlockRef {
	sys.TestSequencer.SequenceBlockWithTxsWhenDue(t, sys.L2Chain, parent, rawTxs)
	block := sys.L2ELSequencer.BlockRefByNumber(parent.Number + 1)
	t.Require().Equalf(parent.Hash, block.ParentHash, "block %d must extend %s", block.Number, parent)
	return block
}

// TestProofsFollowerServesExactlyTheWindowItReports drives a proofs follower with a small window
// until pruning has moved the window past the workload's setup, stops the chain so the window
// stands still, and checks what eth_getProof, debug_executePayload and debug_executionWitness
// answer at and around its edges: every block the window reports is served with a proof that
// verifies and matches the sequencer's own historical state, every block outside it is refused
// with an error naming the block, never with a proof of another block, and more concurrent
// witness requests than the follower has tracing permits all succeed. Every subtest starts with
// the chain stopped; the last two resume it to check a height the store has not flushed yet and
// the hash of a block a reorg removed.
func TestProofsFollowerServesExactlyTheWindowItReports(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range windowMatrix() {
		t.Run(c.name(), func(t devtest.T) {
			t.Parallel()
			sys := newWindowSystem(t, c)
			follower, reference := sys.L2ELValidator, sys.L2ELSequencer
			w := newWorkload(t, sys.FunderL2)

			w.driveUntilBlock(w.setupBlock + proofsHistoryWindow + windowMargin)
			follower.WaitForProofsStorePrunedPast(w.setupBlock, pruneTimeout)
			head, window := stopChainAndSettleStore(t, sys)
			t.Require().Equalf(uint64(proofsHistoryWindow), *window.Latest-*window.Earliest,
				"pruning must trim the store to its window of %d blocks behind the latest; window %s", proofsHistoryWindow, window)
			earliest := *window.Earliest
			recorder := w.accounts()[2]

			t.Run("earliest block is served by number and by hash", func(t devtest.T) {
				requireServedAndMatching(t, follower, reference, w, earliest)
			})

			t.Run("block below the earliest is refused by number and by hash", func(t devtest.T) {
				pruned := earliest - 1
				for _, tag := range blockTags(reference, pruned) {
					_, err := normalizedProofAt(t, follower, recorder, tag)
					requireProofRefused(t, err, pruned, fmt.Sprintf("at pruned block %d (%s)", pruned, tag))
				}
			})

			t.Run("latest block is served by number, by hash and by the latest tag", func(t devtest.T) {
				requireServedAndMatching(t, follower, reference, w, head.Number)
				proof, err := normalizedProofAt(t, follower, recorder, "latest")
				t.Require().NoError(err, "eth_getProof at the latest tag must succeed once the store holds the head")
				referenceProof, err := normalizedProofAt(t, reference, recorder, head.Hash.String())
				t.Require().NoErrorf(err, "the reference must serve eth_getProof at the head %s", head)
				requireProofVerifies(t, follower, proof, head.Number)
				t.Require().Equal(referenceProof, proof, "the latest tag must resolve to the head the store holds")
			})

			t.Run("eth_getProof at the earliest block costs about what it costs at the latest", func(t devtest.T) {
				earliestLatency := medianProofLatency(t, follower, recorder, earliest)
				latestLatency := medianProofLatency(t, follower, recorder, head.Number)
				bound := max(latencyFactor*latestLatency, latencyFloor)
				t.Logger().Info("eth_getProof median latency", "earliest_block", earliest, "earliest", earliestLatency,
					"latest_block", head.Number, "latest", latestLatency, "bound", bound)
				t.Require().LessOrEqualf(earliestLatency, bound,
					"eth_getProof at the earliest block %d took %s, more than %s: latency must not grow with the depth in the window (latest block %d took %s)",
					earliest, earliestLatency, bound, head.Number, latestLatency)
			})

			t.Run("debug_executePayload builds on the earliest block", func(t devtest.T) {
				parent, err := follower.EthClient().InfoByNumber(t.Ctx(), earliest)
				t.Require().NoErrorf(err, "must fetch the earliest block %d", earliest)
				witness, err := follower.EthClient().PayloadExecutionWitness(t.Ctx(), parent.Hash(), utils.NextPayloadAttributes(parent, w.signedTransferFromCarol(t)))
				t.Require().NoErrorf(err, "debug_executePayload on the earliest block %d must succeed", earliest)
				requireWitnessNonEmpty(t, *witness, earliest+1)
			})

			t.Run("debug_executionWitness serves the block after the earliest but not the earliest itself", func(t devtest.T) {
				// A witness executes the block on its parent's state, which for the earliest block
				// is the state pruning removed.
				_, err := executionWitness(t, follower, earliest)
				t.Require().Errorf(err, "debug_executionWitness for the earliest block %d needs the pruned state of block %d", earliest, earliest-1)
				t.Logger().Info("debug_executionWitness refused the earliest block", "block", earliest, "err", err)
				witness, err := executionWitness(t, follower, earliest+1)
				t.Require().NoErrorf(err, "debug_executionWitness for block %d, whose parent is the earliest, must succeed", earliest+1)
				requireWitnessNonEmpty(t, witness, earliest+1)
			})

			t.Run("more concurrent witness requests than tracing permits all succeed", func(t devtest.T) {
				var requests errgroup.Group
				for i := range concurrentWitnessRequests {
					block := earliest + 1 + uint64(i)
					requests.Go(func() error {
						witness, err := executionWitness(t, follower, block)
						if err != nil {
							return fmt.Errorf("debug_executionWitness for block %d: %w", block, err)
						}
						if len(witness.State) == 0 {
							return fmt.Errorf("debug_executionWitness for block %d returned no state", block)
						}
						return nil
					})
				}
				t.Require().NoErrorf(requests.Wait(), "%d concurrent witness requests above the %d tracing permits must all succeed", concurrentWitnessRequests, tracingPermits)
			})

			t.Run("height the store has not flushed is refused until debug_proofsSyncStatus reports it", func(t devtest.T) {
				// Queue a recorder write before sequencing resumes, so the first blocks after the
				// restart change the recorder's storage and a proof of the old head would not
				// verify against them.
				pending := txplan.NewPlannedTx(w.bob.Plan(), txplan.WithTo(&w.recorder))
				_, err := pending.Submitted.Eval(t.Ctx())
				t.Require().NoError(err, "the recorder call must be submitted")
				sys.L2CLSequencer.StartSequencer()
				next := head.Number + 1
				follower.WaitForBlockNumber(next)

				statusBefore, err := follower.ProofsSyncStatus()
				t.Require().NoError(err, "debug_proofsSyncStatus must answer before the request")
				proof, proofErr := normalizedProofAt(t, follower, recorder, blockNumberTag(next))
				statusAfter, err := follower.ProofsSyncStatus()
				t.Require().NoError(err, "debug_proofsSyncStatus must answer after the request")
				t.Logger().Info("eth_getProof at a height just imported", "block", next, "refused", proofErr != nil,
					"store_before", statusBefore, "store_after", statusAfter)
				if proofErr != nil {
					requireProofRefused(t, proofErr, next, fmt.Sprintf("at block %d before the store flushed it", next))
					t.Require().Lessf(*statusBefore.Latest, next, "the store reported block %d and then refused it: %v", next, proofErr)
				} else {
					t.Require().GreaterOrEqualf(*statusAfter.Latest, next, "the follower served block %d before its store reported it; window %s", next, statusAfter)
					requireProofVerifies(t, follower, proof, next)
				}
				includedBlock(t, pending)
				follower.VerifyProofsMatchAt(reference, next, w.accounts()...)
				sys.L2CLSequencer.StopSequencer()
			})

			t.Run("hash of a reorged-out block is refused and the replacement is served", func(t devtest.T) {
				// Blocks that reach L1 are derived back as they were, so the batcher stays off
				// until the replacement block exists and is the one it submits.
				sys.L2Batcher.Stop()
				forkPoint := reference.BlockRefByLabel(eth.Unsafe)
				replaced := sequenceBlockOn(t, sys, forkPoint, utils.SignedTransfer(t, w.alice, w.bob.Address(), eth.OneHundredthEther))
				follower.Reached(eth.Unsafe, replaced.Number, blockAttempts)
				t.Require().Equalf(replaced.Hash, follower.BlockRefByNumber(replaced.Number).Hash, "the follower must import the block to be replaced %s", replaced)
				follower.WaitForProofsStoreBlock(replaced.Number, storeCatchUpTimeout)

				replacement := sequenceBlockOn(t, sys, forkPoint, utils.SignedTransfer(t, w.bob, w.alice.Address(), eth.OneHundredthEther))
				t.Require().NotEqualf(replaced.Hash, replacement.Hash, "the replacement block must differ from the replaced block at height %d", replacement.Number)
				sys.L2Batcher.Start()
				sys.L2CLSequencer.MatchedUnsafe(reference, syncAttempts)
				sys.L2CLSequencer.StartSequencer()
				follower.ReorgExact(replaced, reorgAttempts)
				follower.InSync(reference, safety.LocalUnsafe, syncAttempts)

				_, err := normalizedProofAt(t, follower, dsl.ProofAccount{Address: w.alice.Address()}, replaced.Hash.String())
				t.Require().Errorf(err, "eth_getProof by hash of the reorged-out block %s must be refused", replaced)
				t.Logger().Info("eth_getProof refused the reorged-out block", "block", replaced, "err", err)

				// The store already holds the replaced block at this height, so the window reports
				// the height before the ExEx has processed the reorg; a proof requested by the
				// replacement's hash in that gap carries the replaced block's state. Record whether
				// the gap was hit, then wait for the store to index blocks above the replacement,
				// which it only does after the unwind.
				staleProof, staleErr := normalizedProofAt(t, follower, recorder, replacement.Hash.String())
				staleHeader, err := follower.EthClient().InfoByHash(t.Ctx(), replacement.Hash)
				t.Require().NoErrorf(err, "must fetch the replacement block %s", replacement)
				t.Logger().Info("eth_getProof by hash of the replacement right after the reorg", "block", replacement,
					"refused", staleErr != nil, "stale", staleErr == nil && dsl.VerifyAccountProof(staleProof, staleHeader.Root()) != nil)
				settled := w.driveUntilBlock(replacement.Number + 2)
				follower.WaitForProofsStoreBlock(settled, storeCatchUpTimeout)
				requireServedAndMatching(t, follower, reference, w, replacement.Number)
			})
		})
	}
}
