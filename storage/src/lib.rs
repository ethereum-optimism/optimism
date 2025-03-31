//! Standalone crate for Optimism-Storage Reth.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]
#![cfg_attr(not(feature = "std"), no_std)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]

extern crate alloc;

mod chain;
pub use chain::OptStorage;

#[cfg(test)]
mod tests {
    use reth_codecs::{test_utils::UnusedBits, validate_bitflag_backwards_compat};
    use reth_db_api::models::{
        CompactClientVersion, CompactU256, CompactU64, StoredBlockBodyIndices,
        StoredBlockWithdrawals,
    };
    use reth_primitives_traits::Account;
    use reth_prune_types::{PruneCheckpoint, PruneMode, PruneSegment};
    use reth_stages_types::{
        AccountHashingCheckpoint, CheckpointBlockRange, EntitiesCheckpoint, ExecutionCheckpoint,
        HeadersCheckpoint, IndexHistoryCheckpoint, StageCheckpoint, StageUnitCheckpoint,
        StorageHashingCheckpoint,
    };

    #[test]
    fn test_ensure_backwards_compatibility() {
        assert_eq!(Account::bitflag_encoded_bytes(), 2);
        assert_eq!(AccountHashingCheckpoint::bitflag_encoded_bytes(), 1);
        assert_eq!(CheckpointBlockRange::bitflag_encoded_bytes(), 1);
        assert_eq!(CompactClientVersion::bitflag_encoded_bytes(), 0);
        assert_eq!(CompactU256::bitflag_encoded_bytes(), 1);
        assert_eq!(CompactU64::bitflag_encoded_bytes(), 1);
        assert_eq!(EntitiesCheckpoint::bitflag_encoded_bytes(), 1);
        assert_eq!(ExecutionCheckpoint::bitflag_encoded_bytes(), 0);
        assert_eq!(HeadersCheckpoint::bitflag_encoded_bytes(), 0);
        assert_eq!(IndexHistoryCheckpoint::bitflag_encoded_bytes(), 0);
        assert_eq!(PruneCheckpoint::bitflag_encoded_bytes(), 1);
        assert_eq!(PruneMode::bitflag_encoded_bytes(), 1);
        assert_eq!(PruneSegment::bitflag_encoded_bytes(), 1);
        assert_eq!(StageCheckpoint::bitflag_encoded_bytes(), 1);
        assert_eq!(StageUnitCheckpoint::bitflag_encoded_bytes(), 1);
        assert_eq!(StoredBlockBodyIndices::bitflag_encoded_bytes(), 1);
        assert_eq!(StoredBlockWithdrawals::bitflag_encoded_bytes(), 0);
        assert_eq!(StorageHashingCheckpoint::bitflag_encoded_bytes(), 1);

        // In case of failure, refer to the documentation of the
        // [`validate_bitflag_backwards_compat`] macro for detailed instructions on handling
        // it.
        validate_bitflag_backwards_compat!(Account, UnusedBits::NotZero);
        validate_bitflag_backwards_compat!(AccountHashingCheckpoint, UnusedBits::NotZero);
        validate_bitflag_backwards_compat!(CheckpointBlockRange, UnusedBits::Zero);
        validate_bitflag_backwards_compat!(CompactClientVersion, UnusedBits::Zero);
        validate_bitflag_backwards_compat!(CompactU256, UnusedBits::NotZero);
        validate_bitflag_backwards_compat!(CompactU64, UnusedBits::NotZero);
        validate_bitflag_backwards_compat!(EntitiesCheckpoint, UnusedBits::Zero);
        validate_bitflag_backwards_compat!(ExecutionCheckpoint, UnusedBits::Zero);
        validate_bitflag_backwards_compat!(HeadersCheckpoint, UnusedBits::Zero);
        validate_bitflag_backwards_compat!(IndexHistoryCheckpoint, UnusedBits::Zero);
        validate_bitflag_backwards_compat!(PruneCheckpoint, UnusedBits::NotZero);
        validate_bitflag_backwards_compat!(PruneMode, UnusedBits::Zero);
        validate_bitflag_backwards_compat!(PruneSegment, UnusedBits::Zero);
        validate_bitflag_backwards_compat!(StageCheckpoint, UnusedBits::NotZero);
        validate_bitflag_backwards_compat!(StageUnitCheckpoint, UnusedBits::Zero);
        validate_bitflag_backwards_compat!(StoredBlockBodyIndices, UnusedBits::Zero);
        validate_bitflag_backwards_compat!(StoredBlockWithdrawals, UnusedBits::Zero);
        validate_bitflag_backwards_compat!(StorageHashingCheckpoint, UnusedBits::NotZero);
    }
}
