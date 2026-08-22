//! A task to commit an externally built payload, answering the caller that requested it.

use super::{CommitBlockError, CommitTaskError};
use crate::{EngineClient, EngineState, EngineTaskExt, InsertTask};
use derive_more::Constructor;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;
use tokio::sync::mpsc;

/// Task to commit an externally built payload: `opstack_commitBlockV1`'s write.
///
/// This is an [`InsertTask`] with an answer. The gossip path enqueues its inserts fire-and-forget
/// — a peer that sent a bad payload is not waiting to hear about it — but a caller of the opstack
/// API is: op-node's `CommitBlock` (`op-node/rollup/engine/api.go`) returns the `newPayload`
/// verdict synchronously. So the insert's outcome, success or failure, is delivered over
/// `result_tx`, and — like [`SealTask`]'s channel — delivering it *is* the task succeeding: a
/// refused commit must reach the caller once, not be retried by the queue behind their back.
///
/// [`SealTask`]: crate::SealTask
#[derive(Debug, Clone, Constructor)]
pub struct CommitTask<EngineClient_: EngineClient> {
    /// The engine API client.
    pub engine: Arc<EngineClient_>,
    /// The [`RollupConfig`].
    pub cfg: Arc<RollupConfig>,
    /// The payload to commit.
    pub payload: OpExecutionPayloadEnvelope,
    /// Where the commit's outcome is delivered.
    pub result_tx: mpsc::Sender<Result<L2BlockInfo, CommitBlockError>>,
}

impl<EngineClient_: EngineClient> CommitTask<EngineClient_> {
    /// Runs the insert and reports what happened.
    async fn commit(&self, state: &mut EngineState) -> Result<L2BlockInfo, CommitBlockError> {
        // The same admission rule the gossip path applies before its inserts
        // (`EngineTask::execute_inner`): a payload at or below the local-safe head must not
        // become the unsafe head. The gossip path drops such a payload silently; here the caller
        // hears the refusal.
        let insert =
            InsertTask::new(Arc::clone(&self.engine), self.cfg.clone(), self.payload.clone(), None);
        if !insert.descends_from_local_safe(state) {
            return Err(CommitBlockError::DoesNotDescendFromLocalSafe);
        }

        insert.execute(state).await.map_err(CommitBlockError::from)
    }
}

#[async_trait::async_trait]
impl<EngineClient_: EngineClient> EngineTaskExt for CommitTask<EngineClient_> {
    type Output = ();

    type Error = CommitTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<(), CommitTaskError> {
        let result = self.commit(state).await;
        // A requester that went away before hearing the result — an RPC caller that disconnected —
        // is not a task failure: the commit itself already happened (or was refused), and there is
        // no severity that fits a dead client. Failing Critical would halt the node over it, and
        // Temporary would retry a send that can never succeed.
        if self.result_tx.send(result).await.is_err() {
            warn!(
                target: "engine",
                "The commit requester went away before hearing the result"
            );
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        EngineSyncStateUpdate, LocalSafeHead, task_queue::tasks::task::EngineTask,
        test_utils::MockEngineClient,
    };
    use alloy_consensus::Block;
    use alloy_primitives::B256;
    use alloy_rpc_types_engine::{ExecutionPayloadV1, PayloadStatus, PayloadStatusEnum};
    use kona_protocol::BlockInfo;
    use op_alloy_consensus::OpTxEnvelope;
    use std::time::Duration;

    fn payload_at(number: u64, parent_hash: B256) -> OpExecutionPayloadEnvelope {
        let mut payload = ExecutionPayloadV1::from_block_slow(&Block::<OpTxEnvelope>::default());
        payload.block_number = number;
        payload.parent_hash = parent_hash;
        OpExecutionPayloadEnvelope::V1(payload)
    }

    /// A state whose unsafe and local-safe heads both sit at `head`.
    fn state_at(head: L2BlockInfo) -> EngineState {
        let mut state = EngineState::default();
        state.sync_state = state.apply_sync_update(EngineSyncStateUpdate {
            unsafe_head: Some(head),
            local_safe_head: Some(LocalSafeHead::unpaired(head)),
            ..Default::default()
        });
        state
    }

    /// A rejected commit answers the caller instead of being dropped like a gossiped payload, and
    /// the task completes: the queue must not retry a write whose requester was already refused.
    #[tokio::test]
    async fn a_refused_commit_answers_the_caller_and_completes() {
        let head = L2BlockInfo {
            block_info: BlockInfo {
                number: 7,
                hash: B256::repeat_byte(7),
                parent_hash: B256::repeat_byte(6),
                timestamp: 14,
            },
            ..Default::default()
        };
        let mut state = state_at(head);

        let config = Arc::new(RollupConfig::default());
        let client = Arc::new(MockEngineClient::builder().with_config(config.clone()).build());
        let (result_tx, mut result_rx) = mpsc::channel(1);

        // Number 3 is behind local-safe head 7: the two decidable rejection cases both refuse it.
        let task = EngineTask::Commit(Box::new(CommitTask::new(
            client,
            config,
            payload_at(3, B256::repeat_byte(2)),
            result_tx,
        )));

        tokio::time::timeout(Duration::from_secs(1), task.execute(&mut state))
            .await
            .expect("a refused commit must not retry")
            .expect("delivering the refusal is the task succeeding");

        let answer = result_rx.recv().await.expect("the caller hears the refusal");
        assert!(matches!(answer, Err(CommitBlockError::DoesNotDescendFromLocalSafe)));
        assert_eq!(state.sync_state.unsafe_head(), head, "a refused commit moves no head");
    }

    /// An insert the execution layer rejects reaches the caller as the insert's error, once,
    /// rather than riding the queue's temporary-error retry loop forever.
    #[tokio::test]
    async fn a_rejected_payload_reaches_the_caller_once() {
        let config = Arc::new(RollupConfig::default());
        let client = Arc::new(
            MockEngineClient::builder()
                .with_config(config.clone())
                .with_new_payload_v1_response(PayloadStatus::from_status(
                    PayloadStatusEnum::Invalid { validation_error: "bad".into() },
                ))
                .build(),
        );
        let (result_tx, mut result_rx) = mpsc::channel(1);

        let task = EngineTask::Commit(Box::new(CommitTask::new(
            client,
            config,
            payload_at(1, B256::ZERO),
            result_tx,
        )));

        tokio::time::timeout(Duration::from_secs(1), task.execute(&mut EngineState::default()))
            .await
            .expect("a rejected commit must not retry")
            .expect("delivering the rejection is the task succeeding");

        let answer = result_rx.recv().await.expect("the caller hears the rejection");
        assert!(matches!(
            answer,
            Err(CommitBlockError::Insert(crate::InsertTaskError::UnexpectedPayloadStatus(
                PayloadStatusEnum::Invalid { .. }
            )))
        ));
    }
}
