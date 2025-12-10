//! This module contains the [`SingleBatch`] type.

use crate::{BatchValidity, BlockInfo, L2BlockInfo};
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
    pub fn check_batch_timestamp(
        &self,
        cfg: &RollupConfig,
        l2_safe_head: L2BlockInfo,
        inclusion_block: &BlockInfo,
    ) -> BatchValidity {
        let next_timestamp = l2_safe_head.block_info.timestamp + cfg.block_time;
        if self.timestamp > next_timestamp {
            if cfg.is_holocene_active(inclusion_block.timestamp) {
                return BatchValidity::Drop;
            }
            return BatchValidity::Future;
        }
        if self.timestamp < next_timestamp {
            if cfg.is_holocene_active(inclusion_block.timestamp) {
                return BatchValidity::Past;
            }
            return BatchValidity::Drop;
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
    ) -> BatchValidity {
        // Cannot have empty l1_blocks for batch validation.
        if l1_blocks.is_empty() {
            return BatchValidity::Undecided;
        }

        let epoch = l1_blocks[0];

        // If the batch is not accepted by the timestamp check, return the result.
        let timestamp_check = self.check_batch_timestamp(cfg, l2_safe_head, inclusion_block);
        if !timestamp_check.is_accept() {
            return timestamp_check;
        }

        // Dependent on the above timestamp check.
        // If the timestamp is correct, then it must build on top of the safe head.
        if self.parent_hash != l2_safe_head.block_info.hash {
            return BatchValidity::Drop;
        }

        // Filter out batches that were included too late.
        if self.epoch_num + cfg.seq_window_size < inclusion_block.number {
            return BatchValidity::Drop;
        }

        // Check the L1 origin of the batch
        let mut batch_origin = epoch;
        if self.epoch_num < epoch.number {
            return BatchValidity::Drop;
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
            return BatchValidity::Drop;
        }

        // Validate the batch epoch hash
        if self.epoch_hash != batch_origin.hash {
            return BatchValidity::Drop;
        }

        if self.timestamp < batch_origin.timestamp {
            return BatchValidity::Drop;
        }

        // Check if we ran out of sequencer time drift
        let max_drift = cfg.max_sequencer_drift(batch_origin.timestamp);
        let max = if let Some(max) = batch_origin.timestamp.checked_add(max_drift) {
            max
        } else {
            return BatchValidity::Drop;
        };

        let no_txs = self.transactions.is_empty();
        if self.timestamp > max && !no_txs {
            // If the sequencer is ignoring the time drift rule, then drop the batch and force an
            // empty batch instead, as the sequencer is not allowed to include anything
            // past this point without moving to the next epoch.
            return BatchValidity::Drop;
        }
        if self.timestamp > max && no_txs {
            // If the sequencer is co-operating by producing an empty batch,
            // allow the batch if it was the right thing to do to maintain the L2 time >= L1 time
            // invariant. Only check batches that do not advance the epoch, to ensure
            // epoch advancement regardless of time drift is allowed.
            if epoch.number == batch_origin.number {
                if l1_blocks.len() < 2 {
                    return BatchValidity::Undecided;
                }
                let next_origin = l1_blocks[1];
                // Check if the next L1 Origin could have been adopted
                if self.timestamp >= next_origin.timestamp {
                    return BatchValidity::Drop;
                }
            }
        }

        // If this is the first block in the jovian or interop hardfork, and the batch contains any
        // transactions, it must be dropped.
        if (cfg.is_first_jovian_block(self.timestamp) || cfg.is_first_interop_block(self.timestamp)) &&
            !self.transactions.is_empty()
        {
            warn!(
                target: "single_batch",
                "Sequencer included user transactions in jovian or interop transition block. Dropping batch."
            );
            return BatchValidity::Drop;
        }

        // We can do this check earlier, but it's intensive so we do it last for the sad-path.
        for tx in self.transactions.iter() {
            if tx.is_empty() {
                return BatchValidity::Drop;
            }
            if tx.as_ref().first() == Some(&(OpTxType::Deposit as u8)) {
                return BatchValidity::Drop;
            }
            // If isthmus is not active yet and the transaction is a 7702, drop the batch.
            if !cfg.is_isthmus_active(self.timestamp) &&
                tx.as_ref().first() == Some(&(OpTxType::Eip7702 as u8))
            {
                return BatchValidity::Drop;
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
    use op_alloy_consensus::{OpTxEnvelope, TxDeposit};
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
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block),
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
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block),
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
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block),
            BatchValidity::Drop
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
            batch.check_batch_timestamp(&cfg, l2_safe_head, &inclusion_block),
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
            batch.check_batch_timestamp(&cfg, l2_safe_head, &inclusion_block),
            BatchValidity::Drop
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
            batch.check_batch_timestamp(&cfg, l2_safe_head, &inclusion_block),
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
            batch.check_batch_timestamp(&cfg, l2_safe_head, &inclusion_block),
            BatchValidity::Drop
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
            batch.check_batch_timestamp(&cfg, l2_safe_head, &inclusion_block),
            BatchValidity::Accept
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
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block),
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
            single_batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block),
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
            single_batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block),
            BatchValidity::Drop
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
            single_batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block),
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
            single_batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block),
            BatchValidity::Drop
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
            single_batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block),
            BatchValidity::Drop
        );
    }

    #[test]
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
            hardforks: HardForkConfig { interop_time: Some(1), ..Default::default() },
            ..Default::default()
        };
        let l1_blocks = vec![BlockInfo::default(), BlockInfo::default()];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 0, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        assert_eq!(
            single_batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block),
            BatchValidity::Drop
        );

        assert!(
            trace_store
                .get_by_level(Level::WARN)
                .iter()
                .any(|s| { s.contains("Sequencer included user transactions") })
        )
    }
}
