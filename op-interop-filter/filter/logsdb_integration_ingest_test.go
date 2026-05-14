package filter

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestIntegration_Ingest_HappyPath_SequentialBlocks(t *testing.T) {
	t.Parallel()

	chainID := uint64(901)
	si := newSeededIngester(t, seedSpec{
		ChainID:      chainID,
		AnchorNumber: 99,
		AnchorTime:   1198,
		Blocks: []seedBlock{
			{Num: 100, Ts: 1200, Logs: []seedLog{{}}},
			{Num: 101, Ts: 1202, Logs: []seedLog{{}, {}}},
			{Num: 102, Ts: 1204, Logs: []seedLog{{}}},
		},
	})

	latest, ok := si.LatestBlock()
	require.True(t, ok)
	require.Equal(t, uint64(102), latest.Number)
	require.Nil(t, si.Error())
	require.Equal(t, int64(3), si.metrics.sealedCount(chainID))
	require.Equal(t, int64(4), si.metrics.logsAdded[chainID])
}

func TestIntegration_Ingest_ParentHashMismatch_ErrorReorg(t *testing.T) {
	t.Parallel()

	chainID := uint64(901)
	si := newSeededIngester(t, seedSpec{
		ChainID:      chainID,
		AnchorNumber: 99,
		AnchorTime:   1198,
		Blocks: []seedBlock{
			{Num: 100, Ts: 1200, Logs: []seedLog{{}}},
		},
	})

	wrongParent := common.Hash{0xde, 0xad}
	si.addBlock(101, 1202, wrongParent, []seedLog{{}})

	err := si.ingestBlock(101)
	require.Error(t, err)

	ingErr := si.Error()
	require.NotNil(t, ingErr)
	require.Equal(t, ErrorReorg, ingErr.Reason)
	require.Equal(t, 1, si.metrics.reorgCount(chainID))
}

func TestIntegration_Ingest_NonSequentialBlockNumber_ReturnsError(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		Blocks: []seedBlock{
			{Num: 100, Ts: 1200, Logs: []seedLog{{}}},
		},
	})

	si.addBlock(102, 1204, si.blockInfo[100].hash, []seedLog{{}})

	err := si.ingestBlock(102)
	require.Error(t, err)
	require.Nil(t, si.Error(), "non-sequential should not flip ingester into error state")
}

func TestIntegration_Ingest_InvalidExecutingMessageDecode_ErrorInvalidExecutingMessage(t *testing.T) {
	t.Parallel()

	malformed := &gethTypes.Log{
		Address: params.InteropCrossL2InboxAddress,
		Topics: []common.Hash{
			types.ExecutingMessageEventTopic,
			{0xab, 0xcd},
		},
		Data: []byte{0x01, 0x02, 0x03},
	}

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		NoIngest:     true,
		Blocks: []seedBlock{
			{Num: 100, Ts: 1200, Logs: []seedLog{{Log: malformed}}},
		},
	})

	require.Error(t, si.ingestBlock(100))
	ingErr := si.Error()
	require.NotNil(t, ingErr)
	require.Equal(t, ErrorInvalidExecutingMessage, ingErr.Reason)
}

func TestIntegration_Ingest_AfterIngesterError_SubsequentIngestsSkipped(t *testing.T) {
	t.Parallel()

	chainID := uint64(901)
	si := newSeededIngester(t, seedSpec{
		ChainID:      chainID,
		AnchorNumber: 99,
		AnchorTime:   1198,
		Blocks: []seedBlock{
			{Num: 100, Ts: 1200, Logs: []seedLog{{}}},
		},
	})

	sealedBefore := si.metrics.sealedCount(chainID)
	si.SetError(ErrorConflict, "forced")

	si.addBlock(101, 1202, si.blockInfo[100].hash, []seedLog{{}})

	err := si.ingestBlock(101)
	require.Error(t, err)
	require.Equal(t, sealedBefore, si.metrics.sealedCount(chainID),
		"no further blocks should be sealed once ingester is in error state")
}

// A block with a backwards timestamp is the one path through writeFetchedBlock
// that reaches the logsdb's defensive checks without being pre-guarded — only
// block number and parent hash are pre-checked. The behavioural contract is
// that the resulting SealBlock failure trips the Backend's failsafe state, so
// subsequent CheckAccessList requests are rejected with the failsafe label.
// A logsdb that returns the wrong sentinel here would silently retry instead.
func TestIntegration_Ingest_BackwardsTimestamp_TripsBackendFailsafe(t *testing.T) {
	t.Parallel()

	bk := twoChainBackend(t, 1)
	source := bk.ingesters[eth.ChainIDFromUInt64(901)]

	bk.requireAccepted(executingChain(), inclusionTs, bk.sourceAccess(100, 0))
	require.False(t, bk.FailsafeEnabled())

	source.addBlock(102, 1100, source.blockInfo[101].hash, []seedLog{{}})
	require.Error(t, source.ingestBlock(102))

	require.True(t, bk.FailsafeEnabled(),
		"backwards-timestamp SealBlock failure must transition the Backend into failsafe")
	bk.requireRejection(executingChain(), inclusionTs, "failsafe", bk.sourceAccess(100, 0))
}

func TestIntegration_Ingest_RPCFetchError_LogsAndRetries(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		AnchorNumber: 99,
		AnchorTime:   1198,
		Blocks: []seedBlock{
			{Num: 100, Ts: 1200, Logs: []seedLog{{}}},
		},
	})

	si.addBlock(101, 1202, si.blockInfo[100].hash, []seedLog{{}})
	si.eth.SetFetchReceiptsErr(errors.New("transient rpc failure"))

	err := si.ingestBlock(101)
	require.Error(t, err)
	require.Nil(t, si.Error(), "transient RPC errors must not set IngesterError")

	si.eth.SetFetchReceiptsErr(nil)
	require.NoError(t, si.ingestBlock(101))
	latest, ok := si.LatestBlock()
	require.True(t, ok)
	require.Equal(t, uint64(101), latest.Number)
}
