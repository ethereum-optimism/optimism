use crate::{
    OpExecutionData, OpExecutionPayload, OpExecutionPayloadEnvelope, OpExecutionPayloadV4,
    OpFlashblockError, OpFlashblockPayload, OpPayloadError,
};
use alloc::vec::Vec;
use alloy_consensus::{Block, BlockHeader, Sealable, Transaction};
use alloy_eips::{
    Decodable2718, Encodable2718, Typed2718,
    eip4895::Withdrawal,
    eip7685::{EMPTY_REQUESTS_HASH, Requests},
};
use alloy_primitives::{B256, Bytes};
use alloy_rpc_types_engine::{
    ExecutionPayloadV1, ExecutionPayloadV2, ExecutionPayloadV3, PayloadError,
};

impl OpExecutionPayloadEnvelope {
    /// Converts this payload envelope into a block with decoded transactions.
    pub fn try_into_block<T>(self) -> Result<Block<T>, OpPayloadError>
    where
        T: Decodable2718 + Typed2718,
    {
        let check_blob_transactions = matches!(&self, Self::V3 { .. } | Self::V4 { .. });
        let block = self.try_into_block_raw()?.try_map_transactions(|tx| {
            T::decode_2718_exact(tx.as_ref())
                .map_err(alloy_rlp::Error::from)
                .map_err(PayloadError::from)
        })?;
        if check_blob_transactions && block.body.has_eip4844_transactions() {
            return Err(OpPayloadError::BlobTransaction);
        }

        Ok(block)
    }

    /// Verifies the claimed block hash using the raw encoded transaction bytes.
    ///
    /// This computes the transaction trie root without decoding or validating the transactions.
    /// Full transaction and execution validation remains the responsibility of the execution
    /// layer or [`Self::try_into_checked_block`].
    pub fn check_block_hash(&self) -> Result<(), OpPayloadError> {
        let consensus = self.block_hash();
        let block = self.clone().try_into_block_raw()?;
        let execution = block.header.hash_slow();
        if execution != consensus {
            return Err(PayloadError::BlockHash { execution, consensus }.into());
        }

        Ok(())
    }

    /// Converts this payload envelope into a block and verifies its claimed block hash.
    pub fn try_into_checked_block<T>(self) -> Result<Block<T>, OpPayloadError>
    where
        T: Decodable2718 + Typed2718,
    {
        let consensus = self.block_hash();
        let block = self.try_into_block()?;
        let execution = block.header.hash_slow();
        if execution != consensus {
            return Err(PayloadError::BlockHash { execution, consensus }.into());
        }

        Ok(block)
    }

    /// Converts a block into a payload envelope and recalculates its block hash.
    pub fn from_block_slow<T, H>(block: &Block<T, H>) -> Result<Self, OpPayloadError>
    where
        T: Encodable2718 + Transaction,
        H: BlockHeader + Sealable,
    {
        let (payload, sidecar) = OpExecutionPayload::from_block_slow(block);
        OpExecutionData::new(payload, sidecar).try_into()
    }

    /// Converts a block into a payload envelope with the supplied block hash.
    pub fn from_block_unchecked<T, H>(
        block_hash: B256,
        block: &Block<T, H>,
    ) -> Result<Self, OpPayloadError>
    where
        T: Encodable2718 + Transaction,
        H: BlockHeader,
    {
        let (payload, sidecar) = OpExecutionPayload::from_block_unchecked(block_hash, block);
        OpExecutionData::new(payload, sidecar).try_into()
    }

    /// Converts a sequence of flashblocks into a payload envelope.
    pub fn from_flashblocks(
        flashblocks: &[OpFlashblockPayload],
    ) -> Result<Self, OpFlashblockError> {
        if flashblocks.is_empty() {
            return Err(OpFlashblockError::MissingPayload);
        }
        for (i, flashblock) in flashblocks.iter().enumerate() {
            if flashblock.index as usize != i {
                return Err(OpFlashblockError::InvalidIndex);
            }
        }
        let first = flashblocks.first().expect("flashblocks must not be empty");
        if first.base.is_none() {
            return Err(OpFlashblockError::MissingBasePayload);
        }
        if flashblocks.iter().skip(1).any(|flashblock| flashblock.base.is_some()) {
            return Err(OpFlashblockError::UnexpectedBasePayload);
        }

        Ok(Self::from_flashblocks_unchecked(flashblocks))
    }

    /// Converts a sequence of flashblocks into a payload envelope without validation.
    ///
    /// # Panics
    ///
    /// Panics if the sequence is empty or its first flashblock has no base payload.
    pub fn from_flashblocks_unchecked(flashblocks: &[OpFlashblockPayload]) -> Self {
        let first = flashblocks.first().expect("flashblocks must not be empty");
        let base = first.base.as_ref().expect("first flashblock must have base payload");
        let diff = &flashblocks.last().expect("flashblocks must not be empty").diff;

        let (mut transactions, withdrawals) = flashblocks.iter().fold(
            (Vec::new(), Vec::new()),
            |(mut transactions, mut withdrawals), payload| {
                transactions.extend(payload.diff.transactions.iter().cloned());
                withdrawals.extend(payload.diff.withdrawals.iter().copied());
                (transactions, withdrawals)
            },
        );

        if let Some(post_exec_tx) = diff.post_exec_tx.as_ref() {
            transactions.push(post_exec_tx.clone());
        }

        let payload = ExecutionPayloadV3 {
            blob_gas_used: diff.blob_gas_used.unwrap_or(0),
            excess_blob_gas: 0,
            payload_inner: ExecutionPayloadV2 {
                withdrawals,
                payload_inner: ExecutionPayloadV1 {
                    parent_hash: base.parent_hash,
                    fee_recipient: base.fee_recipient,
                    state_root: diff.state_root,
                    receipts_root: diff.receipts_root,
                    logs_bloom: diff.logs_bloom,
                    prev_randao: base.prev_randao,
                    block_number: base.block_number,
                    gas_limit: base.gas_limit,
                    gas_used: diff.gas_used,
                    timestamp: base.timestamp,
                    extra_data: base.extra_data.clone(),
                    base_fee_per_gas: base.base_fee_per_gas,
                    block_hash: diff.block_hash,
                    transactions,
                },
            },
        };

        if diff.withdrawals_root == B256::ZERO {
            return Self::V3 { payload, parent_beacon_block_root: base.parent_beacon_block_root };
        }

        Self::V4 {
            payload: OpExecutionPayloadV4 {
                withdrawals_root: diff.withdrawals_root,
                payload_inner: payload,
            },
            parent_beacon_block_root: base.parent_beacon_block_root,
        }
    }

    /// Converts this payload envelope into a complete block with raw transactions.
    fn try_into_block_raw(self) -> Result<Block<Bytes>, OpPayloadError> {
        if self.withdrawals().is_some_and(|withdrawals| !withdrawals.is_empty()) {
            return Err(OpPayloadError::NonEmptyL1Withdrawals);
        }

        let block = match self {
            Self::V1(payload) => payload.into_block_raw()?,
            Self::V2(payload) => payload.into_block_raw()?,
            Self::V3 { payload, parent_beacon_block_root } => {
                let mut block = payload.into_block_raw()?;
                block.header.parent_beacon_block_root = Some(parent_beacon_block_root);
                block
            }
            Self::V4 { payload, parent_beacon_block_root } => {
                let mut block = payload.payload_inner.into_block_raw()?;
                block.header.withdrawals_root = Some(payload.withdrawals_root);
                block.header.parent_beacon_block_root = Some(parent_beacon_block_root);
                block.header.requests_hash = Some(EMPTY_REQUESTS_HASH);
                block
            }
        };

        Ok(block)
    }

    /// Returns the V1 payload fields shared by every payload version.
    pub const fn as_v1(&self) -> &ExecutionPayloadV1 {
        match self {
            Self::V1(payload) => payload,
            Self::V2(payload) => &payload.payload_inner,
            Self::V3 { payload, .. } => &payload.payload_inner.payload_inner,
            Self::V4 { payload, .. } => &payload.payload_inner.payload_inner.payload_inner,
        }
    }

    /// Returns the mutable V1 payload fields shared by every payload version.
    pub const fn as_v1_mut(&mut self) -> &mut ExecutionPayloadV1 {
        match self {
            Self::V1(payload) => payload,
            Self::V2(payload) => &mut payload.payload_inner,
            Self::V3 { payload, .. } => &mut payload.payload_inner.payload_inner,
            Self::V4 { payload, .. } => &mut payload.payload_inner.payload_inner.payload_inner,
        }
    }

    /// Returns the V4 payload, if present.
    pub const fn as_v4(&self) -> Option<&OpExecutionPayloadV4> {
        match self {
            Self::V4 { payload, .. } => Some(payload),
            _ => None,
        }
    }

    /// Returns the transactions.
    pub const fn transactions(&self) -> &Vec<alloy_primitives::Bytes> {
        &self.as_v1().transactions
    }

    /// Returns the parent beacon block root for V3 and V4 payloads.
    pub const fn parent_beacon_block_root(&self) -> Option<B256> {
        match self {
            Self::V1(_) | Self::V2(_) => None,
            Self::V3 { parent_beacon_block_root, .. } |
            Self::V4 { parent_beacon_block_root, .. } => Some(*parent_beacon_block_root),
        }
    }

    /// Returns the withdrawals for the payload.
    pub const fn withdrawals(&self) -> Option<&Vec<Withdrawal>> {
        match self {
            Self::V1(_) => None,
            Self::V2(payload) => Some(&payload.withdrawals),
            Self::V3 { payload, .. } => Some(payload.withdrawals()),
            Self::V4 { payload, .. } => Some(payload.payload_inner.withdrawals()),
        }
    }

    /// Returns the parent hash.
    pub const fn parent_hash(&self) -> B256 {
        self.as_v1().parent_hash
    }

    /// Returns the claimed block hash.
    pub const fn block_hash(&self) -> B256 {
        self.as_v1().block_hash
    }

    /// Returns the block number.
    pub const fn block_number(&self) -> u64 {
        self.as_v1().block_number
    }

    /// Returns the timestamp.
    pub const fn timestamp(&self) -> u64 {
        self.as_v1().timestamp
    }

    /// Returns the fee recipient.
    pub const fn fee_recipient(&self) -> alloy_primitives::Address {
        self.as_v1().fee_recipient
    }

    /// Returns the gas limit.
    pub const fn gas_limit(&self) -> u64 {
        self.as_v1().gas_limit
    }
}

impl TryFrom<OpExecutionData> for OpExecutionPayloadEnvelope {
    type Error = OpPayloadError;

    fn try_from(data: OpExecutionData) -> Result<Self, Self::Error> {
        let parent_beacon_block_root = data.sidecar.parent_beacon_block_root();
        let versioned_hashes = data.sidecar.versioned_hashes();
        let requests_hash = data.sidecar.requests_hash();

        match data.payload {
            OpExecutionPayload::V1(payload) => {
                ensure_pre_ecotone_sidecar(parent_beacon_block_root, requests_hash)?;
                Ok(Self::V1(payload))
            }
            OpExecutionPayload::V2(payload) => {
                ensure_pre_ecotone_sidecar(parent_beacon_block_root, requests_hash)?;
                Ok(Self::V2(payload))
            }
            OpExecutionPayload::V3(payload) => {
                let parent_beacon_block_root =
                    parent_beacon_block_root.ok_or(OpPayloadError::MissingParentBeaconBlockRoot)?;
                ensure_empty_versioned_hashes(versioned_hashes)?;
                if requests_hash.is_some() {
                    return Err(OpPayloadError::UnexpectedELRequests);
                }
                Ok(Self::V3 { payload, parent_beacon_block_root })
            }
            OpExecutionPayload::V4(payload) => {
                let parent_beacon_block_root =
                    parent_beacon_block_root.ok_or(OpPayloadError::MissingParentBeaconBlockRoot)?;
                ensure_empty_versioned_hashes(versioned_hashes)?;
                let requests_hash = requests_hash.ok_or(OpPayloadError::MissingELRequests)?;
                if requests_hash != EMPTY_REQUESTS_HASH {
                    return Err(OpPayloadError::NonEmptyELRequests);
                }
                Ok(Self::V4 { payload, parent_beacon_block_root })
            }
        }
    }
}

impl TryFrom<&OpExecutionData> for OpExecutionPayloadEnvelope {
    type Error = OpPayloadError;

    fn try_from(data: &OpExecutionData) -> Result<Self, Self::Error> {
        data.clone().try_into()
    }
}

impl From<OpExecutionPayloadEnvelope> for OpExecutionData {
    fn from(data: OpExecutionPayloadEnvelope) -> Self {
        match data {
            OpExecutionPayloadEnvelope::V1(payload) => {
                Self::new(OpExecutionPayload::V1(payload), Default::default())
            }
            OpExecutionPayloadEnvelope::V2(payload) => {
                Self::new(OpExecutionPayload::V2(payload), Default::default())
            }
            OpExecutionPayloadEnvelope::V3 { payload, parent_beacon_block_root } => {
                Self::v3(payload, Vec::new(), parent_beacon_block_root)
            }
            OpExecutionPayloadEnvelope::V4 { payload, parent_beacon_block_root } => {
                Self::v4(payload, Vec::new(), parent_beacon_block_root, Requests::default())
            }
        }
    }
}

const fn ensure_pre_ecotone_sidecar(
    parent_beacon_block_root: Option<B256>,
    requests_hash: Option<B256>,
) -> Result<(), OpPayloadError> {
    if parent_beacon_block_root.is_some() {
        return Err(OpPayloadError::UnexpectedParentBeaconBlockRoot);
    }
    if requests_hash.is_some() {
        return Err(OpPayloadError::UnexpectedELRequests);
    }
    Ok(())
}

fn ensure_empty_versioned_hashes(
    versioned_hashes: Option<&Vec<B256>>,
) -> Result<(), OpPayloadError> {
    let versioned_hashes = versioned_hashes.ok_or(OpPayloadError::MissingParentBeaconBlockRoot)?;
    if !versioned_hashes.is_empty() {
        return Err(OpPayloadError::NonEmptyBlobVersionedHashes);
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::OpExecutionPayloadSidecar;
    use alloy_primitives::Bytes;
    use alloy_rpc_types_engine::{CancunPayloadFields, PraguePayloadFields};
    use op_alloy_consensus::OpTxEnvelope;

    fn payload_v1() -> ExecutionPayloadV1 {
        ExecutionPayloadV1::from_block_unchecked(B256::ZERO, &Block::<OpTxEnvelope>::default())
    }

    fn payload_v3() -> ExecutionPayloadV3 {
        ExecutionPayloadV3::from_block_unchecked(B256::ZERO, &Block::<OpTxEnvelope>::default())
    }

    #[test]
    fn rejects_missing_v3_parent_beacon_block_root() {
        let payload = OpExecutionPayload::V3(payload_v3());
        let data = OpExecutionData::new(payload, OpExecutionPayloadSidecar::default());
        assert!(matches!(
            OpExecutionPayloadEnvelope::try_from(data),
            Err(OpPayloadError::MissingParentBeaconBlockRoot)
        ));

        assert!(matches!(
            OpExecutionPayloadEnvelope::try_from_payload(
                OpExecutionPayload::V3(payload_v3()),
                None,
            ),
            Err(OpPayloadError::MissingParentBeaconBlockRoot)
        ));
    }

    #[test]
    fn distinguishes_present_zero_parent_beacon_block_root() {
        let data = OpExecutionData::v3(payload_v3(), Vec::new(), B256::ZERO);
        let envelope = OpExecutionPayloadEnvelope::try_from(data).unwrap();
        assert!(matches!(
            envelope,
            OpExecutionPayloadEnvelope::V3 { parent_beacon_block_root: B256::ZERO, .. }
        ));
        assert_eq!(envelope.parent_beacon_block_root(), Some(B256::ZERO));

        let data: OpExecutionData = envelope.clone().into();
        assert_eq!(data.sidecar.parent_beacon_block_root(), Some(B256::ZERO));
        assert_eq!(OpExecutionPayloadEnvelope::try_from(data).unwrap(), envelope);
    }

    #[test]
    fn rejects_unexpected_v1_parent_beacon_block_root() {
        let payload = OpExecutionPayload::V1(payload_v1());
        let sidecar =
            OpExecutionPayloadSidecar::v3(CancunPayloadFields::new(B256::ZERO, Vec::new()));
        assert!(matches!(
            OpExecutionPayloadEnvelope::try_from(OpExecutionData::new(payload, sidecar)),
            Err(OpPayloadError::UnexpectedParentBeaconBlockRoot)
        ));

        assert!(matches!(
            OpExecutionPayloadEnvelope::try_from_payload(
                OpExecutionPayload::V1(payload_v1()),
                Some(B256::ZERO),
            ),
            Err(OpPayloadError::UnexpectedParentBeaconBlockRoot)
        ));
    }

    #[test]
    fn rejects_non_empty_blob_versioned_hashes() {
        let data = OpExecutionData::v3(payload_v3(), vec![B256::ZERO], B256::ZERO);
        assert!(matches!(
            OpExecutionPayloadEnvelope::try_from(data),
            Err(OpPayloadError::NonEmptyBlobVersionedHashes)
        ));
    }

    #[test]
    fn rejects_non_empty_execution_requests() {
        let payload =
            OpExecutionPayloadV4 { payload_inner: payload_v3(), withdrawals_root: B256::ZERO };
        let requests = Requests::new(vec![Bytes::from_static(&[0x01, 0x02])]);
        let data = OpExecutionData::v4(payload, Vec::new(), B256::ZERO, requests);
        assert!(matches!(
            OpExecutionPayloadEnvelope::try_from(data),
            Err(OpPayloadError::NonEmptyELRequests)
        ));
    }

    #[test]
    fn rejects_mismatched_v3_prague_fields() {
        let payload = OpExecutionPayload::V3(payload_v3());
        let sidecar = OpExecutionPayloadSidecar::v4(
            CancunPayloadFields::new(B256::ZERO, Vec::new()),
            PraguePayloadFields::new(Requests::default()),
        );
        assert!(matches!(
            OpExecutionPayloadEnvelope::try_from(OpExecutionData::new(payload, sidecar)),
            Err(OpPayloadError::UnexpectedELRequests)
        ));
    }

    #[test]
    fn rejects_v4_without_prague_fields() {
        let payload = OpExecutionPayload::V4(OpExecutionPayloadV4 {
            payload_inner: payload_v3(),
            withdrawals_root: B256::ZERO,
        });
        let sidecar =
            OpExecutionPayloadSidecar::v3(CancunPayloadFields::new(B256::ZERO, Vec::new()));
        assert!(matches!(
            OpExecutionPayloadEnvelope::try_from(OpExecutionData::new(payload, sidecar)),
            Err(OpPayloadError::MissingELRequests)
        ));
    }

    #[test]
    fn check_block_hash_does_not_decode_transactions() {
        let mut envelope = OpExecutionPayloadEnvelope::V1(payload_v1());
        envelope.as_v1_mut().transactions = vec![Bytes::from_static(&[0xff])];
        let block_hash = envelope.clone().try_into_block_raw().unwrap().header.hash_slow();
        envelope.as_v1_mut().block_hash = block_hash;

        assert!(envelope.check_block_hash().is_ok());
        assert!(envelope.try_into_block::<OpTxEnvelope>().is_err());
    }
}
