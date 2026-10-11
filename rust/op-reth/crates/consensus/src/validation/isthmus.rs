//! Block verification w.r.t. consensus rules new in Isthmus hardfork.

use crate::OpConsensusError;
use alloy_consensus::BlockHeader;
use alloy_primitives::B256;
use alloy_trie::EMPTY_ROOT_HASH;
use reth_optimism_primitives::L2_TO_L1_MESSAGE_PASSER_ADDRESS;
use reth_storage_api::{StorageRootProvider, errors::ProviderResult};
use reth_trie_common::HashedStorage;
use revm::database::BundleState;
use tracing::warn;

/// Verifies that `withdrawals_root` (i.e. `l2tol1-msg-passer` storage root since Isthmus) field is
/// set in block header.
pub fn ensure_withdrawals_storage_root_is_some<H: BlockHeader>(
    header: H,
) -> Result<(), OpConsensusError> {
    header.withdrawals_root().ok_or(OpConsensusError::L2WithdrawalsRootMissing)?;

    Ok(())
}

/// Computes the storage root of predeploy `L2ToL1MessagePasser.sol`.
///
/// Uses state updates from block execution. See also [`withdrawals_root_prehashed`].
pub fn withdrawals_root<DB: StorageRootProvider>(
    state_updates: &BundleState,
    state: DB,
) -> ProviderResult<B256> {
    // if l2 withdrawals transactions were executed there will be storage updates for
    // `L2ToL1MessagePasser.sol` predeploy
    withdrawals_root_prehashed(
        state_updates
            .state()
            .get(&L2_TO_L1_MESSAGE_PASSER_ADDRESS)
            .map(|acc| {
                // The MessagePasser predeploy cannot be destroyed, so its storage is never wiped.
                HashedStorage::from_plain_storage(
                    acc.storage.iter().map(|(slot, value)| (slot, &value.present_value)),
                )
            })
            .unwrap_or_default(),
        state,
    )
}

/// Computes the storage root of predeploy `L2ToL1MessagePasser.sol`.
///
/// Uses pre-hashed storage updates of `L2ToL1MessagePasser.sol` predeploy, resulting from
/// execution of L2 withdrawals transactions. If none, takes empty [`HashedStorage::default`].
pub fn withdrawals_root_prehashed<DB: StorageRootProvider>(
    hashed_storage_updates: HashedStorage,
    state: DB,
) -> ProviderResult<B256> {
    state.storage_root(L2_TO_L1_MESSAGE_PASSER_ADDRESS, hashed_storage_updates)
}

/// Verifies block header field `withdrawals_root` against storage root of
/// `L2ToL1MessagePasser.sol` predeploy post block execution.
///
/// Takes state updates resulting from execution of block.
///
/// See <https://specs.optimism.io/protocol/isthmus/exec-engine.html#l2tol1messagepasser-storage-root-in-header>.
pub fn verify_withdrawals_root<DB, H>(
    state_updates: &BundleState,
    state: DB,
    header: H,
) -> Result<(), OpConsensusError>
where
    DB: StorageRootProvider,
    H: BlockHeader,
{
    let header_storage_root =
        header.withdrawals_root().ok_or(OpConsensusError::L2WithdrawalsRootMissing)?;

    let storage_root = withdrawals_root(state_updates, state)
        .map_err(OpConsensusError::L2WithdrawalsRootCalculationFail)?;

    if storage_root == EMPTY_ROOT_HASH {
        // if there was no MessagePasser contract storage, something is wrong
        // (it should at least store an implementation address and owner address)
        warn!("isthmus: no storage root for L2ToL1MessagePasser contract");
    }

    if header_storage_root != storage_root {
        return Err(OpConsensusError::L2WithdrawalsRootMismatch {
            header: header_storage_root,
            exec_res: storage_root,
        });
    }

    Ok(())
}

/// Verifies block header field `withdrawals_root` against storage root of
/// `L2ToL1MessagePasser.sol` predeploy post block execution.
///
/// Takes pre-hashed storage updates of `L2ToL1MessagePasser.sol` predeploy, resulting from
/// execution of block, if any. Otherwise takes empty [`HashedStorage::default`].
///
/// See <https://specs.optimism.io/protocol/isthmus/exec-engine.html#l2tol1messagepasser-storage-root-in-header>.
pub fn verify_withdrawals_root_prehashed<DB, H>(
    hashed_storage_updates: HashedStorage,
    state: DB,
    header: H,
) -> Result<(), OpConsensusError>
where
    DB: StorageRootProvider,
    H: BlockHeader,
{
    let header_storage_root =
        header.withdrawals_root().ok_or(OpConsensusError::L2WithdrawalsRootMissing)?;

    let storage_root = withdrawals_root_prehashed(hashed_storage_updates, state)
        .map_err(OpConsensusError::L2WithdrawalsRootCalculationFail)?;

    if header_storage_root != storage_root {
        return Err(OpConsensusError::L2WithdrawalsRootMismatch {
            header: header_storage_root,
            exec_res: storage_root,
        });
    }

    Ok(())
}

#[cfg(test)]
mod test {
    use super::*;
    use alloc::{collections::BTreeMap, sync::Arc};
    use alloy_chains::Chain;
    use alloy_consensus::Header;
    use alloy_primitives::{B256, U256, keccak256};
    use core::str::FromStr;
    use reth_chain_state::{ExecutedBlock, test_utils::TestBlockBuilder};
    use reth_db_common::init::init_genesis;
    use reth_optimism_chainspec::OpChainSpecBuilder;
    use reth_optimism_node::OpNode;
    use reth_primitives_traits::Account;
    use reth_provider::{
        BlockWriter, LatestStateProviderRef, StateWriter, StorageSettings, StorageSettingsCache,
        TrieWriter,
        providers::BlockchainProvider,
        test_utils::{create_test_provider_factory, create_test_provider_factory_with_node_types},
    };
    use reth_revm::db::BundleState;
    use reth_stages_types::{FinishCheckpoint, StageCheckpoint, StageId};
    use reth_storage_api::{
        DatabaseProviderROFactory, StageCheckpointWriter, StateProviderBox, StateProviderFactory,
        StateRootProvider,
    };
    use reth_storage_overlay::{OverlayManager, OverlayStateProviderFactory};
    use reth_trie::{HashedStorage, test_utils::storage_root_prehashed};
    use reth_trie_common::{
        ComputedTrieData, HashedPostState, HashedPostStateSorted,
        updates::{StorageTrieUpdatesSorted, TrieUpdatesSorted},
    };

    #[test]
    fn l2tol1_message_passer_no_withdrawals() {
        let hashed_address = keccak256(L2_TO_L1_MESSAGE_PASSER_ADDRESS);

        // create account storage
        let init_storage = HashedStorage::from_iter(
            [
                "50000000000000000000000000000004253371b55351a08cb3267d4d265530b6",
                "512428ed685fff57294d1a9cbb147b18ae5db9cf6ae4b312fa1946ba0561882e",
                "51e6784c736ef8548f856909870b38e49ef7a4e3e77e5e945e0d5e6fcaa3037f",
            ]
            .into_iter()
            .map(|str| (B256::from_str(str).unwrap(), U256::from(1))),
        );
        let mut state = HashedPostState::default();
        state.storages.insert(hashed_address, init_storage.clone());

        // init test db
        // note: must be empty (default) chain spec to ensure storage is empty after init genesis,
        // otherwise can't use `storage_root_prehashed` to determine storage root later
        let provider_factory = create_test_provider_factory_with_node_types::<OpNode>(Arc::new(
            OpChainSpecBuilder::default().chain(Chain::dev()).genesis(Default::default()).build(),
        ));
        let _ = init_genesis(&provider_factory).unwrap();

        // write account storage to database
        let provider_rw = provider_factory.provider_rw().unwrap();
        provider_rw.write_hashed_state(&state.clone().into_sorted()).unwrap();
        provider_rw.commit().unwrap();

        // create block header with withdrawals root set to storage root of l2tol1-msg-passer
        let header = Header {
            withdrawals_root: Some(storage_root_prehashed(init_storage.storage)),
            ..Default::default()
        };

        // create state provider factory
        let state_provider_factory = BlockchainProvider::new(provider_factory).unwrap();

        // validate block against existing state by passing empty state updates
        verify_withdrawals_root(
            &BundleState::default(),
            state_provider_factory.latest().expect("load state"),
            &header,
        )
        .unwrap();
    }

    /// Hashed slot with the given first two bytes, so that the storage trie has stored branch
    /// nodes at depth one that cache the hashes of their children.
    fn hashed_slot(first: u8, second: u8) -> B256 {
        let mut slot = keccak256([first, second]);
        slot.0[0] = first;
        slot.0[1] = second;
        slot
    }

    /// Writes hashed state and trie updates the way `save_blocks` does under partial persistence:
    /// updates of `batch` that are overwritten by the in-memory `mask` blocks are omitted.
    fn write_state_trie<P: StateWriter + TrieWriter>(
        provider: &P,
        batch: &[ExecutedBlock],
        mask: &[ExecutedBlock],
    ) {
        let hashed_state = HashedPostStateSorted::disjointed_merge_batch(
            &ExecutedBlock::hashed_state_refs(batch),
            &ExecutedBlock::hashed_state_refs(mask),
        );
        provider.write_hashed_state(&hashed_state).unwrap();
        let trie_updates = TrieUpdatesSorted::disjointed_merge_batch(
            &ExecutedBlock::trie_updates_refs(batch),
            &ExecutedBlock::trie_updates_refs(mask),
        );
        provider.write_trie_updates_sorted(&trie_updates).unwrap();
    }

    /// The withdrawals root is computed on a state provider whose database trie tables are a
    /// partial persistence snapshot (`Finish.partial_state_trie` behind `Finish`), with the masking
    /// suffix and further unpersisted blocks only held in memory. It must equal the storage root of
    /// the complete post-block storage of the `L2ToL1MessagePasser`.
    #[rstest::rstest]
    fn l2tol1_message_passer_root_over_masked_state_trie(
        #[values(StorageSettings::v1(), StorageSettings::v2())] settings: StorageSettings,
        #[values(true, false)] masked: bool,
    ) {
        let hashed_address = keccak256(L2_TO_L1_MESSAGE_PASSER_ADDRESS);

        // Storage changes per block number. Block 1 is fully persisted, blocks 2..=4 are persisted
        // below the partial state trie frontier, blocks 5..=6 are the masking suffix (persisted
        // except for hashed state and trie) and block 7 is only in memory. The masking suffix
        // changes siblings of slots changed in blocks 2..=4, so the database keeps branch nodes
        // whose cached child hashes predate blocks 2..=4.
        let base = (0..=u8::MAX)
            .flat_map(|first| {
                [0x00, 0x80].map(|second| (hashed_slot(first, second), U256::from(1)))
            })
            .collect::<Vec<_>>();
        let changes: [Vec<(B256, U256)>; 8] = [
            vec![],
            base,
            vec![
                (hashed_slot(0x10, 0x00), U256::from(2)),
                (hashed_slot(0x11, 0x42), U256::from(2)),
                (hashed_slot(0x20, 0x00), U256::ZERO),
            ],
            vec![(hashed_slot(0x30, 0x00), U256::from(3))],
            vec![(hashed_slot(0x40, 0x80), U256::from(4))],
            vec![
                (hashed_slot(0x30, 0x00), U256::from(5)),
                (hashed_slot(0x31, 0x00), U256::from(5)),
                (hashed_slot(0x50, 0x00), U256::from(5)),
                (hashed_slot(0x60, 0x80), U256::ZERO),
            ],
            vec![
                (hashed_slot(0x1f, 0x80), U256::ZERO),
                (hashed_slot(0x50, 0x00), U256::from(6)),
                (hashed_slot(0x70, 0x00), U256::from(6)),
            ],
            vec![(hashed_slot(0x50, 0x00), U256::from(7)), (hashed_slot(0x80, 0x00), U256::ZERO)],
        ];
        // Storage updates of the block whose withdrawals root is computed, on top of block 7.
        let block_updates = vec![
            (hashed_slot(0x90, 0x42), U256::from(1)),
            (hashed_slot(0xa0, 0x00), U256::from(8)),
            (hashed_slot(0xb0, 0x80), U256::ZERO),
        ];

        // Per-block hashed state and trie updates, computed on a fully persisted reference
        // database. Like the engine's trie updates, a block's storage trie updates only contain
        // the nodes it modified.
        let reference = create_test_provider_factory();
        reference.set_storage_settings_cache(settings);
        let mut storage_nodes = BTreeMap::new();
        let blocks = TestBlockBuilder::eth()
            .get_executed_blocks(0..changes.len() as u64)
            .zip(&changes)
            .map(|(block, changes)| {
                let mut hashed_state = HashedPostState::default().with_storages([(
                    hashed_address,
                    HashedStorage::from_iter(changes.iter().copied()),
                )]);
                if block.recovered_block().number == 1 {
                    hashed_state =
                        hashed_state.with_accounts([(hashed_address, Some(Account::default()))]);
                }
                let provider = reference.provider().unwrap();
                let (_, trie_updates) = LatestStateProviderRef::new(&provider)
                    .state_root_with_updates(hashed_state.clone())
                    .unwrap();
                drop(provider);
                let hashed_state = hashed_state.into_sorted();
                let trie_updates = trie_updates.into_sorted();
                let provider_rw = reference.provider_rw().unwrap();
                provider_rw.write_hashed_state(&hashed_state).unwrap();
                provider_rw.write_trie_updates_sorted(&trie_updates).unwrap();
                provider_rw.commit().unwrap();

                let modified_storage_nodes = trie_updates
                    .storage_tries_ref()
                    .get(&hashed_address)
                    .map(|storage_trie| storage_trie.storage_nodes_ref())
                    .unwrap_or_default()
                    .iter()
                    .filter(|(path, node)| match node {
                        Some(node) => {
                            storage_nodes.insert(*path, node.clone()).as_ref() != Some(node)
                        }
                        None => storage_nodes.remove(path).is_some(),
                    })
                    .cloned()
                    .collect();
                let trie_updates = TrieUpdatesSorted::new(
                    trie_updates.account_nodes_ref().to_vec(),
                    core::iter::once((
                        hashed_address,
                        StorageTrieUpdatesSorted { storage_nodes: modified_storage_nodes },
                    ))
                    .collect(),
                );

                ExecutedBlock::new(
                    Arc::clone(&block.recovered_block),
                    Arc::clone(&block.execution_output),
                    ComputedTrieData::new(Arc::new(hashed_state), Arc::new(trie_updates)),
                )
            })
            .collect::<Vec<_>>();

        let factory = create_test_provider_factory();
        factory.set_storage_settings_cache(settings);
        let provider_rw = factory.provider_rw().unwrap();
        for block in &blocks[..=6] {
            provider_rw.insert_block(block.recovered_block()).unwrap();
        }
        write_state_trie(&*provider_rw, &blocks[1..=1], &[]);
        let partial_state_trie = if masked {
            write_state_trie(&*provider_rw, &blocks[2..=4], &blocks[5..=6]);
            4
        } else {
            write_state_trie(&*provider_rw, &blocks[2..=6], &[]);
            6
        };
        provider_rw
            .save_stage_checkpoint(
                StageId::Finish,
                StageCheckpoint::new(6).with_finish_stage_checkpoint(FinishCheckpoint {
                    partial_state_trie: Some(partial_state_trie),
                }),
            )
            .unwrap();
        provider_rw.commit().unwrap();

        let overlay_manager = OverlayManager::default();
        for block in &blocks[5..=7] {
            overlay_manager.insert_block(block.clone());
        }
        let state_provider_factory = OverlayStateProviderFactory::new(
            factory,
            overlay_manager.overlay_builder(blocks[7].recovered_block().hash()),
        );
        let state: StateProviderBox =
            Box::new(state_provider_factory.database_provider_ro().unwrap());

        let mut expected_storage = BTreeMap::new();
        for (slot, value) in changes.iter().flatten().chain(&block_updates) {
            expected_storage.insert(*slot, *value);
        }
        expected_storage.retain(|_, value| !value.is_zero());

        let withdrawals_root = withdrawals_root_prehashed(
            HashedStorage::from_iter(block_updates.iter().copied()),
            state,
        )
        .unwrap();
        assert_eq!(withdrawals_root, storage_root_prehashed(expected_storage));
    }
}
