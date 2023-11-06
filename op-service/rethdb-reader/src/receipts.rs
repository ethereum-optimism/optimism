//! This module contains the logic for reading a block's fully hydrated receipts directly from the
//! [reth] database.

use anyhow::{anyhow, Result};
use reth::{
    blockchain_tree::noop::NoopBlockchainTree,
    primitives::{
        BlockHashOrNumber, Receipt, TransactionKind, TransactionMeta, TransactionSigned, MAINNET,
        U128, U256, U64,
    },
    providers::{providers::BlockchainProvider, BlockReader, ProviderFactory, ReceiptProvider},
    rpc::types::{Log, TransactionReceipt},
    utils::db::open_db_read_only,
};
use std::{ffi::c_char, path::Path};

/// A [ReceiptsResult] is a wrapper around a JSON string containing serialized [TransactionReceipt]s
/// as well as an error status that is compatible with FFI.
///
/// # Safety
/// - When the `error` field is false, the `data` pointer is guaranteed to be valid.
/// - When the `error` field is true, the `data` pointer is guaranteed to be null.
#[repr(C)]
pub struct ReceiptsResult {
    data: *mut char,
    data_len: usize,
    error: bool,
}

impl ReceiptsResult {
    /// Constructs a successful [ReceiptsResult] from a JSON string.
    pub fn success(data: *mut char, data_len: usize) -> Self {
        Self {
            data,
            data_len,
            error: false,
        }
    }

    /// Constructs a failing [ReceiptsResult] with a null pointer to the data.
    pub fn fail() -> Self {
        Self {
            data: std::ptr::null_mut(),
            data_len: 0,
            error: true,
        }
    }
}

/// Read the receipts for a blockhash from the RETH database directly.
///
/// # Safety
/// - All possible nil pointer dereferences are checked, and the function will return a
///   failing [ReceiptsResult] if any are found.
#[inline(always)]
pub(crate) unsafe fn read_receipts_inner(
    block_hash: *const u8,
    block_hash_len: usize,
    db_path: *const c_char,
) -> Result<ReceiptsResult> {
    // Convert the raw pointer and length back to a Rust slice
    let block_hash: [u8; 32] = {
        if block_hash.is_null() {
            anyhow::bail!("block_hash pointer is null");
        }
        std::slice::from_raw_parts(block_hash, block_hash_len)
    }
    .try_into()?;

    // Convert the *const c_char to a Rust &str
    let db_path_str = {
        if db_path.is_null() {
            anyhow::bail!("db path pointer is null");
        }
        std::ffi::CStr::from_ptr(db_path)
    }
    .to_str()?;

    let db = open_db_read_only(Path::new(db_path_str), None).map_err(|e| anyhow!(e))?;
    let factory = ProviderFactory::new(db, MAINNET.clone());

    // Create a read-only BlockChainProvider
    let provider = BlockchainProvider::new(factory, NoopBlockchainTree::default())?;

    // Fetch the block and the receipts within it
    let block = provider
        .block_by_hash(block_hash.into())?
        .ok_or(anyhow!("Failed to fetch block"))?;
    let receipts = provider
        .receipts_by_block(BlockHashOrNumber::Hash(block_hash.into()))?
        .ok_or(anyhow!("Failed to fetch block receipts"))?;

    let block_number = block.number;
    let base_fee = block.base_fee_per_gas;
    let block_hash = block.hash_slow();
    let receipts = block
        .body
        .into_iter()
        .zip(receipts.clone())
        .enumerate()
        .map(|(idx, (tx, receipt))| {
            let meta = TransactionMeta {
                tx_hash: tx.hash,
                index: idx as u64,
                block_hash,
                block_number,
                base_fee,
                excess_blob_gas: None,
            };
            build_transaction_receipt_with_block_receipts(tx, meta, receipt, &receipts)
        })
        .collect::<Option<Vec<_>>>()
        .ok_or(anyhow!("Failed to build receipts"))?;

    // Convert the receipts to JSON for transport
    let mut receipts_json = serde_json::to_string(&receipts)?;

    // Create a ReceiptsResult with a pointer to the json-ified receipts
    let res = ReceiptsResult::success(receipts_json.as_mut_ptr() as *mut char, receipts_json.len());

    // Forget the `receipts_json` string so that its memory isn't freed by the
    // borrow checker at the end of this scope
    std::mem::forget(receipts_json); // Prevent Rust from freeing the memory

    Ok(res)
}

/// Builds a hydrated [TransactionReceipt] from information in the passed transaction,
/// receipt, and block receipts.
///
/// Returns [None] if the transaction's sender could not be recovered from the signature.
#[inline(always)]
fn build_transaction_receipt_with_block_receipts(
    tx: TransactionSigned,
    meta: TransactionMeta,
    receipt: Receipt,
    all_receipts: &[Receipt],
) -> Option<TransactionReceipt> {
    let transaction = tx.clone().into_ecrecovered()?;

    // get the previous transaction cumulative gas used
    let gas_used = if meta.index == 0 {
        receipt.cumulative_gas_used
    } else {
        let prev_tx_idx = (meta.index - 1) as usize;
        all_receipts
            .get(prev_tx_idx)
            .map(|prev_receipt| receipt.cumulative_gas_used - prev_receipt.cumulative_gas_used)
            .unwrap_or_default()
    };

    let mut res_receipt = TransactionReceipt {
        transaction_hash: Some(meta.tx_hash),
        transaction_index: U64::from(meta.index),
        block_hash: Some(meta.block_hash),
        block_number: Some(U256::from(meta.block_number)),
        from: transaction.signer(),
        to: None,
        cumulative_gas_used: U256::from(receipt.cumulative_gas_used),
        gas_used: Some(U256::from(gas_used)),
        contract_address: None,
        logs: Vec::with_capacity(receipt.logs.len()),
        effective_gas_price: U128::from(transaction.effective_gas_price(meta.base_fee)),
        transaction_type: tx.transaction.tx_type().into(),
        // TODO pre-byzantium receipts have a post-transaction state root
        state_root: None,
        logs_bloom: receipt.bloom_slow(),
        status_code: if receipt.success {
            Some(U64::from(1))
        } else {
            Some(U64::from(0))
        },

        // EIP-4844 fields
        blob_gas_price: None,
        blob_gas_used: None,
    };

    match tx.transaction.kind() {
        TransactionKind::Create => {
            res_receipt.contract_address =
                Some(transaction.signer().create(tx.transaction.nonce()));
        }
        TransactionKind::Call(addr) => {
            res_receipt.to = Some(*addr);
        }
    }

    // get number of logs in the block
    let mut num_logs = 0;
    for prev_receipt in all_receipts.iter().take(meta.index as usize) {
        num_logs += prev_receipt.logs.len();
    }

    for (tx_log_idx, log) in receipt.logs.into_iter().enumerate() {
        let rpclog = Log {
            address: log.address,
            topics: log.topics,
            data: log.data,
            block_hash: Some(meta.block_hash),
            block_number: Some(U256::from(meta.block_number)),
            transaction_hash: Some(meta.tx_hash),
            transaction_index: Some(U256::from(meta.index)),
            log_index: Some(U256::from(num_logs + tx_log_idx)),
            removed: false,
        };
        res_receipt.logs.push(rpclog);
    }

    Some(res_receipt)
}
