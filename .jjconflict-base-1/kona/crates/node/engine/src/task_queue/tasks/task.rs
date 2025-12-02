//! Tasks sent to the [`Engine`] for execution.
//!
//! [`Engine`]: crate::Engine

use super::{BuildTask, ConsolidateTask, FinalizeTask, InsertTask};
use crate::{
    BuildTaskError, ConsolidateTaskError, EngineClient, EngineState, FinalizeTaskError,
    InsertTaskError,
    task_queue::{SealTask, SealTaskError},
};
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
    /// Inserts a payload into the execution engine.
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
            Self::Insert(task) => task.execute(state).await?,
            Self::Seal(task) => task.execute(state).await?,
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
            (Self::Insert(_), Self::Insert(_)) => Ordering::Equal,
            (Self::Consolidate(_), Self::Consolidate(_)) => Ordering::Equal,
            (Self::Build(_), Self::Build(_)) => Ordering::Equal,
            (Self::Seal(_), Self::Seal(_)) => Ordering::Equal,
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

                    continue;
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
