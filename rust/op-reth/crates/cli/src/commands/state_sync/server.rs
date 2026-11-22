//! `op-reth state-server`: serves the state of a data directory to `op-reth sync-state`.

use super::{
    HISTORY, MAGIC, MAX_TOKEN_LEN, PrefetchProgress, READY, STALL_TIMEOUT, SetOnDrop, Status,
    ZSTD_LEVEL, first_recent_block, key_chunk, prefetch, read_array, read_bytes, read_magic,
    read_token, write_bytes, write_u32, write_u64,
};
use alloy_consensus::{EMPTY_ROOT_HASH, Header};
use alloy_primitives::{B256, keccak256};
use alloy_rlp::Encodable;
use clap::Parser;
use eyre::{bail, ensure, eyre};
use reth_chainspec::EthChainSpec;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::common::{AccessRights, CliNodeTypes, Environment, EnvironmentArgs};
use reth_db_api::{
    cursor::DbCursorRO,
    database::Database,
    table::{Decompress, Table},
    tables::{self, RawTable},
    transaction::DbTx,
};
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_primitives::OpPrimitives;
use reth_primitives_traits::{Bytecode, SealedHeader};
use reth_provider::{
    BlockBodyIndicesProvider, BlockReader, ChainSpecProvider, ChainStateBlockReader, DBProvider,
    DatabaseProvider, DatabaseProviderFactory, HeaderProvider, ProviderFactory,
    StageCheckpointReader, StaticFileProviderFactory, StorageSettingsCache,
    providers::ProviderNodeTypes,
};
use reth_stages::StageId;
use socket2::{SockRef, TcpKeepalive};
use std::{
    io::{BufWriter, Write},
    net::{SocketAddr, TcpListener, TcpStream},
    path::PathBuf,
    sync::{
        Arc,
        atomic::{AtomicUsize, Ordering},
    },
    time::{Duration, Instant},
};
use tracing::{info, warn};

/// How long a client has to open the stream.
const HANDSHAKE_TIMEOUT: Duration = Duration::from_secs(30);

/// How long a client has to build its chain up to the block whose state it receives.
const PREPARE_TIMEOUT: Duration = Duration::from_secs(60 * 60);

/// Blocks read from the database at a time while sending bodies.
const BLOCKS_PER_READ: usize = 1024;

/// How long a connection is idle before the server probes whether the client is still there.
const KEEPALIVE_IDLE: Duration = Duration::from_secs(60);

/// How long to wait before accepting again after accepting failed, which it keeps doing while, for
/// example, the process is out of file descriptors.
const ACCEPT_RETRY_DELAY: Duration = Duration::from_millis(100);

/// Serves the state of this node's data directory to `op-reth sync-state`.
///
/// Opens the data directory read-only, so it can run next to the node that owns it.
#[derive(Debug, Parser)]
pub struct StateServerCommand<C: ChainSpecParser> {
    #[command(flatten)]
    env: EnvironmentArgs<C>,

    /// Address to accept `sync-state` connections on. Only this host can connect unless it is
    /// changed, to `0.0.0.0:9555` for example.
    #[arg(long, value_name = "ADDR", default_value = "127.0.0.1:9555")]
    addr: SocketAddr,

    /// File holding the token clients must present.
    #[arg(long, value_name = "FILE")]
    token_file: PathBuf,

    /// Most transfers to run at once. Each holds a read transaction open for its whole duration.
    #[arg(long, value_name = "N", default_value_t = 2)]
    max_transfers: usize,

    /// Threads per transfer reading the state ahead of the one that sends it, so the disk serves
    /// many reads at once. More make a transfer faster but slow the node's own reads on the same
    /// disk; 0 turns it off.
    #[arg(long, value_name = "N", default_value_t = PREFETCH_THREADS)]
    prefetch_threads: usize,
}

impl<C: ChainSpecParser<ChainSpec = OpChainSpec>> StateServerCommand<C> {
    /// Serves connections until the process is stopped.
    pub async fn execute<N: CliNodeTypes<ChainSpec = C::ChainSpec, Primitives = OpPrimitives>>(
        self,
        runtime: reth_tasks::Runtime,
    ) -> eyre::Result<()> {
        let token = read_token(&self.token_file)?;
        // The node that owns the data directory keeps writing to it, so its static files and
        // database can be momentarily out of step; each transfer reads one consistent view.
        let Environment { provider_factory, .. } =
            self.env.init::<N>(AccessRights::RoInconsistent, runtime)?;
        let server = Arc::new(
            Server::new(provider_factory, token, self.max_transfers)?
                .with_prefetch_threads(self.prefetch_threads),
        );

        let listener = TcpListener::bind(self.addr)?;
        info!(target: "reth::cli", addr = %self.addr, "Serving state");
        for stream in listener.incoming() {
            let stream = match stream {
                Ok(stream) => stream,
                Err(err) => {
                    warn!(target: "reth::cli", %err, "Failed to accept a connection");
                    std::thread::sleep(ACCEPT_RETRY_DELAY);
                    continue;
                }
            };
            let server = server.clone();
            let spawned = std::thread::Builder::new().name("state-server".to_string()).spawn(
                move || {
                    let peer = stream.peer_addr().ok();
                    if let Err(err) = server.serve(stream) {
                        warn!(target: "reth::cli", ?peer, err = %format_args!("{err:#}"), "State transfer failed");
                    }
                },
            );
            if let Err(err) = spawned {
                warn!(target: "reth::cli", %err, "Failed to start a transfer");
            }
        }
        Ok(())
    }
}

impl<C: ChainSpecParser> StateServerCommand<C> {
    /// Returns the chain this command serves.
    pub const fn chain_spec(&self) -> Option<&Arc<C::ChainSpec>> {
        Some(&self.env.chain)
    }
}

/// Serves the state of one data directory to any number of clients.
#[derive(Debug)]
pub(super) struct Server<N: ProviderNodeTypes> {
    provider_factory: ProviderFactory<N>,
    token_hash: B256,
    genesis_hash: B256,
    max_transfers: usize,
    transfers: AtomicUsize,
    prefetch_threads: usize,
}

impl<N: ProviderNodeTypes<Primitives = OpPrimitives>> Server<N> {
    /// Creates a server for the storage v2 data directory of `provider_factory`.
    pub(super) fn new(
        provider_factory: ProviderFactory<N>,
        token: Vec<u8>,
        max_transfers: usize,
    ) -> eyre::Result<Self> {
        ensure!(
            provider_factory.cached_storage_settings().use_hashed_state(),
            "state-server needs a storage v2 data directory, whose hashed tables hold the state"
        );
        ensure!(max_transfers > 0, "state-server needs to allow at least one transfer");
        let genesis_hash = provider_factory.chain_spec().genesis_hash();
        Ok(Self {
            provider_factory,
            // Comparing hashes takes the same time however much of a guessed token is right.
            token_hash: keccak256(token),
            genesis_hash,
            max_transfers,
            transfers: AtomicUsize::new(0),
            prefetch_threads: PREFETCH_THREADS,
        })
    }

    /// Sets how many threads per transfer read the state ahead of the one that sends it.
    pub(super) const fn with_prefetch_threads(mut self, prefetch_threads: usize) -> Self {
        self.prefetch_threads = prefetch_threads;
        self
    }

    /// Answers one client and, if it is accepted, streams the state to it.
    pub(super) fn serve(&self, mut stream: TcpStream) -> eyre::Result<()> {
        // A client whose host vanishes would otherwise hold its transfer slot and the read
        // transaction until a timeout expires, up to an hour while it builds its chain.
        SockRef::from(&stream).set_tcp_keepalive(&TcpKeepalive::new().with_time(KEEPALIVE_IDLE))?;
        stream.set_read_timeout(Some(HANDSHAKE_TIMEOUT))?;
        stream.set_write_timeout(Some(STALL_TIMEOUT))?;
        read_magic(&mut stream)?;
        let genesis_hash = B256::from(read_array::<32>(&mut stream)?);
        let token = read_bytes(&mut stream, MAX_TOKEN_LEN)?;
        stream.write_all(MAGIC)?;
        if keccak256(token) != self.token_hash {
            stream.write_all(&[Status::Unauthorized as u8])?;
            bail!("client presented a wrong token");
        }
        if genesis_hash != self.genesis_hash {
            stream.write_all(&[Status::WrongChain as u8])?;
            bail!("client syncs the chain with genesis {genesis_hash}");
        }
        let Some(_slot) = self.start_transfer() else {
            stream.write_all(&[Status::Busy as u8])?;
            bail!("already running {} transfers", self.max_transfers);
        };
        let Snapshot { provider, headers, finalized } = match self.snapshot() {
            Ok(snapshot) => snapshot,
            Err(err) => {
                stream.write_all(&[Status::Unavailable as u8])?;
                return Err(err);
            }
        };
        stream.write_all(&[Status::Ok as u8])?;
        write_u32(&mut stream, u32::try_from(headers.len())?)?;
        for header in &headers {
            let mut rlp = Vec::new();
            header.header().encode(&mut rlp);
            write_bytes(&mut stream, &rlp)?;
        }
        write_u64(&mut stream, finalized)?;
        let header = headers.last().ok_or_else(|| eyre!("no headers to send"))?;

        // The client writes a dummy header for every earlier block before it reads the rest.
        stream.set_read_timeout(Some(PREPARE_TIMEOUT))?;
        ensure!(read_array::<1>(&mut stream)? == [READY], "client sent an unknown message");

        let number = header.number;
        info!(target: "reth::cli", number, hash = %header.hash(), finalized, "Sending state");
        let start = Instant::now();
        let mut encoder = zstd::Encoder::new(stream, ZSTD_LEVEL)?;
        encoder.include_checksum(true)?;
        let mut out = BufWriter::new(encoder);
        // The first header is the boundary below the blocks the client gets whole.
        send_bodies(&provider, &headers[1..], &mut out)?;
        let tx = provider.tx_ref();
        let db = self.provider_factory.db_ref();
        send_table::<tables::HashedAccounts, _>(
            db,
            self.prefetch_threads,
            tx,
            &mut out,
            write_bytes,
        )?;
        send_table::<tables::HashedStorages, _>(
            db,
            self.prefetch_threads,
            tx,
            &mut out,
            write_bytes,
        )?;
        // The client analyzes the code itself rather than trusting the analysis stored here.
        send_table::<tables::Bytecodes, _>(
            db,
            self.prefetch_threads,
            tx,
            &mut out,
            |out, value| write_bytes(out, Bytecode::decompress(value)?.original_byte_slice()),
        )?;
        out.into_inner().map_err(|err| err.into_error())?.finish()?;
        info!(target: "reth::cli", number, elapsed = ?start.elapsed(), "State sent");
        Ok(())
    }

    /// Opens the read transaction a transfer reads everything in, so all of it belongs to the same
    /// block, and returns it with the headers the client gets (see [`history_start`]) and the
    /// finalized block number.
    fn snapshot(&self) -> eyre::Result<Snapshot<N>> {
        // A transfer can outlast the timeout meant for read transactions held open by mistake.
        let provider =
            self.provider_factory.database_provider_ro()?.disable_long_read_transaction_safety();
        let number = state_block(&provider)?;
        let finalized = provider.last_finalized_block_number()?.unwrap_or_default();
        // A read-only static file provider sees the files as they were when it was last indexed.
        self.provider_factory.static_file_provider().initialize_index()?;
        let pivot = provider
            .sealed_header(number)?
            .ok_or_else(|| eyre!("no header for persisted block {number}"))?;
        let first = history_start(&provider, &pivot)?;
        let boundary = first - 1;
        let headers = provider.sealed_headers_range(boundary..=number)?;
        ensure!(
            headers.len() as u64 == number - boundary + 1,
            "missing headers between blocks {boundary} and {number}"
        );
        // Refused here rather than partway through the stream, after the client has built its
        // dummy chain.
        let lacking = |reason: &str| {
            eyre!(
                "the node does not hold the blocks since block {first}, {HISTORY:?} before block \
                 {number}: {reason}; serve once it has followed the chain that long"
            )
        };
        let earliest = self.provider_factory.static_file_provider().earliest_history_height();
        if first < earliest {
            return Err(lacking(&format!("bodies before block {earliest} are pruned")));
        }
        // A block without body indices is a dummy, and one with no transactions whose header
        // commits to some was written without its body, as `sync-state` writes its first blocks.
        let indices = provider.block_body_indices_range(first..=number)?;
        if indices.len() != headers.len() - 1 {
            return Err(lacking("some are dummies"));
        }
        for (header, indices) in headers[1..].iter().zip(&indices) {
            if indices.tx_count == 0 && header.transactions_root != EMPTY_ROOT_HASH {
                return Err(lacking(&format!("block {} has no body", header.number)));
            }
        }
        Ok(Snapshot { provider, headers, finalized })
    }

    /// Takes one of the transfer slots, which the returned guard gives back when dropped, or
    /// returns `None` if all are taken.
    pub(super) fn start_transfer(&self) -> Option<TransferSlot<'_>> {
        self.transfers
            .try_update(Ordering::AcqRel, Ordering::Acquire, |running| {
                (running < self.max_transfers).then_some(running + 1)
            })
            .ok()
            .map(|_| TransferSlot(&self.transfers))
    }
}

/// What one transfer sends, read in one transaction.
struct Snapshot<N: ProviderNodeTypes> {
    /// The read transaction.
    provider: DatabaseProvider<<N::DB as Database>::TX, N>,
    /// The headers the client gets, ending with the one of the block whose state is sent.
    headers: Vec<SealedHeader>,
    /// The node's finalized block number, 0 if it has none.
    finalized: u64,
}

/// A running transfer, counted against `--max-transfers` until dropped.
#[derive(Debug)]
pub(super) struct TransferSlot<'a>(&'a AtomicUsize);

impl Drop for TransferSlot<'_> {
    fn drop(&mut self) {
        self.0.fetch_sub(1, Ordering::AcqRel);
    }
}

/// Returns the block whose state the hashed tables hold, or why they do not hold one block's
/// state.
fn state_block(provider: &impl StageCheckpointReader) -> eyre::Result<u64> {
    let finish = provider.get_stage_checkpoint(StageId::Finish)?.unwrap_or_default();
    let number = finish.block_number;
    ensure!(number > 0, "the node has not persisted any block yet");
    // With state masking, which op-reth forbids but reth allows, the engine persists blocks without
    // their hashed state and holds it in memory.
    if let Some(state) = finish.finish_stage_checkpoint().and_then(|f| f.partial_state_trie()) &&
        state != number
    {
        bail!(
            "the hashed state on disk is a mix of blocks {state} to {number}, which state masking \
             leaves; serve once the node has persisted it, for example after stopping it"
        );
    }
    // A pipeline run commits each stage's progress separately.
    for stage in [
        StageId::Execution,
        StageId::AccountHashing,
        StageId::StorageHashing,
        StageId::MerkleExecute,
    ] {
        let at = provider.get_stage_checkpoint(stage)?.unwrap_or_default().block_number;
        ensure!(
            at == number,
            "the {stage} stage is at block {at} but the node finished block {number}; serve once \
             the node follows the tip"
        );
    }
    Ok(number)
}

/// The first block the client gets whole: the newest block at least [`HISTORY`] older than `pivot`,
/// and no later than the first block `BLOCKHASH` reaches from the block after it.
fn history_start(
    provider: &impl HeaderProvider<Header = Header>,
    pivot: &SealedHeader,
) -> eyre::Result<u64> {
    let cutoff = pivot.timestamp.saturating_sub(HISTORY.as_secs());
    // Timestamps grow along the chain: find the first block past the cutoff.
    let (mut low, mut high) = (1, pivot.number);
    while low < high {
        let mid = low + (high - low) / 2;
        let header =
            provider.header_by_number(mid)?.ok_or_else(|| eyre!("no header for block {mid}"))?;
        if header.timestamp <= cutoff {
            low = mid + 1;
        } else {
            high = mid;
        }
    }
    // The newest block old enough, with one below it to serve as the boundary. On a younger chain,
    // op-node would re-derive from genesis, whose state a synced node does not have.
    let old_enough = low - 1;
    ensure!(old_enough >= 2, "the chain is younger than {HISTORY:?}");
    Ok(old_enough.min(first_recent_block(pivot.number)))
}

/// Writes the RLP body of each of `headers`' blocks, reading a range of blocks at a time.
fn send_bodies<N: ProviderNodeTypes<Primitives = OpPrimitives>>(
    provider: &DatabaseProvider<<N::DB as Database>::TX, N>,
    headers: &[SealedHeader],
    out: &mut impl Write,
) -> eyre::Result<()> {
    let (Some(first), Some(last)) = (headers.first(), headers.last()) else { return Ok(()) };
    for start in (first.number..=last.number).step_by(BLOCKS_PER_READ) {
        let end = (start + BLOCKS_PER_READ as u64 - 1).min(last.number);
        let blocks = provider.block_range(start..=end)?;
        ensure!(
            blocks.len() as u64 == end - start + 1,
            "missing bodies between blocks {start} and {end}"
        );
        for block in blocks {
            write_bytes(out, &alloy_rlp::encode(&block.body))?;
        }
    }
    info!(target: "reth::cli", blocks = headers.len(), "Sent blocks");
    Ok(())
}

/// Writes every entry of `T` in key order, the key as raw database bytes and the value through
/// `write_value`, then the end marker.
///
/// A cursor walking a table larger than the page cache waits on one disk read per page, which
/// leaves the disk idle most of the time. So `prefetch_threads` threads read the entries just ahead
/// of it with their own cursors in `db`, and the pages it reads next are already cached.
fn send_table<T: Table, W: Write>(
    db: &impl Database,
    prefetch_threads: usize,
    tx: &impl DbTx,
    out: &mut W,
    write_value: impl Fn(&mut W, &[u8]) -> eyre::Result<()>,
) -> eyre::Result<()> {
    let progress = PrefetchProgress::default();
    std::thread::scope(|scope| {
        let prefetchers: Vec<_> =
            (0..prefetch_threads).map(|_| scope.spawn(|| prefetch::<T>(db, &progress))).collect();
        let sent = {
            // Prefetchers stop once the table is sent, even if sending panics.
            let _done = SetOnDrop(&progress.done);
            send_entries::<T, W>(tx, out, &write_value, &progress)
        };
        for prefetcher in prefetchers {
            // A failed prefetch only slows the transfer down.
            match prefetcher.join() {
                Ok(Ok(())) => {}
                Ok(Err(err)) => {
                    warn!(target: "reth::cli", table = T::NAME, err = %format_args!("{err:#}"), "Prefetch failed")
                }
                Err(_) => warn!(target: "reth::cli", table = T::NAME, "Prefetch panicked"),
            }
        }
        sent
    })
}

/// Writes the entries of `T` and the end marker, recording which chunk it has reached.
fn send_entries<T: Table, W: Write>(
    tx: &impl DbTx,
    out: &mut W,
    write_value: &impl Fn(&mut W, &[u8]) -> eyre::Result<()>,
    progress: &PrefetchProgress,
) -> eyre::Result<()> {
    let mut cursor = tx.cursor_read::<RawTable<T>>()?;
    let mut count = 0u64;
    let mut entry = cursor.first()?;
    while let Some((key, value)) = entry {
        if count.is_multiple_of(1024) {
            progress.position.store(key_chunk(key.raw_key()), Ordering::Relaxed);
        }
        write_bytes(out, key.raw_key())?;
        write_value(out, value.raw_value())?;
        count += 1;
        entry = cursor.next()?;
    }
    write_u32(out, 0)?;
    write_u64(out, count)?;
    info!(target: "reth::cli", table = T::NAME, entries = count, "Sent table");
    Ok(())
}

/// Threads reading a table ahead of the cursor that sends it, unless `--prefetch-threads` says
/// otherwise.
const PREFETCH_THREADS: usize = 16;
