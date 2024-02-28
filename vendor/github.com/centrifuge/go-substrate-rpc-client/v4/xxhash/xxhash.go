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

package xxhash

import (
	"hash"

	"github.com/pierrec/xxHash/xxHash64"
)

const (
	Typ64 int = iota
	Typ64Concat
	Typ128
	Typ256
)

type state struct {
	data   []byte
	typ    int
	rounds int
}

// New64 returns a new hash.Hash computing the xxhash checksum with 1 iteration
func New64(b []byte) hash.Hash {
	return &state{
		data:   b,
		typ:    Typ64,
		rounds: 1,
	}
}

// New64Concat returns a new hash.Hash computing the xxhash checksum with 1 iteration and appending the data
func New64Concat(b []byte) hash.Hash {
	return &state{
		data:   b,
		typ:    Typ64Concat,
		rounds: 1,
	}
}

// New128 returns a new hash.Hash computing the xxhash checksum with 2 iterations
func New128(b []byte) hash.Hash {
	return &state{
		data:   b,
		typ:    Typ128,
		rounds: 2,
	}
}

// New256 returns a new hash.Hash computing the xxhash checksum with 4 iterations
func New256(b []byte) hash.Hash {
	return &state{
		data:   b,
		typ:    Typ256,
		rounds: 4,
	}
}

// Write (via the embedded io.Writer interface) adds more data to the running hash.
// It never returns an error.
func (s *state) Write(p []byte) (n int, err error) {
	s.data = append(s.data, p...)
	return len(p), nil
}

// Sum appends the current hash to b and returns the resulting slice.
// It does not change the underlying hash state.
func (s *state) Sum(b []byte) []byte {
	res := make([]byte, 0, s.rounds*8)

	for i := 0; i < s.rounds; i++ {
		h := xxHash64.New(uint64(i))
		_, err := h.Write(s.data)
		if err != nil {
			panic(err)
		}
		res = append(res, h.Sum(nil)...)
	}

	if s.typ == Typ64Concat {
		res = append(res, s.data...)
	}
	return append(b, res...)
}

// Reset resets the Hash to its initial state.
func (s *state) Reset() {
	s.data = make([]byte, 0)
}

// Size returns the number of bytes Sum will return.
func (s *state) Size() int {
	return len(s.Sum(nil))
}

// BlockSize returns the hash's underlying block size.
// The Write method must be able to accept any amount
// of data, but it may operate more efficiently if all writes
// are a multiple of the block size.
func (s *state) BlockSize() int {
	return 64
}
