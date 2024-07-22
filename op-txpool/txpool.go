package op_txpool

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

var (
	MetricsNameSpace = "op_txpool"
)

type TxPool struct {
	conditionalTxService *ConditionalTxService
}

func NewTxPool(ctx context.Context, log log.Logger, m metrics.Factory, cfg *CLIConfig) (*TxPool, error) {
	conditionalTxService, err := NewConditionalService(ctx, log, m, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create conditional tx service: %w", err)
	}

	return &TxPool{conditionalTxService}, nil
}

func (txp *TxPool) GetAPIs() []gethrpc.API {
	return []gethrpc.API{{Namespace: "eth", Service: txp.conditionalTxService}}
}
