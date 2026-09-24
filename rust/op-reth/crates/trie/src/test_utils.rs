//! Shared test fixtures for OP-reth proofs storage tests.
//!
//! Builds real reth-side chains (genesis + EVM-executed blocks) and
//! initializes a V2 MDBX proofs storage on top, so downstream tests
//! (backfill, snapshot, …) can exercise end-to-end pipelines without
//! duplicating the chain-construction boilerplate.
//!
//! Gated on `#[cfg(test)]` and re-exported via `pub(crate)`; not part of the
//! crate's public surface.

use crate::{MdbxProofsStorageV2, RethTrieStorageLayout, initialize::InitializationJob};
use alloy_consensus::{Header, TxEip2930, constants::ETH_TO_WEI};
use alloy_genesis::{Genesis, GenesisAccount};
use alloy_primitives::{Address, B256, Bytes, TxKind, U256, keccak256};
use reth_chainspec::{ChainSpec, ChainSpecBuilder, EthereumHardfork, MAINNET, MIN_TRANSACTION_GAS};
use reth_db::Database;
use reth_db_common::init::init_genesis;
use reth_ethereum_primitives::{Block, BlockBody, Receipt, Transaction, TransactionSigned};
use reth_evm::{ConfigureEvm, execute::Executor};
use reth_evm_ethereum::EthEvmConfig;
use reth_node_api::{NodePrimitives, NodeTypesWithDB};
use reth_primitives_traits::{AlloyBlockHeader, Block as _, RecoveredBlock};
use reth_provider::{
    BlockWriter as _, ExecutionOutcome, HashedPostStateProvider, LatestStateProviderRef,
    ProviderFactory, StateRootProvider, StorageSettingsCache, providers::ProviderNodeTypes,
    test_utils::create_test_provider_factory_with_chain_spec,
};
use reth_revm::database::StateProviderDatabase;
use secp256k1::{Keypair, Secp256k1, SecretKey};
use std::sync::Arc;
use tempfile::TempDir;

/// Create a fresh V2 MDBX proofs storage backed by a temporary directory.
pub(crate) fn create_storage() -> Arc<MdbxProofsStorageV2> {
    let path = TempDir::new().unwrap();
    Arc::new(MdbxProofsStorageV2::new(path.path()).unwrap())
}

pub(crate) fn public_key_to_address(pubkey: secp256k1::PublicKey) -> Address {
    let hash = keccak256(&pubkey.serialize_uncompressed()[1..]);
    Address::from_slice(&hash[12..])
}

/// Deterministic test keypair. Using a fixed secret key (rather than the
/// thread-local OS RNG) keeps the sender address and thus the entire chain's
/// state roots reproducible across runs.
pub(crate) fn deterministic_keypair() -> Keypair {
    let secp = Secp256k1::new();
    let secret = SecretKey::from_byte_array([0x42u8; 32]).expect("valid secret");
    Keypair::from_secret_key(&secp, &secret)
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

pub(crate) fn chain_spec_with_address(address: Address) -> Arc<ChainSpec> {
    let alloc = std::iter::once((
        address,
        GenesisAccount { balance: U256::from(10 * ETH_TO_WEI), ..Default::default() },
    ))
    .chain(std::iter::once((
        STORAGE_CONTRACT,
        GenesisAccount { code: Some(Bytes::from_static(&STORAGE_BYTECODE)), ..Default::default() },
    )))
    .chain((0x10u8..0x30).map(|i| {
        (
            Address::repeat_byte(i),
            GenesisAccount { balance: U256::from(1u64), ..Default::default() },
        )
    }))
    .collect();

    Arc::new(
        ChainSpecBuilder::default()
            .chain(MAINNET.chain)
            .genesis(Genesis { alloc, ..MAINNET.genesis.clone() })
            .paris_activated()
            .build(),
    )
}

/// Construct an unsealed block with a single simple transfer.
pub(crate) fn build_transfer_block(
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

pub(crate) fn execute_block<N>(
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
        LatestStateProviderRef::new(&provider).hashed_post_state(&execution_result.state).unwrap();
    let state_root = LatestStateProviderRef::new(&provider).state_root(hashed_state).unwrap();
    block.set_state_root(state_root);
    execution_result
}

pub(crate) fn commit_block_to_database<N>(
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
    )
    .unwrap();
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
pub(crate) fn build_chain_and_initialize_storage(
    num_blocks: u64,
) -> (
    ProviderFactory<reth_provider::test_utils::MockNodeTypesWithDB>,
    Arc<MdbxProofsStorageV2>,
    u64,
    B256,
) {
    let key_pair = deterministic_keypair();
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
pub(crate) fn build_chain_with_storage_writes_and_initialize_storage(
    num_blocks: u64,
) -> (
    ProviderFactory<reth_provider::test_utils::MockNodeTypesWithDB>,
    Arc<MdbxProofsStorageV2>,
    u64,
    B256,
) {
    let key_pair = deterministic_keypair();
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

/// Fixtures for `HashedPostStateProvider` tests over the proofs storage: persisted storage
/// seeding, bundles with destroyed accounts, and a provider whose storage reads fail.
pub(crate) mod destroyed_accounts {
    use crate::{
        OpProofsStorageError, OpProofsStorageResult,
        api::{
            BlockStateDiff, OpProofsInitProvider, OpProofsProviderRO, OpProofsStore,
            ProofWindowRange,
        },
        in_memory::{
            InMemoryAccountCursor, InMemoryProofsProvider, InMemoryProofsStorage,
            InMemoryTrieCursor,
        },
    };
    use alloy_eips::{BlockNumHash, NumHash};
    use alloy_primitives::{Address, B256, U256, keccak256};
    use reth_db::DatabaseError;
    use reth_provider::{ProviderError, ProviderResult};
    use reth_revm::{
        db::{AccountStatus, BundleAccount, BundleState, states::StorageSlot},
        state::AccountInfo,
    };
    use reth_trie::{
        HashedPostState, HashedStorage,
        hashed_cursor::{HashedCursor, HashedStorageCursor},
    };

    /// Hashed storage key of the plain slot `slot`.
    pub(crate) fn hashed_slot(slot: u64) -> B256 {
        keccak256(B256::from(U256::from(slot)))
    }

    /// Non-wiped [`HashedStorage`] holding the plain `(slot, value)` pairs.
    pub(crate) fn hashed_storage(slots: &[(u64, u64)]) -> HashedStorage {
        HashedStorage::from_iter(
            slots.iter().map(|&(slot, value)| (hashed_slot(slot), U256::from(value))),
        )
    }

    /// Commits `storages` as the initial state of `store`, anchored at block 0.
    pub(crate) fn seed_initial_storage<S: OpProofsStore>(
        store: &S,
        storages: &[(Address, &[(u64, u64)])],
    ) {
        let provider = store.initialization_provider().unwrap();
        for (address, slots) in storages {
            // The MDBX init writer appends, so slots must arrive in key order.
            let mut slots: Vec<_> = hashed_storage(slots).storage.into_iter().collect();
            slots.sort_unstable();
            provider.store_hashed_storages(keccak256(address), slots).unwrap();
        }
        provider.set_initial_state_anchor(BlockNumHash::new(0, B256::ZERO)).unwrap();
        provider.commit_initial_state().unwrap();
        OpProofsInitProvider::commit(provider).unwrap();
    }

    /// Block diff writing the plain `(slot, value)` storage of each address.
    pub(crate) fn storage_diff(storages: &[(Address, &[(u64, u64)])]) -> BlockStateDiff {
        let mut post_state = HashedPostState::default();
        for (address, slots) in storages {
            post_state.storages.insert(keccak256(address), hashed_storage(slots));
        }
        BlockStateDiff { sorted_post_state: post_state.into_sorted(), ..Default::default() }
    }

    /// Bundle holding `accounts`.
    pub(crate) fn bundle(
        accounts: impl IntoIterator<Item = (Address, BundleAccount)>,
    ) -> BundleState {
        let mut bundle = BundleState::default();
        bundle.state.extend(accounts);
        bundle
    }

    /// Pre-existing account that self-destructed.
    pub(crate) fn destroyed() -> BundleAccount {
        BundleAccount::new(
            Some(AccountInfo::default()),
            None,
            Default::default(),
            AccountStatus::Destroyed,
        )
    }

    /// Pre-existing account that self-destructed and was recreated with the plain
    /// `(slot, value)` storage.
    pub(crate) fn destroyed_and_recreated(storage: &[(u64, u64)]) -> BundleAccount {
        let storage = storage
            .iter()
            .map(|&(slot, value)| {
                (U256::from(slot), StorageSlot::new_changed(U256::ZERO, U256::from(value)))
            })
            .collect();
        BundleAccount::new(
            Some(AccountInfo::default()),
            Some(AccountInfo::default()),
            storage,
            AccountStatus::DestroyedChanged,
        )
    }

    /// Account created and self-destructed within the bundle.
    pub(crate) fn created_and_destroyed() -> BundleAccount {
        BundleAccount::new(None, None, Default::default(), AccountStatus::Destroyed)
    }

    /// Storage key carried by [`injected_error`].
    const INJECTED_KEY: B256 = B256::repeat_byte(0xEE);

    /// Error with which [`FailingStorageProvider`] fails hashed storage reads.
    pub(crate) const fn injected_error() -> OpProofsStorageError {
        OpProofsStorageError::MissingHashedStorageHistory {
            hashed_address: B256::ZERO,
            hashed_storage_key: INJECTED_KEY,
            block_number: 0,
        }
    }

    /// Asserts that `result` failed with [`injected_error`].
    pub(crate) fn assert_injected_error(result: ProviderResult<HashedPostState>) {
        let err = result.expect_err("storage read failure must surface");
        let ProviderError::Database(err) = err else { panic!("unexpected error: {err:?}") };
        assert!(
            matches!(
                OpProofsStorageError::from(err.clone()),
                OpProofsStorageError::MissingHashedStorageHistory { hashed_storage_key, .. }
                    if hashed_storage_key == INJECTED_KEY
            ),
            "unexpected error: {err:?}"
        );
    }

    /// Stage at which [`FailingStorageProvider`] fails a hashed storage read.
    #[derive(Debug, Clone, Copy)]
    pub(crate) enum StorageFailure {
        /// Opening the storage cursor fails.
        Open,
        /// Seeking the opened storage cursor fails.
        Seek,
    }

    /// [`OpProofsProviderRO`] delegating to an [`InMemoryProofsProvider`], except that every
    /// hashed storage read fails with [`injected_error`].
    #[derive(Debug, Clone)]
    pub(crate) struct FailingStorageProvider {
        inner: InMemoryProofsProvider,
        failure: StorageFailure,
    }

    impl FailingStorageProvider {
        /// Creates a provider over an empty in-memory store that fails at `failure`.
        pub(crate) fn new(failure: StorageFailure) -> Self {
            Self { inner: InMemoryProofsStorage::new().provider_ro().unwrap(), failure }
        }
    }

    /// Hashed storage cursor whose reads fail with [`injected_error`].
    #[derive(Debug)]
    pub(crate) struct FailingStorageCursor;

    impl HashedCursor for FailingStorageCursor {
        type Value = U256;

        fn seek(&mut self, _key: B256) -> Result<Option<(B256, U256)>, DatabaseError> {
            Err(injected_error().into())
        }

        fn next(&mut self) -> Result<Option<(B256, U256)>, DatabaseError> {
            Err(injected_error().into())
        }

        fn reset(&mut self) {}
    }

    impl HashedStorageCursor for FailingStorageCursor {
        fn is_storage_empty(&mut self) -> Result<bool, DatabaseError> {
            Err(injected_error().into())
        }

        fn set_hashed_address(&mut self, _hashed_address: B256) {}
    }

    impl OpProofsProviderRO for FailingStorageProvider {
        type StorageTrieCursor<'tx> = InMemoryTrieCursor;
        type AccountTrieCursor<'tx> = InMemoryTrieCursor;
        type StorageCursor<'tx> = FailingStorageCursor;
        type AccountHashedCursor<'tx> = InMemoryAccountCursor;

        fn get_earliest_block(&self) -> OpProofsStorageResult<NumHash> {
            self.inner.get_earliest_block()
        }

        fn get_latest_block(&self) -> OpProofsStorageResult<NumHash> {
            self.inner.get_latest_block()
        }

        fn get_proof_window(&self) -> OpProofsStorageResult<ProofWindowRange> {
            self.inner.get_proof_window()
        }

        fn storage_trie_cursor<'tx>(
            &self,
            hashed_address: B256,
            max_block_number: u64,
        ) -> OpProofsStorageResult<Self::StorageTrieCursor<'tx>> {
            self.inner.storage_trie_cursor(hashed_address, max_block_number)
        }

        fn account_trie_cursor<'tx>(
            &self,
            max_block_number: u64,
        ) -> OpProofsStorageResult<Self::AccountTrieCursor<'tx>> {
            self.inner.account_trie_cursor(max_block_number)
        }

        fn storage_hashed_cursor<'tx>(
            &self,
            _hashed_address: B256,
            _max_block_number: u64,
        ) -> OpProofsStorageResult<Self::StorageCursor<'tx>> {
            match self.failure {
                StorageFailure::Open => Err(injected_error()),
                StorageFailure::Seek => Ok(FailingStorageCursor),
            }
        }

        fn account_hashed_cursor<'tx>(
            &self,
            max_block_number: u64,
        ) -> OpProofsStorageResult<Self::AccountHashedCursor<'tx>> {
            self.inner.account_hashed_cursor(max_block_number)
        }

        fn fetch_trie_updates(&self, block_number: u64) -> OpProofsStorageResult<BlockStateDiff> {
            self.inner.fetch_trie_updates(block_number)
        }
    }
}
