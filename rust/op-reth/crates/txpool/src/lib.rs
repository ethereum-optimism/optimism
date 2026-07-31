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
pub mod interop_filter;
mod transaction;
pub use transaction::{OpPooledTransaction, OpPooledTx};
mod error;
pub mod interop;
pub mod maintain;
pub use error::InvalidCrossTx;
pub mod estimated_da_size;

use reth_transaction_pool::{CoinbaseTipOrdering, Pool, TransactionValidationTaskExecutor};

/// Optimism transaction pool generic over its validator `V` and ordering `O`.
///
/// This is the single definition of the optimism pool's type structure: the [`OpPool`] wrapper
/// around a [`Pool`] with a [`TransactionValidationTaskExecutor`]-wrapped validator. Both the
/// standard pool ([`OpTransactionPool`]) and any downstream-customized pool are expressed in terms
/// of this alias, so the pool's shape is defined in exactly one place and the two cannot diverge.
///
/// Downstream builders use this directly when they substitute a custom ordering or a validator
/// wrapper (e.g. admission control) without forking pool construction.
pub type OpCustomTransactionPool<S, V, O> =
    OpPool<Pool<TransactionValidationTaskExecutor<V>, O, S>>;

/// Type alias for the standard optimism transaction pool.
///
/// This is [`OpCustomTransactionPool`] specialized to the standard validator
/// ([`OpTransactionValidator`]) and ordering ([`CoinbaseTipOrdering`]) — the configuration used by
/// the default node. Because it is defined in terms of [`OpCustomTransactionPool`], it stays in
/// lockstep with the general alias automatically.
pub type OpTransactionPool<Client, S, Evm, T = OpPooledTransaction> =
    OpCustomTransactionPool<S, OpTransactionValidator<Client, T, Evm>, CoinbaseTipOrdering<T>>;
