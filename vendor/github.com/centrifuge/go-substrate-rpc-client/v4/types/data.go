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
	"io"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
)

// Data is a raw data structure, containing raw bytes that are not decoded/encoded (without any length encoding).
// Be careful using this in your own structs – it only works as the last value in a struct since it will consume the
// remainder of the encoded data. The reason for this is that it does not contain any length encoding, so it would
// not know where to stop.
type Data []byte

// NewData creates a new Data type
func NewData(b []byte) Data {
	return Data(b)
}

// Encode implements encoding for Data, which just unwraps the bytes of Data
func (d Data) Encode(encoder scale.Encoder) error {
	return encoder.Write(d)
}

// Decode implements decoding for Data, which just reads all the remaining bytes into Data
func (d *Data) Decode(decoder scale.Decoder) error {
	for i := 0; true; i++ {
		b, err := decoder.ReadOneByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		*d = append((*d)[:i], b)
	}
	return nil
}

// Hex returns a hex string representation of the value
func (d Data) Hex() string {
	return fmt.Sprintf("%#x", d)
}
