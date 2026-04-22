package interop

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	gethparams "github.com/ethereum/go-ethereum/params"
	"golang.org/x/sync/errgroup"

	"github.com/ethereum-optimism/optimism/op-node/params"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// DefaultRPCValidatorTimeout caps how long a single block's validation may take.
// After this elapses, any outstanding remote RPC calls are cancelled and the
// block is rejected. Operators can tune via --interop.rpc-validator.timeout.
const DefaultRPCValidatorTimeout = 60 * time.Second

// L2ReceiptsSource fetches receipts for a block from the local L2 execution engine.
type L2ReceiptsSource interface {
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
}

// remoteClient is the subset of ethclient.Client used by the validator. It
// exists so tests can substitute a fake without dialing real endpoints.
type remoteClient interface {
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
	HeaderByHash(ctx context.Context, h common.Hash) (*types.Header, error)
}

// RPCExecMsgValidator verifies every executing message in a newly-inserted
// unsafe block by consulting a configured remote chain's eth_getLogs endpoint.
// It is intended for light-CL deployments where no op-supervisor is running.
type RPCExecMsgValidator struct {
	log        log.Logger
	rollupCfg  *rollup.Config
	l2         L2ReceiptsSource
	clients    map[eth.ChainID]remoteClient
	timeout    time.Duration
	retryDelay time.Duration
}

// ParseRPCOverrides parses a comma-separated list of chainID=URL pairs.
// Empty input yields an empty map (validator disabled).
func ParseRPCOverrides(s string) (map[eth.ChainID]string, error) {
	out := map[eth.ChainID]string{}
	if strings.TrimSpace(s) == "" {
		return out, nil
	}
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		eq := strings.IndexByte(pair, '=')
		if eq < 0 {
			return nil, fmt.Errorf("invalid rpc-override %q: expected chainID=URL", pair)
		}
		id, err := eth.ChainIDFromString(strings.TrimSpace(pair[:eq]))
		if err != nil {
			return nil, fmt.Errorf("invalid chainID in %q: %w", pair, err)
		}
		url := strings.TrimSpace(pair[eq+1:])
		if url == "" {
			return nil, fmt.Errorf("empty URL for chain %s", id)
		}
		if _, dup := out[id]; dup {
			return nil, fmt.Errorf("duplicate chainID %s in overrides", id)
		}
		out[id] = url
	}
	return out, nil
}

// NewRPCExecMsgValidator parses overridesRaw, dials every configured remote
// RPC up front, and returns the ready-to-use validator. Returns nil (no error)
// if overridesRaw is empty — callers treat nil as "feature disabled" and skip
// the hook entirely.
func NewRPCExecMsgValidator(
	logger log.Logger,
	rollupCfg *rollup.Config,
	l2 L2ReceiptsSource,
	overridesRaw string,
	timeout time.Duration,
) (*RPCExecMsgValidator, error) {
	overrides, err := ParseRPCOverrides(overridesRaw)
	if err != nil {
		return nil, err
	}
	if len(overrides) == 0 {
		return nil, nil
	}
	if timeout <= 0 {
		timeout = DefaultRPCValidatorTimeout
	}
	clients := make(map[eth.ChainID]remoteClient, len(overrides))
	for id, url := range overrides {
		c, err := ethclient.Dial(url)
		if err != nil {
			return nil, fmt.Errorf("dial %s (chain %s): %w", url, id, err)
		}
		clients[id] = c
	}
	return &RPCExecMsgValidator{
		log:        logger,
		rollupCfg:  rollupCfg,
		l2:         l2,
		clients:    clients,
		timeout:    timeout,
		retryDelay: 500 * time.Millisecond,
	}, nil
}

// Validate scans the block's receipts for ExecutingMessage logs and verifies
// each against the configured remote chain RPC. Returns nil if every exec
// message validates, an error otherwise. Safe to call pre-interop (returns nil).
func (v *RPCExecMsgValidator) Validate(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) error {
	execTime := uint64(envelope.ExecutionPayload.Timestamp)
	if !v.rollupCfg.IsInterop(execTime) {
		return nil
	}

	blockHash := envelope.ExecutionPayload.BlockHash
	_, receipts, err := v.l2.FetchReceipts(ctx, blockHash)
	if err != nil {
		return fmt.Errorf("fetch receipts for %s: %w", blockHash, err)
	}

	// Group executing messages by chainID so each chain's RPC is called
	// sequentially within one goroutine but different chains run in parallel.
	grouped := map[eth.ChainID][]supervisortypes.Message{}
	for _, rcpt := range receipts {
		for _, l := range rcpt.Logs {
			if l.Address != gethparams.InteropCrossL2InboxAddress {
				continue
			}
			if len(l.Topics) == 0 || l.Topics[0] != supervisortypes.ExecutingMessageEventTopic {
				continue
			}
			var m supervisortypes.Message
			if err := m.DecodeEvent(l.Topics, l.Data); err != nil {
				return fmt.Errorf("decode executing-message log: %w", err)
			}
			grouped[m.Identifier.ChainID] = append(grouped[m.Identifier.ChainID], m)
		}
	}
	if len(grouped) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, v.timeout)
	defer cancel()

	var g errgroup.Group
	for chainID, msgs := range grouped {
		chainID, msgs := chainID, msgs
		g.Go(func() error { return v.validateChain(ctx, chainID, msgs, execTime) })
	}
	return g.Wait()
}

func (v *RPCExecMsgValidator) validateChain(ctx context.Context, chainID eth.ChainID, msgs []supervisortypes.Message, execTime uint64) error {
	client, ok := v.clients[chainID]
	if !ok {
		return fmt.Errorf("no RPC configured for chain %s referenced by executing message", chainID)
	}
	for i := range msgs {
		if err := v.validateOne(ctx, client, &msgs[i], execTime); err != nil {
			return fmt.Errorf("chain %s: %w", chainID, err)
		}
	}
	return nil
}

func (v *RPCExecMsgValidator) validateOne(ctx context.Context, client remoteClient, m *supervisortypes.Message, execTime uint64) error {
	id := m.Identifier

	// Temporal ordering: initiator must not be in the executor's future.
	if id.Timestamp > execTime {
		return fmt.Errorf("identifier timestamp %d exceeds exec block timestamp %d", id.Timestamp, execTime)
	}
	// Expiry window: use go-ethereum's shared interop constant.
	if execTime-id.Timestamp > uint64(params.MessageExpiryTimeSecondsInterop) {
		return fmt.Errorf("executing message expired (exec %d - init %d > %d)",
			execTime, id.Timestamp, params.MessageExpiryTimeSecondsInterop)
	}

	remoteLog, remoteBlockTime, err := v.fetchRemoteLogWithRetry(ctx, client, id.Origin, id.BlockNumber, id.LogIndex)
	if err != nil {
		return err
	}
	if remoteLog == nil {
		return fmt.Errorf("remote log not found at origin=%s block=%d logIndex=%d",
			id.Origin, id.BlockNumber, id.LogIndex)
	}
	if remoteLog.Address != id.Origin {
		return fmt.Errorf("origin mismatch: remote log address %s != identifier origin %s", remoteLog.Address, id.Origin)
	}
	if remoteBlockTime != id.Timestamp {
		return fmt.Errorf("remote block timestamp %d != identifier timestamp %d", remoteBlockTime, id.Timestamp)
	}
	computed := crypto.Keccak256Hash(supervisortypes.LogToMessagePayload(remoteLog))
	if computed != m.PayloadHash {
		return fmt.Errorf("payload hash mismatch: computed=%s expected=%s", computed, m.PayloadHash)
	}
	return nil
}

func (v *RPCExecMsgValidator) fetchRemoteLogWithRetry(ctx context.Context, client remoteClient, origin common.Address, blockNum uint64, logIndex uint32) (*types.Log, uint64, error) {
	bn := new(big.Int).SetUint64(blockNum)
	q := ethereum.FilterQuery{FromBlock: bn, ToBlock: bn, Addresses: []common.Address{origin}}
	for {
		logs, err := client.FilterLogs(ctx, q)
		if err == nil {
			for i := range logs {
				if logs[i].Index != uint(logIndex) {
					continue
				}
				header, hErr := client.HeaderByHash(ctx, logs[i].BlockHash)
				if hErr != nil {
					err = hErr
					break
				}
				return &logs[i], header.Time, nil
			}
			if err == nil {
				// log not found — authoritative "no", not a retriable error
				return nil, 0, nil
			}
		}
		v.log.Warn("remote RPC error, retrying", "origin", origin, "block", blockNum, "err", err)
		select {
		case <-ctx.Done():
			return nil, 0, fmt.Errorf("remote RPC exceeded validation budget: %w", ctx.Err())
		case <-time.After(v.retryDelay):
		}
	}
}
