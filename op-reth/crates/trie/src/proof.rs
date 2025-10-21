//! Provides proof operation implementations for [`OpProofsStorage`].

use crate::{
    OpProofsHashedAccountCursorFactory, OpProofsStorage, OpProofsStore, OpProofsTrieCursorFactory,
};
use alloy_primitives::{
    keccak256,
    map::{B256Map, HashMap},
    Address, Bytes, B256,
};
use reth_execution_errors::{StateProofError, StateRootError, StorageRootError, TrieWitnessError};
use reth_trie::{
    hashed_cursor::HashedPostStateCursorFactory,
    metrics::TrieRootMetrics,
    proof::{Proof, StorageProof},
    trie_cursor::InMemoryTrieCursorFactory,
    updates::TrieUpdates,
    witness::TrieWitness,
    AccountProof, HashedPostState, HashedPostStateSorted, HashedStorage, MultiProof,
    MultiProofTargets, StateRoot, StorageMultiProof, StorageRoot, TrieInput, TrieType,
};

/// Extends [`Proof`] with operations specific for working with [`OpProofsStorage`].
pub trait DatabaseProof<'tx, S> {
    /// Creates a new `DatabaseProof` instance from external storage.
    fn from_tx(storage: &'tx OpProofsStorage<S>, block_number: u64) -> Self;

    /// Generates the state proof for target account based on [`TrieInput`].
    fn overlay_account_proof(
        storage: &'tx OpProofsStorage<S>,
        block_number: u64,
        input: TrieInput,
        address: Address,
        slots: &[B256],
    ) -> Result<AccountProof, StateProofError>;

    /// Generates the state [`MultiProof`] for target hashed account and storage keys.
    fn overlay_multiproof(
        storage: &'tx OpProofsStorage<S>,
        block_number: u64,
        input: TrieInput,
        targets: MultiProofTargets,
    ) -> Result<MultiProof, StateProofError>;
}

impl<'tx, S> DatabaseProof<'tx, S>
    for Proof<OpProofsTrieCursorFactory<'tx, S>, OpProofsHashedAccountCursorFactory<'tx, S>>
where
    S: OpProofsStore + Clone,
{
    /// Create a new [`Proof`] instance from [`OpProofsStorage`].
    fn from_tx(storage: &'tx OpProofsStorage<S>, block_number: u64) -> Self {
        Self::new(
            OpProofsTrieCursorFactory::new(storage, block_number),
            OpProofsHashedAccountCursorFactory::new(storage, block_number),
        )
    }

    /// Generates the state proof for target account based on [`TrieInput`].
    fn overlay_account_proof(
        storage: &'tx OpProofsStorage<S>,
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
        storage: &'tx OpProofsStorage<S>,
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
pub trait DatabaseStorageProof<'tx, S> {
    /// Create a new [`StorageProof`] from [`OpProofsStorage`] and account address.
    fn from_tx(storage: &'tx OpProofsStorage<S>, block_number: u64, address: Address) -> Self;

    /// Generates the storage proof for target slot based on [`TrieInput`].
    fn overlay_storage_proof(
        storage: &'tx OpProofsStorage<S>,
        block_number: u64,
        address: Address,
        slot: B256,
        storage: HashedStorage,
    ) -> Result<reth_trie::StorageProof, StateProofError>;

    /// Generates the storage multiproof for target slots based on [`TrieInput`].
    fn overlay_storage_multiproof(
        storage: &'tx OpProofsStorage<S>,
        block_number: u64,
        address: Address,
        slots: &[B256],
        storage: HashedStorage,
    ) -> Result<StorageMultiProof, StateProofError>;
}

impl<'tx, S> DatabaseStorageProof<'tx, S>
    for StorageProof<OpProofsTrieCursorFactory<'tx, S>, OpProofsHashedAccountCursorFactory<'tx, S>>
where
    S: OpProofsStore + 'tx + Clone,
{
    /// Create a new [`StorageProof`] from [`OpProofsStorage`] and account address.
    fn from_tx(storage: &'tx OpProofsStorage<S>, block_number: u64, address: Address) -> Self {
        Self::new(
            OpProofsTrieCursorFactory::new(storage, block_number),
            OpProofsHashedAccountCursorFactory::new(storage, block_number),
            address,
        )
    }

    fn overlay_storage_proof(
        storage: &'tx OpProofsStorage<S>,
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
        storage: &'tx OpProofsStorage<S>,
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
pub trait DatabaseStateRoot<'tx, S: OpProofsStore + 'tx + Clone>: Sized {
    /// Calculate the state root for this [`HashedPostState`].
    /// Internally, this method retrieves prefixsets and uses them
    /// to calculate incremental state root.
    ///
    /// # Returns
    ///
    /// The state root for this [`HashedPostState`].
    fn overlay_root(
        storage: &'tx OpProofsStorage<S>,
        block_number: u64,
        post_state: HashedPostState,
    ) -> Result<B256, StateRootError>;

    /// Calculates the state root for this [`HashedPostState`] and returns it alongside trie
    /// updates. See [`Self::overlay_root`] for more info.
    fn overlay_root_with_updates(
        storage: &'tx OpProofsStorage<S>,
        block_number: u64,
        post_state: HashedPostState,
    ) -> Result<(B256, TrieUpdates), StateRootError>;

    /// Calculates the state root for provided [`HashedPostState`] using cached intermediate nodes.
    fn overlay_root_from_nodes(
        storage: &'tx OpProofsStorage<S>,
        block_number: u64,
        input: TrieInput,
    ) -> Result<B256, StateRootError>;

    /// Calculates the state root and trie updates for provided [`HashedPostState`] using
    /// cached intermediate nodes.
    fn overlay_root_from_nodes_with_updates(
        storage: &'tx OpProofsStorage<S>,
        block_number: u64,
        input: TrieInput,
    ) -> Result<(B256, TrieUpdates), StateRootError>;
}

impl<'tx, S> DatabaseStateRoot<'tx, S>
    for StateRoot<OpProofsTrieCursorFactory<'tx, S>, OpProofsHashedAccountCursorFactory<'tx, S>>
where
    S: OpProofsStore + 'tx + Clone,
{
    fn overlay_root(
        storage: &'tx OpProofsStorage<S>,
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
        storage: &'tx OpProofsStorage<S>,
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
        storage: &'tx OpProofsStorage<S>,
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
        storage: &'tx OpProofsStorage<S>,
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
pub trait DatabaseStorageRoot<'tx, S: OpProofsStore + 'tx + Clone> {
    /// Calculates the storage root for provided [`HashedStorage`].
    fn overlay_root(
        storage: &'tx OpProofsStorage<S>,
        block_number: u64,
        address: Address,
        hashed_storage: HashedStorage,
    ) -> Result<B256, StorageRootError>;
}

impl<'tx, S> DatabaseStorageRoot<'tx, S>
    for StorageRoot<OpProofsTrieCursorFactory<'tx, S>, OpProofsHashedAccountCursorFactory<'tx, S>>
where
    S: OpProofsStore + 'tx + Clone,
{
    fn overlay_root(
        storage: &'tx OpProofsStorage<S>,
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
pub trait DatabaseTrieWitness<'tx, S: OpProofsStore + 'tx + Clone> {
    /// Creates a new [`TrieWitness`] instance from [`OpProofsStorage`].
    fn from_tx(storage: &'tx OpProofsStorage<S>, block_number: u64) -> Self;

    /// Generates the trie witness for the target state based on [`TrieInput`].
    fn overlay_witness(
        storage: &'tx OpProofsStorage<S>,
        block_number: u64,
        input: TrieInput,
        target: HashedPostState,
    ) -> Result<B256Map<Bytes>, TrieWitnessError>;
}

impl<'tx, S> DatabaseTrieWitness<'tx, S>
    for TrieWitness<OpProofsTrieCursorFactory<'tx, S>, OpProofsHashedAccountCursorFactory<'tx, S>>
where
    S: OpProofsStore + 'tx + Clone,
{
    fn from_tx(storage: &'tx OpProofsStorage<S>, block_number: u64) -> Self {
        Self::new(
            OpProofsTrieCursorFactory::new(storage, block_number),
            OpProofsHashedAccountCursorFactory::new(storage, block_number),
        )
    }

    fn overlay_witness(
        storage: &'tx OpProofsStorage<S>,
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
