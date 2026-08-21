//! The verified store: the verified frontier, plus a single-slot write-ahead log.
//!
//! One record per verified timestamp, committed one second at a time with no gaps, and one WAL
//! slot holding the effectful decision currently being applied. The WAL slot is what makes a
//! crash mid-apply recoverable: it is written and fsynced *before* any durable side effect and
//! cleared *after* the last one, so a restart either finds no slot (nothing was started) or
//! finds the decision and re-applies it.

use crate::{
    encoding::{Cursor, Sink},
    error::StoreError,
    kv::{Kv, WriteBatch},
};
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, ChainId};
use kona_protocol::OutputRoot;
use std::{
    collections::BTreeMap,
    sync::{PoisonError, RwLock},
};

/// The verified frontier at one timestamp: the L1 block that included every chain's head, and
/// each chain's L2 head as of that timestamp.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct VerifiedResult {
    /// The timestamp this frontier was verified at.
    pub timestamp: u64,
    /// The L1 block from which every head below was derived.
    pub l1_inclusion: BlockNumHash,
    /// Each chain's L2 head at `timestamp`.
    pub l2_heads: BTreeMap<ChainId, BlockNumHash>,
}

/// A block found invalid by verification, with the preimage fields the optimistic superroot
/// branch needs after the block has been replaced.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct InvalidHead {
    /// The invalidated block.
    pub block: BlockNumHash,
    /// The block's state root.
    pub state_root: B256,
    /// The storage root of the `L2ToL1MessagePasser` predeploy at that block.
    pub message_passer_storage_root: B256,
}

impl InvalidHead {
    /// Returns the output root commitment of the invalidated block.
    pub const fn output_root(&self) -> OutputRoot {
        OutputRoot::from_parts(self.state_root, self.message_passer_storage_root, self.block.hash)
    }
}

/// A verification round's outcome: the frontier it establishes, plus any head it found invalid.
///
/// A round with no invalid heads is an advance; a round with invalid heads is an invalidation.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct RoundResult {
    /// The frontier this round establishes.
    pub verified: VerifiedResult,
    /// Heads found invalid this round, by chain.
    pub invalid_heads: BTreeMap<ChainId, InvalidHead>,
}

impl RoundResult {
    /// Returns whether every head in the round was valid.
    pub fn is_valid(&self) -> bool {
        self.invalid_heads.is_empty()
    }
}

/// The effectful decision recorded in the WAL slot while it is being applied.
///
/// Only decisions with durable side effects are ever written here — a round that decides to wait
/// has nothing to recover.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum PendingTransition {
    /// Advance the verified frontier to this round's result.
    Advance(RoundResult),
    /// Invalidate this round's invalid heads and advance the rest.
    Invalidate(RoundResult),
}

impl PendingTransition {
    /// Returns the round result the transition carries.
    pub const fn result(&self) -> &RoundResult {
        match self {
            Self::Advance(result) | Self::Invalidate(result) => result,
        }
    }
}

/// Key columns of the verified store. The prefix byte keeps the columns disjoint so a bounded
/// iterator over one cannot walk into the other.
mod column {
    /// One record per verified timestamp, keyed by big-endian timestamp so lexicographic key
    /// order matches chronological order.
    pub(super) const VERIFIED: u8 = 0;
    /// The single WAL slot.
    pub(super) const PENDING: u8 = 1;

    /// Returns the key of the verified record at `timestamp`.
    pub(super) fn verified_key(timestamp: u64) -> [u8; 9] {
        let mut key = [0u8; 9];
        key[0] = VERIFIED;
        key[1..].copy_from_slice(&timestamp.to_be_bytes());
        key
    }

    /// Returns the inclusive lower and exclusive upper bound of the verified column.
    pub(super) fn verified_bounds() -> ([u8; 9], [u8; 1]) {
        (verified_key(0), [PENDING])
    }

    /// Returns the key of the WAL slot.
    pub(super) const fn pending_key() -> [u8; 1] {
        [PENDING]
    }
}

/// The committed timestamp range, cached so the hot-path bounds queries touch no I/O.
#[derive(Debug, Clone, Copy, Default)]
struct Bounds {
    first: u64,
    last: u64,
    initialized: bool,
}

/// The verified store over a [`Kv`] backend.
#[derive(Debug)]
pub struct VerifiedStore<K> {
    kv: K,
    bounds: RwLock<Bounds>,
}

impl<K: Kv> VerifiedStore<K> {
    /// Opens the store over `kv`, reading the committed timestamp range from what is there.
    pub fn new(kv: K) -> Result<Self, StoreError> {
        let store = Self { kv, bounds: RwLock::new(Bounds::default()) };
        let bounds = store.read_bounds()?;
        *store.bounds.write().unwrap_or_else(PoisonError::into_inner) = bounds;
        Ok(store)
    }

    /// Returns the underlying backend.
    pub const fn backend(&self) -> &K {
        &self.kv
    }

    /// Reads the committed timestamp range from the backend.
    fn read_bounds(&self) -> Result<Bounds, StoreError> {
        let (start, end) = column::verified_bounds();
        let Some((first_key, _)) = self.kv.first_in(&start, &end)? else {
            return Ok(Bounds::default());
        };
        let Some((last_key, _)) = self.kv.last_in(&start, &end)? else {
            return Ok(Bounds::default());
        };
        Ok(Bounds {
            first: decode_timestamp_key(&first_key)?,
            last: decode_timestamp_key(&last_key)?,
            initialized: true,
        })
    }

    fn bounds(&self) -> Bounds {
        *self.bounds.read().unwrap_or_else(PoisonError::into_inner)
    }

    /// Returns the first committed timestamp, or [`None`] if nothing is committed.
    pub fn first_timestamp(&self) -> Option<u64> {
        let bounds = self.bounds();
        bounds.initialized.then_some(bounds.first)
    }

    /// Returns the most recently committed timestamp, or [`None`] if nothing is committed.
    pub fn last_timestamp(&self) -> Option<u64> {
        let bounds = self.bounds();
        bounds.initialized.then_some(bounds.last)
    }

    /// Commits `result` at its own timestamp.
    ///
    /// Timestamps must be committed one per second with no gaps. Re-committing a timestamp the
    /// store already holds succeeds if the stored result is equal — that is the crash-replay
    /// case, where the write landed but the WAL slot had not yet been cleared — and fails with
    /// [`StoreError::AlreadyCommitted`] if it differs.
    pub fn commit(&self, result: &VerifiedResult) -> Result<(), StoreError> {
        let mut bounds = self.bounds.write().unwrap_or_else(PoisonError::into_inner);
        let timestamp = result.timestamp;

        if bounds.initialized && timestamp != bounds.last + 1 {
            if timestamp > bounds.last {
                return Err(StoreError::NonSequential {
                    expected: bounds.last + 1,
                    actual: timestamp,
                });
            }
            // Compare decoded results rather than raw bytes: a legitimate replay must not be
            // turned into a hard error by any future encoding drift.
            let existing = self.get(timestamp)?;
            return if &existing == result {
                Ok(())
            } else {
                Err(StoreError::AlreadyCommitted(timestamp))
            };
        }

        let mut batch = WriteBatch::new();
        batch.put(column::verified_key(timestamp), encode_verified_result(result));
        self.kv.write(batch)?;

        if !bounds.initialized {
            bounds.first = timestamp;
            bounds.initialized = true;
        }
        bounds.last = timestamp;
        Ok(())
    }

    /// Returns the verified result at `timestamp`.
    pub fn get(&self, timestamp: u64) -> Result<VerifiedResult, StoreError> {
        let raw =
            self.kv.get(&column::verified_key(timestamp))?.ok_or(StoreError::NotFound)?;
        decode_verified_result(&raw)
    }

    /// Returns whether a result is committed at `timestamp`.
    pub fn has(&self, timestamp: u64) -> Result<bool, StoreError> {
        Ok(self.kv.get(&column::verified_key(timestamp))?.is_some())
    }

    /// Removes every verified result at or after `timestamp`, returning whether anything was
    /// removed.
    pub fn rewind(&self, timestamp: u64) -> Result<bool, StoreError> {
        let mut bounds = self.bounds.write().unwrap_or_else(PoisonError::into_inner);
        if !bounds.initialized || timestamp > bounds.last {
            return Ok(false);
        }

        let (_, end) = column::verified_bounds();
        let mut batch = WriteBatch::new();
        batch.delete_range(column::verified_key(timestamp), end);
        self.kv.write(batch)?;

        *bounds = self.read_bounds()?;
        Ok(true)
    }

    /// Writes the WAL slot, replacing whatever it held.
    ///
    /// Returns only once the slot is durable. Callers must not begin any durable side effect of
    /// the transition before this returns.
    pub fn set_pending(&self, pending: &PendingTransition) -> Result<(), StoreError> {
        let mut batch = WriteBatch::new();
        batch.put(column::pending_key(), encode_pending(pending));
        self.kv.write(batch)
    }

    /// Returns the WAL slot's contents, or [`None`] if no transition is in flight.
    pub fn pending(&self) -> Result<Option<PendingTransition>, StoreError> {
        match self.kv.get(&column::pending_key())? {
            Some(raw) => decode_pending(&raw).map(Some),
            None => Ok(None),
        }
    }

    /// Clears the WAL slot. Callers must do this only after the transition's last durable side
    /// effect has landed.
    pub fn clear_pending(&self) -> Result<(), StoreError> {
        let mut batch = WriteBatch::new();
        batch.delete(column::pending_key());
        self.kv.write(batch)
    }
}

/// Decodes a verified-column key back into its timestamp.
fn decode_timestamp_key(key: &[u8]) -> Result<u64, StoreError> {
    if key.len() != 9 || key[0] != column::VERIFIED {
        return Err(StoreError::DataCorruption("verified key layout"));
    }
    let bytes: [u8; 8] =
        key[1..].try_into().map_err(|_| StoreError::DataCorruption("verified key layout"))?;
    Ok(u64::from_be_bytes(bytes))
}

/// The format version every record in this store carries as its first byte.
const FORMAT_VERSION: u8 = 1;

/// Encodes a [`VerifiedResult`].
///
/// ```text
/// version (1) | timestamp (8) | l1 number (8) | l1 hash (32) | head count (4)
///   | head count * ( chain id (8) | l2 number (8) | l2 hash (32) )
/// ```
///
/// The count makes the record self-describing: a decoder needs the bytes and nothing else, and
/// checks the declared count against the actual length. Chain ids are 8 bytes because this
/// record only ever names lokahi's own configured chain set.
fn encode_verified_result(result: &VerifiedResult) -> Vec<u8> {
    let mut sink = Sink::with_capacity(53 + result.l2_heads.len() * 48);
    sink.put_u8(FORMAT_VERSION);
    put_verified_body(&mut sink, result);
    sink.into_vec()
}

fn put_verified_body(sink: &mut Sink, result: &VerifiedResult) {
    sink.put_u64(result.timestamp);
    sink.put_u64(result.l1_inclusion.number);
    sink.put_b256(result.l1_inclusion.hash);
    sink.put_u32(result.l2_heads.len() as u32);
    for (chain_id, head) in &result.l2_heads {
        sink.put_u64(*chain_id);
        sink.put_u64(head.number);
        sink.put_b256(head.hash);
    }
}

/// Decodes a [`VerifiedResult`] written by [`encode_verified_result`].
fn decode_verified_result(raw: &[u8]) -> Result<VerifiedResult, StoreError> {
    let mut cursor = Cursor::new(raw, "verified result");
    if cursor.take_u8()? != FORMAT_VERSION {
        return Err(StoreError::DataCorruption("verified result: unknown format version"));
    }
    let result = take_verified_body(&mut cursor)?;
    cursor.finish()?;
    Ok(result)
}

fn take_verified_body(cursor: &mut Cursor<'_>) -> Result<VerifiedResult, StoreError> {
    let timestamp = cursor.take_u64()?;
    let l1_inclusion =
        BlockNumHash { number: cursor.take_u64()?, hash: cursor.take_b256()? };
    let head_count = cursor.take_u32()?;
    let mut l2_heads = BTreeMap::new();
    for _ in 0..head_count {
        let chain_id = cursor.take_u64()?;
        let head = BlockNumHash { number: cursor.take_u64()?, hash: cursor.take_b256()? };
        if l2_heads.insert(chain_id, head).is_some() {
            return Err(StoreError::DataCorruption("verified result: duplicate chain"));
        }
    }
    Ok(VerifiedResult { timestamp, l1_inclusion, l2_heads })
}

/// Decision tags for the WAL slot. A tag is allocated when the transition that uses it exists,
/// so an unrecognised tag is a damaged or downgraded store rather than a future decision.
mod decision_tag {
    pub(super) const ADVANCE: u8 = 1;
    pub(super) const INVALIDATE: u8 = 2;
}

/// Encodes a [`PendingTransition`].
///
/// ```text
/// version (1) | decision tag (1) | verified body | invalid count (4)
///   | invalid count * ( chain id (8) | number (8) | hash (32) | state root (32)
///                       | message passer storage root (32) )
/// ```
fn encode_pending(pending: &PendingTransition) -> Vec<u8> {
    let (tag, result) = match pending {
        PendingTransition::Advance(result) => (decision_tag::ADVANCE, result),
        PendingTransition::Invalidate(result) => (decision_tag::INVALIDATE, result),
    };
    let mut sink = Sink::with_capacity(58 + result.invalid_heads.len() * 112);
    sink.put_u8(FORMAT_VERSION);
    sink.put_u8(tag);
    put_verified_body(&mut sink, &result.verified);
    sink.put_u32(result.invalid_heads.len() as u32);
    for (chain_id, invalid) in &result.invalid_heads {
        sink.put_u64(*chain_id);
        sink.put_u64(invalid.block.number);
        sink.put_b256(invalid.block.hash);
        sink.put_b256(invalid.state_root);
        sink.put_b256(invalid.message_passer_storage_root);
    }
    sink.into_vec()
}

/// Decodes a [`PendingTransition`] written by [`encode_pending`].
fn decode_pending(raw: &[u8]) -> Result<PendingTransition, StoreError> {
    let mut cursor = Cursor::new(raw, "pending transition");
    if cursor.take_u8()? != FORMAT_VERSION {
        return Err(StoreError::DataCorruption("pending transition: unknown format version"));
    }
    let tag = cursor.take_u8()?;
    let verified = take_verified_body(&mut cursor)?;
    let invalid_count = cursor.take_u32()?;
    let mut invalid_heads = BTreeMap::new();
    for _ in 0..invalid_count {
        let chain_id = cursor.take_u64()?;
        let invalid = InvalidHead {
            block: BlockNumHash { number: cursor.take_u64()?, hash: cursor.take_b256()? },
            state_root: cursor.take_b256()?,
            message_passer_storage_root: cursor.take_b256()?,
        };
        if invalid_heads.insert(chain_id, invalid).is_some() {
            return Err(StoreError::DataCorruption("pending transition: duplicate chain"));
        }
    }
    cursor.finish()?;

    let result = RoundResult { verified, invalid_heads };
    match tag {
        decision_tag::ADVANCE => Ok(PendingTransition::Advance(result)),
        decision_tag::INVALIDATE => Ok(PendingTransition::Invalidate(result)),
        _ => Err(StoreError::DataCorruption("pending transition: unknown decision tag")),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::kv::MemoryKv;

    fn head(n: u64) -> BlockNumHash {
        BlockNumHash { number: n, hash: B256::repeat_byte(n as u8) }
    }

    fn result_at(timestamp: u64) -> VerifiedResult {
        VerifiedResult {
            timestamp,
            l1_inclusion: head(timestamp / 2),
            l2_heads: BTreeMap::from([(901, head(timestamp)), (902, head(timestamp + 1))]),
        }
    }

    fn store() -> VerifiedStore<MemoryKv> {
        VerifiedStore::new(MemoryKv::new()).unwrap()
    }

    #[test]
    fn empty_store_has_no_bounds() {
        let store = store();
        assert_eq!(store.first_timestamp(), None);
        assert_eq!(store.last_timestamp(), None);
        assert!(!store.has(1).unwrap());
        assert!(matches!(store.get(1), Err(StoreError::NotFound)));
    }

    #[test]
    fn commits_advance_the_bounds() {
        let store = store();
        store.commit(&result_at(100)).unwrap();
        store.commit(&result_at(101)).unwrap();
        assert_eq!(store.first_timestamp(), Some(100));
        assert_eq!(store.last_timestamp(), Some(101));
        assert_eq!(store.get(101).unwrap(), result_at(101));
        assert!(store.has(100).unwrap());
    }

    #[test]
    fn commit_rejects_a_gap() {
        let store = store();
        store.commit(&result_at(100)).unwrap();
        let err = store.commit(&result_at(102)).unwrap_err();
        assert!(matches!(err, StoreError::NonSequential { expected: 101, actual: 102 }));
    }

    #[test]
    fn recommitting_the_same_result_is_idempotent() {
        let store = store();
        store.commit(&result_at(100)).unwrap();
        store.commit(&result_at(101)).unwrap();
        store.commit(&result_at(101)).unwrap();
        store.commit(&result_at(100)).unwrap();
        assert_eq!(store.last_timestamp(), Some(101));
    }

    #[test]
    fn recommitting_a_different_result_is_rejected() {
        let store = store();
        store.commit(&result_at(100)).unwrap();
        let mut different = result_at(100);
        different.l2_heads.insert(903, head(7));
        assert!(matches!(
            store.commit(&different).unwrap_err(),
            StoreError::AlreadyCommitted(100)
        ));
    }

    #[test]
    fn rewind_drops_from_the_timestamp_onwards() {
        let store = store();
        for ts in 100..105 {
            store.commit(&result_at(ts)).unwrap();
        }
        assert!(store.rewind(102).unwrap());
        assert_eq!(store.first_timestamp(), Some(100));
        assert_eq!(store.last_timestamp(), Some(101));
        assert!(!store.has(102).unwrap());
        // The store is appendable again from the new frontier.
        store.commit(&result_at(102)).unwrap();
    }

    #[test]
    fn rewind_past_the_frontier_is_a_no_op() {
        let store = store();
        store.commit(&result_at(100)).unwrap();
        assert!(!store.rewind(101).unwrap());
        assert_eq!(store.last_timestamp(), Some(100));
    }

    #[test]
    fn rewinding_everything_empties_the_bounds() {
        let store = store();
        store.commit(&result_at(100)).unwrap();
        assert!(store.rewind(100).unwrap());
        assert_eq!(store.first_timestamp(), None);
        assert_eq!(store.last_timestamp(), None);
        // A fresh first commit may start anywhere.
        store.commit(&result_at(500)).unwrap();
        assert_eq!(store.first_timestamp(), Some(500));
    }

    #[test]
    fn bounds_survive_a_reopen() {
        let kv = MemoryKv::new();
        {
            let store = VerifiedStore::new(kv).unwrap();
            store.commit(&result_at(100)).unwrap();
            store.commit(&result_at(101)).unwrap();
            let reopened = VerifiedStore::new(store.kv).unwrap();
            assert_eq!(reopened.first_timestamp(), Some(100));
            assert_eq!(reopened.last_timestamp(), Some(101));
        }
    }

    #[test]
    fn wal_slot_round_trips_and_clears() {
        let store = store();
        assert_eq!(store.pending().unwrap(), None);

        let advance = PendingTransition::Advance(RoundResult {
            verified: result_at(100),
            invalid_heads: BTreeMap::new(),
        });
        store.set_pending(&advance).unwrap();
        assert_eq!(store.pending().unwrap(), Some(advance.clone()));
        assert!(advance.result().is_valid());

        let invalidate = PendingTransition::Invalidate(RoundResult {
            verified: result_at(101),
            invalid_heads: BTreeMap::from([(
                902,
                InvalidHead {
                    block: head(9),
                    state_root: B256::repeat_byte(0xaa),
                    message_passer_storage_root: B256::repeat_byte(0xbb),
                },
            )]),
        });
        store.set_pending(&invalidate).unwrap();
        assert_eq!(store.pending().unwrap(), Some(invalidate.clone()));
        assert!(!invalidate.result().is_valid());

        store.clear_pending().unwrap();
        assert_eq!(store.pending().unwrap(), None);
    }

    #[test]
    fn the_wal_slot_is_not_a_verified_record() {
        let store = store();
        store
            .set_pending(&PendingTransition::Advance(RoundResult {
                verified: result_at(100),
                invalid_heads: BTreeMap::new(),
            }))
            .unwrap();
        // A pending transition must not be mistaken for committed history, in bounds or on
        // reopen.
        assert_eq!(store.last_timestamp(), None);
        assert_eq!(VerifiedStore::new(MemoryKv::new()).unwrap().last_timestamp(), None);
    }

    #[test]
    fn invalid_head_reconstructs_its_output_root() {
        let invalid = InvalidHead {
            block: head(9),
            state_root: B256::repeat_byte(0xaa),
            message_passer_storage_root: B256::repeat_byte(0xbb),
        };
        let output = invalid.output_root();
        assert_eq!(output.block_hash, invalid.block.hash);
        assert_eq!(output.state_root, invalid.state_root);
        assert_eq!(output.bridge_storage_root, invalid.message_passer_storage_root);
    }

    #[test]
    fn truncated_and_mislabelled_records_are_corruption() {
        let encoded = encode_verified_result(&result_at(100));
        assert!(matches!(
            decode_verified_result(&encoded[..encoded.len() - 1]),
            Err(StoreError::DataCorruption(_))
        ));

        let mut trailing = encoded.clone();
        trailing.push(0);
        assert!(matches!(decode_verified_result(&trailing), Err(StoreError::DataCorruption(_))));

        let mut wrong_version = encoded;
        wrong_version[0] = FORMAT_VERSION + 1;
        assert!(matches!(
            decode_verified_result(&wrong_version),
            Err(StoreError::DataCorruption(_))
        ));
    }

    #[test]
    fn an_unknown_decision_tag_is_corruption() {
        let mut encoded = encode_pending(&PendingTransition::Advance(RoundResult {
            verified: result_at(100),
            invalid_heads: BTreeMap::new(),
        }));
        encoded[1] = 0xff;
        assert!(matches!(decode_pending(&encoded), Err(StoreError::DataCorruption(_))));
    }
}
