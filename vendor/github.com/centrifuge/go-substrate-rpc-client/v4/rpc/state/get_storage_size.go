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
)

// GetStorageSize retreives the storage size for the given key
func (s *state) GetStorageSize(key types.StorageKey, blockHash types.Hash) (types.U64, error) {
	return s.getStorageSize(key, &blockHash)
}

// GetStorageSizeLatest retreives the storage size for the given key for the latest block height
func (s *state) GetStorageSizeLatest(key types.StorageKey) (types.U64, error) {
	return s.getStorageSize(key, nil)
}

func (s *state) getStorageSize(key types.StorageKey, blockHash *types.Hash) (types.U64, error) {
	var res types.U64
	err := client.CallWithBlockHash(s.client, &res, "state_getStorageSize", blockHash, key.Hex())
	if err != nil {
		return 0, err
	}
	return res, err
}
