//! State overlay for speculative flashblocks anchored to canonical history.

use alloy_consensus::BlockHeader;
use alloy_primitives::{
    Address, B256, BlockNumber, Bytes, StorageKey, StorageValue, U256, keccak256,
};
use reth_chain_state::ExecutedBlock;
use reth_errors::ProviderResult;
use reth_primitives_traits::{Account, Bytecode, NodePrimitives};
use reth_storage_api::{
    AccountReader, BlockHashReader, BytecodeReader, HashedPostStateProvider, StateProofProvider,
    StateProvider, StateProviderBox, StateRootProvider, StorageRootProvider,
};
use reth_trie_common::{
    AccountProof, DecodedMultiProofV2, HashedPostState, HashedStorage, MultiProof,
    MultiProofTargets, MultiProofTargetsV2, StorageMultiProof, TrieInput, updates::TrieUpdates,
};
use revm::database::BundleState;
use std::{fmt, sync::OnceLock};

/// Overlays executed flashblock state on its canonical anchor.
///
/// UPSTREAM-MIRROR(port): reth@pre-26923 `reth_chain_state::MemoryOverlayStateProviderRef`
///
/// Ported from the provider removed in paradigmxyz/reth#26923. The replacement
/// `StateProviderFactory::state_with_block_appended` requires the appended block's parent to
/// equal the anchor. Speculative flashblocks instead carry accumulated state rooted at a
/// canonical anchor that can differ from their immediate, speculative parent.
/// Only the owned storage of the original reference provider is needed here.
pub(super) struct PendingStateProvider<N: NodePrimitives> {
    historical: StateProviderBox,
    /// Executed blocks, newest to oldest.
    in_memory: Vec<ExecutedBlock<N>>,
    trie_input: OnceLock<TrieInput>,
}

impl<N: NodePrimitives> fmt::Debug for PendingStateProvider<N> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("PendingStateProvider")
            .field("in_memory", &self.in_memory)
            .finish_non_exhaustive()
    }
}

impl<N: NodePrimitives> PendingStateProvider<N> {
    pub(super) const fn new(
        historical: StateProviderBox,
        in_memory: Vec<ExecutedBlock<N>>,
    ) -> Self {
        Self { historical, in_memory, trie_input: OnceLock::new() }
    }

    fn trie_input(&self) -> &TrieInput {
        self.trie_input.get_or_init(|| {
            let mut input = TrieInput::default();
            for block in self.in_memory.iter().rev() {
                let data = block.trie_data();
                input.nodes.extend_from_sorted(&data.sorted.trie_updates);
                input.state.extend_from_sorted(&data.sorted.hashed_state);
            }
            input
        })
    }

    fn merged_hashed_storage(&self, address: Address, storage: HashedStorage) -> HashedStorage {
        let state = &self.trie_input().state;
        let mut hashed = state.storages.get(&keccak256(address)).cloned().unwrap_or_default();
        hashed.extend(&storage);
        hashed
    }
}

impl<N: NodePrimitives> BlockHashReader for PendingStateProvider<N> {
    fn block_hash(&self, number: BlockNumber) -> ProviderResult<Option<B256>> {
        for block in &self.in_memory {
            if block.recovered_block().number() == number {
                return Ok(Some(block.recovered_block().hash()));
            }
        }
        self.historical.block_hash(number)
    }

    fn canonical_hashes_range(
        &self,
        start: BlockNumber,
        end: BlockNumber,
    ) -> ProviderResult<Vec<B256>> {
        let range = start..end;
        let mut earliest_block_number = None;
        let mut in_memory_hashes = Vec::with_capacity(range.size_hint().0);
        for block in &self.in_memory {
            let block_num = block.recovered_block().number();
            if range.contains(&block_num) {
                in_memory_hashes.push(block.recovered_block().hash());
                earliest_block_number = Some(block_num);
            }
        }
        // The overlay is newest to oldest; return hashes in ascending block order.
        in_memory_hashes.reverse();
        let mut hashes =
            self.historical.canonical_hashes_range(start, earliest_block_number.unwrap_or(end))?;
        hashes.append(&mut in_memory_hashes);
        Ok(hashes)
    }
}

impl<N: NodePrimitives> AccountReader for PendingStateProvider<N> {
    fn basic_account(&self, address: &Address) -> ProviderResult<Option<Account>> {
        for block in &self.in_memory {
            if let Some(account) = block.execution_output.account(address) {
                return Ok(account);
            }
        }
        self.historical.basic_account(address)
    }
}

impl<N: NodePrimitives> StateRootProvider for PendingStateProvider<N> {
    fn state_root(&self, state: HashedPostState) -> ProviderResult<B256> {
        self.state_root_from_nodes(TrieInput::from_state(state))
    }

    fn state_root_from_nodes(&self, mut input: TrieInput) -> ProviderResult<B256> {
        input.prepend_self(self.trie_input().clone());
        self.historical.state_root_from_nodes(input)
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
        self.historical.state_root_from_nodes_with_updates(input)
    }
}

impl<N: NodePrimitives> StorageRootProvider for PendingStateProvider<N> {
    fn storage_root(&self, address: Address, storage: HashedStorage) -> ProviderResult<B256> {
        let merged = self.merged_hashed_storage(address, storage);
        self.historical.storage_root(address, merged)
    }

    fn storage_proof(
        &self,
        address: Address,
        slot: B256,
        storage: HashedStorage,
    ) -> ProviderResult<reth_trie_common::StorageProof> {
        let merged = self.merged_hashed_storage(address, storage);
        self.historical.storage_proof(address, slot, merged)
    }

    fn storage_multiproof(
        &self,
        address: Address,
        slots: &[B256],
        storage: HashedStorage,
    ) -> ProviderResult<StorageMultiProof> {
        let merged = self.merged_hashed_storage(address, storage);
        self.historical.storage_multiproof(address, slots, merged)
    }
}

impl<N: NodePrimitives> StateProofProvider for PendingStateProvider<N> {
    fn proof(
        &self,
        mut input: TrieInput,
        address: Address,
        slots: &[B256],
    ) -> ProviderResult<AccountProof> {
        input.prepend_self(self.trie_input().clone());
        self.historical.proof(input, address, slots)
    }

    fn multiproof(
        &self,
        mut input: TrieInput,
        targets: MultiProofTargets,
    ) -> ProviderResult<MultiProof> {
        input.prepend_self(self.trie_input().clone());
        self.historical.multiproof(input, targets)
    }

    fn multiproof_v2(
        &self,
        mut input: TrieInput,
        targets: MultiProofTargetsV2,
    ) -> ProviderResult<DecodedMultiProofV2> {
        input.prepend_self(self.trie_input().clone());
        self.historical.multiproof_v2(input, targets)
    }

    fn witness(
        &self,
        mut input: TrieInput,
        target: HashedPostState,
        mode: reth_trie_common::ExecutionWitnessMode,
    ) -> ProviderResult<Vec<Bytes>> {
        input.prepend_self(self.trie_input().clone());
        self.historical.witness(input, target, mode)
    }
}

impl<N: NodePrimitives> HashedPostStateProvider for PendingStateProvider<N> {
    fn hashed_post_state(&self, bundle_state: &BundleState) -> ProviderResult<HashedPostState> {
        let mut hashed_state = self.historical.hashed_post_state(bundle_state)?;
        for (address, account) in bundle_state.state() {
            if !account.was_destroyed() || account.original_info.is_none() {
                continue;
            }
            let hashed_address = keccak256(address);
            let Some(parent_storage) = self.trie_input().state.storages.get(&hashed_address) else {
                continue;
            };
            let storage = &mut hashed_state.storages.entry(hashed_address).or_default().storage;
            for hashed_slot in parent_storage.storage.keys() {
                storage.entry(*hashed_slot).or_insert(U256::ZERO);
            }
        }
        Ok(hashed_state)
    }
}

impl<N: NodePrimitives> StateProvider for PendingStateProvider<N> {
    fn storage(
        &self,
        address: Address,
        storage_key: StorageKey,
    ) -> ProviderResult<Option<StorageValue>> {
        for block in &self.in_memory {
            if let Some(value) = block.execution_output.storage(&address, storage_key.into()) {
                return Ok(Some(value));
            }
        }
        self.historical.storage(address, storage_key)
    }
}

impl<N: NodePrimitives> BytecodeReader for PendingStateProvider<N> {
    fn bytecode_by_hash(&self, code_hash: &B256) -> ProviderResult<Option<Bytecode>> {
        for block in &self.in_memory {
            if let Some(contract) = block.execution_output.bytecode(code_hash) {
                return Ok(Some(contract));
            }
        }
        self.historical.bytecode_by_hash(code_hash)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::{Block, Header};
    use reth_optimism_primitives::OpPrimitives;
    use reth_primitives_traits::RecoveredBlock;
    use reth_revm::test_utils::StateProviderTest;
    use reth_storage_api::noop::NoopProvider;
    use revm::{
        database::{AccountStatus, BundleAccount, states::StorageSlot},
        state::AccountInfo,
    };
    use std::sync::Arc;

    #[test]
    fn speculative_state_overlays_canonical_history() {
        let changed = Address::with_last_byte(1);
        let untouched = Address::with_last_byte(2);
        let mut historical = StateProviderTest::default();
        historical.insert_account(
            changed,
            Account { balance: U256::from(10), ..Default::default() },
            None,
            std::iter::once((B256::ZERO, U256::from(1))).collect(),
        );
        historical.insert_account(
            untouched,
            Account { balance: U256::from(30), ..Default::default() },
            None,
            std::iter::once((B256::ZERO, U256::from(3))).collect(),
        );

        // The pending block's immediate parent is speculative, not the canonical anchor
        // whose provider is supplied above. The accumulated bundle must still overlay it.
        let mut block = ExecutedBlock::<OpPrimitives> {
            recovered_block: Arc::new(RecoveredBlock::new_unhashed(
                Block {
                    header: Header {
                        parent_hash: B256::repeat_byte(0x11),
                        number: 2,
                        ..Default::default()
                    },
                    body: Default::default(),
                },
                vec![],
            )),
            ..Default::default()
        };
        Arc::make_mut(&mut block.execution_output).state.state.insert(
            changed,
            BundleAccount::new(
                Some(AccountInfo { balance: U256::from(10), ..Default::default() }),
                Some(AccountInfo { balance: U256::from(20), ..Default::default() }),
                std::iter::once((
                    U256::ZERO,
                    StorageSlot::new_changed(U256::from(1), U256::from(2)),
                ))
                .collect(),
                AccountStatus::Changed,
            ),
        );
        let provider = PendingStateProvider::new(Box::new(historical), vec![block]);
        assert_eq!(provider.account_balance(&changed).unwrap(), Some(U256::from(20)));
        assert_eq!(provider.storage(changed, B256::ZERO).unwrap(), Some(U256::from(2)));
        assert_eq!(provider.account_balance(&untouched).unwrap(), Some(U256::from(30)));
        assert_eq!(provider.storage(untouched, B256::ZERO).unwrap(), Some(U256::from(3)));
    }

    #[test]
    fn destroyed_account_zeroes_overlay_slots_without_overwriting_recreated_values() {
        let address = Address::with_last_byte(1);
        let untouched = Address::with_last_byte(2);
        let old_slot = keccak256(B256::from(U256::from(1)));
        let recreated_slot = keccak256(B256::from(U256::from(2)));
        let provider = PendingStateProvider::<OpPrimitives>::new(
            Box::new(StateProviderTest::default()),
            Vec::new(),
        );
        let mut overlay = HashedPostState::default();
        overlay.storages.insert(
            keccak256(address),
            HashedStorage::from_iter([
                (old_slot, U256::from(11)),
                (recreated_slot, U256::from(22)),
            ]),
        );
        overlay
            .storages
            .insert(keccak256(untouched), HashedStorage::from_iter([(old_slot, U256::from(33))]));
        provider.trie_input.set(TrieInput::from_state(overlay)).unwrap();
        let mut bundle = BundleState::default();
        bundle.state.insert(
            address,
            BundleAccount::new(
                Some(AccountInfo::default()),
                Some(AccountInfo::default()),
                std::iter::once((
                    U256::from(2),
                    StorageSlot::new_changed(U256::from(22), U256::from(99)),
                ))
                .collect(),
                AccountStatus::DestroyedChanged,
            ),
        );
        let state = provider.hashed_post_state(&bundle).unwrap();
        let storage = &state.storages[&keccak256(address)].storage;
        assert_eq!(storage[&old_slot], U256::ZERO);
        assert_eq!(storage[&recreated_slot], U256::from(99));
        assert!(!state.storages.contains_key(&keccak256(untouched)));
    }

    #[test]
    fn created_and_destroyed_account_skips_trie_aggregation() {
        let address = Address::with_last_byte(1);
        let provider = PendingStateProvider::<OpPrimitives>::new(
            Box::new(NoopProvider::default()),
            Vec::new(),
        );
        let mut bundle_state = BundleState::default();
        bundle_state.state.insert(
            address,
            BundleAccount::new(None, None, Default::default(), AccountStatus::Destroyed),
        );
        provider.hashed_post_state(&bundle_state).unwrap();
        assert!(provider.trie_input.get().is_none());
    }
}
