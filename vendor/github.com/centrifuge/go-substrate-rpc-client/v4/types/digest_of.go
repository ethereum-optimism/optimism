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

	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
)

// DigestOf contains logs
type DigestOf []DigestItem

// UnmarshalJSON fills u with the JSON encoded byte array given by b
func (d *DigestOf) UnmarshalJSON(bz []byte) error {
	var tmp struct {
		Logs []string `json:"logs"`
	}
	if err := json.Unmarshal(bz, &tmp); err != nil {
		return err
	}
	*d = make([]DigestItem, len(tmp.Logs))
	for i, log := range tmp.Logs {
		err := codec.DecodeFromHex(log, &(*d)[i])
		if err != nil {
			return err
		}
	}
	return nil
}

// MarshalJSON returns a JSON encoded byte array of u
func (d DigestOf) MarshalJSON() ([]byte, error) {
	logs := make([]string, len(d))
	var err error
	for i, di := range d {
		logs[i], err = codec.EncodeToHex(di)
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(struct {
		Logs []string `json:"logs"`
	}{
		Logs: logs,
	})
}
