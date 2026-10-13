//! The derivation actor, which derives payload attributes from L1.

use super::{ActorError, Builder, DerivationStatus, Engine, L2Finalizer, reset_engine};
use crate::{
    Metrics,
    engine::{Consolidation, EngineState, FinalizeBlockId},
};
use kona_derive::{
    ActivationSignal, Pipeline, PipelineError, PipelineErrorKind, ResetError, ResetSignal, Signal,
    SignalReceiver, StepResult,
};
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use std::future::Future;
use tokio::sync::watch;

impl Builder {
    /// Supplies the dependencies of the derivation actor that derives payload attributes from L1,
    /// and produces its lifetime future without spawning it.
    ///
    /// The status reports the pipeline's origin from now on. Runtime work begins when the future is
    /// polled. Dropping the future stops the actor.
    pub fn build<Engine_, PipelineSignalReceiver>(
        self,
        engine: Engine_,
        pipeline: PipelineSignalReceiver,
        head: watch::Receiver<BlockInfo>,
        finalized: watch::Receiver<BlockInfo>,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static
    where
        Engine_: Engine + 'static,
        PipelineSignalReceiver: Pipeline + SignalReceiver + Send + Sync + 'static,
    {
        let Self { handle: _, status_tx } = self;
        Actor::new(engine, status_tx, pipeline, head, finalized).run()
    }
}

/// Steps the derivation pipeline forward to produce payload attributes, and consolidates the
/// engine's safe head with each before deriving more.
///
/// It starts by resetting the engine once the execution layer is synced, and starts over from the
/// engine's safe head whenever the engine resets.
#[derive(Debug)]
struct Actor<Engine_, PipelineSignalReceiver>
where
    Engine_: Engine,
    PipelineSignalReceiver: Pipeline + SignalReceiver,
{
    /// The engine, which derivation drives.
    engine: Engine_,
    /// The derivation pipeline.
    pipeline: PipelineSignalReceiver,
    /// The latest published L1 head.
    head: watch::Receiver<BlockInfo>,
    /// The latest published finalized L1 head.
    finalized: watch::Receiver<BlockInfo>,
    /// Progress published to RPC readers.
    status: watch::Sender<DerivationStatus>,
}

/// What stepping the pipeline produced.
#[derive(Debug)]
#[allow(clippy::large_enum_variant)]
enum Derived {
    /// Attributes to consolidate.
    Attributes(OpAttributesWithParent),
    /// The pipeline ran out of data until L1 extends.
    OutOfData,
    /// The pipeline needs a reset, such as after an L1 reorg.
    ResetNeeded,
}

impl<Engine_, PipelineSignalReceiver> Actor<Engine_, PipelineSignalReceiver>
where
    Engine_: Engine,
    PipelineSignalReceiver: Pipeline + SignalReceiver + Send + Sync,
{
    /// Creates the actor and publishes the pipeline's origin.
    ///
    /// L1 updates are consumed as changes; initial snapshots do not trigger processing.
    fn new(
        engine: Engine_,
        status: watch::Sender<DerivationStatus>,
        pipeline: PipelineSignalReceiver,
        head: watch::Receiver<BlockInfo>,
        finalized: watch::Receiver<BlockInfo>,
    ) -> Self {
        status.send_replace(DerivationStatus { current_l1: pipeline.origin() });
        Self { engine, pipeline, head, finalized, status }
    }

    async fn run(mut self) -> Result<(), ActorError> {
        let mut state = reset_engine(&self.engine).await?;
        loop {
            state = self.derive_from(state).await?;
            warn!(target: "derivation", ?state, "The engine reset; resetting derivation onto it");
        }
    }

    /// Resets the pipeline onto the safe head of the engine after a reset, and derives on it until
    /// the engine resets again. Returns the engine's state after that reset.
    ///
    /// Between sets of attributes, it follows a reset another actor made, then finalizes. When the
    /// pipeline runs out of data, it waits for an L1 head, finalizing meanwhile.
    async fn derive_from(&mut self, mut state: EngineState) -> Result<EngineState, ActorError> {
        info!(target: "derivation", safe_head = ?state.safe_head(), "Deriving from the engine's safe head");
        self.signal(Signal::Reset(ResetSignal { l2_safe_head: state.safe_head() })).await;
        // Blocks derived since the reset, awaiting finalization.
        let mut finalizer = L2Finalizer::default();
        loop {
            let latest = self.engine.state();
            if latest.resets() > state.resets() {
                return Ok(latest);
            }
            if self.finalized.has_changed().map_err(|_| ActorError::L1ReceiveFailed)? {
                self.finalize(&mut finalizer).await?;
            }

            match self.produce_next_attributes(state.safe_head()).await? {
                Derived::Attributes(attributes) => {
                    trace!(target: "derivation", ?attributes, "Produced payload attributes.");
                    finalizer.enqueue_for_finalization(&attributes);
                    let Consolidation { state: consolidated, flush } =
                        self.engine.consolidate(attributes).await?;
                    if consolidated.resets() > state.resets() {
                        // The engine reset before or while consolidating, so the attributes may
                        // be stale.
                        return Ok(consolidated);
                    }
                    if flush {
                        // A deposits-only block replaced the derived one.
                        warn!(target: "derivation", "Flushing the channel of a rejected block");
                        self.signal(Signal::FlushChannel).await;
                    }
                    state = consolidated;
                }
                Derived::ResetNeeded => return reset_engine(&self.engine).await,
                Derived::OutOfData => {
                    info!(target: "derivation", "Yielding derivation until more data is available.");
                    if let Some(reset) =
                        self.wait_for_l1_head(state.resets(), &mut finalizer).await?
                    {
                        return Ok(reset);
                    }
                }
            }
        }
    }

    /// Waits for the next L1 head, finalizing as L1 blocks finalize meanwhile. Returns the engine's
    /// state instead if the engine resets more than `resets` times first.
    async fn wait_for_l1_head(
        &mut self,
        resets: u64,
        finalizer: &mut L2Finalizer,
    ) -> Result<Option<EngineState>, ActorError> {
        loop {
            tokio::select! {
                biased;
                reset = self.engine.reset_after(resets) => return Ok(Some(reset?)),
                result = self.finalized.changed() => {
                    result.map_err(|_| ActorError::L1ReceiveFailed)?;
                    self.finalize(finalizer).await?;
                }
                result = self.head.changed() => {
                    result.map_err(|_| ActorError::L1ReceiveFailed)?;
                    let head = *self.head.borrow_and_update();
                    info!(target: "derivation", l1_head = ?head, "Processing l1 head update");
                    return Ok(None);
                }
            }
        }
    }

    /// Steps the pipeline on the safe head until it produces attributes, runs out of data, or
    /// needs a reset.
    async fn produce_next_attributes(
        &mut self,
        safe_head: L2BlockInfo,
    ) -> Result<Derived, ActorError> {
        // As we start the safe head at the disputed block's parent, we step the pipeline until the
        // first attributes are produced. All batches at and before the safe head will be
        // dropped, so the first payload will always be the disputed one.
        loop {
            let result = self.pipeline.step(safe_head).await;
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
                    PipelineErrorKind::Temporary(e) => {
                        // NotEnoughData is transient, and doesn't imply we need to wait for
                        // more data. We can continue stepping until we receive an Eof.
                        if matches!(e, PipelineError::NotEnoughData) {
                            continue;
                        }

                        debug!(
                            target: "derivation",
                            "Exhausted data source for now; Yielding until the chain has extended."
                        );
                        return Ok(Derived::OutOfData);
                    }
                    PipelineErrorKind::Reset(e) => {
                        warn!(target: "derivation", "Derivation pipeline is being reset: {e}");

                        if !matches!(e, ResetError::HoloceneActivation) {
                            if let ResetError::ReorgDetected(expected, new) = e {
                                warn!(
                                    target: "derivation",
                                    "L1 reorg detected! Expected: {expected} | New: {new}"
                                );

                                metrics::counter!(Metrics::L1_REORG_COUNT).increment(1);
                            }
                            return Ok(Derived::ResetNeeded);
                        }
                        self.pipeline
                            .signal(Signal::Activation(ActivationSignal {
                                l2_safe_head: safe_head,
                            }))
                            .await?;
                        self.publish_status();
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

    /// Finalizes the highest derived L2 block whose L1 origin the latest finalized L1 head covers.
    async fn finalize(&mut self, finalizer: &mut L2Finalizer) -> Result<(), ActorError> {
        let finalized = *self.finalized.borrow_and_update();
        if let Some(l2_block_number) = finalizer.try_finalize_next(finalized) {
            // Finalize by number against the engine's canonical chain.
            self.engine.finalize(FinalizeBlockId::ByNumber(l2_block_number)).await?;
        }
        Ok(())
    }

    /// Signals the pipeline, logging a failure.
    async fn signal(&mut self, signal: Signal) {
        let result = self.pipeline.signal(signal).await;
        self.publish_status();
        match result {
            Ok(_) => info!(target: "derivation", ?signal, "[SIGNAL] Executed Successfully"),
            Err(e) => {
                error!(target: "derivation", ?e, ?signal, "Failed to signal derivation pipeline")
            }
        }
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
        engine::HeadsUpdate,
    };
    use alloy_primitives::B256;
    use async_trait::async_trait;
    use kona_derive::PipelineResult;
    use kona_genesis::{HardForkConfig, RollupConfig, SystemConfig};
    use rstest::rstest;
    use std::{
        collections::VecDeque,
        future::{self, Future},
        sync::Arc,
    };
    use tokio::time::{self, Duration};

    /// A pipeline stub that follows the L1 head, prepares attributes, yields, or reports a reset,
    /// and records the signals it receives and the safe heads it steps on.
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
        prepared: VecDeque<OpAttributesWithParent>,
        signals: Vec<Signal>,
        stepped_on: Vec<L2BlockInfo>,
        /// Whether the last step ran out of data.
        out_of_data: bool,
    }

    impl TestPipeline {
        fn new(origin: Option<BlockInfo>) -> Self {
            Self {
                rollup_config: Arc::new(RollupConfig::default()),
                origin,
                l1_head: None,
                reorg: false,
                holocene_activation: false,
                prepared: VecDeque::new(),
                signals: Vec::new(),
                stepped_on: Vec::new(),
                out_of_data: false,
            }
        }

        /// Returns a pipeline at the default L1 block that prepares the given attributes.
        fn preparing(attributes: impl IntoIterator<Item = OpAttributesWithParent>) -> Self {
            Self {
                prepared: attributes.into_iter().collect(),
                ..Self::new(Some(BlockInfo::default()))
            }
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

    type TestActor = Actor<Arc<TestEngine>, TestPipeline>;

    /// The L1 publishers an actor reads from, and its status.
    struct Publishers {
        heads: watch::Sender<BlockInfo>,
        finalized: watch::Sender<BlockInfo>,
        status: watch::Receiver<DerivationStatus>,
    }

    /// Creates an actor with the publishers it reads from.
    fn actor(engine: Arc<TestEngine>, pipeline: TestPipeline) -> (TestActor, Publishers) {
        let (heads, head_rx) = watch::channel(BlockInfo::default());
        let (finalized, finalized_rx) = watch::channel(BlockInfo::default());
        let (status_tx, status) = watch::channel(DerivationStatus::default());
        let actor = Actor::new(engine, status_tx, pipeline, head_rx, finalized_rx);
        (actor, Publishers { heads, finalized, status })
    }

    /// Derives from the engine state until derivation waits for more data, and returns the engine
    /// state after the reset that stopped it instead, if any.
    async fn derive(actor: &mut TestActor, state: EngineState) -> Option<EngineState> {
        let reset = time::timeout(Duration::from_secs(60), actor.derive_from(state)).await.ok();
        if reset.is_none() {
            assert!(actor.pipeline.out_of_data, "derivation stalled before running out of data");
        }
        reset.map(|result| result.unwrap())
    }

    /// Derives from the engine state while `script` runs, and returns the engine state after the
    /// reset that stopped derivation, if it stopped before the script finished.
    async fn derive_during(
        actor: &mut TestActor,
        state: EngineState,
        script: impl Future<Output = ()>,
    ) -> Option<EngineState> {
        tokio::select! {
            result = actor.derive_from(state) => Some(result.unwrap()),
            () = script => None,
        }
    }

    /// Lets derivation run until it waits.
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

    /// Returns an engine state whose safe head is the given block, after the given number of
    /// resets.
    fn engine_state(safe_head: L2BlockInfo, resets: u64) -> EngineState {
        let state = EngineState::default()
            .apply_update(HeadsUpdate { safe_head: Some(safe_head), ..Default::default() });
        (0..resets).fold(state, |state, _| state.count_reset())
    }

    /// Returns a consolidation reply without a flush.
    const fn consolidated(state: EngineState) -> Consolidation {
        Consolidation { state, flush: false }
    }

    /// Returns attributes derived from the default L1 block, whose parent is the given block.
    fn attributes(parent: L2BlockInfo) -> OpAttributesWithParent {
        OpAttributesWithParent::new(Default::default(), parent, Some(BlockInfo::default()), false)
    }

    /// Building needs no runtime, and the handle's status shows the pipeline's origin as soon as
    /// the actor is built, before it runs.
    #[test]
    fn build_publishes_the_pipeline_origin_without_a_runtime() {
        let origin = BlockInfo { number: 5, ..Default::default() };
        let builder = Builder::new();
        let handle = builder.handle();
        let (_heads, head) = watch::channel(BlockInfo::default());
        let (_finalized, finalized) = watch::channel(BlockInfo::default());
        let task =
            builder.build(TestEngine::new([]), TestPipeline::new(Some(origin)), head, finalized);
        assert_eq!(handle.status().borrow().current_l1, Some(origin));
        drop(task);
    }

    /// Derivation starts by resetting the engine, asking again while the execution layer syncs,
    /// and resets its pipeline onto the engine's safe head, then onto the safe head of each later
    /// reset.
    #[tokio::test(start_paused = true)]
    async fn starts_from_the_engine_reset_once_the_el_is_synced() {
        let origin = BlockInfo { number: 4, ..Default::default() };
        let later_origin = BlockInfo { number: 2, ..Default::default() };
        let engine = TestEngine::new([
            Err(EngineClientError::ElSyncing),
            Ok(engine_state(safe_head(origin), 1)),
        ]);
        let published = engine.published.clone();
        let (actor, publishers) = actor(engine.clone(), TestPipeline::new(None));
        let status = publishers.status.clone();

        tokio::select! {
            result = actor.run() => panic!("derivation stopped: {result:?}"),
            () = async {
                settle().await;
                settle().await;
                assert_eq!(status.borrow().current_l1, Some(origin));
                published.send_replace(engine_state(safe_head(later_origin), 2));
                settle().await;
            } => {}
        }

        assert_eq!(engine.requests(), [Request::Reset, Request::Reset]);
        assert_eq!(publishers.status.borrow().current_l1, Some(later_origin));
    }

    /// A pipeline that asks for a reset resets the engine, asking again while the execution layer
    /// syncs, and derivation starts over from the engine's new state.
    #[rstest]
    #[case::interop_inactive(None)]
    #[case::interop_active(Some(0))]
    #[tokio::test(start_paused = true)]
    async fn pipeline_reset_resets_the_engine(#[case] lagoon_time: Option<u64>) {
        let reset = engine_state(safe_head(BlockInfo { number: 3, ..Default::default() }), 2);
        let engine = TestEngine::new([Err(EngineClientError::ElSyncing), Ok(reset)]);
        let mut pipeline = TestPipeline::new(Some(BlockInfo::default()));
        pipeline.rollup_config = Arc::new(RollupConfig {
            hardforks: HardForkConfig { lagoon_time, ..Default::default() },
            ..Default::default()
        });
        pipeline.reorg = true;
        let (mut actor, _publishers) = actor(engine.clone(), pipeline);

        assert_eq!(derive(&mut actor, engine_state(L2BlockInfo::default(), 1)).await, Some(reset));
        assert_eq!(engine.requests(), [Request::Reset, Request::Reset]);
    }

    /// Holocene's activation signals the pipeline with the safe head, and derivation continues.
    #[tokio::test(start_paused = true)]
    async fn holocene_activation_signals_the_pipeline() {
        let mut pipeline = TestPipeline::new(Some(BlockInfo::default()));
        pipeline.holocene_activation = true;
        let (mut actor, _publishers) = actor(TestEngine::new([]), pipeline);

        assert_eq!(derive(&mut actor, engine_state(block(3), 1)).await, None);
        assert!(matches!(
            actor.pipeline.signals[..],
            [Signal::Reset(_), Signal::Activation(activation)] if activation.l2_safe_head == block(3)
        ));
    }

    /// Derivation acts on the engine's reply to a consolidation: it derives on the new safe head,
    /// after flushing the channel of a replaced block, or starts over if the engine reset.
    #[rstest]
    #[case::consolidated(1, false, None)]
    #[case::flushed(1, true, Some(Signal::FlushChannel))]
    #[case::reset(2, false, None)]
    #[tokio::test(start_paused = true)]
    async fn acts_on_the_consolidation_reply(
        #[case] resets: u64,
        #[case] flush: bool,
        #[case] expected_signal: Option<Signal>,
    ) {
        let reply = engine_state(block(1), resets);
        let engine = TestEngine::new([]);
        engine.consolidates(Consolidation { state: reply, flush });
        let (mut actor, _publishers) =
            actor(engine, TestPipeline::preparing([attributes(L2BlockInfo::default())]));

        let reset = derive(&mut actor, engine_state(L2BlockInfo::default(), 1)).await;

        // The first signal is the reset that started derivation.
        assert_eq!(actor.pipeline.signals[1..], expected_signal.into_iter().collect::<Vec<_>>());
        if resets > 1 {
            assert_eq!(reset, Some(reply));
        } else {
            assert_eq!(reset, None);
            assert_eq!(actor.pipeline.stepped_on.last(), Some(&block(1)));
        }
    }

    /// When another actor resets the engine, derivation starts over from the engine's state, even
    /// if the reset kept the safe head. A published state without a new reset changes nothing.
    #[tokio::test(start_paused = true)]
    async fn follows_resets_made_by_other_actors() {
        let engine = TestEngine::new([]);
        let published = engine.published.clone();
        let (mut actor, _publishers) = actor(engine, TestPipeline::new(Some(BlockInfo::default())));
        let state = engine_state(L2BlockInfo::default(), 1);
        let reset = engine_state(L2BlockInfo::default(), 2);

        let stopped_by = derive_during(&mut actor, state, async {
            published.send_replace(state);
            settle().await;
            published.send_replace(reset);
            future::pending().await
        })
        .await;

        assert_eq!(stopped_by, Some(reset));
    }

    /// A reset another actor makes between consolidations stops derivation before it consolidates
    /// the next attributes.
    #[tokio::test(start_paused = true)]
    async fn follows_a_reset_made_between_consolidations() {
        let reset = engine_state(L2BlockInfo::default(), 2);
        let engine = TestEngine::new([]);
        engine.on_consolidation(move |state| {
            state.send_replace(reset);
            consolidated(engine_state(block(1), 1))
        });
        let first = attributes(L2BlockInfo::default());
        let (mut actor, _publishers) =
            actor(engine.clone(), TestPipeline::preparing([first.clone(), attributes(block(1))]));

        assert_eq!(derive(&mut actor, engine_state(L2BlockInfo::default(), 1)).await, Some(reset));
        assert_eq!(engine.requests(), [Request::Consolidate(first)]);
    }

    /// Derivation handles a finalized L1 head between consolidations, rather than once the pipeline
    /// runs out of data.
    #[tokio::test(start_paused = true)]
    async fn finalizes_between_consolidations() {
        let engine = TestEngine::new([]);
        let (mut actor, publishers) = actor(
            engine.clone(),
            TestPipeline::preparing([attributes(block(0)), attributes(block(1))]),
        );
        let finalized = publishers.finalized.clone();
        engine.on_consolidation(move |_| {
            finalized.send_replace(BlockInfo { number: 1, ..Default::default() });
            consolidated(engine_state(block(1), 1))
        });
        engine.consolidates(consolidated(engine_state(block(2), 1)));

        assert_eq!(derive(&mut actor, engine_state(block(0), 1)).await, None);
        assert_eq!(
            engine.requests(),
            [
                Request::Consolidate(attributes(block(0))),
                Request::Finalize(FinalizeBlockId::ByNumber(1)),
                Request::Consolidate(attributes(block(1))),
            ]
        );
    }

    /// While waiting for more data, a finalized L1 head finalizes the derived blocks it covers,
    /// without stepping the pipeline again.
    #[tokio::test(start_paused = true)]
    async fn finality_changes_finalize_without_new_heads() {
        let derived = OpAttributesWithParent {
            derived_from: Some(BlockInfo { number: 7, ..Default::default() }),
            ..attributes(block(16))
        };
        let engine = TestEngine::new([]);
        engine.consolidates(consolidated(engine_state(block(17), 1)));
        let (mut actor, publishers) =
            actor(engine.clone(), TestPipeline::preparing([derived.clone()]));

        let stopped_by = derive_during(&mut actor, engine_state(block(16), 1), async {
            settle().await;
            publishers.finalized.send_replace(BlockInfo { number: 6, ..Default::default() });
            settle().await;
            publishers.finalized.send_replace(BlockInfo { number: 7, ..Default::default() });
            publishers.finalized.send_replace(BlockInfo { number: 8, ..Default::default() });
            settle().await;
        })
        .await;

        assert_eq!(stopped_by, None);
        assert_eq!(
            engine.requests(),
            [Request::Consolidate(derived), Request::Finalize(FinalizeBlockId::ByNumber(17))]
        );
        // The step that prepared the attributes, and the one that ran out of data.
        assert_eq!(actor.pipeline.stepped_on.len(), 2);
    }

    /// Blocks derived before a reset are not finalized after it, since they may no longer be
    /// canonical.
    #[tokio::test(start_paused = true)]
    async fn reset_drops_blocks_awaiting_finalization() {
        let engine = TestEngine::new([]);
        engine.consolidates(consolidated(engine_state(block(1), 1)));
        let published = engine.published.clone();
        let (mut actor, publishers) =
            actor(engine.clone(), TestPipeline::preparing([attributes(block(0))]));
        let reset = engine_state(block(0), 2);

        let stopped_by = derive_during(&mut actor, engine_state(block(0), 1), async {
            settle().await;
            published.send_replace(reset);
            future::pending().await
        })
        .await;
        assert_eq!(stopped_by, Some(reset));
        let stopped_by = derive_during(&mut actor, reset, async {
            settle().await;
            publishers.finalized.send_replace(BlockInfo { number: 1, ..Default::default() });
            settle().await;
        })
        .await;

        assert_eq!(stopped_by, None);
        assert_eq!(engine.requests(), [Request::Consolidate(attributes(block(0)))]);
    }

    /// L1 heads wake derivation, which follows the latest head, including when it reorgs back.
    /// Heads published while derivation is busy are coalesced.
    #[tokio::test(start_paused = true)]
    async fn head_changes_wake_derivation_and_coalesce_reorgs() {
        let (mut actor, publishers) =
            actor(TestEngine::new([]), TestPipeline::new(Some(BlockInfo::default())));
        actor.pipeline.l1_head = Some(publishers.heads.subscribe());
        let status = publishers.status.clone();

        let stopped_by =
            derive_during(&mut actor, engine_state(L2BlockInfo::default(), 1), async {
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
        assert_eq!(stopped_by, None);
        assert!(!actor.head.has_changed().unwrap());
    }

    /// An L1 head that arrives while deriving makes derivation step the pipeline again once it
    /// runs out of data.
    #[tokio::test(start_paused = true)]
    async fn head_during_derivation_resumes_it_afterwards() {
        let engine = TestEngine::new([]);
        let (mut actor, publishers) =
            actor(engine.clone(), TestPipeline::preparing([attributes(L2BlockInfo::default())]));
        let heads = publishers.heads.clone();
        engine.on_consolidation(move |_| {
            heads.send_replace(BlockInfo { number: 1, ..Default::default() });
            consolidated(engine_state(block(1), 1))
        });

        assert_eq!(derive(&mut actor, engine_state(L2BlockInfo::default(), 1)).await, None);
        // The step that prepared the attributes, the one that ran out of data, and one more after
        // the head.
        assert_eq!(actor.pipeline.stepped_on.len(), 3);
    }

    #[rstest]
    #[case::head(true)]
    #[case::finalized(false)]
    #[tokio::test(start_paused = true)]
    async fn closed_publisher_is_fatal(#[case] head: bool) {
        let (mut actor, publishers) =
            actor(TestEngine::new([]), TestPipeline::new(Some(BlockInfo::default())));
        let Publishers { heads, finalized, .. } = publishers;
        if head {
            drop(heads);
        } else {
            drop(finalized);
        }
        let result = actor.derive_from(engine_state(L2BlockInfo::default(), 1)).await;
        assert!(matches!(result, Err(ActorError::L1ReceiveFailed)));
    }
}
