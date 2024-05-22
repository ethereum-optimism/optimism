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

// Weight is a numeric range of a transaction weight
type Weight struct {
	// The weight of computational time used based on some reference hardware.
	RefTime UCompact
	// The weight of storage space used by proof of validity.
	ProofSize UCompact
}

// NewWeight creates a new Weight type
func NewWeight(refTime UCompact, proofSize UCompact) Weight {
	return Weight{
		RefTime:   refTime,
		ProofSize: proofSize,
	}
}
