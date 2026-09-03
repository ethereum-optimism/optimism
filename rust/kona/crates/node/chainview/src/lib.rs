#![doc = include_str!("../README.md")]
#![doc(issue_tracker_base_url = "https://github.com/op-rs/kona/issues/")]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]
#![recursion_limit = "512"]

pub mod client;
pub mod driver;
pub mod facts;
mod generated;
pub mod handles;
pub mod snapshot;

pub use client::{ChainViewClient, ChainViewQuery};
pub use driver::{ChainViewConfig, ChainViewError, ChainViewHandle, spawn};
pub use facts::{Fact, FactError, L1StatusKind, L2Heads, L2SafeFact, L2StatusKind};
pub use generated::PROGRAM_SCHEMA_JSON;
pub use handles::{Handles, build};
pub use snapshot::{ChainViewSnapshot, FinalizedL2, L1Statuses, SafeHeadEntry};

/// The `LATENESS` of `l2_safe_blocks.derived_from_number` in `chainview.sql`: a derived block
/// whose derived-from L1 block is more than this many blocks behind the newest one seen is
/// rejected (and counted) instead of stored. A host must only run the chain view for a chain
/// whose `seq_window_size + channel_timeout`, the largest backward jump of the derived-from
/// block after a reset, stays below it.
pub const LATENESS: u64 = 4096;

#[cfg(test)]
mod tests {
    #[test]
    fn lateness_matches_the_sql_program() {
        let sql = include_str!("chainview.sql");
        assert!(
            sql.contains(&format!("LATENESS {}", super::LATENESS)),
            "chainview.sql and LATENESS disagree"
        );
    }
}
