//! Provides proof operation implementations for [`OpProofsStorage`].

use crate::api::{
    OpProofsHashedCursor, OpProofsStorage, OpProofsStorageError,
    OpProofsTrieCursor as OpProofsDBTrieCursor,
};
use alloy_primitives::{
    keccak256,
    map::{B256Map, HashMap},
    Address, Bytes, B256, U256,
};
use reth_db::DatabaseError;
use reth_execution_errors::{StateProofError, StateRootError, StorageRootError, TrieWitnessError};
use reth_primitives_traits::Account;
use reth_trie::{
    hashed_cursor::{
        HashedCursor, HashedCursorFactory, HashedPostStateCursorFactory, HashedStorageCursor,
    },
    metrics::TrieRootMetrics,
    proof::{Proof, StorageProof},
    trie_cursor::{InMemoryTrieCursorFactory, TrieCursor, TrieCursorFactory},
    updates::TrieUpdates,
    witness::TrieWitness,
    AccountProof, BranchNodeCompact, HashedPostState, HashedPostStateSorted, HashedStorage,
    MultiProof, MultiProofTargets, Nibbles, StateRoot, StorageMultiProof, StorageRoot, TrieInput,
    TrieType,
};

/// Manages reading storage or account trie nodes from [`OpProofsDBTrieCursor`].
#[derive(Debug, Clone)]
pub struct OpProofsTrieCursor<C: OpProofsDBTrieCursor>(pub C);

impl<C: OpProofsDBTrieCursor> OpProofsTrieCursor<C> {
    /// Creates a new `OpProofsTrieCursor` instance.
    pub const fn new(cursor: C) -> Self {
        Self(cursor)
    }
}

impl From<OpProofsStorageError> for DatabaseError {
    fn from(error: OpProofsStorageError) -> Self {
        Self::Other(error.to_string())
    }
}

impl<C> TrieCursor for OpProofsTrieCursor<C>
where
    C: OpProofsDBTrieCursor + Send + Sync,
{
    fn seek_exact(
        &mut self,
        key: Nibbles,
    ) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        Ok(self.0.seek_exact(key)?)
    }

    fn seek(
        &mut self,
        key: Nibbles,
    ) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        Ok(self.0.seek(key)?)
    }

    fn next(&mut self) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        Ok(self.0.next()?)
    }

    fn current(&mut self) -> Result<Option<Nibbles>, DatabaseError> {
        Ok(self.0.current()?)
    }
}

/// Factory for creating trie cursors for [`OpProofsStorage`].
#[derive(Debug, Clone)]
pub struct OpProofsTrieCursorFactory<'tx, Storage: OpProofsStorage> {
    storage: &'tx Storage,
    block_number: u64,
    _marker: core::marker::PhantomData<&'tx ()>,
}

impl<'tx, Storage: OpProofsStorage> OpProofsTrieCursorFactory<'tx, Storage> {
    /// Initializes new `OpProofsTrieCursorFactory`
    pub const fn new(storage: &'tx Storage, block_number: u64) -> Self {
        Self { storage, block_number, _marker: core::marker::PhantomData }
    }
}

impl<'tx, Storage: OpProofsStorage + 'tx> TrieCursorFactory
    for OpProofsTrieCursorFactory<'tx, Storage>
{
    type AccountTrieCursor = OpProofsTrieCursor<Storage::AccountTrieCursor<'tx>>;
    type StorageTrieCursor = OpProofsTrieCursor<Storage::StorageTrieCursor<'tx>>;

    fn account_trie_cursor(&self) -> Result<Self::AccountTrieCursor, DatabaseError> {
        Ok(OpProofsTrieCursor::new(
            self.storage
                .account_trie_cursor(self.block_number)
                .map_err(Into::<DatabaseError>::into)?,
        ))
    }

    fn storage_trie_cursor(
        &self,
        hashed_address: B256,
    ) -> Result<Self::StorageTrieCursor, DatabaseError> {
        Ok(OpProofsTrieCursor::new(
            self.storage
                .storage_trie_cursor(hashed_address, self.block_number)
                .map_err(Into::<DatabaseError>::into)?,
        ))
    }
}

/// Manages reading hashed account nodes from external storage.
#[derive(Debug, Clone)]
pub struct OpProofsHashedAccountCursor<C>(pub C);

impl<C> OpProofsHashedAccountCursor<C> {
    /// Creates a new `OpProofsHashedAccountCursor` instance.
    pub const fn new(cursor: C) -> Self {
        Self(cursor)
    }
}

impl<C: OpProofsHashedCursor<Value = Account> + Send + Sync> HashedCursor
    for OpProofsHashedAccountCursor<C>
{
    type Value = Account;

    fn seek(&mut self, key: B256) -> Result<Option<(B256, Self::Value)>, DatabaseError> {
        Ok(self.0.seek(key)?)
    }

    fn next(&mut self) -> Result<Option<(B256, Self::Value)>, DatabaseError> {
        Ok(self.0.next()?)
    }
}

/// Manages reading hashed storage nodes from [`OpProofsHashedCursor`].
#[derive(Debug, Clone)]
pub struct OpProofsHashedStorageCursor<C: OpProofsHashedCursor<Value = U256>>(pub C);

impl<C: OpProofsHashedCursor<Value = U256>> OpProofsHashedStorageCursor<C> {
    /// Creates a new `OpProofsHashedStorageCursor` instance.
    pub const fn new(cursor: C) -> Self {
        Self(cursor)
    }
}

impl<C: OpProofsHashedCursor<Value = U256> + Send + Sync> HashedCursor
    for OpProofsHashedStorageCursor<C>
{
    type Value = U256;

    fn seek(&mut self, key: B256) -> Result<Option<(B256, Self::Value)>, DatabaseError> {
        Ok(self.0.seek(key)?)
    }

    fn next(&mut self) -> Result<Option<(B256, Self::Value)>, DatabaseError> {
        Ok(self.0.next()?)
    }
}

impl<C: OpProofsHashedCursor<Value = U256> + Send + Sync> HashedStorageCursor
    for OpProofsHashedStorageCursor<C>
{
    fn is_storage_empty(&mut self) -> Result<bool, DatabaseError> {
        Ok(self.0.is_storage_empty()?)
    }
}

/// Factory for creating hashed account cursors for [`OpProofsStorage`].
#[derive(Debug, Clone)]
pub struct OpProofsHashedAccountCursorFactory<'tx, Storage: OpProofsStorage> {
    storage: &'tx Storage,
    block_number: u64,
    _marker: core::marker::PhantomData<&'tx ()>,
}

impl<'tx, Storage: OpProofsStorage + 'tx> OpProofsHashedAccountCursorFactory<'tx, Storage> {
    /// Creates a new `OpProofsHashedAccountCursorFactory` instance.
    pub const fn new(storage: &'tx Storage, block_number: u64) -> Self {
        Self { storage, block_number, _marker: core::marker::PhantomData }
    }
}

impl<'tx, Storage: OpProofsStorage + 'tx> HashedCursorFactory
    for OpProofsHashedAccountCursorFactory<'tx, Storage>
{
    type AccountCursor = OpProofsHashedAccountCursor<Storage::AccountHashedCursor<'tx>>;
    type StorageCursor = OpProofsHashedStorageCursor<Storage::StorageCursor<'tx>>;

    fn hashed_account_cursor(&self) -> Result<Self::AccountCursor, DatabaseError> {
        Ok(OpProofsHashedAccountCursor::new(self.storage.account_hashed_cursor(self.block_number)?))
    }

    fn hashed_storage_cursor(
        &self,
        hashed_address: B256,
    ) -> Result<Self::StorageCursor, DatabaseError> {
        Ok(OpProofsHashedStorageCursor::new(
            self.storage.storage_hashed_cursor(hashed_address, self.block_number)?,
        ))
    }
}

/// Extends [`Proof`] with operations specific for working with [`OpProofsStorage`].
pub trait DatabaseProof<'tx, Storage> {
    /// Creates a new `DatabaseProof` instance from external storage.
    fn from_tx(storage: &'tx Storage, block_number: u64) -> Self;

    /// Generates the state proof for target account based on [`TrieInput`].
    fn overlay_account_proof(
        storage: &'tx Storage,
        block_number: u64,
        input: TrieInput,
        address: Address,
        slots: &[B256],
    ) -> Result<AccountProof, StateProofError>;

    /// Generates the state [`MultiProof`] for target hashed account and storage keys.
    fn overlay_multiproof(
        storage: &'tx Storage,
        block_number: u64,
        input: TrieInput,
        targets: MultiProofTargets,
    ) -> Result<MultiProof, StateProofError>;
}

impl<'tx, Storage: OpProofsStorage + Clone> DatabaseProof<'tx, Storage>
    for Proof<
        OpProofsTrieCursorFactory<'tx, Storage>,
        OpProofsHashedAccountCursorFactory<'tx, Storage>,
    >
{
    /// Create a new [`Proof`] instance from [`OpProofsStorage`].
    fn from_tx(storage: &'tx Storage, block_number: u64) -> Self {
        Self::new(
            OpProofsTrieCursorFactory::new(storage, block_number),
            OpProofsHashedAccountCursorFactory::new(storage, block_number),
        )
    }

    /// Generates the state proof for target account based on [`TrieInput`].
    fn overlay_account_proof(
        storage: &'tx Storage,
        block_number: u64,
        input: TrieInput,
        address: Address,
        slots: &[B256],
    ) -> Result<AccountProof, StateProofError> {
        let nodes_sorted = input.nodes.into_sorted();
        let state_sorted = input.state.into_sorted();
        Self::from_tx(storage, block_number)
            .with_trie_cursor_factory(InMemoryTrieCursorFactory::new(
                OpProofsTrieCursorFactory::new(storage, block_number),
                &nodes_sorted,
            ))
            .with_hashed_cursor_factory(HashedPostStateCursorFactory::new(
                OpProofsHashedAccountCursorFactory::new(storage, block_number),
                &state_sorted,
            ))
            .with_prefix_sets_mut(input.prefix_sets)
            .account_proof(address, slots)
    }

    /// Generates the state [`MultiProof`] for target hashed account and storage keys.
    fn overlay_multiproof(
        storage: &'tx Storage,
        block_number: u64,
        input: TrieInput,
        targets: MultiProofTargets,
    ) -> Result<MultiProof, StateProofError> {
        let nodes_sorted = input.nodes.into_sorted();
        let state_sorted = input.state.into_sorted();
        Self::from_tx(storage, block_number)
            .with_trie_cursor_factory(InMemoryTrieCursorFactory::new(
                OpProofsTrieCursorFactory::new(storage, block_number),
                &nodes_sorted,
            ))
            .with_hashed_cursor_factory(HashedPostStateCursorFactory::new(
                OpProofsHashedAccountCursorFactory::new(storage, block_number),
                &state_sorted,
            ))
            .with_prefix_sets_mut(input.prefix_sets)
            .multiproof(targets)
    }
}

/// Extends [`StorageProof`] with operations specific for working with [`OpProofsStorage`].
pub trait DatabaseStorageProof<'tx, Storage> {
    /// Create a new [`StorageProof`] from [`OpProofsStorage`] and account address.
    fn from_tx(storage: &'tx Storage, block_number: u64, address: Address) -> Self;

    /// Generates the storage proof for target slot based on [`TrieInput`].
    fn overlay_storage_proof(
        storage: &'tx Storage,
        block_number: u64,
        address: Address,
        slot: B256,
        storage: HashedStorage,
    ) -> Result<reth_trie::StorageProof, StateProofError>;

    /// Generates the storage multiproof for target slots based on [`TrieInput`].
    fn overlay_storage_multiproof(
        storage: &'tx Storage,
        block_number: u64,
        address: Address,
        slots: &[B256],
        storage: HashedStorage,
    ) -> Result<StorageMultiProof, StateProofError>;
}

impl<'tx, Storage: OpProofsStorage + 'tx + Clone> DatabaseStorageProof<'tx, Storage>
    for StorageProof<
        OpProofsTrieCursorFactory<'tx, Storage>,
        OpProofsHashedAccountCursorFactory<'tx, Storage>,
    >
{
    /// Create a new [`StorageProof`] from [`OpProofsStorage`] and account address.
    fn from_tx(storage: &'tx Storage, block_number: u64, address: Address) -> Self {
        Self::new(
            OpProofsTrieCursorFactory::new(storage, block_number),
            OpProofsHashedAccountCursorFactory::new(storage, block_number),
            address,
        )
    }

    fn overlay_storage_proof(
        storage: &'tx Storage,
        block_number: u64,
        address: Address,
        slot: B256,
        hashed_storage: HashedStorage,
    ) -> Result<reth_trie::StorageProof, StateProofError> {
        let hashed_address = keccak256(address);
        let prefix_set = hashed_storage.construct_prefix_set();
        let state_sorted = HashedPostStateSorted::new(
            Default::default(),
            HashMap::from_iter([(hashed_address, hashed_storage.into_sorted())]),
        );
        Self::from_tx(storage, block_number, address)
            .with_hashed_cursor_factory(HashedPostStateCursorFactory::new(
                OpProofsHashedAccountCursorFactory::new(storage, block_number),
                &state_sorted,
            ))
            .with_prefix_set_mut(prefix_set)
            .storage_proof(slot)
    }

    fn overlay_storage_multiproof(
        storage: &'tx Storage,
        block_number: u64,
        address: Address,
        slots: &[B256],
        hashed_storage: HashedStorage,
    ) -> Result<StorageMultiProof, StateProofError> {
        let hashed_address = keccak256(address);
        let targets = slots.iter().map(keccak256).collect();
        let prefix_set = hashed_storage.construct_prefix_set();
        let state_sorted = HashedPostStateSorted::new(
            Default::default(),
            HashMap::from_iter([(hashed_address, hashed_storage.into_sorted())]),
        );
        Self::from_tx(storage, block_number, address)
            .with_hashed_cursor_factory(HashedPostStateCursorFactory::new(
                OpProofsHashedAccountCursorFactory::new(storage, block_number),
                &state_sorted,
            ))
            .with_prefix_set_mut(prefix_set)
            .storage_multiproof(targets)
    }
}

/// Extends [`StateRoot`] with operations specific for working with [`OpProofsStorage`].
pub trait DatabaseStateRoot<'tx, Storage: OpProofsStorage + 'tx + Clone>: Sized {
    /// Calculate the state root for this [`HashedPostState`].
    /// Internally, this method retrieves prefixsets and uses them
    /// to calculate incremental state root.
    ///
    /// # Returns
    ///
    /// The state root for this [`HashedPostState`].
    fn overlay_root(
        storage: &'tx Storage,
        block_number: u64,
        post_state: HashedPostState,
    ) -> Result<B256, StateRootError>;

    /// Calculates the state root for this [`HashedPostState`] and returns it alongside trie
    /// updates. See [`Self::overlay_root`] for more info.
    fn overlay_root_with_updates(
        storage: &'tx Storage,
        block_number: u64,
        post_state: HashedPostState,
    ) -> Result<(B256, TrieUpdates), StateRootError>;

    /// Calculates the state root for provided [`HashedPostState`] using cached intermediate nodes.
    fn overlay_root_from_nodes(
        storage: &'tx Storage,
        block_number: u64,
        input: TrieInput,
    ) -> Result<B256, StateRootError>;

    /// Calculates the state root and trie updates for provided [`HashedPostState`] using
    /// cached intermediate nodes.
    fn overlay_root_from_nodes_with_updates(
        storage: &'tx Storage,
        block_number: u64,
        input: TrieInput,
    ) -> Result<(B256, TrieUpdates), StateRootError>;
}

impl<'tx, Storage: OpProofsStorage + 'tx + Clone> DatabaseStateRoot<'tx, Storage>
    for StateRoot<
        OpProofsTrieCursorFactory<'tx, Storage>,
        OpProofsHashedAccountCursorFactory<'tx, Storage>,
    >
{
    fn overlay_root(
        storage: &'tx Storage,
        block_number: u64,
        post_state: HashedPostState,
    ) -> Result<B256, StateRootError> {
        let prefix_sets = post_state.construct_prefix_sets().freeze();
        let state_sorted = post_state.into_sorted();
        StateRoot::new(
            OpProofsTrieCursorFactory::new(storage, block_number),
            HashedPostStateCursorFactory::new(
                OpProofsHashedAccountCursorFactory::new(storage, block_number),
                &state_sorted,
            ),
        )
        .with_prefix_sets(prefix_sets)
        .root()
    }

    fn overlay_root_with_updates(
        storage: &'tx Storage,
        block_number: u64,
        post_state: HashedPostState,
    ) -> Result<(B256, TrieUpdates), StateRootError> {
        let prefix_sets = post_state.construct_prefix_sets().freeze();
        let state_sorted = post_state.into_sorted();
        StateRoot::new(
            OpProofsTrieCursorFactory::new(storage, block_number),
            HashedPostStateCursorFactory::new(
                OpProofsHashedAccountCursorFactory::new(storage, block_number),
                &state_sorted,
            ),
        )
        .with_prefix_sets(prefix_sets)
        .root_with_updates()
    }

    fn overlay_root_from_nodes(
        storage: &'tx Storage,
        block_number: u64,
        input: TrieInput,
    ) -> Result<B256, StateRootError> {
        let state_sorted = input.state.into_sorted();
        let nodes_sorted = input.nodes.into_sorted();
        StateRoot::new(
            InMemoryTrieCursorFactory::new(
                OpProofsTrieCursorFactory::new(storage, block_number),
                &nodes_sorted,
            ),
            HashedPostStateCursorFactory::new(
                OpProofsHashedAccountCursorFactory::new(storage, block_number),
                &state_sorted,
            ),
        )
        .with_prefix_sets(input.prefix_sets.freeze())
        .root()
    }

    fn overlay_root_from_nodes_with_updates(
        storage: &'tx Storage,
        block_number: u64,
        input: TrieInput,
    ) -> Result<(B256, TrieUpdates), StateRootError> {
        let state_sorted = input.state.into_sorted();
        let nodes_sorted = input.nodes.into_sorted();
        StateRoot::new(
            InMemoryTrieCursorFactory::new(
                OpProofsTrieCursorFactory::new(storage, block_number),
                &nodes_sorted,
            ),
            HashedPostStateCursorFactory::new(
                OpProofsHashedAccountCursorFactory::new(storage, block_number),
                &state_sorted,
            ),
        )
        .with_prefix_sets(input.prefix_sets.freeze())
        .root_with_updates()
    }
}

/// Extends [`StorageRoot`] with operations specific for working with [`OpProofsStorage`].
pub trait DatabaseStorageRoot<'tx, Storage: OpProofsStorage + 'tx + Clone> {
    /// Calculates the storage root for provided [`HashedStorage`].
    fn overlay_root(
        storage: &'tx Storage,
        block_number: u64,
        address: Address,
        hashed_storage: HashedStorage,
    ) -> Result<B256, StorageRootError>;
}

impl<'tx, Storage: OpProofsStorage + 'tx + Clone> DatabaseStorageRoot<'tx, Storage>
    for StorageRoot<
        OpProofsTrieCursorFactory<'tx, Storage>,
        OpProofsHashedAccountCursorFactory<'tx, Storage>,
    >
{
    fn overlay_root(
        storage: &'tx Storage,
        block_number: u64,
        address: Address,
        hashed_storage: HashedStorage,
    ) -> Result<B256, StorageRootError> {
        let prefix_set = hashed_storage.construct_prefix_set().freeze();
        let state_sorted =
            HashedPostState::from_hashed_storage(keccak256(address), hashed_storage).into_sorted();
        StorageRoot::new(
            OpProofsTrieCursorFactory::new(storage, block_number),
            HashedPostStateCursorFactory::new(
                OpProofsHashedAccountCursorFactory::new(storage, block_number),
                &state_sorted,
            ),
            address,
            prefix_set,
            TrieRootMetrics::new(TrieType::Storage),
        )
        .root()
    }
}

/// Extends [`TrieWitness`] with operations specific for working with [`OpProofsStorage`].
pub trait DatabaseTrieWitness<'tx, Storage: OpProofsStorage + 'tx + Clone> {
    /// Creates a new [`TrieWitness`] instance from [`OpProofsStorage`].
    fn from_tx(storage: &'tx Storage, block_number: u64) -> Self;

    /// Generates the trie witness for the target state based on [`TrieInput`].
    fn overlay_witness(
        storage: &'tx Storage,
        block_number: u64,
        input: TrieInput,
        target: HashedPostState,
    ) -> Result<B256Map<Bytes>, TrieWitnessError>;
}

impl<'tx, Storage: OpProofsStorage + 'tx + Clone> DatabaseTrieWitness<'tx, Storage>
    for TrieWitness<
        OpProofsTrieCursorFactory<'tx, Storage>,
        OpProofsHashedAccountCursorFactory<'tx, Storage>,
    >
{
    fn from_tx(storage: &'tx Storage, block_number: u64) -> Self {
        Self::new(
            OpProofsTrieCursorFactory::new(storage, block_number),
            OpProofsHashedAccountCursorFactory::new(storage, block_number),
        )
    }

    fn overlay_witness(
        storage: &'tx Storage,
        block_number: u64,
        input: TrieInput,
        target: HashedPostState,
    ) -> Result<B256Map<Bytes>, TrieWitnessError> {
        let nodes_sorted = input.nodes.into_sorted();
        let state_sorted = input.state.into_sorted();
        Self::from_tx(storage, block_number)
            .with_trie_cursor_factory(InMemoryTrieCursorFactory::new(
                OpProofsTrieCursorFactory::new(storage, block_number),
                &nodes_sorted,
            ))
            .with_hashed_cursor_factory(HashedPostStateCursorFactory::new(
                OpProofsHashedAccountCursorFactory::new(storage, block_number),
                &state_sorted,
            ))
            .with_prefix_sets_mut(input.prefix_sets)
            .always_include_root_node()
            .compute(target)
    }
}
