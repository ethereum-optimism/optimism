//! The verifier's durable store: verified results by timestamp and denied blocks, in `RocksDB`.
//!
//! Every write is one atomic batch, synced to the write-ahead log before it returns. After a
//! crash the store holds a prefix of the writes made: a torn last batch is dropped whole.

use crate::state::{ChainResult, DeniedRecord, VerifiedResult};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use kona_protocol::OutputRoot;
use rocksdb::{DB, DBRecoveryMode, Direction, IteratorMode, Options, WriteBatch, WriteOptions};
use std::{collections::BTreeMap, path::Path};
use thiserror::Error;

/// The key prefix of verified results: `r ‖ timestamp`.
const RESULT: u8 = b'r';
/// The key prefix of denied blocks: `d ‖ chain ID ‖ block number ‖ block hash`.
const DENIED: u8 = b'd';

const RESULT_KEY_LEN: usize = 1 + 8;
const DENIED_KEY_LEN: usize = 1 + 8 + 8 + 32;
/// A block number and hash.
const BLOCK_LEN: usize = 8 + 32;
/// The state root and message passer storage root of an output; the block hash is stored apart.
const OUTPUT_ROOTS_LEN: usize = 32 + 32;
/// A denied value: the output roots, the deciding timestamp and the L1 block it was decided on.
const DENIED_VALUE_LEN: usize = OUTPUT_ROOTS_LEN + 8 + BLOCK_LEN;

/// A failed store operation.
#[derive(Debug, Error)]
pub(crate) enum StoreError {
    /// `RocksDB` failed.
    #[error("rocksdb: {0}")]
    Db(#[from] rocksdb::Error),
    /// The directory of the store could not be created.
    #[error("create the store directory: {0}")]
    Dir(#[from] std::io::Error),
    /// A record does not decode.
    #[error("corrupt {kind} record under key {key}")]
    Corrupt {
        /// The kind of record.
        kind: &'static str,
        /// The record's key, hex.
        key: String,
    },
    /// A result the store must hold is missing: results are contiguous from the first one.
    #[error("no verified result at {0}, below the latest one")]
    Missing(u64),
    /// The store was written for other chains or another activation.
    #[error(
        "the store holds results from {first} for chains {chains:?}, not from {activation} for {expected:?}"
    )]
    OtherConfig {
        /// The first stored timestamp.
        first: u64,
        /// The chains of the latest stored result.
        chains: Vec<u64>,
        /// The configured activation timestamp.
        activation: u64,
        /// The configured chains.
        expected: Vec<u64>,
    },
}

/// The store, at one directory.
#[derive(Debug)]
pub(crate) struct Store {
    db: DB,
}

impl Store {
    /// Opens the store at `path`, creating it if it does not exist.
    pub(crate) fn open(path: &Path) -> Result<Self, StoreError> {
        std::fs::create_dir_all(path)?;
        let mut options = Options::default();
        options.create_if_missing(true);
        // Recover the writes before the first torn or corrupt record, and none after it: the
        // store then holds a prefix of what was written, so every result it holds has every
        // earlier result too.
        options.set_wal_recovery_mode(DBRecoveryMode::PointInTime);
        Ok(Self { db: DB::open(&options, path)? })
    }

    /// Writes the result of `timestamp`.
    pub(crate) fn put_result(
        &self,
        timestamp: u64,
        result: &VerifiedResult,
    ) -> Result<(), StoreError> {
        let mut batch = WriteBatch::default();
        batch.put(result_key(timestamp), encode_result(result));
        self.write(batch)
    }

    /// The result of `timestamp`, if verified.
    pub(crate) fn result(&self, timestamp: u64) -> Result<Option<VerifiedResult>, StoreError> {
        let key = result_key(timestamp);
        self.db.get_pinned(key)?.map(|value| decode_result(&key, &value)).transpose()
    }

    /// The earliest result.
    pub(crate) fn first_result(&self) -> Result<Option<(u64, VerifiedResult)>, StoreError> {
        self.result_from(IteratorMode::From(&result_key(0), Direction::Forward))
    }

    /// The latest result.
    pub(crate) fn last_result(&self) -> Result<Option<(u64, VerifiedResult)>, StoreError> {
        self.result_from(IteratorMode::From(&result_key(u64::MAX), Direction::Reverse))
    }

    fn result_from(
        &self,
        mode: IteratorMode<'_>,
    ) -> Result<Option<(u64, VerifiedResult)>, StoreError> {
        let Some(entry) = self.db.iterator(mode).next() else { return Ok(None) };
        let (key, value) = entry?;
        if key.first() != Some(&RESULT) {
            return Ok(None);
        }
        let timestamp = decode_result_key(&key)?;
        Ok(Some((timestamp, decode_result(&key, &value)?)))
    }

    /// Writes a denied block. Writing the same block again only replaces its record.
    pub(crate) fn put_denied(&self, record: &DeniedRecord) -> Result<(), StoreError> {
        let mut value = Vec::with_capacity(DENIED_VALUE_LEN);
        push_output_roots(&mut value, &record.output);
        value.extend_from_slice(&record.timestamp.to_be_bytes());
        push_block(&mut value, record.l1);
        let mut batch = WriteBatch::default();
        batch.put(denied_key(record.chain_id, record.block.number, record.block.hash), value);
        self.write(batch)
    }

    /// Deletes the results from `timestamp` on and the denied blocks decided at or after it, in
    /// one write. Returns how many results and denied blocks it deleted.
    pub(crate) fn rewind(&self, timestamp: u64) -> Result<(usize, usize), StoreError> {
        let mut batch = WriteBatch::default();
        let mut results = 0;
        let start = result_key(timestamp);
        for entry in self.db.iterator(IteratorMode::From(&start, Direction::Forward)) {
            let (key, _) = entry?;
            if key.first() != Some(&RESULT) {
                break;
            }
            batch.delete(key);
            results += 1;
        }
        let mut denied = 0;
        for entry in self.db.iterator(IteratorMode::From(&[DENIED], Direction::Forward)) {
            let (key, value) = entry?;
            if key.first() != Some(&DENIED) {
                break;
            }
            if decode_denied(&key, &value)?.timestamp >= timestamp {
                batch.delete(key);
                denied += 1;
            }
        }
        self.write(batch)?;
        Ok((results, denied))
    }

    /// Deletes the denial of `record`'s block.
    pub(crate) fn delete_denied(&self, record: &DeniedRecord) -> Result<(), StoreError> {
        let mut batch = WriteBatch::default();
        batch.delete(denied_key(record.chain_id, record.block.number, record.block.hash));
        self.write(batch)
    }

    /// The denied blocks of `chain_id` at or above `from_number`, by number.
    pub(crate) fn denied(
        &self,
        chain_id: u64,
        from_number: u64,
    ) -> Result<Vec<DeniedRecord>, StoreError> {
        let start = denied_key(chain_id, from_number, B256::ZERO);
        let mut records = Vec::new();
        for entry in self.db.iterator(IteratorMode::From(&start, Direction::Forward)) {
            let (key, value) = entry?;
            if key.get(..9) != Some(&start[..9]) {
                break;
            }
            records.push(decode_denied(&key, &value)?);
        }
        Ok(records)
    }

    fn write(&self, batch: WriteBatch) -> Result<(), StoreError> {
        let mut options = WriteOptions::default();
        options.set_sync(true);
        Ok(self.db.write_opt(batch, &options)?)
    }
}

fn result_key(timestamp: u64) -> [u8; RESULT_KEY_LEN] {
    let mut key = [RESULT; RESULT_KEY_LEN];
    key[1..].copy_from_slice(&timestamp.to_be_bytes());
    key
}

fn decode_result_key(key: &[u8]) -> Result<u64, StoreError> {
    key.get(1..)
        .and_then(|timestamp| timestamp.try_into().ok())
        .map(u64::from_be_bytes)
        .ok_or_else(|| corrupt("result", key))
}

fn denied_key(chain_id: u64, number: u64, hash: B256) -> [u8; DENIED_KEY_LEN] {
    let mut key = [DENIED; DENIED_KEY_LEN];
    key[1..9].copy_from_slice(&chain_id.to_be_bytes());
    key[9..17].copy_from_slice(&number.to_be_bytes());
    key[17..].copy_from_slice(hash.as_slice());
    key
}

/// A result value: the L1 inclusion, then per chain its ID, block, output roots, required L1
/// block, and a flag followed, if set, by the block hash and output roots of the block it replaced.
fn encode_result(result: &VerifiedResult) -> Vec<u8> {
    let mut value = Vec::new();
    push_block(&mut value, result.l1_inclusion);
    for (chain_id, chain) in &result.chains {
        value.extend_from_slice(&chain_id.to_be_bytes());
        push_block(&mut value, chain.block);
        push_output_roots(&mut value, &chain.output);
        push_block(&mut value, chain.required_l1);
        match &chain.replaced {
            None => value.push(0),
            Some(replaced) => {
                value.push(1);
                value.extend_from_slice(replaced.block_hash.as_slice());
                push_output_roots(&mut value, replaced);
            }
        }
    }
    value
}

fn decode_result(key: &[u8], value: &[u8]) -> Result<VerifiedResult, StoreError> {
    let corrupt = || corrupt("result", key);
    let mut reader = Reader(value);
    let l1_inclusion = reader.block().ok_or_else(corrupt)?;
    let mut chains = BTreeMap::new();
    while !reader.0.is_empty() {
        let chain_id = reader.u64().ok_or_else(corrupt)?;
        let block = reader.block().ok_or_else(corrupt)?;
        let output = reader.output(block.hash).ok_or_else(corrupt)?;
        let required_l1 = reader.block().ok_or_else(corrupt)?;
        let replaced = match reader.take(1).ok_or_else(corrupt)? {
            [0] => None,
            [1] => {
                let hash = reader.hash().ok_or_else(corrupt)?;
                Some(reader.output(hash).ok_or_else(corrupt)?)
            }
            _ => return Err(corrupt()),
        };
        chains.insert(chain_id, ChainResult { block, output, required_l1, replaced });
    }
    Ok(VerifiedResult { l1_inclusion, chains })
}

fn decode_denied(key: &[u8], value: &[u8]) -> Result<DeniedRecord, StoreError> {
    if key.len() != DENIED_KEY_LEN || value.len() != DENIED_VALUE_LEN {
        return Err(corrupt("denied", key));
    }
    let block = BlockNumHash { number: be_u64(&key[9..17]), hash: B256::from_slice(&key[17..]) };
    let mut reader = Reader(value);
    let decoded = (|| {
        Some(DeniedRecord {
            chain_id: be_u64(&key[1..9]),
            block,
            output: reader.output(block.hash)?,
            timestamp: reader.u64()?,
            l1: reader.block()?,
        })
    })();
    decoded.ok_or_else(|| corrupt("denied", key))
}

/// Reads a value front to back.
struct Reader<'a>(&'a [u8]);

impl<'a> Reader<'a> {
    fn take(&mut self, len: usize) -> Option<&'a [u8]> {
        let (head, rest) = self.0.split_at_checked(len)?;
        self.0 = rest;
        Some(head)
    }

    fn u64(&mut self) -> Option<u64> {
        self.take(8).map(be_u64)
    }

    fn hash(&mut self) -> Option<B256> {
        self.take(32).map(B256::from_slice)
    }

    fn block(&mut self) -> Option<BlockNumHash> {
        Some(BlockNumHash { number: self.u64()?, hash: self.hash()? })
    }

    /// The output of the block with hash `block_hash`.
    fn output(&mut self, block_hash: B256) -> Option<OutputRoot> {
        Some(OutputRoot::from_parts(self.hash()?, self.hash()?, block_hash))
    }
}

fn push_output_roots(value: &mut Vec<u8>, output: &OutputRoot) {
    value.extend_from_slice(output.state_root.as_slice());
    value.extend_from_slice(output.bridge_storage_root.as_slice());
}

fn push_block(value: &mut Vec<u8>, block: BlockNumHash) {
    value.extend_from_slice(&block.number.to_be_bytes());
    value.extend_from_slice(block.hash.as_slice());
}

/// Reads a big-endian `u64` from exactly 8 bytes.
fn be_u64(bytes: &[u8]) -> u64 {
    u64::from_be_bytes(bytes.try_into().expect("8 bytes"))
}

fn corrupt(kind: &'static str, key: &[u8]) -> StoreError {
    StoreError::Corrupt { kind, key: alloy_primitives::hex::encode(key) }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::{fs, path::PathBuf};

    fn block(number: u64) -> BlockNumHash {
        BlockNumHash { number, hash: B256::with_last_byte(number as u8) }
    }

    fn output(block: BlockNumHash, tag: u8) -> OutputRoot {
        OutputRoot::from_parts(B256::repeat_byte(tag), B256::repeat_byte(tag + 1), block.hash)
    }

    /// A result of two chains, the second of whose blocks replaced a denied one.
    fn result(n: u64) -> VerifiedResult {
        let chain = |number: u64, replaced| ChainResult {
            block: block(number),
            output: output(block(number), 0xa0),
            required_l1: block(50 + n),
            replaced,
        };
        let denied = BlockNumHash { number: 2 * n, hash: B256::repeat_byte(0xdd) };
        VerifiedResult {
            l1_inclusion: block(100 + n),
            chains: BTreeMap::from([
                (901, chain(n, None)),
                (902, chain(2 * n, Some(output(denied, 0xb0)))),
            ]),
        }
    }

    fn denied(chain_id: u64, number: u64) -> DeniedRecord {
        DeniedRecord {
            chain_id,
            block: block(number),
            output: output(block(number), 0xe0),
            timestamp: 1000 + number,
            l1: block(200 + number),
        }
    }

    #[test]
    fn reopened_store_holds_what_was_written() {
        let dir = tempfile::tempdir().unwrap();
        {
            let store = Store::open(dir.path()).unwrap();
            for ts in 10..15 {
                store.put_result(ts, &result(ts)).unwrap();
            }
            for record in [denied(901, 7), denied(901, 3), denied(902, 5)] {
                store.put_denied(&record).unwrap();
            }
        }

        let store = Store::open(dir.path()).unwrap();

        assert_eq!(store.first_result().unwrap(), Some((10, result(10))));
        assert_eq!(store.last_result().unwrap(), Some((14, result(14))));
        assert_eq!(store.result(12).unwrap(), Some(result(12)));
        assert_eq!(store.result(15).unwrap(), None);
        assert_eq!(store.denied(901, 0).unwrap(), vec![denied(901, 3), denied(901, 7)]);
        assert_eq!(store.denied(901, 4).unwrap(), vec![denied(901, 7)]);
        assert_eq!(store.denied(902, 0).unwrap(), vec![denied(902, 5)]);
        assert_eq!(store.denied(903, 0).unwrap(), vec![]);

        store.delete_denied(&denied(901, 3)).unwrap();
        assert_eq!(store.denied(901, 0).unwrap(), vec![denied(901, 7)]);
    }

    /// A rewind deletes the results from its timestamp on and the denials decided at or after
    /// it, on every chain, and keeps everything earlier, also across a reopen.
    #[test]
    fn rewind_deletes_later_results_and_denials() {
        let dir = tempfile::tempdir().unwrap();
        {
            let store = Store::open(dir.path()).unwrap();
            for ts in 1000..1015 {
                store.put_result(ts, &result(ts)).unwrap();
            }
            // Decided at 1000 + the block number.
            for record in [denied(901, 2), denied(902, 9), denied(901, 12), denied(902, 10)] {
                store.put_denied(&record).unwrap();
            }

            assert_eq!(store.rewind(1010).unwrap(), (5, 2));
            assert_eq!(store.rewind(1010).unwrap(), (0, 0), "a second rewind finds nothing");
        }

        let store = Store::open(dir.path()).unwrap();
        assert_eq!(store.last_result().unwrap(), Some((1009, result(1009))));
        assert_eq!(store.result(1010).unwrap(), None);
        assert_eq!(store.denied(901, 0).unwrap(), vec![denied(901, 2)]);
        assert_eq!(store.denied(902, 0).unwrap(), vec![denied(902, 9)]);
    }

    #[test]
    fn empty_store_has_no_results() {
        let dir = tempfile::tempdir().unwrap();
        let store = Store::open(dir.path()).unwrap();
        store.put_denied(&denied(901, 1)).unwrap();

        assert_eq!(store.first_result().unwrap(), None);
        assert_eq!(store.last_result().unwrap(), None);
    }

    /// The write-ahead logs of the store at `dir`.
    fn wal_files(dir: &Path) -> Vec<PathBuf> {
        fs::read_dir(dir)
            .unwrap()
            .map(|entry| entry.unwrap().path())
            .filter(|path| path.extension().is_some_and(|ext| ext == "log"))
            .collect()
    }

    /// A crash that tears the last write leaves the store as it was before that write: the
    /// results before it stay, and the torn one is gone whole.
    #[test]
    fn torn_last_write_is_dropped_whole() {
        let dir = tempfile::tempdir().unwrap();
        {
            let store = Store::open(dir.path()).unwrap();
            for ts in 10..15 {
                store.put_result(ts, &result(ts)).unwrap();
            }
        }
        let wals = wal_files(dir.path());
        let [wal] = wals.as_slice() else { panic!("one write-ahead log, got {wals:?}") };
        let len = fs::metadata(wal).unwrap().len();
        // Cut into the last record, which is larger than its encoded value alone.
        let cut = len - (BLOCK_LEN as u64);
        fs::OpenOptions::new().write(true).open(wal).unwrap().set_len(cut).unwrap();

        let store = Store::open(dir.path()).unwrap();

        assert_eq!(store.last_result().unwrap(), Some((13, result(13))));
        assert_eq!(store.result(14).unwrap(), None);
        store.put_result(14, &result(14)).unwrap();
        assert_eq!(store.last_result().unwrap(), Some((14, result(14))));
    }

    #[test]
    fn a_value_of_the_wrong_length_is_corrupt() {
        let key = result_key(1);
        let mut value = encode_result(&result(1));
        value.pop();

        assert!(matches!(decode_result(&key, &value), Err(StoreError::Corrupt { .. })));
    }

    #[test]
    fn a_result_and_a_denial_decode_to_what_was_encoded() {
        let key = result_key(3);
        assert_eq!(decode_result(&key, &encode_result(&result(3))).unwrap(), result(3));

        let dir = tempfile::tempdir().unwrap();
        let store = Store::open(dir.path()).unwrap();
        store.put_denied(&denied(901, 4)).unwrap();
        assert_eq!(store.denied(901, 4).unwrap(), vec![denied(901, 4)]);
    }
}
