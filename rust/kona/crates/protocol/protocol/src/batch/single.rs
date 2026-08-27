//! This module contains the [`SingleBatch`] type.

use crate::{BatchDropReason, BatchValidity, BlockInfo, L2BlockInfo};
use alloc::vec::Vec;
use alloy_eips::BlockNumHash;
use alloy_primitives::{BlockHash, Bytes};
use alloy_rlp::{RlpDecodable, RlpEncodable};
use kona_genesis::RollupConfig;
use op_alloy_consensus::OpTxType;
use tracing::warn;

/// Represents a single batch: a single encoded L2 block
#[derive(Debug, Default, RlpDecodable, RlpEncodable, Clone, PartialEq, Eq)]
pub struct SingleBatch {
    /// Block hash of the previous L2 block. `B256::ZERO` if it has not been set by the Batch
    /// Queue.
    pub parent_hash: BlockHash,
    /// The batch epoch number. Same as the first L1 block number in the epoch.
    pub epoch_num: u64,
    /// The block hash of the first L1 block in the epoch
    pub epoch_hash: BlockHash,
    /// The L2 block timestamp of this batch
    pub timestamp: u64,
    /// The L2 block transactions in this batch
    pub transactions: Vec<Bytes>,
}

impl SingleBatch {
    /// Returns the [`BlockNumHash`] of the batch.
    pub const fn epoch(&self) -> BlockNumHash {
        BlockNumHash { number: self.epoch_num, hash: self.epoch_hash }
    }

    /// Validate the batch timestamp.
    ///
    /// `is_sibling` marks a batch that a span batch flagged as sharing the timestamp of the block
    /// it builds on. It is the only way a batch may repeat its parent's timestamp: a wire singular
    /// batch (`0x00`) cannot express a sibling, so an equal timestamp there is simply an old
    /// batch.
    pub fn check_batch_timestamp(
        &self,
        cfg: &RollupConfig,
        l2_safe_head: L2BlockInfo,
        inclusion_block: &BlockInfo,
        is_sibling: bool,
    ) -> BatchValidity {
        if is_sibling && self.timestamp == l2_safe_head.block_info.timestamp {
            if !cfg.siblings_allowed(self.timestamp) {
                return BatchValidity::Drop(BatchDropReason::SiblingsNotAllowed);
            }
            return BatchValidity::Accept;
        }
        let next_timestamp = l2_safe_head.block_info.timestamp + cfg.block_time;
        if self.timestamp > next_timestamp {
            if cfg.is_holocene_active(inclusion_block.timestamp) {
                return BatchValidity::Drop(BatchDropReason::FutureTimestampHolocene);
            }
            return BatchValidity::Future;
        }
        if self.timestamp < next_timestamp {
            if cfg.is_holocene_active(inclusion_block.timestamp) {
                return BatchValidity::Past;
            }
            return BatchValidity::Drop(BatchDropReason::PastTimestampPreHolocene);
        }
        BatchValidity::Accept
    }

    /// Checks if the batch is valid.
    ///
    /// The batch format type is defined in the [OP Stack Specs][specs].
    ///
    /// [specs]: https://specs.optimism.io/protocol/derivation.html#batch-format
    pub fn check_batch(
        &self,
        cfg: &RollupConfig,
        l1_blocks: &[BlockInfo],
        l2_safe_head: L2BlockInfo,
        inclusion_block: &BlockInfo,
        is_sibling: bool,
    ) -> BatchValidity {
        // Cannot have empty l1_blocks for batch validation.
        if l1_blocks.is_empty() {
            return BatchValidity::Undecided;
        }

        let epoch = l1_blocks[0];

        // If the batch is not accepted by the timestamp check, return the result.
        let timestamp_check =
            self.check_batch_timestamp(cfg, l2_safe_head, inclusion_block, is_sibling);
        if !timestamp_check.is_accept() {
            return timestamp_check;
        }

        // Dependent on the above timestamp check.
        // If the timestamp is correct, then it must build on top of the safe head.
        if self.parent_hash != l2_safe_head.block_info.hash {
            return BatchValidity::Drop(BatchDropReason::ParentHashMismatch);
        }

        // A sibling shares its parent's L1 origin: the whole group is sequenced against one
        // origin, and only the first block of a timestamp may adopt a new one.
        if is_sibling && self.epoch_num != l2_safe_head.l1_origin.number {
            return BatchValidity::Drop(BatchDropReason::SiblingOriginMismatch);
        }

        // Filter out batches that were included too late.
        if self.epoch_num + cfg.seq_window_size < inclusion_block.number {
            return BatchValidity::Drop(BatchDropReason::IncludedTooLate);
        }

        // Check the L1 origin of the batch
        let mut batch_origin = epoch;
        if self.epoch_num < epoch.number {
            return BatchValidity::Drop(BatchDropReason::EpochTooOld);
        } else if self.epoch_num == epoch.number {
            // Batch is sticking to the current epoch, continue.
        } else if self.epoch_num == epoch.number + 1 {
            // With only 1 l1Block we cannot look at the next L1 Origin.
            // Note: This means that we are unable to determine validity of a batch
            // without more information. In this case we should bail out until we have
            // more information otherwise the eager algorithm may diverge from a non-eager
            // algorithm.
            if l1_blocks.len() < 2 {
                return BatchValidity::Undecided;
            }
            batch_origin = l1_blocks[1];
        } else {
            return BatchValidity::Drop(BatchDropReason::EpochTooFarInFuture);
        }

        // Validate the batch epoch hash
        if self.epoch_hash != batch_origin.hash {
            return BatchValidity::Drop(BatchDropReason::EpochHashMismatch);
        }

        if self.timestamp < batch_origin.timestamp {
            return BatchValidity::Drop(BatchDropReason::TimestampBeforeL1Origin);
        }

        // Check if we ran out of sequencer time drift
        let max_drift = cfg.max_sequencer_drift(batch_origin.timestamp);
        let max = if let Some(max) = batch_origin.timestamp.checked_add(max_drift) {
            max
        } else {
            return BatchValidity::Drop(BatchDropReason::SequencerDriftOverflow);
        };

        if self.timestamp > max {
            if !self.transactions.is_empty() {
                // If the sequencer is ignoring the time drift rule, then drop the batch and force
                // an empty batch instead, as the sequencer is not allowed to include anything past
                // this point without moving to the next epoch.
                return BatchValidity::Drop(BatchDropReason::SequencerDriftExceeded);
            }

            // If the sequencer is co-operating by producing an empty batch, allow the batch if it
            // was the right thing to do to maintain the L2 time >= L1 time invariant. Only check
            // batches that do not advance the epoch, to ensure epoch advancement regardless of time
            // drift is allowed.
            if epoch.number == batch_origin.number {
                if l1_blocks.len() < 2 {
                    return BatchValidity::Undecided;
                }
                let next_origin = l1_blocks[1];
                // Check if the next L1 Origin could have been adopted
                if self.timestamp >= next_origin.timestamp {
                    return BatchValidity::Drop(
                        BatchDropReason::SequencerDriftNotAdoptedNextOrigin,
                    );
                }
            }
        }

        // A jovian, karst, or lagoon transition block must be empty; drop it otherwise.
        if (cfg.is_first_jovian_block(self.timestamp) ||
            cfg.is_first_karst_block(self.timestamp) ||
            cfg.is_first_lagoon_block(self.timestamp)) &&
            !self.transactions.is_empty()
        {
            warn!(
                target: "single_batch",
                "Sequencer included user transactions in jovian, karst, or lagoon transition block. Dropping batch."
            );
            return BatchValidity::Drop(BatchDropReason::NonEmptyTransitionBlock);
        }

        // We can do this check earlier, but it's intensive so we do it last for the sad-path.
        for tx in &self.transactions {
            let Some(first_byte) = tx.as_ref().first().copied() else {
                return BatchValidity::Drop(BatchDropReason::EmptyTransaction);
            };
            // A leading byte that doesn't decode to a typed transaction (e.g. a legacy RLP
            // list header) isn't one of the restricted types, so it falls through to `Accept`.
            match OpTxType::try_from(first_byte) {
                Ok(OpTxType::Deposit) => {
                    return BatchValidity::Drop(BatchDropReason::DepositTransaction);
                }
                Ok(OpTxType::Eip7702) if !cfg.is_isthmus_active(self.timestamp) => {
                    return BatchValidity::Drop(BatchDropReason::Eip7702PreIsthmus);
                }
                Ok(OpTxType::PostExec) if !cfg.is_sdm_active(self.timestamp) => {
                    return BatchValidity::Drop(BatchDropReason::PostExecPreLagoon);
                }
                _ => {}
            }
        }

        BatchValidity::Accept
    }
}

#[cfg(test)]
mod tests {
    use crate::test_utils::{CollectingLayer, TraceStorage};

    use super::*;
    use alloc::vec;
    use alloy_consensus::{SignableTransaction, TxEip1559, TxEip7702, TxEnvelope};
    use alloy_eips::eip2718::{Decodable2718, Encodable2718};
    use alloy_primitives::{Address, Sealed, Signature, TxKind, U256};
    use kona_genesis::HardForkConfig;
    use op_alloy_consensus::{
        OpTxEnvelope, POST_EXEC_PAYLOAD_VERSION, PostExecPayload, TxDeposit, TxPostExec,
    };
    use tracing::Level;
    use tracing_subscriber::layer::SubscriberExt;

    #[test]
    fn test_empty_l1_blocks() {
        let cfg = RollupConfig::default();
        let l1_blocks = vec![];
        let l2_safe_head = L2BlockInfo::default();
        let inclusion_block = BlockInfo::default();
        let batch = SingleBatch::default();
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, false),
            BatchValidity::Undecided
        );
    }

    #[test]
    fn test_timestamp_future() {
        let cfg = RollupConfig::default();
        let l1_blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 1, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        let batch = SingleBatch { timestamp: 2, ..Default::default() };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, false),
            BatchValidity::Future
        );
    }

    #[test]
    fn test_parent_hash_mismatch() {
        let cfg = RollupConfig::default();
        let l1_blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { hash: BlockHash::from([0x01; 32]), ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        let batch = SingleBatch { parent_hash: BlockHash::from([0x02; 32]), ..Default::default() };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, false),
            BatchValidity::Drop(BatchDropReason::ParentHashMismatch)
        );
    }

    #[test]
    fn test_check_batch_timestamp_holocene_inactive_future() {
        let cfg = RollupConfig::default();
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 1, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { timestamp: 1, ..Default::default() };
        let batch = SingleBatch { epoch_num: 1, timestamp: 2, ..Default::default() };
        assert_eq!(
            batch.check_batch_timestamp(&cfg, l2_safe_head, &inclusion_block, false),
            BatchValidity::Future
        );
    }

    #[test]
    fn test_check_batch_timestamp_holocene_active_drop() {
        let cfg = RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        };
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 1, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { timestamp: 1, ..Default::default() };
        let batch = SingleBatch { epoch_num: 1, timestamp: 2, ..Default::default() };
        assert_eq!(
            batch.check_batch_timestamp(&cfg, l2_safe_head, &inclusion_block, false),
            BatchValidity::Drop(BatchDropReason::FutureTimestampHolocene)
        );
    }

    #[test]
    fn test_check_batch_timestamp_holocene_active_past() {
        let cfg = RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        };
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 2, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { timestamp: 1, ..Default::default() };
        let batch = SingleBatch { epoch_num: 1, timestamp: 1, ..Default::default() };
        assert_eq!(
            batch.check_batch_timestamp(&cfg, l2_safe_head, &inclusion_block, false),
            BatchValidity::Past
        );
    }

    #[test]
    fn test_check_batch_timestamp_holocene_inactive_drop() {
        let cfg = RollupConfig::default();
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 2, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { timestamp: 1, ..Default::default() };
        let batch = SingleBatch { epoch_num: 1, timestamp: 1, ..Default::default() };
        assert_eq!(
            batch.check_batch_timestamp(&cfg, l2_safe_head, &inclusion_block, false),
            BatchValidity::Drop(BatchDropReason::PastTimestampPreHolocene)
        );
    }

    #[test]
    fn test_check_batch_timestamp_accept() {
        let cfg = RollupConfig::default();
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 2, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        let batch = SingleBatch { timestamp: 2, ..Default::default() };
        assert_eq!(
            batch.check_batch_timestamp(&cfg, l2_safe_head, &inclusion_block, false),
            BatchValidity::Accept
        );
    }

    /// A chain whose blocks may have siblings from timestamp 1000 on, with two blocks already
    /// sharing timestamp 1010 at the safe head.
    fn multi_block_cfg() -> RollupConfig {
        RollupConfig {
            block_time: 2,
            max_sequencer_drift: 100,
            hardforks: HardForkConfig {
                holocene_time: Some(0),
                karst_time: Some(0),
                ..Default::default()
            },
            multi_block_time: Some(1000),
            max_multi_blocks: Some(4),
            ..Default::default()
        }
    }

    fn multi_block_safe_head() -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo { number: 20, timestamp: 1010, ..Default::default() },
            l1_origin: BlockNumHash { number: 1, hash: BlockHash::ZERO },
            seq_num: 3,
        }
    }

    #[test]
    fn test_check_batch_sibling_accepted() {
        let cfg = multi_block_cfg();
        let l1_blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let inclusion_block = BlockInfo::default();
        let batch = SingleBatch { epoch_num: 1, timestamp: 1010, ..Default::default() };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, multi_block_safe_head(), &inclusion_block, true),
            BatchValidity::Accept
        );
    }

    /// The wire singular batch (`0x00`) has no way to say "sibling", so an equal timestamp keeps
    /// its Holocene classification: an already-applied block.
    #[test]
    fn test_check_batch_equal_timestamp_wire_singular_is_past() {
        let cfg = multi_block_cfg();
        let l1_blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let inclusion_block = BlockInfo::default();
        let batch = SingleBatch { epoch_num: 1, timestamp: 1010, ..Default::default() };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, multi_block_safe_head(), &inclusion_block, false),
            BatchValidity::Past
        );
    }

    #[test]
    fn test_check_batch_sibling_with_different_epoch_dropped() {
        let cfg = multi_block_cfg();
        let l1_blocks = vec![BlockInfo::default(), BlockInfo { number: 2, ..Default::default() }];
        let inclusion_block = BlockInfo::default();
        let batch = SingleBatch { epoch_num: 2, timestamp: 1010, ..Default::default() };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, multi_block_safe_head(), &inclusion_block, true),
            BatchValidity::Drop(BatchDropReason::SiblingOriginMismatch)
        );
    }

    /// Siblings are only allowed strictly after the activation timestamp, so the flag alone does
    /// not widen the accept-set.
    #[test]
    fn test_check_batch_sibling_before_activation_dropped() {
        let cfg = RollupConfig { multi_block_time: Some(1010), ..multi_block_cfg() };
        let l1_blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let inclusion_block = BlockInfo::default();
        let batch = SingleBatch { epoch_num: 1, timestamp: 1010, ..Default::default() };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, multi_block_safe_head(), &inclusion_block, true),
            BatchValidity::Drop(BatchDropReason::SiblingsNotAllowed)
        );
    }

    /// The accept-set grows by the parent's own timestamp, not beyond it.
    #[test]
    fn test_check_batch_sibling_two_block_times_ahead_is_future() {
        let cfg = multi_block_cfg();
        let l1_blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let inclusion_block = BlockInfo::default();
        let batch = SingleBatch { epoch_num: 1, timestamp: 1014, ..Default::default() };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, multi_block_safe_head(), &inclusion_block, true),
            BatchValidity::Drop(BatchDropReason::FutureTimestampHolocene)
        );
    }

    #[test]
    fn test_roundtrip_encoding() {
        use alloy_rlp::{Decodable, Encodable};
        let batch = SingleBatch {
            parent_hash: BlockHash::from([0x01; 32]),
            epoch_num: 1,
            epoch_hash: BlockHash::from([0x02; 32]),
            timestamp: 1,
            transactions: vec![Bytes::from(vec![0x01])],
        };
        let mut buf = vec![];
        batch.encode(&mut buf);
        let decoded = SingleBatch::decode(&mut buf.as_slice()).unwrap();
        assert_eq!(batch, decoded);
    }

    #[test]
    fn test_check_batch_succeeds() {
        let cfg = RollupConfig { max_sequencer_drift: 1, ..Default::default() };
        let l1_blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 1, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        let batch = SingleBatch {
            parent_hash: BlockHash::ZERO,
            epoch_num: 1,
            epoch_hash: BlockHash::ZERO,
            timestamp: 1,
            transactions: vec![Bytes::from(vec![0x01])],
        };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, false),
            BatchValidity::Accept
        );
    }

    fn eip_1559_tx() -> TxEip1559 {
        TxEip1559 {
            chain_id: 10u64,
            nonce: 2,
            max_fee_per_gas: 3,
            max_priority_fee_per_gas: 4,
            gas_limit: 5,
            to: Address::left_padding_from(&[6]).into(),
            value: U256::from(7_u64),
            input: vec![8].into(),
            access_list: Default::default(),
        }
    }

    fn example_transactions() -> Vec<Bytes> {
        let mut transactions = Vec::new();

        // First Transaction in the batch.
        let tx = eip_1559_tx();
        let sig = Signature::test_signature();
        let tx_signed = tx.into_signed(sig);
        let envelope: TxEnvelope = tx_signed.into();
        let encoded = envelope.encoded_2718();
        transactions.push(encoded.clone().into());
        let mut slice = encoded.as_slice();
        let decoded = TxEnvelope::decode_2718(&mut slice).unwrap();
        assert!(matches!(decoded, TxEnvelope::Eip1559(_)));

        // Second transaction in the batch.
        let mut tx = eip_1559_tx();
        tx.to = Address::left_padding_from(&[7]).into();
        let sig = Signature::test_signature();
        let tx_signed = tx.into_signed(sig);
        let envelope: TxEnvelope = tx_signed.into();
        let encoded = envelope.encoded_2718();
        transactions.push(encoded.clone().into());
        let mut slice = encoded.as_slice();
        let decoded = TxEnvelope::decode_2718(&mut slice).unwrap();
        assert!(matches!(decoded, TxEnvelope::Eip1559(_)));

        transactions
    }

    #[test]
    fn test_check_batch_full_txs() {
        // Use the example transaction
        let transactions = example_transactions();

        // Construct a basic `SingleBatch`
        let parent_hash = BlockHash::ZERO;
        let epoch_num = 1;
        let epoch_hash = BlockHash::ZERO;
        let timestamp = 1;

        let single_batch =
            SingleBatch { parent_hash, epoch_num, epoch_hash, timestamp, transactions };

        let cfg = RollupConfig { max_sequencer_drift: 1, ..Default::default() };
        let l1_blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 1, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        assert_eq!(
            single_batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, false),
            BatchValidity::Accept
        );
    }

    fn eip_7702_tx() -> TxEip7702 {
        TxEip7702 {
            chain_id: 10u64,
            nonce: 2,
            gas_limit: 5,
            max_fee_per_gas: 3,
            max_priority_fee_per_gas: 4,
            to: Address::left_padding_from(&[7]),
            value: U256::from(7_u64),
            input: vec![8].into(),
            ..Default::default()
        }
    }

    #[test]
    fn test_check_batch_drop_7702_pre_isthmus() {
        // Use the example transaction
        let mut transactions = example_transactions();

        // Extend the transactions with the 7702 transaction
        let eip_7702_tx = eip_7702_tx();
        let sig = Signature::test_signature();
        let tx_signed = eip_7702_tx.into_signed(sig);
        let envelope: TxEnvelope = tx_signed.into();
        let encoded = envelope.encoded_2718();
        transactions.push(encoded.into());

        // Construct a basic `SingleBatch`
        let parent_hash = BlockHash::ZERO;
        let epoch_num = 1;
        let epoch_hash = BlockHash::ZERO;
        let timestamp = 1;

        let single_batch =
            SingleBatch { parent_hash, epoch_num, epoch_hash, timestamp, transactions };

        // Notice: Isthmus is _not_ active yet.
        let cfg = RollupConfig { max_sequencer_drift: 1, ..Default::default() };
        let l1_blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 1, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        assert_eq!(
            single_batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, false),
            BatchValidity::Drop(BatchDropReason::Eip7702PreIsthmus)
        );
    }

    #[test]
    fn test_check_batch_accept_7702_post_isthmus() {
        // Use the example transaction
        let mut transactions = example_transactions();

        // Extend the transactions with the 7702 transaction
        let eip_7702_tx = eip_7702_tx();
        let sig = Signature::test_signature();
        let tx_signed = eip_7702_tx.into_signed(sig);
        let envelope: TxEnvelope = tx_signed.into();
        let encoded = envelope.encoded_2718();
        transactions.push(encoded.into());

        // Construct a basic `SingleBatch`
        let parent_hash = BlockHash::ZERO;
        let epoch_num = 1;
        let epoch_hash = BlockHash::ZERO;
        let timestamp = 1;

        let single_batch =
            SingleBatch { parent_hash, epoch_num, epoch_hash, timestamp, transactions };

        // Notice: Isthmus is active.
        let cfg = RollupConfig {
            max_sequencer_drift: 1,
            hardforks: HardForkConfig { isthmus_time: Some(0), ..Default::default() },
            ..Default::default()
        };
        let l1_blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 1, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        assert_eq!(
            single_batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, false),
            BatchValidity::Accept
        );
    }

    #[test]
    fn test_check_batch_drop_post_exec_pre_sdm() {
        let mut transactions = example_transactions();
        let tx: OpTxEnvelope = TxPostExec::new(PostExecPayload {
            version: POST_EXEC_PAYLOAD_VERSION,
            block_number: 1,
            gas_refund_entries: vec![],
        })
        .into();
        transactions.push(tx.encoded_2718().into());

        let single_batch = SingleBatch {
            parent_hash: BlockHash::ZERO,
            epoch_num: 1,
            epoch_hash: BlockHash::ZERO,
            timestamp: 1,
            transactions,
        };

        let cfg = RollupConfig { max_sequencer_drift: 1, ..Default::default() };
        let l1_blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 1, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        assert_eq!(
            single_batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, false),
            BatchValidity::Drop(BatchDropReason::PostExecPreLagoon)
        );
    }

    #[test]
    fn test_check_batch_accept_post_exec_post_sdm() {
        let mut transactions = example_transactions();
        let tx: OpTxEnvelope = TxPostExec::new(PostExecPayload {
            version: POST_EXEC_PAYLOAD_VERSION,
            block_number: 1,
            gas_refund_entries: vec![],
        })
        .into();
        transactions.push(tx.encoded_2718().into());

        let single_batch = SingleBatch {
            parent_hash: BlockHash::ZERO,
            epoch_num: 1,
            epoch_hash: BlockHash::ZERO,
            timestamp: 1,
            transactions,
        };

        let cfg = RollupConfig {
            max_sequencer_drift: 1,
            hardforks: HardForkConfig { lagoon_time: Some(0), ..Default::default() },
            ..Default::default()
        };
        let l1_blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 1, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        assert_eq!(
            single_batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, false),
            BatchValidity::Accept
        );
    }

    #[test]
    fn test_check_batch_drop_empty_tx() {
        // An empty tx is not valid 2718 encoding.
        // The batch must be dropped.
        let transactions = vec![Default::default()];

        // Construct a basic `SingleBatch`
        let parent_hash = BlockHash::ZERO;
        let epoch_num = 1;
        let epoch_hash = BlockHash::ZERO;
        let timestamp = 1;

        let single_batch =
            SingleBatch { parent_hash, epoch_num, epoch_hash, timestamp, transactions };

        // Notice: Isthmus is _not_ active yet.
        let cfg = RollupConfig { max_sequencer_drift: 1, ..Default::default() };
        let l1_blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 1, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        assert_eq!(
            single_batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, false),
            BatchValidity::Drop(BatchDropReason::EmptyTransaction)
        );
    }

    #[test]
    fn test_check_batch_drop_2718_deposit() {
        // Add a 2718 deposit transaction to the batch.
        let mut transactions = example_transactions();

        // Extend the transactions with the 2718 deposit transaction
        let tx = TxDeposit {
            source_hash: Default::default(),
            from: Address::left_padding_from(&[7]),
            to: TxKind::Create,
            mint: 0,
            value: U256::from(7_u64),
            gas_limit: 5,
            is_system_transaction: false,
            input: Default::default(),
        };
        let envelope = OpTxEnvelope::Deposit(Sealed::new(tx));
        let encoded = envelope.encoded_2718();
        transactions.push(encoded.into());

        // Construct a basic `SingleBatch`
        let parent_hash = BlockHash::ZERO;
        let epoch_num = 1;
        let epoch_hash = BlockHash::ZERO;
        let timestamp = 1;

        let single_batch =
            SingleBatch { parent_hash, epoch_num, epoch_hash, timestamp, transactions };

        // Notice: Isthmus is _not_ active yet.
        let cfg = RollupConfig { max_sequencer_drift: 1, ..Default::default() };
        let l1_blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 1, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        assert_eq!(
            single_batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, false),
            BatchValidity::Drop(BatchDropReason::DepositTransaction)
        );
    }

    #[test]
    #[cfg(feature = "std")]
    fn test_check_batch_drop_non_empty_interop_transition() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        // Gather a few test transactions for the batch.
        let transactions = example_transactions();

        // Construct a basic `SingleBatch`
        let parent_hash = BlockHash::ZERO;
        let epoch_num = 1;
        let epoch_hash = BlockHash::ZERO;
        let timestamp = 1;

        let single_batch =
            SingleBatch { parent_hash, epoch_num, epoch_hash, timestamp, transactions };

        let cfg = RollupConfig {
            max_sequencer_drift: 1,
            block_time: 1,
            hardforks: HardForkConfig { lagoon_time: Some(1), ..Default::default() },
            ..Default::default()
        };
        let l1_blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 0, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        assert_eq!(
            single_batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, false),
            BatchValidity::Drop(BatchDropReason::NonEmptyTransitionBlock)
        );

        assert!(
            trace_store
                .get_by_level(Level::WARN)
                .iter()
                .any(|s| { s.contains("Sequencer included user transactions") })
        )
    }
}
