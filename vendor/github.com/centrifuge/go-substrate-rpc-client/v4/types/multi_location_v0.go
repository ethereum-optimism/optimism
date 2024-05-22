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

type MultiLocationV0 struct {
	IsNull bool

	IsX1 bool
	X1   JunctionV0

	IsX2 bool
	X2   [2]JunctionV0

	IsX3 bool
	X3   [3]JunctionV0

	IsX4 bool
	X4   [4]JunctionV0

	IsX5 bool
	X5   [5]JunctionV0

	IsX6 bool
	X6   [6]JunctionV0

	IsX7 bool
	X7   [7]JunctionV0

	IsX8 bool
	X8   [8]JunctionV0
}

func (m *MultiLocationV0) Decode(decoder scale.Decoder) error { //nolint:dupl
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		m.IsNull = true
	case 1:
		m.IsX1 = true

		return decoder.Decode(&m.X1)
	case 2:
		m.IsX2 = true

		return decoder.Decode(&m.X2)
	case 3:
		m.IsX3 = true

		return decoder.Decode(&m.X3)
	case 4:
		m.IsX4 = true

		return decoder.Decode(&m.X4)
	case 5:
		m.IsX5 = true

		return decoder.Decode(&m.X5)
	case 6:
		m.IsX6 = true

		return decoder.Decode(&m.X6)
	case 7:
		m.IsX7 = true

		return decoder.Decode(&m.X7)
	case 8:
		m.IsX8 = true

		return decoder.Decode(&m.X8)
	}

	return nil
}

func (m MultiLocationV0) Encode(encoder scale.Encoder) error {
	switch {
	case m.IsNull:
		return encoder.PushByte(0)
	case m.IsX1:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		return encoder.Encode(m.X1)
	case m.IsX2:
		if err := encoder.PushByte(2); err != nil {
			return err
		}

		return encoder.Encode(m.X2)
	case m.IsX3:
		if err := encoder.PushByte(3); err != nil {
			return err
		}

		return encoder.Encode(m.X3)
	case m.IsX4:
		if err := encoder.PushByte(4); err != nil {
			return err
		}

		return encoder.Encode(m.X4)
	case m.IsX5:
		if err := encoder.PushByte(5); err != nil {
			return err
		}

		return encoder.Encode(m.X5)
	case m.IsX6:
		if err := encoder.PushByte(6); err != nil {
			return err
		}

		return encoder.Encode(m.X6)
	case m.IsX7:
		if err := encoder.PushByte(7); err != nil {
			return err
		}

		return encoder.Encode(m.X7)
	case m.IsX8:
		if err := encoder.PushByte(8); err != nil {
			return err
		}

		return encoder.Encode(m.X8)
	}

	return nil
}
