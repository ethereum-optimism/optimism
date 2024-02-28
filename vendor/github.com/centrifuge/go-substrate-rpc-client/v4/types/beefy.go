// Go Substrate RPC Client (GSRPC) provides APIs and types around Polkadot and any Substrate-based chain RPC calls
//
// Copyright 2019 Centrifuge GmbH
// Copyright 2021 Snowfork
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
	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
)

// PayloadItem ...
type PayloadItem struct {
	ID   [2]byte
	Data []byte
}

// Commitment is a beefy commitment
type Commitment struct {
	Payload        []PayloadItem
	BlockNumber    uint32
	ValidatorSetID uint64
}

// SignedCommitment is a beefy commitment with optional signatures from the set of validators
type SignedCommitment struct {
	Commitment Commitment
	Signatures []OptionBeefySignature
}

type OptionalSignedCommitment struct {
	option
	value SignedCommitment
}

type CompactSignedCommitment struct {
	Commitment        Commitment
	SignaturesFrom    []byte
	ValidatorSetLen   uint32
	SignaturesCompact []BeefySignature
}

// BeefySignature is a beefy signature
type BeefySignature [65]byte

// OptionBeefySignature is a structure that can store a BeefySignature or a missing value
type OptionBeefySignature struct {
	option
	value BeefySignature
}

// NewOptionBeefySignature creates an OptionBeefySignature with a value
func NewOptionBeefySignature(value BeefySignature) OptionBeefySignature {
	return OptionBeefySignature{option{true}, value}
}

// NewOptionBeefySignatureEmpty creates an OptionBeefySignature without a value
func NewOptionBeefySignatureEmpty() OptionBeefySignature {
	return OptionBeefySignature{option: option{false}}
}

func (o OptionBeefySignature) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionBeefySignature) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

// SetSome sets a value
func (o *OptionBeefySignature) SetSome(value BeefySignature) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionBeefySignature) SetNone() {
	o.hasValue = false
	o.value = BeefySignature{}
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o OptionBeefySignature) Unwrap() (ok bool, value BeefySignature) {
	return o.hasValue, o.value
}

// bits are packed into chunks of this size
const containerBitSize = 8

func (s *SignedCommitment) Decode(decoder scale.Decoder) error {
	compact := CompactSignedCommitment{}

	err := decoder.Decode(&compact)
	if err != nil {
		return err
	}

	var bits []byte

	for _, block := range compact.SignaturesFrom {
		for bit := 0; bit < containerBitSize; bit++ {
			bits = append(bits, (block>>(containerBitSize-bit-1))&1)
		}
	}

	bits = bits[0:compact.ValidatorSetLen]

	var signatures []OptionBeefySignature
	sigIndex := 0

	for _, bit := range bits {
		if bit == 1 {
			signatures = append(signatures, NewOptionBeefySignature(compact.SignaturesCompact[sigIndex]))
			sigIndex++
		} else {
			signatures = append(signatures, NewOptionBeefySignatureEmpty())
		}
	}

	s.Commitment = compact.Commitment
	s.Signatures = signatures

	return nil
}

func (s SignedCommitment) Encode(encoder scale.Encoder) error {
	var compact CompactSignedCommitment
	var bits []byte
	var signaturesFrom []byte
	var signaturesCompact []BeefySignature

	validatorSetLen := len(s.Signatures)

	for _, optionSig := range s.Signatures {
		if optionSig.IsSome() {
			bits = append(bits, 1)
			_, signature := optionSig.Unwrap()
			signaturesCompact = append(signaturesCompact, signature)
		} else {
			bits = append(bits, 0)
		}
	}

	excessBitsLen := containerBitSize - (validatorSetLen % containerBitSize)
	bits = append(bits, make([]byte, excessBitsLen)...)

	for _, chunk := range makeChunks(bits, containerBitSize) {
		acc := chunk[0]
		for i := 1; i < containerBitSize; i++ {
			acc <<= 1
			acc |= chunk[i]
		}
		signaturesFrom = append(signaturesFrom, acc)
	}

	compact.Commitment = s.Commitment
	compact.SignaturesCompact = signaturesCompact
	compact.SignaturesFrom = signaturesFrom
	compact.ValidatorSetLen = uint32(validatorSetLen)

	return encoder.Encode(compact)
}

func makeChunks(slice []byte, chunkSize int) [][]byte {
	var chunks [][]byte
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize

		// necessary check to avoid slicing beyond
		// slice capacity
		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
}

// UnmarshalText deserializes hex string into a SignedCommitment.
// Used for decoding JSON-RPC subscription messages (beefy_subscribeJustifications)
func (s *SignedCommitment) UnmarshalText(text []byte) error {
	return codec.DecodeFromHex(string(text), s)
}

func (o OptionalSignedCommitment) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

func (o *OptionalSignedCommitment) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

func (o OptionalSignedCommitment) Unwrap() (ok bool, value SignedCommitment) {
	return o.hasValue, o.value
}

func (o *OptionalSignedCommitment) SetSome(value SignedCommitment) {
	o.hasValue = true
	o.value = value
}

func (o *OptionalSignedCommitment) SetNone() {
	o.hasValue = false
	o.value = SignedCommitment{}
}
