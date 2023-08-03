package disputegame

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-service/client/utils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

const faultGameType uint8 = 0
const alphabetGameDepth = 4
const lastAlphabetTraceIndex = 1<<alphabetGameDepth - 1

type Status uint8

const (
	StatusInProgress Status = iota
	StatusChallengerWins
	StatusDefenderWins
)

var alphaExtraData = common.Hex2Bytes("1000000000000000000000000000000000000000000000000000000000000000")
var alphabetVMAbsolutePrestate = common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000060")
var alphabetVMAbsolutePrestateClaim = crypto.Keccak256Hash(alphabetVMAbsolutePrestate)
var CorrectAlphabet = "abcdefghijklmnop"

type FactoryHelper struct {
	t       *testing.T
	require *require.Assertions
	client  *ethclient.Client
	opts    *bind.TransactOpts
	factory *bindings.DisputeGameFactory
}

func NewFactoryHelper(t *testing.T, ctx context.Context, client *ethclient.Client, gameDuration uint64) *FactoryHelper {
	require := require.New(t)
	chainID, err := client.ChainID(ctx)
	require.NoError(err)
	opts, err := bind.NewKeyedTransactorWithChainID(deployer.TestKey, chainID)
	require.NoError(err)

	factory := deployDisputeGameContracts(require, ctx, client, opts, gameDuration)

	return &FactoryHelper{
		t:       t,
		require: require,
		client:  client,
		opts:    opts,
		factory: factory,
	}
}

func (h *FactoryHelper) StartAlphabetGame(ctx context.Context, claimedAlphabet string) *FaultGameHelper {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	trace := alphabet.NewTraceProvider(claimedAlphabet, 4)
	rootClaim, err := trace.Get(ctx, lastAlphabetTraceIndex)
	h.require.NoError(err, "get root claim")
	tx, err := h.factory.Create(h.opts, faultGameType, rootClaim, alphaExtraData)
	h.require.NoError(err, "create fault dispute game")
	rcpt, err := utils.WaitReceiptOK(ctx, h.client, tx.Hash())
	h.require.NoError(err, "wait for create fault dispute game receipt to be OK")
	h.require.Len(rcpt.Logs, 1, "should have emitted a single DisputeGameCreated event")
	createdEvent, err := h.factory.ParseDisputeGameCreated(*rcpt.Logs[0])
	h.require.NoError(err)
	game, err := bindings.NewFaultDisputeGame(createdEvent.DisputeProxy, h.client)
	h.require.NoError(err)
	return &FaultGameHelper{
		t:               h.t,
		require:         h.require,
		client:          h.client,
		opts:            h.opts,
		game:            game,
		maxDepth:        alphabetGameDepth,
		addr:            createdEvent.DisputeProxy,
		claimedAlphabet: claimedAlphabet,
	}
}

type FaultGameHelper struct {
	t               *testing.T
	require         *require.Assertions
	client          *ethclient.Client
	opts            *bind.TransactOpts
	game            *bindings.FaultDisputeGame
	maxDepth        int
	addr            common.Address
	claimedAlphabet string
}

func (g *FaultGameHelper) StartChallenger(ctx context.Context, l1Endpoint string, name string, options ...challenger.Option) *challenger.Helper {
	opts := []challenger.Option{
		func(c *config.Config) {
			c.GameAddress = g.addr
			c.GameDepth = alphabetGameDepth
			c.TraceType = config.TraceTypeAlphabet
			// By default the challenger agrees with the root claim (thus disagrees with the proposed output)
			// This can be overridden by passing in options
			c.AlphabetTrace = g.claimedAlphabet
			c.AgreeWithProposedOutput = false
		},
	}
	opts = append(opts, options...)
	c := challenger.NewChallenger(g.t, ctx, l1Endpoint, name, opts...)
	g.t.Cleanup(func() {
		_ = c.Close()
	})
	return c
}

func (g *FaultGameHelper) WaitForClaimCount(ctx context.Context, count int64) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	err := utils.WaitFor(ctx, 1*time.Second, func() (bool, error) {
		actual, err := g.game.ClaimDataLen(&bind.CallOpts{Context: ctx})
		if err != nil {
			return false, err
		}
		g.t.Log("Waiting for claim count", "current", actual, "expected", count, "game", g.addr)
		return actual.Cmp(big.NewInt(count)) == 0, nil
	})
	g.require.NoError(err)
}

type ContractClaim struct {
	ParentIndex uint32
	Countered   bool
	Claim       [32]byte
	Position    *big.Int
	Clock       *big.Int
}

func (g *FaultGameHelper) WaitForClaim(ctx context.Context, predicate func(claim ContractClaim) bool) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	err := utils.WaitFor(ctx, 1*time.Second, func() (bool, error) {
		count, err := g.game.ClaimDataLen(&bind.CallOpts{Context: ctx})
		if err != nil {
			return false, fmt.Errorf("retrieve number of claims: %w", err)
		}
		// Search backwards because the new claims are at the end and more likely the ones we want.
		for i := count.Int64() - 1; i >= 0; i-- {
			claimData, err := g.game.ClaimData(&bind.CallOpts{Context: ctx}, big.NewInt(i))
			if err != nil {
				return false, fmt.Errorf("retrieve claim %v: %w", i, err)
			}
			if predicate(claimData) {
				return true, nil
			}
		}
		return false, nil
	})
	g.require.NoError(err)
}

func (g *FaultGameHelper) WaitForClaimAtMaxDepth(ctx context.Context, countered bool) {
	g.WaitForClaim(ctx, func(claim ContractClaim) bool {
		pos := types.NewPositionFromGIndex(claim.Position.Uint64())
		return pos.Depth() == g.maxDepth && claim.Countered == countered
	})
}

func (g *FaultGameHelper) Resolve(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	tx, err := g.game.Resolve(g.opts)
	g.require.NoError(err)
	_, err = utils.WaitReceiptOK(ctx, g.client, tx.Hash())
	g.require.NoError(err)
}

func (g *FaultGameHelper) WaitForGameStatus(ctx context.Context, expected Status) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	err := utils.WaitFor(ctx, 1*time.Second, func() (bool, error) {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		status, err := g.game.Status(&bind.CallOpts{Context: ctx})
		if err != nil {
			return false, fmt.Errorf("game status unavailable: %w", err)
		}

		return expected == Status(status), nil
	})
	g.require.NoError(err, "wait for game status")
}
