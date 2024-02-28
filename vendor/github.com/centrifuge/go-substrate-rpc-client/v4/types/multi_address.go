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
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
)

type MultiAddress struct {
	IsID        bool
	AsID        AccountID
	IsIndex     bool
	AsIndex     AccountIndex
	IsRaw       bool
	AsRaw       []byte
	IsAddress32 bool
	AsAddress32 [32]byte
	IsAddress20 bool
	AsAddress20 [20]byte
}

// NewMultiAddressFromAccountID creates an Address from the given AccountID (public key)
func NewMultiAddressFromAccountID(b []byte) (MultiAddress, error) {
	accountID, err := NewAccountID(b)
	if err != nil {
		return MultiAddress{}, err
	}

	return MultiAddress{
		IsID: true,
		AsID: *accountID,
	}, nil
}

// NewMultiAddressFromHexAccountID creates an Address from the given hex string that contains an AccountID (public key)
func NewMultiAddressFromHexAccountID(str string) (MultiAddress, error) {
	b, err := codec.HexDecodeString(str)
	if err != nil {
		return MultiAddress{}, err
	}
	return NewMultiAddressFromAccountID(b)
}

func (m MultiAddress) Encode(encoder scale.Encoder) error {
	var err error
	switch {
	case m.IsID:
		if err = encoder.PushByte(0); err != nil {
			return err
		}

		return encoder.Encode(m.AsID)
	case m.IsIndex:
		if err = encoder.PushByte(1); err != nil {
			return err
		}

		return encoder.Encode(m.AsIndex)
	case m.IsRaw:
		if err = encoder.PushByte(2); err != nil {
			return err
		}

		return encoder.Encode(m.AsRaw)
	case m.IsAddress32:
		if err = encoder.PushByte(3); err != nil {
			return err
		}

		return encoder.Encode(m.AsAddress32)
	case m.IsAddress20:
		if err = encoder.PushByte(4); err != nil {
			return err
		}

		return encoder.Encode(m.AsAddress20)
	}

	return nil
}

func (m *MultiAddress) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		m.IsID = true

		return decoder.Decode(&m.AsID)
	case 1:
		m.IsIndex = true

		return decoder.Decode(&m.AsIndex)
	case 2:
		m.IsRaw = true

		return decoder.Decode(&m.AsRaw)
	case 3:
		m.IsAddress32 = true

		return decoder.Decode(&m.AsAddress32)
	case 4:
		m.IsAddress20 = true

		return decoder.Decode(&m.AsAddress20)
	}

	return nil
}
