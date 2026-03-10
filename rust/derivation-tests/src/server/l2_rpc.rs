//! L2 JSON-RPC server implementation.

use alloy_primitives::{B256, Bytes};
use alloy_rlp::Encodable;
use jsonrpsee::{proc_macros::rpc, types::ErrorObjectOwned};
use serde_json::Value;
use std::collections::HashMap;

use crate::{l2::L2Block, state::StateSnapshot};

/// L2 RPC server trait.
#[rpc(server)]
pub(crate) trait L2Rpc {
    /// Returns the L2 chain ID.
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

    /// Returns account and storage proofs.
    #[method(name = "eth_getProof")]
    fn get_proof(
        &self,
        address: alloy_primitives::Address,
        storage_keys: Vec<B256>,
        block_id: String,
    ) -> Result<Value, ErrorObjectOwned>;

    /// Returns raw trie node or contract code by hash.
    #[method(name = "debug_dbGet")]
    fn db_get(&self, key: Bytes) -> Result<Bytes, ErrorObjectOwned>;

    /// Returns RLP-encoded header.
    #[method(name = "debug_getRawHeader")]
    fn get_raw_header(&self, block_id: String) -> Result<Bytes, ErrorObjectOwned>;

    /// Unsupported — returns method not found.
    #[method(name = "debug_executePayload")]
    fn execute_payload(&self) -> Result<Value, ErrorObjectOwned>;
}

/// L2 RPC server implementation.
pub(crate) struct L2RpcImpl {
    blocks: Vec<L2Block>,
    snapshots: Vec<StateSnapshot>,
    by_hash: HashMap<B256, usize>,
    chain_id: u64,
}

impl L2RpcImpl {
    /// Create from L2 blocks and their state snapshots.
    pub(crate) fn new(blocks: Vec<L2Block>, snapshots: Vec<StateSnapshot>, chain_id: u64) -> Self {
        let by_hash: HashMap<B256, usize> =
            blocks.iter().enumerate().map(|(i, b)| (b.header.hash(), i)).collect();
        Self { blocks, snapshots, by_hash, chain_id }
    }

    fn resolve_block(&self, block_id: &str) -> Option<usize> {
        // Try as hash first
        if block_id.starts_with("0x") &&
            block_id.len() == 66 &&
            let Ok(hash) = block_id.parse::<B256>()
        {
            return self.by_hash.get(&hash).copied();
        }

        // Try as number or tag
        match block_id {
            "latest" | "safe" | "finalized" => Some(self.blocks.len() - 1),
            "earliest" => Some(0),
            hex => {
                let n = u64::from_str_radix(hex.trim_start_matches("0x"), 16).ok()?;
                ((n as usize) < self.blocks.len()).then_some(n as usize)
            }
        }
    }

    fn block_to_json(&self, block: &L2Block, _full_txs: bool) -> Value {
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
            "miner": header.beneficiary,
            "extraData": "0x",
            "nonce": "0x0000000000000000",
            "sha3Uncles": header.ommers_hash,
            "logsBloom": header.logs_bloom,
            "size": "0x0",
            "withdrawalsRoot": header.withdrawals_root.unwrap_or(crate::state::roots::EMPTY_ROOT_HASH),
            "withdrawals": [],
            "mixHash": header.mix_hash,
            "transactions": block.transactions.iter().enumerate().map(|(i, _)| {
                format!("0x{:064x}", i)
            }).collect::<Vec<_>>(),
        })
    }
}

impl L2RpcServer for L2RpcImpl {
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
        Ok(self.resolve_block(&number).map(|i| self.block_to_json(&self.blocks[i], full_txs)))
    }

    fn get_proof(
        &self,
        address: alloy_primitives::Address,
        storage_keys: Vec<B256>,
        block_id: String,
    ) -> Result<Value, ErrorObjectOwned> {
        let idx = self
            .resolve_block(&block_id)
            .ok_or_else(|| ErrorObjectOwned::owned(-32602, "block not found", None::<String>))?;

        let snapshot = &self.snapshots[idx];

        // Compute storage roots for the proof generation
        let mut storage_roots = std::collections::BTreeMap::new();
        for (addr, storage) in &snapshot.storage {
            if !storage.is_empty() {
                let mut node_store = crate::state::TrieNodeStore::new();
                let root = crate::state::compute_storage_root(storage, &mut node_store);
                storage_roots.insert(*addr, root);
            }
        }

        let proof = crate::state::generate_account_proof(
            &snapshot.accounts,
            &snapshot.storage,
            &storage_roots,
            address,
            &storage_keys,
        );

        serde_json::to_value(&proof).map_err(|e| {
            ErrorObjectOwned::owned(-32603, format!("serialization error: {e}"), None::<String>)
        })
    }

    fn db_get(&self, key: Bytes) -> Result<Bytes, ErrorObjectOwned> {
        // Look up in the latest snapshot's node store
        let snapshot = self
            .snapshots
            .last()
            .ok_or_else(|| ErrorObjectOwned::owned(-32603, "no state available", None::<String>))?;

        // kona-host convention: 0x63 prefix + 32-byte code hash → contract code lookup
        if key.len() == 33 && key[0] == 0x63 {
            let code_hash = B256::from_slice(&key[1..]);
            if let Some(code) = snapshot.code.get(&code_hash) {
                return Ok(Bytes::from(code.clone()));
            }
            return Err(ErrorObjectOwned::owned(-32602, "code not found", None::<String>));
        }

        // Try direct hash lookup for trie nodes
        if key.len() == 32 {
            let hash = B256::from_slice(&key);
            if let Some(data) = snapshot.node_store.get(&hash) {
                return Ok(data.clone());
            }
        }

        Err(ErrorObjectOwned::owned(-32602, "key not found", None::<String>))
    }

    fn get_raw_header(&self, block_id: String) -> Result<Bytes, ErrorObjectOwned> {
        let idx = self
            .resolve_block(&block_id)
            .ok_or_else(|| ErrorObjectOwned::owned(-32602, "block not found", None::<String>))?;

        let mut buf = Vec::new();
        self.blocks[idx].header.inner().encode(&mut buf);
        Ok(Bytes::from(buf))
    }

    fn execute_payload(&self) -> Result<Value, ErrorObjectOwned> {
        Err(ErrorObjectOwned::owned(-32601, "method not found", None::<String>))
    }
}
