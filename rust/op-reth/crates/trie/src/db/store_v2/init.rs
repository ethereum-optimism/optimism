//! [`OpProofsInitProvider`] implementation for [`MdbxProofsProviderV2`].

use super::MdbxProofsProviderV2;
use crate::{
    OpProofsStorageError, OpProofsStorageResult,
    api::{InitialStateAnchor, InitialStateStatus, OpProofsInitProvider},
    db::{
        HashedStorageKey, ProofWindowKey, StorageTrieKey, V2ProofWindow,
        models::{V2AccountsTrie, V2HashedAccounts, V2HashedStorages, V2StoragesTrie},
    },
};
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, U256};
use reth_db::{
    cursor::{DbCursorRO, DbCursorRW, DbDupCursorRW},
    transaction::{DbTx, DbTxMut},
};
use reth_primitives_traits::{Account, StorageEntry};
use reth_trie::{BranchNodeCompact, Nibbles, StorageTrieEntry, StoredNibbles, StoredNibblesSubKey};
use std::fmt::Debug;

impl<TX: DbTxMut + DbTx + Send + Sync + Debug + 'static> OpProofsInitProvider
    for MdbxProofsProviderV2<TX>
{
    fn initial_state_anchor(&self) -> OpProofsStorageResult<InitialStateAnchor> {
        let Some(block) = self.get_initial_state_anchor_inner()? else {
            return Ok(InitialStateAnchor::default());
        };

        let status = match self.get_block_number_hash_inner(ProofWindowKey::EarliestBlock) {
            Ok(_) => InitialStateStatus::Completed,
            Err(OpProofsStorageError::NoBlocksFound) => InitialStateStatus::InProgress,
            Err(err) => return Err(err),
        };

        // Scan the last entry in each current-state table to determine resume
        // keys. This allows multi-step initialization: if the process is
        // interrupted, the next run picks up where it left off.
        let latest_hashed_account_key =
            self.tx.cursor_read::<V2HashedAccounts>()?.last()?.map(|(k, _)| k);

        let latest_hashed_storage_key = self
            .tx
            .cursor_read::<V2HashedStorages>()?
            .last()?
            .map(|(addr, entry)| HashedStorageKey::new(addr, entry.key));

        let latest_account_trie_key =
            self.tx.cursor_read::<V2AccountsTrie>()?.last()?.map(|(k, _)| k);

        let latest_storage_trie_key = self
            .tx
            .cursor_read::<V2StoragesTrie>()?
            .last()?
            .map(|(addr, entry)| StorageTrieKey::new(addr, StoredNibbles(entry.nibbles.0)));

        Ok(InitialStateAnchor {
            block: Some(block),
            status,
            latest_account_trie_key,
            latest_storage_trie_key,
            latest_hashed_account_key,
            latest_hashed_storage_key,
        })
    }

    fn set_initial_state_anchor(&self, anchor: BlockNumHash) -> OpProofsStorageResult<()> {
        let mut cur = self.tx.cursor_write::<V2ProofWindow>()?;
        cur.insert(ProofWindowKey::InitialStateAnchor, &anchor.into())?;
        Ok(())
    }

    fn store_account_branches(
        &self,
        account_nodes: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> OpProofsStorageResult<()> {
        if account_nodes.is_empty() {
            return Ok(());
        }

        let mut cursor = self.tx.cursor_write::<V2AccountsTrie>()?;
        for (nibbles, maybe_node) in account_nodes {
            if let Some(node) = maybe_node {
                cursor.upsert(StoredNibbles(nibbles), &node)?;
            }
        }
        Ok(())
    }

    fn store_storage_branches(
        &self,
        hashed_address: B256,
        storage_nodes: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> OpProofsStorageResult<()> {
        if storage_nodes.is_empty() {
            return Ok(());
        }

        let mut cursor = self.tx.cursor_dup_write::<V2StoragesTrie>()?;
        for (nibbles, maybe_node) in storage_nodes {
            if let Some(node) = maybe_node {
                cursor.append_dup(
                    hashed_address,
                    StorageTrieEntry { nibbles: StoredNibblesSubKey(nibbles), node },
                )?;
            }
        }
        Ok(())
    }

    fn store_hashed_accounts(
        &self,
        accounts: Vec<(B256, Option<Account>)>,
    ) -> OpProofsStorageResult<()> {
        if accounts.is_empty() {
            return Ok(());
        }

        let mut cursor = self.tx.cursor_write::<V2HashedAccounts>()?;
        for (hashed_address, maybe_account) in accounts {
            if let Some(account) = maybe_account {
                cursor.append(hashed_address, &account)?;
            }
        }
        Ok(())
    }

    fn store_hashed_storages(
        &self,
        hashed_address: B256,
        storages: Vec<(B256, U256)>,
    ) -> OpProofsStorageResult<()> {
        if storages.is_empty() {
            return Ok(());
        }

        let mut cursor = self.tx.cursor_dup_write::<V2HashedStorages>()?;
        for (storage_key, value) in storages {
            cursor.append_dup(hashed_address, StorageEntry { key: storage_key, value })?;
        }
        Ok(())
    }

    fn commit_initial_state(&self) -> OpProofsStorageResult<BlockNumHash> {
        let anchor =
            self.get_initial_state_anchor_inner()?.ok_or(OpProofsStorageError::NoBlocksFound)?;
        self.set_earliest_block_number_inner(anchor.number, anchor.hash)?;
        self.set_latest_block_number_inner(anchor.number, anchor.hash)?;
        Ok(anchor)
    }

    fn commit(self) -> OpProofsStorageResult<()> {
        self.tx.commit()?;
        Ok(())
    }
}
