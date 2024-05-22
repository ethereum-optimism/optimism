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
	"github.com/centrifuge/go-substrate-rpc-client/v4/client"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

// GetHeader retrieves the header for the specific block
func (c *chain) GetHeader(blockHash types.Hash) (*types.Header, error) {
	return c.getHeader(&blockHash)
}

// GetHeaderLatest retrieves the header of the latest block
func (c *chain) GetHeaderLatest() (*types.Header, error) {
	return c.getHeader(nil)
}

func (c *chain) getHeader(blockHash *types.Hash) (*types.Header, error) {
	var Header types.Header
	err := client.CallWithBlockHash(c.client, &Header, "chain_getHeader", blockHash)
	if err != nil {
		return nil, err
	}
	return &Header, err
}
