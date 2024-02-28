// Go Substrate RPC Client (GSRPC) provides APIs and types around Polkadot and any Substrate-based chain RPC calls
//
// Copyright 2019 Centrifuge GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"fmt"
	"io"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/xxhash"
)

// StorageKey represents typically hashed storage keys of the system.
// Be careful using this in your own structs – it only works as the last value in a struct since it will consume the
// remainder of the encoded data. The reason for this is that it does not contain any length encoding, so it would
// not know where to stop.
type StorageKey []byte

// NewStorageKey creates a new StorageKey type
func NewStorageKey(b []byte) StorageKey {
	return b
}

// CreateStorageKey uses the given metadata and to derive the right hashing of method, prefix as well as arguments to
// create a hashed StorageKey
// Using variadic argument, so caller do not need to construct array of arguments
func CreateStorageKey(meta *Metadata, prefix, method string, args ...[]byte) (StorageKey, error) { //nolint:funlen
	stringKey := []byte(prefix + " " + method)

	validateAndTrimArgs := func(args [][]byte) ([][]byte, error) {
		nonNilCount := -1
		for i, arg := range args {
			if len(arg) == 0 {
				nonNilCount = i
				break
			}
		}

		if nonNilCount == -1 {
			return args, nil
		}

		for i := nonNilCount; i < len(args); i++ {
			if len(args[i]) != 0 {
				return nil, fmt.Errorf("non-nil arguments cannot be preceded by nil arguments")
			}
		}

		trimmedArgs := make([][]byte, nonNilCount)
		for i := 0; i < nonNilCount; i++ {
			trimmedArgs[i] = args[i]
		}

		return trimmedArgs, nil
	}

	validatedArgs, err := validateAndTrimArgs(args)
	if err != nil {
		return nil, err
	}

	entryMeta, err := meta.FindStorageEntryMetadata(prefix, method)
	if err != nil {
		return nil, err
	}

	// From metadata >= v14, there is only one representation of Map,
	// which is more alike the old 'NMap': a Map with n keys (n >= 1).
	// The old variants are now unified as thus IsMap() is true for all.
	if entryMeta.IsMap() {
		hashers, err := entryMeta.Hashers()
		if err != nil {
			return nil, fmt.Errorf("unable to get hashers for %s map", method)
		}
		if len(hashers) != len(validatedArgs) {
			return nil, fmt.Errorf("%s:%s is a map, therefore requires that number of arguments should "+
				"exactly match number of hashers in metadata. "+
				"Expected: %d, received: %d", prefix, method, len(hashers), len(validatedArgs))
		}
		return createKeyMap(method, prefix, validatedArgs, entryMeta)
	}

	if entryMeta.IsPlain() && len(validatedArgs) != 0 {
		return nil, fmt.Errorf("%s:%s is a plain key, therefore requires no argument. "+
			"received: %d", prefix, method, len(validatedArgs))
	}

	return createKey(meta, method, prefix, stringKey, nil, entryMeta)
}

// Encode implements encoding for StorageKey, which just unwraps the bytes of StorageKey
func (s StorageKey) Encode(encoder scale.Encoder) error {
	return encoder.Write(s)
}

// Decode implements decoding for StorageKey, which just reads all the remaining bytes into StorageKey
func (s *StorageKey) Decode(decoder scale.Decoder) error {
	for i := 0; true; i++ {
		b, err := decoder.ReadOneByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		*s = append((*s)[:i], b)
	}
	return nil
}

// Hex returns a hex string representation of the value (not of the encoded value)
func (s StorageKey) Hex() string {
	return fmt.Sprintf("%#x", s)
}

// Create a key for a Map.
// The number of keys of the map should match with the number of key hashers.
func createKeyMap(method, prefix string, args [][]byte, entryMeta StorageEntryMetadata) (StorageKey, error) {
	hashers, err := entryMeta.Hashers()
	if err != nil {
		return nil, err
	}

	key := createPrefixedKey(method, prefix)

	for i, arg := range args {
		_, err := hashers[i].Write(arg)
		if err != nil {
			return nil, fmt.Errorf("unable to hash args[%d]: %s Error: %v", i, arg, err)
		}
		key = append(key, hashers[i].Sum(nil)...)
	}

	return key, nil
}

// createKey creates a key for a plain value
func createKey(
	meta *Metadata,
	method,
	prefix string,
	stringKey,
	arg []byte,
	entryMeta StorageEntryMetadata,
) (StorageKey, error) {
	hasher, err := entryMeta.Hasher()
	if err != nil {
		return nil, err
	}

	if meta.Version <= 8 {
		_, err := hasher.Write(append(stringKey, arg...))
		return hasher.Sum(nil), err
	}

	return append(createPrefixedKey(method, prefix), arg...), nil
}

func createPrefixedKey(method, prefix string) []byte {
	return append(xxhash.New128([]byte(prefix)).Sum(nil), xxhash.New128([]byte(method)).Sum(nil)...)
}
