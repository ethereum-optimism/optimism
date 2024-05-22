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

type NetworkID struct {
	IsAny bool

	IsNamed      bool
	NamedNetwork []U8

	IsPolkadot bool

	IsKusama bool
}

func (n *NetworkID) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		n.IsAny = true
	case 1:
		n.IsNamed = true

		return decoder.Decode(&n.NamedNetwork)
	case 2:
		n.IsPolkadot = true
	case 3:
		n.IsKusama = true
	}

	return nil
}

func (n NetworkID) Encode(encoder scale.Encoder) error {
	switch {
	case n.IsAny:
		return encoder.PushByte(0)
	case n.IsNamed:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		return encoder.Encode(n.NamedNetwork)
	case n.IsPolkadot:
		return encoder.PushByte(2)
	case n.IsKusama:
		return encoder.PushByte(3)
	}

	return nil
}
