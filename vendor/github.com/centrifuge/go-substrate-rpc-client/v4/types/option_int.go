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

import "github.com/centrifuge/go-substrate-rpc-client/v4/scale"

// OptionI8 is a structure that can store a I8 or a missing value
type OptionI8 struct {
	option
	value I8
}

// NewOptionI8 creates an OptionI8 with a value
func NewOptionI8(value I8) OptionI8 {
	return OptionI8{option{true}, value}
}

// NewOptionI8Empty creates an OptionI8 without a value
func NewOptionI8Empty() OptionI8 {
	return OptionI8{option: option{false}}
}

func (o OptionI8) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionI8) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionI8) SetSome(value I8) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionI8) SetNone() {
	o.hasValue = false
	o.value = I8(0)
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionI8) Unwrap() (ok bool, value I8) {
	return o.hasValue, o.value
}

// OptionI16 is a structure that can store a I16 or a missing value
type OptionI16 struct {
	option
	value I16
}

// NewOptionI16 creates an OptionI16 with a value
func NewOptionI16(value I16) OptionI16 {
	return OptionI16{option{true}, value}
}

// NewOptionI16Empty creates an OptionI16 without a value
func NewOptionI16Empty() OptionI16 {
	return OptionI16{option: option{false}}
}

func (o OptionI16) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionI16) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionI16) SetSome(value I16) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionI16) SetNone() {
	o.hasValue = false
	o.value = I16(0)
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionI16) Unwrap() (ok bool, value I16) {
	return o.hasValue, o.value
}

// OptionI32 is a structure that can store a I32 or a missing value
type OptionI32 struct {
	option
	value I32
}

// NewOptionI32 creates an OptionI32 with a value
func NewOptionI32(value I32) OptionI32 {
	return OptionI32{option{true}, value}
}

// NewOptionI32Empty creates an OptionI32 without a value
func NewOptionI32Empty() OptionI32 {
	return OptionI32{option: option{false}}
}

func (o OptionI32) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionI32) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionI32) SetSome(value I32) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionI32) SetNone() {
	o.hasValue = false
	o.value = I32(0)
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionI32) Unwrap() (ok bool, value I32) {
	return o.hasValue, o.value
}

// OptionI64 is a structure that can store a I64 or a missing value
type OptionI64 struct {
	option
	value I64
}

// NewOptionI64 creates an OptionI64 with a value
func NewOptionI64(value I64) OptionI64 {
	return OptionI64{option{true}, value}
}

// NewOptionI64Empty creates an OptionI64 without a value
func NewOptionI64Empty() OptionI64 {
	return OptionI64{option: option{false}}
}

func (o OptionI64) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionI64) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionI64) SetSome(value I64) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionI64) SetNone() {
	o.hasValue = false
	o.value = I64(0)
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionI64) Unwrap() (ok bool, value I64) {
	return o.hasValue, o.value
}
