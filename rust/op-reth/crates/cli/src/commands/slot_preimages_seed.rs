//! Seed the slot-preimage database during state-dump import.
//!
//! reth's V2 storage layout (`--storage.v2`, hashed state as the canonical state
//! representation) keeps storage under `HashedStorages` (keyed by `keccak256(slot)`).
//! For pre-Cancun blocks, `SELFDESTRUCT` wipes storage, and the execution stage must
//! rewrite the storage-changeset reverts using *plain* slot keys. To recover a plain
//! key from its hash it consults an auxiliary MDBX database at `<datadir>/db/preimage`,
//! populated only while executing blocks (see reth
//! `crates/stages/stages/src/stages/execution/slot_preimages.rs`).
//!
//! A state-dump import (`init-state --without-ovm`, i.e. the OP Mainnet Bedrock bootstrap)
//! writes hashed storage directly and never populates that preimage DB. Any contract whose
//! storage came only from the imported snapshot and is later self-destructed in a pre-Ecotone
//! block then fails the execution stage with `missing slot preimage for 0x… (addr=0x…)`.
//!
//! This module closes that gap: it re-reads the dump (which carries plain slot keys) and seeds
//! `keccak256(slot) → slot` for every imported slot, so the execution stage can resolve them.
//! The preimage DB is address-independent and reth deletes it once Ecotone (Cancun) activates.
//!
//! The upstream [`SlotPreimages`] API owns the MDBX layout and insertion behavior. This module
//! only handles streaming the OP state-dump format and batching its slot preimages.

use alloy_genesis::GenesisAccount;
use alloy_primitives::{Address, B256, keccak256};
use reth_stages::stages::slot_preimages::SlotPreimages;
use serde::Deserialize;
use std::{
    io::{BufRead, BufReader},
    path::Path,
};
use tracing::info;

/// Number of preimage entries to accumulate before flushing a write transaction.
const FLUSH_THRESHOLD: usize = 1_000_000;

/// A single account as laid out in the state-dump file. Mirrors reth's private
/// `GenesisAccountWithAddress`; only the storage map is used here, but `address` must be
/// present for the flattened deserialization to match the dump format.
#[derive(Deserialize)]
struct DumpAccount {
    #[serde(flatten)]
    account: GenesisAccount,
    #[allow(dead_code)]
    address: Address,
}

/// Batch-inserts `hashed_slot → plain_slot` entries, skipping keys already present, and
/// clears `batch` on success so the caller can keep reusing the same allocation.
fn flush(preimages: &SlotPreimages, batch: &mut Vec<(B256, B256)>) -> eyre::Result<()> {
    if batch.is_empty() {
        return Ok(());
    }

    // Sorted inserts hit MDBX's append fast path.
    batch.sort_unstable_by_key(|(hashed, _)| *hashed);

    preimages.insert_preimages(batch)?;
    batch.clear();
    Ok(())
}

/// Seeds the slot-preimage DB at `preimage_dir` from the state dump at `dump_path`.
///
/// Streams the dump so memory stays bounded regardless of state size. Idempotent:
/// existing preimage entries are skipped, so re-running over the same dump is safe.
pub(crate) fn seed_slot_preimages(preimage_dir: &Path, dump_path: &Path) -> eyre::Result<()> {
    info!(target: "reth::cli", path = %preimage_dir.display(), "Seeding slot preimages from state dump");

    // `SlotPreimages::open` uses MDBX subdirectory mode and expects the directory to exist.
    std::fs::create_dir_all(preimage_dir)?;
    let preimages = SlotPreimages::open(preimage_dir)?;
    let mut reader = BufReader::new(reth_fs_util::open(dump_path)?);

    // First line is the state root; the remaining lines are accounts.
    let mut header = String::new();
    reader.read_line(&mut header)?;

    let mut batch: Vec<(B256, B256)> = Vec::with_capacity(FLUSH_THRESHOLD);
    let mut total_slots: u64 = 0;

    let stream = serde_json::Deserializer::from_reader(reader).into_iter::<DumpAccount>();
    for account in stream {
        let DumpAccount { account, .. } = account?;
        let Some(storage) = account.storage else { continue };
        for slot in storage.keys() {
            batch.push((keccak256(slot), *slot));
            total_slots += 1;
            if batch.len() >= FLUSH_THRESHOLD {
                flush(&preimages, &mut batch)?;
                info!(target: "reth::cli", total_slots, "Seeding slot preimages...");
            }
        }
    }
    flush(&preimages, &mut batch)?;

    info!(target: "reth::cli", total_slots, "Slot preimages seeded");
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::U256;
    use std::{collections::BTreeMap, io::Write};
    use tempfile::tempdir;

    fn slot(n: u64) -> B256 {
        B256::from(U256::from(n).to_be_bytes())
    }

    /// Renders one dump line: a flattened [`GenesisAccount`] plus an `address` field, matching
    /// the format produced/consumed by reth's state dump.
    fn dump_line(address: Address, storage: BTreeMap<B256, B256>) -> String {
        let account =
            GenesisAccount { balance: U256::ZERO, storage: Some(storage), ..Default::default() };
        let mut value = serde_json::to_value(&account).unwrap();
        value
            .as_object_mut()
            .unwrap()
            .insert("address".to_string(), serde_json::to_value(address).unwrap());
        value.to_string()
    }

    #[test]
    fn seeds_and_roundtrips_preimages() {
        let dir = tempdir().unwrap();
        let dump_path = dir.path().join("state.jsonl");
        let preimage_dir = dir.path().join("preimage");

        let (slot_a, slot_b) = (slot(1), slot(42));
        let storage = BTreeMap::from([(slot_a, slot(7)), (slot_b, slot(7))]);

        let mut file = std::fs::File::create(&dump_path).unwrap();
        // First line is the state root, then one account per line.
        writeln!(file, "{}", serde_json::json!({ "root": B256::ZERO })).unwrap();
        writeln!(file, "{}", dump_line(Address::ZERO, storage)).unwrap();
        drop(file);

        seed_slot_preimages(&preimage_dir, &dump_path).unwrap();

        // Reopen the store and confirm `keccak256(slot) → slot` for every imported slot.
        let preimages = SlotPreimages::open(&preimage_dir).unwrap();
        let reader = preimages.reader().unwrap();
        for s in [slot_a, slot_b] {
            let got = reader.get(&keccak256(s)).unwrap();
            assert_eq!(got, Some(s), "missing/incorrect preimage for {s:?}");
        }
        // A slot that was never imported must be absent.
        let absent = reader.get(&keccak256(slot(999))).unwrap();
        assert!(absent.is_none(), "unexpected preimage for non-imported slot");
    }

    #[test]
    fn seeding_is_idempotent() {
        let dir = tempdir().unwrap();
        let dump_path = dir.path().join("state.jsonl");
        let preimage_dir = dir.path().join("preimage");

        let mut file = std::fs::File::create(&dump_path).unwrap();
        writeln!(file, "{}", serde_json::json!({ "root": B256::ZERO })).unwrap();
        // Storage value (slot(2)) is irrelevant; only the key's preimage is recorded.
        writeln!(file, "{}", dump_line(Address::ZERO, BTreeMap::from([(slot(1), slot(2))])))
            .unwrap();
        drop(file);

        // Running twice over the same dump must not error (existing keys are skipped).
        seed_slot_preimages(&preimage_dir, &dump_path).unwrap();
        seed_slot_preimages(&preimage_dir, &dump_path).unwrap();

        let preimages = SlotPreimages::open(&preimage_dir).unwrap();
        let reader = preimages.reader().unwrap();
        let got = reader.get(&keccak256(slot(1))).unwrap();
        assert_eq!(got, Some(slot(1)));
    }
}
