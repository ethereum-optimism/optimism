#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(any(test, feature = "test-utils")), no_std)]

extern crate alloc;

#[macro_use]
extern crate tracing;

mod db;
pub use db::{NoopTrieDBProvider, TrieDB, TrieDBProvider};

mod builder;
pub use builder::{BlockBuildingOutcome, StatelessL2Builder, compute_receipts_root};

mod errors;
pub use errors::{
    Eip1559ValidationError, ExecutorError, ExecutorResult, TrieDBError, TrieDBResult,
};

pub(crate) mod util;

#[cfg(any(test, feature = "test-utils"))]
pub mod test_utils;
