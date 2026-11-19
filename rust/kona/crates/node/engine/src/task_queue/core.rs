//! The [`Engine`] is a task queue that receives and executes [`EngineTask`]s.

use super::ExecutableEngineTask;
use crate::{
    EngineClient, EngineState, EngineSyncState, EngineSyncStateUpdate, EngineTask, EngineTaskError,
    EngineTaskErrorSeverity, L2ForkchoiceState, Metrics, SyncStartError, SynchronizeTaskError,
    find_starting_forkchoice,
    sync::l2_block_info_at,
    task_queue::{
        EngineTaskErrors, ForkchoiceReply, restore_forkchoice, synchronize_unconditionally,
    },
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
    /// [`find_starting_forkchoice`]. Synchronizes the execution layer and publishes the
    /// recovered state before returning, even if no tasks are queued.
    pub async fn reset(
        &mut self,
        client: Arc<EngineClient>,
        config: Arc<RollupConfig>,
    ) -> Result<L2BlockInfo, EngineResetError> {
        let start = self.find_starting_forkchoice_restoring_finality(&client, &config).await?;

        // One attempt. The actor retains the reset request and retries transient failures. The
        // execution layer may have lost or changed its forkchoice, so the reset sends one even when
        // the recovered labels equal the current ones.
        synchronize_unconditionally(
            client.as_ref(),
            &mut self.state,
            EngineSyncStateUpdate {
                unsafe_head: Some(start.un_safe),
                local_safe_head: Some(start.safe),
                safe_head: Some(start.safe),
                finalized_head: Some(start.finalized),
            },
        )
        .await?;

        self.state_sender.send_replace(self.state);
        metrics::counter!(Metrics::ENGINE_RESET_COUNT).increment(1);

        Ok(start.safe)
    }

    /// Finds the sync start. A walk that misses the finalized block follows a forkchoice update
    /// that the execution layer made canonical before rejecting it. The finalized block anchors
    /// the restore: the engine's established one, or after a restart, the execution layer's own
    /// finalized label. While the execution layer still holds the anchor, the engine restores a
    /// forkchoice on its chain and walks once more; otherwise restoring cannot help, and the
    /// mismatch stands.
    async fn find_starting_forkchoice_restoring_finality(
        &self,
        client: &Arc<EngineClient>,
        config: &Arc<RollupConfig>,
    ) -> Result<L2ForkchoiceState, EngineResetError> {
        let (mismatch, el_finalized) = match find_starting_forkchoice(config, client.as_ref()).await
        {
            Err(mismatch @ SyncStartError::MismatchedFinalizedBlock { finalized, .. }) => {
                (mismatch, finalized)
            }
            result => return Ok(result?),
        };
        let established_finalized = self.state.sync_state.established_finalized_head().is_some();
        let anchor = if established_finalized {
            self.state.sync_state.finalized_head()
        } else {
            l2_block_info_at(el_finalized)
        };
        if !client.l2_block_is_stored(anchor.block_info.hash).await.map_err(SyncStartError::from)? {
            return Err(mismatch.into());
        }

        warn!(target: "engine", %mismatch, ?anchor, "Restoring a forkchoice on the finalized chain before walking again");
        if established_finalized {
            self.restore_engine_forkchoice(client, anchor).await?;
        } else {
            restore_forkchoice(client.as_ref(), &finalized_only(anchor)).await?;
        }
        match find_starting_forkchoice(config, client.as_ref()).await {
            Err(mismatch @ SyncStartError::MismatchedFinalizedBlock { .. }) => {
                Err(EngineResetError::FinalityNotRestored(mismatch))
            }
            result => Ok(result?),
        }
    }

    /// Sends the engine's own forkchoice again. Its labels may include a delegated safe head that
    /// the execution layer never accepted, or an unsafe head it does not have; when it rejects
    /// them or answers `SYNCING`, a forkchoice of the `finalized` block alone, which it holds,
    /// restores its chain instead.
    async fn restore_engine_forkchoice(
        &self,
        client: &Arc<EngineClient>,
        finalized: L2BlockInfo,
    ) -> Result<(), SynchronizeTaskError> {
        match restore_forkchoice(client.as_ref(), &self.state.sync_state).await {
            Ok(ForkchoiceReply::Valid) => Ok(()),
            Ok(ForkchoiceReply::Syncing) | Err(SynchronizeTaskError::InvalidForkchoiceState) => {
                restore_forkchoice(client.as_ref(), &finalized_only(finalized)).await.map(|_| ())
            }
            Err(err) => Err(err),
        }
    }

    /// Clears the task queue.
    pub fn clear(&mut self) {
        self.tasks.clear();
        self.active = None;
    }

    /// Attempts to drain the queue by executing all [`EngineTask`]s in-order. If any task returns
    /// an error along the way, the error is returned and the task is not popped from the queue (in
    /// case it must be retried), unless it is a flush or skip error.
    pub async fn drain(&mut self) -> Result<(), EngineTaskErrors> {
        // Drain tasks in order of priority, halting on errors for a retry to be attempted.
        loop {
            if self.active.is_none() {
                self.active = self.tasks.pop().map(|queued| queued.task);
            }
            let Some(task) = &self.active else { break };
            let result = task.execute(&mut self.state).await;

            // A flush reports that the task replaced its block with a deposits-only block, so the
            // task is done; retrying it would replace that block again. A skip reports a task that
            // no longer applies, so retrying it cannot succeed either.
            if let Err(err) = &result &&
                !matches!(
                    err.severity(),
                    EngineTaskErrorSeverity::Flush | EngineTaskErrorSeverity::Skip
                )
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

/// A sync state whose every label is `finalized`.
fn finalized_only(finalized: L2BlockInfo) -> EngineSyncState {
    EngineSyncState::default().with_update(EngineSyncStateUpdate {
        finalized_head: Some(finalized),
        local_safe_head: Some(finalized),
        safe_head: Some(finalized),
        unsafe_head: Some(finalized),
    })
}

/// An error occurred while attempting to reset the [`Engine`].
#[derive(Debug, Error)]
pub enum EngineResetError {
    /// The walk still missed the finalized block after a forkchoice on that block's chain was
    /// restored, as while the execution layer has not yet made it canonical.
    #[error("Sync start still misses the finalized block after a restore: {0}")]
    FinalityNotRestored(SyncStartError),
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
            // After a restore the execution layer holds the engine's chain, and retrying gives it
            // time to make that chain canonical. A missing L1 origin of the finalized block means
            // the L1 provider is behind; one it serves only by number may be a backend behind a
            // load balancer whose hash index lags; a missing block at the finalized height means
            // the execution layer answered inconsistently.
            Self::FinalityNotRestored(_) |
            Self::SyncStart(
                SyncStartError::FinalizedHeightUnavailable(_) |
                SyncStartError::FinalizedL1OriginNotServedByHash { .. } |
                SyncStartError::FinalizedL1OriginUnavailable { .. } |
                SyncStartError::RpcError(_),
            ) => true,
            // An L1 provider on another chain than the finalized block's L1 origin needs
            // reconfiguring.
            Self::SyncStart(_) => false,
            Self::Forkchoice(err) => err.severity() != EngineTaskErrorSeverity::Critical,
        }
    }
}

#[cfg(test)]
mod recovery_tests {
    use super::*;
    use crate::{
        ConsolidateInput, ConsolidateTask, FinalizeBlockId, FinalizeTask, InsertTask,
        NoopBlockSink,
        test_utils::{
            RpcMock, TestAttributesBuilder, TestEngineStateBuilder, expect_sync_start,
            expect_sync_start_walk, test_block_info, test_engine_client, test_l2_chain,
        },
    };
    use alloy_primitives::B256;
    use alloy_rpc_types_engine::{
        BlobsBundleV1, ExecutionPayloadV1, ExecutionPayloadV3, ForkchoiceState, ForkchoiceUpdated,
        PayloadId, PayloadStatus, PayloadStatusEnum,
    };
    use op_alloy_consensus::OpBlock;
    use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpExecutionPayloadEnvelopeV3};

    /// A canonical chain `1..=4` and a fork from block 1 at `2..=4`, as an execution layer holds
    /// them after a rejected forkchoice update made the fork canonical before checking block 2,
    /// the finalized block.
    struct ForkedExecutionLayer {
        canonical: Vec<OpBlock>,
        fork: Vec<OpBlock>,
    }

    impl ForkedExecutionLayer {
        fn new() -> Self {
            let canonical = test_l2_chain(B256::ZERO, 1, 4, 0);
            let fork = test_l2_chain(canonical[0].header.hash_slow(), 2, 4, 1);
            Self { canonical, fork }
        }

        fn info(block: &OpBlock) -> L2BlockInfo {
            L2BlockInfo::from_block_and_genesis(block, &RollupConfig::default().genesis).unwrap()
        }

        fn finalized(&self) -> L2BlockInfo {
            Self::info(&self.canonical[1])
        }

        /// The engine's labels: unsafe at canonical block 4, safe and finalized at block 2.
        fn engine_state(&self) -> EngineState {
            TestEngineStateBuilder::new()
                .with_unsafe_head(Self::info(&self.canonical[3]))
                .with_safe_head(self.finalized())
                .with_finalized_head(self.finalized())
                .build()
        }

        fn expect_walk_on_fork(&self, l1: &RpcMock, l2: &RpcMock) {
            expect_sync_start_walk(
                l1,
                l2,
                &self.canonical[1],
                &self.fork[2],
                true,
                &[&self.fork[1], &self.fork[0]],
            );
        }

        fn expect_walk_on_canonical_chain(&self, l1: &RpcMock, l2: &RpcMock) {
            expect_sync_start_walk(
                l1,
                l2,
                &self.canonical[1],
                &self.canonical[3],
                true,
                &[&self.canonical[2], &self.canonical[1]],
            );
        }

        /// The forkchoice of the finalized block alone.
        fn finalized_only_forkchoice(&self) -> ForkchoiceState {
            let finalized = self.finalized().block_info.hash;
            ForkchoiceState {
                finalized_block_hash: finalized,
                head_block_hash: finalized,
                safe_block_hash: finalized,
            }
        }

        /// The execution layer's answer to the lookup of the engine's finalized block by hash.
        fn expect_finalized_lookup(&self, l2: &RpcMock, el_holds_it: bool) {
            l2.expect_block_header_by_hash(
                self.finalized().block_info.hash,
                el_holds_it.then(|| crate::test_utils::test_rpc_block(&self.canonical[1])),
            );
        }
    }

    async fn reset_from(
        initial: EngineState,
        client: EngineClient,
    ) -> Result<L2BlockInfo, EngineResetError> {
        let (state_tx, _state_rx) = tokio::sync::watch::channel(initial);
        let mut engine = Engine::new(initial, state_tx);
        engine.reset(Arc::new(client), Arc::new(RollupConfig::default())).await
    }

    /// A reset whose walk misses the finalized block, which the execution layer still holds,
    /// sends the engine's own labels again, which makes its chain canonical, and then starts from
    /// the walk that meets the finalized block.
    #[tokio::test]
    async fn reset_restores_the_engine_forkchoice_when_the_walk_misses_finality() {
        let el = ForkedExecutionLayer::new();
        let initial = el.engine_state();
        let forkchoice = initial.sync_state.create_forkchoice_state();
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        el.expect_walk_on_fork(&l1, &l2);
        el.expect_finalized_lookup(&l2, true);
        l2.expect_forkchoice_updated(forkchoice, PayloadStatusEnum::Valid);
        el.expect_walk_on_canonical_chain(&l1, &l2);
        l2.expect_forkchoice_updated(forkchoice, PayloadStatusEnum::Valid);

        let safe = reset_from(initial, client).await.unwrap();

        assert_eq!(safe, el.finalized());
        l1.assert_finished();
        l2.assert_finished();
    }

    /// The engine's own labels may carry a safe head the execution layer never accepted; when it
    /// rejects them, a forkchoice of the finalized block alone restores its chain instead.
    #[tokio::test]
    async fn rejected_restore_falls_back_to_the_finalized_block() {
        let el = ForkedExecutionLayer::new();
        let initial = el.engine_state();
        let finalized_only = el.finalized_only_forkchoice();
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        el.expect_walk_on_fork(&l1, &l2);
        el.expect_finalized_lookup(&l2, true);
        l2.expect_error_code(
            "engine_forkchoiceUpdatedV3",
            i64::from(alloy_rpc_types_engine::INVALID_FORK_CHOICE_STATE_ERROR),
        );
        l2.expect_forkchoice_updated(finalized_only, PayloadStatusEnum::Valid);
        el.expect_walk_on_canonical_chain(&l1, &l2);
        l2.expect_forkchoice_updated(
            initial.sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Valid,
        );

        let safe = reset_from(initial, client).await.unwrap();

        assert_eq!(safe, el.finalized());
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A walk that still misses the finalized block after the engine's labels and then the
    /// finalized block alone were sent, as while the execution layer syncs back to that chain,
    /// fails the reset with a retryable error.
    #[tokio::test]
    async fn reset_retries_while_the_restored_forkchoice_is_not_yet_canonical() {
        let el = ForkedExecutionLayer::new();
        let initial = el.engine_state();
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        el.expect_walk_on_fork(&l1, &l2);
        el.expect_finalized_lookup(&l2, true);
        l2.expect_forkchoice_updated(
            initial.sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Syncing,
        );
        l2.expect_forkchoice_updated(el.finalized_only_forkchoice(), PayloadStatusEnum::Syncing);
        el.expect_walk_on_fork(&l1, &l2);

        let error = reset_from(initial, client).await.unwrap_err();

        assert!(matches!(error, EngineResetError::FinalityNotRestored(_)), "got {error:?}");
        assert!(error.is_temporary());
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A walk that misses the finalized block stops the node, like op-node's `ReorgFinalizedErr`,
    /// when the execution layer no longer holds the restore's anchor: the engine's established
    /// finalized block, or after a restart, the execution layer's own finalized label.
    #[rstest::rstest]
    #[case::engine_finality_lost(ForkedExecutionLayer::new().engine_state())]
    #[case::el_finalized_label_lost_after_a_restart(EngineState::default())]
    #[tokio::test]
    async fn walk_that_misses_finality_stops_the_node_when_the_anchor_is_lost(
        #[case] initial: EngineState,
    ) {
        let el = ForkedExecutionLayer::new();
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        el.expect_walk_on_fork(&l1, &l2);
        el.expect_finalized_lookup(&l2, false);

        let error = reset_from(initial, client).await.unwrap_err();

        assert!(
            matches!(
                error,
                EngineResetError::SyncStart(SyncStartError::MismatchedFinalizedBlock { .. })
            ),
            "got {error:?}"
        );
        assert!(!error.is_temporary());
        l1.assert_finished();
        l2.assert_finished();
    }

    /// After a restart the engine has no finality of its own, so a rejected forkchoice update
    /// that left the execution layer on a fork from below its finalized block is undone with the
    /// execution layer's own finalized label: a forkchoice of that block alone, from which the
    /// walk then starts.
    #[tokio::test]
    async fn reset_after_a_restart_restores_the_el_finalized_block() {
        let el = ForkedExecutionLayer::new();
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        el.expect_walk_on_fork(&l1, &l2);
        el.expect_finalized_lookup(&l2, true);
        l2.expect_forkchoice_updated(el.finalized_only_forkchoice(), PayloadStatusEnum::Valid);
        expect_sync_start_walk(&l1, &l2, &el.canonical[1], &el.canonical[1], true, &[]);
        l2.expect_forkchoice_updated(el.finalized_only_forkchoice(), PayloadStatusEnum::Valid);

        let safe = reset_from(EngineState::default(), client).await.unwrap();

        assert_eq!(safe, el.finalized());
        l1.assert_finished();
        l2.assert_finished();
    }

    /// An engine whose labels the execution layer answers `SYNCING`, as when it lacks the
    /// engine's unsafe head, restores the finalized block alone instead, and the walk then meets
    /// it.
    #[tokio::test]
    async fn syncing_restore_falls_back_to_the_finalized_block() {
        let el = ForkedExecutionLayer::new();
        let initial = el.engine_state();
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        el.expect_walk_on_fork(&l1, &l2);
        el.expect_finalized_lookup(&l2, true);
        l2.expect_forkchoice_updated(
            initial.sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Syncing,
        );
        l2.expect_forkchoice_updated(el.finalized_only_forkchoice(), PayloadStatusEnum::Valid);
        el.expect_walk_on_canonical_chain(&l1, &l2);
        l2.expect_forkchoice_updated(
            initial.sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Valid,
        );

        let safe = reset_from(initial, client).await.unwrap();

        assert_eq!(safe, el.finalized());
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A permanent sync-start failure on the walk after a restore stays fatal instead of being
    /// retried as a finality that is not restored yet.
    #[tokio::test]
    async fn permanent_sync_start_error_after_a_restore_stays_fatal() {
        let el = ForkedExecutionLayer::new();
        let initial = el.engine_state();
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        el.expect_walk_on_fork(&l1, &l2);
        el.expect_finalized_lookup(&l2, true);
        l2.expect_forkchoice_updated(
            initial.sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Valid,
        );
        expect_sync_start_walk(&l1, &l2, &el.canonical[1], &el.canonical[3], true, &[]);
        l2.expect("eth_getBlockByHash", None::<crate::test_utils::TestRpcBlock>);

        let error = reset_from(initial, client).await.unwrap_err();

        assert!(
            matches!(error, EngineResetError::SyncStart(SyncStartError::BlockNotFound(_))),
            "got {error:?}"
        );
        assert!(!error.is_temporary());
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A reset retries the sync-start errors that a lagging or briefly inconsistent provider
    /// causes, and stops the node on the others.
    #[rstest::rstest]
    #[case::finalized_height_unavailable(SyncStartError::FinalizedHeightUnavailable(2), true)]
    #[case::block_not_found(SyncStartError::BlockNotFound(2u64.into()), false)]
    fn sync_start_errors_a_reset_retries(
        #[case] error: SyncStartError,
        #[case] is_temporary: bool,
    ) {
        assert_eq!(EngineResetError::SyncStart(error).is_temporary(), is_temporary);
    }

    /// An L1 provider without a block at the height of the finalized block's L1 origin lags, so
    /// the reset retries.
    fn l1_lacks_the_height(l1: &RpcMock) {
        l1.expect("eth_getBlockByNumber", None::<crate::test_utils::TestRpcBlock>);
    }

    /// An L1 provider with another block at that height follows another L1 chain, which retrying
    /// cannot fix.
    fn l1_on_another_chain(l1: &RpcMock) {
        let other_l1_block = alloy_rpc_types_eth::Block::<alloy_rpc_types_eth::Transaction> {
            header: alloy_rpc_types_eth::Header {
                hash: B256::repeat_byte(0x11),
                ..Default::default()
            },
            ..Default::default()
        };
        l1.expect("eth_getBlockByNumber", other_l1_block);
    }

    /// An L1 provider whose block at that height is the finalized block's L1 origin serves it by
    /// number but not by hash, as a lagging block-hash index behind a load balancer does.
    fn l1_serves_the_origin_only_by_number(l1: &RpcMock) {
        let l1_origin = ForkedExecutionLayer::new().finalized().l1_origin;
        let l1_block = alloy_rpc_types_eth::Block::<alloy_rpc_types_eth::Transaction> {
            header: alloy_rpc_types_eth::Header { hash: l1_origin.hash, ..Default::default() },
            ..Default::default()
        };
        l1.expect("eth_getBlockByNumber", l1_block);
    }

    fn is_l1_origin_not_served_by_hash(error: &EngineResetError) -> bool {
        matches!(
            error,
            EngineResetError::SyncStart(SyncStartError::FinalizedL1OriginNotServedByHash { .. })
        )
    }

    fn is_l1_origin_unavailable(error: &EngineResetError) -> bool {
        matches!(
            error,
            EngineResetError::SyncStart(SyncStartError::FinalizedL1OriginUnavailable { .. })
        )
    }

    fn is_l1_origin_not_canonical(error: &EngineResetError) -> bool {
        matches!(
            error,
            EngineResetError::SyncStart(SyncStartError::FinalizedL1OriginNotCanonical { .. })
        )
    }

    /// A finalized block whose L1 origin the L1 provider does not serve by hash is retried only
    /// when the provider lags behind that height, and stops the node when the provider follows
    /// another L1 chain.
    #[rstest::rstest]
    #[case::l1_lags(l1_lacks_the_height, is_l1_origin_unavailable, true)]
    #[case::l1_on_another_chain(l1_on_another_chain, is_l1_origin_not_canonical, false)]
    #[case::l1_serves_the_origin_only_by_number(
        l1_serves_the_origin_only_by_number,
        is_l1_origin_not_served_by_hash,
        true
    )]
    #[tokio::test]
    async fn missing_l1_origin_of_the_finalized_block(
        #[case] expect_l1_at_origin_height: fn(&RpcMock),
        #[case] is_expected: fn(&EngineResetError) -> bool,
        #[case] is_temporary: bool,
    ) {
        let el = ForkedExecutionLayer::new();
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        expect_sync_start_walk(&l1, &l2, &el.canonical[1], &el.canonical[1], false, &[]);
        expect_l1_at_origin_height(&l1);

        let error = reset_from(el.engine_state(), client).await.unwrap_err();

        assert!(is_expected(&error), "got {error:?}");
        assert_eq!(error.is_temporary(), is_temporary);
        l1.assert_finished();
        l2.assert_finished();
    }

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
        l2.expect_error("engine_newPayloadV2");
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
            l2.expect("engine_newPayloadV2", PayloadStatus::from_status(PayloadStatusEnum::Valid));
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

    // An execution layer that restarted can have lost its forkchoice, so a reset sends the
    // recovered forkchoice even when the labels already equal it.
    #[tokio::test]
    async fn reset_resends_an_unchanged_forkchoice() {
        let block: <op_alloy_network::Optimism as alloy_provider::Network>::BlockResponse =
            Default::default();
        let mut config = RollupConfig::default();
        config.genesis.l2.hash = block.header.inner.hash_slow();
        let genesis =
            L2BlockInfo::from_block_and_genesis(&block.clone().into_consensus(), &config.genesis)
                .unwrap();
        let config = Arc::new(config);
        let state = TestEngineStateBuilder::new().with_unsafe_head(genesis).build();
        let forkchoice = state.sync_state.create_forkchoice_state();
        let (client, l1, l2) = test_engine_client(config.clone());
        expect_sync_start(&l1, &l2, &block, forkchoice);
        let (state_tx, _state_rx) = tokio::sync::watch::channel(state);
        let mut engine = Engine::new(state, state_tx);

        engine.reset(Arc::new(client), config).await.unwrap();

        assert_eq!(engine.state().sync_state, state.sync_state);
        l1.assert_finished();
        l2.assert_finished();
    }

    // A skipped task is dropped instead of being kept for a retry, so it cannot hold the queue.
    #[tokio::test]
    async fn skipped_task_is_dropped_from_the_queue() {
        let config = Arc::new(RollupConfig::default());
        let (client, l1, l2) = test_engine_client(config.clone());
        let state = TestEngineStateBuilder::new()
            .with_unsafe_head(test_block_info(20))
            .with_finalized_head(test_block_info(10))
            .build();
        let (state_tx, _state_rx) = tokio::sync::watch::channel(state);
        let mut engine = Engine::new(state, state_tx);
        engine.enqueue(EngineTask::Finalize(Box::new(FinalizeTask::new(
            Arc::new(client),
            config,
            FinalizeBlockId::ByNumber(5),
        ))));

        let error = engine.drain().await.unwrap_err();

        assert_eq!(error.severity(), EngineTaskErrorSeverity::Skip, "got {error:?}");
        assert!(engine.is_empty());
        assert_eq!(engine.state().sync_state, state.sync_state);
        l1.assert_finished();
        l2.assert_finished();
    }
}
