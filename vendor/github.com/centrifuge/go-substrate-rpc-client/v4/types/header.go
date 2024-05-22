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
	"encoding/json"
	"math/big"
	"strconv"
	"strings"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
)

type Tuple struct {
	Start  U32
	Offset U32
}
type AppId UCompact

type DataLookupIndexItem struct {
	AppId AppId    `json:"appId"`
	Start UCompact `json:"start"`
}
type DataLookup struct {
	Size  UCompact              `json:"size"`
	Index []DataLookupIndexItem `json:"index"`
}

type KateCommitment struct {
	Rows       UCompact `json:"rows"`
	Cols       UCompact `json:"cols"`
	DataRoot   Hash     `json:"dataRoot"`
	Commitment []U8     `json:"commitment"`
}

type V1HeaderExtension struct {
	Commitment KateCommitment `json:"commitment"`
	AppLookup  DataLookup     `json:"appLookup"`
}
type VTHeaderExtension struct {
	NewField   []U8           `json:"newField"`
	Commitment KateCommitment `json:"commitment"`
	AppLookup  DataLookup     `json:"appLookup"`
}

type HeaderExtensionEnum struct {
	V1    V1HeaderExtension `json:"V1"`
	VTest VTHeaderExtension `json:"VTest"`
}

type HeaderExtension struct {
	Enum HeaderExtensionEnum `json:"HeaderExtension"`
}

type Header struct {
	ParentHash     Hash            `json:"parentHash"`
	Number         BlockNumber     `json:"number"`
	StateRoot      Hash            `json:"stateRoot"`
	ExtrinsicsRoot Hash            `json:"extrinsicsRoot"`
	Digest         Digest          `json:"digest"`
	Extension      HeaderExtension `json:"extension"`
}

type BlockNumber U32

// UnmarshalJSON fills BlockNumber with the JSON encoded byte array given by bz
func (b *BlockNumber) UnmarshalJSON(bz []byte) error {
	var tmp string
	if err := json.Unmarshal(bz, &tmp); err != nil {
		return err
	}

	s := strings.TrimPrefix(tmp, "0x")

	p, err := strconv.ParseUint(s, 16, 32)
	*b = BlockNumber(p)
	return err
}

// MarshalJSON returns a JSON encoded byte array of BlockNumber
func (b BlockNumber) MarshalJSON() ([]byte, error) {
	s := strconv.FormatUint(uint64(b), 16)
	return json.Marshal(s)
}

// Encode implements encoding for BlockNumber, which just unwraps the bytes of BlockNumber
func (b BlockNumber) Encode(encoder scale.Encoder) error {
	return encoder.EncodeUintCompact(*big.NewInt(0).SetUint64(uint64(b)))
}

// Decode implements decoding for BlockNumber, which just wraps the bytes in BlockNumber
func (b *BlockNumber) Decode(decoder scale.Decoder) error {
	u, err := decoder.DecodeUintCompact()
	if err != nil {
		return err
	}
	*b = BlockNumber(u.Uint64())
	return err
}

func (a AppId) Decode(decoder scale.Decoder) error {
	u := UCompact(a)
	return u.Decode(decoder)
}

func (a AppId) Encode(encoder scale.Encoder) error {
	u := UCompact(a)
	return u.Encode(encoder)
}
