//! Overlay Provider for external proofs storage

use crate::{BlockStateDiff, api::OpProofsProviderRO, provider::OpProofsStateProviderRef};
use alloy_eips::eip1898::BlockWithParent;
use alloy_primitives::{Address, B256, BlockNumber, Bytes, StorageValue, keccak256};
use reth_primitives_traits::{Account, Bytecode};
use reth_provider::{
    AccountReader, BlockHashReader, BytecodeReader, HashedPostStateProvider, ProviderResult,
    StateProofProvider, StateProvider, StateRootProvider, StorageRootProvider,
};
use reth_revm::db::BundleState;
use reth_trie::{
    AccountProof, ExecutionWitnessMode, HashedPostState, HashedStorage, MultiProof,
    MultiProofTargets, StorageMultiProof, TrieInput, updates::TrieUpdates,
};
use std::{
    fmt::Debug,
    sync::{Arc, OnceLock},
};

/// A state provider that overlays in-memory buffered blocks on top of the persistent proofs
/// storage.
#[derive(Debug)]
pub(crate) struct MemoryOverlayOpProofsStateProviderRef<'a, P>
where
    P: OpProofsProviderRO,
{
    inner: OpProofsStateProviderRef<'a, P>,
    /// Ordered list of buffered blocks (Oldest to Newest).
    memory: Vec<Arc<(BlockWithParent, BlockStateDiff)>>,
    trie_input: OnceLock<TrieInput>,
}

impl<'a, P> MemoryOverlayOpProofsStateProviderRef<'a, P>
where
    P: OpProofsProviderRO + Clone,
{
    /// Create a new overlay provider.
    ///
    /// `memory` should be strictly ordered from oldest to newest.
    pub(crate) const fn new(
        inner: OpProofsStateProviderRef<'a, P>,
        memory: Vec<Arc<(BlockWithParent, BlockStateDiff)>>,
    ) -> Self {
        Self { inner, memory, trie_input: OnceLock::new() }
    }

    /// Aggregates trie updates from the memory buffer.
    fn trie_input(&self) -> &TrieInput {
        self.trie_input.get_or_init(|| {
            let mut input = TrieInput::default();
            // Iterate over buffered blocks to collect all trie nodes and state updates.
            // memory is expected to be ordered Oldest -> Newest.
            for state in &self.memory {
                let diff = &state.1;
                input.nodes.extend_from_sorted(&diff.sorted_trie_updates);
                input.state.extend_from_sorted(&diff.sorted_post_state);
            }
            input
        })
    }

    /// Merges the overlay storage for the given address with the provided storage.
    fn merged_hashed_storage(&self, address: Address, storage: HashedStorage) -> HashedStorage {
        let hashed_address = keccak256(address);
        // Start with the overlay storage from our trie input cache
        let state = &self.trie_input().state;
        let mut overlay = state.storages.get(&hashed_address).cloned().unwrap_or_default();

        overlay.extend(&storage);
        overlay
    }
}

impl<'a, P> AccountReader for MemoryOverlayOpProofsStateProviderRef<'a, P>
where
    P: OpProofsProviderRO + Clone,
{
    fn basic_account(&self, address: &Address) -> ProviderResult<Option<Account>> {
        let hashed_address = keccak256(address);
        // Check buffer via trie_input cache
        if let Some(account) = self.trie_input().state.accounts.get(&hashed_address) {
            return Ok(*account);
        }
        self.inner.basic_account(address)
    }
}

impl<'a, P> StateProvider for MemoryOverlayOpProofsStateProviderRef<'a, P>
where
    P: OpProofsProviderRO + Clone,
{
    fn storage(&self, address: Address, storage_key: B256) -> ProviderResult<Option<StorageValue>> {
        let hashed_slot = keccak256(storage_key);
        let hashed_address = keccak256(address);

        // Check buffer via trie_input cache
        let state = &self.trie_input().state;

        // Check for storage updates or wipes in the overlay
        if let Some(account_storage) = state.storages.get(&hashed_address) {
            // Check specific slot
            if let Some(value) = account_storage.storage.get(&hashed_slot) {
                return Ok(Some(*value));
            }
            // If the whole storage was wiped in the overlay (e.g. reused address), we return 0
            if account_storage.wiped {
                return Ok(Some(StorageValue::ZERO));
            }
        }

        // Check if account was destroyed in the overlay (implicit storage wipe)
        if let Some(account) = state.accounts.get(&hashed_address) &&
            account.is_none()
        {
            return Ok(Some(StorageValue::ZERO));
        }

        self.inner.storage(address, storage_key)
    }
}

impl<'a, P> BytecodeReader for MemoryOverlayOpProofsStateProviderRef<'a, P>
where
    P: OpProofsProviderRO + Clone,
{
    fn bytecode_by_hash(&self, code_hash: &B256) -> ProviderResult<Option<Bytecode>> {
        // HashedPostStateSorted does not store bytecode, so we cannot look it up in the overlay.
        // We fallback strictly to the inner provider.
        self.inner.bytecode_by_hash(code_hash)
    }
}

impl<'a, P> StateRootProvider for MemoryOverlayOpProofsStateProviderRef<'a, P>
where
    P: OpProofsProviderRO + Clone,
{
    fn state_root(&self, state: HashedPostState) -> ProviderResult<B256> {
        self.state_root_from_nodes(TrieInput::from_state(state))
    }

    fn state_root_from_nodes(&self, mut input: TrieInput) -> ProviderResult<B256> {
        // Combine updates from the buffer (overlay) with the current input
        input.prepend_self(self.trie_input().clone());

        // Delegate to inner to calculate root against disk + overlay
        self.inner.state_root_from_nodes(input)
    }

    fn state_root_with_updates(
        &self,
        state: HashedPostState,
    ) -> ProviderResult<(B256, TrieUpdates)> {
        self.state_root_from_nodes_with_updates(TrieInput::from_state(state))
    }

    fn state_root_from_nodes_with_updates(
        &self,
        mut input: TrieInput,
    ) -> ProviderResult<(B256, TrieUpdates)> {
        input.prepend_self(self.trie_input().clone());
        self.inner.state_root_from_nodes_with_updates(input)
    }
}

impl<'a, P> StorageRootProvider for MemoryOverlayOpProofsStateProviderRef<'a, P>
where
    P: OpProofsProviderRO + Clone,
{
    fn storage_root(
        &self,
        address: Address,
        hashed_storage: HashedStorage,
    ) -> ProviderResult<B256> {
        let merged = self.merged_hashed_storage(address, hashed_storage);
        self.inner.storage_root(address, merged)
    }

    fn storage_proof(
        &self,
        address: Address,
        slot: B256,
        hashed_storage: HashedStorage,
    ) -> ProviderResult<reth_trie::StorageProof> {
        let merged = self.merged_hashed_storage(address, hashed_storage);
        self.inner.storage_proof(address, slot, merged)
    }

    fn storage_multiproof(
        &self,
        address: Address,
        slots: &[B256],
        hashed_storage: HashedStorage,
    ) -> ProviderResult<StorageMultiProof> {
        let merged = self.merged_hashed_storage(address, hashed_storage);
        self.inner.storage_multiproof(address, slots, merged)
    }
}

impl<'a, P> StateProofProvider for MemoryOverlayOpProofsStateProviderRef<'a, P>
where
    P: OpProofsProviderRO + Clone,
{
    fn proof(
        &self,
        mut input: TrieInput,
        address: Address,
        slots: &[B256],
    ) -> ProviderResult<AccountProof> {
        input.prepend_self(self.trie_input().clone());
        self.inner.proof(input, address, slots)
    }

    fn multiproof(
        &self,
        mut input: TrieInput,
        targets: MultiProofTargets,
    ) -> ProviderResult<MultiProof> {
        input.prepend_self(self.trie_input().clone());
        self.inner.multiproof(input, targets)
    }

    fn witness(
        &self,
        mut input: TrieInput,
        target: HashedPostState,
        mode: ExecutionWitnessMode,
    ) -> ProviderResult<Vec<Bytes>> {
        input.prepend_self(self.trie_input().clone());
        self.inner.witness(input, target, mode)
    }
}

impl<'a, P> BlockHashReader for MemoryOverlayOpProofsStateProviderRef<'a, P>
where
    P: OpProofsProviderRO,
{
    fn block_hash(&self, number: BlockNumber) -> ProviderResult<Option<B256>> {
        // Iterate backwards (Newest to Oldest) to find most recent definition
        for state in self.memory.iter().rev() {
            if state.0.block.number == number {
                return Ok(Some(state.0.block.hash));
            }
        }
        self.inner.block_hash(number)
    }

    fn canonical_hashes_range(
        &self,
        start: BlockNumber,
        end: BlockNumber,
    ) -> ProviderResult<Vec<B256>> {
        let mut hashes = self.inner.canonical_hashes_range(start, end)?;

        // Overlay with in-memory blocks in [start, end).
        let blocks_in_range = self
            .memory
            .iter()
            .map(|state| (state.0.block.number, state.0.block.hash))
            .filter(|(num, _)| *num >= start && *num < end);

        for (num, block_hash) in blocks_in_range {
            let idx = (num - start) as usize;
            match idx.cmp(&hashes.len()) {
                std::cmp::Ordering::Less => hashes[idx] = block_hash,
                std::cmp::Ordering::Equal => hashes.push(block_hash),
                // Gap in the requested range: disk + in-memory view is not contiguous.
                // This should never happen: the buffer must be contiguous with disk.
                std::cmp::Ordering::Greater => panic!(
                    "canonical_hashes_range: gap detected at block {num} (index {idx}, \
                     current len {}); disk and in-memory buffer are not contiguous",
                    hashes.len()
                ),
            }
        }
        Ok(hashes)
    }
}

impl<'a, P> HashedPostStateProvider for MemoryOverlayOpProofsStateProviderRef<'a, P>
where
    P: OpProofsProviderRO + Clone,
{
    fn hashed_post_state(&self, bundle_state: &BundleState) -> HashedPostState {
        self.inner.hashed_post_state(bundle_state)
    }
}
