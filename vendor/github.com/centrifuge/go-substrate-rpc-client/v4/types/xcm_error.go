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

type XCMError struct {
	IsOverflow bool

	IsUnimplemented bool

	IsUntrustedReserveLocation bool

	IsUntrustedTeleportLocation bool

	IsMultiLocationFull bool

	IsMultiLocationNotInvertible bool

	IsBadOrigin bool

	IsInvalidLocation bool

	IsAssetNotFound bool

	IsFailedToTransactAsset bool

	IsNotWithdrawable bool

	IsLocationCannotHold bool

	IsExceedsMaxMessageSize bool

	IsDestinationUnsupported bool

	IsTransport bool
	Transport   string

	IsUnroutable bool

	IsUnknownClaim bool

	IsFailedToDecode bool

	IsMaxWeightInvalid bool

	IsNotHoldingFees bool

	IsTooExpensive bool

	IsTrap   bool
	TrapCode U64

	IsUnhandledXcmVersion bool

	IsWeightLimitReached bool
	Weight               Weight

	IsBarrier bool

	IsWeightNotComputable bool
}

func (x *XCMError) Decode(decoder scale.Decoder) error { //nolint: funlen
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		x.IsOverflow = true
	case 1:
		x.IsUnimplemented = true
	case 2:
		x.IsUntrustedReserveLocation = true
	case 3:
		x.IsUntrustedTeleportLocation = true
	case 4:
		x.IsMultiLocationFull = true
	case 5:
		x.IsMultiLocationNotInvertible = true
	case 6:
		x.IsBadOrigin = true
	case 7:
		x.IsInvalidLocation = true
	case 8:
		x.IsAssetNotFound = true
	case 9:
		x.IsFailedToTransactAsset = true
	case 10:
		x.IsNotWithdrawable = true
	case 11:
		x.IsLocationCannotHold = true
	case 12:
		x.IsExceedsMaxMessageSize = true
	case 13:
		x.IsDestinationUnsupported = true
	case 14:
		x.IsTransport = true

		return decoder.Decode(&x.Transport)
	case 15:
		x.IsUnroutable = true
	case 16:
		x.IsUnknownClaim = true
	case 17:
		x.IsFailedToDecode = true
	case 18:
		x.IsMaxWeightInvalid = true
	case 19:
		x.IsNotHoldingFees = true
	case 20:
		x.IsTooExpensive = true
	case 21:
		x.IsTrap = true

		return decoder.Decode(&x.TrapCode)
	case 22:
		x.IsUnhandledXcmVersion = true
	case 23:
		x.IsWeightLimitReached = true

		return decoder.Decode(&x.Weight)
	case 24:
		x.IsBarrier = true
	case 25:
		x.IsWeightNotComputable = true
	}

	return nil
}

func (x XCMError) Encode(encoder scale.Encoder) error { //nolint:gocyclo,funlen
	switch {
	case x.IsOverflow:
		return encoder.PushByte(0)
	case x.IsUnimplemented:
		return encoder.PushByte(1)
	case x.IsUntrustedReserveLocation:
		return encoder.PushByte(2)
	case x.IsUntrustedTeleportLocation:
		return encoder.PushByte(3)
	case x.IsMultiLocationFull:
		return encoder.PushByte(4)
	case x.IsMultiLocationNotInvertible:
		return encoder.PushByte(5)
	case x.IsBadOrigin:
		return encoder.PushByte(6)
	case x.IsInvalidLocation:
		return encoder.PushByte(7)
	case x.IsAssetNotFound:
		return encoder.PushByte(8)
	case x.IsFailedToTransactAsset:
		return encoder.PushByte(9)
	case x.IsNotWithdrawable:
		return encoder.PushByte(10)
	case x.IsLocationCannotHold:
		return encoder.PushByte(11)
	case x.IsExceedsMaxMessageSize:
		return encoder.PushByte(12)
	case x.IsDestinationUnsupported:
		return encoder.PushByte(13)
	case x.IsTransport:
		if err := encoder.PushByte(14); err != nil {
			return err
		}

		return encoder.Encode(x.Transport)
	case x.IsUnroutable:
		return encoder.PushByte(15)
	case x.IsUnknownClaim:
		return encoder.PushByte(16)
	case x.IsFailedToDecode:
		return encoder.PushByte(17)
	case x.IsMaxWeightInvalid:
		return encoder.PushByte(18)
	case x.IsNotHoldingFees:
		return encoder.PushByte(19)
	case x.IsTooExpensive:
		return encoder.PushByte(20)
	case x.IsTrap:
		if err := encoder.PushByte(21); err != nil {
			return err
		}

		return encoder.Encode(x.TrapCode)
	case x.IsUnhandledXcmVersion:
		return encoder.PushByte(22)
	case x.IsWeightLimitReached:
		if err := encoder.PushByte(23); err != nil {
			return err
		}

		return encoder.Encode(x.Weight)
	case x.IsBarrier:
		return encoder.PushByte(24)
	case x.IsWeightNotComputable:
		return encoder.PushByte(25)
	}

	return nil
}
