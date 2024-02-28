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

// GetStorage retreives the stored data and decodes them into the provided interface. Ok is true if the value is not
// empty.
func (s *state) GetStorage(key types.StorageKey, target interface{}, blockHash types.Hash) (ok bool, err error) {
	raw, err := s.getStorageRaw(key, &blockHash)
	if err != nil {
		return false, err
	}
	if len(*raw) == 0 {
		return false, nil
	}
	return true, codec.Decode(*raw, target)
}

// GetStorageLatest retreives the stored data for the latest block height and decodes them into the provided interface.
// Ok is true if the value is not empty.
func (s *state) GetStorageLatest(key types.StorageKey, target interface{}) (ok bool, err error) {
	raw, err := s.getStorageRaw(key, nil)
	if err != nil {
		return false, err
	}
	if len(*raw) == 0 {
		return false, nil
	}
	return true, codec.Decode(*raw, target)
}

// GetStorageRaw retreives the stored data as raw bytes, without decoding them
func (s *state) GetStorageRaw(key types.StorageKey, blockHash types.Hash) (*types.StorageDataRaw, error) {
	return s.getStorageRaw(key, &blockHash)
}

// GetStorageRawLatest retreives the stored data for the latest block height as raw bytes, without decoding them
func (s *state) GetStorageRawLatest(key types.StorageKey) (*types.StorageDataRaw, error) {
	return s.getStorageRaw(key, nil)
}

func (s *state) getStorageRaw(key types.StorageKey, blockHash *types.Hash) (*types.StorageDataRaw, error) {
	var res string
	err := client.CallWithBlockHash(s.client, &res, "state_getStorage", blockHash, key.Hex())
	if err != nil {
		return nil, err
	}

	bz, err := codec.HexDecodeString(res)
	if err != nil {
		return nil, err
	}

	data := types.NewStorageDataRaw(bz)
	return &data, nil
}
