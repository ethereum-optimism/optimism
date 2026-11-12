package foundry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"math/big"
	"os"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type ForgeAllocs struct {
	Accounts types.GenesisAlloc
}

// Copy returns a deep copy of the ForgeAllocs. We can't
// perform a shallow copy here because some tests modify
// the accounts individually.
func (d *ForgeAllocs) Copy() *ForgeAllocs {
	out := make(types.GenesisAlloc, len(d.Accounts))
	for k, v := range d.Accounts {
		out[k] = types.Account{
			Code:    bytes.Clone(v.Code),
			Storage: maps.Clone(v.Storage),
			Balance: new(big.Int).Set(v.Balance),
			Nonce:   v.Nonce,
		}
	}
	return &ForgeAllocs{Accounts: out}
}

func (d ForgeAllocs) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Accounts)
}

func (d *ForgeAllocs) UnmarshalJSON(b []byte) error {
	// forge, since integrating Alloy, likes to hex-encode everything.
	type forgeAllocAccount struct {
		Balance hexutil.U256                `json:"balance"`
		Nonce   hexutil.Uint64              `json:"nonce"`
		Code    hexutil.Bytes               `json:"code,omitempty"`
		Storage map[common.Hash]common.Hash `json:"storage,omitempty"`
	}
	var allocs map[common.Address]forgeAllocAccount
	if err := json.Unmarshal(b, &allocs); err != nil {
		return err
	}
	d.Accounts = make(types.GenesisAlloc, len(allocs))
	for addr, acc := range allocs {
		acc := acc
		d.Accounts[addr] = types.Account{
			Code:    acc.Code,
			Storage: acc.Storage,
			Balance: (*uint256.Int)(&acc.Balance).ToBig(),
			Nonce:   (uint64)(acc.Nonce),
		}
	}
	return nil
}

func LoadForgeAllocs(allocsPath string) (*ForgeAllocs, error) {
	f, err := os.OpenFile(allocsPath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open forge allocs %q: %w", allocsPath, err)
	}
	defer f.Close()
	var out ForgeAllocs
	if err := json.NewDecoder(f).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to json-decode forge allocs %q: %w", allocsPath, err)
	}
	return &out, nil
}
