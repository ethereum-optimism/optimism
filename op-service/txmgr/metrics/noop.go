package metrics

import "github.com/ethereum/go-ethereum/core/types"

type NoopTxMetrics struct{}

func (*NoopTxMetrics) RecordNonce(uint64)                {}
func (*NoopTxMetrics) RecordPendingTx(int64)             {}
func (*NoopTxMetrics) RecordGasBumpCount(int)            {}
func (*NoopTxMetrics) RecordTxConfirmationLatency(int64) {}
func (*NoopTxMetrics) TxConfirmed(*types.Receipt)        {}
func (*NoopTxMetrics) TxPublished(string)                {}
func (*NoopTxMetrics) RPCError()                         {}
