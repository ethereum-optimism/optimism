package sources

import (
	"context"
	"log/slog"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type NativeRollupClient struct {
	rpc client.RPC
}

func NewNativeRollupClient(rpc client.RPC) *NativeRollupClient {
	return &NativeRollupClient{rpc}
}

func (r *NativeRollupClient) StatelessOutputAtBlock(ctx context.Context, blockNum uint64) (*eth.StatelessOutputResponse, error) {
	var output *eth.StatelessOutputResponse
	err := r.rpc.CallContext(ctx, &output, "optimism_statelessOutputAtBlock", hexutil.Uint64(blockNum))
	return output, err
}

func (r *NativeRollupClient) OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	var output *eth.OutputResponse
	err := r.rpc.CallContext(ctx, &output, "optimism_outputAtBlock", hexutil.Uint64(blockNum))
	return output, err
}

func (r *NativeRollupClient) SafeHeadAtL1Block(ctx context.Context, blockNum uint64) (*eth.SafeHeadResponse, error) {
	var output *eth.SafeHeadResponse
	err := r.rpc.CallContext(ctx, &output, "optimism_safeHeadAtL1Block", hexutil.Uint64(blockNum))
	return output, err
}

func (r *NativeRollupClient) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	var output *eth.SyncStatus
	err := r.rpc.CallContext(ctx, &output, "optimism_syncStatus")
	return output, err
}

func (r *NativeRollupClient) RollupConfig(ctx context.Context) (*rollup.Config, error) {
	var output *rollup.Config
	err := r.rpc.CallContext(ctx, &output, "optimism_rollupConfig")
	return output, err
}

func (r *NativeRollupClient) Version(ctx context.Context) (string, error) {
	var output string
	err := r.rpc.CallContext(ctx, &output, "optimism_version")
	return output, err
}

func (r *NativeRollupClient) StartSequencer(ctx context.Context, unsafeHead common.Hash) error {
	return r.rpc.CallContext(ctx, nil, "admin_startSequencer", unsafeHead)
}

func (r *NativeRollupClient) StopSequencer(ctx context.Context) (common.Hash, error) {
	var result common.Hash
	err := r.rpc.CallContext(ctx, &result, "admin_stopSequencer")
	return result, err
}

func (r *NativeRollupClient) SequencerActive(ctx context.Context) (bool, error) {
	var result bool
	err := r.rpc.CallContext(ctx, &result, "admin_sequencerActive")
	return result, err
}

func (r *NativeRollupClient) PostUnsafePayload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) error {
	return r.rpc.CallContext(ctx, nil, "admin_postUnsafePayload", payload)
}

func (r *NativeRollupClient) OverrideLeader(ctx context.Context) error {
	return r.rpc.CallContext(ctx, nil, "admin_overrideLeader")
}

func (r *NativeRollupClient) ConductorEnabled(ctx context.Context) (bool, error) {
	var result bool
	err := r.rpc.CallContext(ctx, &result, "admin_conductorEnabled")
	return result, err
}

func (r *NativeRollupClient) SetLogLevel(ctx context.Context, lvl slog.Level) error {
	return r.rpc.CallContext(ctx, nil, "admin_setLogLevel", lvl.String())
}

func (r *NativeRollupClient) Close() {
	r.rpc.Close()
}
