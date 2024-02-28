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

// OptionBool is a structure that can store a Bool or a missing value
// Note that encoding rules are slightly different from other "option" fields
// This implementation was adopted from https://github.com/Joystream/parity-codec-go/blob/develop/noreflect/codec.go
type OptionBool struct {
	option
	value Bool
}

// NewOptionBool creates an OptionBool with a value
func NewOptionBool(value Bool) OptionBool {
	return OptionBool{option{true}, value}
}

// NewOptionBoolEmpty creates an OptionBool without a value
func NewOptionBoolEmpty() OptionBool {
	return OptionBool{option{false}, false}
}

// Encode implements encoding for OptionBool as per Rust implementation
func (o OptionBool) Encode(encoder scale.Encoder) error {
	var err error
	if !o.hasValue {
		err = encoder.PushByte(0)
	} else {
		if o.value {
			err = encoder.PushByte(1)
		} else {
			err = encoder.PushByte(2)
		}
	}
	if err != nil {
		return err
	}
	return nil
}

// Decode implements decoding for OptionBool as per Rust implementation
func (o *OptionBool) Decode(decoder scale.Decoder) error {
	b, _ := decoder.ReadOneByte()
	switch b {
	case 0:
		o.hasValue = false
		o.value = false
	case 1:
		o.hasValue = true
		o.value = true
	case 2:
		o.hasValue = true
		o.value = false
	default:
		return fmt.Errorf("unknown byte prefix for encoded OptionBool: %d", b)
	}
	return nil
}

// SetSome sets a value
func (o *OptionBool) SetSome(value Bool) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionBool) SetNone() {
	o.hasValue = false
	o.value = Bool(false)
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionBool) Unwrap() (ok bool, value Bool) {
	return o.hasValue, o.value
}
