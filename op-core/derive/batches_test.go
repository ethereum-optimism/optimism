package derive

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	opderive "github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestDecodeBatches_SingularBatch(t *testing.T) {
	cfg := testRollupConfig()
	safeHead := testSafeHead(cfg)
	l1Ref := testL1Ref(1)

	batch := &opderive.SingularBatch{
		ParentHash: safeHead.Hash,
		EpochNum:   rollup.Epoch(l1Ref.Number),
		EpochHash:  l1Ref.Hash,
		Timestamp:  safeHead.Time + cfg.BlockTime,
	}

	channelData := encodeBatchToChannelData(t, batch)

	cursor := newCursor(safeHead)
	l1Origins := []eth.L1BlockRef{testL1Ref(0), l1Ref}

	batches := decodeBatches(testLogger, bytes.NewReader(channelData), cfg, l1Origins, cursor, l1Ref)
	require.Len(t, batches, 1)

	decoded := batches[0]
	require.Equal(t, batch.ParentHash, decoded.ParentHash)
	require.Equal(t, batch.EpochNum, decoded.EpochNum)
	require.Equal(t, batch.EpochHash, decoded.EpochHash)
	require.Equal(t, batch.Timestamp, decoded.Timestamp)
}
