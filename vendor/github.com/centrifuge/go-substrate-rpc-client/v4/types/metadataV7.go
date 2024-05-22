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
)

// Modelled after packages/types/src/Metadata/v7/Metadata.ts
type MetadataV7 struct {
	Modules []ModuleMetadataV7
}

func (m *MetadataV7) Decode(decoder scale.Decoder) error {
	err := decoder.Decode(&m.Modules)
	if err != nil {
		return err
	}
	return nil
}

func (m MetadataV7) Encode(encoder scale.Encoder) error {
	err := encoder.Encode(m.Modules)
	if err != nil {
		return err
	}
	return nil
}

func (m *MetadataV7) FindCallIndex(call string) (CallIndex, error) {
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

func (m *MetadataV7) FindEventNamesForEventID(eventID EventID) (Text, Text, error) {
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

func (m *MetadataV7) FindConstantValue(module Text, constant Text) ([]byte, error) {
	for _, mod := range m.Modules {
		if mod.Name == module {
			for _, cons := range mod.Constants {
				if cons.Name == constant {
					return cons.Value, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("could not find constant %s.%s", module, constant)
}

func (m *MetadataV7) FindStorageEntryMetadata(module string, fn string) (StorageEntryMetadata, error) {
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

func (m *MetadataV7) ExistsModuleMetadata(module string) bool {
	for _, mod := range m.Modules {
		if string(mod.Name) == module {
			return true
		}
	}
	return false
}

type ModuleMetadataV7 struct {
	Name       Text
	HasStorage bool
	Storage    StorageMetadata
	HasCalls   bool
	Calls      []FunctionMetadataV4
	HasEvents  bool
	Events     []EventMetadataV4
	Constants  []ModuleConstantMetadataV6
}

func (m *ModuleMetadataV7) Decode(decoder scale.Decoder) error {
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

	return decoder.Decode(&m.Constants)
}

func (m ModuleMetadataV7) Encode(encoder scale.Encoder) error {
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

	return encoder.Encode(m.Constants)
}

type StorageMetadata struct {
	Prefix Text
	Items  []StorageFunctionMetadataV5
}

type StorageFunctionMetadataV5 struct {
	Name          Text
	Modifier      StorageFunctionModifierV0
	Type          StorageFunctionTypeV5
	Fallback      Bytes
	Documentation []Text
}

type StorageFunctionTypeV5 struct {
	IsType      bool
	AsType      Type // 0
	IsMap       bool
	AsMap       MapTypeV4 // 1
	IsDoubleMap bool
	AsDoubleMap DoubleMapTypeV5 // 2
}

func (s StorageFunctionMetadataV5) IsPlain() bool {
	return s.Type.IsType
}

func (s StorageFunctionMetadataV5) Hasher() (hash.Hash, error) {
	return DefaultPlainHasher(s)
}

func (s StorageFunctionMetadataV5) IsMap() bool {
	return s.Type.IsMap || s.Type.IsDoubleMap
}

func (s StorageFunctionMetadataV5) Hashers() ([]hash.Hash, error) {
	if !s.IsMap() {
		return nil, fmt.Errorf("Hashers() is only to be called on Maps")
	}

	var hashers = collectHashersV5(s.Type)
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

func collectHashersV5(x StorageFunctionTypeV5) []StorageHasher {
	switch {
	case x.IsMap:
		return []StorageHasher{x.AsMap.Hasher}
	case x.IsDoubleMap:
		return []StorageHasher{x.AsDoubleMap.Hasher, x.AsDoubleMap.Key2Hasher}
	default:
		panic("Unexpexted type")
	}
}

func (s *StorageFunctionTypeV5) Decode(decoder scale.Decoder) error {
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

func (s StorageFunctionTypeV5) Encode(encoder scale.Encoder) error {
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

type DoubleMapTypeV5 struct {
	Hasher     StorageHasher
	Key1       Type
	Key2       Type
	Value      Type
	Key2Hasher StorageHasher
}

type ModuleConstantMetadataV6 struct {
	Name          Text
	Type          Type
	Value         Bytes
	Documentation []Text
}
