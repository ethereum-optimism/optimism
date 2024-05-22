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
	"fmt"
	"hash"
	"strings"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/xxhash"
)

const MagicNumber uint32 = 0x6174656d

// Modelled after https://github.com/paritytech/substrate/blob/v1.0.0rc2/srml/metadata/src/lib.rs

type Metadata struct {
	MagicNumber uint32
	// The version in use
	Version uint8

	// The right metadata version should be used based on the
	// version defined above.

	AsMetadataV4  MetadataV4
	AsMetadataV7  MetadataV7
	AsMetadataV8  MetadataV8
	AsMetadataV9  MetadataV9
	AsMetadataV10 MetadataV10
	AsMetadataV11 MetadataV11
	AsMetadataV12 MetadataV12
	AsMetadataV13 MetadataV13
	AsMetadataV14 MetadataV14
}

type StorageEntryMetadata interface {
	// Check whether the entry is a plain type
	IsPlain() bool
	// Get the hasher to store the plain type
	Hasher() (hash.Hash, error)

	// Check whether the entry is a map type.
	// Since v14, a Map is the union of the old Map, DoubleMap, and NMap.
	IsMap() bool
	// Get the hashers of the map keys. It should contain one hash per key.
	Hashers() ([]hash.Hash, error)
}

func NewMetadataV4() *Metadata {
	return &Metadata{Version: 4, AsMetadataV4: MetadataV4{make([]ModuleMetadataV4, 0)}}
}

func NewMetadataV7() *Metadata {
	return &Metadata{Version: 7, AsMetadataV7: MetadataV7{make([]ModuleMetadataV7, 0)}}
}

func NewMetadataV8() *Metadata {
	return &Metadata{Version: 8, AsMetadataV8: MetadataV8{make([]ModuleMetadataV8, 0)}}
}

func NewMetadataV9() *Metadata {
	return &Metadata{Version: 9, AsMetadataV9: MetadataV9{make([]ModuleMetadataV8, 0)}}
}

func NewMetadataV10() *Metadata {
	return &Metadata{Version: 10, AsMetadataV10: MetadataV10{make([]ModuleMetadataV10, 0)}}
}

func NewMetadataV11() *Metadata {
	return &Metadata{
		Version:       11,
		AsMetadataV11: MetadataV11{MetadataV10: MetadataV10{Modules: make([]ModuleMetadataV10, 0)}},
	}
}

func NewMetadataV12() *Metadata {
	return &Metadata{
		Version:       12,
		AsMetadataV12: MetadataV12{Modules: make([]ModuleMetadataV12, 0)},
	}
}

func NewMetadataV13() *Metadata {
	return &Metadata{
		Version:       13,
		AsMetadataV13: MetadataV13{Modules: make([]ModuleMetadataV13, 0)},
	}
}

func NewMetadataV14() *Metadata {
	return &Metadata{
		Version:       14,
		AsMetadataV14: MetadataV14{Pallets: make([]PalletMetadataV14, 0)},
	}
}

func (m *Metadata) Decode(decoder scale.Decoder) error {
	err := decoder.Decode(&m.MagicNumber)
	if err != nil {
		return err
	}
	if m.MagicNumber != MagicNumber {
		return fmt.Errorf("magic number mismatch: expected %#x, found %#x", MagicNumber, m.MagicNumber)
	}

	err = decoder.Decode(&m.Version)
	if err != nil {
		return err
	}

	switch m.Version {
	case 4:
		err = decoder.Decode(&m.AsMetadataV4)
	case 7:
		err = decoder.Decode(&m.AsMetadataV7)
	case 8:
		err = decoder.Decode(&m.AsMetadataV8)
	case 9:
		err = decoder.Decode(&m.AsMetadataV9)
	case 10:
		err = decoder.Decode(&m.AsMetadataV10)
	case 11:
		err = decoder.Decode(&m.AsMetadataV11)
	case 12:
		err = decoder.Decode(&m.AsMetadataV12)
	case 13:
		err = decoder.Decode(&m.AsMetadataV13)
	case 14:
		err = decoder.Decode(&m.AsMetadataV14)
	default:
		return fmt.Errorf("unsupported metadata version %v", m.Version)
	}

	return err
}

func (m Metadata) Encode(encoder scale.Encoder) error {
	err := encoder.Encode(m.MagicNumber)
	if err != nil {
		return err
	}

	err = encoder.Encode(m.Version)
	if err != nil {
		return err
	}

	switch m.Version {
	case 4:
		err = encoder.Encode(m.AsMetadataV4)
	case 7:
		err = encoder.Encode(m.AsMetadataV7)
	case 8:
		err = encoder.Encode(m.AsMetadataV8)
	case 9:
		err = encoder.Encode(m.AsMetadataV9)
	case 10:
		err = encoder.Encode(m.AsMetadataV10)
	case 11:
		err = encoder.Encode(m.AsMetadataV11)
	case 12:
		err = encoder.Encode(m.AsMetadataV12)
	case 13:
		err = encoder.Encode(m.AsMetadataV13)
	case 14:
		err = encoder.Encode(m.AsMetadataV14)
	default:
		return fmt.Errorf("unsupported metadata version %v", m.Version)
	}

	return err
}

type MetadataError struct {
	Name  string
	Value string
}

const (
	metadataErrorValueSeparator = ", "
)

func NewMetadataError(variant Si1Variant) *MetadataError {
	var docs []string

	for _, doc := range variant.Docs {
		docs = append(docs, string(doc))
	}

	return &MetadataError{
		Name:  string(variant.Name),
		Value: strings.Join(docs, metadataErrorValueSeparator),
	}
}

func (m *Metadata) FindError(moduleIndex U8, errorIndex [4]U8) (*MetadataError, error) {
	if m.Version != 14 {
		return nil, fmt.Errorf("invalid metadata version %d", m.Version)
	}

	return m.AsMetadataV14.FindError(moduleIndex, errorIndex)
}

func (m *Metadata) FindConstantValue(module string, constantName string) ([]byte, error) {
	txtModule := Text(module)
	txtConstantName := Text(constantName)

	switch m.Version {
	case 4:
		return m.AsMetadataV4.FindConstantValue(txtModule, txtConstantName)
	case 7:
		return m.AsMetadataV7.FindConstantValue(txtModule, txtConstantName)
	case 8:
		return m.AsMetadataV8.FindConstantValue(txtModule, txtConstantName)
	case 9:
		return m.AsMetadataV9.FindConstantValue(txtModule, txtConstantName)
	case 10:
		return m.AsMetadataV10.FindConstantValue(txtModule, txtConstantName)
	case 11:
		return m.AsMetadataV11.FindConstantValue(txtModule, txtConstantName)
	case 12:
		return m.AsMetadataV12.FindConstantValue(txtModule, txtConstantName)
	case 13:
		return m.AsMetadataV13.FindConstantValue(txtModule, txtConstantName)
	case 14:
		return m.AsMetadataV14.FindConstantValue(txtModule, txtConstantName)
	default:
		return nil, fmt.Errorf("unsupported metadata version")
	}
}

func (m *Metadata) FindCallIndex(call string) (CallIndex, error) {
	switch m.Version {
	case 4:
		return m.AsMetadataV4.FindCallIndex(call)
	case 7:
		return m.AsMetadataV7.FindCallIndex(call)
	case 8:
		return m.AsMetadataV8.FindCallIndex(call)
	case 9:
		return m.AsMetadataV9.FindCallIndex(call)
	case 10:
		return m.AsMetadataV10.FindCallIndex(call)
	case 11:
		return m.AsMetadataV11.FindCallIndex(call)
	case 12:
		return m.AsMetadataV12.FindCallIndex(call)
	case 13:
		return m.AsMetadataV13.FindCallIndex(call)
	case 14:
		return m.AsMetadataV14.FindCallIndex(call)
	default:
		return CallIndex{}, fmt.Errorf("unsupported metadata version")
	}
}

func (m *Metadata) FindEventNamesForEventID(eventID EventID) (Text, Text, error) {
	switch m.Version {
	case 4:
		return m.AsMetadataV4.FindEventNamesForEventID(eventID)
	case 7:
		return m.AsMetadataV7.FindEventNamesForEventID(eventID)
	case 8:
		return m.AsMetadataV8.FindEventNamesForEventID(eventID)
	case 9:
		return m.AsMetadataV9.FindEventNamesForEventID(eventID)
	case 10:
		return m.AsMetadataV10.FindEventNamesForEventID(eventID)
	case 11:
		return m.AsMetadataV11.FindEventNamesForEventID(eventID)
	case 12:
		return m.AsMetadataV12.FindEventNamesForEventID(eventID)
	case 13:
		return m.AsMetadataV13.FindEventNamesForEventID(eventID)
	case 14:
		return m.AsMetadataV14.FindEventNamesForEventID(eventID)
	default:
		return "", "", fmt.Errorf("unsupported metadata version")
	}
}

func (m *Metadata) FindStorageEntryMetadata(module string, fn string) (StorageEntryMetadata, error) {
	switch m.Version {
	case 4:
		return m.AsMetadataV4.FindStorageEntryMetadata(module, fn)
	case 7:
		return m.AsMetadataV7.FindStorageEntryMetadata(module, fn)
	case 8:
		return m.AsMetadataV8.FindStorageEntryMetadata(module, fn)
	case 9:
		return m.AsMetadataV9.FindStorageEntryMetadata(module, fn)
	case 10:
		return m.AsMetadataV10.FindStorageEntryMetadata(module, fn)
	case 11:
		return m.AsMetadataV11.FindStorageEntryMetadata(module, fn)
	case 12:
		return m.AsMetadataV12.FindStorageEntryMetadata(module, fn)
	case 13:
		return m.AsMetadataV13.FindStorageEntryMetadata(module, fn)
	case 14:
		return m.AsMetadataV14.FindStorageEntryMetadata(module, fn)
	default:
		return nil, fmt.Errorf("unsupported metadata version")
	}
}

func (m *Metadata) ExistsModuleMetadata(module string) bool {
	switch m.Version {
	case 4:
		return m.AsMetadataV4.ExistsModuleMetadata(module)
	case 7:
		return m.AsMetadataV7.ExistsModuleMetadata(module)
	case 8:
		return m.AsMetadataV8.ExistsModuleMetadata(module)
	case 9:
		return m.AsMetadataV9.ExistsModuleMetadata(module)
	case 10:
		return m.AsMetadataV10.ExistsModuleMetadata(module)
	case 11:
		return m.AsMetadataV11.ExistsModuleMetadata(module)
	case 12:
		return m.AsMetadataV12.ExistsModuleMetadata(module)
	case 13:
		return m.AsMetadataV13.ExistsModuleMetadata(module)
	case 14:
		return m.AsMetadataV14.ExistsModuleMetadata(module)
	default:
		return false
	}
}

// Default implementation of Hasher() for a Storage entry
// It fails when called if entry is not a plain type.
func DefaultPlainHasher(entry StorageEntryMetadata) (hash.Hash, error) {
	if entry.IsPlain() {
		return xxhash.New128(nil), nil
	}

	return nil, fmt.Errorf("Hasher() is only to be called on a Plain entry")
}
