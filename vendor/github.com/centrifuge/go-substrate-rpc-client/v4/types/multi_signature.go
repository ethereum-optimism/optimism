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

import "github.com/centrifuge/go-substrate-rpc-client/v4/scale"

// MultiSignature
type MultiSignature struct {
	IsEd25519 bool           // 0:: Ed25519(Ed25519Signature)
	AsEd25519 Signature      // Ed25519Signature
	IsSr25519 bool           // 1:: Sr25519(Sr25519Signature)
	AsSr25519 Signature      // Sr25519Signature
	IsEcdsa   bool           // 2:: Ecdsa(EcdsaSignature)
	AsEcdsa   EcdsaSignature // EcdsaSignature
}

func (m *MultiSignature) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		m.IsEd25519 = true
		err = decoder.Decode(&m.AsEd25519)
	case 1:
		m.IsSr25519 = true
		err = decoder.Decode(&m.AsSr25519)
	case 2:
		m.IsEcdsa = true
		err = decoder.Decode(&m.AsEcdsa)
	}

	if err != nil {
		return err
	}

	return nil
}

func (m MultiSignature) Encode(encoder scale.Encoder) error {
	var err1, err2 error
	switch {
	case m.IsEd25519:
		err1 = encoder.PushByte(0)
		err2 = encoder.Encode(m.AsEd25519)
	case m.IsSr25519:
		err1 = encoder.PushByte(1)
		err2 = encoder.Encode(m.AsSr25519)
	case m.IsEcdsa:
		err1 = encoder.PushByte(2)
		err2 = encoder.Encode(m.AsEcdsa)
	}

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}

	return nil
}
