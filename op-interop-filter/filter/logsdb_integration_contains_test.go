package filter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
)

func TestIntegration_Contains_HappyPath_AccessListAccepted(t *testing.T) {
	t.Parallel()

	bk := twoChainBackend(t, 1)
	bk.requireAccepted(executingChain(), inclusionTs, bk.sourceAccess(100, 0))
}

func TestIntegration_Contains_FutureLog_RejectedAsInvalidExecutingMessage(t *testing.T) {
	t.Parallel()

	bk := twoChainBackend(t, 1)
	future := bk.sourceAccess(100, 0)
	future.BlockNumber = 200
	future.Timestamp = 1202 // == latestSealedTs on source, satisfies gate, triggers ErrFuture
	bk.requireRejection(executingChain(), inclusionTs, "invalid_executing_message", future)
}

func TestIntegration_Contains_LogIdxOutOfRangeOnSealedBlock_ExpiredMessage(t *testing.T) {
	t.Parallel()

	bk := twoChainBackend(t, 2)
	bad := bk.sourceAccess(100, 0)
	bad.LogIndex = 5
	bk.requireRejection(executingChain(), inclusionTs, "expired_message", bad)
}

func TestIntegration_Contains_BlockPastTipWithStaleLatestTs_ExpiredMessage(t *testing.T) {
	t.Parallel()

	bk := newSeededBackend(t, backendOpts{
		Specs: []seedSpec{
			{ChainID: 901, AnchorNumber: 99, AnchorTime: 1198,
				Blocks: []seedBlock{
					{Num: 100, Ts: 1200, Logs: []seedLog{{}}},
					{Num: 101, Ts: 1500},
				}},
			{ChainID: 902, AnchorNumber: 99, AnchorTime: 1198,
				Blocks: []seedBlock{
					{Num: 100, Ts: 1200},
					{Num: 101, Ts: 1500},
				}},
		},
	})

	bad := bk.sourceAccess(100, 0)
	bad.BlockNumber = 102
	bad.Timestamp = 1300
	bk.requireRejection(executingChain(), 1500, "expired_message", bad)
}

func TestIntegration_Contains_ChecksumMismatch_ExpiredMessage(t *testing.T) {
	t.Parallel()

	bk := twoChainBackend(t, 1)
	bad := withChecksum(bk.sourceAccess(100, 0), [32]byte{0x03, 0xff, 0xff, 0xff, 0xff})
	bk.requireRejection(executingChain(), inclusionTs, "expired_message", bad)
}

func TestIntegration_Contains_StoredTimestampMismatch_ExpiredMessage(t *testing.T) {
	t.Parallel()

	bk := twoChainBackend(t, 1)
	bad := bk.sourceAccess(100, 0)
	bad.Timestamp = 1199 // block 100 was sealed at 1200; gate uses access.Timestamp <= minIngestedTs
	bk.requireRejection(executingChain(), inclusionTs, "expired_message", bad)
}

func TestIntegration_Contains_BelowEarliestStored_RejectedAsInvalidExecutingMessage(t *testing.T) {
	t.Parallel()

	bk := twoChainBackend(t, 1)
	bad := bk.sourceAccess(100, 0)
	bad.BlockNumber = 50
	bad.Timestamp = 1100
	bk.requireRejection(executingChain(), inclusionTs, "invalid_executing_message", bad)
}

func TestIntegration_Contains_UnknownChain_UnknownChainReason(t *testing.T) {
	t.Parallel()

	bk := twoChainBackend(t, 1)
	bk.requireRejection(eth.ChainIDFromUInt64(7777), inclusionTs, "unknown_chain")
}

func TestIntegration_Contains_BeforeInit_Uninitialized(t *testing.T) {
	t.Parallel()

	si := newSeededIngester(t, seedSpec{
		NoSealAnchor: true,
		NoIngest:     true,
	})
	require.NoError(t, si.logsDB.Close())
	si.logsDB = nil

	_, err := si.Contains(messages.ContainsQuery{BlockNum: 100, Timestamp: 1200})
	require.ErrorIs(t, err, types.ErrUninitialized)
}

func TestIntegration_Contains_FailsafeAlreadyEnabled_Rejected(t *testing.T) {
	t.Parallel()

	bk := twoChainBackend(t, 1)
	bk.SetFailsafeEnabled(true)
	bk.requireRejection(executingChain(), inclusionTs, "failsafe", bk.sourceAccess(100, 0))
}
