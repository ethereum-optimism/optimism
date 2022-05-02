package rollupclient

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/opnode/l2"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type RollupClient struct {
	rpc *rpc.Client
}

func NewRollupClient(rpc *rpc.Client) *RollupClient {
	return &RollupClient{rpc}
}

func (r *RollupClient) OutputAtBlock(ctx context.Context, blockNum *big.Int) ([]l2.Bytes32, error) {
	var output []l2.Bytes32
	err := r.rpc.CallContext(ctx, &output, "optimism_outputAtBlock", hexutil.EncodeBig(blockNum))
	return output, err
}
