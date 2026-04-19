//! Implements [`TrieCursorFactory`] and [`HashedCursorFactory`] for [`crate::OpProofsStore`] types.

use crate::{
    api::OpProofsProviderRO,
    cursor::{OpProofsHashedAccountCursor, OpProofsHashedStorageCursor, OpProofsTrieCursor},
};
use alloy_primitives::B256;
use reth_db::DatabaseError;
use reth_trie::{hashed_cursor::HashedCursorFactory, trie_cursor::TrieCursorFactory};

/// Factory for creating trie cursors for [`OpProofsProviderRO`].
#[derive(Debug, Clone)]
pub struct OpProofsTrieCursorFactory<P> {
    provider: P,
    block_number: u64,
}

impl<P: OpProofsProviderRO> OpProofsTrieCursorFactory<P> {
    /// Initializes new `OpProofsTrieCursorFactory`
    pub const fn new(provider: P, block_number: u64) -> Self {
        Self { provider, block_number }
    }
}

impl<P> TrieCursorFactory for OpProofsTrieCursorFactory<P>
where
    P: OpProofsProviderRO,
{
    type AccountTrieCursor<'a>
        = OpProofsTrieCursor<P::AccountTrieCursor<'a>>
    where
        Self: 'a;
    type StorageTrieCursor<'a>
        = OpProofsTrieCursor<P::StorageTrieCursor<'a>>
    where
        Self: 'a;

    fn account_trie_cursor(&self) -> Result<Self::AccountTrieCursor<'_>, DatabaseError> {
        Ok(OpProofsTrieCursor::new(
            self.provider
                .account_trie_cursor(self.block_number)
                .map_err(Into::<DatabaseError>::into)?,
        ))
    }

    fn storage_trie_cursor(
        &self,
        hashed_address: B256,
    ) -> Result<Self::StorageTrieCursor<'_>, DatabaseError> {
        Ok(OpProofsTrieCursor::new(
            self.provider
                .storage_trie_cursor(hashed_address, self.block_number)
                .map_err(Into::<DatabaseError>::into)?,
        ))
    }
}

/// Factory for creating hashed account cursors for [`OpProofsProviderRO`].
#[derive(Debug, Clone)]
pub struct OpProofsHashedAccountCursorFactory<P> {
    provider: P,
    block_number: u64,
}

impl<P: OpProofsProviderRO> OpProofsHashedAccountCursorFactory<P> {
    /// Creates a new `OpProofsHashedAccountCursorFactory` instance.
    pub const fn new(provider: P, block_number: u64) -> Self {
        Self { provider, block_number }
    }
}

impl<P> HashedCursorFactory for OpProofsHashedAccountCursorFactory<P>
where
    P: OpProofsProviderRO,
{
    type AccountCursor<'a>
        = OpProofsHashedAccountCursor<P::AccountHashedCursor<'a>>
    where
        Self: 'a;
    type StorageCursor<'a>
        = OpProofsHashedStorageCursor<P::StorageCursor<'a>>
    where
        Self: 'a;

    fn hashed_account_cursor(&self) -> Result<Self::AccountCursor<'_>, DatabaseError> {
        Ok(OpProofsHashedAccountCursor::new(
            self.provider
                .account_hashed_cursor(self.block_number)
                .map_err(Into::<DatabaseError>::into)?,
        ))
    }

    fn hashed_storage_cursor(
        &self,
        hashed_address: B256,
    ) -> Result<Self::StorageCursor<'_>, DatabaseError> {
        Ok(OpProofsHashedStorageCursor::new(
            self.provider
                .storage_hashed_cursor(hashed_address, self.block_number)
                .map_err(Into::<DatabaseError>::into)?,
        ))
    }
}
