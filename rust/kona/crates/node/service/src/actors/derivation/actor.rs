//! Derivation follows acknowledged engine state and resets its pipeline at epoch boundaries.

use crate::{
    DerivationActorRequest, EngineClientError, EngineContext, EngineHandle, Metrics, NodeActor,
    actors::derivation::L2Finalizer,
};
use async_trait::async_trait;
use kona_derive::{
    ActivationSignal, Pipeline, PipelineError, PipelineErrorKind, ResetError, ResetSignal, Signal,
    SignalReceiver, StepResult,
};
use kona_engine::{Consolidation, EnginePhase, EngineState, FinalizationSource};
use kona_protocol::{BlockInfo, OpAttributesWithParent};
use kona_rpc::DerivationStatus;
use std::time::Duration;
use thiserror::Error;
use tokio::{
    select,
    sync::{mpsc, watch},
    time::Instant,
};

/// Drives local derivation using the safe parent actually accepted by the engine.
#[derive(Debug)]
pub struct DerivationActor<P>
where
    P: Pipeline + SignalReceiver,
{
    inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
    engine: EngineHandle,
    engine_state: watch::Receiver<EngineState>,
    pipeline: P,
    status: watch::Sender<DerivationStatus>,
    /// Installed only after a successful pipeline reset or an accepted consolidation.
    applied_pipeline_context: Option<EngineContext>,
    observed_epoch: u64,
    pending_attributes: Option<OpAttributesWithParent>,
    pending_signal: Option<Signal>,
    deriving: bool,
    retry_at: Option<Instant>,
    /// Repeated local resets without acknowledged L2 progress are paced up to twelve seconds.
    reset_delay: Duration,
    finalized_l1: Option<BlockInfo>,
    finality_retry_at: Option<Instant>,
    pub(crate) finalizer: L2Finalizer,
}

impl<P: Pipeline + SignalReceiver> DerivationActor<P> {
    /// Creates a derivation actor. Initial engine readiness still requires a pipeline reset.
    pub fn new(
        engine: EngineHandle,
        inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
        pipeline: P,
        status: watch::Sender<DerivationStatus>,
    ) -> Self {
        let engine_state = engine.subscribe();
        let observed_epoch = engine.state().epoch;
        Self {
            inbound_request_rx,
            engine,
            engine_state,
            pipeline,
            status,
            applied_pipeline_context: None,
            observed_epoch,
            pending_attributes: None,
            pending_signal: None,
            deriving: false,
            retry_at: None,
            reset_delay: Duration::ZERO,
            finalized_l1: None,
            finality_retry_at: None,
            finalizer: L2Finalizer::default(),
        }
    }

    fn invalidate_pipeline(&mut self) {
        self.applied_pipeline_context = None;
        self.pending_attributes = None;
        self.pending_signal = None;
        self.retry_at = None;
        self.deriving = false;
        while self.pipeline.next().is_some() {}
        self.status.send_if_modified(|status| status.current_l1.take().is_some());
    }

    fn publish_status(&self) {
        let state = self.engine.state();
        let current_l1 = self
            .applied_pipeline_context
            .filter(|context| context.epoch == state.epoch && state.phase == EnginePhase::Ready)
            .and_then(|_| self.pipeline.origin());
        self.status.send_if_modified(|status| {
            std::mem::replace(&mut status.current_l1, current_l1) != current_l1
        });
    }

    fn observe_engine(&mut self) -> EngineState {
        let state = *self.engine_state.borrow_and_update();
        if state.epoch != self.observed_epoch || state.phase != EnginePhase::Ready {
            self.invalidate_pipeline();
            self.finalizer.clear();
            self.finality_retry_at = None;
            self.observed_epoch = state.epoch;
            self.reset_delay = Duration::ZERO;
        }
        state
    }

    fn retry(&mut self) {
        self.retry_at = Some(Instant::now() + Duration::from_millis(100));
    }

    fn receive_l1(
        &mut self,
        request: Option<DerivationActorRequest>,
    ) -> Result<(), DerivationError> {
        match request.ok_or(DerivationError::RequestReceiveFailed)? {
            DerivationActorRequest::ProcessFinalizedL1Block(block) => {
                self.finalized_l1 = Some(*block);
            }
            DerivationActorRequest::ProcessL1HeadUpdateRequest(_) => self.deriving = true,
        }
        Ok(())
    }

    /// A cancelled signal/step is never reused before another complete reset succeeds.
    async fn reset_pipeline(&mut self, state: EngineState) -> Result<(), DerivationError> {
        // A pipeline-local reset preserves accepted span progress in this engine epoch.
        let parent = state.pending_safe_head;
        self.reset_delay =
            (self.reset_delay * 2).clamp(Duration::from_secs(1), Duration::from_secs(12));
        let result = select! {
            biased;
            result = context_changed(&mut self.engine_state, state.epoch) => {
                result?;
                self.observe_engine();
                return Ok(());
            }
            result = self.pipeline.signal(Signal::Reset(ResetSignal { l2_safe_head: parent })) => result,
        };
        match result {
            Ok(()) => {
                // Pipeline stages reset independently of their prepared output queue.
                while self.pipeline.next().is_some() {}
                let latest = self.observe_engine();
                if latest.phase == EnginePhase::Ready &&
                    latest.epoch == state.epoch &&
                    latest.pending_safe_head == parent
                {
                    self.applied_pipeline_context =
                        Some(EngineContext { epoch: state.epoch, parent });
                    self.retry_at = None;
                    self.deriving = true;
                }
            }
            Err(error @ PipelineErrorKind::Critical(_)) => return Err(error.into()),
            Err(error) => {
                warn!(target: "derivation", %error, "Retrying pipeline reset");
                self.retry_at = Some(Instant::now() + self.reset_delay);
            }
        }
        self.publish_status();
        Ok(())
    }

    async fn signal_pipeline(
        &mut self,
        context: EngineContext,
        signal: Signal,
    ) -> Result<(), DerivationError> {
        let result = select! {
            biased;
            result = context_changed(&mut self.engine_state, context.epoch) => {
                result?;
                self.observe_engine();
                return Ok(());
            }
            result = self.pipeline.signal(signal) => result,
        };
        match result {
            Ok(()) => {
                while self.pipeline.next().is_some() {}
                self.pending_signal = None;
                self.deriving = true;
            }
            Err(error @ PipelineErrorKind::Critical(_)) => return Err(error.into()),
            Err(error) => {
                warn!(target: "derivation", %error, ?signal, "Retrying pipeline signal");
                self.retry();
            }
        }
        self.publish_status();
        Ok(())
    }

    async fn consolidate(
        &mut self,
        context: EngineContext,
        attributes: OpAttributesWithParent,
    ) -> Result<(), DerivationError> {
        let source = attributes.derived_from.ok_or(DerivationError::MissingSource)?;
        let previous_safe = self.engine.state().sync_state.safe_head();
        let result = select! {
            biased;
            result = context_changed(&mut self.engine_state, context.epoch) => {
                result?;
                self.observe_engine();
                return Ok(());
            }
            result = self.engine.consolidate(context, attributes) => result,
        };
        let latest = self.observe_engine();
        if latest.epoch != context.epoch || latest.phase != EnginePhase::Ready {
            return Ok(());
        }
        match result {
            Ok(result) => {
                self.pending_attributes = None;
                if result.state.pending_safe_head.block_info.number >
                    context.parent.block_info.number
                {
                    self.reset_delay = Duration::ZERO;
                }
                // Pending progress may advance within a span, or rewind when a legacy span
                // is dropped. Only completed safe progress can enter the finality map.
                self.applied_pipeline_context = Some(EngineContext {
                    epoch: context.epoch,
                    parent: result.state.pending_safe_head,
                });
                let safe = result.state.sync_state.safe_head();
                if result.outcome != Consolidation::Dropped &&
                    safe.block_info.number > previous_safe.block_info.number
                {
                    self.finalizer.record(safe, source);
                }
                if result.outcome == Consolidation::Replaced {
                    self.pending_signal = Some(Signal::FlushChannel);
                }
                self.deriving = true;
            }
            Err(EngineClientError::Temporary(error)) => {
                debug!(target: "derivation", %error, "Retrying derived attributes");
                self.retry();
            }
            Err(EngineClientError::Stale) => self.invalidate_pipeline(),
            Err(error) => return Err(error.into()),
        }
        Ok(())
    }

    async fn finalize(&mut self, context: EngineContext) -> Result<(), DerivationError> {
        let Some(finalized_l1) = self.finalized_l1 else { return Ok(()) };
        let Some(record) = self.finalizer.candidate(finalized_l1.number) else { return Ok(()) };
        let result = select! {
            biased;
            result = context_changed(&mut self.engine_state, context.epoch) => {
                result?;
                self.observe_engine();
                return Ok(());
            }
            result = self.engine.finalize(context.epoch, record.block.block_info.id(), FinalizationSource::Local(record.source.id())) => result,
        };
        let state = self.observe_engine();
        if state.epoch != context.epoch || state.phase != EnginePhase::Ready {
            return Ok(());
        }
        match result {
            Ok(()) => {
                self.finalizer.acknowledge(record);
                self.finality_retry_at = None;
            }
            Err(EngineClientError::Temporary(error)) => {
                debug!(target: "derivation", %error, "Retrying finalization");
                self.finality_retry_at = Some(Instant::now() + Duration::from_millis(100));
            }
            Err(EngineClientError::Stale) => self.invalidate_pipeline(),
            Err(error) => return Err(error.into()),
        }
        Ok(())
    }

    async fn advance(&mut self, context: EngineContext) -> Result<(), DerivationError> {
        let (result, attributes) = select! {
            biased;
            result = context_changed(&mut self.engine_state, context.epoch) => {
                result?;
                self.observe_engine();
                return Ok(());
            }
            result = async {
                let result = self.pipeline.step(context.parent).await;
                // No await separates the completed step from consuming its prepared output.
                (result, self.pipeline.next())
            } => result,
        };
        match result {
            StepResult::PreparedAttributes => self.pending_attributes = attributes,
            StepResult::AdvancedOrigin => {
                let origin = self.pipeline.origin().ok_or(PipelineError::MissingOrigin.crit())?;
                kona_macros::set!(counter, Metrics::DERIVATION_L1_ORIGIN, origin.number);
            }
            StepResult::OriginAdvanceErr(error) | StepResult::StepFailed(error) => match error {
                PipelineErrorKind::Temporary(PipelineError::NotEnoughData) => {}
                PipelineErrorKind::Temporary(_) => self.deriving = false,
                PipelineErrorKind::Reset(ResetError::HoloceneActivation) => {
                    self.pending_signal =
                        Some(Signal::Activation(ActivationSignal { l2_safe_head: context.parent }));
                }
                PipelineErrorKind::Reset(error) => {
                    if matches!(error, ResetError::ReorgDetected(..)) {
                        kona_macros::inc!(counter, Metrics::L1_REORG_COUNT);
                    }
                    self.invalidate_pipeline();
                    // The engine independently observes canonicality. Restart only this pipeline;
                    // repeating a reset without accepted L2 progress must not form a busy loop.
                    if !self.reset_delay.is_zero() {
                        self.retry_at = Some(Instant::now() + self.reset_delay);
                    }
                }
                error @ PipelineErrorKind::Critical(_) => {
                    kona_macros::inc!(counter, Metrics::DERIVATION_CRITICAL_ERROR);
                    return Err(error.into());
                }
            },
        }
        self.publish_status();
        Ok(())
    }
}

/// Only epoch/readiness changes invalidate work; ordinary unsafe-head publications do not.
pub(super) async fn context_changed(
    state: &mut watch::Receiver<EngineState>,
    epoch: u64,
) -> Result<(), DerivationError> {
    state
        .wait_for(|state| state.epoch != epoch || state.phase != EnginePhase::Ready)
        .await
        .map(|_| ())
        .map_err(|_| DerivationError::Engine(EngineClientError::Closed))
}

#[async_trait]
impl<P: Pipeline + SignalReceiver + Send + Sync + 'static> NodeActor for DerivationActor<P> {
    type Error = DerivationError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        let state = self.observe_engine();
        match self.inbound_request_rx.try_recv() {
            Ok(request) => self.receive_l1(Some(request))?,
            Err(mpsc::error::TryRecvError::Disconnected) => {
                return Err(DerivationError::RequestReceiveFailed);
            }
            Err(mpsc::error::TryRecvError::Empty) => {}
        }
        if let Some(deadline) = self.retry_at {
            if deadline <= Instant::now() {
                self.retry_at = None;
                return Ok(());
            }
            select! {
                biased;
                result = self.engine_state.changed() => {
                    result.map_err(|_| EngineClientError::Closed)?;
                    self.observe_engine();
                },
                request = self.inbound_request_rx.recv() => self.receive_l1(request)?,
                _ = tokio::time::sleep_until(deadline) => self.retry_at = None,
            }
            return Ok(());
        }
        if state.phase == EnginePhase::Ready {
            let Some(context) = self.applied_pipeline_context else {
                return self.reset_pipeline(state).await;
            };
            if let Some(signal) = self.pending_signal {
                return self.signal_pipeline(context, signal).await;
            }
            if let Some(attributes) = self.pending_attributes.clone() {
                return self.consolidate(context, attributes).await;
            }
            if self
                .finalized_l1
                .is_some_and(|block| self.finalizer.candidate(block.number).is_some()) &&
                self.finality_retry_at.is_none_or(|deadline| deadline <= Instant::now())
            {
                return self.finalize(context).await;
            }
            if self.deriving {
                return self.advance(context).await;
            }
        }
        let finality_deadline = self.finality_retry_at;
        select! {
            biased;
            result = self.engine_state.changed() => {
                    result.map_err(|_| EngineClientError::Closed)?;
                    self.observe_engine();
                },
            request = self.inbound_request_rx.recv() => self.receive_l1(request)?,
            _ = async {
                if let Some(deadline) = finality_deadline { tokio::time::sleep_until(deadline).await; }
                else { std::future::pending::<()>().await; }
            } => self.finality_retry_at = None,
        }
        Ok(())
    }
}

/// A terminal failure of the local or delegated derivation actor.
#[derive(Error, Debug)]
pub enum DerivationError {
    /// A critical pipeline failure.
    #[error(transparent)]
    Pipeline(#[from] PipelineErrorKind),
    /// The engine owner stopped or rejected an invariant.
    #[error(transparent)]
    Engine(#[from] EngineClientError),
    /// Local attributes must identify their real L1 inclusion source.
    #[error("Locally derived attributes have no L1 inclusion source")]
    MissingSource,
    /// The L1 notification service stopped.
    #[error("Derivation L1 request channel closed")]
    RequestReceiveFailed,
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{Consolidated, actors::engine::handle::Message};
    use alloy_primitives::B256;
    use kona_derive::{OriginProvider, PipelineResult};
    use kona_genesis::{RollupConfig, SystemConfig};
    use kona_protocol::L2BlockInfo;
    use std::{collections::VecDeque, sync::Arc};
    use tokio::sync::Notify;

    #[derive(Debug, Default)]
    struct TestPipeline {
        config: RollupConfig,
        prepared: VecDeque<OpAttributesWithParent>,
        signals: Vec<Signal>,
        signal_results: VecDeque<PipelineResult<()>>,
        steps: usize,
        parents: Vec<L2BlockInfo>,
        step_result: Option<StepResult>,
        blocked_step: Option<Arc<Notify>>,
        origin: Option<BlockInfo>,
        next_origin: Option<BlockInfo>,
    }

    impl Iterator for TestPipeline {
        type Item = OpAttributesWithParent;
        fn next(&mut self) -> Option<Self::Item> {
            self.prepared.pop_front()
        }
    }
    impl OriginProvider for TestPipeline {
        fn origin(&self) -> Option<BlockInfo> {
            self.origin
        }
    }
    #[async_trait]
    impl SignalReceiver for TestPipeline {
        async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
            self.signals.push(signal);
            self.signal_results.pop_front().unwrap_or(Ok(()))
        }
    }
    #[async_trait]
    impl Pipeline for TestPipeline {
        fn peek(&self) -> Option<&OpAttributesWithParent> {
            self.prepared.front()
        }
        async fn step(&mut self, parent: L2BlockInfo) -> StepResult {
            self.steps += 1;
            self.parents.push(parent);
            if let Some(origin) = self.next_origin.take() {
                self.origin = Some(origin);
            }
            if let Some(started) = &self.blocked_step {
                started.notify_one();
                std::future::pending::<()>().await;
            }
            self.step_result.take().unwrap_or(StepResult::StepFailed(PipelineError::Eof.temp()))
        }
        fn rollup_config(&self) -> &RollupConfig {
            &self.config
        }
        async fn system_config_by_l2_hash(
            &mut self,
            _: B256,
        ) -> Result<SystemConfig, PipelineErrorKind> {
            Ok(SystemConfig::default())
        }
    }

    fn ready(epoch: u64) -> EngineState {
        EngineState { epoch, phase: EnginePhase::Ready, ..Default::default() }
    }

    fn attributes(parent: L2BlockInfo) -> OpAttributesWithParent {
        OpAttributesWithParent {
            parent,
            derived_from: Some(BlockInfo {
                number: 4,
                hash: B256::repeat_byte(4),
                ..Default::default()
            }),
            attributes: Default::default(),
            is_last_in_span: true,
        }
    }

    #[tokio::test]
    async fn reset_installs_context_only_after_success_and_discards_prepared_output() {
        let (engine, _owner) = EngineHandle::test_channel(ready(1));
        let (_l1, requests) = mpsc::channel(1);
        let actual_origin = BlockInfo { number: 3, ..Default::default() };
        let (status, progress) = watch::channel(DerivationStatus::default());
        let pipeline = TestPipeline {
            origin: Some(actual_origin),
            prepared: VecDeque::from([attributes(L2BlockInfo::default())]),
            signal_results: VecDeque::from([Err(PipelineError::Eof.temp()), Ok(())]),
            ..Default::default()
        };
        let mut actor = DerivationActor::new(engine, requests, pipeline, status);
        actor.step().await.unwrap();
        assert_eq!(actor.applied_pipeline_context, None);
        assert_eq!(progress.borrow().current_l1, None);
        assert_eq!(actor.pipeline.steps, 0);
        assert!(actor.retry_at.is_some());
        actor.reset_pipeline(ready(1)).await.unwrap();
        assert_eq!(
            actor.applied_pipeline_context,
            Some(EngineContext { epoch: 1, parent: L2BlockInfo::default() })
        );
        assert!(actor.pipeline.prepared.is_empty());
        assert_eq!(progress.borrow().current_l1, Some(actual_origin));
        assert_eq!(actor.pipeline.signals.len(), 2);
    }

    #[tokio::test]
    async fn a_newer_epoch_supersedes_a_failed_pipeline_reset() {
        let (engine, owner) = EngineHandle::test_channel(ready(1));
        let (_l1, requests) = mpsc::channel(1);
        let pipeline = TestPipeline {
            signal_results: VecDeque::from([Err(PipelineError::Eof.temp()), Ok(())]),
            ..Default::default()
        };
        let mut actor = DerivationActor::new(
            engine,
            requests,
            pipeline,
            watch::channel(DerivationStatus::default()).0,
        );
        actor.step().await.unwrap();
        owner.state.send_replace(ready(2));
        actor.step().await.unwrap();
        assert_eq!(actor.applied_pipeline_context.unwrap().epoch, 2);
        assert_eq!(actor.pipeline.steps, 0);
    }

    #[tokio::test]
    async fn critical_pipeline_reset_failure_stops_derivation() {
        let (engine, _owner) = EngineHandle::test_channel(ready(1));
        let (_l1, requests) = mpsc::channel(1);
        let pipeline = TestPipeline {
            signal_results: VecDeque::from([Err(PipelineError::MissingOrigin.crit())]),
            ..Default::default()
        };
        let mut actor = DerivationActor::new(
            engine,
            requests,
            pipeline,
            watch::channel(DerivationStatus::default()).0,
        );
        assert!(matches!(
            actor.step().await,
            Err(DerivationError::Pipeline(PipelineErrorKind::Critical(_)))
        ));
        assert_eq!(actor.applied_pipeline_context, None);
    }

    #[rstest::rstest]
    #[case::applied(Consolidation::Applied)]
    #[case::replaced(Consolidation::Replaced)]
    #[case::dropped(Consolidation::Dropped)]
    #[tokio::test]
    async fn accepted_outcome_is_the_only_source_of_context_finality_and_flush(
        #[case] outcome: Consolidation,
    ) {
        let (engine, mut owner) = EngineHandle::test_channel(ready(1));
        let (_l1, requests) = mpsc::channel(1);
        let mut actor = DerivationActor::new(
            engine,
            requests,
            TestPipeline::default(),
            watch::channel(DerivationStatus::default()).0,
        );
        actor.step().await.unwrap();
        let context = actor.applied_pipeline_context.unwrap();
        let attrs = attributes(context.parent);
        let source = attrs.derived_from.unwrap();
        actor.pending_attributes = Some(attrs.clone());
        assert_eq!(actor.finalizer.candidate(4), None);
        let mut accepted = ready(1);
        if outcome != Consolidation::Dropped {
            let safe = L2BlockInfo {
                block_info: BlockInfo {
                    number: 7,
                    hash: B256::repeat_byte(7),
                    ..Default::default()
                },
                ..Default::default()
            };
            accepted.pending_safe_head = safe;
            accepted.sync_state =
                accepted.sync_state.apply_update(kona_engine::EngineSyncStateUpdate {
                    unsafe_head: Some(safe),
                    local_safe_head: Some(safe),
                    safe_head: Some(safe),
                    ..Default::default()
                });
        }
        let (result, ()) = tokio::join!(actor.step(), async {
            let Message::Consolidate(received_context, received_attrs, reply) =
                owner.commands.recv().await.unwrap()
            else {
                panic!("expected consolidation")
            };
            assert_eq!(received_context, context);
            assert_eq!(received_attrs, attrs);
            owner.state.send_replace(accepted);
            reply.send(Ok(Consolidated { state: accepted, outcome })).unwrap();
        });
        result.unwrap();
        assert_eq!(actor.pending_attributes, None);
        if outcome == Consolidation::Dropped {
            assert_eq!(actor.finalizer.candidate(4), None);
            assert_eq!(actor.applied_pipeline_context, Some(context));
            assert_eq!(actor.pending_signal, None);
            return;
        }
        assert_eq!(actor.applied_pipeline_context.unwrap().parent, accepted.sync_state.safe_head());
        assert_eq!(actor.reset_delay, Duration::ZERO);
        let record = actor.finalizer.candidate(4).unwrap();
        assert_eq!(record.block, accepted.sync_state.safe_head());
        assert_eq!(record.source, source);
        assert_eq!(
            actor.pending_signal,
            (outcome == Consolidation::Replaced).then_some(Signal::FlushChannel)
        );
        if outcome == Consolidation::Replaced {
            actor.pipeline.signal_results =
                VecDeque::from([Err(PipelineError::Eof.temp()), Ok(())]);
            actor.step().await.unwrap();
            assert_eq!(actor.pending_signal, Some(Signal::FlushChannel));
            assert_eq!(actor.pipeline.steps, 0);
            actor.retry_at = None;
            actor.step().await.unwrap();
            assert_eq!(actor.pending_signal, None);
            assert_eq!(actor.pipeline.steps, 0);
        }
    }

    #[tokio::test]
    async fn a_local_reset_preserves_pending_span_progress_and_drop_rewinds_without_finality() {
        let (engine, mut owner) = EngineHandle::test_channel(ready(1));
        let (_l1, requests) = mpsc::channel(1);
        let mut actor = DerivationActor::new(
            engine,
            requests,
            TestPipeline::default(),
            watch::channel(DerivationStatus::default()).0,
        );
        actor.step().await.unwrap();
        let completed = L2BlockInfo::default();
        let pending = L2BlockInfo {
            block_info: BlockInfo { number: 1, hash: B256::repeat_byte(1), ..Default::default() },
            ..Default::default()
        };
        for outcome in [Consolidation::Applied, Consolidation::Dropped] {
            let context = actor.applied_pipeline_context.unwrap();
            let mut attrs = attributes(context.parent);
            attrs.is_last_in_span = false;
            actor.pending_attributes = Some(attrs);
            let mut accepted = EngineState {
                pending_safe_head: if outcome == Consolidation::Applied {
                    pending
                } else {
                    completed
                },
                ..ready(1)
            };
            accepted.sync_state =
                accepted.sync_state.apply_update(kona_engine::EngineSyncStateUpdate {
                    unsafe_head: Some(pending),
                    ..Default::default()
                });
            let (result, ()) = tokio::join!(actor.step(), async {
                let Message::Consolidate(received, _, reply) = owner.commands.recv().await.unwrap()
                else {
                    panic!("expected consolidation")
                };
                assert_eq!(received, context);
                owner.state.send_replace(accepted);
                reply.send(Ok(Consolidated { state: accepted, outcome })).unwrap();
            });
            result.unwrap();
            assert_eq!(actor.applied_pipeline_context.unwrap().parent, accepted.pending_safe_head);
            assert_eq!(actor.finalizer.candidate(u64::MAX), None);
            assert_eq!(actor.pending_signal, None);
            if outcome == Consolidation::Applied {
                actor.pipeline.step_result = Some(StepResult::StepFailed(
                    ResetError::ReorgDetected(B256::ZERO, B256::repeat_byte(1)).reset(),
                ));
                actor.step().await.unwrap();
                assert_eq!(actor.applied_pipeline_context, None);
                actor.step().await.unwrap();
                assert_eq!(
                    actor.applied_pipeline_context,
                    Some(EngineContext { epoch: 1, parent: pending })
                );
                assert_eq!(
                    actor.pipeline.signals.last(),
                    Some(&Signal::Reset(ResetSignal { l2_safe_head: pending }))
                );
                assert_eq!(owner.state.borrow().sync_state.safe_head(), completed);
                assert_eq!(actor.finalizer.candidate(u64::MAX), None);
                assert!(owner.resets.try_recv().is_err());
            }
            assert_eq!(actor.pipeline.signals.len(), 2);
        }
        actor.step().await.unwrap();
        assert_eq!(actor.pipeline.parents, [pending, completed]);
        assert_eq!(actor.pipeline.signals.len(), 2);
    }

    #[tokio::test]
    async fn same_head_reset_discards_an_in_flight_reply_and_its_finality() {
        let (engine, mut owner) = EngineHandle::test_channel(ready(1));
        let (_l1, requests) = mpsc::channel(1);
        let mut actor = DerivationActor::new(
            engine,
            requests,
            TestPipeline::default(),
            watch::channel(DerivationStatus::default()).0,
        );
        actor.step().await.unwrap();
        let context = actor.applied_pipeline_context.unwrap();
        actor.pending_attributes = Some(attributes(context.parent));
        actor.finalizer.record(context.parent, BlockInfo::default());
        let (result, ()) = tokio::join!(actor.step(), async {
            let Message::Consolidate(_, _, reply) = owner.commands.recv().await.unwrap() else {
                panic!("expected consolidation")
            };
            owner.state.send_replace(EngineState { phase: EnginePhase::Resetting, ..ready(2) });
            let _ =
                reply.send(Ok(Consolidated { state: ready(1), outcome: Consolidation::Replaced }));
        });
        result.unwrap();
        assert_eq!(actor.applied_pipeline_context, None);
        assert_eq!(actor.pending_attributes, None);
        assert_eq!(actor.pending_signal, None);
        assert_eq!(actor.finalizer.candidate(u64::MAX), None);
        owner.state.send_replace(ready(2));
        actor.step().await.unwrap();
        assert_eq!(actor.applied_pipeline_context.unwrap().epoch, 2);
        assert_eq!(actor.pipeline.signals.len(), 2);
    }

    #[tokio::test]
    async fn reset_cancels_a_step_and_requires_a_successful_reset_before_reuse() {
        let (engine, owner) = EngineHandle::test_channel(ready(1));
        let (_l1, requests) = mpsc::channel(1);
        let started = Arc::new(Notify::new());
        let pipeline = TestPipeline { blocked_step: Some(started.clone()), ..Default::default() };
        let mut actor = DerivationActor::new(
            engine,
            requests,
            pipeline,
            watch::channel(DerivationStatus::default()).0,
        );
        actor.step().await.unwrap();
        let (result, ()) = tokio::join!(actor.step(), async {
            started.notified().await;
            owner.state.send_replace(EngineState { phase: EnginePhase::Resetting, ..ready(2) });
        });
        result.unwrap();
        assert_eq!(actor.applied_pipeline_context, None);
        assert_eq!(actor.pipeline.steps, 1);
        owner.state.send_replace(ready(2));
        actor.step().await.unwrap();
        assert_eq!(actor.pipeline.steps, 1);
        assert_eq!(actor.pipeline.signals.len(), 2);
    }

    #[tokio::test]
    async fn temporary_attributes_and_finalization_retain_the_original_work_until_acknowledged() {
        let (engine, mut owner) = EngineHandle::test_channel(ready(1));
        let (_l1, requests) = mpsc::channel(1);
        let mut actor = DerivationActor::new(
            engine,
            requests,
            TestPipeline::default(),
            watch::channel(DerivationStatus::default()).0,
        );
        actor.step().await.unwrap();
        let context = actor.applied_pipeline_context.unwrap();
        let attrs = attributes(context.parent);
        actor.pending_attributes = Some(attrs.clone());
        let mut accepted = ready(1);
        accepted.pending_safe_head = L2BlockInfo {
            block_info: BlockInfo { number: 1, hash: B256::repeat_byte(1), ..Default::default() },
            ..Default::default()
        };
        accepted.sync_state =
            accepted.sync_state.apply_update(kona_engine::EngineSyncStateUpdate {
                unsafe_head: Some(accepted.pending_safe_head),
                local_safe_head: Some(accepted.pending_safe_head),
                safe_head: Some(accepted.pending_safe_head),
                ..Default::default()
            });
        for attempt in 0..2 {
            let (result, ()) = tokio::join!(actor.consolidate(context, attrs.clone()), async {
                let Message::Consolidate(received, received_attrs, reply) =
                    owner.commands.recv().await.unwrap()
                else {
                    panic!("expected consolidation")
                };
                assert_eq!(received, context);
                assert_eq!(received_attrs, attrs);
                let result = if attempt == 0 {
                    Err(EngineClientError::Temporary("unavailable".into()))
                } else {
                    owner.state.send_replace(accepted);
                    Ok(Consolidated { state: accepted, outcome: Consolidation::Applied })
                };
                reply.send(result).unwrap();
            });
            result.unwrap();
            if attempt == 0 {
                assert_eq!(actor.pending_attributes, Some(attrs.clone()));
            }
        }
        actor.finalized_l1 = attrs.derived_from;
        let record = actor.finalizer.candidate(4).unwrap();
        for attempt in 0..2 {
            let (result, ()) = tokio::join!(actor.finalize(context), async {
                let Message::Finalize(epoch, block, source, reply) =
                    owner.commands.recv().await.unwrap()
                else {
                    panic!("expected finalization")
                };
                assert_eq!(epoch, 1);
                assert_eq!(block, record.block.block_info.id());
                assert_eq!(source, FinalizationSource::Local(record.source.id()));
                reply
                    .send(if attempt == 0 {
                        Err(EngineClientError::Temporary("unavailable".into()))
                    } else {
                        Ok(())
                    })
                    .unwrap();
            });
            result.unwrap();
            assert_eq!(actor.finalizer.candidate(4), (attempt == 0).then_some(record));
        }
    }

    #[tokio::test]
    async fn finalization_backoff_does_not_block_pipeline_progress() {
        let (engine, _owner) = EngineHandle::test_channel(ready(1));
        let (_l1, requests) = mpsc::channel(1);
        let mut actor = DerivationActor::new(
            engine,
            requests,
            TestPipeline::default(),
            watch::channel(DerivationStatus::default()).0,
        );
        actor.step().await.unwrap();
        actor.finalizer.record(L2BlockInfo::default(), BlockInfo::default());
        actor.finalized_l1 = Some(BlockInfo::default());
        actor.finality_retry_at = Some(Instant::now() + Duration::from_secs(60));
        actor.step().await.unwrap();
        assert_eq!(actor.pipeline.steps, 1);
        assert!(actor.finalizer.candidate(0).is_some());
    }

    #[tokio::test(start_paused = true)]
    async fn a_local_pipeline_reset_preserves_accepted_finality_and_its_retry() {
        let block = L2BlockInfo {
            block_info: BlockInfo { number: 7, hash: B256::repeat_byte(7), ..Default::default() },
            ..Default::default()
        };
        let source = BlockInfo { number: 4, hash: B256::repeat_byte(4), ..Default::default() };
        let mut state = ready(1);
        state.pending_safe_head = block;
        state.sync_state = state.sync_state.apply_update(kona_engine::EngineSyncStateUpdate {
            unsafe_head: Some(block),
            local_safe_head: Some(block),
            safe_head: Some(block),
            ..Default::default()
        });
        let (engine, mut owner) = EngineHandle::test_channel(state);
        let (_l1, requests) = mpsc::channel(1);
        let pipeline = TestPipeline {
            step_result: Some(StepResult::StepFailed(
                ResetError::ReorgDetected(B256::ZERO, B256::repeat_byte(1)).reset(),
            )),
            ..Default::default()
        };
        let mut actor = DerivationActor::new(
            engine,
            requests,
            pipeline,
            watch::channel(DerivationStatus::default()).0,
        );
        actor.step().await.unwrap();
        actor.finalizer.record(block, source);
        let record = actor.finalizer.candidate(source.number).unwrap();
        let retry = Instant::now() + Duration::from_secs(30);
        actor.finality_retry_at = Some(retry);
        actor.step().await.unwrap(); // A local reset invalidates pipeline context, not accepted safe.
        assert_eq!(actor.applied_pipeline_context, None);
        assert_eq!(actor.finalizer.candidate(source.number), Some(record));
        assert_eq!(actor.finality_retry_at, Some(retry));
        tokio::time::advance(Duration::from_secs(1)).await;
        actor.step().await.unwrap();
        actor.step().await.unwrap(); // Reinstall the same engine epoch's pipeline context.
        assert!(actor.applied_pipeline_context.is_some());
        actor.finalized_l1 = Some(source);
        tokio::time::advance(retry - Instant::now()).await;
        let (result, ()) = tokio::join!(actor.step(), async {
            let Message::Finalize(epoch, received, provenance, reply) =
                owner.commands.recv().await.unwrap()
            else {
                panic!("expected retained finalization")
            };
            assert_eq!(epoch, 1);
            assert_eq!(received, block.block_info.id());
            assert_eq!(provenance, FinalizationSource::Local(source.id()));
            reply.send(Ok(())).unwrap();
        });
        result.unwrap();
        assert_eq!(actor.finalizer.candidate(source.number), None);
    }

    #[tokio::test(start_paused = true)]
    async fn pipeline_resets_retry_locally_with_backoff_and_never_step_while_reset_failed() {
        let (engine, mut owner) = EngineHandle::test_channel(ready(1));
        let (_l1, requests) = mpsc::channel(1);
        let pipeline = TestPipeline {
            signal_results: (0..7).map(|_| Err(PipelineError::Eof.temp())).collect(),
            ..Default::default()
        };
        let mut actor = DerivationActor::new(
            engine,
            requests,
            pipeline,
            watch::channel(DerivationStatus::default()).0,
        );
        actor.step().await.unwrap();
        for delay in [1, 2, 4, 8, 12, 12] {
            let signals = actor.pipeline.signals.len();
            assert!(tokio::time::timeout(Duration::from_millis(500), actor.step()).await.is_err());
            assert_eq!(actor.pipeline.signals.len(), signals);
            let deadline = actor.retry_at.unwrap();
            assert_eq!(
                deadline - Instant::now(),
                Duration::from_secs(delay) - Duration::from_millis(500)
            );
            tokio::time::advance(deadline - Instant::now()).await;
            actor.step().await.unwrap(); // Consume the retry deadline before the next attempt.
            actor.step().await.unwrap();
            assert_eq!(actor.pipeline.steps, 0);
        }
        assert!(owner.resets.try_recv().is_err());
    }

    #[tokio::test(start_paused = true)]
    async fn a_pipeline_reorg_resets_only_the_pipeline_and_a_new_epoch_supersedes_its_backoff() {
        let (engine, mut owner) = EngineHandle::test_channel(ready(1));
        let (_l1, requests) = mpsc::channel(1);
        let pipeline = TestPipeline {
            step_result: Some(StepResult::StepFailed(
                ResetError::ReorgDetected(B256::ZERO, B256::repeat_byte(1)).reset(),
            )),
            ..Default::default()
        };
        let mut actor = DerivationActor::new(
            engine,
            requests,
            pipeline,
            watch::channel(DerivationStatus::default()).0,
        );
        actor.step().await.unwrap();
        actor.step().await.unwrap();
        assert_eq!(actor.applied_pipeline_context, None);
        assert_eq!(actor.retry_at.unwrap() - Instant::now(), Duration::from_secs(1));
        assert!(owner.resets.try_recv().is_err());
        owner.state.send_replace(ready(2));
        actor.step().await.unwrap();
        assert_eq!(actor.applied_pipeline_context.unwrap().epoch, 2);
        assert_eq!(actor.pipeline.signals.len(), 2);
    }

    #[tokio::test]
    async fn progress_reports_actual_pipeline_origin_even_when_safe_does_not_move() {
        let (engine, owner) = EngineHandle::test_channel(ready(1));
        let (_l1, requests) = mpsc::channel(1);
        let (status, current) = watch::channel(DerivationStatus::default());
        let reset_origin =
            BlockInfo { number: 3, hash: B256::repeat_byte(3), ..Default::default() };
        let next_origin = BlockInfo { number: 4, hash: B256::repeat_byte(4), ..Default::default() };
        let pipeline = TestPipeline {
            origin: Some(reset_origin),
            next_origin: Some(next_origin),
            ..Default::default()
        };
        let mut actor = DerivationActor::new(engine, requests, pipeline, status);
        actor.step().await.unwrap();
        assert_eq!(current.borrow().current_l1, Some(reset_origin));
        actor.step().await.unwrap(); // EOF still reports the origin reached by the step.
        assert_eq!(current.borrow().current_l1, Some(next_origin));
        assert_eq!(owner.state.borrow().sync_state.safe_head(), L2BlockInfo::default());
        owner.state.send_replace(EngineState { phase: EnginePhase::Resetting, ..ready(2) });
        actor.observe_engine();
        assert_eq!(current.borrow().current_l1, None);
    }
}
