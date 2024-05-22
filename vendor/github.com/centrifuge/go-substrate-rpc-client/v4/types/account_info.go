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

package types

// Deprecated: AccountInfoV4 is an account information structure for contracts
type AccountInfoV4 struct {
	TrieID           []byte
	CurrentMemStored uint64
}

// Deprecated: NewAccountInfoV4 creates a new AccountInfoV4 type
func NewAccountInfoV4(trieID []byte, currentMemStored uint64) AccountInfoV4 {
	return AccountInfoV4{trieID, currentMemStored}
}
