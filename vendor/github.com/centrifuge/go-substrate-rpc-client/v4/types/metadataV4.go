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
	"hash"
	"strings"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/xxhash"
	"golang.org/x/crypto/blake2b"
)

// Modelled after https://github.com/paritytech/substrate/blob/v1.0.0rc2/srml/metadata/src/lib.rs
type MetadataV4 struct {
	Modules []ModuleMetadataV4
}

func (m *MetadataV4) Decode(decoder scale.Decoder) error {
	err := decoder.Decode(&m.Modules)
	if err != nil {
		return err
	}
	return nil
}

func (m MetadataV4) Encode(encoder scale.Encoder) error {
	err := encoder.Encode(m.Modules)
	if err != nil {
		return err
	}
	return nil
}

func (m *MetadataV4) FindCallIndex(call string) (CallIndex, error) {
	s := strings.Split(call, ".")
	mi := uint8(0)
	for _, mod := range m.Modules {
		if !mod.HasCalls {
			continue
		}
		if string(mod.Name) != s[0] {
			mi++
			continue
		}
		for ci, f := range mod.Calls {
			if string(f.Name) == s[1] {
				return CallIndex{mi, uint8(ci)}, nil
			}
		}
		return CallIndex{}, fmt.Errorf("method %v not found within module %v for call %v", s[1], mod.Name, call)
	}
	return CallIndex{}, fmt.Errorf("module %v not found in metadata for call %v", s[0], call)
}

func (m *MetadataV4) FindEventNamesForEventID(eventID EventID) (Text, Text, error) {
	mi := uint8(0)
	for _, mod := range m.Modules {
		if !mod.HasEvents {
			continue
		}
		if mi != eventID[0] {
			mi++
			continue
		}
		if int(eventID[1]) >= len(mod.Events) {
			return "", "", fmt.Errorf("event index %v for module %v out of range", eventID[1], mod.Name)
		}
		return mod.Prefix, mod.Events[eventID[1]].Name, nil
	}
	return "", "", fmt.Errorf("module index %v out of range", eventID[0])
}

func (m *MetadataV4) FindConstantValue(_module Text, _constant Text) ([]byte, error) {
	return nil, fmt.Errorf("constants are only supported from metadata v6 and up")
}

func (m *MetadataV4) FindStorageEntryMetadata(module string, fn string) (StorageEntryMetadata, error) {
	for _, mod := range m.Modules {
		if !mod.HasStorage {
			continue
		}
		if string(mod.Prefix) != module {
			continue
		}
		for _, s := range mod.Storage {
			if string(s.Name) != fn {
				continue
			}
			return s, nil
		}
		return nil, fmt.Errorf("storage %v not found within module %v", fn, module)
	}
	return nil, fmt.Errorf("module %v not found in metadata", module)
}

func (m *MetadataV4) ExistsModuleMetadata(module string) bool {
	for _, mod := range m.Modules {
		if string(mod.Prefix) == module {
			return true
		}
	}
	return false
}

type ModuleMetadataV4 struct {
	Name       Text
	Prefix     Text
	HasStorage bool
	Storage    []StorageFunctionMetadataV4
	HasCalls   bool
	Calls      []FunctionMetadataV4
	HasEvents  bool
	Events     []EventMetadataV4
}

func (m *ModuleMetadataV4) Decode(decoder scale.Decoder) error {
	err := decoder.Decode(&m.Name)
	if err != nil {
		return err
	}

	err = decoder.Decode(&m.Prefix)
	if err != nil {
		return err
	}

	err = decoder.DecodeOption(&m.HasStorage, &m.Storage)
	if err != nil {
		return err
	}

	err = decoder.DecodeOption(&m.HasCalls, &m.Calls)
	if err != nil {
		return err
	}

	err = decoder.DecodeOption(&m.HasEvents, &m.Events)
	if err != nil {
		return err
	}
	return nil
}

func (m ModuleMetadataV4) Encode(encoder scale.Encoder) error {
	err := encoder.Encode(m.Name)
	if err != nil {
		return err
	}

	err = encoder.Encode(m.Prefix)
	if err != nil {
		return err
	}

	err = encoder.EncodeOption(m.HasStorage, m.Storage)
	if err != nil {
		return err
	}

	err = encoder.EncodeOption(m.HasCalls, m.Calls)
	if err != nil {
		return err
	}

	err = encoder.EncodeOption(m.HasEvents, m.Events)
	if err != nil {
		return err
	}
	return nil
}

type StorageFunctionMetadataV4 struct {
	Name          Text
	Modifier      StorageFunctionModifierV0
	Type          StorageFunctionTypeV4
	Fallback      Bytes
	Documentation []Text
}

type StorageFunctionTypeV4 struct {
	IsType      bool
	AsType      Type // 0
	IsMap       bool
	AsMap       MapTypeV4 // 1
	IsDoubleMap bool
	AsDoubleMap DoubleMapTypeV4 // 2
}

func (s StorageFunctionMetadataV4) IsPlain() bool {
	return s.Type.IsType
}

func (s StorageFunctionMetadataV4) Hasher() (hash.Hash, error) {
	if s.IsMap() {
		return s.Type.AsMap.Hasher.HashFunc()
	}

	return DefaultPlainHasher(s)
}

func (s StorageFunctionMetadataV4) IsMap() bool {
	return s.Type.IsMap || s.Type.IsDoubleMap
}

func (s StorageFunctionMetadataV4) Hashers() ([]hash.Hash, error) {
	if !s.IsMap() {
		return nil, fmt.Errorf("Hashers() is only to be called on Maps")
	}

	if s.Type.IsDoubleMap {
		return nil, fmt.Errorf("getting the two hashers of a DoubleMap is not supported for metadata v4. " +
			"Please upgrade to use metadata v8 or newer")
	}

	hashFn, err := s.Type.AsMap.Hasher.HashFunc()
	if err != nil {
		return nil, err
	}

	return []hash.Hash{hashFn}, nil
}

func (s *StorageFunctionTypeV4) Decode(decoder scale.Decoder) error {
	var t uint8
	err := decoder.Decode(&t)
	if err != nil {
		return err
	}

	switch t {
	case 0:
		s.IsType = true
		err = decoder.Decode(&s.AsType)
		if err != nil {
			return err
		}
	case 1:
		s.IsMap = true
		err = decoder.Decode(&s.AsMap)
		if err != nil {
			return err
		}
	case 2:
		s.IsDoubleMap = true
		err = decoder.Decode(&s.AsDoubleMap)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("received unexpected type %v", t)
	}
	return nil
}

func (s StorageFunctionTypeV4) Encode(encoder scale.Encoder) error {
	switch {
	case s.IsType:
		err := encoder.PushByte(0)
		if err != nil {
			return err
		}
		err = encoder.Encode(s.AsType)
		if err != nil {
			return err
		}
	case s.IsMap:
		err := encoder.PushByte(1)
		if err != nil {
			return err
		}
		err = encoder.Encode(s.AsMap)
		if err != nil {
			return err
		}
	case s.IsDoubleMap:
		err := encoder.PushByte(2)
		if err != nil {
			return err
		}
		err = encoder.Encode(s.AsDoubleMap)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("expected to be either type, map or double map, but none was set: %v", s)
	}
	return nil
}

type DoubleMapTypeV4 struct {
	Hasher     StorageHasher
	Key1       Type
	Key2       Type
	Value      Type
	Key2Hasher Text
}

type MapTypeV4 struct {
	Hasher StorageHasher
	Key    Type
	Value  Type
	Linked bool
}

type StorageHasher struct {
	IsBlake2_128   bool // 0
	IsBlake2_256   bool // 1
	IsTwox128      bool // 2
	IsTwox256      bool // 3
	IsTwox64Concat bool // 4
}

func (s *StorageHasher) Decode(decoder scale.Decoder) error {
	var t uint8
	err := decoder.Decode(&t)
	if err != nil {
		return err
	}

	switch t {
	case 0:
		s.IsBlake2_128 = true
	case 1:
		s.IsBlake2_256 = true
	case 2:
		s.IsTwox128 = true
	case 3:
		s.IsTwox256 = true
	case 4:
		s.IsTwox64Concat = true
	default:
		return fmt.Errorf("received unexpected storage hasher type %v", t)
	}
	return nil
}

func (s StorageHasher) Encode(encoder scale.Encoder) error {
	var t uint8
	switch {
	case s.IsBlake2_128:
		t = 0
	case s.IsBlake2_256:
		t = 1
	case s.IsTwox128:
		t = 2
	case s.IsTwox256:
		t = 3
	case s.IsTwox64Concat:
		t = 4
	default:
		return fmt.Errorf("expected storage hasher, but none was set: %v", s)
	}
	return encoder.PushByte(t)
}

type FunctionMetadataV4 struct {
	Name          Text
	Args          []FunctionArgumentMetadata
	Documentation []Text
}

type EventMetadataV4 struct {
	Name          Text
	Args          []Type
	Documentation []Text
}

func (s StorageHasher) HashFunc() (hash.Hash, error) {
	// Blake2_128
	if s.IsBlake2_128 {
		return blake2b.New(128, nil)
	}

	// Blake2_256
	if s.IsBlake2_256 {
		return blake2b.New256(nil)
	}

	// Twox128
	if s.IsTwox128 {
		return xxhash.New128(nil), nil
	}

	// Twox256
	if s.IsTwox256 {
		return xxhash.New256(nil), nil
	}

	// Twox64Concat
	if s.IsTwox64Concat {
		return xxhash.New64Concat(nil), nil
	}

	return nil, errors.New("hash function type not yet supported")
}

type FunctionArgumentMetadata struct {
	Name Text
	Type Type
}

type StorageFunctionModifierV0 struct {
	IsOptional bool // 0
	IsDefault  bool // 1
	IsRequired bool // 2
}

func (s *StorageFunctionModifierV0) Decode(decoder scale.Decoder) error {
	var t uint8
	err := decoder.Decode(&t)
	if err != nil {
		return err
	}

	switch t {
	case 0:
		s.IsOptional = true
	case 1:
		s.IsDefault = true
	case 2:
		s.IsRequired = true
	default:
		return fmt.Errorf("received unexpected storage function modifier type %v", t)
	}
	return nil
}

func (s StorageFunctionModifierV0) Encode(encoder scale.Encoder) error {
	var t uint8
	switch {
	case s.IsOptional:
		t = 0
	case s.IsDefault:
		t = 1
	case s.IsRequired:
		t = 2
	default:
		return fmt.Errorf("expected storage function modifier, but none was set: %v", s)
	}
	return encoder.PushByte(t)
}
