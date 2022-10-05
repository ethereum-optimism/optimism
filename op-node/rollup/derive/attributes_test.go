package derive

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestPreparePayloadAttributes(t *testing.T) {
	// test config, only init the necessary fields
	cfg := &rollup.Config{
		BlockTime:              2,
		L1ChainID:              big.NewInt(101),
		L2ChainID:              big.NewInt(102),
		FeeRecipientAddress:    common.Address{0xaa},
		DepositContractAddress: common.Address{0xbb},
	}

	t.Run("inconsistent next height origin", func(t *testing.T) {
		rng := rand.New(rand.NewSource(1234))
		l1Fetcher := &testutils.MockL1Source{}
		defer l1Fetcher.AssertExpectations(t)
		l2Parent := testutils.RandomL2BlockRef(rng)
		l2Time := l2Parent.Time + cfg.BlockTime
		l1Info := testutils.RandomBlockInfo(rng)
		l1Info.InfoNum = l2Parent.L1Origin.Number + 1
		epoch := l1Info.ID()
		l1Fetcher.ExpectFetch(epoch.Hash, l1Info, nil, nil)
		_, err := PreparePayloadAttributes(context.Background(), cfg, l1Fetcher, l2Parent, l2Time, epoch)
		require.NotNil(t, err, "inconsistent L1 origin error expected")
		require.ErrorIs(t, err, ErrReset, "inconsistent L1 origin transition must be handled like a critical error with reorg")
	})
	t.Run("inconsistent equal height origin", func(t *testing.T) {
		rng := rand.New(rand.NewSource(1234))
		l1Fetcher := &testutils.MockL1Source{}
		defer l1Fetcher.AssertExpectations(t)
		l2Parent := testutils.RandomL2BlockRef(rng)
		l2Time := l2Parent.Time + cfg.BlockTime
		l1Info := testutils.RandomBlockInfo(rng)
		l1Info.InfoNum = l2Parent.L1Origin.Number
		epoch := l1Info.ID()
		_, err := PreparePayloadAttributes(context.Background(), cfg, l1Fetcher, l2Parent, l2Time, epoch)
		require.NotNil(t, err, "inconsistent L1 origin error expected")
		require.ErrorIs(t, err, ErrReset, "inconsistent L1 origin transition must be handled like a critical error with reorg")
	})
	t.Run("rpc fail Fetch", func(t *testing.T) {
		rng := rand.New(rand.NewSource(1234))
		l1Fetcher := &testutils.MockL1Source{}
		defer l1Fetcher.AssertExpectations(t)
		l2Parent := testutils.RandomL2BlockRef(rng)
		l2Time := l2Parent.Time + cfg.BlockTime
		epoch := l2Parent.L1Origin
		epoch.Number += 1
		mockRPCErr := errors.New("mock rpc error")
		l1Fetcher.ExpectFetch(epoch.Hash, nil, nil, mockRPCErr)
		_, err := PreparePayloadAttributes(context.Background(), cfg, l1Fetcher, l2Parent, l2Time, epoch)
		require.ErrorIs(t, err, mockRPCErr, "mock rpc error expected")
		require.ErrorIs(t, err, ErrTemporary, "rpc errors should not be critical, it is not necessary to reorg")
	})
	t.Run("rpc fail InfoByHash", func(t *testing.T) {
		rng := rand.New(rand.NewSource(1234))
		l1Fetcher := &testutils.MockL1Source{}
		defer l1Fetcher.AssertExpectations(t)
		l2Parent := testutils.RandomL2BlockRef(rng)
		l2Time := l2Parent.Time + cfg.BlockTime
		epoch := l2Parent.L1Origin
		mockRPCErr := errors.New("mock rpc error")
		l1Fetcher.ExpectInfoByHash(epoch.Hash, nil, mockRPCErr)
		_, err := PreparePayloadAttributes(context.Background(), cfg, l1Fetcher, l2Parent, l2Time, epoch)
		require.ErrorIs(t, err, mockRPCErr, "mock rpc error expected")
		require.ErrorIs(t, err, ErrTemporary, "rpc errors should not be critical, it is not necessary to reorg")
	})
	t.Run("next origin without deposits", func(t *testing.T) {
		rng := rand.New(rand.NewSource(1234))
		l1Fetcher := &testutils.MockL1Source{}
		defer l1Fetcher.AssertExpectations(t)
		l2Parent := testutils.RandomL2BlockRef(rng)
		l2Time := l2Parent.Time + cfg.BlockTime
		l1Info := testutils.RandomBlockInfo(rng)
		l1Info.InfoParentHash = l2Parent.L1Origin.Hash
		l1Info.InfoNum = l2Parent.L1Origin.Number + 1
		epoch := l1Info.ID()
		l1InfoTx, err := L1InfoDepositBytes(0, l1Info)
		require.NoError(t, err)
		l1Fetcher.ExpectFetch(epoch.Hash, l1Info, nil, nil)
		attrs, err := PreparePayloadAttributes(context.Background(), cfg, l1Fetcher, l2Parent, l2Time, epoch)
		require.NoError(t, err)
		require.NotNil(t, attrs)
		require.Equal(t, l2Parent.Time+cfg.BlockTime, uint64(attrs.Timestamp))
		require.Equal(t, eth.Bytes32(l1Info.InfoMixDigest), attrs.PrevRandao)
		require.Equal(t, cfg.FeeRecipientAddress, attrs.SuggestedFeeRecipient)
		require.Equal(t, 1, len(attrs.Transactions))
		require.Equal(t, l1InfoTx, []byte(attrs.Transactions[0]))
		require.True(t, attrs.NoTxPool)
	})
	t.Run("next origin with deposits", func(t *testing.T) {
		rng := rand.New(rand.NewSource(1234))
		l1Fetcher := &testutils.MockL1Source{}
		defer l1Fetcher.AssertExpectations(t)
		l2Parent := testutils.RandomL2BlockRef(rng)
		l2Time := l2Parent.Time + cfg.BlockTime
		l1Info := testutils.RandomBlockInfo(rng)
		l1Info.InfoParentHash = l2Parent.L1Origin.Hash
		l1Info.InfoNum = l2Parent.L1Origin.Number + 1

		receipts, depositTxs := makeReceipts(rng, l1Info.InfoHash, cfg.DepositContractAddress, []receiptData{
			{goodReceipt: true, DepositLogs: []bool{true, false}},
			{goodReceipt: true, DepositLogs: []bool{true}},
			{goodReceipt: false, DepositLogs: []bool{true}},
			{goodReceipt: false, DepositLogs: []bool{false}},
		})
		usedDepositTxs, err := encodeDeposits(depositTxs)
		require.NoError(t, err)

		epoch := l1Info.ID()
		l1InfoTx, err := L1InfoDepositBytes(0, l1Info)
		require.NoError(t, err)

		l2Txs := append(append(make([]eth.Data, 0), l1InfoTx), usedDepositTxs...)

		l1Fetcher.ExpectFetch(epoch.Hash, l1Info, receipts, nil)
		attrs, err := PreparePayloadAttributes(context.Background(), cfg, l1Fetcher, l2Parent, l2Time, epoch)
		require.NoError(t, err)
		require.NotNil(t, attrs)
		require.Equal(t, l2Parent.Time+cfg.BlockTime, uint64(attrs.Timestamp))
		require.Equal(t, eth.Bytes32(l1Info.InfoMixDigest), attrs.PrevRandao)
		require.Equal(t, cfg.FeeRecipientAddress, attrs.SuggestedFeeRecipient)
		require.Equal(t, len(l2Txs), len(attrs.Transactions), "Expected txs to equal l1 info tx + user deposit txs")
		require.Equal(t, l2Txs, attrs.Transactions)
		require.True(t, attrs.NoTxPool)
	})
	t.Run("same origin again", func(t *testing.T) {
		rng := rand.New(rand.NewSource(1234))
		l1Fetcher := &testutils.MockL1Source{}
		defer l1Fetcher.AssertExpectations(t)
		l2Parent := testutils.RandomL2BlockRef(rng)
		l2Time := l2Parent.Time + cfg.BlockTime
		l1Info := testutils.RandomBlockInfo(rng)
		l1Info.InfoHash = l2Parent.L1Origin.Hash
		l1Info.InfoNum = l2Parent.L1Origin.Number

		epoch := l1Info.ID()
		l1InfoTx, err := L1InfoDepositBytes(l2Parent.SequenceNumber+1, l1Info)
		require.NoError(t, err)

		l1Fetcher.ExpectInfoByHash(epoch.Hash, l1Info, nil)
		attrs, err := PreparePayloadAttributes(context.Background(), cfg, l1Fetcher, l2Parent, l2Time, epoch)
		require.NoError(t, err)
		require.NotNil(t, attrs)
		require.Equal(t, l2Parent.Time+cfg.BlockTime, uint64(attrs.Timestamp))
		require.Equal(t, eth.Bytes32(l1Info.InfoMixDigest), attrs.PrevRandao)
		require.Equal(t, cfg.FeeRecipientAddress, attrs.SuggestedFeeRecipient)
		require.Equal(t, 1, len(attrs.Transactions))
		require.Equal(t, l1InfoTx, []byte(attrs.Transactions[0]))
		require.True(t, attrs.NoTxPool)
	})
}

func encodeDeposits(deposits []*types.DepositTx) (out []eth.Data, err error) {
	for i, tx := range deposits {
		opaqueTx, err := types.NewTx(tx).MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("bad deposit %d: %w", i, err)
		}
		out = append(out, opaqueTx)
	}
	return
}
