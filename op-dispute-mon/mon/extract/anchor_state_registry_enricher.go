package extract

import (
	"context"
	"errors"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/log"
)

var _ Enricher = (*AnchorStateRegistryEnricher)(nil)

// AnchorStateRegistryEnricher records the address of the AnchorStateRegistry each game builds on.
// This is best-effort: a game whose contract version does not expose anchorStateRegistry() is skipped
// silently, and any other read error is logged but never fails enrichment, so a failure to read the
// anchor state registry cannot drop the game from the rest of the monitoring.
type AnchorStateRegistryEnricher struct {
	logger log.Logger
}

func NewAnchorStateRegistryEnricher(logger log.Logger) *AnchorStateRegistryEnricher {
	return &AnchorStateRegistryEnricher{logger: logger}
}

func (e *AnchorStateRegistryEnricher) Enrich(ctx context.Context, block rpcblock.Block, caller GameCaller, game *monTypes.EnrichedGameData) error {
	addr, err := caller.GetAnchorStateRegistry(ctx, block)
	if errors.Is(err, contracts.ErrAnchorStateRegistryNotSupported) {
		return nil
	}
	if err != nil {
		e.logger.Warn("Failed to retrieve anchor state registry", "game", game.Proxy, "err", err)
		return nil
	}
	game.AnchorStateRegistry = addr
	return nil
}
