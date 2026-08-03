package sdmtest

import (
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
)

// VerifyPostExecSpanCrossesActivation waits for batcher submissions and verifies that one span
// starts before the activation timestamp and contains the selected post-activation PostExec block.
func VerifyPostExecSpanCrossesActivation(
	t devtest.T,
	network *dsl.L2Network,
	l1BlocksToMonitor int,
	activationTime uint64,
	postExecBlockNumber uint64,
) {
	spans := network.DeriveSpanBatches(l1BlocksToMonitor)
	for spanIndex, span := range spans {
		hasPreActivationBatch := false
		hasTargetPostExec := false
		var targetTimestamp uint64
		var targetEntryCount int

		for _, batch := range span.Batches {
			if batch.Timestamp < activationTime {
				hasPreActivationBatch = true
			}
			for _, rawTx := range batch.Transactions {
				if len(rawTx) == 0 || rawTx[0] != optypes.PostExecTxType {
					continue
				}
				postExecTx, err := optypes.UnmarshalPostExecTx(rawTx)
				t.Require().NoError(err, "span PostExec transaction must decode")
				payload, err := optypes.DecodePostExecPayload(postExecTx.Data)
				t.Require().NoError(err, "span PostExec payload must decode")
				if payload.BlockNumber != postExecBlockNumber {
					continue
				}
				t.Require().Greater(batch.Timestamp, activationTime,
					"selected PostExec block must be after the activation block")
				t.Require().NotEmpty(payload.GasRefundEntries,
					"production PostExec payload in span must contain gas refunds")
				hasTargetPostExec = true
				targetTimestamp = batch.Timestamp
				targetEntryCount = len(payload.GasRefundEntries)
			}
		}

		if hasPreActivationBatch && hasTargetPostExec {
			t.Logger().Info("verified PostExec span crosses activation",
				"span_index", spanIndex,
				"span_start_timestamp", span.Batches[0].Timestamp,
				"activation_timestamp", activationTime,
				"post_exec_block", postExecBlockNumber,
				"post_exec_timestamp", targetTimestamp,
				"refund_entries", targetEntryCount)
			return
		}
	}

	t.Require().Fail("missing cross-activation PostExec span",
		"no span started before timestamp %d and contained PostExec block %d; decoded %d spans",
		activationTime, postExecBlockNumber, len(spans))
}
