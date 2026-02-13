//! End-to-end test of the live trie collector.

use alloy_consensus::{BlockHeader, Header, TxEip2930, constants::ETH_TO_WEI};
use alloy_genesis::{Genesis, GenesisAccount};
use alloy_primitives::{Address, B256, TxKind, U256, keccak256};
use derive_more::Constructor;
use reth_chainspec::{ChainSpec, ChainSpecBuilder, EthereumHardfork, MAINNET, MIN_TRANSACTION_GAS};
use reth_db::Database;
use reth_db_common::init::init_genesis;
use reth_ethereum_primitives::{Block, BlockBody, Receipt, Transaction, TransactionSigned};
use reth_evm::{ConfigureEvm, execute::Executor};
use reth_evm_ethereum::EthEvmConfig;
use reth_node_api::{NodePrimitives, NodeTypesWithDB};
use reth_optimism_trie::{
    MdbxProofsStorage, OpProofsStorage, OpProofsStorageError, initialize::InitializationJob,
    live::LiveTrieCollector,
};
use reth_primitives_traits::{Block as _, RecoveredBlock};
use reth_provider::{
    BlockWriter as _, ExecutionOutcome, HashedPostStateProvider, LatestStateProviderRef,
    ProviderFactory, StateRootProvider,
    providers::{BlockchainProvider, ProviderNodeTypes},
    test_utils::create_test_provider_factory_with_chain_spec,
};
use reth_revm::database::StateProviderDatabase;
use secp256k1::{Keypair, Secp256k1, rand::rng};
use std::sync::Arc;
use tempfile::TempDir;

/// Converts a secp256k1 public key to an Ethereum address.
fn public_key_to_address(pubkey: secp256k1::PublicKey) -> Address {
    let hash = keccak256(&pubkey.serialize_uncompressed()[1..]);
    Address::from_slice(&hash[12..])
}

/// Signs a transaction with the given keypair.
fn sign_tx_with_key_pair(key_pair: Keypair, tx: Transaction) -> TransactionSigned {
    use alloy_consensus::SignableTransaction;
    use reth_primitives_traits::crypto::secp256k1::sign_message;
    let secret = B256::from_slice(&key_pair.secret_bytes());
    let sig = sign_message(secret, tx.signature_hash()).unwrap();
    tx.into_signed(sig).into()
}

/// Specification for a transaction within a block
#[derive(Debug, Clone)]
struct TxSpec {
    /// Recipient address for the transaction
    to: Address,
    /// Value to transfer (in wei)
    value: U256,
    /// Nonce for the transaction (will be automatically assigned if None)
    nonce: Option<u64>,
}

impl TxSpec {
    /// Create a simple transfer transaction
    const fn transfer(to: Address, value: U256) -> Self {
        Self { to, value, nonce: None }
    }
}

/// Specification for a block in the test chain
#[derive(Debug, Clone, Constructor)]
struct BlockSpec {
    /// Transactions to include in this block
    txs: Vec<TxSpec>,
}

/// Configuration for a test scenario
#[derive(Debug, Constructor)]
struct TestScenario {
    /// Blocks to execute before running the initialization job
    blocks_before_initialization: Vec<BlockSpec>,
    /// Blocks to execute after initialization using the live collector
    blocks_after_initialization: Vec<BlockSpec>,
}

/// Helper to create a chain spec with a genesis account funded
fn chain_spec_with_address(address: Address) -> Arc<ChainSpec> {
    Arc::new(
        ChainSpecBuilder::default()
            .chain(MAINNET.chain)
            .genesis(Genesis {
                alloc: [(
                    address,
                    GenesisAccount { balance: U256::from(10 * ETH_TO_WEI), ..Default::default() },
                )]
                .into(),
                ..MAINNET.genesis.clone()
            })
            .paris_activated()
            .build(),
    )
}

/// Creates a block from a spec, executing transactions with the given keypair
fn create_block_from_spec(
    spec: &BlockSpec,
    block_number: u64,
    parent_hash: B256,
    chain_spec: &Arc<ChainSpec>,
    key_pair: Keypair,
    nonce_counter: &mut u64,
) -> RecoveredBlock<Block> {
    let transactions: Vec<TransactionSigned> = spec
        .txs
        .iter()
        .map(|tx_spec| {
            let nonce = tx_spec.nonce.unwrap_or_else(|| {
                let current = *nonce_counter;
                *nonce_counter += 1;
                current
            });

            sign_tx_with_key_pair(
                key_pair,
                TxEip2930 {
                    chain_id: chain_spec.chain.id(),
                    nonce,
                    gas_limit: MIN_TRANSACTION_GAS,
                    gas_price: 1_500_000_000,
                    to: TxKind::Call(tx_spec.to),
                    value: tx_spec.value,
                    ..Default::default()
                }
                .into(),
            )
        })
        .collect();

    let gas_total = transactions.len() as u64 * MIN_TRANSACTION_GAS;

    Block {
        header: Header {
            parent_hash,
            receipts_root: alloy_primitives::b256!(
                "0xd3a6acf9a244d78b33831df95d472c4128ea85bf079a1d41e32ed0b7d2244c9e"
            ),
            difficulty: chain_spec.fork(EthereumHardfork::Paris).ttd().expect("Paris TTD"),
            number: block_number,
            gas_limit: gas_total.max(MIN_TRANSACTION_GAS),
            gas_used: gas_total,
            state_root: B256::ZERO, // Will be calculated by executor
            ..Default::default()
        },
        body: BlockBody { transactions, ..Default::default() },
    }
    .try_into_recovered()
    .unwrap()
}

/// Executes a block and returns the updated block with correct state root
fn execute_block<N>(
    block: &mut RecoveredBlock<Block>,
    provider_factory: &ProviderFactory<N>,
    chain_spec: &Arc<ChainSpec>,
) -> eyre::Result<reth_evm::execute::BlockExecutionOutput<Receipt>>
where
    N: ProviderNodeTypes<
            Primitives: NodePrimitives<Block = Block, BlockBody = BlockBody, Receipt = Receipt>,
        > + NodeTypesWithDB,
{
    let provider = provider_factory.provider()?;
    let db = StateProviderDatabase::new(LatestStateProviderRef::new(&provider));
    let evm_config = EthEvmConfig::ethereum(chain_spec.clone());
    let block_executor = evm_config.batch_executor(db);

    let execution_result = block_executor.execute(block)?;

    let hashed_state =
        LatestStateProviderRef::new(&provider).hashed_post_state(&execution_result.state);
    let state_root = LatestStateProviderRef::new(&provider).state_root(hashed_state)?;

    block.set_state_root(state_root);

    Ok(execution_result)
}

/// Commits a block and its execution output to the database
fn commit_block_to_database<N>(
    block: &RecoveredBlock<Block>,
    execution_output: &reth_evm::execute::BlockExecutionOutput<Receipt>,
    provider_factory: &ProviderFactory<N>,
) -> eyre::Result<()>
where
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

    // Calculate hashed state from execution result
    let state_provider = provider_factory.provider()?;
    let hashed_state = HashedPostStateProvider::hashed_post_state(
        &LatestStateProviderRef::new(&state_provider),
        &execution_output.state,
    );

    let provider_rw = provider_factory.provider_rw()?;
    provider_rw.append_blocks_with_state(
        vec![block.clone()],
        &execution_outcome,
        hashed_state.into_sorted(),
    )?;
    provider_rw.commit()?;

    Ok(())
}

/// Runs a test scenario with the given configuration
fn run_test_scenario<N>(
    scenario: TestScenario,
    provider_factory: ProviderFactory<N>,
    chain_spec: Arc<ChainSpec>,
    key_pair: Keypair,
    storage: OpProofsStorage<Arc<MdbxProofsStorage>>,
) -> eyre::Result<()>
where
    N: ProviderNodeTypes<
            Primitives: NodePrimitives<Block = Block, BlockBody = BlockBody, Receipt = Receipt>,
        > + NodeTypesWithDB,
{
    let genesis_hash = chain_spec.genesis_hash();
    let mut nonce_counter = 0u64;
    let mut last_block_hash = genesis_hash;
    let mut last_block_number = 0u64;

    // Execute blocks before initialization
    for (idx, block_spec) in scenario.blocks_before_initialization.iter().enumerate() {
        let block_number = idx as u64 + 1;
        let mut block = create_block_from_spec(
            block_spec,
            block_number,
            last_block_hash,
            &chain_spec,
            key_pair,
            &mut nonce_counter,
        );

        let execution_output = execute_block(&mut block, &provider_factory, &chain_spec)?;
        commit_block_to_database(&block, &execution_output, &provider_factory)?;

        last_block_hash = block.hash();
        last_block_number = block_number;
    }

    {
        let provider = provider_factory.db_ref();
        let tx = provider.tx()?;
        let initialization_job = InitializationJob::new(storage.clone(), tx);
        initialization_job.run(last_block_number, last_block_hash)?;
    }

    // Execute blocks after initialization using live collector
    let evm_config = EthEvmConfig::ethereum(chain_spec.clone());

    for (idx, block_spec) in scenario.blocks_after_initialization.iter().enumerate() {
        let block_number = last_block_number + idx as u64 + 1;
        let mut block = create_block_from_spec(
            block_spec,
            block_number,
            last_block_hash,
            &chain_spec,
            key_pair,
            &mut nonce_counter,
        );

        // Execute the block to get the correct state root
        let execution_output = execute_block(&mut block, &provider_factory, &chain_spec)?;

        // Create a fresh blockchain provider to ensure it sees all committed blocks
        let blockchain_db = BlockchainProvider::new(provider_factory.clone())?;
        let live_trie_collector =
            LiveTrieCollector::new(evm_config.clone(), blockchain_db, &storage);

        // Use the live collector to execute and store trie updates
        live_trie_collector.execute_and_store_block_updates(&block)?;

        // Commit the block to the database so subsequent blocks can build on it
        commit_block_to_database(&block, &execution_output, &provider_factory)?;

        last_block_hash = block.hash();
    }

    Ok(())
}

/// End-to-end test of a single live collector iteration.
/// (1) Creates a chain with some state
/// (2) Stores the genesis state into storage via initialization
/// (3) Executes a block and calculates the state root using the stored state
#[test]
fn test_execute_and_store_block_updates() {
    let dir = TempDir::new().unwrap();
    let storage = Arc::new(MdbxProofsStorage::new(dir.path()).expect("env")).into();

    // Create a keypair for signing transactions
    let secp = Secp256k1::new();
    let key_pair = Keypair::new(&secp, &mut rng());
    let sender = public_key_to_address(key_pair.public_key());

    // Create chain spec with the sender address funded in genesis
    let chain_spec = chain_spec_with_address(sender);

    // Create test database and provider factory
    let provider_factory = create_test_provider_factory_with_chain_spec(chain_spec.clone());

    // Insert genesis state into the database
    init_genesis(&provider_factory).unwrap();

    // Define the test scenario:
    // - No blocks before initialization
    // - Initialization to genesis (block 0)
    // - Execute one block with a single transaction after initialization
    let recipient = Address::repeat_byte(0x42);
    let scenario = TestScenario::new(
        vec![],
        vec![BlockSpec::new(vec![TxSpec::transfer(recipient, U256::from(1))])],
    );

    run_test_scenario(scenario, provider_factory, chain_spec, key_pair, storage).unwrap();
}

#[test]
fn test_execute_and_store_block_updates_missing_parent_block() {
    let dir = TempDir::new().unwrap();
    let storage: OpProofsStorage<Arc<MdbxProofsStorage>> =
        Arc::new(MdbxProofsStorage::new(dir.path()).expect("env")).into();

    let secp = Secp256k1::new();
    let key_pair = Keypair::new(&secp, &mut rng());
    let sender = public_key_to_address(key_pair.public_key());

    let chain_spec = chain_spec_with_address(sender);
    let provider_factory = create_test_provider_factory_with_chain_spec(chain_spec.clone());
    init_genesis(&provider_factory).unwrap();

    // No blocks before initialization; initialization only inserts genesis.
    let scenario = TestScenario::new(vec![], vec![]);

    // Run initialization (block 0 only)
    run_test_scenario(
        scenario,
        provider_factory.clone(),
        chain_spec.clone(),
        key_pair,
        storage.clone(),
    )
    .unwrap();

    // Create a block whose parent block number is missing.
    let incorrect_block_number = 2;
    let incorrect_parent_hash = B256::repeat_byte(0x11);

    let mut nonce_counter = 0;
    let incorrect_block = create_block_from_spec(
        &BlockSpec::new(vec![]),
        incorrect_block_number,
        incorrect_parent_hash,
        &chain_spec,
        key_pair,
        &mut nonce_counter,
    );

    let blockchain_db = BlockchainProvider::new(provider_factory).unwrap();
    let collector =
        LiveTrieCollector::new(EthEvmConfig::ethereum(chain_spec.clone()), blockchain_db, &storage);

    // EXPECT: MissingParentBlock
    let err = collector.execute_and_store_block_updates(&incorrect_block).unwrap_err();

    assert!(matches!(err, OpProofsStorageError::MissingParentBlock { .. }));
}

#[test]
fn test_execute_and_store_block_updates_state_root_mismatch() {
    let dir = TempDir::new().unwrap();
    let storage: OpProofsStorage<Arc<MdbxProofsStorage>> =
        Arc::new(MdbxProofsStorage::new(dir.path()).expect("env")).into();

    let secp = Secp256k1::new();
    let key_pair = Keypair::new(&secp, &mut rng());
    let sender = public_key_to_address(key_pair.public_key());

    let chain_spec = chain_spec_with_address(sender);
    let provider_factory = create_test_provider_factory_with_chain_spec(chain_spec.clone());
    init_genesis(&provider_factory).unwrap();

    // Run normal scenario: no blocks before initialization, one block after.
    let recipient = Address::repeat_byte(0x42);
    let scenario = TestScenario::new(
        vec![],
        vec![BlockSpec::new(vec![TxSpec::transfer(recipient, U256::from(1))])],
    );

    run_test_scenario(
        scenario,
        provider_factory.clone(),
        chain_spec.clone(),
        key_pair,
        storage.clone(),
    )
    .unwrap();

    // Generate a second block normally
    let blockchain_db = BlockchainProvider::new(provider_factory.clone()).unwrap();
    let collector =
        LiveTrieCollector::new(EthEvmConfig::ethereum(chain_spec.clone()), blockchain_db, &storage);

    // Create the next block
    let mut nonce_counter = 0;
    let last_block_hash = chain_spec.genesis_hash(); // because scenario executes 1 block
    let next_number = 2;

    let mut block = create_block_from_spec(
        &BlockSpec::new(vec![]),
        next_number,
        last_block_hash,
        &chain_spec,
        key_pair,
        &mut nonce_counter,
    );

    // Execute it to compute a correct state root
    let _ = execute_block(&mut block, &provider_factory, &chain_spec).unwrap();

    // Change the state root to induce the error
    block.header_mut().state_root = B256::repeat_byte(0xAA);

    // EXPECT: StateRootMismatch
    let err = collector.execute_and_store_block_updates(&block).unwrap_err();

    assert!(matches!(err, OpProofsStorageError::StateRootMismatch { .. }));
}

/// Test with multiple blocks before and after initialization
#[test]
fn test_multiple_blocks_before_and_after_initialization() {
    let dir = TempDir::new().unwrap();
    let storage = Arc::new(MdbxProofsStorage::new(dir.path()).expect("env")).into();

    let secp = Secp256k1::new();
    let key_pair = Keypair::new(&secp, &mut rng());
    let sender = public_key_to_address(key_pair.public_key());

    let chain_spec = chain_spec_with_address(sender);
    let provider_factory = create_test_provider_factory_with_chain_spec(chain_spec.clone());
    init_genesis(&provider_factory).unwrap();

    // Define the test scenario:
    // - Execute 3 blocks before initialization (will be committed to db)
    // - Initialization to block 3
    // - Execute 2 more blocks using the live collector
    let recipient1 = Address::repeat_byte(0x42);
    let recipient2 = Address::repeat_byte(0x43);
    let recipient3 = Address::repeat_byte(0x44);

    let scenario = TestScenario::new(
        vec![
            BlockSpec::new(vec![TxSpec::transfer(recipient1, U256::from(1))]),
            BlockSpec::new(vec![TxSpec::transfer(recipient2, U256::from(2))]),
            BlockSpec::new(vec![TxSpec::transfer(recipient3, U256::from(3))]),
        ],
        vec![
            BlockSpec::new(vec![TxSpec::transfer(recipient1, U256::from(4))]),
            BlockSpec::new(vec![TxSpec::transfer(recipient2, U256::from(5))]),
        ],
    );

    run_test_scenario(scenario, provider_factory, chain_spec, key_pair, storage).unwrap();
}

/// Test with blocks containing multiple transactions
#[test]
fn test_blocks_with_multiple_transactions() {
    let dir = TempDir::new().unwrap();
    let storage = Arc::new(MdbxProofsStorage::new(dir.path()).expect("env")).into();

    let secp = Secp256k1::new();
    let key_pair = Keypair::new(&secp, &mut rng());
    let sender = public_key_to_address(key_pair.public_key());

    let chain_spec = chain_spec_with_address(sender);
    let provider_factory = create_test_provider_factory_with_chain_spec(chain_spec.clone());
    init_genesis(&provider_factory).unwrap();

    let recipient1 = Address::repeat_byte(0x42);
    let recipient2 = Address::repeat_byte(0x43);
    let recipient3 = Address::repeat_byte(0x44);

    // Block with 3 transactions
    let scenario = TestScenario::new(
        vec![],
        vec![BlockSpec::new(vec![
            TxSpec::transfer(recipient1, U256::from(1)),
            TxSpec::transfer(recipient2, U256::from(2)),
            TxSpec::transfer(recipient3, U256::from(3)),
        ])],
    );

    run_test_scenario(scenario, provider_factory, chain_spec, key_pair, storage).unwrap();
}
