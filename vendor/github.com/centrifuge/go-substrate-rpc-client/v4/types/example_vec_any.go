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

// ExampleVecAny - VecAny is used in polkadot-js as a list of elements that are of any type, while Vec and VecFixed
// require fixed types.
// Albeit Go has no dynamic types, VecAny can be implemented using arrays/slices of custom types with custom encoding.
// An example is provided here.
// The ExampleVecAny type itself is not used anywhere, it's just here for documentation purposes.
type ExampleVecAny struct{}
