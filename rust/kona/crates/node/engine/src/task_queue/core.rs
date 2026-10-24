//! The [`Engine`] is a task queue that receives and executes [`EngineTask`]s.

use super::EngineTaskExt;
use crate::{
    DerivedSafeHead, EngineClient, EngineState, EngineSyncStateUpdate, EngineTask, EngineTaskError,
    EngineTaskErrorSeverity, Metrics, SyncStartError, SynchronizeTask, SynchronizeTaskError,
    find_reorg_safe_head, find_starting_forkchoice, task_queue::EngineTaskErrors,
};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::{cmp::Ordering, collections::BinaryHeap, sync::Arc};
use thiserror::Error;
use tokio::sync::watch::Sender;

/// A stable ordering within each task priority. Retried work can accumulate many same-kind
/// requests, which must keep their original order (especially unsafe payload imports).
#[derive(Debug)]
struct QueuedTask {
    sequence: u64,
    task: EngineTask,
}
impl PartialEq for QueuedTask {
    fn eq(&self, other: &Self) -> bool {
        self.cmp(other) == Ordering::Equal
    }
}
impl Eq for QueuedTask {}
impl PartialOrd for QueuedTask {
    fn partial_cmp(&self, other: &Self) -> Option<Ordering> {
        Some(self.cmp(other))
    }
}
impl Ord for QueuedTask {
    fn cmp(&self, other: &Self) -> Ordering {
        self.task.cmp(&other.task).then_with(|| other.sequence.cmp(&self.sequence))
    }
}

/// The [`Engine`] task queue.
///
/// Tasks of a shared [`EngineTask`] variant are processed in FIFO order, providing synchronization
/// guarantees for the L2 execution layer and other actors. A priority queue, ordered by
/// [`EngineTask`]'s [`Ord`] implementation, is used to prioritize tasks executed by the
/// [`Engine::drain`] method.
///
///  Because tasks are executed one at a time, they are considered to be atomic operations over the
/// [`EngineState`], and are given exclusive access to the engine state during execution.
///
/// Tasks within the queue are also considered fallible. If they fail with a temporary error,
/// they are not popped from the queue, the error is returned, and they are retried on the
/// next call to [`Engine::drain`]. A task that fails with a flush error has already replaced its
/// block, so it is popped before the error is returned.
#[derive(Debug)]
pub struct Engine {
    /// The state of the engine.
    state: EngineState,
    /// A sender that can be used to notify the engine actor of state changes.
    state_sender: Sender<EngineState>,
    /// The task queue.
    tasks: BinaryHeap<QueuedTask>,
    next_sequence: u64,
    /// Preserve the task being retried ahead of newly enqueued work: it may have already
    /// performed part of an Engine API operation.
    active: Option<EngineTask>,
}

impl Engine {
    /// Creates a new [`Engine`] with an empty task queue and the passed initial [`EngineState`].
    pub fn new(initial_state: EngineState, state_sender: Sender<EngineState>) -> Self {
        Self {
            state: initial_state,
            state_sender,
            tasks: BinaryHeap::default(),
            active: None,
            next_sequence: 0,
        }
    }

    /// Returns a reference to the inner [`EngineState`].
    pub const fn state(&self) -> &EngineState {
        &self.state
    }

    /// Returns a receiver that can be used to listen to engine state updates.
    pub fn state_subscribe(&self) -> tokio::sync::watch::Receiver<EngineState> {
        self.state_sender.subscribe()
    }

    /// Enqueues a new [`EngineTask`] for execution.
    pub fn enqueue(&mut self, task: EngineTask) {
        let sequence = self.next_sequence;
        self.next_sequence =
            self.next_sequence.checked_add(1).expect("engine task sequence exhausted");
        self.tasks.push(QueuedTask { sequence, task });
    }

    /// Number of queued and active tasks, including work awaiting a retry.
    pub fn len(&self) -> usize {
        self.tasks.len() + usize::from(self.active.is_some())
    }

    /// Whether there is no pending work.
    pub fn is_empty(&self) -> bool {
        self.len() == 0
    }

    /// Resets the engine by finding a plausible sync starting point via
    /// [`find_starting_forkchoice`]. The state will be updated to the starting point, and a
    /// forkchoice update will be enqueued in order to reorg the execution layer.
    pub async fn reset(
        &mut self,
        client: Arc<EngineClient>,
        config: Arc<RollupConfig>,
    ) -> Result<L2BlockInfo, EngineResetError> {
        let start = find_starting_forkchoice(&config, client.as_ref()).await?;

        // One attempt. The actor retains the reset request and retries transient failures.
        let synchronize = SynchronizeTask::new(
            client,
            config,
            EngineSyncStateUpdate {
                unsafe_head: Some(start.un_safe),
                local_safe_head: Some(start.safe),
                safe_head: Some(start.safe),
                finalized_head: Some(start.finalized),
            },
        );
        synchronize.execute(&mut self.state).await?;

        metrics::counter!(Metrics::ENGINE_RESET_COUNT).increment(1);

        Ok(start.safe)
    }

    /// Resets the engine after an L1 reorg, keeping the newest of `candidates` (ordered newest
    /// first) that [`find_reorg_safe_head`] shows was derived from L1 data the reorg left
    /// canonical. The unsafe head is kept, and the safe heads are rewound to the kept one if they
    /// are above it. Falls back to [`Engine::reset`] when no candidate can be kept.
    pub async fn reset_after_l1_reorg(
        &mut self,
        client: Arc<EngineClient>,
        config: Arc<RollupConfig>,
        candidates: &[DerivedSafeHead],
    ) -> Result<L2BlockInfo, EngineResetError> {
        let Some(safe_head) =
            find_reorg_safe_head(client.as_ref(), &self.state, candidates).await?
        else {
            return self.reset(client, config).await;
        };

        let sync_state = self.state.sync_state;
        if safe_head != sync_state.local_safe_head() {
            let above = |head: L2BlockInfo| head.block_info.number > safe_head.block_info.number;
            let synchronize = SynchronizeTask::new(
                client,
                config,
                EngineSyncStateUpdate {
                    unsafe_head: None,
                    local_safe_head: Some(safe_head),
                    safe_head: above(sync_state.safe_head()).then_some(safe_head),
                    finalized_head: None,
                },
            );
            synchronize.execute(&mut self.state).await?;
        }

        metrics::counter!(Metrics::ENGINE_REORG_RESET_COUNT).increment(1);

        Ok(safe_head)
    }

    /// Clears the task queue.
    pub fn clear(&mut self) {
        self.tasks.clear();
        self.active = None;
    }

    /// Attempts to drain the queue by executing all [`EngineTask`]s in-order. If any task returns
    /// an error along the way, the error is returned and the task is not popped from the queue (in
    /// case it must be retried), unless it is a flush error.
    pub async fn drain(&mut self) -> Result<(), EngineTaskErrors> {
        // Drain tasks in order of priority, halting on errors for a retry to be attempted.
        loop {
            if self.active.is_none() {
                self.active = self.tasks.pop().map(|queued| queued.task);
            }
            let Some(task) = &self.active else { break };
            let result = task.execute(&mut self.state).await;

            // A flush reports that the task replaced its block with a deposits-only block, so the
            // task is done; retrying it would replace that block again.
            if let Err(err) = &result &&
                err.severity() != EngineTaskErrorSeverity::Flush
            {
                return result;
            }

            self.state_sender.send_replace(self.state);
            self.active = None;
            result?;
        }

        Ok(())
    }
}

/// An error occurred while attempting to reset the [`Engine`].
#[derive(Debug, Error)]
pub enum EngineResetError {
    /// An error that occurred while updating the forkchoice state.
    #[error(transparent)]
    Forkchoice(#[from] SynchronizeTaskError),
    /// An error occurred while traversing the L1 for the sync starting point.
    #[error(transparent)]
    SyncStart(#[from] SyncStartError),
}

impl EngineResetError {
    /// Whether retrying the reset can recover without changing the node configuration.
    pub fn is_temporary(&self) -> bool {
        match self {
            Self::SyncStart(SyncStartError::RpcError(_)) => true,
            Self::SyncStart(_) => false,
            Self::Forkchoice(err) => err.severity() != EngineTaskErrorSeverity::Critical,
        }
    }
}

#[cfg(test)]
mod recovery_tests {
    use super::*;
    use crate::{
        ConsolidateInput, ConsolidateTask, InsertTask, NoopBlockSink,
        test_utils::{
            TestAttributesBuilder, TestEngineStateBuilder, test_block_info, test_engine_client,
        },
    };
    use alloy_rpc_types_engine::{
        BlobsBundleV1, ExecutionPayloadV1, ExecutionPayloadV3, ForkchoiceUpdated, PayloadId,
        PayloadStatus, PayloadStatusEnum,
    };
    use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpExecutionPayloadEnvelopeV3};

    #[tokio::test]
    async fn temporary_failure_returns_with_task_retained_for_recovery() {
        let payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
            &alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default(),
        ));
        let block: op_alloy_consensus::OpBlock = payload.clone().try_into_block().unwrap();
        let mut config = RollupConfig::default();
        config.genesis.l2.hash = block.header.hash_slow();
        let config = Arc::new(config);
        let (client, l1, l2) = test_engine_client(config.clone());
        let client = Arc::new(client);
        l2.expect_error("engine_newPayloadV1");
        let (state_tx, _state_rx) = tokio::sync::watch::channel(EngineState::default());
        let mut engine = Engine::new(EngineState::default(), state_tx);
        engine.enqueue(EngineTask::Insert(Box::new(InsertTask::new(
            client.clone(),
            config,
            payload,
            false,
            Arc::new(NoopBlockSink),
        ))));
        // The old inline retry loop never returned from this call.
        let error = tokio::time::timeout(std::time::Duration::from_secs(1), engine.drain())
            .await
            .expect("temporary errors must yield to the actor")
            .unwrap_err();
        assert_eq!(error.severity(), EngineTaskErrorSeverity::Temporary);
        assert_eq!(engine.len(), 1);
        // Accumulate additional unsafe imports during the outage. Equal-priority heap
        // entries must remain FIFO; otherwise the final unsafe head can move backwards.
        for number in 1..=3 {
            let mut block = alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default();
            block.header.number = number;
            let payload =
                OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(&block));
            let imported: op_alloy_consensus::OpBlock = payload.clone().try_into_block().unwrap();
            let mut config = RollupConfig::default();
            config.genesis.l2.number = number;
            config.genesis.l2.hash = imported.header.hash_slow();
            engine.enqueue(EngineTask::Insert(Box::new(InsertTask::new(
                client.clone(),
                Arc::new(config),
                payload,
                false,
                Arc::new(NoopBlockSink),
            ))));
        }
        for _ in 0..4 {
            l2.expect("engine_newPayloadV1", PayloadStatus::from_status(PayloadStatusEnum::Valid));
            l2.expect(
                "engine_forkchoiceUpdatedV3",
                ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
            );
        }
        engine.drain().await.unwrap();
        assert!(engine.is_empty());
        assert_eq!(engine.state().sync_state.unsafe_head().block_info.number, 3);
        l1.assert_finished();
        l2.assert_finished();
    }

    // A flush means the task already replaced its block with a deposits-only block. Keeping it
    // would rebuild that replacement on every later drain.
    #[tokio::test]
    async fn flushed_task_is_dropped_not_retried() {
        let parent = test_block_info(10);
        let deposit = alloy_primitives::Bytes::from(vec![op_alloy_consensus::DEPOSIT_TX_TYPE_ID]);
        let attributes = TestAttributesBuilder::new()
            .with_parent(parent)
            .with_transactions(vec![deposit, alloy_primitives::Bytes::from(vec![0x02])])
            .build();

        // The deposits-only block the engine builds. It is genesis, so its block info needs no
        // L1-info deposit.
        let mut block = alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default();
        block.header.number = parent.block_info.number + 1;
        let replacement = OpExecutionPayloadEnvelopeV3 {
            execution_payload: ExecutionPayloadV3::from_block_slow(&block),
            block_value: Default::default(),
            blobs_bundle: BlobsBundleV1::default(),
            should_override_builder: false,
            parent_beacon_block_root: Default::default(),
        };
        let imported: op_alloy_consensus::OpBlock = OpExecutionPayloadEnvelope::V3 {
            payload: replacement.execution_payload.clone(),
            parent_beacon_block_root: replacement.parent_beacon_block_root,
        }
        .try_into_block()
        .unwrap();
        let mut config = RollupConfig::default();
        config.hardforks.ecotone_time = Some(0);
        config.hardforks.holocene_time = Some(0);
        config.genesis.l2.number = imported.header.number;
        config.genesis.l2.hash = imported.header.hash_slow();
        let config = Arc::new(config);

        let valid = PayloadStatus::from_status(PayloadStatusEnum::Valid);
        let (client, l1, l2) = test_engine_client(config.clone());
        let client = Arc::new(client);
        l2.expect_error_code("engine_forkchoiceUpdatedV3", -38003);
        let forkchoice = TestEngineStateBuilder::new()
            .with_unsafe_head(parent)
            .build()
            .sync_state
            .create_forkchoice_state();
        l2.expect_params(
            "engine_forkchoiceUpdatedV3",
            serde_json::json!([forkchoice, attributes.as_deposits_only().attributes]),
            ForkchoiceUpdated {
                payload_status: valid.clone(),
                payload_id: Some(PayloadId::new([1; 8])),
            },
        );
        l2.expect("engine_getPayloadV3", replacement);
        l2.expect("engine_newPayloadV3", valid.clone());
        l2.expect("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid));
        // The safe head is at the unsafe head, so the task builds the attributes instead of
        // consolidating them against an unsafe block.
        let state = TestEngineStateBuilder::new().with_unsafe_head(parent).build();
        let (state_tx, state_rx) = tokio::sync::watch::channel(EngineState::default());
        let mut engine = Engine::new(state, state_tx);
        engine.enqueue(EngineTask::Consolidate(Box::new(ConsolidateTask::new(
            client.clone(),
            config,
            ConsolidateInput::Attributes(Box::new(attributes)),
            Arc::new(NoopBlockSink),
        ))));

        let error = engine.drain().await.unwrap_err();
        assert_eq!(error.severity(), EngineTaskErrorSeverity::Flush, "got {error:?}");
        l1.assert_finished();
        l2.assert_finished();
        assert!(engine.is_empty());
        // The task updates `engine.state()` itself; only drain publishes it.
        assert_eq!(
            state_rx.borrow().sync_state.safe_head().block_info.hash,
            imported.header.hash_slow()
        );
    }
}

#[cfg(test)]
mod l1_reorg_reset_tests {
    use super::*;
    use crate::test_utils::{RpcMock, test_engine_client};
    use alloy_eips::BlockNumHash;
    use alloy_primitives::B256;
    use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum};
    use kona_protocol::BlockInfo;
    use rstest::rstest;
    use serde_json::json;

    /// L1 block `number` on `fork`: fork 0 is the chain derivation read, fork 1 replaced it from
    /// the reorg height on.
    fn l1_id(number: u64, fork: u8) -> BlockNumHash {
        BlockNumHash { number, hash: B256::left_padding_from(&[fork, number as u8]) }
    }

    fn l2_block(number: u64, l1_origin: u64) -> L2BlockInfo {
        let hash = |n: u64| B256::left_padding_from(&[0xee, n as u8]);
        L2BlockInfo {
            block_info: BlockInfo {
                number,
                hash: hash(number),
                parent_hash: hash(number - 1),
                timestamp: number,
            },
            l1_origin: l1_id(l1_origin, 0),
            seq_num: 0,
        }
    }

    /// Scripts the L1 lookup of block `number`, answered from the chain reorged at
    /// `reorg_height`.
    fn expect_l1(l1: &RpcMock, number: u64, reorg_height: u64) {
        let mut block = alloy_rpc_types_eth::Block::<alloy_rpc_types_eth::Transaction>::default();
        block.header.hash = l1_id(number, u8::from(number >= reorg_height)).hash;
        block.header.inner.number = number;
        l1.expect_params("eth_getBlockByNumber", json!([format!("{number:#x}"), false]), block);
    }

    /// Scripts the engine's lookup of L2 block `number`, answered with `hash`.
    fn expect_l2(l2: &RpcMock, number: u64, hash: B256) {
        let mut block: <op_alloy_network::Optimism as alloy_network::Network>::BlockResponse =
            Default::default();
        block.header.hash = hash;
        block.header.inner.number = number;
        l2.expect_params("eth_getBlockByNumber", json!([format!("{number:#x}"), false]), block);
    }

    const FINALIZED: u64 = 5;
    const UNSAFE: u64 = 40;
    const UNSAFE_ORIGIN: u64 = 102;

    /// Derivation's recent safe heads, newest first: L2 block 20 depends on L1 data up to block
    /// 106, and L2 block 18 on L1 data up to block 103.
    fn recent_safe_heads() -> [DerivedSafeHead; 2] {
        [
            DerivedSafeHead { safe_head: l2_block(20, 100), derived_from: l1_id(106, 0) },
            DerivedSafeHead { safe_head: l2_block(18, 99), derived_from: l1_id(103, 0) },
        ]
    }

    /// An engine at the local safe head L2 block 20 and the unsafe head L2 block 40, whose L1
    /// origin is `unsafe_origin`.
    fn setup(
        unsafe_origin: u64,
    ) -> (Engine, Arc<EngineClient>, RpcMock, RpcMock, Arc<RollupConfig>) {
        let mut config = RollupConfig::default();
        config.hardforks.ecotone_time = Some(0);
        let config = Arc::new(config);
        let (client, l1, l2) = test_engine_client(config.clone());

        let mut state = EngineState::default();
        state.sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
            unsafe_head: Some(l2_block(UNSAFE, unsafe_origin)),
            local_safe_head: Some(l2_block(20, 100)),
            safe_head: Some(l2_block(20, 100)),
            finalized_head: Some(l2_block(FINALIZED, 90)),
        });
        let (state_tx, _) = tokio::sync::watch::channel(state);
        (Engine::new(state, state_tx), Arc::new(client), l1, l2, config)
    }

    /// Each case scripts exactly the lookups expected, in order: the unsafe head's L1 origin,
    /// then each recent safe head's L1 block until one is canonical, then that safe head on the
    /// engine's chain.
    #[rstest]
    #[case::tip_reorg_keeps_the_safe_head(108, UNSAFE_ORIGIN, &[UNSAFE_ORIGIN, 106], Some(20))]
    #[case::reorg_of_the_safe_heads_l1_block_keeps_the_previous_one(
        105, UNSAFE_ORIGIN, &[UNSAFE_ORIGIN, 106, 103], Some(18)
    )]
    #[case::reorg_of_every_recent_safe_heads_l1_block(
        103, UNSAFE_ORIGIN, &[UNSAFE_ORIGIN, 106, 103], None
    )]
    // L2 block 20's L1 block 106 is still canonical; only the unsafe head's origin was replaced.
    #[case::reorg_of_the_unsafe_heads_l1_origin(107, 107, &[107], None)]
    #[tokio::test]
    async fn finds_the_newest_safe_head_derived_from_canonical_l1(
        #[case] reorg_height: u64,
        #[case] unsafe_origin: u64,
        #[case] l1_lookups: &[u64],
        #[case] kept: Option<u64>,
    ) {
        let (engine, client, l1, l2, _) = setup(unsafe_origin);
        for &number in l1_lookups {
            expect_l1(&l1, number, reorg_height);
        }
        if let Some(kept) = kept {
            expect_l2(&l2, kept, l2_block(kept, 0).block_info.hash);
        }

        let safe_head =
            find_reorg_safe_head(&client, engine.state(), &recent_safe_heads()).await.unwrap();

        assert_eq!(safe_head.map(|head| head.block_info.number), kept);
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A safe head outside the engine's local-safe and finalized heads is not kept, even if its L1
    /// block is canonical.
    #[rstest]
    #[case::above_the_local_safe_head(25)]
    #[case::below_the_finalized_head(FINALIZED - 1)]
    #[tokio::test]
    async fn does_not_keep_a_safe_head_outside_the_engines_safe_range(#[case] number: u64) {
        let (engine, client, l1, l2, _) = setup(UNSAFE_ORIGIN);
        expect_l1(&l1, UNSAFE_ORIGIN, 108);
        expect_l1(&l1, 106, 108);
        let candidate =
            DerivedSafeHead { safe_head: l2_block(number, 99), derived_from: l1_id(106, 0) };

        let safe_head = find_reorg_safe_head(&client, engine.state(), &[candidate]).await.unwrap();

        assert_eq!(safe_head, None);
        l1.assert_finished();
        l2.assert_finished();
    }

    /// Scripts the forkchoice update that moves the safe labels, checking the heads it sends.
    fn expect_forkchoice_update(l2: &RpcMock, head: L2BlockInfo, safe: L2BlockInfo) {
        let state = alloy_rpc_types_engine::ForkchoiceState {
            head_block_hash: head.block_info.hash,
            safe_block_hash: safe.block_info.hash,
            finalized_block_hash: l2_block(FINALIZED, 90).block_info.hash,
        };
        l2.expect_params(
            "engine_forkchoiceUpdatedV3",
            json!([state, null]),
            ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
        );
    }

    /// A safe head no longer on the engine's chain is not kept, even if its L1 block is canonical.
    #[tokio::test]
    async fn does_not_keep_a_safe_head_off_the_engines_chain() {
        let (engine, client, l1, l2, _) = setup(UNSAFE_ORIGIN);
        expect_l1(&l1, UNSAFE_ORIGIN, 108);
        expect_l1(&l1, 106, 108);
        expect_l2(&l2, 20, B256::repeat_byte(0xaa));

        let safe_head =
            find_reorg_safe_head(&client, engine.state(), &recent_safe_heads()).await.unwrap();

        assert_eq!(safe_head, None);
        l1.assert_finished();
        l2.assert_finished();
    }

    #[tokio::test]
    async fn tip_reorg_keeps_every_head_without_a_forkchoice_update() {
        let (mut engine, client, l1, l2, config) = setup(UNSAFE_ORIGIN);
        let before = engine.state().sync_state;
        expect_l1(&l1, UNSAFE_ORIGIN, 108);
        expect_l1(&l1, 106, 108);
        expect_l2(&l2, 20, l2_block(20, 100).block_info.hash);

        let safe_head =
            engine.reset_after_l1_reorg(client, config, &recent_safe_heads()).await.unwrap();

        assert_eq!(safe_head, l2_block(20, 100));
        assert_eq!(engine.state().sync_state, before);
        l1.assert_finished();
        l2.assert_finished();
    }

    #[tokio::test]
    async fn reorg_of_the_safe_heads_l1_block_rewinds_only_the_safe_heads() {
        let (mut engine, client, l1, l2, config) = setup(UNSAFE_ORIGIN);
        for number in [UNSAFE_ORIGIN, 106, 103] {
            expect_l1(&l1, number, 105);
        }
        expect_l2(&l2, 18, l2_block(18, 99).block_info.hash);
        expect_forkchoice_update(&l2, l2_block(UNSAFE, UNSAFE_ORIGIN), l2_block(18, 99));

        let safe_head =
            engine.reset_after_l1_reorg(client, config, &recent_safe_heads()).await.unwrap();

        assert_eq!(safe_head, l2_block(18, 99));
        let sync_state = engine.state().sync_state;
        assert_eq!(sync_state.local_safe_head(), safe_head);
        assert_eq!(sync_state.safe_head(), safe_head);
        assert_eq!(sync_state.unsafe_head(), l2_block(UNSAFE, UNSAFE_ORIGIN));
        assert_eq!(sync_state.finalized_head(), l2_block(FINALIZED, 90));
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A cross-safe head already below the kept safe head stays where it is: only local-safe moves.
    #[tokio::test]
    async fn a_cross_safe_head_below_the_kept_one_is_left_alone() {
        let (mut engine, client, l1, l2, config) = setup(UNSAFE_ORIGIN);
        let cross_safe = l2_block(15, 98);
        engine.state.sync_state = engine.state.sync_state.apply_update(EngineSyncStateUpdate {
            unsafe_head: None,
            local_safe_head: None,
            safe_head: Some(cross_safe),
            finalized_head: None,
        });
        for number in [UNSAFE_ORIGIN, 106, 103] {
            expect_l1(&l1, number, 105);
        }
        expect_l2(&l2, 18, l2_block(18, 99).block_info.hash);
        expect_forkchoice_update(&l2, l2_block(UNSAFE, UNSAFE_ORIGIN), cross_safe);

        let safe_head =
            engine.reset_after_l1_reorg(client, config, &recent_safe_heads()).await.unwrap();

        assert_eq!(engine.state().sync_state.local_safe_head(), safe_head);
        assert_eq!(engine.state().sync_state.safe_head(), cross_safe);
        l1.assert_finished();
        l2.assert_finished();
    }

    /// With no safe head to keep, the reset falls back to the full walk, which starts by reading
    /// the engine's finalized head.
    #[tokio::test]
    async fn falls_back_to_the_full_walk() {
        let (mut engine, client, l1, l2, config) = setup(UNSAFE_ORIGIN);
        for number in [UNSAFE_ORIGIN, 106, 103] {
            expect_l1(&l1, number, 103);
        }
        l2.expect_error("eth_getBlockByNumber");

        let err =
            engine.reset_after_l1_reorg(client, config, &recent_safe_heads()).await.unwrap_err();

        assert!(
            matches!(err, EngineResetError::SyncStart(SyncStartError::RpcError(_))),
            "got {err:?}"
        );
        l1.assert_finished();
        l2.assert_finished();
    }
}
