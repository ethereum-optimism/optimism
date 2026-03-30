//! L1 JSON-RPC server implementation.

use alloy_consensus::{Transaction, transaction::SignerRecoverable};
use alloy_eips::{Decodable2718, Typed2718};
use alloy_primitives::{B256, Bytes, keccak256};
use alloy_rlp::Encodable;
use jsonrpsee::{proc_macros::rpc, types::ErrorObjectOwned};
use serde_json::Value;
use std::collections::HashMap;

use crate::l1::L1Block;

/// Compute a deterministic fake transaction hash from the block hash and tx index.
fn fake_tx_hash(block_hash: B256, tx_index: usize) -> B256 {
    let mut preimage = [0u8; 64];
    preimage[..32].copy_from_slice(block_hash.as_slice());
    preimage[56..64].copy_from_slice(&(tx_index as u64).to_be_bytes());
    keccak256(preimage)
}

/// L1 RPC server trait.
#[rpc(server)]
pub(crate) trait L1Rpc {
    /// Returns the L1 chain ID.
    #[method(name = "eth_chainId")]
    fn chain_id(&self) -> Result<String, ErrorObjectOwned>;

    /// Returns a block by hash.
    #[method(name = "eth_getBlockByHash")]
    fn get_block_by_hash(
        &self,
        hash: B256,
        full_txs: bool,
    ) -> Result<Option<Value>, ErrorObjectOwned>;

    /// Returns a block by number.
    #[method(name = "eth_getBlockByNumber")]
    fn get_block_by_number(
        &self,
        number: String,
        full_txs: bool,
    ) -> Result<Option<Value>, ErrorObjectOwned>;

    /// Returns block receipts.
    #[method(name = "eth_getBlockReceipts")]
    fn get_block_receipts(&self, hash: B256) -> Result<Option<Value>, ErrorObjectOwned>;

    /// Returns the RLP-encoded header.
    #[method(name = "debug_getRawHeader")]
    fn get_raw_header(&self, hash: B256) -> Result<Bytes, ErrorObjectOwned>;

    /// Returns RLP-encoded receipts.
    #[method(name = "debug_getRawReceipts")]
    fn get_raw_receipts(&self, hash: B256) -> Result<Vec<Bytes>, ErrorObjectOwned>;

    /// Returns a transaction receipt by hash.
    #[method(name = "eth_getTransactionReceipt")]
    fn get_transaction_receipt(&self, tx_hash: B256) -> Result<Option<Value>, ErrorObjectOwned>;
}

/// L1 RPC server implementation backed by in-memory blocks.
pub(crate) struct L1RpcImpl {
    blocks: Vec<L1Block>,
    by_hash: HashMap<B256, usize>,
    /// Maps fake tx hash to (block index, transaction index).
    tx_index: HashMap<B256, (usize, usize)>,
    chain_id: u64,
}

impl L1RpcImpl {
    /// Create from a list of L1 blocks.
    pub(crate) fn new(blocks: Vec<L1Block>, chain_id: u64) -> Self {
        let by_hash: HashMap<B256, usize> =
            blocks.iter().enumerate().map(|(i, b)| (b.header.hash(), i)).collect();

        let mut tx_index = HashMap::new();
        for (block_idx, block) in blocks.iter().enumerate() {
            for (tx_idx, tx_bytes) in block.transactions.iter().enumerate() {
                // Try to get real tx hash from decoded envelope, fall back to fake hash
                let hash = alloy_consensus::TxEnvelope::decode_2718(&mut tx_bytes.as_ref())
                    .map(|env| *env.tx_hash())
                    .unwrap_or_else(|_| fake_tx_hash(block.header.hash(), tx_idx));
                tx_index.insert(hash, (block_idx, tx_idx));
            }
        }

        Self { blocks, by_hash, tx_index, chain_id }
    }

    fn resolve_block_number(&self, number: &str) -> Option<usize> {
        match number {
            "latest" | "safe" | "finalized" | "pending" => Some(self.blocks.len() - 1),
            "earliest" => Some(0),
            hex => {
                let n = u64::from_str_radix(hex.trim_start_matches("0x"), 16).ok()?;
                ((n as usize) < self.blocks.len()).then_some(n as usize)
            }
        }
    }

    fn block_to_json(&self, block: &L1Block, full_txs: bool) -> Value {
        let header = block.header.inner();
        serde_json::json!({
            "hash": block.header.hash(),
            "parentHash": header.parent_hash,
            "number": format!("0x{:x}", header.number),
            "timestamp": format!("0x{:x}", header.timestamp),
            "stateRoot": header.state_root,
            "transactionsRoot": header.transactions_root,
            "receiptsRoot": header.receipts_root,
            "gasLimit": format!("0x{:x}", header.gas_limit),
            "gasUsed": format!("0x{:x}", header.gas_used),
            "baseFeePerGas": format!("0x{:x}", header.base_fee_per_gas.unwrap_or(0)),
            "difficulty": "0x0",
            "totalDifficulty": "0x0",
            "miner": header.beneficiary,
            "extraData": "0x",
            "nonce": "0x0000000000000000",
            "sha3Uncles": header.ommers_hash,
            "logsBloom": header.logs_bloom,
            "size": "0x0",
            "mixHash": header.mix_hash,
            "withdrawalsRoot": header.withdrawals_root.unwrap_or(crate::state::roots::EMPTY_ROOT_HASH),
            "withdrawals": [],
            "blobGasUsed": format!("0x{:x}", header.blob_gas_used.unwrap_or(0)),
            "excessBlobGas": format!("0x{:x}", header.excess_blob_gas.unwrap_or(0)),
            "parentBeaconBlockRoot": header.parent_beacon_block_root.unwrap_or(alloy_primitives::B256::ZERO),
            "requestsHash": header.requests_hash.unwrap_or(alloy_eips::eip7685::EMPTY_REQUESTS_HASH),
            "transactions": block.transactions.iter().enumerate().map(|(i, tx_bytes)| {
                if full_txs {
                    tx_to_json(block.header.hash(), block.header.inner().number, i, tx_bytes)
                } else {
                    let hash = alloy_consensus::TxEnvelope::decode_2718(&mut tx_bytes.as_ref())
                        .map(|env| *env.tx_hash())
                        .unwrap_or_else(|_| fake_tx_hash(block.header.hash(), i));
                    serde_json::json!(hash)
                }
            }).collect::<Vec<_>>(),
        })
    }
}

/// Build a JSON representation of a signed L1 transaction for `full_txs` responses.
fn tx_to_json(block_hash: B256, block_number: u64, tx_index: usize, tx_bytes: &Bytes) -> Value {
    // Try to decode as a signed transaction envelope
    if let Ok(envelope) = alloy_consensus::TxEnvelope::decode_2718(&mut tx_bytes.as_ref()) {
        let tx_hash = *envelope.tx_hash();
        let from = envelope.recover_signer().unwrap_or_default();
        let to = envelope.to();
        let nonce = envelope.nonce();
        let value = envelope.value();
        let gas_limit = envelope.gas_limit();
        let input = envelope.input().clone();

        let chain_id = match &envelope {
            alloy_consensus::TxEnvelope::Eip1559(tx) => Some(tx.tx().chain_id),
            alloy_consensus::TxEnvelope::Eip2930(tx) => Some(tx.tx().chain_id),
            alloy_consensus::TxEnvelope::Eip4844(tx) => Some(tx.tx().tx().chain_id),
            _ => None,
        };

        let sig = envelope.signature();

        return serde_json::json!({
            "hash": tx_hash,
            "blockHash": block_hash,
            "blockNumber": format!("0x{:x}", block_number),
            "transactionIndex": format!("0x{:x}", tx_index),
            "from": from,
            "to": to,
            "nonce": format!("0x{:x}", nonce),
            "value": format!("0x{:x}", value),
            "gas": format!("0x{:x}", gas_limit),
            "input": input,
            "type": format!("0x{:x}", envelope.ty()),
            "chainId": format!("0x{:x}", chain_id.unwrap_or(0)),
            "v": format!("0x{:x}", sig.v() as u64),
            "r": format!("0x{:x}", sig.r()),
            "s": format!("0x{:x}", sig.s()),
            "maxFeePerGas": format!("0x{:x}", envelope.max_fee_per_gas()),
            "maxPriorityFeePerGas": format!("0x{:x}", envelope.max_priority_fee_per_gas().unwrap_or(0)),
            "gasPrice": format!("0x{:x}", envelope.max_fee_per_gas()),
        });
    }

    // Fallback: raw data with fake hash
    serde_json::json!(fake_tx_hash(block_hash, tx_index))
}

/// Serialize a receipt envelope to JSON, including its logs.
fn receipt_to_json(
    receipt: &alloy_consensus::ReceiptEnvelope,
    block_hash: B256,
    block_number: u64,
    tx_idx: usize,
    tx_hash: B256,
    log_index_offset: usize,
) -> Value {
    let logs: Vec<Value> = receipt
        .logs()
        .iter()
        .enumerate()
        .map(|(li, log)| {
            serde_json::json!({
                "address": log.address,
                "topics": log.topics(),
                "data": log.data.data,
                "blockHash": block_hash,
                "blockNumber": format!("0x{:x}", block_number),
                "transactionHash": tx_hash,
                "transactionIndex": format!("0x{:x}", tx_idx),
                "logIndex": format!("0x{:x}", log_index_offset + li),
                "removed": false,
            })
        })
        .collect();

    serde_json::json!({
        "status": format!("0x{:x}", receipt.status() as u8),
        "cumulativeGasUsed": format!("0x{:x}", receipt.cumulative_gas_used()),
        "logs": logs,
        "logsBloom": receipt.logs_bloom(),
        "transactionHash": tx_hash,
        "transactionIndex": format!("0x{:x}", tx_idx),
        "blockHash": block_hash,
        "blockNumber": format!("0x{:x}", block_number),
        "type": format!("0x{:x}", receipt.ty()),
    })
}

impl L1RpcServer for L1RpcImpl {
    fn chain_id(&self) -> Result<String, ErrorObjectOwned> {
        Ok(format!("0x{:x}", self.chain_id))
    }

    fn get_block_by_hash(
        &self,
        hash: B256,
        full_txs: bool,
    ) -> Result<Option<Value>, ErrorObjectOwned> {
        Ok(self.by_hash.get(&hash).map(|&i| self.block_to_json(&self.blocks[i], full_txs)))
    }

    fn get_block_by_number(
        &self,
        number: String,
        full_txs: bool,
    ) -> Result<Option<Value>, ErrorObjectOwned> {
        Ok(self
            .resolve_block_number(&number)
            .map(|i| self.block_to_json(&self.blocks[i], full_txs)))
    }

    fn get_block_receipts(&self, hash: B256) -> Result<Option<Value>, ErrorObjectOwned> {
        Ok(self.by_hash.get(&hash).map(|&i| {
            let block = &self.blocks[i];
            let block_number = block.header.inner().number;
            let mut log_index_offset = 0;
            let receipts: Vec<Value> = block
                .receipts
                .iter()
                .enumerate()
                .map(|(j, r)| {
                    let tx_hash = alloy_consensus::TxEnvelope::decode_2718(
                        &mut block.transactions.get(j).map(|b| b.as_ref()).unwrap_or(&[]),
                    )
                    .map(|env| *env.tx_hash())
                    .unwrap_or_else(|_| fake_tx_hash(hash, j));
                    let json = receipt_to_json(r, hash, block_number, j, tx_hash, log_index_offset);
                    log_index_offset += r.logs().len();
                    json
                })
                .collect();
            serde_json::json!(receipts)
        }))
    }

    fn get_raw_header(&self, hash: B256) -> Result<Bytes, ErrorObjectOwned> {
        let block =
            self.by_hash.get(&hash).map(|&i| &self.blocks[i]).ok_or_else(|| {
                ErrorObjectOwned::owned(-32602, "block not found", None::<String>)
            })?;
        let mut buf = Vec::new();
        block.header.inner().encode(&mut buf);
        Ok(Bytes::from(buf))
    }

    fn get_raw_receipts(&self, hash: B256) -> Result<Vec<Bytes>, ErrorObjectOwned> {
        let block =
            self.by_hash.get(&hash).map(|&i| &self.blocks[i]).ok_or_else(|| {
                ErrorObjectOwned::owned(-32602, "block not found", None::<String>)
            })?;

        use alloy_eips::Encodable2718;
        Ok(block
            .receipts
            .iter()
            .map(|r| {
                let mut buf = Vec::new();
                r.encode_2718(&mut buf);
                Bytes::from(buf)
            })
            .collect())
    }

    fn get_transaction_receipt(&self, tx_hash: B256) -> Result<Option<Value>, ErrorObjectOwned> {
        Ok(self.tx_index.get(&tx_hash).map(|&(block_idx, tx_idx)| {
            let block = &self.blocks[block_idx];
            let block_hash = block.header.hash();
            let block_number = block.header.inner().number;
            let receipt = &block.receipts[tx_idx];
            let log_index_offset: usize =
                block.receipts[..tx_idx].iter().map(|r| r.logs().len()).sum();

            receipt_to_json(receipt, block_hash, block_number, tx_idx, tx_hash, log_index_offset)
        }))
    }
}
