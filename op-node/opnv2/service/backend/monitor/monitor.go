package monitor

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/rwel"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

type Monitor struct {
	log log.Logger
}

func NewMonitor(log log.Logger) *Monitor {
	return &Monitor{
		log: log,
	}
}

func (m *Monitor) OnEvent(ctx context.Context, ev event.Event) bool {
	// TODO watch for more events, and log the interesting things

	switch x := ev.(type) {
	case rwel.LocalUnsafeUpdateEvent:
		age := time.Unix(int64(x.Ref.Time), 0)
		m.log.Info("New L2 head", "l2_local_unsafe", x.Ref, "l2_time", x.Ref.Time, "age", time.Since(age))
	case rwel.CrossSafeUpdateEvent:
		age := time.Unix(int64(x.CrossSafe.Time), 0)
		m.log.Info("New L2 cross-safe", "l2_cross_unsafe", x.CrossSafe, "age", time.Since(age))
	case rwel.PayloadSuccessEvent:
		logValues := getBlockProcessingMetrics(&x)
		m.log.Info("Processed new L2 unsafe block", logValues...)
	}
	return false
}

// Returns key/value pairs that can be logged and are useful for plotting
// block build/insert time as a way to measure performance.
func getBlockProcessingMetrics(ev *rwel.PayloadSuccessEvent) []any {
	fcuFinish := time.Now()
	payload := ev.Envelope.ExecutionPayload

	logValues := []any{
		"hash", payload.BlockHash,
		"number", uint64(payload.BlockNumber),
		"state_root", payload.StateRoot,
		"timestamp", uint64(payload.Timestamp),
		"parent", payload.ParentHash,
		"prev_randao", payload.PrevRandao,
		"fee_recipient", payload.FeeRecipient,
		"txs", len(payload.Transactions),
	}

	var totalTime time.Duration
	var mgasps float64
	if !ev.BuildStarted.IsZero() {
		totalTime = fcuFinish.Sub(ev.BuildStarted)
		logValues = append(logValues,
			"build_time", common.PrettyDuration(ev.InsertStarted.Sub(ev.BuildStarted)),
			"insert_time", common.PrettyDuration(fcuFinish.Sub(ev.InsertStarted)),
		)
	} else if !ev.InsertStarted.IsZero() {
		totalTime = fcuFinish.Sub(ev.InsertStarted)
	}

	// Avoid divide-by-zero for mgasps
	if totalTime > 0 {
		mgasps = float64(payload.GasUsed) * 1000 / float64(totalTime)
	}

	logValues = append(logValues,
		"total_time", common.PrettyDuration(totalTime),
		"mgas", float64(payload.GasUsed)/1000000,
		"mgasps", mgasps,
	)

	return logValues
}
