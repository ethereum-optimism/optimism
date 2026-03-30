//! OP-Reth Transaction pool.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg))]

mod validator;
pub use validator::{OpL1BlockInfo, OpTransactionValidator};

pub mod conditional;
mod pool;
pub use pool::OpPool;
pub mod supervisor;
mod transaction;
pub use transaction::{OpPooledTransaction, OpPooledTx};
mod error;
pub mod interop;
pub mod maintain;
pub use error::InvalidCrossTx;
pub mod estimated_da_size;

use reth_transaction_pool::{CoinbaseTipOrdering, Pool, TransactionValidationTaskExecutor};

/// Type alias for default optimism transaction pool.
///
/// The [`OpPool`] wrapper delegates most behavior to the inner [`Pool`] handle,
/// and overrides only a subset of the functions.
/// This enables implementing custom behaviors and filtering of the pooled transactions.
pub type OpTransactionPool<Client, S, Evm, T = OpPooledTransaction> = OpPool<
    Pool<
        TransactionValidationTaskExecutor<OpTransactionValidator<Client, T, Evm>>,
        CoinbaseTipOrdering<T>,
        S,
    >,
>;
