package derive

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	opderive "github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive/params"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var testLogger = log.NewLogger(log.DiscardHandler())

// addBatchToL1 adds a batcher transaction containing the given singular batch
// to a pre-existing L1Input.
func addBatchToL1(t *testing.T, l1 *L1Input, batch *opderive.SingularBatch) {
	t.Helper()
	channelData := encodeBatchToChannelData(t, batch)
	var chID opderive.ChannelID
	copy(chID[:], l1.Header.Hash().Bytes())
	l1.BatcherData = [][]byte{wrapInFrames(channelData, chID)}
}

func TestDeriver_SingleBatch(t *testing.T) {
	cfg := testRollupConfig()
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	d, err := NewDeriver(cfg, testL1ChainConfig(), testLogger, safeHead, sysConfig)
	require.NoError(t, err)

	chain := makeTestL1Chain(2)
	l1Ref1 := chain[1].BlockRef()
	addBatchToL1(t, chain[1], &opderive.SingularBatch{
		ParentHash: safeHead.Hash,
		EpochNum:   rollup.Epoch(l1Ref1.Number),
		EpochHash:  l1Ref1.Hash,
		Timestamp:  safeHead.Time + cfg.BlockTime,
	})

	require.NoError(t, d.AddL1Block(*chain[0], *chain[1]))

	attrs, l1Ref, err := d.Next(safeHead)
	require.NoError(t, err)
	require.NotNil(t, attrs)
	require.Equal(t, hexutil.Uint64(safeHead.Time+cfg.BlockTime), attrs.Timestamp)
	require.True(t, attrs.NoTxPool)
	require.Equal(t, chain[1].BlockRef(), l1Ref)

	// No more to opderive.
	_, _, err = d.Next(safeHead)
	require.True(t, errors.Is(err, ErrNeedL1Data))
}

func TestDeriver_NeedL1Data(t *testing.T) {
	cfg := testRollupConfig()
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	d, err := NewDeriver(cfg, testL1ChainConfig(), testLogger, safeHead, sysConfig)
	require.NoError(t, err)

	_, _, err = d.Next(safeHead)
	require.True(t, errors.Is(err, ErrNeedL1Data))
}

func TestDeriver_IncrementalL1(t *testing.T) {
	cfg := testRollupConfig()
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	d, err := NewDeriver(cfg, testL1ChainConfig(), testLogger, safeHead, sysConfig)
	require.NoError(t, err)

	chain := makeTestL1Chain(2)
	require.NoError(t, d.AddL1Block(*chain[0]))

	// No batches in block 0, need more data.
	_, _, err = d.Next(safeHead)
	require.True(t, errors.Is(err, ErrNeedL1Data))

	// Add block 1 with a batch.
	l1Ref1 := chain[1].BlockRef()
	addBatchToL1(t, chain[1], &opderive.SingularBatch{
		ParentHash: safeHead.Hash,
		EpochNum:   rollup.Epoch(l1Ref1.Number),
		EpochHash:  l1Ref1.Hash,
		Timestamp:  safeHead.Time + cfg.BlockTime,
	})
	require.NoError(t, d.AddL1Block(*chain[1]))

	attrs, _, err := d.Next(safeHead)
	require.NoError(t, err)
	require.NotNil(t, attrs)
	require.Equal(t, hexutil.Uint64(safeHead.Time+cfg.BlockTime), attrs.Timestamp)
}

func TestDeriver_EmptyBatches(t *testing.T) {
	cfg := testRollupConfig()
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	d, err := NewDeriver(cfg, testL1ChainConfig(), testLogger, safeHead, sysConfig)
	require.NoError(t, err)

	// Add SeqWindowSize + 2 L1 blocks with no batcher data.
	numBlocks := cfg.SeqWindowSize + 2
	chain := makeTestL1Chain(numBlocks)
	for _, block := range chain {
		require.NoError(t, d.AddL1Block(*block))
	}

	// Should generate empty batches when the sequencing window expires.
	currentSafeHead := safeHead
	var derived []*eth.PayloadAttributes
	for {
		attrs, _, err := d.Next(currentSafeHead)
		if errors.Is(err, ErrNeedL1Data) {
			break
		}
		require.NoError(t, err)
		require.NotNil(t, attrs)
		derived = append(derived, attrs)

		// Advance the safe head for the next call.
		currentSafeHead = eth.L2BlockRef{
			Hash:           common.Hash{byte(len(derived))},
			Number:         currentSafeHead.Number + 1,
			Time:           uint64(attrs.Timestamp),
			L1Origin:       currentSafeHead.L1Origin,
			SequenceNumber: currentSafeHead.SequenceNumber + 1,
		}
	}

	require.Greater(t, len(derived), 0, "empty batches should be generated when sequencer window expires")

	expectedTimestamp := safeHead.Time + cfg.BlockTime
	for _, attrs := range derived {
		require.Equal(t, hexutil.Uint64(expectedTimestamp), attrs.Timestamp)
		expectedTimestamp += cfg.BlockTime
	}
}

func TestDeriver_ReorgDetection(t *testing.T) {
	cfg := testRollupConfig()
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	d, err := NewDeriver(cfg, testL1ChainConfig(), testLogger, safeHead, sysConfig)
	require.NoError(t, err)

	chain := makeTestL1Chain(1)
	require.NoError(t, d.AddL1Block(*chain[0]))

	// Create a block that doesn't chain to block 0.
	reorgedBlock := makeTestL1Input(1)
	reorgedBlock.Header.ParentHash = common.HexToHash("0xbadparent")

	err = d.AddL1Block(*reorgedBlock)
	require.True(t, errors.Is(err, ErrReorg))
}

func TestDeriver_ReorgReset(t *testing.T) {
	cfg := testRollupConfig()
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	d, err := NewDeriver(cfg, testL1ChainConfig(), testLogger, safeHead, sysConfig)
	require.NoError(t, err)

	chain := makeTestL1Chain(2)
	l1Ref1 := chain[1].BlockRef()
	addBatchToL1(t, chain[1], &opderive.SingularBatch{
		ParentHash: safeHead.Hash,
		EpochNum:   rollup.Epoch(l1Ref1.Number),
		EpochHash:  l1Ref1.Hash,
		Timestamp:  safeHead.Time + cfg.BlockTime,
	})
	require.NoError(t, d.AddL1Block(*chain[0], *chain[1]))

	// Derive the first block.
	attrs, _, err := d.Next(safeHead)
	require.NoError(t, err)
	require.NotNil(t, attrs)

	// Now reset (simulating reorg).
	d.Reset(safeHead, sysConfig)

	// Need L1 data again.
	_, _, err = d.Next(safeHead)
	require.True(t, errors.Is(err, ErrNeedL1Data))

	// Re-add blocks, can derive again.
	require.NoError(t, d.AddL1Block(*chain[0], *chain[1]))
	attrs, _, err = d.Next(safeHead)
	require.NoError(t, err)
	require.NotNil(t, attrs)
}

func TestDeriver_ChannelTimeout(t *testing.T) {
	cfg := testRollupConfig()
	cfg.ChannelTimeoutBedrock = 2
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	d, err := NewDeriver(cfg, testL1ChainConfig(), testLogger, safeHead, sysConfig)
	require.NoError(t, err)

	chain := makeTestL1Chain(5)
	l1Block0Ref := chain[0].BlockRef()

	// Incomplete channel at L1 block 1.
	incompleteChID := testChannelID(0xAA)
	channelData := encodeBatchToChannelData(t, &opderive.SingularBatch{
		ParentHash: safeHead.Hash,
		EpochNum:   rollup.Epoch(l1Block0Ref.Number),
		EpochHash:  l1Block0Ref.Hash,
		Timestamp:  safeHead.Time + cfg.BlockTime,
	})
	frame0 := opderive.Frame{
		ID:          incompleteChID,
		FrameNumber: 0,
		Data:        channelData,
		IsLast:      false,
	}
	var buf bytes.Buffer
	buf.WriteByte(params.DerivationVersion0)
	require.NoError(t, frame0.MarshalBinary(&buf))
	chain[1].BatcherData = [][]byte{buf.Bytes()}

	// Complete channel at L1 block 4 (after timeout: 4 > 1 + 2).
	completeChID := testChannelID(0xBB)
	completeBatch := &opderive.SingularBatch{
		ParentHash: safeHead.Hash,
		EpochNum:   rollup.Epoch(l1Block0Ref.Number),
		EpochHash:  l1Block0Ref.Hash,
		Timestamp:  safeHead.Time + cfg.BlockTime,
	}
	completeChannelData := encodeBatchToChannelData(t, completeBatch)
	completeTx := wrapInFrames(completeChannelData, completeChID)
	chain[4].BatcherData = [][]byte{completeTx}

	for _, block := range chain {
		require.NoError(t, d.AddL1Block(*block))
	}

	attrs, _, err := d.Next(safeHead)
	require.NoError(t, err)
	require.NotNil(t, attrs)
	require.Equal(t, hexutil.Uint64(safeHead.Time+cfg.BlockTime), attrs.Timestamp)
}

func TestDeriver_InvalidBatchDropped(t *testing.T) {
	cfg := testRollupConfig()
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	d, err := NewDeriver(cfg, testL1ChainConfig(), testLogger, safeHead, sysConfig)
	require.NoError(t, err)

	chain := makeTestL1Chain(2)
	l1Ref1 := chain[1].BlockRef()

	// Batch with wrong timestamp.
	addBatchToL1(t, chain[1], &opderive.SingularBatch{
		ParentHash: safeHead.Hash,
		EpochNum:   rollup.Epoch(l1Ref1.Number),
		EpochHash:  l1Ref1.Hash,
		Timestamp:  safeHead.Time + cfg.BlockTime + 999,
	})

	require.NoError(t, d.AddL1Block(*chain[0], *chain[1]))

	// The invalid batch should be dropped, returning ErrNeedL1Data.
	_, _, err = d.Next(safeHead)
	require.True(t, errors.Is(err, ErrNeedL1Data))
}

func TestDeriver_ParentHashCheck(t *testing.T) {
	cfg := testRollupConfig()
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	d, err := NewDeriver(cfg, testL1ChainConfig(), testLogger, safeHead, sysConfig)
	require.NoError(t, err)

	chain := makeTestL1Chain(2)
	l1Ref1 := chain[1].BlockRef()

	// Create a batch with WRONG parent hash.
	addBatchToL1(t, chain[1], &opderive.SingularBatch{
		ParentHash: common.HexToHash("0xwrongparent"),
		EpochNum:   rollup.Epoch(l1Ref1.Number),
		EpochHash:  l1Ref1.Hash,
		Timestamp:  safeHead.Time + cfg.BlockTime,
	})

	require.NoError(t, d.AddL1Block(*chain[0], *chain[1]))

	// CheckBatch will reject this because ParentHash != safeHead.Hash.
	_, _, err = d.Next(safeHead)
	require.True(t, errors.Is(err, ErrNeedL1Data))
}

func TestDeriver_RejectsPreKarst(t *testing.T) {
	cfg := testRollupConfig()
	cfg.KarstTime = nil
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	_, err := NewDeriver(cfg, testL1ChainConfig(), testLogger, safeHead, sysConfig)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Karst fork")
}

func TestDeriver_MultipleChannelsAndEpochs(t *testing.T) {
	cfg := testRollupConfig()
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	d, err := NewDeriver(cfg, testL1ChainConfig(), testLogger, safeHead, sysConfig)
	require.NoError(t, err)

	chain := makeTestL1Chain(4)

	// Parent hashes must match the safe head hashes we'll pass to Next().
	// We use a deterministic scheme: genesis hash, then Hash{1}, Hash{2}, etc.
	l2Hashes := []common.Hash{
		safeHead.Hash,
		{1},
		{2},
		{3},
	}

	// Block 1: batch for epoch 1
	l1Ref1 := chain[1].BlockRef()
	addBatchToL1(t, chain[1], &opderive.SingularBatch{
		ParentHash: l2Hashes[0],
		EpochNum:   rollup.Epoch(l1Ref1.Number),
		EpochHash:  l1Ref1.Hash,
		Timestamp:  safeHead.Time + cfg.BlockTime,
	})

	// Block 2: batch for epoch 2
	l1Ref2 := chain[2].BlockRef()
	chData2 := encodeBatchToChannelData(t, &opderive.SingularBatch{
		ParentHash: l2Hashes[1],
		EpochNum:   rollup.Epoch(l1Ref2.Number),
		EpochHash:  l1Ref2.Hash,
		Timestamp:  safeHead.Time + 2*cfg.BlockTime,
	})
	var chID2 opderive.ChannelID
	chID2[0] = 0x02
	chain[2].BatcherData = [][]byte{wrapInFrames(chData2, chID2)}

	// Block 3: batch for epoch 3
	l1Ref3 := chain[3].BlockRef()
	chData3 := encodeBatchToChannelData(t, &opderive.SingularBatch{
		ParentHash: l2Hashes[2],
		EpochNum:   rollup.Epoch(l1Ref3.Number),
		EpochHash:  l1Ref3.Hash,
		Timestamp:  safeHead.Time + 3*cfg.BlockTime,
	})
	var chID3 opderive.ChannelID
	chID3[0] = 0x03
	chain[3].BatcherData = [][]byte{wrapInFrames(chData3, chID3)}

	for _, block := range chain {
		require.NoError(t, d.AddL1Block(*block))
	}

	var derived []*eth.PayloadAttributes
	currentSafeHead := safeHead
	for {
		attrs, _, err := d.Next(currentSafeHead)
		if errors.Is(err, ErrNeedL1Data) {
			break
		}
		require.NoError(t, err)
		require.NotNil(t, attrs)
		derived = append(derived, attrs)

		epochIdx := len(derived) // epoch advances 1:1 with derived blocks here
		currentSafeHead = eth.L2BlockRef{
			Hash:           l2Hashes[epochIdx],
			Number:         currentSafeHead.Number + 1,
			Time:           uint64(attrs.Timestamp),
			L1Origin:       chain[epochIdx].BlockRef().ID(),
			SequenceNumber: 0,
		}
	}

	require.Greater(t, len(derived), 1, "should derive multiple blocks from multiple epochs")

	expectedTimestamp := safeHead.Time + cfg.BlockTime
	for i, attrs := range derived {
		require.Equal(t, hexutil.Uint64(expectedTimestamp), attrs.Timestamp,
			"block %d should have timestamp %d", i, expectedTimestamp)
		expectedTimestamp += cfg.BlockTime
	}
}
