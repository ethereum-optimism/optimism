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

//go:generate mockery --name Chain --filename chain.go

package chain

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/client"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Chain interface {
	SubscribeFinalizedHeads() (*FinalizedHeadsSubscription, error)
	SubscribeNewHeads() (*NewHeadsSubscription, error)
	GetBlockHash(blockNumber uint64) (types.Hash, error)
	GetBlockHashLatest() (types.Hash, error)
	GetFinalizedHead() (types.Hash, error)
	GetBlock(blockHash types.Hash) (*types.SignedBlock, error)
	GetBlockLatest() (*types.SignedBlock, error)
	GetHeader(blockHash types.Hash) (*types.Header, error)
	GetHeaderLatest() (*types.Header, error)
}

// chain exposes methods for retrieval of chain data
type chain struct {
	client client.Client
}

// NewChain creates a new chain struct
func NewChain(cl client.Client) Chain {
	return &chain{cl}
}
