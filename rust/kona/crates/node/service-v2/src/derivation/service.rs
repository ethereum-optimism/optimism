//! Top-level local L1 derivation and finality service.

use crate::{
    derivation::{
        DerivationAdminAdapter, DerivationControlError, control::ResetRequest,
        finalizer::L2Finalizer,
    },
    engine::{ENGINE_RETRY_DELAY, EngineError, EngineHandle},
    l1::{L1CursorStatus, L1Reader, L1Snapshot},
};
use kona_derive::{
    ActivationSignal, Pipeline, PipelineError, PipelineErrorKind, ResetError, ResetSignal, Signal,
    SignalReceiver, StepResult,
};
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use thiserror::Error;
use tokio::sync::{mpsc, oneshot, watch};

#[cfg(feature = "metrics")]
use crate::Metrics;

/// Object-safe combination used to hide the concrete online derivation pipeline.
pub trait DerivationPipeline: Pipeline + SignalReceiver + core::fmt::Debug + Send + Sync {}

impl<T> DerivationPipeline for T where T: Pipeline + SignalReceiver + core::fmt::Debug + Send + Sync {}

/// Owns local derivation, safe reconciliation, finality mapping, and pipeline reset.
#[derive(Debug)]
pub struct DerivationService {
    pipeline: Box<dyn DerivationPipeline>,
    engine: EngineHandle,
    l1: L1Reader,
    snapshots: watch::Receiver<L1Snapshot>,
    reset_rx: mpsc::Receiver<ResetRequest>,
    finalizer: L2Finalizer,
    safe_head: L2BlockInfo,
    finalized_l1: Option<BlockInfo>,
}

impl DerivationService {
    /// Creates local Derivation and its RPC-only administrative adapter.
    pub fn new(
        pipeline: Box<dyn DerivationPipeline>,
        engine: EngineHandle,
        initial_safe_head: L2BlockInfo,
        l1: L1Reader,
        snapshots: watch::Receiver<L1Snapshot>,
    ) -> (Self, DerivationAdminAdapter) {
        let (admin, reset_rx) = DerivationAdminAdapter::channel();
        (
            Self {
                pipeline,
                engine,
                l1,
                snapshots,
                reset_rx,
                finalizer: L2Finalizer::default(),
                safe_head: initial_safe_head,
                finalized_l1: None,
            },
            admin,
        )
    }

    /// Runs until the Node-owned lifecycle sender requests shutdown.
    pub async fn run(
        mut self,
        mut shutdown: oneshot::Receiver<()>,
    ) -> Result<(), DerivationServiceError> {
        self.reset_pipeline().await?;
        self.process_snapshot().await?;

        loop {
            tokio::select! {
                biased;
                _ = &mut shutdown => return Ok(()),
                request = self.reset_rx.recv() => {
                    if let Some(request) = request {
                        let result = self.reset_pipeline().await.map_err(|error| {
                            DerivationControlError::Reset(error.to_string())
                        });
                        let _ = request.response.send(result);
                    }
                }
                changed = self.snapshots.changed() => {
                    changed.map_err(|_| DerivationServiceError::L1WatcherStopped)?;
                    self.process_snapshot().await?;
                }
            }
        }
    }

    async fn process_snapshot(&mut self) -> Result<(), DerivationServiceError> {
        let snapshot = *self.snapshots.borrow_and_update();
        if snapshot.head.is_none() {
            return Ok(());
        }

        if let Some(cursor) = self.pipeline.origin() &&
            let L1CursorStatus::Reorg { previous, common_ancestor } =
                self.l1.validate_cursor(cursor).await?
        {
            warn!(target: "derivation", ?previous, ?common_ancestor, "Derivation L1 cursor was replaced; resetting pipeline");
            kona_macros::inc!(counter, Metrics::L1_REORG_COUNT);
            self.reset_pipeline().await?;
        }

        self.derive_available().await?;
        if let Some(finalized) = snapshot.finalized {
            self.apply_finality(finalized).await?;
        }
        Ok(())
    }

    async fn derive_available(&mut self) -> Result<(), DerivationServiceError> {
        loop {
            match self.pipeline.step(self.safe_head).await {
                StepResult::PreparedAttributes => {}
                StepResult::AdvancedOrigin => {
                    let origin = self.pipeline.origin().ok_or_else(|| {
                        DerivationServiceError::Critical(PipelineError::MissingOrigin.to_string())
                    })?;
                    debug!(target: "derivation", number = origin.number, "Advanced derivation L1 origin");
                    kona_macros::set!(counter, Metrics::DERIVATION_L1_ORIGIN, origin.number);
                }
                StepResult::OriginAdvanceErr(error) | StepResult::StepFailed(error) => {
                    if !self.handle_pipeline_error(error).await? {
                        return Ok(());
                    }
                }
            }

            while let Some(attributes) = self.pipeline.next() {
                self.apply_safe(attributes).await?;
            }
        }
    }

    /// Returns true when the pipeline may continue stepping immediately.
    async fn handle_pipeline_error(
        &mut self,
        error: PipelineErrorKind,
    ) -> Result<bool, DerivationServiceError> {
        match error {
            PipelineErrorKind::Temporary(PipelineError::NotEnoughData) => Ok(true),
            PipelineErrorKind::Temporary(_) => Ok(false),
            PipelineErrorKind::Reset(ResetError::HoloceneActivation) => {
                self.pipeline
                    .signal(Signal::Activation(ActivationSignal { l2_safe_head: self.safe_head }))
                    .await?;
                Ok(true)
            }
            PipelineErrorKind::Reset(error) => {
                if let ResetError::ReorgDetected(expected, actual) = error {
                    warn!(target: "derivation", ?expected, ?actual, "Canonical L1 reorg detected by pipeline");
                    kona_macros::inc!(counter, Metrics::L1_REORG_COUNT);
                }
                self.reset_pipeline().await?;
                Ok(false)
            }
            PipelineErrorKind::Critical(error) => {
                kona_macros::inc!(counter, Metrics::DERIVATION_CRITICAL_ERROR);
                Err(DerivationServiceError::Critical(error.to_string()))
            }
        }
    }

    async fn apply_safe(
        &mut self,
        attributes: OpAttributesWithParent,
    ) -> Result<(), DerivationServiceError> {
        let derived_from = attributes.derived_from.ok_or_else(|| {
            DerivationServiceError::Critical(
                "derived attributes did not identify their L1 source".to_string(),
            )
        })?;

        loop {
            match self.engine.update_safe(attributes.clone()).await {
                Ok(safe_head) => {
                    self.safe_head = safe_head;
                    self.finalizer.record(derived_from, safe_head);
                    return Ok(());
                }
                Err(EngineError::Temporary(_) | EngineError::ResponseDropped) => {
                    tokio::time::sleep(ENGINE_RETRY_DELAY).await;
                }
                Err(EngineError::FlushRequired { reason, safe_head }) => {
                    warn!(target: "derivation", %reason, "Flushing derivation channel after invalid payload fallback");
                    if let Some(safe_head) = safe_head {
                        self.safe_head = safe_head;
                    }
                    self.pipeline.signal(Signal::FlushChannel).await?;
                    return Ok(());
                }
                Err(EngineError::ResetRequired { reason, safe_head }) => {
                    warn!(target: "derivation", %reason, "Engine recovered authoritative state; resetting only derivation pipeline");
                    if let Some(safe_head) = safe_head {
                        self.safe_head = safe_head;
                    }
                    self.reset_pipeline().await?;
                    return Ok(());
                }
                Err(error) => return Err(error.into()),
            }
        }
    }

    async fn apply_finality(
        &mut self,
        finalized_l1: BlockInfo,
    ) -> Result<(), DerivationServiceError> {
        if let Some(previous) = self.finalized_l1 &&
            (finalized_l1.number < previous.number ||
                (finalized_l1.number == previous.number &&
                    finalized_l1.hash != previous.hash))
        {
            return Err(DerivationServiceError::FinalizedL1Changed { previous, finalized_l1 });
        }
        self.finalized_l1 = Some(finalized_l1);

        let Some((derived_from, finalized_l2)) = self.finalizer.finalized_by(finalized_l1) else {
            return Ok(());
        };
        let canonical_source = self.l1.block_by_number(derived_from.number).await?;
        if canonical_source.is_none_or(|canonical| canonical.hash != derived_from.hash) {
            warn!(target: "derivation", ?derived_from, "Discarding finality mapping from replaced L1 source");
            self.reset_pipeline().await?;
            return Ok(());
        }
        loop {
            match self.engine.update_finalized(finalized_l2).await {
                Ok(()) => return Ok(()),
                Err(EngineError::Temporary(_) | EngineError::ResponseDropped) => {
                    tokio::time::sleep(ENGINE_RETRY_DELAY).await;
                }
                Err(error) => return Err(error.into()),
            }
        }
    }

    async fn reset_pipeline(&mut self) -> Result<(), DerivationServiceError> {
        self.finalizer.clear();
        self.pipeline.signal(Signal::Reset(ResetSignal { l2_safe_head: self.safe_head })).await?;
        Ok(())
    }
}

/// Terminal Derivation failure.
#[derive(Debug, Error)]
pub enum DerivationServiceError {
    /// Semantic Engine operation failed.
    #[error(transparent)]
    Engine(#[from] EngineError),
    /// Derivation pipeline or direct L1 access failed.
    #[error(transparent)]
    Pipeline(#[from] PipelineErrorKind),
    /// A consensus-critical derivation invariant failed.
    #[error("critical derivation failure: {0}")]
    Critical(String),
    /// Shared canonical L1 observation stopped unexpectedly.
    #[error("L1 watcher stopped")]
    L1WatcherStopped,
    /// Finalized L1 changed after it was applied.
    #[error("finalized L1 changed from {previous:?} to {finalized_l1:?}")]
    FinalizedL1Changed {
        /// Previously accepted finalized L1 block.
        previous: BlockInfo,
        /// Conflicting or regressed finalized L1 block.
        finalized_l1: BlockInfo,
    },
}
