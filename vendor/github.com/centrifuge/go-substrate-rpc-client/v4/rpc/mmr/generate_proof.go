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

package mmr

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/client"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

// GenerateProof retrieves a MMR proof and leaf for the specified leave index, at the given blockHash (useful to query a
// proof at an earlier block, likely with antoher MMR root)
func (c *mmr) GenerateProof(leafIndex uint64, blockHash types.Hash) (types.GenerateMMRProofResponse, error) {
	return c.generateProof(leafIndex, &blockHash)
}

// GenerateProofLatest retrieves the latest MMR proof and leaf for the specified leave index
func (c *mmr) GenerateProofLatest(leafIndex uint64) (types.GenerateMMRProofResponse, error) {
	return c.generateProof(leafIndex, nil)
}

func (c *mmr) generateProof(leafIndex uint64, blockHash *types.Hash) (types.GenerateMMRProofResponse, error) {
	var res types.GenerateMMRProofResponse
	err := client.CallWithBlockHash(c.client, &res, "mmr_generateProof", blockHash, leafIndex)
	if err != nil {
		return types.GenerateMMRProofResponse{}, err
	}

	return res, nil
}
