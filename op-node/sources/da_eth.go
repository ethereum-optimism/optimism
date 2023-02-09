package sources

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/sources/caching"

	"github.com/ethereum-optimism/optimism/op-node/da"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// EthereumDA satisfies DAChain interface
type EthereumDA struct {
	client *L1Client
}

func (e EthereumDA) TxByBlockNumberAndIndex(ctx context.Context, number uint64, index uint64) (*types.Transaction, error) {
	txs, err := e.TxsByNumber(ctx, number)
	if err != nil {
		return &types.Transaction{}, err
	}
	return txs[index], nil
}

func (e EthereumDA) FetchFrame(ctx context.Context, ref da.FrameRef) (da.Frame, error) {
	// fetching the tx not from API or db - it will just be fetching from the
	// existing types.Transactions by index
	// return types.Transactions[index]
	tx, err := e.TxByBlockNumberAndIndex(ctx, ref.Number, ref.Index)
	if err != nil {
		return da.Frame{}, err
	}
	return da.Frame{tx.Data(), &da.FrameRef{
		Number: ref.Number,
		Index:  ref.Index,
	}}, nil
}

func (e EthereumDA) WriteFrame(ctx context.Context, data []byte) (da.FrameRef, error) {
	return da.FrameRef{}, nil
}

func (e EthereumDA) TxsByNumber(ctx context.Context, number uint64) (types.Transactions, error) {
	// TODO: caching
	_, txs, err := e.client.InfoAndTxsByNumber(ctx, number)
	return txs, err
}

var _ da.DAChain = (&EthereumDA{})

// NewEthereumDA creates a new Ethereum DA
func NewEthereumDA(log log.Logger, client client.RPC, metrics caching.Metrics) (*EthereumDA, error) {
	ethClient, err := NewL1Client(
		client, log, metrics, L1ClientDefaultConfig(&rollup.Config{SeqWindowSize: 10}, true, RPCKindBasic),
	)
	if err != nil {
		return nil, err
	}
	return &EthereumDA{
		client: ethClient,
	}, nil
}
