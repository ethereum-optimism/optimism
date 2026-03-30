//! L1 JSON-RPC server implementation.

use alloy_consensus::{
    ReceiptEnvelope, ReceiptWithBloom, Transaction, transaction::SignerRecoverable,
};
use alloy_eips::Decodable2718;
use alloy_primitives::{Address, B256, Bytes, keccak256};
use alloy_rlp::Encodable;
use alloy_rpc_types_eth::BlockTransactions;
use jsonrpsee::{proc_macros::rpc, types::ErrorObjectOwned};
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
    ) -> Result<
        Option<alloy_rpc_types_eth::Block<alloy_rpc_types_eth::Transaction>>,
        ErrorObjectOwned,
    >;

    /// Returns a block by number.
    #[method(name = "eth_getBlockByNumber")]
    fn get_block_by_number(
        &self,
        number: String,
        full_txs: bool,
    ) -> Result<
        Option<alloy_rpc_types_eth::Block<alloy_rpc_types_eth::Transaction>>,
        ErrorObjectOwned,
    >;

    /// Returns block receipts.
    #[method(name = "eth_getBlockReceipts")]
    fn get_block_receipts(
        &self,
        hash: B256,
    ) -> Result<Option<Vec<alloy_rpc_types_eth::TransactionReceipt>>, ErrorObjectOwned>;

    /// Returns the RLP-encoded header.
    #[method(name = "debug_getRawHeader")]
    fn get_raw_header(&self, hash: B256) -> Result<Bytes, ErrorObjectOwned>;

    /// Returns RLP-encoded receipts.
    #[method(name = "debug_getRawReceipts")]
    fn get_raw_receipts(&self, hash: B256) -> Result<Vec<Bytes>, ErrorObjectOwned>;

    /// Returns a transaction receipt by hash.
    #[method(name = "eth_getTransactionReceipt")]
    fn get_transaction_receipt(
        &self,
        tx_hash: B256,
    ) -> Result<Option<alloy_rpc_types_eth::TransactionReceipt>, ErrorObjectOwned>;
}

/// L1 RPC server implementation backed by in-memory blocks.
pub(crate) struct L1RpcImpl {
    blocks: Vec<L1Block>,
    by_hash: HashMap<B256, usize>,
    /// Maps tx hash to (block index, transaction index).
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

    fn to_rpc_block(
        &self,
        block: &L1Block,
        full_txs: bool,
    ) -> alloy_rpc_types_eth::Block<alloy_rpc_types_eth::Transaction> {
        let header = block.header.inner();
        let block_hash = block.header.hash();

        let transactions = if full_txs {
            BlockTransactions::Full(
                block
                    .transactions
                    .iter()
                    .enumerate()
                    .map(|(i, tx_bytes)| {
                        Self::to_rpc_transaction(tx_bytes, block_hash, header.number, i as u64)
                    })
                    .collect(),
            )
        } else {
            BlockTransactions::Hashes(
                block
                    .transactions
                    .iter()
                    .enumerate()
                    .map(|(i, tx_bytes)| {
                        alloy_consensus::TxEnvelope::decode_2718(&mut tx_bytes.as_ref())
                            .map(|tx| *tx.tx_hash())
                            .unwrap_or_else(|_| fake_tx_hash(block_hash, i))
                    })
                    .collect(),
            )
        };

        alloy_rpc_types_eth::Block {
            header: alloy_rpc_types_eth::Header {
                hash: block_hash,
                inner: header.clone(),
                total_difficulty: Some(alloy_primitives::U256::ZERO),
                size: None,
            },
            transactions,
            uncles: vec![],
            withdrawals: Some(Default::default()),
        }
    }

    fn to_rpc_transaction(
        tx_bytes: &Bytes,
        block_hash: B256,
        block_number: u64,
        tx_index: u64,
    ) -> alloy_rpc_types_eth::Transaction {
        let envelope = alloy_consensus::TxEnvelope::decode_2718(&mut tx_bytes.as_ref())
            .expect("L1 transactions should be valid RLP");
        let effective_gas_price = envelope.max_fee_per_gas();
        let recovered =
            envelope.try_into_recovered().expect("L1 transactions should have valid signatures");

        alloy_rpc_types_eth::Transaction {
            inner: recovered,
            block_hash: Some(block_hash),
            block_number: Some(block_number),
            transaction_index: Some(tx_index),
            effective_gas_price: Some(effective_gas_price),
        }
    }

    fn to_rpc_receipt(
        receipt: &ReceiptEnvelope,
        block: &L1Block,
        tx_idx: usize,
        tx_hash: B256,
        log_index_offset: usize,
    ) -> alloy_rpc_types_eth::TransactionReceipt {
        let block_hash = block.header.hash();
        let block_number = block.header.inner().number;
        let inner_receipt = receipt.as_receipt().expect("receipt should decode");

        let logs: Vec<alloy_rpc_types_eth::Log> = inner_receipt
            .logs
            .iter()
            .enumerate()
            .map(|(i, log)| alloy_rpc_types_eth::Log {
                inner: log.clone(),
                block_hash: Some(block_hash),
                block_number: Some(block_number),
                block_timestamp: Some(block.header.inner().timestamp),
                transaction_hash: Some(tx_hash),
                transaction_index: Some(tx_idx as u64),
                log_index: Some((log_index_offset + i) as u64),
                removed: false,
            })
            .collect();

        let rpc_inner_receipt = alloy_consensus::Receipt {
            status: inner_receipt.status,
            cumulative_gas_used: inner_receipt.cumulative_gas_used,
            logs,
        };

        let rpc_envelope = match receipt {
            ReceiptEnvelope::Legacy(_) => ReceiptEnvelope::Legacy(ReceiptWithBloom::new(
                rpc_inner_receipt,
                *receipt.logs_bloom(),
            )),
            ReceiptEnvelope::Eip2930(_) => ReceiptEnvelope::Eip2930(ReceiptWithBloom::new(
                rpc_inner_receipt,
                *receipt.logs_bloom(),
            )),
            ReceiptEnvelope::Eip1559(_) => ReceiptEnvelope::Eip1559(ReceiptWithBloom::new(
                rpc_inner_receipt,
                *receipt.logs_bloom(),
            )),
            ReceiptEnvelope::Eip4844(_) => ReceiptEnvelope::Eip4844(ReceiptWithBloom::new(
                rpc_inner_receipt,
                *receipt.logs_bloom(),
            )),
            ReceiptEnvelope::Eip7702(_) => ReceiptEnvelope::Eip7702(ReceiptWithBloom::new(
                rpc_inner_receipt,
                *receipt.logs_bloom(),
            )),
        };

        alloy_rpc_types_eth::TransactionReceipt {
            inner: rpc_envelope,
            transaction_hash: tx_hash,
            transaction_index: Some(tx_idx as u64),
            block_hash: Some(block_hash),
            block_number: Some(block_number),
            gas_used: inner_receipt.cumulative_gas_used,
            effective_gas_price: 0,
            blob_gas_used: None,
            blob_gas_price: None,
            from: Address::ZERO,
            to: None,
            contract_address: None,
        }
    }
}

impl L1RpcServer for L1RpcImpl {
    fn chain_id(&self) -> Result<String, ErrorObjectOwned> {
        Ok(format!("0x{:x}", self.chain_id))
    }

    fn get_block_by_hash(
        &self,
        hash: B256,
        full_txs: bool,
    ) -> Result<
        Option<alloy_rpc_types_eth::Block<alloy_rpc_types_eth::Transaction>>,
        ErrorObjectOwned,
    > {
        Ok(self.by_hash.get(&hash).map(|&i| self.to_rpc_block(&self.blocks[i], full_txs)))
    }

    fn get_block_by_number(
        &self,
        number: String,
        full_txs: bool,
    ) -> Result<
        Option<alloy_rpc_types_eth::Block<alloy_rpc_types_eth::Transaction>>,
        ErrorObjectOwned,
    > {
        Ok(self.resolve_block_number(&number).map(|i| self.to_rpc_block(&self.blocks[i], full_txs)))
    }

    fn get_block_receipts(
        &self,
        hash: B256,
    ) -> Result<Option<Vec<alloy_rpc_types_eth::TransactionReceipt>>, ErrorObjectOwned> {
        Ok(self.by_hash.get(&hash).map(|&i| {
            let block = &self.blocks[i];
            let mut log_index_offset = 0;
            block
                .receipts
                .iter()
                .enumerate()
                .map(|(j, r)| {
                    let tx_hash = alloy_consensus::TxEnvelope::decode_2718(
                        &mut block.transactions.get(j).map(|b| b.as_ref()).unwrap_or(&[]),
                    )
                    .map(|env| *env.tx_hash())
                    .unwrap_or_else(|_| fake_tx_hash(hash, j));
                    let receipt = Self::to_rpc_receipt(r, block, j, tx_hash, log_index_offset);
                    log_index_offset += r.logs().len();
                    receipt
                })
                .collect()
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

    fn get_transaction_receipt(
        &self,
        tx_hash: B256,
    ) -> Result<Option<alloy_rpc_types_eth::TransactionReceipt>, ErrorObjectOwned> {
        Ok(self.tx_index.get(&tx_hash).map(|&(block_idx, tx_idx)| {
            let block = &self.blocks[block_idx];
            let receipt = &block.receipts[tx_idx];
            let log_index_offset: usize =
                block.receipts[..tx_idx].iter().map(|r| r.logs().len()).sum();

            Self::to_rpc_receipt(receipt, block, tx_idx, tx_hash, log_index_offset)
        }))
    }
}
