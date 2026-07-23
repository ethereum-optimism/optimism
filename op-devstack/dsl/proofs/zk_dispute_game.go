package proofs

import (
	"context"
	"encoding/binary"
	"math/big"
	"time"

	challengerContracts "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/gameargs"
	challengerTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
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
	RootClaim            func() bindings.TypedCall[common.Hash]            `sol:"rootClaim"`
	L2SequenceNumber     func() bindings.TypedCall[*big.Int]               `sol:"l2SequenceNumber"`
	ParentIndex          func() bindings.TypedCall[uint32]                 `sol:"parentIndex"`
	Status               func() bindings.TypedCall[uint8]                  `sol:"status"`
	ClaimData            func() bindings.TypedCall[ZKClaimData]            `sol:"claimData"`
	ChallengerBond       func() bindings.TypedCall[*big.Int]               `sol:"challengerBond"`
	GameOver             func() bindings.TypedCall[bool]                   `sol:"gameOver"`
	ResolvedAt           func() bindings.TypedCall[uint64]                 `sol:"resolvedAt"`
	BondDistributionMode func() bindings.TypedCall[uint8]                  `sol:"bondDistributionMode"`
	Challenge            func() bindings.TypedCall[uint8]                  `sol:"challenge"`
	Prove                func([]byte) bindings.TypedCall[uint8]            `sol:"prove"`
	Resolve              func() bindings.TypedCall[uint8]                  `sol:"resolve"`
	CloseGame            func() bindings.TypedCall[any]                    `sol:"closeGame"`
	Credit               func(common.Address) bindings.TypedCall[*big.Int] `sol:"credit"`
	ClaimCredit          func(common.Address) bindings.TypedCall[any]      `sol:"claimCredit"`
}

// ZKGame is a deployed ZK dispute-game instance. It exposes the contract calls
// used to exercise the game lifecycle in acceptance tests.
type ZKGame struct {
	t            devtest.T
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
	return &ZKGame{t: t, require: require, contract: &game, Address: addr}
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

func (g *ZKGame) Credit(recipient common.Address) eth.ETH {
	return eth.WeiBig(contract.Read(g.contract.Credit(recipient)))
}

func (g *ZKGame) ClaimCredit(eoa *dsl.EOA, recipient common.Address) {
	receipt := contract.Write(eoa, g.contract.ClaimCredit(recipient), txplan.WithGasRatio(2))
	g.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
}

// WaitForGameStatus polls the game status until it reaches expected or the timeout elapses.
func (g *ZKGame) WaitForGameStatus(expected gameTypes.GameStatus) {
	g.t.Logf("Waiting for zk game %v to have status %v", g.Address, expected)
	timedCtx, cancel := context.WithTimeout(g.t.Ctx(), defaultTimeout)
	defer cancel()

	var actual gameTypes.GameStatus
	var lastReadErr error
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		var raw uint8
		raw, lastReadErr = contractio.Read(g.contract.Status(), timedCtx)
		if lastReadErr != nil {
			g.t.Logf("Zk game %v status unavailable while waiting for %v: %v", g.Address, expected, lastReadErr)
			return false, nil
		}
		actual = gameTypes.GameStatus(raw)
		g.t.Logf("Zk game %v has status %v, waiting for %v", g.Address, actual, expected)
		return actual == expected, nil
	})
	g.require.NoErrorf(err, "zk game %v status mismatch: expected %s, got %s; last read error: %v", g.Address, expected, actual, lastReadErr)
}

// WaitForClaimedCredit waits until the game has distributed bonds (NormalDistributionMode) and the
// recipient has no unclaimed credit remaining, i.e. the honest challenger closed the game and claimed.
func (g *ZKGame) WaitForClaimedCredit(recipient common.Address) {
	g.t.Logf("Waiting for zk game %v to distribute bonds and for %v to claim its credit", g.Address, recipient)
	timedCtx, cancel := context.WithTimeout(g.t.Ctx(), defaultTimeout)
	defer cancel()

	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		mode, err := contractio.Read(g.contract.BondDistributionMode(), timedCtx)
		if err != nil {
			return false, nil
		}
		if challengerTypes.BondDistributionMode(mode) != challengerTypes.NormalDistributionMode {
			g.t.Logf("Zk game %v not yet closed (bond distribution mode %v)", g.Address, mode)
			return false, nil
		}
		credit, err := contractio.Read(g.contract.Credit(recipient), timedCtx)
		if err != nil {
			return false, nil
		}
		g.t.Logf("Zk game %v closed, %v unclaimed credit %v", g.Address, recipient, credit)
		return credit.Sign() == 0, nil
	})
	g.require.NoErrorf(err, "challenger %v did not claim its credit for zk game %v", recipient, g.Address)
}

// WaitForProposalStatus polls the proposal status until it reaches expected or the timeout elapses.
func (g *ZKGame) WaitForProposalStatus(expected ZKProposalStatus) {
	g.t.Logf("Waiting for zk game %v proposal to have status %v", g.Address, expected)
	timedCtx, cancel := context.WithTimeout(g.t.Ctx(), defaultTimeout)
	defer cancel()

	var actual ZKProposalStatus
	var lastReadErr error
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		var claim ZKClaimData
		claim, lastReadErr = contractio.Read(g.contract.ClaimData(), timedCtx)
		if lastReadErr != nil {
			g.t.Logf("Zk game %v proposal status unavailable while waiting for %v: %v", g.Address, expected, lastReadErr)
			return false, nil
		}
		actual = ZKProposalStatus(claim.Status)
		g.t.Logf("Zk game %v proposal has status %v, waiting for %v", g.Address, actual, expected)
		return actual == expected, nil
	})
	g.require.NoErrorf(err, "zk game %v proposal status mismatch: expected %v, got %v; last read error: %v", g.Address, expected, actual, lastReadErr)
}

// ZKGameImpl returns the ZK dispute game implementation address and its parsed
// constructor args from the DisputeGameFactory.
func (f *DisputeGameFactory) ZKGameImpl() *ZKDisputeGame {
	impl := f.GameImpl(gameTypes.ZKDisputeGameType)
	raw := f.GameArgs(gameTypes.ZKDisputeGameType)
	f.require.Len(raw, gameargs.ZKArgsLength, "ZK game args must be exactly %d bytes", gameargs.ZKArgsLength)

	var prestate common.Hash
	copy(prestate[:], raw[0:32])

	var verifier common.Address
	copy(verifier[:], raw[32:52])

	var asr common.Address
	copy(asr[:], raw[100:120])

	var weth common.Address
	copy(weth[:], raw[120:140])

	return &ZKDisputeGame{
		Address: impl.Address,
		Args: gameargs.ZKGameArgs{
			AbsolutePrestate:     prestate,
			Verifier:             verifier,
			MaxChallengeDuration: binary.BigEndian.Uint64(raw[52:60]),
			MaxProveDuration:     binary.BigEndian.Uint64(raw[60:68]),
			ChallengerBond:       new(big.Int).SetBytes(raw[68:100]),
			AnchorStateRegistry:  asr,
			Weth:                 weth,
		},
	}
}
