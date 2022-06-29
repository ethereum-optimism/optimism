package rollupclient

import (
	"context"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type RollupClient struct {
	rpc *rpc.Client
}

func NewRollupClient(rpc *rpc.Client) *RollupClient {
	return &RollupClient{rpc}
}

func (r *RollupClient) OutputAtBlock(ctx context.Context, blockNum *big.Int) ([]eth.Bytes32, error) {
	var output []eth.Bytes32
	err := r.rpc.CallContext(ctx, &output, "optimism_outputAtBlock", hexutil.EncodeBig(blockNum))
	return output, err
}

func (r *RollupClient) SyncStatus(ctx context.Context) (*driver.SyncStatus, error) {
	var output *driver.SyncStatus
	err := r.rpc.CallContext(ctx, &output, "optimism_syncStatus")
	return output, err
}

func (r *RollupClient) Version(ctx context.Context) (string, error) {
	var output string
	err := r.rpc.CallContext(ctx, &output, "optimism_version")
	return output, err
}
