package state

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// EncodeStorageKeyValue encodes the key value pair that is stored in state
// given a StorageLayoutEntry and StorageLayoutType. A single input may result
// in multiple outputs. Unknown or unimplemented types will return an error.
// Note that encoding uints is *not* overflow safe, so be sure to check
// the ABI before setting very large values
func EncodeStorageKeyValue(value any, entry solc.StorageLayoutEntry, storageType solc.StorageLayoutType) ([]*EncodedStorage, error) {
	label := storageType.Label
	encoded := make([]*EncodedStorage, 0)

	key := encodeSlotKey(entry)
	switch storageType.Encoding {
	case "inplace":
		switch label {
		case "bool":
			val, err := EncodeBoolValue(value, entry.Offset)
			if err != nil {
				return nil, fmt.Errorf("cannot encode %s: %w", storageType.Encoding, err)
			}
			encoded = append(encoded, &EncodedStorage{key, val})
		case "address":
			val, err := EncodeAddressValue(value, entry.Offset)
			if err != nil {
				return nil, fmt.Errorf("cannot encode %s: %w", storageType.Encoding, err)
			}
			encoded = append(encoded, &EncodedStorage{key, val})
		case "bytes":
			return nil, fmt.Errorf("%w: %s", errUnimplemented, label)
		case "bytes32":
			val, err := EncodeBytes32Value(value, entry.Offset)
			if err != nil {
				return nil, fmt.Errorf("cannot encode %s: %w", storageType.Encoding, err)
			}
			encoded = append(encoded, &EncodedStorage{key, val})
		default:
			switch true {
			case strings.HasPrefix(label, "contract"):
				val, err := EncodeAddressValue(value, entry.Offset)
				if err != nil {
					return nil, fmt.Errorf("cannot encode %s: %w", storageType.Encoding, err)
				}
				encoded = append(encoded, &EncodedStorage{key, val})
			case strings.HasPrefix(label, "uint"):
				val, err := EncodeUintValue(value, entry.Offset)
				if err != nil {
					return nil, fmt.Errorf("cannot encode %s: %w", storageType.Encoding, err)
				}
				encoded = append(encoded, &EncodedStorage{key, val})
			default:
				// structs are not supported
				return nil, fmt.Errorf("cannot encode %s: %w", storageType.Encoding, errUnimplemented)
			}
		}
	case "dynamic_array":
	case "bytes":
		switch label {
		case "string":
			val, err := EncodeStringValue(value, entry.Offset)
			if err != nil {
				return nil, fmt.Errorf("cannot encode %s: %w", storageType.Encoding, errUnimplemented)
			}
			encoded = append(encoded, &EncodedStorage{key, val})
		default:
			return nil, fmt.Errorf("%w: %s", errUnimplemented, label)
		}
	case "mapping":
		if strings.HasPrefix(storageType.Value, "mapping") {
			return nil, fmt.Errorf("%w: %s", errUnimplemented, "nested mappings")
		}

		values, ok := value.(map[any]any)
		if !ok {
			return nil, fmt.Errorf("mapping must be map[any]any")
		}

		keyEncoder, err := getElementEncoder(storageType, "key")
		if err != nil {
			return nil, err
		}
		valueEncoder, err := getElementEncoder(storageType, "value")
		if err != nil {
			return nil, err
		}

		// Mapping values have 0 offset
		for rawKey, rawVal := range values {
			encodedKey, err := keyEncoder(rawKey, 0)
			if err != nil {
				return nil, err
			}

			encodedSlot := encodeSlotKey(entry)

			preimage := [64]byte{}
			copy(preimage[0:32], encodedKey.Bytes())
			copy(preimage[32:64], encodedSlot.Bytes())

			hash := crypto.Keccak256(preimage[:])
			key := common.BytesToHash(hash)

			val, err := valueEncoder(rawVal, 0)
			if err != nil {
				return nil, err
			}
			encoded = append(encoded, &EncodedStorage{key, val})
		}
	default:
		return nil, fmt.Errorf("unknown encoding %s: %w", storageType.Encoding, errUnimplemented)
	}
	return encoded, nil
}

// encodeSlotKey will encode the storage slot key. This does not
// support mappings.
func encodeSlotKey(entry solc.StorageLayoutEntry) common.Hash {
	slot := new(big.Int).SetUint64(uint64(entry.Slot))
	return common.BigToHash(slot)
}

// ElementEncoder is a function that can encode an element
// based on a solidity type
type ElementEncoder func(value any, offset uint) (common.Hash, error)

// getElementEncoder will return the correct ElementEncoder
// given a solidity type. The kind refers to the key or the value
// when getting an encoder for a mapping. This is only useful
// if the key itself is not populated for some reason.
func getElementEncoder(storageType solc.StorageLayoutType, kind string) (ElementEncoder, error) {
	var target string
	if kind == "key" {
		target = storageType.Key
	} else if kind == "value" {
		target = storageType.Value
	} else {
		return nil, fmt.Errorf("unknown storage %s", kind)
	}

	switch target {
	case "t_address":
		return EncodeAddressValue, nil
	case "t_bool":
		return EncodeBoolValue, nil
	case "t_bytes32":
		return EncodeBytes32Value, nil
	default:
		if strings.HasPrefix(target, "t_uint") {
			return EncodeUintValue, nil
		}
	}

	// Special case fallback if the target is empty, pull it
	// from the label. This requires knowledge of whether we want
	// the key or the value in the label.
	if target == "" {
		r := regexp.MustCompile(`mapping\((?P<key>[[:alnum:]]*) => (?P<value>[[:alnum:]]*)\)`)
		result := r.FindStringSubmatch(storageType.Label)

		for i, key := range r.SubexpNames() {
			if kind == key {
				res := "t_" + result[i]
				layout := solc.StorageLayoutType{}
				if kind == "key" {
					layout.Key = res
				} else if kind == "value" {
					layout.Value = res
				} else {
					return nil, fmt.Errorf("unknown storage %s", kind)
				}
				return getElementEncoder(layout, kind)
			}
		}
	}
	return nil, fmt.Errorf("unsupported type: %s", target)
}

// EncodeBytes32Value will encode a bytes32 value. The offset
// is included so that it can implement the ElementEncoder
// interface, but the offset must always be 0.
func EncodeBytes32Value(value any, offset uint) (common.Hash, error) {
	if offset != 0 {
		return common.Hash{}, errors.New("offset must be 0")
	}
	return encodeBytes32Value(value)
}

// encodeBytes32Value implements the encoding of a bytes32
// value into a common.Hash that is suitable for storage.
func encodeBytes32Value(value any) (common.Hash, error) {
	name := reflect.TypeOf(value).Name()
	switch name {
	case "string":
		str, ok := value.(string)
		if !ok {
			return common.Hash{}, errInvalidType
		}
		val, err := hexutil.Decode(str)
		if err != nil {
			return common.Hash{}, err
		}
		return common.BytesToHash(val), nil
	case "Hash":
		hash, ok := value.(common.Hash)
		if !ok {
			return common.Hash{}, errInvalidType
		}
		return hash, nil
	default:
		return common.Hash{}, errInvalidType
	}
}

// EncodeStringValue will encode a string to a type suitable
// for storage in state. The offset must be 0.
func EncodeStringValue(value any, offset uint) (common.Hash, error) {
	if offset != 0 {
		return common.Hash{}, errors.New("offset must be 0")
	}
	return encodeStringValue(value)
}

// encodeStringValue implements the string encoding. Values larger
// than 31 bytes are not supported because they will be stored
// in multiple storage slots.
func encodeStringValue(value any) (common.Hash, error) {
	name := reflect.TypeOf(value).Name()

	switch name {
	case "string":
		str, ok := value.(string)
		if !ok {
			return common.Hash{}, errInvalidType
		}

		data := []byte(str)
		// Values that are 32 bytes or longer are not supported
		if len(data) >= 32 {
			return common.Hash{}, errors.New("string value too long")
		}
		// The data is right padded with 2 * the length
		// of the data in the final byte
		padded := common.RightPadBytes(data, 32)
		padded[len(padded)-1] = byte(len(data) * 2)

		return common.BytesToHash(padded), nil

	default:
		return common.Hash{}, errInvalidType
	}
}

// EncodeBoolValue will encode a boolean value given a storage
// offset.
func EncodeBoolValue(value any, offset uint) (common.Hash, error) {
	val, err := encodeBoolValue(value)
	if err != nil {
		return common.Hash{}, fmt.Errorf("invalid bool: %w", err)
	}
	return handleOffset(val, offset), nil
}

// encodeBoolValue will encode a boolean value into a type
// suitable for solidity storage.
func encodeBoolValue(value any) (common.Hash, error) {
	name := reflect.TypeOf(value).Name()
	switch name {
	case "bool":
		boolean, ok := value.(bool)
		if !ok {
			return common.Hash{}, errInvalidType
		}
		if boolean {
			return common.BigToHash(common.Big1), nil
		} else {
			return common.Hash{}, nil
		}
	case "string":
		boolean, ok := value.(string)
		if !ok {
			return common.Hash{}, errInvalidType
		}
		if boolean == "true" {
			return common.BigToHash(common.Big1), nil
		} else {
			return common.Hash{}, nil
		}
	default:
		return common.Hash{}, errInvalidType
	}
}

// EncodeAddressValue will encode an address like value given a
// storage offset.
func EncodeAddressValue(value any, offset uint) (common.Hash, error) {
	val, err := encodeAddressValue(value)
	if err != nil {
		return common.Hash{}, fmt.Errorf("invalid address: %w", err)
	}
	return handleOffset(val, offset), nil
}

// encodeAddressValue will encode an address value into
// a type suitable for solidity storage.
func encodeAddressValue(value any) (common.Hash, error) {
	typ := reflect.TypeOf(value)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	name := typ.Name()
	switch name {
	case "Address":
		if reflect.TypeOf(value).Kind() == reflect.Ptr {
			address, ok := value.(*common.Address)
			if !ok {
				return common.Hash{}, errInvalidType
			}
			return eth.AddressAsLeftPaddedHash(*address), nil
		} else {
			address, ok := value.(common.Address)
			if !ok {
				return common.Hash{}, errInvalidType
			}
			return eth.AddressAsLeftPaddedHash(address), nil
		}
	case "string":
		address, ok := value.(string)
		if !ok {
			return common.Hash{}, errInvalidType
		}
		return eth.AddressAsLeftPaddedHash(common.HexToAddress(address)), nil
	default:
		return common.Hash{}, errInvalidType
	}
}

// EncodeUintValue will encode a uint value given a storage offset
func EncodeUintValue(value any, offset uint) (common.Hash, error) {
	val, err := encodeUintValue(value)
	if err != nil {
		return common.Hash{}, fmt.Errorf("invalid uint: %w", err)
	}
	return handleOffset(val, offset), nil
}

// encodeUintValue will encode a uint like type into a
// type suitable for solidity storage.
func encodeUintValue(value any) (common.Hash, error) {
	val := reflect.ValueOf(value)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	name := val.Type().Name()
	switch name {
	case "uint":
		val, ok := value.(uint)
		if !ok {
			return common.Hash{}, errInvalidType
		}
		result := new(big.Int).SetUint64((uint64(val)))
		return common.BigToHash(result), nil
	case "int":
		val, ok := value.(int)
		if !ok {
			return common.Hash{}, errInvalidType
		}
		result := new(big.Int).SetUint64(uint64(val))
		return common.BigToHash(result), nil
	case "uint64":
		val, ok := value.(uint64)
		if !ok {
			return common.Hash{}, errInvalidType
		}
		result := new(big.Int).SetUint64(val)
		return common.BigToHash(result), nil
	case "uint32":
		val, ok := value.(uint32)
		if !ok {
			return common.Hash{}, errInvalidType
		}
		result := new(big.Int).SetUint64(uint64(val))
		return common.BigToHash(result), nil
	case "uint16":
		val, ok := value.(uint16)
		if !ok {
			return common.Hash{}, errInvalidType
		}
		result := new(big.Int).SetUint64(uint64(val))
		return common.BigToHash(result), nil
	case "uint8":
		val, ok := value.(uint8)
		if !ok {
			return common.Hash{}, errInvalidType
		}
		result := new(big.Int).SetUint64(uint64(val))
		return common.BigToHash(result), nil
	case "string":
		val, ok := value.(string)
		if !ok {
			return common.Hash{}, errInvalidType
		}
		number, err := hexutil.DecodeBig(val)
		if err != nil {
			if errors.Is(err, hexutil.ErrMissingPrefix) {
				number, ok = new(big.Int).SetString(val, 10)
				if !ok {
					return common.Hash{}, errInvalidType
				}
			} else if errors.Is(err, hexutil.ErrLeadingZero) {
				number, ok = new(big.Int).SetString(val[2:], 16)
				if !ok {
					return common.Hash{}, errInvalidType
				}
			}
		}
		return common.BigToHash(number), nil
	case "bool":
		val, ok := value.(bool)
		if !ok {
			return common.Hash{}, errInvalidType
		}
		if val {
			return common.Hash{31: 0x01}, nil
		} else {
			return common.Hash{}, nil
		}
	case "Int":
		val, ok := value.(*big.Int)
		if !ok {
			return common.Hash{}, errInvalidType
		}
		return common.BigToHash(val), nil
	default:
		return common.Hash{}, errInvalidType
	}
}

// handleOffset will offset a value in storage by shifting
// it to the left. This is useful for when multiple variables
// are tightly packed in a storage slot.
func handleOffset(hash common.Hash, offset uint) common.Hash {
	if offset == 0 {
		return hash
	}
	number := hash.Big()
	shifted := new(big.Int).Lsh(number, offset*8)
	return common.BigToHash(shifted)
}
