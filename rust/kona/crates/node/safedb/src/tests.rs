//! Unit tests for the safe-head database.

#[cfg(feature = "rocksdb")]
mod encoding {
    use crate::{encoding::SafeByL1BlockNum, error::SafeDbError};
    use alloy_eips::BlockNumHash;
    use alloy_primitives::{B256, b256};
    use rstest::rstest;

    fn l1() -> BlockNumHash {
        BlockNumHash {
            hash: b256!("0100000000000000000000000000000000000000000000000000000000000000"),
            number: 84298,
        }
    }

    fn l2() -> BlockNumHash {
        BlockNumHash {
            hash: b256!("0200000000000000000000000000000000000000000000000000000000000000"),
            number: 3224,
        }
    }

    #[test]
    fn roundtrip() {
        let key = SafeByL1BlockNum::key(l1().number);
        let val = SafeByL1BlockNum::value(l1(), l2());
        let record = SafeByL1BlockNum::decode(&key, &val).unwrap();
        assert_eq!(l1(), record.l1);
        assert_eq!(l2(), record.safe_head);
    }

    #[rstest]
    #[case::empty_key(vec![], valid_value())]
    #[case::too_short_key(vec![1, 2, 3, 4], valid_value())]
    #[case::too_long_key(too_long_key(), valid_value())]
    #[case::wrong_key_prefix(wrong_prefix_key(), valid_value())]
    #[case::empty_value(valid_key().to_vec(), vec![])]
    #[case::too_short_value(valid_key().to_vec(), vec![1, 2, 3, 4])]
    #[case::too_long_value(valid_key().to_vec(), too_long_value())]
    fn invalid_entries(#[case] key: Vec<u8>, #[case] val: Vec<u8>) {
        assert!(matches!(SafeByL1BlockNum::decode(&key, &val), Err(SafeDbError::InvalidEntry)));
    }

    #[test]
    fn key_layout_is_exact() {
        // One prefix byte (0x00) followed by the big-endian L1 block number.
        let key = SafeByL1BlockNum::key(0x0102_0304_0506_0708);
        assert_eq!(key, [0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08]);
    }

    #[test]
    fn value_layout_is_exact() {
        let l1 = BlockNumHash { hash: B256::repeat_byte(0xab), number: 0 };
        let l2 = BlockNumHash { hash: B256::repeat_byte(0xcd), number: 0x1122_3344_5566_7788 };
        let value = SafeByL1BlockNum::value(l1, l2);

        let mut expected = [0u8; 72];
        expected[0..32].fill(0xab);
        expected[32..64].fill(0xcd);
        expected[64..72].copy_from_slice(&0x1122_3344_5566_7788u64.to_be_bytes());
        assert_eq!(value, expected);
    }

    #[test]
    fn keys_follow_natural_byte_ordering() {
        let vals: [u64; 7] = [
            0,
            1,
            u32::MAX as u64 - 1,
            u32::MAX as u64,
            u32::MAX as u64 + 1,
            u64::MAX - 1,
            u64::MAX,
        ];
        for window in vals.windows(2) {
            let prev = SafeByL1BlockNum::key(window[0]);
            let cur = SafeByL1BlockNum::key(window[1]);
            assert!(prev < cur, "expected key for {} to be less than {}", window[0], window[1]);
        }
    }

    fn valid_key() -> [u8; SafeByL1BlockNum::KEY_LEN] {
        SafeByL1BlockNum::key(l1().number)
    }

    fn valid_value() -> Vec<u8> {
        SafeByL1BlockNum::value(l1(), l2()).to_vec()
    }

    fn too_long_key() -> Vec<u8> {
        let mut key = valid_key().to_vec();
        key.push(2);
        key
    }

    fn wrong_prefix_key() -> Vec<u8> {
        let mut key = valid_key();
        key[0] = 49;
        key.to_vec()
    }

    fn too_long_value() -> Vec<u8> {
        let mut val = valid_value();
        val.push(2);
        val
    }
}

#[cfg(feature = "rocksdb")]
mod safe_db {
    use crate::{SafeDatabase, SafeDb, SafeDbError, SafeHeadRecord};
    use alloy_eips::BlockNumHash;
    use alloy_primitives::B256;
    use kona_protocol::{BlockInfo, L2BlockInfo};
    use rstest::rstest;
    use tempfile::TempDir;

    fn hash(a: u8, b: u8) -> B256 {
        let mut bytes = [0u8; 32];
        bytes[0] = a;
        bytes[1] = b;
        B256::from(bytes)
    }

    fn id(a: u8, b: u8, number: u64) -> BlockNumHash {
        BlockNumHash { hash: hash(a, b), number }
    }

    fn l2(a: u8, b: u8, number: u64, l1_origin_number: u64) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo {
                hash: hash(a, b),
                number,
                parent_hash: B256::ZERO,
                timestamp: 0,
            },
            l1_origin: BlockNumHash { hash: B256::ZERO, number: l1_origin_number },
            seq_num: 0,
        }
    }

    fn open() -> (TempDir, SafeDatabase) {
        let dir = TempDir::new().unwrap();
        let db = SafeDatabase::new(dir.path()).unwrap();
        (dir, db)
    }

    fn assert_record(record: SafeHeadRecord, l1: BlockNumHash, safe_head: BlockNumHash) {
        assert_eq!(record.l1, l1);
        assert_eq!(record.safe_head, safe_head);
    }

    #[test]
    fn store_safe_heads() {
        let dir = TempDir::new().unwrap();
        let l2a = l2(0x02, 0xaa, 20, 0);
        let l2b = l2(0x02, 0xbb, 25, 0);
        let l1a = id(0x01, 0xaa, 100);
        let l1b = id(0x01, 0xbb, 150);

        let verify = |db: &SafeDatabase| {
            assert!(matches!(db.safe_head_at_l1(l1a.number - 1), Err(SafeDbError::NotFound)));
            assert_record(db.safe_head_at_l1(l1a.number).unwrap(), l1a, l2a.block_info.id());
            assert_record(db.safe_head_at_l1(l1a.number + 1).unwrap(), l1a, l2a.block_info.id());
            assert_record(db.safe_head_at_l1(l1b.number).unwrap(), l1b, l2b.block_info.id());
            assert_record(db.safe_head_at_l1(l1b.number + 1).unwrap(), l1b, l2b.block_info.id());
        };

        let db = SafeDatabase::new(dir.path()).unwrap();
        db.safe_head_updated(l2a, l1a).unwrap();
        db.safe_head_updated(l2b, l1b).unwrap();
        verify(&db);

        // Close and reopen to confirm the data is durable.
        db.close().unwrap();
        let reopened = SafeDatabase::new(dir.path()).unwrap();
        verify(&reopened);
    }

    #[test]
    fn safe_head_at_l1_empty_database() {
        let (_dir, db) = open();
        assert!(matches!(db.safe_head_at_l1(100), Err(SafeDbError::NotFound)));
    }

    #[test]
    fn first_entry_empty_database() {
        let (_dir, db) = open();
        assert!(matches!(db.first_entry(), Err(SafeDbError::NotFound)));
    }

    #[test]
    fn first_entry_returns_lowest_l1() {
        let (_dir, db) = open();
        let l2a = l2(0x02, 0xaa, 20, 0);
        let l2b = l2(0x02, 0xbb, 25, 0);
        let l1a = id(0x01, 0xaa, 100);
        let l1b = id(0x01, 0xbb, 150);

        // Insert out of order to confirm we return the lowest L1, not the first-inserted entry.
        db.safe_head_updated(l2b, l1b).unwrap();
        db.safe_head_updated(l2a, l1a).unwrap();

        assert_record(db.first_entry().unwrap(), l1a, l2a.block_info.id());
    }

    #[test]
    fn first_entry_stable_after_reset_ahead() {
        let (_dir, db) = open();
        let l1a = id(0x01, 0xaa, 100);
        let l1b = id(0x01, 0xbb, 150);
        let l2a = l2(0x02, 0xaa, 20, l1a.number);
        let l2b = l2(0x02, 0xbb, 25, l1b.number);

        db.safe_head_updated(l2a, l1a).unwrap();
        db.safe_head_updated(l2b, l1b).unwrap();

        // Resetting to l2b truncates entries at or after it; the l2a entry must remain first.
        db.safe_head_reset(l2b).unwrap();

        assert_record(db.first_entry().unwrap(), l1a, l2a.block_info.id());
    }

    #[test]
    fn last_entry_empty_database() {
        let (_dir, db) = open();
        assert!(matches!(db.last_entry(), Err(SafeDbError::NotFound)));
    }

    #[test]
    fn last_entry_returns_highest_l1() {
        let (_dir, db) = open();
        let l2a = l2(0x02, 0xaa, 20, 0);
        let l2b = l2(0x02, 0xbb, 25, 0);
        let l1a = id(0x01, 0xaa, 100);
        let l1b = id(0x01, 0xbb, 150);

        // Insert out of order to confirm we return the highest L1, not the last-inserted entry.
        db.safe_head_updated(l2b, l1b).unwrap();
        db.safe_head_updated(l2a, l1a).unwrap();

        assert_record(db.last_entry().unwrap(), l1b, l2b.block_info.id());
    }

    #[test]
    fn reports_enabled() {
        let (_dir, db) = open();
        assert!(db.enabled());
    }

    #[test]
    fn truncate_on_safe_head_reset() {
        let (_dir, db) = open();
        let l2a = l2(0x02, 0xaa, 20, 60);
        let l2b = l2(0x02, 0xbb, 22, 90);
        let l2c = l2(0x02, 0xcc, 25, 110);
        let l2d = l2(0x02, 0xdd, 30, 120);
        let l1a = id(0x01, 0xaa, 100);
        let l1b = id(0x01, 0xbb, 150);
        let l1c = id(0x01, 0xcc, 160);

        db.safe_head_updated(l2a, l1a).unwrap();
        db.safe_head_updated(l2c, l1b).unwrap();
        db.safe_head_updated(l2d, l1c).unwrap();

        // Reset to between the existing entries.
        db.safe_head_reset(l2b).unwrap();

        // Only the reset safe head is now safe at the previous L1 block number.
        assert_record(db.safe_head_at_l1(l1b.number).unwrap(), l1b, l2b.block_info.id());
        assert_record(db.safe_head_at_l1(l1c.number).unwrap(), l1b, l2b.block_info.id());
        // l2a is still safe from its original update.
        assert_record(db.safe_head_at_l1(l1a.number).unwrap(), l1a, l2a.block_info.id());
    }

    #[test]
    fn truncate_on_safe_head_reset_before_first_entry() {
        let (_dir, db) = open();
        let l2b = l2(0x02, 0xbb, 22, 90);
        let l2c = l2(0x02, 0xcc, 25, 110);
        let l2d = l2(0x02, 0xdd, 30, 120);
        let l1a = id(0x01, 0xaa, 100);
        let l1b = id(0x01, 0xbb, 150);
        let l1c = id(0x01, 0xcc, 160);

        db.safe_head_updated(l2c, l1b).unwrap();
        db.safe_head_updated(l2d, l1c).unwrap();

        // Reset to before all existing entries removes everything.
        db.safe_head_reset(l2b).unwrap();

        assert!(matches!(db.safe_head_at_l1(l1a.number), Err(SafeDbError::NotFound)));
        assert!(matches!(db.safe_head_at_l1(l1b.number), Err(SafeDbError::NotFound)));
        assert!(matches!(db.safe_head_at_l1(l1c.number), Err(SafeDbError::NotFound)));
    }

    #[test]
    fn truncate_on_safe_head_reset_after_last_entry() {
        let (_dir, db) = open();
        let l2a = l2(0x02, 0xaa, 20, 60);
        let l2b = l2(0x02, 0xbb, 22, 90);
        let l2c = l2(0x02, 0xcc, 25, 110);
        let l1a = id(0x01, 0xaa, 100);
        let l1b = id(0x01, 0xbb, 150);
        let l1c = id(0x01, 0xcc, 160);

        db.safe_head_updated(l2a, l1a).unwrap();
        db.safe_head_updated(l2b, l1b).unwrap();
        db.safe_head_updated(l2c, l1c).unwrap();

        let verify = |db: &SafeDatabase| {
            assert_record(db.safe_head_at_l1(l1a.number).unwrap(), l1a, l2a.block_info.id());
            assert_record(db.safe_head_at_l1(l1b.number).unwrap(), l1b, l2b.block_info.id());
            assert_record(db.safe_head_at_l1(l1c.number).unwrap(), l1c, l2c.block_info.id());
        };
        verify(&db);

        // Reset to an L2 block after all entries with an origin after all L1 entries.
        db.safe_head_reset(l2(0x02, 0xdd, 30, l1c.number + 1)).unwrap();
        verify(&db);

        // Reset to an L2 block after all entries with an origin before some L1 entries.
        db.safe_head_reset(l2(0x02, 0xdd, 30, l1b.number - 1)).unwrap();
        verify(&db);
    }

    /// A reset that wipes the database, followed by a refill from a later point, moves the floor
    /// *up* -- and a target below the new floor is permanently unanswerable.
    ///
    /// Each half is already covered: `truncate_on_safe_head_reset_before_first_entry` pins the
    /// wipe, and `l1_at_safe_head`'s `target_below_first` case pins the verdict below the floor.
    /// What was not covered is the two in sequence, which is the only way the floor ever rises.
    /// Entries are otherwise appended above it and `first_entry_stable_after_reset_ahead` shows a
    /// reset ahead of the floor leaves it alone, so a target that answered once keeps answering.
    /// After a wipe-and-refill it does not.
    ///
    /// The distinction that matters to a reader is which error it becomes. While the database
    /// holds nothing the answer is `L1AtSafeHeadNotFound`, which means "not yet" and is worth
    /// retrying; once it has refilled above the old floor the same target is
    /// `L1AtSafeHeadUnavailable`, which is permanent. A reader that resolved its lower bound once
    /// and cached it is therefore wrong from here on, through no fault of its own -- which is why
    /// `lokahi_interop::Verifier` re-reads that bound every round instead of latching it at
    /// startup.
    #[test]
    fn a_wipe_and_refill_moves_the_floor_up_permanently() {
        let (_dir, db) = open();

        // A database recording from L2 block 500 upwards.
        db.safe_head_updated(l2(0x02, 0xaa, 500, 40), id(0x01, 0xaa, 100)).unwrap();
        db.safe_head_updated(l2(0x02, 0xbb, 510, 41), id(0x01, 0xbb, 110)).unwrap();
        assert_eq!(db.first_entry().unwrap().safe_head.number, 500);
        // The floor answers, so a reader may legitimately take it as its lower bound.
        assert_record(db.l1_at_safe_head(500).unwrap(), id(0x01, 0xaa, 100), id(0x02, 0xaa, 500));

        // A reset at or before the earliest entry deletes everything and re-records nothing:
        // there is no previous entry whose L1 block the reset head could be attached to, so the
        // database cannot know whether that head became safe there or before its records began.
        db.safe_head_reset(l2(0x02, 0x99, 499, 39)).unwrap();
        assert!(matches!(db.first_entry(), Err(SafeDbError::NotFound)));
        // Holding nothing, every target is a retry rather than a verdict.
        assert!(matches!(db.l1_at_safe_head(500), Err(SafeDbError::L1AtSafeHeadNotFound)));

        // Derivation resumes from where it now is, so the refill begins above the old floor.
        db.safe_head_updated(l2(0x02, 0xcc, 700, 60), id(0x01, 0xcc, 200)).unwrap();
        db.safe_head_updated(l2(0x02, 0xdd, 710, 61), id(0x01, 0xdd, 210)).unwrap();

        // The floor has risen, and the target that answered before the wipe is now permanent.
        assert_eq!(db.first_entry().unwrap().safe_head.number, 700);
        assert!(matches!(db.l1_at_safe_head(500), Err(SafeDbError::L1AtSafeHeadUnavailable)));

        // The new floor answers, and so does a target above it.
        assert_record(db.l1_at_safe_head(700).unwrap(), id(0x01, 0xcc, 200), id(0x02, 0xcc, 700));
        assert_record(db.l1_at_safe_head(705).unwrap(), id(0x01, 0xdd, 210), id(0x02, 0xdd, 710));
    }

    fn populated_l1_at_safe_head_db() -> (TempDir, SafeDatabase) {
        let (dir, db) = open();
        db.safe_head_updated(l2(0x02, 0xaa, 500, 0), id(0x01, 0xaa, 100)).unwrap();
        db.safe_head_updated(l2(0x02, 0xbb, 510, 0), id(0x01, 0xbb, 110)).unwrap();
        db.safe_head_updated(l2(0x02, 0xcc, 520, 0), id(0x01, 0xcc, 120)).unwrap();
        (dir, db)
    }

    #[derive(Debug)]
    enum Expected {
        Record(BlockNumHash, BlockNumHash),
        NotFound,
        Unavailable,
    }

    #[rstest]
    #[case::target_equals_first(500, Expected::Record(id(0x01, 0xaa, 100), id(0x02, 0xaa, 500)))]
    #[case::target_between_entries(505, Expected::Record(id(0x01, 0xbb, 110), id(0x02, 0xbb, 510)))]
    #[case::target_equals_recorded(510, Expected::Record(id(0x01, 0xbb, 110), id(0x02, 0xbb, 510)))]
    #[case::target_equals_latest(520, Expected::Record(id(0x01, 0xcc, 120), id(0x02, 0xcc, 520)))]
    #[case::target_above_latest(521, Expected::NotFound)]
    #[case::target_below_first(499, Expected::Unavailable)]
    fn l1_at_safe_head(#[case] target: u64, #[case] expected: Expected) {
        let (_dir, db) = populated_l1_at_safe_head_db();
        match (db.l1_at_safe_head(target), expected) {
            (Ok(record), Expected::Record(l1, safe_head)) => assert_record(record, l1, safe_head),
            (Err(SafeDbError::L1AtSafeHeadNotFound), Expected::NotFound) |
            (Err(SafeDbError::L1AtSafeHeadUnavailable), Expected::Unavailable) => {}
            (result, expected) => panic!("unexpected result {result:?} for {expected:?}"),
        }
    }

    #[test]
    fn l1_at_safe_head_empty_database() {
        let (_dir, db) = open();
        assert!(matches!(db.l1_at_safe_head(500), Err(SafeDbError::L1AtSafeHeadNotFound)));
    }

    #[test]
    fn operations_after_close_report_closed() {
        let (_dir, db) = open();
        db.close().unwrap();
        assert!(matches!(db.safe_head_at_l1(100), Err(SafeDbError::Closed)));
        // Close is idempotent.
        db.close().unwrap();
    }

    #[test]
    fn open_on_locked_path_returns_backend_error() {
        // rocksdb holds an exclusive lock on an open database directory, so a second open of the
        // same path surfaces a backend error.
        let (dir, _held) = open();
        assert!(matches!(SafeDatabase::new(dir.path()), Err(SafeDbError::Backend(_))));
    }
}

mod disabled {
    use crate::{DisabledDatabase, SafeDb, SafeDbError};

    #[test]
    fn l1_at_safe_head_disabled() {
        assert!(matches!(DisabledDatabase.l1_at_safe_head(500), Err(SafeDbError::NotEnabled)));
    }

    #[test]
    fn reports_disabled() {
        assert!(!DisabledDatabase.enabled());
    }
}
