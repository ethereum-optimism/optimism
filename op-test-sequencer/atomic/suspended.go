package atomic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// PrefixAccount preserves the distinction between an absent and an empty account.
// Snapshot.Account returns nil for absence. Reads must be immutable for the build.
type PrefixAccount struct {
	Balance *hexutil.Big  `json:"balance"`
	Nonce   uint64        `json:"nonce"`
	Code    hexutil.Bytes `json:"code"`
}

// PrefixSnapshot is the exact candidate state after system calls and all preceding
// transactions, before the atomic envelope. BlockHash follows its pinned ancestry.
// The worker never writes to this snapshot.
type PrefixSnapshot interface {
	Account(context.Context, common.Address) (*PrefixAccount, error)
	Storage(context.Context, common.Address, common.Hash) (common.Hash, error)
	BlockHash(context.Context, uint64) (common.Hash, error)
}

// SuspendedChain pins the candidate environment and provides local signing.
// Prepare signs both discovery and final envelopes; keys never enter the worker.
// Only EIP-1559 envelopes on Lagoon are supported by this demo bridge.
type SuspendedChain struct {
	Chain          uint64         `json:"chain"`
	Router         common.Address `json:"router"`
	Sender         common.Address `json:"sender"`
	ApplicationGas uint64         `json:"application_gas"`
	PrefixLogs     uint32         `json:"prefix_logs"`
	// PrefixGas is cumulative gas before SDM refunds, matching block admission.
	PrefixGas  uint64         `json:"prefix_gas"`
	Spec       string         `json:"spec"`
	Number     uint64         `json:"number"`
	Timestamp  uint64         `json:"timestamp"`
	GasLimit   uint64         `json:"gas_limit"`
	BaseFee    uint64         `json:"base_fee"`
	Coinbase   common.Address `json:"coinbase"`
	PrevRandao common.Hash    `json:"prev_randao"`

	Snapshot PrefixSnapshot `json:"-"`

	Prepare func(context.Context, []byte, types.AccessList) (*types.Transaction, error) `json:"-"`
}

// SuspendedResult contains exact signed envelopes, plus canonical execution receipts.
// It is not a publication authorization: whole-block execution and the interop
// verifier must accept the resulting blocks. Receipt checks alone do not prove state.
type SuspendedResult struct {
	Reverted bool
	Included map[uint64]SuspendedInclusion
}

type SuspendedInclusion struct {
	Transaction *types.Transaction
	GasUsed     uint64
	Success     bool
	Logs        []*types.Log
}

// BuildSuspended invokes one private Rust worker, serves pinned reads/signing, and
// returns only after the coordinator completes its single final replay on each chain.
// Cancellation kills the worker. There is no retry or RPC discovery fallback.
func BuildSuspended(ctx context.Context, binary string, chains []SuspendedChain, root uint64, nonce uint64, target common.Address, data []byte, maxCalls uint16) (*SuspendedResult, error) {
	if binary == "" || maxCalls == 0 || len(chains) == 0 || len(chains) > 32 {
		return nil, fmt.Errorf("invalid suspended builder configuration")
	}
	byID := make(map[uint64]SuspendedChain, len(chains))
	for _, chain := range chains {
		if _, exists := byID[chain.Chain]; exists || chain.Snapshot == nil || chain.Prepare == nil || chain.Spec != "LAGOON" || chain.Number == 0 || chain.PrefixGas > chain.GasLimit {
			return nil, fmt.Errorf("invalid or duplicate suspended chain")
		}
		byID[chain.Chain] = chain
	}
	if _, ok := byID[root]; !ok {
		return nil, fmt.Errorf("missing root chain")
	}
	chains = append([]SuspendedChain(nil), chains...)
	sort.Slice(chains, func(i, j int) bool { return chains[i].Chain < chains[j].Chain })
	cmd := exec.CommandContext(ctx, binary)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, err
	}
	waited := false
	defer func() {
		_ = stdin.Close()
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	enc := json.NewEncoder(stdin)
	if err := enc.Encode(map[string]any{"version": 1, "root": root, "nonce": hexutil.EncodeUint64(nonce), "target": target, "data": hexutil.Bytes(data), "max_calls": maxCalls, "max_bytes": 1 << 20, "chains": chains}); err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), 16<<20)
	counts := make(map[uint64]int)
	signed := make(map[uint64]*types.Transaction)
	for requests := 0; requests < 100_000 && scanner.Scan(); requests++ {
		var msg struct {
			Method   string           `json:"method"`
			Chain    uint64           `json:"chain"`
			Address  common.Address   `json:"address"`
			Index    hexutil.Big      `json:"index"`
			Number   uint64           `json:"number"`
			Data     hexutil.Bytes    `json:"data"`
			Accesses types.AccessList `json:"accesses"`
			Error    string           `json:"error"`
			Reverted bool             `json:"reverted"`
			Included []struct {
				Chain   uint64        `json:"chain"`
				Raw     hexutil.Bytes `json:"raw"`
				GasUsed uint64        `json:"gas_used"`
				Success bool          `json:"success"`
				Logs    []struct {
					Address common.Address `json:"address"`
					Topics  []common.Hash  `json:"topics"`
					Data    hexutil.Bytes  `json:"data"`
				} `json:"logs"`
			} `json:"included"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			return nil, fmt.Errorf("decode suspended worker: %w", err)
		}
		if msg.Method == "error" {
			return nil, fmt.Errorf("suspended build: %s", msg.Error)
		}
		if msg.Method == "complete" {
			result := &SuspendedResult{Reverted: msg.Reverted, Included: make(map[uint64]SuspendedInclusion)}
			for _, item := range msg.Included {
				tx := signed[item.Chain]
				if tx == nil || counts[item.Chain] != 2 {
					return nil, fmt.Errorf("missing unique canonical preparation")
				}
				if _, exists := result.Included[item.Chain]; exists {
					return nil, fmt.Errorf("duplicate included chain")
				}
				raw, err := tx.MarshalBinary()
				if err != nil || !bytes.Equal(raw, item.Raw) || item.GasUsed > tx.Gas() {
					return nil, fmt.Errorf("worker returned a different envelope or invalid gas")
				}
				inclusion := SuspendedInclusion{Transaction: tx, GasUsed: item.GasUsed, Success: item.Success}
				for _, log := range item.Logs {
					inclusion.Logs = append(inclusion.Logs, &types.Log{Address: log.Address, Topics: log.Topics, Data: log.Data})
				}
				result.Included[item.Chain] = inclusion
			}
			if _, ok := result.Included[root]; !ok {
				return nil, fmt.Errorf("worker omitted root")
			}
			_ = stdin.Close()
			// Require EOF and a successful exit, including after a purported completion.
			if scanner.Scan() {
				return nil, fmt.Errorf("unexpected data after worker completion")
			}
			if err := scanner.Err(); err != nil {
				return nil, err
			}
			err := cmd.Wait()
			waited = true
			if err != nil {
				return nil, fmt.Errorf("suspended worker exit: %w", err)
			}
			return result, nil
		}
		chain, ok := byID[msg.Chain]
		if !ok {
			return nil, fmt.Errorf("worker requested unknown chain")
		}
		var value any
		switch msg.Method {
		case "account":
			value, err = chain.Snapshot.Account(ctx, msg.Address)
		case "storage":
			value, err = chain.Snapshot.Storage(ctx, msg.Address, common.BigToHash(msg.Index.ToInt()))
		case "block_hash":
			value, err = chain.Snapshot.BlockHash(ctx, msg.Number)
		case "envelope":
			counts[msg.Chain]++
			if counts[msg.Chain] > 2 {
				return nil, fmt.Errorf("worker requested a discovery retry")
			}
			var tx *types.Transaction
			tx, err = chain.Prepare(ctx, msg.Data, msg.Accesses)
			if err == nil {
				if tx == nil || tx.Type() != types.DynamicFeeTxType || !tx.ChainId().IsUint64() || bigs.Uint64Strict(tx.ChainId()) != msg.Chain {
					return nil, fmt.Errorf("invalid prepared envelope")
				}
				var raw []byte
				raw, err = tx.MarshalBinary()
				value = hexutil.Bytes(raw)
				signed[msg.Chain] = tx
			}
		default:
			return nil, fmt.Errorf("unsupported worker request %q", msg.Method)
		}
		if err != nil {
			return nil, fmt.Errorf("suspended %s: %w", msg.Method, err)
		}
		if err = enc.Encode(map[string]any{"result": value}); err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("suspended worker ended without a bundle or exhausted its request budget")
}
