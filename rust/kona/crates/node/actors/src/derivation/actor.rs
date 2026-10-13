//! The derivation actor, which derives payload attributes from L1.

use super::{ActorError, Builder, DerivationStatus, Engine, L1, reset_engine};
use crate::{Metrics, engine::Consolidation};
use kona_derive::{
    ActivationSignal, Pipeline, PipelineError, PipelineErrorKind, ResetError, ResetSignal, Signal,
    SignalReceiver, StepResult,
};
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use std::future::Future;
use tokio::{
    sync::watch,
    time::{self, Duration},
};

/// How long to wait before trying again to reset the pipeline after a temporary failure, unless the
/// engine resets again first.
const PIPELINE_RESET_RETRY_INTERVAL: Duration = Duration::from_secs(1);

impl Builder {
    /// Supplies the dependencies of the derivation actor that derives payload attributes from L1,
    /// and produces its lifetime future without spawning it.
    ///
    /// The status reports the pipeline's origin from now on. Derivation waits on `l1` for a new
    /// head whenever the pipeline runs out of data. Runtime work begins when the future is polled.
    /// Dropping the future stops the actor.
    pub fn build<Engine_, PipelineSignalReceiver, L1_>(
        self,
        engine: Engine_,
        pipeline: PipelineSignalReceiver,
        l1: L1_,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static
    where
        Engine_: Engine + 'static,
        PipelineSignalReceiver: Pipeline + SignalReceiver + Send + Sync + 'static,
        L1_: L1 + 'static,
    {
        let Self { handle: _, status_tx } = self;
        let mut actor = Actor { engine, deriver: Deriver::new(pipeline, l1, status_tx) };
        async move { actor.run().await }
    }
}

/// Derives payload attributes from L1, and consolidates the engine's safe head with each.
///
/// It resets the engine once the execution layer is synced, and resets the pipeline onto the
/// engine's safe head whenever the engine resets.
#[derive(Debug)]
struct Actor<Engine_, PipelineSignalReceiver, L1_> {
    /// The engine, which derivation drives.
    engine: Engine_,
    /// The pipeline, stepped on the engine's safe head.
    deriver: Deriver<PipelineSignalReceiver, L1_>,
}

impl<Engine_, PipelineSignalReceiver, L1_> Actor<Engine_, PipelineSignalReceiver, L1_>
where
    Engine_: Engine,
    PipelineSignalReceiver: Pipeline + SignalReceiver + Send + Sync,
    L1_: L1,
{
    async fn run(&mut self) -> Result<(), ActorError> {
        reset_engine(&self.engine).await?;
        loop {
            self.step().await?;
        }
    }

    /// Resets the pipeline if the engine reset, or else derives the next attributes and
    /// consolidates the engine's safe head with them.
    ///
    /// An engine reset cancels a pipeline step in progress, which the pipeline reset then
    /// discards.
    async fn step(&mut self) -> Result<(), ActorError> {
        tokio::select! {
            biased;
            state = self.engine.next_reset() => {
                let safe_head = state?.safe_head();
                info!(target: "derivation", ?safe_head, "The engine reset; deriving from its safe head");
                self.reset_pipeline(safe_head).await
            }
            derived = self.deriver.derive() => match derived? {
                Derived::Attributes(attributes) => {
                    trace!(target: "derivation", ?attributes, "Produced payload attributes.");
                    let Consolidation { state, flush } = self.engine.consolidate(attributes).await?;
                    self.deriver.advance(state.safe_head(), flush).await;
                    Ok(())
                }
                Derived::ResetNeeded => reset_engine(&self.engine).await,
            },
        }
    }

    /// Resets the pipeline onto the safe head. A temporary failure is retried until the reset
    /// succeeds, onto the safe head of the latest engine reset. A pipeline that cannot reset onto
    /// the safe head, such as one whose ancestors an L1 reorg replaced, asks the engine to reset.
    async fn reset_pipeline(&mut self, mut safe_head: L2BlockInfo) -> Result<(), ActorError> {
        loop {
            match self.deriver.reset(safe_head).await {
                Ok(()) => return Ok(()),
                Err(PipelineErrorKind::Temporary(err)) => {
                    warn!(target: "derivation", ?err, "Failed to reset the pipeline; retrying");
                }
                Err(PipelineErrorKind::Reset(err)) => {
                    warn!(target: "derivation", ?err, "The pipeline cannot reset onto the safe head");
                    return reset_engine(&self.engine).await;
                }
                Err(err) => {
                    error!(target: "derivation", ?err, "Failed to reset the pipeline");
                    return Err(err.into());
                }
            }
            tokio::select! {
                biased;
                state = self.engine.next_reset() => safe_head = state?.safe_head(),
                () = time::sleep(PIPELINE_RESET_RETRY_INTERVAL) => {}
            }
        }
    }
}

/// What stepping the pipeline produced.
#[derive(Debug)]
#[allow(clippy::large_enum_variant)]
enum Derived {
    /// Attributes to consolidate.
    Attributes(OpAttributesWithParent),
    /// The pipeline needs a reset, such as after an L1 reorg.
    ResetNeeded,
}

/// The derivation pipeline, the safe head it derives on, and the L1 head it waits for.
#[derive(Debug)]
struct Deriver<PipelineSignalReceiver, L1_> {
    /// The derivation pipeline.
    pipeline: PipelineSignalReceiver,
    /// The engine's safe head, which the next attributes extend.
    safe_head: L2BlockInfo,
    /// The L1 chain, whose new heads wake derivation when the pipeline runs out of data.
    l1: L1_,
    /// Progress published to RPC readers.
    status: watch::Sender<DerivationStatus>,
}

impl<PipelineSignalReceiver, L1_> Deriver<PipelineSignalReceiver, L1_>
where
    PipelineSignalReceiver: Pipeline + SignalReceiver + Send + Sync,
    L1_: L1,
{
    /// Creates the deriver and publishes the pipeline's origin.
    fn new(
        pipeline: PipelineSignalReceiver,
        l1: L1_,
        status: watch::Sender<DerivationStatus>,
    ) -> Self {
        status.send_replace(DerivationStatus { current_l1: pipeline.origin() });
        Self { pipeline, safe_head: L2BlockInfo::default(), l1, status }
    }

    /// Steps the pipeline on the safe head until it produces attributes or needs a reset, waiting
    /// for the next L1 head whenever it runs out of data.
    ///
    /// Cancelling it can leave a step half done, so only a pipeline reset may follow. The pipeline
    /// queues the attributes a step prepares, which a reset does not clear, but they are taken
    /// right after the step, with no wait between, so the queue is empty whenever it is cancelled.
    async fn derive(&mut self) -> Result<Derived, ActorError> {
        // As we start the safe head at the disputed block's parent, we step the pipeline until the
        // first attributes are produced. All batches at and before the safe head will be
        // dropped, so the first payload will always be the disputed one.
        loop {
            let result = self.pipeline.step(self.safe_head).await;
            self.publish_status();
            match result {
                StepResult::PreparedAttributes => { /* continue; attributes will be sent off. */ }
                StepResult::AdvancedOrigin => {
                    let origin =
                        self.pipeline.origin().ok_or(PipelineError::MissingOrigin.crit())?.number;

                    metrics::counter!(Metrics::DERIVATION_L1_ORIGIN).absolute(origin);
                    debug!(target: "derivation", l1_block = origin, "Advanced L1 origin");
                }
                StepResult::OriginAdvanceErr(e) | StepResult::StepFailed(e) => match e {
                    // NotEnoughData is transient, and doesn't imply we need to wait for more data.
                    // We can continue stepping until we receive an Eof.
                    PipelineErrorKind::Temporary(PipelineError::NotEnoughData) => continue,
                    PipelineErrorKind::Temporary(_) => {
                        info!(target: "derivation", "Yielding derivation until more data is available.");
                        let head = self.l1.next_head().await.ok_or(ActorError::L1Stopped)?;
                        info!(target: "derivation", l1_head = ?head, "Processing l1 head update");
                    }
                    PipelineErrorKind::Reset(ResetError::HoloceneActivation) => {
                        info!(target: "derivation", "Activating Holocene in the pipeline");
                        self.pipeline
                            .signal(Signal::Activation(ActivationSignal {
                                l2_safe_head: self.safe_head,
                            }))
                            .await?;
                        self.publish_status();
                    }
                    PipelineErrorKind::Reset(ResetError::ReorgDetected(expected, new)) => {
                        warn!(target: "derivation", "L1 reorg detected! Expected: {expected} | New: {new}");
                        metrics::counter!(Metrics::L1_REORG_COUNT).increment(1);
                        return Ok(Derived::ResetNeeded);
                    }
                    PipelineErrorKind::Reset(e) => {
                        warn!(target: "derivation", "Derivation pipeline is being reset: {e}");
                        return Ok(Derived::ResetNeeded);
                    }
                    PipelineErrorKind::Critical(_) => {
                        error!(target: "derivation", "Critical derivation error: {e}");
                        metrics::counter!(Metrics::DERIVATION_CRITICAL_ERROR).increment(1);
                        return Err(e.into());
                    }
                },
            }

            // If there are any new attributes, send them to the execution actor.
            if let Some(attributes) = self.pipeline.next() {
                return Ok(Derived::Attributes(attributes));
            }
        }
    }

    /// Resets the pipeline onto the safe head, once.
    async fn reset(&mut self, safe_head: L2BlockInfo) -> Result<(), PipelineErrorKind> {
        self.safe_head = safe_head;
        let signal = Signal::Reset(ResetSignal { l2_safe_head: safe_head });
        let result = self.pipeline.signal(signal).await;
        self.publish_status();
        result.inspect(|()| info!(target: "derivation", ?signal, "[SIGNAL] Executed Successfully"))
    }

    /// Advances the safe head to the one the engine consolidated, after flushing the channel if a
    /// deposits-only block replaced the derived one.
    async fn advance(&mut self, safe_head: L2BlockInfo, flush: bool) {
        if flush {
            warn!(target: "derivation", "Flushing the channel of a rejected block");
            let result = self.pipeline.signal(Signal::FlushChannel).await;
            self.publish_status();
            if let Err(err) = result {
                error!(target: "derivation", ?err, "Failed to flush the pipeline's channel");
            }
        }
        self.safe_head = safe_head;
    }

    fn publish_status(&self) {
        let current_l1 = self.pipeline.origin();
        self.status.send_if_modified(|state| {
            std::mem::replace(&mut state.current_l1, current_l1) != current_l1
        });
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        EngineClientError,
        derivation::engine::{Request, TestEngine},
        engine::{EngineState, HeadsUpdate},
    };
    use alloy_primitives::B256;
    use async_trait::async_trait;
    use kona_derive::PipelineResult;
    use kona_genesis::{HardForkConfig, RollupConfig, SystemConfig};
    use kona_protocol::BlockInfo;
    use rstest::rstest;
    use std::{
        collections::VecDeque,
        future::{self, Future},
        sync::Arc,
    };

    /// A pipeline stub that follows the L1 head, prepares attributes, runs out of data, reports a
    /// reset, or hangs, and records the signals it receives and the safe heads it steps on.
    #[derive(Debug)]
    struct TestPipeline {
        rollup_config: Arc<RollupConfig>,
        origin: Option<BlockInfo>,
        /// The L1 head the origin advances to, one step at a time.
        l1_head: Option<watch::Receiver<BlockInfo>>,
        /// Reports an L1 reorg on the next step.
        reorg: bool,
        /// Reports Holocene's activation on the next step.
        holocene_activation: bool,
        /// Never finishes the next step.
        hang: bool,
        /// Errors the next reset signals fail with, in order.
        reset_errors: VecDeque<PipelineErrorKind>,
        prepared: VecDeque<OpAttributesWithParent>,
        /// The signals the pipeline applied.
        signals: Vec<Signal>,
        stepped_on: Vec<L2BlockInfo>,
        /// Whether the last step ran out of data.
        out_of_data: bool,
    }

    impl TestPipeline {
        fn new() -> Self {
            Self {
                rollup_config: Arc::new(RollupConfig::default()),
                origin: Some(BlockInfo::default()),
                l1_head: None,
                reorg: false,
                holocene_activation: false,
                hang: false,
                reset_errors: VecDeque::new(),
                prepared: VecDeque::new(),
                signals: Vec::new(),
                stepped_on: Vec::new(),
                out_of_data: false,
            }
        }

        /// Returns a pipeline that prepares the given attributes.
        fn preparing(attributes: impl IntoIterator<Item = OpAttributesWithParent>) -> Self {
            Self { prepared: attributes.into_iter().collect(), ..Self::new() }
        }
    }

    impl Iterator for TestPipeline {
        type Item = OpAttributesWithParent;

        fn next(&mut self) -> Option<Self::Item> {
            self.prepared.pop_front()
        }
    }

    impl kona_derive::OriginProvider for TestPipeline {
        fn origin(&self) -> Option<BlockInfo> {
            self.origin
        }
    }

    #[async_trait]
    impl SignalReceiver for TestPipeline {
        async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
            if let Signal::Reset(reset) = signal {
                if let Some(err) = self.reset_errors.pop_front() {
                    return Err(err);
                }
                self.origin = Some(BlockInfo {
                    number: reset.l2_safe_head.l1_origin.number,
                    hash: reset.l2_safe_head.l1_origin.hash,
                    ..Default::default()
                });
            }
            self.signals.push(signal);
            Ok(())
        }
    }

    #[async_trait]
    impl Pipeline for TestPipeline {
        fn peek(&self) -> Option<&OpAttributesWithParent> {
            self.prepared.front()
        }

        async fn step(&mut self, safe_head: L2BlockInfo) -> StepResult {
            self.stepped_on.push(safe_head);
            self.out_of_data = false;
            if std::mem::take(&mut self.hang) {
                return future::pending().await;
            }
            let l1_head = self.l1_head.as_ref().map(|head| *head.borrow());
            if std::mem::take(&mut self.reorg) {
                StepResult::StepFailed(PipelineErrorKind::Reset(ResetError::ReorgDetected(
                    B256::ZERO,
                    B256::repeat_byte(1),
                )))
            } else if std::mem::take(&mut self.holocene_activation) {
                StepResult::StepFailed(PipelineErrorKind::Reset(ResetError::HoloceneActivation))
            } else if l1_head.is_some() && l1_head != self.origin {
                self.origin = l1_head;
                StepResult::AdvancedOrigin
            } else if !self.prepared.is_empty() {
                StepResult::PreparedAttributes
            } else {
                self.out_of_data = true;
                StepResult::StepFailed(PipelineError::Eof.temp())
            }
        }

        fn rollup_config(&self) -> &RollupConfig {
            &self.rollup_config
        }

        async fn system_config_by_l2_hash(
            &mut self,
            _: B256,
        ) -> Result<SystemConfig, PipelineErrorKind> {
            Ok(SystemConfig::default())
        }
    }

    #[async_trait]
    impl L1 for watch::Receiver<BlockInfo> {
        async fn next_head(&mut self) -> Option<BlockInfo> {
            self.changed().await.ok()?;
            Some(*self.borrow_and_update())
        }
    }

    type TestActor = Actor<TestEngine, TestPipeline, watch::Receiver<BlockInfo>>;

    /// The L1 head publisher an actor reads from, and its status.
    struct Publishers {
        heads: watch::Sender<BlockInfo>,
        status: watch::Receiver<DerivationStatus>,
    }

    /// Creates an actor with the publishers it reads from.
    fn actor(engine: TestEngine, pipeline: TestPipeline) -> (TestActor, Publishers) {
        let (heads, head_rx) = watch::channel(BlockInfo::default());
        let (status_tx, status) = watch::channel(DerivationStatus::default());
        let actor = Actor { engine, deriver: Deriver::new(pipeline, head_rx, status_tx) };
        (actor, Publishers { heads, status })
    }

    /// Runs the actor while `script` runs, and returns the actor's error if it stops first.
    async fn run_during(
        actor: &mut TestActor,
        script: impl Future<Output = ()>,
    ) -> Option<ActorError> {
        tokio::select! {
            result = actor.run() => Some(result.unwrap_err()),
            () = script => None,
        }
    }

    /// Runs the actor until it waits for more L1 data, leaving time for any retries.
    async fn run_until_out_of_data(actor: &mut TestActor) {
        let wait = time::sleep(Duration::from_secs(60));
        assert!(run_during(actor, wait).await.is_none(), "the actor stopped");
        assert!(
            actor.deriver.pipeline.out_of_data,
            "derivation stalled before running out of data"
        );
    }

    /// Lets the actor handle what was published.
    async fn settle() {
        time::sleep(Duration::from_secs(1)).await;
    }

    /// Returns the L2 block with the given number.
    fn block(number: u64) -> L2BlockInfo {
        L2BlockInfo { block_info: BlockInfo { number, ..Default::default() }, ..Default::default() }
    }

    /// Returns a safe head whose L1 origin is the given L1 block.
    fn safe_head(l1_origin: BlockInfo) -> L2BlockInfo {
        L2BlockInfo { l1_origin: l1_origin.id(), ..Default::default() }
    }

    /// Returns an engine state whose safe head is the given block.
    fn engine_state(safe_head: L2BlockInfo) -> EngineState {
        EngineState::default()
            .apply_update(HeadsUpdate { safe_head: Some(safe_head), ..Default::default() })
    }

    /// Returns a consolidation reply without a flush.
    const fn consolidated(state: EngineState) -> Consolidation {
        Consolidation { state, flush: false }
    }

    /// Returns attributes derived from the default L1 block, whose parent is the given block.
    fn attributes(parent: L2BlockInfo) -> OpAttributesWithParent {
        OpAttributesWithParent::new(Default::default(), parent, Some(BlockInfo::default()), false)
    }

    /// Returns a reset signal onto the given safe head.
    const fn reset(l2_safe_head: L2BlockInfo) -> Signal {
        Signal::Reset(ResetSignal { l2_safe_head })
    }

    /// Building needs no runtime, and the handle's status shows the pipeline's origin as soon as
    /// the actor is built, before it runs.
    #[test]
    fn build_publishes_the_pipeline_origin_without_a_runtime() {
        let origin = BlockInfo { number: 5, ..Default::default() };
        let builder = Builder::new();
        let handle = builder.handle();
        let (_heads, head) = watch::channel(BlockInfo::default());
        let pipeline = TestPipeline { origin: Some(origin), ..TestPipeline::new() };
        let task = builder.build(TestEngine::new([]), pipeline, head);
        assert_eq!(handle.status().borrow().current_l1, Some(origin));
        drop(task);
    }

    /// Derivation starts by resetting the engine, asking again while the execution layer syncs,
    /// and resets its pipeline onto the engine's safe head, then onto the safe head of each later
    /// reset, including one another actor makes.
    #[tokio::test(start_paused = true)]
    async fn resets_the_pipeline_whenever_the_engine_resets() {
        let origin = BlockInfo { number: 4, ..Default::default() };
        let later_origin = BlockInfo { number: 2, ..Default::default() };
        let engine = TestEngine::new([
            Err(EngineClientError::ElSyncing),
            Ok(engine_state(safe_head(origin))),
        ]);
        let (mut actor, publishers) = actor(engine.clone(), TestPipeline::new());

        let stopped = run_during(&mut actor, async {
            settle().await;
            settle().await;
            assert_eq!(publishers.status.borrow().current_l1, Some(origin));
            engine.publish_reset(engine_state(safe_head(later_origin)));
            settle().await;
        })
        .await;

        assert!(stopped.is_none());
        assert_eq!(engine.requests(), [Request::Reset, Request::Reset]);
        assert_eq!(
            actor.deriver.pipeline.signals,
            [reset(safe_head(origin)), reset(safe_head(later_origin))]
        );
        assert_eq!(publishers.status.borrow().current_l1, Some(later_origin));
    }

    /// Another actor's reset that keeps the safe head still resets the pipeline.
    #[tokio::test(start_paused = true)]
    async fn follows_a_reset_that_keeps_the_safe_head() {
        let state = engine_state(block(3));
        let engine = TestEngine::new([Ok(state)]);
        let (mut actor, _publishers) = actor(engine.clone(), TestPipeline::new());

        run_during(&mut actor, async {
            settle().await;
            engine.publish_reset(state);
            settle().await;
        })
        .await;

        assert_eq!(actor.deriver.pipeline.signals, [reset(block(3)), reset(block(3))]);
    }

    /// A pipeline that asks for a reset resets the engine, asking again while the execution layer
    /// syncs, and derivation resets the pipeline onto the engine's new safe head.
    #[rstest]
    #[case::interop_inactive(None)]
    #[case::interop_active(Some(0))]
    #[tokio::test(start_paused = true)]
    async fn pipeline_reset_resets_the_engine(#[case] lagoon_time: Option<u64>) {
        let engine = TestEngine::new([
            Ok(engine_state(block(0))),
            Err(EngineClientError::ElSyncing),
            Ok(engine_state(block(3))),
        ]);
        let mut pipeline = TestPipeline::new();
        pipeline.rollup_config = Arc::new(RollupConfig {
            hardforks: HardForkConfig { lagoon_time, ..Default::default() },
            ..Default::default()
        });
        pipeline.reorg = true;
        let (mut actor, _publishers) = actor(engine.clone(), pipeline);

        run_until_out_of_data(&mut actor).await;

        assert_eq!(engine.requests(), [Request::Reset, Request::Reset, Request::Reset]);
        assert_eq!(actor.deriver.pipeline.signals, [reset(block(0)), reset(block(3))]);
        assert_eq!(actor.deriver.pipeline.stepped_on.last(), Some(&block(3)));
    }

    /// An engine reset cancels a pipeline step in progress, and the pipeline reset discards it.
    #[tokio::test(start_paused = true)]
    async fn reset_cancels_a_step_in_progress() {
        let engine = TestEngine::new([Ok(engine_state(block(0)))]);
        let pipeline = TestPipeline { hang: true, ..TestPipeline::new() };
        let (mut actor, _publishers) = actor(engine.clone(), pipeline);

        run_during(&mut actor, async {
            settle().await;
            engine.publish_reset(engine_state(block(3)));
            settle().await;
        })
        .await;

        assert_eq!(actor.deriver.pipeline.signals, [reset(block(0)), reset(block(3))]);
        assert_eq!(actor.deriver.pipeline.stepped_on, [block(0), block(3)]);
        assert!(actor.deriver.pipeline.out_of_data);
    }

    /// A pipeline reset that fails temporarily is retried after an interval.
    #[tokio::test(start_paused = true)]
    async fn temporary_pipeline_reset_failure_is_retried() {
        let engine = TestEngine::new([Ok(engine_state(block(3)))]);
        let pipeline = TestPipeline {
            reset_errors: VecDeque::from([PipelineError::Eof.temp()]),
            ..TestPipeline::new()
        };
        let (mut actor, _publishers) = actor(engine, pipeline);

        let stopped = run_during(&mut actor, time::sleep(PIPELINE_RESET_RETRY_INTERVAL * 2)).await;

        assert!(stopped.is_none());
        assert_eq!(actor.deriver.pipeline.signals, [reset(block(3))]);
    }

    /// A pipeline reset that keeps failing temporarily is retried onto the safe head of a newer
    /// engine reset.
    #[tokio::test(start_paused = true)]
    async fn pipeline_reset_retries_onto_a_newer_reset() {
        let engine = TestEngine::new([Ok(engine_state(block(3)))]);
        let pipeline = TestPipeline {
            reset_errors: VecDeque::from([PipelineError::Eof.temp()]),
            ..TestPipeline::new()
        };
        let (mut actor, _publishers) = actor(engine.clone(), pipeline);

        run_during(&mut actor, async {
            time::sleep(PIPELINE_RESET_RETRY_INTERVAL / 2).await;
            engine.publish_reset(engine_state(block(5)));
            settle().await;
        })
        .await;

        assert_eq!(actor.deriver.pipeline.signals, [reset(block(5))]);
    }

    /// A pipeline that cannot reset onto the safe head, such as one whose ancestors an L1 reorg
    /// replaced, asks the engine to reset, and resets onto the engine's new safe head.
    #[tokio::test(start_paused = true)]
    async fn pipeline_that_cannot_reset_resets_the_engine() {
        let engine = TestEngine::new([Ok(engine_state(block(3))), Ok(engine_state(block(2)))]);
        let pipeline = TestPipeline {
            reset_errors: VecDeque::from([ResetError::BlockNotFound(B256::ZERO.into()).reset()]),
            ..TestPipeline::new()
        };
        let (mut actor, _publishers) = actor(engine.clone(), pipeline);

        run_until_out_of_data(&mut actor).await;

        assert_eq!(engine.requests(), [Request::Reset, Request::Reset]);
        assert_eq!(actor.deriver.pipeline.signals, [reset(block(2))]);
    }

    /// A pipeline reset that fails critically stops the actor.
    #[tokio::test(start_paused = true)]
    async fn critical_pipeline_reset_failure_stops_the_actor() {
        let engine = TestEngine::new([Ok(engine_state(block(3)))]);
        let pipeline = TestPipeline {
            reset_errors: VecDeque::from([PipelineError::MissingOrigin.crit()]),
            ..TestPipeline::new()
        };
        let (mut actor, _publishers) = actor(engine, pipeline);

        assert!(matches!(actor.run().await, Err(ActorError::Pipeline(_))));
    }

    /// Engine resets published together reset the pipeline once, onto the latest.
    #[tokio::test(start_paused = true)]
    async fn coalesced_resets_reset_the_pipeline_once() {
        let engine = TestEngine::new([Ok(engine_state(block(0)))]);
        let (mut actor, _publishers) = actor(engine.clone(), TestPipeline::new());

        run_during(&mut actor, async {
            settle().await;
            engine.publish_reset(engine_state(block(2)));
            engine.publish_reset(engine_state(block(3)));
            settle().await;
        })
        .await;

        assert_eq!(actor.deriver.pipeline.signals, [reset(block(0)), reset(block(3))]);
    }

    /// Holocene's activation signals the pipeline with the safe head, and derivation continues.
    #[tokio::test(start_paused = true)]
    async fn holocene_activation_signals_the_pipeline() {
        let engine = TestEngine::new([Ok(engine_state(block(3)))]);
        let pipeline = TestPipeline { holocene_activation: true, ..TestPipeline::new() };
        let (mut actor, _publishers) = actor(engine, pipeline);

        run_until_out_of_data(&mut actor).await;

        assert!(matches!(
            actor.deriver.pipeline.signals[..],
            [Signal::Reset(_), Signal::Activation(activation)] if activation.l2_safe_head == block(3)
        ));
    }

    /// Derivation acts on the engine's reply to a consolidation: it derives on the new safe head,
    /// after flushing the channel of a replaced block, or resets the pipeline if the engine reset
    /// while consolidating.
    #[rstest]
    #[case::consolidated(false, false, None, block(1))]
    #[case::flushed(true, false, Some(Signal::FlushChannel), block(1))]
    #[case::reset(false, true, Some(reset(block(5))), block(5))]
    #[tokio::test(start_paused = true)]
    async fn acts_on_the_consolidation_reply(
        #[case] flush: bool,
        #[case] resets: bool,
        #[case] expected_signal: Option<Signal>,
        #[case] derives_on: L2BlockInfo,
    ) {
        let engine = TestEngine::new([Ok(engine_state(L2BlockInfo::default()))]);
        let state = engine_state(derives_on);
        engine.on_consolidation(move |published_resets| {
            if resets {
                published_resets.send_replace(state);
            }
            Consolidation { state, flush }
        });
        let pipeline = TestPipeline::preparing([attributes(L2BlockInfo::default())]);
        let (mut actor, _publishers) = actor(engine, pipeline);

        run_until_out_of_data(&mut actor).await;

        // The first signal is the reset that started derivation.
        let signals = &actor.deriver.pipeline.signals[1..];
        assert_eq!(signals, expected_signal.into_iter().collect::<Vec<_>>());
        assert_eq!(actor.deriver.pipeline.stepped_on.last(), Some(&derives_on));
    }

    /// L1 heads wake derivation, which follows the latest head, including when it reorgs back.
    /// Heads published while derivation is busy are coalesced.
    #[tokio::test(start_paused = true)]
    async fn head_changes_wake_derivation_and_coalesce_reorgs() {
        let engine = TestEngine::new([Ok(engine_state(L2BlockInfo::default()))]);
        let (mut actor, publishers) = actor(engine, TestPipeline::new());
        actor.deriver.pipeline.l1_head = Some(publishers.heads.subscribe());
        let status = publishers.status.clone();

        let stopped = run_during(&mut actor, async {
            settle().await;
            for (number, hash) in [(7, 1), (7, 2), (6, 3)] {
                let block =
                    BlockInfo { number, hash: B256::repeat_byte(hash), ..Default::default() };
                publishers.heads.send_replace(block);
                settle().await;
                assert_eq!(status.borrow().current_l1, Some(block));
            }
            publishers.heads.send_replace(BlockInfo { number: 8, ..Default::default() });
            let latest = BlockInfo { number: 9, ..Default::default() };
            publishers.heads.send_replace(latest);
            settle().await;
            assert_eq!(status.borrow().current_l1, Some(latest));
        })
        .await;

        assert!(stopped.is_none());
        assert!(!actor.deriver.l1.has_changed().unwrap());
    }

    /// An L1 head that arrives while deriving makes derivation step the pipeline again once it
    /// runs out of data.
    #[tokio::test(start_paused = true)]
    async fn head_during_derivation_resumes_it_afterwards() {
        let engine = TestEngine::new([Ok(engine_state(L2BlockInfo::default()))]);
        let (mut actor, publishers) =
            actor(engine.clone(), TestPipeline::preparing([attributes(L2BlockInfo::default())]));
        let heads = publishers.heads.clone();
        engine.on_consolidation(move |_| {
            heads.send_replace(BlockInfo { number: 1, ..Default::default() });
            consolidated(engine_state(block(1)))
        });

        run_until_out_of_data(&mut actor).await;

        // The step that prepared the attributes, the one that ran out of data, and one more after
        // the head.
        assert_eq!(actor.deriver.pipeline.stepped_on.len(), 3);
    }

    #[tokio::test(start_paused = true)]
    async fn closed_head_publisher_is_fatal() {
        let engine = TestEngine::new([Ok(engine_state(L2BlockInfo::default()))]);
        let (mut actor, publishers) = actor(engine, TestPipeline::new());
        drop(publishers.heads);

        assert!(matches!(actor.run().await, Err(ActorError::L1Stopped)));
    }
}
