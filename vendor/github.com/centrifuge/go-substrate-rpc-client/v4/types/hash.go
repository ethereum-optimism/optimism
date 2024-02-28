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
	"encoding/json"
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
)

// H160 is a hash containing 160 bits (20 bytes), typically used in blocks, extrinsics and as a sane default
type H160 [20]byte

// NewH160 creates a new H160 type
func NewH160(b []byte) H160 {
	h := H160{}
	copy(h[:], b)
	return h
}

// Hex returns a hex string representation of the value (not of the encoded value)
func (h H160) Hex() string {
	return fmt.Sprintf("%#x", h[:])
}

// H256 is a hash containing 256 bits (32 bytes), typically used in blocks, extrinsics and as a sane default
type H256 [32]byte

// NewH256 creates a new H256 type
func NewH256(b []byte) H256 {
	h := H256{}
	copy(h[:], b)
	return h
}

// Hex returns a hex string representation of the value (not of the encoded value)
func (h H256) Hex() string {
	return fmt.Sprintf("%#x", h[:])
}

// H512 is a hash containing 512 bits (64 bytes), typically used for signature
type H512 [64]byte

// NewH512 creates a new H512 type
func NewH512(b []byte) H512 {
	h := H512{}
	copy(h[:], b)
	return h
}

// Hex returns a hex string representation of the value (not of the encoded value)
func (h H512) Hex() string {
	return fmt.Sprintf("%#x", h[:])
}

// Hash is the default hash that is used across the system. It is just a thin wrapper around H256
type Hash H256

// NewHash creates a new Hash type
func NewHash(b []byte) Hash {
	h := Hash{}
	copy(h[:], b)
	return h
}

// NewHashFromHexString creates a new Hash type from a hex string
func NewHashFromHexString(s string) (Hash, error) {
	bz, err := codec.HexDecodeString(s)
	if err != nil {
		return Hash{}, err
	}

	if len(bz) != 32 {
		return Hash{}, fmt.Errorf("required result to be 32 bytes, but got %v", len(bz))
	}

	return NewHash(bz), nil
}

// Hex returns a hex string representation of the value (not of the encoded value)
func (h Hash) Hex() string {
	return fmt.Sprintf("%#x", h[:])
}

// UnmarshalJSON fills h with the JSON encoded byte array given by b
func (h *Hash) UnmarshalJSON(b []byte) error {
	var tmp string
	err := json.Unmarshal(b, &tmp)
	if err != nil {
		return err
	}
	*h, err = NewHashFromHexString(tmp)
	return err
}

// MarshalJSON returns a JSON encoded byte array of h
func (h Hash) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.Hex())
}
