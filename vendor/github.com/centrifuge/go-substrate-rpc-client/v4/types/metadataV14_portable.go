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
	"errors"
	"fmt"
	"math/big"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
)

type PortableTypeV14 struct {
	ID   Si1LookupTypeID
	Type Si1Type
}

type Si0LookupTypeID UCompact

type Si0Path []Text

// `byte` can only be one of the variants listed below
type Si0TypeDefPrimitive byte

// Si0TypeDefPrimitive variants
const (
	IsBool = 0
	IsChar = 1
	IsStr  = 2
	IsU8   = 3
	IsU16  = 4
	IsU32  = 5
	IsU64  = 6
	IsU128 = 7
	IsU256 = 8
	IsI8   = 9
	IsI16  = 10
	IsI32  = 11
	IsI64  = 12
	IsI128 = 13
	IsI256 = 14
)

func (d *Si0TypeDefPrimitive) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}
	switch b {
	case IsBool:
		*d = IsBool
	case IsChar:
		*d = IsChar
	case IsStr:
		*d = IsStr
	case IsU8:
		*d = IsU8
	case IsU16:
		*d = IsU16
	case IsU32:
		*d = IsU32
	case IsU64:
		*d = IsU64
	case IsU128:
		*d = IsU128
	case IsU256:
		*d = IsU256
	case IsI8:
		*d = IsI8
	case IsI16:
		*d = IsI16
	case IsI32:
		*d = IsI32
	case IsI64:
		*d = IsI64
	case IsI128:
		*d = IsI128
	case IsI256:
		*d = IsI256
	default:
		return fmt.Errorf("Si0TypeDefPrimitive do not support this type: %d", b)
	}
	return nil
}

type Si1LookupTypeID struct {
	UCompact
}

func NewSi1LookupTypeID(value *big.Int) Si1LookupTypeID {
	return Si1LookupTypeID{NewUCompact(value)}
}

func NewSi1LookupTypeIDFromUInt(value uint64) Si1LookupTypeID {
	return NewSi1LookupTypeID(new(big.Int).SetUint64(value))
}

type Si1Path Si0Path

type Si1Type struct {
	Path   Si1Path
	Params []Si1TypeParameter
	Def    Si1TypeDef
	Docs   []Text
}

type Si1TypeParameter struct {
	Name    Text
	HasType bool
	Type    Si1LookupTypeID
}

func (d *Si1TypeParameter) Decode(decoder scale.Decoder) error {
	err := decoder.Decode(&d.Name)
	if err != nil {
		return err
	}

	return decoder.DecodeOption(&d.HasType, &d.Type)
}

func (d Si1TypeParameter) Encode(encoder scale.Encoder) error {
	err := encoder.Encode(d.Name)
	if err != nil {
		return err
	}

	return encoder.EncodeOption(d.HasType, &d.Type)
}

type Si1TypeDef struct {
	IsComposite bool
	Composite   Si1TypeDefComposite

	IsVariant bool
	Variant   Si1TypeDefVariant

	IsSequence bool
	Sequence   Si1TypeDefSequence

	IsArray bool
	Array   Si1TypeDefArray

	IsTuple bool
	Tuple   Si1TypeDefTuple

	IsPrimitive bool
	Primitive   Si1TypeDefPrimitive

	IsCompact bool
	Compact   Si1TypeDefCompact

	IsBitSequence bool
	BitSequence   Si1TypeDefBitSequence

	IsHistoricMetaCompat bool
	HistoricMetaCompat   Type
}

func (d *Si1TypeDef) Decode(decoder scale.Decoder) error {
	num, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}
	switch num {
	case 0:
		d.IsComposite = true
		return decoder.Decode(&d.Composite)
	case 1:
		d.IsVariant = true
		return decoder.Decode(&d.Variant)
	case 2:
		d.IsSequence = true
		return decoder.Decode(&d.Sequence)
	case 3:
		d.IsArray = true
		return decoder.Decode(&d.Array)
	case 4:
		d.IsTuple = true
		return decoder.Decode(&d.Tuple)
	case 5:
		d.IsPrimitive = true
		return decoder.Decode(&d.Primitive)
	case 6:
		d.IsCompact = true
		return decoder.Decode(&d.Compact)
	case 7:
		d.IsBitSequence = true
		return decoder.Decode(&d.BitSequence)
	case 8:
		d.IsHistoricMetaCompat = true
		return decoder.Decode(&d.HistoricMetaCompat)

	default:
		return fmt.Errorf("Si1TypeDef unknow type : %d", num)
	}
}

func (d Si1TypeDef) Encode(encoder scale.Encoder) error { //nolint:funlen
	switch {
	case d.IsComposite:
		err := encoder.PushByte(0)
		if err != nil {
			return err
		}
		return encoder.Encode(&d.Composite)
	case d.IsVariant:
		err := encoder.PushByte(1)
		if err != nil {
			return err
		}
		return encoder.Encode(&d.Variant)
	case d.IsSequence:
		err := encoder.PushByte(2)
		if err != nil {
			return err
		}
		return encoder.Encode(&d.Sequence)
	case d.IsArray:
		err := encoder.PushByte(3)
		if err != nil {
			return err
		}
		return encoder.Encode(&d.Array)
	case d.IsTuple:
		err := encoder.PushByte(4)
		if err != nil {
			return err
		}
		return encoder.Encode(&d.Tuple)
	case d.IsPrimitive:
		err := encoder.PushByte(5)
		if err != nil {
			return err
		}
		return encoder.Encode(&d.Primitive)
	case d.IsCompact:
		err := encoder.PushByte(6)
		if err != nil {
			return err
		}
		return encoder.Encode(&d.Compact)
	case d.IsBitSequence:
		err := encoder.PushByte(7)
		if err != nil {
			return err
		}
		return encoder.Encode(&d.BitSequence)
	case d.IsHistoricMetaCompat:
		err := encoder.PushByte(8)
		if err != nil {
			return err
		}
		d.IsHistoricMetaCompat = true
		return encoder.Encode(&d.HistoricMetaCompat)

	default:
		return errors.New("expected Si1TypeDef instance to be one of the valid variants")
	}
}

type Si1TypeDefComposite struct {
	Fields []Si1Field
}

type Si1Field struct {
	HasName     bool
	Name        Text
	Type        Si1LookupTypeID
	HasTypeName bool
	TypeName    Text
	Docs        []Text
}

func (d *Si1Field) Decode(decoder scale.Decoder) error {
	err := decoder.DecodeOption(&d.HasName, &d.Name)
	if err != nil {
		return err
	}

	err = decoder.Decode(&d.Type)
	if err != nil {
		return err
	}

	err = decoder.DecodeOption(&d.HasTypeName, &d.TypeName)
	if err != nil {
		return err
	}

	return decoder.Decode(&d.Docs)
}

func (d Si1Field) Encode(encoder scale.Encoder) error {
	err := encoder.EncodeOption(d.HasName, d.Name)
	if err != nil {
		return err
	}
	err = encoder.Encode(d.Type)
	if err != nil {
		return err
	}
	err = encoder.EncodeOption(d.HasTypeName, d.TypeName)
	if err != nil {
		return err
	}
	return encoder.Encode(&d.Docs)
}

type Si1TypeDefVariant struct {
	Variants []Si1Variant
}

type Si1Variant struct {
	Name   Text
	Fields []Si1Field
	Index  U8
	Docs   []Text
}

type Si1TypeDefSequence struct {
	Type Si1LookupTypeID
}

type Si1TypeDefArray struct {
	Len  U32
	Type Si1LookupTypeID
}

type Si1TypeDefTuple []Si1LookupTypeID

type Si1TypeDefPrimitive struct {
	Si0TypeDefPrimitive
}

type Si1TypeDefCompact struct {
	Type Si1LookupTypeID
}

type Si1TypeDefBitSequence struct {
	BitStoreType Si1LookupTypeID
	BitOrderType Si1LookupTypeID
}
