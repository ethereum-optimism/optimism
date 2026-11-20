package sources

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type RollupClient struct {
	rpc client.RPC
}

var _ apis.RollupClient = (*RollupClient)(nil)

func NewRollupClient(rpc client.RPC) *RollupClient {
	return &RollupClient{rpc}
}

func (r *RollupClient) OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	return callNonNull[eth.OutputResponse](ctx, r.rpc, "optimism_outputAtBlock", hexutil.Uint64(blockNum))
}

func (r *RollupClient) SafeHeadAtL1Block(ctx context.Context, blockNum uint64) (*eth.SafeHeadResponse, error) {
	output, err := callNonNull[eth.SafeHeadResponse](ctx, r.rpc, "optimism_safeHeadAtL1Block", hexutil.Uint64(blockNum))
	if err != nil && strings.Contains(err.Error(), "not found") {
		return nil, safedb.ErrNotFound
	}
	return output, err
}

func (r *RollupClient) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	return callNonNull[eth.SyncStatus](ctx, r.rpc, "optimism_syncStatus")
}

func (r *RollupClient) RollupConfig(ctx context.Context) (*rollup.Config, error) {
	return callNonNull[rollup.Config](ctx, r.rpc, "optimism_rollupConfig")
}

func (r *RollupClient) DependencySet(ctx context.Context) (depset.DependencySet, error) {
	output, err := callNonNull[depset.StaticConfigDependencySet](ctx, r.rpc, "optimism_dependencySet")
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (r *RollupClient) Version(ctx context.Context) (string, error) {
	var output string
	err := r.rpc.CallContext(ctx, &output, "optimism_version")
	return output, err
}

func (r *RollupClient) StartSequencer(ctx context.Context, unsafeHead common.Hash) error {
	return r.rpc.CallContext(ctx, nil, "admin_startSequencer", unsafeHead)
}

func (r *RollupClient) StopSequencer(ctx context.Context) (common.Hash, error) {
	var result common.Hash
	err := r.rpc.CallContext(ctx, &result, "admin_stopSequencer")
	return result, err
}

func (r *RollupClient) SequencerActive(ctx context.Context) (bool, error) {
	var result bool
	err := r.rpc.CallContext(ctx, &result, "admin_sequencerActive")
	return result, err
}

func (r *RollupClient) PostUnsafePayload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) error {
	return r.rpc.CallContext(ctx, nil, "admin_postUnsafePayload", payload)
}

func (r *RollupClient) OverrideLeader(ctx context.Context) error {
	return r.rpc.CallContext(ctx, nil, "admin_overrideLeader")
}

func (r *RollupClient) ConductorEnabled(ctx context.Context) (bool, error) {
	var result bool
	err := r.rpc.CallContext(ctx, &result, "admin_conductorEnabled")
	return result, err
}

func (r *RollupClient) SetLogLevel(ctx context.Context, lvl slog.Level) error {
	return r.rpc.CallContext(ctx, nil, "admin_setLogLevel", lvl.String())
}

func (r *RollupClient) SetRecoverMode(ctx context.Context, mode bool) error {
	return r.rpc.CallContext(ctx, nil, "admin_setRecoverMode", mode)
}

func (r *RollupClient) Close() {
	r.rpc.Close()
}

// callNonNull calls method and decodes its result, returning an error if the result is JSON
// null. The methods using it never answer null on success: absence is reported as an error.
func callNonNull[T any](ctx context.Context, rpc client.RPC, method string, args ...any) (*T, error) {
	var out *T
	if err := rpc.CallContext(ctx, &out, method, args...); err != nil {
		return nil, err
	}
	if out == nil {
		return nil, fmt.Errorf("%s returned null", method)
	}
	return out, nil
}
