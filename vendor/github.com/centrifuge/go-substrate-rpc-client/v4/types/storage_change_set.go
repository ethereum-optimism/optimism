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
	"encoding/json"
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
)

// StorageChangeSet contains changes from storage subscriptions
type StorageChangeSet struct {
	Block   Hash             `json:"block"`
	Changes []KeyValueOption `json:"changes"`
}

type KeyValueOption struct {
	StorageKey     StorageKey
	HasStorageData bool
	StorageData    StorageDataRaw
}

func (r *KeyValueOption) UnmarshalJSON(b []byte) error {
	var tmp []string
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	switch len(tmp) {
	case 0:
		return fmt.Errorf("expected at least one entry for KeyValueOption")
	case 2:
		r.HasStorageData = true
		data, err := codec.HexDecodeString(tmp[1])
		if err != nil {
			return err
		}
		r.StorageData = data
		fallthrough
	case 1:
		key, err := codec.HexDecodeString(tmp[0])
		if err != nil {
			return err
		}
		r.StorageKey = key
	default:
		return fmt.Errorf("expected 1 or 2 entries for KeyValueOption, got %v", len(tmp))
	}
	return nil
}

func (r KeyValueOption) MarshalJSON() ([]byte, error) {
	var tmp []string
	if r.HasStorageData {
		tmp = []string{r.StorageKey.Hex(), r.StorageData.Hex()}
	} else {
		tmp = []string{r.StorageKey.Hex()}
	}
	return json.Marshal(tmp)
}
