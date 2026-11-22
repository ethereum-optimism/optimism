//! `op-reth sync-state`: initializes an empty data directory with the state of an
//! `op-reth state-server`.

use super::{
    HISTORY, KEY_LEN, MAGIC, MAX_BODY_LEN, MAX_HEADER_LEN, MAX_HEADERS, MAX_VALUE_LEN, READY,
    STALL_TIMEOUT, Status, first_recent_block, read_array, read_bytes, read_magic, read_token,
    read_u32, read_u64, read_vec, write_bytes,
};
use alloy_consensus::{BlockHeader, Header};
use alloy_primitives::B256;
use clap::Parser;
use eyre::{WrapErr, bail, ensure, eyre};
use reth_chainspec::{EthChainSpec, EthereumHardforks};
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::{
    common::{AccessRights, CliNodeTypes, Environment, EnvironmentArgs},
    init_state::without_evm::setup_without_evm,
};
use reth_db_api::{
    DatabaseError,
    cursor::{DbCursorRW, DbDupCursorRW},
    database::Database,
    models::SnapAttempt,
    table::Table,
    tables::{self, RawDupSort, RawKey, RawTable, RawValue},
    transaction::DbTxMut,
};
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_consensus::validation::validate_body_against_header_op;
use reth_optimism_primitives::{OpBlock, OpBlockBody, OpPrimitives};
use reth_primitives_traits::{Bytecode, SealedBlock, SealedHeader, header::HeaderMut};
use reth_provider::{
    BlockBodyIndicesProvider, BlockNumReader, BlockWriter, ChainSpecProvider, DBProvider,
    DatabaseProvider, DatabaseProviderFactory, MetadataProvider, MetadataWriter, ProviderFactory,
    ProviderResult, PruneCheckpointWriter, StageCheckpointWriter, StaticFileProviderFactory,
    StaticFileSegment, StorageSettingsCache,
    providers::{
        ProviderNodeTypes, StaticFileProvider, StaticFileProviderRWRefMut, StaticFileWriter,
    },
};
use reth_prune_types::{PruneCheckpoint, PruneMode, PruneSegment};
use reth_stages::{ExecInput, Stage, StageCheckpoint, StageId, stages::MerkleStage};
use std::{
    io::{BufReader, Read, Write},
    net::TcpStream,
    path::PathBuf,
    sync::Arc,
    time::Instant,
};
use tracing::{info, warn};

/// Bytes of entries written per database transaction while loading a table.
const COMMIT_BYTES: usize = 256 * 1024 * 1024;

/// Size of the buffer between the decompressor and the many small reads of the entries.
const READ_BUFFER: usize = 1024 * 1024;

/// Initializes an empty data directory with the state of an `op-reth state-server`, so the node
/// starts syncing from the block that state belongs to.
///
/// If it fails after it starts writing, delete the data directory before running it again.
#[derive(Debug, Parser)]
pub struct SyncStateCommand<C: ChainSpecParser> {
    #[command(flatten)]
    env: EnvironmentArgs<C>,

    /// `host:port` of the `state-server` to copy the state from.
    #[arg(long, value_name = "ADDR")]
    from: String,

    /// File holding the token the server expects.
    #[arg(long, value_name = "FILE")]
    token_file: PathBuf,
}

impl<C: ChainSpecParser<ChainSpec = OpChainSpec>> SyncStateCommand<C> {
    /// Copies the server's state into this data directory.
    pub async fn execute<N: CliNodeTypes<ChainSpec = C::ChainSpec, Primitives = OpPrimitives>>(
        self,
        runtime: reth_tasks::Runtime,
    ) -> eyre::Result<()> {
        let token = read_token(&self.token_file)?;
        let Environment { provider_factory, .. } = self.env.init::<N>(AccessRights::RW, runtime)?;
        let stream = TcpStream::connect(&self.from)?;
        let header = sync(&provider_factory, &token, COMMIT_BYTES, stream)?;
        info!(target: "reth::cli", number = header.number(), hash = %header.hash(), "State synced");
        Ok(())
    }
}

impl<C: ChainSpecParser> SyncStateCommand<C> {
    /// Returns the chain this command initializes.
    pub const fn chain_spec(&self) -> Option<&Arc<C::ChainSpec>> {
        Some(&self.env.chain)
    }
}

/// Loads the state the server streams over `stream` into the empty data directory of
/// `provider_factory`, committing every `commit_bytes` of entries, and returns the header of the
/// block it belongs to.
pub(super) fn sync<N: ProviderNodeTypes<Primitives = OpPrimitives, ChainSpec = OpChainSpec>>(
    provider_factory: &ProviderFactory<N>,
    token: &[u8],
    commit_bytes: usize,
    mut stream: TcpStream,
) -> eyre::Result<SealedHeader> {
    ensure!(
        provider_factory.cached_storage_settings().use_hashed_state(),
        "sync-state needs a storage v2 data directory, whose hashed tables hold the state"
    );
    let last = provider_factory.provider()?.last_block_number()?;
    ensure!(
        last == 0,
        "the data directory holds blocks up to {last}, but sync-state only initializes an empty \
         one; if an earlier sync-state failed, delete the data directory and run it again"
    );

    handshake(&mut stream, provider_factory.chain_spec().genesis_hash(), token)?;
    let headers = read_headers(&mut stream, &provider_factory.chain_spec())?;
    load(provider_factory, &headers, commit_bytes, stream).wrap_err(
        "sync-state failed after writing to the data directory; delete it before running again",
    )?;
    Ok(pivot(&headers)?.clone())
}

/// The header of the block whose state is sent, which comes last.
fn pivot(headers: &[SealedHeader]) -> eyre::Result<&SealedHeader> {
    headers.last().ok_or_else(|| eyre!("the server sent no header"))
}

/// Opens the stream, and returns once the server has accepted this client.
fn handshake(stream: &mut TcpStream, genesis_hash: B256, token: &[u8]) -> eyre::Result<()> {
    stream.set_read_timeout(Some(STALL_TIMEOUT))?;
    stream.write_all(MAGIC)?;
    stream.write_all(genesis_hash.as_slice())?;
    write_bytes(stream, token)?;
    read_magic(stream)?;
    let refusal = match Status::try_from(read_array::<1>(stream)?[0])? {
        Status::Ok => return Ok(()),
        Status::Unauthorized => "the state server rejected the token",
        Status::WrongChain => "the state server serves a different chain",
        Status::Busy => "the state server is running as many transfers as it allows",
        Status::Unavailable => "the state server cannot serve its state now; its log says why",
    };
    Err(eyre!(refusal))
}

/// Reads the headers of the blocks up to the one whose state follows, and checks that they link up
/// to it, cover the `HISTORY` and `BLOCKHASH` reach before it, and that this node can start from
/// that block. The first header is the boundary below the blocks whose bodies follow.
fn read_headers(
    input: &mut impl Read,
    chain_spec: &OpChainSpec,
) -> eyre::Result<Vec<SealedHeader>> {
    let count = u64::from(read_u32(input)?);
    ensure!(
        (2..=MAX_HEADERS).contains(&count),
        "the server sent {count} headers, not 2 to {MAX_HEADERS}"
    );
    let mut headers: Vec<SealedHeader> = Vec::new();
    for _ in 0..count {
        let header: Header = alloy_rlp::decode_exact(read_bytes(input, MAX_HEADER_LEN)?)?;
        let header = SealedHeader::seal_slow(header);
        if let Some(parent) = headers.last() {
            ensure!(
                header.parent_hash() == parent.hash() && header.number() == parent.number() + 1,
                "the server's header of block {} does not follow block {}",
                header.number(),
                parent.number()
            );
        }
        headers.push(header);
    }
    let finalized = read_u64(input)?;
    let header = pivot(&headers)?;
    let number = header.number();
    let (boundary, first) = (&headers[0], &headers[1]);
    ensure!(boundary.number() > 0, "the server's headers start at genesis");
    ensure!(
        first.timestamp() <= header.timestamp().saturating_sub(HISTORY.as_secs()),
        "the server's blocks start at block {}, less than {HISTORY:?} before block {number}",
        first.number()
    );
    ensure!(
        first.number() <= first_recent_block(number),
        "the server's blocks start at block {}, after block {}, the first BLOCKHASH reaches",
        first.number(),
        first_recent_block(number)
    );
    ensure!(
        boundary.number() > 1 || boundary.parent_hash() == chain_spec.genesis_hash(),
        "the server's header of block 1 does not follow this chain's genesis"
    );
    // Before Cancun, SELFDESTRUCT can wipe storage that came with the state, which needs the
    // plain slot keys of the slot preimage database; the server does not send them.
    ensure!(
        chain_spec.is_cancun_active_at_timestamp(header.timestamp()),
        "block {number} is before Cancun, whose state also needs slot preimages"
    );
    if number > finalized {
        warn!(
            target: "reth::cli",
            number,
            finalized,
            "The server has not finalized the block whose state it sends. If a reorg removes it, \
             this node cannot unwind below it: delete the data directory and sync again"
        );
    }
    info!(target: "reth::cli", number, hash = %header.hash(), blocks = count - 1, "Syncing state");
    Ok(headers)
}

/// Writes the chain of `headers`, receiving the bodies of all but the first, then receives the
/// state of the last and verifies it.
fn load<N: ProviderNodeTypes<Primitives = OpPrimitives, ChainSpec = OpChainSpec>>(
    provider_factory: &ProviderFactory<N>,
    headers: &[SealedHeader],
    commit_bytes: usize,
    mut stream: TcpStream,
) -> eyre::Result<()> {
    let (boundary, blocks) = headers.split_first().ok_or_else(|| eyre!("no headers to write"))?;
    // Writing each block also marks every stage done at it. One transaction, up to the guard that
    // `prepare` records, keeps a sync that stops partway from leaving blocks that claim state. It
    // holds the transaction hash index of every block in memory until it commits, some hundreds
    // of megabytes for OP Mainnet's `HISTORY`.
    let provider_rw = provider_factory.database_provider_rw()?;
    setup_without_evm(&provider_rw, boundary.clone(), dummy_header)?;
    stream.write_all(&[READY])?;
    let mut input = BufReader::with_capacity(READ_BUFFER, zstd::Decoder::new(stream)?);
    receive_blocks(&mut input, &provider_rw, &provider_factory.chain_spec(), blocks)?;
    prepare(&provider_rw, pivot(headers)?)?;
    // Commits the static files first, so the merkle stage sees the header it checks the root of.
    provider_rw.commit()?;

    let start = Instant::now();
    receive(
        &mut input,
        provider_factory,
        commit_bytes,
        tables::HashedAccounts::NAME,
        |tx| tx.cursor_write::<RawTable<tables::HashedAccounts>>(),
        |cursor, key, value| Ok(cursor.append(RawKey::from_vec(key), &RawValue::from_vec(value))?),
    )?;
    receive(
        &mut input,
        provider_factory,
        commit_bytes,
        tables::HashedStorages::NAME,
        |tx| tx.cursor_dup_write::<RawDupSort<tables::HashedStorages>>(),
        |cursor, key, value| {
            Ok(cursor.append_dup(RawKey::from_vec(key), RawValue::from_vec(value))?)
        },
    )?;
    receive(
        &mut input,
        provider_factory,
        commit_bytes,
        tables::Bytecodes::NAME,
        |tx| tx.cursor_write::<tables::Bytecodes>(),
        |cursor, key, code| {
            // Analyzed here rather than trusted, and checked against its hash, since the state
            // root commits to code hashes but not to code.
            let code = Bytecode::new_raw_checked(code.into())?;
            let hash = B256::from_slice(&key);
            ensure!(code.hash_slow() == hash, "the server sent code for {hash} with another hash");
            Ok(cursor.append(hash, &code)?)
        },
    )?;
    // Reading past the end makes the decompressor check the frame's checksum.
    ensure!(input.read(&mut [0])? == 0, "the server sent more than the state");
    info!(target: "reth::cli", elapsed = ?start.elapsed(), "State received, rebuilding the trie");

    let number = pivot(headers)?.number();
    rebuild_trie(provider_factory, number)?;
    finish(provider_factory, number)
}

/// The stages whose checkpoint claims the state. `Finish` also sets the best block a node reports.
pub(super) fn state_stages() -> impl Iterator<Item = StageId> {
    StageId::STATE_REQUIRED.into_iter().chain([StageId::Finish])
}

/// A header for a block before the first one whose state the node has.
pub(super) fn dummy_header(number: u64) -> Header {
    let mut header = Header::default();
    header.set_number(number);
    header
}

/// Writes `block` after the chain's last block, without receipts or state changes, as
/// `init-state --without-evm` writes its first block.
pub(super) fn append_block<N: ProviderNodeTypes<Primitives = OpPrimitives>>(
    provider_rw: &DatabaseProvider<<N::DB as Database>::TXMut, N>,
    block: SealedBlock<OpBlock>,
) -> eyre::Result<()> {
    let number = block.number();
    let block = block
        .try_recover()
        .map_err(|_| eyre!("a transaction of block {number} has no recoverable sender"))?;
    provider_rw.insert_block(&block)?;
    provider_rw
        .static_file_provider()
        .latest_writer(StaticFileSegment::Receipts)?
        .increment_block(number)?;
    Ok(())
}

/// Reads the body of each of `headers`' blocks, checks it against its header, and writes the block.
fn receive_blocks<N: ProviderNodeTypes<Primitives = OpPrimitives>>(
    input: &mut impl Read,
    provider_rw: &DatabaseProvider<<N::DB as Database>::TXMut, N>,
    chain_spec: &OpChainSpec,
    headers: &[SealedHeader],
) -> eyre::Result<()> {
    let start = Instant::now();
    for (received, header) in headers.iter().enumerate() {
        if received > 0 && received.is_multiple_of(1000) {
            info!(target: "reth::cli", received, blocks = headers.len(), "Receiving blocks");
        }
        let body: OpBlockBody = alloy_rlp::decode_exact(read_bytes(input, MAX_BODY_LEN)?)?;
        validate_body_against_header_op(chain_spec, &body, header.header()).map_err(|err| {
            eyre!("the server's body of block {} does not match its header: {err}", header.number())
        })?;
        append_block(provider_rw, SealedBlock::from_sealed_parts(header.clone(), body))?;
    }
    info!(target: "reth::cli", blocks = headers.len(), elapsed = ?start.elapsed(), "Received blocks");
    Ok(())
}

/// Marks the stages that need the state of `pivot` not started and every other stage done at it,
/// pads the changesets up to it, and clears the genesis state.
fn prepare<N: ProviderNodeTypes<Primitives = OpPrimitives>>(
    provider_rw: &DatabaseProvider<<N::DB as Database>::TXMut, N>,
    pivot: &SealedHeader,
) -> eyre::Result<()> {
    // Until the root is verified, the stages that need the state are marked not started, and an
    // unverified snap attempt stops a node from advancing its Finish checkpoint. A node started on
    // what a failed sync leaves still runs until it tries to.
    let needs_state: Vec<StageId> = state_stages().collect();
    for stage in StageId::ALL {
        let done = if needs_state.contains(&stage) { 0 } else { pivot.number() };
        provider_rw.save_stage_checkpoint(stage, StageCheckpoint::new(done))?;
    }
    let attempt =
        SnapAttempt::start(provider_rw.snap_attempt()?, pivot.num_hash(), pivot.state_root());
    provider_rw.write_snap_attempt(&attempt)?;

    let static_files = provider_rw.static_file_provider();
    pad_changesets(&static_files, StaticFileSegment::AccountChangeSets, pivot.number(), |w, b| {
        w.append_account_changeset(Vec::new(), b)
    })?;
    pad_changesets(&static_files, StaticFileSegment::StorageChangeSets, pivot.number(), |w, b| {
        w.append_storage_changeset(Vec::new(), b)
    })?;

    // Initializing the data directory wrote the genesis state; only the pivot's belongs here.
    let tx = provider_rw.tx_ref();
    tx.clear::<tables::HashedAccounts>()?;
    tx.clear::<tables::HashedStorages>()?;
    tx.clear::<tables::Bytecodes>()?;
    tx.clear::<tables::PlainAccountState>()?;
    tx.clear::<tables::PlainStorageState>()?;
    Ok(())
}

/// Ends the changeset static files of `segment` with empty changesets up to `number`, as reth's
/// state import does, so they reach the same block as the headers. The files from genesis are
/// deleted first, so the new ones start at the file range holding `number` and the padding stays
/// within one file.
fn pad_changesets(
    static_files: &StaticFileProvider<OpPrimitives>,
    segment: StaticFileSegment,
    number: u64,
    append_empty: impl Fn(&mut StaticFileProviderRWRefMut<'_, OpPrimitives>, u64) -> ProviderResult<()>,
) -> ProviderResult<()> {
    if static_files.get_highest_static_file_block(segment).is_some_and(|highest| highest < number) {
        static_files.delete_segment(segment)?;
    }
    let mut writer = static_files.get_writer(number, segment)?;
    for block in writer.next_block_number()..=number {
        append_empty(&mut writer, block)?;
    }
    Ok(())
}

/// Reads one table's entries and writes each through a cursor `open` creates, committing every
/// `commit_bytes` of entries.
fn receive<N: ProviderNodeTypes, C>(
    input: &mut impl Read,
    provider_factory: &ProviderFactory<N>,
    commit_bytes: usize,
    table: &str,
    open: impl Fn(&<N::DB as Database>::TXMut) -> Result<C, DatabaseError>,
    append: impl Fn(&mut C, Vec<u8>, Vec<u8>) -> eyre::Result<()>,
) -> eyre::Result<()> {
    let mut count = 0u64;
    loop {
        let provider_rw = provider_factory.database_provider_rw()?;
        let mut cursor = open(provider_rw.tx_ref())?;
        let ended = receive_chunk(input, &mut cursor, &append, commit_bytes, &mut count, table)?;
        drop(cursor);
        provider_rw.commit()?;
        if ended {
            info!(target: "reth::cli", table, entries = count, "Received table");
            return Ok(());
        }
    }
}

/// Appends entries until they add up to `commit_bytes` or the table ends, and returns whether it
/// ended.
fn receive_chunk<C>(
    input: &mut impl Read,
    cursor: &mut C,
    append: &impl Fn(&mut C, Vec<u8>, Vec<u8>) -> eyre::Result<()>,
    commit_bytes: usize,
    count: &mut u64,
    table: &str,
) -> eyre::Result<bool> {
    let mut bytes = 0;
    loop {
        let Some((key, value)) = read_entry(input)? else {
            let sent = read_u64(input)?;
            ensure!(sent == *count, "the server sent {sent} {table} entries but {count} arrived");
            return Ok(true);
        };
        bytes += key.len() + value.len();
        append(cursor, key, value)?;
        *count += 1;
        if bytes >= commit_bytes {
            return Ok(false);
        }
    }
}

/// Reads one entry, or `None` at the end of the table.
fn read_entry(input: &mut impl Read) -> eyre::Result<Option<(Vec<u8>, Vec<u8>)>> {
    let key_len = read_u32(input)?;
    if key_len == 0 {
        return Ok(None);
    }
    ensure!(key_len == KEY_LEN, "the server sent a {key_len}-byte key, not a {KEY_LEN}-byte hash");
    let key = read_vec(input, KEY_LEN as usize)?;
    Ok(Some((key, read_bytes(input, MAX_VALUE_LEN)?)))
}

/// Builds the trie from the loaded hashed state with the merkle stage, which fails unless its
/// root matches the state root of block `number`.
fn rebuild_trie<N: ProviderNodeTypes>(
    provider_factory: &ProviderFactory<N>,
    number: u64,
) -> eyre::Result<()> {
    let mut stage = MerkleStage::default_execution();
    // From block 0 the stage rebuilds the whole trie instead of applying changesets.
    let mut checkpoint = StageCheckpoint::new(0);
    loop {
        let provider_rw = provider_factory.database_provider_rw()?;
        let output = stage.execute(
            &provider_rw,
            ExecInput { target: Some(number), checkpoint: Some(checkpoint) },
        )?;
        provider_rw.save_stage_checkpoint(StageId::MerkleExecute, output.checkpoint)?;
        provider_rw.commit()?;
        if output.done {
            return Ok(());
        }
        checkpoint = output.checkpoint;
    }
}

/// Marks the verified state as the state of block `number`.
fn finish<N: ProviderNodeTypes>(
    provider_factory: &ProviderFactory<N>,
    number: u64,
) -> eyre::Result<()> {
    let provider_rw = provider_factory.database_provider_rw()?;
    let Some(mut attempt) = provider_rw.snap_attempt()? else {
        bail!("the record of this state sync disappeared");
    };
    attempt.verify();
    provider_rw.write_snap_attempt(&attempt)?;
    for stage in state_stages() {
        provider_rw.save_stage_checkpoint(stage, StageCheckpoint::new(number))?;
    }
    // The node has no history before this block, so a read at a later block finds no record of a
    // key that has not changed since and would take it to be absent. Marking that history pruned
    // makes such reads use the state instead.
    for segment in [PruneSegment::AccountHistory, PruneSegment::StorageHistory] {
        let checkpoint = PruneCheckpoint {
            block_number: Some(number),
            tx_number: None,
            prune_mode: PruneMode::Before(number + 1),
        };
        provider_rw.save_prune_checkpoint(segment, checkpoint)?;
    }
    // Nor has it the receipts of the blocks it got whole, which would otherwise read as missing
    // rather than pruned.
    let last_tx = provider_rw.block_body_indices(number)?.map(|indices| indices.last_tx_num());
    let checkpoint = PruneCheckpoint {
        block_number: Some(number),
        tx_number: last_tx,
        prune_mode: PruneMode::Before(number + 1),
    };
    provider_rw.save_prune_checkpoint(PruneSegment::Receipts, checkpoint)?;
    provider_rw.commit()?;
    Ok(())
}
