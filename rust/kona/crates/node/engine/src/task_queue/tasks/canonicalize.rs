//! Import a sequenced payload after committing it to the conductor.

use super::util::{submit_payload, update_forkchoice};
use crate::{
    EngineClient, EngineState, EngineTaskError, EngineTaskExt, ImportedBlockSink, InsertTaskError,
    SynchronizeTaskError, state::EngineSyncStateUpdate,
    task_queue::tasks::task::EngineTaskErrorSeverity,
};
use alloy_rpc_types_engine::PayloadStatusEnum;
use async_trait::async_trait;
use derive_more::Constructor;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use op_alloy_consensus::OpBlock;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;
use thiserror::Error;
use tokio::sync::{mpsc, watch};

/// An error that occurs when running the [`CanonicalizeTask`].
#[derive(Debug, Error)]
pub enum CanonicalizeTaskError {
    /// The unsafe head changed after the payload was built, so the payload no longer extends it.
    #[error("Unsafe head changed between build and canonicalization")]
    UnsafeHeadChangedSinceBuild,
    /// Impossible to insert the payload into the engine.
    #[error(transparent)]
    PayloadInsertionFailed(#[from] Box<InsertTaskError>),
}

impl EngineTaskError for CanonicalizeTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::PayloadInsertionFailed(inner) => inner.severity(),
            Self::UnsafeHeadChangedSinceBuild => EngineTaskErrorSeverity::Critical,
        }
    }
}

/// Canonicalizes a sequenced payload, checking that its build parent is still the unsafe head.
///
/// The conductor RPC runs outside the engine queue. Derivation or another unsafe payload may
/// advance the head while that RPC is pending, so this check and insertion must be atomic.
/// Unlike derived attributes, an invalid sequenced payload must not be replaced by a deposits-only
/// block here: that replacement would need its own conductor commit.
#[derive(Debug, Clone, Constructor)]
pub struct CanonicalizeTask<C: EngineClient> {
    /// The engine API client.
    pub engine: Arc<C>,
    /// The rollup configuration.
    pub cfg: Arc<RollupConfig>,
    /// The payload to import.
    pub payload: OpExecutionPayloadEnvelope,
    /// The unsafe head on which the build started.
    pub parent: L2BlockInfo,
    /// The response channel. Errors are handled by the sequencer, which owns retry policy.
    pub result_tx: mpsc::Sender<Result<OpExecutionPayloadEnvelope, CanonicalizeTaskError>>,
    /// Publish the new unsafe head before acknowledging canonicalization to the sequencer.
    pub unsafe_head_tx: Option<watch::Sender<L2BlockInfo>>,
    /// Where to hand the block after canonicalization.
    pub block_sink: Arc<dyn ImportedBlockSink>,
}

impl<C: EngineClient> CanonicalizeTask<C> {
    /// Requires successful execution and forkchoice validation before publishing the block.
    async fn canonicalize(&self, state: &mut EngineState) -> Result<(), InsertTaskError> {
        let response = submit_payload(self.engine.as_ref(), self.payload.clone()).await?;
        if response.status != PayloadStatusEnum::Valid {
            return Err(InsertTaskError::UnexpectedPayloadStatus(response.status));
        }

        let block: OpBlock = self.payload.clone().try_into_block()?;
        let block_info = L2BlockInfo::from_block_and_genesis(&block, &self.cfg.genesis)?;
        let new_sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
            unsafe_head: Some(block_info),
            ..Default::default()
        });

        // Always ask the EL to validate forkchoice, even if the local heads already match.
        let response = update_forkchoice(self.engine.as_ref(), &new_sync_state).await?;
        if response.payload_status.status != PayloadStatusEnum::Valid {
            return Err(SynchronizeTaskError::UnexpectedPayloadStatus(
                response.payload_status.status,
            )
            .into());
        }

        if !state.el_sync_finished {
            info!(target: "engine", "Finished execution layer sync.");
            state.el_sync_finished = true;
        }
        state.sync_state = new_sync_state;
        self.block_sink.block_imported(block, block_info);
        info!(
            target: "engine",
            hash = %block_info.block_info.hash,
            number = block_info.block_info.number,
            "Canonicalized sequenced block"
        );
        Ok(())
    }
}

#[async_trait]
impl<C: EngineClient> EngineTaskExt for CanonicalizeTask<C> {
    type Output = ();
    type Error = CanonicalizeTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<(), CanonicalizeTaskError> {
        let head = state.sync_state.unsafe_head().block_info;
        let result = if head.hash != self.parent.block_info.hash ||
            head.number != self.parent.block_info.number
        {
            Err(CanonicalizeTaskError::UnsafeHeadChangedSinceBuild)
        } else {
            self.canonicalize(state)
                .await
                .map(|_| self.payload.clone())
                .map_err(|err| CanonicalizeTaskError::PayloadInsertionFailed(Box::new(err)))
        };
        if result.is_ok() &&
            let Some(tx) = &self.unsafe_head_tx
        {
            tx.send_replace(state.sync_state.unsafe_head());
        }
        // Completion belongs to accepted work even when its original caller disconnects.
        let _ = self.result_tx.send(result).await;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        InsertTask, NoopBlockSink, SynchronizeTask,
        test_utils::{MockEngineClient, TestEngineStateBuilder, test_block_info},
    };
    use alloy_rpc_types_engine::{
        ExecutionPayloadV1, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
    };
    use std::sync::atomic::{AtomicUsize, Ordering};

    #[derive(Debug, Default)]
    struct CountingBlockSink(AtomicUsize);

    impl ImportedBlockSink for CountingBlockSink {
        fn block_imported(&self, _: OpBlock, _: L2BlockInfo) {
            self.0.fetch_add(1, Ordering::Relaxed);
        }
    }

    fn fixture() -> (OpExecutionPayloadEnvelope, Arc<RollupConfig>) {
        let mut block = alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default();
        block.header.number = 11;
        block.header.base_fee_per_gas = Some(0);
        let payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(&block));
        let decoded: op_alloy_consensus::OpBlock = payload.clone().try_into_block().unwrap();
        let mut config = RollupConfig::default();
        config.genesis.l2.number = 11;
        config.genesis.l2.hash = decoded.header.hash_slow();
        (payload, Arc::new(config))
    }

    #[rstest::rstest]
    #[case::new_payload_syncing(true)]
    #[case::forkchoice_syncing(false)]
    #[tokio::test]
    async fn syncing_is_not_publication_and_retry_uses_the_committed_payload(
        #[case] new_payload_syncing: bool,
    ) {
        let (payload, config) = fixture();
        let status = |syncing| {
            PayloadStatus::from_status(if syncing {
                PayloadStatusEnum::Syncing
            } else {
                PayloadStatusEnum::Valid
            })
        };
        let client = Arc::new(
            MockEngineClient::builder()
                .with_config(config.clone())
                .with_new_payload_v1_response(status(new_payload_syncing))
                .with_fork_choice_updated_v3_response(ForkchoiceUpdated::new(status(
                    !new_payload_syncing,
                )))
                .build(),
        );
        let parent = test_block_info(10);
        let mut state = TestEngineStateBuilder::new()
            .with_unsafe_head(parent)
            .with_el_sync_finished(false)
            .build();
        let original = state;
        let (reply, mut replies) = mpsc::channel(1);
        let (unsafe_tx, unsafe_rx) = watch::channel(parent);
        let sink = Arc::new(CountingBlockSink::default());
        let task = CanonicalizeTask::new(
            client.clone(),
            config,
            payload.clone(),
            parent,
            reply,
            Some(unsafe_tx),
            sink.clone(),
        );
        task.execute(&mut state).await.unwrap();
        assert!(replies.recv().await.unwrap().is_err());
        assert_eq!(state, original);
        assert_eq!(*unsafe_rx.borrow(), parent);
        assert_eq!(sink.0.load(Ordering::Relaxed), 0);
        client.set_new_payload_v1_response(status(false)).await;
        client.set_fork_choice_updated_v3_response(ForkchoiceUpdated::new(status(false))).await;
        task.execute(&mut state).await.unwrap();
        // Publication precedes the response, and retries retain the exact envelope.
        assert_eq!(unsafe_rx.borrow().block_info.number, 11);
        assert_eq!(replies.recv().await.unwrap().unwrap(), payload);
        assert_eq!(sink.0.load(Ordering::Relaxed), 1);
        assert!(state.el_sync_finished);
        assert_eq!(state.sync_state.safe_head(), original.sync_state.safe_head());
        assert_eq!(state.sync_state.local_safe_head(), original.sync_state.local_safe_head());
        assert_eq!(state.sync_state.finalized_head(), original.sync_state.finalized_head());
    }

    #[tokio::test]
    async fn sync_insertion_accepts_syncing_without_finishing_el_sync() {
        let (payload, config) = fixture();
        let syncing = PayloadStatus::from_status(PayloadStatusEnum::Syncing);
        let client = Arc::new(
            MockEngineClient::builder()
                .with_config(config.clone())
                .with_new_payload_v1_response(syncing.clone())
                .with_fork_choice_updated_v3_response(ForkchoiceUpdated::new(syncing))
                .build(),
        );
        let mut state = TestEngineStateBuilder::new()
            .with_unsafe_head(test_block_info(10))
            .with_el_sync_finished(false)
            .build();
        let info = InsertTask::new(client, config, payload, false, Arc::new(NoopBlockSink))
            .execute(&mut state)
            .await
            .unwrap();
        assert_eq!(info.block_info.number, 11);
        assert_eq!(state.sync_state.unsafe_head(), info);
        assert!(!state.el_sync_finished);
    }

    #[tokio::test]
    async fn only_sync_may_skip_forkchoice_when_local_heads_match() {
        let (payload, config) = fixture();
        let block: OpBlock = payload.clone().try_into_block().unwrap();
        let head = L2BlockInfo::from_block_and_genesis(&block, &config.genesis).unwrap();
        let client = Arc::new(
            MockEngineClient::builder()
                .with_config(config.clone())
                .with_new_payload_v1_response(PayloadStatus::from_status(PayloadStatusEnum::Valid))
                .with_fork_choice_updated_v3_response(ForkchoiceUpdated::new(
                    PayloadStatus::from_status(PayloadStatusEnum::Syncing),
                ))
                .build(),
        );
        let mut state = TestEngineStateBuilder::new()
            .with_unsafe_head(head)
            .with_el_sync_finished(false)
            .build();
        SynchronizeTask::new(client.clone(), config.clone(), EngineSyncStateUpdate::default())
            .execute(&mut state)
            .await
            .unwrap();
        assert!(client.fork_choice_updated_v3_attributes().await.is_empty());

        let original = state;
        let (reply, _replies) = mpsc::channel(1);
        let sink = Arc::new(CountingBlockSink::default());
        let task =
            CanonicalizeTask::new(client.clone(), config, payload, head, reply, None, sink.clone());
        // A matching local target does not establish that the EL validated forkchoice.
        assert!(matches!(
            task.canonicalize(&mut state).await,
            Err(InsertTaskError::ForkchoiceUpdateFailed(
                SynchronizeTaskError::UnexpectedPayloadStatus(PayloadStatusEnum::Syncing)
            ))
        ));
        assert_eq!(client.fork_choice_updated_v3_attributes().await.len(), 1);
        assert_eq!(state, original);
        assert_eq!(sink.0.load(Ordering::Relaxed), 0);
    }

    #[tokio::test]
    async fn stale_payload_is_rejected_and_disconnected_callers_do_not_undo_import() {
        let (payload, config) = fixture();
        let valid = PayloadStatus::from_status(PayloadStatusEnum::Valid);
        let client = Arc::new(
            MockEngineClient::builder()
                .with_config(config.clone())
                .with_new_payload_v1_response(valid.clone())
                .with_fork_choice_updated_v3_response(ForkchoiceUpdated::new(valid))
                .build(),
        );
        let parent = test_block_info(10);
        let mut state = TestEngineStateBuilder::new().with_unsafe_head(parent).build();
        let (reply, mut replies) = mpsc::channel(1);
        let mut task = CanonicalizeTask::new(
            client,
            config,
            payload,
            test_block_info(9),
            reply,
            None,
            Arc::new(NoopBlockSink),
        );
        task.execute(&mut state).await.unwrap();
        assert!(matches!(
            replies.recv().await.unwrap(),
            Err(CanonicalizeTaskError::UnsafeHeadChangedSinceBuild)
        ));
        assert_eq!(state.sync_state.unsafe_head(), parent);
        task.parent = parent;
        drop(replies);
        task.execute(&mut state).await.unwrap();
        assert_eq!(state.sync_state.unsafe_head().block_info.number, 11);
    }
}
