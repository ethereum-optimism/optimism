//! Provider for external proofs storage

use crate::{
    proof::{
        DatabaseProof, DatabaseStateRoot, DatabaseStorageProof, DatabaseStorageRoot,
        DatabaseTrieWitness,
    },
    OpProofsStorage, OpProofsStorageError, OpProofsStore,
};
use alloy_primitives::keccak256;
use derive_more::Constructor;
use reth_primitives_traits::{Account, Bytecode};
use reth_provider::{
    AccountReader, BlockHashReader, BytecodeReader, HashedPostStateProvider, ProviderError,
    ProviderResult, StateProofProvider, StateProvider, StateRootProvider, StorageRootProvider,
};
use reth_revm::{
    db::BundleState,
    primitives::{alloy_primitives::BlockNumber, Address, Bytes, StorageValue, B256},
};
use reth_trie::{
    hashed_cursor::HashedCursor,
    proof::{Proof, StorageProof},
    updates::TrieUpdates,
    witness::TrieWitness,
    AccountProof, HashedPostState, HashedStorage, KeccakKeyHasher, MultiProof, MultiProofTargets,
    StateRoot, StorageMultiProof, StorageRoot, TrieInput,
};
use std::fmt::Debug;

/// State provider for external proofs storage.
#[derive(Constructor)]
pub struct OpProofsStateProviderRef<'a, Storage: OpProofsStore> {
    /// Historical state provider for non-state related tasks.
    latest: Box<dyn StateProvider + 'a>,

    /// Storage provider for state lookups.
    storage: &'a OpProofsStorage<Storage>,

    /// Max block number that can be used for state lookups.
    block_number: BlockNumber,
}

impl<'a, Storage> Debug for OpProofsStateProviderRef<'a, Storage>
where
    Storage: OpProofsStore + 'a + Debug,
{
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("OpProofsStateProviderRef")
            .field("storage", &self.storage)
            .field("block_number", &self.block_number)
            .finish()
    }
}

impl From<OpProofsStorageError> for ProviderError {
    fn from(error: OpProofsStorageError) -> Self {
        Self::other(error)
    }
}

impl<'a, Storage: OpProofsStore> BlockHashReader for OpProofsStateProviderRef<'a, Storage> {
    fn block_hash(&self, number: BlockNumber) -> ProviderResult<Option<B256>> {
        self.latest.block_hash(number)
    }

    fn canonical_hashes_range(
        &self,
        start: BlockNumber,
        end: BlockNumber,
    ) -> ProviderResult<Vec<B256>> {
        self.latest.canonical_hashes_range(start, end)
    }
}

impl<'a, Storage: OpProofsStore + Clone> StateRootProvider
    for OpProofsStateProviderRef<'a, Storage>
{
    fn state_root(&self, state: HashedPostState) -> ProviderResult<B256> {
        StateRoot::overlay_root(self.storage, self.block_number, state)
            .map_err(|err| ProviderError::Database(err.into()))
    }

    fn state_root_from_nodes(&self, input: TrieInput) -> ProviderResult<B256> {
        StateRoot::overlay_root_from_nodes(self.storage, self.block_number, input)
            .map_err(|err| ProviderError::Database(err.into()))
    }

    fn state_root_with_updates(
        &self,
        state: HashedPostState,
    ) -> ProviderResult<(B256, TrieUpdates)> {
        StateRoot::overlay_root_with_updates(self.storage, self.block_number, state)
            .map_err(|err| ProviderError::Database(err.into()))
    }

    fn state_root_from_nodes_with_updates(
        &self,
        input: TrieInput,
    ) -> ProviderResult<(B256, TrieUpdates)> {
        StateRoot::overlay_root_from_nodes_with_updates(self.storage, self.block_number, input)
            .map_err(|err| ProviderError::Database(err.into()))
    }
}

impl<'a, Storage: OpProofsStore + Clone> StorageRootProvider
    for OpProofsStateProviderRef<'a, Storage>
{
    fn storage_root(&self, address: Address, storage: HashedStorage) -> ProviderResult<B256> {
        StorageRoot::overlay_root(self.storage, self.block_number, address, storage)
            .map_err(|err| ProviderError::Database(err.into()))
    }

    fn storage_proof(
        &self,
        address: Address,
        slot: B256,
        storage: HashedStorage,
    ) -> ProviderResult<reth_trie::StorageProof> {
        StorageProof::overlay_storage_proof(self.storage, self.block_number, address, slot, storage)
            .map_err(ProviderError::from)
    }

    fn storage_multiproof(
        &self,
        address: Address,
        slots: &[B256],
        storage: HashedStorage,
    ) -> ProviderResult<StorageMultiProof> {
        StorageProof::overlay_storage_multiproof(
            self.storage,
            self.block_number,
            address,
            slots,
            storage,
        )
        .map_err(ProviderError::from)
    }
}

impl<'a, Storage: OpProofsStore + Clone> StateProofProvider
    for OpProofsStateProviderRef<'a, Storage>
{
    fn proof(
        &self,
        input: TrieInput,
        address: Address,
        slots: &[B256],
    ) -> ProviderResult<AccountProof> {
        Proof::overlay_account_proof(self.storage, self.block_number, input, address, slots)
            .map_err(ProviderError::from)
    }

    fn multiproof(
        &self,
        input: TrieInput,
        targets: MultiProofTargets,
    ) -> ProviderResult<MultiProof> {
        Proof::overlay_multiproof(self.storage, self.block_number, input, targets)
            .map_err(ProviderError::from)
    }

    fn witness(&self, input: TrieInput, target: HashedPostState) -> ProviderResult<Vec<Bytes>> {
        TrieWitness::overlay_witness(self.storage, self.block_number, input, target)
            .map_err(ProviderError::from)
            .map(|hm| hm.into_values().collect())
    }
}

impl<'a, Storage: OpProofsStore> HashedPostStateProvider for OpProofsStateProviderRef<'a, Storage> {
    fn hashed_post_state(&self, bundle_state: &BundleState) -> HashedPostState {
        HashedPostState::from_bundle_state::<KeccakKeyHasher>(bundle_state.state())
    }
}

impl<'a, Storage: OpProofsStore> AccountReader for OpProofsStateProviderRef<'a, Storage> {
    fn basic_account(&self, address: &Address) -> ProviderResult<Option<Account>> {
        let hashed_key = keccak256(address.0);
        Ok(self
            .storage
            .account_hashed_cursor(self.block_number)
            .map_err(Into::<ProviderError>::into)?
            .seek(hashed_key)
            .map_err(Into::<ProviderError>::into)?
            .and_then(|(key, account)| (key == hashed_key).then_some(account)))
    }
}

impl<'a, Storage> StateProvider for OpProofsStateProviderRef<'a, Storage>
where
    Storage: OpProofsStore + Clone,
{
    fn storage(&self, address: Address, storage_key: B256) -> ProviderResult<Option<StorageValue>> {
        let hashed_key = keccak256(storage_key);
        Ok(self
            .storage
            .storage_hashed_cursor(keccak256(address.0), self.block_number)
            .map_err(Into::<ProviderError>::into)?
            .seek(hashed_key)
            .map_err(Into::<ProviderError>::into)?
            .and_then(|(key, storage)| (key == hashed_key).then_some(storage)))
    }
}

impl<'a, Storage: OpProofsStore> BytecodeReader for OpProofsStateProviderRef<'a, Storage> {
    fn bytecode_by_hash(&self, code_hash: &B256) -> ProviderResult<Option<Bytecode>> {
        self.latest.bytecode_by_hash(code_hash)
    }
}

#[cfg(all(test, not(feature = "metrics")))]
mod tests {
    use super::*;
    use crate::InMemoryProofsStorage;
    use reth_provider::noop::NoopProvider;

    #[test]
    fn test_op_proofs_state_provider_ref_debug() {
        let latest: Box<dyn StateProvider> = Box::new(NoopProvider::default());
        let storage: crate::OpProofsStorage<InMemoryProofsStorage> =
            InMemoryProofsStorage::new().into();
        let block_number = 42u64;

        let provider = OpProofsStateProviderRef::new(latest, &storage, block_number);

        assert_eq!(
            format!("{:?}", provider),
            "OpProofsStateProviderRef { storage: InMemoryProofsStorage { inner: RwLock { data: InMemoryStorageInner { account_branches: {}, storage_branches: {}, hashed_accounts: {}, hashed_storages: {}, address_mappings: {}, trie_updates: {}, post_states: {}, earliest_block: None } } }, block_number: 42 }"
        );
    }
}
