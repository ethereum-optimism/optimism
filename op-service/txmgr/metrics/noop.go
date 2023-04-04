package metrics

import "github.com/ethereum/go-ethereum/core/types"

type NoopTxMetrics struct{}

func (*NoopTxMetrics) RecordL1GasFee(*types.Receipt)     {}
func (*NoopTxMetrics) RecordGasBumpCount(int)            {}
func (*NoopTxMetrics) RecordTxConfirmationLatency(int64) {}
