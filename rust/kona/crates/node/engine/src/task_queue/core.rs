//! The [`Engine`] is a task queue that receives and executes [`EngineTask`]s.

use super::EngineTaskExt;
use crate::{
    EngineClient, EngineState, EngineSyncStateUpdate, EngineTask, EngineTaskError,
    EngineTaskErrorSeverity, Metrics, SyncStartError, SynchronizeTask, SynchronizeTaskError,
    sync::{SafeCheckpoint, find_starting_forkchoice_with_checkpoints},
    task_queue::EngineTaskErrors,
};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::{cmp::Ordering, collections::BinaryHeap, sync::Arc};
use thiserror::Error;
use tokio::sync::watch::Sender;

/// A stable ordering within each task priority. Retried work can accumulate many same-kind
/// requests, which must keep their original order (especially unsafe payload imports).
#[derive(Debug)]
struct QueuedTask<C: EngineClient> {
    sequence: u64,
    task: EngineTask<C>,
}
impl<C: EngineClient> PartialEq for QueuedTask<C> {
    fn eq(&self, other: &Self) -> bool {
        self.cmp(other) == Ordering::Equal
    }
}
impl<C: EngineClient> Eq for QueuedTask<C> {}
impl<C: EngineClient> PartialOrd for QueuedTask<C> {
    fn partial_cmp(&self, other: &Self) -> Option<Ordering> {
        Some(self.cmp(other))
    }
}
impl<C: EngineClient> Ord for QueuedTask<C> {
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
pub struct Engine<EngineClient_: EngineClient> {
    /// The state of the engine.
    state: EngineState,
    /// A sender that can be used to notify the engine actor of state changes.
    state_sender: Sender<EngineState>,
    /// A sender that can be used to notify the engine actor of task queue length changes.
    task_queue_length: Sender<usize>,
    /// The task queue.
    tasks: BinaryHeap<QueuedTask<EngineClient_>>,
    next_sequence: u64,
    /// Preserve the task being retried ahead of newly enqueued work: it may have already
    /// performed part of an Engine API operation.
    active: Option<EngineTask<EngineClient_>>,
    /// Recent accepted completed-safe checkpoints; no unverified EL labels are recorded.
    safe_checkpoints: Vec<SafeCheckpoint>,
}

impl<EngineClient_: EngineClient> Engine<EngineClient_> {
    /// Creates a new [`Engine`] with an empty task queue and the passed initial [`EngineState`].
    pub fn new(
        initial_state: EngineState,
        state_sender: Sender<EngineState>,
        task_queue_length: Sender<usize>,
    ) -> Self {
        Self {
            state: initial_state,
            state_sender,
            task_queue_length,
            tasks: BinaryHeap::default(),
            active: None,
            safe_checkpoints: Vec::new(),
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

    /// Returns a receiver that can be used to listen to engine queue length updates.
    pub fn queue_length_subscribe(&self) -> tokio::sync::watch::Receiver<usize> {
        self.task_queue_length.subscribe()
    }

    /// Enqueues a new [`EngineTask`] for execution.
    /// Updates the queue length and notifies listeners of the change.
    pub fn enqueue(&mut self, task: EngineTask<EngineClient_>) {
        let sequence = self.next_sequence;
        self.next_sequence =
            self.next_sequence.checked_add(1).expect("engine task sequence exhausted");
        self.tasks.push(QueuedTask { sequence, task });
        self.task_queue_length.send_replace(self.len());
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
    /// [`crate::find_starting_forkchoice`]. The state will be updated to the starting point, and a
    /// forkchoice update will be enqueued in order to reorg the execution layer.
    pub async fn reset(
        &mut self,
        client: Arc<EngineClient_>,
        config: Arc<RollupConfig>,
    ) -> Result<L2BlockInfo, EngineResetError> {
        let previous_safe = self.state.sync_state.safe_head();
        let start = find_starting_forkchoice_with_checkpoints(
            &config,
            client.as_ref(),
            &self.safe_checkpoints,
        )
        .await?;

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

        self.safe_checkpoints
            .retain(|checkpoint| checkpoint.safe.block_info.number <= start.safe.block_info.number);
        self.state_sender.send_replace(self.state);
        if start.safe.block_info.number < previous_safe.block_info.number {
            warn!(target: "engine", old_safe = previous_safe.block_info.number,
                new_safe = start.safe.block_info.number,
                rollback_blocks = previous_safe.block_info.number - start.safe.block_info.number,
                "Reset demoted completed safe head");
        }
        kona_macros::inc!(counter, Metrics::ENGINE_RESET_COUNT);

        Ok(start.safe)
    }

    /// Clears the task queue.
    pub fn clear(&mut self) {
        self.tasks.clear();
        self.active = None;
        self.task_queue_length.send_replace(0);
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

            if let EngineTask::Consolidate(task) = task &&
                let crate::ConsolidateInput::Attributes(attributes) = &task.input &&
                attributes.is_last_in_span &&
                let Some(source) = attributes.derived_from &&
                self.state.sync_state.safe_head().block_info.number == attributes.block_number()
            {
                let safe = self.state.sync_state.safe_head();
                // Replace records after a rewind, and coalesce multiple blocks derived from
                // one L1 input. Bound memory; older reorgs use conservative walkback.
                self.safe_checkpoints.retain(|entry| {
                    entry.safe.block_info.number < safe.block_info.number &&
                        entry.source.number < source.number
                });
                self.safe_checkpoints.push(SafeCheckpoint { safe, source });
                if self.safe_checkpoints.len() > 4096 {
                    self.safe_checkpoints.remove(0);
                }
            }
            self.state_sender.send_replace(self.state);
            self.active = None;
            self.task_queue_length.send_replace(self.len());

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
            Self::SyncStart(
                SyncStartError::RpcError(_) |
                SyncStartError::ChainViewChanged |
                SyncStartError::CheckpointNotValidated,
            ) => true,
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
            MockEngineClient, TestAttributesBuilder, TestEngineStateBuilder, test_block_info,
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
        let client = Arc::new(MockEngineClient::builder().with_config(config.clone()).build());
        let (state_tx, _state_rx) = tokio::sync::watch::channel(EngineState::default());
        let (queue_tx, queue_rx) = tokio::sync::watch::channel(0);
        let mut engine = Engine::new(EngineState::default(), state_tx, queue_tx);
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
        assert_eq!(*queue_rx.borrow(), 1);
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
        client
            .set_new_payload_v1_response(PayloadStatus::from_status(PayloadStatusEnum::Valid))
            .await;
        client
            .set_fork_choice_updated_v3_response(ForkchoiceUpdated::new(
                PayloadStatus::from_status(PayloadStatusEnum::Valid),
            ))
            .await;
        engine.drain().await.unwrap();
        assert!(engine.is_empty());
        assert_eq!(*queue_rx.borrow(), 0);
        assert_eq!(engine.state().sync_state.unsafe_head().block_info.number, 3);
    }

    // A flush means the task already replaced its block with a deposits-only block. Keeping it
    // would rebuild that replacement on every later drain.
    #[rstest::rstest]
    #[case(true)]
    #[case(false)]
    #[tokio::test]
    async fn flushed_task_is_dropped_not_retried(#[case] completed_span: bool) {
        let parent = test_block_info(10);
        let deposit = alloy_primitives::Bytes::from(vec![op_alloy_consensus::DEPOSIT_TX_TYPE_ID]);
        let mut attributes = TestAttributesBuilder::new()
            .with_parent(parent)
            .with_transactions(vec![deposit, alloy_primitives::Bytes::from(vec![0x02])])
            .build();
        let source = kona_protocol::BlockInfo { number: 20, ..Default::default() };
        attributes.derived_from = Some(source);
        attributes.is_last_in_span = completed_span;

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
        let client = Arc::new(
            MockEngineClient::builder()
                .with_config(config.clone())
                .with_fork_choice_updated_v3_rejecting_non_deposit_attributes()
                .with_fork_choice_updated_v3_response(ForkchoiceUpdated {
                    payload_status: valid.clone(),
                    payload_id: Some(PayloadId::new([1; 8])),
                })
                .with_execution_payload_v3(replacement)
                .with_new_payload_v3_response(valid)
                .build(),
        );
        // The safe head is at the unsafe head, so the task builds the attributes instead of
        // consolidating them against an unsafe block.
        let state = TestEngineStateBuilder::new().with_unsafe_head(parent).build();
        let (state_tx, state_rx) = tokio::sync::watch::channel(EngineState::default());
        let (queue_tx, queue_rx) = tokio::sync::watch::channel(0);
        let mut engine = Engine::new(state, state_tx, queue_tx);
        engine.enqueue(EngineTask::Consolidate(Box::new(ConsolidateTask::new(
            client.clone(),
            config,
            ConsolidateInput::Attributes(Box::new(attributes)),
            Arc::new(NoopBlockSink),
        ))));

        let error = engine.drain().await.unwrap_err();
        assert_eq!(error.severity(), EngineTaskErrorSeverity::Flush, "got {error:?}");
        let builds = client.fork_choice_updated_v3_attributes().await.iter().flatten().count();
        assert_eq!(builds, 2, "one build for the batch, one for its replacement");
        assert!(engine.is_empty());
        assert_eq!(*queue_rx.borrow(), 0);
        assert_eq!(engine.safe_checkpoints.len(), usize::from(completed_span));
        if completed_span {
            assert_eq!(engine.safe_checkpoints[0].source, source);
            assert_eq!(engine.safe_checkpoints[0].safe.hash(), imported.header.hash_slow());
        }
        // The task updates `engine.state()` itself; only drain publishes it.
        assert_eq!(
            state_rx.borrow().sync_state.safe_head().block_info.hash,
            imported.header.hash_slow()
        );
    }
}

#[cfg(test)]
mod checkpoint_reset_tests {
    use super::*;
    use crate::test_utils::{MockEngineClient, TestEngineStateBuilder};
    use alloy_eips::BlockId;
    use alloy_primitives::{B256, Sealed};
    use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum};
    use kona_protocol::{BlockInfo, L1BlockInfoBedrock};

    fn l2_block(
        number: u64,
        origin: u64,
    ) -> alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction> {
        let data = L1BlockInfoBedrock::new(
            origin,
            0,
            0,
            B256::repeat_byte(1),
            0,
            Default::default(),
            Default::default(),
            Default::default(),
        )
        .encode_calldata();
        let transaction =
            op_alloy_consensus::OpTxEnvelope::Deposit(Sealed::new(op_alloy_consensus::TxDeposit {
                input: data,
                ..Default::default()
            }));
        let transaction = op_alloy_rpc_types::Transaction::from_transaction(
            alloy_consensus::transaction::Recovered::new_unchecked(transaction, Default::default()),
            Default::default(),
        );
        let mut block: alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction> =
            Default::default();
        block.header.inner.number = number;
        block.header.inner.timestamp = number * 2;
        block.header.hash = block.header.inner.hash_slow();
        block.transactions = alloy_rpc_types_eth::BlockTransactions::Full(vec![transaction]);
        block
    }

    #[tokio::test]
    async fn reset_retains_recent_completed_safe_instead_of_walking_seventy_thousand_blocks() {
        let config = Arc::new(RollupConfig::default());
        let safe = l2_block(117029, 10635);
        let unsafe_block = l2_block(117174, 10690);
        let finalized = l2_block(44302, 7084);
        let safe_info =
            L2BlockInfo::from_block_and_genesis(&safe.clone().into_consensus(), &config.genesis)
                .unwrap();
        let unsafe_info = L2BlockInfo::from_block_and_genesis(
            &unsafe_block.clone().into_consensus(),
            &config.genesis,
        )
        .unwrap();
        let mut head: <alloy_network::Ethereum as alloy_provider::Network>::BlockResponse =
            Default::default();
        head.header.inner.number = 10690;
        head.header.hash = B256::repeat_byte(2); // A replacement head, not the old branch's hash.
        let mut source: <alloy_network::Ethereum as alloy_provider::Network>::BlockResponse =
            Default::default();
        source.header.inner.number = 10679;
        source.header.hash = B256::repeat_byte(3);
        let checkpoint = SafeCheckpoint {
            safe: safe_info,
            source: BlockInfo {
                number: source.header.number,
                hash: source.header.hash,
                ..Default::default()
            },
        };
        let client = Arc::new(
            MockEngineClient::builder()
                .with_config(config.clone())
                .with_l2_block(BlockId::latest(), unsafe_block.clone())
                .with_l2_block(BlockId::safe(), safe.clone())
                .with_l2_block(BlockId::finalized(), finalized)
                .with_l2_block(safe_info.block_info.number.into(), safe)
                .with_l2_block(unsafe_info.block_info.number.into(), unsafe_block)
                .with_l1_block(unsafe_info.l1_origin.hash.into(), head.clone())
                .with_l1_block(BlockId::latest(), head.clone())
                .with_l1_block(head.header.number.into(), head)
                .with_l1_block(checkpoint.source.number.into(), source)
                .with_fork_choice_updated_v3_response(ForkchoiceUpdated::new(
                    PayloadStatus::from_status(PayloadStatusEnum::Valid),
                ))
                .build(),
        );
        let state = TestEngineStateBuilder::new()
            .with_safe_head(safe_info)
            .with_unsafe_head(unsafe_info)
            .build();
        let (sender, receiver) = tokio::sync::watch::channel(state);
        let (queue, _) = tokio::sync::watch::channel(0);
        let mut engine = Engine::new(state, sender, queue);
        engine.safe_checkpoints.push(checkpoint);
        // No old L2 ancestors are supplied: the historical full-window walk fails this fixture.
        assert_eq!(engine.reset(client, config).await.unwrap(), safe_info);
        assert_eq!(engine.state().sync_state.safe_head(), safe_info);
        assert_eq!(receiver.borrow().sync_state.safe_head(), safe_info);
    }
}
