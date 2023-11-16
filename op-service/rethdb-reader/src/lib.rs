#![doc = include_str!("../README.md")]

use receipts::{read_receipts_inner, ReceiptsResult};
use std::ffi::CString;
use std::os::raw::c_char;

mod receipts;

/// Read the receipts for a blockhash from the RETH database directly.
///
/// # Safety
/// - All possible nil pointer dereferences are checked, and the function will return a
///   failing [ReceiptsResult] if any are found.
#[no_mangle]
pub unsafe extern "C" fn rdb_read_receipts(
    block_hash: *const u8,
    block_hash_len: usize,
    db_path: *const c_char,
) -> *mut ReceiptsResult {
    let res = read_receipts_inner(block_hash, block_hash_len, db_path);
    match res {
        Ok(res) => Box::into_raw(Box::new(res)),
        Err(err) => Box::into_raw(Box::new(ReceiptsResult::fail(err.to_string()))),
    }
}

/// Free a ReceiptsResult that was allocated in Rust and passed to C.
///
/// # Safety
/// - All possible nil pointer dereferences are checked.
#[no_mangle]
pub unsafe extern "C" fn rdb_free_receipts(res: *mut ReceiptsResult) {
    // Convert the raw pointer back to a ReceiptResult and let it go out of scope,
    // which will deallocate the memory.
    if !res.is_null() {
        if !(*res).data.is_null() {
            // Same deal with the data string
            let _ = CString::from_raw((*res).data as *mut c_char);
        }
        if !(*res).error.is_null() {
            // Same deal with the error string
            let _ = CString::from_raw((*res).error);
        }
        let _ = Box::from_raw(res);
    }
}

/// Return the data from a ReceiptsResult.
///
/// # Safety
/// - All possible nil pointer dereferences are checked.
#[no_mangle]
pub unsafe extern "C" fn rdb_get_receipts_data(res: *mut ReceiptsResult) -> *const c_char {
    if res.is_null() {
        return std::ptr::null();
    }
    let res = &*res;
    if res.error.is_null() {
        return std::ptr::null();
    }
    res.data as *const c_char
}

/// Return the length of the data from a ReceiptsResult.
///
/// # Safety
/// - All possible nil pointer dereferences are checked.
#[no_mangle]
pub unsafe extern "C" fn rdb_get_receipts_data_len(res: *mut ReceiptsResult) -> usize {
    res.as_ref().map_or(0, |res| {
        res.error.is_null().then_some(0).unwrap_or(res.data_len)
    })
}

/// Return the error string, if it exists, from a ReceiptsResult.
///
/// # Safety
/// - All possible nil pointer dereferences are checked.
#[no_mangle]
pub unsafe extern "C" fn rdb_get_receipts_error(res: *mut ReceiptsResult) -> *const c_char {
    if res.is_null() {
        return std::ptr::null();
    }
    let res = &*res;
    res.error as *const c_char
}
