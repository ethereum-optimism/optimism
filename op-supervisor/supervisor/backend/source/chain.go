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

// TODO(optimism#10999) Make these configurable and a sensible default
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
	logger.Info("Monitoring chain", "rpc", rpc)
	headMonitor := NewHeadMonitor(logger, epochPollInterval, cl, &loggingCallback{logger})
	return &ChainMonitor{
		headMonitor: headMonitor,
	}, nil
}

func (c *ChainMonitor) Start() error {
	return c.headMonitor.Start()
}

func (c *ChainMonitor) Stop() error {
	return c.headMonitor.Stop()
}

// loggingCallback is a temporary implementation of the head monitor callback that just logs the events.
// TODO(optimism#10999): Replace this with something that actually detects reorgs, fetches logs, and does consolidation
type loggingCallback struct {
	log log.Logger
}

func (n *loggingCallback) OnNewUnsafeHead(_ context.Context, block eth.L1BlockRef) {
	n.log.Info("New unsafe head", "block", block)
}

func (n *loggingCallback) OnNewSafeHead(_ context.Context, block eth.L1BlockRef) {
	n.log.Info("New safe head", "block", block)
}

func (n *loggingCallback) OnNewFinalizedHead(_ context.Context, block eth.L1BlockRef) {
	n.log.Info("New finalized head", "block", block)
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
