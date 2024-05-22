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

	ghash "github.com/centrifuge/go-substrate-rpc-client/v4/hash"
	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/xxhash"
)

// Modelled after packages/types/src/Metadata/v10/Metadata.ts
type MetadataV10 struct {
	Modules []ModuleMetadataV10
}

func (m *MetadataV10) Decode(decoder scale.Decoder) error {
	err := decoder.Decode(&m.Modules)
	if err != nil {
		return err
	}
	return nil
}

func (m MetadataV10) Encode(encoder scale.Encoder) error {
	err := encoder.Encode(m.Modules)
	if err != nil {
		return err
	}
	return nil
}

func (m *MetadataV10) FindCallIndex(call string) (CallIndex, error) {
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

func (m *MetadataV10) FindEventNamesForEventID(eventID EventID) (Text, Text, error) {
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
		return mod.Name, mod.Events[eventID[1]].Name, nil
	}
	return "", "", fmt.Errorf("module index %v out of range", eventID[0])
}

func (m *MetadataV10) FindConstantValue(module Text, constant Text) ([]byte, error) {
	for _, mod := range m.Modules {
		if mod.Name == module {
			value, err := mod.FindConstantValue(constant)
			if err == nil {
				return value, nil
			}
		}
	}
	return nil, fmt.Errorf("could not find constant %s.%s", module, constant)
}

func (m *MetadataV10) FindStorageEntryMetadata(module string, fn string) (StorageEntryMetadata, error) {
	for _, mod := range m.Modules {
		if !mod.HasStorage {
			continue
		}
		if string(mod.Storage.Prefix) != module {
			continue
		}
		for _, s := range mod.Storage.Items {
			if string(s.Name) != fn {
				continue
			}
			return s, nil
		}
		return nil, fmt.Errorf("storage %v not found within module %v", fn, module)
	}
	return nil, fmt.Errorf("module %v not found in metadata", module)
}

func (m *MetadataV10) ExistsModuleMetadata(module string) bool {
	for _, mod := range m.Modules {
		if string(mod.Name) == module {
			return true
		}
	}
	return false
}

type ModuleMetadataV10 struct {
	Name       Text
	HasStorage bool
	Storage    StorageMetadataV10
	HasCalls   bool
	Calls      []FunctionMetadataV4
	HasEvents  bool
	Events     []EventMetadataV4
	Constants  []ModuleConstantMetadataV6
	Errors     []ErrorMetadataV8
}

func (m *ModuleMetadataV10) Decode(decoder scale.Decoder) error {
	err := decoder.Decode(&m.Name)
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

	err = decoder.Decode(&m.Constants)
	if err != nil {
		return err
	}

	return decoder.Decode(&m.Errors)
}

func (m ModuleMetadataV10) Encode(encoder scale.Encoder) error {
	err := encoder.Encode(m.Name)
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

	err = encoder.Encode(m.Constants)
	if err != nil {
		return err
	}

	return encoder.Encode(m.Errors)
}

func (m *ModuleMetadataV10) FindConstantValue(constant Text) ([]byte, error) {
	for _, cons := range m.Constants {
		if cons.Name == constant {
			return cons.Value, nil
		}
	}
	return nil, fmt.Errorf("could not find constant %s", constant)
}

type StorageMetadataV10 struct {
	Prefix Text
	Items  []StorageFunctionMetadataV10
}

type StorageFunctionMetadataV10 struct {
	Name          Text
	Modifier      StorageFunctionModifierV0
	Type          StorageFunctionTypeV10
	Fallback      Bytes
	Documentation []Text
}

func (s StorageFunctionMetadataV10) IsPlain() bool {
	return s.Type.IsType
}

func (s StorageFunctionMetadataV10) Hasher() (hash.Hash, error) {
	return DefaultPlainHasher(s)
}

func (s StorageFunctionMetadataV10) IsMap() bool {
	return s.Type.IsMap || s.Type.IsDoubleMap
}

func (s StorageFunctionMetadataV10) Hashers() ([]hash.Hash, error) {
	if !s.IsMap() {
		return nil, fmt.Errorf("Hashers() is only to be called on Maps")
	}

	var hashers = collectHashersV10(s.Type)
	hasherFns := make([]hash.Hash, len(hashers))
	for i, hasher := range hashers {
		hasherFn, err := hasher.HashFunc()
		if err != nil {
			return nil, err
		}
		hasherFns[i] = hasherFn
	}
	return hasherFns, nil
}

func collectHashersV10(x StorageFunctionTypeV10) []StorageHasherV10 {
	switch {
	case x.IsMap:
		return []StorageHasherV10{x.AsMap.Hasher}
	case x.IsDoubleMap:
		return []StorageHasherV10{x.AsDoubleMap.Hasher, x.AsDoubleMap.Key2Hasher}
	default:
		panic("Unexpexted type")
	}
}

type StorageFunctionTypeV10 struct {
	IsType      bool
	AsType      Type // 0
	IsMap       bool
	AsMap       MapTypeV10 // 1
	IsDoubleMap bool
	AsDoubleMap DoubleMapTypeV10 // 2
}

func (s *StorageFunctionTypeV10) Decode(decoder scale.Decoder) error {
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

func (s StorageFunctionTypeV10) Encode(encoder scale.Encoder) error {
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

type MapTypeV10 struct {
	Hasher StorageHasherV10
	Key    Type
	Value  Type
	Linked bool
}

type DoubleMapTypeV10 struct {
	Hasher     StorageHasherV10
	Key1       Type
	Key2       Type
	Value      Type
	Key2Hasher StorageHasherV10
}

type StorageHasherV10 struct {
	IsBlake2_128       bool // 0
	IsBlake2_256       bool // 1
	IsBlake2_128Concat bool // 2
	IsTwox128          bool // 3
	IsTwox256          bool // 4
	IsTwox64Concat     bool // 5
	IsIdentity         bool // 6
}

func (s *StorageHasherV10) Decode(decoder scale.Decoder) error {
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
		s.IsBlake2_128Concat = true
	case 3:
		s.IsTwox128 = true
	case 4:
		s.IsTwox256 = true
	case 5:
		s.IsTwox64Concat = true
	case 6:
		s.IsIdentity = true
	default:
		return fmt.Errorf("received unexpected storage hasher type %v", t)
	}
	return nil
}

func (s StorageHasherV10) Encode(encoder scale.Encoder) error {
	var t uint8
	switch {
	case s.IsBlake2_128:
		t = 0
	case s.IsBlake2_256:
		t = 1
	case s.IsBlake2_128Concat:
		t = 2
	case s.IsTwox128:
		t = 3
	case s.IsTwox256:
		t = 4
	case s.IsTwox64Concat:
		t = 5
	case s.IsIdentity:
		t = 6
	default:
		return fmt.Errorf("expected storage hasher, but none was set: %v", s)
	}
	return encoder.PushByte(t)
}

func (s StorageHasherV10) HashFunc() (hash.Hash, error) {
	// Blake2_128
	if s.IsBlake2_128 {
		return ghash.NewBlake2b128(nil)
	}

	// Blake2_256
	if s.IsBlake2_256 {
		return ghash.NewBlake2b256(nil)
	}

	// Blake2_128Concat
	if s.IsBlake2_128Concat {
		return ghash.NewBlake2b128Concat(nil)
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

	// Identity
	if s.IsIdentity {
		return ghash.NewIdentity(nil), nil
	}

	return nil, errors.New("hash function type not yet supported")
}
