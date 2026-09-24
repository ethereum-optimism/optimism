package zk

import (
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// TestPrivateProjectionHonestProposerLifecycleMockVerifier is an honest-proposer
// lifecycle test of the ZK dispute game over a super root that includes a private
// chain's PUBLIC PROJECTION. It covers only the projection -> super root layer.
//
// What it exercises: the native SP1 range and consolidation cores replay the public
// projection (a user deposit distinguishes private execution from inert projection
// execution), the honest proposer defends a challenged correct root, and a corrupted
// projection root is rejected.
//
// What it does NOT establish: the ZK dispute game here uses the on-chain
// MockSP1Verifier, so the submitted proof bytes are not cryptographic, and nothing in
// this test proves private execution. The private -> projection layer is checked at
// span admission (op-private-interop/docs/BATCHES.md); here the pair runs the legacy,
// test-only execution-mock-v1 verifier.
func TestPrivateProjectionHonestProposerLifecycleMockVerifier(gt *testing.T) {
	t := devtest.SerialT(gt)
	command, err := (rustbin.Spec{SrcDir: "rust", Package: "kona-sp1-super-range-executor", Binary: "kona-sp1-private-projection-executor"}).EnsureExists(t.Ctx(), t.Logger())
	t.Require().NoError(err)
	pair := presets.NewPrivateInteropProofs(t, presets.WithZK(), presets.WithoutHonestProposer(),
		presets.WithPrivateInteropChain(sysgo.WithPrivateInteropExecutionMock(command)))
	sys := pair.SimpleInterop
	alice := sys.FunderL1.NewFundedEOA(eth.OneEther)
	bridge := dsl.NewStandardBridge(t, pair.Private.L2B, sys.L1EL)
	bridge.Deposit(eth.OneTenthEther, alice)
	alice.AsEL(pair.Private.L2ELB).VerifyBalanceExact(eth.OneTenthEther)
	// Wait until the super root includes the deposit on the public projection.
	privateHead := pair.Private.L2ELB.BlockRefByLabel(eth.Unsafe)
	sys.L2ELB.WaitL1OriginReached(eth.Safe, privateHead.L1Origin.Number, 180)
	alice.AsEL(sys.L2ELB).VerifyBalanceExact(eth.ZeroWei)
	factory := sys.DisputeGameFactory()
	creator, challenger := fundedActors(sys)
	sequence, _ := factory.WaitForSafeSuperRootAfter(privateHead.Time)
	game := factory.StartZKGame(creator, proofs.WithL2SequenceNumber(sequence))
	game.Challenge(challenger)
	sys.StartZKProposer()
	game.WaitForProposalStatus(proofs.ZKProposalChallengedAndValidProofProvided)
	t.Require().Equal(zkProposerAddress(t, sys), game.ClaimData().Prover)
	game.WaitForGameStatus(gameTypes.GameStatusDefenderWon)

	// Corrupt chain B's PUBLIC projection root in the expanded super root. The
	// challenger must reject it; a correct private root is not a substitute.
	sequence, outputs := factory.WaitForSafeSuperRootAfter(sequence)
	t.Require().Len(outputs, 2)
	t.Require().Less(sys.L2ChainA.ChainID().ToBig().Cmp(sys.L2ChainB.ChainID().ToBig()), 0,
		"the projection must be the second chain in the sorted super root")
	outputs[1][0] ^= 0xff
	invalid := factory.StartZKGame(creator, proofs.WithL2SequenceNumber(sequence), proofs.WithSuperRootFrom(outputs...))
	invalid.WaitForProposalStatus(proofs.ZKProposalChallenged)
	advanceL1To(&sys.SingleChainInterop, invalid.ClaimData().Deadline+1)
	invalid.WaitForGameStatus(gameTypes.GameStatusChallengerWon)
	t.Require().Equal(common.Address{}, invalid.ClaimData().Prover)
}
