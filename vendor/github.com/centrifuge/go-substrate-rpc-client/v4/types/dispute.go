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

type DisputeLocation struct {
	IsLocal bool

	IsRemote bool
}

func (d *DisputeLocation) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()

	if err != nil {
		return err
	}

	switch b {
	case 0:
		d.IsLocal = true
	case 1:
		d.IsRemote = true
	}

	return nil
}

func (d DisputeLocation) Encode(encoder scale.Encoder) error {
	switch {
	case d.IsLocal:
		return encoder.PushByte(0)
	case d.IsRemote:
		return encoder.PushByte(1)
	}

	return nil
}

type DisputeResult struct {
	IsValid bool

	IsInvalid bool
}

func (d *DisputeResult) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()

	if err != nil {
		return err
	}

	switch b {
	case 0:
		d.IsValid = true
	case 1:
		d.IsInvalid = true
	}

	return nil
}

func (d DisputeResult) Encode(encoder scale.Encoder) error {
	switch {
	case d.IsValid:
		return encoder.PushByte(0)
	case d.IsInvalid:
		return encoder.PushByte(1)
	}

	return nil
}
