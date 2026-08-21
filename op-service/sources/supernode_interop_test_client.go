package sources

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SupernodeInteropTestClient drives a supernode's interop verifier over RPC.
//
// It is the out-of-process implementation of apis.SupernodeInteropTestAPI: the
// in-process Go op-supernode satisfies that interface by calling itself, and a
// supernode in another process — lokahi — satisfies it by serving these four
// methods. Both sides therefore reach the devstack DSL through one interface,
// and the DSL has no idea which it is talking to, which is the point: the
// acceptance suites measure the two implementations against the same
// assertions.
//
// Test-only, like the interface it implements. Production code paths must not
// use it.
//
// # Why the methods are named the way they are
//
// These are lokahi's own methods rather than a wire op-supernode also serves:
// the Go supernode has no RPC for them at all, since nothing outside a test
// ever asks a supernode to stop verifying. They live in lokahi's process-wide
// `lokahi` namespace, alongside lokahi_chains, because pausing a verifier is a
// statement about the process rather than about one of its chains.
//
// The values crossing the wire are the same eth types the interface is defined
// in, so there is no second Go representation of them to drift.
type SupernodeInteropTestClient struct {
	rpc client.RPC
}

var _ apis.SupernodeInteropTestAPI = (*SupernodeInteropTestClient)(nil)

// NewSupernodeInteropTestClient wraps an RPC client addressing a supernode's
// process-wide admin endpoint.
func NewSupernodeInteropTestClient(rpc client.RPC) *SupernodeInteropTestClient {
	return &SupernodeInteropTestClient{rpc: rpc}
}

// PauseInterop asks the verifier to stop when it reaches ts. Zero clears the
// pause, which is the interface's documented behaviour and is why the timestamp
// crosses the wire as a bare number rather than as a nullable one.
func (c *SupernodeInteropTestClient) PauseInterop(ctx context.Context, ts uint64) error {
	return c.rpc.CallContext(ctx, nil, "lokahi_pauseInterop", ts)
}

func (c *SupernodeInteropTestClient) ResumeInterop(ctx context.Context) error {
	return c.rpc.CallContext(ctx, nil, "lokahi_resumeInterop")
}

func (c *SupernodeInteropTestClient) InteropStatus(ctx context.Context) (result eth.SupernodeInteropStatus, err error) {
	err = c.rpc.CallContext(ctx, &result, "lokahi_interopStatus")
	return
}

// InteropSealedBlocks reports how far one chain's interop logs DB extends.
//
// The chain id is sent as a uint64 rather than as eth.ChainID's decimal-string
// marshalling, matching lokahi_chains on the same endpoint: every chain a
// supernode hosts came out of one configuration file that names them as plain
// numbers, so a u256-shaped identifier would buy nothing here.
func (c *SupernodeInteropTestClient) InteropSealedBlocks(ctx context.Context, chainID eth.ChainID) (result eth.SupernodeSealedBlocks, err error) {
	err = c.rpc.CallContext(ctx, &result, "lokahi_interopSealedBlocks", eth.EvilChainIDToUInt64(chainID))
	return
}

func (c *SupernodeInteropTestClient) Close() {
	c.rpc.Close()
}
