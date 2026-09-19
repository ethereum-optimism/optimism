//! Tasks sent to the [`Engine`] for execution.
//!
//! [`Engine`]: crate::Engine

use super::{BuildTask, ConsolidateTask, FinalizeTask, InsertTask};
use crate::{
    BuildTaskError, ConsolidateTaskError, EngineClient, EngineState, FinalizeTaskError,
    InsertTaskError,
    task_queue::{SealTask, SealTaskError},
};
use alloy_rpc_types_engine::PayloadStatusEnum;
use async_trait::async_trait;
use derive_more::Display;
use std::{cmp::Ordering, time::Duration};
use thiserror::Error;
use tokio::{task::yield_now, time::sleep};

/// The delay before the second attempt at a task that keeps failing temporarily. Each further
/// attempt doubles it, up to [`MAX_RETRY_BACKOFF_SHIFT`] doublings.
const RETRY_BACKOFF_BASE: Duration = Duration::from_millis(10);

/// Caps the exponential backoff between retries of a temporarily failing task at
/// `RETRY_BACKOFF_BASE * 2^MAX_RETRY_BACKOFF_SHIFT`.
const MAX_RETRY_BACKOFF_SHIFT: u32 = 7;

/// The severity of an engine task error.
///
/// This is used to determine how to handle the error when draining the engine task queue.
#[derive(Debug, PartialEq, Eq, Display, Clone, Copy)]
pub enum EngineTaskErrorSeverity {
    /// The error is temporary and the task is retried.
    #[display("temporary")]
    Temporary,
    /// The error is critical and is propagated to the engine actor.
    #[display("critical")]
    Critical,
    /// The error indicates that the engine should be reset.
    #[display("reset")]
    Reset,
    /// The error indicates that the engine should be flushed.
    #[display("flush")]
    Flush,
}

/// The interface for an engine task error.
///
/// An engine task error should have an associated severity level to specify how to handle the error
/// when draining the engine task queue.
pub trait EngineTaskError {
    /// The severity of the error.
    fn severity(&self) -> EngineTaskErrorSeverity;
}

/// The interface for an engine task.
#[async_trait]
pub trait EngineTaskExt {
    /// The output type of the task.
    type Output;

    /// The error type of the task.
    type Error: EngineTaskError;

    /// Executes the task, taking a shared lock on the engine state and `self`.
    async fn execute(&self, state: &mut EngineState) -> Result<Self::Output, Self::Error>;
}

/// An error that may occur during an [`EngineTask`]'s execution.
#[derive(Error, Debug)]
pub enum EngineTaskErrors {
    /// An error that occurred while inserting a block into the engine.
    #[error(transparent)]
    Insert(#[from] InsertTaskError),
    /// An error that occurred while building a block.
    #[error(transparent)]
    Build(#[from] BuildTaskError),
    /// An error that occurred while sealing a block.
    #[error(transparent)]
    Seal(#[from] SealTaskError),
    /// An error that occurred while consolidating the engine state.
    #[error(transparent)]
    Consolidate(#[from] ConsolidateTaskError),
    /// An error that occurred while finalizing an L2 block.
    #[error(transparent)]
    Finalize(#[from] FinalizeTaskError),
}

impl EngineTaskError for EngineTaskErrors {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::Insert(inner) => inner.severity(),
            Self::Build(inner) => inner.severity(),
            Self::Seal(inner) => inner.severity(),
            Self::Consolidate(inner) => inner.severity(),
            Self::Finalize(inner) => inner.severity(),
        }
    }
}

/// Tasks that may be inserted into and executed by the [`Engine`].
///
/// [`Engine`]: crate::Engine
#[derive(Debug, Clone)]
pub enum EngineTask<EngineClient_: EngineClient> {
    /// Inserts an unsafe payload into the execution engine.
    Insert(Box<InsertTask<EngineClient_>>),
    /// Begins building a new block with the given attributes, producing a new payload ID.
    Build(Box<BuildTask<EngineClient_>>),
    /// Seals the block with the given payload ID and attributes, inserting it into the execution
    /// engine.
    Seal(Box<SealTask<EngineClient_>>),
    /// Performs consolidation on the engine state, reverting to payload attribute processing
    /// via the [`BuildTask`] if consolidation fails.
    Consolidate(Box<ConsolidateTask<EngineClient_>>),
    /// Finalizes an L2 block
    Finalize(Box<FinalizeTask<EngineClient_>>),
}

impl<EngineClient_: EngineClient> EngineTask<EngineClient_> {
    /// Executes the task without consuming it.
    async fn execute_inner(&self, state: &mut EngineState) -> Result<(), EngineTaskErrors> {
        match self {
            Self::Insert(task) => match task.execute(state).await {
                // INVALID is terminal for an externally sourced unsafe payload. Drop it so the
                // queue can process competing or subsequent payloads instead of retrying forever.
                Err(InsertTaskError::UnexpectedPayloadStatus(
                    status @ PayloadStatusEnum::Invalid { .. },
                )) => {
                    warn!(target: "engine", %status, "Dropping invalid unsafe payload");
                }
                Err(err) => return Err(err.into()),
                Ok(_) => {}
            },
            Self::Seal(task) => task.execute(state).await?,
            Self::Consolidate(task) => task.execute(state).await?,
            Self::Finalize(task) => task.execute(state).await?,
            Self::Build(task) => match task.execute(state).await {
                // The caller picked this parent from a snapshot of the unsafe head that has since
                // been reorged out. It has been told to rebuild, so drop the job instead of
                // retrying a forkchoice update the execution layer is guaranteed to reject.
                // Without a channel there is nobody to rebuild, which is a bug: let the error out.
                Err(BuildTaskError::UnsafeHeadChangedSinceBuild) if task.result_tx.is_some() => {
                    warn!(target: "engine", "Dropping stale block build job");
                }
                Err(err) => return Err(err.into()),
                Ok(_) => {}
            },
        };

        Ok(())
    }

    const fn task_metrics_label(&self) -> &'static str {
        match self {
            Self::Insert(_) => crate::Metrics::INSERT_TASK_LABEL,
            Self::Consolidate(_) => crate::Metrics::CONSOLIDATE_TASK_LABEL,
            Self::Build(_) => crate::Metrics::BUILD_TASK_LABEL,
            Self::Seal(_) => crate::Metrics::SEAL_TASK_LABEL,
            Self::Finalize(_) => crate::Metrics::FINALIZE_TASK_LABEL,
        }
    }
}

impl<EngineClient_: EngineClient> PartialEq for EngineTask<EngineClient_> {
    fn eq(&self, other: &Self) -> bool {
        matches!(
            (self, other),
            (Self::Insert(_), Self::Insert(_)) |
                (Self::Build(_), Self::Build(_)) |
                (Self::Seal(_), Self::Seal(_)) |
                (Self::Consolidate(_), Self::Consolidate(_)) |
                (Self::Finalize(_), Self::Finalize(_))
        )
    }
}

impl<EngineClient_: EngineClient> Eq for EngineTask<EngineClient_> {}

impl<EngineClient_: EngineClient> PartialOrd for EngineTask<EngineClient_> {
    fn partial_cmp(&self, other: &Self) -> Option<std::cmp::Ordering> {
        Some(self.cmp(other))
    }
}

impl<EngineClient_: EngineClient> Ord for EngineTask<EngineClient_> {
    fn cmp(&self, other: &Self) -> Ordering {
        // Order (descending): BuildBlock -> InsertUnsafe -> Consolidate -> Finalize
        //
        // https://specs.optimism.io/protocol/derivation.html#forkchoice-synchronization
        //
        // - Block building jobs are prioritized above all other tasks, to give priority to the
        //   sequencer. BuildTask handles forkchoice updates automatically.
        // - InsertUnsafe tasks are prioritized over Consolidate tasks, to ensure that unsafe block
        //   gossip is imported promptly.
        // - Consolidate tasks are prioritized over Finalize tasks, as they advance the safe chain
        //   via derivation.
        // - Finalize tasks have the lowest priority, as they only update finalized status.
        match (self, other) {
            // Same variant cases
            (Self::Insert(_), Self::Insert(_)) |
            (Self::Consolidate(_), Self::Consolidate(_)) |
            (Self::Build(_), Self::Build(_)) |
            (Self::Seal(_), Self::Seal(_)) |
            (Self::Finalize(_), Self::Finalize(_)) => Ordering::Equal,

            // SealBlock tasks are prioritized over all others
            (Self::Seal(_), _) => Ordering::Greater,
            (_, Self::Seal(_)) => Ordering::Less,

            // BuildBlock tasks are prioritized over InsertUnsafe and Consolidate tasks
            (Self::Build(_), _) => Ordering::Greater,
            (_, Self::Build(_)) => Ordering::Less,

            // InsertUnsafe tasks are prioritized over Consolidate and Finalize tasks
            (Self::Insert(_), _) => Ordering::Greater,
            (_, Self::Insert(_)) => Ordering::Less,

            // Consolidate tasks are prioritized over Finalize tasks
            (Self::Consolidate(_), _) => Ordering::Greater,
            (_, Self::Consolidate(_)) => Ordering::Less,
        }
    }
}

#[async_trait]
impl<EngineClient_: EngineClient> EngineTaskExt for EngineTask<EngineClient_> {
    type Output = ();

    type Error = EngineTaskErrors;

    async fn execute(&self, state: &mut EngineState) -> Result<(), Self::Error> {
        let mut temporary_failures = 0u32;

        // Retry the task until it succeeds or a critical error occurs.
        while let Err(e) = self.execute_inner(state).await {
            let severity = e.severity();

            kona_macros::inc!(
                counter,
                crate::Metrics::ENGINE_TASK_FAILURE,
                self.task_metrics_label() => severity.to_string()
            );

            match severity {
                EngineTaskErrorSeverity::Temporary => {
                    trace!(target: "engine", temporary_failures, "{e}");

                    // Yield the task to allow other tasks to execute to avoid starvation. An
                    // engine API call that keeps failing - an execution layer that is restarting,
                    // say - would otherwise spin this loop as fast as the transport can answer,
                    // so back off once the first retry has not cleared it.
                    match temporary_failures {
                        0 => yield_now().await,
                        n => {
                            sleep(RETRY_BACKOFF_BASE * 2u32.pow(n.min(MAX_RETRY_BACKOFF_SHIFT)))
                                .await
                        }
                    }
                    temporary_failures += 1;
                }
                EngineTaskErrorSeverity::Critical => {
                    error!(target: "engine", "{e}");
                    return Err(e);
                }
                EngineTaskErrorSeverity::Reset => {
                    warn!(target: "engine", "Engine requested derivation reset");
                    return Err(e);
                }
                EngineTaskErrorSeverity::Flush => {
                    warn!(target: "engine", "Engine requested derivation flush");
                    return Err(e);
                }
            }
        }

        kona_macros::inc!(counter, crate::Metrics::ENGINE_TASK_SUCCESS, self.task_metrics_label());

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        BuildSealCoupling, EngineState,
        test_utils::{
            MockEngineClient, TestAttributesBuilder, TestEngineStateBuilder, test_block_info,
            test_engine_client_builder,
        },
    };
    use alloy_consensus::Block;
    use alloy_json_rpc::ErrorPayload;
    use alloy_primitives::Bytes;
    use alloy_rpc_types_engine::{
        ExecutionPayloadV1, INVALID_FORK_CHOICE_STATE_ERROR, PayloadStatus,
    };
    use kona_genesis::RollupConfig;
    use op_alloy_consensus::OpTxEnvelope;
    use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
    use std::{sync::Arc, time::Duration};

    /// Records the blocks the engine hands over after a successful import.
    #[derive(Debug, Default)]
    struct RecordingSink(std::sync::Mutex<Vec<(alloy_primitives::B256, u64)>>);

    impl crate::ImportedBlockSink for RecordingSink {
        fn block_imported(
            &self,
            block: op_alloy_consensus::OpBlock,
            info: kona_protocol::L2BlockInfo,
        ) {
            self.0.lock().unwrap().push((info.block_info.hash, block.header.number));
        }
    }

    #[tokio::test]
    async fn imported_blocks_are_handed_to_the_block_sink() {
        let payload = ExecutionPayloadV1::from_block_slow(&Block::<OpTxEnvelope>::default());
        let envelope = OpExecutionPayloadEnvelope::V1(payload);
        // Pin genesis to this block so the L2BlockInfo can be built without an L1-info deposit.
        // The engine hashes the block it reconstructs from the payload, so key off that.
        let imported: op_alloy_consensus::OpBlock =
            envelope.clone().try_into_block().expect("payload converts to a block");
        let imported_hash = imported.header.hash_slow();
        let config = Arc::new(RollupConfig {
            genesis: kona_genesis::ChainGenesis {
                l2: alloy_eips::BlockNumHash { hash: imported_hash, number: 0 },
                ..Default::default()
            },
            ..Default::default()
        });
        let valid = || PayloadStatus::from_status(PayloadStatusEnum::Valid);
        let client = Arc::new(
            MockEngineClient::builder()
                .with_config(config.clone())
                .with_new_payload_v1_response(valid())
                .with_fork_choice_updated_v2_response(
                    alloy_rpc_types_engine::ForkchoiceUpdated::new(valid()),
                )
                .with_fork_choice_updated_v3_response(
                    alloy_rpc_types_engine::ForkchoiceUpdated::new(valid()),
                )
                .build(),
        );

        let sink = Arc::new(RecordingSink::default());
        let task = EngineTask::Insert(Box::new(InsertTask::new(
            client,
            config,
            envelope,
            false,
            sink.clone(),
        )));

        task.execute(&mut EngineState::default()).await.unwrap();

        assert_eq!(
            sink.0.lock().unwrap().as_slice(),
            &[(imported_hash, 0)],
            "a successfully imported block must reach the sink"
        );
    }

    #[tokio::test]
    async fn invalid_unsafe_payload_completes_without_retry() {
        let config = Arc::new(RollupConfig::default());
        let client = Arc::new(
            MockEngineClient::builder()
                .with_config(config.clone())
                .with_new_payload_v1_response(PayloadStatus::from_status(
                    PayloadStatusEnum::Invalid { validation_error: "invalid transaction".into() },
                ))
                .build(),
        );
        let mut payload = ExecutionPayloadV1::from_block_slow(&Block::<OpTxEnvelope>::default());
        payload.transactions = vec![Bytes::from_static(&[0xff])];
        let task = EngineTask::Insert(Box::new(InsertTask::new(
            client,
            config,
            OpExecutionPayloadEnvelope::V1(payload),
            false,
            Arc::new(crate::NoopBlockSink),
        )));

        tokio::time::timeout(Duration::from_secs(1), task.execute(&mut EngineState::default()))
            .await
            .expect("invalid unsafe payload task should not retry")
            .unwrap();
    }

    #[tokio::test]
    async fn build_with_invalid_forkchoice_state_does_not_retry() {
        // `-38002` means the engine considers the forkchoice state itself inconsistent: the safe
        // or finalized block is not an ancestor of the head. Re-sending it can never succeed, so
        // the queue must stop and ask for a reset rather than spin on the engine API.
        let parent = test_block_info(7);
        let client = Arc::new(
            test_engine_client_builder()
                .with_fork_choice_updated_error(ErrorPayload {
                    code: INVALID_FORK_CHOICE_STATE_ERROR as i64,
                    message: "invalid forkchoice state".into(),
                    data: None,
                })
                .build(),
        );
        let attributes = TestAttributesBuilder::new()
            .with_parent(parent)
            .with_timestamp(parent.block_info.timestamp)
            .build();
        let task = EngineTask::Build(Box::new(BuildTask::new(
            client,
            Arc::new(RollupConfig::default()),
            attributes,
            BuildSealCoupling::Atomic,
            None,
        )));
        let mut state = TestEngineStateBuilder::new().with_unsafe_head(parent).build();

        let err = tokio::time::timeout(Duration::from_secs(1), task.execute(&mut state))
            .await
            .expect("an invalid forkchoice state must not be retried")
            .expect_err("the task fails");
        assert_eq!(err.severity(), EngineTaskErrorSeverity::Reset);
    }

    #[tokio::test]
    async fn stale_build_completes_without_retry() {
        // The build was requested against an unsafe head that has since been reorged out. The
        // caller has been told to re-build, so the queue drops the job and moves on.
        // Same height, different hash: exactly the shape a force-included derived block takes.
        let stale_parent = test_block_info(42);
        let unsafe_head = test_block_info(42);

        // No forkchoice response is configured: the mock errors if the engine is called at all.
        let client = Arc::new(test_engine_client_builder().build());
        let attributes = TestAttributesBuilder::new()
            .with_parent(stale_parent)
            .with_timestamp(unsafe_head.block_info.timestamp)
            .build();
        let (tx, mut rx) = tokio::sync::mpsc::channel(1);
        let task = EngineTask::Build(Box::new(BuildTask::new(
            client,
            Arc::new(RollupConfig::default()),
            attributes,
            BuildSealCoupling::Detached,
            Some(tx),
        )));
        let mut state = TestEngineStateBuilder::new().with_unsafe_head(unsafe_head).build();

        tokio::time::timeout(Duration::from_secs(1), task.execute(&mut state))
            .await
            .expect("a stale build must not be retried")
            .expect("the queue drops the stale job instead of failing");
        assert!(matches!(rx.recv().await, Some(Err(BuildTaskError::UnsafeHeadChangedSinceBuild))));
    }
}
