package forge

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// Transaction is an unsigned transaction produced by forge's script execution. Fees and gas
// limits are deliberately not carried over: txmgr estimates them against the live chain.
type Transaction struct {
	From       common.Address   `json:"from"`
	To         *common.Address  `json:"to"`
	Nonce      hexutil.Uint64   `json:"nonce"`
	Value      *hexutil.U256    `json:"value"`
	Input      hexutil.Bytes    `json:"input"`
	ChainID    *hexutil.Big     `json:"chainId"`
	Type       hexutil.Uint64   `json:"type"`
	AccessList types.AccessList `json:"accessList"`
}

// readTransactions reads the sole dry-run manifest from this invocation's private broadcast
// directory before RunScript removes it. Only these private records are a signing authority;
// transaction lists returned by an RPC endpoint are not script output.
func readTransactions(dir string, stdout string) ([]Transaction, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*", "*", "dry-run", "*-latest.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to locate forge transactions: %w", err)
	}
	// Pinned forge emits no manifest for a successful script with no broadcasts (including
	// idempotent deploy scripts). RunScript has already checked its exit status and compile guard.
	// forge prints the other markers only when there are transactions, so a forge that stops
	// writing the manifest fails here instead of looking like a script that sends nothing.
	noTransactions := strings.Contains(stdout, "Script ran successfully.") &&
		!strings.Contains(stdout, "Transactions saved to:") &&
		!strings.Contains(stdout, "SIMULATION COMPLETE") &&
		!strings.Contains(stdout, "SKIPPING ON CHAIN SIMULATION")
	if len(paths) == 0 && noTransactions {
		return []Transaction{}, nil
	}
	if len(paths) != 1 {
		return nil, fmt.Errorf("expected one forge dry-run transaction file, got %d:\n%s", len(paths), stdout)
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read forge transactions: %w", err)
	}
	var manifest struct {
		Transactions *[]struct {
			Type        string          `json:"transactionType"`
			Transaction json.RawMessage `json:"transaction"`
		} `json:"transactions"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("invalid forge transaction file: %w", err)
	}
	if manifest.Transactions == nil {
		return nil, errors.New("forge transaction file has no transaction list")
	}
	out := make([]Transaction, 0, len(*manifest.Transactions))
	for i, record := range *manifest.Transactions {
		tx, err := decodeTransaction(record.Transaction)
		if err != nil {
			return nil, fmt.Errorf("invalid forge transaction %d: %w", i, err)
		}
		switch record.Type {
		case "CREATE":
			if tx.To != nil {
				return nil, fmt.Errorf("forge CREATE transaction %d has a target", i)
			}
		case "CALL", "CREATE2":
			if tx.To == nil {
				return nil, fmt.Errorf("forge %s transaction %d has no target", record.Type, i)
			}
		default:
			return nil, fmt.Errorf("unsupported forge transaction kind %q", record.Type)
		}
		out = append(out, tx)
	}
	return out, nil
}

func decodeTransaction(data []byte) (Transaction, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return Transaction{}, err
	}
	for _, name := range []string{"from", "to", "nonce", "value", "input", "chainId"} {
		value, ok := fields[name]
		if !ok || (name != "to" && bytes.Equal(bytes.TrimSpace(value), []byte("null"))) {
			return Transaction{}, fmt.Errorf("missing %s", name)
		}
	}
	// Fail closed rather than silently dropping authorizations, blob data, or future fields
	// that could change what txmgr signs. Only ordinary transaction fields are supported.
	for name := range fields {
		switch name {
		case "from", "to", "nonce", "value", "input", "chainId", "type", "accessList",
			"gas", "gasPrice", "maxFeePerGas", "maxPriorityFeePerGas":
		default:
			return Transaction{}, fmt.Errorf("unsupported transaction field %q", name)
		}
	}
	var tx Transaction
	if err := json.Unmarshal(data, &tx); err != nil {
		return Transaction{}, err
	}
	return tx, nil
}
