// Go Substrate RPC Client (GSRPC) provides APIs and types around Polkadot and any Substrate-based chain RPC calls
//
// Copyright 2022 Snowfork
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

package offchain

import (
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
)

// StorageKind ...
type StorageKind string

const (
	// Persistent storage
	Persistent StorageKind = "PERSISTENT"
	// Local storage
	Local StorageKind = "LOCAL"
)

// LocalStorageGet retrieves the stored data
func (c *offchain) LocalStorageGet(kind StorageKind, key []byte) (*types.StorageDataRaw, error) {
	var res string

	err := c.client.Call(&res, "offchain_localStorageGet", kind, fmt.Sprintf("%#x", key))
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	b, err := codec.HexDecodeString(res)
	if err != nil {
		return nil, err
	}

	data := types.NewStorageDataRaw(b)
	return &data, nil
}

// LocalStorageSet saves the data
func (c *offchain) LocalStorageSet(kind StorageKind, key []byte, value []byte) error {
	var res string

	err := c.client.Call(&res, "offchain_localStorageSet", kind, fmt.Sprintf("%#x", key), fmt.Sprintf("%#x", value))
	if err != nil {
		return err
	}

	return nil
}
