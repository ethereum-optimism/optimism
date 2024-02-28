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

type JunctionV1 struct {
	IsParachain bool
	ParachainID UCompact

	IsAccountID32        bool
	AccountID32NetworkID NetworkID
	AccountID            []U8

	IsAccountIndex64        bool
	AccountIndex64NetworkID NetworkID
	AccountIndex            U64

	IsAccountKey20        bool
	AccountKey20NetworkID NetworkID
	AccountKey            []U8

	IsPalletInstance bool
	PalletIndex      U8

	IsGeneralIndex bool
	GeneralIndex   U128

	IsGeneralKey bool
	GeneralKey   []U8

	IsOnlyChild bool

	IsPlurality bool
	BodyID      BodyID
	BodyPart    BodyPart
}

func (j *JunctionV1) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		j.IsParachain = true

		return decoder.Decode(&j.ParachainID)
	case 1:
		j.IsAccountID32 = true

		if err := decoder.Decode(&j.AccountID32NetworkID); err != nil {
			return nil
		}

		return decoder.Decode(&j.AccountID)
	case 2:
		j.IsAccountIndex64 = true

		if err := decoder.Decode(&j.AccountIndex64NetworkID); err != nil {
			return nil
		}

		return decoder.Decode(&j.AccountIndex)
	case 3:
		j.IsAccountKey20 = true

		if err := decoder.Decode(&j.AccountKey20NetworkID); err != nil {
			return nil
		}

		return decoder.Decode(&j.AccountKey)
	case 4:
		j.IsPalletInstance = true

		return decoder.Decode(&j.PalletIndex)
	case 5:
		j.IsGeneralIndex = true

		return decoder.Decode(&j.GeneralIndex)
	case 6:
		j.IsGeneralKey = true

		return decoder.Decode(&j.GeneralKey)
	case 7:
		j.IsOnlyChild = true
	case 8:
		j.IsPlurality = true

		if err := decoder.Decode(&j.BodyID); err != nil {
			return err
		}

		return decoder.Decode(&j.BodyPart)
	}

	return nil
}

func (j JunctionV1) Encode(encoder scale.Encoder) error { //nolint:funlen
	switch {
	case j.IsParachain:
		if err := encoder.PushByte(0); err != nil {
			return err
		}

		return encoder.Encode(j.ParachainID)
	case j.IsAccountID32:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		if err := encoder.Encode(j.AccountID32NetworkID); err != nil {
			return err
		}

		return encoder.Encode(j.AccountID)
	case j.IsAccountIndex64:
		if err := encoder.PushByte(2); err != nil {
			return err
		}

		if err := encoder.Encode(j.AccountIndex64NetworkID); err != nil {
			return err
		}

		return encoder.Encode(j.AccountIndex)
	case j.IsAccountKey20:
		if err := encoder.PushByte(3); err != nil {
			return err
		}

		if err := encoder.Encode(j.AccountKey20NetworkID); err != nil {
			return err
		}

		return encoder.Encode(j.AccountKey)
	case j.IsPalletInstance:
		if err := encoder.PushByte(4); err != nil {
			return err
		}

		return encoder.Encode(j.PalletIndex)
	case j.IsGeneralIndex:
		if err := encoder.PushByte(5); err != nil {
			return err
		}

		return encoder.Encode(j.GeneralIndex)
	case j.IsGeneralKey:
		if err := encoder.PushByte(6); err != nil {
			return err
		}

		return encoder.Encode(j.GeneralKey)
	case j.IsOnlyChild:
		return encoder.PushByte(7)
	case j.IsPlurality:
		if err := encoder.PushByte(8); err != nil {
			return err
		}

		if err := encoder.Encode(j.BodyID); err != nil {
			return err
		}

		return encoder.Encode(j.BodyPart)
	}

	return nil
}

type JunctionsV1 struct {
	IsHere bool

	IsX1 bool
	X1   JunctionV1

	IsX2 bool
	X2   [2]JunctionV1

	IsX3 bool
	X3   [3]JunctionV1

	IsX4 bool
	X4   [4]JunctionV1

	IsX5 bool
	X5   [5]JunctionV1

	IsX6 bool
	X6   [6]JunctionV1

	IsX7 bool
	X7   [7]JunctionV1

	IsX8 bool
	X8   [8]JunctionV1
}

func (j *JunctionsV1) Decode(decoder scale.Decoder) error { //nolint:dupl
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		j.IsHere = true
	case 1:
		j.IsX1 = true

		return decoder.Decode(&j.X1)
	case 2:
		j.IsX2 = true

		return decoder.Decode(&j.X2)
	case 3:
		j.IsX3 = true

		return decoder.Decode(&j.X3)
	case 4:
		j.IsX4 = true

		return decoder.Decode(&j.X4)
	case 5:
		j.IsX5 = true

		return decoder.Decode(&j.X5)
	case 6:
		j.IsX6 = true

		return decoder.Decode(&j.X6)
	case 7:
		j.IsX7 = true

		return decoder.Decode(&j.X7)
	case 8:
		j.IsX8 = true

		return decoder.Decode(&j.X8)
	}

	return nil
}

func (j JunctionsV1) Encode(encoder scale.Encoder) error {
	switch {
	case j.IsHere:
		return encoder.PushByte(0)
	case j.IsX1:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		return encoder.Encode(j.X1)
	case j.IsX2:
		if err := encoder.PushByte(2); err != nil {
			return err
		}

		return encoder.Encode(j.X2)
	case j.IsX3:
		if err := encoder.PushByte(3); err != nil {
			return err
		}

		return encoder.Encode(j.X3)
	case j.IsX4:
		if err := encoder.PushByte(4); err != nil {
			return err
		}

		return encoder.Encode(j.X4)
	case j.IsX5:
		if err := encoder.PushByte(5); err != nil {
			return err
		}

		return encoder.Encode(j.X5)
	case j.IsX6:
		if err := encoder.PushByte(6); err != nil {
			return err
		}

		return encoder.Encode(j.X6)
	case j.IsX7:
		if err := encoder.PushByte(7); err != nil {
			return err
		}

		return encoder.Encode(j.X7)
	case j.IsX8:
		if err := encoder.PushByte(8); err != nil {
			return err
		}

		return encoder.Encode(j.X8)
	}

	return nil
}
