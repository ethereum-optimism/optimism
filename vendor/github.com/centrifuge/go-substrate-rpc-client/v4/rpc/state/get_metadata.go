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

// GetMetadata returns the metadata at the given block
func (s *state) GetMetadata(blockHash types.Hash) (*types.Metadata, error) {
	return s.getMetadata(&blockHash)
}

// GetMetadataLatest returns the latest metadata
func (s *state) GetMetadataLatest() (*types.Metadata, error) {
	return s.getMetadata(nil)
}

func (s *state) getMetadata(blockHash *types.Hash) (*types.Metadata, error) {
	var res string
	err := client.CallWithBlockHash(s.client, &res, "state_getMetadata", blockHash)
	if err != nil {
		return nil, err
	}

	var metadata types.Metadata
	err = codec.DecodeFromHex(res, &metadata)
	return &metadata, err
}
