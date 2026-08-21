//! Per-chain log stores: the sealed blocks the verifier asks about message existence.
//!
//! One store per chain. A sealed block is one record — the block's identity, one 32-byte hash per
//! log, and one 88-byte entry per executing message the block contains — so
//! [`LogsDb::contains`] is a single record read plus fixed-offset indexing into it, with no
//! per-log lookup.
//!
//! The record layout is the one op-supernode uses inside its raft-wal entries, kept byte for byte
//! so the two implementations can be compared record by record:
//!
//! ```text
//! [ 0.. 80) block record
//!     [ 0..32) hash
//!     [32..64) parent hash
//!     [64..72) timestamp        (BE u64)
//!     [72..76) log count        (BE u32)
//!     [76..80) exec msg count   (BE u32)
//! [ 80 .. 80 + 32*N )                  log hashes, N = log count
//! [ 80 + 32*N .. 80 + 32*N + 88*M )    executing messages, M = exec msg count, each:
//!     [ 0.. 4) log index within this block   (BE u32)
//!     [ 4..36) referenced chain id           (BE u256)
//!     [36..44) referenced block number       (BE u64)
//!     [44..48) referenced log index          (BE u32)
//!     [48..56) referenced timestamp          (BE u64)
//!     [56..88) message checksum
//! ```
//!
//! The referenced chain id keeps its full 32 bytes because it is read out of an on-chain log and
//! may name anything; the store's own chain is a configured [`ChainId`].

use crate::{
    checksum::{ChecksumArgs, MessageChecksum},
    encoding::{Cursor, Sink},
    error::StoreError,
    kv::{Kv, WriteBatch},
};
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, ChainId, U256};
use kona_protocol::BlockInfo;
use std::{
    collections::BTreeMap,
    fmt::Debug,
    sync::{PoisonError, RwLock},
};

/// The identity and timestamp of a sealed block.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct BlockSeal {
    /// The block's hash.
    pub hash: B256,
    /// The block's number.
    pub number: u64,
    /// The block's timestamp.
    pub timestamp: u64,
}

impl BlockSeal {
    /// Returns the block's number and hash.
    pub const fn id(&self) -> BlockNumHash {
        BlockNumHash { number: self.number, hash: self.hash }
    }
}

/// An executing message as the store holds it: the message it references, and the checksum that
/// answers whether that message exists.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct StoredExecutingMessage {
    /// The chain the referenced initiating message was emitted on.
    pub chain_id: U256,
    /// The referenced message's block number on that chain.
    pub block_number: u64,
    /// The referenced message's log index within that block.
    pub log_index: u32,
    /// The referenced message's block timestamp.
    pub timestamp: u64,
    /// The checksum committing to all of the above plus the referenced log's hash.
    pub checksum: MessageChecksum,
}

/// A sealed block opened for inspection: its reference, how many logs it holds, and the
/// executing messages among them by log index.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct OpenedBlock {
    /// The block's reference.
    pub block: BlockInfo,
    /// How many logs the block holds.
    pub log_count: u32,
    /// The executing messages the block holds, keyed by their log index within the block.
    pub executing_messages: BTreeMap<u32, StoredExecutingMessage>,
}

/// The question the verifier asks about an initiating message: does a log with this checksum sit
/// at this position, in a block with this timestamp?
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ContainsQuery {
    /// The block the initiating message claims to be in.
    pub block_number: u64,
    /// The log index the initiating message claims to be at.
    pub log_index: u32,
    /// The block timestamp the initiating message claims.
    pub timestamp: u64,
    /// The checksum the executing message carried.
    pub checksum: MessageChecksum,
}

/// One chain's log store.
///
/// Blocks are appended by seal: [`LogsDb::add_log`] buffers a block's logs in order and
/// [`LogsDb::seal_block`] writes them as one durable record, so a block is either fully present
/// or absent.
pub trait LogsDb: Debug + Send + Sync {
    /// Returns the latest sealed block, or [`None`] if the store is empty.
    fn latest_sealed_block(&self) -> Option<BlockNumHash>;

    /// Returns the earliest sealed block.
    fn first_sealed_block(&self) -> Result<BlockSeal, StoreError>;

    /// Returns the seal of the block at `number`.
    fn find_sealed_block(&self, number: u64) -> Result<BlockSeal, StoreError>;

    /// Returns the block at `number` with its log count and executing messages.
    fn open_block(&self, number: u64) -> Result<OpenedBlock, StoreError>;

    /// Returns the seal of the block holding the queried initiating message.
    ///
    /// [`StoreError::Future`] means the block is not indexed yet and the answer may change;
    /// [`StoreError::Conflict`] means the store holds that position and it does not match;
    /// [`StoreError::Skipped`] means the block predates this store's history.
    fn contains(&self, query: &ContainsQuery) -> Result<BlockSeal, StoreError>;

    /// Buffers one log of the block being built on top of `parent_block`.
    ///
    /// Logs must arrive in index order starting at zero.
    fn add_log(
        &self,
        log_hash: B256,
        parent_block: BlockNumHash,
        log_index: u32,
        executing_message: Option<StoredExecutingMessage>,
    ) -> Result<(), StoreError>;

    /// Seals the buffered logs as the block `block`, durably.
    fn seal_block(
        &self,
        parent_hash: B256,
        block: BlockNumHash,
        timestamp: u64,
    ) -> Result<(), StoreError>;

    /// Drops every block above `new_head`, which must be the block the store holds at that
    /// height.
    fn rewind(&self, new_head: BlockNumHash) -> Result<(), StoreError>;

    /// Drops every block.
    fn clear(&self) -> Result<(), StoreError>;
}

/// Fixed offsets of the sealed-block record. See the module documentation for the layout.
mod layout {
    /// Length of the block record header.
    pub(super) const HEADER: usize = 80;
    /// Length of one log hash.
    pub(super) const LOG_HASH: usize = 32;
    /// Length of one executing-message entry.
    pub(super) const EXEC_MSG: usize = 88;

    /// Returns the total record length for the given counts.
    pub(super) const fn record_len(log_count: u32, exec_msg_count: u32) -> usize {
        HEADER + log_count as usize * LOG_HASH + exec_msg_count as usize * EXEC_MSG
    }

    /// Returns the offset of the log hash at `index`.
    pub(super) const fn log_hash_offset(index: u32) -> usize {
        HEADER + index as usize * LOG_HASH
    }
}

/// The key column holding sealed blocks, keyed by big-endian block number.
mod column {
    pub(super) const SEALED: u8 = 0;

    pub(super) fn key(block_number: u64) -> [u8; 9] {
        let mut key = [0u8; 9];
        key[0] = SEALED;
        key[1..].copy_from_slice(&block_number.to_be_bytes());
        key
    }

    /// Returns the inclusive lower and exclusive upper bound of the column.
    pub(super) fn bounds() -> ([u8; 9], [u8; 1]) {
        (key(0), [SEALED + 1])
    }
}

/// The header of a sealed-block record, plus the byte ranges its arrays occupy.
#[derive(Debug, Clone, PartialEq, Eq)]
struct SealedBlockRecord<'a> {
    hash: B256,
    parent_hash: B256,
    timestamp: u64,
    log_count: u32,
    exec_msg_count: u32,
    log_hashes: &'a [u8],
    exec_msgs: &'a [u8],
}

impl<'a> SealedBlockRecord<'a> {
    /// Parses a record, checking its length against the counts it declares.
    fn decode(raw: &'a [u8]) -> Result<Self, StoreError> {
        let mut cursor = Cursor::new(raw, "sealed block");
        let hash = cursor.take_b256()?;
        let parent_hash = cursor.take_b256()?;
        let timestamp = cursor.take_u64()?;
        let log_count = cursor.take_u32()?;
        let exec_msg_count = cursor.take_u32()?;

        if raw.len() != layout::record_len(log_count, exec_msg_count) {
            return Err(StoreError::DataCorruption("sealed block: length disagrees with counts"));
        }
        let hashes_end = layout::log_hash_offset(log_count);
        Ok(Self {
            hash,
            parent_hash,
            timestamp,
            log_count,
            exec_msg_count,
            log_hashes: &raw[layout::HEADER..hashes_end],
            exec_msgs: &raw[hashes_end..],
        })
    }

    /// Returns the log hash at `index`, which the caller has checked is in range.
    fn log_hash(&self, index: u32) -> B256 {
        let offset = index as usize * layout::LOG_HASH;
        B256::from_slice(&self.log_hashes[offset..offset + layout::LOG_HASH])
    }

    /// Returns the `index`-th executing message with the log index it sits at.
    fn exec_msg(&self, index: u32) -> Result<(u32, StoredExecutingMessage), StoreError> {
        let offset = index as usize * layout::EXEC_MSG;
        let mut cursor =
            Cursor::new(&self.exec_msgs[offset..offset + layout::EXEC_MSG], "executing message");
        let log_index = cursor.take_u32()?;
        let message = StoredExecutingMessage {
            chain_id: cursor.take_u256()?,
            block_number: cursor.take_u64()?,
            log_index: cursor.take_u32()?,
            timestamp: cursor.take_u64()?,
            checksum: MessageChecksum(cursor.take_b256()?),
        };
        cursor.finish()?;
        Ok((log_index, message))
    }

    /// Returns the block reference the record describes.
    const fn block_info(&self, number: u64) -> BlockInfo {
        BlockInfo {
            hash: self.hash,
            number,
            parent_hash: self.parent_hash,
            timestamp: self.timestamp,
        }
    }
}

/// One buffered log awaiting its block's seal.
#[derive(Debug, Clone)]
struct PendingLog {
    hash: B256,
    log_index: u32,
    executing_message: Option<StoredExecutingMessage>,
}

/// The append cursor and buffered logs of a [`LogStore`].
#[derive(Debug, Default)]
struct Cursors {
    first_block: u64,
    latest: Option<BlockSeal>,
    pending_parent: Option<BlockNumHash>,
    pending_logs: Vec<PendingLog>,
}

/// One chain's [`LogsDb`] over a [`Kv`] backend.
#[derive(Debug)]
pub struct LogStore<K> {
    kv: K,
    chain_id: ChainId,
    cursors: RwLock<Cursors>,
}

impl<K: Kv> LogStore<K> {
    /// Opens the store for `chain_id` over `kv`, reading the append cursor from what is there.
    pub fn new(chain_id: ChainId, kv: K) -> Result<Self, StoreError> {
        let store = Self { kv, chain_id, cursors: RwLock::new(Cursors::default()) };
        let cursors = store.read_cursors()?;
        *store.cursors.write().unwrap_or_else(PoisonError::into_inner) = cursors;
        Ok(store)
    }

    /// Returns the chain this store holds blocks for.
    pub const fn chain_id(&self) -> ChainId {
        self.chain_id
    }

    /// Returns the underlying backend.
    pub const fn backend(&self) -> &K {
        &self.kv
    }

    fn read_cursors(&self) -> Result<Cursors, StoreError> {
        let (start, end) = column::bounds();
        let Some((first_key, _)) = self.kv.first_in(&start, &end)? else {
            return Ok(Cursors::default());
        };
        let (last_key, last_raw) =
            self.kv.last_in(&start, &end)?.ok_or(StoreError::DataCorruption("sealed block"))?;
        let number = decode_block_key(&last_key)?;
        let record = SealedBlockRecord::decode(&last_raw)?;
        Ok(Cursors {
            first_block: decode_block_key(&first_key)?,
            latest: Some(BlockSeal { hash: record.hash, number, timestamp: record.timestamp }),
            pending_parent: None,
            pending_logs: Vec::new(),
        })
    }

    /// Reads the record at `number`, mapping absence to the reason it is absent.
    fn record_bytes(&self, cursors: &Cursors, number: u64) -> Result<Vec<u8>, StoreError> {
        let latest = cursors.latest.ok_or(StoreError::Future)?;
        if number > latest.number {
            return Err(StoreError::Future);
        }
        if number < cursors.first_block {
            return Err(StoreError::Skipped);
        }
        self.kv
            .get(&column::key(number))?
            .ok_or(StoreError::DataCorruption("sealed block: gap below the frontier"))
    }
}

impl<K: Kv> LogsDb for LogStore<K> {
    fn latest_sealed_block(&self) -> Option<BlockNumHash> {
        let cursors = self.cursors.read().unwrap_or_else(PoisonError::into_inner);
        cursors.latest.map(|seal| seal.id())
    }

    fn first_sealed_block(&self) -> Result<BlockSeal, StoreError> {
        let cursors = self.cursors.read().unwrap_or_else(PoisonError::into_inner);
        let first = cursors.first_block;
        let raw = self.record_bytes(&cursors, first)?;
        let record = SealedBlockRecord::decode(&raw)?;
        Ok(BlockSeal { hash: record.hash, number: first, timestamp: record.timestamp })
    }

    fn find_sealed_block(&self, number: u64) -> Result<BlockSeal, StoreError> {
        let cursors = self.cursors.read().unwrap_or_else(PoisonError::into_inner);
        let raw = self.record_bytes(&cursors, number)?;
        let record = SealedBlockRecord::decode(&raw)?;
        Ok(BlockSeal { hash: record.hash, number, timestamp: record.timestamp })
    }

    fn open_block(&self, number: u64) -> Result<OpenedBlock, StoreError> {
        let cursors = self.cursors.read().unwrap_or_else(PoisonError::into_inner);
        let raw = self.record_bytes(&cursors, number)?;
        let record = SealedBlockRecord::decode(&raw)?;
        let mut executing_messages = BTreeMap::new();
        for index in 0..record.exec_msg_count {
            let (log_index, message) = record.exec_msg(index)?;
            if executing_messages.insert(log_index, message).is_some() {
                return Err(StoreError::DataCorruption(
                    "sealed block: two executing messages at one log index",
                ));
            }
        }
        Ok(OpenedBlock {
            block: record.block_info(number),
            log_count: record.log_count,
            executing_messages,
        })
    }

    fn contains(&self, query: &ContainsQuery) -> Result<BlockSeal, StoreError> {
        let cursors = self.cursors.read().unwrap_or_else(PoisonError::into_inner);
        let latest = cursors.latest.ok_or(StoreError::Future)?;
        if query.block_number > latest.number {
            // A block we have not sealed yet may still arrive — unless we have already sealed
            // past the timestamp the message claims, in which case it never will.
            return Err(if latest.timestamp > query.timestamp {
                StoreError::Conflict("initiating message claims a block we have sealed past")
            } else {
                StoreError::Future
            });
        }
        let raw = self.record_bytes(&cursors, query.block_number)?;
        let record = SealedBlockRecord::decode(&raw)?;

        if query.log_index >= record.log_count {
            return Err(StoreError::Conflict("initiating message log index out of range"));
        }
        if record.timestamp != query.timestamp {
            return Err(StoreError::Conflict("initiating message block timestamp mismatch"));
        }
        let expected = ChecksumArgs {
            block_number: query.block_number,
            log_index: query.log_index,
            timestamp: record.timestamp,
            chain_id: U256::from(self.chain_id),
            log_hash: record.log_hash(query.log_index),
        }
        .checksum();
        if expected != query.checksum {
            return Err(StoreError::Conflict("initiating message checksum mismatch"));
        }
        Ok(BlockSeal { hash: record.hash, number: query.block_number, timestamp: record.timestamp })
    }

    fn add_log(
        &self,
        log_hash: B256,
        parent_block: BlockNumHash,
        log_index: u32,
        executing_message: Option<StoredExecutingMessage>,
    ) -> Result<(), StoreError> {
        let mut cursors = self.cursors.write().unwrap_or_else(PoisonError::into_inner);

        // The EVM never executes the genesis block, so a log whose parent is the zero block
        // cannot come from a legitimate writer.
        if parent_block == BlockNumHash::default() {
            return Err(StoreError::OutOfOrder("genesis has no logs"));
        }
        if let Some(latest) = cursors.latest &&
            parent_block != latest.id()
        {
            return Err(StoreError::OutOfOrder("log's parent is not the latest sealed block"));
        }
        match cursors.pending_parent {
            Some(pending_parent) => {
                if parent_block != pending_parent {
                    return Err(StoreError::OutOfOrder(
                        "log's parent is not the block being built",
                    ));
                }
                if log_index as usize != cursors.pending_logs.len() {
                    return Err(StoreError::OutOfOrder("log index is not the next one"));
                }
            }
            None => {
                if log_index != 0 {
                    return Err(StoreError::OutOfOrder("a block's first log must be index 0"));
                }
                cursors.pending_parent = Some(parent_block);
            }
        }
        cursors.pending_logs.push(PendingLog { hash: log_hash, log_index, executing_message });
        Ok(())
    }

    fn seal_block(
        &self,
        parent_hash: B256,
        block: BlockNumHash,
        timestamp: u64,
    ) -> Result<(), StoreError> {
        let mut cursors = self.cursors.write().unwrap_or_else(PoisonError::into_inner);

        if let Some(latest) = cursors.latest {
            if block.number != latest.number + 1 {
                return Err(StoreError::Conflict("sealed block is not the next height"));
            }
            if parent_hash != latest.hash {
                return Err(StoreError::Conflict("sealed block's parent is not the latest block"));
            }
            if timestamp < latest.timestamp {
                return Err(StoreError::Conflict("sealed block's timestamp goes backwards"));
            }
        }
        if let Some(pending_parent) = cursors.pending_parent {
            let parent_number = block
                .number
                .checked_sub(1)
                .ok_or(StoreError::Conflict("genesis cannot carry buffered logs"))?;
            let expected = BlockNumHash { number: parent_number, hash: parent_hash };
            if pending_parent != expected {
                return Err(StoreError::Conflict(
                    "sealed block's parent is not the buffered logs' parent",
                ));
            }
        }

        let log_count = u32::try_from(cursors.pending_logs.len())
            .map_err(|_| StoreError::Conflict("too many logs in one block"))?;
        let exec_msg_count = cursors
            .pending_logs
            .iter()
            .filter(|log| log.executing_message.is_some())
            .count()
            .try_into()
            .map_err(|_| StoreError::Conflict("too many executing messages in one block"))?;

        let mut sink = Sink::with_capacity(layout::record_len(log_count, exec_msg_count));
        sink.put_b256(block.hash);
        sink.put_b256(parent_hash);
        sink.put_u64(timestamp);
        sink.put_u32(log_count);
        sink.put_u32(exec_msg_count);
        for log in &cursors.pending_logs {
            sink.put_b256(log.hash);
        }
        for log in &cursors.pending_logs {
            let Some(message) = log.executing_message else { continue };
            sink.put_u32(log.log_index);
            sink.put_u256(message.chain_id);
            sink.put_u64(message.block_number);
            sink.put_u32(message.log_index);
            sink.put_u64(message.timestamp);
            sink.put_b256(message.checksum.as_b256());
        }

        let mut batch = WriteBatch::new();
        batch.put(column::key(block.number), sink.into_vec());
        self.kv.write(batch)?;

        if cursors.latest.is_none() {
            cursors.first_block = block.number;
        }
        cursors.latest = Some(BlockSeal { hash: block.hash, number: block.number, timestamp });
        cursors.pending_logs.clear();
        cursors.pending_parent = None;
        Ok(())
    }

    fn rewind(&self, new_head: BlockNumHash) -> Result<(), StoreError> {
        let mut cursors = self.cursors.write().unwrap_or_else(PoisonError::into_inner);
        let Some(latest) = cursors.latest else {
            return clear_locked(&self.kv, &mut cursors);
        };
        if new_head.number < cursors.first_block {
            return clear_locked(&self.kv, &mut cursors);
        }
        if new_head.number > latest.number {
            return Err(StoreError::Future);
        }

        let raw = self.record_bytes(&cursors, new_head.number)?;
        let record = SealedBlockRecord::decode(&raw)?;
        if record.hash != new_head.hash {
            return Err(StoreError::Conflict("rewind target is not the block we hold"));
        }

        if new_head.number < latest.number {
            let (_, end) = column::bounds();
            let mut batch = WriteBatch::new();
            batch.delete_range(column::key(new_head.number + 1), end);
            self.kv.write(batch)?;
        }

        cursors.latest = Some(BlockSeal {
            hash: new_head.hash,
            number: new_head.number,
            timestamp: record.timestamp,
        });
        cursors.pending_logs.clear();
        cursors.pending_parent = None;
        Ok(())
    }

    fn clear(&self) -> Result<(), StoreError> {
        let mut cursors = self.cursors.write().unwrap_or_else(PoisonError::into_inner);
        clear_locked(&self.kv, &mut cursors)
    }
}

fn clear_locked<K: Kv>(kv: &K, cursors: &mut Cursors) -> Result<(), StoreError> {
    let (start, end) = column::bounds();
    let mut batch = WriteBatch::new();
    batch.delete_range(start, end);
    kv.write(batch)?;
    *cursors = Cursors::default();
    Ok(())
}

/// Decodes a sealed-block key back into its block number.
fn decode_block_key(key: &[u8]) -> Result<u64, StoreError> {
    if key.len() != 9 || key[0] != column::SEALED {
        return Err(StoreError::DataCorruption("sealed block key layout"));
    }
    let bytes: [u8; 8] =
        key[1..].try_into().map_err(|_| StoreError::DataCorruption("sealed block key layout"))?;
    Ok(u64::from_be_bytes(bytes))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{checksum::log_hash, kv::MemoryKv};
    use alloy_primitives::Address;

    const CHAIN: ChainId = 901;

    fn block(number: u64) -> BlockNumHash {
        BlockNumHash { number, hash: B256::repeat_byte(number as u8) }
    }

    fn store() -> LogStore<MemoryKv> {
        LogStore::new(CHAIN, MemoryKv::new()).unwrap()
    }

    fn a_log_hash(seed: u8) -> B256 {
        log_hash(Address::repeat_byte(seed), B256::repeat_byte(seed))
    }

    fn exec_msg(seed: u8) -> StoredExecutingMessage {
        StoredExecutingMessage {
            chain_id: U256::from(902),
            block_number: seed as u64,
            log_index: 0,
            timestamp: 1000,
            checksum: MessageChecksum(B256::repeat_byte(seed)),
        }
    }

    /// Seals `log_count` logs into the block at `number`, the last of which optionally carries an
    /// executing message.
    fn seal(
        store: &LogStore<MemoryKv>,
        number: u64,
        timestamp: u64,
        log_count: u32,
        with_exec_msg: bool,
    ) {
        for index in 0..log_count {
            let executing_message =
                (with_exec_msg && index + 1 == log_count).then(|| exec_msg(index as u8));
            store
                .add_log(a_log_hash(index as u8), block(number - 1), index, executing_message)
                .unwrap();
        }
        store.seal_block(block(number - 1).hash, block(number), timestamp).unwrap();
    }

    #[test]
    fn an_empty_store_answers_future() {
        let store = store();
        assert_eq!(store.latest_sealed_block(), None);
        assert!(matches!(store.first_sealed_block(), Err(StoreError::Future)));
        assert!(matches!(store.find_sealed_block(1), Err(StoreError::Future)));
        assert!(matches!(store.open_block(1), Err(StoreError::Future)));
        assert_eq!(store.chain_id(), CHAIN);
    }

    #[test]
    fn a_sealed_block_reads_back() {
        let store = store();
        seal(&store, 10, 1000, 3, true);

        assert_eq!(store.latest_sealed_block(), Some(block(10)));
        assert_eq!(store.find_sealed_block(10).unwrap().timestamp, 1000);
        assert_eq!(store.first_sealed_block().unwrap().id(), block(10));

        let opened = store.open_block(10).unwrap();
        assert_eq!(opened.log_count, 3);
        assert_eq!(opened.block.number, 10);
        assert_eq!(opened.block.parent_hash, block(9).hash);
        assert_eq!(opened.block.timestamp, 1000);
        assert_eq!(opened.executing_messages.len(), 1);
        assert_eq!(opened.executing_messages[&2], exec_msg(2));
    }

    #[test]
    fn a_block_with_no_logs_seals() {
        let store = store();
        store.seal_block(block(9).hash, block(10), 1000).unwrap();
        let opened = store.open_block(10).unwrap();
        assert_eq!(opened.log_count, 0);
        assert!(opened.executing_messages.is_empty());
    }

    #[test]
    fn contains_matches_the_checksum_of_the_stored_log() {
        let store = store();
        seal(&store, 10, 1000, 3, false);

        let query = ContainsQuery {
            block_number: 10,
            log_index: 1,
            timestamp: 1000,
            checksum: ChecksumArgs {
                block_number: 10,
                log_index: 1,
                timestamp: 1000,
                chain_id: U256::from(CHAIN),
                log_hash: a_log_hash(1),
            }
            .checksum(),
        };
        assert_eq!(store.contains(&query).unwrap().id(), block(10));

        // A different log's checksum at the same position conflicts.
        let mismatched = ContainsQuery {
            checksum: ChecksumArgs {
                block_number: 10,
                log_index: 1,
                timestamp: 1000,
                chain_id: U256::from(CHAIN),
                log_hash: a_log_hash(2),
            }
            .checksum(),
            ..query
        };
        assert!(matches!(store.contains(&mismatched), Err(StoreError::Conflict(_))));

        // So does a position past the block's logs, or a wrong timestamp.
        assert!(matches!(
            store.contains(&ContainsQuery { log_index: 3, ..query }),
            Err(StoreError::Conflict(_))
        ));
        assert!(matches!(
            store.contains(&ContainsQuery { timestamp: 999, ..query }),
            Err(StoreError::Conflict(_))
        ));
    }

    #[test]
    fn contains_separates_not_yet_from_never() {
        let store = store();
        seal(&store, 10, 1000, 1, false);
        let query = ContainsQuery {
            block_number: 11,
            log_index: 0,
            timestamp: 1002,
            checksum: MessageChecksum::default(),
        };
        // Claimed timestamp is not behind ours: the block may still arrive.
        assert!(matches!(store.contains(&query), Err(StoreError::Future)));
        // Claimed timestamp is behind ours: a block at that height with that timestamp cannot
        // exist any more.
        assert!(matches!(
            store.contains(&ContainsQuery { timestamp: 999, ..query }),
            Err(StoreError::Conflict(_))
        ));
    }

    #[test]
    fn contains_below_history_is_skipped() {
        let store = store();
        seal(&store, 10, 1000, 1, false);
        assert!(matches!(
            store.contains(&ContainsQuery {
                block_number: 9,
                log_index: 0,
                timestamp: 998,
                checksum: MessageChecksum::default(),
            }),
            Err(StoreError::Skipped)
        ));
        assert!(matches!(store.find_sealed_block(9), Err(StoreError::Skipped)));
    }

    #[test]
    fn logs_must_arrive_in_order_on_the_right_parent() {
        let store = store();
        seal(&store, 10, 1000, 1, false);

        assert!(matches!(
            store.add_log(a_log_hash(0), BlockNumHash::default(), 0, None),
            Err(StoreError::OutOfOrder(_))
        ));
        assert!(matches!(
            store.add_log(a_log_hash(0), block(8), 0, None),
            Err(StoreError::OutOfOrder(_))
        ));
        assert!(matches!(
            store.add_log(a_log_hash(0), block(10), 1, None),
            Err(StoreError::OutOfOrder(_))
        ));
        store.add_log(a_log_hash(0), block(10), 0, None).unwrap();
        assert!(matches!(
            store.add_log(a_log_hash(1), block(10), 2, None),
            Err(StoreError::OutOfOrder(_))
        ));
    }

    #[test]
    fn seals_must_extend_the_frontier() {
        let store = store();
        seal(&store, 10, 1000, 1, false);

        assert!(matches!(
            store.seal_block(block(10).hash, block(12), 1002),
            Err(StoreError::Conflict(_))
        ));
        assert!(matches!(
            store.seal_block(B256::repeat_byte(0xff), block(11), 1002),
            Err(StoreError::Conflict(_))
        ));
        assert!(matches!(
            store.seal_block(block(10).hash, block(11), 999),
            Err(StoreError::Conflict(_))
        ));
    }

    #[test]
    fn genesis_cannot_carry_buffered_logs() {
        let store = store();
        // `add_log` already refuses the zero parent, so reaching a seal at height 0 with
        // buffered logs takes a bogus parent height; it must be rejected, not underflow.
        store
            .add_log(a_log_hash(0), BlockNumHash { number: u64::MAX, hash: B256::ZERO }, 0, None)
            .unwrap();
        assert!(matches!(
            store.seal_block(B256::ZERO, block(0), 1000),
            Err(StoreError::Conflict(_))
        ));
    }

    #[test]
    fn a_seal_must_match_the_buffered_logs_parent() {
        let store = store();
        store.add_log(a_log_hash(0), block(9), 0, None).unwrap();
        assert!(matches!(
            store.seal_block(B256::repeat_byte(0xfe), block(10), 1000),
            Err(StoreError::Conflict(_))
        ));
    }

    #[test]
    fn rewind_drops_the_blocks_above_the_target() {
        let store = store();
        for number in 10..15 {
            seal(&store, number, 1000 + number, 1, false);
        }
        store.rewind(block(12)).unwrap();
        assert_eq!(store.latest_sealed_block(), Some(block(12)));
        assert!(matches!(store.find_sealed_block(13), Err(StoreError::Future)));
        // The store is appendable again from the new head.
        seal(&store, 13, 1013, 1, false);
        assert_eq!(store.latest_sealed_block(), Some(block(13)));
    }

    #[test]
    fn rewind_checks_the_target_is_the_block_we_hold() {
        let store = store();
        seal(&store, 10, 1000, 1, false);
        assert!(matches!(
            store.rewind(BlockNumHash { number: 10, hash: B256::repeat_byte(0xff) }),
            Err(StoreError::Conflict(_))
        ));
        assert!(matches!(store.rewind(block(11)), Err(StoreError::Future)));
    }

    #[test]
    fn rewinding_below_history_clears_the_store() {
        let store = store();
        seal(&store, 10, 1000, 1, false);
        store.rewind(block(5)).unwrap();
        assert_eq!(store.latest_sealed_block(), None);
    }

    #[test]
    fn rewind_discards_buffered_logs() {
        let store = store();
        seal(&store, 10, 1000, 1, false);
        store.add_log(a_log_hash(0), block(10), 0, None).unwrap();
        store.rewind(block(10)).unwrap();
        // The buffer is gone, so index 0 is accepted again rather than rejected as a repeat.
        store.add_log(a_log_hash(0), block(10), 0, None).unwrap();
    }

    #[test]
    fn clear_empties_the_store() {
        let store = store();
        seal(&store, 10, 1000, 1, false);
        store.clear().unwrap();
        assert_eq!(store.latest_sealed_block(), None);
        // A cleared store may restart at any height.
        seal(&store, 50, 5000, 1, false);
        assert_eq!(store.first_sealed_block().unwrap().number, 50);
    }

    #[test]
    fn the_cursor_survives_a_reopen() {
        let kv = MemoryKv::new();
        let store = LogStore::new(CHAIN, kv).unwrap();
        seal(&store, 10, 1000, 2, true);
        seal(&store, 11, 1002, 1, false);

        let reopened = LogStore::new(CHAIN, store.kv).unwrap();
        assert_eq!(reopened.latest_sealed_block(), Some(block(11)));
        assert_eq!(reopened.first_sealed_block().unwrap().number, 10);
        assert_eq!(reopened.open_block(10).unwrap().executing_messages.len(), 1);
    }

    #[test]
    fn the_record_layout_is_the_op_supernode_one() {
        let store = store();
        seal(&store, 10, 1000, 2, true);
        let raw = store.kv.get(&column::key(10)).unwrap().unwrap();
        // 80-byte header, 32 bytes per log hash, 88 bytes per executing message.
        assert_eq!(raw.len(), 80 + 2 * 32 + 88);
        assert_eq!(&raw[0..32], block(10).hash.as_slice());
        assert_eq!(&raw[32..64], block(9).hash.as_slice());
        assert_eq!(u64::from_be_bytes(raw[64..72].try_into().unwrap()), 1000);
        assert_eq!(u32::from_be_bytes(raw[72..76].try_into().unwrap()), 2);
        assert_eq!(u32::from_be_bytes(raw[76..80].try_into().unwrap()), 1);
        assert_eq!(&raw[80..112], a_log_hash(0).as_slice());
        assert_eq!(&raw[112..144], a_log_hash(1).as_slice());
        assert_eq!(u32::from_be_bytes(raw[144..148].try_into().unwrap()), 1);
    }

    #[test]
    fn a_record_whose_length_disagrees_with_its_counts_is_corruption() {
        let store = store();
        seal(&store, 10, 1000, 2, false);
        let mut raw = store.kv.get(&column::key(10)).unwrap().unwrap();
        raw[75] = 3; // claim three log hashes where two are stored
        let mut batch = WriteBatch::new();
        batch.put(column::key(10), raw);
        store.kv.write(batch).unwrap();

        assert!(matches!(store.open_block(10), Err(StoreError::DataCorruption(_))));
    }
}
