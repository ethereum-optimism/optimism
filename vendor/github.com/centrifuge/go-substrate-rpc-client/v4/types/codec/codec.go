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

package codec

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
)

// Hexer interface is implemented by any type that has a Hex() function returning a string
type Hexer interface {
	Hex() string
}

// Encode encodes `value` with the scale codec with passed EncoderOptions, returning []byte
func Encode(value interface{}) ([]byte, error) {
	var buffer = bytes.Buffer{}
	err := scale.NewEncoder(&buffer).Encode(value)
	if err != nil {
		return buffer.Bytes(), err
	}
	return buffer.Bytes(), nil
}

// EncodeToHex encodes `value` with the scale codec, returning a hex string (prefixed by 0x)
func EncodeToHex(value interface{}) (string, error) {
	bz, err := Encode(value)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%#x", bz), nil
}

// Decode decodes `bz` with the scale codec into `target`. `target` should be a pointer.
func Decode(bz []byte, target interface{}) error {
	return scale.NewDecoder(bytes.NewReader(bz)).Decode(target)
}

// DecodeFromHex decodes `str` with the scale codec into `target`. `target` should be a pointer.
func DecodeFromHex(str string, target interface{}) error {
	bz, err := HexDecodeString(str)
	if err != nil {
		return err
	}
	return Decode(bz, target)
}

// EncodedLength returns the length of the value when encoded as a byte array
func EncodedLength(value interface{}) (int, error) {
	var buffer = bytes.Buffer{}
	err := scale.NewEncoder(&buffer).Encode(value)
	if err != nil {
		return 0, err
	}
	return buffer.Len(), nil
}

// Eq compares the value of the input to see if there is a match
func Eq(one, other interface{}) bool {
	return reflect.DeepEqual(one, other)
}

// HexDecodeString decodes bytes from a hex string. Contrary to hex.DecodeString, this function does not error if "0x"
// is prefixed, and adds an extra 0 if the hex string has an odd length.
func HexDecodeString(s string) ([]byte, error) {
	s = strings.TrimPrefix(s, "0x")

	if len(s)%2 != 0 {
		s = "0" + s
	}

	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// MustHexDecodeString panics if str cannot be decoded
func MustHexDecodeString(str string) []byte {
	bz, err := HexDecodeString(str)
	if err != nil {
		panic(err)
	}
	return bz
}

// HexEncodeToString encodes bytes to a hex string. Contrary to hex.EncodeToString,
// this function prefixes the hex string with "0x"
func HexEncodeToString(b []byte) string {
	return "0x" + hex.EncodeToString(b)
}

// Hex returns a hex string representation of the value (not of the encoded value)
func Hex(value interface{}) (string, error) {
	switch v := value.(type) {
	case Hexer:
		return v.Hex(), nil
	case []byte:
		return fmt.Sprintf("%#x", v), nil
	default:
		return "", fmt.Errorf("does not support %T", v)
	}
}
