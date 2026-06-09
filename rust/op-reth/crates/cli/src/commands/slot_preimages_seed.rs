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
//! NOTE: the MDBX env layout below is replicated from reth's private `SlotPreimages::open` /
//! `insert_preimages` at rev `7680d6d8a931c0af4f4eed26e971596970238b54`. It must stay
//! byte-compatible with that code; revisit on every reth bump until an upstream helper exists.

use alloy_genesis::GenesisAccount;
use alloy_primitives::{Address, B256, keccak256};
use reth_db::mdbx::{
    DatabaseFlags, Environment, EnvironmentFlags, Geometry, Mode, PageSize, SyncMode, WriteFlags,
};
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

/// Opens (creating if necessary) the slot-preimage MDBX environment at `path`.
///
/// Replicates reth's `SlotPreimages::open` so the execution stage can reopen the env later.
fn open_env(path: &Path) -> eyre::Result<Environment> {
    const GIGABYTE: usize = 1024 * 1024 * 1024;
    const TERABYTE: usize = GIGABYTE * 1024;

    // Subdir mode (`no_sub_dir = false`) treats `path` as a directory holding `mdbx.dat`.
    // Unlike upstream (which only ever opens this env from the execution stage, after the
    // datadir exists), we may be the first to touch it during `init-state`, so create the
    // directory here. This does not affect the on-disk MDBX format.
    std::fs::create_dir_all(path)?;

    let mut builder = Environment::builder();
    builder.set_max_dbs(1);
    let os_page_size = page_size::get().clamp(4096, 0x10000);
    builder.set_geometry(Geometry {
        size: Some(0..(8 * TERABYTE)),
        growth_step: Some(4 * GIGABYTE as isize),
        shrink_threshold: Some(0),
        page_size: Some(PageSize::Set(os_page_size)),
    });
    builder.write_map();
    builder.set_flags(EnvironmentFlags {
        no_sub_dir: false,
        no_rdahead: true,
        mode: Mode::ReadWrite { sync_mode: SyncMode::Durable },
        ..Default::default()
    });

    let env = builder.open(path).map_err(|e| {
        eyre::eyre!("failed to open slot-preimage MDBX env at {}: {e}", path.display())
    })?;

    // Ensure the unnamed default DB exists.
    {
        let tx = env.begin_rw_txn()?;
        let _db = tx.create_db(None, DatabaseFlags::empty())?;
        tx.commit()?;
    }

    Ok(env)
}

/// Batch-inserts `hashed_slot → plain_slot` entries, skipping keys already present, and
/// clears `batch` on success so the caller can keep reusing the same allocation.
fn flush(env: &Environment, batch: &mut Vec<(B256, B256)>) -> eyre::Result<()> {
    if batch.is_empty() {
        return Ok(());
    }

    // Sorted inserts hit MDBX's append fast path.
    batch.sort_unstable_by_key(|(hashed, _)| *hashed);

    let tx = env.begin_rw_txn()?;
    let db = tx.open_db(None)?;
    let mut cursor = tx.cursor(db.dbi())?;

    for (hashed_slot, plain_slot) in batch.iter() {
        if cursor.set_key::<[u8; 32], [u8; 32]>(hashed_slot.as_slice())?.is_some() {
            continue;
        }
        cursor.put(hashed_slot.as_slice(), plain_slot.as_slice(), WriteFlags::empty())?;
    }

    tx.commit()?;
    batch.clear();
    Ok(())
}

/// Seeds the slot-preimage DB at `preimage_dir` from the state dump at `dump_path`.
///
/// Streams the dump so memory stays bounded regardless of state size. Idempotent:
/// existing preimage entries are skipped, so re-running over the same dump is safe.
pub(crate) fn seed_slot_preimages(preimage_dir: &Path, dump_path: &Path) -> eyre::Result<()> {
    info!(target: "reth::cli", path = %preimage_dir.display(), "Seeding slot preimages from state dump");

    let env = open_env(preimage_dir)?;
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
                flush(&env, &mut batch)?;
                info!(target: "reth::cli", total_slots, "Seeding slot preimages...");
            }
        }
    }
    flush(&env, &mut batch)?;

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

        // Reopen the env and confirm `keccak256(slot) → slot` for every imported slot.
        let env = open_env(&preimage_dir).unwrap();
        let tx = env.begin_ro_txn().unwrap();
        let dbi = tx.open_db(None).unwrap().dbi();
        for s in [slot_a, slot_b] {
            let got: Option<[u8; 32]> = tx.get(dbi, keccak256(s).as_ref()).unwrap();
            assert_eq!(got, Some(s.0), "missing/incorrect preimage for {s:?}");
        }
        // A slot that was never imported must be absent.
        let absent: Option<[u8; 32]> = tx.get(dbi, keccak256(slot(999)).as_ref()).unwrap();
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

        let env = open_env(&preimage_dir).unwrap();
        let tx = env.begin_ro_txn().unwrap();
        let dbi = tx.open_db(None).unwrap().dbi();
        let got: Option<[u8; 32]> = tx.get(dbi, keccak256(slot(1)).as_ref()).unwrap();
        assert_eq!(got, Some(slot(1).0));
    }
}
