package driver

import (
	"testing"
)

func TestSequencer(t *testing.T) {
	// TODO: we should cover the sequencing logic, but mock the external dependencies to cover all edge-cases.
	// To do so, we need to refactor the package calls to something like:
	// interface {
	//	   PreparePayloadAttributes(ctx context.Context, l2Parent eth.L2BlockRef, timestamp uint64, epoch eth.BlockID) (attrs *eth.PayloadAttributes, err error)
	//	   StartPayload(ctx context.Context, fc eth.ForkchoiceState, attrs *eth.PayloadAttributes) (id eth.PayloadID, errType derive.BlockInsertionErrType, err error)
	//	   ConfirmPayload(ctx context.Context, fc eth.ForkchoiceState, id eth.PayloadID, updateSafe bool) (out *eth.ExecutionPayload, errTyp derive.BlockInsertionErrType, err error)
	// }
	// So we don't have to mock even deeper levels of block production things that can better be covered by individual unit tests.
	// And then cover the sequencing edge cases.
	// ToB only covered:
	// - picking a bad timestamp  (not possible anymore now that we refactored the Sequencer away from the Driver, but we can still fuzz block creation and see if the timestamp is valid)
	//
	// We can additionally the different L2 heads that we may sequence upon, relating to the L1 origin:
	//    - with old L1 origin
	//    - with unknown L1 origin
	//    - with new L1 origin
	//    - with valid L1 origin
	//    - old enough origin to hit seq time drift
	t.Skipf("todo sequencer payload start/confirm payload methods refactor")
}
