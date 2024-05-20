package state

import (
	"fmt"

	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/kv"
)

func GetAccount(tx kv.Tx, addr common.Address) ([]byte, error) {
	cursor, err := tx.Cursor(kv.PlainState)
	if err != nil {
		return nil, fmt.Errorf("failed to create plain state cursor: %w", err)
	}
	defer cursor.Close()

	var value []byte
	for k, v, err := cursor.First(); k != nil; k, v, err = cursor.Next() {
		if err != nil {
			return nil, fmt.Errorf("failed to read storage from database: %w", err)
		}
		if len(k) == 20 && common.BytesToAddress(k[:20]) == addr {
			value = v
			break
		}
	}
	return value, nil
}

func GetAllAccounts(tx kv.Tx) (map[common.Address][]byte, error) {
	cursor, err := tx.Cursor(kv.PlainState)
	if err != nil {
		return nil, fmt.Errorf("failed to create plain state cursor: %w", err)
	}
	defer cursor.Close()

	accounts := make(map[common.Address][]byte)
	for k, v, err := cursor.First(); k != nil; k, v, err = cursor.Next() {
		if err != nil {
			return nil, fmt.Errorf("failed to read storage from database: %w", err)
		}
		if len(k) == 20 {
			accounts[common.BytesToAddress(k[:20])] = v
		}
	}
	return accounts, nil
}

func GetStorage(tx kv.Tx, addr common.Address, hash common.Hash) (*common.Hash, error) {
	cursor, err := tx.Cursor(kv.PlainState)
	if err != nil {
		return nil, fmt.Errorf("failed to create plain state cursor: %w", err)
	}
	defer cursor.Close()

	var value common.Hash
	for k, v, err := cursor.First(); k != nil; k, v, err = cursor.Next() {
		if err != nil {
			return nil, fmt.Errorf("failed to read storage from database: %w", err)
		}
		// Storage is 20 bytes account address + 8 byte incarnation + 32 byte storage key
		if len(k) == 60 && common.BytesToAddress(k[:20]) == addr && common.BytesToHash(k[28:]) == hash {
			value = common.BytesToHash(v)
			break
		}
	}
	return &value, nil
}

func GetAllStorages(tx kv.Tx) (map[common.Address]map[common.Hash]common.Hash, error) {
	cursor, err := tx.Cursor(kv.PlainState)
	if err != nil {
		return nil, fmt.Errorf("failed to create plain state cursor: %w", err)
	}
	defer cursor.Close()

	storages := make(map[common.Address]map[common.Hash]common.Hash)
	for k, v, err := cursor.First(); k != nil; k, v, err = cursor.Next() {
		if err != nil {
			return nil, fmt.Errorf("failed to read storage from database: %w", err)
		}
		// Storage is 20 bytes account address + 8 byte incarnation + 32 byte storage key
		if len(k) == 60 {
			addr := common.BytesToAddress(k[:20])
			key := common.BytesToHash(k[28:])
			val := common.BytesToHash(v)
			if _, ok := storages[addr]; !ok {
				storages[addr] = make(map[common.Hash]common.Hash)
			}
			storages[addr][key] = val
		}
	}
	return storages, nil
}

func GetContractCode(tx kv.Tx, addr common.Address) (*common.Hash, error) {
	contractCursor, err := tx.Cursor(kv.PlainContractCode)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract code cursor: %w", err)
	}
	codeCursor, err := tx.Cursor(kv.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to create code cursor: %w", err)
	}

	var code common.Hash
	for contractKey, contractValue, err := contractCursor.First(); contractKey != nil; contractKey, contractValue, err = contractCursor.Next() {
		if err != nil {
			return nil, fmt.Errorf("failed to read contract code from database: %w", err)
		}
		if common.BytesToAddress(contractKey[:20]) == addr {
			codeHash := common.BytesToHash(contractValue)
			for codeKey, codeValue, err := codeCursor.First(); codeKey != nil; codeKey, codeValue, err = codeCursor.Next() {
				if err != nil {
					return nil, fmt.Errorf("failed to read code from database: %w", err)
				}
				if common.BytesToHash(codeKey) == codeHash {
					code = common.BytesToHash(codeValue)
					break
				}
			}
		}
	}
	return &code, nil
}

func GetContractCodeHash(tx kv.Tx, addr common.Address) (*common.Hash, error) {
	cursor, err := tx.Cursor(kv.PlainContractCode)
	if err != nil {
		return nil, fmt.Errorf("failed to create plain contract code cursor: %w", err)
	}
	defer cursor.Close()
	var hash common.Hash
	for k, v, err := cursor.First(); k != nil; k, v, err = cursor.Next() {
		if err != nil {
			return nil, fmt.Errorf("failed to read code from database: %w", err)
		}
		if common.BytesToAddress(k[:20]) == addr {
			hash = common.BytesToHash(v)
			break
		}
	}
	return &hash, nil
}

func ForEachStorage(tx kv.Tx, addr common.Address, cb func(key, value common.Hash) bool) error {
	cursor, err := tx.Cursor(kv.PlainState)
	if err != nil {
		return fmt.Errorf("failed to create plain state cursor: %w", err)
	}
	defer cursor.Close()

	for k, v, err := cursor.First(); k != nil; k, v, err = cursor.Next() {
		if err != nil {
			return fmt.Errorf("failed to read storage from database: %w", err)
		}
		// Storage is 20 bytes account address + 8 byte incarnation + 32 byte storage key
		if len(k) == 60 && common.BytesToAddress(k[:20]) == addr {
			if !cb(common.BytesToHash(k[28:]), common.BytesToHash(v)) {
				return nil
			}
		}
	}
	return nil
}
