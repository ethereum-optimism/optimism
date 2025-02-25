//! Utilities for accessing Optimism predeploy state

use alloy_primitives::{address, Address, B256};
use reth_storage_api::{errors::ProviderResult, StorageRootProvider};
use reth_trie_common::HashedStorage;
use revm::database::BundleState;

/// The L2 contract `L2ToL1MessagePasser`, stores commitments to withdrawal transactions.
pub const ADDRESS_L2_TO_L1_MESSAGE_PASSER: Address =
    address!("4200000000000000000000000000000000000016");

/// Computes the storage root of predeploy `L2ToL1MessagePasser.sol`.
///
/// Uses state updates from block execution. See also [`withdrawals_root_prehashed`].
pub fn withdrawals_root<DB: StorageRootProvider>(
    state_updates: &BundleState,
    state: DB,
) -> ProviderResult<B256> {
    // if l2 withdrawals transactions were executed there will be storage updates for
    // `L2ToL1MessagePasser.sol` predeploy
    withdrawals_root_prehashed(
        state_updates
            .state()
            .get(&ADDRESS_L2_TO_L1_MESSAGE_PASSER)
            .map(|acc| {
                HashedStorage::from_plain_storage(
                    acc.status,
                    acc.storage.iter().map(|(slot, value)| (slot, &value.present_value)),
                )
            })
            .unwrap_or_default(),
        state,
    )
}

/// Computes the storage root of predeploy `L2ToL1MessagePasser.sol`.
///
/// Uses pre-hashed storage updates of `L2ToL1MessagePasser.sol` predeploy, resulting from
/// execution of L2 withdrawals transactions. If none, takes empty [`HashedStorage::default`].
pub fn withdrawals_root_prehashed<DB: StorageRootProvider>(
    hashed_storage_updates: HashedStorage,
    state: DB,
) -> ProviderResult<B256> {
    state.storage_root(ADDRESS_L2_TO_L1_MESSAGE_PASSER, hashed_storage_updates)
}
