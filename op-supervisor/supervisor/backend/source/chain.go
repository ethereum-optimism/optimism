package source

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// TODO(optimism#11032) Make these configurable and a sensible default
const epochPollInterval = 30 * time.Second
const pollInterval = 2 * time.Second
const trustRpc = false
const rpcKind = sources.RPCKindStandard

type Metrics interface {
	caching.Metrics
}

// ChainMonitor monitors a source L2 chain, retrieving the data required to populate the database and perform
// interop consolidation. It detects and notifies when reorgs occur.
type ChainMonitor struct {
	log         log.Logger
	headMonitor *HeadMonitor
}

func NewChainMonitor(ctx context.Context, logger log.Logger, m Metrics, chainID *big.Int, rpc string, client client.RPC) (*ChainMonitor, error) {
	logger = logger.New("chainID", chainID)
	cl, err := newClient(ctx, logger, m, rpc, client, pollInterval, trustRpc, rpcKind)
	if err != nil {
		return nil, err
	}

	// TODO(optimism#11023): Load the starting block from log db
	startingHead := eth.L1BlockRef{}

	fetchReceipts := newLogFetcher(cl, &loggingReceiptProcessor{logger})
	unsafeBlockProcessor := NewChainProcessor(logger, cl, startingHead, fetchReceipts)

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

type loggingReceiptProcessor struct {
	log log.Logger
}

func (n *loggingReceiptProcessor) ProcessLogs(_ context.Context, block eth.L1BlockRef, rcpts types.Receipts) error {
	n.log.Info("Process unsafe block", "block", block, "rcpts", len(rcpts))
	return nil
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
