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

type Outcome struct {
	IsComplete     bool
	CompleteWeight Weight

	IsIncomplete     bool
	IncompleteWeight Weight
	IncompleteError  XCMError

	IsError bool
	Error   XCMError
}

func (o *Outcome) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		o.IsComplete = true

		if err = decoder.Decode(&o.CompleteWeight); err != nil {
			return err
		}
	case 1:
		o.IsIncomplete = true

		if err = decoder.Decode(&o.IncompleteWeight); err != nil {
			return err
		}

		if err = decoder.Decode(&o.IncompleteError); err != nil {
			return err
		}
	case 2:
		o.IsError = true

		if err = decoder.Decode(&o.Error); err != nil {
			return err
		}
	}

	return nil
}

func (o Outcome) Encode(encoder scale.Encoder) error {
	switch {
	case o.IsComplete:
		if err := encoder.PushByte(0); err != nil {
			return err
		}

		return encoder.Encode(o.CompleteWeight)
	case o.IsIncomplete:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		if err := encoder.Encode(o.IncompleteWeight); err != nil {
			return err
		}

		return encoder.Encode(o.IncompleteError)
	case o.IsError:
		if err := encoder.PushByte(2); err != nil {
			return err
		}

		return encoder.Encode(o.Error)
	}

	return nil
}
