//! Implements [`TrieCursorFactory`] and [`HashedCursorFactory`] for [`OpProofsStore`] types.

use crate::{
    OpProofsHashedAccountCursor, OpProofsHashedStorageCursor, OpProofsStorage, OpProofsStore,
    OpProofsTrieCursor,
};
use alloy_primitives::B256;
use reth_db::DatabaseError;
use reth_trie::{hashed_cursor::HashedCursorFactory, trie_cursor::TrieCursorFactory};
use std::marker::PhantomData;

/// Factory for creating trie cursors for [`OpProofsStore`].
#[derive(Debug, Clone)]
pub struct OpProofsTrieCursorFactory<'tx, S: OpProofsStore> {
    storage: &'tx OpProofsStorage<S>,
    block_number: u64,
    _marker: PhantomData<&'tx ()>,
}

impl<'tx, S: OpProofsStore> OpProofsTrieCursorFactory<'tx, S> {
    /// Initializes new `OpProofsTrieCursorFactory`
    pub const fn new(storage: &'tx OpProofsStorage<S>, block_number: u64) -> Self {
        Self { storage, block_number, _marker: PhantomData }
    }
}

impl<'tx, S> TrieCursorFactory for OpProofsTrieCursorFactory<'tx, S>
where
    S: OpProofsStore + 'tx,
{
    type AccountTrieCursor<'a>
        = OpProofsTrieCursor<S::AccountTrieCursor<'a>>
    where
        Self: 'a;
    type StorageTrieCursor<'a>
        = OpProofsTrieCursor<S::StorageTrieCursor<'a>>
    where
        Self: 'a;

    fn account_trie_cursor(&self) -> Result<Self::AccountTrieCursor<'_>, DatabaseError> {
        Ok(OpProofsTrieCursor::new(
            self.storage
                .account_trie_cursor(self.block_number)
                .map_err(Into::<DatabaseError>::into)?,
        ))
    }

    fn storage_trie_cursor(
        &self,
        hashed_address: B256,
    ) -> Result<Self::StorageTrieCursor<'_>, DatabaseError> {
        Ok(OpProofsTrieCursor::new(
            self.storage
                .storage_trie_cursor(hashed_address, self.block_number)
                .map_err(Into::<DatabaseError>::into)?,
        ))
    }
}

/// Factory for creating hashed account cursors for [`OpProofsStore`].
#[derive(Debug, Clone)]
pub struct OpProofsHashedAccountCursorFactory<'tx, S: OpProofsStore> {
    storage: &'tx OpProofsStorage<S>,
    block_number: u64,
    _marker: PhantomData<&'tx ()>,
}

impl<'tx, S: OpProofsStore> OpProofsHashedAccountCursorFactory<'tx, S> {
    /// Creates a new `OpProofsHashedAccountCursorFactory` instance.
    pub const fn new(storage: &'tx OpProofsStorage<S>, block_number: u64) -> Self {
        Self { storage, block_number, _marker: PhantomData }
    }
}

impl<'tx, S> HashedCursorFactory for OpProofsHashedAccountCursorFactory<'tx, S>
where
    S: OpProofsStore + 'tx,
{
    type AccountCursor<'a>
        = OpProofsHashedAccountCursor<S::AccountHashedCursor<'a>>
    where
        Self: 'a;
    type StorageCursor<'a>
        = OpProofsHashedStorageCursor<S::StorageCursor<'a>>
    where
        Self: 'a;

    fn hashed_account_cursor(&self) -> Result<Self::AccountCursor<'_>, DatabaseError> {
        Ok(OpProofsHashedAccountCursor::new(self.storage.account_hashed_cursor(self.block_number)?))
    }

    fn hashed_storage_cursor(
        &self,
        hashed_address: B256,
    ) -> Result<Self::StorageCursor<'_>, DatabaseError> {
        Ok(OpProofsHashedStorageCursor::new(
            self.storage.storage_hashed_cursor(hashed_address, self.block_number)?,
        ))
    }
}
