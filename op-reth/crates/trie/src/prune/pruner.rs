use crate::{
    prune::{
        error::{OpProofStoragePrunerResult, PrunerError, PrunerOutput},
        metrics::Metrics,
    },
    BlockStateDiff, OpProofsStore,
};
use alloy_eips::{eip1898::BlockWithParent, BlockNumHash};
use reth_provider::BlockHashReader;
use tokio::time::Instant;
use tracing::{error, info, trace};

/// Prunes the proof storage by calling `prune_earliest_state` on the storage provider.
#[derive(Debug)]
pub struct OpProofStoragePruner<P, H> {
    // Database provider for the prune
    provider: P,
    /// Reader to fetch block hash by block number
    block_hash_reader: H,
    /// Keep at least these many recent blocks
    min_block_interval: u64,
    // TODO: add timeout - Maximum time for one pruner run. If `None`, no timeout.
    #[doc(hidden)]
    metrics: Metrics,
}

impl<P, H> OpProofStoragePruner<P, H> {
    /// Create a new pruner.
    pub fn new(provider: P, block_hash_reader: H, min_block_interval: u64) -> Self {
        Self { provider, block_hash_reader, min_block_interval, metrics: Metrics::default() }
    }
}

impl<P, H> OpProofStoragePruner<P, H>
where
    P: OpProofsStore,
    H: BlockHashReader,
{
    async fn run_inner(&self) -> OpProofStoragePrunerResult {
        let t = Instant::now();
        // TODO: handle timeout

        let latest_block_opt = self.provider.get_latest_block_number().await?;
        if latest_block_opt.is_none() {
            trace!(target: "trie::pruner", "No latest blocks in the proof storage");
            return Ok(PrunerOutput::default())
        }

        let earliest_block_opt = self.provider.get_earliest_block_number().await?;
        if earliest_block_opt.is_none() {
            trace!(target: "trie::pruner", "No earliest blocks in the proof storage");
            return Ok(PrunerOutput::default())
        }

        let latest_block = latest_block_opt.unwrap().0;
        let earliest_block = earliest_block_opt.unwrap().0;

        let interval = latest_block.saturating_sub(earliest_block);
        if interval <= self.min_block_interval {
            trace!(target: "trie::pruner", "Nothing to prune");
            return Ok(PrunerOutput::default())
        }

        // at this point `latest_block` is always greater than `min_block_interval`
        let new_earliest_block = latest_block - self.min_block_interval;

        info!(
            target: "trie::pruner",
            from_block = earliest_block,
            to_block = new_earliest_block,
           "Starting pruning proof storage",
        );

        let mut final_diff = BlockStateDiff::default();
        // Fetch all diffs from (earliest_block + 1) to new_earliest_block (inclusive)
        // initial proof data contains the state at `earliest_block`, so we start from
        // earliest_block + 1
        for i in (earliest_block + 1)..=new_earliest_block {
            let diff = self.provider.fetch_trie_updates(i).await.inspect_err(|err| {
                error!(
                    target: "trie::pruner",
                    block = i,
                    ?err,
                    "Failed to fetch trie updates for block during pruning"
                )
            })?;
            final_diff.extend(diff);
        }
        let stat_diff_fetch_duration = t.elapsed();

        let new_earliest_block_hash = self
            .block_hash_reader
            .block_hash(new_earliest_block)
            .inspect_err(|err| {
                error!(
                    target: "trie::pruner",
                    block = new_earliest_block,
                    ?err,
                    "Failed to fetch block hash for new earliest block during pruning"
                )
            })?
            .ok_or(PrunerError::BlockNotFound(new_earliest_block))?;

        let parent_block_num = new_earliest_block - 1;
        let parent_block_hash = self
            .block_hash_reader
            .block_hash(parent_block_num)
            .inspect_err(|err| {
                error!(
                    target: "trie::pruner",
                    block = parent_block_num,
                    ?err,
                    "Failed to fetch block hash for parent block during pruning"
                )
            })?
            .ok_or(PrunerError::BlockNotFound(parent_block_num))?;

        let block_with_parent = BlockWithParent {
            parent: parent_block_hash,
            block: BlockNumHash { number: new_earliest_block, hash: new_earliest_block_hash },
        };

        self.provider.prune_earliest_state(block_with_parent, final_diff).await?;

        let total_duration = t.elapsed();
        let prune_output = PrunerOutput {
            duration: total_duration,
            fetch_duration: stat_diff_fetch_duration,
            prune_duration: total_duration.saturating_sub(stat_diff_fetch_duration),
            start_block: earliest_block,
            end_block: new_earliest_block,
            total_entries_pruned: 0, // TODO: get it from the prune_earliest_state
        };
        self.metrics.record_prune_result(prune_output.clone());

        Ok(prune_output)
    }

    /// Run the pruner
    pub async fn run(&self) {
        let res = self.run_inner().await;
        if let Err(e) = res {
            error!(target: "trie::pruner", err=%e, "Pruner failed");
            return;
        }
        info!(target: "trie::pruner", result = %res.unwrap(), "Finished pruning proof storage");
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::db::MdbxProofsStorage;
    use alloy_eips::{BlockHashOrNumber, NumHash};
    use alloy_primitives::{BlockNumber, B256, U256};
    use mockall::mock;
    use reth_primitives_traits::Account;
    use reth_storage_errors::provider::ProviderResult;
    use reth_trie::{
        hashed_cursor::HashedCursor, trie_cursor::TrieCursor, updates::StorageTrieUpdates,
        BranchNodeCompact, HashedStorage, Nibbles,
    };
    use std::sync::Arc;
    use tempfile::TempDir;

    mock! (
        #[derive(Debug)]
        pub BlockHashReader {}

        impl BlockHashReader for BlockHashReader {
            fn block_hash(&self, number: BlockNumber) -> ProviderResult<Option<B256>>;

            fn convert_block_hash(
                &self,
                _hash_or_number: BlockHashOrNumber,
            ) -> ProviderResult<Option<B256>>;

            fn canonical_hashes_range(
                &self,
                _start: BlockNumber,
                _end: BlockNumber,
            ) -> ProviderResult<Vec<B256>>;
        }
    );

    fn b256(n: u64) -> B256 {
        use alloy_primitives::keccak256;
        keccak256(n.to_be_bytes())
    }

    /// Build a block-with-parent for number `n` with deterministic hash.
    fn block(n: u64, parent: B256) -> BlockWithParent {
        BlockWithParent::new(parent, NumHash::new(n, b256(n)))
    }

    #[tokio::test]
    async fn run_inner_and_and_verify_updated_state() {
        // --- env/store ---
        let dir = TempDir::new().unwrap();
        let store = Arc::new(MdbxProofsStorage::new(dir.path()).expect("env"));
        store.set_earliest_block_number(0, B256::ZERO).await.expect("set earliest");

        // --- entities ---
        // accounts
        let a1 = B256::from([0xA1; 32]);
        let a2 = B256::from([0xA2; 32]);
        let a3 = B256::from([0xA3; 32]); // introduced later

        // one storage address with 3 slots
        let stor_addr = B256::from([0x10; 32]);
        let s1 = B256::from([0xB1; 32]);
        let s2 = B256::from([0xB2; 32]);
        let s3 = B256::from([0xB3; 32]);

        // account-trie paths (p1 gets removed by block 3; p2 remains; p3 added later)
        let p1 = Nibbles::from_nibbles_unchecked([0x01, 0x02]);
        let p2 = Nibbles::from_nibbles_unchecked([0x03, 0x04]);
        let p3 = Nibbles::from_nibbles_unchecked([0x05, 0x06]);

        let node_p1 = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::from([0x11; 32])));
        let node_p2 = BranchNodeCompact::new(0b10, 0, 0, vec![], Some(B256::from([0x22; 32])));
        let node_p3 = BranchNodeCompact::new(0b11, 0, 0, vec![], Some(B256::from([0x33; 32])));

        // storage-trie paths (st1 removed by block 3; st2 remains; st3 added later)
        let st1 = Nibbles::from_nibbles_unchecked([0x0A]);
        let st2 = Nibbles::from_nibbles_unchecked([0x0B]);
        let st3 = Nibbles::from_nibbles_unchecked([0x0C]);

        let node_st2 = BranchNodeCompact::new(0b101, 0, 0, vec![], Some(B256::from([0x44; 32])));
        let node_st3 = BranchNodeCompact::new(0b110, 0, 0, vec![], Some(B256::from([0x55; 32])));

        // --- write 5 blocks manually ---
        let mut parent = B256::ZERO;

        // Block 1: add a1,a2; s1=100, s2=200; add p1, st1
        {
            let b1 = block(1, parent);
            let mut d = BlockStateDiff::default();

            d.post_state.accounts.insert(
                a1,
                Some(Account { nonce: 1, balance: U256::from(1_001), ..Default::default() }),
            );
            d.post_state.accounts.insert(
                a2,
                Some(Account { nonce: 1, balance: U256::from(1_002), ..Default::default() }),
            );

            let mut hs = HashedStorage::default();
            hs.storage.insert(s1, U256::from(100));
            hs.storage.insert(s2, U256::from(200));
            d.post_state.storages.insert(stor_addr, hs);

            d.trie_updates.account_nodes.insert(p1, node_p1.clone());
            let e = d.trie_updates.storage_tries.entry(stor_addr).or_default();
            e.storage_nodes.insert(st1, BranchNodeCompact::default());

            store.store_trie_updates(b1, d).await.expect("b1");
            parent = b256(1);
        }

        // Block 2: update a2; add a3; s2=220, s3=300; add p2, st2
        {
            let b2 = block(2, parent);
            let mut d = BlockStateDiff::default();

            d.post_state.accounts.insert(
                a2,
                Some(Account { nonce: 2, balance: U256::from(2_002), ..Default::default() }),
            );
            d.post_state.accounts.insert(
                a3,
                Some(Account { nonce: 1, balance: U256::from(1_003), ..Default::default() }),
            );

            let mut hs = HashedStorage::default();
            hs.storage.insert(s2, U256::from(220));
            hs.storage.insert(s3, U256::from(300));
            d.post_state.storages.insert(stor_addr, hs);

            d.trie_updates.account_nodes.insert(p2, node_p2.clone());
            let e = d.trie_updates.storage_tries.entry(stor_addr).or_default();
            e.storage_nodes.insert(st2, node_st2.clone());

            store.store_trie_updates(b2, d).await.expect("b2");
            parent = b256(2);
        }

        // Block 3: delete a1; leave a2,a3; remove p1; remove st1 (storage-trie)
        {
            let b3 = block(3, parent);
            let mut d = BlockStateDiff::default();

            // delete a1, keep a2 & a3 values unchanged for this block
            d.post_state.accounts.insert(a1, None);

            // remove account trie node p1
            d.trie_updates.removed_nodes.insert(p1);

            // remove storage-trie node st1
            let mut st_upd = StorageTrieUpdates::default();
            st_upd.removed_nodes.insert(st1);
            d.trie_updates.storage_tries.insert(stor_addr, st_upd);

            store.store_trie_updates(b3, d).await.expect("b3");
            parent = b256(3);
        }

        // Block 4 (kept): update a2; s1=140; add p3, st3
        {
            let b4 = block(4, parent);
            let mut d = BlockStateDiff::default();

            d.post_state.accounts.insert(
                a2,
                Some(Account { nonce: 3, balance: U256::from(3_002), ..Default::default() }),
            );

            let mut hs = HashedStorage::default();
            hs.storage.insert(s1, U256::from(140));
            d.post_state.storages.insert(stor_addr, hs);

            d.trie_updates.account_nodes.insert(p3, node_p3.clone());
            let e = d.trie_updates.storage_tries.entry(stor_addr).or_default();
            e.storage_nodes.insert(st3, node_st3.clone());

            store.store_trie_updates(b4, d).await.expect("b4");
            parent = b256(4);
        }

        // Block 5 (kept): update a3; s3=330
        {
            let b5 = block(5, parent);
            let mut d = BlockStateDiff::default();

            d.post_state.accounts.insert(
                a3,
                Some(Account { nonce: 2, balance: U256::from(2_003), ..Default::default() }),
            );

            let mut hs = HashedStorage::default();
            hs.storage.insert(s3, U256::from(330));
            d.post_state.storages.insert(stor_addr, hs);

            store.store_trie_updates(b5, d).await.expect("b5");
        }

        // sanity: earliest=0, latest=5
        {
            let e = store.get_earliest_block_number().await.expect("earliest").expect("some");
            let l = store.get_latest_block_number().await.expect("latest").expect("some");
            assert_eq!(e.0, 0);
            assert_eq!(l.0, 5);
        }

        // --- prune: remove the first 3 blocks, keep 4 and 5
        // new_earliest = 5-1 = 4
        let mut block_hash_reader = MockBlockHashReader::new();
        block_hash_reader
            .expect_block_hash()
            .withf(move |block_num| *block_num == 4)
            .returning(move |_| Ok(Some(b256(4))));

        block_hash_reader
            .expect_block_hash()
            .withf(move |block_num| *block_num == 3)
            .returning(move |_| Ok(Some(b256(3))));

        let pruner = OpProofStoragePruner::new(store.clone(), block_hash_reader, 1);
        let out = pruner.run_inner().await.expect("pruner ok");
        assert_eq!(out.start_block, 0);
        assert_eq!(out.end_block, 4, "pruned up to 4 (inclusive); new earliest is 4");

        // proof window moved: earliest=4, latest=5
        {
            let e = store.get_earliest_block_number().await.expect("earliest").expect("some");
            let l = store.get_latest_block_number().await.expect("latest").expect("some");
            assert_eq!(e.0, 4);
            assert_eq!(e.1, b256(4));
            assert_eq!(l.0, 5);
            assert_eq!(l.1, b256(5));
        }

        // --- DB checks
        let mut acc_cur = store.account_hashed_cursor(4).expect("acc cur");
        let mut stor_cur = store.storage_hashed_cursor(stor_addr, 4).expect("stor cur");
        let mut acc_trie_cur = store.account_trie_cursor(4).expect("acc trie cur");
        let mut stor_trie_cur = store.storage_trie_cursor(stor_addr, 4).expect("stor trie cur");

        // Check these histories have been removed
        let pruned_hashed_account = a1;
        let pruned_trie_accounts = p1;
        let pruned_trie_storage = st1;

        assert_ne!(
            acc_cur.seek(pruned_hashed_account).expect("seek").unwrap().0,
            pruned_hashed_account,
            "deleted account must not exist in earliest snapshot"
        );
        assert_ne!(
            acc_trie_cur.seek(pruned_trie_accounts).expect("seek").unwrap().0,
            pruned_trie_accounts,
            "deleted account trie must not exist in earliest snapshot"
        );
        assert_ne!(
            stor_trie_cur.seek(pruned_trie_storage).expect("seek").unwrap().0,
            pruned_trie_storage,
            "deleted storage trie must not exist in earliest snapshot"
        );

        // Check these histories have been updated - till block 4
        let updated_hashed_accounts = vec![
            (a2, Account { nonce: 3, balance: U256::from(3_002), ..Default::default() }), /* block 4 */
            (a3, Account { nonce: 1, balance: U256::from(1_003), ..Default::default() }), /* block 2 */
        ];
        let updated_hashed_storage = vec![
            (s1, U256::from(140)), // block 4
            (s2, U256::from(220)), // block 2
            (s3, U256::from(300)), // block 2
        ];
        let updated_trie_accounts = vec![
            (p2, node_p2.clone()), // block 2
            (p3, node_p3.clone()), // block 4
        ];
        let updated_trie_storage = vec![
            (st2, node_st2.clone()), // block 2
            (st3, node_st3.clone()), // block 4
        ];

        for (key, val) in updated_hashed_accounts {
            let (k, vv) = acc_cur.seek(key).expect("seek").unwrap();
            assert_eq!(key, k, "key must exist");
            assert_eq!(val, vv, "value must be updated");
        }

        for (key, val) in updated_hashed_storage {
            let (k, vv) = stor_cur.seek(key).expect("seek").unwrap();
            assert_eq!(key, k, "key must exist");
            assert_eq!(val, vv, "value must be updated");
        }

        for (key, val) in updated_trie_accounts {
            let (k, vv) = acc_trie_cur.seek(key).expect("seek").unwrap();
            assert_eq!(key, k, "key must exist");
            assert_eq!(val, vv, "value must be updated");
        }
        for (key, val) in updated_trie_storage {
            let (k, vv) = stor_trie_cur.seek(key).expect("seek").unwrap();
            assert_eq!(key, k, "key must exist");
            assert_eq!(val, vv, "value must be updated");
        }
    }

    // Both latest and earliest blocks are None -> early return default; DB untouched.
    #[tokio::test]
    async fn run_inner_where_latest_block_is_none() {
        let dir = TempDir::new().unwrap();
        let store = Arc::new(MdbxProofsStorage::new(dir.path()).expect("env"));

        let earliest = store.get_earliest_block_number().await.unwrap();
        let latest = store.get_latest_block_number().await.unwrap();
        println!("{:?} {:?}", earliest, latest);
        assert!(earliest.is_none());
        assert!(latest.is_none());

        let block_hash_reader = MockBlockHashReader::new();
        let pruner = OpProofStoragePruner::new(store, block_hash_reader, 10);
        let out = pruner.run_inner().await.expect("ok");
        assert_eq!(out, PrunerOutput::default(), "should early-return default output");
    }

    // The earliest block is None, but the latest block exists -> early return default.
    #[tokio::test]
    async fn run_inner_earliest_none_real_db() {
        use crate::BlockStateDiff;

        let dir = TempDir::new().unwrap();
        let store = Arc::new(MdbxProofsStorage::new(dir.path()).expect("env"));

        // Write a single block to set *latest* only.
        store
            .store_trie_updates(block(3, B256::ZERO), BlockStateDiff::default())
            .await
            .expect("store b1");

        let earliest = store.get_earliest_block_number().await.unwrap();
        let latest = store.get_latest_block_number().await.unwrap();
        assert!(earliest.is_none(), "earliest must remain None");
        assert_eq!(latest.unwrap().0, 3);

        let block_hash_reader = MockBlockHashReader::new();
        let pruner = OpProofStoragePruner::new(store, block_hash_reader, 1);
        let out = pruner.run_inner().await.expect("ok");
        assert_eq!(out, PrunerOutput::default(), "should early-return default output");
    }

    // interval < min_block_interval -> "Nothing to prune" path; default output.
    #[tokio::test]
    async fn run_inner_interval_too_small_real_db() {
        use crate::BlockStateDiff;

        let dir = TempDir::new().unwrap();
        let store = Arc::new(MdbxProofsStorage::new(dir.path()).expect("env"));

        // Set earliest=4 explicitly
        let earliest_num = 4u64;
        let h4 = b256(4);
        store.set_earliest_block_number(earliest_num, h4).await.expect("set earliest");

        // Set latest=5 by storing block 5
        let b5 = block(5, h4);
        store.store_trie_updates(b5, BlockStateDiff::default()).await.expect("store b5");

        // Sanity: earliest=4, latest=5 => interval=1
        let e = store.get_earliest_block_number().await.unwrap().unwrap();
        let l = store.get_latest_block_number().await.unwrap().unwrap();
        assert_eq!(e.0, 4);
        assert_eq!(l.0, 5);

        // Require min_block_interval=2 (or greater) so interval < min
        let block_hash_reader = MockBlockHashReader::new();
        let pruner = OpProofStoragePruner::new(store, block_hash_reader, 2);
        let out = pruner.run_inner().await.expect("ok");
        assert_eq!(out, PrunerOutput::default(), "no pruning should occur");
    }
}
