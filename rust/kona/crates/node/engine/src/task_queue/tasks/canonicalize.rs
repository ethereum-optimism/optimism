//! Import a sequenced payload after committing it to the conductor.

use crate::{
    EngineClient, EngineState, EngineTaskError, EngineTaskExt, ImportedBlockSink, InsertTaskError,
    task_queue::{InsertTask, tasks::task::EngineTaskErrorSeverity},
};
use async_trait::async_trait;
use derive_more::Constructor;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
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
            InsertTask::new(
                self.engine.clone(),
                self.cfg.clone(),
                self.payload.clone(),
                false,
                self.block_sink.clone(),
            )
            .requiring_valid()
            .execute(state)
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
        NoopBlockSink,
        test_utils::{MockEngineClient, TestEngineStateBuilder, test_block_info},
    };
    use alloy_rpc_types_engine::{
        ExecutionPayloadV1, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
    };

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
        let mut state = TestEngineStateBuilder::new().with_unsafe_head(parent).build();
        let original = state;
        let (reply, mut replies) = mpsc::channel(1);
        let (unsafe_tx, unsafe_rx) = watch::channel(parent);
        let task = CanonicalizeTask::new(
            client.clone(),
            config,
            payload.clone(),
            parent,
            reply,
            Some(unsafe_tx),
            Arc::new(NoopBlockSink),
        );
        task.execute(&mut state).await.unwrap();
        assert!(replies.recv().await.unwrap().is_err());
        assert_eq!(state, original);
        assert_eq!(*unsafe_rx.borrow(), parent);
        client.set_new_payload_v1_response(status(false)).await;
        client.set_fork_choice_updated_v3_response(ForkchoiceUpdated::new(status(false))).await;
        task.execute(&mut state).await.unwrap();
        // Publication precedes the response, and retries retain the exact envelope.
        assert_eq!(unsafe_rx.borrow().block_info.number, 11);
        assert_eq!(replies.recv().await.unwrap().unwrap(), payload);
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
