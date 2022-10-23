package sources

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum/go-ethereum/common"
)

type PoDAClient struct {
	rpc client.RPC
}

func NewPoDAClient(rpc client.RPC) *PoDAClient {
	return &PoDAClient{rpc}
}

func (r *PoDAClient) GetPoDA(ctx context.Context, vh common.Hash) ([]byte, error) {
	err := r.rpc.CallContext(ctx, nil, "getnevmblobdata", vh)
	return nil, err
}