//! Integration tests for [`BackfillJob`].
//!
//! Helpers here mirror `crates/trie/tests/live.rs` because integration-test
//! helpers in `tests/` are not reachable from `src/`. If this duplication grows,
//! consider extracting a shared `test_utils` module behind a feature flag.

use super::{BackfillError, BackfillJob};
use crate::{
    BlockStateDiff, MdbxProofsStorageV2, OpProofsStorageError, OpProofsStore,
    RethTrieStorageLayout,
    api::{OpProofsProviderRO, OpProofsProviderRw},
    initialize::InitializationJob,
    proof::DatabaseStateRoot,
};
use alloy_consensus::{BlockHeader, Header, TxEip2930, constants::ETH_TO_WEI};
use alloy_eips::{NumHash, eip1898::BlockWithParent};
use alloy_genesis::{Genesis, GenesisAccount};
use alloy_primitives::{Address, B256, Bytes, TxKind, U256, keccak256};
use reth_chainspec::{ChainSpec, ChainSpecBuilder, EthereumHardfork, MAINNET, MIN_TRANSACTION_GAS};
use reth_db::Database;
use reth_db_common::init::init_genesis;
use reth_ethereum_primitives::{Block, BlockBody, Receipt, Transaction, TransactionSigned};
use reth_evm::{ConfigureEvm, execute::Executor};
use reth_evm_ethereum::EthEvmConfig;
use reth_node_api::{NodePrimitives, NodeTypesWithDB};
use reth_primitives_traits::{Block as _, RecoveredBlock};
use reth_provider::{
    BlockWriter as _, DatabaseProviderFactory, ExecutionOutcome, HashedPostStateProvider,
    LatestStateProviderRef, ProviderFactory, StateRootProvider, StorageSettingsCache,
    providers::ProviderNodeTypes, test_utils::create_test_provider_factory_with_chain_spec,
};
use reth_revm::database::StateProviderDatabase;
use reth_trie::{HashedPostState, StateRoot};
use secp256k1::{Keypair, Secp256k1, rand::rng};
use serial_test::serial;
use std::sync::Arc;
use tempfile::TempDir;

// ============================ Chain construction helpers ============================

fn create_storage() -> Arc<MdbxProofsStorageV2> {
    let path = TempDir::new().unwrap();
    Arc::new(MdbxProofsStorageV2::new(path.path()).unwrap())
}

fn public_key_to_address(pubkey: secp256k1::PublicKey) -> Address {
    let hash = keccak256(&pubkey.serialize_uncompressed()[1..]);
    Address::from_slice(&hash[12..])
}

fn sign_tx_with_key_pair(key_pair: Keypair, tx: Transaction) -> TransactionSigned {
    use alloy_consensus::SignableTransaction;
    use reth_primitives_traits::crypto::secp256k1::sign_message;
    let secret = B256::from_slice(&key_pair.secret_bytes());
    let sig = sign_message(secret, tx.signature_hash()).unwrap();
    tx.into_signed(sig).into()
}

/// Pre-allocated contract address for storage-write tests.
const STORAGE_CONTRACT: Address = Address::repeat_byte(0xAB);

/// Minimal contract that writes `BLOCKNUMBER` (i.e. current block.number) to
/// storage slot 0:
///
/// ```text
///   0x43        BLOCKNUMBER     push block.number
///   0x60 0x00   PUSH1 0x00      push slot 0
///   0x55        SSTORE          store
///   0x00        STOP
/// ```
const STORAGE_BYTECODE: [u8; 5] = [0x43, 0x60, 0x00, 0x55, 0x00];

fn chain_spec_with_address(address: Address) -> Arc<ChainSpec> {
    Arc::new(
        ChainSpecBuilder::default()
            .chain(MAINNET.chain)
            .genesis(Genesis {
                alloc: [
                    (
                        address,
                        GenesisAccount {
                            balance: U256::from(10 * ETH_TO_WEI),
                            ..Default::default()
                        },
                    ),
                    (
                        STORAGE_CONTRACT,
                        GenesisAccount {
                            code: Some(Bytes::from_static(&STORAGE_BYTECODE)),
                            ..Default::default()
                        },
                    ),
                ]
                .into(),
                ..MAINNET.genesis.clone()
            })
            .paris_activated()
            .build(),
    )
}

/// Construct an unsealed block with a single simple transfer.
fn build_transfer_block(
    block_number: u64,
    parent_hash: B256,
    chain_spec: &Arc<ChainSpec>,
    key_pair: Keypair,
    nonce: u64,
    recipient: Address,
) -> RecoveredBlock<Block> {
    let tx = sign_tx_with_key_pair(
        key_pair,
        TxEip2930 {
            chain_id: chain_spec.chain.id(),
            nonce,
            gas_limit: MIN_TRANSACTION_GAS,
            gas_price: 1_500_000_000,
            to: TxKind::Call(recipient),
            value: U256::from(1),
            ..Default::default()
        }
        .into(),
    );
    Block {
        header: Header {
            parent_hash,
            receipts_root: alloy_primitives::b256!(
                "0xd3a6acf9a244d78b33831df95d472c4128ea85bf079a1d41e32ed0b7d2244c9e"
            ),
            difficulty: chain_spec.fork(EthereumHardfork::Paris).ttd().expect("Paris TTD"),
            number: block_number,
            gas_limit: MIN_TRANSACTION_GAS,
            gas_used: MIN_TRANSACTION_GAS,
            state_root: B256::ZERO, // filled in by execute_block
            ..Default::default()
        },
        body: BlockBody { transactions: vec![tx], ..Default::default() },
    }
    .try_into_recovered()
    .unwrap()
}

fn execute_block<N>(
    block: &mut RecoveredBlock<Block>,
    provider_factory: &ProviderFactory<N>,
    chain_spec: &Arc<ChainSpec>,
) -> reth_evm::execute::BlockExecutionOutput<Receipt>
where
    N: ProviderNodeTypes<
            Primitives: NodePrimitives<Block = Block, BlockBody = BlockBody, Receipt = Receipt>,
        > + NodeTypesWithDB,
{
    let provider = provider_factory.provider().unwrap();
    let db = StateProviderDatabase::new(LatestStateProviderRef::new(&provider));
    let evm_config = EthEvmConfig::ethereum(chain_spec.clone());
    let block_executor = evm_config.batch_executor(db);
    let execution_result = block_executor.execute(block).unwrap();

    let hashed_state =
        LatestStateProviderRef::new(&provider).hashed_post_state(&execution_result.state);
    let state_root = LatestStateProviderRef::new(&provider).state_root(hashed_state).unwrap();
    block.set_state_root(state_root);
    execution_result
}

fn commit_block_to_database<N>(
    block: &RecoveredBlock<Block>,
    execution_output: &reth_evm::execute::BlockExecutionOutput<Receipt>,
    provider_factory: &ProviderFactory<N>,
) where
    N: ProviderNodeTypes<
            Primitives: NodePrimitives<Block = Block, BlockBody = BlockBody, Receipt = Receipt>,
        > + NodeTypesWithDB,
{
    let execution_outcome = ExecutionOutcome {
        bundle: execution_output.state.clone(),
        receipts: vec![execution_output.receipts.clone()],
        first_block: block.number(),
        requests: vec![execution_output.requests.clone()],
    };
    let state_provider = provider_factory.provider().unwrap();
    let hashed_state = HashedPostStateProvider::hashed_post_state(
        &LatestStateProviderRef::new(&state_provider),
        &execution_output.state,
    );
    let provider_rw = provider_factory.provider_rw().unwrap();
    provider_rw
        .append_blocks_with_state(
            vec![block.clone()],
            &execution_outcome,
            hashed_state.into_sorted(),
        )
        .unwrap();
    provider_rw.commit().unwrap();
}

/// Construct an unsealed block whose sole tx calls [`STORAGE_CONTRACT`],
/// triggering an SSTORE of `block.number` into slot 0 of the contract's storage.
///
/// Gas accounting: the executor recomputes `gas_used` against the actual EVM
/// trace, so we deliberately set `gas_limit == gas_used` to a value large
/// enough to cover both the 21 000-gas tx base cost and the worst-case cold
/// SSTORE (~22 100 gas).
fn build_storage_call_block(
    block_number: u64,
    parent_hash: B256,
    chain_spec: &Arc<ChainSpec>,
    key_pair: Keypair,
    nonce: u64,
) -> RecoveredBlock<Block> {
    const CALL_GAS_LIMIT: u64 = 100_000;
    let tx = sign_tx_with_key_pair(
        key_pair,
        TxEip2930 {
            chain_id: chain_spec.chain.id(),
            nonce,
            gas_limit: CALL_GAS_LIMIT,
            gas_price: 1_500_000_000,
            to: TxKind::Call(STORAGE_CONTRACT),
            value: U256::ZERO,
            ..Default::default()
        }
        .into(),
    );
    Block {
        header: Header {
            parent_hash,
            receipts_root: alloy_primitives::b256!(
                "0xd3a6acf9a244d78b33831df95d472c4128ea85bf079a1d41e32ed0b7d2244c9e"
            ),
            difficulty: chain_spec.fork(EthereumHardfork::Paris).ttd().expect("Paris TTD"),
            number: block_number,
            gas_limit: CALL_GAS_LIMIT,
            gas_used: CALL_GAS_LIMIT,
            state_root: B256::ZERO,
            ..Default::default()
        },
        body: BlockBody { transactions: vec![tx], ..Default::default() },
    }
    .try_into_recovered()
    .unwrap()
}

/// Build a chain of `num_blocks` simple transfer blocks on top of a freshly
/// initialized genesis, then initialize the v2 proofs storage at the latest
/// block. Returns the provider factory, the storage, and the latest
/// (number, hash) pair.
fn build_chain_and_initialize_storage(
    num_blocks: u64,
) -> (
    ProviderFactory<reth_provider::test_utils::MockNodeTypesWithDB>,
    Arc<MdbxProofsStorageV2>,
    u64,
    B256,
) {
    let secp = Secp256k1::new();
    let key_pair = Keypair::new(&secp, &mut rng());
    let sender = public_key_to_address(key_pair.public_key());

    let chain_spec = chain_spec_with_address(sender);
    let provider_factory = create_test_provider_factory_with_chain_spec(chain_spec.clone());
    init_genesis(&provider_factory).unwrap();

    let recipient = Address::repeat_byte(0x42);
    let mut last_hash = chain_spec.genesis_hash();
    let mut last_number = 0u64;
    for n in 1..=num_blocks {
        let mut block = build_transfer_block(n, last_hash, &chain_spec, key_pair, n - 1, recipient);
        let exec = execute_block(&mut block, &provider_factory, &chain_spec);
        commit_block_to_database(&block, &exec, &provider_factory);
        last_hash = block.hash();
        last_number = n;
    }

    let storage = create_storage();
    {
        let trie_layout = if provider_factory.cached_storage_settings().is_v2() {
            RethTrieStorageLayout::Packed
        } else {
            RethTrieStorageLayout::Legacy
        };
        let tx = provider_factory.db_ref().tx().unwrap();
        InitializationJob::new(storage.clone(), tx, trie_layout)
            .run(last_number, last_hash)
            .unwrap();
    }

    (provider_factory, storage, last_number, last_hash)
}

/// Like [`build_chain_and_initialize_storage`] but every block calls
/// [`STORAGE_CONTRACT`], so each block produces hashed-storage changesets in
/// addition to the account-level ones.
fn build_chain_with_storage_writes_and_initialize_storage(
    num_blocks: u64,
) -> (
    ProviderFactory<reth_provider::test_utils::MockNodeTypesWithDB>,
    Arc<MdbxProofsStorageV2>,
    u64,
    B256,
) {
    let secp = Secp256k1::new();
    let key_pair = Keypair::new(&secp, &mut rng());
    let sender = public_key_to_address(key_pair.public_key());

    let chain_spec = chain_spec_with_address(sender);
    let provider_factory = create_test_provider_factory_with_chain_spec(chain_spec.clone());
    init_genesis(&provider_factory).unwrap();

    let mut last_hash = chain_spec.genesis_hash();
    let mut last_number = 0u64;
    for n in 1..=num_blocks {
        let mut block = build_storage_call_block(n, last_hash, &chain_spec, key_pair, n - 1);
        let exec = execute_block(&mut block, &provider_factory, &chain_spec);
        commit_block_to_database(&block, &exec, &provider_factory);
        last_hash = block.hash();
        last_number = n;
    }

    let storage = create_storage();
    {
        let trie_layout = if provider_factory.cached_storage_settings().is_v2() {
            RethTrieStorageLayout::Packed
        } else {
            RethTrieStorageLayout::Legacy
        };
        let tx = provider_factory.db_ref().tx().unwrap();
        InitializationJob::new(storage.clone(), tx, trie_layout)
            .run(last_number, last_hash)
            .unwrap();
    }

    (provider_factory, storage, last_number, last_hash)
}

// ============================ Tests ============================

#[test]
#[serial]
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
#[serial]
fn run_errors_when_storage_uninitialized() {
    let secp = Secp256k1::new();
    let key_pair = Keypair::new(&secp, &mut rng());
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
#[serial]
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
#[serial]
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
#[serial]
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
#[serial]
fn backfill_then_forward_write_preserves_state_roots() {
    // End-to-end check that backfill and forward writes can share a proofs DB
    // without corrupting historical reads:
    //
    //   1. Build a 5-block reth chain. Init proofs at block 5 → earliest=latest=5.
    //   2. Backfill earliest from 5 down to 2.
    //   3. Build + forward-write blocks 6 and 7 via `store_trie_updates`.
    //   4. Assert state roots at every block in [2, 7] match reth's headers.
    let secp = Secp256k1::new();
    let key_pair = Keypair::new(&secp, &mut rng());
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
#[serial]
fn run_aborts_with_state_root_mismatch_when_header_corrupted() {
    // Custom chain build
    let secp = Secp256k1::new();
    let key_pair = Keypair::new(&secp, &mut rng());
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
