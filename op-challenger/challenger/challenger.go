package challenger

import (
	"context"
	_ "net/http/pprof"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

type OutputAPI interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
}

// Challenger contests invalid L2OutputOracle outputs
type Challenger struct {
	txMgr txmgr.TxManager
	wg    sync.WaitGroup
	done  chan struct{}

	log  log.Logger
	metr metrics.Metricer

	ctx    context.Context
	cancel context.CancelFunc

	l1Client *ethclient.Client

	rollupClient OutputAPI

	// l2 Output Oracle contract
	l2ooContract     *bindings.L2OutputOracleCaller
	l2ooContractAddr common.Address
	l2ooABI          *abi.ABI
	l2ooLogs         chan types.Log

	// dispute game factory contract
	dgfContract     *bindings.DisputeGameFactoryCaller
	dgfContractAddr common.Address
	dgfABI          *abi.ABI
	dgfLogs         chan types.Log

	networkTimeout time.Duration
}

// From returns the address of the account used to send transactions.
func (c *Challenger) From() common.Address {
	return c.txMgr.From()
}

// Client returns the client for the settlement layer.
func (c *Challenger) Client() *ethclient.Client {
	return c.l1Client
}

func (c *Challenger) NewOracleSubscription() (*Subscription, error) {
	query, err := BuildOutputLogFilter(c.l2ooABI)
	if err != nil {
		return nil, err
	}
	return NewSubscription(query, c.Client(), c.log), nil
}

// NewFactorySubscription creates a new [Subscription] listening to the DisputeGameFactory contract.
func (c *Challenger) NewFactorySubscription() (*Subscription, error) {
	query, err := BuildDisputeGameLogFilter(c.dgfABI)
	if err != nil {
		return nil, err
	}
	return NewSubscription(query, c.Client(), c.log), nil
}

// NewChallenger creates a new Challenger
func NewChallenger(cfg config.Config, l log.Logger, m metrics.Metricer) (*Challenger, error) {
	ctx, cancel := context.WithCancel(context.Background())

	txManager, err := txmgr.NewSimpleTxManager("challenger", l, m, *cfg.TxMgrConfig)
	if err != nil {
		cancel()
		return nil, err
	}

	// Connect to L1 and L2 providers. Perform these last since they are the most expensive.
	l1Client, err := client.DialEthClientWithTimeout(ctx, cfg.L1EthRpc, client.DefaultDialTimeout)
	if err != nil {
		cancel()
		return nil, err
	}

	rollupClient, err := client.DialRollupClientWithTimeout(ctx, cfg.RollupRpc, client.DefaultDialTimeout)
	if err != nil {
		cancel()
		return nil, err
	}

	l2ooContract, err := bindings.NewL2OutputOracleCaller(cfg.L2OOAddress, l1Client)
	if err != nil {
		cancel()
		return nil, err
	}

	dgfContract, err := bindings.NewDisputeGameFactoryCaller(cfg.DGFAddress, l1Client)
	if err != nil {
		cancel()
		return nil, err
	}

	cCtx, cCancel := context.WithTimeout(ctx, cfg.NetworkTimeout)
	defer cCancel()
	version, err := l2ooContract.Version(&bind.CallOpts{Context: cCtx})
	if err != nil {
		cancel()
		return nil, err
	}
	l.Info("Connected to L2OutputOracle", "address", cfg.L2OOAddress, "version", version)

	parsedL2oo, err := bindings.L2OutputOracleMetaData.GetAbi()
	if err != nil {
		cancel()
		return nil, err
	}

	parsedDgf, err := bindings.DisputeGameFactoryMetaData.GetAbi()
	if err != nil {
		cancel()
		return nil, err
	}

	return &Challenger{
		txMgr: txManager,
		done:  make(chan struct{}),

		log:  l,
		metr: m,

		ctx:    ctx,
		cancel: cancel,

		rollupClient: rollupClient,

		l1Client: l1Client,

		l2ooContract:     l2ooContract,
		l2ooContractAddr: cfg.L2OOAddress,
		l2ooABI:          parsedL2oo,

		dgfContract:     dgfContract,
		dgfContractAddr: cfg.DGFAddress,
		dgfABI:          parsedDgf,

		networkTimeout: cfg.NetworkTimeout,
	}, nil
}

// Start runs the core challenger components.
// Method calls are non-blocking and spawn goroutines.
func (c *Challenger) Start() error {
	c.log.Info("Challenger starting...")
	c.indexer()
	c.log.Info("Indexer spawned.")
	c.dispatch()
	c.log.Info("Dispatching initiated.")
	return nil
}

// Stop closes the challenger and waits for spawned goroutines to exit.
func (c *Challenger) Stop() {
	c.cancel()
	close(c.done)
	c.wg.Wait()
}
