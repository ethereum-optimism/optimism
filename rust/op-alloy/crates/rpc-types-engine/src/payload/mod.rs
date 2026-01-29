//! Versioned Optimism execution payloads

pub mod error;
pub mod v3;
pub mod v4;

use crate::{OpExecutionPayloadSidecar, OpExecutionPayloadV4};
use alloc::vec::Vec;
use alloy_consensus::{Block, BlockHeader, HeaderInfo, Transaction};
use alloy_eips::{Decodable2718, Encodable2718, Typed2718, eip7685::EMPTY_REQUESTS_HASH};
use alloy_primitives::{Address, B256, Bytes, Sealable, U256};
use alloy_rpc_types_engine::{
    ExecutionPayload, ExecutionPayloadInputV2, ExecutionPayloadV1, ExecutionPayloadV2,
    ExecutionPayloadV3, PayloadError,
};
use error::OpPayloadError;

/// An execution payload, which can be either [`ExecutionPayloadV2`], [`ExecutionPayloadV3`], or
/// [`OpExecutionPayloadV4`].
#[derive(Clone, Debug, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize))]
#[cfg_attr(feature = "std", derive(ssz_derive::Encode, ssz_derive::Decode))]
#[cfg_attr(feature = "std", ssz(enum_behaviour = "transparent"))]
#[cfg_attr(feature = "serde", serde(untagged))]
pub enum OpExecutionPayload {
    /// V1 payload
    V1(ExecutionPayloadV1),
    /// V2 payload
    V2(ExecutionPayloadV2),
    /// V3 payload
    V3(ExecutionPayloadV3),
    /// V4 payload
    V4(OpExecutionPayloadV4),
}

#[cfg(feature = "serde")]
impl<'de> serde::Deserialize<'de> for OpExecutionPayload {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct ExecutionPayloadVisitor;

        impl<'de> serde::de::Visitor<'de> for ExecutionPayloadVisitor {
            type Value = OpExecutionPayload;

            fn expecting(&self, formatter: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
                formatter.write_str("a valid OpExecutionPayload object")
            }

            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: serde::de::MapAccess<'de>,
            {
                use alloc::string::String;
                use alloy_primitives::{U64, map::HashMap};
                use alloy_rpc_types_engine::ExecutionPayloadV1;

                enum Fields {
                    ParentHash,
                    FeeRecipient,
                    StateRoot,
                    ReceiptsRoot,
                    LogsBloom,
                    PrevRandao,
                    BlockNumber,
                    GasLimit,
                    GasUsed,
                    Timestamp,
                    ExtraData,
                    BaseFeePerGas,
                    BlockHash,
                    Transactions,
                    Withdrawals,
                    BlobGasUsed,
                    ExcessBlobGas,
                    WithdrawalsRoot,
                    Unknown(alloc::string::String),
                }

                impl<'de> serde::Deserialize<'de> for Fields {
                    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
                    where
                        D: serde::Deserializer<'de>,
                    {
                        struct FieldVisitor;

                        impl serde::de::Visitor<'_> for FieldVisitor {
                            type Value = Fields;

                            fn expecting(
                                &self,
                                formatter: &mut core::fmt::Formatter<'_>,
                            ) -> core::fmt::Result {
                                formatter.write_str("a known field")
                            }

                            fn visit_str<E>(self, value: &str) -> Result<Fields, E>
                            where
                                E: serde::de::Error,
                            {
                                Ok(match value {
                                    "parentHash" => Fields::ParentHash,
                                    "feeRecipient" => Fields::FeeRecipient,
                                    "stateRoot" => Fields::StateRoot,
                                    "receiptsRoot" => Fields::ReceiptsRoot,
                                    "logsBloom" => Fields::LogsBloom,
                                    "prevRandao" => Fields::PrevRandao,
                                    "blockNumber" => Fields::BlockNumber,
                                    "gasLimit" => Fields::GasLimit,
                                    "gasUsed" => Fields::GasUsed,
                                    "timestamp" => Fields::Timestamp,
                                    "extraData" => Fields::ExtraData,
                                    "baseFeePerGas" => Fields::BaseFeePerGas,
                                    "blockHash" => Fields::BlockHash,
                                    "transactions" => Fields::Transactions,
                                    "withdrawals" => Fields::Withdrawals,
                                    "blobGasUsed" => Fields::BlobGasUsed,
                                    "excessBlobGas" => Fields::ExcessBlobGas,
                                    "withdrawalsRoot" => Fields::WithdrawalsRoot,
                                    _ => Fields::Unknown(value.into()),
                                })
                            }
                        }

                        deserializer.deserialize_str(FieldVisitor)
                    }
                }

                let mut parent_hash = None;
                let mut fee_recipient = None;
                let mut state_root = None;
                let mut receipts_root = None;
                let mut logs_bloom = None;
                let mut prev_randao = None;
                let mut block_number = None;
                let mut gas_limit = None;
                let mut gas_used = None;
                let mut timestamp = None;
                let mut extra_data = None;
                let mut base_fee_per_gas = None;
                let mut block_hash = None;
                let mut transactions = None;
                let mut withdrawals = None;
                let mut blob_gas_used = None;
                let mut excess_blob_gas = None;
                let mut withdrawals_root = None;

                let mut extra_fields = HashMap::new();

                while let Some(key) = map.next_key()? {
                    match key {
                        Fields::ParentHash => parent_hash = Some(map.next_value()?),
                        Fields::FeeRecipient => fee_recipient = Some(map.next_value()?),
                        Fields::StateRoot => state_root = Some(map.next_value()?),
                        Fields::ReceiptsRoot => receipts_root = Some(map.next_value()?),
                        Fields::LogsBloom => logs_bloom = Some(map.next_value()?),
                        Fields::PrevRandao => prev_randao = Some(map.next_value()?),
                        Fields::BlockNumber => {
                            let raw = map.next_value::<U64>()?;
                            block_number = Some(raw.to());
                        }
                        Fields::GasLimit => {
                            let raw = map.next_value::<U64>()?;
                            gas_limit = Some(raw.to());
                        }
                        Fields::GasUsed => {
                            let raw = map.next_value::<U64>()?;
                            gas_used = Some(raw.to());
                        }
                        Fields::Timestamp => {
                            let raw = map.next_value::<U64>()?;
                            timestamp = Some(raw.to());
                        }
                        Fields::ExtraData => extra_data = Some(map.next_value()?),
                        Fields::BaseFeePerGas => base_fee_per_gas = Some(map.next_value()?),
                        Fields::BlockHash => block_hash = Some(map.next_value()?),
                        Fields::Transactions => transactions = Some(map.next_value()?),
                        Fields::Withdrawals => withdrawals = Some(map.next_value()?),
                        Fields::BlobGasUsed => {
                            let raw = map.next_value::<U64>()?;
                            blob_gas_used = Some(raw.to());
                        }
                        Fields::ExcessBlobGas => {
                            let raw = map.next_value::<U64>()?;
                            excess_blob_gas = Some(raw.to());
                        }
                        Fields::WithdrawalsRoot => withdrawals_root = Some(map.next_value()?),
                        Fields::Unknown(field) => {
                            let raw = map.next_value::<String>()?;
                            extra_fields.insert(field, raw);
                        }
                    }
                }

                let v1 = ExecutionPayloadV1 {
                    parent_hash: parent_hash
                        .ok_or_else(|| serde::de::Error::missing_field("parentHash"))?,
                    fee_recipient: fee_recipient
                        .ok_or_else(|| serde::de::Error::missing_field("feeRecipient"))?,
                    state_root: state_root
                        .ok_or_else(|| serde::de::Error::missing_field("stateRoot"))?,
                    receipts_root: receipts_root
                        .ok_or_else(|| serde::de::Error::missing_field("receiptsRoot"))?,
                    logs_bloom: logs_bloom
                        .ok_or_else(|| serde::de::Error::missing_field("logsBloom"))?,
                    prev_randao: prev_randao
                        .ok_or_else(|| serde::de::Error::missing_field("prevRandao"))?,
                    block_number: block_number
                        .ok_or_else(|| serde::de::Error::missing_field("blockNumber"))?,
                    gas_limit: gas_limit
                        .ok_or_else(|| serde::de::Error::missing_field("gasLimit"))?,
                    gas_used: gas_used.ok_or_else(|| serde::de::Error::missing_field("gasUsed"))?,
                    timestamp: timestamp
                        .ok_or_else(|| serde::de::Error::missing_field("timestamp"))?,
                    extra_data: extra_data
                        .ok_or_else(|| serde::de::Error::missing_field("extraData"))?,
                    base_fee_per_gas: base_fee_per_gas
                        .ok_or_else(|| serde::de::Error::missing_field("baseFeePerGas"))?,
                    block_hash: block_hash
                        .ok_or_else(|| serde::de::Error::missing_field("blockHash"))?,
                    transactions: transactions
                        .ok_or_else(|| serde::de::Error::missing_field("transactions"))?,
                };

                // Ensure `withdrawals` is present before proceeding
                let withdrawals =
                    withdrawals.ok_or_else(|| serde::de::Error::missing_field("withdrawals"))?;

                // Construct base V2 payload
                let payload_v2 = ExecutionPayloadV2 { payload_inner: v1, withdrawals };

                // Ensure `blob_gas_used` and `excess_blob_gas` are either both present or both
                // absent
                match (blob_gas_used, excess_blob_gas) {
                    // If both are present, create V3
                    (Some(blob_gas_used), Some(excess_blob_gas)) => {
                        let payload_v3 = ExecutionPayloadV3 {
                            payload_inner: payload_v2,
                            blob_gas_used,
                            excess_blob_gas,
                        };

                        // If `withdrawals_root` is present, wrap into V4; otherwise, return V3
                        if let Some(withdrawals_root) = withdrawals_root {
                            Ok(OpExecutionPayload::V4(OpExecutionPayloadV4 {
                                payload_inner: payload_v3,
                                withdrawals_root,
                            }))
                        } else {
                            Ok(OpExecutionPayload::V3(payload_v3))
                        }
                    }
                    // If one is missing, reject as invalid
                    (Some(_), None) | (None, Some(_)) => {
                        Err(serde::de::Error::custom("invalid enum variant"))
                    }
                    // If neither are present, return V2
                    (None, None) => Ok(OpExecutionPayload::V2(payload_v2)),
                }
            }
        }

        const FIELDS: &[&str] = &[
            "parentHash",
            "feeRecipient",
            "stateRoot",
            "receiptsRoot",
            "logsBloom",
            "prevRandao",
            "blockNumber",
            "gasLimit",
            "gasUsed",
            "timestamp",
            "extraData",
            "baseFeePerGas",
            "blockHash",
            "transactions",
            "withdrawals",
            "blobGasUsed",
            "excessBlobGas",
            "withdrawalsRoot",
        ];

        deserializer.deserialize_struct("OpExecutionPayload", FIELDS, ExecutionPayloadVisitor)
    }
}

impl OpExecutionPayload {
    /// Conversion from [`alloy_consensus::Block`]. Also returns the
    /// [`OpExecutionPayloadSidecar`] extracted from the block.
    ///
    /// See also [`from_block_unchecked`](OpExecutionPayload::from_block_unchecked).
    ///
    /// Note: This re-calculates the block hash.
    pub fn from_block_slow<T, H>(block: &Block<T, H>) -> (Self, OpExecutionPayloadSidecar)
    where
        T: Encodable2718 + Transaction,
        H: BlockHeader + Sealable,
    {
        Self::from_block_unchecked(block.hash_slow(), block)
    }

    /// Conversion from [`alloy_consensus::Block`]. Also returns the
    /// [`OpExecutionPayloadSidecar`] extracted from the block.
    ///
    /// See also [`ExecutionPayload::from_block_unchecked`].
    /// See also [`OpExecutionPayloadSidecar::from_block`].
    pub fn from_block_unchecked<T, H>(
        block_hash: B256,
        block: &Block<T, H>,
    ) -> (Self, OpExecutionPayloadSidecar)
    where
        T: Encodable2718 + Transaction,
        H: BlockHeader,
    {
        let sidecar = OpExecutionPayloadSidecar::from_block(block);

        let execution_payload = match block.withdrawals_root() {
            Some(withdrawals_root) if sidecar.isthmus().is_some() => {
                // block with (empty) request hashes: V4
                Self::V4(OpExecutionPayloadV4::from_v3_with_withdrawals_root(
                    ExecutionPayloadV3::from_block_unchecked(block_hash, block),
                    withdrawals_root,
                ))
            }
            Some(_) if block.header.parent_beacon_block_root().is_some() => {
                // block with parent beacon block root: at least V3
                Self::V3(ExecutionPayloadV3::from_block_unchecked(block_hash, block))
            }
            Some(_) => {
                // block with withdrawals root: at least V2
                Self::V2(ExecutionPayloadV2::from_block_unchecked(block_hash, block))
            }
            None => {
                // otherwise V1
                Self::V1(ExecutionPayloadV1::from_block_unchecked(block_hash, block))
            }
        };

        (execution_payload, sidecar)
    }

    /// Creates a new instance from `newPayloadV2` payload, i.e. [`V1`](Self::V1) or
    /// [`V2`](Self::V2) variant.
    ///
    /// Spec: <https://specs.optimism.io/protocol/exec-engine.html#engine_newpayloadv2>
    pub fn v2(payload: ExecutionPayloadInputV2) -> Self {
        match payload.into_payload() {
            ExecutionPayload::V1(payload) => Self::V1(payload),
            ExecutionPayload::V2(payload) => Self::V2(payload),
            _ => unreachable!(),
        }
    }

    /// Creates a new instance from `newPayloadV3` payload, i.e. [`V3`](Self::V3) variant.
    ///
    /// Spec: <https://specs.optimism.io/protocol/exec-engine.html#engine_newpayloadv3>
    pub const fn v3(payload: ExecutionPayloadV3) -> Self {
        Self::V3(payload)
    }

    /// Creates a new instance from `newPayloadV4` payload, i.e. [`V4`](Self::V4) variant.
    ///
    /// Spec: <https://specs.optimism.io/protocol/exec-engine.html#engine_newpayloadv4>
    pub const fn v4(payload: OpExecutionPayloadV4) -> Self {
        Self::V4(payload)
    }

    /// Returns a reference to the V1 payload.
    pub const fn as_v1(&self) -> &ExecutionPayloadV1 {
        match self {
            Self::V1(payload) => payload,
            Self::V2(payload) => &payload.payload_inner,
            Self::V3(payload) => &payload.payload_inner.payload_inner,
            Self::V4(payload) => &payload.payload_inner.payload_inner.payload_inner,
        }
    }

    /// Returns a mutable reference to the V1 payload.
    pub const fn as_v1_mut(&mut self) -> &mut ExecutionPayloadV1 {
        match self {
            Self::V1(payload) => payload,
            Self::V2(payload) => &mut payload.payload_inner,
            Self::V3(payload) => &mut payload.payload_inner.payload_inner,
            Self::V4(payload) => &mut payload.payload_inner.payload_inner.payload_inner,
        }
    }

    /// Returns a reference to the V2 payload, if any.
    pub const fn as_v2(&self) -> Option<&ExecutionPayloadV2> {
        match self {
            Self::V1(_) => None,
            Self::V2(payload) => Some(payload),
            Self::V3(payload) => Some(&payload.payload_inner),
            Self::V4(payload) => Some(&payload.payload_inner.payload_inner),
        }
    }

    /// Returns a mutable reference to the V2 payload, if any.
    pub const fn as_v2_mut(&mut self) -> Option<&mut ExecutionPayloadV2> {
        match self {
            Self::V1(_) => None,
            Self::V2(payload) => Some(payload),
            Self::V3(payload) => Some(&mut payload.payload_inner),
            Self::V4(payload) => Some(&mut payload.payload_inner.payload_inner),
        }
    }

    /// Returns a reference to the V3 payload, if any.
    pub const fn as_v3(&self) -> Option<&ExecutionPayloadV3> {
        match self {
            Self::V1(_) | Self::V2(_) => None,
            Self::V3(payload) => Some(payload),
            Self::V4(payload) => Some(&payload.payload_inner),
        }
    }

    /// Returns a mutable reference to the V3 payload, if any.
    pub const fn as_v3_mut(&mut self) -> Option<&mut ExecutionPayloadV3> {
        match self {
            Self::V1(_) | Self::V2(_) => None,
            Self::V3(payload) => Some(payload),
            Self::V4(payload) => Some(&mut payload.payload_inner),
        }
    }

    /// Returns a reference to the V4 payload, if any.
    pub const fn as_v4(&self) -> Option<&OpExecutionPayloadV4> {
        match self {
            Self::V1(_) | Self::V2(_) | Self::V3(_) => None,
            Self::V4(payload) => Some(payload),
        }
    }

    /// Returns a mutable reference to the V4 payload, if any.
    pub const fn as_v4_mut(&mut self) -> Option<&mut OpExecutionPayloadV4> {
        match self {
            Self::V1(_) | Self::V2(_) | Self::V3(_) => None,
            Self::V4(payload) => Some(payload),
        }
    }

    /// Returns the transactions for the payload.
    pub const fn transactions(&self) -> &Vec<Bytes> {
        &self.as_v1().transactions
    }

    /// Returns a mutable reference to the transactions for the payload.
    pub const fn transactions_mut(&mut self) -> &mut Vec<Bytes> {
        &mut self.as_v1_mut().transactions
    }

    /// Returns the parent hash for the payload.
    pub const fn parent_hash(&self) -> B256 {
        self.as_v1().parent_hash
    }

    /// Returns the block hash for the payload.
    pub const fn block_hash(&self) -> B256 {
        self.as_v1().block_hash
    }

    /// Returns the block number for this payload.
    pub const fn block_number(&self) -> u64 {
        self.as_v1().block_number
    }

    /// Returns the timestamp for this payload.
    pub const fn timestamp(&self) -> u64 {
        self.as_v1().timestamp
    }

    /// Returns the fee recipient for this payload.
    pub const fn fee_recipient(&self) -> Address {
        self.as_v1().fee_recipient
    }

    /// Returns the gas limit for this payload.
    pub const fn gas_limit(&self) -> u64 {
        self.as_v1().gas_limit
    }

    /// Returns the saturated base fee per gas for this payload.
    pub fn saturated_base_fee_per_gas(&self) -> u64 {
        self.as_v1().base_fee_per_gas.saturating_to()
    }

    /// Returns the excess blob gas for this payload.
    pub fn excess_blob_gas(&self) -> Option<u64> {
        self.as_v3().map(|payload| payload.excess_blob_gas)
    }

    /// Returns the blob gas used for this payload.
    pub fn blob_gas_used(&self) -> Option<u64> {
        self.as_v3().map(|payload| payload.blob_gas_used)
    }

    /// Returns the prev randao for this payload.
    pub const fn prev_randao(&self) -> B256 {
        self.as_v1().prev_randao
    }

    /// Extracts essential information into one container type.
    pub fn header_info(&self) -> HeaderInfo {
        HeaderInfo {
            number: self.block_number(),
            beneficiary: self.fee_recipient(),
            timestamp: self.timestamp(),
            gas_limit: self.gas_limit(),
            base_fee_per_gas: Some(self.saturated_base_fee_per_gas()),
            excess_blob_gas: self.excess_blob_gas(),
            blob_gas_used: self.blob_gas_used(),
            difficulty: U256::ZERO,
            mix_hash: Some(self.prev_randao()),
        }
    }

    /// Converts [`OpExecutionPayload`] to [`Block`] with raw transactions.
    ///
    /// Caution: This does not set fields that are not part of the payload and only part of the
    /// [`OpExecutionPayloadSidecar`]:
    /// - parent_beacon_block_root
    ///
    /// See also: [`OpExecutionPayload::into_block_with_sidecar_raw`]
    pub fn into_block_raw(self) -> Result<Block<alloy_primitives::Bytes>, PayloadError> {
        match self {
            Self::V1(payload) => payload.into_block_raw(),
            Self::V2(payload) => payload.into_block_raw(),
            Self::V3(payload) => payload.into_block_raw(),
            Self::V4(payload) => payload.into_block_raw(),
        }
    }

    /// Creates a new unsealed block from the given payload and payload sidecar with raw
    /// transactions.
    ///
    /// This sets the `parent_beacon_block_root` and `requests_hash` if present in the sidecar.
    /// Also validates that L1 withdrawals are empty.
    ///
    /// See also: [`OpExecutionPayload::try_into_block_with_sidecar`]
    pub fn into_block_with_sidecar_raw(
        self,
        sidecar: &OpExecutionPayloadSidecar,
    ) -> Result<Block<alloy_primitives::Bytes>, OpPayloadError> {
        if let Some(payload) = self.as_v2()
            && !payload.withdrawals.is_empty()
        {
            return Err(OpPayloadError::NonEmptyL1Withdrawals);
        }

        let mut block = self.into_block_raw()?;

        if let Some(blobs_hashes) = sidecar.versioned_hashes()
            && !blobs_hashes.is_empty()
        {
            return Err(OpPayloadError::NonEmptyBlobVersionedHashes);
        }
        if let Some(reqs_hash) = sidecar.requests_hash() {
            if reqs_hash != EMPTY_REQUESTS_HASH {
                return Err(OpPayloadError::NonEmptyELRequests);
            }
            block.header.requests_hash = Some(EMPTY_REQUESTS_HASH)
        }
        block.header.parent_beacon_block_root = sidecar.parent_beacon_block_root();

        Ok(block)
    }

    #[allow(rustdoc::broken_intra_doc_links)]
    /// Converts [`OpExecutionPayload`] to [`Block`].
    ///
    /// Checks that payload doesn't contain:
    /// - blob transactions
    /// - L1 withdrawals
    ///
    /// Caution: This does not set fields that are not part of the payload and only part of the
    /// [`OpExecutionPayloadSidecar`]:
    /// - parent_beacon_block_root
    ///
    /// See also: [`OpExecutionPayload::try_into_block_with_sidecar`]
    pub fn try_into_block<T: Decodable2718 + Typed2718>(self) -> Result<Block<T>, OpPayloadError> {
        self.try_into_block_with(|tx| {
            T::decode_2718_exact(tx.as_ref())
                .map_err(alloy_rlp::Error::from)
                .map_err(PayloadError::from)
        })
    }

    #[allow(rustdoc::broken_intra_doc_links)]
    /// Converts [`OpExecutionPayload`] to [`Block`] with a custom transaction mapper.
    ///
    /// Checks that payload doesn't contain:
    /// - blob transactions
    /// - L1 withdrawals
    ///
    /// Caution: This does not set fields that are not part of the payload and only part of the
    /// [`OpExecutionPayloadSidecar`]:
    /// - parent_beacon_block_root
    ///
    /// See also: [`OpExecutionPayload::try_into_block_with_sidecar_with`]
    pub fn try_into_block_with<T, F, E>(self, f: F) -> Result<Block<T>, OpPayloadError>
    where
        T: Typed2718,
        F: FnMut(alloy_primitives::Bytes) -> Result<T, E>,
        E: Into<PayloadError>,
    {
        if let Some(payload) = self.as_v2()
            && !payload.withdrawals.is_empty()
        {
            return Err(OpPayloadError::NonEmptyL1Withdrawals);
        }
        let block = match self {
            Self::V1(payload) => return Ok(payload.try_into_block_with(f)?),
            Self::V2(payload) => return Ok(payload.try_into_block_with(f)?),
            Self::V3(payload) => payload.try_into_block_with(f)?,
            Self::V4(payload) => payload.try_into_block_with(f)?,
        };
        if block.body.has_eip4844_transactions() {
            return Err(OpPayloadError::BlobTransaction);
        }

        Ok(block)
    }

    /// Tries to create a new unsealed block from the given payload and payload sidecar.
    ///
    /// Additional to checks performed in [`OpExecutionPayload::try_into_block`], which is called
    /// under the hood, also checks that sidecar doesn't contain:
    /// - blob versioned hashes
    /// - execution layer requests
    ///
    /// See also docs for
    /// [`ExecutionPayload::try_into_block_with_sidecar`](alloy_rpc_types_engine::ExecutionPayload::try_into_block_with_sidecar).
    pub fn try_into_block_with_sidecar<T: Decodable2718 + Typed2718>(
        self,
        sidecar: &OpExecutionPayloadSidecar,
    ) -> Result<Block<T>, OpPayloadError> {
        self.try_into_block_with_sidecar_with(sidecar, |tx| {
            T::decode_2718_exact(tx.as_ref())
                .map_err(alloy_rlp::Error::from)
                .map_err(PayloadError::from)
        })
    }

    /// Tries to create a new unsealed block from the given payload and payload sidecar with a
    /// custom transaction mapper.
    ///
    /// Additional to checks performed in [`OpExecutionPayload::try_into_block_with`], which is
    /// called under the hood, also checks that sidecar doesn't contain:
    /// - blob versioned hashes
    /// - execution layer requests
    ///
    /// See also docs for
    /// [`ExecutionPayload::try_into_block_with_sidecar_with`](alloy_rpc_types_engine::ExecutionPayload::try_into_block_with_sidecar_with).
    pub fn try_into_block_with_sidecar_with<T, F, E>(
        self,
        sidecar: &OpExecutionPayloadSidecar,
        f: F,
    ) -> Result<Block<T>, OpPayloadError>
    where
        T: Typed2718,
        F: FnMut(alloy_primitives::Bytes) -> Result<T, E>,
        E: Into<PayloadError>,
    {
        let mut base_payload = self.try_into_block_with(f)?;
        if let Some(blobs_hashes) = sidecar.versioned_hashes()
            && !blobs_hashes.is_empty()
        {
            return Err(OpPayloadError::NonEmptyBlobVersionedHashes);
        }
        if let Some(reqs_hash) = sidecar.requests_hash() {
            if reqs_hash != EMPTY_REQUESTS_HASH {
                return Err(OpPayloadError::NonEmptyELRequests);
            }
            base_payload.header.requests_hash = Some(EMPTY_REQUESTS_HASH)
        }
        base_payload.header.parent_beacon_block_root = sidecar.parent_beacon_block_root();

        Ok(base_payload)
    }

    /// Returns an iterator over the decoded transactions in this payload.
    ///
    /// This iterator will decode transactions on the fly.
    pub fn decoded_transactions<T: Decodable2718>(
        &self,
    ) -> impl Iterator<Item = alloy_eips::eip2718::Eip2718Result<T>> + '_ {
        self.transactions().iter().map(|tx_bytes| T::decode_2718_exact(tx_bytes.as_ref()))
    }

    /// Returns iterator over decoded transactions with their original encoded bytes.
    ///
    /// This iterator will decode transactions on the fly and return them with their bytes.
    pub fn decoded_transactions_with_encoded<T: Decodable2718>(
        &self,
    ) -> impl Iterator<Item = alloy_eips::eip2718::Eip2718Result<alloy_eips::eip2718::WithEncoded<T>>> + '_
    {
        self.transactions().iter().map(|tx_bytes| {
            T::decode_2718_exact(tx_bytes.as_ref())
                .map(|tx| alloy_eips::eip2718::WithEncoded::new(tx_bytes.clone(), tx))
        })
    }

    /// Returns an iterator over the recovered transactions in this payload.
    ///
    /// This iterator will decode and recover signer addresses for transactions on the fly.
    pub fn recovered_transactions<T>(
        &self,
    ) -> impl Iterator<
        Item = Result<
            alloy_consensus::transaction::Recovered<T>,
            alloy_consensus::crypto::RecoveryError,
        >,
    > + '_
    where
        T: Decodable2718 + alloy_consensus::transaction::SignerRecoverable,
    {
        self.decoded_transactions::<T>().map(|res| {
            res.map_err(alloy_consensus::crypto::RecoveryError::from_source)
                .and_then(|tx| tx.try_into_recovered())
        })
    }

    /// Returns an iterator over the recovered transactions in this payload with their
    /// original encoded bytes.
    ///
    /// This iterator will decode and recover signer addresses for transactions on the fly
    /// and return them with their bytes.
    pub fn recovered_transactions_with_encoded<T>(
        &self,
    ) -> impl Iterator<
        Item = Result<
            alloy_eips::eip2718::WithEncoded<alloy_consensus::transaction::Recovered<T>>,
            alloy_consensus::crypto::RecoveryError,
        >,
    > + '_
    where
        T: Decodable2718 + alloy_consensus::transaction::SignerRecoverable,
    {
        self.transactions().iter().map(|tx_bytes| {
            T::decode_2718_exact(tx_bytes.as_ref())
                .map_err(alloy_consensus::crypto::RecoveryError::from_source)
                .and_then(|tx| {
                    tx.try_into_recovered().map(|recovered| {
                        alloy_eips::eip2718::WithEncoded::new(tx_bytes.clone(), recovered)
                    })
                })
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    #[cfg(feature = "serde")]
    fn serde_payload_input_enum_v4() {
        let response_v4 = r#"{"parentHash":"0xe927a1448525fb5d32cb50ee1408461a945ba6c39bd5cf5621407d500ecc8de9","feeRecipient":"0x0000000000000000000000000000000000000000","stateRoot":"0x10f8a0830000e8edef6d00cc727ff833f064b1950afd591ae41357f97e543119","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","prevRandao":"0xe0d8b4521a7da1582a713244ffb6a86aa1726932087386e2dc7973f43fc6cb24","blockNumber":"0x1","gasLimit":"0x2ffbd2","gasUsed":"0x0","timestamp":"0x1235","extraData":"0xd883010d00846765746888676f312e32312e30856c696e7578","baseFeePerGas":"0x342770c0","blockHash":"0x44d0fa5f2f73a938ebb96a2a21679eb8dea3e7b7dd8fd9f35aa756dda8bf0a8a","transactions":[],"withdrawals":[],"blobGasUsed":"0x0","excessBlobGas":"0x0","withdrawalsRoot":"0x10f8a0830000e8edef6d00cc727ff833f064b1950afd591ae41357f97e543119"}"#;

        let payload: OpExecutionPayload = serde_json::from_str(response_v4).unwrap();
        assert!(payload.as_v4().is_some());
        assert_eq!(serde_json::to_string(&payload).unwrap(), response_v4);

        let payload_v4: OpExecutionPayloadV4 = serde_json::from_str(response_v4).unwrap();
        assert_eq!(payload.as_v4().unwrap(), &payload_v4);
    }

    #[test]
    #[cfg(feature = "serde")]
    fn serde_payload_input_enum_v3() {
        let response_v3 = r#"{"parentHash":"0xe927a1448525fb5d32cb50ee1408461a945ba6c39bd5cf5621407d500ecc8de9","feeRecipient":"0x0000000000000000000000000000000000000000","stateRoot":"0x10f8a0830000e8edef6d00cc727ff833f064b1950afd591ae41357f97e543119","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","prevRandao":"0xe0d8b4521a7da1582a713244ffb6a86aa1726932087386e2dc7973f43fc6cb24","blockNumber":"0x1","gasLimit":"0x2ffbd2","gasUsed":"0x0","timestamp":"0x1235","extraData":"0xd883010d00846765746888676f312e32312e30856c696e7578","baseFeePerGas":"0x342770c0","blockHash":"0x44d0fa5f2f73a938ebb96a2a21679eb8dea3e7b7dd8fd9f35aa756dda8bf0a8a","transactions":[],"withdrawals":[],"blobGasUsed":"0x0","excessBlobGas":"0x0"}"#;

        let payload: OpExecutionPayload = serde_json::from_str(response_v3).unwrap();
        assert!(payload.as_v3().is_some());
        assert_eq!(serde_json::to_string(&payload).unwrap(), response_v3);

        let payload_v3: ExecutionPayloadV3 = serde_json::from_str(response_v3).unwrap();
        assert_eq!(payload.as_v3().unwrap(), &payload_v3);
    }

    #[test]
    #[cfg(feature = "serde")]
    fn serde_payload_input_enum_v2() {
        let response_v2 = r#"{"parentHash":"0xe927a1448525fb5d32cb50ee1408461a945ba6c39bd5cf5621407d500ecc8de9","feeRecipient":"0x0000000000000000000000000000000000000000","stateRoot":"0x10f8a0830000e8edef6d00cc727ff833f064b1950afd591ae41357f97e543119","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","prevRandao":"0xe0d8b4521a7da1582a713244ffb6a86aa1726932087386e2dc7973f43fc6cb24","blockNumber":"0x1","gasLimit":"0x2ffbd2","gasUsed":"0x0","timestamp":"0x1235","extraData":"0xd883010d00846765746888676f312e32312e30856c696e7578","baseFeePerGas":"0x342770c0","blockHash":"0x44d0fa5f2f73a938ebb96a2a21679eb8dea3e7b7dd8fd9f35aa756dda8bf0a8a","transactions":[],"withdrawals":[]}"#;

        let payload: OpExecutionPayload = serde_json::from_str(response_v2).unwrap();
        assert!(payload.as_v3().is_none());
        assert_eq!(serde_json::to_string(&payload).unwrap(), response_v2);

        let payload_v2: ExecutionPayloadV2 = serde_json::from_str(response_v2).unwrap();
        assert_eq!(payload.as_v2(), Some(&payload_v2));
    }

    #[test]
    #[cfg(feature = "serde")]
    fn serde_payload_input_enum_faulty_v2() {
        // incomplete V3 payload should be rejected even if it has all V2 fields
        let response_faulty = r#"{"parentHash":"0xe927a1448525fb5d32cb50ee1408461a945ba6c39bd5cf5621407d500ecc8de9","feeRecipient":"0x0000000000000000000000000000000000000000","stateRoot":"0x10f8a0830000e8edef6d00cc727ff833f064b1950afd591ae41357f97e543119","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","prevRandao":"0xe0d8b4521a7da1582a713244ffb6a86aa1726932087386e2dc7973f43fc6cb24","blockNumber":"0x1","gasLimit":"0x2ffbd2","gasUsed":"0x0","timestamp":"0x1235","extraData":"0xd883010d00846765746888676f312e32312e30856c696e7578","baseFeePerGas":"0x342770c0","blockHash":"0x44d0fa5f2f73a938ebb96a2a21679eb8dea3e7b7dd8fd9f35aa756dda8bf0a8a","transactions":[],"withdrawals":[], "blobGasUsed": "0x0"}"#;

        let payload: Result<OpExecutionPayload, serde_json::Error> =
            serde_json::from_str(response_faulty);
        assert!(payload.is_err());
    }

    #[test]
    #[cfg(feature = "serde")]
    fn serde_payload_input_enum_faulty_v1() {
        // incomplete V3 payload should be rejected even if it has all V1 fields
        let response_faulty = r#"{"parentHash":"0xe927a1448525fb5d32cb50ee1408461a945ba6c39bd5cf5621407d500ecc8de9","feeRecipient":"0x0000000000000000000000000000000000000000","stateRoot":"0x10f8a0830000e8edef6d00cc727ff833f064b1950afd591ae41357f97e543119","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","prevRandao":"0xe0d8b4521a7da1582a713244ffb6a86aa1726932087386e2dc7973f43fc6cb24","blockNumber":"0x1","gasLimit":"0x2ffbd2","gasUsed":"0x0","timestamp":"0x1235","extraData":"0xd883010d00846765746888676f312e32312e30856c696e7578","baseFeePerGas":"0x342770c0","blockHash":"0x44d0fa5f2f73a938ebb96a2a21679eb8dea3e7b7dd8fd9f35aa756dda8bf0a8a","transactions":[],"blobGasUsed": "0x0"}"#;

        let payload: Result<OpExecutionPayload, serde_json::Error> =
            serde_json::from_str(response_faulty);
        assert!(payload.is_err());
    }
}
