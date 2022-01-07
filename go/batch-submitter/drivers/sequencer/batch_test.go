package sequencer_test

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/go/batch-submitter/drivers/sequencer"
	l2common "github.com/ethereum-optimism/optimism/l2geth/common"
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
