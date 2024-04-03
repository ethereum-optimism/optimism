package superchain

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type Backend interface {
	MessageSafety(context.Context, MessageIdentifier, hexutil.Bytes) (MessageSafetyLabel, error)
}

type backend struct {
	log log.Logger
	mu  sync.RWMutex

	l2FinalizedHeadSub  ethereum.Subscription
	l2FinalizedBlockRef *eth.L1BlockRef

	logProviders map[string]LogsProvider
}

func NewBackend(ctx context.Context, log log.Logger, m metrics.Factory, cfg *Config) (Backend, error) {
	log = log.New("module", "superchain")
	backend := backend{log: log, logProviders: map[string]LogsProvider{}}

	rpcOpts := []client.RPCOption{client.WithDialBackoff(10)}
	l2Clnt, err := client.NewRPC(ctx, log, cfg.L2NodeAddr, rpcOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to L2 node: %w", err)
	}

	for chainId, l2NodeAddr := range cfg.PeerL2NodeAddrs {
		clnt, err := client.NewRPC(ctx, log, l2NodeAddr, rpcOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to Peer L2 node, %d: %w", chainId, err)
		}
		backend.logProviders[fmt.Sprintf("%d", chainId)] = NewLogProvider(clnt)
	}

	// retrieve the current references before setting up the poll
	l2BlockRefsClient := &blockRefsClient{l2Clnt}
	finalizedHeadRef, err := l2BlockRefsClient.L1BlockRefByLabel(ctx, eth.Finalized)
	if err != nil {
		return nil, fmt.Errorf("failed to query finalized block ref: %w", err)
	}

	backend.l2FinalizedBlockRef = &finalizedHeadRef
	log.Info("detected finalized head", "number", finalizedHeadRef.Number, "hash", finalizedHeadRef.Hash)

	l2FinalizedHeadSignal := func(ctx context.Context, sig eth.L1BlockRef) {
		log.Info("new finalized head", "number", sig.Number, "hash", sig.Hash)
		backend.mu.Lock()
		backend.l2FinalizedBlockRef = &sig
		backend.mu.Unlock()
	}

	pollInterval, timeout := time.Second, time.Second*10
	backend.l2FinalizedHeadSub = eth.PollBlockChanges(log, l2BlockRefsClient, l2FinalizedHeadSignal, eth.Finalized, pollInterval, timeout)

	return &backend, nil
}

func (b *backend) MessageSafety(ctx context.Context, id MessageIdentifier, payload hexutil.Bytes) (MessageSafetyLabel, error) {
	b.log.Info("message safety check", "chain_id", id.ChainId, "block_number", id.BlockNumber, "log_index", id.LogIndex)

	// ChainID Invariant.
	//   TODO: Assumption here that the configured peers exactly maps to the registered dependency set.
	//   When the predeploy is ready, this needs to be tied to the dependency set registered on-chain
	logsProvider, ok := b.logProviders[id.ChainId.String()]
	if !ok {
		return MessageUnknown, fmt.Errorf("peer with chain id %d is not configured", id.ChainId)
	}

	blockNum := rpc.BlockNumber(id.BlockNumber.Int64())
	block, logs, err := logsProvider.FetchLogs(ctx, rpc.BlockNumberOrHash{BlockNumber: &blockNum})
	if err != nil {
		return MessageUnknown, fmt.Errorf("unable to fetch logs: %w", err)
	}

	// validity with the block
	if id.Timestamp != block.Time() {
		return MessageInvalid, fmt.Errorf("message id and header timestamp mismatch")
	}
	if id.LogIndex >= uint64(len(logs)) {
		return MessageInvalid, fmt.Errorf("invalid log index")
	}

	// Check message validity against the remote log
	log := logs[id.LogIndex]
	if err := CheckMessageLog(id, payload, &log); err != nil {
		return MessageInvalid, fmt.Errorf("failed log check: %w", err)
	}

	// Message Safety
	var finalizedL2Timestamp uint64
	b.mu.RLock()
	finalizedL2Timestamp = b.l2FinalizedBlockRef.Time
	b.mu.RUnlock()

	if id.Timestamp <= finalizedL2Timestamp {
		return MessageFinalized, nil
	}

	// TODO: support for the other safety labels

	// Cant determine validity
	return MessageUnknown, nil
}
