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
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type OptionAccountID struct {
	option
	value AccountID
}

func NewOptionAccountID(value AccountID) OptionAccountID {
	return OptionAccountID{option{hasValue: true}, value}
}

func NewOptionAccountIDEmpty() OptionAccountID {
	return OptionAccountID{option: option{hasValue: false}}
}

func (o *OptionAccountID) Decode(decoder scale.Decoder) error {
	return decoder.DecodeOption(&o.hasValue, &o.value)
}

func (o OptionAccountID) Encode(encoder scale.Encoder) error {
	return encoder.EncodeOption(o.hasValue, o.value)
}

// SetSome sets a value
func (o *OptionAccountID) SetSome(value AccountID) {
	o.hasValue = true
	o.value = value
}

// SetNone removes a value and marks it as missing
func (o *OptionAccountID) SetNone() {
	o.hasValue = false
	o.value = AccountID{}
}

// Unwrap returns a flag that indicates whether a value is present and the stored value
func (o *OptionAccountID) Unwrap() (ok bool, value AccountID) {
	return o.hasValue, o.value
}

const (
	AccountIDLen = 32
)

// AccountID represents a public key (an 32 byte array)
type AccountID [AccountIDLen]byte

func (a *AccountID) ToBytes() []byte {
	if a == nil {
		return nil
	}

	b := a[:]

	return b
}

func (a *AccountID) ToHexString() string {
	if a == nil {
		return ""
	}

	return hexutil.Encode(a.ToBytes())
}

func (a *AccountID) Equal(accountID *AccountID) bool {
	return bytes.Equal(a.ToBytes(), accountID.ToBytes())
}

func (a AccountID) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.ToHexString())
}

func (a *AccountID) UnmarshalJSON(data []byte) error {
	accID, err := NewAccountIDFromHexString(strings.Trim(string(data), "\""))

	if err != nil {
		return err
	}

	*a = *accID

	return nil
}

var (
	ErrInvalidAccountIDBytes = errors.New("invalid account ID bytes")
)

// NewAccountID creates a new AccountID type
func NewAccountID(b []byte) (*AccountID, error) {
	if len(b) != AccountIDLen {
		return nil, ErrInvalidAccountIDBytes
	}

	a := AccountID{}

	copy(a[:], b)

	return &a, nil
}

func NewAccountIDFromHexString(accountIDHex string) (*AccountID, error) {
	b, err := hexutil.Decode(accountIDHex)

	if err != nil {
		return nil, err
	}

	return NewAccountID(b)
}
