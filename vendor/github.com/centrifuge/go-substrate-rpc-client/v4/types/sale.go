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

type Tranche struct {
	FirstVal  U64
	SecondVal [16]U8
}

func (t *Tranche) Decode(decoder scale.Decoder) error {
	if err := decoder.Decode(&t.FirstVal); err != nil {
		return err
	}

	return decoder.Decode(&t.SecondVal)
}

func (t Tranche) Encode(encoder scale.Encoder) error {
	if err := encoder.Encode(t.FirstVal); err != nil {
		return err
	}

	return encoder.Encode(t.SecondVal)
}

type PermissionedCurrency struct {
	//  At the moment of writing this, this enum is empty in altair.
}

func (p *PermissionedCurrency) Decode(_ scale.Decoder) error {
	return nil
}

func (p *PermissionedCurrency) Encode(_ scale.Encoder) error {
	return nil
}

type CurrencyID struct {
	IsNative bool

	IsUsd bool

	IsTranche bool
	Tranche   Tranche

	IsKSM bool

	IsKUSD bool

	IsPermissioned       bool
	PermissionedCurrency PermissionedCurrency
}

func (c *CurrencyID) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		c.IsNative = true
	case 1:
		c.IsUsd = true
	case 2:
		c.IsTranche = true

		return decoder.Decode(&c.Tranche)
	case 3:
		c.IsKSM = true
	case 4:
		c.IsKUSD = true
	case 5:
		c.IsPermissioned = true

		return decoder.Decode(&c.PermissionedCurrency)
	}

	return nil
}

func (c CurrencyID) Encode(encoder scale.Encoder) error {
	switch {
	case c.IsNative:
		return encoder.PushByte(0)
	case c.IsUsd:
		return encoder.PushByte(1)
	case c.IsTranche:
		if err := encoder.PushByte(2); err != nil {
			return err
		}

		return encoder.Encode(c.Tranche)
	case c.IsKSM:
		return encoder.PushByte(3)
	case c.IsKUSD:
		return encoder.PushByte(4)
	case c.IsPermissioned:
		if err := encoder.PushByte(5); err != nil {
			return err
		}

		return encoder.Encode(c.PermissionedCurrency)
	}

	return nil
}

type Price struct {
	CurrencyID CurrencyID
	Amount     U128
}

func (p *Price) Decode(decoder scale.Decoder) error {
	if err := decoder.Decode(&p.CurrencyID); err != nil {
		return err
	}

	return decoder.Decode(&p.Amount)
}

func (p Price) Encode(encoder scale.Encoder) error {
	if err := encoder.Encode(p.CurrencyID); err != nil {
		return err
	}

	return encoder.Encode(p.Amount)
}

type Sale struct {
	Seller AccountID
	Price  Price
}

func (s *Sale) Decode(decoder scale.Decoder) error {
	if err := decoder.Decode(&s.Seller); err != nil {
		return err
	}

	return decoder.Decode(&s.Price)
}

func (s Sale) Encode(encoder scale.Encoder) error {
	if err := encoder.Encode(s.Seller); err != nil {
		return err
	}

	return encoder.Encode(s.Price)
}
