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

import (
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
)

// ExtrinsicPayloadV3 is a signing payload for an Extrinsic. For the final encoding, it is variable length based on
// the contents included. Note that `BytesBare` is absolutely critical – we don't want the method (Bytes)
// to have the length prefix included. This means that the data-as-signed is un-decodable,
// but is also doesn't need the extra information, only the pure data (and is not decoded)
// ... The same applies to V1 & V1, if we have a V4, carry move this comment to latest
type ExtrinsicPayloadV3 struct {
	Method      BytesBare
	Era         ExtrinsicEra // extra via system::CheckEra
	Nonce       UCompact     // extra via system::CheckNonce (Compact<Index> where Index is u32)
	Tip         UCompact     // extra via balances::TakeFees (Compact<Balance> where Balance is u128)
	SpecVersion U32          // additional via system::CheckVersion
	GenesisHash Hash         // additional via system::CheckGenesis
	BlockHash   Hash         // additional via system::CheckEra
}

// Sign the extrinsic payload with the given derivation path
func (e ExtrinsicPayloadV3) Sign(signer signature.KeyringPair) (Signature, error) {
	b, err := codec.Encode(e)
	if err != nil {
		return Signature{}, err
	}

	sig, err := signature.Sign(b, signer.URI)
	return NewSignature(sig), err
}

// Encode implements encoding for ExtrinsicPayloadV3, which just unwraps the bytes of ExtrinsicPayloadV3 without
// adding a compact length prefix
func (e ExtrinsicPayloadV3) Encode(encoder scale.Encoder) error {
	err := encoder.Encode(e.Method)
	if err != nil {
		return err
	}

	err = encoder.Encode(e.Era)
	if err != nil {
		return err
	}

	err = encoder.Encode(e.Nonce)
	if err != nil {
		return err
	}

	err = encoder.Encode(e.Tip)
	if err != nil {
		return err
	}

	err = encoder.Encode(e.SpecVersion)
	if err != nil {
		return err
	}

	err = encoder.Encode(e.GenesisHash)
	if err != nil {
		return err
	}

	err = encoder.Encode(e.BlockHash)
	if err != nil {
		return err
	}

	return nil
}

// Decode does nothing and always returns an error. ExtrinsicPayloadV3 is only used for encoding, not for decoding
func (e *ExtrinsicPayloadV3) Decode(decoder scale.Decoder) error {
	return fmt.Errorf("decoding of ExtrinsicPayloadV3 is not supported")
}

type ExtrinsicPayloadV4 struct {
	ExtrinsicPayloadV3
	TransactionVersion U32
	AppID              UCompact
}

// Sign the extrinsic payload with the given derivation path
func (e ExtrinsicPayloadV4) Sign(signer signature.KeyringPair) (Signature, error) {
	b, err := codec.Encode(e)
	if err != nil {
		return Signature{}, err
	}

	sig, err := signature.Sign(b, signer.URI)
	return NewSignature(sig), err
}

func (e ExtrinsicPayloadV4) Encode(encoder scale.Encoder) error {
	err := encoder.Encode(e.Method)
	if err != nil {
		return err
	}

	err = encoder.Encode(e.Era)
	if err != nil {
		return err
	}

	err = encoder.Encode(e.Nonce)
	if err != nil {
		return err
	}

	err = encoder.Encode(e.Tip)
	if err != nil {
		return err
	}

	err = encoder.Encode(e.AppID)
	if err != nil {
		return err
	}

	err = encoder.Encode(e.SpecVersion)
	if err != nil {
		return err
	}

	err = encoder.Encode(e.TransactionVersion)
	if err != nil {
		return err
	}

	err = encoder.Encode(e.GenesisHash)
	if err != nil {
		return err
	}

	err = encoder.Encode(e.BlockHash)
	if err != nil {
		return err
	}

	return nil
}

// Decode does nothing and always returns an error. ExtrinsicPayloadV4 is only used for encoding, not for decoding
func (e *ExtrinsicPayloadV4) Decode(decoder scale.Decoder) error {
	return fmt.Errorf("decoding of ExtrinsicPayloadV4 is not supported")
}
