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

// QueryStorage queries historical storage entries (by key) starting from a start block until an end block
func (s *state) QueryStorage(keys []types.StorageKey, startBlock types.Hash, block types.Hash) (
	[]types.StorageChangeSet, error) {
	return s.queryStorage(keys, startBlock, &block)
}

// QueryStorageLatest queries historical storage entries (by key) starting from a start block until the latest block
func (s *state) QueryStorageLatest(keys []types.StorageKey, startBlock types.Hash) ([]types.StorageChangeSet, error) {
	return s.queryStorage(keys, startBlock, nil)
}

func (s *state) queryStorage(keys []types.StorageKey, startBlock types.Hash, block *types.Hash) (
	[]types.StorageChangeSet, error) {
	hexKeys := make([]string, len(keys))
	for i, key := range keys {
		hexKeys[i] = key.Hex()
	}

	var res []types.StorageChangeSet
	err := client.CallWithBlockHash(s.client, &res, "state_queryStorage", block, hexKeys, startBlock.Hex())
	if err != nil {
		return nil, err
	}

	return res, nil
}
