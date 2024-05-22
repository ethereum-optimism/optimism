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

package chain

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

// GetFinalizedHead returns the hash of the last finalized block in the canon chain
func (c *chain) GetFinalizedHead() (types.Hash, error) {
	var res string

	err := c.client.Call(&res, "chain_getFinalizedHead")
	if err != nil {
		return types.Hash{}, err
	}

	return types.NewHashFromHexString(res)
}
