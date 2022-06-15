package rollupclient

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type RollupClient struct {
	rpc *rpc.Client
}

func NewRollupClient(rpc *rpc.Client) *RollupClient {
	return &RollupClient{rpc}
}

func (r *RollupClient) GetBatchBundle(
	ctx context.Context,
	req *node.BatchBundleRequest,
) (*node.BatchBundleResponse, error) {

	var batchResponse = new(node.BatchBundleResponse)
	err := r.rpc.CallContext(ctx, &batchResponse, "optimism_getBatchBundle", req)
	return batchResponse, err
}

func (r *RollupClient) OutputAtBlock(ctx context.Context, blockNum *big.Int) ([]eth.Bytes32, error) {
	var output []eth.Bytes32
	err := r.rpc.CallContext(ctx, &output, "optimism_outputAtBlock", hexutil.EncodeBig(blockNum))
	return output, err
}
