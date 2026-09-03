//! The host-integrated state of the chain view.
//!
//! The circuit emits deltas; the driver integrates them and publishes this snapshot through
//! a watch channel. Consumers read scalars from it directly. The safe-head history (one
//! entry per derived-from L1 block) is kept by the driver and queried through
//! [`ChainViewClient::safe_head_at_l1`](crate::ChainViewClient::safe_head_at_l1) rather than
//! copied into every snapshot.

use alloy_eips::BlockNumHash;
use alloy_primitives::Address;
use kona_protocol::BlockInfo;

use crate::facts::L2Heads;

/// The current L1 statuses the chain view has been told about.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct L1Statuses {
    /// The canonical tracker's tip.
    pub head: Option<BlockInfo>,
    /// The L1 `safe` tag.
    pub safe: Option<BlockInfo>,
    /// The L1 `finalized` tag.
    pub finalized: Option<BlockInfo>,
    /// The derivation pipeline's current L1 origin.
    pub current: Option<BlockInfo>,
}

/// The block the `finalized_l2` view says the engine may finalize next.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct FinalizedL2 {
    /// The L2 block.
    pub id: BlockNumHash,
    /// The L1 block it was derived from.
    pub derived_from: BlockNumHash,
}

/// The safe L2 head after derivation consumed an L1 block.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct SafeHeadEntry {
    /// The L1 block.
    pub l1: BlockNumHash,
    /// The safe L2 head derived from it.
    pub l2: BlockNumHash,
}

/// Everything the chain view currently knows, as scalars.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ChainViewSnapshot {
    /// Current L1 statuses.
    pub l1: L1Statuses,
    /// The engine's head labels, once received.
    pub l2: Option<L2Heads>,
    /// The next block to finalize, if the view says there is one.
    pub finalized_l2: Option<FinalizedL2>,
    /// The unsafe-block signer read from the `SystemConfig` contract at the latest L1 head.
    pub signer: Option<Address>,
    /// Number of safe-head history entries held by the driver.
    pub history_len: usize,
    /// `l1_blocks` rows held: the pipeline's origins from finalized L1 upward.
    pub l1_window_len: usize,
    /// Rows the circuit rejected as late (reported through its error view).
    pub lateness_drops: u64,
}
