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

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
)

// Bytes represents byte slices. Bytes has a variable length, it is encoded with a scale prefix
type Bytes []byte

// NewBytes creates a new Bytes type
func NewBytes(b []byte) Bytes {
	return Bytes(b)
}

// BytesBare represents byte slices that will be encoded bare, i. e. without a compact length prefix. This makes it
// impossible to decode the bytes, but is used as the payload for signing.
type BytesBare []byte

// Encode implements encoding for BytesBare, which just unwraps the bytes of BytesBare without adding a compact
// length prefix
func (b BytesBare) Encode(encoder scale.Encoder) error {
	return encoder.Write(b)
}

// Decode does nothing and always returns an error. BytesBare is only used for encoding, not for decoding
func (b *BytesBare) Decode(decoder scale.Decoder) error {
	return fmt.Errorf("decoding of BytesBare is not supported")
}

// Bytes8 represents an 8 byte array
type Bytes8 [8]byte

// NewBytes8 creates a new Bytes8 type
func NewBytes8(b [8]byte) Bytes8 {
	return Bytes8(b)
}

// Bytes16 represents an 16 byte array
type Bytes16 [16]byte

// NewBytes16 creates a new Bytes16 type
func NewBytes16(b [16]byte) Bytes16 {
	return Bytes16(b)
}

// Bytes32 represents an 32 byte array
type Bytes32 [32]byte

// NewBytes32 creates a new Bytes32 type
func NewBytes32(b [32]byte) Bytes32 {
	return Bytes32(b)
}

// Bytes64 represents an 64 byte array
type Bytes64 [64]byte

// NewBytes64 creates a new Bytes64 type
func NewBytes64(b [64]byte) Bytes64 {
	return Bytes64(b)
}

// Bytes128 represents an 128 byte array
type Bytes128 [128]byte

// NewBytes128 creates a new Bytes128 type
func NewBytes128(b [128]byte) Bytes128 {
	return Bytes128(b)
}

// Bytes256 represents an 256 byte array
type Bytes256 [256]byte

// NewBytes256 creates a new Bytes256 type
func NewBytes256(b [256]byte) Bytes256 {
	return Bytes256(b)
}

// Bytes512 represents an 512 byte array
type Bytes512 [512]byte

// NewBytes512 creates a new Bytes512 type
func NewBytes512(b [512]byte) Bytes512 {
	return Bytes512(b)
}

// Bytes1024 represents an 1024 byte array
type Bytes1024 [1024]byte

// NewBytes1024 creates a new Bytes1024 type
func NewBytes1024(b [1024]byte) Bytes1024 {
	return Bytes1024(b)
}

// Bytes2048 represents an 2048 byte array
type Bytes2048 [2048]byte

// NewBytes2048 creates a new Bytes2048 type
func NewBytes2048(b [2048]byte) Bytes2048 {
	return Bytes2048(b)
}
