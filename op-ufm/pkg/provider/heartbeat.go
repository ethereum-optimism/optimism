package provider

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

// Heartbeat polls for expected in-flight transactions
func (p *Provider) Heartbeat(ctx context.Context) {
	log.Debug("heartbeat", "provider", p.name)

	if len(p.txPool.Transactions) == 0 {
		log.Debug("no in-flight txs", "provider", p.name)
		return
	}

	ethClient, err := p.dial(ctx)
	if err != nil {
		log.Error("cant dial to provider", "provider", p.name, "url", p.config.URL, "err", err)
	}

	log.Debug("checking in-flight tx", "count", len(p.txPool.Transactions), "provider", p.name)
	for hash, st := range p.txPool.Transactions {
		log.Debug(hash, "st", st)

		_, isPending, err := ethClient.TransactionByHash(ctx, st.Hash)
		if err != nil && !errors.Is(err, ethereum.NotFound) {
			log.Error("cant check transaction", "provider", p.name, "url", p.config.URL, "err", err)
			continue
		}

		log.Debug("got transaction", "provider", p.name, "hash", hash, "isPending", isPending)
		st.M.Lock()
		if st.FirstSeen.IsZero() {
			st.FirstSeen = time.Now()
		}
		if _, exist := st.SeenBy[p.name]; !exist {
			st.SeenBy[p.name] = time.Now()
		}
		st.M.Unlock()

		p.txPool.M.Lock()
		// every provider has seen this transaction
		if len(st.SeenBy) == p.txPool.Expected {
			log.Debug("transaction seen by all", "hash", hash)
			delete(p.txPool.Transactions, hash)
		}
		p.txPool.M.Unlock()
	}
}
