//! The verification round loop, driven against fake chains.
//!
//! Every test here builds a two-chain world in memory, runs whole rounds through
//! [`Verifier::step`], and asserts on what the stores hold afterwards. The fakes stand in for a
//! chain controller and a read-only execution layer; nothing else about the round loop is
//! substituted, so what is exercised is the real observe → decide → write-ahead → apply sequence.

use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, ChainId, Log, LogData, U256, address};
use alloy_sol_types::SolEvent;
use async_trait::async_trait;
use kona_genesis::{ChainGenesis, HardForkConfig, Predeploys, RollupConfig};
use kona_interop::{ExecutingMessage, MessageIdentifier};
use kona_protocol::{BlockInfo, OutputRoot};
use lokahi_interop::{
    BlockLogs, ChainAt, ChainError, ChecksumArgs, InteropChain, L1Canonical, LogStore, LogStores,
    LogsDb, MemoryKv, Pace, PendingTransition, RoundResult, VerifiedResult, VerifiedStore,
    Verifier, VerifierConfig, VerifierState, log_to_log_hash,
};
use std::{
    collections::BTreeMap,
    sync::{Arc, RwLock},
};

/// Interop activates at this timestamp on every chain in the set.
const ACTIVATION: u64 = 100;
/// The first timestamp both chains have a safe head for, and so the first round's timestamp.
const START: u64 = 110;
/// Chain ids chosen well outside the superchain registry, so the shared rules' registry-first
/// config lookup falls through to the ones these tests supply.
const CHAIN_A: ChainId = 990_901;
const CHAIN_B: ChainId = 990_902;

/// The L1 block a chain's block at `number` was derived from.
fn l1_at(number: u64) -> BlockNumHash {
    let number = 500 + number;
    BlockNumHash { number, hash: B256::from(U256::from(number)) }
}

/// One fake chain's block hash at `number`.
fn block_hash(chain_id: ChainId, number: u64) -> B256 {
    B256::from(U256::from(chain_id * 1_000_000 + number))
}

/// A block of a fake chain. Timestamps equal block numbers, so a round's timestamp names a block
/// on both chains without any flooring.
#[derive(Debug, Clone)]
struct FakeBlock {
    number: u64,
    logs: Vec<Log>,
}

impl FakeBlock {
    const fn empty(number: u64) -> Self {
        Self { number, logs: Vec::new() }
    }

    const fn with_logs(number: u64, logs: Vec<Log>) -> Self {
        Self { number, logs }
    }

    fn info(&self, chain_id: ChainId) -> BlockInfo {
        BlockInfo {
            hash: block_hash(chain_id, self.number),
            number: self.number,
            parent_hash: block_hash(chain_id, self.number - 1),
            timestamp: self.number,
        }
    }
}

/// A chain the verifier can observe, backed by an in-memory block list.
#[derive(Debug)]
struct FakeChain {
    chain_id: ChainId,
    config: RollupConfig,
    blocks: RwLock<BTreeMap<u64, FakeBlock>>,
    /// The highest timestamp this chain answers `Derived` for.
    local_safe_through: RwLock<u64>,
    /// The first timestamp the chain has a safe head for, or `None` while it has none.
    first_safe_head: RwLock<Option<u64>>,
    /// When set, the chain reports that the L1 pairing at or below this timestamp is gone.
    history_gap: RwLock<bool>,
}

impl FakeChain {
    /// A chain with empty blocks from `ACTIVATION` through 200.
    fn new(chain_id: ChainId) -> Self {
        let blocks = (ACTIVATION..=200).map(|n| (n, FakeBlock::empty(n))).collect();
        Self {
            chain_id,
            config: RollupConfig {
                block_time: 1,
                genesis: ChainGenesis { l2_time: ACTIVATION, ..Default::default() },
                hardforks: HardForkConfig { lagoon_time: Some(ACTIVATION), ..Default::default() },
                l2_chain_id: chain_id.into(),
                ..Default::default()
            },
            blocks: RwLock::new(blocks),
            local_safe_through: RwLock::new(200),
            first_safe_head: RwLock::new(Some(START)),
            history_gap: RwLock::new(false),
        }
    }

    fn set_block(&self, block: FakeBlock) {
        self.blocks.write().unwrap().insert(block.number, block);
    }

    fn set_local_safe_through(&self, timestamp: u64) {
        *self.local_safe_through.write().unwrap() = timestamp;
    }

    fn set_first_safe_head(&self, timestamp: Option<u64>) {
        *self.first_safe_head.write().unwrap() = timestamp;
    }

    fn set_history_gap(&self, gap: bool) {
        *self.history_gap.write().unwrap() = gap;
    }
}

#[async_trait]
impl InteropChain for FakeChain {
    fn chain_id(&self) -> ChainId {
        self.chain_id
    }

    fn rollup_config(&self) -> &RollupConfig {
        &self.config
    }

    async fn local_safe_at(&self, timestamp: u64) -> Result<ChainAt, ChainError> {
        if timestamp < self.config.genesis.l2_time {
            return Ok(ChainAt::BeforeGenesis);
        }
        if *self.history_gap.read().unwrap() {
            return Ok(ChainAt::HistoryUnavailable);
        }
        if timestamp > *self.local_safe_through.read().unwrap() {
            return Ok(ChainAt::NotYet);
        }
        let blocks = self.blocks.read().unwrap();
        let Some(block) = blocks.get(&timestamp) else { return Ok(ChainAt::NotYet) };
        Ok(ChainAt::Derived {
            block: BlockNumHash {
                number: block.number,
                hash: block_hash(self.chain_id, block.number),
            },
            l1: l1_at(block.number),
        })
    }

    async fn block_logs(&self, block: BlockNumHash) -> Result<BlockLogs, ChainError> {
        let blocks = self.blocks.read().unwrap();
        let fake = blocks
            .get(&block.number)
            .ok_or_else(|| ChainError::Unreachable(format!("no block {}", block.number)))?;
        Ok(BlockLogs { block: fake.info(self.chain_id), logs: fake.logs.clone() })
    }

    async fn output_at(&self, number: u64) -> Result<OutputRoot, ChainError> {
        Ok(OutputRoot::from_parts(
            B256::repeat_byte(0xaa),
            B256::repeat_byte(0xbb),
            block_hash(self.chain_id, number),
        ))
    }

    async fn first_safe_head_timestamp(&self) -> Result<u64, ChainError> {
        self.first_safe_head.read().unwrap().ok_or(ChainError::NotReady)
    }

    async fn block_number_at_timestamp(&self, timestamp: u64) -> Result<u64, ChainError> {
        // Timestamps equal block numbers in this world.
        Ok(timestamp)
    }
}

/// The L1, with every block canonical unless a test says otherwise.
#[derive(Debug, Default)]
struct FakeL1 {
    reorged: RwLock<Vec<u64>>,
}

impl FakeL1 {
    fn reorg(&self, number: u64) {
        self.reorged.write().unwrap().push(number);
    }
}

#[async_trait]
impl L1Canonical for FakeL1 {
    async fn canonical_hash_at(&self, number: u64) -> Result<B256, ChainError> {
        if self.reorged.read().unwrap().contains(&number) {
            return Ok(B256::repeat_byte(0xff));
        }
        Ok(B256::from(U256::from(number)))
    }
}

/// An ordinary log, usable as an initiating message.
fn initiating_log(seed: u8) -> Log {
    Log {
        address: address!("4200000000000000000000000000000000000023"),
        data: LogData::new_unchecked(vec![B256::repeat_byte(seed)], vec![seed, seed].into()),
    }
}

/// An executing-message log referencing an initiating message at a position.
fn executing_log(
    initiating_chain: ChainId,
    initiating: &Log,
    block_number: u64,
    log_index: u32,
    timestamp: u64,
) -> Log {
    let payload_hash =
        alloy_primitives::keccak256(kona_interop::RawMessagePayload::from(initiating).as_ref());
    let message = ExecutingMessage {
        identifier: MessageIdentifier {
            origin: initiating.address,
            blockNumber: U256::from(block_number),
            logIndex: U256::from(log_index),
            timestamp: U256::from(timestamp),
            chainId: U256::from(initiating_chain),
        },
        payloadHash: payload_hash,
    };
    Log { address: Predeploys::CROSS_L2_INBOX, data: message.encode_log_data() }
}

/// The whole world under test.
struct World {
    chain_a: Arc<FakeChain>,
    chain_b: Arc<FakeChain>,
    l1: Arc<FakeL1>,
    stores: BTreeMap<ChainId, Arc<LogStore<MemoryKv>>>,
}

impl World {
    fn new() -> Self {
        Self {
            chain_a: Arc::new(FakeChain::new(CHAIN_A)),
            chain_b: Arc::new(FakeChain::new(CHAIN_B)),
            l1: Arc::new(FakeL1::default()),
            stores: [CHAIN_A, CHAIN_B]
                .into_iter()
                .map(|id| (id, Arc::new(LogStore::new(id, MemoryKv::new()).unwrap())))
                .collect(),
        }
    }

    fn log_stores(&self) -> LogStores {
        self.stores.iter().map(|(&id, store)| (id, store.clone() as Arc<dyn LogsDb>)).collect()
    }

    fn store(&self, chain_id: ChainId) -> &Arc<LogStore<MemoryKv>> {
        &self.stores[&chain_id]
    }

    const fn config(&self) -> VerifierConfig {
        VerifierConfig {
            activation_timestamp: ACTIVATION,
            message_expiry_window: 20,
            log_backfill_depth: 5,
        }
    }

    fn verifier(&self) -> Verifier<MemoryKv> {
        self.verifier_with(VerifiedStore::new(MemoryKv::new()).unwrap())
    }

    fn verifier_with(&self, verified: VerifiedStore<MemoryKv>) -> Verifier<MemoryKv> {
        Verifier::new(
            vec![
                self.chain_a.clone() as Arc<dyn InteropChain>,
                self.chain_b.clone() as Arc<dyn InteropChain>,
            ],
            self.l1.clone(),
            verified,
            self.log_stores(),
            self.config(),
        )
        .unwrap()
    }
}

/// Runs `steps` iterations, returning the pace of each.
async fn run(verifier: &mut Verifier<MemoryKv>, steps: usize) -> Vec<Pace> {
    let mut paces = Vec::with_capacity(steps);
    for _ in 0..steps {
        paces.push(verifier.step().await.expect("the verifier did not halt"));
    }
    paces
}

#[tokio::test]
async fn cold_start_waits_for_every_chain_to_record_a_safe_head() {
    let world = World::new();
    world.chain_b.set_first_safe_head(None);
    let mut verifier = world.verifier();

    assert_eq!(verifier.state(), VerifierState::ColdStart);
    assert_eq!(verifier.step().await.unwrap(), Pace::Idle);
    // Still cold: a chain with no safe head cannot bound the starting timestamp.
    assert_eq!(verifier.state(), VerifierState::ColdStart);
    assert_eq!(verifier.first_verifiable_timestamp(), None);

    world.chain_b.set_first_safe_head(Some(START));
    assert_eq!(verifier.step().await.unwrap(), Pace::Immediate);
    assert_eq!(verifier.state(), VerifierState::Running);
    assert_eq!(verifier.first_verifiable_timestamp(), Some(START));
}

#[tokio::test]
async fn cold_start_backfills_the_window_behind_the_first_timestamp() {
    let world = World::new();
    let mut verifier = world.verifier();

    verifier.step().await.unwrap();

    // Depth 5 behind timestamp 110, so blocks 104 through 109 inclusive.
    for chain_id in [CHAIN_A, CHAIN_B] {
        let store = world.store(chain_id);
        assert_eq!(store.first_sealed_block().unwrap().number, 104);
        assert_eq!(store.latest_sealed_block().unwrap().number, 109);
    }
}

#[tokio::test]
async fn the_starting_timestamp_is_the_latest_chains_first_safe_head() {
    let world = World::new();
    world.chain_b.set_first_safe_head(Some(130));
    let mut verifier = world.verifier();

    verifier.step().await.unwrap();

    assert_eq!(verifier.first_verifiable_timestamp(), Some(130));
}

#[tokio::test]
async fn an_advancing_round_commits_the_frontier_and_seals_its_blocks() {
    let world = World::new();
    let mut verifier = world.verifier();

    // One cold-start step, then one verification round.
    assert_eq!(run(&mut verifier, 2).await, vec![Pace::Immediate, Pace::Immediate]);

    let verified = verifier.verified().get(START).unwrap();
    assert_eq!(verified.timestamp, START);
    // The frontier's L1 inclusion is the highest of the chains' own, and both chains derive
    // block 110 from the same L1 block here.
    assert_eq!(verified.l1_inclusion, l1_at(START));
    assert_eq!(
        verified.l2_heads,
        BTreeMap::from([
            (CHAIN_A, BlockNumHash { number: START, hash: block_hash(CHAIN_A, START) }),
            (CHAIN_B, BlockNumHash { number: START, hash: block_hash(CHAIN_B, START) }),
        ])
    );
    for chain_id in [CHAIN_A, CHAIN_B] {
        assert_eq!(world.store(chain_id).latest_sealed_block().unwrap().number, START);
    }
    // Nothing is left in flight once the round has applied.
    assert_eq!(verifier.verified().pending().unwrap(), None);
    assert_eq!(verifier.current_l1(), Some(l1_at(START)));
}

#[tokio::test]
async fn rounds_advance_one_timestamp_at_a_time() {
    let world = World::new();
    let mut verifier = world.verifier();

    run(&mut verifier, 4).await;

    assert_eq!(verifier.verified().first_timestamp(), Some(START));
    assert_eq!(verifier.verified().last_timestamp(), Some(START + 2));
}

#[tokio::test]
async fn a_chain_that_has_not_reached_the_timestamp_makes_the_round_wait() {
    let world = World::new();
    world.chain_b.set_local_safe_through(START);
    let mut verifier = world.verifier();

    // Cold start, one round at 110, then nothing: chain B has no block at 111.
    assert_eq!(run(&mut verifier, 3).await, vec![Pace::Immediate, Pace::Immediate, Pace::Idle]);
    assert_eq!(verifier.verified().last_timestamp(), Some(START));

    world.chain_b.set_local_safe_through(START + 1);
    assert_eq!(verifier.step().await.unwrap(), Pace::Immediate);
    assert_eq!(verifier.verified().last_timestamp(), Some(START + 1));
}

#[tokio::test]
async fn a_valid_cross_chain_message_advances_the_frontier() {
    let world = World::new();
    let initiating = initiating_log(1);
    // Chain A emits an initiating message at timestamp 111; chain B executes it at 112, by which
    // time chain A's block is sealed in its log store.
    world.chain_a.set_block(FakeBlock::with_logs(START + 1, vec![initiating.clone()]));
    world.chain_b.set_block(FakeBlock::with_logs(
        START + 2,
        vec![executing_log(CHAIN_A, &initiating, START + 1, 0, START + 1)],
    ));
    let mut verifier = world.verifier();

    // Cold start, then rounds 110, 111, 112.
    assert_eq!(run(&mut verifier, 4).await, vec![Pace::Immediate; 4]);
    assert_eq!(verifier.verified().last_timestamp(), Some(START + 2));
}

#[tokio::test]
async fn a_same_timestamp_message_is_resolved_from_the_round_itself() {
    let world = World::new();
    let initiating = initiating_log(2);
    // Both blocks are at timestamp 111, so chain A's block is not in any log store when chain B's
    // message is checked — only the round's own view can answer.
    world.chain_a.set_block(FakeBlock::with_logs(START + 1, vec![initiating.clone()]));
    world.chain_b.set_block(FakeBlock::with_logs(
        START + 1,
        vec![executing_log(CHAIN_A, &initiating, START + 1, 0, START + 1)],
    ));
    let mut verifier = world.verifier();

    assert_eq!(run(&mut verifier, 3).await, vec![Pace::Immediate; 3]);
    assert_eq!(verifier.verified().last_timestamp(), Some(START + 1));
}

#[tokio::test]
async fn a_message_referencing_a_log_that_does_not_exist_holds_the_frontier() {
    let world = World::new();
    let initiating = initiating_log(3);
    // Chain A's block at 111 holds no logs, so the referenced position is empty.
    world.chain_b.set_block(FakeBlock::with_logs(
        START + 2,
        vec![executing_log(CHAIN_A, &initiating, START + 1, 0, START + 1)],
    ));
    let mut verifier = world.verifier();

    // Cold start, 110, 111, then the round at 112 decides to invalidate and applies nothing.
    assert_eq!(
        run(&mut verifier, 5).await,
        vec![Pace::Immediate, Pace::Immediate, Pace::Immediate, Pace::Idle, Pace::Idle]
    );
    assert_eq!(verifier.verified().last_timestamp(), Some(START + 1));
    // No write-ahead entry is left behind for a decision this phase cannot apply.
    assert_eq!(verifier.verified().pending().unwrap(), None);
    assert_eq!(verifier.state(), VerifierState::Running);
}

#[tokio::test]
async fn a_message_older_than_the_expiry_window_holds_the_frontier() {
    let world = World::new();
    let initiating = initiating_log(4);
    // Sealed during backfill at timestamp 105, which is more than the 20-second expiry window
    // before the executing block at 130.
    world.chain_a.set_block(FakeBlock::with_logs(105, vec![initiating.clone()]));
    world.chain_b.set_block(FakeBlock::with_logs(
        130,
        vec![executing_log(CHAIN_A, &initiating, 105, 0, 105)],
    ));
    let mut verifier = world.verifier();
    verifier.step().await.unwrap();
    // Round 110 through 129 all advance; the round at 130 holds.
    for _ in START..130 {
        assert_eq!(verifier.step().await.unwrap(), Pace::Immediate);
    }
    assert_eq!(verifier.step().await.unwrap(), Pace::Idle);
    assert_eq!(verifier.verified().last_timestamp(), Some(129));
}

#[tokio::test]
async fn a_message_from_the_executing_blocks_future_holds_the_frontier() {
    let world = World::new();
    let initiating = initiating_log(5);
    world.chain_a.set_block(FakeBlock::with_logs(START + 3, vec![initiating.clone()]));
    // Executed at 111, referencing an initiating message at 113.
    world.chain_b.set_block(FakeBlock::with_logs(
        START + 1,
        vec![executing_log(CHAIN_A, &initiating, START + 3, 0, START + 3)],
    ));
    let mut verifier = world.verifier();

    run(&mut verifier, 3).await;
    assert_eq!(verifier.verified().last_timestamp(), Some(START));
}

#[tokio::test]
async fn a_message_in_the_activation_block_holds_the_frontier() {
    let world = World::new();
    let initiating = initiating_log(6);
    // Chain A's block at the activation timestamp is the activation block, on which interop has
    // not been active for a full block. Referencing a message there is invalid however valid the
    // reference itself looks.
    world.chain_a.set_block(FakeBlock::with_logs(ACTIVATION, vec![initiating.clone()]));
    world.chain_b.set_block(FakeBlock::with_logs(
        START + 1,
        vec![executing_log(CHAIN_A, &initiating, ACTIVATION, 0, ACTIVATION)],
    ));
    let mut verifier = world.verifier();

    run(&mut verifier, 3).await;
    assert_eq!(verifier.verified().last_timestamp(), Some(START));
}

#[tokio::test]
async fn a_message_referencing_a_chain_outside_the_set_holds_the_frontier() {
    let world = World::new();
    let initiating = initiating_log(7);
    world.chain_b.set_block(FakeBlock::with_logs(
        START + 1,
        vec![executing_log(123_456_789, &initiating, START, 0, START)],
    ));
    let mut verifier = world.verifier();

    run(&mut verifier, 3).await;
    assert_eq!(verifier.verified().last_timestamp(), Some(START));
}

#[tokio::test]
async fn a_reorged_committed_l1_inclusion_holds_the_frontier() {
    let world = World::new();
    let mut verifier = world.verifier();

    run(&mut verifier, 2).await;
    assert_eq!(verifier.verified().last_timestamp(), Some(START));

    // The L1 block the committed frontier rests on is no longer canonical.
    world.l1.reorg(l1_at(START).number);
    assert_eq!(verifier.step().await.unwrap(), Pace::Idle);
    // The frontier is neither advanced nor — in this phase — rewound.
    assert_eq!(verifier.verified().last_timestamp(), Some(START));
    assert_eq!(verifier.verified().pending().unwrap(), None);
}

#[tokio::test]
async fn a_reorged_frontier_l1_head_only_waits() {
    let world = World::new();
    let mut verifier = world.verifier();

    run(&mut verifier, 2).await;
    // The *next* round's L1 head is stale while the committed one is fine: the chains are behind
    // an L1 reorg they have not caught up with.
    world.l1.reorg(l1_at(START + 1).number);
    assert_eq!(verifier.step().await.unwrap(), Pace::Idle);
    assert_eq!(verifier.verified().last_timestamp(), Some(START));
}

#[tokio::test]
async fn a_missing_l1_pairing_halts_the_verifier() {
    let world = World::new();
    let mut verifier = world.verifier();
    run(&mut verifier, 2).await;

    world.chain_a.set_history_gap(true);
    let halted = verifier.step().await.expect_err("a history gap is not recoverable");
    assert!(halted.reason.contains("no longer recorded"), "{}", halted.reason);
    assert_eq!(verifier.state(), VerifierState::Halted);

    // It stays halted even once the gap is gone: an operator has to intervene.
    world.chain_a.set_history_gap(false);
    assert!(verifier.step().await.is_err());
    assert_eq!(verifier.verified().last_timestamp(), Some(START));
}

#[tokio::test]
async fn a_write_ahead_entry_left_by_a_crash_is_applied_before_anything_is_observed() {
    let world = World::new();
    let verified = VerifiedStore::new(MemoryKv::new()).unwrap();

    // A crash between writing the slot and committing the frontier: the log stores must still be
    // filled behind the timestamp, as a real crash would have left them.
    {
        let mut warm = world.verifier_with(VerifiedStore::new(MemoryKv::new()).unwrap());
        warm.step().await.unwrap();
    }
    let pending = PendingTransition::Advance(RoundResult {
        verified: VerifiedResult {
            timestamp: START,
            l1_inclusion: l1_at(START),
            l2_heads: BTreeMap::from([
                (CHAIN_A, BlockNumHash { number: START, hash: block_hash(CHAIN_A, START) }),
                (CHAIN_B, BlockNumHash { number: START, hash: block_hash(CHAIN_B, START) }),
            ]),
        },
        invalid_heads: BTreeMap::new(),
    });
    verified.set_pending(&pending).unwrap();

    let mut verifier = world.verifier_with(verified);
    // The slot is applied without a fresh observation, and the frontier it names is committed.
    verifier.step().await.unwrap();
    assert_eq!(verifier.verified().last_timestamp(), Some(START));
    assert_eq!(verifier.verified().pending().unwrap(), None);
}

#[tokio::test]
async fn replaying_an_already_applied_entry_changes_nothing() {
    let world = World::new();
    let mut verifier = world.verifier();
    run(&mut verifier, 2).await;

    let committed = verifier.verified().get(START).unwrap();
    // The crash case where every side effect landed but the slot had not been cleared.
    verifier
        .verified()
        .set_pending(&PendingTransition::Advance(RoundResult {
            verified: committed.clone(),
            invalid_heads: BTreeMap::new(),
        }))
        .unwrap();

    verifier.step().await.unwrap();
    assert_eq!(verifier.verified().get(START).unwrap(), committed);
    assert_eq!(verifier.verified().last_timestamp(), Some(START));
    assert_eq!(verifier.verified().pending().unwrap(), None);
}

#[tokio::test]
async fn a_restart_resumes_from_the_verified_store_without_backfilling() {
    let world = World::new();
    let verified = VerifiedStore::new(MemoryKv::new()).unwrap();
    verified
        .commit(&VerifiedResult {
            timestamp: START,
            l1_inclusion: l1_at(START),
            l2_heads: BTreeMap::from([
                (CHAIN_A, BlockNumHash { number: START, hash: block_hash(CHAIN_A, START) }),
                (CHAIN_B, BlockNumHash { number: START, hash: block_hash(CHAIN_B, START) }),
            ]),
        })
        .unwrap();

    let mut verifier = world.verifier_with(verified);
    // Already running: there is nothing to cold-start.
    assert_eq!(verifier.state(), VerifierState::Running);
    assert_eq!(verifier.first_verifiable_timestamp(), Some(START));

    // The next round is the one after the committed frontier, and the log stores were not
    // backfilled — the resume path trusts what the previous run sealed.
    assert_eq!(verifier.step().await.unwrap(), Pace::Immediate);
    assert_eq!(verifier.verified().last_timestamp(), Some(START + 1));
    assert_eq!(world.store(CHAIN_A).first_sealed_block().unwrap().number, START + 1);
}

#[tokio::test]
async fn a_chain_without_a_log_store_is_refused_at_construction() {
    let world = World::new();
    let mut stores = world.log_stores();
    stores.remove(&CHAIN_B);

    let err = Verifier::new(
        vec![
            world.chain_a.clone() as Arc<dyn InteropChain>,
            world.chain_b.clone() as Arc<dyn InteropChain>,
        ],
        world.l1.clone(),
        VerifiedStore::new(MemoryKv::new()).unwrap(),
        stores,
        world.config(),
    )
    .expect_err("a chain with no log store cannot answer existence questions");
    assert!(err.to_string().contains("has no log store"), "{err}");
}

#[test]
fn the_checksum_a_stored_message_carries_matches_the_referenced_log() {
    // The two ends of the existence question, computed independently: the executing message's
    // checksum comes from the identifier its log carried, and the log store's answer comes from
    // the initiating log's own hash at its own position.
    let initiating = initiating_log(9);
    let expected = ChecksumArgs {
        block_number: 111,
        log_index: 0,
        timestamp: 111,
        chain_id: U256::from(CHAIN_A),
        log_hash: log_to_log_hash(&initiating),
    }
    .checksum();

    let executing = executing_log(CHAIN_A, &initiating, 111, 0, 111);
    let parsed = kona_interop::parse_log_to_executing_message(&executing).unwrap();
    let actual = ChecksumArgs {
        block_number: u64::try_from(parsed.identifier.blockNumber).unwrap(),
        log_index: u32::try_from(parsed.identifier.logIndex).unwrap(),
        timestamp: u64::try_from(parsed.identifier.timestamp).unwrap(),
        chain_id: parsed.identifier.chainId,
        log_hash: lokahi_interop::log_hash(parsed.identifier.origin, parsed.payloadHash),
    }
    .checksum();

    assert_eq!(actual, expected);
}

#[tokio::test]
async fn a_same_timestamp_cycle_holds_the_frontier() {
    let world = World::new();
    // Each chain's block at timestamp 111 holds an executing message at log index 0 that
    // references the *other* chain's ordinary log at index 1, at the same timestamp. Both
    // messages are individually valid — the referenced logs exist in the round's own view — but
    // the two executing messages depend on each other's position, which is a cycle.
    let plain_a = initiating_log(0x11);
    let plain_b = initiating_log(0x22);
    world.chain_a.set_block(FakeBlock::with_logs(
        START + 1,
        vec![executing_log(CHAIN_B, &plain_b, START + 1, 1, START + 1), plain_a.clone()],
    ));
    world.chain_b.set_block(FakeBlock::with_logs(
        START + 1,
        vec![executing_log(CHAIN_A, &plain_a, START + 1, 1, START + 1), plain_b.clone()],
    ));
    let mut verifier = world.verifier();

    assert_eq!(run(&mut verifier, 3).await, vec![Pace::Immediate, Pace::Immediate, Pace::Idle]);
    assert_eq!(verifier.verified().last_timestamp(), Some(START));
    assert_eq!(verifier.verified().pending().unwrap(), None);
}

#[tokio::test]
async fn two_same_timestamp_messages_that_do_not_depend_on_each_other_advance() {
    let world = World::new();
    // The mirror of the cycle case: each chain's executing message references a log that comes
    // *before* the other chain's executing message, so neither waits on the other.
    let plain_a = initiating_log(0x33);
    let plain_b = initiating_log(0x44);
    world.chain_a.set_block(FakeBlock::with_logs(
        START + 1,
        vec![plain_a.clone(), executing_log(CHAIN_B, &plain_b, START + 1, 0, START + 1)],
    ));
    world.chain_b.set_block(FakeBlock::with_logs(
        START + 1,
        vec![plain_b.clone(), executing_log(CHAIN_A, &plain_a, START + 1, 0, START + 1)],
    ));
    let mut verifier = world.verifier();

    assert_eq!(run(&mut verifier, 3).await, vec![Pace::Immediate; 3]);
    assert_eq!(verifier.verified().last_timestamp(), Some(START + 1));
}
