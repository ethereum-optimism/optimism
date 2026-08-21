//! Tasks sent to the [`Engine`] for execution.
//!
//! [`Engine`]: crate::Engine

use super::{BuildTask, ConsolidateTask, FinalizeTask, InsertTask, PromoteCrossSafeTask};
use crate::{
    BuildTaskError, ConsolidateTaskError, EngineClient, EngineState, FinalizeTaskError,
    InsertTaskError, PromoteCrossSafeTaskError,
    task_queue::{SealTask, SealTaskError},
};
use alloy_rpc_types_engine::PayloadStatusEnum;
use async_trait::async_trait;
use derive_more::Display;
use std::cmp::Ordering;
use thiserror::Error;
use tokio::task::yield_now;

/// The severity of an engine task error.
///
/// This is used to determine how to handle the error when draining the engine task queue.
#[derive(Debug, PartialEq, Eq, Display, Clone, Copy)]
pub enum EngineTaskErrorSeverity {
    /// The error is temporary and the task is retried.
    #[display("temporary")]
    Temporary,
    /// The error is critical and is propagated to the chain controller.
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
    /// An error that occurred while promoting the cross-safe head.
    #[error(transparent)]
    PromoteCrossSafe(#[from] PromoteCrossSafeTaskError),
}

impl EngineTaskError for EngineTaskErrors {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::Insert(inner) => inner.severity(),
            Self::Build(inner) => inner.severity(),
            Self::Seal(inner) => inner.severity(),
            Self::Consolidate(inner) => inner.severity(),
            Self::Finalize(inner) => inner.severity(),
            Self::PromoteCrossSafe(inner) => inner.severity(),
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
    /// Promotes the cross-safe head, moving the forkchoice `safeBlockHash`.
    PromoteCrossSafe(Box<PromoteCrossSafeTask<EngineClient_>>),
}

impl<EngineClient_: EngineClient> EngineTask<EngineClient_> {
    /// Executes the task without consuming it.
    async fn execute_inner(&self, state: &mut EngineState) -> Result<(), EngineTaskErrors> {
        match self {
            // A gossiped payload that does not descend from the local-safe head is terminal:
            // dropping it before the `engine_newPayload` round trip keeps the unsafe head from
            // rewinding under a head derived from L1. See
            // `InsertTask::descends_from_local_safe`.
            Self::Insert(task) if !task.descends_from_local_safe(state) => {
                warn!(
                    target: "engine",
                    number = task.payload_block_number(),
                    local_safe = state.sync_state.local_safe_head().block_info.number,
                    "Dropping unsafe payload that does not descend from the local-safe head"
                );
            }
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
            Self::PromoteCrossSafe(task) => task.execute(state).await?,
            Self::Build(task) => {
                task.execute(state).await?;
            }
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
            Self::PromoteCrossSafe(_) => crate::Metrics::PROMOTE_CROSS_SAFE_TASK_LABEL,
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
                (Self::Finalize(_), Self::Finalize(_)) |
                (Self::PromoteCrossSafe(_), Self::PromoteCrossSafe(_))
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
        // Order (descending): BuildBlock -> InsertUnsafe -> Consolidate -> PromoteCrossSafe ->
        // Finalize
        //
        // https://specs.optimism.io/protocol/derivation.html#forkchoice-synchronization
        //
        // - Block building jobs are prioritized above all other tasks, to give priority to the
        //   sequencer. BuildTask handles forkchoice updates automatically.
        // - InsertUnsafe tasks are prioritized over Consolidate tasks, to ensure that unsafe block
        //   gossip is imported promptly.
        // - Consolidate tasks are prioritized over PromoteCrossSafe tasks, as they advance the
        //   local-safe chain via derivation, which is what a promotion is about to ratify.
        // - Finalize tasks have the lowest priority, as they only update finalized status.
        match (self, other) {
            // Same variant cases
            (Self::Insert(_), Self::Insert(_)) |
            (Self::Consolidate(_), Self::Consolidate(_)) |
            (Self::Build(_), Self::Build(_)) |
            (Self::Seal(_), Self::Seal(_)) |
            (Self::PromoteCrossSafe(_), Self::PromoteCrossSafe(_)) |
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

            // Consolidate tasks are prioritized over PromoteCrossSafe and Finalize tasks
            (Self::Consolidate(_), _) => Ordering::Greater,
            (_, Self::Consolidate(_)) => Ordering::Less,

            // PromoteCrossSafe tasks are prioritized over Finalize tasks
            (Self::PromoteCrossSafe(_), _) => Ordering::Greater,
            (_, Self::PromoteCrossSafe(_)) => Ordering::Less,
        }
    }
}

#[async_trait]
impl<EngineClient_: EngineClient> EngineTaskExt for EngineTask<EngineClient_> {
    type Output = ();

    type Error = EngineTaskErrors;

    async fn execute(&self, state: &mut EngineState) -> Result<(), Self::Error> {
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
                    trace!(target: "engine", "{e}");

                    // Yield the task to allow other tasks to execute to avoid starvation.
                    yield_now().await;
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
    use crate::{EngineSyncStateUpdate, LocalSafeHead, LocalSafeOrigin, test_utils::MockEngineClient};
    use alloy_consensus::Block;
    use alloy_primitives::{B256, Bytes};
    use alloy_rpc_types_engine::{ExecutionPayloadV1, PayloadStatus};
    use kona_genesis::RollupConfig;
    use kona_protocol::{BlockInfo, L2BlockInfo};
    use op_alloy_consensus::OpTxEnvelope;
    use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
    use rstest::rstest;
    use std::{sync::Arc, time::Duration};

    fn l2_block(number: u64) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo {
                number,
                hash: B256::repeat_byte(number as u8),
                parent_hash: B256::repeat_byte(number.saturating_sub(1) as u8),
                timestamp: number * 2,
            },
            ..Default::default()
        }
    }

    /// A state whose unsafe and local-safe heads both sit at `head`.
    fn state_at(head: L2BlockInfo) -> EngineState {
        let mut state = EngineState::default();
        state.sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
            unsafe_head: Some(head),
            local_safe_head: Some(LocalSafeHead::unpaired(head)),
            ..Default::default()
        });
        state
    }

    fn payload_at(number: u64, parent_hash: B256) -> OpExecutionPayloadEnvelope {
        let mut payload = ExecutionPayloadV1::from_block_slow(&Block::<OpTxEnvelope>::default());
        payload.block_number = number;
        payload.parent_hash = parent_hash;
        OpExecutionPayloadEnvelope::V1(payload)
    }

    /// An insert task over a client that answers nothing: reaching the execution layer fails.
    fn insert_task(
        payload: OpExecutionPayloadEnvelope,
        is_derived: bool,
    ) -> InsertTask<MockEngineClient> {
        let config = Arc::new(RollupConfig::default());
        let client = Arc::new(MockEngineClient::builder().with_config(config.clone()).build());
        // These tests are about the unsafe-head admission check, which only reads whether the
        // payload is local-safe at all, so an unpaired origin stands in for a derived payload.
        InsertTask::new(client, config, payload, is_derived.then_some(LocalSafeOrigin::Unpaired))
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
            None,
        )));

        tokio::time::timeout(Duration::from_secs(1), task.execute(&mut EngineState::default()))
            .await
            .expect("invalid unsafe payload task should not retry")
            .unwrap();
    }

    /// A gossiped payload may only become the unsafe head if it descends from the local-safe head.
    /// Only the two decidable cases are rejected; anything further ahead is admitted and settled by
    /// derivation later.
    #[rstest]
    #[case::below_local_safe(7, 3, B256::repeat_byte(2), false)]
    #[case::at_local_safe(7, 7, B256::repeat_byte(6), false)]
    #[case::next_but_forked(7, 8, B256::repeat_byte(0xaa), false)]
    #[case::next_from_local_safe(7, 8, B256::repeat_byte(7), true)]
    #[case::two_ahead(7, 9, B256::repeat_byte(0xaa), true)]
    fn gossip_payload_must_descend_from_local_safe(
        #[case] local_safe: u64,
        #[case] payload_number: u64,
        #[case] payload_parent: B256,
        #[case] admitted: bool,
    ) {
        let state = state_at(l2_block(local_safe));
        let task = insert_task(payload_at(payload_number, payload_parent), false);
        assert_eq!(task.descends_from_local_safe(&state), admitted);
    }

    /// Before the engine has a local-safe head — the execution-layer sync bootstrap — there is
    /// nothing to descend from, so gossip is admitted whatever it claims.
    #[test]
    fn gossip_payload_admitted_without_a_local_safe_head() {
        let task = insert_task(payload_at(9, B256::repeat_byte(0xaa)), false);
        assert!(task.descends_from_local_safe(&EngineState::default()));
    }

    /// A derived payload is a local-safe write. It defines the head the check compares against
    /// rather than being checked by it, so it is admitted where a gossiped one would be rejected.
    #[test]
    fn derived_payload_is_not_subject_to_the_check() {
        let state = state_at(l2_block(7));
        let payload = payload_at(3, B256::repeat_byte(2));
        assert!(!insert_task(payload.clone(), false).descends_from_local_safe(&state));
        assert!(insert_task(payload, true).descends_from_local_safe(&state));
    }

    /// The rejection is terminal: the payload is dropped before the `engine_newPayload` round trip
    /// and the queue moves on. Were it retried, the client below — which answers nothing — would
    /// keep failing with a temporary error and the task would never complete.
    #[tokio::test]
    async fn rejected_unsafe_payload_completes_without_reaching_the_engine() {
        let mut state = state_at(l2_block(7));
        let task =
            EngineTask::Insert(Box::new(insert_task(payload_at(3, B256::repeat_byte(2)), false)));

        tokio::time::timeout(Duration::from_secs(1), task.execute(&mut state))
            .await
            .expect("rejected unsafe payload task should not retry")
            .unwrap();

        assert_eq!(state.sync_state.unsafe_head(), l2_block(7));
    }
}
