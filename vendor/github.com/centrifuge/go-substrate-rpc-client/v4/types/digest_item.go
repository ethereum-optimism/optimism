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
)

// DigestItem specifies the item in the logs of a digest
type DigestItem struct {
	IsChangesTrieRoot   bool // 2
	AsChangesTrieRoot   Hash
	IsPreRuntime        bool // 6
	AsPreRuntime        PreRuntime
	IsConsensus         bool // 4
	AsConsensus         Consensus
	IsSeal              bool // 5
	AsSeal              Seal
	IsChangesTrieSignal bool // 7
	AsChangesTrieSignal ChangesTrieSignal
	IsOther             bool // 0
	AsOther             Bytes
}

func (m *DigestItem) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()

	if err != nil {
		return err
	}

	switch b {
	case 2:
		m.IsChangesTrieRoot = true
		err = decoder.Decode(&m.AsChangesTrieRoot)
	case 6:
		m.IsPreRuntime = true
		err = decoder.Decode(&m.AsPreRuntime)
	case 4:
		m.IsConsensus = true
		err = decoder.Decode(&m.AsConsensus)
	case 5:
		m.IsSeal = true
		err = decoder.Decode(&m.AsSeal)
	case 7:
		m.IsChangesTrieSignal = true
		err = decoder.Decode(&m.AsChangesTrieSignal)
	case 0:
		m.IsOther = true
		err = decoder.Decode(&m.AsOther)
	}

	if err != nil {
		return err
	}

	return nil
}

func (m DigestItem) Encode(encoder scale.Encoder) error {
	var err1, err2 error
	switch {
	case m.IsOther:
		err1 = encoder.PushByte(0)
		err2 = encoder.Encode(m.AsOther)
	case m.IsChangesTrieRoot:
		err1 = encoder.PushByte(2)
		err2 = encoder.Encode(m.AsChangesTrieRoot)
	case m.IsConsensus:
		err1 = encoder.PushByte(4)
		err2 = encoder.Encode(m.AsConsensus)
	case m.IsSeal:
		err1 = encoder.PushByte(5)
		err2 = encoder.Encode(m.AsSeal)
	case m.IsPreRuntime:
		err1 = encoder.PushByte(6)
		err2 = encoder.Encode(m.AsPreRuntime)
	case m.IsChangesTrieSignal:
		err1 = encoder.PushByte(7)
		err2 = encoder.Encode(m.AsChangesTrieSignal)
	}

	if err1 != nil {
		return err1
	}

	if err2 != nil {
		return err2
	}

	return nil
}

// AuthorityID represents a public key (an 32 byte array)
type AuthorityID [32]byte

// NewAuthorityID creates a new AuthorityID type
func NewAuthorityID(b [32]byte) AuthorityID {
	return AuthorityID(b)
}

type Seal struct {
	ConsensusEngineID ConsensusEngineID
	Bytes             Bytes
}

// ConsensusEngineID is a 4-byte identifier (actually a [u8; 4]) identifying the engine, e.g. for Aura it would be
// [b'a', b'u', b'r', b'a']
type ConsensusEngineID U32

type Consensus struct {
	ConsensusEngineID ConsensusEngineID
	Bytes             Bytes
}

type PreRuntime struct {
	ConsensusEngineID ConsensusEngineID
	Bytes             Bytes
}

type ChangesTrieSignal struct {
	IsNewConfiguration bool
	AsNewConfiguration Bytes
}

func (c ChangesTrieSignal) Encode(encoder scale.Encoder) error {
	switch {
	case c.IsNewConfiguration:
		err := encoder.PushByte(0)
		if err != nil {
			return err
		}
		err = encoder.Encode(c.AsNewConfiguration)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("no such variant for ChangesTrieSignal")
	}

	return nil
}

func (c *ChangesTrieSignal) Decode(decoder scale.Decoder) error {
	tag, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch tag {
	case 0:
		c.IsNewConfiguration = true
		err = decoder.Decode(&c.AsNewConfiguration)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("no such variant for ChangesTrieSignal")
	}

	return nil
}

type ChangesTrieConfiguration struct {
	DigestInterval U32
	DigestLevels   U32
}
