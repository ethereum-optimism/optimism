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

package state

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/client"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
)

// GetChildKeys retreives the keys with the given prefix of a specific child storage
func (s *state) GetChildKeys(childStorageKey, prefix types.StorageKey, blockHash types.Hash) (
	[]types.StorageKey, error) {
	return s.getChildKeys(childStorageKey, prefix, &blockHash)
}

// GetChildKeysLatest retreives the keys with the given prefix of a specific child storage for the latest block height
func (s *state) GetChildKeysLatest(childStorageKey, prefix types.StorageKey) ([]types.StorageKey, error) {
	return s.getChildKeys(childStorageKey, prefix, nil)
}

func (s *state) getChildKeys(childStorageKey, prefix types.StorageKey, blockHash *types.Hash) (
	[]types.StorageKey, error) {
	var res []string
	err := client.CallWithBlockHash(s.client, &res, "state_getChildKeys", blockHash, childStorageKey.Hex(), prefix.Hex())
	if err != nil {
		return nil, err
	}

	keys := make([]types.StorageKey, len(res))
	for i, r := range res {
		err = codec.DecodeFromHex(r, &keys[i])
		if err != nil {
			return nil, err
		}
	}
	return keys, err
}
