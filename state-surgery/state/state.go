package state

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/state-surgery/solc"
	"github.com/ethereum/go-ethereum/common"
)

// StorageValues represents the values to be set in storage.
// The key is the name of the storage variable and the value
// is the value to set in storage.
type StorageValues map[string]any

// EncodedStorage represents the storage key and value serialized
// to be placed in Ethereum state.
type EncodedStorage struct {
	Key   common.Hash
	Value common.Hash
}

// EncodedStorage will encode a storage layout
func EncodeStorage(entry solc.StorageLayoutEntry, value any, storageType solc.StorageLayoutType) ([]*EncodedStorage, error) {
	if storageType.NumberOfBytes > 32 {
		return nil, fmt.Errorf("%s is larger than 32 bytes", entry.Label)
	}

	encoded, err := EncodeStorageKeyValue(value, entry, storageType)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

var errInvalidType = errors.New("invalid type")
var errUnimplemented = errors.New("type unimplemented")

// ComputeStorageSlots will compute the storage slots for a given contract.
func ComputeStorageSlots(layout *solc.StorageLayout, values StorageValues) ([]*EncodedStorage, error) {
	encodedStorage := make([]*EncodedStorage, 0)

	for label, value := range values {
		var target solc.StorageLayoutEntry
		for _, entry := range layout.Storage {
			if label == entry.Label {
				target = entry
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
			return nil, fmt.Errorf("cannot encode storage: %w", err)
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
	encoded := make(map[common.Hash]common.Hash)
	for _, storage := range storage {
		if prev, ok := encoded[storage.Key]; ok {
			combined := new(big.Int).Or(prev.Big(), storage.Value.Big())
			encoded[storage.Key] = common.BigToHash(combined)
		} else {
			encoded[storage.Key] = storage.Value
		}
	}

	results := make([]*EncodedStorage, 0)
	for key, val := range encoded {
		results = append(results, &EncodedStorage{key, val})
	}
	return results
}
