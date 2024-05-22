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

type AssetID struct {
	IsConcrete    bool
	MultiLocation MultiLocationV1

	IsAbstract  bool
	AbstractKey []U8
}

func (a *AssetID) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		a.IsConcrete = true

		return decoder.Decode(&a.MultiLocation)
	case 1:
		a.IsAbstract = true

		return decoder.Decode(&a.AbstractKey)
	}

	return nil
}

func (a AssetID) Encode(encoder scale.Encoder) error {
	switch {
	case a.IsConcrete:
		if err := encoder.PushByte(0); err != nil {
			return err
		}

		return encoder.Encode(&a.MultiLocation)
	case a.IsAbstract:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		return encoder.Encode(&a.AbstractKey)
	}

	return nil
}

type AssetInstance struct {
	IsUndefined bool

	IsIndex bool
	Index   U128

	IsArray4 bool
	Array4   [4]U8

	IsArray8 bool
	Array8   [8]U8

	IsArray16 bool
	Array16   [16]U8

	IsArray32 bool
	Array32   [32]U8

	IsBlob bool
	Blob   []U8
}

func (a *AssetInstance) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		a.IsUndefined = true
	case 1:
		a.IsIndex = true

		return decoder.Decode(&a.Index)
	case 2:
		a.IsArray4 = true

		return decoder.Decode(&a.Array4)
	case 3:
		a.IsArray8 = true

		return decoder.Decode(&a.Array8)
	case 4:
		a.IsArray16 = true

		return decoder.Decode(&a.Array16)
	case 5:
		a.IsArray32 = true

		return decoder.Decode(&a.Array32)
	case 6:
		a.IsBlob = true

		return decoder.Decode(&a.Blob)
	}

	return nil
}

func (a AssetInstance) Encode(encoder scale.Encoder) error {
	switch {
	case a.IsUndefined:
		return encoder.PushByte(0)
	case a.IsIndex:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		return encoder.Encode(a.Index)
	case a.IsArray4:
		if err := encoder.PushByte(2); err != nil {
			return err
		}

		return encoder.Encode(a.Array4)
	case a.IsArray8:
		if err := encoder.PushByte(3); err != nil {
			return err
		}

		return encoder.Encode(a.Array8)
	case a.IsArray16:
		if err := encoder.PushByte(4); err != nil {
			return err
		}

		return encoder.Encode(a.Array16)
	case a.IsArray32:
		if err := encoder.PushByte(5); err != nil {
			return err
		}

		return encoder.Encode(a.Array32)
	case a.IsBlob:
		if err := encoder.PushByte(6); err != nil {
			return err
		}

		return encoder.Encode(a.Blob)
	}

	return nil
}

type Fungibility struct {
	IsFungible bool
	Amount     UCompact

	IsNonFungible bool
	AssetInstance AssetInstance
}

func (f *Fungibility) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		f.IsFungible = true

		return decoder.Decode(&f.Amount)
	case 1:
		f.IsNonFungible = true

		return decoder.Decode(&f.AssetInstance)
	}

	return nil
}

func (f Fungibility) Encode(encoder scale.Encoder) error {
	switch {
	case f.IsFungible:
		if err := encoder.PushByte(0); err != nil {
			return err
		}

		return encoder.Encode(f.Amount)
	case f.IsNonFungible:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		return encoder.Encode(f.AssetInstance)
	}

	return nil
}

type MultiAssetV1 struct {
	ID          AssetID
	Fungibility Fungibility
}

func (m *MultiAssetV1) Decode(decoder scale.Decoder) error {
	if err := decoder.Decode(&m.ID); err != nil {
		return err
	}

	return decoder.Decode(&m.Fungibility)
}

func (m MultiAssetV1) Encode(encoder scale.Encoder) error {
	if err := encoder.Encode(m.ID); err != nil {
		return err
	}

	return encoder.Encode(m.Fungibility)
}

type MultiAssetsV1 []MultiAssetV1

type MultiAssetV0 struct {
	IsNone bool

	IsAll bool

	IsAllFungible bool

	IsAllNonFungible bool

	IsAllAbstractFungible bool
	AllAbstractFungibleID []U8

	IsAllAbstractNonFungible    bool
	AllAbstractNonFungibleClass []U8

	IsAllConcreteFungible bool
	AllConcreteFungibleID MultiLocationV1

	IsAllConcreteNonFungible    bool
	AllConcreteNonFungibleClass MultiLocationV1

	IsAbstractFungible bool
	AbstractFungibleID []U8
	AbstractFungible   U128

	IsAbstractNonFungible       bool
	AbstractNonFungibleClass    []U8
	AbstractNonFungibleInstance AssetInstance

	IsConcreteFungible     bool
	ConcreteFungibleID     MultiLocationV1
	ConcreteFungibleAmount U128

	IsConcreteNonFungible       bool
	ConcreteNonFungibleClass    MultiLocationV1
	ConcreteNonFungibleInstance AssetInstance
}

func (m *MultiAssetV0) Decode(decoder scale.Decoder) error { //nolint:funlen
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		m.IsNone = true
	case 1:
		m.IsAll = true
	case 2:
		m.IsAllFungible = true
	case 3:
		m.IsAllNonFungible = true
	case 4:
		m.IsAllAbstractFungible = true

		return decoder.Decode(&m.AllAbstractFungibleID)
	case 5:
		m.IsAllAbstractNonFungible = true

		return decoder.Decode(&m.AllAbstractNonFungibleClass)
	case 6:
		m.IsAllConcreteFungible = true

		return decoder.Decode(&m.AllConcreteFungibleID)
	case 7:
		m.IsAllConcreteNonFungible = true

		return decoder.Decode(&m.AllConcreteNonFungibleClass)
	case 8:
		m.IsAbstractFungible = true

		if err := decoder.Decode(&m.AbstractFungibleID); err != nil {
			return err
		}

		return decoder.Decode(&m.AbstractFungible)
	case 9:
		m.IsAbstractNonFungible = true

		if err := decoder.Decode(&m.AbstractNonFungibleClass); err != nil {
			return err
		}

		return decoder.Decode(&m.AbstractNonFungibleInstance)
	case 10:
		m.IsConcreteFungible = true

		if err := decoder.Decode(&m.ConcreteFungibleID); err != nil {
			return err
		}

		return decoder.Decode(&m.ConcreteFungibleAmount)
	case 11:
		m.IsConcreteNonFungible = true

		if err := decoder.Decode(&m.ConcreteNonFungibleClass); err != nil {
			return err
		}

		return decoder.Decode(&m.ConcreteNonFungibleInstance)
	}

	return nil
}

func (m MultiAssetV0) Encode(encoder scale.Encoder) error { //nolint:funlen
	switch {
	case m.IsNone:
		return encoder.PushByte(0)
	case m.IsAll:
		return encoder.PushByte(1)
	case m.IsAllFungible:
		return encoder.PushByte(2)
	case m.IsAllNonFungible:
		return encoder.PushByte(3)
	case m.IsAllAbstractFungible:
		if err := encoder.PushByte(4); err != nil {
			return err
		}

		return encoder.Encode(m.AllAbstractFungibleID)
	case m.IsAllAbstractNonFungible:
		if err := encoder.PushByte(5); err != nil {
			return err
		}

		return encoder.Encode(m.AllAbstractNonFungibleClass)
	case m.IsAllConcreteFungible:
		if err := encoder.PushByte(6); err != nil {
			return err
		}

		return encoder.Encode(m.AllConcreteFungibleID)
	case m.IsAllConcreteNonFungible:
		if err := encoder.PushByte(7); err != nil {
			return err
		}

		return encoder.Encode(m.AllConcreteNonFungibleClass)
	case m.IsAbstractFungible:
		if err := encoder.PushByte(8); err != nil {
			return err
		}

		if err := encoder.Encode(m.AbstractFungibleID); err != nil {
			return err
		}

		return encoder.Encode(m.AbstractFungible)
	case m.IsAbstractNonFungible:
		if err := encoder.PushByte(9); err != nil {
			return err
		}

		if err := encoder.Encode(m.AbstractNonFungibleClass); err != nil {
			return err
		}

		return encoder.Encode(m.AbstractNonFungibleInstance)
	case m.IsConcreteFungible:
		if err := encoder.PushByte(10); err != nil {
			return err
		}

		if err := encoder.Encode(m.ConcreteFungibleID); err != nil {
			return err
		}

		return encoder.Encode(m.ConcreteFungibleAmount)
	case m.IsConcreteNonFungible:
		if err := encoder.PushByte(11); err != nil {
			return err
		}

		if err := encoder.Encode(m.ConcreteNonFungibleClass); err != nil {
			return err
		}

		return encoder.Encode(m.ConcreteNonFungibleInstance)
	}

	return nil
}

type VersionedMultiAssets struct {
	IsV0          bool
	MultiAssetsV0 []MultiAssetV0

	IsV1          bool
	MultiAssetsV1 MultiAssetsV1
}

func (v *VersionedMultiAssets) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()

	if err != nil {
		return err
	}

	switch b {
	case 0:
		v.IsV0 = true

		return decoder.Decode(&v.MultiAssetsV0)
	case 1:
		v.IsV1 = true

		return decoder.Decode(&v.MultiAssetsV1)
	}

	return nil
}

func (v VersionedMultiAssets) Encode(encoder scale.Encoder) error {
	switch {
	case v.IsV0:
		if err := encoder.PushByte(0); err != nil {
			return err
		}

		return encoder.Encode(v.MultiAssetsV0)
	case v.IsV1:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		return encoder.Encode(v.MultiAssetsV1)
	}

	return nil
}

type Response struct {
	IsNull bool

	IsAssets    bool
	MultiAssets MultiAssetsV1

	IsExecutionResult bool
	ExecutionResult   ExecutionResult

	IsVersion bool
	Version   U32
}

func (r *Response) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		r.IsNull = true
	case 1:
		r.IsAssets = true

		return decoder.Decode(&r.MultiAssets)
	case 2:
		r.IsExecutionResult = true

		return decoder.Decode(&r.ExecutionResult)
	case 3:
		r.IsVersion = true

		return decoder.Decode(&r.Version)
	}

	return nil
}

func (r Response) Encode(encoder scale.Encoder) error {
	switch {
	case r.IsNull:
		return encoder.PushByte(0)
	case r.IsAssets:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		return encoder.Encode(r.MultiAssets)
	case r.IsExecutionResult:
		if err := encoder.PushByte(2); err != nil {
			return err
		}

		return encoder.Encode(r.ExecutionResult)
	case r.IsVersion:
		if err := encoder.PushByte(3); err != nil {
			return err
		}

		return encoder.Encode(r.Version)
	}

	return nil
}

type OriginKind struct {
	IsNative bool

	IsSovereignAccount bool

	IsSuperuser bool

	IsXcm bool
}

func (o *OriginKind) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		o.IsNative = true
	case 1:
		o.IsSovereignAccount = true
	case 2:
		o.IsSuperuser = true
	case 3:
		o.IsXcm = true
	}

	return nil
}

func (o OriginKind) Encode(encoder scale.Encoder) error {
	switch {
	case o.IsNative:
		return encoder.PushByte(0)
	case o.IsSovereignAccount:
		return encoder.PushByte(1)
	case o.IsSuperuser:
		return encoder.PushByte(2)
	case o.IsXcm:
		return encoder.PushByte(3)
	}

	return nil
}

type EncodedCall struct {
	Call []U8
}

func (e *EncodedCall) Decode(decoder scale.Decoder) error {
	return decoder.Decode(&e.Call)
}

func (e EncodedCall) Encode(encoder scale.Encoder) error {
	return encoder.Encode(e.Call)
}

type WildFungibility struct {
	IsFungible    bool
	IsNonFungible bool
}

func (w *WildFungibility) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		w.IsFungible = true
	case 1:
		w.IsNonFungible = true
	}

	return nil
}

func (w WildFungibility) Encode(encoder scale.Encoder) error {
	switch {
	case w.IsFungible:
		return encoder.PushByte(0)
	case w.IsNonFungible:
		return encoder.PushByte(1)
	}

	return nil
}

type WildMultiAsset struct {
	IsAll bool

	IsAllOf bool
	ID      AssetID
	Fun     WildFungibility
}

func (w *WildMultiAsset) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		w.IsAll = true
	case 1:
		w.IsAllOf = true

		if err := decoder.Decode(&w.ID); err != nil {
			return err
		}

		return decoder.Decode(&w.Fun)
	}

	return nil
}

func (w WildMultiAsset) Encode(encoder scale.Encoder) error {
	switch {
	case w.IsAll:
		return encoder.PushByte(0)
	case w.IsAllOf:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		if err := encoder.Encode(w.ID); err != nil {
			return err
		}

		return encoder.Encode(w.Fun)
	}

	return nil
}

type MultiAssetFilter struct {
	IsDefinite  bool
	MultiAssets MultiAssetsV1

	IsWild         bool
	WildMultiAsset WildMultiAsset
}

func (m *MultiAssetFilter) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		m.IsDefinite = true

		return decoder.Decode(&m.MultiAssets)
	case 1:
		m.IsWild = true

		return decoder.Decode(&m.WildMultiAsset)
	}

	return nil
}

func (m MultiAssetFilter) Encode(encoder scale.Encoder) error {
	switch {
	case m.IsDefinite:
		if err := encoder.PushByte(0); err != nil {
			return err
		}

		return encoder.Encode(m.MultiAssets)
	case m.IsWild:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		return encoder.Encode(m.WildMultiAsset)
	}

	return nil
}

type WeightLimit struct {
	IsUnlimited bool

	IsLimited bool
	Limit     U64
}

func (w *WeightLimit) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		w.IsUnlimited = true
	case 1:
		w.IsLimited = true

		return decoder.Decode(&w.Limit)
	}

	return nil
}

func (w WeightLimit) Encode(encoder scale.Encoder) error {
	switch {
	case w.IsUnlimited:
		return encoder.PushByte(0)
	case w.IsLimited:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		return encoder.Encode(w.Limit)
	}

	return nil
}

type Instruction struct {
	IsWithdrawAsset          bool
	WithdrawAssetMultiAssets MultiAssetsV1

	IsReserveAssetDeposited          bool
	ReserveAssetDepositedMultiAssets MultiAssetsV1

	IsReceiveTeleportedAsset          bool
	ReceiveTeleportedAssetMultiAssets MultiAssetsV1

	IsQueryResponse        bool
	QueryResponseQueryID   UCompact
	QueryResponseResponse  Response
	QueryResponseMaxWeight UCompact

	IsTransferAsset          bool
	TransferAssetAssets      MultiAssetsV1
	TransferAssetBeneficiary MultiLocationV1

	IsTransferReserveAsset          bool
	TransferReserveAssetMultiAssets MultiAssetsV1
	TransferReserveAssetDest        MultiLocationV1
	TransferReserveAssetXCM         []Instruction

	IsTransact                  bool
	TransactOriginType          OriginKind
	TransactRequireWeightAtMost UCompact
	// NOTE:
	//
	// As per https://github.com/paritytech/polkadot/blob/c254e5975711a6497af256f6831e9a6c752d28f5/xcm/src/v2/mod.rs#L343
	// The `Call` should be wrapped by the `DoubleEncoded` found here:
	// https://github.com/paritytech/polkadot/blob/c254e5975711a6497af256f6831e9a6c752d28f5/xcm/src/double_encoded.rs#L27
	//
	// However, since the decoded option is skipped by the codec, we are not adding it here.
	TransactCall EncodedCall

	IsHrmpNewChannelOpenRequest             bool
	HrmpNewChannelOpenRequestSender         U32
	HrmpNewChannelOpenRequestMaxMessageSize U32
	HrmpNewChannelOpenRequestMaxCapacity    U32

	IsHrmpChannelAccepted        bool
	HrmpChannelAcceptedRecipient U32

	IsHrmpChannelClosing        bool
	HrmpChannelClosingInitiator U32
	HrmpChannelClosingSender    U32
	HrmpChannelClosingRecipient U32

	IsClearOrigin bool

	IsDescendOrigin       bool
	DescendOriginLocation JunctionsV1

	IsReportError                bool
	ReportErrorQueryID           U64
	ReportErrorDestination       MultiLocationV1
	ReportErrorMaxResponseWeight U64

	IsDepositAsset               bool
	DepositAssetMultiAssetFilter MultiAssetFilter
	DepositAssetMaxAssets        U32
	DepositAssetBeneficiary      MultiLocationV1

	IsDepositReserveAsset               bool
	DepositReserveAssetMultiAssetFilter MultiAssetFilter
	DepositReserveAssetMaxAssets        U32
	DepositReserveAssetDest             MultiLocationV1
	DepositReserveAssetXCM              []Instruction

	IsExchangeAsset      bool
	ExchangeAssetGive    MultiAssetFilter
	ExchangeAssetReceive MultiAssetsV1

	IsInitiateReserveWithdraw      bool
	InitiateReserveWithdrawAssets  MultiAssetFilter
	InitiateReserveWithdrawReserve MultiLocationV1
	InitiateReserveWithdrawXCM     []Instruction

	IsInitiateTeleport     bool
	InitiateTeleportAssets MultiAssetFilter
	InitiateTeleportDest   MultiLocationV1
	InitiateTeleportXCM    []Instruction

	IsQueryHolding                bool
	QueryHoldingQueryID           U64
	QueryHoldingDest              MultiLocationV1
	QueryHoldingAssets            MultiAssetFilter
	QueryHoldingMaxResponseWeight U64

	IsBuyExecution          bool
	BuyExecutionFees        MultiAssetV1
	BuyExecutionWeightLimit WeightLimit

	IsRefundSurplus bool

	IsSetErrorHandler  bool
	SetErrorHandlerXCM []Instruction

	IsSetAppendix  bool
	SetAppendixXCM []Instruction

	IsClearError bool

	IsClaimAsset     bool
	ClaimAssetAssets MultiAssetsV1
	ClaimAssetTicket MultiLocationV1

	IsTrap   bool
	TrapCode U64

	IsSubscribeVersion                bool
	SubscribeVersionQueryID           U64
	SubscribeVersionMaxResponseWeight U64

	IsUnsubscribeVersion bool
}

func (i *Instruction) Decode(decoder scale.Decoder) error { //nolint:gocyclo,funlen
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		i.IsWithdrawAsset = true

		return decoder.Decode(&i.WithdrawAssetMultiAssets)
	case 1:
		i.IsReserveAssetDeposited = true

		return decoder.Decode(&i.ReserveAssetDepositedMultiAssets)
	case 2:
		i.IsReceiveTeleportedAsset = true

		return decoder.Decode(&i.ReceiveTeleportedAssetMultiAssets)
	case 3:
		i.IsQueryResponse = true

		if err := decoder.Decode(&i.QueryResponseQueryID); err != nil {
			return err
		}

		if err := decoder.Decode(&i.QueryResponseResponse); err != nil {
			return err
		}

		return decoder.Decode(&i.QueryResponseMaxWeight)
	case 4:
		i.IsTransferAsset = true

		if err := decoder.Decode(&i.TransferAssetAssets); err != nil {
			return err
		}

		return decoder.Decode(&i.TransferAssetBeneficiary)
	case 5:
		i.IsTransferReserveAsset = true

		if err := decoder.Decode(&i.TransferReserveAssetMultiAssets); err != nil {
			return err
		}

		if err := decoder.Decode(&i.TransferReserveAssetDest); err != nil {
			return err
		}

		return decoder.Decode(&i.TransferReserveAssetXCM)
	case 6:
		i.IsTransact = true

		if err := decoder.Decode(&i.TransactOriginType); err != nil {
			return err
		}

		if err := decoder.Decode(&i.TransactRequireWeightAtMost); err != nil {
			return err
		}

		return decoder.Decode(&i.TransactCall)
	case 7:
		i.IsHrmpNewChannelOpenRequest = true

		if err := decoder.Decode(&i.HrmpNewChannelOpenRequestSender); err != nil {
			return err
		}

		if err := decoder.Decode(&i.HrmpNewChannelOpenRequestMaxMessageSize); err != nil {
			return err
		}

		return decoder.Decode(&i.HrmpNewChannelOpenRequestMaxCapacity)
	case 8:
		i.IsHrmpChannelAccepted = true

		return decoder.Decode(&i.HrmpChannelAcceptedRecipient)
	case 9:
		i.IsHrmpChannelClosing = true

		if err := decoder.Decode(&i.HrmpChannelClosingInitiator); err != nil {
			return err
		}

		if err := decoder.Decode(&i.HrmpChannelClosingSender); err != nil {
			return err
		}

		return decoder.Decode(&i.HrmpChannelClosingRecipient)
	case 10:
		i.IsClearOrigin = true
	case 11:
		i.IsDescendOrigin = true

		return decoder.Decode(&i.DescendOriginLocation)
	case 12:
		i.IsReportError = true

		if err := decoder.Decode(&i.ReportErrorQueryID); err != nil {
			return err
		}

		if err := decoder.Decode(&i.ReportErrorDestination); err != nil {
			return err
		}

		return decoder.Decode(&i.ReportErrorMaxResponseWeight)
	case 13:
		i.IsDepositAsset = true

		if err := decoder.Decode(&i.DepositAssetMultiAssetFilter); err != nil {
			return err
		}

		if err := decoder.Decode(&i.DepositAssetMaxAssets); err != nil {
			return err
		}

		return decoder.Decode(&i.DepositAssetBeneficiary)
	case 14:
		i.IsDepositReserveAsset = true

		if err := decoder.Decode(&i.DepositReserveAssetMultiAssetFilter); err != nil {
			return err
		}

		if err := decoder.Decode(&i.DepositReserveAssetMaxAssets); err != nil {
			return err
		}

		if err := decoder.Decode(&i.DepositReserveAssetDest); err != nil {
			return err
		}

		return decoder.Decode(&i.DepositReserveAssetXCM)
	case 15:
		i.IsExchangeAsset = true

		if err := decoder.Decode(&i.ExchangeAssetGive); err != nil {
			return err
		}

		return decoder.Decode(&i.ExchangeAssetReceive)
	case 16:
		i.IsInitiateReserveWithdraw = true

		if err := decoder.Decode(&i.InitiateReserveWithdrawAssets); err != nil {
			return err
		}

		if err := decoder.Decode(&i.InitiateReserveWithdrawReserve); err != nil {
			return err
		}

		return decoder.Decode(&i.InitiateReserveWithdrawXCM)
	case 17:
		i.IsInitiateTeleport = true

		if err := decoder.Decode(&i.InitiateTeleportAssets); err != nil {
			return err
		}

		if err := decoder.Decode(&i.InitiateTeleportDest); err != nil {
			return err
		}

		return decoder.Decode(&i.InitiateTeleportXCM)
	case 18:
		i.IsQueryHolding = true

		if err := decoder.Decode(&i.QueryHoldingQueryID); err != nil {
			return err
		}

		if err := decoder.Decode(&i.QueryHoldingDest); err != nil {
			return err
		}

		if err := decoder.Decode(&i.QueryHoldingAssets); err != nil {
			return err
		}

		return decoder.Decode(&i.QueryHoldingMaxResponseWeight)
	case 19:
		i.IsBuyExecution = true

		if err := decoder.Decode(&i.BuyExecutionFees); err != nil {
			return err
		}

		return decoder.Decode(&i.BuyExecutionWeightLimit)
	case 20:
		i.IsRefundSurplus = true
	case 21:
		i.IsSetErrorHandler = true

		return decoder.Decode(&i.SetErrorHandlerXCM)
	case 22:
		i.IsSetAppendix = true

		return decoder.Decode(&i.SetAppendixXCM)
	case 23:
		i.IsClearError = true
	case 24:
		i.IsClaimAsset = true

		if err := decoder.Decode(&i.ClaimAssetAssets); err != nil {
			return err
		}

		return decoder.Decode(&i.ClaimAssetTicket)
	case 25:
		i.IsTrap = true

		return decoder.Decode(&i.TrapCode)
	case 26:
		i.IsSubscribeVersion = true

		if err := decoder.Decode(&i.SubscribeVersionQueryID); err != nil {
			return err
		}

		return decoder.Decode(&i.SubscribeVersionMaxResponseWeight)
	case 27:
		i.IsUnsubscribeVersion = true
	}

	return nil
}

func (i Instruction) Encode(encoder scale.Encoder) error { //nolint:gocyclo,funlen
	switch {
	case i.IsWithdrawAsset:
		if err := encoder.PushByte(0); err != nil {
			return err
		}

		return encoder.Encode(i.WithdrawAssetMultiAssets)
	case i.IsReserveAssetDeposited:
		if err := encoder.PushByte(1); err != nil {
			return err
		}

		return encoder.Encode(i.ReserveAssetDepositedMultiAssets)
	case i.IsReceiveTeleportedAsset:
		if err := encoder.PushByte(2); err != nil {
			return err
		}

		return encoder.Encode(i.ReceiveTeleportedAssetMultiAssets)
	case i.IsQueryResponse:
		if err := encoder.PushByte(3); err != nil {
			return err
		}

		if err := encoder.Encode(i.QueryResponseQueryID); err != nil {
			return err
		}

		if err := encoder.Encode(i.QueryResponseResponse); err != nil {
			return err
		}

		return encoder.Encode(i.QueryResponseMaxWeight)
	case i.IsTransferAsset:
		if err := encoder.PushByte(4); err != nil {
			return err
		}

		if err := encoder.Encode(i.TransferAssetAssets); err != nil {
			return err
		}

		return encoder.Encode(i.TransferAssetBeneficiary)
	case i.IsTransferReserveAsset:
		if err := encoder.PushByte(5); err != nil {
			return err
		}

		if err := encoder.Encode(i.TransferReserveAssetMultiAssets); err != nil {
			return err
		}

		if err := encoder.Encode(i.TransferReserveAssetDest); err != nil {
			return err
		}

		return encoder.Encode(i.TransferReserveAssetXCM)
	case i.IsTransact:
		if err := encoder.PushByte(6); err != nil {
			return err
		}

		if err := encoder.Encode(i.TransactOriginType); err != nil {
			return err
		}

		if err := encoder.Encode(i.TransactRequireWeightAtMost); err != nil {
			return err
		}

		return encoder.Encode(i.TransactCall)
	case i.IsHrmpNewChannelOpenRequest:
		if err := encoder.PushByte(7); err != nil {
			return err
		}

		if err := encoder.Encode(i.HrmpNewChannelOpenRequestSender); err != nil {
			return err
		}

		if err := encoder.Encode(i.HrmpNewChannelOpenRequestMaxMessageSize); err != nil {
			return err
		}

		return encoder.Encode(i.HrmpNewChannelOpenRequestMaxCapacity)
	case i.IsHrmpChannelAccepted:
		if err := encoder.PushByte(8); err != nil {
			return err
		}

		return encoder.Encode(i.HrmpChannelAcceptedRecipient)
	case i.IsHrmpChannelClosing:
		if err := encoder.PushByte(9); err != nil {
			return err
		}

		if err := encoder.Encode(i.HrmpChannelClosingInitiator); err != nil {
			return err
		}

		if err := encoder.Encode(i.HrmpChannelClosingSender); err != nil {
			return err
		}

		return encoder.Encode(i.HrmpChannelClosingRecipient)
	case i.IsClearOrigin:
		return encoder.PushByte(10)
	case i.IsDescendOrigin:
		if err := encoder.PushByte(11); err != nil {
			return err
		}

		return encoder.Encode(i.DescendOriginLocation)
	case i.IsReportError:
		if err := encoder.PushByte(12); err != nil {
			return err
		}

		if err := encoder.Encode(i.ReportErrorQueryID); err != nil {
			return err
		}

		if err := encoder.Encode(i.ReportErrorDestination); err != nil {
			return err
		}

		return encoder.Encode(i.ReportErrorMaxResponseWeight)
	case i.IsDepositAsset:
		if err := encoder.PushByte(13); err != nil {
			return err
		}

		if err := encoder.Encode(i.DepositAssetMultiAssetFilter); err != nil {
			return err
		}

		if err := encoder.Encode(i.DepositAssetMaxAssets); err != nil {
			return err
		}

		return encoder.Encode(i.DepositAssetBeneficiary)
	case i.IsDepositReserveAsset:
		if err := encoder.PushByte(14); err != nil {
			return err
		}

		if err := encoder.Encode(i.DepositReserveAssetMultiAssetFilter); err != nil {
			return err
		}

		if err := encoder.Encode(i.DepositReserveAssetMaxAssets); err != nil {
			return err
		}

		if err := encoder.Encode(i.DepositReserveAssetDest); err != nil {
			return err
		}

		return encoder.Encode(i.DepositReserveAssetXCM)
	case i.IsExchangeAsset:
		if err := encoder.PushByte(15); err != nil {
			return err
		}

		if err := encoder.Encode(i.ExchangeAssetGive); err != nil {
			return err
		}

		return encoder.Encode(i.ExchangeAssetReceive)
	case i.IsInitiateReserveWithdraw:
		if err := encoder.PushByte(16); err != nil {
			return err
		}

		if err := encoder.Encode(i.InitiateReserveWithdrawAssets); err != nil {
			return err
		}

		if err := encoder.Encode(i.InitiateReserveWithdrawReserve); err != nil {
			return err
		}

		return encoder.Encode(i.InitiateReserveWithdrawXCM)
	case i.IsInitiateTeleport:
		if err := encoder.PushByte(17); err != nil {
			return err
		}

		if err := encoder.Encode(i.InitiateTeleportAssets); err != nil {
			return err
		}

		if err := encoder.Encode(i.InitiateTeleportDest); err != nil {
			return err
		}

		return encoder.Encode(i.InitiateTeleportXCM)
	case i.IsQueryHolding:
		if err := encoder.PushByte(18); err != nil {
			return err
		}

		if err := encoder.Encode(i.QueryHoldingQueryID); err != nil {
			return err
		}

		if err := encoder.Encode(i.QueryHoldingDest); err != nil {
			return err
		}

		if err := encoder.Encode(i.QueryHoldingAssets); err != nil {
			return err
		}

		return encoder.Encode(i.QueryHoldingMaxResponseWeight)
	case i.IsBuyExecution:
		if err := encoder.PushByte(19); err != nil {
			return err
		}

		if err := encoder.Encode(i.BuyExecutionFees); err != nil {
			return err
		}

		return encoder.Encode(i.BuyExecutionWeightLimit)
	case i.IsRefundSurplus:
		return encoder.PushByte(20)
	case i.IsSetErrorHandler:
		if err := encoder.PushByte(21); err != nil {
			return err
		}

		return encoder.Encode(i.SetErrorHandlerXCM)
	case i.IsSetAppendix:
		if err := encoder.PushByte(22); err != nil {
			return err
		}

		return encoder.Encode(i.SetAppendixXCM)
	case i.IsClearError:
		return encoder.PushByte(23)
	case i.IsClaimAsset:
		if err := encoder.PushByte(24); err != nil {
			return err
		}

		if err := encoder.Encode(i.ClaimAssetAssets); err != nil {
			return err
		}

		return encoder.Encode(i.ClaimAssetTicket)
	case i.IsTrap:
		if err := encoder.PushByte(25); err != nil {
			return err
		}

		return encoder.Encode(i.TrapCode)
	case i.IsSubscribeVersion:
		if err := encoder.PushByte(26); err != nil {
			return err
		}

		if err := encoder.Encode(i.SubscribeVersionQueryID); err != nil {
			return err
		}

		return encoder.Encode(i.SubscribeVersionMaxResponseWeight)
	case i.IsUnsubscribeVersion:
		return encoder.PushByte(27)
	}

	return nil
}
