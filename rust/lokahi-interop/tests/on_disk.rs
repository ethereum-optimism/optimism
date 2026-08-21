//! The property the in-memory unit tests cannot show: records written to the on-disk backend are
//! still there after the process that wrote them is gone.
//!
//! Each store is reopened from a fresh handle over the same directory, which is what a restart
//! looks like from the store's point of view.

#![cfg(feature = "rocksdb")]

use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, ChainId, U256};
use kona_protocol::OutputRoot;
use lokahi_interop::{
    ArchivedOutput, ChecksumArgs, ContainsQuery, InvalidHead, LogsDb, PendingTransition,
    RoundResult, StoreError, StoredExecutingMessage, VerifiedResult, log_hash, open_log_store,
    open_output_archive, open_verified_store,
};
use std::collections::BTreeMap;

const CHAIN: ChainId = 901;

fn head(n: u64) -> BlockNumHash {
    BlockNumHash { number: n, hash: B256::repeat_byte(n as u8) }
}

fn verified_result(timestamp: u64) -> VerifiedResult {
    VerifiedResult {
        timestamp,
        l1_inclusion: head(timestamp / 12),
        l2_heads: BTreeMap::from([(901, head(timestamp)), (902, head(timestamp + 1))]),
    }
}

#[test]
fn the_verified_frontier_and_wal_slot_survive_a_restart() {
    let dir = tempfile::tempdir().unwrap();

    {
        let store = open_verified_store(dir.path()).unwrap();
        store.commit(&verified_result(1000)).unwrap();
        store.commit(&verified_result(1001)).unwrap();
        store
            .set_pending(&PendingTransition::Advance(RoundResult {
                verified: verified_result(1002),
                invalid_heads: BTreeMap::from([(
                    902,
                    InvalidHead {
                        block: head(9),
                        state_root: B256::repeat_byte(0xaa),
                        message_passer_storage_root: B256::repeat_byte(0xbb),
                    },
                )]),
            }))
            .unwrap();
        store.backend().close();
    }

    let store = open_verified_store(dir.path()).unwrap();
    assert_eq!(store.first_timestamp(), Some(1000));
    assert_eq!(store.last_timestamp(), Some(1001));
    assert_eq!(store.get(1001).unwrap(), verified_result(1001));

    // The WAL slot is what a restart mid-apply reads: the decision is still there to re-apply,
    // and it has not been mistaken for committed history.
    let pending = store.pending().unwrap().expect("pending transition");
    assert_eq!(pending.result().verified.timestamp, 1002);
    assert_eq!(pending.result().invalid_heads.len(), 1);
    assert!(!store.has(1002).unwrap());

    // Re-applying commits the frontier; only then is the slot cleared.
    store.commit(&pending.result().verified).unwrap();
    store.clear_pending().unwrap();
    assert_eq!(store.last_timestamp(), Some(1002));
    assert_eq!(store.pending().unwrap(), None);
}

#[test]
fn sealed_blocks_survive_a_restart_and_stay_queryable() {
    let dir = tempfile::tempdir().unwrap();
    let a_log_hash = log_hash(alloy_primitives::Address::repeat_byte(7), B256::repeat_byte(7));
    let executing = StoredExecutingMessage {
        chain_id: U256::from(902),
        block_number: 5,
        log_index: 0,
        timestamp: 990,
        checksum: ChecksumArgs {
            block_number: 5,
            log_index: 0,
            timestamp: 990,
            chain_id: U256::from(902),
            log_hash: a_log_hash,
        }
        .checksum(),
    };

    {
        let store = open_log_store(dir.path(), CHAIN).unwrap();
        store.add_log(a_log_hash, head(9), 0, None).unwrap();
        store.add_log(a_log_hash, head(9), 1, Some(executing)).unwrap();
        store.seal_block(head(9).hash, head(10), 1000).unwrap();
        store.backend().close();
    }

    let store = open_log_store(dir.path(), CHAIN).unwrap();
    assert_eq!(store.latest_sealed_block(), Some(head(10)));
    assert_eq!(store.first_sealed_block().unwrap().number, 10);

    let opened = store.open_block(10).unwrap();
    assert_eq!(opened.log_count, 2);
    assert_eq!(opened.executing_messages[&1], executing);

    // The existence question the verifier asks, answered from the stored hash alone.
    let query = ContainsQuery {
        block_number: 10,
        log_index: 1,
        timestamp: 1000,
        checksum: ChecksumArgs {
            block_number: 10,
            log_index: 1,
            timestamp: 1000,
            chain_id: U256::from(CHAIN),
            log_hash: a_log_hash,
        }
        .checksum(),
    };
    assert_eq!(store.contains(&query).unwrap().id(), head(10));
    assert!(matches!(
        store.contains(&ContainsQuery { log_index: 0, ..query }),
        Err(StoreError::Conflict(_))
    ));

    // A rewind is durable too.
    store.rewind(head(10)).unwrap();
    store.add_log(a_log_hash, head(10), 0, None).unwrap();
    store.seal_block(head(10).hash, head(11), 1002).unwrap();
    store.backend().close();
    let store = open_log_store(dir.path(), CHAIN).unwrap();
    assert_eq!(store.latest_sealed_block(), Some(head(11)));
}

#[test]
fn each_chains_log_store_is_its_own_database() {
    let dir = tempfile::tempdir().unwrap();
    let first = open_log_store(dir.path(), 901).unwrap();
    let second = open_log_store(dir.path(), 902).unwrap();

    first.seal_block(head(9).hash, head(10), 1000).unwrap();
    assert_eq!(first.latest_sealed_block(), Some(head(10)));
    assert_eq!(second.latest_sealed_block(), None);

    second.seal_block(head(19).hash, head(20), 1000).unwrap();
    assert_eq!(first.latest_sealed_block(), Some(head(10)));
    assert_eq!(second.latest_sealed_block(), Some(head(20)));
}

#[test]
fn archived_outputs_survive_a_restart() {
    let dir = tempfile::tempdir().unwrap();
    let output = ArchivedOutput {
        output_root: OutputRoot::from_parts(
            B256::repeat_byte(1),
            B256::repeat_byte(2),
            B256::repeat_byte(3),
        ),
        decision_timestamp: 1000,
    };

    {
        let archive = open_output_archive(dir.path()).unwrap();
        archive.record(10, output).unwrap();
        // Idempotent: the interop actor drives this from its WAL entry on every replay.
        archive.record(10, output).unwrap();
        archive.backend().close();
    }

    let archive = open_output_archive(dir.path()).unwrap();
    assert_eq!(archive.at(10).unwrap(), vec![output]);
    assert_eq!(archive.last_at(10).unwrap(), Some(output));
    assert_eq!(archive.get(10, output.block_hash()).unwrap(), Some(output));
    assert_eq!(archive.max_height().unwrap(), Some(10));

    assert_eq!(archive.prune_at_or_after(1000).unwrap().len(), 1);
    archive.backend().close();
    let archive = open_output_archive(dir.path()).unwrap();
    assert_eq!(archive.max_height().unwrap(), None);
}

#[test]
fn the_archive_does_not_share_a_database_with_the_verified_store() {
    let dir = tempfile::tempdir().unwrap();
    let store = open_verified_store(dir.path()).unwrap();
    let archive = open_output_archive(dir.path()).unwrap();

    store.commit(&verified_result(1000)).unwrap();
    archive
        .record(
            10,
            ArchivedOutput {
                output_root: OutputRoot::from_parts(
                    B256::repeat_byte(1),
                    B256::repeat_byte(2),
                    B256::repeat_byte(3),
                ),
                decision_timestamp: 1000,
            },
        )
        .unwrap();

    // Separate files: the re-derivable store can be discarded without taking the archive, whose
    // loss is unrecoverable, with it.
    assert!(dir.path().join(lokahi_interop::VERIFIED_DIR).is_dir());
    assert!(dir.path().join(lokahi_interop::ARCHIVE_DIR).is_dir());
    assert_eq!(store.last_timestamp(), Some(1000));
    assert_eq!(archive.max_height().unwrap(), Some(10));
}
