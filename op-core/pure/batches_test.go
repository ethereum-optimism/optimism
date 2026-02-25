package pure

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestDecodeBatches_SingularBatch(t *testing.T) {
	cfg := testRollupConfig()
	safeHead := testSafeHead(cfg)
	l1Ref := testL1Ref(1)

	batch := &derive.SingularBatch{
		ParentHash: safeHead.Hash,
		EpochNum:   rollup.Epoch(l1Ref.Number),
		EpochHash:  l1Ref.Hash,
		Timestamp:  safeHead.Time + cfg.BlockTime,
	}

	channelData := encodeBatchToChannelData(t, batch)

	cursor := newCursor(safeHead)
	l1Origins := []eth.L1BlockRef{testL1Ref(0), l1Ref}

	batches, err := decodeBatches(bytes.NewReader(channelData), cfg, l1Origins, cursor, l1Ref)
	require.NoError(t, err)
	require.Len(t, batches, 1)

	decoded := batches[0]
	require.Equal(t, batch.ParentHash, decoded.ParentHash)
	require.Equal(t, batch.EpochNum, decoded.EpochNum)
	require.Equal(t, batch.EpochHash, decoded.EpochHash)
	require.Equal(t, batch.Timestamp, decoded.Timestamp)
}

func TestValidateBatch_ValidSingular(t *testing.T) {
	cfg := testRollupConfig()
	l1Origin := testL1Ref(5)

	cursor := l2Cursor{
		Number:    10,
		Timestamp: 100,
		L1Origin:  l1Origin.ID(),
	}

	batch := &derive.SingularBatch{
		EpochNum:  rollup.Epoch(l1Origin.Number),
		EpochHash: l1Origin.Hash,
		Timestamp: cursor.Timestamp + cfg.BlockTime,
	}

	l1Origins := []eth.L1BlockRef{testL1Ref(4), l1Origin, testL1Ref(6)}

	require.True(t, validateBatch(batch, cursor, l1Origins, cfg, l1Origin.Number))
}

func TestValidateBatch_WrongTimestamp(t *testing.T) {
	cfg := testRollupConfig()
	l1Origin := testL1Ref(5)

	cursor := l2Cursor{
		Number:    10,
		Timestamp: 100,
		L1Origin:  l1Origin.ID(),
	}

	batch := &derive.SingularBatch{
		EpochNum:  rollup.Epoch(l1Origin.Number),
		EpochHash: l1Origin.Hash,
		Timestamp: cursor.Timestamp + cfg.BlockTime + 1, // wrong
	}

	l1Origins := []eth.L1BlockRef{testL1Ref(4), l1Origin, testL1Ref(6)}

	require.False(t, validateBatch(batch, cursor, l1Origins, cfg, l1Origin.Number))
}

func TestValidateBatch_SpanBatchNoOverlap(t *testing.T) {
	cfg := testRollupConfig()
	l1Origin := testL1Ref(5)

	cursor := l2Cursor{
		Number:    10,
		Timestamp: 100,
		L1Origin:  l1Origin.ID(),
	}

	// Timestamp before cursor (overlap) -- this will fail the timestamp == cursor + blockTime check
	batch := &derive.SingularBatch{
		EpochNum:  rollup.Epoch(l1Origin.Number),
		EpochHash: l1Origin.Hash,
		Timestamp: cursor.Timestamp - 2,
	}

	l1Origins := []eth.L1BlockRef{testL1Ref(4), l1Origin, testL1Ref(6)}

	require.False(t, validateBatch(batch, cursor, l1Origins, cfg, l1Origin.Number))
}

func TestValidateBatch_EpochTooOld(t *testing.T) {
	cfg := testRollupConfig()
	l1Origin := testL1Ref(5)

	cursor := l2Cursor{
		Number:    10,
		Timestamp: 100,
		L1Origin:  l1Origin.ID(),
	}

	oldOrigin := testL1Ref(3)
	batch := &derive.SingularBatch{
		EpochNum:  rollup.Epoch(oldOrigin.Number), // before cursor's L1 origin
		EpochHash: oldOrigin.Hash,
		Timestamp: cursor.Timestamp + cfg.BlockTime,
	}

	l1Origins := []eth.L1BlockRef{oldOrigin, testL1Ref(4), l1Origin, testL1Ref(6)}

	require.False(t, validateBatch(batch, cursor, l1Origins, cfg, l1Origin.Number))
}

func TestValidateBatch_EpochTooNew(t *testing.T) {
	cfg := testRollupConfig()
	l1Origin := testL1Ref(5)

	cursor := l2Cursor{
		Number:    10,
		Timestamp: 100,
		L1Origin:  l1Origin.ID(),
	}

	batch := &derive.SingularBatch{
		EpochNum:  rollup.Epoch(100), // way beyond latest L1 origin
		EpochHash: common.Hash{0xab},
		Timestamp: cursor.Timestamp + cfg.BlockTime,
	}

	l1Origins := []eth.L1BlockRef{testL1Ref(4), l1Origin, testL1Ref(6)}

	require.False(t, validateBatch(batch, cursor, l1Origins, cfg, l1Origin.Number))
}

func TestValidateBatch_SequenceWindowExpired(t *testing.T) {
	cfg := testRollupConfig()
	l1Origin := testL1Ref(5)

	cursor := l2Cursor{
		Number:    10,
		Timestamp: 100,
		L1Origin:  l1Origin.ID(),
	}

	batch := &derive.SingularBatch{
		EpochNum:  rollup.Epoch(l1Origin.Number),
		EpochHash: l1Origin.Hash,
		Timestamp: cursor.Timestamp + cfg.BlockTime,
	}

	l1Origins := []eth.L1BlockRef{testL1Ref(4), l1Origin, testL1Ref(6)}

	// Inclusion at block 16: epochNum(5) + SeqWindowSize(10) = 15 < 16 → expired
	require.False(t, validateBatch(batch, cursor, l1Origins, cfg, 16))
}

func TestValidateBatch_EpochSkip(t *testing.T) {
	cfg := testRollupConfig()
	l1Origin := testL1Ref(5)

	cursor := l2Cursor{
		Number:    10,
		Timestamp: 100,
		L1Origin:  l1Origin.ID(),
	}

	// Epoch 7 skips over epoch 6 (cursor is at 5, can only go to 6)
	batch := &derive.SingularBatch{
		EpochNum:  rollup.Epoch(7),
		EpochHash: testL1Ref(7).Hash,
		Timestamp: cursor.Timestamp + cfg.BlockTime,
	}

	l1Origins := []eth.L1BlockRef{testL1Ref(4), l1Origin, testL1Ref(6), testL1Ref(7)}

	require.False(t, validateBatch(batch, cursor, l1Origins, cfg, l1Origin.Number))
}

func TestValidateBatch_DepositTxRejected(t *testing.T) {
	cfg := testRollupConfig()
	l1Origin := testL1Ref(5)

	cursor := l2Cursor{
		Number:    10,
		Timestamp: 100,
		L1Origin:  l1Origin.ID(),
	}

	batch := &derive.SingularBatch{
		EpochNum:     rollup.Epoch(l1Origin.Number),
		EpochHash:    l1Origin.Hash,
		Timestamp:    cursor.Timestamp + cfg.BlockTime,
		Transactions: []hexutil.Bytes{{0x7e, 0x01, 0x02}}, // deposit tx type
	}

	l1Origins := []eth.L1BlockRef{testL1Ref(4), l1Origin, testL1Ref(6)}

	require.False(t, validateBatch(batch, cursor, l1Origins, cfg, l1Origin.Number))
}
