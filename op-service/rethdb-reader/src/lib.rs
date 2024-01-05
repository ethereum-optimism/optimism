#![doc = include_str!("../README.md")]

use receipts::{read_receipts_inner, ReceiptsResult};
use std::os::raw::c_char;

mod receipts;

/// Read the receipts for a blockhash from the RETH database directly.
///
/// # Safety
/// - All possible nil pointer dereferences are checked, and the function will return a failing
///   [ReceiptsResult] if any are found.
#[no_mangle]
pub unsafe extern "C" fn rdb_read_receipts(
    block_hash: *const u8,
    block_hash_len: usize,
    db_path: *const c_char,
) -> ReceiptsResult {
    read_receipts_inner(block_hash, block_hash_len, db_path).unwrap_or(ReceiptsResult::fail())
}

/// Free a string that was allocated in Rust and passed to C.
///
/// # Safety
/// - All possible nil pointer dereferences are checked.
#[no_mangle]
pub unsafe extern "C" fn rdb_free_string(string: *mut c_char) {
    // Convert the raw pointer back to a CString and let it go out of scope,
    // which will deallocate the memory.
    if !string.is_null() {
        let _ = std::ffi::CString::from_raw(string);
    }
}
