//! A no-op [`SafeDbV2`] implementation for hosts that do not record derivation.

use crate::{
    error::SafeDbError,
    traits::{SafeDbV2, SafeHeadRecord},
};
use alloy_eips::BlockNumHash;
use kona_protocol::L2BlockInfo;

/// A [`SafeDbV2`] that records nothing and reports itself disabled.
///
/// Writes succeed as no-ops so callers need not branch on whether recording is enabled, while
/// reads return [`SafeDbError::NotEnabled`].
#[derive(Debug, Default, Clone, Copy)]
pub struct DisabledDatabaseV2;

impl SafeDbV2 for DisabledDatabaseV2 {
    fn enabled(&self) -> bool {
        false
    }

    fn safe_head_updated(
        &self,
        _safe_head: L2BlockInfo,
        _l1_head: BlockNumHash,
    ) -> Result<(), SafeDbError> {
        Ok(())
    }

    fn safe_head_reset(&self, _safe_head: L2BlockInfo) -> Result<(), SafeDbError> {
        Ok(())
    }

    fn safe_head_at_l1(&self, _l1_block_num: u64) -> Result<SafeHeadRecord, SafeDbError> {
        Err(SafeDbError::NotEnabled)
    }

    fn first_entry(&self) -> Result<SafeHeadRecord, SafeDbError> {
        Err(SafeDbError::NotEnabled)
    }

    fn last_entry(&self) -> Result<SafeHeadRecord, SafeDbError> {
        Err(SafeDbError::NotEnabled)
    }

    fn l1_at_safe_head(&self, _target_l2_num: u64) -> Result<SafeHeadRecord, SafeDbError> {
        Err(SafeDbError::NotEnabled)
    }

    fn close(&self) -> Result<(), SafeDbError> {
        Ok(())
    }
}
