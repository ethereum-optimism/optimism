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
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/l2oo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/transactions"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
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
	cannonGameType    uint8 = 0
	alphabetGameType  uint8 = 255
	alphabetGameDepth       = 4
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

type DisputeSystem interface {
	NodeEndpoint(name string) string
	NodeClient(name string) *ethclient.Client
	RollupEndpoint(name string) string
	RollupClient(name string) *sources.RollupClient

	L1Deployments() *genesis.L1Deployments
	RollupCfg() *rollup.Config
	L2Genesis() *core.Genesis
}

type FactoryHelper struct {
	t           *testing.T
	require     *require.Assertions
	system      DisputeSystem
	client      *ethclient.Client
	opts        *bind.TransactOpts
	factoryAddr common.Address
	factory     *bindings.DisputeGameFactory
	l2ooHelper  *l2oo.L2OOHelper
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
		l2ooHelper:  l2oo.NewL2OOHelperReadOnly(t, l1Deployments, client),
	}
}

func (h *FactoryHelper) StartOutputCannonGameWithCorrectRoot(ctx context.Context, l2Node string, l2BlockNumber uint64) *OutputCannonGameHelper {
	h.waitForBlockToBeSafe(l2Node, l2BlockNumber)
	output, err := h.system.RollupClient(l2Node).OutputAtBlock(ctx, l2BlockNumber)
	h.require.NoErrorf(err, "Failed to get output at block %v", l2BlockNumber)
	return h.StartOutputCannonGame(ctx, l2Node, l2BlockNumber, common.Hash(output.OutputRoot))
}

func (h *FactoryHelper) StartOutputCannonGame(ctx context.Context, l2Node string, l2BlockNumber uint64, rootClaim common.Hash) *OutputCannonGameHelper {
	logger := testlog.Logger(h.t, log.LvlInfo).New("role", "OutputCannonGameHelper")
	rollupClient := h.system.RollupClient(l2Node)

	extraData := h.createBisectionGameExtraData(l2Node, l2BlockNumber)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	tx, err := transactions.PadGasEstimate(h.opts, 2, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return h.factory.Create(opts, cannonGameType, rootClaim, extraData)
	})
	h.require.NoError(err, "create fault dispute game")
	rcpt, err := wait.ForReceiptOK(ctx, h.client, tx.Hash())
	h.require.NoError(err, "wait for create fault dispute game receipt to be OK")
	h.require.Len(rcpt.Logs, 1, "should have emitted a single DisputeGameCreated event")
	createdEvent, err := h.factory.ParseDisputeGameCreated(*rcpt.Logs[0])
	h.require.NoError(err)
	game, err := bindings.NewFaultDisputeGame(createdEvent.DisputeProxy, h.client)
	h.require.NoError(err)

	prestateBlock, err := game.GenesisBlockNumber(&bind.CallOpts{Context: ctx})
	h.require.NoError(err, "Failed to load genesis block number")
	poststateBlock, err := game.L2BlockNumber(&bind.CallOpts{Context: ctx})
	h.require.NoError(err, "Failed to load l2 block number")
	splitDepth, err := game.SplitDepth(&bind.CallOpts{Context: ctx})
	h.require.NoError(err, "Failed to load split depth")
	prestateProvider := outputs.NewPrestateProvider(ctx, logger, rollupClient, prestateBlock.Uint64())
	provider := outputs.NewTraceProviderFromInputs(logger, prestateProvider, rollupClient, faultTypes.Depth(splitDepth.Uint64()), prestateBlock.Uint64(), poststateBlock.Uint64())

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

func (h *FactoryHelper) StartOutputAlphabetGameWithCorrectRoot(ctx context.Context, l2Node string, l2BlockNumber uint64) *OutputAlphabetGameHelper {
	h.waitForBlockToBeSafe(l2Node, l2BlockNumber)
	output, err := h.system.RollupClient(l2Node).OutputAtBlock(ctx, l2BlockNumber)
	h.require.NoErrorf(err, "Failed to get output at block %v", l2BlockNumber)
	return h.StartOutputAlphabetGame(ctx, l2Node, l2BlockNumber, common.Hash(output.OutputRoot))
}

func (h *FactoryHelper) StartOutputAlphabetGame(ctx context.Context, l2Node string, l2BlockNumber uint64, rootClaim common.Hash) *OutputAlphabetGameHelper {
	logger := testlog.Logger(h.t, log.LvlInfo).New("role", "OutputAlphabetGameHelper")
	rollupClient := h.system.RollupClient(l2Node)

	extraData := h.createBisectionGameExtraData(l2Node, l2BlockNumber)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	tx, err := transactions.PadGasEstimate(h.opts, 2, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return h.factory.Create(opts, alphabetGameType, rootClaim, extraData)
	})
	h.require.NoError(err, "create output bisection game")
	rcpt, err := wait.ForReceiptOK(ctx, h.client, tx.Hash())
	h.require.NoError(err, "wait for create output bisection game receipt to be OK")
	h.require.Len(rcpt.Logs, 1, "should have emitted a single DisputeGameCreated event")
	createdEvent, err := h.factory.ParseDisputeGameCreated(*rcpt.Logs[0])
	h.require.NoError(err)
	game, err := bindings.NewFaultDisputeGame(createdEvent.DisputeProxy, h.client)
	h.require.NoError(err)

	prestateBlock, err := game.GenesisBlockNumber(&bind.CallOpts{Context: ctx})
	h.require.NoError(err, "Failed to load genesis block number")
	poststateBlock, err := game.L2BlockNumber(&bind.CallOpts{Context: ctx})
	h.require.NoError(err, "Failed to load l2 block number")
	splitDepth, err := game.SplitDepth(&bind.CallOpts{Context: ctx})
	h.require.NoError(err, "Failed to load split depth")
	prestateProvider := outputs.NewPrestateProvider(ctx, logger, rollupClient, prestateBlock.Uint64())
	provider := outputs.NewTraceProviderFromInputs(logger, prestateProvider, rollupClient, faultTypes.Depth(splitDepth.Uint64()), prestateBlock.Uint64(), poststateBlock.Uint64())

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

func (h *FactoryHelper) createBisectionGameExtraData(l2Node string, l2BlockNumber uint64) []byte {
	h.waitForBlockToBeSafe(l2Node, l2BlockNumber)
	h.t.Logf("Creating game with l2 block number: %v", l2BlockNumber)
	extraData := make([]byte, 32)
	binary.BigEndian.PutUint64(extraData[24:], l2BlockNumber)
	return extraData
}

func (h *FactoryHelper) waitForBlockToBeSafe(l2Node string, l2BlockNumber uint64) {
	l2Client := h.system.NodeClient(l2Node)
	_, err := geth.WaitForBlockToBeSafe(new(big.Int).SetUint64(l2BlockNumber), l2Client, 1*time.Minute)
	h.require.NoErrorf(err, "Block number %v did not become safe", l2BlockNumber)
}

func (h *FactoryHelper) StartChallenger(ctx context.Context, name string, options ...challenger.Option) *challenger.Helper {
	opts := []challenger.Option{
		challenger.WithFactoryAddress(h.factoryAddr),
	}
	opts = append(opts, options...)
	c := challenger.NewChallenger(h.t, ctx, h.system.NodeEndpoint("l1"), name, opts...)
	h.t.Cleanup(func() {
		_ = c.Close()
	})
	return c
}
