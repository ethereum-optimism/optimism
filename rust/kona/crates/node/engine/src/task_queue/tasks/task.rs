//! Tasks sent to the [`Engine`] for execution.
//!
//! [`Engine`]: crate::Engine

use super::{
    BuildTask, CanonicalizeTask, CanonicalizeTaskError, ConsolidateTask, FinalizeTask, InsertTask,
};
use crate::{
    BuildTaskError, ConsolidateTaskError, EngineState, FinalizeTaskError, InsertTaskError,
    task_queue::{SealTask, SealTaskError},
};
use alloy_rpc_types_engine::PayloadStatusEnum;
use async_trait::async_trait;
use derive_more::Display;
use std::cmp::Ordering;
use thiserror::Error;

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
    /// An error that occurred while canonicalizing a sequenced block.
    #[error(transparent)]
    Canonicalize(#[from] CanonicalizeTaskError),
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
            Self::Canonicalize(inner) => inner.severity(),
            Self::Consolidate(inner) => inner.severity(),
            Self::Finalize(inner) => inner.severity(),
        }
    }
}

/// Tasks that may be inserted into and executed by the [`Engine`].
///
/// [`Engine`]: crate::Engine
#[derive(Debug, Clone)]
pub enum EngineTask {
    /// Inserts an unsafe payload into the execution engine.
    Insert(Box<InsertTask>),
    /// Begins building a new block with the given attributes, producing a new payload ID.
    Build(Box<BuildTask>),
    /// Fetches the block with the given payload ID and optionally imports it.
    Seal(Box<SealTask>),
    /// Imports a sequenced payload after its conductor commit has succeeded.
    Canonicalize(Box<CanonicalizeTask>),
    /// Performs consolidation on the engine state. If consolidation fails, a block is built from
    /// the payload attributes and imported instead.
    Consolidate(Box<ConsolidateTask>),
    /// Finalizes an L2 block
    Finalize(Box<FinalizeTask>),
}

impl EngineTask {
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
            Self::Canonicalize(task) => task.execute(state).await?,
            Self::Consolidate(task) => task.execute(state).await?,
            Self::Finalize(task) => task.execute(state).await?,
            Self::Build(task) => {
                task.execute(state).await?;
            }
        };

        Ok(())
    }

    const fn task_metrics_label(&self) -> &'static str {
        match self {
            Self::Insert(_) | Self::Canonicalize(_) => crate::Metrics::INSERT_TASK_LABEL,
            Self::Consolidate(_) => crate::Metrics::CONSOLIDATE_TASK_LABEL,
            Self::Build(_) => crate::Metrics::BUILD_TASK_LABEL,
            Self::Seal(_) => crate::Metrics::SEAL_TASK_LABEL,
            Self::Finalize(_) => crate::Metrics::FINALIZE_TASK_LABEL,
        }
    }
}

impl PartialEq for EngineTask {
    fn eq(&self, other: &Self) -> bool {
        matches!(
            (self, other),
            (Self::Insert(_), Self::Insert(_)) |
                (Self::Build(_), Self::Build(_)) |
                (Self::Seal(_), Self::Seal(_)) |
                (Self::Canonicalize(_), Self::Canonicalize(_)) |
                (Self::Consolidate(_), Self::Consolidate(_)) |
                (Self::Finalize(_), Self::Finalize(_))
        )
    }
}

impl Eq for EngineTask {}

impl PartialOrd for EngineTask {
    fn partial_cmp(&self, other: &Self) -> Option<std::cmp::Ordering> {
        Some(self.cmp(other))
    }
}

impl Ord for EngineTask {
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
            (Self::Canonicalize(_), Self::Canonicalize(_)) |
            (Self::Finalize(_), Self::Finalize(_)) => Ordering::Equal,

            // Finish importing a sequenced block before starting another build.
            (Self::Canonicalize(_), _) => Ordering::Greater,
            (_, Self::Canonicalize(_)) => Ordering::Less,

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
impl EngineTaskExt for EngineTask {
    type Output = ();

    type Error = EngineTaskErrors;

    async fn execute(&self, state: &mut EngineState) -> Result<(), Self::Error> {
        // The queue retains failed work. Its owner schedules the next attempt so a dependency
        // outage cannot monopolize the actor or turn into a tight retry loop.
        if let Err(e) = self.execute_inner(state).await {
            metrics::counter!(crate::Metrics::ENGINE_TASK_FAILURE, self.task_metrics_label() => e.severity().to_string()).increment(1);
            return Err(e);
        }

        metrics::counter!(crate::Metrics::ENGINE_TASK_SUCCESS, "type" => self.task_metrics_label())
            .increment(1);

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::test_engine_client;
    use alloy_consensus::Block;
    use alloy_primitives::Bytes;
    use alloy_rpc_types_engine::{ExecutionPayloadV1, PayloadStatus};
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
        let (client, l1, l2) = test_engine_client(config.clone());
        l2.expect("engine_newPayloadV1", valid());
        l2.expect(
            "engine_forkchoiceUpdatedV3",
            alloy_rpc_types_engine::ForkchoiceUpdated::new(valid()),
        );
        let client = Arc::new(client);

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
        l1.assert_finished();
        l2.assert_finished();
    }

    #[tokio::test]
    async fn invalid_unsafe_payload_completes_without_retry() {
        let config = Arc::new(RollupConfig::default());
        let (client, l1, l2) = test_engine_client(config.clone());
        l2.expect(
            "engine_newPayloadV1",
            PayloadStatus::from_status(PayloadStatusEnum::Invalid {
                validation_error: "invalid transaction".into(),
            }),
        );
        let client = Arc::new(client);
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
        l1.assert_finished();
        l2.assert_finished();
    }
}
