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
	"fmt"
	"strings"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
)

// ExtrinsicStatus is an enum containing the result of an extrinsic submission
type ExtrinsicStatus struct {
	IsFuture          bool // 00:: Future
	IsReady           bool // 1:: Ready
	IsBroadcast       bool // 2:: Broadcast(Vec<Text>)
	AsBroadcast       []Text
	IsInBlock         bool // 3:: InBlock(BlockHash)
	AsInBlock         Hash
	IsRetracted       bool // 4:: Retracted(BlockHash)
	AsRetracted       Hash
	IsFinalityTimeout bool // 5:: FinalityTimeout(BlockHash)
	AsFinalityTimeout Hash
	IsFinalized       bool // 6:: Finalized(BlockHash)
	AsFinalized       Hash
	IsUsurped         bool // 7:: Usurped(Hash)
	AsUsurped         Hash
	IsDropped         bool // 8:: Dropped
	IsInvalid         bool // 9:: Invalid
}

func (e *ExtrinsicStatus) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()

	if err != nil {
		return err
	}

	switch b {
	case 0:
		e.IsFuture = true
	case 1:
		e.IsReady = true
	case 2:
		e.IsBroadcast = true
		err = decoder.Decode(&e.AsBroadcast)
	case 3:
		e.IsInBlock = true
		err = decoder.Decode(&e.AsInBlock)
	case 4:
		e.IsRetracted = true
		err = decoder.Decode(&e.AsRetracted)
	case 5:
		e.IsFinalityTimeout = true
		err = decoder.Decode(&e.AsFinalityTimeout)
	case 6:
		e.IsFinalized = true
		err = decoder.Decode(&e.AsFinalized)
	case 7:
		e.IsUsurped = true
		err = decoder.Decode(&e.AsUsurped)
	case 8:
		e.IsDropped = true
	case 9:
		e.IsInvalid = true
	}

	if err != nil {
		return err
	}

	return nil
}

func (e ExtrinsicStatus) Encode(encoder scale.Encoder) error {
	var err1, err2 error
	switch {
	case e.IsFuture:
		err1 = encoder.PushByte(0)
	case e.IsReady:
		err1 = encoder.PushByte(1)
	case e.IsBroadcast:
		err1 = encoder.PushByte(2)
		err2 = encoder.Encode(e.AsBroadcast)
	case e.IsInBlock:
		err1 = encoder.PushByte(3)
		err2 = encoder.Encode(e.AsInBlock)
	case e.IsRetracted:
		err1 = encoder.PushByte(4)
		err2 = encoder.Encode(e.AsRetracted)
	case e.IsFinalityTimeout:
		err1 = encoder.PushByte(5)
		err2 = encoder.Encode(e.AsFinalityTimeout)
	case e.IsFinalized:
		err1 = encoder.PushByte(6)
		err2 = encoder.Encode(e.AsFinalized)
	case e.IsUsurped:
		err1 = encoder.PushByte(7)
		err2 = encoder.Encode(e.AsUsurped)
	case e.IsDropped:
		err1 = encoder.PushByte(8)
	case e.IsInvalid:
		err1 = encoder.PushByte(9)
	}

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}

	return nil
}

func (e *ExtrinsicStatus) UnmarshalJSON(b []byte) error { //nolint:funlen
	input := strings.TrimSpace(string(b))
	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		input = input[1 : len(input)-1]
	}

	switch {
	case input == "future":
		e.IsFuture = true
		return nil
	case input == "ready":
		e.IsReady = true
		return nil
	case input == "dropped":
		e.IsDropped = true
		return nil
	case input == "invalid":
		e.IsInvalid = true
		return nil
	}

	// no simple case, decode into helper
	var tmp struct {
		AsBroadcast       []Text `json:"broadcast"`
		AsInBlock         Hash   `json:"inBlock"`
		AsRetracted       Hash   `json:"retracted"`
		AsFinalityTimeout Hash   `json:"finalityTimeout"`
		AsFinalized       Hash   `json:"finalized"`
		AsUsurped         Hash   `json:"usurped"`
	}
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	switch {
	case strings.HasPrefix(input, "{\"broadcast\""):
		e.IsBroadcast = true
		e.AsBroadcast = tmp.AsBroadcast
		return nil
	case strings.HasPrefix(input, "{\"inBlock\""):
		e.IsInBlock = true
		e.AsInBlock = tmp.AsInBlock
		return nil
	case strings.HasPrefix(input, "{\"retracted\""):
		e.IsRetracted = true
		e.AsRetracted = tmp.AsRetracted
		return nil
	case strings.HasPrefix(input, "{\"finalityTimeout\""):
		e.IsFinalityTimeout = true
		e.AsFinalityTimeout = tmp.AsFinalityTimeout
		return nil
	case strings.HasPrefix(input, "{\"finalized\""):
		e.IsFinalized = true
		e.AsFinalized = tmp.AsFinalized
		return nil
	case strings.HasPrefix(input, "{\"usurped\""):
		e.IsUsurped = true
		e.AsUsurped = tmp.AsUsurped
		return nil
	}

	return fmt.Errorf("unexpected JSON for ExtrinsicStatus, got %v", string(b))
}

func (e ExtrinsicStatus) MarshalJSON() ([]byte, error) {
	switch {
	case e.IsFuture:
		return []byte("\"future\""), nil
	case e.IsReady:
		return []byte("\"ready\""), nil
	case e.IsDropped:
		return []byte("\"dropped\""), nil
	case e.IsInvalid:
		return []byte("\"invalid\""), nil
	case e.IsBroadcast:
		var tmp struct {
			AsBroadcast []Text `json:"broadcast"`
		}
		tmp.AsBroadcast = e.AsBroadcast
		return json.Marshal(tmp)
	case e.IsInBlock:
		var tmp struct {
			AsInBlock Hash `json:"inBlock"`
		}
		tmp.AsInBlock = e.AsInBlock
		return json.Marshal(tmp)
	case e.IsRetracted:
		var tmp struct {
			AsRetracted Hash `json:"retracted"`
		}
		tmp.AsRetracted = e.AsRetracted
		return json.Marshal(tmp)
	case e.IsFinalityTimeout:
		var tmp struct {
			AsFinalityTimeout Hash `json:"finalityTimeout"`
		}
		tmp.AsFinalityTimeout = e.AsFinalityTimeout
		return json.Marshal(tmp)
	case e.IsFinalized:
		var tmp struct {
			AsFinalized Hash `json:"finalized"`
		}
		tmp.AsFinalized = e.AsFinalized
		return json.Marshal(tmp)
	case e.IsUsurped:
		var tmp struct {
			AsUsurped Hash `json:"usurped"`
		}
		tmp.AsUsurped = e.AsUsurped
		return json.Marshal(tmp)
	}
	return nil, fmt.Errorf("cannot marshal ExtrinsicStatus, got %#v", e)
}
