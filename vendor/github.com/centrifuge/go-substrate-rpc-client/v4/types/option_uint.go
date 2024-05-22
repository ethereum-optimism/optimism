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
	"math/big"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
)

// OptionU8 is a structure that can store a U8 or a missing value
type OptionU8 struct {
	option
	value U8
}

// NewOptionU8 creates an OptionU8 with a value
func NewOptionU8(value U8) OptionU8 {
	return OptionU8{option{true}, value}
}

// NewOptionU8Empty creates an OptionU8 without a value
func NewOptionU8Empty() OptionU8 {
	return OptionU8{option: option{false}}
}

func (o OptionU8) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionU8) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionU8) SetSome(value U8) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionU8) SetNone() {
	o.hasValue = false
	o.value = U8(0)
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionU8) Unwrap() (ok bool, value U8) {
	return o.hasValue, o.value
}

// OptionU16 is a structure that can store a U16 or a missing value
type OptionU16 struct {
	option
	value U16
}

// NewOptionU16 creates an OptionU16 with a value
func NewOptionU16(value U16) OptionU16 {
	return OptionU16{option{true}, value}
}

// NewOptionU16Empty creates an OptionU16 without a value
func NewOptionU16Empty() OptionU16 {
	return OptionU16{option: option{false}}
}

func (o OptionU16) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionU16) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionU16) SetSome(value U16) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionU16) SetNone() {
	o.hasValue = false
	o.value = U16(0)
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionU16) Unwrap() (ok bool, value U16) {
	return o.hasValue, o.value
}

// OptionU32 is a structure that can store a U32 or a missing value
type OptionU32 struct {
	option
	value U32
}

// NewOptionU32 creates an OptionU32 with a value
func NewOptionU32(value U32) OptionU32 {
	return OptionU32{option{true}, value}
}

// NewOptionU32Empty creates an OptionU32 without a value
func NewOptionU32Empty() OptionU32 {
	return OptionU32{option: option{false}}
}

func (o OptionU32) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionU32) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionU32) SetSome(value U32) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionU32) SetNone() {
	o.hasValue = false
	o.value = U32(0)
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionU32) Unwrap() (ok bool, value U32) {
	return o.hasValue, o.value
}

// OptionU64 is a structure that can store a U64 or a missing value
type OptionU64 struct {
	option
	value U64
}

// NewOptionU64 creates an OptionU64 with a value
func NewOptionU64(value U64) OptionU64 {
	return OptionU64{option{true}, value}
}

// NewOptionU64Empty creates an OptionU64 without a value
func NewOptionU64Empty() OptionU64 {
	return OptionU64{option: option{false}}
}

func (o OptionU64) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionU64) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionU64) SetSome(value U64) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionU64) SetNone() {
	o.hasValue = false
	o.value = U64(0)
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionU64) Unwrap() (ok bool, value U64) {
	return o.hasValue, o.value
}

// OptionU128 is a structure that can store a U128 or a missing value
type OptionU128 struct {
	option
	value U128
}

// NewOptionU128 creates an OptionU128 with a value
func NewOptionU128(value U128) OptionU128 {
	return OptionU128{option{true}, value}
}

// NewOptionU128Empty creates an OptionU128 without a value
func NewOptionU128Empty() OptionU128 {
	return OptionU128{option: option{false}}
}

func (o OptionU128) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionU128) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionU128) SetSome(value U128) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionU128) SetNone() {
	o.hasValue = false
	o.value = NewU128(*big.NewInt(0))
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionU128) Unwrap() (ok bool, value U128) {
	return o.hasValue, o.value
}
