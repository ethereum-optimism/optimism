//! Flashblock payload types.

use super::{OpFlashblockPayloadBase, OpFlashblockPayloadDelta};
use crate::flashblock::metadata::OpFlashblockPayloadMetadata;
use alloy_eips::{Decodable2718, eip2718::Eip2718Result};
use alloy_primitives::{B256, Bytes};
use alloy_rpc_types_engine::PayloadId;

/// Flashblock payload.
///
/// Represents a Flashblock, a real-time block-like structure emitted by the Base L2 chain.
/// A Flashblock provides a snapshot of a block's effects before finalization,
/// allowing faster insight into state transitions, balance changes, and logs.
///
/// See: [Base Flashblocks Documentation](https://docs.base.org/chain/flashblocks)
#[derive(Clone, Debug, Default, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct OpFlashblockPayload {
    /// The unique payload ID as assigned by the execution engine for this block.
    pub payload_id: PayloadId,
    /// A sequential index that identifies the order of this Flashblock.
    pub index: u64,
    /// Immutable block properties shared across all flashblocks in the sequence.
    /// This is `None` for all flashblocks except the first in a sequence.
    #[cfg_attr(feature = "serde", serde(skip_serializing_if = "Option::is_none"))]
    pub base: Option<OpFlashblockPayloadBase>,
    /// Accumulating and changing block properties for this flashblock.
    pub diff: OpFlashblockPayloadDelta,
    /// Additional metadata about the flashblock such as receipts and balance changes.
    pub metadata: OpFlashblockPayloadMetadata,
}

impl OpFlashblockPayload {
    /// Returns the block number of this flashblock.
    pub const fn block_number(&self) -> u64 {
        self.metadata.block_number
    }

    /// Returns the first parent hash of this flashblock.
    pub fn parent_hash(&self) -> Option<B256> {
        Some(self.base.as_ref()?.parent_hash)
    }

    /// Returns the raw transactions in this flashblock.
    pub fn raw_transactions(&self) -> &[Bytes] {
        &self.diff.transactions
    }

    /// Returns an iterator over the decoded transaction in this flashblock.
    ///
    /// This iterator will be empty if there are no transactions in this flashblock.
    pub fn decoded_transaction<T>(&self) -> impl Iterator<Item = Eip2718Result<T>> + '_
    where
        T: Decodable2718,
    {
        self.raw_transactions().iter().map(|tx| T::decode_2718_exact(tx))
    }

    /// Recovers transactions from flashblocks lazily.
    ///
    /// This is done only when we actually need to build a sequence, avoiding wasted computation.
    #[cfg(feature = "k256")]
    pub fn recover_transactions<T>(
        &self,
    ) -> impl Iterator<
        Item = alloy_rlp::Result<
            alloy_eips::eip2718::WithEncoded<alloy_consensus::transaction::Recovered<T>>,
            alloy_consensus::crypto::RecoveryError,
        >,
    > + '_
    where
        T: Decodable2718 + alloy_consensus::transaction::SignerRecoverable,
    {
        self.raw_transactions().iter().map(|raw| {
            let tx = T::decode_2718_exact(raw)
                .map_err(alloy_consensus::crypto::RecoveryError::from_source)?;
            tx.try_into_recovered().map(|tx| tx.into_encoded_with(raw.clone()))
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::flashblock::{
        OpFlashblockPayloadBase, OpFlashblockPayloadDelta, OpFlashblockPayloadMetadata,
    };
    use alloc::{collections::BTreeMap, vec};
    use alloy_primitives::{B256, Bloom, Bytes, U256, address};

    fn sample_payload() -> OpFlashblockPayload {
        let base = OpFlashblockPayloadBase {
            parent_beacon_block_root: B256::ZERO,
            parent_hash: B256::ZERO,
            fee_recipient: address!("0000000000000000000000000000000000000001"),
            prev_randao: B256::ZERO,
            block_number: 100,
            gas_limit: 30_000_000,
            timestamp: 1234567890,
            extra_data: Bytes::default(),
            base_fee_per_gas: U256::from(1000000000u64),
        };

        let diff = OpFlashblockPayloadDelta {
            state_root: B256::ZERO,
            receipts_root: B256::ZERO,
            logs_bloom: Bloom::ZERO,
            gas_used: 21000,
            block_hash: B256::ZERO,
            transactions: vec![],
            withdrawals: vec![],
            withdrawals_root: B256::ZERO,
            blob_gas_used: Some(0),
        };

        let metadata = OpFlashblockPayloadMetadata {
            block_number: 100,
            new_account_balances: BTreeMap::new(),
            receipts: BTreeMap::new(),
        };

        OpFlashblockPayload {
            payload_id: PayloadId::new([1u8; 8]),
            index: 0,
            base: Some(base),
            diff,
            metadata,
        }
    }

    #[test]
    fn test_payload_accessors() {
        let payload = sample_payload();

        // Direct field access via public fields
        assert_eq!(payload.metadata.block_number, 100);
        assert_eq!(payload.base.as_ref().map(|b| b.parent_hash), Some(B256::ZERO));
        assert!(!payload.metadata.receipts.contains_key(&B256::ZERO));
    }

    #[test]
    fn test_payload_without_base() {
        let mut payload = sample_payload();
        payload.base = None;

        // Direct field access via public fields
        assert_eq!(payload.metadata.block_number, 100);
        assert_eq!(payload.base.as_ref().map(|b| b.parent_hash), None);
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_payload_serde_roundtrip() {
        let payload = sample_payload();

        let json = serde_json::to_string(&payload).unwrap();
        let decoded: OpFlashblockPayload = serde_json::from_str(&json).unwrap();
        assert_eq!(payload, decoded);
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_payload_snake_case_serialization() {
        let payload = sample_payload();

        let json = serde_json::to_string(&payload).unwrap();
        assert!(json.contains("payload_id"));
        assert!(json.contains("\"index\""));
        assert!(json.contains("\"base\""));
        assert!(json.contains("\"diff\""));
        assert!(json.contains("\"metadata\""));
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_payload_base_omitted_when_none() {
        let mut payload = sample_payload();
        payload.base = None;

        let json = serde_json::to_string(&payload).unwrap();
        // Base should not be serialized when None (skip_serializing_if)
        assert!(!json.contains("\"base\""));
    }

    #[test]
    fn test_payload_default() {
        let payload = OpFlashblockPayload::default();
        assert_eq!(payload.payload_id, PayloadId::new([0u8; 8]));
        assert_eq!(payload.index, 0);
        assert_eq!(payload.base, None);
        assert_eq!(payload.diff, OpFlashblockPayloadDelta::default());
        assert_eq!(payload.metadata, OpFlashblockPayloadMetadata::default());
    }
}
