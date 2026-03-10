//! L1 chain builder for constructing deterministic L1 blocks.

use alloy_consensus::{Header, Receipt, ReceiptEnvelope, ReceiptWithBloom};
use alloy_primitives::{Bloom, Bytes, Sealable};
use std::collections::BTreeMap;

use crate::config::DeterministicConfig;
use crate::state::roots::EMPTY_ROOT_HASH;

use super::types::{BatchSubmission, BlobWithCommitment, L1Block};

/// Builds a deterministic L1 chain block by block.
#[allow(missing_debug_implementations)]
pub struct L1ChainBuilder {
    config: DeterministicConfig,
    blocks: Vec<L1Block>,
    /// Blobs indexed by (slot, `versioned_hash`).
    blobs: BTreeMap<u64, Vec<BlobWithCommitment>>,
}

impl L1ChainBuilder {
    /// Create a new L1 chain builder with a genesis block.
    pub fn new(config: &DeterministicConfig) -> Self {
        let genesis_header = Header {
            number: 0,
            timestamp: config.genesis_timestamp,
            state_root: EMPTY_ROOT_HASH,
            transactions_root: EMPTY_ROOT_HASH,
            receipts_root: EMPTY_ROOT_HASH,
            gas_limit: 30_000_000,
            ..Default::default()
        };
        let sealed = genesis_header.seal_slow();

        let genesis_block = L1Block {
            header: sealed,
            transactions: vec![],
            receipts: vec![],
        };

        Self {
            config: config.clone(),
            blocks: vec![genesis_block],
            blobs: BTreeMap::new(),
        }
    }

    /// Emit an empty L1 block with no transactions.
    pub fn emit_empty_block(&mut self) {
        let prev = self.blocks.last().expect("always have genesis");
        let header = Header {
            parent_hash: prev.header.hash(),
            number: prev.header.inner().number + 1,
            timestamp: prev.header.inner().timestamp + self.config.l1_block_time,
            state_root: EMPTY_ROOT_HASH,
            transactions_root: EMPTY_ROOT_HASH,
            receipts_root: EMPTY_ROOT_HASH,
            gas_limit: 30_000_000,
            ..Default::default()
        };
        let sealed = header.seal_slow();

        self.blocks.push(L1Block {
            header: sealed,
            transactions: vec![],
            receipts: vec![],
        });
    }

    /// Emit an L1 block containing batch submissions.
    pub fn emit_block_with_batches(&mut self, batches: Vec<BatchSubmission>) {
        let prev = self.blocks.last().expect("always have genesis");
        let block_num = prev.header.inner().number + 1;
        let timestamp = prev.header.inner().timestamp + self.config.l1_block_time;

        let mut transactions = Vec::new();
        let mut receipts = Vec::new();

        for batch in batches {
            match batch {
                BatchSubmission::Calldata(data) => {
                    transactions.push(data);
                    receipts.push(success_receipt(receipts.len() as u64));
                }
                BatchSubmission::Blob(blob_data) => {
                    let slot = self.timestamp_to_slot(timestamp);
                    self.blobs
                        .entry(slot)
                        .or_default()
                        .push(blob_data.clone());
                    transactions.push(Bytes::new());
                    receipts.push(success_receipt(receipts.len() as u64));
                }
            }
        }

        let header = Header {
            parent_hash: prev.header.hash(),
            number: block_num,
            timestamp,
            state_root: EMPTY_ROOT_HASH,
            transactions_root: EMPTY_ROOT_HASH,
            receipts_root: EMPTY_ROOT_HASH,
            gas_limit: 30_000_000,
            ..Default::default()
        };
        let sealed = header.seal_slow();

        self.blocks.push(L1Block {
            header: sealed,
            transactions,
            receipts,
        });
    }

    /// Emit an L1 block with raw transaction data.
    pub fn emit_block_with_raw_txs(&mut self, txs: Vec<Bytes>) {
        let prev = self.blocks.last().expect("always have genesis");
        let header = Header {
            parent_hash: prev.header.hash(),
            number: prev.header.inner().number + 1,
            timestamp: prev.header.inner().timestamp + self.config.l1_block_time,
            state_root: EMPTY_ROOT_HASH,
            transactions_root: EMPTY_ROOT_HASH,
            receipts_root: EMPTY_ROOT_HASH,
            gas_limit: 30_000_000,
            ..Default::default()
        };
        let sealed = header.seal_slow();

        let receipts: Vec<_> = (0..txs.len())
            .map(|i| success_receipt(i as u64))
            .collect();

        self.blocks.push(L1Block {
            header: sealed,
            transactions: txs,
            receipts,
        });
    }

    /// Get all blocks.
    pub fn blocks(&self) -> &[L1Block] {
        &self.blocks
    }

    /// Get the latest block.
    pub fn head(&self) -> &L1Block {
        self.blocks.last().expect("always have genesis")
    }

    /// Get a block by number.
    pub fn block_at(&self, number: u64) -> Option<&L1Block> {
        self.blocks.get(number as usize)
    }

    /// Get blobs at a particular slot.
    pub fn blobs_at_slot(&self, slot: u64) -> Option<&Vec<BlobWithCommitment>> {
        self.blobs.get(&slot)
    }

    /// Convert a timestamp to a beacon slot number.
    pub const fn timestamp_to_slot(&self, timestamp: u64) -> u64 {
        (timestamp - self.config.genesis_timestamp) / self.config.seconds_per_slot
    }

    /// Get the config.
    pub const fn config(&self) -> &DeterministicConfig {
        &self.config
    }
}

/// Create a simple success receipt.
fn success_receipt(cumulative_gas: u64) -> ReceiptEnvelope {
    let receipt = Receipt {
        status: alloy_consensus::Eip658Value::Eip658(true),
        cumulative_gas_used: cumulative_gas * 21_000,
        logs: vec![],
    };
    ReceiptEnvelope::Eip1559(ReceiptWithBloom::new(receipt, Bloom::default()))
}
