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

type BodyID struct {
	IsUnit bool

	IsNamed bool
	Body    []U8

	IsIndex bool
	Index   U32

	IsExecutive bool

	IsTechnical bool

	IsLegislative bool

	IsJudicial bool
}

func (b *BodyID) Decode(decoder scale.Decoder) error {
	bb, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch bb {
	case 0:
		b.IsUnit = true
	case 1:
		b.IsNamed = true

		return decoder.Decode(&b.Body)
	case 2:
		b.IsIndex = true

		return decoder.Decode(&b.Index)
	case 3:
		b.IsExecutive = true
	case 4:
		b.IsTechnical = true
	case 5:
		b.IsLegislative = true
	case 6:
		b.IsJudicial = true
	}

	return nil
}

func (b BodyID) Encode(encoder scale.Encoder) error {
	switch {
	case b.IsUnit:
		return encoder.PushByte(0)
	case b.IsNamed:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		return encoder.Encode(b.Body)
	case b.IsIndex:
		if err := encoder.PushByte(2); err != nil {
			return err
		}

		return encoder.Encode(b.Index)
	case b.IsExecutive:
		return encoder.PushByte(3)
	case b.IsTechnical:
		return encoder.PushByte(4)
	case b.IsLegislative:
		return encoder.PushByte(5)
	case b.IsJudicial:
		return encoder.PushByte(6)
	}

	return nil
}

type BodyPart struct {
	IsVoice bool

	IsMembers    bool
	MembersCount U32

	IsFraction    bool
	FractionNom   U32
	FractionDenom U32

	IsAtLeastProportion    bool
	AtLeastProportionNom   U32
	AtLeastProportionDenom U32

	IsMoreThanProportion    bool
	MoreThanProportionNom   U32
	MoreThanProportionDenom U32
}

func (b *BodyPart) Decode(decoder scale.Decoder) error {
	bb, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch bb {
	case 0:
		b.IsVoice = true
	case 1:
		b.IsMembers = true

		return decoder.Decode(&b.MembersCount)
	case 2:
		b.IsFraction = true

		if err := decoder.Decode(&b.FractionNom); err != nil {
			return err
		}

		return decoder.Decode(&b.FractionDenom)
	case 3:
		b.IsAtLeastProportion = true

		if err := decoder.Decode(&b.AtLeastProportionNom); err != nil {
			return err
		}

		return decoder.Decode(&b.AtLeastProportionDenom)
	case 4:
		b.IsMoreThanProportion = true

		if err := decoder.Decode(&b.MoreThanProportionNom); err != nil {
			return err
		}

		return decoder.Decode(&b.MoreThanProportionDenom)
	}

	return nil
}

func (b BodyPart) Encode(encoder scale.Encoder) error {
	switch {
	case b.IsVoice:
		return encoder.PushByte(0)
	case b.IsMembers:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		return encoder.Encode(b.MembersCount)
	case b.IsFraction:
		if err := encoder.PushByte(2); err != nil {
			return err
		}

		if err := encoder.Encode(b.FractionNom); err != nil {
			return err
		}

		return encoder.Encode(b.FractionDenom)
	case b.IsAtLeastProportion:
		if err := encoder.PushByte(3); err != nil {
			return err
		}

		if err := encoder.Encode(b.AtLeastProportionNom); err != nil {
			return err
		}

		return encoder.Encode(b.AtLeastProportionDenom)
	case b.IsMoreThanProportion:
		if err := encoder.PushByte(4); err != nil {
			return err
		}

		if err := encoder.Encode(b.MoreThanProportionNom); err != nil {
			return err
		}

		return encoder.Encode(b.MoreThanProportionDenom)
	}

	return nil
}
