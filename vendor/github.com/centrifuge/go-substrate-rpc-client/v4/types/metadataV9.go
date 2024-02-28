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
	"strings"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
)

// Modelled after packages/types/src/Metadata/v9/Metadata.ts
type MetadataV9 struct {
	Modules []ModuleMetadataV8
}

func (m *MetadataV9) Decode(decoder scale.Decoder) error {
	err := decoder.Decode(&m.Modules)
	if err != nil {
		return err
	}
	return nil
}

func (m MetadataV9) Encode(encoder scale.Encoder) error {
	err := encoder.Encode(m.Modules)
	if err != nil {
		return err
	}
	return nil
}

func (m *MetadataV9) FindCallIndex(call string) (CallIndex, error) {
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

func (m *MetadataV9) FindEventNamesForEventID(eventID EventID) (Text, Text, error) {
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

func (m *MetadataV9) FindConstantValue(module Text, constant Text) ([]byte, error) {
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

func (m *MetadataV9) FindStorageEntryMetadata(module string, fn string) (StorageEntryMetadata, error) {
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

func (m *MetadataV9) ExistsModuleMetadata(module string) bool {
	for _, mod := range m.Modules {
		if string(mod.Name) == module {
			return true
		}
	}
	return false
}
