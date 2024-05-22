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

// PostDispatchInfo is used in DispatchResultWithPostInfo.
// Weight information that is only available post dispatch.
type PostDispatchInfo struct {
	ActualWeight Option[Weight]
	PaysFee      Pays
}

func (p *PostDispatchInfo) Decode(decoder scale.Decoder) error {
	if err := decoder.Decode(&p.ActualWeight); err != nil {
		return err
	}

	return decoder.Decode(&p.PaysFee)
}

func (p PostDispatchInfo) Encode(encoder scale.Encoder) error {
	if err := encoder.Encode(p.ActualWeight); err != nil {
		return err
	}

	return encoder.Encode(p.PaysFee)
}

// DispatchErrorWithPostInfo is used in DispatchResultWithPostInfo.
type DispatchErrorWithPostInfo struct {
	PostInfo PostDispatchInfo
	Error    DispatchError
}

func (d *DispatchErrorWithPostInfo) Decode(decoder scale.Decoder) error {
	if err := decoder.Decode(&d.PostInfo); err != nil {
		return err
	}

	return decoder.Decode(&d.Error)
}

func (d DispatchErrorWithPostInfo) Encode(encoder scale.Encoder) error {
	if err := encoder.Encode(d.PostInfo); err != nil {
		return err
	}

	return encoder.Encode(d.Error)
}

// DispatchResultWithPostInfo can be returned from dispatch able functions.
type DispatchResultWithPostInfo struct {
	IsOk bool
	Ok   PostDispatchInfo

	IsError bool
	Error   DispatchErrorWithPostInfo
}

func (d *DispatchResultWithPostInfo) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		d.IsOk = true

		return decoder.Decode(&d.Ok)
	case 1:
		d.IsError = true

		return decoder.Decode(&d.Error)
	}

	return nil
}

func (d DispatchResultWithPostInfo) Encode(encoder scale.Encoder) error {
	switch {
	case d.IsOk:
		if err := encoder.PushByte(0); err != nil {
			return err
		}

		return encoder.Encode(d.Ok)
	case d.IsError:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		return encoder.Encode(d.Error)
	}

	return nil
}
