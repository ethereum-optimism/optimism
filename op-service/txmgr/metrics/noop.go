package metrics

import "github.com/ethereum/go-ethereum/core/types"

type NoopTxMetrics struct{}

func (*NoopTxMetrics) RecordNonce(uint64)         {}
func (*NoopTxMetrics) TxConfirmed(*types.Receipt) {}
func (*NoopTxMetrics) TxPublished(error)          {}
func (*NoopTxMetrics) RPCError()                  {}
