//! Long-running L1 derivation and finality service.

use crate::{
    engine::{ENGINE_RETRY_DELAY, EngineClient, EngineError},
    safe_chain::{
        SafeChainControlError, SafeChainHandle, control::ResetRequest, finalizer::L2Finalizer,
    },
    unsafe_chain::SequencerHandle,
};
use kona_derive::{
    ActivationSignal, Pipeline, PipelineError, PipelineErrorKind, ResetError, ResetSignal, Signal,
    SignalReceiver, StepResult,
};
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use thiserror::Error;
use tokio::sync::{mpsc, watch};
use tokio_util::sync::CancellationToken;

#[cfg(feature = "metrics")]
use crate::Metrics;

/// Object-safe combination used to hide the concrete online derivation pipeline from composition.
pub trait DerivationPipeline: Pipeline + SignalReceiver + core::fmt::Debug + Send + Sync {}

impl<T> DerivationPipeline for T where T: Pipeline + SignalReceiver + core::fmt::Debug + Send + Sync {}

/// Owns normal in-process derivation, safe reconciliation, and L2 finality.
#[derive(Debug)]
pub struct SafeChainService {
    pipeline: Box<dyn DerivationPipeline>,
    engine: EngineClient,
    l1_head: watch::Receiver<Option<BlockInfo>>,
    l1_finalized: watch::Receiver<Option<BlockInfo>>,
    sequencer: Option<SequencerHandle>,
    reset_rx: mpsc::Receiver<ResetRequest>,
    finalizer: L2Finalizer,
    safe_head: L2BlockInfo,
}

impl SafeChainService {
    /// Creates a safe-chain service and derivation reset handle.
    pub fn new(
        pipeline: Box<dyn DerivationPipeline>,
        engine: EngineClient,
        l1_head: watch::Receiver<Option<BlockInfo>>,
        l1_finalized: watch::Receiver<Option<BlockInfo>>,
        sequencer: Option<SequencerHandle>,
    ) -> (Self, SafeChainHandle) {
        let (handle, reset_rx) = SafeChainHandle::channel();
        (
            Self {
                pipeline,
                engine,
                l1_head,
                l1_finalized,
                sequencer,
                reset_rx,
                finalizer: L2Finalizer::default(),
                safe_head: L2BlockInfo::default(),
            },
            handle,
        )
    }

    /// Runs derivation and finality until shutdown.
    pub async fn run(mut self, shutdown: CancellationToken) -> Result<(), SafeChainServiceError> {
        self.engine.wait_ready().await?;
        self.safe_head = self.engine.state().await?.safe_head();
        self.reset_pipeline().await?;

        loop {
            tokio::select! {
                biased;
                _ = shutdown.cancelled() => return Ok(()),
                request = self.reset_rx.recv() => {
                    let request = request.ok_or(SafeChainServiceError::ControlChannelClosed)?;
                    let result = self.reset_pipeline().await.map_err(|error| {
                        SafeChainControlError::Reset(error.to_string())
                    });
                    let _ = request.response.send(result);
                }
                changed = self.l1_finalized.changed() => {
                    changed.map_err(|_| SafeChainServiceError::L1ServiceStopped)?;
                    let finalized = *self.l1_finalized.borrow_and_update();
                    if let Some(finalized) = finalized {
                        self.apply_finality(finalized).await?;
                    }
                }
                changed = self.l1_head.changed() => {
                    changed.map_err(|_| SafeChainServiceError::L1ServiceStopped)?;
                    let head = *self.l1_head.borrow_and_update();
                    if let Some(head) = head {
                        debug!(target: "safe_chain", ?head, "Canonical L1 head advanced");
                        self.derive_available().await?;
                    }
                }
            }
        }
    }

    async fn derive_available(&mut self) -> Result<(), SafeChainServiceError> {
        loop {
            match self.pipeline.step(self.safe_head).await {
                StepResult::PreparedAttributes => {}
                StepResult::AdvancedOrigin => {
                    let origin = self.pipeline.origin().ok_or_else(|| {
                        SafeChainServiceError::Critical(PipelineError::MissingOrigin.to_string())
                    })?;
                    debug!(target: "safe_chain", number = origin.number, "Advanced derivation L1 origin");
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
    ) -> Result<bool, SafeChainServiceError> {
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
                    warn!(target: "safe_chain", ?expected, ?actual, "Canonical L1 reorg detected");
                    kona_macros::inc!(counter, Metrics::L1_REORG_COUNT);
                }
                self.reset_pipeline().await?;
                Ok(false)
            }
            PipelineErrorKind::Critical(error) => {
                kona_macros::inc!(counter, Metrics::DERIVATION_CRITICAL_ERROR);
                Err(SafeChainServiceError::Critical(error.to_string()))
            }
        }
    }

    async fn apply_safe(
        &mut self,
        attributes: OpAttributesWithParent,
    ) -> Result<(), SafeChainServiceError> {
        let derived_from = attributes.derived_from.ok_or_else(|| {
            SafeChainServiceError::Critical(
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
                Err(EngineError::FlushRequired(error)) => {
                    warn!(target: "safe_chain", %error, "Flushing derivation channel after invalid payload fallback");
                    self.pipeline.signal(Signal::FlushChannel).await?;
                    self.safe_head = self.engine.state().await?.safe_head();
                    self.finalizer.record(derived_from, self.safe_head);
                    return Ok(());
                }
                Err(EngineError::ResetRequired(error)) => {
                    warn!(target: "safe_chain", %error, "Engine requires recovery during safe reconciliation");
                    self.recover_engine().await?;
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
    ) -> Result<(), SafeChainServiceError> {
        let Some(finalized_l2) = self.finalizer.finalized_by(finalized_l1) else {
            return Ok(());
        };
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

    async fn recover_engine(&mut self) -> Result<(), SafeChainServiceError> {
        if self.sequencer.as_ref().is_some_and(SequencerHandle::is_active) {
            return Err(SafeChainServiceError::Engine(EngineError::RecoveryWhileSequencing));
        }
        self.safe_head = self.engine.recover().await?;
        Ok(())
    }

    async fn reset_pipeline(&mut self) -> Result<(), SafeChainServiceError> {
        self.finalizer.clear();
        self.pipeline.signal(Signal::Reset(ResetSignal { l2_safe_head: self.safe_head })).await?;
        Ok(())
    }
}

/// Terminal safe-chain service failure.
#[derive(Debug, Error)]
pub enum SafeChainServiceError {
    /// Semantic engine operation failed.
    #[error(transparent)]
    Engine(#[from] EngineError),
    /// Derivation pipeline failed.
    #[error(transparent)]
    Pipeline(#[from] PipelineErrorKind),
    /// A consensus-critical derivation invariant failed.
    #[error("critical derivation failure: {0}")]
    Critical(String),
    /// Shared L1 service stopped unexpectedly.
    #[error("L1 service stopped")]
    L1ServiceStopped,
    /// Every safe-chain control handle was dropped.
    #[error("safe-chain control channel closed")]
    ControlChannelClosed,
}
