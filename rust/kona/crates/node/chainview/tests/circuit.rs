//! Circuit-level tests: facts in, views out, through the public client.
//!
//! Every test drives a real circuit on its own thread. `client.sync()` is the barrier that
//! guarantees the snapshot read afterwards reflects the facts pushed before it.

use std::sync::Arc;

use alloy_eips::BlockNumHash;
use alloy_primitives::{Address, B256};
use kona_chainview::{
    ChainViewClient, ChainViewConfig, ChainViewHandle, Fact, ImportedL2Block, L1StatusKind,
    L2Heads, L2SafeFact, spawn,
};
use kona_protocol::{BlockInfo, L2BlockInfo};
use op_alloy_consensus::OpBlock;

fn l1_hash(number: u64, fork: u8) -> B256 {
    let mut bytes = [0u8; 32];
    bytes[0] = fork;
    bytes[24..].copy_from_slice(&number.to_be_bytes());
    B256::from(bytes)
}

fn l1(number: u64) -> BlockInfo {
    l1_on(number, 0)
}

fn l1_on(number: u64, fork: u8) -> BlockInfo {
    BlockInfo {
        hash: l1_hash(number, fork),
        number,
        parent_hash: if number == 0 { B256::ZERO } else { l1_hash(number - 1, fork) },
        timestamp: 12 * number,
    }
}

fn l2(number: u64) -> L2BlockInfo {
    let mut bytes = [0xaau8; 32];
    bytes[24..].copy_from_slice(&number.to_be_bytes());
    L2BlockInfo {
        block_info: BlockInfo {
            hash: B256::from(bytes),
            number,
            parent_hash: B256::ZERO,
            timestamp: 2 * number,
        },
        l1_origin: BlockNumHash { number: number / 6, hash: l1_hash(number / 6, 0) },
        seq_num: number % 6,
    }
}

fn heads(safe: u64, finalized: u64) -> L2Heads {
    L2Heads {
        unsafe_head: l2(safe + 4),
        local_safe_head: l2(safe),
        safe_head: l2(safe),
        finalized_head: l2(finalized),
    }
}

struct Harness {
    handle: Option<ChainViewHandle>,
    seq: u64,
}

impl Harness {
    fn start() -> Self {
        Self::start_with(ChainViewConfig::default())
    }

    fn start_with(config: ChainViewConfig) -> Self {
        let handle = spawn(config).expect("spawn circuit");
        Self { handle: Some(handle), seq: 0 }
    }

    const fn client(&self) -> &ChainViewClient {
        &self.handle.as_ref().expect("running").client
    }

    async fn push(&self, fact: Fact) {
        self.client().push(fact).await.expect("push");
    }

    /// The pipeline advanced through `blocks`.
    async fn l1_chain(&self, blocks: impl IntoIterator<Item = BlockInfo>) {
        let mut head = None;
        for block in blocks {
            self.push(Fact::L1Origin(block)).await;
            head = Some(block);
        }
        let head = head.expect("non-empty chain");
        self.push(Fact::L1Status { kind: L1StatusKind::Head, block: head }).await;
    }

    /// The signer read from the contract at `l1`.
    async fn signer_at(&self, l1: BlockInfo, signer: Address) {
        self.push(Fact::UnsafeBlockSigner { l1, signer }).await;
    }

    async fn finalized_l1(&self, block: BlockInfo) {
        self.push(Fact::L1Status { kind: L1StatusKind::Finalized, block }).await;
    }

    async fn confirm(&mut self, l2_number: u64, derived_from: BlockInfo) -> L2SafeFact {
        self.seq += 1;
        let fact = L2SafeFact { seq: self.seq, block: l2(l2_number), derived_from };
        self.push(Fact::L2Safe(fact)).await;
        fact
    }

    async fn sync(&self) {
        self.client().sync().await.expect("sync");
    }

    async fn finalized_l2(&self) -> Option<u64> {
        self.sync().await;
        self.client().snapshot().finalized_l2.map(|f| f.id.number)
    }

    /// The held imported block whose summary is `l2(number)`.
    async fn imported(&self, number: u64) -> Option<ImportedL2Block> {
        self.client().imported_l2_block(l2(number).block_info.hash).await.expect("query")
    }

    async fn safe_head_at(&self, l1_number: u64) -> Option<(u64, u64)> {
        self.client()
            .safe_head_at_l1(l1_number)
            .await
            .expect("query")
            .map(|entry| (entry.l1.number, entry.l2.number))
    }
}

impl Drop for Harness {
    fn drop(&mut self) {
        if let Some(handle) = self.handle.take() {
            handle.shutdown();
        }
    }
}

#[tokio::test]
async fn finality_follows_derived_from_below_finalized_l1() {
    let mut h = Harness::start();
    h.l1_chain((1..=20).map(l1)).await;
    h.push(Fact::L2Status(Box::new(heads(30, 0)))).await;
    h.confirm(10, l1(5)).await;
    h.confirm(11, l1(8)).await;
    h.confirm(12, l1(11)).await;
    assert_eq!(h.finalized_l2().await, None, "no finalized L1 yet");

    h.finalized_l1(l1(10)).await;
    assert_eq!(h.finalized_l2().await, Some(11), "newest block derived at or below L1 10");

    h.push(Fact::L2Status(Box::new(heads(30, 11)))).await;
    assert_eq!(h.finalized_l2().await, None, "nothing above the engine's finalized head");

    h.finalized_l1(l1(11)).await;
    assert_eq!(h.finalized_l2().await, Some(12));
}

#[tokio::test]
async fn finality_never_exceeds_the_engine_safe_head() {
    let mut h = Harness::start();
    h.l1_chain((1..=20).map(l1)).await;
    h.push(Fact::L2Status(Box::new(heads(11, 0)))).await;
    h.confirm(11, l1(5)).await;
    h.confirm(12, l1(6)).await;
    h.finalized_l1(l1(10)).await;
    assert_eq!(h.finalized_l2().await, Some(11), "block 12 is above the engine safe head");
}

#[tokio::test]
async fn l1_reorg_retracts_derived_blocks_on_the_dropped_branch() {
    let mut h = Harness::start();
    h.l1_chain((1..=15).map(l1)).await;
    h.push(Fact::L2Status(Box::new(heads(40, 0)))).await;
    h.confirm(20, l1(10)).await;
    h.confirm(21, l1(12)).await;
    h.finalized_l1(l1(11)).await;
    assert_eq!(h.finalized_l2().await, Some(20));
    assert_eq!(h.safe_head_at(12).await, Some((12, 21)));

    // Blocks 12..=15 are replaced by a different branch as the pipeline re-walks them.
    for n in 12..=15 {
        let mut block = l1_on(n, 1);
        if n == 12 {
            block.parent_hash = l1(11).hash;
        }
        h.push(Fact::L1Origin(block)).await;
    }
    h.sync().await;
    assert_eq!(h.safe_head_at(12).await, Some((10, 20)), "block 21 left the canonical view");
    assert_eq!(h.finalized_l2().await, Some(20), "finality did not regress");

    // Re-derived on the new branch with a later seq: it wins.
    h.confirm(21, l1_on(12, 1)).await;
    assert_eq!(h.safe_head_at(12).await, Some((12, 21)));
    h.finalized_l1(l1(13)).await;
    assert_eq!(h.finalized_l2().await, Some(21));
}

#[tokio::test]
async fn pipeline_reset_retracts_blocks_above_the_reset_head() {
    let mut h = Harness::start();
    h.l1_chain((1..=15).map(l1)).await;
    h.push(Fact::L2Status(Box::new(heads(40, 0)))).await;
    h.confirm(20, l1(10)).await;
    h.confirm(21, l1(11)).await;
    h.confirm(22, l1(12)).await;
    assert_eq!(h.safe_head_at(12).await, Some((12, 22)));

    h.push(Fact::L2SafeRetractAbove { l2_number: 20, l2_hash: l2(20).block_info.hash }).await;
    h.sync().await;
    assert_eq!(h.safe_head_at(12).await, Some((10, 20)));

    // A re-derivation at the same height replaces the earlier row.
    h.confirm(21, l1(13)).await;
    assert_eq!(h.safe_head_at(13).await, Some((13, 21)));
    assert_eq!(h.safe_head_at(11).await, Some((10, 20)));
}

#[tokio::test]
async fn safe_head_lookup_is_nearest_at_or_before() {
    let mut h = Harness::start();
    h.l1_chain((1..=15).map(l1)).await;
    h.confirm(30, l1(5)).await;
    h.confirm(31, l1(8)).await;
    h.confirm(32, l1(8)).await;
    h.confirm(33, l1(11)).await;
    h.sync().await;
    assert_eq!(h.safe_head_at(4).await, None);
    assert_eq!(h.safe_head_at(5).await, Some((5, 30)));
    assert_eq!(
        h.safe_head_at(9).await,
        Some((8, 32)),
        "two confirmations at one L1: the later one"
    );
    assert_eq!(h.safe_head_at(11).await, Some((11, 33)));
    assert_eq!(h.safe_head_at(1_000).await, Some((11, 33)));
    assert_eq!(h.client().snapshot().history_len, 3);
}

#[tokio::test]
async fn signer_follows_the_read_at_each_head() {
    let h = Harness::start();
    let (s0, s1) = (Address::repeat_byte(0x01), Address::repeat_byte(0x02));
    h.signer_at(l1(10), s0).await;
    h.sync().await;
    assert_eq!(h.client().snapshot().signer, Some(s0));

    // A rotation shows up in the read at the next head.
    h.signer_at(l1(11), s1).await;
    h.sync().await;
    assert_eq!(h.client().snapshot().signer, Some(s1));

    // The rotation block is reorged out: the read at the replacement head is what counts.
    h.signer_at(l1_on(11, 1), s0).await;
    h.sync().await;
    assert_eq!(h.client().snapshot().signer, Some(s0));
}

/// The signer is read from the contract's storage at each polled head, so a rotation in a
/// block the poller never returned is in force at the next head it does return. (The baseline
/// scans the logs of the polled block only; see the `develop` characterization test
/// `a_rotation_in_a_skipped_block_is_never_seen`.)
#[tokio::test]
async fn a_rotation_in_a_skipped_block_is_not_lost() {
    let h = Harness::start();
    let (before, after) = (Address::repeat_byte(0x01), Address::repeat_byte(0x02));
    h.signer_at(l1(100), before).await;
    h.sync().await;
    assert_eq!(h.client().snapshot().signer, Some(before));

    // Block 101 rotated the signer and was never polled; the read at 102 returns the fold.
    h.signer_at(l1(102), after).await;
    h.sync().await;
    assert_eq!(h.client().snapshot().signer, Some(after));
}

#[tokio::test]
async fn finalized_l1_prunes_blocks_below_it() {
    let h = Harness::start();
    h.l1_chain((1..=10).map(l1)).await;
    h.sync().await;
    assert_eq!(h.client().snapshot().l1_window_len, 10);

    h.finalized_l1(l1(8)).await;
    h.sync().await;
    assert_eq!(h.client().snapshot().l1_window_len, 3, "8, 9 and 10 remain");
}

#[tokio::test]
async fn late_rows_are_rejected_and_counted() {
    let mut h = Harness::start();
    h.l1_chain((1..=6000).map(l1)).await;
    h.confirm(1, l1(5_000)).await;
    h.sync().await;
    assert_eq!(h.client().snapshot().lateness_drops, 0);

    // 4096 is the table's LATENESS: a derived-from block further back than that is late.
    h.confirm(2, l1(800)).await;
    h.sync().await;
    assert_eq!(h.client().snapshot().lateness_drops, 1);
    assert_eq!(h.safe_head_at(800).await, None, "the late row never entered the view");

    // Inside the window, a retraction is honored.
    let kept = h.confirm(3, l1(4_000)).await;
    h.sync().await;
    assert_eq!(h.safe_head_at(4_000).await, Some((4_000, 3)));
    h.push(Fact::L2SafeRetractAbove {
        l2_number: kept.block.block_info.number - 1,
        l2_hash: l2(kept.block.block_info.number - 1).block_info.hash,
    })
    .await;
    h.sync().await;
    assert_eq!(h.safe_head_at(4_000).await, None, "retracted inside the window");
    assert_eq!(h.safe_head_at(5_000).await, Some((5_000, 1)));
}

/// An imported block whose summary is `l2(number)`.
fn imported(number: u64) -> ImportedL2Block {
    let mut block = OpBlock::default();
    block.header.number = number;
    ImportedL2Block { info: l2(number), block: Arc::new(block) }
}

#[tokio::test]
async fn imported_blocks_are_held_until_finalized_or_pushed_out_by_newer_ones() {
    let h = Harness::start_with(ChainViewConfig { imported_limit: 3, ..Default::default() });
    for number in 1..=3 {
        h.push(Fact::L2Imported(imported(number))).await;
    }
    assert_eq!(h.imported(1).await, Some(imported(1)), "held with its block");

    // A fourth block is one more than the limit: the lowest goes.
    h.push(Fact::L2Imported(imported(4))).await;
    assert_eq!(h.imported(1).await, None);
    assert!(h.imported(2).await.is_some() && h.imported(4).await.is_some());

    // Finalizing block 3 drops everything below it; 3 itself stays.
    h.push(Fact::L2Status(Box::new(heads(4, 3)))).await;
    assert_eq!(h.imported(2).await, None);
    assert_eq!(h.imported(3).await.map(|b| b.info.block_info.number), Some(3));
    assert!(h.imported(5).await.is_none(), "never imported");
}
