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

#[cfg(test)]
mod test {
    use super::*;
    use reth_db::database::Database;
    use reth_primitives::{
        address, b256, bloom, hex, AccessList, Block, Bytes, Header, Log as RethLog, Receipts,
        SealedBlockWithSenders, Signature, Transaction, TxEip1559, TxType, TxValue,
        EMPTY_OMMER_ROOT_HASH,
    };
    use reth_provider::{BlockWriter, BundleStateWithReceipts, DatabaseProvider};
    use reth_revm::revm::db::BundleState;
    use std::{path::Path, str::FromStr};

    #[test]
    fn generate_testdata_db() {
        let db = reth_db::init_db(Path::new("testdata"), None).unwrap();
        let pr = DatabaseProvider::new_rw(db.tx_mut().unwrap(), MAINNET.clone());

        let block = Block {
            header: Header {
                parent_hash: b256!(
                    "a2feb804b2ec06df67df4851a2ef75524820febc1a140ad5db424b80f9c3114d"
                ),
                ommers_hash: EMPTY_OMMER_ROOT_HASH,
                beneficiary: address!("0000000000000000000000000000000000000000"),
                state_root: b256!(
                    "56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
                ),
                transactions_root: b256!(
                    "78aecefe9a8944f627b6ffef3aad9ab5f5a5031e360bd014a10a50bcf37979c6"
                ),
                receipts_root: b256!(
                    "99bdc617e7e3781b02ce06c06a77acd45988be16be63d58578a4399f3cc10fed"
                ),
                withdrawals_root: Some(b256!(
                    "558291986c64e0ef409d79093c5f4306257fa56179f07efe4483eeaa14299a0c"
                )),
                logs_bloom: bloom!("00b8830810238200002802008031000400a80400054013c04083000a11000082820028c40500100209140a4202018028000a0a344921910c001286001024000010834000ec4004010000002b82108423461b8460020600001404031680200020004010008e4a08500528418800010804100000c809600200008a0098800810c2008220100112250062c044050001404080651013422442da000101400500041002281000031100000300008010104a0800110208800051804ac41a2420000110e0104103102242c0020a2000041042c8040201024004871471018012404065280c30021c202082030800040000020808020104421010c241c80a400408020054"),
                difficulty: U256::ZERO,
                number: 9942861,
                gas_limit: 0x1c9c380,
                gas_used: 0xc91a7e,
                timestamp: 0x653c5c8c,
                mix_hash: b256!("c7bd100be413127b4e4695b29835cb15592c81e98b704b49838d358d13642c56"),
                nonce: 0,
                base_fee_per_gas: Some(9),
                blob_gas_used: None,
                excess_blob_gas: None,
                parent_beacon_block_root: None,
                extra_data: hex!("d883010b04846765746888676f312e32302e32856c696e7578").into(),
            },
            body: vec![
                TransactionSigned {
                    hash: b256!("12c0074a4a7916fe6f39de8417fe93f1fa77bcadfd5fc31a317fb6c344f66602"),
                    signature: Signature {
                        r: U256::from_str("0x200a045cf9b74dc7eaa71cbbc257c0d8365a11c3dc3f547267f4d93e3863e358").unwrap(),
                        s:  U256::from_str("0x1f9f7a37b2fa471c9212009c1f19daf3f03dbfd1787be7e227b56765daf084a").unwrap(),
                        odd_y_parity: true
                    },
                    transaction: Transaction::Eip1559(TxEip1559 {
                        chain_id: 5,
                        nonce: 0x4b4b,
                        gas_limit: 0x3c03f,
                        max_fee_per_gas: 0x59682f12,
                        max_priority_fee_per_gas: 0x59682f00,
                        to: TransactionKind::Call(address!("4ce63f351597214ef0b9a319124eea9e0f9668bb")),
                        value: TxValue::from(U256::ZERO),
                        access_list: AccessList::default(),
                        input: hex!("70ab1eb60000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000c200000000000000000000000000000000000000000000000000000000000000003ed1c85eb0477c9ac0308a4c7022c37e606627b328daa4ab363f44981e287d69bb075d81fcbff15450b978f9b84ca9fd9ca96b1e8faf3ea1f2951e496980b466186ae4a9f759f4d75d4fe28fde9d6ebad99f49cb30f791a2bfc85a8a2a36569f00000000000000000000000000000000000000000000000000000000653c5bf50c07ca9327b541241b9a7d856294622c1b03d4991fdf44537d97173709a7c7f4084a7f906d3e5594377cd9d7c36fc66c53716e69c8114b8fa425ad06e53807302eb1efd7eaf8c72107458873cda1b771bb5bf0154caa2ed63d3073e970cf63da0c1d1e58f31dff4dba615c61b3996a01d41e1f45999ea132e254c8e6129e535817235adea1ec0def8111508cc9b658347db64bdf3904c592f5ad4d9258f57b0c167f59373778385fc2f01ee9539befaaf97a8d540ae926242061d2da5fea4a91152ea7d88c390b356fb780a6f93c57efa6aab34d9409dec4dd23bc0ffa8f3f7825dd47e27434b2e4d9d9730db0ae0c2faa556f0e7440724d2c44c527c4d1ad8e29da7229592b10d727c8a7d633c8a0e6240db2452282ecee26ef3d8d9980b463").into()
                    }
              ) }
            ],
            ommers: vec![],
            withdrawals: None,
        };

        pr.append_blocks_with_bundle_state(
            vec![SealedBlockWithSenders {
                block: block.seal_slow(),
                senders: vec![address!("a24efab96523efa6abb2de9b2c16205cfa3c1dc8")],
            }],
            BundleStateWithReceipts::new(
                BundleState::default(),
                Receipts::from_block_receipt(vec![Receipt {
                    tx_type: TxType::EIP1559,
                    success: true,
                    cumulative_gas_used: 0x3aefc,
                    logs: vec![RethLog {
                        address: address!("4ce63f351597214ef0b9a319124eea9e0f9668bb"),
                        topics: vec![
                            b256!(
                                "0cdbd8bd7813095001c5fe7917bd69d834dc01db7c1dfcf52ca135bd20384413"
                            ),
                            b256!(
                                "00000000000000000000000000000000000000000000000000000000000000c2"
                            ),
                        ],
                        data: Bytes::default(),
                    }],
                }]),
                9942861,
            ),
            None,
        )
        .unwrap();
        pr.commit().unwrap();
    }
}
