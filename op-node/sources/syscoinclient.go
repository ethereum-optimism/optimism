package sources

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
)

type SyscoinClient struct {
	rpc client.RPC
}

func NewSyscoinClient(rpc client.RPC) *SyscoinClient {
	return &SyscoinClient{rpc}
}

func (r *SyscoinClient) CreatePoDA(ctx context.Context, data []byte) (common.Hash, error) {
	var output []eth.Bytes32
	err := r.rpc.CallContext(ctx, &output, "syscoincreatenevmblob", data, false)
	return common.Hash{}, err
}

func (r *SyscoinClient) IsPoDAConfirmed(ctx context.Context, vh common.Hash) (bool, error) {
	err := r.rpc.CallContext(ctx, nil, "getnevmblobdata", vh)
	return true, err
}

func (r *SyscoinClient) GetPoDA(ctx context.Context, vh common.Hash) ([]byte, error) {
	err := r.rpc.CallContext(ctx, nil, "getnevmblobdata", vh)
	return nil, err
}