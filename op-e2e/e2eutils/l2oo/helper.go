package l2oo

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

type L2OOHelper struct {
	t       *testing.T
	require *require.Assertions
	client  *ethclient.Client
	l2oo    *bindings.L2OutputOracle

	// Nil when read-only
	transactOpts *bind.TransactOpts
	rollupCfg    *rollup.Config
}

func NewL2OOHelperReadOnly(t *testing.T, deployments *genesis.L1Deployments, client *ethclient.Client) *L2OOHelper {
	require := require.New(t)
	l2oo, err := bindings.NewL2OutputOracle(deployments.L2OutputOracleProxy, client)
	require.NoError(err, "Error creating l2oo bindings")

	return &L2OOHelper{
		t:       t,
		require: require,
		client:  client,
		l2oo:    l2oo,
	}
}

func NewL2OOHelper(t *testing.T, deployments *genesis.L1Deployments, client *ethclient.Client, proposerKey *ecdsa.PrivateKey, rollupCfg *rollup.Config) *L2OOHelper {
	h := NewL2OOHelperReadOnly(t, deployments, client)

	chainID, err := client.ChainID(context.Background())
	h.require.NoError(err, "Failed to get chain ID")
	transactOpts, err := bind.NewKeyedTransactorWithChainID(proposerKey, chainID)
	h.require.NoError(err)
	h.transactOpts = transactOpts
	h.rollupCfg = rollupCfg
	return h
}

// WaitForProposals waits until there are at least the specified number of proposals in the output oracle
// Returns the index of the latest output proposal
func (h *L2OOHelper) WaitForProposals(ctx context.Context, req int64) uint64 {
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
			return index != nil && index.Cmp(big.NewInt(req-1)) >= 0
		})
	h.require.NoErrorf(err, "Did not get %v output roots", req)
	return latestOutputIndex.Uint64()
}

func (h *L2OOHelper) GetL2Output(ctx context.Context, idx uint64) bindings.TypesOutputProposal {
	output, err := h.l2oo.GetL2Output(&bind.CallOpts{Context: ctx}, new(big.Int).SetUint64(idx))
	h.require.NoErrorf(err, "Failed to get output root at index: %v", idx)
	return output
}

func (h *L2OOHelper) GetL2OutputAfter(ctx context.Context, l2BlockNum uint64) bindings.TypesOutputProposal {
	opts := &bind.CallOpts{Context: ctx}
	outputIdx, err := h.l2oo.GetL2OutputIndexAfter(opts, new(big.Int).SetUint64(l2BlockNum))
	h.require.NoError(err, "Fetch challenged output index")
	output, err := h.l2oo.GetL2Output(opts, outputIdx)
	h.require.NoError(err, "Fetch challenged output")
	return output
}

func (h *L2OOHelper) GetL2OutputBefore(ctx context.Context, l2BlockNum uint64) bindings.TypesOutputProposal {
	opts := &bind.CallOpts{Context: ctx}
	latestBlockNum, err := h.l2oo.LatestBlockNumber(opts)
	h.require.NoError(err, "Failed to get latest output root block number")
	var outputIdx *big.Int
	if latestBlockNum.Uint64() < l2BlockNum {
		outputIdx, err = h.l2oo.LatestOutputIndex(opts)
		h.require.NoError(err, "Failed to get latest output index")
	} else {
		outputIdx, err = h.l2oo.GetL2OutputIndexAfter(opts, new(big.Int).SetUint64(l2BlockNum))
		h.require.NoErrorf(err, "Failed to get output index after block %v", l2BlockNum)
		h.require.NotZerof(outputIdx.Uint64(), "No l2 output before block %v", l2BlockNum)
		outputIdx = new(big.Int).Sub(outputIdx, common.Big1)
	}
	return h.GetL2Output(ctx, outputIdx.Uint64())
}

func (h *L2OOHelper) PublishNextOutput(ctx context.Context, outputRoot common.Hash) {
	h.require.NotNil(h.transactOpts, "Can't publish outputs from a read only L2OOHelper")
	nextBlockNum, err := h.l2oo.NextBlockNumber(&bind.CallOpts{Context: ctx})
	h.require.NoError(err, "Should get next block number")

	genesis := h.rollupCfg.Genesis
	targetTimestamp := genesis.L2Time + ((nextBlockNum.Uint64() - genesis.L2.Number) * h.rollupCfg.BlockTime)
	timedCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	h.require.NoErrorf(
		wait.ForBlockWithTimestamp(timedCtx, h.client, targetTimestamp),
		"Wait for L1 block with timestamp >= %v", targetTimestamp)

	tx, err := h.l2oo.ProposeL2Output(h.transactOpts, outputRoot, nextBlockNum, [32]byte{}, common.Big0)
	h.require.NoErrorf(err, "Failed to propose output root for l2 block number %v", nextBlockNum)
	_, err = wait.ForReceiptOK(ctx, h.client, tx.Hash())
	h.require.NoErrorf(err, "Proposal for l2 block %v failed", nextBlockNum)
}
