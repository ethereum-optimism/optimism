//! The safe-head database interface.

use crate::error::SafeDbError;
use alloy_eips::BlockNumHash;
use kona_protocol::L2BlockInfo;
use std::{fmt::Debug, sync::Arc};

/// A recorded pairing of an L1 block and the L2 safe head derived as of that block.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct SafeHeadRecord {
    /// The L1 block at which the safe head became safe.
    pub l1: BlockNumHash,
    /// The L2 safe head derived as of [`SafeHeadRecord::l1`].
    pub safe_head: BlockNumHash,
}

/// A [`SafeDb`] shared between the actor that records into it and the readers that query it.
///
/// Spelled as an alias because every holder wants the same shape: the recording writer and each
/// reader hold the *same* database, and swapping the on-disk backend for
/// [`DisabledDatabase`](crate::DisabledDatabase) has to be a construction-site decision rather
/// than a type change rippling through the holders.
pub type SharedSafeDb = Arc<dyn SafeDb + Send + Sync>;

/// The safe-head database interface.
///
/// Implementations record, per L1 block, the L2 safe head that derivation reached as of that
/// block, and answer queries in both directions (L1 → safe head and safe head → L1).
///
/// Methods are synchronous: the backing store is queried in-process, and a future actor is
/// expected to own a single instance and serialize access to it.
///
/// [`Debug`] is a supertrait so that a [`SharedSafeDb`] can sit in a `#[derive(Debug)]` holder;
/// the actors and node configurations that carry one are all debug-printable.
pub trait SafeDb: Debug {
    /// Reports whether this database actively records and serves derivation data.
    fn enabled(&self) -> bool;

    /// Records that `safe_head` became safe as of `l1_head`.
    ///
    /// `l1_head` is the first L1 block containing all data required to derive `safe_head`.
    ///
    /// Callers must deliver updates in ascending L1 order and must not miss any update. Only
    /// observed advances are stored, so a missed update leaves a gap that queries cannot detect:
    /// [`SafeDb::safe_head_at_l1`] then answers with the last safe head before the gap. Derivation
    /// must therefore resume from the L1 block of [`SafeDb::last_entry`].
    fn safe_head_updated(
        &self,
        safe_head: L2BlockInfo,
        l1_head: BlockNumHash,
    ) -> Result<(), SafeDbError>;

    /// Rewinds the recorded history so that `safe_head` is once again the tip.
    ///
    /// Locates the boundary — the first L1 block whose recorded safe head reached at or beyond
    /// `safe_head` — then removes it and every entry after it. If an earlier entry remains,
    /// `safe_head` is re-recorded at the boundary L1 block; if the reset predates all records,
    /// nothing is re-recorded, since the L1 block that made `safe_head` safe is then unknown.
    fn safe_head_reset(&self, safe_head: L2BlockInfo) -> Result<(), SafeDbError>;

    /// Returns the safe head recorded as of the highest L1 block at or below `l1_block_num`.
    fn safe_head_at_l1(&self, l1_block_num: u64) -> Result<SafeHeadRecord, SafeDbError>;

    /// Returns the lowest recorded entry.
    fn first_entry(&self) -> Result<SafeHeadRecord, SafeDbError>;

    /// Returns the highest recorded entry (the database tip).
    fn last_entry(&self) -> Result<SafeHeadRecord, SafeDbError>;

    /// Returns the earliest L1 block at which the recorded safe head reached at least
    /// `target_l2_num`, along with the safe head recorded at that L1 block.
    fn l1_at_safe_head(&self, target_l2_num: u64) -> Result<SafeHeadRecord, SafeDbError>;

    /// Closes the database, releasing the underlying store.
    fn close(&self) -> Result<(), SafeDbError>;
}
