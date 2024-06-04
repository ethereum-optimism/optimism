package extract

import (
	"context"
	"fmt"

	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var _ Enricher = (*L1HeadBlockNumEnricher)(nil)

type BlockFetcher interface {
	HeaderByHash(ctx context.Context, block common.Hash) (*types.Header, error)
}

type L1HeadBlockNumEnricher struct {
	client BlockFetcher
}

func NewL1HeadBlockNumEnricher(client BlockFetcher) *L1HeadBlockNumEnricher {
	return &L1HeadBlockNumEnricher{client: client}
}

func (e *L1HeadBlockNumEnricher) Enrich(ctx context.Context, _ rpcblock.Block, _ GameCaller, game *monTypes.EnrichedGameData) error {
	header, err := e.client.HeaderByHash(ctx, game.L1Head)
	if err != nil {
		return fmt.Errorf("failed to retrieve header for L1 head block %v: %w", game.L1Head, err)
	}
	game.L1HeadNum = header.Number.Uint64()
	return nil
}
