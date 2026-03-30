//! L2 JSON-RPC server implementation.

use alloy_consensus::transaction::Recovered;
use alloy_primitives::{B256, Bytes};
use alloy_rlp::Encodable;
use jsonrpsee::{proc_macros::rpc, types::ErrorObjectOwned};
use op_alloy_consensus::OpTxEnvelope;
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
        number: Value,
        full_txs: bool,
    ) -> Result<Option<Value>, ErrorObjectOwned>;

    /// Returns account and storage proofs.
    #[method(name = "eth_getProof")]
    fn get_proof(
        &self,
        address: alloy_primitives::Address,
        storage_keys: Vec<B256>,
        block_id: Value,
    ) -> Result<Value, ErrorObjectOwned>;

    /// Returns raw trie node or contract code by hash.
    #[method(name = "debug_dbGet")]
    fn db_get(&self, key: Value) -> Result<Bytes, ErrorObjectOwned>;

    /// Returns RLP-encoded header.
    #[method(name = "debug_getRawHeader")]
    fn get_raw_header(&self, block_id: Value) -> Result<Bytes, ErrorObjectOwned>;

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

    fn resolve_block_str(&self, block_id: &str) -> Option<usize> {
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

    /// Resolve a block ID from a JSON value.
    /// Handles both string values ("0x1", "latest", "0xhash...") and
    /// object values ({"blockNumber": "0x1"}, {"blockHash": "0x..."}).
    fn resolve_block(&self, block_id: &Value) -> Option<usize> {
        match block_id {
            Value::String(s) => self.resolve_block_str(s),
            Value::Object(map) => {
                if let Some(Value::String(hash)) = map.get("blockHash") {
                    self.resolve_block_str(hash)
                } else if let Some(Value::String(num)) = map.get("blockNumber") {
                    self.resolve_block_str(num)
                } else {
                    None
                }
            }
            _ => None,
        }
    }

    /// Parse a key from a JSON value (string hex bytes).
    fn parse_bytes_key(val: &Value) -> Option<Bytes> {
        match val {
            Value::String(s) => s.parse::<Bytes>().ok(),
            _ => None,
        }
    }

    /// Convert an [`L2Block`] to its JSON-RPC representation using standard alloy types.
    ///
    /// This produces the same JSON as op-reth's `eth_getBlockByHash` response,
    /// ensuring compatibility with kona-host and op-program.
    fn block_to_json(&self, block: &L2Block, full_txs: bool) -> Value {
        let block_hash = block.header.hash();
        let block_number = block.header.inner().number;

        let rpc_header = alloy_rpc_types_eth::Header {
            hash: block_hash,
            inner: block.header.inner().clone(),
            total_difficulty: Some(alloy_primitives::U256::ZERO),
            size: Some(alloy_primitives::U256::ZERO),
        };

        type RpcTx = alloy_rpc_types_eth::Transaction<OpTxEnvelope>;

        let transactions = if full_txs {
            let txs: Vec<RpcTx> = block
                .transactions
                .iter()
                .enumerate()
                .map(|(i, tx)| {
                    let sender = match tx {
                        OpTxEnvelope::Deposit(sealed) => sealed.inner().from,
                        _ => alloy_primitives::Address::ZERO,
                    };
                    alloy_rpc_types_eth::Transaction {
                        inner: Recovered::new_unchecked(tx.clone(), sender),
                        block_hash: Some(block_hash),
                        block_number: Some(block_number),
                        transaction_index: Some(i as u64),
                        effective_gas_price: Some(0),
                    }
                })
                .collect();
            alloy_rpc_types_eth::BlockTransactions::Full(txs)
        } else {
            let hashes: Vec<B256> = block
                .transactions
                .iter()
                .map(|tx| {
                    let mut buf = Vec::new();
                    tx.encode(&mut buf);
                    alloy_primitives::keccak256(&buf)
                })
                .collect();
            alloy_rpc_types_eth::BlockTransactions::Hashes(hashes)
        };

        let rpc_block: alloy_rpc_types_eth::Block<RpcTx> = alloy_rpc_types_eth::Block {
            header: rpc_header,
            transactions,
            uncles: vec![],
            withdrawals: Some(vec![].into()),
        };

        serde_json::to_value(&rpc_block).expect("block serialization should not fail")
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
        number: Value,
        full_txs: bool,
    ) -> Result<Option<Value>, ErrorObjectOwned> {
        Ok(self.resolve_block(&number).map(|i| self.block_to_json(&self.blocks[i], full_txs)))
    }

    fn get_proof(
        &self,
        address: alloy_primitives::Address,
        storage_keys: Vec<B256>,
        block_id: Value,
    ) -> Result<Value, ErrorObjectOwned> {
        let idx = self.resolve_block(&block_id).ok_or_else(|| {
            ErrorObjectOwned::owned(-32602, format!("block not found: {block_id}"), None::<String>)
        })?;

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

    fn db_get(&self, key: Value) -> Result<Bytes, ErrorObjectOwned> {
        let key_bytes = Self::parse_bytes_key(&key).ok_or_else(|| {
            ErrorObjectOwned::owned(-32602, format!("invalid key: {key}"), None::<String>)
        })?;

        if self.snapshots.is_empty() {
            return Err(ErrorObjectOwned::owned(-32603, "no state available", None::<String>));
        }

        // kona-host convention: 0x63 prefix + 32-byte code hash → contract code lookup
        if key_bytes.len() == 33 && key_bytes[0] == 0x63 {
            let code_hash = B256::from_slice(&key_bytes[1..]);
            for snapshot in &self.snapshots {
                if let Some(code) = snapshot.code.get(&code_hash) {
                    return Ok(Bytes::from(code.clone()));
                }
            }
            return Err(ErrorObjectOwned::owned(-32602, "code not found", None::<String>));
        }

        // Try hash lookup for trie nodes across all snapshots.
        if key_bytes.len() == 32 {
            let hash = B256::from_slice(&key_bytes);
            for snapshot in &self.snapshots {
                if let Some(data) = snapshot.node_store.get(&hash) {
                    return Ok(data.clone());
                }
            }
        }

        Err(ErrorObjectOwned::owned(-32602, "key not found", None::<String>))
    }

    fn get_raw_header(&self, block_id: Value) -> Result<Bytes, ErrorObjectOwned> {
        let idx = self.resolve_block(&block_id).ok_or_else(|| {
            ErrorObjectOwned::owned(-32602, format!("block not found: {block_id}"), None::<String>)
        })?;

        let mut buf = Vec::new();
        self.blocks[idx].header.inner().encode(&mut buf);
        Ok(Bytes::from(buf))
    }

    fn execute_payload(&self) -> Result<Value, ErrorObjectOwned> {
        Err(ErrorObjectOwned::owned(-32601, "method not found", None::<String>))
    }
}
