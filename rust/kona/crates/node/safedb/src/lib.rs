#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

mod disabled;
mod error;
mod traits;

#[cfg(feature = "rocksdb")]
mod encoding;
#[cfg(feature = "rocksdb")]
mod safe_db;

#[cfg(test)]
mod tests;

pub use disabled::DisabledDatabase;
pub use error::SafeDbError;
pub use traits::{SafeDb, SafeHeadRecord, SharedSafeDb};

#[cfg(feature = "rocksdb")]
pub use safe_db::SafeDatabase;
