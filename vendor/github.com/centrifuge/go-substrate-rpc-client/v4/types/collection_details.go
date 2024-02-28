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

type CollectionDetails struct {
	Owner             AccountID
	Issuer            AccountID
	Admin             AccountID
	Freezer           AccountID
	TotalDeposit      U128
	FreeHolding       bool
	Instances         U32
	InstanceMetadatas U32
	Attributes        U32
	IsFrozen          bool
}

func (c *CollectionDetails) Decode(decoder scale.Decoder) error {
	if err := decoder.Decode(&c.Owner); err != nil {
		return err
	}

	if err := decoder.Decode(&c.Issuer); err != nil {
		return err
	}
	if err := decoder.Decode(&c.Admin); err != nil {
		return err
	}
	if err := decoder.Decode(&c.Freezer); err != nil {
		return err
	}
	if err := decoder.Decode(&c.TotalDeposit); err != nil {
		return err
	}
	if err := decoder.Decode(&c.FreeHolding); err != nil {
		return err
	}
	if err := decoder.Decode(&c.Instances); err != nil {
		return err
	}
	if err := decoder.Decode(&c.InstanceMetadatas); err != nil {
		return err
	}
	if err := decoder.Decode(&c.Attributes); err != nil {
		return err
	}

	return decoder.Decode(&c.IsFrozen)
}

func (c CollectionDetails) Encode(encoder scale.Encoder) error {
	if err := encoder.Encode(c.Owner); err != nil {
		return err
	}

	if err := encoder.Encode(c.Issuer); err != nil {
		return err
	}

	if err := encoder.Encode(c.Admin); err != nil {
		return err
	}

	if err := encoder.Encode(c.Freezer); err != nil {
		return err
	}

	if err := encoder.Encode(c.TotalDeposit); err != nil {
		return err
	}

	if err := encoder.Encode(c.FreeHolding); err != nil {
		return err
	}

	if err := encoder.Encode(c.Instances); err != nil {
		return err
	}

	if err := encoder.Encode(c.InstanceMetadatas); err != nil {
		return err
	}

	if err := encoder.Encode(c.Attributes); err != nil {
		return err
	}

	return encoder.Encode(c.IsFrozen)
}
