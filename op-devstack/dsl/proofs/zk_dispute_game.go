package proofs

import (
	"math/big"

	challengerContracts "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/gameargs"
	challengerTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// ZKDisputeGame holds the impl address and parsed args for a deployed ZK dispute game.
type ZKDisputeGame struct {
	Address common.Address
	Args    gameargs.ZKGameArgs
}

type ZKProposalStatus = challengerContracts.ProposalStatus

const (
	ZKProposalUnchallenged                      = challengerContracts.ProposalStatusUnchallenged
	ZKProposalChallenged                        = challengerContracts.ProposalStatusChallenged
	ZKProposalUnchallengedAndValidProofProvided = challengerContracts.ProposalStatusUnchallengedAndValidProofProvided
	ZKProposalChallengedAndValidProofProvided   = challengerContracts.ProposalStatusChallengedAndValidProofProvided
	ZKProposalResolved                          = challengerContracts.ProposalStatusResolved
)

type ZKClaimData struct {
	ParentIndex uint32
	Status      uint8
	Challenger  common.Address
	Prover      common.Address
	Deadline    uint64
	Claim       common.Hash
}

type zkDisputeGameBinding struct {
	RootClaim            func() bindings.TypedCall[common.Hash] `sol:"rootClaim"`
	L2SequenceNumber     func() bindings.TypedCall[*big.Int]    `sol:"l2SequenceNumber"`
	ParentIndex          func() bindings.TypedCall[uint32]      `sol:"parentIndex"`
	Status               func() bindings.TypedCall[uint8]       `sol:"status"`
	ClaimData            func() bindings.TypedCall[ZKClaimData] `sol:"claimData"`
	ChallengerBond       func() bindings.TypedCall[*big.Int]    `sol:"challengerBond"`
	GameOver             func() bindings.TypedCall[bool]        `sol:"gameOver"`
	ResolvedAt           func() bindings.TypedCall[uint64]      `sol:"resolvedAt"`
	BondDistributionMode func() bindings.TypedCall[uint8]       `sol:"bondDistributionMode"`
	Challenge            func() bindings.TypedCall[uint8]       `sol:"challenge"`
	Prove                func([]byte) bindings.TypedCall[uint8] `sol:"prove"`
	Resolve              func() bindings.TypedCall[uint8]       `sol:"resolve"`
	CloseGame            func() bindings.TypedCall[any]         `sol:"closeGame"`
}

// ZKGame is a deployed ZK dispute-game instance. It exposes the contract calls
// used to exercise the game lifecycle in acceptance tests.
type ZKGame struct {
	require      *require.Assertions
	contract     *zkDisputeGameBinding
	Address      common.Address
	factoryIndex uint32
}

func newZKGame(t devtest.T, require *require.Assertions, client apis.EthClient, addr common.Address) *ZKGame {
	game := bindings.NewBindings[zkDisputeGameBinding](
		bindings.WithClient(client),
		bindings.WithTo(addr),
		bindings.WithTest(t),
	)
	return &ZKGame{require: require, contract: &game, Address: addr}
}

func (g *ZKGame) withFactoryIndex(index uint32) *ZKGame {
	g.factoryIndex = index
	return g
}

func (g *ZKGame) FactoryIndex() uint32 {
	return g.factoryIndex
}

func (g *ZKGame) RootClaimValue() common.Hash {
	return contract.Read(g.contract.RootClaim())
}

func (g *ZKGame) L2SequenceNumber() uint64 {
	return bigs.Uint64Strict(contract.Read(g.contract.L2SequenceNumber()))
}

func (g *ZKGame) ParentIndex() uint32 {
	return contract.Read(g.contract.ParentIndex())
}

func (g *ZKGame) ClaimData() ZKClaimData {
	return contract.Read(g.contract.ClaimData())
}

func (g *ZKGame) ProposalStatus() ZKProposalStatus {
	return ZKProposalStatus(g.ClaimData().Status)
}

func (g *ZKGame) GameStatus() gameTypes.GameStatus {
	return gameTypes.GameStatus(contract.Read(g.contract.Status()))
}

func (g *ZKGame) ChallengerBond() eth.ETH {
	return eth.WeiBig(contract.Read(g.contract.ChallengerBond()))
}

func (g *ZKGame) GameOver() bool {
	return contract.Read(g.contract.GameOver())
}

func (g *ZKGame) ResolvedAt() uint64 {
	return contract.Read(g.contract.ResolvedAt())
}

func (g *ZKGame) BondDistributionMode() challengerTypes.BondDistributionMode {
	return challengerTypes.BondDistributionMode(contract.Read(g.contract.BondDistributionMode()))
}

func (g *ZKGame) Challenge(challenger *dsl.EOA) ZKClaimData {
	receipt := contract.Write(challenger, g.contract.Challenge(), txplan.WithValue(g.ChallengerBond()), txplan.WithGasRatio(2))
	g.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
	claim := g.ClaimData()
	g.require.Equal(ZKProposalChallenged, ZKProposalStatus(claim.Status))
	return claim
}

func (g *ZKGame) Prove(prover *dsl.EOA, proof []byte) ZKClaimData {
	receipt := contract.Write(prover, g.contract.Prove(proof), txplan.WithGasRatio(2))
	g.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
	return g.ClaimData()
}

func (g *ZKGame) Resolve(eoa *dsl.EOA) gameTypes.GameStatus {
	receipt := contract.Write(eoa, g.contract.Resolve(), txplan.WithGasRatio(2))
	g.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
	status := g.GameStatus()
	g.require.NotEqual(gameTypes.GameStatusInProgress, status)
	g.require.Equal(ZKProposalResolved, g.ProposalStatus())
	return status
}

func (g *ZKGame) Close(eoa *dsl.EOA) {
	receipt := contract.Write(eoa, g.contract.CloseGame(), txplan.WithGasRatio(2))
	g.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
}

// ZKGameImpl returns the ZK dispute game implementation address and its parsed
// constructor args from the DisputeGameFactory.
func (f *DisputeGameFactory) ZKGameImpl() *ZKDisputeGame {
	impl := f.GameImpl(gameTypes.ZKDisputeGameType)
	raw := f.GameArgs(gameTypes.ZKDisputeGameType)
	args, err := gameargs.ParseZK(raw)
	f.require.NoError(err, "failed to parse ZK game args")

	return &ZKDisputeGame{
		Address: impl.Address,
		Args:    args,
	}
}
