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

/*
Package gsrpc (Go Substrate RPC Client) provides APIs and types around Polkadot and any Substrate-based chain RPC calls.
This client is modelled after [polkadot-js/api](https://github.com/polkadot-js/api).

Calling RPC methods

Simply instantiate the gsrpc with a URL of your choice, e. g.

	api, err := gsrpc.NewSubstrateAPI("wss://poc3-rpc.polkadot.io")

and run any of the provided RPC methods from the api:

	hash, err := api.RPC.Chain.GetBlockHashLatest()

Further examples can be found below.

Signing extrinsics

In order to sign extrinsics, you need to have [subkey](https://github.com/paritytech/substrate/tree/master/subkey) installed. Please make sure that you use subkey in the version of your relay chain.

Types

The package [types](https://godoc.org/github.com/centrifuge/go-substrate-rpc-client/v4/types/) exports a number
of useful basic types including functions for encoding and decoding them.

To use your own custom types, you can simply create structs and arrays composing those basic types. Here are some
examples using composition of a mix of these basic and builtin Go types:

1. Vectors, lists, series, sets, arrays, slices: https://godoc.org/github.com/centrifuge/go-substrate-rpc-client/v4/types/#example_Vec_simple

2. Structs: https://godoc.org/github.com/centrifuge/go-substrate-rpc-client/v4/types/#example_Struct_simple

There are some caveats though that you should be aware of:

1. The order of the values in your structs is of relevance to the encoding. The scale codec Substrate/Polkadot
uses does not encode labels/keys.

2. Some types do not have corresponding types in Go. Working with them requires a custom struct with Encoding/Decoding
methods that implement the Encodeable/Decodeable interfaces. Examples for that are enums, tuples and vectors with any
types, you can find reference implementations of those here: types/enum_test.go , types/tuple_test.go and
types/vec_any_test.go

For more information about the types sub-package, see https://godoc.org/github.com/centrifuge/go-substrate-rpc-client/v4/types
*/
package gsrpc
