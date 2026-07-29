//! Contains the tip cursor for the derivation driver.
//!
//! This module provides the [`TipCursor`] which encapsulates the L2 safe head state
//! including block information, header, and output root for a specific derivation tip.

use alloy_consensus::{Header, Sealed};
use alloy_primitives::B256;
use kona_protocol::L2BlockInfo;

/// A cursor that encapsulates the L2 safe head state at a specific derivation tip.
///
/// The [`TipCursor`] represents a snapshot of the L2 chain state at a particular point
/// in the derivation process. It contains all the essential information needed to
/// represent an L2 safe head including the block metadata, sealed header, and output root.
///
/// # Components
/// - **Block Info**: L2 block metadata and consensus information
/// - **Sealed Header**: Complete block header with computed hash
/// - **Output Root**: State commitment for fraud proof verification
///
/// # Usage
/// The tip cursor is used by the pipeline cursor to cache L2 safe head states
/// corresponding to different L1 origins, enabling efficient reorg recovery and
/// state management during derivation.
///
/// # Immutability
/// Once created, a tip cursor represents an immutable snapshot of the L2 state.
/// New tip cursors are created as derivation progresses rather than mutating
/// existing ones.
#[derive(Debug, Clone)]
pub struct TipCursor {
    /// The L2 block information for the safe head.
    ///
    /// Contains all the L2-specific metadata including block number, timestamp,
    /// L1 origin information, and other consensus-critical data derived from
    /// the L1 chain.
    pub l2_safe_head: L2BlockInfo,
    /// The sealed header of the L2 safe head block.
    ///
    /// The sealed header includes the complete block header with the computed
    /// block hash, providing access to parent hash, state root, transaction root,
    /// and other header fields needed for block validation.
    pub l2_safe_head_header: Sealed<Header>,
    /// The output root computed for the L2 safe head state.
    ///
    /// The output root is a cryptographic commitment to the L2 state after
    /// executing this block. It's used for fraud proof verification and
    /// enables efficient state challenges without requiring full state data.
    pub l2_safe_head_output_root: B256,
}

impl TipCursor {
    /// Creates a new tip cursor with the specified L2 safe head components.
    ///
    /// # Arguments
    /// * `l2_safe_head` - The L2 block information for the safe head
    /// * `l2_safe_head_header` - The sealed header of the L2 safe head block
    /// * `l2_safe_head_output_root` - The computed output root for this state
    ///
    /// # Returns
    /// A new [`TipCursor`] encapsulating the provided L2 safe head state
    ///
    /// # Usage
    /// Called when the driver completes derivation of a new L2 block to create
    /// a snapshot of the resulting safe head state for caching and reorg recovery.
    pub const fn new(
        l2_safe_head: L2BlockInfo,
        l2_safe_head_header: Sealed<Header>,
        l2_safe_head_output_root: B256,
    ) -> Self {
        Self { l2_safe_head, l2_safe_head_header, l2_safe_head_output_root }
    }

    /// Returns a reference to the L2 safe head block information.
    ///
    /// Provides access to the L2 block metadata including block number, timestamp,
    /// L1 origin information, and other consensus-critical data.
    pub const fn l2_safe_head(&self) -> &L2BlockInfo {
        &self.l2_safe_head
    }

    /// Returns a reference to the sealed header of the L2 safe head.
    ///
    /// The sealed header contains the complete block header with computed hash,
    /// providing access to parent hash, state root, transaction root, and other
    /// header fields needed for block validation and chain verification.
    pub const fn l2_safe_head_header(&self) -> &Sealed<Header> {
        &self.l2_safe_head_header
    }

    /// Returns a reference to the output root of the L2 safe head.
    ///
    /// The output root is a cryptographic commitment to the L2 state after
    /// executing this block. It enables fraud proof verification and efficient
    /// state challenges without requiring access to the full state data.
    pub const fn l2_safe_head_output_root(&self) -> &B256 {
        &self.l2_safe_head_output_root
    }
}
