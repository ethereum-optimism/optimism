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

//go:generate mockery --name Offchain --filename offchain.go

package offchain

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/client"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Offchain interface {
	LocalStorageGet(kind StorageKind, key []byte) (*types.StorageDataRaw, error)
	LocalStorageSet(kind StorageKind, key []byte, value []byte) error
}

// offchain exposes methods for retrieval of off-chain data
type offchain struct {
	client client.Client
}

// NewOffchain creates a new offchain struct
func NewOffchain(c client.Client) Offchain {
	return &offchain{client: c}
}
