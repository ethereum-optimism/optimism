package batcher

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// privateInteropEndpoints lets stock batcher pruning skip blocks already sealed by public
// sequencing-window fallback, even though those empty blocks carry no private range claims.
// The projection endpoint must be derivation-only, with no sequencing or unsafe P2P input.
// Its latest head includes fallback blocks before cross-safe catches up; using the EL safe
// label would keep loading already-expired blocks. This cursor only controls publishing,
// and does not mark private execution safe in the CL.
type privateInteropEndpoints struct {
	dial.L2EndpointProvider
	projection *rpcPublicProjectionFollower
	rollup     *rollup.Config
}

func (p *privateInteropEndpoints) RollupClient(ctx context.Context) (dial.RollupClientInterface, error) {
	cl, err := p.L2EndpointProvider.RollupClient(ctx)
	if err != nil {
		return nil, err
	}
	return &privateInteropRollupClient{RollupClientInterface: cl, endpoints: p}, nil
}

func (p *privateInteropEndpoints) Close() {
	p.L2EndpointProvider.Close()
	p.projection.rpc.Close()
}

type privateInteropRollupClient struct {
	dial.RollupClientInterface
	endpoints *privateInteropEndpoints
}

func (c *privateInteropRollupClient) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	status, err := c.RollupClientInterface.SyncStatus(ctx)
	if err != nil || status.HeadL1 == (eth.L1BlockRef{}) {
		return status, err
	}
	public, err := c.endpoints.projection.LatestBlock(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading the public projection's derived cursor: %w", err)
	}
	number := min(public.Number, status.UnsafeL2.Number)
	if number <= status.LocalSafeL2.Number {
		return status, nil
	}
	source, err := c.endpoints.PayloadSource(ctx)
	if err != nil {
		return nil, err
	}
	payload, err := source.PayloadByNumber(ctx, number)
	if err != nil {
		return nil, fmt.Errorf("reading private block at the publication cursor: %w", err)
	}
	ref, err := derive.PayloadToBlockRef(c.endpoints.rollup, payload.ExecutionPayload)
	if err != nil {
		return nil, err
	}
	if ref.Number != number {
		return nil, fmt.Errorf("private publication cursor requested block %d, got %d", number, ref.Number)
	}
	// Pruning compares private payload hashes, so never substitute the public block's hash.
	out := *status
	out.LocalSafeL2 = ref
	return &out, nil
}
