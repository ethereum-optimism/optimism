package interop

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/clock"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type PostieConfig struct {
	Postie *ecdsa.PrivateKey

	DestinationChain *ethclient.Client
	ConnectedChains  []node.EthClient

	UpdateInterval time.Duration
}

type Postie struct {
	log log.Logger

	chains               map[uint64]node.EthClient
	outboxStorageRoots   map[uint64]common.Hash
	outboxUpdateInterval time.Duration

	crossL2Inbox *bindings.CrossL2Inbox
	tOpts        *bind.TransactOpts

	worker  *clock.LoopFn
	stopped atomic.Bool
}

func NewPostie(log log.Logger, cfg PostieConfig) (*Postie, error) {
	connectedChains := map[uint64]node.EthClient{}
	for _, clnt := range cfg.ConnectedChains {
		chainIdBig, err := clnt.ChainID()
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve chain id: %w", err)
		}

		chainId := chainIdBig.Uint64()
		connectedChains[chainId] = clnt
	}

	destChainId, err := cfg.DestinationChain.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("unable to fetch chain id of destination chain: %w", err)
	}

	tOpts, err := bind.NewKeyedTransactorWithChainID(cfg.Postie, destChainId)
	if err != nil {
		return nil, fmt.Errorf("unable to create transactor for the postiee account: %w", err)
	}

	crossL2Inbox, err := bindings.NewCrossL2Inbox(predeploys.CrossL2InboxAddr, cfg.DestinationChain)
	if err != nil {
		return nil, fmt.Errorf("unable to construct inbox contract: %w", err)
	}

	//  NOTE: Since we dont pre-populate the previous state, this will cause an
	// 	on-chain tx on startup to deliver an update for all chains which is a
	//  is a no-op if nothing has happened since the restart of this daemon. This
	//  can be fixed by correctly bootstrapping `p.outboxStorageRoots` with the
	//  correct previous state. Choosing not to due this due to the current design
	//  of CrossL2Inbox. Not worth the effort for a prototype.
	outboxStorageRoots := map[uint64]common.Hash{}
	return &Postie{
		log:                  log,
		chains:               connectedChains,
		outboxUpdateInterval: cfg.UpdateInterval,
		outboxStorageRoots:   outboxStorageRoots,
		crossL2Inbox:         crossL2Inbox,
		tOpts:                tOpts,
	}, nil
}

func (p *Postie) Start(ctx context.Context) error {
	p.log.Info("starting postie...")

	// Run once on startup, then start the loop
	p.tick(context.Background())
	p.worker = clock.NewLoopFn(clock.SystemClock, p.tick, func() error {
		p.log.Info("worker stopped")
		return nil
	}, p.outboxUpdateInterval)

	return nil
}

func (p *Postie) Stop(ctx context.Context) error {
	p.log.Info("stopping postie...")
	if p.worker == nil {
		return nil
	}

	err := p.worker.Close()
	p.stopped.Store(true)
	return err
}

func (p *Postie) Stopped() bool {
	return p.stopped.Load()
}

func (p *Postie) tick(_ context.Context) {
	p.log.Info("checking outboxes")

	// NOTE: There are some potential lifecycle isssues casued by the delay between
	// tx submission and inclusion that would cause repeated mail delivery. As long as
	// the tick interval >> block time we're fine for the prototype
	mail := []bindings.InboxEntry{}
	for chainId, clnt := range p.chains {
		oldStorageRoot := p.outboxStorageRoots[chainId]

		outboxStorageRoot, err := clnt.StorageHash(predeploys.CrossL2OutboxAddr, nil)
		if err != nil {
			p.log.Error("unable to fetch outbox storage root", "chain_id", chainId, "err", err)
		}

		if outboxStorageRoot == oldStorageRoot {
			p.log.Info("no change in state", "chain_id", chainId)
		} else {
			p.log.Info("detected new outbox storage root", "chain_id", chainId, "root", outboxStorageRoot.String(), "old_root", oldStorageRoot.String())
			mail = append(mail, bindings.InboxEntry{Chain: common.BigToHash(big.NewInt(int64(chainId))), Output: outboxStorageRoot})
		}
	}

	if len(mail) > 0 {
		p.log.Info("delivering mail to inbox...")
		tx, err := p.crossL2Inbox.DeliverMail(p.tOpts, mail)
		if err != nil {
			p.log.Error("unable to deliver mail", "err", err)
		}

		p.log.Info("mail delivered", "tx_hash", tx.Hash().String())

		// NOTE: Technically we should only be updating on succesful inclusion. We would
		// ideally read from some on-chain events of updated roots. Since for the prototype
		// inclusion should be guaranteed, this is fine
		for _, mail := range mail {
			chainId := new(big.Int).SetBytes(mail.Chain[:]).Uint64()
			p.outboxStorageRoots[chainId] = mail.Output
		}
	}
}
