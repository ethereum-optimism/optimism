package source

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// TODO(optimism#11032) Make these configurable and a sensible default
const epochPollInterval = 30 * time.Second
const pollInterval = 2 * time.Second
const trustRpc = false
const rpcKind = sources.RPCKindStandard

type Metrics interface {
	CacheAdd(chainID *big.Int, label string, cacheSize int, evicted bool)
	CacheGet(chainID *big.Int, label string, hit bool)
}

// ChainMonitor monitors a source L2 chain, retrieving the data required to populate the database and perform
// interop consolidation. It detects and notifies when reorgs occur.
type ChainMonitor struct {
	log         log.Logger
	headMonitor *HeadMonitor
}

func NewChainMonitor(ctx context.Context, logger log.Logger, genericMetrics Metrics, rpc string) (*ChainMonitor, error) {
	// First dial a simple client and get the chain ID so we have a simple identifier for the chain.
	ethClient, err := dial.DialEthClientWithTimeout(ctx, 10*time.Second, logger, rpc)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rpc %v: %w", rpc, err)
	}
	chainID, err := ethClient.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load chain id for rpc %v: %w", rpc, err)
	}
	logger = logger.New("chainID", chainID)
	m := newChainMetrics(chainID, genericMetrics)
	cl, err := newClient(ctx, logger, m, rpc, ethClient.Client(), pollInterval, trustRpc, rpcKind)
	if err != nil {
		return nil, err
	}

	// TODO(optimism#11023): Load the starting block from log db
	startingHead := eth.L1BlockRef{}
	unsafeBlockProcessor := NewChainProcessor(logger, cl, startingHead, &loggingBlockProcessor{logger})

	unsafeProcessors := []HeadProcessor{unsafeBlockProcessor}
	callback := newHeadUpdateProcessor(logger, unsafeProcessors, nil, nil)
	headMonitor := NewHeadMonitor(logger, epochPollInterval, cl, callback)

	return &ChainMonitor{
		log:         logger,
		headMonitor: headMonitor,
	}, nil
}

func (c *ChainMonitor) Start() error {
	c.log.Info("Started monitoring chain")
	return c.headMonitor.Start()
}

func (c *ChainMonitor) Stop() error {
	return c.headMonitor.Stop()
}

type loggingBlockProcessor struct {
	log log.Logger
}

func (n *loggingBlockProcessor) ProcessBlock(_ context.Context, block eth.L1BlockRef) error {
	n.log.Info("Process unsafe block", "block", block)
	return nil
}

func newClient(ctx context.Context, logger log.Logger, m caching.Metrics, rpc string, rpcClient *rpc.Client, pollRate time.Duration, trustRPC bool, kind sources.RPCProviderKind) (*sources.L1Client, error) {
	c, err := client.NewRPCWithClient(ctx, logger, rpc, client.NewBaseRPCClient(rpcClient), pollRate)
	if err != nil {
		return nil, fmt.Errorf("failed to create new RPC client: %w", err)
	}

	l1Client, err := sources.NewL1Client(c, logger, m, sources.L1ClientSimpleConfig(trustRPC, kind, 100))
	if err != nil {
		return nil, fmt.Errorf("failed to connect client: %w", err)
	}
	return l1Client, nil
}
