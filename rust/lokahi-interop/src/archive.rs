//! The invalidated-output archive: output preimages of blocks that were replaced.
//!
//! When verification invalidates a block, the block is replaced by a deposits-only one and stops
//! being canonical anywhere. Its output root is still asked for: the superroot API's optimistic
//! branch serves it, and op-challenger reads that branch at step > 0. Nothing else in the system
//! can reconstruct it — the EL may have pruned the replaced block — so this is the one store
//! here whose loss is neither recoverable nor silent, and it gets its own database.
//!
//! Two consequences shape the API.
//!
//! A miss must be distinguishable from an answer. A caller that treats a missing record as
//! "not invalidated" serves the *replacement* block's output instead: well-formed, wrong, and
//! raising nothing. So the lookups return [`Option`] and it is the caller's job to treat
//! [`None`] at a height it believes was invalidated as an error, never as a fallthrough.
//!
//! And [`OutputArchive::record`] is idempotent. The write is driven from the interop actor's
//! write-ahead log entry, which a crash may replay onto a path that finds the block already
//! replaced and has nothing left to do. Recording must therefore be safe to repeat and must not
//! be conditional on the rest of the apply having work left — a replay that skips the write and
//! then clears the WAL entry loses the roots permanently.

use crate::{
    encoding::{Cursor, Sink},
    error::StoreError,
    kv::{Kv, WriteBatch},
};
use alloy_primitives::B256;
use kona_protocol::OutputRoot;
use std::collections::BTreeMap;

/// One archived output: the output root of an invalidated block, and the verification timestamp
/// whose decision archived it.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ArchivedOutput {
    /// The output root commitment of the invalidated block.
    pub output_root: OutputRoot,
    /// The verification timestamp at which the block was invalidated.
    pub decision_timestamp: u64,
}

impl ArchivedOutput {
    /// Returns the invalidated block's hash.
    pub const fn block_hash(&self) -> B256 {
        self.output_root.block_hash
    }
}

/// The key column holding archived outputs, keyed by big-endian block height.
mod column {
    pub(super) const ARCHIVED: u8 = 0;

    pub(super) fn key(height: u64) -> [u8; 9] {
        let mut key = [0u8; 9];
        key[0] = ARCHIVED;
        key[1..].copy_from_slice(&height.to_be_bytes());
        key
    }

    /// Returns the inclusive lower and exclusive upper bound of the column.
    pub(super) fn bounds() -> ([u8; 9], [u8; 1]) {
        (key(0), [ARCHIVED + 1])
    }
}

/// The invalidated-output archive over a [`Kv`] backend.
///
/// A height may hold more than one archived output, in the order they were recorded.
#[derive(Debug)]
pub struct OutputArchive<K> {
    kv: K,
}

impl<K: Kv> OutputArchive<K> {
    /// Opens the archive over `kv`.
    pub const fn new(kv: K) -> Self {
        Self { kv }
    }

    /// Returns the underlying backend.
    pub const fn backend(&self) -> &K {
        &self.kv
    }

    /// Archives `output` at `height`.
    ///
    /// Repeating a call for a block hash already archived at that height is a no-op, so the
    /// interop actor can drive this from its write-ahead log entry on every replay.
    pub fn record(&self, height: u64, output: ArchivedOutput) -> Result<(), StoreError> {
        let mut outputs = self.at(height)?;
        if outputs.iter().any(|existing| existing.block_hash() == output.block_hash()) {
            return Ok(());
        }
        outputs.push(output);
        let mut batch = WriteBatch::new();
        batch.put(column::key(height), encode_outputs(&outputs));
        self.kv.write(batch)
    }

    /// Returns every output archived at `height`, in the order they were recorded.
    pub fn at(&self, height: u64) -> Result<Vec<ArchivedOutput>, StoreError> {
        self.kv
            .get(&column::key(height))?
            .map_or_else(|| Ok(Vec::new()), |raw| decode_outputs(&raw))
    }

    /// Returns the most recently archived output at `height`, if any.
    ///
    /// This is the optimistic branch's answer for that height. [`None`] means the archive holds
    /// nothing there, which is not the same as "the block was valid" — see the module
    /// documentation.
    pub fn last_at(&self, height: u64) -> Result<Option<ArchivedOutput>, StoreError> {
        Ok(self.at(height)?.pop())
    }

    /// Returns the output archived at `height` for `block_hash`, if any.
    pub fn get(&self, height: u64, block_hash: B256) -> Result<Option<ArchivedOutput>, StoreError> {
        Ok(self.at(height)?.into_iter().find(|output| output.block_hash() == block_hash))
    }

    /// Returns the highest height holding an archived output, if any.
    pub fn max_height(&self) -> Result<Option<u64>, StoreError> {
        let (start, end) = column::bounds();
        let Some((key, _)) = self.kv.last_in(&start, &end)? else {
            return Ok(None);
        };
        decode_height_key(&key).map(Some)
    }

    /// Returns whether anything was archived by a decision at or after `decision_timestamp`.
    pub fn has_at_or_after(&self, decision_timestamp: u64) -> Result<bool, StoreError> {
        let (start, end) = column::bounds();
        for (_, raw) in self.kv.range(&start, &end)? {
            if decode_outputs(&raw)?
                .iter()
                .any(|output| output.decision_timestamp >= decision_timestamp)
            {
                return Ok(true);
            }
        }
        Ok(false)
    }

    /// Removes every output archived by a decision at or after `decision_timestamp`, returning
    /// the block hashes removed by height.
    ///
    /// This undoes archiving whose basis was reorged out. It must run before the engine is reset
    /// past those blocks, so that no window exists in which the archive claims a block was
    /// invalidated by a decision that no longer stands.
    pub fn prune_at_or_after(
        &self,
        decision_timestamp: u64,
    ) -> Result<BTreeMap<u64, Vec<B256>>, StoreError> {
        let (start, end) = column::bounds();
        let mut removed = BTreeMap::new();
        let mut batch = WriteBatch::new();

        for (key, raw) in self.kv.range(&start, &end)? {
            let height = decode_height_key(&key)?;
            let outputs = decode_outputs(&raw)?;
            let (kept, dropped): (Vec<_>, Vec<_>) = outputs
                .into_iter()
                .partition(|output| output.decision_timestamp < decision_timestamp);
            if dropped.is_empty() {
                continue;
            }
            removed.insert(height, dropped.iter().map(ArchivedOutput::block_hash).collect());
            if kept.is_empty() {
                batch.delete(key);
            } else {
                batch.put(key, encode_outputs(&kept));
            }
        }

        if !batch.is_empty() {
            self.kv.write(batch)?;
            tracing::info!(
                decision_timestamp,
                heights = removed.len(),
                "Pruned archived outputs whose decision basis was reorged out"
            );
        }
        Ok(removed)
    }
}

/// The format version every record in this store carries as its first byte.
const FORMAT_VERSION: u8 = 1;

/// Encodes the outputs archived at one height.
///
/// ```text
/// version (1) | count (4) | count * ( decision timestamp (8) | block hash (32)
///                                     | state root (32) | message passer storage root (32) )
/// ```
fn encode_outputs(outputs: &[ArchivedOutput]) -> Vec<u8> {
    let mut sink = Sink::with_capacity(5 + outputs.len() * 104);
    sink.put_u8(FORMAT_VERSION);
    sink.put_u32(outputs.len() as u32);
    for output in outputs {
        sink.put_u64(output.decision_timestamp);
        sink.put_b256(output.output_root.block_hash);
        sink.put_b256(output.output_root.state_root);
        sink.put_b256(output.output_root.bridge_storage_root);
    }
    sink.into_vec()
}

/// Decodes the outputs written by [`encode_outputs`].
fn decode_outputs(raw: &[u8]) -> Result<Vec<ArchivedOutput>, StoreError> {
    let mut cursor = Cursor::new(raw, "archived outputs");
    if cursor.take_u8()? != FORMAT_VERSION {
        return Err(StoreError::DataCorruption("archived outputs: unknown format version"));
    }
    let count = cursor.take_u32()?;
    let mut outputs = Vec::with_capacity(count as usize);
    for _ in 0..count {
        let decision_timestamp = cursor.take_u64()?;
        let block_hash = cursor.take_b256()?;
        let state_root = cursor.take_b256()?;
        let bridge_storage_root = cursor.take_b256()?;
        outputs.push(ArchivedOutput {
            output_root: OutputRoot::from_parts(state_root, bridge_storage_root, block_hash),
            decision_timestamp,
        });
    }
    cursor.finish()?;
    Ok(outputs)
}

/// Decodes an archive key back into its height.
fn decode_height_key(key: &[u8]) -> Result<u64, StoreError> {
    if key.len() != 9 || key[0] != column::ARCHIVED {
        return Err(StoreError::DataCorruption("archived outputs key layout"));
    }
    let bytes: [u8; 8] = key[1..]
        .try_into()
        .map_err(|_| StoreError::DataCorruption("archived outputs key layout"))?;
    Ok(u64::from_be_bytes(bytes))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::kv::MemoryKv;

    fn output(seed: u8, decision_timestamp: u64) -> ArchivedOutput {
        ArchivedOutput {
            output_root: OutputRoot::from_parts(
                B256::repeat_byte(seed),
                B256::repeat_byte(seed.wrapping_add(1)),
                B256::repeat_byte(seed.wrapping_add(2)),
            ),
            decision_timestamp,
        }
    }

    fn archive() -> OutputArchive<MemoryKv> {
        OutputArchive::new(MemoryKv::new())
    }

    #[test]
    fn an_empty_archive_answers_nothing_rather_than_erroring() {
        let archive = archive();
        assert_eq!(archive.at(10).unwrap(), Vec::new());
        assert_eq!(archive.last_at(10).unwrap(), None);
        assert_eq!(archive.get(10, B256::ZERO).unwrap(), None);
        assert_eq!(archive.max_height().unwrap(), None);
        assert!(!archive.has_at_or_after(0).unwrap());
    }

    #[test]
    fn an_archived_output_reads_back_by_height_and_by_hash() {
        let archive = archive();
        let archived = output(1, 1000);
        archive.record(10, archived).unwrap();

        assert_eq!(archive.at(10).unwrap(), vec![archived]);
        assert_eq!(archive.last_at(10).unwrap(), Some(archived));
        assert_eq!(archive.get(10, archived.block_hash()).unwrap(), Some(archived));
        assert_eq!(archive.get(10, B256::repeat_byte(0xff)).unwrap(), None);
        assert_eq!(archive.max_height().unwrap(), Some(10));
    }

    #[test]
    fn recording_the_same_block_twice_is_a_no_op() {
        let archive = archive();
        let archived = output(1, 1000);
        archive.record(10, archived).unwrap();
        archive.record(10, archived).unwrap();
        // A replay that re-runs the write must not double the record, and must not fail.
        archive.record(10, ArchivedOutput { decision_timestamp: 1005, ..archived }).unwrap();
        assert_eq!(archive.at(10).unwrap(), vec![archived]);
    }

    #[test]
    fn a_height_can_hold_more_than_one_output_in_record_order() {
        let archive = archive();
        archive.record(10, output(1, 1000)).unwrap();
        archive.record(10, output(2, 1001)).unwrap();
        assert_eq!(archive.at(10).unwrap(), vec![output(1, 1000), output(2, 1001)]);
        assert_eq!(archive.last_at(10).unwrap(), Some(output(2, 1001)));
    }

    #[test]
    fn max_height_tracks_the_highest_archived_height() {
        let archive = archive();
        archive.record(10, output(1, 1000)).unwrap();
        archive.record(300, output(2, 1001)).unwrap();
        archive.record(20, output(3, 1002)).unwrap();
        // Big-endian keys, so this is numeric order rather than lexicographic-on-decimal order.
        assert_eq!(archive.max_height().unwrap(), Some(300));
    }

    #[test]
    fn prune_removes_only_decisions_at_or_after_the_timestamp() {
        let archive = archive();
        archive.record(10, output(1, 1000)).unwrap();
        archive.record(11, output(2, 1005)).unwrap();
        archive.record(11, output(3, 1006)).unwrap();
        archive.record(12, output(4, 1007)).unwrap();

        assert!(archive.has_at_or_after(1006).unwrap());
        let removed = archive.prune_at_or_after(1006).unwrap();
        assert_eq!(
            removed,
            BTreeMap::from([
                (11, vec![output(3, 1006).block_hash()]),
                (12, vec![output(4, 1007).block_hash()]),
            ])
        );

        assert_eq!(archive.at(10).unwrap(), vec![output(1, 1000)]);
        assert_eq!(archive.at(11).unwrap(), vec![output(2, 1005)]);
        assert_eq!(archive.at(12).unwrap(), Vec::new());
        assert_eq!(archive.max_height().unwrap(), Some(11));
        assert!(!archive.has_at_or_after(1006).unwrap());
    }

    #[test]
    fn pruning_nothing_reports_nothing() {
        let archive = archive();
        archive.record(10, output(1, 1000)).unwrap();
        assert!(archive.prune_at_or_after(1001).unwrap().is_empty());
        assert_eq!(archive.at(10).unwrap(), vec![output(1, 1000)]);
    }

    #[test]
    fn a_pruned_height_can_be_archived_again() {
        let archive = archive();
        archive.record(10, output(1, 1000)).unwrap();
        archive.prune_at_or_after(1000).unwrap();
        assert_eq!(archive.at(10).unwrap(), Vec::new());
        archive.record(10, output(5, 1010)).unwrap();
        assert_eq!(archive.last_at(10).unwrap(), Some(output(5, 1010)));
    }

    #[test]
    fn truncated_and_mislabelled_records_are_corruption() {
        let encoded = encode_outputs(&[output(1, 1000), output(2, 1001)]);
        assert!(matches!(
            decode_outputs(&encoded[..encoded.len() - 1]),
            Err(StoreError::DataCorruption(_))
        ));
        let mut wrong_version = encoded;
        wrong_version[0] = FORMAT_VERSION + 1;
        assert!(matches!(decode_outputs(&wrong_version), Err(StoreError::DataCorruption(_))));
    }
}
