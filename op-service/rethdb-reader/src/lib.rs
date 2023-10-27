use reth::{
    blockchain_tree::noop::NoopBlockchainTree,
    primitives::{
        alloy_primitives::private::alloy_rlp::Encodable, BlockHashOrNumber, ChainSpecBuilder,
    },
    providers::{providers::BlockchainProvider, ProviderFactory, ReceiptProvider},
    utils::db::open_db_read_only,
};
use std::{os::raw::c_char, path::Path, sync::Arc};

#[repr(C)]
pub struct ByteArray {
    data: *mut u8,
    len: usize,
}

#[repr(C)]
pub struct ByteArrays {
    data: *mut ByteArray,
    len: usize,
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
) -> ByteArrays {
    // Convert the raw pointer and length back to a Rust slice
    let block_hash: [u8; 32] = unsafe { std::slice::from_raw_parts(block_hash, block_hash_len) }
        .try_into()
        .expect("Block hash must be 32 bytes long");

    // Convert the *const c_char to a Rust &str
    let db_path_str = unsafe {
        assert!(!db_path.is_null(), "Null pointer for database path");
        std::ffi::CStr::from_ptr(db_path)
            .to_str()
            .expect("Invalid UTF-8 for database path")
    };

    let db = open_db_read_only(&Path::new(db_path_str), None).expect("Could not open reth DB");
    let spec = Arc::new(ChainSpecBuilder::mainnet().build());
    let factory = ProviderFactory::new(db, spec.clone());

    // Create a read-only BlockChainProvider
    let provider = BlockchainProvider::new(factory, NoopBlockchainTree::default())
        .expect("Failed to create blockchain provider.");
    let receipts = provider
        .receipts_by_block(BlockHashOrNumber::Hash(block_hash.into()))
        .expect("Could not fetch receipts for block")
        .expect("No receipts found for block");

    // Serialize receipts to RLP for the FFI interface.
    let receipts_rlp = receipts
        .into_iter()
        .map(|r| {
            // todo - reduce alloc?
            // RLP encode the receipt with a bloom filter.
            let mut buf = Vec::default();
            r.with_bloom().encode(&mut buf);

            // Return a pointer to the `buf` and its length
            let res = ByteArray {
                data: buf.as_mut_ptr(),
                len: buf.len(),
            };

            // Forget the `buf` so that its memory isn't freed by the
            // borrow checker at the end of this scope
            std::mem::forget(buf);

            res
        })
        .collect::<Vec<_>>();

    let result = ByteArrays {
        data: receipts_rlp.as_ptr() as *mut ByteArray,
        len: receipts_rlp.len(),
    };

    // Forget the `receipts_rlp` arr so that its memory isn't freed by the
    // borrow checker at the end of this scope
    std::mem::forget(receipts_rlp); // Prevent Rust from freeing the memory

    result
}

/// Free the [ByteArrays] data structure and its sub-components when they are no longer needed.
#[no_mangle]
pub extern "C" fn free_byte_arrays(array: ByteArrays) {
    unsafe {
        let arrays = Vec::from_raw_parts(array.data, array.len, array.len);
        for inner_array in arrays {
            let _ = Vec::from_raw_parts(inner_array.data, inner_array.len, inner_array.len);
        }
    }
}
