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
	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
)

// ExtrinsicEra indicates either a mortal or immortal extrinsic
type ExtrinsicEra struct {
	IsImmortalEra bool
	// AsImmortalEra ImmortalEra
	IsMortalEra bool
	AsMortalEra MortalEra
}

func (e *ExtrinsicEra) Decode(decoder scale.Decoder) error {
	first, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	if first == 0 {
		e.IsImmortalEra = true
		return nil
	}

	e.IsMortalEra = true

	second, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	e.AsMortalEra = MortalEra{first, second}

	return nil
}

func (e ExtrinsicEra) Encode(encoder scale.Encoder) error {
	if e.IsImmortalEra {
		return encoder.PushByte(0)
	}

	err := encoder.PushByte(e.AsMortalEra.First)
	if err != nil {
		return err
	}

	err = encoder.PushByte(e.AsMortalEra.Second)
	if err != nil {
		return err
	}

	return nil
}

// MortalEra for an extrinsic, indicating period and phase
type MortalEra struct {
	First  byte
	Second byte
}
