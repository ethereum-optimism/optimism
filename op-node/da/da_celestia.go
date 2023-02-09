package da

import (
	"context"
	"time"

	"github.com/celestiaorg/go-cnc"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// CelestiaDA satisfies DAChain interface
type CelestiaDA struct {
	cfg    *rollup.DAConfig
	client *cnc.Client
}

func (c CelestiaDA) FetchFrame(ctx context.Context, ref FrameRef) (Frame, error) {
	tx, err := c.client.NamespacedData(ctx, c.cfg.NamespaceId, ref.Number)
	if err != nil {
		return Frame{}, err
	}
	return Frame{tx[ref.Index], &FrameRef{
		Number: ref.Number,
		Index:  ref.Index,
	}}, nil
}

func (c CelestiaDA) WriteFrame(ctx context.Context, data []byte) (FrameRef, error) {
	return FrameRef{}, nil
}

func (c CelestiaDA) TxsByNumber(ctx context.Context, number uint64) (types.Transactions, error) {
	return nil, nil
}

var _ DAChain = (&CelestiaDA{})

// NewCelestiaDA creates a new Celestia client
func NewCelestiaDA(cfg *rollup.DAConfig, log log.Logger) (*CelestiaDA, error) {
	client, err := cnc.NewClient(cfg.Rpc, cnc.WithTimeout(90*time.Second))
	if err != nil {
		return nil, err
	}
	return &CelestiaDA{
		cfg:    cfg,
		client: client,
	}, nil
}
