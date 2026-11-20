use super::{
    MAGIC, MAX_VALUE_LEN, READY, Status,
    client::{self, dummy_header, state_stages},
    read_u32,
    server::Server,
    write_bytes, write_u64,
};
use alloy_consensus::Header;
use alloy_primitives::{Address, B256, U256, keccak256};
use alloy_rlp::Encodable;
use reth_chainspec::EthChainSpec;
use reth_cli_commands::init_state::without_evm::setup_without_evm;
use reth_db::{
    DatabaseEnv,
    test_utils::{TempDatabase, create_test_rw_db_with_datadir, tempdir_path},
};
use reth_db_api::{
    cursor::DbCursorRO,
    models::StorageSettings,
    table::Table,
    tables::{self, RawTable},
    transaction::{DbTx, DbTxMut},
};
use reth_db_common::init::init_genesis_with_settings;
use reth_node_builder::NodeTypesWithDBAdapter;
use reth_optimism_chainspec::{OP_MAINNET, OP_SEPOLIA, OpChainSpec};
use reth_optimism_node::OpNode;
use reth_primitives_traits::{Account, Bytecode, SealedHeader, StorageEntry};
use reth_provider::{
    AccountReader, BlockNumReader, ChainSpecProvider, DBProvider, DatabaseProviderFactory,
    MetadataProvider, ProviderError, ProviderFactory, StageCheckpointReader, StageCheckpointWriter,
    StateProviderFactory, StaticFileProviderFactory, StaticFileSegment,
    providers::{
        BlockchainProvider, ProviderNodeTypes, RocksDBBuilder, StaticFileProviderBuilder,
        StaticFileWriter,
    },
};
use reth_stages::{FinishCheckpoint, StageCheckpoint, StageId};
use reth_trie::StateRoot;
use reth_trie_db::{
    DatabaseHashedCursorFactory, DatabaseStateRoot, DatabaseTrieCursorFactory, PackedKeyAdapter,
};
use std::{
    io::{Read, Write},
    net::{SocketAddr, TcpListener, TcpStream},
    sync::Arc,
    thread,
};

const TOKEN: &[u8] = b"test-token";
const PIVOT: u64 = 10;
/// Well past Cancun on OP Mainnet and OP Sepolia.
const CANCUN_TIMESTAMP: u64 = 1_800_000_000;
/// Commits each table once.
const COMMIT_BYTES: usize = usize::MAX;

type NodeTypes = NodeTypesWithDBAdapter<OpNode, Arc<TempDatabase<DatabaseEnv>>>;
type Factory = ProviderFactory<NodeTypes>;

/// A storage v2 data directory of `chain` holding only the genesis block, as `sync-state` expects.
fn empty_v2_of(chain: &Arc<OpChainSpec>) -> Factory {
    let datadir = tempdir_path();
    let static_files = datadir.join("static_files");
    std::fs::create_dir_all(&static_files).unwrap();
    let factory = ProviderFactory::new(
        create_test_rw_db_with_datadir(&datadir),
        chain.clone(),
        // Small files put block `PIVOT` past the first one, as any real block is.
        StaticFileProviderBuilder::read_write(static_files)
            .with_blocks_per_file(4)
            .build()
            .unwrap(),
        RocksDBBuilder::new(datadir.join("rocksdb")).with_default_tables().build().unwrap(),
        reth_tasks::Runtime::test(),
    )
    .unwrap();
    init_genesis_with_settings(&factory, StorageSettings::v2()).unwrap();
    factory
}

fn empty_v2() -> Factory {
    empty_v2_of(&OP_MAINNET)
}

/// Account `i` of the state [`source_of`] builds. Every third has code, and every seventh
/// delegates to account 1 as EIP-7702 allows.
fn account(i: u8) -> (Address, Account, Bytecode) {
    let code = if i.is_multiple_of(7) {
        let delegation =
            [[0xef, 0x01, 0x00].as_slice(), Address::repeat_byte(1).as_slice()].concat();
        Bytecode::new_raw_checked(delegation.into()).unwrap()
    } else {
        Bytecode::new_raw(vec![0x60, i, 0x00].into())
    };
    let account = Account {
        nonce: i.into(),
        balance: U256::from(i) * U256::from(1_000_000_000u64),
        bytecode_hash: (i.is_multiple_of(3) || i.is_multiple_of(7)).then(|| code.hash_slow()),
    };
    (Address::repeat_byte(i), account, code)
}

/// The value account `i` holds in storage slot `slot`, if it has code.
fn slot_value(slot: u64) -> U256 {
    U256::from(slot + 1)
}

/// A data directory of `chain` at block [`PIVOT`] holding its genesis state plus a few accounts,
/// storage slots and bytecodes, whose header has `timestamp` and commits to `state_root`, or to
/// their real root if `None`.
fn source_of(
    chain: &Arc<OpChainSpec>,
    timestamp: u64,
    state_root: Option<B256>,
) -> (Factory, SealedHeader) {
    let factory = empty_v2_of(chain);
    let provider = factory.database_provider_rw().unwrap();
    let tx = provider.tx_ref();
    for i in 1..=50u8 {
        let (address, account, code) = account(i);
        let hashed_address = keccak256(address);
        tx.put::<tables::HashedAccounts>(hashed_address, account).unwrap();
        if let Some(code_hash) = account.bytecode_hash {
            tx.put::<tables::Bytecodes>(code_hash, code).unwrap();
            for slot in 0..u64::from(i) {
                let key = keccak256(B256::from(U256::from(slot)));
                let entry = StorageEntry { key, value: slot_value(slot) };
                tx.put::<tables::HashedStorages>(hashed_address, entry).unwrap();
            }
        }
    }
    // Without changes to walk, a root computation reuses whatever trie is stored, here genesis's.
    tx.clear::<tables::PackedAccountsTrie>().unwrap();
    tx.clear::<tables::PackedStoragesTrie>().unwrap();
    // Storage v2 keeps the trie in the packed tables.
    let root = StateRoot::<
        DatabaseTrieCursorFactory<_, PackedKeyAdapter>,
        DatabaseHashedCursorFactory<_>,
    >::from_tx(tx)
    .root()
    .unwrap();
    provider.commit().unwrap();

    let header = SealedHeader::seal_slow(Header {
        number: PIVOT,
        timestamp,
        state_root: state_root.unwrap_or(root),
        ..Default::default()
    });
    let provider = factory.database_provider_rw().unwrap();
    setup_without_evm(&provider, header.clone(), dummy_header).unwrap();
    provider.commit().unwrap();
    (factory, header)
}

fn source(state_root: Option<B256>) -> (Factory, SealedHeader) {
    source_of(&OP_MAINNET, CANCUN_TIMESTAMP, state_root)
}

/// Accepts one client on a new port and answers it with `answer`, returning the port and what
/// `answer` returned.
fn serve_one<T: Send + 'static>(
    answer: impl FnOnce(TcpStream) -> T + Send + 'static,
) -> (SocketAddr, thread::JoinHandle<T>) {
    let listener = TcpListener::bind("127.0.0.1:0").unwrap();
    let addr = listener.local_addr().unwrap();
    (addr, thread::spawn(move || answer(listener.accept().unwrap().0)))
}

fn sync(
    target: &Factory,
    token: &[u8],
    commit_bytes: usize,
    addr: SocketAddr,
) -> eyre::Result<SealedHeader> {
    client::sync(target, token, commit_bytes, TcpStream::connect(addr).unwrap())
}

/// Serves `server` to one client syncing `target`, and returns what each side returned.
fn transfer_from(
    server: Server<NodeTypes>,
    target: &Factory,
    token: &[u8],
    commit_bytes: usize,
) -> (eyre::Result<SealedHeader>, eyre::Result<()>) {
    let (addr, served) = serve_one(move |stream| server.serve(stream));
    let synced = sync(target, token, commit_bytes, addr);
    (synced, served.join().unwrap())
}

fn transfer(
    source: Factory,
    target: &Factory,
    token: &[u8],
) -> (eyre::Result<SealedHeader>, eyre::Result<()>) {
    transfer_from(Server::new(source, TOKEN.to_vec(), 1).unwrap(), target, token, COMMIT_BYTES)
}

/// The whole error chain, where the reason for a failure is.
fn chain(err: eyre::Report) -> String {
    format!("{err:#}")
}

fn entries<T: Table, N: ProviderNodeTypes>(
    factory: &ProviderFactory<N>,
) -> Vec<(Vec<u8>, Vec<u8>)> {
    let provider = factory.provider().unwrap();
    let mut cursor = provider.tx_ref().cursor_read::<RawTable<T>>().unwrap();
    cursor
        .walk(None)
        .unwrap()
        .map(|entry| {
            let (key, value) = entry.unwrap();
            (key.raw_key().clone(), value.raw_value().to_vec())
        })
        .collect()
}

/// Persists an empty block on top of `parent`, as a node does once it follows the chain, so the
/// state at `parent` is read as history.
fn persist_empty_child(factory: &Factory, parent: &SealedHeader) {
    let number = parent.number + 1;
    let header = SealedHeader::seal_slow(Header {
        number,
        parent_hash: parent.hash(),
        timestamp: parent.timestamp + 2,
        state_root: parent.state_root,
        ..Default::default()
    });
    let static_files = factory.static_file_provider();
    let mut headers = static_files.get_writer(number, StaticFileSegment::Headers).unwrap();
    headers.append_header(header.header(), &header.hash()).unwrap();
    drop(headers);
    let mut accounts =
        static_files.get_writer(number, StaticFileSegment::AccountChangeSets).unwrap();
    accounts.append_account_changeset(Vec::new(), number).unwrap();
    drop(accounts);
    let mut storages =
        static_files.get_writer(number, StaticFileSegment::StorageChangeSets).unwrap();
    storages.append_storage_changeset(Vec::new(), number).unwrap();
    drop(storages);
    let provider = factory.database_provider_rw().unwrap();
    provider.update_pipeline_stages(number, false).unwrap();
    provider.commit().unwrap();
}

/// Syncs a new data directory of `chain` from a source, committing every `commit_bytes`, and checks
/// it holds the source's state, passes the checks a node runs when it starts, and reads the state
/// as history once the node moves past it.
fn assert_copies_the_state(chain: &Arc<OpChainSpec>, commit_bytes: usize) {
    let (source, header) = source_of(chain, CANCUN_TIMESTAMP, None);
    let expected = (
        entries::<tables::HashedAccounts, _>(&source),
        entries::<tables::HashedStorages, _>(&source),
        entries::<tables::Bytecodes, _>(&source),
    );
    let target = empty_v2_of(chain);

    let server = Server::new(source, TOKEN.to_vec(), 1).unwrap();
    let (synced, served) = transfer_from(server, &target, TOKEN, commit_bytes);
    served.unwrap();
    assert_eq!(synced.unwrap(), header);

    assert_eq!(entries::<tables::HashedAccounts, _>(&target), expected.0);
    assert_eq!(entries::<tables::HashedStorages, _>(&target), expected.1);
    assert_eq!(entries::<tables::Bytecodes, _>(&target), expected.2);
    let provider = target.provider().unwrap();
    for stage in state_stages() {
        assert_eq!(provider.get_stage_checkpoint(stage).unwrap().unwrap().block_number, PIVOT);
    }
    assert!(provider.snap_attempt().unwrap().unwrap().is_verified());
    assert!(provider.tx_ref().entries::<tables::PackedAccountsTrie>().unwrap() > 0);
    drop(provider);
    // The node runs this check on startup and unwinds or panics if it fails.
    assert_eq!(target.check_consistency().unwrap(), (None, None));
    // The changesets start at the file holding the synced block rather than at genesis, which at a
    // real chain's height spares padding every earlier block.
    let static_files = target.static_file_provider();
    for segment in [StaticFileSegment::AccountChangeSets, StaticFileSegment::StorageChangeSets] {
        let first_file = static_files.find_fixed_range(segment, PIVOT).start();
        assert_eq!(static_files.get_lowest_range_start(segment), Some(first_file), "{segment}");
    }

    persist_empty_child(&target, &header);
    let blockchain = BlockchainProvider::new(target).unwrap();
    let (address, account, _) = account(3);
    let state = blockchain.history_by_block_number(PIVOT).unwrap();
    assert_eq!(state.basic_account(&address).unwrap(), Some(account));
    assert_eq!(state.storage(address, B256::ZERO).unwrap(), Some(slot_value(0)));
}

#[test]
fn copies_the_state_and_verifies_its_root() {
    assert_copies_the_state(&OP_MAINNET, COMMIT_BYTES);
}

#[test]
fn copies_the_state_committing_after_every_entry() {
    assert_copies_the_state(&OP_MAINNET, 1);
}

/// Unlike OP Mainnet's, OP Sepolia's genesis holds state, which the sync must replace.
#[test]
fn copies_the_state_over_a_genesis_state() {
    assert_copies_the_state(&OP_SEPOLIA, COMMIT_BYTES);
}

#[test]
fn rejects_a_wrong_token() {
    let (source, _) = source(None);
    let (synced, served) = transfer(source, &empty_v2(), b"wrong-token");
    assert!(chain(synced.unwrap_err()).contains("rejected the token"));
    assert!(served.is_err());
}

#[test]
fn rejects_a_client_of_another_chain() {
    let (source, _) = source(None);
    let target = empty_v2_of(&OP_SEPOLIA);
    assert_ne!(target.chain_spec().genesis_hash(), source.chain_spec().genesis_hash());
    let (synced, served) = transfer(source, &target, TOKEN);
    assert!(chain(synced.unwrap_err()).contains("different chain"));
    assert!(chain(served.unwrap_err()).contains("genesis"));
}

#[test]
fn refuses_clients_beyond_its_transfer_limit() {
    let (source, _) = source(None);
    let server = Arc::new(Server::new(source, TOKEN.to_vec(), 1).unwrap());
    let running = server.start_transfer().unwrap();
    let (addr, served) = {
        let server = server.clone();
        serve_one(move |stream| server.serve(stream))
    };
    let synced = sync(&empty_v2(), TOKEN, COMMIT_BYTES, addr);
    assert!(chain(synced.unwrap_err()).contains("as many transfers"));
    assert!(served.join().unwrap().is_err());

    drop(running);
    let (addr, served) = serve_one(move |stream| server.serve(stream));
    sync(&empty_v2(), TOKEN, COMMIT_BYTES, addr).unwrap();
    served.join().unwrap().unwrap();
}

#[test]
fn refuses_to_serve_state_that_is_not_one_blocks() {
    let lagging_state = StageCheckpoint::new(PIVOT)
        .with_finish_stage_checkpoint(FinishCheckpoint { partial_state_trie: Some(PIVOT - 1) });
    let cases = [
        (StageId::Finish, lagging_state, "state masking"),
        (StageId::MerkleExecute, StageCheckpoint::new(PIVOT - 1), "follows the tip"),
    ];
    for (stage, checkpoint, reason) in cases {
        let (source, _) = source(None);
        let provider = source.database_provider_rw().unwrap();
        provider.save_stage_checkpoint(stage, checkpoint).unwrap();
        provider.commit().unwrap();

        let target = empty_v2();
        let (synced, served) = transfer(source, &target, TOKEN);
        assert!(chain(synced.unwrap_err()).contains("cannot serve"));
        let err = chain(served.unwrap_err());
        assert!(err.contains(reason), "{stage}: {err}");
        assert_eq!(target.provider().unwrap().last_block_number().unwrap(), 0);
    }
}

#[test]
fn refuses_state_from_before_cancun() {
    let (source, _) = source_of(&OP_MAINNET, 1_600_000_000, None);
    let target = empty_v2();
    let (synced, _) = transfer(source, &target, TOKEN);
    assert!(chain(synced.unwrap_err()).contains("before Cancun"));
    assert_eq!(target.provider().unwrap().last_block_number().unwrap(), 0);
}

#[test]
fn rejects_state_whose_root_does_not_match_the_header() {
    let (source, _) = source(Some(B256::repeat_byte(0xab)));
    let target = empty_v2();
    let (synced, served) = transfer(source.clone(), &target, TOKEN);
    served.unwrap();
    let err = chain(synced.unwrap_err());
    assert!(err.to_lowercase().contains("state root"), "{err}");
    assert!(err.contains("delete it"), "{err}");

    let provider = target.database_provider_rw().unwrap();
    for stage in state_stages() {
        assert_eq!(provider.get_stage_checkpoint(stage).unwrap().unwrap().block_number, 0);
    }
    // A node started on what the failed sync left cannot claim the state.
    assert!(matches!(
        provider.save_stage_checkpoint(StageId::Finish, StageCheckpoint::new(PIVOT + 1)),
        Err(ProviderError::UnverifiedSnapState { .. })
    ));
    drop(provider);
    // Nor can a second sync reuse it.
    let (synced, _) = transfer(source, &target, TOKEN);
    assert!(chain(synced.unwrap_err()).contains("delete the data directory"));
}

#[test]
fn rejects_code_that_does_not_match_its_hash() {
    let (source, _) = source(None);
    let provider = source.database_provider_rw().unwrap();
    let (hash, _) =
        provider.tx_ref().cursor_read::<tables::Bytecodes>().unwrap().first().unwrap().unwrap();
    provider.tx_ref().put::<tables::Bytecodes>(hash, Bytecode::new_raw(vec![0xfe].into())).unwrap();
    provider.commit().unwrap();

    let (synced, served) = transfer(source, &empty_v2(), TOKEN);
    served.unwrap();
    assert!(chain(synced.unwrap_err()).contains("with another hash"));
}

/// Compresses `tables` into a checksummed zstd frame, as the server sends them.
fn frame(tables: &[u8]) -> Vec<u8> {
    let mut encoder = zstd::Encoder::new(Vec::new(), 1).unwrap();
    encoder.include_checksum(true).unwrap();
    encoder.write_all(tables).unwrap();
    encoder.finish().unwrap()
}

/// Answers one client like a state server whose tables arrive as `frame`.
fn fake_server(frame: Vec<u8>) -> (SocketAddr, thread::JoinHandle<()>) {
    serve_one(move |mut stream| {
        // The client's opening: magic, genesis hash and token.
        let mut opening = vec![0; MAGIC.len() + 32 + 4 + TOKEN.len()];
        stream.read_exact(&mut opening).unwrap();
        stream.write_all(MAGIC).unwrap();
        stream.write_all(&[Status::Ok as u8]).unwrap();
        let mut rlp = Vec::new();
        Header { number: PIVOT, timestamp: CANCUN_TIMESTAMP, ..Default::default() }
            .encode(&mut rlp);
        write_bytes(&mut stream, &rlp).unwrap();
        write_u64(&mut stream, PIVOT).unwrap();
        stream.read_exact(&mut [0]).unwrap();
        // The client may stop reading as soon as it finds what is wrong.
        let _ = stream.write_all(&frame);
    })
}

#[test]
fn rejects_malformed_streams() {
    let long_key = [33u32.to_le_bytes().as_slice(), &[0; 33]].concat();
    let long_value = [
        32u32.to_le_bytes().as_slice(),
        &[0; 32],
        &u32::try_from(MAX_VALUE_LEN + 1).unwrap().to_le_bytes(),
    ]
    .concat();
    let wrong_count = [0u32.to_le_bytes().as_slice(), &5u64.to_le_bytes()].concat();
    let empty_tables = [0u32.to_le_bytes().as_slice(), &0u64.to_le_bytes()].concat().repeat(3);
    let trailing = [empty_tables.as_slice(), &[1]].concat();
    let mut bad_checksum = frame(&empty_tables);
    *bad_checksum.last_mut().unwrap() ^= 1;

    let cases = [
        (frame(&long_key), "33-byte key"),
        (frame(&long_value), "at most"),
        (frame(&wrong_count), "sent 5 HashedAccounts entries but 0 arrived"),
        (frame(&trailing), "more than the state"),
        (bad_checksum, "checksum"),
    ];
    for (frame, reason) in cases {
        let (addr, served) = fake_server(frame);
        let err = chain(sync(&empty_v2(), TOKEN, COMMIT_BYTES, addr).unwrap_err());
        assert!(err.contains(reason), "{reason}: {err}");
        served.join().unwrap();
    }
}

#[test]
fn streams_the_state_in_a_checksummed_frame() {
    let (source, _) = source(None);
    let server = Server::new(source, TOKEN.to_vec(), 1).unwrap();
    let (addr, served) = serve_one(move |stream| server.serve(stream));
    let mut stream = TcpStream::connect(addr).unwrap();
    stream.write_all(MAGIC).unwrap();
    stream.write_all(OP_MAINNET.genesis_hash().as_slice()).unwrap();
    write_bytes(&mut stream, TOKEN).unwrap();
    let mut answer = [0; MAGIC.len() + 1];
    stream.read_exact(&mut answer).unwrap();
    assert_eq!(answer[MAGIC.len()], Status::Ok as u8);
    let header_len = read_u32(&mut stream).unwrap() as usize;
    // The header, then the finalized block number.
    stream.read_exact(&mut vec![0; header_len + 8]).unwrap();
    stream.write_all(&[READY]).unwrap();
    let mut frame = Vec::new();
    stream.read_to_end(&mut frame).unwrap();
    served.join().unwrap().unwrap();

    // A zstd frame starts with its magic number, then a descriptor whose third bit is set when the
    // frame ends with a checksum.
    assert_eq!(frame[..4], [0x28, 0xb5, 0x2f, 0xfd]);
    assert_ne!(frame[4] & 0b100, 0);
}
