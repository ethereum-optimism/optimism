//! Provider for external proofs storage

use crate::{
    OpProofsProviderRO, OpProofsStorageError,
    proof::{
        DatabaseProof, DatabaseStateRoot, DatabaseStorageProof, DatabaseStorageRoot,
        DatabaseTrieWitness,
    },
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
    primitives::{Address, B256, Bytes, StorageValue, alloy_primitives::BlockNumber},
};
use reth_trie::{
    StateRoot, StorageRoot,
    hashed_cursor::HashedCursor,
    proof::{self, Proof},
    witness::TrieWitness,
};
use reth_trie_common::{
    AccountProof, HashedPostState, HashedStorage, KeccakKeyHasher, MultiProof, MultiProofTargets,
    StorageMultiProof, StorageProof, TrieInput, updates::TrieUpdates,
};
use std::fmt::Debug;

/// State provider for external proofs storage.
#[derive(Constructor)]
pub struct OpProofsStateProviderRef<'a, P> {
    /// Historical state provider for non-state related tasks.
    latest: Box<dyn StateProvider + Send + 'a>,

    /// Storage provider for state lookups.
    provider: P,

    /// Max block number that can be used for state lookups.
    block_number: BlockNumber,
}

impl<'a, P> Debug for OpProofsStateProviderRef<'a, P>
where
    P: Debug,
{
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("OpProofsStateProviderRef")
            .field("provider", &self.provider)
            .field("block_number", &self.block_number)
            .finish()
    }
}

impl From<OpProofsStorageError> for ProviderError {
    fn from(error: OpProofsStorageError) -> Self {
        Self::other(error)
    }
}

impl<'a, P> BlockHashReader for OpProofsStateProviderRef<'a, P> {
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

impl<'a, P> StateRootProvider for OpProofsStateProviderRef<'a, P>
where
    P: OpProofsProviderRO + Clone,
{
    fn state_root(&self, state: HashedPostState) -> ProviderResult<B256> {
        Ok(StateRoot::overlay_root(self.provider.clone(), self.block_number, state)?)
    }

    fn state_root_from_nodes(&self, input: TrieInput) -> ProviderResult<B256> {
        Ok(StateRoot::overlay_root_from_nodes(self.provider.clone(), self.block_number, input)?)
    }

    fn state_root_with_updates(
        &self,
        state: HashedPostState,
    ) -> ProviderResult<(B256, TrieUpdates)> {
        Ok(StateRoot::overlay_root_with_updates(self.provider.clone(), self.block_number, state)?)
    }

    fn state_root_from_nodes_with_updates(
        &self,
        input: TrieInput,
    ) -> ProviderResult<(B256, TrieUpdates)> {
        Ok(StateRoot::overlay_root_from_nodes_with_updates(
            self.provider.clone(),
            self.block_number,
            input,
        )?)
    }
}

impl<'a, P> StorageRootProvider for OpProofsStateProviderRef<'a, P>
where
    P: OpProofsProviderRO + Clone,
{
    fn storage_root(&self, address: Address, storage: HashedStorage) -> ProviderResult<B256> {
        StorageRoot::overlay_root(self.provider.clone(), self.block_number, address, storage)
            .map_err(|err| ProviderError::Database(err.into()))
    }

    fn storage_proof(
        &self,
        address: Address,
        slot: B256,
        storage: HashedStorage,
    ) -> ProviderResult<StorageProof> {
        proof::StorageProof::overlay_storage_proof(
            self.provider.clone(),
            self.block_number,
            address,
            slot,
            storage,
        )
        .map_err(ProviderError::from)
    }

    fn storage_multiproof(
        &self,
        address: Address,
        slots: &[B256],
        storage: HashedStorage,
    ) -> ProviderResult<StorageMultiProof> {
        proof::StorageProof::overlay_storage_multiproof(
            self.provider.clone(),
            self.block_number,
            address,
            slots,
            storage,
        )
        .map_err(ProviderError::from)
    }
}

impl<'a, P> StateProofProvider for OpProofsStateProviderRef<'a, P>
where
    P: OpProofsProviderRO + Clone,
{
    fn proof(
        &self,
        input: TrieInput,
        address: Address,
        slots: &[B256],
    ) -> ProviderResult<AccountProof> {
        Proof::overlay_account_proof(
            self.provider.clone(),
            self.block_number,
            input,
            address,
            slots,
        )
        .map_err(ProviderError::from)
    }

    fn multiproof(
        &self,
        input: TrieInput,
        targets: MultiProofTargets,
    ) -> ProviderResult<MultiProof> {
        Proof::overlay_multiproof(self.provider.clone(), self.block_number, input, targets)
            .map_err(ProviderError::from)
    }

    fn witness(&self, input: TrieInput, target: HashedPostState) -> ProviderResult<Vec<Bytes>> {
        TrieWitness::overlay_witness(self.provider.clone(), self.block_number, input, target)
            .map_err(ProviderError::from)
            .map(|hm| hm.into_values().collect())
    }
}

impl<'a, P> HashedPostStateProvider for OpProofsStateProviderRef<'a, P> {
    fn hashed_post_state(&self, bundle_state: &BundleState) -> HashedPostState {
        HashedPostState::from_bundle_state::<KeccakKeyHasher>(bundle_state.state())
    }
}

impl<'a, P> AccountReader for OpProofsStateProviderRef<'a, P>
where
    P: OpProofsProviderRO,
{
    fn basic_account(&self, address: &Address) -> ProviderResult<Option<Account>> {
        let hashed_key = keccak256(address.0);
        Ok(self
            .provider
            .account_hashed_cursor(self.block_number)
            .map_err(Into::<ProviderError>::into)?
            .seek(hashed_key)
            .map_err(Into::<ProviderError>::into)?
            .and_then(|(key, account)| (key == hashed_key).then_some(account)))
    }
}

impl<'a, P> StateProvider for OpProofsStateProviderRef<'a, P>
where
    P: OpProofsProviderRO + Clone,
{
    fn storage(&self, address: Address, storage_key: B256) -> ProviderResult<Option<StorageValue>> {
        let hashed_key = keccak256(storage_key);
        Ok(self
            .provider
            .storage_hashed_cursor(keccak256(address.0), self.block_number)
            .map_err(Into::<ProviderError>::into)?
            .seek(hashed_key)
            .map_err(Into::<ProviderError>::into)?
            .and_then(|(key, storage)| (key == hashed_key).then_some(storage)))
    }
}

impl<'a, P> BytecodeReader for OpProofsStateProviderRef<'a, P> {
    fn bytecode_by_hash(&self, code_hash: &B256) -> ProviderResult<Option<Bytecode>> {
        self.latest.bytecode_by_hash(code_hash)
    }
}

#[cfg(all(test, not(feature = "metrics")))]
mod tests {
    use super::*;
    use crate::{InMemoryProofsStorage, api::OpProofsStore};
    use reth_provider::noop::NoopProvider;

    #[test]
    fn test_op_proofs_state_provider_ref_debug() {
        let latest: Box<dyn StateProvider + Send> = Box::new(NoopProvider::default());
        let storage: crate::OpProofsStorage<InMemoryProofsStorage> =
            InMemoryProofsStorage::new().into();
        // Create a provider from the store (in memory storage implements OpProofsStore)
        let provider_ro = storage.provider_ro().unwrap();
        let block_number = 42u64;

        let provider = OpProofsStateProviderRef::new(latest, provider_ro, block_number);

        assert_eq!(
            format!("{:?}", provider),
            "OpProofsStateProviderRef { provider: InMemoryProofsProvider { inner: RwLock { data: InMemoryStorageInner { account_branches: {}, storage_branches: {}, hashed_accounts: {}, hashed_storages: {}, trie_updates: {}, post_states: {}, earliest_block: None, anchor_block: None } } }, block_number: 42 }"
        );
    }
}
