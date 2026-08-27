use crate::{
    EngineClientError, EngineClientResult, SequencerActor, SequencerAdminQuery,
    SequencerEngineClient,
    actors::{
        MockConductor, MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient,
    },
};
use alloy_consensus::Block;
use alloy_eips::{BlockNumHash, eip1898::NumHash};
use alloy_primitives::{B256, U256};
use alloy_rpc_types_engine::{ExecutionPayloadV1, PayloadAttributes, PayloadId};
use async_trait::async_trait;
use kona_derive::{AttributesBuilder, PipelineErrorKind, PipelineResult};
use kona_engine::{SealTaskError, SealedPayload};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use mockall::TimesRange;
use op_alloy_consensus::OpTxEnvelope;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadAttributes};
use std::{
    collections::{HashMap, VecDeque},
    sync::{
        Arc, Mutex,
        atomic::{AtomicU64, Ordering},
    },
    time::{Duration, SystemTime, UNIX_EPOCH},
};
use tokio::{sync::mpsc, time::Instant};

/// The L2 block time the fixtures configure.
pub(crate) const BLOCK_TIME: u64 = 2;

/// The largest block group the fixtures allow.
pub(crate) const MAX_MULTI_BLOCKS: u64 = 4;

/// The current wall-clock time in L2 timestamp terms.
pub(crate) fn now_unix() -> u64 {
    SystemTime::now().duration_since(UNIX_EPOCH).expect("clock is past the unix epoch").as_secs()
}

/// A [`RollupConfig`] for a chain that builds exactly one block per block time.
pub(crate) fn single_block_config() -> RollupConfig {
    RollupConfig { block_time: BLOCK_TIME, max_sequencer_drift: 600, ..Default::default() }
}

/// A [`RollupConfig`] whose chain may build block groups, activated at genesis so that every
/// timestamp the fixtures use allows siblings and none of them is a fork activation.
pub(crate) fn multi_block_config() -> RollupConfig {
    let mut config = RollupConfig {
        multi_block_time: Some(0),
        max_multi_blocks: Some(MAX_MULTI_BLOCKS),
        ..single_block_config()
    };
    config.hardforks.karst_time = Some(0);
    assert_eq!(config.check_multi_block_config(), Ok(()));
    config
}

/// One `prepare_payload_attributes` request.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) struct AttributesRequest {
    /// The hash of the block the attributes build on.
    pub parent: B256,
    /// The L1 origin the attributes were asked to reference.
    pub epoch: BlockNumHash,
    /// The timestamp the attributes were asked to carry.
    pub timestamp: u64,
}

/// An [`AttributesBuilder`] that records every request and answers with attributes carrying the
/// requested timestamp.
#[derive(Debug, Default)]
pub(crate) struct RecordingAttributesBuilder {
    /// Every request the builder served, in call order.
    pub requests: Arc<Mutex<Vec<AttributesRequest>>>,
    /// Outcomes to serve in place of attributes, consumed in order. [`None`] and an exhausted
    /// queue both build attributes.
    pub outcomes: VecDeque<Option<PipelineErrorKind>>,
}

#[async_trait]
impl AttributesBuilder for RecordingAttributesBuilder {
    async fn prepare_payload_attributes(
        &mut self,
        l2_parent: L2BlockInfo,
        epoch: BlockNumHash,
        timestamp: u64,
    ) -> PipelineResult<OpPayloadAttributes> {
        self.requests.lock().unwrap().push(AttributesRequest {
            parent: l2_parent.block_info.hash,
            epoch,
            timestamp,
        });

        if let Some(Some(err)) = self.outcomes.pop_front() {
            return Err(err);
        }

        Ok(OpPayloadAttributes {
            payload_attributes: PayloadAttributes { timestamp, ..Default::default() },
            ..Default::default()
        })
    }
}

/// The chain and call log the [`FakeSequencerEngineClient`] keeps, shared with the test.
#[derive(Debug)]
pub(crate) struct EngineFixture {
    /// The current unsafe head.
    pub head: L2BlockInfo,
    /// Every block of the chain, by hash.
    pub blocks: HashMap<B256, BlockInfo>,
    /// The attributes of every block the sequencer sealed, in order.
    pub sealed: Vec<OpAttributesWithParent>,
    /// The hash of every block the sequencer sealed, in order.
    pub sealed_hashes: Vec<B256>,
    /// The readiness deadline of every seal request, in order.
    pub seal_deadlines: Vec<Option<Instant>>,
    /// The seal duration the engine reports back, which paces the sequencer's next slot.
    pub seal_duration: Duration,
    /// Failures to serve in place of a seal, consumed in order.
    pub seal_outcomes: VecDeque<Option<SealTaskError>>,
    /// Whether the execution layer reports payloads worth sealing as soon as they are built.
    /// When it does not, a seal carrying a readiness deadline waits for it, as a real one does.
    pub seals_immediately: bool,
    /// How many times the forkchoice was reset.
    pub resets: usize,
}

impl EngineFixture {
    /// Extends the chain with the block described by `attributes`, makes it the unsafe head and
    /// returns it.
    fn apply(&mut self, attributes: &OpAttributesWithParent) -> L2BlockInfo {
        let parent = attributes.parent;
        let number = parent.block_info.number + 1;
        let block = BlockInfo {
            hash: block_hash(number, attributes.attributes.payload_attributes.timestamp),
            parent_hash: parent.block_info.hash,
            number,
            timestamp: attributes.attributes.payload_attributes.timestamp,
        };
        self.blocks.insert(block.hash, block);
        self.head = L2BlockInfo {
            block_info: block,
            l1_origin: parent.l1_origin,
            seq_num: parent.seq_num + 1,
        };
        self.head
    }

    /// The timestamps of the sealed blocks, in order.
    pub(crate) fn sealed_timestamps(&self) -> Vec<u64> {
        self.sealed.iter().map(|a| a.attributes.payload_attributes.timestamp).collect()
    }
}

/// A [`SequencerEngineClient`] backed by an in-memory chain, so the sequencer sees the blocks it
/// seals when it re-reads the unsafe head and walks back over parent hashes.
#[derive(Clone, Debug)]
pub(crate) struct FakeSequencerEngineClient {
    /// The chain and call log, shared with the test.
    pub fixture: Arc<Mutex<EngineFixture>>,
}

#[async_trait]
impl SequencerEngineClient for FakeSequencerEngineClient {
    async fn reset_engine_forkchoice(&self) -> EngineClientResult<()> {
        self.fixture.lock().unwrap().resets += 1;
        Ok(())
    }

    async fn start_build_block(
        &self,
        _attributes: OpAttributesWithParent,
    ) -> EngineClientResult<PayloadId> {
        Ok(PayloadId::new([1u8; 8]))
    }

    async fn seal_and_canonicalize_block(
        &self,
        _payload_id: PayloadId,
        attributes: OpAttributesWithParent,
        ready_deadline: Option<Instant>,
    ) -> EngineClientResult<SealedPayload> {
        let wait = {
            let fixture = self.fixture.lock().unwrap();
            (!fixture.seals_immediately).then_some(ready_deadline).flatten()
        };
        if let Some(deadline) = wait {
            tokio::time::sleep_until(deadline).await;
        }

        let mut fixture = self.fixture.lock().unwrap();
        fixture.seal_deadlines.push(ready_deadline);
        if let Some(err) = fixture.seal_outcomes.pop_front().flatten() {
            return Err(EngineClientError::SealError(err));
        }
        let block = fixture.apply(&attributes);
        fixture.sealed_hashes.push(block.block_info.hash);
        fixture.sealed.push(attributes);
        Ok(SealedPayload {
            payload: payload_envelope(),
            block,
            seal_duration: fixture.seal_duration,
        })
    }

    async fn get_unsafe_head(&self) -> EngineClientResult<L2BlockInfo> {
        Ok(self.fixture.lock().unwrap().head)
    }

    async fn l2_block_info_by_hash(&self, hash: B256) -> EngineClientResult<Option<BlockInfo>> {
        Ok(self.fixture.lock().unwrap().blocks.get(&hash).copied())
    }
}

/// A [`SequencerActor`] wired to the fakes, together with the handles a test drives it by.
pub(crate) struct SequencerFixture {
    /// The actor under test.
    pub actor: SequencerActor<
        RecordingAttributesBuilder,
        MockConductor,
        MockOriginSelector,
        FakeSequencerEngineClient,
        MockUnsafePayloadGossipClient,
    >,
    /// The chain and call log the actor's engine client keeps.
    pub engine: Arc<Mutex<EngineFixture>>,
    /// Every attributes request the actor made.
    pub attributes: Arc<Mutex<Vec<AttributesRequest>>>,
    /// The admin API sender, kept alive so the actor's admin channel stays open.
    pub admin_tx: mpsc::Sender<SequencerAdminQuery>,
    /// The first L1 origin the origin selector hands out, the one the unsafe head references.
    pub l1_origin: BlockInfo,
}

impl SequencerFixture {
    /// The timestamps of the blocks the actor sealed, in order.
    pub(crate) fn sealed_timestamps(&self) -> Vec<u64> {
        self.engine.lock().unwrap().sealed_timestamps()
    }
}

/// Builds a sequencer whose chain starts at an unsafe head with the given timestamp, on a chain
/// that allows block groups.
pub(crate) fn sequencer_fixture(
    head_timestamp: u64,
    origin_calls: impl Into<TimesRange>,
) -> SequencerFixture {
    sequencer_fixture_with_config(head_timestamp, origin_calls, multi_block_config())
}

/// Builds a sequencer whose chain starts at an unsafe head with the given timestamp.
///
/// The origin selector is called `origin_calls` times. Its first answer is the origin the head
/// already references; every later answer is the next L1 block, so a slot that reselects mid-group
/// is visible in the epochs the attributes carry.
pub(crate) fn sequencer_fixture_with_config(
    head_timestamp: u64,
    origin_calls: impl Into<TimesRange>,
    rollup_config: RollupConfig,
) -> SequencerFixture {
    let l1_origin = BlockInfo {
        hash: l1_origin_hash(100),
        parent_hash: l1_origin_hash(99),
        number: 100,
        timestamp: head_timestamp,
    };

    // The head's parent is part of the chain, so the group-length walk can leave the group.
    let parent = BlockInfo {
        hash: block_hash(9, head_timestamp - BLOCK_TIME),
        parent_hash: B256::with_last_byte(0x08),
        number: 9,
        timestamp: head_timestamp - BLOCK_TIME,
    };
    let head_block = BlockInfo {
        hash: block_hash(10, head_timestamp),
        parent_hash: parent.hash,
        number: 10,
        timestamp: head_timestamp,
    };
    let head = L2BlockInfo {
        block_info: head_block,
        l1_origin: NumHash { number: l1_origin.number, hash: l1_origin.hash },
        seq_num: 0,
    };

    let engine = Arc::new(Mutex::new(EngineFixture {
        head,
        blocks: HashMap::from([(parent.hash, parent), (head_block.hash, head_block)]),
        sealed: Vec::new(),
        sealed_hashes: Vec::new(),
        seal_deadlines: Vec::new(),
        seal_duration: Duration::ZERO,
        seal_outcomes: VecDeque::new(),
        seals_immediately: true,
        resets: 0,
    }));

    let origin_selections = AtomicU64::new(0);
    let mut origin_selector = MockOriginSelector::new();
    origin_selector.expect_next_l1_origin().times(origin_calls).returning(move |_, _, _| {
        // Every answer after the first is the L1 block that follows the head's origin, so it is
        // still a consistent origin to build on but a distinguishable epoch.
        let number =
            l1_origin.number + u64::from(origin_selections.fetch_add(1, Ordering::Relaxed) > 0);
        Ok(BlockInfo {
            hash: l1_origin_hash(number),
            parent_hash: l1_origin_hash(number - 1),
            number,
            timestamp: l1_origin.timestamp,
        })
    });

    let mut gossip_client = MockUnsafePayloadGossipClient::new();
    gossip_client.expect_schedule_execution_payload_gossip().returning(|_| Ok(()));

    let attributes = Arc::new(Mutex::new(Vec::new()));
    let attributes_builder =
        RecordingAttributesBuilder { requests: Arc::clone(&attributes), outcomes: VecDeque::new() };

    let (admin_tx, admin_rx) = mpsc::channel(20);
    let actor = SequencerActor::new(
        admin_rx,
        attributes_builder,
        None,
        FakeSequencerEngineClient { fixture: Arc::clone(&engine) },
        true,
        false,
        origin_selector,
        Arc::new(rollup_config),
        gossip_client,
    );

    SequencerFixture { actor, engine, attributes, admin_tx, l1_origin }
}

/// Returns a sequencer actor for a one-block-per-block-time chain whose collaborators are all
/// mocks, for tests that drive its methods directly instead of running a slot.
pub(crate) fn test_actor() -> SequencerActor<
    RecordingAttributesBuilder,
    MockConductor,
    MockOriginSelector,
    MockSequencerEngineClient,
    MockUnsafePayloadGossipClient,
> {
    // The sender is intentionally dropped, so the channel starts closed.
    // If future tests need to send messages, keep the sender instead of dropping it.
    let (_admin_api_tx, admin_api_rx) = mpsc::channel(20);
    SequencerActor::new(
        admin_api_rx,
        RecordingAttributesBuilder::default(),
        None,
        MockSequencerEngineClient::new(),
        true,
        false,
        MockOriginSelector::new(),
        Arc::new(single_block_config()),
        MockUnsafePayloadGossipClient::new(),
    )
}

/// The hash of the L1 block with the given number.
fn l1_origin_hash(number: u64) -> B256 {
    B256::from(U256::from(number) << 128)
}

/// A block hash that is unique per `(number, timestamp)`, so siblings do not collide.
fn block_hash(number: u64, timestamp: u64) -> B256 {
    B256::from(U256::from(number) << 64 | U256::from(timestamp))
}

/// The envelope a successful seal hands back. Its contents are irrelevant: the actor only
/// forwards it to the gossip client.
fn payload_envelope() -> OpExecutionPayloadEnvelope {
    OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
        &Block::<OpTxEnvelope>::default(),
    ))
}
