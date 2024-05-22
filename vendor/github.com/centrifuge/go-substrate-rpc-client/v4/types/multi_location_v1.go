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

type OptionMultiLocationV1 struct {
	option
	value MultiLocationV1
}

func NewOptionMultiLocationV1(value MultiLocationV1) OptionMultiLocationV1 {
	return OptionMultiLocationV1{option{hasValue: true}, value}
}

func NewOptionMultiLocationV1Empty() OptionMultiLocationV1 {
	return OptionMultiLocationV1{option: option{hasValue: false}}
}

func (o *OptionMultiLocationV1) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

func (o OptionMultiLocationV1) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

// SetSome sets a value
func (o *OptionMultiLocationV1) SetSome(value MultiLocationV1) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionMultiLocationV1) SetNone() {
	o.hasValue = false
	o.value = MultiLocationV1{}
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o *OptionMultiLocationV1) Unwrap() (ok bool, value MultiLocationV1) {
	return o.hasValue, o.value
}

type MultiLocationV1 struct {
	Parents  U8
	Interior JunctionsV1
}

func (m *MultiLocationV1) Decode(decoder scale.Decoder) error {
	if err := decoder.Decode(&m.Parents); err != nil {
		return err
	}

	return decoder.Decode(&m.Interior)
}

func (m *MultiLocationV1) Encode(encoder scale.Encoder) error {
	if err := encoder.Encode(&m.Parents); err != nil {
		return err
	}

	return encoder.Encode(&m.Interior)
}

type VersionedMultiLocation struct {
	IsV0            bool
	MultiLocationV0 MultiLocationV0

	IsV1            bool
	MultiLocationV1 MultiLocationV1
}

func (m *VersionedMultiLocation) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		m.IsV0 = true

		return decoder.Decode(&m.MultiLocationV0)
	case 1:
		m.IsV1 = true

		return decoder.Decode(&m.MultiLocationV1)
	}

	return nil
}

func (m VersionedMultiLocation) Encode(encoder scale.Encoder) error {
	switch {
	case m.IsV0:
		if err := encoder.PushByte(0); err != nil {
			return err
		}

		return encoder.Encode(m.MultiLocationV0)
	case m.IsV1:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		return encoder.Encode(m.MultiLocationV1)
	}

	return nil
}
