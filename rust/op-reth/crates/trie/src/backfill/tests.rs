//! Integration tests for [`BackfillJob`].
//!
//! Chain-construction helpers live in [`crate::test_utils`] and are shared
//! with the snapshot tests.

use super::{BackfillError, BackfillJob};
use crate::{
    BlockStateDiff, OpProofsStorageError, OpProofsStore, RethTrieStorageLayout,
    api::{OpProofsProviderRO, OpProofsProviderRw},
    initialize::InitializationJob,
    proof::DatabaseStateRoot,
    test_utils::{
        build_chain_and_initialize_storage, build_chain_with_storage_writes_and_initialize_storage,
        build_transfer_block, chain_spec_with_address, commit_block_to_database, create_storage,
        deterministic_keypair, execute_block, public_key_to_address,
    },
};
use alloy_consensus::BlockHeader;
use alloy_eips::{NumHash, eip1898::BlockWithParent};
use alloy_primitives::{Address, B256};
use reth_db::Database;
use reth_db_common::init::init_genesis;
use reth_evm::{ConfigureEvm, execute::Executor};
use reth_evm_ethereum::EthEvmConfig;
use reth_provider::{
    DatabaseProviderFactory, HashedPostStateProvider, LatestStateProviderRef, StateRootProvider,
    StorageSettingsCache, test_utils::create_test_provider_factory_with_chain_spec,
};
use reth_revm::database::StateProviderDatabase;
use reth_trie::{HashedPostState, StateRoot};

// ============================ Tests ============================

#[test]
fn run_is_noop_when_target_at_or_above_earliest() {
    // Build a chain of 3 blocks; storage initialized at block 3 (earliest = 3).
    let (provider_factory, storage, latest_num, latest_hash) =
        build_chain_and_initialize_storage(3);

    // target == earliest: no-op.
    {
        let provider = provider_factory.database_provider_ro().unwrap();
        BackfillJob::new(provider, storage.clone()).run(latest_num).unwrap();
        let ro = storage.provider_ro().unwrap();
        assert_eq!(ro.get_earliest_block().unwrap(), NumHash::new(latest_num, latest_hash));
    }

    // target > earliest: also no-op.
    {
        let provider = provider_factory.database_provider_ro().unwrap();
        BackfillJob::new(provider, storage.clone()).run(latest_num + 100).unwrap();
        let ro = storage.provider_ro().unwrap();
        assert_eq!(ro.get_earliest_block().unwrap(), NumHash::new(latest_num, latest_hash));
    }
}

#[test]
fn run_errors_when_storage_uninitialized() {
    let key_pair = deterministic_keypair();
    let chain_spec = chain_spec_with_address(public_key_to_address(key_pair.public_key()));
    let provider_factory = create_test_provider_factory_with_chain_spec(chain_spec);
    init_genesis(&provider_factory).unwrap();

    // Storage created but never initialized — no earliest marker.
    let storage = create_storage();
    let provider = provider_factory.database_provider_ro().unwrap();
    let err = BackfillJob::new(provider, storage).run(0).unwrap_err();
    assert!(
        matches!(err, BackfillError::Storage(OpProofsStorageError::NoBlocksFound)),
        "expected NoBlocksFound, got {err:?}"
    );
}

#[test]
fn run_extends_window_backward_multi_block() {
    // 5-block chain — exercises descending iteration across multiple
    // `BackfillContext::step` calls.
    let (provider_factory, storage, latest_num, latest_hash) =
        build_chain_and_initialize_storage(5);

    {
        let ro = storage.provider_ro().unwrap();
        assert_eq!(ro.get_earliest_block().unwrap(), NumHash::new(latest_num, latest_hash));
    }

    {
        let provider = provider_factory.database_provider_ro().unwrap();
        BackfillJob::new(provider, storage.clone()).run(0).unwrap();
    }

    let provider = provider_factory.database_provider_ro().unwrap();
    let genesis_hash = reth_provider::BlockHashReader::block_hash(&provider, 0).unwrap().unwrap();
    let ro = storage.provider_ro().unwrap();
    assert_eq!(ro.get_earliest_block().unwrap(), NumHash::new(0, genesis_hash));
}

#[test]
fn run_extends_window_backward() {
    // Smallest possible case: 1-block chain, single backfill step from 1 → 0.
    let (provider_factory, storage, latest_num, latest_hash) =
        build_chain_and_initialize_storage(1);

    // Sanity: earliest starts at the latest block.
    {
        let ro = storage.provider_ro().unwrap();
        assert_eq!(ro.get_earliest_block().unwrap(), NumHash::new(latest_num, latest_hash));
    }

    // Backfill all the way down to block 0 (genesis).
    {
        let provider = provider_factory.database_provider_ro().unwrap();
        BackfillJob::new(provider, storage.clone()).run(0).unwrap();
    }

    // Earliest should now point at block 0 (the genesis hash).
    let provider = provider_factory.database_provider_ro().unwrap();
    let genesis_hash = reth_provider::BlockHashReader::block_hash(&provider, 0).unwrap().unwrap();
    let ro = storage.provider_ro().unwrap();
    assert_eq!(ro.get_earliest_block().unwrap(), NumHash::new(0, genesis_hash));
}

#[test]
fn run_extends_window_backward_with_storage_writes() {
    // Every block calls `STORAGE_CONTRACT`, writing `block.number` to slot 0.
    // This exercises the backfill code paths that are silent in plain-transfer
    // tests:
    //   - `V2HashedStorageChangeSets` / `V2HashedStoragesHistory` writes during `prepend_block`
    //     (the slot value changes every block).
    //   - Storage-side reconstruction via `V2StorageCursor` at each historical block during the
    //     in-job `StateRoot::overlay_root` validation.
    let (provider_factory, storage, latest_num, latest_hash) =
        build_chain_with_storage_writes_and_initialize_storage(5);

    {
        let ro = storage.provider_ro().unwrap();
        assert_eq!(ro.get_earliest_block().unwrap(), NumHash::new(latest_num, latest_hash));
    }

    {
        let provider = provider_factory.database_provider_ro().unwrap();
        BackfillJob::new(provider, storage.clone()).run(0).unwrap();
    }

    let provider = provider_factory.database_provider_ro().unwrap();
    let genesis_hash = reth_provider::BlockHashReader::block_hash(&provider, 0).unwrap().unwrap();
    let ro = storage.provider_ro().unwrap();
    assert_eq!(ro.get_earliest_block().unwrap(), NumHash::new(0, genesis_hash));
}

#[test]
fn backfill_then_forward_write_preserves_state_roots() {
    // End-to-end check that backfill and forward writes can share a proofs DB
    // without corrupting historical reads:
    //
    //   1. Build a 5-block reth chain. Init proofs at block 5 → earliest=latest=5.
    //   2. Backfill earliest from 5 down to 2.
    //   3. Build + forward-write blocks 6 and 7 via `store_trie_updates`.
    //   4. Assert state roots at every block in [2, 7] match reth's headers.
    let key_pair = deterministic_keypair();
    let sender = public_key_to_address(key_pair.public_key());
    let chain_spec = chain_spec_with_address(sender);
    let provider_factory = create_test_provider_factory_with_chain_spec(chain_spec.clone());
    init_genesis(&provider_factory).unwrap();

    let recipient = Address::repeat_byte(0x42);
    let mut last_hash = chain_spec.genesis_hash();

    // 1. Build blocks 1..=5 in reth.
    for n in 1..=5u64 {
        let mut block = build_transfer_block(n, last_hash, &chain_spec, key_pair, n - 1, recipient);
        let exec = execute_block(&mut block, &provider_factory, &chain_spec);
        commit_block_to_database(&block, &exec, &provider_factory);
        last_hash = block.hash();
    }

    // 2. Initialize proofs storage at block 5.
    let storage = create_storage();
    {
        let trie_layout = if provider_factory.cached_storage_settings().is_v2() {
            RethTrieStorageLayout::Packed
        } else {
            RethTrieStorageLayout::Legacy
        };
        let tx = provider_factory.db_ref().tx().unwrap();
        InitializationJob::new(storage.clone(), tx, trie_layout).run(5, last_hash).unwrap();
    }

    // 3. Backfill earliest from 5 down to 2.
    {
        let provider = provider_factory.database_provider_ro().unwrap();
        BackfillJob::new(provider, storage.clone()).run(2).unwrap();
    }
    {
        let window = storage.provider_ro().unwrap().get_proof_window().unwrap();
        assert_eq!(window.earliest.number, 2);
        assert_eq!(window.latest.number, 5);
    }

    // 4. Build + forward-write blocks 6 and 7.
    for n in 6..=7u64 {
        let mut block = build_transfer_block(n, last_hash, &chain_spec, key_pair, n - 1, recipient);

        // Execute the block + compute (state_root, trie_updates) against reth's
        // current state. Mirrors `execute_block` but also returns the trie
        // updates + hashed post-state needed to build a `BlockStateDiff`.
        let (exec, hashed_state, trie_updates) = {
            let provider = provider_factory.provider().unwrap();
            let db = StateProviderDatabase::new(LatestStateProviderRef::new(&provider));
            let evm_config = EthEvmConfig::ethereum(chain_spec.clone());
            let block_executor = evm_config.batch_executor(db);
            let exec = block_executor.execute(&block).unwrap();
            let hashed_state =
                LatestStateProviderRef::new(&provider).hashed_post_state(&exec.state);
            let (state_root, trie_updates) = LatestStateProviderRef::new(&provider)
                .state_root_with_updates(hashed_state.clone())
                .unwrap();
            block.set_state_root(state_root);
            (exec, hashed_state, trie_updates)
        };

        // Advance reth's chain to block n.
        commit_block_to_database(&block, &exec, &provider_factory);

        // Forward-write block n into the proofs storage.
        let rw = storage.provider_rw().unwrap();
        rw.store_trie_updates(
            BlockWithParent { block: NumHash::new(n, block.hash()), parent: last_hash },
            BlockStateDiff {
                sorted_trie_updates: trie_updates.into_sorted(),
                sorted_post_state: hashed_state.into_sorted(),
            },
        )
        .unwrap();
        OpProofsProviderRw::commit(rw).unwrap();

        last_hash = block.hash();
    }

    // Window should now span [2, 7].
    {
        let window = storage.provider_ro().unwrap().get_proof_window().unwrap();
        assert_eq!(window.earliest.number, 2);
        assert_eq!(window.latest.number, 7);
    }

    // 5. Validate state roots at every block in [2, 7] against reth's headers.
    let reth_provider = provider_factory.database_provider_ro().unwrap();
    for n in 2..=7u64 {
        let expected = reth_provider::HeaderProvider::header_by_number(&reth_provider, n)
            .unwrap()
            .unwrap()
            .state_root();
        let computed =
            StateRoot::overlay_root(storage.provider_ro().unwrap(), n, HashedPostState::default())
                .unwrap();
        assert_eq!(computed, expected, "state root mismatch at block {n}");
    }
}

/// Negative test for the validation safety net in [`BackfillJob`]. Every
/// "happy path" test feeds a self-consistent chain, so the
/// [`BackfillError::StateRootMismatch`] arm in `validate_state_root` is never
/// taken — a bug there (wrong overlay block, inverted compare, etc.) would let
/// corrupt history through silently. Here we commit one block with a deliberately
/// wrong `state_root` field and assert the job aborts at the right block.
#[test]
fn run_aborts_with_state_root_mismatch_when_header_corrupted() {
    // Custom chain build
    let key_pair = deterministic_keypair();
    let sender = public_key_to_address(key_pair.public_key());
    let chain_spec = chain_spec_with_address(sender);
    let provider_factory = create_test_provider_factory_with_chain_spec(chain_spec.clone());
    init_genesis(&provider_factory).unwrap();

    let recipient = Address::repeat_byte(0x42);
    const NUM_BLOCKS: u64 = 3;
    const CORRUPTED_BLOCK: u64 = NUM_BLOCKS - 1;
    const BOGUS_ROOT: B256 = B256::repeat_byte(0xAB);

    let mut last_hash = chain_spec.genesis_hash();
    for n in 1..=NUM_BLOCKS {
        let mut block = build_transfer_block(n, last_hash, &chain_spec, key_pair, n - 1, recipient);
        let exec = execute_block(&mut block, &provider_factory, &chain_spec);
        if n == CORRUPTED_BLOCK {
            block.set_state_root(BOGUS_ROOT);
        }
        commit_block_to_database(&block, &exec, &provider_factory);
        last_hash = block.hash();
    }

    // Initialization captures *real* state at the latest block, so backfill's
    // reconstruction at any prior block will produce the real state root —
    // which disagrees with the bogus header at CORRUPTED_BLOCK.
    let storage = create_storage();
    {
        let trie_layout = if provider_factory.cached_storage_settings().is_v2() {
            RethTrieStorageLayout::Packed
        } else {
            RethTrieStorageLayout::Legacy
        };
        let tx = provider_factory.db_ref().tx().unwrap();
        InitializationJob::new(storage.clone(), tx, trie_layout)
            .run(NUM_BLOCKS, last_hash)
            .unwrap();
    }

    let provider = provider_factory.database_provider_ro().unwrap();
    let err = BackfillJob::new(provider, storage).run(0).unwrap_err();
    match err {
        BackfillError::StateRootMismatch { block_number, expected, .. } => {
            // Backfill descends from `latest`; the first prepend is NUM_BLOCKS,
            // which validates at CORRUPTED_BLOCK.
            assert_eq!(block_number, NUM_BLOCKS, "validation must fire on the first prepend");
            assert_eq!(expected, BOGUS_ROOT, "expected root must come from the tampered header");
        }
        other => panic!("expected StateRootMismatch, got {other:?}"),
    }
}
