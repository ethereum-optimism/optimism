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
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestPureDerive_SingleBatch(t *testing.T) {
	cfg := testRollupConfig()
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	l1 := makeL1WithBatch(t, cfg, 1, safeHead, sysConfig)

	derived, err := PureDerive(cfg, safeHead, sysConfig, []L1Input{*l1})
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
	safeHead := testSafeHead(cfg)
	sysConfig := testSystemConfig()

	// Create an incomplete channel at L1 block 1 (frame 0 of 2, not last).
	incompleteL1 := makeTestL1Input(1)
	incompleteChID := testChannelID(0xAA)

	batch := &derive.SingularBatch{
		ParentHash: safeHead.Hash,
		EpochNum:   rollup.Epoch(incompleteL1.Number),
		EpochHash:  incompleteL1.Hash,
		Timestamp:  safeHead.Time + cfg.BlockTime,
	}
	channelData := encodeBatchToChannelData(t, batch)

	// Split into two frames but only include the first (non-last) frame.
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

	// Fill gap L1 blocks until timeout. Channel timeout is 50, so we need
	// blocks 2..52 to cause timeout at block 52.
	var l1Blocks []L1Input
	l1Blocks = append(l1Blocks, *incompleteL1)
	for i := uint64(2); i <= cfg.ChannelTimeoutBedrock+2; i++ {
		l1Blocks = append(l1Blocks, *makeTestL1Input(i))
	}

	// After timeout, add a complete channel.
	completeL1Num := cfg.ChannelTimeoutBedrock + 3
	completeL1 := makeTestL1Input(completeL1Num)
	completeChID := testChannelID(0xBB)

	// The batch must reference an L1 block we have. Use block 1's ref as epoch.
	completeBatch := &derive.SingularBatch{
		ParentHash: safeHead.Hash,
		EpochNum:   rollup.Epoch(1),
		EpochHash:  incompleteL1.Hash,
		Timestamp:  safeHead.Time + cfg.BlockTime,
	}
	completeChannelData := encodeBatchToChannelData(t, completeBatch)
	completeTx := wrapInFrames(completeChannelData, completeChID)
	completeL1.BatcherData = [][]byte{completeTx}
	l1Blocks = append(l1Blocks, *completeL1)

	derived, err := PureDerive(cfg, safeHead, sysConfig, l1Blocks)
	require.NoError(t, err)

	// We should get at least one derived block from the complete channel.
	// The incomplete channel should have timed out and produced nothing.
	foundFromComplete := false
	for _, block := range derived {
		if uint64(block.Attributes.Timestamp) == safeHead.Time+cfg.BlockTime {
			foundFromComplete = true
			break
		}
	}
	require.True(t, foundFromComplete, "should have a derived block from the complete channel after timeout")
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

	derived, err := PureDerive(cfg, safeHead, sysConfig, []L1Input{*l1})
	require.NoError(t, err)
	require.Empty(t, derived, "invalid batch should be skipped without error")
}

func TestFindL1Origin(t *testing.T) {
	l1Blocks := []L1Input{
		*makeTestL1Input(5),
		*makeTestL1Input(10),
		*makeTestL1Input(15),
	}

	found := findL1Origin(l1Blocks, 10)
	require.NotNil(t, found)
	require.Equal(t, uint64(10), found.Number)

	notFound := findL1Origin(l1Blocks, 99)
	require.Nil(t, notFound)
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

	return []L1Input{*l1Block1, *l1Block2, *l1Block3}
}
