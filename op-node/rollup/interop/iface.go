package interop

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/HashKeyChain/verse/op-node/rollup"
	"github.com/HashKeyChain/verse/op-node/rollup/interop/indexing"
	"github.com/HashKeyChain/verse/op-service/event"
	opmetrics "github.com/HashKeyChain/verse/op-service/metrics"
)

type SubSystem interface {
	event.Deriver
	event.AttachEmitter
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

var _ SubSystem = (*indexing.IndexingMode)(nil)

type L1Source interface {
	indexing.L1Source
}

type L2Source interface {
	indexing.L2Source
}

type Setup interface {
	Setup(ctx context.Context, logger log.Logger, rollupCfg *rollup.Config, l1 L1Source, l2 L2Source, m opmetrics.RPCMetricer) (SubSystem, error)
	Check() error
}
