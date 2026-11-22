//! Trusted state sync: copies the state of a trusted op-reth node into an empty data directory over
//! one TCP stream, so a new node starts near the tip instead of executing every block since
//! Bedrock.
//!
//! `op-reth state-server` runs next to the trusted node and opens its data directory read-only.
//! `op-reth sync-state` connects to it, writes the server's latest persisted header as its first
//! block (on top of a dummy chain, as `init-state --without-evm` does), bulk-loads the hashed state
//! in key order and rebuilds the trie with the merkle stage, which accepts the state only if its
//! root matches that header. The node then syncs the remaining blocks as usual.
//!
//! # Trust
//!
//! The root check makes a partial or corrupt transfer fail. The client analyzes every bytecode it
//! receives itself and checks it against its hash, which the root does not cover; it does not check
//! that every account's code arrived. What none of this can check is that the header is canonical:
//! that is the trust placed in the server, and it is confirmed once the node syncs past the header,
//! since every later canonical block must descend from it.
//!
//! The header is the server's latest persisted block, which is normally newer than its finalized
//! block. A node cannot unwind below its first block, so if a reorg removes that block, delete the
//! data directory and sync again. The client warns when the block is not finalized.
//!
//! The token keeps strangers from reading the state; the stream is neither encrypted nor
//! authenticated. The server listens on localhost unless told otherwise: run it on a network you
//! trust, or tunnel it.
//!
//! # The synced node
//!
//! The node has no state history before the block it starts from. The client marks the account and
//! storage history up to it as pruned, so that a read at a later block finds a key that has not
//! changed since in the state, rather than taking it to be absent.
//!
//! The server also sends the blocks of the `HISTORY` before that block, with their transactions
//! but without receipts. op-node needs them: once a node has synced, it re-derives the blocks of
//! that span (its default `--syncmode.offset-el-safe`; a larger offset needs a longer history
//! than this client gets) from L1. They also give the EVM's
//! `BLOCKHASH` the real hashes of the blocks it reads while the node executes the blocks after.
//! The client checks that their headers link up to the block's header and their bodies match
//! their headers. Below them, it writes a dummy header per block, as `init-state --without-evm`
//! does, which takes minutes at OP Mainnet's height; the server waits for it. The node has the
//! blocks of that span but not their state, so state queries there, such as `eth_getProof` or
//! `optimism_outputAtBlock` for the heads op-node starts from, fail until its safe head passes the
//! block whose state it got.
//!
//! # The source node
//!
//! The server sends the hashed state tables as they are on disk, so they must hold the state of
//! exactly one block. They do once an op-reth node follows the tip: op-reth does not let the
//! engine persist blocks without their hashed state (state masking, which reth allows). The server
//! refuses while the node is syncing with the pipeline, whose stages do not finish at the same
//! block, and if the data directory shows state masking.
//!
//! The server refuses to serve until the node holds the blocks of the `HISTORY` before the block,
//! which a node synced with `sync-state` does once it has followed the chain that long.
//!
//! Each transfer reads the whole state in one database transaction, which is also open while the
//! client builds its dummy chain. While it is open, the source's database cannot reuse the pages
//! the node frees, so it grows by what the node writes meanwhile.
//!
//! # Wire format
//!
//! Integers are little-endian, and byte strings are a `u32` length followed by the bytes.
//!
//! 1. The client opens with `MAGIC`, the hash of its chain's genesis block (32 bytes) and its
//!    token.
//! 2. The server answers with `MAGIC` and a `Status` byte. After `Status::Ok` it sends the number
//!    of headers that follow (`u32`), the RLP headers, each a byte string, of the blocks from the
//!    one before the `HISTORY` up to the one whose state follows, then its finalized block number
//!    (`u64`, 0 if it has none).
//! 3. The client builds its dummy chain up to the first of those blocks and sends `READY`.
//! 4. The rest of the stream is one checksummed zstd frame holding the RLP bodies of the blocks
//!    after the first header, each a byte string, then, for each of the `HashedAccounts`,
//!    `HashedStorages` and `Bytecodes` tables, which with storage v2 hold the whole state, its
//!    entries in key order: a key and a value, each a byte string. Keys and the values of the first
//!    two tables are raw database bytes; a bytecode's value is its code. A zero key length ends the
//!    table and is followed by its entry count (`u64`).

pub mod client;
pub mod server;

use eyre::{ensure, eyre};
use std::{
    io::{Read, Write},
    path::Path,
    sync::atomic::{AtomicBool, Ordering},
    time::Duration,
};

/// Opens both directions of the stream: protocol name and version.
const MAGIC: &[u8; 8] = b"OPSTATE2";

/// Sent by the client once its chain reaches the block whose state follows.
const READY: u8 = 1;

/// Longest token accepted, so a stray connection cannot make the server allocate much.
const MAX_TOKEN_LEN: usize = 1024;

/// How many blocks back the EVM's `BLOCKHASH` reaches.
#[cfg(not(test))]
const BLOCKHASH_WINDOW: u64 = 256;
/// Smaller in tests, where writing each real block costs several file syncs.
#[cfg(test)]
const BLOCKHASH_WINDOW: u64 = 8;

/// The first of the blocks whose real headers a node starting after block `number` needs: those
/// `BLOCKHASH` reaches from block `number + 1`, which end with block `number`.
fn first_recent_block(number: u64) -> u64 {
    (number + 1).saturating_sub(BLOCKHASH_WINDOW).max(1)
}

/// How far back before the block whose state is sent the client gets whole blocks: op-node's
/// default `--syncmode.offset-el-safe`, the span it re-derives from L1 once a node has synced.
#[cfg(not(test))]
const HISTORY: Duration = Duration::from_secs(12 * 60 * 60);
/// Shorter in tests, where writing each block costs several file syncs.
#[cfg(test)]
const HISTORY: Duration = Duration::from_secs(20);

/// Most headers accepted: [`HISTORY`] of blocks at a quarter of a second each, and more.
const MAX_HEADERS: u64 = 1 << 18;

/// Longest header accepted, several times an OP header's size, which bounds the [`MAX_HEADERS`]
/// headers a client holds before checking them.
const MAX_HEADER_LEN: usize = 4096;

/// Longest block body accepted, far above what a block's gas limit allows.
const MAX_BODY_LEN: usize = 64 * 1024 * 1024;

/// Length of every key streamed: each table is keyed by a hash.
const KEY_LEN: u32 = 32;

/// Longest value accepted, far above the largest bytecode.
const MAX_VALUE_LEN: usize = 1024 * 1024;

/// Chunks the key space of a hashed table is split into, by the first two bytes of its keys, which
/// hashes spread evenly over.
const KEY_CHUNKS: u32 = 1 << 16;

/// Sets its flag when dropped, including while a panic unwinds, so threads waiting on the flag
/// stop rather than wait forever.
struct SetOnDrop<'a>(&'a AtomicBool);

impl Drop for SetOnDrop<'_> {
    fn drop(&mut self) {
        self.0.store(true, Ordering::Release);
    }
}

/// The chunk `key` falls in.
fn key_chunk(key: &[u8]) -> u32 {
    key.get(..2).map_or(0, |prefix| u32::from(u16::from_be_bytes([prefix[0], prefix[1]])))
}

/// zstd level for the state stream. Hashed keys are random and do not compress, so a fast level
/// gives up little.
const ZSTD_LEVEL: i32 = 1;

/// How long either side waits for the other to read or write before giving up on the transfer,
/// which keeps a stalled client from holding the server's read transaction open.
const STALL_TIMEOUT: Duration = Duration::from_secs(300);

/// The server's answer to a client's opening.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u8)]
enum Status {
    /// The client was accepted and the state follows.
    Ok = 0,
    /// The token was rejected.
    Unauthorized = 1,
    /// The client syncs a different chain.
    WrongChain = 2,
    /// The server runs as many transfers as it allows.
    Busy = 3,
    /// The server's state is not one block's state now, or could not be read; its log says why.
    Unavailable = 4,
}

impl TryFrom<u8> for Status {
    type Error = eyre::Report;

    fn try_from(byte: u8) -> eyre::Result<Self> {
        Ok(match byte {
            0 => Self::Ok,
            1 => Self::Unauthorized,
            2 => Self::WrongChain,
            3 => Self::Busy,
            4 => Self::Unavailable,
            other => return Err(eyre!("unknown server status {other}")),
        })
    }
}

fn write_u32(out: &mut impl Write, value: u32) -> std::io::Result<()> {
    out.write_all(&value.to_le_bytes())
}

fn write_u64(out: &mut impl Write, value: u64) -> std::io::Result<()> {
    out.write_all(&value.to_le_bytes())
}

/// Writes `bytes` prefixed with their `u32` length.
fn write_bytes(out: &mut impl Write, bytes: &[u8]) -> eyre::Result<()> {
    write_u32(out, u32::try_from(bytes.len())?)?;
    Ok(out.write_all(bytes)?)
}

fn read_array<const N: usize>(input: &mut impl Read) -> std::io::Result<[u8; N]> {
    let mut buf = [0; N];
    input.read_exact(&mut buf)?;
    Ok(buf)
}

fn read_u32(input: &mut impl Read) -> std::io::Result<u32> {
    read_array(input).map(u32::from_le_bytes)
}

fn read_u64(input: &mut impl Read) -> std::io::Result<u64> {
    read_array(input).map(u64::from_le_bytes)
}

/// Reads `len` bytes into a new vector.
fn read_vec(input: &mut impl Read, len: usize) -> std::io::Result<Vec<u8>> {
    let mut buf = vec![0; len];
    input.read_exact(&mut buf)?;
    Ok(buf)
}

/// Reads a byte string, refusing one longer than `max` before allocating it.
fn read_bytes(input: &mut impl Read, max: usize) -> eyre::Result<Vec<u8>> {
    let len = read_u32(input)? as usize;
    ensure!(len <= max, "peer sent {len} bytes where at most {max} are allowed");
    Ok(read_vec(input, len)?)
}

/// Reads the token shared by server and client from `path`, ignoring surrounding whitespace.
fn read_token(path: &Path) -> eyre::Result<Vec<u8>> {
    let token = reth_fs_util::read_to_string(path)?.trim().as_bytes().to_vec();
    ensure!(
        !token.is_empty() && token.len() <= MAX_TOKEN_LEN,
        "the token in {} must be 1 to {MAX_TOKEN_LEN} bytes long",
        path.display()
    );
    Ok(token)
}

fn read_magic(input: &mut impl Read) -> eyre::Result<()> {
    let magic = read_array::<8>(input)?;
    ensure!(&magic == MAGIC, "peer does not speak the state sync protocol");
    Ok(())
}

#[cfg(test)]
mod tests;
