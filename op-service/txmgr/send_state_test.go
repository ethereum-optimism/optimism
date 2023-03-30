package txmgr_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

var (
	testHash = common.HexToHash("0x01")
)

const testSafeAbortNonceTooLowCount = 3

func newSendState() *txmgr.SendState {
	return newSendStateWithTimeout(time.Hour, time.Now)
}

func newSendStateWithTimeout(t time.Duration, now func() time.Time) *txmgr.SendState {
	return txmgr.NewSendStateWithNow(testSafeAbortNonceTooLowCount, t, now)
}

func processNSendErrors(sendState *txmgr.SendState, err error, n int) {
	for i := 0; i < n; i++ {
		sendState.ProcessSendError(err)
	}
}

// TestSendStateNoAbortAfterInit asserts that the default SendState won't
// trigger an abort even after the safe abort interval has elapsed.
func TestSendStateNoAbortAfterInit(t *testing.T) {
	sendState := newSendState()
	require.False(t, sendState.ShouldAbortImmediately())
	require.False(t, sendState.IsWaitingForConfirmation())
}

// TestSendStateNoAbortAfterProcessNilError asserts that nil errors are not
// considered for abort status.
func TestSendStateNoAbortAfterProcessNilError(t *testing.T) {
	sendState := newSendState()

	processNSendErrors(sendState, nil, testSafeAbortNonceTooLowCount)
	require.False(t, sendState.ShouldAbortImmediately())
}

// TestSendStateNoAbortAfterProcessOtherError asserts that non-nil errors other
// than ErrNonceTooLow are not considered for abort status.
func TestSendStateNoAbortAfterProcessOtherError(t *testing.T) {
	sendState := newSendState()

	otherError := errors.New("other error")
	processNSendErrors(sendState, otherError, testSafeAbortNonceTooLowCount)
	require.False(t, sendState.ShouldAbortImmediately())
}

// TestSendStateAbortSafelyAfterNonceTooLowButNoTxMined asserts that we will
// abort after the safe abort interval has elapsed if we haven't mined a tx.
func TestSendStateAbortSafelyAfterNonceTooLowButNoTxMined(t *testing.T) {
	sendState := newSendState()

	sendState.ProcessSendError(core.ErrNonceTooLow)
	require.False(t, sendState.ShouldAbortImmediately())
	sendState.ProcessSendError(core.ErrNonceTooLow)
	require.False(t, sendState.ShouldAbortImmediately())
	sendState.ProcessSendError(core.ErrNonceTooLow)
	require.True(t, sendState.ShouldAbortImmediately())
}

// TestSendStateMiningTxCancelsAbort asserts that a tx getting mined after
// processing ErrNonceTooLow takes precedence and doesn't cause an abort.
func TestSendStateMiningTxCancelsAbort(t *testing.T) {
	sendState := newSendState()

	sendState.ProcessSendError(core.ErrNonceTooLow)
	sendState.ProcessSendError(core.ErrNonceTooLow)
	sendState.TxMined(testHash)
	require.False(t, sendState.ShouldAbortImmediately())
	sendState.ProcessSendError(core.ErrNonceTooLow)
	require.False(t, sendState.ShouldAbortImmediately())
}

// TestSendStateReorgingTxResetsAbort asserts that unmining a tx does not
// consider ErrNonceTooLow's prior to being mined when determining whether
// to abort.
func TestSendStateReorgingTxResetsAbort(t *testing.T) {
	sendState := newSendState()

	sendState.ProcessSendError(core.ErrNonceTooLow)
	sendState.ProcessSendError(core.ErrNonceTooLow)
	sendState.TxMined(testHash)
	sendState.TxNotMined(testHash)
	sendState.ProcessSendError(core.ErrNonceTooLow)
	require.False(t, sendState.ShouldAbortImmediately())
}

// TestSendStateNoAbortEvenIfNonceTooLowAfterTxMined asserts that we will not
// abort if we continue to get ErrNonceTooLow after a tx has been mined.
//
// NOTE: This is the most crucial role of the SendState, as we _expect_ to get
// ErrNonceTooLow failures after one of our txs has been mined, but that
// shouldn't cause us to not continue waiting for confirmations.
func TestSendStateNoAbortEvenIfNonceTooLowAfterTxMined(t *testing.T) {
	sendState := newSendState()

	sendState.TxMined(testHash)
	processNSendErrors(
		sendState, core.ErrNonceTooLow, testSafeAbortNonceTooLowCount,
	)
	require.False(t, sendState.ShouldAbortImmediately())
}

// TestSendStateSafeAbortIfNonceTooLowPersistsAfterUnmine asserts that we will
// correctly abort if we continue to get ErrNonceTooLow after a tx is unmined
// but not remined.
func TestSendStateSafeAbortIfNonceTooLowPersistsAfterUnmine(t *testing.T) {
	sendState := newSendState()

	sendState.TxMined(testHash)
	sendState.TxNotMined(testHash)
	sendState.ProcessSendError(core.ErrNonceTooLow)
	sendState.ProcessSendError(core.ErrNonceTooLow)
	require.False(t, sendState.ShouldAbortImmediately())
	sendState.ProcessSendError(core.ErrNonceTooLow)
	require.True(t, sendState.ShouldAbortImmediately())
}

// TestSendStateSafeAbortWhileCallingNotMinedOnUnminedTx asserts that we will
// correctly abort if we continue to call TxNotMined on txns that haven't been
// mined.
func TestSendStateSafeAbortWhileCallingNotMinedOnUnminedTx(t *testing.T) {
	sendState := newSendState()

	processNSendErrors(
		sendState, core.ErrNonceTooLow, testSafeAbortNonceTooLowCount,
	)
	sendState.TxNotMined(testHash)
	require.True(t, sendState.ShouldAbortImmediately())
}

// TestSendStateIsWaitingForConfirmationAfterTxMined asserts that we are waiting
// for confirmation after a tx is mined.
func TestSendStateIsWaitingForConfirmationAfterTxMined(t *testing.T) {
	sendState := newSendState()

	testHash2 := common.HexToHash("0x02")

	sendState.TxMined(testHash)
	require.True(t, sendState.IsWaitingForConfirmation())
	sendState.TxMined(testHash2)
	require.True(t, sendState.IsWaitingForConfirmation())
}

// TestSendStateIsNotWaitingForConfirmationAfterTxUnmined asserts that we are
// not waiting for confirmation after a tx is mined then unmined.
func TestSendStateIsNotWaitingForConfirmationAfterTxUnmined(t *testing.T) {
	sendState := newSendState()

	sendState.TxMined(testHash)
	sendState.TxNotMined(testHash)
	require.False(t, sendState.IsWaitingForConfirmation())
}

func stepClock(step time.Duration) func() time.Time {
	i := 0
	return func() time.Time {
		var start time.Time
		i += 1
		return start.Add(time.Duration(i) * step)
	}
}

// TestSendStateTimeoutAbort ensure that this will abort if it passes the tx pool timeout
// when no successful transactions have been recorded
func TestSendStateTimeoutAbort(t *testing.T) {
	sendState := newSendStateWithTimeout(10*time.Millisecond, stepClock(20*time.Millisecond))
	require.True(t, sendState.ShouldAbortImmediately(), "Should abort after timing out")
}

// TestSendStateNoTimeoutAbortIfPublishedTx ensure that this will not abort if there is
// a successful transaction send.
func TestSendStateNoTimeoutAbortIfPublishedTx(t *testing.T) {
	sendState := newSendStateWithTimeout(10*time.Millisecond, stepClock(20*time.Millisecond))
	sendState.ProcessSendError(nil)
	require.False(t, sendState.ShouldAbortImmediately(), "Should not abort if published transcation successfully")
}
