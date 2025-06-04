package loadtest

import (
	"sync"
	"sync/atomic"

	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type SyncEOA struct {
	Inner *dsl.EOA
	Nonce atomic.Int64
}

type EOAPool struct {
	eoas  []*SyncEOA
	index atomic.Uint64
}

func NewEOAPool(funder *dsl.Funder, num uint64, amount eth.ETH) *EOAPool {
	eoas := make([]*SyncEOA, num)
	var wg sync.WaitGroup
	defer wg.Wait()
	for i := range num {
		wg.Add(1)
		go func() {
			defer wg.Done()
			eoas[i] = &SyncEOA{
				Inner: funder.NewFundedEOA(amount),
			}
		}()
	}
	return &EOAPool{
		eoas: eoas,
	}
}

func (p *EOAPool) Get() *SyncEOA {
	next := (p.index.Add(1) - 1) % uint64(len(p.eoas))
	return p.eoas[next]
}
