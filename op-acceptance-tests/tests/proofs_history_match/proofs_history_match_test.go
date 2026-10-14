package proofs_history_match

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
)

const (
	// referenceEthProofWindow lets the reference node answer eth_getProof for every block of the
	// run from reth's own historical state; reth's default window of zero rejects them all.
	referenceEthProofWindow = 1_000_000
	workloadRounds          = 4
)

// proofsHistoryCase is one cell of the matrix the proofs follower runs under.
type proofsHistoryCase struct {
	persistenceThreshold uint64
	stateMaskingBlocks   uint64
	verificationInterval uint64
}

func (c proofsHistoryCase) name() string {
	proofsPath := "index"
	if c.verificationInterval != 0 {
		proofsPath = "execute"
	}
	return fmt.Sprintf("threshold_%d-masking_%d-%s", c.persistenceThreshold, c.stateMaskingBlocks, proofsPath)
}

func (c proofsHistoryCase) opRethArgs() []string {
	args := []string{
		fmt.Sprintf("--engine.persistence-threshold=%d", c.persistenceThreshold),
		fmt.Sprintf("--proofs-history.verification-interval=%d", c.verificationInterval),
	}
	if c.stateMaskingBlocks != 0 {
		args = append(args, fmt.Sprintf("--engine.num-state-masking-blocks=%d", c.stateMaskingBlocks))
	}
	return args
}

func proofsHistoryMatrix() []proofsHistoryCase {
	var cases []proofsHistoryCase
	for _, persistenceThreshold := range []uint64{7, 50} {
		for _, stateMaskingBlocks := range []uint64{0, 30} {
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
	FunderL2       *dsl.FunderEOA
	ProofsFollower *dsl.L2ELNode
	Reference      *dsl.L2ELNode
}

// newProofsHistorySystem starts a sequencer without proofs history, which serves as the
// independent eth_getProof reference, and a follower running proofs history under the case's
// reth and ExEx settings.
func newProofsHistorySystem(t devtest.T, c proofsHistoryCase) *proofsHistorySystem {
	sysgo.SkipOnOpGeth(t, "proofs history is an op-reth feature")
	clKind := sysgo.ResolveMixedL2CLKind()

	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
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
				},
			},
			{
				CLKey:      "follower",
				CLKind:     clKind,
				ELKey:      "proofs-follower",
				ELKind:     sysgo.MixedL2ELOpReth,
				OpRethOpts: []sysgo.OpRethOption{sysgo.OpRethWithExtraArgs(c.opRethArgs()...)},
			},
		},
	})
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	t.Require().Len(frontends.Nodes, 2, "expected the sequencer and the proofs follower")

	var follower *dsl.L2ELNode
	for _, node := range frontends.Nodes {
		if !node.Spec.IsSequencer {
			follower = node.EL
		}
	}
	t.Require().NotNil(follower, "missing the proofs follower")

	return &proofsHistorySystem{
		FunderL2:       frontends.FunderL2,
		ProofsFollower: follower,
		Reference:      frontends.L2Network.PrimaryEL(),
	}
}

// deployBlockRecorder deploys a contract that writes the block number to slot 0 and the block
// timestamp to slot 1 on every call, so every call changes its storage.
func deployBlockRecorder(t devtest.T, deployer *dsl.EOA) common.Address {
	runtimeCode := program.New().
		Op(vm.NUMBER).Push(0).Op(vm.SSTORE).
		Op(vm.TIMESTAMP).Push(1).Op(vm.SSTORE).
		Bytes()
	initCode := program.New().ReturnViaCodeCopy(runtimeCode).Bytes()

	tx := deployer.Transact(deployer.Plan(), txplan.WithData(initCode))
	receipt, err := tx.Included.Eval(t.Ctx())
	t.Require().NoError(err, "block recorder deployment must be included")
	return receipt.ContractAddress
}

// TestProofsHistoryFollowerServesMatchingHistoricalProofs runs transfers and contract writes
// against a sequencer that serves eth_getProof from its own historical state, then sweeps every
// block of a proofs-history follower's window: each of the follower's proofs must verify against
// the block header and equal the sequencer's.
func TestProofsHistoryFollowerServesMatchingHistoricalProofs(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range proofsHistoryMatrix() {
		t.Run(c.name(), func(t devtest.T) {
			t.Parallel()
			if c.stateMaskingBlocks != 0 {
				t.Skipf("state masking is compiled out of the reth 2.5.2 on develop and #23046 rejects a nonzero --engine.num-state-masking-blocks; re-enable with ethereum-optimism/protocol-team#457")
			}
			sys := newProofsHistorySystem(t, c)

			alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
			bob := sys.FunderL2.NewFundedEOA(eth.OneEther)
			recorder := deployBlockRecorder(t, alice)
			for range workloadRounds {
				alice.Transfer(bob.Address(), eth.OneHundredthEther)
				bob.Transfer(alice.Address(), eth.OneHundredthEther)
				alice.Transact(alice.Plan(), txplan.WithTo(&recorder))
			}

			sys.ProofsFollower.VerifyHistoricalProofsMatch(sys.Reference, func(opts *dsl.HistoricalProofsOpts) {
				opts.Accounts = []dsl.ProofAccount{
					{Address: alice.Address()},
					{Address: bob.Address()},
					{Address: recorder, StorageKeys: []common.Hash{common.HexToHash("0x0"), common.HexToHash("0x1")}},
				}
			})
		})
	}
}
