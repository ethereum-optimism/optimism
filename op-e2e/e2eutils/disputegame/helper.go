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
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/alphabet"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

const alphabetGameType uint8 = 0
const cannonGameType uint8 = 1
const alphabetGameDepth = 4
const lastAlphabetTraceIndex = 1<<alphabetGameDepth - 1

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

var CorrectAlphabet = "abcdefghijklmnop"

type FactoryHelper struct {
	t           *testing.T
	require     *require.Assertions
	client      *ethclient.Client
	opts        *bind.TransactOpts
	factoryAddr common.Address
	factory     *bindings.DisputeGameFactory
	blockOracle *bindings.BlockOracle
	l2oo        *bindings.L2OutputOracleCaller
}

func NewFactoryHelper(t *testing.T, ctx context.Context, deployments *genesis.L1Deployments, client *ethclient.Client) *FactoryHelper {
	require := require.New(t)
	chainID, err := client.ChainID(ctx)
	require.NoError(err)
	opts, err := bind.NewKeyedTransactorWithChainID(deployer.TestKey, chainID)
	require.NoError(err)

	require.NotNil(deployments, "No deployments")
	factoryAddr := deployments.DisputeGameFactoryProxy
	factory, err := bindings.NewDisputeGameFactory(factoryAddr, client)
	require.NoError(err)
	blockOracle, err := bindings.NewBlockOracle(deployments.BlockOracle, client)
	require.NoError(err)
	l2oo, err := bindings.NewL2OutputOracleCaller(deployments.L2OutputOracleProxy, client)
	require.NoError(err, "Error creating l2oo caller")

	return &FactoryHelper{
		t:           t,
		require:     require,
		client:      client,
		opts:        opts,
		factory:     factory,
		factoryAddr: factoryAddr,
		blockOracle: blockOracle,
		l2oo:        l2oo,
	}
}

func (h *FactoryHelper) StartAlphabetGame(ctx context.Context, claimedAlphabet string) *AlphabetGameHelper {
	l2BlockNumber := h.waitForProposals(ctx)
	l1Head := h.checkpointL1Block(ctx)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	trace := alphabet.NewTraceProvider(claimedAlphabet, alphabetGameDepth)
	rootClaim, err := trace.Get(ctx, lastAlphabetTraceIndex)
	h.require.NoError(err, "get root claim")
	extraData := make([]byte, 64)
	binary.BigEndian.PutUint64(extraData[24:], l2BlockNumber)
	binary.BigEndian.PutUint64(extraData[56:], l1Head.Uint64())
	tx, err := h.factory.Create(h.opts, alphabetGameType, rootClaim, extraData)
	h.require.NoError(err, "create fault dispute game")
	rcpt, err := wait.ForReceiptOK(ctx, h.client, tx.Hash())
	h.require.NoError(err, "wait for create fault dispute game receipt to be OK")
	h.require.Len(rcpt.Logs, 1, "should have emitted a single DisputeGameCreated event")
	createdEvent, err := h.factory.ParseDisputeGameCreated(*rcpt.Logs[0])
	h.require.NoError(err)
	game, err := bindings.NewFaultDisputeGame(createdEvent.DisputeProxy, h.client)
	h.require.NoError(err)

	return &AlphabetGameHelper{
		FaultGameHelper: FaultGameHelper{
			t:       h.t,
			require: h.require,
			client:  h.client,
			opts:    h.opts,
			game:    game,
			addr:    createdEvent.DisputeProxy,
		},
		claimedAlphabet: claimedAlphabet,
	}
}

func (h *FactoryHelper) StartCannonGame(ctx context.Context, rootClaim common.Hash) *CannonGameHelper {
	l2BlockNumber := h.waitForProposals(ctx)
	l1Head := h.checkpointL1Block(ctx)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	extraData := make([]byte, 64)
	binary.BigEndian.PutUint64(extraData[24:], l2BlockNumber)
	binary.BigEndian.PutUint64(extraData[56:], l1Head.Uint64())
	tx, err := h.factory.Create(h.opts, cannonGameType, rootClaim, extraData)
	h.require.NoError(err, "create fault dispute game")
	rcpt, err := wait.ForReceiptOK(ctx, h.client, tx.Hash())
	h.require.NoError(err, "wait for create fault dispute game receipt to be OK")
	h.require.Len(rcpt.Logs, 1, "should have emitted a single DisputeGameCreated event")
	createdEvent, err := h.factory.ParseDisputeGameCreated(*rcpt.Logs[0])
	h.require.NoError(err)
	game, err := bindings.NewFaultDisputeGame(createdEvent.DisputeProxy, h.client)
	h.require.NoError(err)

	return &CannonGameHelper{
		FaultGameHelper: FaultGameHelper{
			t:       h.t,
			require: h.require,
			client:  h.client,
			opts:    h.opts,
			game:    game,
			addr:    createdEvent.DisputeProxy,
		},
	}
}
func (h *FactoryHelper) StartChallenger(ctx context.Context, l1Endpoint string, name string, options ...challenger.Option) *challenger.Helper {
	opts := []challenger.Option{
		func(c *config.Config) {
			// Uncomment when challenger actually supports setting the game factory address
			//c.FactoryAddress = h.factoryAddr
			c.TraceType = config.TraceTypeAlphabet
		},
	}
	opts = append(opts, options...)
	c := challenger.NewChallenger(h.t, ctx, l1Endpoint, name, opts...)
	h.t.Cleanup(func() {
		_ = c.Close()
	})
	return c
}

// waitForProposals waits until there are at least two proposals in the output oracle
// This is the minimum required for creating a game.
// Returns the l2 block number of the latest available proposal
func (h *FactoryHelper) waitForProposals(ctx context.Context) uint64 {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	opts := &bind.CallOpts{Context: ctx}
	latestOutputIndex, err := wait.AndGet(
		ctx,
		time.Second,
		func() (*big.Int, error) {
			index, err := h.l2oo.LatestOutputIndex(opts)
			if err != nil {
				h.t.Logf("Could not get latest output index: %v", err.Error())
				return nil, nil
			}
			h.t.Logf("Latest output index: %v", index)
			return index, nil
		},
		func(index *big.Int) bool {
			return index != nil && index.Cmp(big.NewInt(1)) >= 0
		})
	h.require.NoError(err, "Did not get two output roots")
	output, err := h.l2oo.GetL2Output(opts, latestOutputIndex)
	h.require.NoErrorf(err, "Could not get latst output root index: %v", latestOutputIndex)
	return output.L2BlockNumber.Uint64()
}

// checkpointL1Block stores the current L1 block in the oracle
// Returns the L1 block number that was stored as the checkpoint
func (h *FactoryHelper) checkpointL1Block(ctx context.Context) *big.Int {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	// Store the current block in the oracle
	tx, err := h.blockOracle.Checkpoint(h.opts)
	h.require.NoError(err)
	r, err := wait.ForReceiptOK(ctx, h.client, tx.Hash())
	h.require.NoError(err, "failed to store block in block oracle")
	return new(big.Int).Sub(r.BlockNumber, big.NewInt(1))
}
