package mon

import (
	"context"
	"math"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type AnchorStateMetrics interface {
	RecordAnchorStateL2SequenceNumber(anchorStateRegistry common.Address, l2SequenceNumber uint64)
}

// AnchorRootProvider fetches the anchor root for a single AnchorStateRegistry.
type AnchorRootProvider interface {
	GetAnchorRoot(ctx context.Context, block rpcblock.Block) (common.Hash, *big.Int, error)
}

type AnchorRootProviderCreator func(addr common.Address) AnchorRootProvider

type AnchorStateMonitor struct {
	logger      log.Logger
	metrics     AnchorStateMetrics
	newContract AnchorRootProviderCreator
}

func NewAnchorStateMonitor(logger log.Logger, metrics AnchorStateMetrics, newContract AnchorRootProviderCreator) *AnchorStateMonitor {
	return &AnchorStateMonitor{
		logger:      logger,
		metrics:     metrics,
		newContract: newContract,
	}
}

// CheckAnchorState reads the current anchor state L2 sequence number for every distinct
// AnchorStateRegistry referenced by the monitored games and records it as a metric.
func (m *AnchorStateMonitor) CheckAnchorState(ctx context.Context, blockHash common.Hash, games []*types.EnrichedGameData) {
	seen := make(map[common.Address]bool)
	for _, game := range games {
		asr := game.AnchorStateRegistry
		if asr == (common.Address{}) || seen[asr] {
			continue
		}
		seen[asr] = true
		_, l2SequenceNumber, err := m.newContract(asr).GetAnchorRoot(ctx, rpcblock.ByHash(blockHash))
		if err != nil {
			m.logger.Warn("Failed to retrieve anchor root", "anchorStateRegistry", asr, "err", err)
			continue
		}
		l2 := uint64(math.MaxUint64)
		if l2SequenceNumber.IsUint64() {
			l2 = bigs.Uint64Strict(l2SequenceNumber)
		}
		m.metrics.RecordAnchorStateL2SequenceNumber(asr, l2)
	}
}
