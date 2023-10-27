use reth::{
    blockchain_tree::noop::NoopBlockchainTree,
    primitives::{
        BlockHashOrNumber, ChainSpecBuilder, Receipt, TransactionMeta, TransactionSigned, U128,
        U256, U64,
    },
    providers::{providers::BlockchainProvider, BlockReader, ProviderFactory, ReceiptProvider},
    revm::primitives::calc_blob_gasprice,
    rpc::types::{Log, TransactionReceipt},
    utils::db::open_db_read_only,
};
use std::{os::raw::c_char, path::Path, sync::Arc};

#[repr(C)]
pub struct ReceiptsResult {
    data: *mut char,
    data_len: usize,
    error: bool,
}

impl ReceiptsResult {
    pub fn success(data: *mut char, data_len: usize) -> Self {
        Self {
            data,
            data_len,
            error: false,
        }
    }

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
/// WARNING: Will panic on error.
/// TODO: Gracefully return OK status.
#[no_mangle]
pub extern "C" fn read_receipts(
    block_hash: *const u8,
    block_hash_len: usize,
    db_path: *const c_char,
) -> ReceiptsResult {
    // Convert the raw pointer and length back to a Rust slice
    let Ok(block_hash): Result<[u8; 32], _> =
        unsafe { std::slice::from_raw_parts(block_hash, block_hash_len) }.try_into()
    else {
        return ReceiptsResult::fail();
    };

    // Convert the *const c_char to a Rust &str
    let Ok(db_path_str) = unsafe {
        assert!(!db_path.is_null(), "Null pointer for database path");
        std::ffi::CStr::from_ptr(db_path)
    }
    .to_str() else {
        return ReceiptsResult::fail();
    };

    let Ok(db) = open_db_read_only(&Path::new(db_path_str), None) else {
        return ReceiptsResult::fail();
    };
    let spec = Arc::new(ChainSpecBuilder::mainnet().build());
    let factory = ProviderFactory::new(db, spec.clone());

    // Create a read-only BlockChainProvider
    let Ok(provider) = BlockchainProvider::new(factory, NoopBlockchainTree::default()) else {
        return ReceiptsResult::fail();
    };

    let Ok(block) = provider.block_by_hash(block_hash.into()) else {
        return ReceiptsResult::fail();
    };

    let Ok(receipts) = provider.receipts_by_block(BlockHashOrNumber::Hash(block_hash.into()))
    else {
        return ReceiptsResult::fail();
    };

    if let (Some(block), Some(receipts)) = (block, receipts) {
        let block_number = block.number;
        let base_fee = block.base_fee_per_gas;
        let block_hash = block.hash_slow();
        let excess_blob_gas = block.excess_blob_gas;
        let Some(receipts) = block
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
                    excess_blob_gas,
                };
                build_transaction_receipt_with_block_receipts(tx, meta, receipt, &receipts)
            })
            .collect::<Option<Vec<_>>>()
        else {
            return ReceiptsResult::fail();
        };

        // Convert the receipts to JSON for transport
        let Ok(mut receipts_json) = serde_json::to_string(&receipts) else {
            return ReceiptsResult::fail();
        };

        let res =
            ReceiptsResult::success(receipts_json.as_mut_ptr() as *mut char, receipts_json.len());

        // Forget the `receipts_json` string so that its memory isn't freed by the
        // borrow checker at the end of this scope
        std::mem::forget(receipts_json); // Prevent Rust from freeing the memory

        res
    } else {
        ReceiptsResult::fail()
    }
}

/// Free a string that was allocated in Rust and passed to C.
#[no_mangle]
pub extern "C" fn free_string(string: *mut c_char) {
    unsafe {
        // Convert the raw pointer back to a CString and let it go out of scope,
        // which will deallocate the memory.
        if !string.is_null() {
            let _ = std::ffi::CString::from_raw(string);
        }
    }
}

pub(crate) fn build_transaction_receipt_with_block_receipts(
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
        blob_gas_price: meta.excess_blob_gas.map(calc_blob_gasprice).map(U128::from),
        blob_gas_used: transaction.transaction.blob_gas_used().map(U128::from),
    };

    match tx.transaction.kind() {
        reth::primitives::TransactionKind::Create => {
            res_receipt.contract_address =
                Some(transaction.signer().create(tx.transaction.nonce()));
        }
        reth::primitives::TransactionKind::Call(addr) => {
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
