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

package hash

import (
	"hash"

	"golang.org/x/crypto/blake2b"
)

type blake2b128Concat struct {
	hasher hash.Hash
	data   []byte
}

// NewBlake2b128Concat returns an instance of blake2b concat hasher
func NewBlake2b128Concat(k []byte) (hash.Hash, error) {
	h, err := blake2b.New(16, k)
	if err != nil {
		return nil, err
	}
	return &blake2b128Concat{hasher: h, data: k}, nil
}

// Write (via the embedded io.Writer interface) adds more data to the running hash.
func (bc *blake2b128Concat) Write(p []byte) (n int, err error) {
	bc.data = append(bc.data, p...)
	return bc.hasher.Write(p)
}

// Sum appends the current hash to b and returns the resulting slice.
// It does not change the underlying hash state.
func (bc *blake2b128Concat) Sum(b []byte) []byte {
	return append(bc.hasher.Sum(b), bc.data...)
}

// Reset resets the Hash to its initial state.
func (bc *blake2b128Concat) Reset() {
	bc.data = nil
	bc.hasher.Reset()
}

// Size returns the number of bytes Sum will return.
func (bc *blake2b128Concat) Size() int {
	return len(bc.Sum(nil))
}

// BlockSize returns the hash's underlying block size.
// The Write method must be able to accept any amount
// of data, but it may operate more efficiently if all writes
// are a multiple of the block size.
func (bc *blake2b128Concat) BlockSize() int {
	return bc.hasher.BlockSize()
}

// NewBlake2b128 returns blake2b-128 hasher
func NewBlake2b128(k []byte) (hash.Hash, error) {
	return blake2b.New(16, k)
}

// NewBlake2b256 returns blake2b-256 hasher
func NewBlake2b256(k []byte) (hash.Hash, error) {
	return blake2b.New256(k)
}

// NewBlake2b512 returns blake2b-512 hasher
func NewBlake2b512(k []byte) (hash.Hash, error) {
	return blake2b.New512(k)
}
