package extract

import (
	"context"

	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/log"
)

// ZKAgreementEnricher reconciles a ZK proposal with configured super-root sources.
type ZKAgreementEnricher struct {
	superAgreement SuperAgreementEnricher
}

var _ ZKEnricher = (*ZKAgreementEnricher)(nil)

func NewZKAgreementEnricher(logger log.Logger, metrics OutputMetrics, clients []SuperRootProvider, cl clock.Clock) *ZKAgreementEnricher {
	return &ZKAgreementEnricher{
		superAgreement: *NewSuperAgreementEnricher(logger, metrics, clients, cl),
	}
}

func (e *ZKAgreementEnricher) Enrich(ctx context.Context, _ rpcblock.Block, _ ZKGameCaller, zkGame *monTypes.ZKGameData) error {
	return e.superAgreement.enrich(ctx, zkGame.Common(), zkGameAgreement)
}
