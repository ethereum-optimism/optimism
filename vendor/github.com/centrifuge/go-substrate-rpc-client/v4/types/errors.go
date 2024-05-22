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

type ModuleError struct {
	Index U8

	Error [4]U8
}

func (m *ModuleError) Decode(decoder scale.Decoder) error {
	if err := decoder.Decode(&m.Index); err != nil {
		return err
	}

	return decoder.Decode(&m.Error)
}

func (m ModuleError) Encode(encoder scale.Encoder) error {
	if err := encoder.Encode(m.Index); err != nil {
		return err
	}

	return encoder.Encode(m.Error)
}

type TokenError struct {
	IsNoFunds bool

	IsWouldDie bool

	IsBelowMinimum bool

	IsCannotCreate bool

	IsUnknownAsset bool

	IsFrozen bool

	IsUnsupported bool
}

func (t *TokenError) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()

	if err != nil {
		return err
	}

	switch b {
	case 0:
		t.IsNoFunds = true
	case 1:
		t.IsWouldDie = true
	case 2:
		t.IsBelowMinimum = true
	case 3:
		t.IsCannotCreate = true
	case 4:
		t.IsUnknownAsset = true
	case 5:
		t.IsFrozen = true
	case 6:
		t.IsUnsupported = true
	}

	return nil
}

func (t TokenError) Encode(encoder scale.Encoder) error {
	switch {
	case t.IsNoFunds:
		return encoder.PushByte(0)
	case t.IsWouldDie:
		return encoder.PushByte(1)
	case t.IsBelowMinimum:
		return encoder.PushByte(2)
	case t.IsCannotCreate:
		return encoder.PushByte(3)
	case t.IsUnknownAsset:
		return encoder.PushByte(4)
	case t.IsFrozen:
		return encoder.PushByte(5)
	case t.IsUnsupported:
		return encoder.PushByte(6)
	}

	return nil
}

type ArithmeticError struct {
	IsUnderflow bool

	IsOverflow bool

	IsDivisionByZero bool
}

func (a *ArithmeticError) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()

	if err != nil {
		return err
	}

	switch b {
	case 0:
		a.IsUnderflow = true
	case 1:
		a.IsOverflow = true
	case 2:
		a.IsDivisionByZero = true
	}

	return nil
}

func (a ArithmeticError) Encode(encoder scale.Encoder) error {
	switch {
	case a.IsUnderflow:
		return encoder.PushByte(0)
	case a.IsOverflow:
		return encoder.PushByte(1)
	case a.IsDivisionByZero:
		return encoder.PushByte(2)
	}

	return nil
}

type TransactionalError struct {
	IsLimitReached bool

	IsNoLayer bool
}

func (t *TransactionalError) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()

	if err != nil {
		return err
	}

	switch b {
	case 0:
		t.IsLimitReached = true
	case 1:
		t.IsNoLayer = true
	}

	return nil
}

func (t TransactionalError) Encode(encoder scale.Encoder) error {
	switch {
	case t.IsLimitReached:
		return encoder.PushByte(0)
	case t.IsNoLayer:
		return encoder.PushByte(1)
	}

	return nil
}

// DispatchError is an error occurring during extrinsic dispatch
type DispatchError struct {
	IsOther bool
	// Skipped by codec in substrate
	// OtherString string

	IsCannotLookup bool

	IsBadOrigin bool

	IsModule    bool
	ModuleError ModuleError

	IsConsumerRemaining bool

	IsNoProviders bool

	IsTooManyConsumers bool

	IsToken    bool
	TokenError TokenError

	IsArithmetic    bool
	ArithmeticError ArithmeticError

	IsTransactional    bool
	TransactionalError TransactionalError
}

func (d *DispatchError) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		d.IsOther = true
	case 1:
		d.IsCannotLookup = true
	case 2:
		d.IsBadOrigin = true
	case 3:
		d.IsModule = true

		return decoder.Decode(&d.ModuleError)
	case 4:
		d.IsConsumerRemaining = true
	case 5:
		d.IsNoProviders = true
	case 6:
		d.IsTooManyConsumers = true
	case 7:
		d.IsToken = true

		return decoder.Decode(&d.TokenError)
	case 8:
		d.IsArithmetic = true

		return decoder.Decode(&d.ArithmeticError)
	case 9:
		d.IsTransactional = true

		return decoder.Decode(&d.TransactionalError)
	}

	return nil
}

func (d DispatchError) Encode(encoder scale.Encoder) error {
	switch {
	case d.IsOther:
		return encoder.PushByte(0)
	case d.IsCannotLookup:
		return encoder.PushByte(1)
	case d.IsBadOrigin:
		return encoder.PushByte(2)
	case d.IsModule:
		if err := encoder.PushByte(3); err != nil {
			return err
		}

		return encoder.Encode(d.ModuleError)
	case d.IsConsumerRemaining:
		return encoder.PushByte(4)
	case d.IsNoProviders:
		return encoder.PushByte(5)
	case d.IsTooManyConsumers:
		return encoder.PushByte(6)
	case d.IsToken:
		if err := encoder.PushByte(7); err != nil {
			return err
		}

		return encoder.Encode(d.TokenError)
	case d.IsArithmetic:
		if err := encoder.PushByte(8); err != nil {
			return err
		}

		return encoder.Encode(d.ArithmeticError)
	case d.IsTransactional:
		if err := encoder.PushByte(9); err != nil {
			return err
		}

		return encoder.Encode(d.TransactionalError)
	}

	return nil
}
