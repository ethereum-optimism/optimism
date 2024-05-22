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
	"math/big"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
)

// I8 is a signed 8-bit integer
type I8 int8

// NewI8 creates a new I8 type
func NewI8(i int8) I8 {
	return I8(i)
}

// UnmarshalJSON fills i with the JSON encoded byte array given by b
func (i *I8) UnmarshalJSON(b []byte) error {
	var tmp int8
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	*i = I8(tmp)
	return nil
}

// MarshalJSON returns a JSON encoded byte array of i
func (i I8) MarshalJSON() ([]byte, error) {
	return json.Marshal(int8(i))
}

// I16 is a signed 16-bit integer
type I16 int16

// NewI16 creates a new I16 type
func NewI16(i int16) I16 {
	return I16(i)
}

// UnmarshalJSON fills i with the JSON encoded byte array given by b
func (i *I16) UnmarshalJSON(b []byte) error {
	var tmp int16
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	*i = I16(tmp)
	return nil
}

// MarshalJSON returns a JSON encoded byte array of i
func (i I16) MarshalJSON() ([]byte, error) {
	return json.Marshal(int16(i))
}

// I32 is a signed 32-bit integer
type I32 int32

// NewI32 creates a new I32 type
func NewI32(i int32) I32 {
	return I32(i)
}

// UnmarshalJSON fills i with the JSON encoded byte array given by b
func (i *I32) UnmarshalJSON(b []byte) error {
	var tmp int32
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	*i = I32(tmp)
	return nil
}

// MarshalJSON returns a JSON encoded byte array of i
func (i I32) MarshalJSON() ([]byte, error) {
	return json.Marshal(int32(i))
}

// I64 is a signed 64-bit integer
type I64 int64

// NewI64 creates a new I64 type
func NewI64(i int64) I64 {
	return I64(i)
}

// UnmarshalJSON fills i with the JSON encoded byte array given by b
func (i *I64) UnmarshalJSON(b []byte) error {
	var tmp int64
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	*i = I64(tmp)
	return nil
}

// MarshalJSON returns a JSON encoded byte array of i
func (i I64) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(i))
}

// I128 is a signed 128-bit integer, it is represented as a big.Int in Go.
type I128 struct {
	*big.Int
}

// NewI128 creates a new I128 type
func NewI128(i big.Int) I128 {
	return I128{&i}
}

// Decode implements decoding as per the Scale specification
func (i *I128) Decode(decoder scale.Decoder) error {
	bs := make([]byte, 16)
	err := decoder.Read(bs)
	if err != nil {
		return err
	}
	// reverse bytes, scale uses little-endian encoding, big.int's bytes are expected in big-endian
	scale.Reverse(bs)

	b, err := IntBytesToBigInt(bs)
	if err != nil {
		return err
	}

	// deal with zero differently to get a nil representation (this is how big.Int deals with 0)
	if b.Sign() == 0 {
		*i = I128{big.NewInt(0)}
		return nil
	}

	*i = I128{b}
	return nil
}

// Encode implements encoding as per the Scale specification
func (i I128) Encode(encoder scale.Encoder) error {
	b, err := BigIntToIntBytes(i.Int, 16)
	if err != nil {
		return err
	}

	// reverse bytes, scale uses little-endian encoding, big.int's bytes are expected in big-endian
	scale.Reverse(b)

	return encoder.Write(b)
}

// I256 is a signed 256-bit integer, it is represented as a big.Int in Go.
type I256 struct {
	*big.Int
}

// NewI256 creates a new I256 type
func NewI256(i big.Int) I256 {
	return I256{&i}
}

// Decode implements decoding as per the Scale specification
func (i *I256) Decode(decoder scale.Decoder) error {
	bs := make([]byte, 32)
	err := decoder.Read(bs)
	if err != nil {
		return err
	}
	// reverse bytes, scale uses little-endian encoding, big.int's bytes are expected in big-endian
	scale.Reverse(bs)

	b, err := IntBytesToBigInt(bs)
	if err != nil {
		return err
	}

	// deal with zero differently to get a nil representation (this is how big.Int deals with 0)
	if b.Sign() == 0 {
		*i = I256{big.NewInt(0)}
		return nil
	}

	*i = I256{b}
	return nil
}

// Encode implements encoding as per the Scale specification
func (i I256) Encode(encoder scale.Encoder) error {
	b, err := BigIntToIntBytes(i.Int, 32)
	if err != nil {
		return err
	}

	// reverse bytes, scale uses little-endian encoding, big.int's bytes are expected in big-endian
	scale.Reverse(b)

	return encoder.Write(b)
}

// BigIntToIntBytes encodes the given big.Int to a big endian encoded signed integer byte slice of the given byte
// length, using a two's complement if the big.Int is negative and returning an error if the given big.Int would be
// bigger than the maximum positive (negative) numbers the byte slice of the given length could hold
func BigIntToIntBytes(i *big.Int, bytelen int) ([]byte, error) {
	res := make([]byte, bytelen)

	maxNeg := big.NewInt(0).Exp(big.NewInt(2), big.NewInt(int64(bytelen*8-1)), nil)
	maxPos := big.NewInt(0).Sub(maxNeg, big.NewInt(1))

	if i.Sign() >= 0 {
		if i.CmpAbs(maxPos) > 0 {
			return nil, fmt.Errorf("cannot encode big.Int to []byte: given big.Int exceeds highest positive number "+
				"%v for an int with %v bits", maxPos, bytelen*8)
		}

		bs := i.Bytes()
		copy(res[len(res)-len(bs):], bs)
		return res, nil
	}

	// negative, two's complement

	if i.CmpAbs(maxNeg) > 0 {
		return nil, fmt.Errorf("cannot encode big.Int to []byte: given big.Int exceeds highest negative number -"+
			"%v for an int with %v bits", maxNeg, bytelen*8)
	}

	i = big.NewInt(0).Add(i, big.NewInt(1))
	bs := i.Bytes()
	copy(res[len(res)-len(bs):], bs)

	// apply not to every byte
	for j, b := range res {
		res[j] = ^b
	}

	return res, nil
}

// IntBytesToBigInt decodes the given byte slice containing a big endian encoded signed integer to a big.Int, using a
// two's complement if the most significant bit is 1
func IntBytesToBigInt(b []byte) (*big.Int, error) {
	if len(b) == 0 {
		return nil, fmt.Errorf("cannot decode an empty byte slice")
	}

	if b[0]&0x80 == 0x00 {
		// positive
		return big.NewInt(0).SetBytes(b), nil
	}

	// negative, two's complement
	t := make([]byte, len(b))
	copy(t, b)

	// apply not to every byte
	for j, b := range b {
		t[j] = ^b
	}

	res := big.NewInt(0).SetBytes(t)
	res = res.Add(res, big.NewInt(1))
	res = res.Neg(res)

	return res, nil
}
