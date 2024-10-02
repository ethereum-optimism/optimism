package source

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TODO(optimism#11032) Make these configurable and a sensible default
const epochPollInterval = 3 * time.Second
const pollInterval = 2 * time.Second
const trustRpc = false
const rpcKind = sources.RPCKindStandard

type Metrics interface {
	caching.Metrics
}

type Storage interface {
	ChainsDBClientForLogProcessor
	DatabaseRewinder
	LatestBlockNum(chainID types.ChainID) (num uint64, ok bool)
}

// ChainMonitor monitors a source L2 chain, retrieving the data required to populate the database and perform
// interop consolidation. It detects and notifies when reorgs occur.
type ChainMonitor struct {
	log            log.Logger
	headMonitor    *HeadMonitor
	chainProcessor *ChainProcessor
}

func NewChainMonitor(ctx context.Context, logger log.Logger, m Metrics, chainID types.ChainID, rpc string, client client.RPC, store Storage) (*ChainMonitor, error) {
	logger = logger.New("chainID", chainID)
	cl, err := newClient(ctx, logger, m, rpc, client, pollInterval, trustRpc, rpcKind)
	if err != nil {
		return nil, err
	}

	// Create the log processor and fetcher
	processLogs := newLogProcessor(chainID, store)
	unsafeBlockProcessor := NewChainProcessor(logger, cl, chainID, processLogs, store)

	unsafeProcessors := []HeadProcessor{unsafeBlockProcessor}

	callback := newHeadUpdateProcessor(logger, unsafeProcessors, nil, nil)
	headMonitor := NewHeadMonitor(logger, epochPollInterval, cl, callback)

	return &ChainMonitor{
		log:            logger,
		headMonitor:    headMonitor,
		chainProcessor: unsafeBlockProcessor,
	}, nil
}

func (c *ChainMonitor) Start() error {
	c.log.Info("Started monitoring chain")
	return c.headMonitor.Start()
}

func (c *ChainMonitor) Stop() error {
	c.chainProcessor.Close()
	return c.headMonitor.Stop()
}

func newClient(ctx context.Context, logger log.Logger, m caching.Metrics, rpc string, rpcClient client.RPC, pollRate time.Duration, trustRPC bool, kind sources.RPCProviderKind) (*sources.L1Client, error) {
	c, err := client.NewRPCWithClient(ctx, logger, rpc, rpcClient, pollRate)
	if err != nil {
		return nil, fmt.Errorf("failed to create new RPC client: %w", err)
	}

	l1Client, err := sources.NewL1Client(c, logger, m, sources.L1ClientSimpleConfig(trustRPC, kind, 100))
	if err != nil {
		return nil, fmt.Errorf("failed to connect client: %w", err)
	}
	return l1Client, nil
}
