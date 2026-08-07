//! [`OpProofsProviderRO`] implementation for [`MdbxProofsProvider`].

use super::{
    MdbxProofsProvider,
    cursor::{MdbxAccountCursor, MdbxAccountTrieCursor, MdbxStorageCursor, MdbxStorageTrieCursor},
};
use crate::{
    BlockStateDiff, OpProofsStorageResult,
    api::{OpProofsProviderRO, ProofWindowRange},
    db::{
        ProofWindowKey,
        models::{
            V2AccountTrieChangeSets, V2AccountsTrie, V2AccountsTrieHistory,
            V2HashedAccountChangeSets, V2HashedAccounts, V2HashedAccountsHistory,
            V2HashedStorageChangeSets, V2HashedStorages, V2HashedStoragesHistory,
            V2StorageTrieChangeSets, V2StoragesTrie, V2StoragesTrieHistory,
        },
    },
};
use alloy_eips::NumHash;
use alloy_primitives::B256;
use reth_db::transaction::DbTx;
use std::fmt::Debug;

impl<TX: DbTx + Send + Sync + Debug + 'static> OpProofsProviderRO for MdbxProofsProvider<TX> {
    type StorageTrieCursor<'tx>
        = MdbxStorageTrieCursor<
        TX::DupCursor<V2StoragesTrie>,
        TX::Cursor<V2StoragesTrieHistory>,
        TX::DupCursor<V2StorageTrieChangeSets>,
    >
    where
        Self: 'tx,
        TX: 'tx;

    type AccountTrieCursor<'tx>
        = MdbxAccountTrieCursor<
        TX::Cursor<V2AccountsTrie>,
        TX::Cursor<V2AccountsTrieHistory>,
        TX::DupCursor<V2AccountTrieChangeSets>,
    >
    where
        Self: 'tx,
        TX: 'tx;

    type StorageCursor<'tx>
        = MdbxStorageCursor<
        TX::DupCursor<V2HashedStorages>,
        TX::Cursor<V2HashedStoragesHistory>,
        TX::DupCursor<V2HashedStorageChangeSets>,
    >
    where
        Self: 'tx,
        TX: 'tx;

    type AccountHashedCursor<'tx>
        = MdbxAccountCursor<
        TX::Cursor<V2HashedAccounts>,
        TX::Cursor<V2HashedAccountsHistory>,
        TX::DupCursor<V2HashedAccountChangeSets>,
    >
    where
        Self: 'tx,
        TX: 'tx;

    fn get_earliest_block(&self) -> OpProofsStorageResult<NumHash> {
        self.get_block_number_hash_inner(ProofWindowKey::EarliestBlock)
    }

    fn get_latest_block(&self) -> OpProofsStorageResult<NumHash> {
        self.get_block_number_hash_inner(ProofWindowKey::LatestBlock)
    }

    fn get_proof_window(&self) -> OpProofsStorageResult<ProofWindowRange> {
        self.get_proof_window_inner()
    }

    fn storage_trie_cursor<'tx>(
        &self,
        hashed_address: B256,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::StorageTrieCursor<'tx>> {
        let is_latest = self.is_latest_block(max_block_number)?;
        Ok(MdbxStorageTrieCursor::new(
            self.tx.cursor_dup_read::<V2StoragesTrie>()?,
            self.tx.cursor_read::<V2StoragesTrieHistory>()?,
            self.tx.cursor_read::<V2StoragesTrieHistory>()?,
            self.tx.cursor_dup_read::<V2StorageTrieChangeSets>()?,
            hashed_address,
            max_block_number,
            is_latest,
        ))
    }

    fn account_trie_cursor<'tx>(
        &self,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::AccountTrieCursor<'tx>> {
        let is_latest = self.is_latest_block(max_block_number)?;
        Ok(MdbxAccountTrieCursor::new(
            self.tx.cursor_read::<V2AccountsTrie>()?,
            self.tx.cursor_read::<V2AccountsTrieHistory>()?,
            self.tx.cursor_read::<V2AccountsTrieHistory>()?,
            self.tx.cursor_dup_read::<V2AccountTrieChangeSets>()?,
            max_block_number,
            is_latest,
        ))
    }

    fn storage_hashed_cursor<'tx>(
        &self,
        hashed_address: B256,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::StorageCursor<'tx>> {
        let is_latest = self.is_latest_block(max_block_number)?;
        Ok(MdbxStorageCursor::new(
            self.tx.cursor_dup_read::<V2HashedStorages>()?,
            self.tx.cursor_read::<V2HashedStoragesHistory>()?,
            self.tx.cursor_read::<V2HashedStoragesHistory>()?,
            self.tx.cursor_dup_read::<V2HashedStorageChangeSets>()?,
            hashed_address,
            max_block_number,
            is_latest,
        ))
    }

    fn account_hashed_cursor<'tx>(
        &self,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::AccountHashedCursor<'tx>> {
        let is_latest = self.is_latest_block(max_block_number)?;
        Ok(MdbxAccountCursor::new(
            self.tx.cursor_read::<V2HashedAccounts>()?,
            self.tx.cursor_read::<V2HashedAccountsHistory>()?,
            self.tx.cursor_read::<V2HashedAccountsHistory>()?,
            self.tx.cursor_dup_read::<V2HashedAccountChangeSets>()?,
            max_block_number,
            is_latest,
        ))
    }

    fn fetch_trie_updates(&self, block_number: u64) -> OpProofsStorageResult<BlockStateDiff> {
        Ok(BlockStateDiff {
            sorted_trie_updates: self.fetch_block_trie_updates(block_number)?.into_sorted(),
            sorted_post_state: self.fetch_block_post_state(block_number)?.into_sorted(),
        })
    }
}
