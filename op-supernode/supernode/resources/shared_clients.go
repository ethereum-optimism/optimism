package resources

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/rpc"
)

// NonCloseableL1Client wraps an L1Client to prevent it from being closed.
// This is used when sharing a single L1Client across multiple virtual nodes,
// where we don't want any individual node to close the shared resource.
type NonCloseableL1Client struct {
	*sources.L1Client
}

// NewNonCloseableL1Client wraps an L1Client to prevent closure.
func NewNonCloseableL1Client(client *sources.L1Client) *NonCloseableL1Client {
	return &NonCloseableL1Client{L1Client: client}
}

// Close is a no-op for the non-closeable wrapper.
// The underlying client should only be closed when the supernode itself shuts down.
func (c *NonCloseableL1Client) Close() {
}

// NonCloseableL1BeaconClient wraps an L1BeaconClient to prevent it from being closed.
// This is used when sharing a single beacon client across multiple virtual nodes.
type NonCloseableL1BeaconClient struct {
	*sources.L1BeaconClient
}

// NewNonCloseableL1BeaconClient wraps an L1BeaconClient to prevent closure.
// Returns nil if the input client is nil (beacon client is optional).
func NewNonCloseableL1BeaconClient(client *sources.L1BeaconClient) *NonCloseableL1BeaconClient {
	if client == nil {
		return nil
	}
	return &NonCloseableL1BeaconClient{L1BeaconClient: client}
}

func (c *NonCloseableL1BeaconClient) Close() {
}

// NonCloseableRPC wraps an RPC client to prevent it from being closed.
// This is used when sharing a single RPC connection across multiple clients.
type NonCloseableRPC struct {
	client.RPC
}

// NewNonCloseableRPC wraps an RPC client to prevent closure.
func NewNonCloseableRPC(rpcClient client.RPC) *NonCloseableRPC {
	return &NonCloseableRPC{RPC: rpcClient}
}

// Close is a no-op for the non-closeable wrapper.
func (c *NonCloseableRPC) Close() {
}

// CallContext delegates to the underlying RPC client
func (c *NonCloseableRPC) CallContext(ctx context.Context, result any, method string, args ...any) error {
	return c.RPC.CallContext(ctx, result, method, args...)
}

// BatchCallContext delegates to the underlying RPC client
func (c *NonCloseableRPC) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	return c.RPC.BatchCallContext(ctx, b)
}

// Subscribe delegates to the underlying RPC client
func (c *NonCloseableRPC) Subscribe(ctx context.Context, namespace string, channel any, args ...any) (ethereum.Subscription, error) {
	return c.RPC.Subscribe(ctx, namespace, channel, args...)
}
