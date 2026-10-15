//! Process-interrupted storage-v2 pipeline unwind and restart regression.

use alloy_consensus::{Header, TxEip1559, TxReceipt, constants::ETH_TO_WEI};
use alloy_eips::eip1559::INITIAL_BASE_FEE;
use alloy_genesis::{Genesis, GenesisAccount};
use alloy_primitives::{Address, B256, Bytes, TxKind, U256, bytes};
use reth_chainspec::{ChainSpecBuilder, ChainSpecProvider, MAINNET};
use reth_config::config::StageConfig;
use reth_consensus::noop::NoopConsensus;
use reth_db_common::init::init_genesis;
use reth_downloaders::{
    bodies::bodies::BodiesDownloaderBuilder, file_client::FileClient,
    headers::reverse_headers::ReverseHeadersDownloaderBuilder,
};
use reth_ethereum_primitives::{Block, BlockBody, Transaction};
use reth_evm::{ConfigureEvm, execute::Executor};
use reth_evm_ethereum::EthEvmConfig;
use reth_network_p2p::{
    bodies::downloader::BodyDownloader,
    headers::downloader::{HeaderDownloader, SyncTarget},
};
use reth_primitives_traits::{
    RecoveredBlock, SealedBlock,
    crypto::secp256k1::public_key_to_address,
    proofs::{calculate_receipt_root, calculate_transaction_root},
};
use reth_provider::{
    AccountReader, BlockNumReader, DBProvider, DatabaseProviderFactory, HeaderProvider,
    OriginalValuesKnown, ProviderFactory, StageCheckpointReader, StateWriter,
    StaticFileProviderFactory,
    providers::{RocksDBBuilder, StaticFileProviderBuilder},
    test_utils::create_test_provider_factory_with_chain_spec,
};
use reth_prune_types::PruneModes;
use reth_revm::database::StateProviderDatabase;
use reth_stages::{sets::DefaultStages, stages::StorageHashingStage};
use reth_stages_api::{
    ExecInput, ExecOutput, Pipeline, Stage, StageError, StageId, StageSet, UnwindInput,
    UnwindOutput,
};
use reth_static_file::StaticFileProducer;
use reth_storage_api::{
    ChangeSetReader, StateProvider, StorageChangeSetReader, StorageSettings, StorageSettingsCache,
};
use reth_testing_utils::generators::sign_tx_with_key_pair;
use reth_trie::{HashedPostState, KeccakKeyHasher, StateRoot};
use reth_trie_db::DatabaseStateRoot;
use secp256k1::{Keypair, Secp256k1};
use std::{
    fs::File,
    io::{Read, Write},
    os::unix::net::{UnixListener, UnixStream},
    path::Path,
    process::{Command, Stdio},
    sync::Arc,
    time::Duration,
};
use tokio::sync::watch;
use wait_timeout::ChildExt;

const DATADIR_ENV: &str = "OP_RETH_INTERRUPTED_UNWIND_DATADIR";
const SIGNAL_ENV: &str = "OP_RETH_INTERRUPTED_UNWIND_SIGNAL";
const PHASE_ENV: &str = "OP_RETH_INTERRUPTED_UNWIND_PHASE";

/// Delegates all actual work to the production stage. The signal is sent only after the
/// pipeline has durably committed the stage and its checkpoint.
struct StorageCommitGate(StorageHashingStage);

impl<P> Stage<P> for StorageCommitGate
where
    StorageHashingStage: Stage<P>,
{
    fn id(&self) -> StageId {
        StageId::StorageHashing
    }

    fn execute(&mut self, provider: &P, input: ExecInput) -> Result<ExecOutput, StageError> {
        self.0.execute(provider, input)
    }

    fn unwind(&mut self, provider: &P, input: UnwindInput) -> Result<UnwindOutput, StageError> {
        self.0.unwind(provider, input)
    }

    fn post_unwind_commit(&mut self) -> Result<(), StageError> {
        self.0.post_unwind_commit()?;
        if let Some(path) = std::env::var_os(SIGNAL_ENV) {
            let mut stream = UnixStream::connect(path).expect("connect unwind commit signal");
            stream.write_all(b"S").expect("send unwind commit signal");
            // The parent kills this process. Do not run any cleanup or later stage.
            loop {
                std::thread::park();
            }
        }
        Ok(())
    }
}

/// Counter contract deployed bytecode compiled with Solidity 0.8.31.
/// ```solidity
/// contract Counter {
///     uint256 public count;
///     function increment() public { count += 1; }
/// }
/// ```
const COUNTER_DEPLOYED_BYTECODE: Bytes = bytes!(
    "6080604052348015600e575f5ffd5b50600436106030575f3560e01c806306661abd146034578063d09de08a14604e575b5f5ffd5b603a6056565b604051604591906089565b60405180910390f35b6054605b565b005b5f5481565b60015f5f828254606a919060cd565b92505081905550565b5f819050919050565b6083816073565b82525050565b5f602082019050609a5f830184607c565b92915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f60d5826073565b915060de836073565b925082820190508082111560f35760f260a0565b5b9291505056fea2646970667358221220576016d010ec2f4f83b992fb97d16efd1bc54110c97aa5d5cb47d20d3b39a35264736f6c634300081f0033"
);

/// `increment()` function selector: `keccak256("increment()")`[:4]
const INCREMENT_SELECTOR: [u8; 4] = [0xd0, 0x9d, 0xe0, 0x8a];

/// Contract address (deterministic for test)
const CONTRACT_ADDRESS: Address = Address::new([0x42; 20]);

/// Creates a `FileClient` populated with the given blocks.
fn create_file_client_from_blocks(blocks: Vec<SealedBlock<Block>>) -> Arc<FileClient<Block>> {
    Arc::new(FileClient::from_blocks(blocks))
}

/// Verify the v2 changesets needed to finish Execution unwind are still available.
fn assert_changesets_queryable(
    provider_factory: &reth_provider::ProviderFactory<
        reth_provider::test_utils::MockNodeTypesWithDB,
    >,
    block_range: std::ops::RangeInclusive<u64>,
) -> eyre::Result<()> {
    let provider = provider_factory.provider()?;
    assert!(provider.cached_storage_settings().storage_v2);
    let static_file_provider = provider_factory.static_file_provider();
    static_file_provider.initialize_index()?;
    assert!(!static_file_provider.storage_changesets_range(block_range.clone())?.is_empty());
    assert!(!static_file_provider.account_changesets_range(block_range)?.is_empty());

    Ok(())
}

/// Builds downloaders from a `FileClient`.
fn build_downloaders_from_file_client(
    file_client: Arc<FileClient<Block>>,
    local_head: reth_primitives_traits::SealedHeader<Header>,
    from_block: u64,
    stages_config: StageConfig,
    consensus: Arc<NoopConsensus>,
    provider_factory: reth_provider::ProviderFactory<
        reth_provider::test_utils::MockNodeTypesWithDB,
    >,
) -> (impl HeaderDownloader<Header = Header>, impl BodyDownloader<Block = Block>, reth_tasks::Runtime)
{
    let tip = file_client.tip().expect("file client should have tip");
    let max_block = file_client.max_block().expect("file client should have max block");

    let runtime = reth_tasks::Runtime::test();

    let mut header_downloader = ReverseHeadersDownloaderBuilder::new(stages_config.headers)
        .build(file_client.clone(), consensus.clone())
        .into_task_with(&runtime);
    header_downloader.update_local_head(local_head);
    header_downloader.update_sync_target(SyncTarget::Tip(tip));

    let mut body_downloader = BodiesDownloaderBuilder::new(stages_config.bodies)
        .build(file_client, consensus, provider_factory)
        .into_task_with(&runtime);
    body_downloader.set_download_range(from_block..=max_block).expect("set download range");

    (header_downloader, body_downloader, runtime)
}

/// Builds a pipeline with `DefaultStages`.
fn build_pipeline<H, B>(
    provider_factory: reth_provider::ProviderFactory<
        reth_provider::test_utils::MockNodeTypesWithDB,
    >,
    header_downloader: H,
    body_downloader: B,
    max_block: u64,
    tip: B256,
) -> Pipeline<reth_provider::test_utils::MockNodeTypesWithDB>
where
    H: HeaderDownloader<Header = Header> + 'static,
    B: BodyDownloader<Block = Block> + 'static,
{
    let consensus = NoopConsensus::arc();
    let mut stages_config = StageConfig::default();
    stages_config.merkle.rebuild_threshold = 1;
    let evm_config = EthEvmConfig::new(provider_factory.chain_spec());

    let (tip_tx, tip_rx) = watch::channel(B256::ZERO);
    let static_file_producer =
        StaticFileProducer::new(provider_factory.clone(), PruneModes::default());

    let stages = DefaultStages::new(
        provider_factory.clone(),
        tip_rx,
        consensus,
        header_downloader,
        body_downloader,
        evm_config,
        stages_config.clone(),
        PruneModes::default(),
        None,
    )
    .set(StorageCommitGate(StorageHashingStage::new(
        stages_config.storage_hashing,
        stages_config.etl,
    )));

    let pipeline = Pipeline::builder()
        .with_tip_sender(tip_tx)
        .with_max_block(max_block)
        .with_fail_on_unwind(true)
        .add_stages(stages)
        .build(provider_factory, static_file_producer);
    pipeline.set_tip(tip);
    pipeline
}

fn persistent_factory(
    datadir: &Path,
    chain_spec: Arc<reth_chainspec::ChainSpec>,
) -> eyre::Result<ProviderFactory<reth_provider::test_utils::MockNodeTypesWithDB>> {
    let static_files = datadir.join("static_files");
    std::fs::create_dir_all(&static_files)?;
    // The test owns the datadir across child processes. Let TempDatabase close the MDBX handle
    // normally while it cleans only a separate disposable directory on drop.
    let db = reth_db::init_db(datadir.join("db"), reth_db::mdbx::DatabaseArguments::test())?;
    let db =
        Arc::new(reth_db::test_utils::TempDatabase::new(db, reth_db::test_utils::tempdir_path()));
    Ok(ProviderFactory::new(
        db,
        chain_spec,
        StaticFileProviderBuilder::read_write(static_files).build()?,
        RocksDBBuilder::new(datadir.join("rocksdb")).with_default_tables().build()?,
        reth_tasks::Runtime::test(),
    )?)
}

/// Shared helper for pipeline forward sync and unwind tests.
///
/// 1. Pre-funds a signer account and deploys a Counter contract in genesis
/// 2. Each block contains two transactions:
///    - ETH transfer to a recipient (account state changes)
///    - Counter `increment()` call (storage state changes)
/// 3. Runs the full staged pipeline in a persistent storage-v2 database.
async fn run_interrupted_unwind_case(num_blocks: u64, unwind_target: u64) -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let phase = std::env::var(PHASE_ENV).expect("child phase");

    // Generate a keypair for signing transactions
    let key_pair = Keypair::from_seckey_slice(&Secp256k1::new(), &[7u8; 32]).unwrap();
    let signer_address = public_key_to_address(key_pair.public_key());

    // Recipient address for ETH transfers
    let recipient_address = Address::new([0x11; 20]);

    // Create a chain spec with:
    // - Signer pre-funded with 1000 ETH
    // - Counter contract pre-deployed at CONTRACT_ADDRESS
    let initial_balance = U256::from(ETH_TO_WEI) * U256::from(1000);
    let chain_spec = Arc::new(
        ChainSpecBuilder::default()
            .chain(MAINNET.chain)
            .genesis(Genesis {
                alloc: [
                    (
                        signer_address,
                        GenesisAccount { balance: initial_balance, ..Default::default() },
                    ),
                    (
                        CONTRACT_ADDRESS,
                        GenesisAccount {
                            code: Some(COUNTER_DEPLOYED_BYTECODE),
                            ..Default::default()
                        },
                    ),
                ]
                .into(),
                ..MAINNET.genesis.clone()
            })
            .shanghai_activated()
            .build(),
    );

    let provider_factory = create_test_provider_factory_with_chain_spec(chain_spec.clone());
    init_genesis(&provider_factory).expect("init genesis");

    let genesis = provider_factory.sealed_header(0)?.expect("genesis should exist");
    let evm_config = EthEvmConfig::new(chain_spec.clone());

    // Build blocks by actually executing transactions to get correct state roots
    let mut blocks: Vec<SealedBlock<Block>> = Vec::new();
    let mut parent_hash = genesis.hash();

    let gas_price = INITIAL_BASE_FEE as u128;
    let transfer_value = U256::from(ETH_TO_WEI); // 1 ETH per block

    for block_num in 1..=num_blocks {
        // Each block has 2 transactions: ETH transfer + Counter increment
        let base_nonce = (block_num - 1) * 2;

        // Transaction 1: ETH transfer
        let eth_transfer_tx = sign_tx_with_key_pair(
            key_pair,
            Transaction::Eip1559(TxEip1559 {
                chain_id: chain_spec.chain.id(),
                nonce: base_nonce,
                gas_limit: 21_000,
                max_fee_per_gas: gas_price,
                max_priority_fee_per_gas: 0,
                to: TxKind::Call(recipient_address),
                value: transfer_value,
                input: Bytes::new(),
                ..Default::default()
            }),
        );

        // Transaction 2: Counter increment
        let counter_tx = sign_tx_with_key_pair(
            key_pair,
            Transaction::Eip1559(TxEip1559 {
                chain_id: chain_spec.chain.id(),
                nonce: base_nonce + 1,
                gas_limit: 100_000, // Enough gas for SSTORE operations
                max_fee_per_gas: gas_price,
                max_priority_fee_per_gas: 0,
                to: TxKind::Call(CONTRACT_ADDRESS),
                value: U256::ZERO,
                input: Bytes::from(INCREMENT_SELECTOR.to_vec()),
                ..Default::default()
            }),
        );

        let transactions = vec![eth_transfer_tx, counter_tx];
        let tx_root = calculate_transaction_root(&transactions);

        // Build a temporary header for execution
        let temp_header = Header {
            parent_hash,
            number: block_num,
            gas_limit: 30_000_000,
            base_fee_per_gas: Some(INITIAL_BASE_FEE),
            timestamp: block_num * 12,
            ..Default::default()
        };

        // Execute the block to get the state changes
        let provider = provider_factory.database_provider_rw()?;

        let block_with_senders = RecoveredBlock::new_unhashed(
            Block::new(
                temp_header.clone(),
                BlockBody {
                    transactions: transactions.clone(),
                    ommers: Vec::new(),
                    withdrawals: None,
                },
            ),
            vec![signer_address, signer_address], // Both txs from same sender
        );

        // Execute in a scope so state_provider is dropped before we use provider for writes
        let output = {
            let state_provider = provider.latest();
            let db = StateProviderDatabase::new(&*state_provider);
            let executor = evm_config.batch_executor(db);
            executor.execute(&block_with_senders)?
        };

        let gas_used = output.gas_used;

        // Convert bundle state to hashed post state and compute state root
        let hashed_state =
            HashedPostState::from_bundle_state::<KeccakKeyHasher>(output.state.state());
        type TestStateRoot<'a, TX, A> = StateRoot<
            reth_trie_db::DatabaseTrieCursorFactory<&'a TX, A>,
            reth_trie_db::DatabaseHashedCursorFactory<&'a TX>,
        >;
        let (state_root, _trie_updates) = reth_trie_db::with_adapter!(provider, |A| {
            TestStateRoot::<_, A>::overlay_root_with_updates(
                provider.tx_ref(),
                &hashed_state.clone().into_sorted(),
            )
        })?;

        // Create receipts for receipt root calculation (one per transaction)
        let receipts: Vec<_> = output.receipts.iter().map(|r| r.with_bloom_ref()).collect();
        let receipts_root = calculate_receipt_root(&receipts);

        let header = Header {
            parent_hash,
            number: block_num,
            state_root,
            transactions_root: tx_root,
            receipts_root,
            gas_limit: 30_000_000,
            gas_used,
            base_fee_per_gas: Some(INITIAL_BASE_FEE),
            timestamp: block_num * 12,
            ..Default::default()
        };

        let block: SealedBlock<Block> = SealedBlock::seal_parts(
            header.clone(),
            BlockBody { transactions, ommers: Vec::new(), withdrawals: None },
        );

        // Write the plain state to database so subsequent blocks build on it
        let plain_state = output.state.to_plain_state(OriginalValuesKnown::Yes);
        provider.write_state_changes(plain_state)?;
        provider.write_hashed_state(&hashed_state.into_sorted())?;
        provider.commit()?;

        parent_hash = block.hash();
        blocks.push(block);
    }

    // Create a fresh provider factory for the pipeline (clean state from genesis)
    // This is needed because we wrote state during block generation for computing state roots
    let expected_tip = blocks.last().expect("five generated blocks");
    let expected_hash = expected_tip.hash();
    let expected_root = expected_tip.header().state_root;
    let expected_target = &blocks[(unwind_target - 1) as usize];
    let expected_target_hash = expected_target.hash();
    let expected_target_root = expected_target.header().state_root;
    let datadir = std::env::var_os(DATADIR_ENV).expect("persistent datadir");
    let pipeline_provider_factory = persistent_factory(Path::new(&datadir), chain_spec.clone())?;
    if phase == "seed" {
        pipeline_provider_factory.set_storage_settings_cache(StorageSettings::v2());
        init_genesis(&pipeline_provider_factory).expect("init genesis");
    }
    assert!(pipeline_provider_factory.cached_storage_settings().storage_v2);

    if phase == "inspect" {
        let provider = pipeline_provider_factory.provider()?;
        let target_header = provider.sealed_header(unwind_target)?.unwrap();
        assert_eq!(target_header.hash(), expected_target_hash);
        assert_eq!(target_header.state_root, expected_target_root);
        assert_eq!(
            provider.get_stage_checkpoint(StageId::Execution)?.unwrap().block_number,
            num_blocks
        );
        assert_eq!(
            provider.get_stage_checkpoint(StageId::StorageHashing)?.unwrap().block_number,
            unwind_target
        );
        assert_eq!(
            provider.get_stage_checkpoint(StageId::AccountHashing)?.unwrap().block_number,
            num_blocks
        );
        let state = provider.latest();
        assert_eq!(state.basic_account(&signer_address)?.unwrap().nonce, num_blocks * 2);
        assert_eq!(state.storage(CONTRACT_ADDRESS, B256::ZERO)?, Some(U256::from(unwind_target)));
        drop(state);
        drop(provider);
        return Ok(());
    }

    if phase == "verify" {
        let provider = pipeline_provider_factory.provider()?;
        assert_eq!(provider.last_block_number()?, num_blocks);
        let header = provider.sealed_header(num_blocks)?.unwrap();
        assert_eq!(header.hash(), expected_hash);
        assert_eq!(header.state_root, expected_root);
        for stage in [
            StageId::Execution,
            StageId::AccountHashing,
            StageId::StorageHashing,
            StageId::MerkleExecute,
            StageId::Finish,
        ] {
            assert_eq!(
                provider.get_stage_checkpoint(stage)?.unwrap().block_number,
                num_blocks,
                "{stage} checkpoint"
            );
        }
        let state = provider.latest();
        assert_eq!(state.basic_account(&signer_address)?.unwrap().nonce, num_blocks * 2);
        assert_eq!(state.storage(CONTRACT_ADDRESS, B256::ZERO)?, Some(U256::from(num_blocks)));
        drop(state);
        drop(provider);
        return Ok(());
    }

    let recovery_target = if phase == "recover" {
        let (rocksdb, static_file) = pipeline_provider_factory.check_consistency()?;
        eprintln!("startup consistency targets: rocksdb={rocksdb:?}, static_file={static_file:?}");
        rocksdb.into_iter().chain(static_file).min()
    } else {
        None
    };
    let pipeline_genesis =
        pipeline_provider_factory.sealed_header(0)?.expect("genesis should exist");
    let pipeline_consensus = NoopConsensus::arc();

    let blocks_clone = blocks.clone();
    let file_client = create_file_client_from_blocks(blocks);
    let max_block = file_client.max_block().unwrap();
    let tip = file_client.tip().expect("tip");

    let stages_config = StageConfig::default();
    let (header_downloader, body_downloader, mut runtime) = build_downloaders_from_file_client(
        file_client,
        pipeline_genesis,
        1,
        stages_config,
        pipeline_consensus,
        pipeline_provider_factory.clone(),
    );

    let mut pipeline = build_pipeline(
        pipeline_provider_factory.clone(),
        header_downloader,
        body_downloader,
        max_block,
        tip,
    );

    if let Some(target) = recovery_target {
        pipeline.unwind(target, None)?;
        {
            let provider = pipeline_provider_factory.provider()?;
            let target_header = provider.sealed_header(target)?.unwrap();
            assert_eq!(target_header.hash(), expected_target_hash);
            assert_eq!(target_header.state_root, expected_target_root);
            let state = provider.latest();
            assert_eq!(state.basic_account(&signer_address)?.unwrap().nonce, target * 2);
            assert_eq!(state.storage(CONTRACT_ADDRESS, B256::ZERO)?, Some(U256::from(target)));
            for stage in [StageId::Execution, StageId::AccountHashing, StageId::StorageHashing] {
                assert_eq!(provider.get_stage_checkpoint(stage)?.unwrap().block_number, target);
            }
        }
        let file_client = create_file_client_from_blocks(blocks_clone.clone());
        let head = pipeline_provider_factory.sealed_header(target)?.expect("unwind head");
        let (header_downloader, body_downloader, recovery_runtime) =
            build_downloaders_from_file_client(
                file_client,
                head,
                target + 1,
                StageConfig::default(),
                NoopConsensus::arc(),
                pipeline_provider_factory.clone(),
            );
        // The first downloaders are no longer used after the unwind.
        drop(runtime);
        runtime = recovery_runtime;
        pipeline = build_pipeline(
            pipeline_provider_factory.clone(),
            header_downloader,
            body_downloader,
            max_block,
            tip,
        );
    }

    let (mut pipeline, result) = pipeline.run_as_fut(None).await;
    drop(runtime);
    let control = result?;
    if phase == "recover" && control.is_unwind() {
        eyre::bail!("forward recovery rejected the canonical state: {control:?}");
    }

    if phase == "recover" {
        let provider = pipeline_provider_factory.provider()?;
        assert_eq!(provider.last_block_number()?, num_blocks);
        let header = provider.sealed_header(num_blocks)?.unwrap();
        assert_eq!(header.hash(), expected_hash);
        assert_eq!(header.state_root, expected_root);
        for stage in [
            StageId::Execution,
            StageId::AccountHashing,
            StageId::StorageHashing,
            StageId::MerkleExecute,
            StageId::Finish,
        ] {
            assert_eq!(
                provider.get_stage_checkpoint(stage)?.unwrap().block_number,
                num_blocks,
                "{stage} checkpoint"
            );
        }
        let state = provider.latest();
        assert_eq!(state.basic_account(&signer_address)?.unwrap().nonce, num_blocks * 2);
        assert_eq!(state.storage(CONTRACT_ADDRESS, B256::ZERO)?, Some(U256::from(num_blocks)));
        drop(state);
        drop(provider);
        return Ok(());
    }

    // Verify forward sync
    {
        let provider = pipeline_provider_factory.provider()?;
        let last_block = provider.last_block_number()?;
        assert_eq!(last_block, num_blocks, "should have synced {num_blocks} blocks");

        for stage_id in [
            StageId::Headers,
            StageId::Bodies,
            StageId::SenderRecovery,
            StageId::Execution,
            StageId::AccountHashing,
            StageId::StorageHashing,
            StageId::MerkleExecute,
            StageId::TransactionLookup,
            StageId::IndexAccountHistory,
            StageId::IndexStorageHistory,
            StageId::Finish,
        ] {
            let checkpoint = provider.get_stage_checkpoint(stage_id)?;
            assert_eq!(
                checkpoint.map(|c| c.block_number),
                Some(num_blocks),
                "{stage_id} checkpoint should be at block {num_blocks}"
            );
        }

        // Verify the counter contract's storage was updated
        // After num_blocks blocks with 1 increment each, slot 0 should be num_blocks
        let state = provider.latest();
        let counter_storage = state.storage(CONTRACT_ADDRESS, B256::ZERO)?;
        assert_eq!(
            counter_storage,
            Some(U256::from(num_blocks)),
            "Counter storage slot 0 should be {num_blocks} after {num_blocks} increments"
        );
    }

    // Verify changesets are queryable before unwind
    // This validates that the #21561 fix works - unwind needs to read changesets from the correct
    // source
    assert_changesets_queryable(&pipeline_provider_factory, 1..=num_blocks)?;

    // Unwind to unwind_target
    pipeline.unwind(unwind_target, None)?;
    Ok(())
}

#[test]
fn interrupted_storage_v2_unwind_recovers_on_restart() {
    if std::env::var_os(PHASE_ENV).is_some() {
        tokio::runtime::Builder::new_multi_thread()
            .enable_all()
            .build()
            .unwrap()
            .block_on(run_interrupted_unwind_case(5, 2))
            .unwrap();
        return;
    }

    let datadir = tempfile::Builder::new().prefix("op-reth-unwind-").tempdir().unwrap();
    let signal_dir = tempfile::Builder::new().prefix("uw-").tempdir().unwrap();
    let signal = signal_dir.path().join("s");
    let seed_stdout = signal_dir.path().join("seed.stdout");
    let seed_stderr = signal_dir.path().join("seed.stderr");
    let listener = UnixListener::bind(&signal).unwrap();
    let mut child = Command::new(std::env::current_exe().unwrap())
        .args([
            "--exact",
            "interrupted_unwind::interrupted_storage_v2_unwind_recovers_on_restart",
            "--nocapture",
        ])
        .env(PHASE_ENV, "seed")
        .env(DATADIR_ENV, datadir.path())
        .env(SIGNAL_ENV, &signal)
        .stdout(Stdio::from(File::create(&seed_stdout).unwrap()))
        .stderr(Stdio::from(File::create(&seed_stderr).unwrap()))
        .spawn()
        .unwrap();

    let (sent, received) = std::sync::mpsc::sync_channel(1);
    std::thread::spawn(move || {
        let result = listener.accept().and_then(|(mut stream, _)| {
            let mut byte = [0];
            stream.read_exact(&mut byte)?;
            Ok(byte)
        });
        let _ = sent.send(result);
    });
    match received.recv_timeout(Duration::from_secs(90)) {
        Ok(Ok([b'S'])) => {}
        other => {
            child.kill().unwrap();
            child.wait().unwrap();
            panic!(
                "storage stage did not commit: {other:?}\n{}",
                std::fs::read_to_string(&seed_stderr).unwrap_or_default()
            );
        }
    }
    child.kill().unwrap();
    let status = child.wait().unwrap();
    assert!(!status.success(), "the unwind process should be killed");

    for phase in ["inspect", "recover", "verify"] {
        let phase_stdout = signal_dir.path().join(format!("{phase}.stdout"));
        let phase_stderr = signal_dir.path().join(format!("{phase}.stderr"));
        let mut child = Command::new(std::env::current_exe().unwrap())
            .args([
                "--exact",
                "interrupted_unwind::interrupted_storage_v2_unwind_recovers_on_restart",
                "--nocapture",
            ])
            .env(PHASE_ENV, phase)
            .env(DATADIR_ENV, datadir.path())
            .stdout(Stdio::from(File::create(&phase_stdout).unwrap()))
            .stderr(Stdio::from(File::create(&phase_stderr).unwrap()))
            .spawn()
            .unwrap();
        let status = child.wait_timeout(Duration::from_secs(90)).unwrap().unwrap_or_else(|| {
            child.kill().unwrap();
            child.wait().unwrap();
            panic!(
                "{phase} timed out:\n{}",
                std::fs::read_to_string(&phase_stderr).unwrap_or_default()
            );
        });
        let stdout = std::fs::read_to_string(&phase_stdout).unwrap();
        let stderr = std::fs::read_to_string(&phase_stderr).unwrap();
        assert!(status.success(), "{phase} failed:\n{stdout}\n{stderr}",);
        if phase == "recover" {
            assert!(
                stderr.contains("static_file=Some(2)"),
                "startup did not request the persisted unwind: {stderr}",
            );
        }
    }
}
