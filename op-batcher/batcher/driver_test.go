// package batcher

// import (
// 	"errors"
// 	"testing"

// 	"github.com/stretchr/testify/require"
// )

// func TestRecordConfirmedTx(t *testing.T) {

// 	bs, err := NewBatchSubmitter
// 	err := InputError{
// 		Inner: errors.New("test error"),
// 		Code:  InvalidForkchoiceState,
// 	}
// 	var x InputError
// 	if !errors.As(err, &x) {
// 		t.Fatalf("need InputError to be detected as such")
// 	}
// 	require.ErrorIs(t, err, InputError{}, "need to detect input error with errors.Is")
// }
