package disputegame

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer"
	"github.com/ethereum-optimism/optimism/op-challenger/fault"
	"github.com/ethereum-optimism/optimism/op-service/client/utils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

const faultGameType uint8 = 0
const alphabetGameDepth = 4

type Status uint8

const (
	StatusInProgress Status = iota
	StatusChallengerWins
	StatusDefenderWins
)

var alphaExtraData = common.Hex2Bytes("1000000000000000000000000000000000000000000000000000000000000000")
var alphabetVMAbsolutePrestate = uint256.NewInt(140).Bytes32()

type FactoryHelper struct {
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
		require: require,
		client:  client,
		opts:    opts,
		factory: factory,
	}
}

func (h *FactoryHelper) StartAlphabetGame(ctx context.Context, claimedAlphabet string) *FaultHelper {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	trace := fault.NewAlphabetProvider(claimedAlphabet, 4)
	rootClaim, err := trace.Get(2 ^ alphabetGameDepth - 1)
	h.require.NoError(err)
	tx, err := h.factory.Create(h.opts, faultGameType, rootClaim, alphaExtraData)
	h.require.NoError(err)
	rcpt, err := utils.WaitReceiptOK(ctx, h.client, tx.Hash())
	h.require.NoError(err)
	h.require.Len(rcpt.Logs, 1, "should have emitted a single DisputeGameCreated event")
	createdEvent, err := h.factory.ParseDisputeGameCreated(*rcpt.Logs[0])
	h.require.NoError(err)
	game, err := bindings.NewFaultDisputeGame(createdEvent.DisputeProxy, h.client)
	h.require.NoError(err)
	return &FaultHelper{
		require: h.require,
		client:  h.client,
		opts:    h.opts,
		game:    game,
	}
}

type FaultHelper struct {
	require *require.Assertions
	client  *ethclient.Client
	opts    *bind.TransactOpts
	game    *bindings.FaultDisputeGame
}

func (g *FaultHelper) Resolve(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	tx, err := g.game.Resolve(g.opts)
	g.require.NoError(err)
	_, err = utils.WaitReceiptOK(ctx, g.client, tx.Hash())
	g.require.NoError(err)
}

func (g *FaultHelper) AssertStatusEquals(expected Status) {
	status, err := g.game.Status(&bind.CallOpts{
		From: g.opts.From,
	})
	g.require.NoError(err)
	g.require.Equal(expected, Status(status))
}
