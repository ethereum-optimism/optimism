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

//go:generate mockery --name State --filename state.go

package state

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/client"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type State interface {
	GetStorage(key types.StorageKey, target interface{}, blockHash types.Hash) (ok bool, err error)
	GetStorageLatest(key types.StorageKey, target interface{}) (ok bool, err error)
	GetStorageRaw(key types.StorageKey, blockHash types.Hash) (*types.StorageDataRaw, error)
	GetStorageRawLatest(key types.StorageKey) (*types.StorageDataRaw, error)

	GetChildStorageSize(childStorageKey, key types.StorageKey, blockHash types.Hash) (types.U64, error)
	GetChildStorageSizeLatest(childStorageKey, key types.StorageKey) (types.U64, error)
	GetChildStorage(childStorageKey, key types.StorageKey, target interface{}, blockHash types.Hash) (ok bool, err error)
	GetChildStorageLatest(childStorageKey, key types.StorageKey, target interface{}) (ok bool, err error)
	GetChildStorageRaw(childStorageKey, key types.StorageKey, blockHash types.Hash) (*types.StorageDataRaw, error)
	GetChildStorageRawLatest(childStorageKey, key types.StorageKey) (*types.StorageDataRaw, error)

	GetMetadata(blockHash types.Hash) (*types.Metadata, error)
	GetMetadataLatest() (*types.Metadata, error)

	GetStorageHash(key types.StorageKey, blockHash types.Hash) (types.Hash, error)
	GetStorageHashLatest(key types.StorageKey) (types.Hash, error)

	SubscribeStorageRaw(keys []types.StorageKey) (*StorageSubscription, error)

	GetRuntimeVersion(blockHash types.Hash) (*types.RuntimeVersion, error)
	GetRuntimeVersionLatest() (*types.RuntimeVersion, error)

	GetChildKeys(childStorageKey, prefix types.StorageKey, blockHash types.Hash) ([]types.StorageKey, error)
	GetChildKeysLatest(childStorageKey, prefix types.StorageKey) ([]types.StorageKey, error)

	SubscribeRuntimeVersion() (*RuntimeVersionSubscription, error)

	QueryStorage(keys []types.StorageKey, startBlock types.Hash, block types.Hash) ([]types.StorageChangeSet, error)
	QueryStorageLatest(keys []types.StorageKey, startBlock types.Hash) ([]types.StorageChangeSet, error)

	QueryStorageAt(keys []types.StorageKey, block types.Hash) ([]types.StorageChangeSet, error)
	QueryStorageAtLatest(keys []types.StorageKey) ([]types.StorageChangeSet, error)

	GetKeys(prefix types.StorageKey, blockHash types.Hash) ([]types.StorageKey, error)
	GetKeysLatest(prefix types.StorageKey) ([]types.StorageKey, error)

	GetStorageSize(key types.StorageKey, blockHash types.Hash) (types.U64, error)
	GetStorageSizeLatest(key types.StorageKey) (types.U64, error)

	GetChildStorageHash(childStorageKey, key types.StorageKey, blockHash types.Hash) (types.Hash, error)
	GetChildStorageHashLatest(childStorageKey, key types.StorageKey) (types.Hash, error)
}

// state exposes methods for querying state
type state struct {
	client client.Client
}

// NewState creates a new state struct
func NewState(c client.Client) State {
	return &state{client: c}
}
