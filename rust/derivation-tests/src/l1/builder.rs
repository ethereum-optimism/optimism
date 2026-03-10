//! L1 chain builder for constructing deterministic L1 blocks.

use alloy_consensus::{Header, Receipt, ReceiptEnvelope, ReceiptWithBloom, SignableTransaction, TxEip1559};
use alloy_eips::Encodable2718;
use alloy_primitives::{Bloom, Bytes, Log, LogData, Sealable, TxKind};
use alloy_signer::SignerSync;
use alloy_signer_local::PrivateKeySigner;
use kona_genesis::{CONFIG_UPDATE_EVENT_VERSION_0, CONFIG_UPDATE_TOPIC};
use std::collections::BTreeMap;

use crate::{config::DeterministicConfig, state::roots::EMPTY_ROOT_HASH};

use super::types::{BatchSubmission, BlobWithCommitment, L1Block, SystemConfigUpdate};

/// Builds a deterministic L1 chain block by block.
#[allow(missing_debug_implementations)]
pub struct L1ChainBuilder {
    config: DeterministicConfig,
    blocks: Vec<L1Block>,
    /// Blobs indexed by (slot, `versioned_hash`).
    blobs: BTreeMap<u64, Vec<BlobWithCommitment>>,
    /// Batcher transaction nonce counter.
    batcher_nonce: u64,
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
            withdrawals_root: Some(EMPTY_ROOT_HASH),
            gas_limit: 30_000_000,
            ..Default::default()
        };
        let sealed = genesis_header.seal_slow();

        let genesis_block = L1Block { header: sealed, transactions: vec![], receipts: vec![] };

        Self {
            config: config.clone(),
            blocks: vec![genesis_block],
            blobs: BTreeMap::new(),
            batcher_nonce: 0,
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
            withdrawals_root: Some(EMPTY_ROOT_HASH),
            gas_limit: 30_000_000,
            ..Default::default()
        };
        let sealed = header.seal_slow();
        self.blocks.push(L1Block { header: sealed, transactions: vec![], receipts: vec![] });
    }

    /// Emit an L1 block containing batch submissions.
    pub fn emit_block_with_batches(&mut self, batches: Vec<BatchSubmission>) {
        let prev = self.blocks.last().expect("always have genesis");
        let parent_hash = prev.header.hash();
        let block_num = prev.header.inner().number + 1;
        let timestamp = prev.header.inner().timestamp + self.config.l1_block_time;

        let signer = PrivateKeySigner::from_bytes(&self.config.batcher_key)
            .expect("valid batcher key");

        let mut transactions = Vec::new();
        let mut receipts = Vec::new();

        for batch in batches {
            match batch {
                BatchSubmission::Calldata(data) => {
                    let signed_tx = self.sign_batcher_tx(&signer, data);
                    transactions.push(signed_tx);
                    receipts.push(success_receipt(receipts.len() as u64));
                }
                BatchSubmission::Blob(blob_data) => {
                    let slot = self.timestamp_to_slot(timestamp);
                    self.blobs.entry(slot).or_default().push(blob_data.clone());
                    // Blob txs still need a signed envelope on L1, but the data is in the blob.
                    // Create a signed tx with empty calldata for the blob case.
                    let signed_tx = self.sign_batcher_tx(&signer, Bytes::new());
                    transactions.push(signed_tx);
                    receipts.push(success_receipt(receipts.len() as u64));
                }
            }
        }

        let transactions_root = compute_raw_transactions_root(&transactions);
        let receipts_root = compute_l1_receipts_root(&receipts);

        let header = Header {
            parent_hash,
            number: block_num,
            timestamp,
            state_root: EMPTY_ROOT_HASH,
            transactions_root,
            receipts_root,
            withdrawals_root: Some(EMPTY_ROOT_HASH),
            gas_limit: 30_000_000,
            ..Default::default()
        };
        let sealed = header.seal_slow();

        self.blocks.push(L1Block { header: sealed, transactions, receipts });
    }

    /// Emit an L1 block with raw transaction data.
    pub fn emit_block_with_raw_txs(&mut self, txs: Vec<Bytes>) {
        let prev = self.blocks.last().expect("always have genesis");

        let receipts: Vec<_> = (0..txs.len()).map(|i| success_receipt(i as u64)).collect();
        let transactions_root = compute_raw_transactions_root(&txs);
        let receipts_root = compute_l1_receipts_root(&receipts);

        let header = Header {
            parent_hash: prev.header.hash(),
            number: prev.header.inner().number + 1,
            timestamp: prev.header.inner().timestamp + self.config.l1_block_time,
            state_root: EMPTY_ROOT_HASH,
            transactions_root,
            receipts_root,
            withdrawals_root: Some(EMPTY_ROOT_HASH),
            gas_limit: 30_000_000,
            ..Default::default()
        };
        let sealed = header.seal_slow();

        self.blocks.push(L1Block { header: sealed, transactions: txs, receipts });
    }

    /// Emit an L1 block containing a system config update log event.
    ///
    /// The log is emitted from the `SystemConfig` proxy address with the standard
    /// `ConfigUpdate(uint256,uint8,bytes)` topic layout defined in the OP Stack spec.
    pub fn emit_block_with_system_config_update(&mut self, update: SystemConfigUpdate) {
        let prev = self.blocks.last().expect("always have genesis");
        let block_num = prev.header.inner().number + 1;
        let timestamp = prev.header.inner().timestamp + self.config.l1_block_time;

        let log = system_config_update_log(self.config.system_config, &update);
        let receipt = Receipt {
            status: alloy_consensus::Eip658Value::Eip658(true),
            cumulative_gas_used: 21_000,
            logs: vec![log],
        };
        let receipt_envelope = ReceiptEnvelope::Eip1559(ReceiptWithBloom::new(
            receipt,
            Bloom::default(),
        ));

        let transactions = vec![Bytes::new()];
        let receipts = vec![receipt_envelope];
        let transactions_root = compute_raw_transactions_root(&transactions);
        let receipts_root = compute_l1_receipts_root(&receipts);

        let header = Header {
            parent_hash: prev.header.hash(),
            number: block_num,
            timestamp,
            state_root: EMPTY_ROOT_HASH,
            transactions_root,
            receipts_root,
            withdrawals_root: Some(EMPTY_ROOT_HASH),
            gas_limit: 30_000_000,
            ..Default::default()
        };
        let sealed = header.seal_slow();

        self.blocks.push(L1Block { header: sealed, transactions, receipts });
    }

    /// Sign a batcher transaction with the batcher key, returning RLP-encoded signed tx bytes.
    fn sign_batcher_tx(&mut self, signer: &PrivateKeySigner, calldata: Bytes) -> Bytes {
        let tx = TxEip1559 {
            chain_id: self.config.l1_chain_id,
            nonce: self.batcher_nonce,
            gas_limit: 1_000_000,
            max_fee_per_gas: 0,
            max_priority_fee_per_gas: 0,
            to: TxKind::Call(self.config.batch_inbox),
            value: alloy_primitives::U256::ZERO,
            input: calldata,
            ..Default::default()
        };
        self.batcher_nonce += 1;

        let sig = signer.sign_hash_sync(&tx.signature_hash()).expect("signing should not fail");
        let signed = tx.into_signed(sig);
        let envelope = alloy_consensus::TxEnvelope::Eip1559(signed);

        let mut buf = Vec::new();
        envelope.encode_2718(&mut buf);
        Bytes::from(buf)
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

    /// Get all blobs indexed by slot.
    pub const fn blobs(&self) -> &BTreeMap<u64, Vec<BlobWithCommitment>> {
        &self.blobs
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

/// Compute the transactions trie root from raw RLP-encoded transaction bytes.
fn compute_raw_transactions_root(txs: &[Bytes]) -> alloy_primitives::B256 {
    if txs.is_empty() {
        return EMPTY_ROOT_HASH;
    }
    // Raw tx bytes are already in the correct encoding for the trie — just write them as-is.
    kona_mpt::ordered_trie_with_encoder(txs, |tx, buf| buf.put_slice(tx)).root()
}

/// Build a log event for a system config update.
///
/// The log format follows the OP Stack spec:
/// - topic[0]: `CONFIG_UPDATE_TOPIC` (`keccak256("ConfigUpdate(uint256,uint8,bytes)")`)
/// - topic[1]: `CONFIG_UPDATE_EVENT_VERSION_0` (`B256::ZERO`)
/// - topic[2]: update type (0 = batcher address, 1 = gas config)
/// - data: ABI-encoded update payload
fn system_config_update_log(
    system_config_address: alloy_primitives::Address,
    update: &SystemConfigUpdate,
) -> Log {
    match update {
        SystemConfigUpdate::BatcherAddress(addr) => {
            let update_type = alloy_primitives::B256::ZERO; // type 0
            let mut data = vec![0u8; 96];
            // ABI offset to bytes: 0x20
            data[31] = 0x20;
            // ABI length of bytes: 0x20
            data[63] = 0x20;
            // Address padded to 32 bytes
            data[76..96].copy_from_slice(addr.as_slice());

            Log {
                address: system_config_address,
                data: LogData::new_unchecked(
                    vec![CONFIG_UPDATE_TOPIC, CONFIG_UPDATE_EVENT_VERSION_0, update_type],
                    data.into(),
                ),
            }
        }
        SystemConfigUpdate::GasConfig { overhead, scalar } => {
            let update_type = alloy_primitives::B256::with_last_byte(1); // type 1
            let mut data = vec![0u8; 128];
            // ABI offset to bytes: 0x20
            data[31] = 0x20;
            // ABI length of bytes: 0x40
            data[63] = 0x40;
            // overhead as U256 in bytes 64..96
            data[64..96].copy_from_slice(&overhead.to_be_bytes::<32>());
            // scalar as U256 in bytes 96..128
            data[96..128].copy_from_slice(&scalar.to_be_bytes::<32>());

            Log {
                address: system_config_address,
                data: LogData::new_unchecked(
                    vec![CONFIG_UPDATE_TOPIC, CONFIG_UPDATE_EVENT_VERSION_0, update_type],
                    data.into(),
                ),
            }
        }
    }
}

/// Compute the receipts trie root from L1 receipt envelopes.
fn compute_l1_receipts_root(receipts: &[ReceiptEnvelope]) -> alloy_primitives::B256 {
    if receipts.is_empty() {
        return EMPTY_ROOT_HASH;
    }
    kona_mpt::ordered_trie_with_encoder(receipts, |r, buf| r.encode_2718(buf)).root()
}
