use anyhow::{anyhow, Result};
use reth_blockchain_tree::noop::NoopBlockchainTree;
use reth_db::{mdbx::DatabaseArguments, open_db_read_only};
use reth_primitives::MAINNET;
use reth_provider::{providers::BlockchainProvider, ProviderFactory};
use std::{
    ffi::{c_char, c_void},
    path::Path,
};

/// A [OpenDBResult] is a wrapper of DB instance [BlockchainProvider]
/// as well as an error status that is compatible with FFI.
///
/// # Safety
/// - When the `error` field is false, the `data` pointer is guaranteed to be valid.
/// - When the `error` field is true, the `data` pointer is guaranteed to be null.
#[repr(C)]
pub struct OpenDBResult {
    pub(crate) data: *const c_void,
    pub(crate) error: bool,
}

impl OpenDBResult {
    /// Constructs a successful [OpenDBResult] from a DB instance.
    pub fn success(data: *const c_void) -> Self {
        Self { data, error: false }
    }

    /// Constructs a failing [OpenDBResult] with a null pointer to the data.
    pub fn fail() -> Self {
        Self { data: std::ptr::null_mut(), error: true }
    }
}

/// Open and return a DB instance.
///
/// # Safety
/// - All possible nil pointer dereferences are checked, and the function will return a failing
///   [OpenDBResult] if any are found.
#[inline(always)]
pub(crate) unsafe fn open_db_read_only_inner(db_path: *const c_char) -> Result<OpenDBResult> {
    // Convert the *const c_char to a Rust &str
    let db_path_str = {
        if db_path.is_null() {
            anyhow::bail!("db path pointer is null");
        }
        std::ffi::CStr::from_ptr(db_path)
    }
    .to_str()?;

    let db = open_db_read_only(Path::new(db_path_str), DatabaseArguments::default())
        .map_err(|e| anyhow!(e))?;
    let factory = ProviderFactory::new(db, MAINNET.clone());

    // Create a read-only BlockChainProvider
    let provider = Box::new(BlockchainProvider::new(factory, NoopBlockchainTree::default())?);
    let res = OpenDBResult::success(Box::into_raw(provider) as *const c_void);
    Ok(res)
}
