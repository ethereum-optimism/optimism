package zk

import (
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum/go-ethereum/common"
)

// The live proposer keeps creating games while these tests run (including
// during time travel), so assertions only reference the specific games
// returned by WaitForZKGameCount and never assume a total game count.

func TestProposerCreatesRootZKGame(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newProposerSystem(t)
	factory := sys.DisputeGameFactory()
	_, anchorSequence := sys.AnchorStateRegistry(sys.L2ChainA).AnchorRoot()

	game0 := factory.WaitForZKGameCount(1)

	t.Require().Equal(uint32(math.MaxUint32), game0.ParentIndex(),
		"first proposer game must be a root game using the max-uint32 parent sentinel")
	t.Require().Greater(game0.L2SequenceNumber(), anchorSequence,
		"root game must propose a sequence number beyond the anchor")
}

func TestProposerChainsSecondZKGameOnFirst(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newProposerSystem(t)
	factory := sys.DisputeGameFactory()

	game0 := factory.WaitForZKGameCount(1)
	game1 := factory.WaitForZKGameCount(2)

	t.Require().Equal(uint32(0), game1.ParentIndex(),
		"second proposer game must chain on the first game at factory index 0")
	t.Require().Greater(game1.L2SequenceNumber(), game0.L2SequenceNumber(),
		"second game must propose a sequence number beyond its parent")
}

func TestProposerResolvesOwnUnchallengedGame(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newProposerSystem(t)
	factory := sys.DisputeGameFactory()

	game0 := factory.WaitForZKGameCount(1)
	advanceL1To(&sys.SingleChainInterop, game0.ClaimData().Deadline+1)

	// The proposer's resolution task must resolve its own unchallenged game;
	// the test never calls resolve itself.
	game0.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
}

func TestProposerClaimsBondAfterResolution(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newProposerSystem(t)
	factory := sys.DisputeGameFactory()
	proposerAddr := zkProposerAddress(t, sys)
	weth := factory.DelayedWETH(factory.ZKGameImpl().Args.Weth)

	game0 := factory.WaitForZKGameCount(1)
	advanceL1To(&sys.SingleChainInterop, game0.ClaimData().Deadline+1)
	game0.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
	advanceL1To(&sys.SingleChainInterop, game0.ResolvedAt()+uint64(zkFinalityDelay/time.Second)+1)

	// Phase 1 (unlock): the proposer's claim task closes the game and unlocks
	// its bond credit into DelayedWETH.
	var withdrawal proofs.ZKWithdrawal
	t.Require().Eventuallyf(func() bool {
		withdrawal = weth.Withdrawal(game0.Address, proposerAddr)
		return withdrawal.Amount.Sign() > 0
	}, 2*time.Minute, time.Second, "proposer did not unlock its bond credit")

	// Phase 2 (payout): only possible once the WETH withdrawal delay has
	// elapsed in L1 time. The payout transfer itself is enforced by
	// DelayedWETH.withdraw, so "withdrawal fully drained and credit zeroed"
	// is the deterministic claim-completion observable; a raw balance-growth
	// check would race the live proposer bonding new games in this window.
	advanceL1To(&sys.SingleChainInterop, uint64(new(big.Int).Add(withdrawal.Timestamp, weth.Delay()).Int64())+1)

	t.Require().Eventuallyf(func() bool {
		return weth.Withdrawal(game0.Address, proposerAddr).Amount.Sign() == 0 &&
			game0.Credit(proposerAddr).IsZero()
	}, 2*time.Minute, time.Second, "proposer did not claim its bond after the withdrawal delay")

	t.Require().True(game0.Credit(proposerAddr).IsZero(), "claimed game must hold no credit for the proposer")
	t.Require().Zero(weth.Withdrawal(game0.Address, proposerAddr).Amount.Sign(), "claimed game must hold no pending withdrawal")
}

// zkProposerAddress derives the address the sysgo ZK proposer signs with: the
// ProposerRole dev key for the proof chain (chain A).
func zkProposerAddress(t devtest.T, sys *presets.SimpleInterop) common.Address {
	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	t.Require().NoError(err, "derive dev keys")
	addr, err := keys.Address(devkeys.ProposerRole.Key(sys.L2ChainA.ChainID().ToBig()))
	t.Require().NoError(err, "derive proposer address")
	return addr
}
