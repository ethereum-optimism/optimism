package l1

import (
	"context"
	"sync"

	"github.com/ethereum-optimism/optimism/op-node/client"

	"github.com/ethereum/go-ethereum/rpc"
)

type limitClient struct {
	c    client.RPC
	sema chan struct{}
	wg   sync.WaitGroup
}

// LimitRPC limits concurrent RPC requests (excluding subscriptions) to a given number by wrapping the client with a semaphore.
func LimitRPC(c client.RPC, concurrentRequests int) client.RPC {
	return &limitClient{
		c: c,
		// the capacity of the channel determines how many go-routines can concurrently execute requests with the wrapped client.
		sema: make(chan struct{}, concurrentRequests),
	}
}

func (lc *limitClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	lc.wg.Add(1)
	defer lc.wg.Done()
	lc.sema <- struct{}{}
	defer func() { <-lc.sema }()
	return lc.c.BatchCallContext(ctx, b)
}

func (lc *limitClient) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	lc.wg.Add(1)
	defer lc.wg.Done()
	lc.sema <- struct{}{}
	defer func() { <-lc.sema }()
	return lc.c.CallContext(ctx, result, method, args...)
}

func (lc *limitClient) EthSubscribe(ctx context.Context, channel interface{}, args ...interface{}) (*rpc.ClientSubscription, error) {
	// subscription doesn't count towards request limit
	return lc.c.EthSubscribe(ctx, channel, args...)
}

func (lc *limitClient) Close() {
	lc.wg.Wait()
	close(lc.sema)
	lc.c.Close()
}
