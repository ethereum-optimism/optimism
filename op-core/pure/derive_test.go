package pure

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive/params"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestPureDerive_SingleBatch(t *testing.T) {
	cfg := testRollupConfig()
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	l1Origin := makeTestL1Input(0) // safe head's L1 origin
	l1 := makeL1WithBatch(t, cfg, 1, safeHead, sysConfig)

	derived, err := PureDerive(cfg, safeHead, sysConfig, []L1Input{*l1Origin, *l1})
	require.NoError(t, err)
	require.Len(t, derived, 1)

	block := derived[0]
	require.Equal(t, hexutil.Uint64(safeHead.Time+cfg.BlockTime), block.Attributes.Timestamp)
	require.True(t, block.Attributes.NoTxPool)
	require.Equal(t, l1.BlockRef(), block.DerivedFrom)
}

func TestPureDerive_EmptyEpoch(t *testing.T) {
	cfg := testRollupConfig()
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	// Create SeqWindowSize + 2 L1 blocks with no batcher data.
	// The sequencer window expires once we get far enough ahead of the cursor's L1 origin.
	numBlocks := cfg.SeqWindowSize + 2
	l1Blocks := make([]L1Input, numBlocks)
	for i := uint64(0); i < numBlocks; i++ {
		l1Blocks[i] = *makeTestL1Input(i)
	}

	derived, err := PureDerive(cfg, safeHead, sysConfig, l1Blocks)
	require.NoError(t, err)
	require.Greater(t, len(derived), 0, "empty batches should be generated when sequencer window expires")

	// Each derived block should have sequential timestamps.
	expectedTimestamp := safeHead.Time + cfg.BlockTime
	for _, block := range derived {
		require.Equal(t, hexutil.Uint64(expectedTimestamp), block.Attributes.Timestamp)
		expectedTimestamp += cfg.BlockTime
	}
}

func TestPureDerive_MultipleChannelsAndEpochs(t *testing.T) {
	cfg := testRollupConfig()
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	l1Blocks := makeMultiEpochL1Inputs(t, cfg, safeHead, sysConfig)

	derived, err := PureDerive(cfg, safeHead, sysConfig, l1Blocks)
	require.NoError(t, err)
	require.Greater(t, len(derived), 1, "should derive multiple blocks from multiple epochs")

	// Each derived block should have sequential timestamps.
	expectedTimestamp := safeHead.Time + cfg.BlockTime
	for i, block := range derived {
		require.Equal(t, hexutil.Uint64(expectedTimestamp), block.Attributes.Timestamp,
			"block %d should have timestamp %d", i, expectedTimestamp)
		expectedTimestamp += cfg.BlockTime
	}
}

func TestPureDerive_ChannelTimeout(t *testing.T) {
	cfg := testRollupConfig()
	cfg.ChannelTimeoutBedrock = 2 // short timeout for testing
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	l1Block0 := makeTestL1Input(0) // safe head's L1 origin
	l1Block0Ref := l1Block0.BlockRef()

	// Create an incomplete channel at L1 block 1 (frame 0, not last).
	incompleteL1 := makeTestL1Input(1)
	incompleteChID := testChannelID(0xAA)

	channelData := encodeBatchToChannelData(t, &derive.SingularBatch{
		ParentHash: safeHead.Hash,
		EpochNum:   rollup.Epoch(l1Block0Ref.Number),
		EpochHash:  l1Block0Ref.Hash,
		Timestamp:  safeHead.Time + cfg.BlockTime,
	})

	frame0 := derive.Frame{
		ID:          incompleteChID,
		FrameNumber: 0,
		Data:        channelData,
		IsLast:      false,
	}
	var buf bytes.Buffer
	buf.WriteByte(params.DerivationVersion0)
	require.NoError(t, frame0.MarshalBinary(&buf))
	incompleteL1.BatcherData = [][]byte{buf.Bytes()}

	// L1 blocks: 0 (origin), 1 (incomplete channel), 2, 3, 4 (complete channel).
	// Timeout fires at block 4 (4 > 1 + 2). No empty batches generated
	// because we're well within SeqWindowSize (10).
	var l1Blocks []L1Input
	l1Blocks = append(l1Blocks, *l1Block0)
	l1Blocks = append(l1Blocks, *incompleteL1)
	l1Blocks = append(l1Blocks, *makeTestL1Input(2))
	l1Blocks = append(l1Blocks, *makeTestL1Input(3))

	// Complete channel at L1 block 4 (after timeout).
	completeL1 := makeTestL1Input(4)
	completeChID := testChannelID(0xBB)

	completeBatch := &derive.SingularBatch{
		ParentHash: safeHead.Hash,
		EpochNum:   rollup.Epoch(l1Block0Ref.Number),
		EpochHash:  l1Block0Ref.Hash,
		Timestamp:  safeHead.Time + cfg.BlockTime,
	}
	completeChannelData := encodeBatchToChannelData(t, completeBatch)
	completeTx := wrapInFrames(completeChannelData, completeChID)
	completeL1.BatcherData = [][]byte{completeTx}
	l1Blocks = append(l1Blocks, *completeL1)

	derived, err := PureDerive(cfg, safeHead, sysConfig, l1Blocks)
	require.NoError(t, err)

	// The incomplete channel timed out. Only the complete channel produces a block.
	foundFromComplete := false
	for _, block := range derived {
		if uint64(block.Attributes.Timestamp) == safeHead.Time+cfg.BlockTime {
			foundFromComplete = true
			break
		}
	}
	require.True(t, foundFromComplete, "should derive block from complete channel after timeout")
}

func TestPureDerive_InvalidBatchSkipped(t *testing.T) {
	cfg := testRollupConfig()
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	l1 := makeTestL1Input(1)
	l1Ref := l1.BlockRef()

	// Create a batch with wrong timestamp (should be safeHead.Time + BlockTime).
	invalidBatch := &derive.SingularBatch{
		ParentHash: safeHead.Hash,
		EpochNum:   rollup.Epoch(l1Ref.Number),
		EpochHash:  l1Ref.Hash,
		Timestamp:  safeHead.Time + cfg.BlockTime + 999, // wrong timestamp
	}

	channelData := encodeBatchToChannelData(t, invalidBatch)
	var chID derive.ChannelID
	copy(chID[:], common.Hex2Bytes("cccccccccccccccccccccccccccccccc"))
	batcherTx := wrapInFrames(channelData, chID)
	l1.BatcherData = [][]byte{batcherTx}

	l1Origin := makeTestL1Input(0) // safe head's L1 origin
	derived, err := PureDerive(cfg, safeHead, sysConfig, []L1Input{*l1Origin, *l1})
	require.NoError(t, err)
	require.Empty(t, derived, "invalid batch should be skipped without error")
}

func TestPureDerive_RejectsPreKarst(t *testing.T) {
	cfg := testRollupConfig()
	cfg.KarstTime = nil // disable Karst
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	_, err := PureDerive(cfg, safeHead, sysConfig, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Karst fork")
}

func TestPureDerive_ValidatesL1BlockRange(t *testing.T) {
	cfg := testRollupConfig()
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	// Start L1 blocks after the safe head's L1 origin (gap)
	l1Blocks := []L1Input{*makeTestL1Input(5)}

	_, err := PureDerive(cfg, safeHead, sysConfig, l1Blocks)
	require.Error(t, err)
	require.Contains(t, err.Error(), "l1Blocks start at")
}

func TestPureDerive_EmptyL1Blocks(t *testing.T) {
	cfg := testRollupConfig()
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	derived, err := PureDerive(cfg, safeHead, sysConfig, nil)
	require.NoError(t, err)
	require.Nil(t, derived)
}

// makeMultiEpochL1Inputs builds several L1 blocks with batches at different
// epochs, suitable for testing multi-channel, multi-epoch derivation.
func makeMultiEpochL1Inputs(t *testing.T, cfg *rollup.Config, safeHead eth.L2BlockRef, sysConfig eth.SystemConfig) []L1Input {
	t.Helper()
	_ = sysConfig

	// Block 1: batch for epoch 1, timestamp = safeHead.Time + BlockTime
	l1Block1 := makeTestL1Input(1)
	l1Ref1 := l1Block1.BlockRef()
	batch1 := &derive.SingularBatch{
		ParentHash: safeHead.Hash,
		EpochNum:   rollup.Epoch(l1Ref1.Number),
		EpochHash:  l1Ref1.Hash,
		Timestamp:  safeHead.Time + cfg.BlockTime,
	}
	chData1 := encodeBatchToChannelData(t, batch1)
	var chID1 derive.ChannelID
	chID1[0] = 0x01
	l1Block1.BatcherData = [][]byte{wrapInFrames(chData1, chID1)}

	// Block 2: batch for epoch 2, timestamp = safeHead.Time + 2*BlockTime
	l1Block2 := makeTestL1Input(2)
	l1Ref2 := l1Block2.BlockRef()
	batch2 := &derive.SingularBatch{
		EpochNum:  rollup.Epoch(l1Ref2.Number),
		EpochHash: l1Ref2.Hash,
		Timestamp: safeHead.Time + 2*cfg.BlockTime,
	}
	chData2 := encodeBatchToChannelData(t, batch2)
	var chID2 derive.ChannelID
	chID2[0] = 0x02
	l1Block2.BatcherData = [][]byte{wrapInFrames(chData2, chID2)}

	// Block 3: batch for epoch 3, timestamp = safeHead.Time + 3*BlockTime
	l1Block3 := makeTestL1Input(3)
	l1Ref3 := l1Block3.BlockRef()
	batch3 := &derive.SingularBatch{
		EpochNum:  rollup.Epoch(l1Ref3.Number),
		EpochHash: l1Ref3.Hash,
		Timestamp: safeHead.Time + 3*cfg.BlockTime,
	}
	chData3 := encodeBatchToChannelData(t, batch3)
	var chID3 derive.ChannelID
	chID3[0] = 0x03
	l1Block3.BatcherData = [][]byte{wrapInFrames(chData3, chID3)}

	// Include block 0 (safe head's L1 origin) at the start.
	l1Block0 := makeTestL1Input(0)
	return []L1Input{*l1Block0, *l1Block1, *l1Block2, *l1Block3}
}

// Verify that test inputs are constructed correctly through BlockRef/BlockID.
func TestL1InputIntegration(t *testing.T) {
	l1 := makeTestL1Input(10)
	ref := l1.BlockRef()
	require.Equal(t, bigs.Uint64Strict(l1.Header.Number), ref.Number)
	require.Equal(t, l1.Header.Hash(), ref.Hash)
	require.Equal(t, l1.Header.ParentHash, ref.ParentHash)
	require.Equal(t, l1.Header.Time, ref.Time)

	id := l1.BlockID()
	require.Equal(t, ref.Hash, id.Hash)
	require.Equal(t, ref.Number, id.Number)
}
