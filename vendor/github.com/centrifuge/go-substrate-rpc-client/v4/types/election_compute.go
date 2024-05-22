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

type OptionElectionCompute struct {
	option
	value ElectionCompute
}

func NewOptionElectionCompute(value ElectionCompute) OptionElectionCompute {
	return OptionElectionCompute{option{hasValue: true}, value}
}

func NewOptionElectionComputeEmpty() OptionElectionCompute {
	return OptionElectionCompute{option: option{hasValue: false}}
}

func (o OptionElectionCompute) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionElectionCompute) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionElectionCompute) SetSome(value ElectionCompute) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionElectionCompute) SetNone() {
	o.hasValue = false
	o.value = ElectionCompute(byte(0))
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o *OptionElectionCompute) Unwrap() (ok bool, value ElectionCompute) {
	return o.hasValue, o.value
}

type ElectionCompute byte

func NewElectionCompute(b byte) ElectionCompute {
	return ElectionCompute(b)
}

const (
	// OnChain means that the result was forcefully computed on chain at the end of the session.
	OnChain ElectionCompute = 0
	// Signed means that the result was submitted and accepted to the chain via a signed transaction.
	Signed ElectionCompute = 1
	// Unsigned means that the result was submitted and accepted to the chain via
	// an unsigned transaction (by an authority).
	Unsigned ElectionCompute = 2
)

func (ec *ElectionCompute) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	vb := ElectionCompute(b)
	switch vb {
	case OnChain, Signed, Unsigned:
		*ec = vb
	default:
		return fmt.Errorf("unknown ElectionCompute enum: %v", vb)
	}
	return err
}

func (ec ElectionCompute) Encode(encoder scale.Encoder) error {
	return encoder.PushByte(byte(ec))
}
