package sequencer_test

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/batch-submitter/drivers/sequencer"
	l2common "github.com/ethereum-optimism/optimism/l2geth/common"
	"github.com/ethereum-optimism/optimism/l2geth/core/types"
	l2types "github.com/ethereum-optimism/optimism/l2geth/core/types"
	"github.com/stretchr/testify/require"
)

func TestBatchElementFromBlock(t *testing.T) {
	expTime := uint64(42)
	expBlockNumber := uint64(43)

	header := &l2types.Header{
		Time: expTime,
	}
	expTx := l2types.NewTransaction(
		1, l2common.Address{}, new(big.Int).SetUint64(2), 3,
		new(big.Int).SetUint64(4), []byte{},
	)
	expTx.SetL1BlockNumber(expBlockNumber)

	txs := []*l2types.Transaction{expTx}

	block := l2types.NewBlock(header, txs, nil, nil)
	element := sequencer.BatchElementFromBlock(block)

	require.Equal(t, element.Timestamp, expTime)
	require.Equal(t, element.BlockNumber, expBlockNumber)
	require.True(t, element.IsSequencerTx())
	require.Equal(t, element.Tx.Tx(), expTx)

	queueMeta := l2types.NewTransactionMeta(
		new(big.Int).SetUint64(expBlockNumber), 0, nil,
		l2types.QueueOriginL1ToL2, nil, nil, nil,
	)

	expTx.SetTransactionMeta(queueMeta)

	element = sequencer.BatchElementFromBlock(block)

	require.Equal(t, element.Timestamp, expTime)
	require.Equal(t, element.BlockNumber, expBlockNumber)
	require.False(t, element.IsSequencerTx())
	require.Nil(t, element.Tx)
}

func TestGenSequencerParams(t *testing.T) {
	tx := types.NewTransaction(0, l2common.Address{}, big.NewInt(0), 0, big.NewInt(0), []byte{})

	shouldStartAtElement := uint64(1)
	blockOffset := uint64(1)
	batches := []sequencer.BatchElement{
		{Timestamp: 1, BlockNumber: 1},
		{Timestamp: 1, BlockNumber: 1, Tx: sequencer.NewCachedTx(tx)},
	}

	params, err := sequencer.GenSequencerBatchParams(shouldStartAtElement, blockOffset, batches)
	require.NoError(t, err)

	require.Equal(t, uint64(0), params.ShouldStartAtElement)
	require.Equal(t, uint64(len(batches)), params.TotalElementsToAppend)
	require.Equal(t, len(batches), len(params.Contexts))
	// There is only 1 sequencer tx
	require.Equal(t, 1, len(params.Txs))

	// There are 2 contexts
	// The first context contains the deposit
	context1 := params.Contexts[0]
	require.Equal(t, uint64(0), context1.NumSequencedTxs)
	require.Equal(t, uint64(1), context1.NumSubsequentQueueTxs)
	require.Equal(t, uint64(1), context1.Timestamp)
	require.Equal(t, uint64(1), context1.BlockNumber)

	// The second context contains the sequencer tx
	context2 := params.Contexts[1]
	require.Equal(t, uint64(1), context2.NumSequencedTxs)
	require.Equal(t, uint64(0), context2.NumSubsequentQueueTxs)
	require.Equal(t, uint64(1), context2.Timestamp)
	require.Equal(t, uint64(1), context2.BlockNumber)
}

func TestGenSequencerParamsOnlyDeposits(t *testing.T) {
	shouldStartAtElement := uint64(1)
	blockOffset := uint64(1)
	batches := []sequencer.BatchElement{
		{Timestamp: 1, BlockNumber: 1},
		{Timestamp: 1, BlockNumber: 1},
		{Timestamp: 2, BlockNumber: 2},
	}

	params, err := sequencer.GenSequencerBatchParams(shouldStartAtElement, blockOffset, batches)
	require.NoError(t, err)

	// The batches will pack deposits into the same context when their
	// timestamps and blocknumbers are the same
	require.Equal(t, uint64(0), params.ShouldStartAtElement)
	require.Equal(t, uint64(len(batches)), params.TotalElementsToAppend)
	// 2 deposits have the same timestamp + blocknumber, they go in the
	// same context. 1 deposit has a different timestamp + blocknumber,
	// it goes into a different context. Therefore there are 2 contexts
	require.Equal(t, 2, len(params.Contexts))
	// No sequencer txs
	require.Equal(t, 0, len(params.Txs))

	// There are 2 contexts
	// The first context contains the deposit
	context1 := params.Contexts[0]
	require.Equal(t, uint64(0), context1.NumSequencedTxs)
	require.Equal(t, uint64(2), context1.NumSubsequentQueueTxs)
	require.Equal(t, uint64(1), context1.Timestamp)
	require.Equal(t, uint64(1), context1.BlockNumber)

	context2 := params.Contexts[1]
	require.Equal(t, uint64(0), context2.NumSequencedTxs)
	require.Equal(t, uint64(1), context2.NumSubsequentQueueTxs)
	require.Equal(t, uint64(2), context2.Timestamp)
	require.Equal(t, uint64(2), context2.BlockNumber)
}
