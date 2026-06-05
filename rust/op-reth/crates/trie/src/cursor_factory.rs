//! Implements [`TrieCursorFactory`] and [`HashedCursorFactory`] for [`crate::OpProofsStore`] types.

use crate::{
    api::{OpProofsProviderRO, OpProofsSnapshotProviderRO},
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

/// Factory for creating trie cursors backed by a snapshot reader.
///
/// Unlike [`OpProofsTrieCursorFactory`] (which reads history-aware cursors at
/// a given block number), this factory reads directly from the snapshot
/// tables. It carries no block-number context: the snapshot already reflects
/// trie state at a fixed anchor block. The caller is responsible for first
/// resolving that anchor via
/// [`crate::api::OpProofsSnapshotProviderRO::snapshot_anchor`] and ensuring
/// the block being queried matches it.
#[derive(Debug, Clone)]
pub struct SnapshotTrieCursorFactory<P> {
    reader: P,
}

impl<P: OpProofsSnapshotProviderRO> SnapshotTrieCursorFactory<P> {
    /// Create a new snapshot-backed trie cursor factory.
    pub const fn new(reader: P) -> Self {
        Self { reader }
    }
}

impl<P> TrieCursorFactory for SnapshotTrieCursorFactory<P>
where
    P: OpProofsSnapshotProviderRO,
{
    type AccountTrieCursor<'a>
        = P::SnapshotAccountTrieCursor<'a>
    where
        Self: 'a;
    type StorageTrieCursor<'a>
        = P::SnapshotStorageTrieCursor<'a>
    where
        Self: 'a;

    fn account_trie_cursor(&self) -> Result<Self::AccountTrieCursor<'_>, DatabaseError> {
        self.reader.snapshot_account_trie_cursor().map_err(Into::<DatabaseError>::into)
    }

    fn storage_trie_cursor(
        &self,
        hashed_address: B256,
    ) -> Result<Self::StorageTrieCursor<'_>, DatabaseError> {
        self.reader
            .snapshot_storage_trie_cursor(hashed_address)
            .map_err(Into::<DatabaseError>::into)
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
