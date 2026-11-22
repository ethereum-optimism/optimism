//! OP-Reth Transaction pool.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg))]

mod validator;
pub use validator::{OpGasLimitConfig, OpL1BlockInfo, OpTransactionValidator};

pub mod conditional;
mod pool;
pub use pool::OpPool;
pub mod interop_filter;
mod transaction;
pub use transaction::{OpPooledTransaction, OpPooledTx};
mod error;
pub mod interop;
pub mod maintain;
pub use error::{DaFootprintExceedsGasLimit, InvalidCrossTx};
pub mod estimated_da_size;

use reth_transaction_pool::{
    Pool, TransactionValidationTaskExecutor, blobstore::DiskFileBlobStore,
};

/// Optimism transaction pool generic over its validator `V` and ordering `O`.
///
/// This is the single definition of the optimism pool's type structure: the [`OpPool`] wrapper
/// around a [`Pool`] with a [`TransactionValidationTaskExecutor`]-wrapped validator and a
/// [`DiskFileBlobStore`].
///
/// Downstream builders use this directly when they substitute a custom ordering or a validator
/// wrapper (e.g. admission control) without forking pool construction.
pub type OpCustomTransactionPool<V, O> =
    OpPool<Pool<TransactionValidationTaskExecutor<V>, O, DiskFileBlobStore>>;
