//! Provider for tracking block safety head reference
use crate::{StorageError, models::SafetyHeadRefs};
use alloy_primitives::ChainId;
use derive_more::Constructor;
use kona_protocol::BlockInfo;
use op_alloy_consensus::interop::SafetyLevel;
use reth_db_api::transaction::{DbTx, DbTxMut};
use tracing::{error, warn};

/// A Safety Head Reference storage that wraps transactional reference.
#[derive(Debug, Constructor)]
pub(crate) struct SafetyHeadRefProvider<'tx, TX> {
    tx: &'tx TX,
    chain_id: ChainId,
}

impl<TX> SafetyHeadRefProvider<'_, TX>
where
    TX: DbTx,
{
    pub(crate) fn get_safety_head_ref(
        &self,
        safety_level: SafetyLevel,
    ) -> Result<BlockInfo, StorageError> {
        let head_ref_key = safety_level.into();
        let result = self.tx.get::<SafetyHeadRefs>(head_ref_key).inspect_err(|err| {
            error!(
                target: "supervisor::storage",
                chain_id = %self.chain_id,
                %safety_level,
                %err,
                "Failed to seek head reference"
            );
        })?;
        let block_ref = result.ok_or_else(|| StorageError::FutureData)?;
        Ok(block_ref.into())
    }
}

impl<Tx> SafetyHeadRefProvider<'_, Tx>
where
    Tx: DbTxMut + DbTx,
{
    /// Updates the safety head reference with the provided block info.
    /// If the block info's number is less than the current head reference's number,
    /// it will not update the head reference and will log a warning.
    pub(crate) fn update_safety_head_ref(
        &self,
        safety_level: SafetyLevel,
        incoming_head_ref: &BlockInfo,
    ) -> Result<(), StorageError> {
        // Ensure the block_info.number is greater than the stored head reference
        // If the head reference is not set, this check will be skipped.
        if let Ok(current_head_ref) = self.get_safety_head_ref(safety_level) {
            if current_head_ref.number > incoming_head_ref.number {
                warn!(
                    target: "supervisor::storage",
                    chain_id = %self.chain_id,
                    %current_head_ref,
                    %incoming_head_ref,
                    %safety_level,
                    "Attempting to update head reference with a block that has a lower number than the current head reference",
                );
                return Ok(());
            }
        }

        self.tx
            .put::<SafetyHeadRefs>(safety_level.into(), (*incoming_head_ref).into())
            .inspect_err(|err| {
                error!(
                    target: "supervisor::storage",
                    chain_id = %self.chain_id,
                    %incoming_head_ref,
                    %safety_level,
                    %err,
                    "Failed to store head reference"
                )
            })?;
        Ok(())
    }

    /// Forcefully resets the head reference only if the current stored head is ahead of the
    /// incoming one.
    ///
    /// This is intended for internal use during rewinds, where the safety head needs to be directly
    /// set to a previous block regardless of the current head state.
    pub(crate) fn reset_safety_head_ref_if_ahead(
        &self,
        safety_level: SafetyLevel,
        incoming_head_ref: &BlockInfo,
    ) -> Result<(), StorageError> {
        // Skip if the current head is behind or missing.
        match self.get_safety_head_ref(safety_level) {
            Ok(current_head_ref) => {
                if current_head_ref.number < incoming_head_ref.number {
                    return Ok(());
                }
            }
            Err(StorageError::FutureData) => {
                return Ok(());
            }
            Err(err) => return Err(err),
        }

        self.tx
            .put::<SafetyHeadRefs>(safety_level.into(), (*incoming_head_ref).into())
            .inspect_err(|err| {
                error!(
                    target: "supervisor::storage",
                    chain_id = %self.chain_id,
                    %incoming_head_ref,
                    %safety_level,
                    %err,
                    "Failed to reset head reference"
                )
            })?;
        Ok(())
    }

    /// Removes the safety head reference for the specified safety level.
    pub(crate) fn remove_safety_head_ref(
        &self,
        safety_level: SafetyLevel,
    ) -> Result<(), StorageError> {
        self.tx.delete::<SafetyHeadRefs>(safety_level.into(), None).inspect_err(|err| {
            error!(
                target: "supervisor::storage",
                chain_id = %self.chain_id,
                %safety_level,
                %err,
                "Failed to remove head reference"
            )
        })?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::Tables;
    use alloy_primitives::B256;
    use reth_db::{
        DatabaseEnv,
        mdbx::{DatabaseArguments, init_db_for},
    };
    use reth_db_api::Database;
    use tempfile::TempDir;

    static CHAIN_ID: ChainId = 1;

    fn setup_db() -> DatabaseEnv {
        let temp_dir = TempDir::new().expect("Could not create temp dir");
        init_db_for::<_, Tables>(temp_dir.path(), DatabaseArguments::default())
            .expect("Failed to init database")
    }

    #[test]
    fn test_safety_head_ref_retrieval() {
        let db = setup_db();

        // Create write transaction first
        let write_tx = db.tx_mut().expect("Failed to create write transaction");
        let write_provider = SafetyHeadRefProvider::new(&write_tx, CHAIN_ID);

        // Initially, there should be no head ref
        let result = write_provider.get_safety_head_ref(SafetyLevel::CrossSafe);
        assert!(result.is_err());

        // Update head ref
        let block_info = BlockInfo::default();
        write_provider
            .update_safety_head_ref(SafetyLevel::CrossSafe, &block_info)
            .expect("Failed to update head ref");

        // Commit the write transaction
        write_tx.commit().expect("Failed to commit the write transaction");

        // Create a new read transaction to verify
        let tx = db.tx().expect("Failed to create transaction");
        let provider = SafetyHeadRefProvider::new(&tx, CHAIN_ID);
        let result =
            provider.get_safety_head_ref(SafetyLevel::CrossSafe).expect("Failed to get head ref");
        assert_eq!(result, block_info);
    }

    #[test]
    fn test_safety_head_ref_update() {
        let db = setup_db();
        let write_tx = db.tx_mut().expect("Failed to create write transaction");
        let write_provider = SafetyHeadRefProvider::new(&write_tx, CHAIN_ID);

        // Create initial block info
        let initial_block_info = BlockInfo {
            hash: Default::default(),
            number: 1,
            parent_hash: Default::default(),
            timestamp: 100,
        };
        write_provider
            .update_safety_head_ref(SafetyLevel::CrossSafe, &initial_block_info)
            .expect("Failed to update head ref");

        // Create updated block info
        let mut updated_block_info = BlockInfo {
            hash: Default::default(),
            number: 1,
            parent_hash: Default::default(),
            timestamp: 200,
        };
        updated_block_info.number = 100;
        write_provider
            .update_safety_head_ref(SafetyLevel::CrossSafe, &updated_block_info)
            .expect("Failed to update head ref");

        // Commit the write transaction
        write_tx.commit().expect("Failed to commit the write transaction");

        // Verify the updated value
        let tx = db.tx().expect("Failed to create transaction");
        let provider = SafetyHeadRefProvider::new(&tx, CHAIN_ID);
        let result =
            provider.get_safety_head_ref(SafetyLevel::CrossSafe).expect("Failed to get head ref");
        assert_eq!(result, updated_block_info);
    }

    #[test]
    fn test_reset_safety_head_ref_if_ahead() {
        let db = setup_db();
        let tx = db.tx_mut().expect("Failed to start write tx");
        let provider = SafetyHeadRefProvider::new(&tx, CHAIN_ID);

        // Set initial head at 100
        let head_100 = BlockInfo {
            number: 100,
            hash: B256::from([1u8; 32]),
            parent_hash: B256::ZERO,
            timestamp: 1234,
        };
        provider.update_safety_head_ref(SafetyLevel::CrossSafe, &head_100).expect("update failed");

        // Try to reset to 101 (should NOT update — current is behind)
        let head_101 = BlockInfo { number: 101, ..head_100 };
        provider
            .reset_safety_head_ref_if_ahead(SafetyLevel::CrossSafe, &head_101)
            .expect("reset failed");

        // Should still be 100
        let current = provider.get_safety_head_ref(SafetyLevel::CrossSafe).expect("get failed");
        assert_eq!(current.number, 100);

        // Now try to reset to 90 (should update — current is ahead)
        let head_90 = BlockInfo { number: 90, ..head_100 };
        provider
            .reset_safety_head_ref_if_ahead(SafetyLevel::CrossSafe, &head_90)
            .expect("reset failed");

        // Should now be 90
        let current = provider.get_safety_head_ref(SafetyLevel::CrossSafe).expect("get failed");
        assert_eq!(current.number, 90);

        tx.commit().expect("commit failed");
    }

    #[test]
    fn test_reset_safety_head_ref_should_ignore_future_data() {
        let db = setup_db();
        let tx = db.tx_mut().expect("Failed to start write tx");
        let provider = SafetyHeadRefProvider::new(&tx, CHAIN_ID);

        // Set initial head at 100
        let head_100 = BlockInfo {
            number: 100,
            hash: B256::from([1u8; 32]),
            parent_hash: B256::ZERO,
            timestamp: 1234,
        };

        provider
            .reset_safety_head_ref_if_ahead(SafetyLevel::CrossSafe, &head_100)
            .expect("reset should succeed");

        // check head is not updated and still returns FutureData Err
        let result = provider.get_safety_head_ref(SafetyLevel::CrossSafe);
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), StorageError::FutureData));

        tx.commit().expect("commit failed");
    }

    #[test]
    fn test_remove_safety_head_ref_removes_existing() {
        let db = setup_db();
        let tx = db.tx_mut().expect("Failed to start write tx");
        let provider = SafetyHeadRefProvider::new(&tx, CHAIN_ID);

        // Set a head ref
        let block_info = BlockInfo {
            hash: Default::default(),
            number: 42,
            parent_hash: Default::default(),
            timestamp: 1234,
        };
        provider
            .update_safety_head_ref(SafetyLevel::CrossSafe, &block_info)
            .expect("update failed");

        // Remove it
        provider.remove_safety_head_ref(SafetyLevel::CrossSafe).expect("remove failed");

        // Should now return FutureData error
        let result = provider.get_safety_head_ref(SafetyLevel::CrossSafe);
        assert!(matches!(result, Err(StorageError::FutureData)));
    }

    #[test]
    fn test_remove_safety_head_ref_no_existing() {
        let db = setup_db();
        let tx = db.tx_mut().expect("Failed to start write tx");
        let provider = SafetyHeadRefProvider::new(&tx, CHAIN_ID);

        // Remove when nothing exists
        let result = provider.remove_safety_head_ref(SafetyLevel::CrossSafe);
        assert!(result.is_ok());

        // Still returns FutureData error
        let result = provider.get_safety_head_ref(SafetyLevel::CrossSafe);
        assert!(matches!(result, Err(StorageError::FutureData)));
    }
}
