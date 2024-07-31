package foundry

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// AccountDiff represents a Forge allocs diff.
type AccountDiff struct {
	// Balance diff, ignored if null
	Balance *hexutil.U256 `json:"balance,omitempty"`
	// Nonce diff, ignored if null
	Nonce *hexutil.Uint64 `json:"nonce,omitempty"`
	// Code diff, if not null but empty bytes, then code was deleted. Ignored if null.
	Code *hexutil.Bytes `json:"code,omitempty"`
	// Storage diff. Deleted storage slots are set to "null"
	Storage map[common.Hash]*common.Hash `json:"storage,omitempty"`
}

// ForgeAllocsDiff is the set of changes to a ForgeAllocs.
type ForgeAllocsDiff struct {
	// Accounts diff.
	// Accounts that were deleted have explicit "null" entries.
	Accounts map[common.Address]*AccountDiff `json:"accounts"`
}

func diffAccount(preAcc, postAcc *types.Account) (*AccountDiff, error) {
	if preAcc.Balance == nil || postAcc.Balance == nil {
		return nil, fmt.Errorf("balance account value may not be nil: pre: %v, post: %v", preAcc.Balance, postAcc.Balance)
	}
	var accDiff AccountDiff
	if postAcc.Balance.Cmp(preAcc.Balance) != 0 {
		u256, overflow := uint256.FromBig(postAcc.Balance)
		if overflow {
			return nil, fmt.Errorf("post-account has invalid balance %d", postAcc.Balance)
		}
		accDiff.Balance = (*hexutil.U256)(u256)
	}
	if preAcc.Nonce != postAcc.Nonce {
		nonceCpy := postAcc.Nonce
		accDiff.Nonce = (*hexutil.Uint64)(&nonceCpy)
	}
	if bytes.Compare(preAcc.Code, postAcc.Code) != 0 {
		cpy := bytes.Clone(postAcc.Code)
		accDiff.Code = (*hexutil.Bytes)(&cpy)
	}
	// Storage diff
	for k, preV := range preAcc.Storage {
		if postV, ok := postAcc.Storage[k]; ok {
			if preV != postV {
				accDiff.Storage[k] = &postV // modified storage
			}
		} else {
			accDiff.Storage[k] = nil // deleted storage
		}
	}
	for k, postV := range postAcc.Storage {
		if _, ok := preAcc.Storage[k]; !ok {
			postV := postV
			accDiff.Storage[k] = &postV // new storage
		}
	}
	return &accDiff, nil
}

func accToDiffAcc(acc *types.Account) (*AccountDiff, error) {
	if acc.Balance == nil {
		return nil, fmt.Errorf("balance account value may not be nil: %v", acc.Balance)
	}
	var accDiff AccountDiff
	u256, overflow := uint256.FromBig(acc.Balance)
	if overflow {
		return nil, fmt.Errorf("post-account has invalid balance %d", acc.Balance)
	}
	accDiff.Balance = (*hexutil.U256)(u256)
	nonceCpy := acc.Nonce
	accDiff.Nonce = (*hexutil.Uint64)(&nonceCpy)
	if len(acc.Code) > 0 {
		cpy := bytes.Clone(acc.Code)
		accDiff.Code = (*hexutil.Bytes)(&cpy)
	}
	accDiff.Storage = make(map[common.Hash]*common.Hash, len(acc.Storage))
	for k, v := range acc.Storage {
		v := v // Technically not a problem in later Go versions, but linters might still complain otherwise
		accDiff.Storage[k] = &v
	}
	return &accDiff, nil
}

func ComputeDiff(pre, post *ForgeAllocs) (*ForgeAllocsDiff, error) {
	diff := &ForgeAllocsDiff{Accounts: make(map[common.Address]*AccountDiff)}
	for addr, preAcc := range pre.Accounts {
		if postAcc, ok := post.Accounts[addr]; ok { // modified account
			accDiff, err := diffAccount(&preAcc, &postAcc)
			if err != nil {
				return nil, fmt.Errorf("failed to diff accounts at address %s: %w", addr, err)
			}
			diff.Accounts[addr] = accDiff
		} else {
			// account was deleted
			diff.Accounts[addr] = nil
		}
	}
	for addr, postAcc := range post.Accounts {
		if _, ok := pre.Accounts[addr]; !ok { // new account in post-state
			accDiff, err := accToDiffAcc(&postAcc)
			if err != nil {
				return nil, fmt.Errorf("failed to build diff at address %s: %w", addr, err)
			}
			diff.Accounts[addr] = accDiff
		}
	}
	return diff, nil
}

func ApplyDiff(pre *ForgeAllocs, diff *ForgeAllocsDiff) *ForgeAllocs {
	post := pre.Copy()

	for addr, accDiff := range diff.Accounts {
		acc := post.Accounts[addr]
		if accDiff.Balance != nil {
			acc.Balance = (*uint256.Int)(accDiff.Balance).ToBig()
		}
		if accDiff.Nonce != nil {
			acc.Nonce = uint64(*accDiff.Nonce)
		}
		if accDiff.Code != nil {
			acc.Code = bytes.Clone(*accDiff.Code)
		}
		if len(accDiff.Storage) > 0 && acc.Storage == nil {
			acc.Storage = make(map[common.Hash]common.Hash)
		}
		for k, v := range accDiff.Storage {
			if v == nil {
				delete(acc.Storage, k)
			} else {
				acc.Storage[k] = *v
			}
		}
		post.Accounts[addr] = acc
	}
	return post
}
