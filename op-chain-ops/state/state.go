package state

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/solc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
)

var (
	errInvalidType   = errors.New("invalid type")
	errUnimplemented = errors.New("type unimplemented")
)

// StorageValues represents the values to be set in storage.
// The key is the name of the storage variable and the value
// is the value to set in storage.
type StorageValues map[string]any

// StorageConfig represents the storage configuration for the L2 predeploy
// contracts.
type StorageConfig map[string]StorageValues

// EncodedStorage represents the storage key and value serialized
// to be placed in Ethereum state.
type EncodedStorage struct {
	Key   common.Hash
	Value common.Hash
}

// EncodeStorage will encode a storage layout
func EncodeStorage(entry solc.StorageLayoutEntry, value any, storageType solc.StorageLayoutType) ([]*EncodedStorage, error) {
	if storageType.NumberOfBytes > 32 {
		return nil, fmt.Errorf("%s is larger than 32 bytes", storageType.Encoding)
	}

	encoded, err := EncodeStorageKeyValue(value, entry, storageType)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

// SetStorage will set the storage values in a db given a contract name,
// address and the storage values
func SetStorage(name string, address common.Address, values StorageValues, db vm.StateDB) error {
	layout, err := bindings.GetStorageLayout(name)
	if err != nil {
		return fmt.Errorf("cannot set storage: %w", err)
	}
	slots, err := ComputeStorageSlots(layout, values)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	for _, slot := range slots {
		db.SetState(address, slot.Key, slot.Value)
		log.Trace("setting storage", "address", address.Hex(), "key", slot.Key.Hex(), "value", slot.Value.Hex())
	}
	return nil
}

// ComputeStorageSlots will compute the storage slots for a given contract.
func ComputeStorageSlots(layout *solc.StorageLayout, values StorageValues) ([]*EncodedStorage, error) {
	encodedStorage := make([]*EncodedStorage, 0)

	for label, value := range values {
		// Handle directly storing key value pairs where the key is a 32 byte storage slot
		// instead of a name of a storage slot.
		if has0xPrefix(label) && isHex(label) && len(label) == 66 {
			key := common.HexToHash(label)
			val, err := EncodeBytes32Value(value, 0)
			if err != nil {
				return nil, fmt.Errorf("cannot encode storage for %s: %w", label, err)
			}
			encodedStorage = append(encodedStorage, &EncodedStorage{key, val})
			continue
		}

		var target solc.StorageLayoutEntry
		for _, entry := range layout.Storage {
			if label == entry.Label {
				target = entry
				break
			}
		}
		if target.Label == "" {
			return nil, fmt.Errorf("storage layout entry for %s not found", label)
		}

		storageType := layout.Types[target.Type]
		if storageType.Label == "" {
			return nil, fmt.Errorf("storage type for %s not found", label)

		}

		storage, err := EncodeStorage(target, value, storageType)
		if err != nil {
			return nil, fmt.Errorf("cannot encode storage for %s: %w", target.Label, err)
		}

		encodedStorage = append(encodedStorage, storage...)
	}

	results := MergeStorage(encodedStorage)

	return results, nil
}

// MergeStorage will combine any overlapping storage slots for
// when values are tightly packed. Do this by checking to see if any
// of the produced storage slots have a matching key, if so use a
// binary or to add the storage values together
func MergeStorage(storage []*EncodedStorage) []*EncodedStorage {
	encodedKV := make(map[common.Hash]common.Hash)
	var encodedKeys []common.Hash // for deterministic result order
	for _, storage := range storage {
		if prev, ok := encodedKV[storage.Key]; ok {
			combined := new(big.Int).Or(prev.Big(), storage.Value.Big())
			encodedKV[storage.Key] = common.BigToHash(combined)
		} else {
			encodedKV[storage.Key] = storage.Value
			encodedKeys = append(encodedKeys, storage.Key)
		}
	}

	results := make([]*EncodedStorage, 0)
	for _, key := range encodedKeys {
		val := encodedKV[key]
		results = append(results, &EncodedStorage{key, val})
	}
	return results
}

// has0xPrefix validates str begins with '0x' or '0X'.
func has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

// isHexCharacter returns bool of c being a valid hexadecimal.
func isHexCharacter(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

// isHex validates whether each byte is valid hexadecimal string.
func isHex(str string) bool {
	if len(str)%2 != 0 {
		return false
	}
	for _, c := range []byte(str) {
		if !isHexCharacter(c) {
			return false
		}
	}
	return true
}
