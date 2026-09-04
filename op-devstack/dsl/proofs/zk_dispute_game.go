package proofs

import (
	"context"
	"fmt"
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

const claimDataReadTimeout = time.Minute

type ZKClaimData struct {
	ParentIndex uint32
	Status      uint8
	Challenger  common.Address
	Prover      common.Address
	Deadline    uint64
	Claim       common.Hash
}

type zkDisputeGameBinding struct {
	L1Head               func() bindings.TypedCall[common.Hash]            `sol:"l1Head"`
	RootClaim            func() bindings.TypedCall[common.Hash]            `sol:"rootClaim"`
	L2SequenceNumber     func() bindings.TypedCall[*big.Int]               `sol:"l2SequenceNumber"`
	ParentIndex          func() bindings.TypedCall[uint32]                 `sol:"parentIndex"`
	Status               func() bindings.TypedCall[uint8]                  `sol:"status"`
	ClaimData            func() bindings.TypedCall[ZKClaimData]            `sol:"claimData"`
	ChallengerBond       func() bindings.TypedCall[*big.Int]               `sol:"challengerBond"`
	TotalBonds           func() bindings.TypedCall[*big.Int]               `sol:"totalBonds"`
	Weth                 func() bindings.TypedCall[common.Address]         `sol:"weth"`
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
	t             devtest.T
	require       *require.Assertions
	ethClient     apis.EthClient
	contract      *zkDisputeGameBinding
	claimDataRead func(context.Context) (ZKClaimData, error)
	Address       common.Address
	factoryIndex  uint32
}

func newZKGame(t devtest.T, require *require.Assertions, client apis.EthClient, addr common.Address) *ZKGame {
	game := bindings.NewBindings[zkDisputeGameBinding](
		bindings.WithClient(client),
		bindings.WithTo(addr),
		bindings.WithTest(t),
	)
	return &ZKGame{t: t, require: require, ethClient: client, contract: &game, Address: addr}
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

func (g *ZKGame) L1Head() common.Hash {
	return contract.Read(g.contract.L1Head())
}

// AwaitRootSourcePastL1Head waits until source has fully processed the L1 snapshot
// pinned by this game.
func (g *ZKGame) AwaitRootSourcePastL1Head(source dsl.SuperRootSource) {
	g.t.Logf("Waiting for the root source to process past the L1 head of ZK game %v", g.Address)
	ctx, cancel := context.WithTimeout(g.t.Ctx(), defaultTimeout)
	defer cancel()

	var target uint64
	var lastReadErr error
	err := wait.For(ctx, time.Second, func() (bool, error) {
		l1Head, err := contractio.Read(g.contract.L1Head(), ctx)
		if err != nil {
			lastReadErr = err
			g.t.Logf("ZK game %v L1 head unavailable: %v", g.Address, err)
			return false, nil
		}
		block, err := g.ethClient.BlockRefByHash(ctx, l1Head)
		if err != nil {
			lastReadErr = err
			g.t.Logf("ZK game %v L1 head block %v unavailable: %v", g.Address, l1Head, err)
			return false, nil
		}
		target = block.Number
		lastReadErr = nil
		return true, nil
	})
	g.require.NoErrorf(err, "could not resolve the L1 head of ZK game %v; last read error: %v", g.Address, lastReadErr)
	source.AwaitFullyProcessedL1(target)
}

func (g *ZKGame) L2SequenceNumber() uint64 {
	return bigs.Uint64Strict(contract.Read(g.contract.L2SequenceNumber()))
}

func (g *ZKGame) ParentIndex() uint32 {
	return contract.Read(g.contract.ParentIndex())
}

func (g *ZKGame) ClaimData() ZKClaimData {
	claim, err := g.readClaimData(g.t.Ctx())
	g.require.NoError(err, "read ZK game claim data")
	return claim
}

func (g *ZKGame) readClaimData(ctx context.Context) (ZKClaimData, error) {
	readCtx, cancel := context.WithTimeout(ctx, claimDataReadTimeout)
	defer cancel()
	if g.claimDataRead != nil {
		return g.claimDataRead(readCtx)
	}
	return contractio.Read(g.contract.ClaimData(), readCtx)
}

// VerifyUnproven checks that a challenged game has no accepted proof.
func (g *ZKGame) VerifyUnproven() {
	claim, err := g.verifyUnproven(g.t.Ctx())
	g.require.NoErrorf(err, "ZK game %v must remain challenged and unproven; last observation: %+v", g.Address, claim)
}

func (g *ZKGame) verifyUnproven(ctx context.Context) (ZKClaimData, error) {
	claim, err := g.readClaimData(ctx)
	if err != nil {
		return claim, fmt.Errorf("read claim data: %w", err)
	}
	return claim, challengedUnproven(claim)
}

func challengedUnproven(claim ZKClaimData) error {
	status := ZKProposalStatus(claim.Status)
	if status != ZKProposalChallenged {
		return fmt.Errorf("expected proposal status %v but observed %v", ZKProposalChallenged, status)
	}
	if claim.Prover != (common.Address{}) {
		return fmt.Errorf("expected no prover but observed %v", claim.Prover)
	}
	return nil
}

// VerifyUnprovenFor checks that a challenged game remains unproven for window.
func (g *ZKGame) VerifyUnprovenFor(window time.Duration) {
	err := g.verifyUnprovenFor(g.t.Ctx(), window)
	g.require.NoErrorf(err, "ZK game %v must remain challenged and unproven for %v", g.Address, window)
}

func (g *ZKGame) verifyUnprovenFor(ctx context.Context, window time.Duration) error {
	if window <= 0 {
		return fmt.Errorf("unproven verification window must be positive")
	}

	boundary := time.Now().Add(window)
	readCtx, cancel := context.WithDeadline(ctx, boundary.Add(defaultTimeout))
	defer cancel()

	var lastClaim *ZKClaimData
	var lastReadErr error
	for {
		readStartedAt := time.Now()
		claim, err := g.readClaimData(readCtx)
		validReadCrossedBoundary := false
		if err != nil {
			lastReadErr = err
			g.t.Logf("ZK game %v claim data unavailable while verifying it remains unproven: %v", g.Address, err)
		} else {
			lastReadErr = nil
			lastClaim = &claim
			g.t.Logf("ZK game %v proposal status %v and prover %v while verifying it remains unproven", g.Address, ZKProposalStatus(claim.Status), claim.Prover)
			if ctx.Err() != nil {
				return claimReadFailure("parent context ended", ctx.Err(), lastClaim, lastReadErr)
			}
			if err := challengedUnproven(claim); err != nil {
				return fmt.Errorf("ZK game changed before the unproven verification boundary: %w", err)
			}
			if !readStartedAt.Before(boundary) {
				return nil
			}
			validReadCrossedBoundary = !time.Now().Before(boundary)
		}

		if ctx.Err() != nil {
			return claimReadFailure("parent context ended", ctx.Err(), lastClaim, lastReadErr)
		}
		if readCtx.Err() != nil {
			return claimReadFailure("claim data remained unavailable after the verification boundary", readCtx.Err(), lastClaim, lastReadErr)
		}
		if validReadCrossedBoundary {
			continue
		}

		delay := time.Second
		if untilBoundary := time.Until(boundary); untilBoundary > 0 && untilBoundary < delay {
			delay = untilBoundary
		}
		timer := time.NewTimer(delay)
		select {
		case <-readCtx.Done():
			timer.Stop()
			if ctx.Err() != nil {
				return claimReadFailure("parent context ended", ctx.Err(), lastClaim, lastReadErr)
			}
			return claimReadFailure("claim data remained unavailable after the verification boundary", readCtx.Err(), lastClaim, lastReadErr)
		case <-timer.C:
		}
	}
}

func claimReadFailure(reason string, cause error, lastClaim *ZKClaimData, lastReadErr error) error {
	observation := "none"
	if lastClaim != nil {
		observation = fmt.Sprintf("status=%v prover=%v", ZKProposalStatus(lastClaim.Status), lastClaim.Prover)
	}
	if lastReadErr == nil {
		return fmt.Errorf("%s: %w; last observation: %s; last read error: none", reason, cause, observation)
	}
	return fmt.Errorf("%s: %w; last observation: %s; last read error: %w", reason, cause, observation, lastReadErr)
}

// ProvenByFn waits for the game's persistent prover address to equal expected.
func (g *ZKGame) ProvenByFn(ctx context.Context, expected common.Address) dsl.CheckFunc {
	return func() error {
		return g.waitForProvenBy(ctx, expected)
	}
}

func (g *ZKGame) waitForProvenBy(ctx context.Context, expected common.Address) error {
	if expected == (common.Address{}) {
		return fmt.Errorf("expected prover must not be the zero address")
	}

	var lastClaim *ZKClaimData
	var lastReadErr error
	for {
		claim, err := g.readClaimData(ctx)
		if err != nil {
			lastReadErr = err
			g.t.Logf("ZK game %v claim data unavailable while waiting for prover %v: %v", g.Address, expected, err)
		} else {
			lastReadErr = nil
			lastClaim = &claim
			g.t.Logf("ZK game %v has prover %v, waiting for %v", g.Address, claim.Prover, expected)
			if ctx.Err() != nil {
				return claimReadFailure("context ended while waiting for expected prover", ctx.Err(), lastClaim, lastReadErr)
			}
			if claim.Prover == expected {
				return nil
			}
			if claim.Prover != (common.Address{}) {
				return fmt.Errorf("ZK game %v was proven by %v, expected %v", g.Address, claim.Prover, expected)
			}
		}

		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return claimReadFailure("context ended while waiting for expected prover", ctx.Err(), lastClaim, lastReadErr)
		case <-timer.C:
		}
	}
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

func (g *ZKGame) TotalBonds() eth.ETH {
	return eth.WeiBig(contract.Read(g.contract.TotalBonds()))
}

func (g *ZKGame) WETHAddress() common.Address {
	return contract.Read(g.contract.Weth())
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

	var lastReadErr error
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		var mode uint8
		mode, lastReadErr = contractio.Read(g.contract.BondDistributionMode(), timedCtx)
		if lastReadErr != nil {
			g.t.Logf("Zk game %v bond distribution mode unavailable while waiting for %v to claim its credit: %v", g.Address, recipient, lastReadErr)
			return false, nil
		}
		if challengerTypes.BondDistributionMode(mode) != challengerTypes.NormalDistributionMode {
			g.t.Logf("Zk game %v not yet closed (bond distribution mode %v)", g.Address, mode)
			return false, nil
		}
		var credit *big.Int
		credit, lastReadErr = contractio.Read(g.contract.Credit(recipient), timedCtx)
		if lastReadErr != nil {
			g.t.Logf("Zk game %v credit for %v unavailable while waiting to claim: %v", g.Address, recipient, lastReadErr)
			return false, nil
		}
		g.t.Logf("Zk game %v closed, %v unclaimed credit %v", g.Address, recipient, credit)
		return credit.Sign() == 0, nil
	})
	g.require.NoErrorf(err, "challenger %v did not claim its credit for zk game %v; last read error: %v", recipient, g.Address, lastReadErr)
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
		claim, lastReadErr = g.readClaimData(timedCtx)
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
	args, err := gameargs.ParseZK(raw)
	f.require.NoError(err, "failed to parse ZK game args")

	return &ZKDisputeGame{
		Address: impl.Address,
		Args:    args,
	}
}

// ZKWithdrawal mirrors DelayedWETH's WithdrawalRequest struct
// (amount, timestamp), as returned by the public withdrawals mapping getter.
type ZKWithdrawal struct {
	Amount    *big.Int
	Timestamp *big.Int
}

// MaturesAt returns the first L1 timestamp at which this withdrawal can be
// paid out under the given DelayedWETH delay (DelayedWETH.withdraw requires
// `timestamp + delay <= block.timestamp`).
func (w ZKWithdrawal) MaturesAt(delay *big.Int) uint64 {
	return bigs.Uint64Strict(new(big.Int).Add(w.Timestamp, delay))
}

type delayedWETHBinding struct {
	Withdrawals func(common.Address, common.Address) bindings.TypedCall[ZKWithdrawal] `sol:"withdrawals"`
	Delay       func() bindings.TypedCall[*big.Int]                                   `sol:"delay"`
}

// DelayedWETH exposes the minimal DelayedWETH read surface needed to observe
// two-phase bond claiming (unlock, then withdraw after the delay).
type DelayedWETH struct {
	contract *delayedWETHBinding
}

// DelayedWETH binds the DelayedWETH contract at addr (e.g. the ZK game
// implementation's weth arg) using the factory's L1 client.
func (f *DisputeGameFactory) DelayedWETH(addr common.Address) *DelayedWETH {
	weth := bindings.NewBindings[delayedWETHBinding](
		bindings.WithClient(f.ethClient),
		bindings.WithTo(addr),
		bindings.WithTest(f.t),
	)
	return &DelayedWETH{contract: &weth}
}

// Withdrawal returns the pending withdrawal request the game holds for recipient.
func (w *DelayedWETH) Withdrawal(game, recipient common.Address) ZKWithdrawal {
	return contract.Read(w.contract.Withdrawals(game, recipient))
}

// Delay returns the withdrawal delay in seconds.
func (w *DelayedWETH) Delay() *big.Int {
	return contract.Read(w.contract.Delay())
}
