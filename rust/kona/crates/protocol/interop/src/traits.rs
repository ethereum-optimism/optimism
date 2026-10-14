//! Traits for the `kona-interop` crate.

use alloc::{boxed::Box, vec::Vec};
use alloy_consensus::Header;
use alloy_primitives::{Address, B256, Log};
use async_trait::async_trait;
use core::{error::Error, fmt::Debug};

/// Describes the interface of the interop data provider. This provider is multiplexed over several
/// chains, with each method consuming a chain ID to determine the target chain.
#[async_trait]
pub trait InteropProvider {
    /// The error type for the provider.
    type Error: Error;

    /// Fetch a [Header] by its number.
    async fn header_by_number(&self, chain_id: u64, number: u64) -> Result<Header, Self::Error>;

    /// The logs of the canonical block `header` on `chain_id`, which
    /// [`MessageGraph`](crate::MessageGraph) checks initiating messages against.
    async fn initiating_logs(
        &self,
        chain_id: u64,
        header: &Header,
    ) -> Result<InitiatingLogs, Self::Error>;
}

/// The logs of an initiating block, as an [`InteropProvider`] serves them.
#[derive(Debug)]
pub enum InitiatingLogs {
    /// Every log of the block's receipts, in block order.
    Receipts(Vec<Log>),
    /// A commitment to the block's logs.
    Committed(Box<dyn LogCommitment>),
}

/// A commitment to the log list of one canonical block: its receipts' logs in block order.
pub trait LogCommitment: Debug + Send + Sync {
    /// The number of logs in the block.
    fn log_count(&self) -> u64;

    /// Whether log `index` has `origin` and `payload_hash`, for `index` below
    /// [`Self::log_count`]. `false` must not stand for a log that cannot be checked: a proof
    /// program panics instead.
    fn log_matches(&self, index: u64, origin: &Address, payload_hash: &B256) -> bool;
}
