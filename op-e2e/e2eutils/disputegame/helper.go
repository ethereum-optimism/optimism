package disputegame

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/outputs"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame/preimage"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/transactions"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

const (
	cannonGameType    uint32 = 0
	alphabetGameType  uint32 = 255
	alphabetGameDepth        = 4
)

type Status uint8

const (
	StatusInProgress Status = iota
	StatusChallengerWins
	StatusDefenderWins
)

func (s Status) String() string {
	switch s {
	case StatusInProgress:
		return "In Progress"
	case StatusChallengerWins:
		return "Challenger Wins"
	case StatusDefenderWins:
		return "Defender Wins"
	default:
		return fmt.Sprintf("Unknown status: %v", int(s))
	}
}

type gameCfg struct {
	allowFuture bool
	allowUnsafe bool
}
type GameOpt interface {
	Apply(cfg *gameCfg)
}
type gameOptFn func(c *gameCfg)

func (g gameOptFn) Apply(cfg *gameCfg) {
	g(cfg)
}

func WithUnsafeProposal() GameOpt {
	return gameOptFn(func(c *gameCfg) {
		c.allowUnsafe = true
	})
}

func WithFutureProposal() GameOpt {
	return gameOptFn(func(c *gameCfg) {
		c.allowFuture = true
	})
}

type DisputeSystem interface {
	L1BeaconEndpoint() string
	NodeEndpoint(name string) string
	NodeClient(name string) *ethclient.Client
	RollupEndpoint(name string) string
	RollupClient(name string) *sources.RollupClient

	L1Deployments() *genesis.L1Deployments
	RollupCfg() *rollup.Config
	L2Genesis() *core.Genesis

	AdvanceTime(time.Duration)
}

type FactoryHelper struct {
	t           *testing.T
	require     *require.Assertions
	system      DisputeSystem
	client      *ethclient.Client
	opts        *bind.TransactOpts
	factoryAddr common.Address
	factory     *bindings.DisputeGameFactory
}

func NewFactoryHelper(t *testing.T, ctx context.Context, system DisputeSystem) *FactoryHelper {
	require := require.New(t)
	client := system.NodeClient("l1")
	chainID, err := client.ChainID(ctx)
	require.NoError(err)
	opts, err := bind.NewKeyedTransactorWithChainID(deployer.TestKey, chainID)
	require.NoError(err)

	l1Deployments := system.L1Deployments()
	factoryAddr := l1Deployments.DisputeGameFactoryProxy
	factory, err := bindings.NewDisputeGameFactory(factoryAddr, client)
	require.NoError(err)

	return &FactoryHelper{
		t:           t,
		require:     require,
		system:      system,
		client:      client,
		opts:        opts,
		factory:     factory,
		factoryAddr: factoryAddr,
	}
}

func (h *FactoryHelper) PreimageHelper(ctx context.Context) *preimage.Helper {
	opts := &bind.CallOpts{Context: ctx}
	gameAddr, err := h.factory.GameImpls(opts, cannonGameType)
	h.require.NoError(err)
	game, err := bindings.NewFaultDisputeGameCaller(gameAddr, h.client)
	h.require.NoError(err)
	vmAddr, err := game.Vm(opts)
	h.require.NoError(err)
	vm, err := bindings.NewMIPSCaller(vmAddr, h.client)
	h.require.NoError(err)
	oracleAddr, err := vm.Oracle(opts)
	h.require.NoError(err)
	return preimage.NewHelper(h.t, h.opts, h.client, oracleAddr)
}

func newGameCfg(opts ...GameOpt) *gameCfg {
	cfg := &gameCfg{}
	for _, opt := range opts {
		opt.Apply(cfg)
	}
	return cfg
}

func (h *FactoryHelper) StartOutputCannonGameWithCorrectRoot(ctx context.Context, l2Node string, l2BlockNumber uint64, opts ...GameOpt) *OutputCannonGameHelper {
	cfg := newGameCfg(opts...)
	h.waitForBlock(l2Node, l2BlockNumber, cfg)
	output, err := h.system.RollupClient(l2Node).OutputAtBlock(ctx, l2BlockNumber)
	h.require.NoErrorf(err, "Failed to get output at block %v", l2BlockNumber)
	return h.StartOutputCannonGame(ctx, l2Node, l2BlockNumber, common.Hash(output.OutputRoot), opts...)
}

func (h *FactoryHelper) StartOutputCannonGame(ctx context.Context, l2Node string, l2BlockNumber uint64, rootClaim common.Hash, opts ...GameOpt) *OutputCannonGameHelper {
	cfg := newGameCfg(opts...)
	logger := testlog.Logger(h.t, log.LevelInfo).New("role", "OutputCannonGameHelper")
	rollupClient := h.system.RollupClient(l2Node)

	extraData := h.createBisectionGameExtraData(l2Node, l2BlockNumber, cfg)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	tx, err := transactions.PadGasEstimate(h.opts, 2, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return h.factory.Create(opts, cannonGameType, rootClaim, extraData)
	})
	h.require.NoError(err, "create fault dispute game")
	rcpt, err := wait.ForReceiptOK(ctx, h.client, tx.Hash())
	h.require.NoError(err, "wait for create fault dispute game receipt to be OK")
	h.require.Len(rcpt.Logs, 2, "should have emitted a single DisputeGameCreated event")
	createdEvent, err := h.factory.ParseDisputeGameCreated(*rcpt.Logs[1])
	h.require.NoError(err)
	game, err := bindings.NewFaultDisputeGame(createdEvent.DisputeProxy, h.client)
	h.require.NoError(err)

	callOpts := &bind.CallOpts{Context: ctx}
	prestateBlock, err := game.StartingBlockNumber(callOpts)
	h.require.NoError(err, "Failed to load starting block number")
	poststateBlock, err := game.L2BlockNumber(callOpts)
	h.require.NoError(err, "Failed to load l2 block number")
	splitDepth, err := game.SplitDepth(callOpts)
	h.require.NoError(err, "Failed to load split depth")
	l1Head := h.getL1Head(ctx, game)

	prestateProvider := outputs.NewPrestateProvider(rollupClient, prestateBlock.Uint64())
	provider := outputs.NewTraceProvider(logger, prestateProvider, rollupClient, l1Head, faultTypes.Depth(splitDepth.Uint64()), prestateBlock.Uint64(), poststateBlock.Uint64())

	return &OutputCannonGameHelper{
		OutputGameHelper: OutputGameHelper{
			t:                     h.t,
			require:               h.require,
			client:                h.client,
			opts:                  h.opts,
			game:                  game,
			factoryAddr:           h.factoryAddr,
			addr:                  createdEvent.DisputeProxy,
			correctOutputProvider: provider,
			system:                h.system,
		},
	}
}

func (h *FactoryHelper) getL1Head(ctx context.Context, game *bindings.FaultDisputeGame) eth.BlockID {
	l1HeadHash, err := game.L1Head(&bind.CallOpts{Context: ctx})
	h.require.NoError(err, "Failed to load L1 head")
	l1Header, err := h.client.HeaderByHash(ctx, l1HeadHash)
	h.require.NoError(err, "Failed to load L1 header")
	l1Head := eth.HeaderBlockID(l1Header)
	return l1Head
}

func (h *FactoryHelper) StartOutputAlphabetGameWithCorrectRoot(ctx context.Context, l2Node string, l2BlockNumber uint64, opts ...GameOpt) *OutputAlphabetGameHelper {
	cfg := newGameCfg(opts...)
	h.waitForBlock(l2Node, l2BlockNumber, cfg)
	output, err := h.system.RollupClient(l2Node).OutputAtBlock(ctx, l2BlockNumber)
	h.require.NoErrorf(err, "Failed to get output at block %v", l2BlockNumber)
	return h.StartOutputAlphabetGame(ctx, l2Node, l2BlockNumber, common.Hash(output.OutputRoot))
}

func (h *FactoryHelper) StartOutputAlphabetGame(ctx context.Context, l2Node string, l2BlockNumber uint64, rootClaim common.Hash, opts ...GameOpt) *OutputAlphabetGameHelper {
	cfg := newGameCfg(opts...)
	logger := testlog.Logger(h.t, log.LevelInfo).New("role", "OutputAlphabetGameHelper")
	rollupClient := h.system.RollupClient(l2Node)

	extraData := h.createBisectionGameExtraData(l2Node, l2BlockNumber, cfg)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	tx, err := transactions.PadGasEstimate(h.opts, 2, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return h.factory.Create(opts, alphabetGameType, rootClaim, extraData)
	})
	h.require.NoError(err, "create output bisection game")
	rcpt, err := wait.ForReceiptOK(ctx, h.client, tx.Hash())
	h.require.NoError(err, "wait for create output bisection game receipt to be OK")
	h.require.Len(rcpt.Logs, 2, "should have emitted a single DisputeGameCreated event")
	createdEvent, err := h.factory.ParseDisputeGameCreated(*rcpt.Logs[1])
	h.require.NoError(err)
	game, err := bindings.NewFaultDisputeGame(createdEvent.DisputeProxy, h.client)
	h.require.NoError(err)

	callOpts := &bind.CallOpts{Context: ctx}
	prestateBlock, err := game.StartingBlockNumber(callOpts)
	h.require.NoError(err, "Failed to load starting block number")
	poststateBlock, err := game.L2BlockNumber(callOpts)
	h.require.NoError(err, "Failed to load l2 block number")
	splitDepth, err := game.SplitDepth(callOpts)
	h.require.NoError(err, "Failed to load split depth")
	l1Head := h.getL1Head(ctx, game)
	prestateProvider := outputs.NewPrestateProvider(rollupClient, prestateBlock.Uint64())

	provider := outputs.NewTraceProvider(logger, prestateProvider, rollupClient, l1Head, faultTypes.Depth(splitDepth.Uint64()), prestateBlock.Uint64(), poststateBlock.Uint64())

	return &OutputAlphabetGameHelper{
		OutputGameHelper: OutputGameHelper{
			t:                     h.t,
			require:               h.require,
			client:                h.client,
			opts:                  h.opts,
			game:                  game,
			factoryAddr:           h.factoryAddr,
			addr:                  createdEvent.DisputeProxy,
			correctOutputProvider: provider,
			system:                h.system,
		},
	}
}

func (h *FactoryHelper) createBisectionGameExtraData(l2Node string, l2BlockNumber uint64, cfg *gameCfg) []byte {
	h.waitForBlock(l2Node, l2BlockNumber, cfg)
	h.t.Logf("Creating game with l2 block number: %v", l2BlockNumber)
	extraData := make([]byte, 32)
	binary.BigEndian.PutUint64(extraData[24:], l2BlockNumber)
	return extraData
}

func (h *FactoryHelper) waitForBlock(l2Node string, l2BlockNumber uint64, cfg *gameCfg) {
	if cfg.allowFuture {
		// Proposing a block that doesn't exist yet, so don't perform any checks
		return
	}

	l2Client := h.system.NodeClient(l2Node)
	if cfg.allowUnsafe {
		_, err := geth.WaitForBlock(new(big.Int).SetUint64(l2BlockNumber), l2Client, 1*time.Minute)
		h.require.NoErrorf(err, "Block number %v did not become unsafe", l2BlockNumber)
	} else {
		_, err := geth.WaitForBlockToBeSafe(new(big.Int).SetUint64(l2BlockNumber), l2Client, 1*time.Minute)
		h.require.NoErrorf(err, "Block number %v did not become safe", l2BlockNumber)
	}
}

func (h *FactoryHelper) StartChallenger(ctx context.Context, name string, options ...challenger.Option) *challenger.Helper {
	opts := []challenger.Option{
		challenger.WithFactoryAddress(h.factoryAddr),
	}
	opts = append(opts, options...)
	c := challenger.NewChallenger(h.t, ctx, h.system, name, opts...)
	h.t.Cleanup(func() {
		_ = c.Close()
	})
	return c
}
