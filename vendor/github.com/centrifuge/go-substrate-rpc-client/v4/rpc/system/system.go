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

//go:generate mockery --name System --filename system.go

package system

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/client"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type System interface {
	Properties() (types.ChainProperties, error)
	Health() (types.Health, error)
	Peers() ([]types.PeerInfo, error)
	Name() (types.Text, error)
	Chain() (types.Text, error)
	Version() (types.Text, error)
	NetworkState() (types.NetworkState, error)
}

// system exposes methods for retrieval of system data
type system struct {
	client client.Client
}

// NewSystem creates a new system struct
func NewSystem(cl client.Client) System {
	return &system{cl}
}
