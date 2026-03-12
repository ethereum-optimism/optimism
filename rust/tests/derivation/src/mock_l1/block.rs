//! L1 block construction with proper header hashing.

use alloy_consensus::{Header, Receipt, TxEnvelope};
use alloy_primitives::{B256, Keccak256, U256};
use alloy_rlp::Encodable;
use kona_protocol::BlockInfo;

/// An L1 block stored in the mock chain.
#[derive(Debug, Clone)]
pub struct L1Block {
    /// Block info (number, hash, parent_hash, timestamp).
    pub info: BlockInfo,
    /// Full header.
    pub header: Header,
    /// Transactions in this block.
    pub transactions: Vec<TxEnvelope>,
    /// Receipts for the transactions.
    pub receipts: Vec<Receipt>,
}

impl L1Block {
    /// Creates a genesis L1 block from the given block info.
    pub fn genesis(info: BlockInfo) -> Self {
        let header = Header {
            number: info.number,
            timestamp: info.timestamp,
            parent_hash: info.parent_hash,
            ..Default::default()
        };
        Self {
            info,
            header,
            transactions: Vec::new(),
            receipts: Vec::new(),
        }
    }

    /// Creates a new L1 block.
    pub fn new(
        number: u64,
        timestamp: u64,
        parent_hash: B256,
        transactions: Vec<TxEnvelope>,
        receipts: Vec<Receipt>,
    ) -> Self {
        let transactions_root = compute_tx_root(&transactions);
        let receipts_root = compute_receipts_root(&receipts);

        let header = Header {
            number,
            timestamp,
            parent_hash,
            transactions_root,
            receipts_root,
            gas_used: receipts.last().map(|r| r.cumulative_gas_used).unwrap_or(0),
            difficulty: U256::from(1),
            ..Default::default()
        };

        let hash = header.hash_slow();

        let info = BlockInfo {
            number,
            hash,
            parent_hash,
            timestamp,
        };

        Self { info, header, transactions, receipts }
    }
}

/// Computes a simple transaction root by hashing encoded transactions.
fn compute_tx_root(txs: &[TxEnvelope]) -> B256 {
    if txs.is_empty() {
        return alloy_consensus::constants::EMPTY_ROOT_HASH;
    }
    let mut hasher = Keccak256::new();
    hasher.update(b"tx_root");
    for tx in txs {
        let mut buf = Vec::new();
        tx.encode(&mut buf);
        hasher.update(&buf);
    }
    hasher.finalize()
}

/// Computes a simple receipts root by hashing receipt data.
fn compute_receipts_root(receipts: &[Receipt]) -> B256 {
    if receipts.is_empty() {
        return alloy_consensus::constants::EMPTY_ROOT_HASH;
    }
    let mut hasher = Keccak256::new();
    hasher.update(b"receipt_root");
    for receipt in receipts {
        // Hash the status and cumulative gas
        hasher.update(&[receipt.status.coerce_status() as u8]);
        hasher.update(&receipt.cumulative_gas_used.to_be_bytes());
    }
    hasher.finalize()
}
