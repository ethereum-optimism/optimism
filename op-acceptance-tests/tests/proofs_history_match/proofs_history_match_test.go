package proofs_history_match

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// memoryBlockBufferTarget is reth's --engine.memory-block-buffer-target. It is passed
	// explicitly, like the persistence threshold, so how many blocks reth holds in memory does
	// not depend on its defaults.
	memoryBlockBufferTarget = 5
	workloadRounds          = 4
)

// proofsHistoryCase is one cell of the matrix the proofs follower runs under.
type proofsHistoryCase struct {
	persistenceThreshold uint64
	stateMaskingBlocks   uint64
	verificationInterval uint64
}

func (c proofsHistoryCase) String() string {
	proofsPath := "index"
	if c.verificationInterval != 0 {
		proofsPath = "execute"
	}
	return fmt.Sprintf("threshold_%d-masking_%d-%s", c.persistenceThreshold, c.stateMaskingBlocks, proofsPath)
}

func (c proofsHistoryCase) opRethArgs() []string {
	args := []string{
		fmt.Sprintf("--engine.memory-block-buffer-target=%d", memoryBlockBufferTarget),
		fmt.Sprintf("--engine.persistence-threshold=%d", c.persistenceThreshold),
		fmt.Sprintf("--proofs-history.verification-interval=%d", c.verificationInterval),
	}
	if c.stateMaskingBlocks != 0 {
		args = append(args, fmt.Sprintf("--engine.num-state-masking-blocks=%d", c.stateMaskingBlocks))
	}
	return args
}

// blocksBeforeLastRound is a chain height by which reth has persisted at least once under the
// case's threshold: persistence starts once more than the threshold of blocks sit in memory and
// keeps the buffer target in memory afterwards.
func (c proofsHistoryCase) blocksBeforeLastRound() uint64 {
	return c.persistenceThreshold + 2*memoryBlockBufferTarget
}

func proofsHistoryMatrix() []proofsHistoryCase {
	var cases []proofsHistoryCase
	for _, persistenceThreshold := range []uint64{7, 50} {
		// op-reth rejects a nonzero --engine.num-state-masking-blocks; add a masking depth here
		// once ethereum-optimism/protocol-team#457 lands.
		for _, stateMaskingBlocks := range []uint64{0} {
			for _, verificationInterval := range []uint64{0, 1} {
				cases = append(cases, proofsHistoryCase{
					persistenceThreshold: persistenceThreshold,
					stateMaskingBlocks:   stateMaskingBlocks,
					verificationInterval: verificationInterval,
				})
			}
		}
	}
	return cases
}

type proofsHistorySystem struct {
	Bridge         *dsl.StandardBridge
	FunderL2       *dsl.FunderEOA
	ProofsFollower *dsl.L2ELNode
	Reference      *dsl.L2ELNode
}

// newProofsHistorySystem starts a sequencer without proofs history, which serves as the
// independent eth_getProof reference, and a follower running proofs history under the case's
// reth and ExEx settings.
func newProofsHistorySystem(t devtest.T, c proofsHistoryCase) *proofsHistorySystem {
	sys := presets.NewProofsHistoryFollowerSystem(t, func(cfg *presets.ProofsHistoryFollowerConfig) {
		cfg.FollowerOpRethOpts = []sysgo.OpRethOption{sysgo.OpRethWithExtraArgs(c.opRethArgs()...)}
	})
	return &proofsHistorySystem{
		Bridge:         dsl.NewStandardBridge(t, sys.L2Network, sys.L1EL),
		FunderL2:       sys.FunderL2,
		ProofsFollower: sys.Follower.EL,
		Reference:      sys.Sequencer.EL,
	}
}

// TestProofsHistoryFollowerServesMatchingHistoricalProofs runs transfers, withdrawals and contract
// writes against a sequencer that serves eth_getProof from its own historical state, then sweeps a
// proofs-history follower from its window's first block to the last workload block: each of the
// follower's proofs must verify against the block header and equal the sequencer's, and from
// Isthmus on the L2ToL1MessagePasser storage root must equal the header's withdrawalsRoot.
func TestProofsHistoryFollowerServesMatchingHistoricalProofs(gt *testing.T) {
	t := devtest.ParallelT(gt)
	devtest.RunParallel(t, proofsHistoryMatrix(), func(t devtest.T, c proofsHistoryCase) {
		sys := newProofsHistorySystem(t, c)

		alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
		bob := sys.FunderL2.NewFundedEOA(eth.OneEther)
		recorder := alice.DeployBlockRecorder()
		var lastWorkloadBlock uint64
		for round := range workloadRounds {
			if round == workloadRounds-1 {
				// The last round lands after reth has persisted at least once, so the sweep spans
				// blocks that left reth's memory and blocks that have not.
				target := c.blocksBeforeLastRound()
				sys.ProofsFollower.Reached(eth.Unsafe, target, int(target)+30)
			}
			alice.Transfer(bob.Address(), eth.OneHundredthEther)
			bob.Transfer(alice.Address(), eth.OneHundredthEther)
			sys.Bridge.InitiateWithdrawal(eth.OneHundredthEther, bob)
			lastWorkloadBlock = dsl.IncludedBlockNumber(t, alice.Transact(alice.Plan(), txplan.WithTo(&recorder)))
		}

		sys.ProofsFollower.VerifyHistoricalProofsMatch(sys.Reference, func(opts *dsl.HistoricalProofsOpts) {
			opts.Accounts = []dsl.ProofAccount{
				{Address: alice.Address()},
				{Address: bob.Address()},
				{Address: predeploys.L2ToL1MessagePasserAddr},
				{Address: recorder, StorageKeys: []common.Hash{common.HexToHash("0x0"), common.HexToHash("0x1")}},
			}
			opts.LastBlock = lastWorkloadBlock
		})
	})
}
