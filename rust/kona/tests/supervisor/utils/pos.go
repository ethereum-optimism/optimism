package utils

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type TestPOS struct {
	t devtest.CommonT

	ethClient    *ethclient.Client
	blockBuilder *TestBlockBuilder

	// background management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewTestPOS(t devtest.CommonT, rpcURL string, blockBuilder *TestBlockBuilder) *TestPOS {
	ethClient, err := ethclient.Dial(rpcURL)
	if err != nil {
		t.Errorf("failed to connect to RPC: %v", err)
		return nil
	}

	return &TestPOS{t: t, ethClient: ethClient, blockBuilder: blockBuilder}
}

// Starts a background process to build blocks
func (p *TestPOS) Start() error {
	p.t.Log("Starting sequential block builder")
	// already started
	if p.ctx != nil {
		return nil
	}

	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.wg.Add(1)

	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(time.Second * 5)
		defer ticker.Stop()

		for {
			select {
			case <-p.ctx.Done():
				return
			case <-ticker.C:
				_, err := p.ethClient.BlockByNumber(p.ctx, big.NewInt(rpc.LatestBlockNumber.Int64()))
				if err != nil {
					p.t.Errorf("failed to fetch latest block: %v", err)
				}

				// Build a new block
				p.blockBuilder.BuildBlock(p.ctx, nil)
			}
		}
	}()

	return nil
}

// Stops the background process
func (p *TestPOS) Stop() {
	// cancel the context to signal the goroutine to exit
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	// wait for goroutine to finish
	p.wg.Wait()
	// clear the context to mark stopped
	p.ctx = nil
}
