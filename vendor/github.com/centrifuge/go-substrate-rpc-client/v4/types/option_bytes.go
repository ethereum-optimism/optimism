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

// OptionBytes is a structure that can store a Bytes or a missing value
type OptionBytes struct {
	option
	value Bytes
}

// NewOptionBytes creates an OptionBytes with a value
func NewOptionBytes(value Bytes) OptionBytes {
	return OptionBytes{option{true}, value}
}

// NewOptionBytesEmpty creates an OptionBytes without a value
func NewOptionBytesEmpty() OptionBytes {
	return OptionBytes{option: option{false}}
}

func (o OptionBytes) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionBytes) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionBytes) SetSome(value Bytes) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionBytes) SetNone() {
	o.hasValue = false
	o.value = Bytes{}
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionBytes) Unwrap() (ok bool, value Bytes) {
	return o.hasValue, o.value
}

// OptionBytes8 is a structure that can store a Bytes8 or a missing value
type OptionBytes8 struct {
	option
	value Bytes8
}

// NewOptionBytes8 creates an OptionBytes8 with a value
func NewOptionBytes8(value Bytes8) OptionBytes8 {
	return OptionBytes8{option{true}, value}
}

// NewOptionBytes8Empty creates an OptionBytes8 without a value
func NewOptionBytes8Empty() OptionBytes8 {
	return OptionBytes8{option: option{false}}
}

func (o OptionBytes8) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionBytes8) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionBytes8) SetSome(value Bytes8) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionBytes8) SetNone() {
	o.hasValue = false
	o.value = Bytes8{}
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionBytes8) Unwrap() (ok bool, value Bytes8) {
	return o.hasValue, o.value
}

// OptionBytes16 is a structure that can store a Bytes16 or a missing value
type OptionBytes16 struct {
	option
	value Bytes16
}

// NewOptionBytes16 creates an OptionBytes16 with a value
func NewOptionBytes16(value Bytes16) OptionBytes16 {
	return OptionBytes16{option{true}, value}
}

// NewOptionBytes16Empty creates an OptionBytes16 without a value
func NewOptionBytes16Empty() OptionBytes16 {
	return OptionBytes16{option: option{false}}
}

func (o OptionBytes16) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionBytes16) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionBytes16) SetSome(value Bytes16) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionBytes16) SetNone() {
	o.hasValue = false
	o.value = Bytes16{}
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionBytes16) Unwrap() (ok bool, value Bytes16) {
	return o.hasValue, o.value
}

// OptionBytes32 is a structure that can store a Bytes32 or a missing value
type OptionBytes32 struct {
	option
	value Bytes32
}

// NewOptionBytes32 creates an OptionBytes32 with a value
func NewOptionBytes32(value Bytes32) OptionBytes32 {
	return OptionBytes32{option{true}, value}
}

// NewOptionBytes32Empty creates an OptionBytes32 without a value
func NewOptionBytes32Empty() OptionBytes32 {
	return OptionBytes32{option: option{false}}
}

func (o OptionBytes32) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionBytes32) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionBytes32) SetSome(value Bytes32) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionBytes32) SetNone() {
	o.hasValue = false
	o.value = Bytes32{}
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionBytes32) Unwrap() (ok bool, value Bytes32) {
	return o.hasValue, o.value
}

// OptionBytes64 is a structure that can store a Bytes64 or a missing value
type OptionBytes64 struct {
	option
	value Bytes64
}

// NewOptionBytes64 creates an OptionBytes64 with a value
func NewOptionBytes64(value Bytes64) OptionBytes64 {
	return OptionBytes64{option{true}, value}
}

// NewOptionBytes64Empty creates an OptionBytes64 without a value
func NewOptionBytes64Empty() OptionBytes64 {
	return OptionBytes64{option: option{false}}
}

func (o OptionBytes64) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionBytes64) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionBytes64) SetSome(value Bytes64) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionBytes64) SetNone() {
	o.hasValue = false
	o.value = Bytes64{}
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionBytes64) Unwrap() (ok bool, value Bytes64) {
	return o.hasValue, o.value
}

// OptionBytes128 is a structure that can store a Bytes128 or a missing value
type OptionBytes128 struct {
	option
	value Bytes128
}

// NewOptionBytes128 creates an OptionBytes128 with a value
func NewOptionBytes128(value Bytes128) OptionBytes128 {
	return OptionBytes128{option{true}, value}
}

// NewOptionBytes128Empty creates an OptionBytes128 without a value
func NewOptionBytes128Empty() OptionBytes128 {
	return OptionBytes128{option: option{false}}
}

func (o OptionBytes128) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionBytes128) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionBytes128) SetSome(value Bytes128) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionBytes128) SetNone() {
	o.hasValue = false
	o.value = Bytes128{}
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionBytes128) Unwrap() (ok bool, value Bytes128) {
	return o.hasValue, o.value
}

// OptionBytes256 is a structure that can store a Bytes256 or a missing value
type OptionBytes256 struct {
	option
	value Bytes256
}

// NewOptionBytes256 creates an OptionBytes256 with a value
func NewOptionBytes256(value Bytes256) OptionBytes256 {
	return OptionBytes256{option{true}, value}
}

// NewOptionBytes256Empty creates an OptionBytes256 without a value
func NewOptionBytes256Empty() OptionBytes256 {
	return OptionBytes256{option: option{false}}
}

func (o OptionBytes256) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionBytes256) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionBytes256) SetSome(value Bytes256) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionBytes256) SetNone() {
	o.hasValue = false
	o.value = Bytes256{}
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionBytes256) Unwrap() (ok bool, value Bytes256) {
	return o.hasValue, o.value
}

// OptionBytes512 is a structure that can store a Bytes512 or a missing value
type OptionBytes512 struct {
	option
	value Bytes512
}

// NewOptionBytes512 creates an OptionBytes512 with a value
func NewOptionBytes512(value Bytes512) OptionBytes512 {
	return OptionBytes512{option{true}, value}
}

// NewOptionBytes512Empty creates an OptionBytes512 without a value
func NewOptionBytes512Empty() OptionBytes512 {
	return OptionBytes512{option: option{false}}
}

func (o OptionBytes512) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionBytes512) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionBytes512) SetSome(value Bytes512) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionBytes512) SetNone() {
	o.hasValue = false
	o.value = Bytes512{}
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionBytes512) Unwrap() (ok bool, value Bytes512) {
	return o.hasValue, o.value
}

// OptionBytes1024 is a structure that can store a Bytes1024 or a missing value
type OptionBytes1024 struct {
	option
	value Bytes1024
}

// NewOptionBytes1024 creates an OptionBytes1024 with a value
func NewOptionBytes1024(value Bytes1024) OptionBytes1024 {
	return OptionBytes1024{option{true}, value}
}

// NewOptionBytes1024Empty creates an OptionBytes1024 without a value
func NewOptionBytes1024Empty() OptionBytes1024 {
	return OptionBytes1024{option: option{false}}
}

func (o OptionBytes1024) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionBytes1024) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionBytes1024) SetSome(value Bytes1024) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionBytes1024) SetNone() {
	o.hasValue = false
	o.value = Bytes1024{}
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionBytes1024) Unwrap() (ok bool, value Bytes1024) {
	return o.hasValue, o.value
}

// OptionBytes2048 is a structure that can store a Bytes2048 or a missing value
type OptionBytes2048 struct {
	option
	value Bytes2048
}

// NewOptionBytes2048 creates an OptionBytes2048 with a value
func NewOptionBytes2048(value Bytes2048) OptionBytes2048 {
	return OptionBytes2048{option{true}, value}
}

// NewOptionBytes2048Empty creates an OptionBytes2048 without a value
func NewOptionBytes2048Empty() OptionBytes2048 {
	return OptionBytes2048{option: option{false}}
}

func (o OptionBytes2048) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionBytes2048) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionBytes2048) SetSome(value Bytes2048) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionBytes2048) SetNone() {
	o.hasValue = false
	o.value = Bytes2048{}
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionBytes2048) Unwrap() (ok bool, value Bytes2048) {
	return o.hasValue, o.value
}
