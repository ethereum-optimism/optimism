package dial

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// PayloadSource fetches L2 execution payloads from an execution node. It is
// the L2 block currency of the op-batcher: opaque transaction bytes that
// round-trip OP Stack synthetic transactions without go-ethereum's typed
// decoding.
type PayloadSource interface {
	PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error)
	Close()
}

// L2EndpointProvider is an interface for providing a RollupClient and an L2 payload source.
// It manages the lifecycle of both for callers.
// It does this by extending the RollupProvider interface to add the ability to get a PayloadSource.
type L2EndpointProvider interface {
	RollupProvider
	// PayloadSource(ctx) returns a payload source pointing to the L2 execution node.
	// Note: ctx should be a lifecycle context without an attached timeout as client selection may involve
	// multiple network operations, specifically in the case of failover.
	PayloadSource(ctx context.Context) (PayloadSource, error)
}

// newL2PayloadSource wraps a dialed L2 execution node RPC into a payload
// source. The RPC is trusted: the batcher reads its own sequencer/EL, and
// hash/tx-root verification here would add per-block hashing the previous
// ethclient-based fetching never did.
func newL2PayloadSource(log log.Logger, rpcCl *rpc.Client) (PayloadSource, error) {
	cfg := sources.DefaultEthClientConfig(10)
	cfg.TrustRPC = true
	return sources.NewEthClient(client.NewBaseRPCClient(rpcCl), log, nil, cfg)
}

// StaticL2EndpointProvider is a L2EndpointProvider that always returns the same static RollupClient and payload source.
// It is meant for scenarios where a single, unchanging (L2 rollup node, L2 execution node) pair is used.
type StaticL2EndpointProvider struct {
	StaticL2RollupProvider
	payloadSource PayloadSource
}

func NewStaticL2EndpointProvider(ctx context.Context, log log.Logger, ethClientUrl string, rollupClientUrl string) (*StaticL2EndpointProvider, error) {
	// A bounded retrying dial, matching the DialEthClientWithTimeout behavior
	// this provider used before payload sources: transient L2 unavailability
	// must not abort startup. The active provider deliberately dials one-shot
	// instead, so its failover moves on to the next endpoint quickly.
	dialCtx, cancel := context.WithTimeout(ctx, DefaultDialTimeout)
	defer cancel()
	rpcCl, err := DialRPCClientWithTimeout(dialCtx, log, ethClientUrl)
	if err != nil {
		return nil, err
	}
	payloadSource, err := newL2PayloadSource(log, rpcCl)
	if err != nil {
		return nil, err
	}
	rollupProvider, err := NewStaticL2RollupProvider(ctx, log, rollupClientUrl)
	if err != nil {
		return nil, err
	}
	return &StaticL2EndpointProvider{
		StaticL2RollupProvider: *rollupProvider,
		payloadSource:          payloadSource,
	}, nil
}

func (p *StaticL2EndpointProvider) PayloadSource(context.Context) (PayloadSource, error) {
	return p.payloadSource, nil
}

func (p *StaticL2EndpointProvider) Close() {
	if p.payloadSource != nil {
		p.payloadSource.Close()
	}
	p.StaticL2RollupProvider.Close()
}
