package l1

import (
	"context"

	"github.com/ethereum/go-ethereum/rpc"
)

type limitClient struct {
	c    RPCClient
	sema chan struct{}
}

// LimitRPC limits concurrent RPC requests (excluding subscriptions) to a given number by wrapping the client with a semaphore.
func LimitRPC(c RPCClient, concurrentRequests int) RPCClient {
	return &limitClient{
		c: c,
		// the capacity of the channel determines how many go-routines can concurrently execute requests with the wrapped client.
		sema: make(chan struct{}, concurrentRequests),
	}
}

func (lc *limitClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	lc.sema <- struct{}{}
	defer func() { <-lc.sema }()
	return lc.c.BatchCallContext(ctx, b)
}

func (lc *limitClient) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	lc.sema <- struct{}{}
	defer func() { <-lc.sema }()
	return lc.c.CallContext(ctx, result, method, args...)
}

func (lc *limitClient) EthSubscribe(ctx context.Context, channel interface{}, args ...interface{}) (*rpc.ClientSubscription, error) {
	// subscription doesn't count towards request limit
	return lc.c.EthSubscribe(ctx, channel, args...)
}

func (lc *limitClient) Close() {
	lc.c.Close()
}
