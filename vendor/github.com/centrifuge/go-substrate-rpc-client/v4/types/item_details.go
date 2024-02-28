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

type ItemDetails struct {
	Owner    AccountID
	Approved OptionAccountID
	IsFrozen bool
	Deposit  U128
}

func (i *ItemDetails) Decode(decoder scale.Decoder) error {
	if err := decoder.Decode(&i.Owner); err != nil {
		return err
	}
	if err := decoder.Decode(&i.Approved); err != nil {
		return err
	}
	if err := decoder.Decode(&i.IsFrozen); err != nil {
		return err
	}

	return decoder.Decode(&i.Deposit)
}

func (i ItemDetails) Encode(encoder scale.Encoder) error {
	if err := encoder.Encode(i.Owner); err != nil {
		return err
	}
	if err := encoder.Encode(i.Approved); err != nil {
		return err
	}
	if err := encoder.Encode(i.IsFrozen); err != nil {
		return err
	}

	return encoder.Encode(i.Deposit)
}
