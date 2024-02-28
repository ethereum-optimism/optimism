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

import "sync"

// SerDeOptions are serialise and deserialize options for types
type SerDeOptions struct {
	// NoPalletIndices enable this to work with substrate chains that do not have indices pallet in runtime
	NoPalletIndices bool
}

var defaultOptions = SerDeOptions{}
var mu sync.RWMutex

// SetSerDeOptions overrides default serialise and deserialize options
func SetSerDeOptions(so SerDeOptions) {
	defer mu.Unlock()
	mu.Lock()
	defaultOptions = so
}

// SerDeOptionsFromMetadata returns Serialise and deserialize options from metadata
func SerDeOptionsFromMetadata(meta *Metadata) SerDeOptions {
	var opts SerDeOptions
	if !meta.ExistsModuleMetadata("Indices") {
		opts.NoPalletIndices = true
	}
	return opts
}
