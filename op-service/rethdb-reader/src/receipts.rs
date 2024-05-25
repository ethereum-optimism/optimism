//! This module contains the logic for reading a block's fully hydrated receipts directly from the
//! [reth] database.

use anyhow::{anyhow, Result};
use reth_blockchain_tree::noop::NoopBlockchainTree;
use reth_db::open_db_read_only;
use reth_primitives::{
    BlockHashOrNumber, Receipt, TransactionKind, TransactionMeta, TransactionSigned, MAINNET, U128,
    U256, U64,
};
use reth_provider::{providers::BlockchainProvider, BlockReader, ProviderFactory, ReceiptProvider};
use reth_rpc_types::{Log, TransactionReceipt};
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
        Self { data, data_len, error: false }
    }

    /// Constructs a failing [ReceiptsResult] with a null pointer to the data.
    pub fn fail() -> Self {
        Self { data: std::ptr::null_mut(), data_len: 0, error: true }
    }
}

/// Read the receipts for a blockhash from the RETH database directly.
///
/// # Safety
/// - All possible nil pointer dereferences are checked, and the function will return a failing
///   [ReceiptsResult] if any are found.
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
    let block =
        provider.block_by_hash(block_hash.into())?.ok_or(anyhow!("Failed to fetch block"))?;
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
        status_code: if receipt.success { Some(U64::from(1)) } else { Some(U64::from(0)) },

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

#[cfg(test)]
mod test {
    use super::*;
    use alloy_rlp::Decodable;
    use reth_db::database::Database;
    use reth_primitives::{
        address, b256, bloom, hex, Address, Block, Bytes, ReceiptWithBloom, Receipts,
        SealedBlockWithSenders, U8,
    };
    use reth_provider::{BlockWriter, BundleStateWithReceipts, DatabaseProvider};
    use reth_revm::revm::db::BundleState;
    use std::{ffi::CString, fs::File, path::Path};

    #[inline]
    fn dummy_block_with_receipts() -> Result<(Block, Vec<Receipt>)> {
        // To generate testdata (block 18,663,292 on Ethereum Mainnet):
        // 1. BLOCK RLP: `cast rpc debug_getRawBlock 0x11CC77C | jq -r | xxd -r -p >
        //    testdata/block.rlp`
        // 2. RECEIPTS RLP: `cast rpc debug_getRawReceipts 0x11CC77C | jq -r >
        //    testdata/receipts.json`
        let block_rlp = include_bytes!("../testdata/block.rlp");
        let block = Block::decode(&mut block_rlp.as_ref())?;

        let receipt_rlp: Vec<Vec<u8>> = serde_json::from_str(include_str!(
            "../testdata/receipts.json"
        ))
        .map(|v: Vec<String>| {
            v.into_iter().map(|s| hex::decode(s)).collect::<Result<Vec<Vec<u8>>, _>>()
        })??;
        let receipts = receipt_rlp
            .iter()
            .map(|r| ReceiptWithBloom::decode(&mut r.as_slice()).map(|r| r.receipt))
            .collect::<Result<Vec<Receipt>, _>>()?;

        Ok((block, receipts))
    }

    #[inline]
    fn open_receipts_testdata_db() -> Result<()> {
        if File::open("testdata/db").is_ok() {
            return Ok(())
        }

        // Open a RW handle to the MDBX database
        let db = reth_db::init_db(Path::new("testdata/db"), None).map_err(|e| anyhow!(e))?;
        let pr = DatabaseProvider::new_rw(db.tx_mut()?, MAINNET.clone());

        // Grab the dummy block and receipts
        let (mut block, receipts) = dummy_block_with_receipts()?;

        // Patch: The block's current state root expects the rest of the chain history to be in the
        // DB; manually override it. Otherwise, the DatabaseProvider will fail to commit the
        // block.
        block.header.state_root = reth_primitives::constants::EMPTY_ROOT_HASH;

        // Fetch the block number and tx senders for bundle state creation.
        let block_number = block.header.number;
        let senders = block
            .body
            .iter()
            .map(|tx| tx.recover_signer())
            .collect::<Option<Vec<Address>>>()
            .ok_or(anyhow!("Failed to recover signers"))?;

        // Commit the bundle state to the database
        pr.append_blocks_with_bundle_state(
            vec![SealedBlockWithSenders { block: block.seal_slow(), senders }],
            BundleStateWithReceipts::new(
                BundleState::default(),
                Receipts::from_block_receipt(receipts),
                block_number,
            ),
            None,
        )?;
        pr.commit()?;

        Ok(())
    }

    #[test]
    fn fetch_receipts() {
        open_receipts_testdata_db().unwrap();

        unsafe {
            let mut block_hash =
                b256!("6a229123d607c2232a8b0bdd36f90745945d05181018e64e60ff2b93ab6b52e5");
            let receipts_res = super::read_receipts_inner(
                block_hash.as_mut_ptr(),
                32,
                CString::new("testdata/db").unwrap().into_raw() as *const c_char,
            )
            .unwrap();

            let receipts_data =
                std::slice::from_raw_parts(receipts_res.data as *const u8, receipts_res.data_len);
            let receipt = {
                let mut receipts: Vec<TransactionReceipt> =
                    serde_json::from_slice(receipts_data).unwrap();
                receipts.remove(0)
            };

            // Check the first receipt in the block for validity
            assert_eq!(receipt.transaction_type, U8::from(2));
            assert_eq!(receipt.status_code, Some(U64::from(1)));
            assert_eq!(receipt.cumulative_gas_used, U256::from(115_316));
            assert_eq!(receipt.logs_bloom, bloom!("00200000000000000000000080001000000000000000000000000000000000000000000000000000000000000000100002000100080000000000000000000000000000000000000000000008000000200000000400000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000400000000000001000000000000000100000000080000004000000000000000000000000000000000000002000000000000000000000000000000000000000006000000000000000000000000000000000000001000000000000000000000200000000000000100000000020000000000000000000000000000000010"));
            assert_eq!(
                receipt.logs[0].address,
                address!("c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")
            );
            assert_eq!(
                receipt.logs[0].topics[0],
                b256!("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
            );
            assert_eq!(
                receipt.logs[0].topics[1],
                b256!("00000000000000000000000000000000003b3cc22af3ae1eac0440bcee416b40")
            );
            assert_eq!(
                receipt.logs[0].data,
                Bytes::from_static(
                    hex!("00000000000000000000000000000000000000000000000008a30cd230000000")
                        .as_slice()
                )
            );
            assert_eq!(receipt.from, address!("41d3ab85aafed2ef9e644cb7d3bbca2fc4d8cac8"));
            assert_eq!(receipt.to, Some(address!("00000000003b3cc22af3ae1eac0440bcee416b40")));
            assert_eq!(
                receipt.transaction_hash,
                Some(b256!("88b2d153a4e893ba91ac235325c44b1aa0c802fcb42657701e1a73e1c675f7ca"))
            );

            assert_eq!(receipt.block_number, Some(U256::from(18_663_292)));
            assert_eq!(receipt.block_hash, Some(block_hash));
            assert_eq!(receipt.transaction_index, U64::from(0));

            crate::rdb_free_string(receipts_res.data as *mut c_char);
        }
    }
}
