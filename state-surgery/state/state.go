package state

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum-optimism/optimism/l2geth/common/hexutil"
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
func EncodeStorage(entry solc.StorageLayoutEntry, value any, storageType solc.StorageLayoutType) (*EncodedStorage, error) {
	// TODO: handle nested storage
	slot := new(big.Int).SetUint64(uint64(entry.Slot))
	key := common.BigToHash(slot)

	if entry.Offset != 0 {
		return nil, fmt.Errorf("%s has a non zero offset", entry.Label)
	}
	if storageType.NumberOfBytes > 32 {
		return nil, fmt.Errorf("%s is larger than 32 bytes", entry.Label)
	}

	if storageType.Encoding == "inplace" {
		if storageType.Label == "address" || strings.HasPrefix(storageType.Label, "contract") {
			address, ok := value.(common.Address)
			if !ok {
				str, ok := value.(string)
				if !ok {
					return nil, fmt.Errorf("invalid address for %s", entry.Label)
				}
				address = common.HexToAddress(str)
			}

			value := address.Hash()
			return &EncodedStorage{
				Key:   key,
				Value: value,
			}, nil
		}

		if storageType.Label == "bool" {
			name := reflect.TypeOf(value).Name()
			val := common.Hash{}
			switch name {
			case "bool":
				boolean, ok := value.(bool)
				if !ok {
					return nil, fmt.Errorf("cannot parse value for %s", entry.Label)
				}
				if boolean {
					val = common.BigToHash(common.Big1)
				}
			case "string":
				boolean, ok := value.(string)
				if !ok {
					return nil, fmt.Errorf("cannot parse value for %s", entry.Label)
				}
				if boolean == "true" {
					val = common.BigToHash(common.Big1)
				}
			}

			return &EncodedStorage{
				Key:   key,
				Value: val,
			}, nil
		}

		if strings.HasPrefix(storageType.Label, "bytes") {
			panic("bytes unimplemented")
		}

		if strings.HasPrefix(storageType.Label, "uint") {
			name := reflect.TypeOf(value).Name()
			var number *big.Int
			switch name {
			case "uint":
				val, ok := value.(uint)
				if !ok {
					return nil, fmt.Errorf("cannot parse value for %s", entry.Label)
				}
				number = new(big.Int).SetUint64(uint64(val))
			case "int":
				val, ok := value.(int)
				if !ok {
					return nil, fmt.Errorf("cannot parse value for %s", entry.Label)
				}
				number = new(big.Int).SetUint64(uint64(val))
			case "uint64":
				val, ok := value.(uint64)
				if !ok {
					return nil, fmt.Errorf("cannot parse value for %s", entry.Label)
				}
				number = new(big.Int).SetUint64(val)
			case "string":
				val, ok := value.(string)
				if !ok {
					return nil, fmt.Errorf("cannot parse value for %s", entry.Label)
				}
				var err error
				number, err = hexutil.DecodeBig(val)
				if err != nil {
					if errors.Is(err, hexutil.ErrMissingPrefix) {
						number, ok = new(big.Int).SetString(val, 10)
						if !ok {
							return nil, fmt.Errorf("cannot parse value for %s", entry.Label)
						}
					} else if errors.Is(err, hexutil.ErrLeadingZero) {
						number, ok = new(big.Int).SetString(val[2:], 16)
						if !ok {
							return nil, fmt.Errorf("cannot parse value for %s", entry.Label)
						}
					}
				}

			case "":
				val, ok := value.(*big.Int)
				if !ok {
					return nil, fmt.Errorf("cannot parse value for %s", entry.Label)
				}
				number = val
			}

			if number == nil {
				return nil, fmt.Errorf("cannot parse value for %s", entry.Label)
			}

			return &EncodedStorage{
				Key:   key,
				Value: common.BigToHash(number),
			}, nil
		}

		if strings.HasPrefix(storageType.Label, "int") {
			panic("setting int storage slots is unimplemented")
		}
		// end handling inplace storage
	} else if storageType.Encoding == "bytes" {
		panic("setting bytes storage slots is unimplemented")
	} else if storageType.Encoding == "mapping" {
		panic("setting mapping storage slots is unimplemented")
	} else if storageType.Encoding == "dynamic_array" {
		panic("setting dynamic_array storage slots is unimplemented")
	}

	return nil, nil
}

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
			return nil, err
		}

		encodedStorage = append(encodedStorage, storage)
	}

	return encodedStorage, nil
}
