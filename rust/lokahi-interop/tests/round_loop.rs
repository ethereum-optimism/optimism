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
    BlockLogs, ChainAt, ChainError, ChainReplacement, ChecksumArgs, InteropChain, InvalidHead,
    L1Canonical, LogStore, LogStores, LogsDb, MemoryKv, OutputArchive, Pace, PendingTransition,
    RewindableChain, RoundResult, VerifiedResult, VerifiedStore, Verifier, VerifierConfig,
    VerifierState, log_to_log_hash,
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
    /// When set, the chain reports that its genesis is later than the round's timestamp.
    before_genesis: RwLock<bool>,
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
            before_genesis: RwLock::new(false),
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

    fn set_before_genesis(&self, before: bool) {
        *self.before_genesis.write().unwrap() = before;
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
        if timestamp < self.config.genesis.l2_time || *self.before_genesis.read().unwrap() {
            return Ok(ChainAt::BeforeGenesis);
        }
        if *self.history_gap.read().unwrap() {
            return Ok(ChainAt::HistoryUnavailable);
        }
        // What the real safe-head database does, which this fake used to skip. `l1_at_safe_head`
        // walks back from the tip and answers `L1AtSafeHeadUnavailable` — the permanent verdict —
        // for a block below its earliest record, and `L1AtSafeHeadNotFound` — a retry — when it
        // holds nothing at all (`kona/crates/node/safedb/src/safe_db.rs:170-198`). Answering
        // `Derived` for any timestamp with a block, as this did, is why a start left below a
        // chain's earliest record read as healthy here and halted in production.
        match *self.first_safe_head.read().unwrap() {
            Some(first) if timestamp < first => return Ok(ChainAt::HistoryUnavailable),
            None => return Ok(ChainAt::NotYet),
            Some(_) => {}
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

/// The rewind seam over a fake chain, recording every invocation.
///
/// A real rewind takes the chain back onto the invalidated block's parent; here that is modelled
/// as the chain's local-safe coverage dropping below the invalidated height, so the next round
/// observes `NotYet` until the test installs the replacement block the way derivation would.
#[derive(Debug)]
struct FakeRewind {
    chain: Arc<FakeChain>,
    /// Every block `rewind_off` was invoked for, in order.
    calls: RwLock<Vec<BlockNumHash>>,
    /// When set, the chain reports it is already off the block: no rewind happens.
    already_replaced: RwLock<bool>,
    /// When set, every rewind attempt fails transiently.
    unreachable: RwLock<bool>,
}

impl FakeRewind {
    const fn new(chain: Arc<FakeChain>) -> Self {
        Self {
            chain,
            calls: RwLock::new(Vec::new()),
            already_replaced: RwLock::new(false),
            unreachable: RwLock::new(false),
        }
    }

    fn calls(&self) -> Vec<BlockNumHash> {
        self.calls.read().unwrap().clone()
    }

    fn set_already_replaced(&self, already: bool) {
        *self.already_replaced.write().unwrap() = already;
    }

    fn set_unreachable(&self, unreachable: bool) {
        *self.unreachable.write().unwrap() = unreachable;
    }
}

#[async_trait]
impl RewindableChain for FakeRewind {
    async fn rewind_off(&self, invalidated: BlockNumHash) -> Result<bool, ChainError> {
        self.calls.write().unwrap().push(invalidated);
        if *self.unreachable.read().unwrap() {
            return Err(ChainError::Unreachable("the chain controller is down".into()));
        }
        if *self.already_replaced.read().unwrap() {
            return Ok(false);
        }
        self.chain.set_local_safe_through(invalidated.number - 1);
        Ok(true)
    }
}

/// The whole world under test.
struct World {
    chain_a: Arc<FakeChain>,
    chain_b: Arc<FakeChain>,
    l1: Arc<FakeL1>,
    stores: BTreeMap<ChainId, Arc<LogStore<MemoryKv>>>,
    archives: BTreeMap<ChainId, Arc<OutputArchive<MemoryKv>>>,
    rewinds: BTreeMap<ChainId, Arc<FakeRewind>>,
}

impl World {
    fn new() -> Self {
        let chain_a = Arc::new(FakeChain::new(CHAIN_A));
        let chain_b = Arc::new(FakeChain::new(CHAIN_B));
        Self {
            rewinds: BTreeMap::from([
                (CHAIN_A, Arc::new(FakeRewind::new(chain_a.clone()))),
                (CHAIN_B, Arc::new(FakeRewind::new(chain_b.clone()))),
            ]),
            chain_a,
            chain_b,
            l1: Arc::new(FakeL1::default()),
            stores: [CHAIN_A, CHAIN_B]
                .into_iter()
                .map(|id| (id, Arc::new(LogStore::new(id, MemoryKv::new()).unwrap())))
                .collect(),
            archives: [CHAIN_A, CHAIN_B]
                .into_iter()
                .map(|id| (id, Arc::new(OutputArchive::new(MemoryKv::new()))))
                .collect(),
        }
    }

    /// The invalidation routes over this world's chains.
    fn replacements(&self) -> BTreeMap<ChainId, ChainReplacement<MemoryKv>> {
        [CHAIN_A, CHAIN_B]
            .into_iter()
            .map(|id| {
                (
                    id,
                    ChainReplacement {
                        archive: self.archives[&id].clone(),
                        chain: self.rewinds[&id].clone() as Arc<dyn RewindableChain>,
                    },
                )
            })
            .collect()
    }

    /// A verifier that applies invalidations, wired the way a supernode wires it.
    fn replacing_verifier(&self) -> Verifier<MemoryKv> {
        self.replacing_verifier_with(VerifiedStore::new(MemoryKv::new()).unwrap())
    }

    /// A replacing verifier over a prepared verified store.
    fn replacing_verifier_with(&self, verified: VerifiedStore<MemoryKv>) -> Verifier<MemoryKv> {
        self.verifier_of(verified, self.config())
            .with_replacements(self.replacements())
            .expect("every chain has a route")
    }

    fn log_stores(&self) -> LogStores {
        self.stores.iter().map(|(&id, store)| (id, store.clone() as Arc<dyn LogsDb>)).collect()
    }

    fn store(&self, chain_id: ChainId) -> &Arc<LogStore<MemoryKv>> {
        &self.stores[&chain_id]
    }

    /// Backfill is as deep as the expiry window, as the real default is: a shallower store could
    /// not answer every message a round is allowed to reference, which the verifier now refuses to
    /// resume on.
    const fn config(&self) -> VerifierConfig {
        VerifierConfig {
            activation_timestamp: ACTIVATION,
            message_expiry_window: 20,
            log_backfill_depth: 20,
        }
    }

    fn verifier(&self) -> Verifier<MemoryKv> {
        self.verifier_with(VerifiedStore::new(MemoryKv::new()).unwrap())
    }

    fn verifier_with(&self, verified: VerifiedStore<MemoryKv>) -> Verifier<MemoryKv> {
        self.verifier_of(verified, self.config())
    }

    fn verifier_with_config(&self, config: VerifierConfig) -> Verifier<MemoryKv> {
        self.verifier_of(VerifiedStore::new(MemoryKv::new()).unwrap(), config)
    }

    fn verifier_of(
        &self,
        verified: VerifiedStore<MemoryKv>,
        config: VerifierConfig,
    ) -> Verifier<MemoryKv> {
        Verifier::new(
            vec![
                self.chain_a.clone() as Arc<dyn InteropChain>,
                self.chain_b.clone() as Arc<dyn InteropChain>,
            ],
            self.l1.clone(),
            verified,
            self.log_stores(),
            config,
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
    // A depth shallower than the expiry window, to pin the depth rather than the activation clamp.
    let mut verifier =
        world.verifier_with_config(VerifierConfig { log_backfill_depth: 5, ..world.config() });

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

/// A timestamp below a chain's genesis cannot resolve by waiting, but it is still not a halt.
///
/// op-supernode reaches the same condition through `TargetBlockNumber`, whose plain error its
/// interop activity logs and backs off on rather than halting, and lokahi mirrors op-supernode.
/// The round makes no progress and warns every attempt, but the verifier stays alive and picks
/// straight back up once the disagreement is resolved.
#[tokio::test]
async fn a_timestamp_below_a_chains_genesis_is_retried_rather_than_halting() {
    let world = World::new();
    let mut verifier = world.verifier();
    run(&mut verifier, 2).await;

    world.chain_a.set_before_genesis(true);
    let pace = verifier.step().await.expect("a pre-genesis timestamp must not halt the verifier");
    assert_eq!(pace, Pace::Retry);
    assert_eq!(verifier.state(), VerifierState::Running);
    assert_eq!(verifier.verified().last_timestamp(), Some(START));

    // Still alive, and still not advancing, for as long as the disagreement stands.
    assert_eq!(verifier.step().await.unwrap(), Pace::Retry);
    assert_eq!(verifier.state(), VerifierState::Running);

    // Once the chain set and the start agree again, the loop carries on where it stopped.
    world.chain_a.set_before_genesis(false);
    assert_eq!(verifier.step().await.unwrap(), Pace::Immediate);
    assert_eq!(verifier.verified().last_timestamp(), Some(START + 1));
}

/// A safe-head database wiped by a derivation reset and refilled higher must not be terminal.
///
/// `safe_head_reset` deletes every entry when the reset target is at or before the first one, and
/// deliberately re-records nothing (`kona/crates/node/safedb/src/safe_db.rs:100-121`); the
/// database then refills from wherever derivation now is. The wipe on its own was always
/// survivable, because an empty database is a retry. What halted the verifier for good was the
/// refill coming back *above* the start cold start had latched, leaving every round asking below
/// the earliest record. op-supernode never latches — it re-reads `FirstSafeHeadTimestamp` on
/// every backoff pass (`op-node/rollup/interop/log_backfill.go:71`, `:85-95`) — so neither does
/// lokahi now, and while nothing is committed the start follows the history up instead.
#[tokio::test]
async fn a_safe_head_history_that_moves_up_before_any_commit_raises_the_start() {
    let world = World::new();
    let mut verifier = world.verifier();

    // Cold start chooses a start from the history as it stands, and commits nothing yet.
    assert_eq!(verifier.step().await.unwrap(), Pace::Immediate);
    assert_eq!(verifier.verification_start(), Some(START));
    assert_eq!(verifier.verified().last_timestamp(), None);

    // A derivation reset at or before the earliest record wipes the database outright. Nothing
    // can be paired while it holds nothing, and the round simply waits.
    world.chain_a.set_first_safe_head(None);
    assert_eq!(verifier.step().await.unwrap(), Pace::Idle);
    assert_eq!(verifier.state(), VerifierState::Running);
    assert_eq!(verifier.verification_start(), Some(START));

    // It refills from where derivation now is, above the start already chosen. This is the step
    // that used to halt the verifier permanently.
    world.chain_a.set_first_safe_head(Some(START + 20));
    assert_eq!(verifier.step().await.unwrap(), Pace::Immediate);
    assert_eq!(verifier.state(), VerifierState::Running);
    assert_eq!(verifier.verification_start(), Some(START + 20));

    // And it verifies from the timestamp the history can actually answer for, rather than from
    // one no chain can.
    assert_eq!(verifier.verified().last_timestamp(), Some(START + 20));
    assert_eq!(verifier.step().await.unwrap(), Pace::Immediate);
    assert_eq!(verifier.verified().last_timestamp(), Some(START + 21));
}

/// Once a frontier is committed the start cannot move, so the same gap is correctly terminal.
///
/// This is the resume path's check as much as the running one's: resuming takes `last_verified +
/// 1` from the verified store and consults no safe-head database at all, and the startup coverage
/// check reads only the log stores. The frontier has to stay contiguous, so a history that no
/// longer reaches the next timestamp is a real gap — named here, at the bound, rather than
/// several reads later as a chain that cannot answer.
#[tokio::test]
async fn a_safe_head_history_that_moves_up_after_a_commit_halts_with_the_gap_named() {
    let world = World::new();
    let mut verifier = world.verifier();
    run(&mut verifier, 2).await;
    assert_eq!(verifier.verified().last_timestamp(), Some(START));

    world.chain_a.set_first_safe_head(Some(START + 20));
    let halted = verifier.step().await.expect_err("a committed frontier cannot skip a gap");
    assert!(halted.reason.contains("must continue from"), "{}", halted.reason);
    assert_eq!(verifier.state(), VerifierState::Halted);
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

    // The log stores must hold what the run being resumed sealed: backfill behind its start, plus
    // the round at 110 it committed. Resuming onto empty stores is not a restart any run can
    // produce, and is refused — see `a_restart_onto_a_lost_log_store_halts_the_verifier`.
    {
        let mut previous = world.verifier_with(VerifiedStore::new(MemoryKv::new()).unwrap());
        run(&mut previous, 2).await;
    }
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
    // backfilled — the resume path trusts what the previous run sealed and only adds its own
    // round's blocks on top.
    assert_eq!(verifier.step().await.unwrap(), Pace::Immediate);
    assert_eq!(verifier.verified().last_timestamp(), Some(START + 1));
    assert_eq!(world.store(CHAIN_A).first_sealed_block().unwrap().number, ACTIVATION);
    assert_eq!(world.store(CHAIN_A).latest_sealed_block().unwrap().number, START + 1);
}

/// Part two of the missing-history fix: the resume path checks that the log stores still cover the
/// window rounds will ask about, instead of assuming it.
#[tokio::test]
async fn a_restart_onto_a_lost_log_store_halts_the_verifier() {
    let world = World::new();
    let verified = VerifiedStore::new(MemoryKv::new()).unwrap();
    // A committed frontier at 110, but log stores holding nothing: the verified store survived and
    // the per-chain log databases did not.
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

    // Halted before the first round, with the chain and the window named — not `Running`, which is
    // what an unchecked resume would report while never verifying anything again.
    assert_eq!(verifier.state(), VerifierState::Halted);
    let halted = verifier.step().await.expect_err("a lost log store is not recoverable");
    assert!(halted.reason.contains("the log store is empty"), "{}", halted.reason);
    assert!(
        halted.reason.contains(&format!("resumes at timestamp {}", START + 1)),
        "{}",
        halted.reason
    );
    // Nothing was verified, and it stays halted.
    assert_eq!(verifier.verified().last_timestamp(), Some(START));
    assert!(verifier.step().await.is_err());
}

/// Part one of the missing-history fix: a store whose history starts *above* a referenced
/// timestamp that the rules allow. The message is valid as far as the protocol is concerned, so the
/// verifier must halt rather than charge the block with a violation it cannot substantiate.
#[tokio::test]
async fn a_message_the_log_store_lost_its_history_for_halts_the_verifier() {
    let world = World::new();
    let initiating = initiating_log(8);
    // Chain A's block at 102 holds the initiating message. It is inside the 20-second expiry
    // window of the executing block at 111 and past activation, so every rule above the store
    // lookup passes — but a depth-5 backfill only reaches back to 104, so the store cannot answer.
    world.chain_a.set_block(FakeBlock::with_logs(102, vec![initiating.clone()]));
    world.chain_b.set_block(FakeBlock::with_logs(
        START + 1,
        vec![executing_log(CHAIN_A, &initiating, 102, 0, 102)],
    ));
    let mut verifier =
        world.verifier_with_config(VerifierConfig { log_backfill_depth: 5, ..world.config() });

    // Cold start, then the round at 110 advances.
    assert_eq!(run(&mut verifier, 2).await, vec![Pace::Immediate; 2]);
    assert_eq!(world.store(CHAIN_A).first_sealed_block().unwrap().number, 104);

    let halted = verifier.step().await.expect_err("a hole in local history is not recoverable");
    assert!(
        halted.reason.contains("local log history does not cover timestamp 102"),
        "{}",
        halted.reason
    );
    assert_eq!(verifier.state(), VerifierState::Halted);
    // The frontier held where it was rather than invalidating a block the node cannot judge, and
    // no write-ahead entry was left behind.
    assert_eq!(verifier.verified().last_timestamp(), Some(START));
    assert_eq!(verifier.verified().pending().unwrap(), None);
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

// ---------------------------------------------------------------------------
// Test control
//
// The pause and the cold-start attempt counter exist for the acceptance suites, which drive them
// over RPC through a supernode's test-control API. They are asserted here, against the real round
// loop, because what the suites depend on is the loop's behaviour rather than the plumbing: a
// pause the loop checked in the wrong place would still be a pause the RPC set successfully.
// ---------------------------------------------------------------------------

#[tokio::test]
async fn a_pause_stops_the_loop_at_the_timestamp_it_names() {
    let world = World::new();
    let mut verifier = world.verifier();

    // Cold start, then two rounds: START and START + 1 commit.
    run(&mut verifier, 3).await;
    assert_eq!(verifier.verified().last_timestamp(), Some(START + 1));

    verifier.set_pause(Some(START + 2));

    // Every further round is a no-op. The pause is checked where the loop picks its timestamp, so
    // nothing is observed and nothing is committed.
    assert_eq!(run(&mut verifier, 3).await, vec![Pace::Idle, Pace::Idle, Pace::Idle]);
    assert_eq!(verifier.verified().last_timestamp(), Some(START + 1));
}

#[tokio::test]
async fn a_pause_the_loop_is_already_past_stops_it_too() {
    let world = World::new();
    let mut verifier = world.verifier();

    run(&mut verifier, 3).await;
    assert_eq!(verifier.verified().last_timestamp(), Some(START + 1));

    // Behind where the loop has reached. The check is inclusive and forward-looking — as
    // op-supernode's is — so a test that asks late is not silently given a running verifier.
    verifier.set_pause(Some(START));

    assert_eq!(run(&mut verifier, 2).await, vec![Pace::Idle, Pace::Idle]);
    assert_eq!(verifier.verified().last_timestamp(), Some(START + 1));
}

#[tokio::test]
async fn clearing_a_pause_lets_the_loop_carry_on_where_it_stopped() {
    let world = World::new();
    let mut verifier = world.verifier();

    run(&mut verifier, 2).await;
    let paused_at = verifier.verified().last_timestamp().expect("a frontier was committed");

    verifier.set_pause(Some(paused_at + 1));
    run(&mut verifier, 2).await;
    assert_eq!(verifier.verified().last_timestamp(), Some(paused_at));

    verifier.set_pause(None);
    run(&mut verifier, 2).await;
    assert_eq!(verifier.verified().last_timestamp(), Some(paused_at + 2));
}

#[tokio::test]
async fn a_pause_during_cold_start_does_not_hold_up_choosing_a_start() {
    let world = World::new();
    let mut verifier = world.verifier();

    // Cold start picks the starting timestamp from the chains' first safe heads; there is no
    // timestamp being attempted yet for a pause to be about. Holding it back here would leave a
    // paused verifier reporting `backfill_completed` false forever, and every acceptance test
    // gates on that flag before it does anything else.
    verifier.set_pause(Some(START));
    assert_eq!(verifier.step().await.unwrap(), Pace::Immediate);
    assert_eq!(verifier.state(), VerifierState::Running);
    assert_eq!(verifier.verification_start(), Some(START));

    // The pause then holds the first round, as it was asked to.
    assert_eq!(verifier.step().await.unwrap(), Pace::Idle);
    assert_eq!(verifier.verified().last_timestamp(), None);
}

#[tokio::test]
async fn cold_start_attempts_are_counted_per_try() {
    let world = World::new();
    // No safe head on one chain, so cold start cannot choose a start and retries.
    world.chain_b.set_first_safe_head(None);
    let mut verifier = world.verifier();

    assert_eq!(verifier.backfill_attempts(), 0);
    run(&mut verifier, 3).await;
    // Three tries, none of which finished: this is the signal an acceptance test waits on to know
    // the retry loop has engaged rather than the verifier never having been stepped.
    assert_eq!(verifier.backfill_attempts(), 3);
    assert_eq!(verifier.verification_start(), None);

    world.chain_b.set_first_safe_head(Some(START));
    verifier.step().await.unwrap();
    assert_eq!(verifier.backfill_attempts(), 4);
    assert_eq!(verifier.verification_start(), Some(START));

    // Cold start is over, so rounds do not count as attempts.
    run(&mut verifier, 2).await;
    assert_eq!(verifier.backfill_attempts(), 4);
}

#[tokio::test]
async fn a_resumed_verifier_reports_a_start_without_ever_attempting_cold_start() {
    let world = World::new();
    // A previous run over these log stores, so the resume path finds the history it checks for.
    {
        let mut previous = world.verifier();
        run(&mut previous, 2).await;
    }
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

    // Resuming reads the start out of the verified store, so the verifier is `Running` before its
    // first step and has made no attempt. Test control reports that as cold start having
    // completed, which is what op-supernode reports for a resume too.
    let resumed = world.verifier_with(verified);
    assert_eq!(resumed.state(), VerifierState::Running);
    assert_eq!(resumed.verification_start(), Some(START + 1));
    assert_eq!(resumed.backfill_attempts(), 0);
}

#[tokio::test]
async fn the_activation_timestamp_and_the_log_stores_are_readable() {
    let world = World::new();
    let mut verifier = world.verifier();
    run(&mut verifier, 2).await;

    assert_eq!(verifier.activation_timestamp(), ACTIVATION);
    // The store handed out is the one the loop seals into, not a copy, so what a reader reports
    // cannot lag what the loop has written.
    let logs = verifier.logs(CHAIN_A).expect("chain A is followed");
    assert_eq!(logs.latest_sealed_block().unwrap().number, START);
    // A chain id neither fake chain has: an unfollowed chain has no store, which is how the
    // test-control surface tells "nothing sealed yet" from "not a chain I verify".
    assert!(verifier.logs(CHAIN_A + CHAIN_B).is_none(), "an unfollowed chain has no store");
}

// ---------------------------------------------------------------------------
// Applying `Decision::Invalidate`: the deposits-only block replacement flow.
//
// The verifier's half of the replacement is exactly two side effects, in this order: archive the
// invalidated block's output preimage (the archive doubles as the deny list the chain's engine
// consults), then rewind the chain onto the block's parent. Derivation rebuilds the height, the
// rebuild hits the deny list, and a deposits-only block replaces the invalid one — mirroring
// op-supernode's `applyPendingTransition` for `DecisionInvalidate`
// (`op-supernode/supernode/activity/interop/interop.go`) and `InvalidateBlock`
// (`op-supernode/supernode/chain_container/invalidation.go`).
// ---------------------------------------------------------------------------

/// The block chain B carries at `START + 2`, which the tests below invalidate.
fn invalid_b_block() -> BlockNumHash {
    BlockNumHash { number: START + 2, hash: block_hash(CHAIN_B, START + 2) }
}

/// Drives a replacing verifier to the invalidation at `START + 2`: chain B executes a message
/// whose referenced initiating log does not exist on chain A.
fn world_with_an_invalid_message_at_start_plus_2() -> World {
    let world = World::new();
    let initiating = initiating_log(3);
    world.chain_b.set_block(FakeBlock::with_logs(
        START + 2,
        vec![executing_log(CHAIN_A, &initiating, START + 1, 0, START + 1)],
    ));
    world
}

#[tokio::test]
async fn an_invalid_message_archives_the_output_and_rewinds_the_chain() {
    let world = world_with_an_invalid_message_at_start_plus_2();
    let mut verifier = world.replacing_verifier();

    // Cold start, 110, 111 advance; the round at 112 invalidates and applies it.
    run(&mut verifier, 4).await;

    // The frontier does not move on an invalidation: the same timestamp is re-verified against
    // the replaced chain, so nothing is committed for 112.
    assert_eq!(verifier.verified().last_timestamp(), Some(START + 1));
    // The apply ran to completion: the WAL slot is cleared and the verifier keeps running.
    assert_eq!(verifier.verified().pending().unwrap(), None);
    assert_eq!(verifier.state(), VerifierState::Running);

    // Side effect one, and durably first: the invalidated output is archived with the decision
    // timestamp and the preimage fields the optimistic superroot branch serves after the block is
    // gone. This record is the deny list — without it derivation would rebuild the same block.
    let archived = world.archives[&CHAIN_B]
        .get(START + 2, invalid_b_block().hash)
        .unwrap()
        .expect("the invalidated output is archived");
    assert_eq!(archived.decision_timestamp, START + 2);
    assert_eq!(archived.output_root.state_root, B256::repeat_byte(0xaa));
    assert_eq!(archived.output_root.bridge_storage_root, B256::repeat_byte(0xbb));

    // Side effect two: the chain was rewound off the invalidated block — and only that chain.
    assert_eq!(world.rewinds[&CHAIN_B].calls(), vec![invalid_b_block()]);
    assert!(world.rewinds[&CHAIN_A].calls().is_empty(), "the valid chain is not rewound");
    assert!(world.archives[&CHAIN_A].at(START + 2).unwrap().is_empty());

    // Derivation's half, modelled: a deposits-only replacement lands at the same height and the
    // chain answers for the timestamp again. The round loop then advances through it.
    world.chain_b.set_block(FakeBlock::empty(START + 2));
    world.chain_b.set_local_safe_through(200);
    run(&mut verifier, 2).await;
    assert_eq!(verifier.verified().last_timestamp(), Some(START + 3));
}

/// The #22540 crash-replay hazard, pinned: a write-ahead-logged invalidation replayed onto a
/// chain that already replaced the block must still write the archive. The "already replaced,
/// just ack" branch skips the *rewind*, never the archive record — a replay that skipped the
/// write and then cleared the slot would lose the roots permanently, and a missing archive entry
/// is silent: the optimistic superroot branch would serve the replacement block's output as a
/// well-formed but wrong answer.
#[tokio::test]
async fn a_replayed_invalidation_still_archives_when_the_rewind_is_skipped() {
    let world = World::new();
    let verified = VerifiedStore::new(MemoryKv::new()).unwrap();

    // Fill the log stores the way the crashed run would have.
    {
        let mut warm = world.verifier_with(VerifiedStore::new(MemoryKv::new()).unwrap());
        warm.step().await.unwrap();
    }

    // The slot a crash left behind: an invalidation of chain B's block at START, decided but not
    // yet applied — and, by the time this process looks, already replaced on the chain.
    let invalid = InvalidHead {
        block: BlockNumHash { number: START, hash: block_hash(CHAIN_B, START) },
        state_root: B256::repeat_byte(0xaa),
        message_passer_storage_root: B256::repeat_byte(0xbb),
    };
    verified
        .set_pending(&PendingTransition::Invalidate(RoundResult {
            verified: VerifiedResult {
                timestamp: START,
                l1_inclusion: l1_at(START),
                l2_heads: BTreeMap::from([
                    (CHAIN_A, BlockNumHash { number: START, hash: block_hash(CHAIN_A, START) }),
                    (CHAIN_B, invalid.block),
                ]),
            },
            invalid_heads: BTreeMap::from([(CHAIN_B, invalid)]),
        }))
        .unwrap();
    world.rewinds[&CHAIN_B].set_already_replaced(true);

    let mut verifier = world.replacing_verifier_with(verified);
    verifier.step().await.unwrap();

    // The archive write happened even though the rewind had nothing to do.
    let archived = world.archives[&CHAIN_B]
        .get(START, invalid.block.hash)
        .unwrap()
        .expect("the replayed invalidation still archives the output");
    assert_eq!(archived.decision_timestamp, START);
    // The rewind was asked and answered "already off the block".
    assert_eq!(world.rewinds[&CHAIN_B].calls(), vec![invalid.block]);
    // The slot is cleared, and the invalidation committed no frontier.
    assert_eq!(verifier.verified().pending().unwrap(), None);
    assert_eq!(verifier.verified().last_timestamp(), None);
}

/// A rewind that fails transiently preserves the write-ahead slot, so the apply is retried whole
/// — op-supernode's "invalidation failed, transition preserved for retry on restart". The archive
/// write from the failed attempt stands; re-recording it on the retry is a no-op.
#[tokio::test]
async fn a_failed_rewind_keeps_the_invalidation_pending_and_retries_it() {
    let world = world_with_an_invalid_message_at_start_plus_2();
    world.rewinds[&CHAIN_B].set_unreachable(true);
    let mut verifier = world.replacing_verifier();

    let paces = run(&mut verifier, 4).await;
    assert_eq!(paces[3], Pace::Retry, "a failed apply is a transient round failure");
    assert!(
        matches!(verifier.verified().pending().unwrap(), Some(PendingTransition::Invalidate(_))),
        "the transition is preserved for retry"
    );
    // The archive write preceded the rewind and stands.
    assert!(world.archives[&CHAIN_B].get(START + 2, invalid_b_block().hash).unwrap().is_some());

    // The chain heals; the pending transition is applied before anything else is observed.
    world.rewinds[&CHAIN_B].set_unreachable(false);
    assert_eq!(verifier.step().await.unwrap(), Pace::Idle);
    assert_eq!(verifier.verified().pending().unwrap(), None);
    assert_eq!(world.rewinds[&CHAIN_B].calls(), vec![invalid_b_block(); 2]);
}

/// Genesis has no parent to rewind onto, so an invalidation naming height zero is a permanent
/// halt with the cause named — not a rewind attempt (op-supernode's `InvalidateBlock` refuses
/// height 0 the same way, `invalidation.go:398-400`).
#[tokio::test]
async fn an_invalidation_of_genesis_halts_rather_than_rewinding() {
    let world = World::new();
    let verified = VerifiedStore::new(MemoryKv::new()).unwrap();
    {
        let mut warm = world.verifier_with(VerifiedStore::new(MemoryKv::new()).unwrap());
        warm.step().await.unwrap();
    }
    let invalid = InvalidHead {
        block: BlockNumHash { number: 0, hash: block_hash(CHAIN_B, 0) },
        state_root: B256::repeat_byte(0xaa),
        message_passer_storage_root: B256::repeat_byte(0xbb),
    };
    verified
        .set_pending(&PendingTransition::Invalidate(RoundResult {
            verified: VerifiedResult {
                timestamp: START,
                l1_inclusion: l1_at(START),
                l2_heads: BTreeMap::from([(CHAIN_B, invalid.block)]),
            },
            invalid_heads: BTreeMap::from([(CHAIN_B, invalid)]),
        }))
        .unwrap();

    let mut verifier = world.replacing_verifier_with(verified);
    let halted = verifier.step().await.expect_err("genesis cannot be invalidated");
    assert!(halted.reason.contains("genesis"), "{}", halted.reason);
    assert!(world.rewinds[&CHAIN_B].calls().is_empty(), "no rewind is attempted");
}

/// A verifier whose chain set is wider than its routes is refused at construction: an
/// invalidation names its chains only at decision time, and a chain that could be verified but
/// not rewound would turn an applied decision into a permanent error mid-apply.
#[tokio::test]
async fn a_chain_without_an_invalidation_route_is_refused_at_construction() {
    let world = World::new();
    let mut routes = world.replacements();
    routes.remove(&CHAIN_B);
    let err = world
        .verifier()
        .with_replacements(routes)
        .expect_err("a partial route set must be refused")
        .to_string();
    assert!(err.contains("no invalidation route"), "{err}");
}
