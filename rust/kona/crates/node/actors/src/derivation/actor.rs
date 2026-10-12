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
use std::future::{self, Future};
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
        engine_state: watch::Receiver<EngineState>,
        pipeline: PipelineSignalReceiver,
        head: watch::Receiver<BlockInfo>,
        finalized: watch::Receiver<BlockInfo>,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static
    where
        Engine_: Engine + 'static,
        PipelineSignalReceiver: Pipeline + SignalReceiver + Send + Sync + 'static,
    {
        let Self { handle: _, status_tx } = self;
        Actor::new(engine, engine_state, status_tx, pipeline, head, finalized).run()
    }
}

/// Steps the derivation pipeline forward to produce payload attributes, and consolidates the
/// engine's safe head with each before deriving more.
///
/// It starts by resetting the engine once the execution layer is synced, and follows every later
/// engine reset through the reset count in the engine's state. It derives one set of attributes at
/// a time, so engine resets and L1 updates are handled between them.
#[derive(Debug)]
struct Actor<Engine_, PipelineSignalReceiver>
where
    Engine_: Engine,
    PipelineSignalReceiver: Pipeline + SignalReceiver,
{
    /// The engine's published state, whose reset count moves when another actor resets the engine.
    engine_state: watch::Receiver<EngineState>,
    /// The engine's reset count when derivation last reset onto the engine's safe head.
    engine_resets: u64,
    /// The engine's safe head, which derivation derives the next attributes on.
    safe_head: L2BlockInfo,
    /// Whether the pipeline may have attributes to derive. New L1 data and resets set it, and it
    /// is cleared when the pipeline runs out of data.
    deriving: bool,
    /// The latest published L1 head.
    head: watch::Receiver<BlockInfo>,
    /// The latest published finalized L1 head.
    finalized: watch::Receiver<BlockInfo>,
    /// The Engine client used to interact with the engine.
    engine: Engine_,

    /// Progress published to RPC readers.
    status: watch::Sender<DerivationStatus>,

    /// The derivation pipeline.
    pipeline: PipelineSignalReceiver,
    /// The [`L2Finalizer`] tracks derived L2 blocks awaiting finalization.
    finalizer: L2Finalizer,
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
        engine_state: watch::Receiver<EngineState>,
        status: watch::Sender<DerivationStatus>,
        pipeline: PipelineSignalReceiver,
        head: watch::Receiver<BlockInfo>,
        finalized: watch::Receiver<BlockInfo>,
    ) -> Self {
        status.send_replace(DerivationStatus { current_l1: pipeline.origin() });
        Self {
            status,
            pipeline,
            engine_state,
            engine_resets: 0,
            safe_head: L2BlockInfo::default(),
            deriving: false,
            head,
            finalized,
            engine,
            finalizer: L2Finalizer::default(),
        }
    }

    async fn run(mut self) -> Result<(), ActorError> {
        self.start().await?;
        loop {
            self.step().await?;
        }
    }

    /// Resets the engine once the execution layer is synced, and the pipeline onto its safe head.
    async fn start(&mut self) -> Result<(), ActorError> {
        let state = reset_engine(&self.engine).await?;
        info!(target: "derivation", safe_head = ?state.safe_head(), "Starting derivation from the engine's safe head");
        self.reset_onto(state).await;
        Ok(())
    }

    /// Handles an engine reset, the finalized L1 head, or the L1 head, in that order, or else
    /// derives the next attributes if there may be any.
    ///
    /// An L1 head is only read while derivation is idle, so one that arrives while deriving
    /// resumes derivation after the pipeline runs out of data.
    async fn step(&mut self) -> Result<(), ActorError> {
        tokio::select! {
            biased;
            result = self.engine_state.changed() => {
                result.map_err(|_| ActorError::EngineStateClosed)?;
                let state = *self.engine_state.borrow_and_update();
                self.follow_reset(state).await;
                Ok(())
            }
            result = self.finalized.changed() => {
                result.map_err(|_| ActorError::L1ReceiveFailed)?;
                let finalized = *self.finalized.borrow_and_update();
                self.finalize(finalized).await
            }
            result = self.head.changed(), if !self.deriving => {
                result.map_err(|_| ActorError::L1ReceiveFailed)?;
                let head = *self.head.borrow_and_update();
                info!(target: "derivation", l1_head = ?head, "Processing l1 head update");
                self.deriving = true;
                Ok(())
            }
            () = future::ready(()), if self.deriving => self.derive_next().await,
        }
    }

    /// Derives the next attributes and consolidates the engine's safe head with them, or stops
    /// deriving if the pipeline has run out of data.
    async fn derive_next(&mut self) -> Result<(), ActorError> {
        let Some(attributes) = self.produce_next_attributes().await? else {
            info!(target: "derivation", "Yielding derivation until more data is available.");
            self.deriving = false;
            return Ok(());
        };
        trace!(target: "derivation", ?attributes, "Produced payload attributes.");
        self.finalizer.enqueue_for_finalization(&attributes);

        let Consolidation { state, flush } = self.engine.consolidate(attributes).await?;
        // The engine may have reset before or while consolidating, so the attributes may be stale.
        if self.follow_reset(state).await {
            return Ok(());
        }
        if flush {
            // A deposits-only block replaced the derived one.
            warn!(target: "derivation", "Flushing the channel of a rejected block");
            self.signal(Signal::FlushChannel).await;
        }
        self.safe_head = state.safe_head();
        Ok(())
    }

    /// Steps the pipeline until it produces attributes, and returns them, or `None` once it runs
    /// out of data.
    ///
    /// A pipeline that asks for a reset resets the engine, then the pipeline onto the engine's new
    /// safe head, and keeps stepping.
    async fn produce_next_attributes(
        &mut self,
    ) -> Result<Option<OpAttributesWithParent>, ActorError> {
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
                StepResult::OriginAdvanceErr(e) | StepResult::StepFailed(e) => {
                    match e {
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
                            return Ok(None);
                        }
                        PipelineErrorKind::Reset(e) => {
                            warn!(target: "derivation", "Derivation pipeline is being reset: {e}");

                            if matches!(e, ResetError::HoloceneActivation) {
                                self.pipeline
                                    .signal(Signal::Activation(ActivationSignal {
                                        l2_safe_head: self.safe_head,
                                    }))
                                    .await?;
                                self.publish_status();
                            } else {
                                if let ResetError::ReorgDetected(expected, new) = e {
                                    warn!(
                                        target: "derivation",
                                        "L1 reorg detected! Expected: {expected} | New: {new}"
                                    );

                                    metrics::counter!(Metrics::L1_REORG_COUNT).increment(1);
                                }
                                let state = reset_engine(&self.engine).await?;
                                self.reset_onto(state).await;
                                continue;
                            }
                        }
                        PipelineErrorKind::Critical(_) => {
                            error!(target: "derivation", "Critical derivation error: {e}");
                            metrics::counter!(Metrics::DERIVATION_CRITICAL_ERROR).increment(1);
                            return Err(e.into());
                        }
                    }
                }
            }

            // If there are any new attributes, send them to the execution actor.
            if let Some(attrs) = self.pipeline.next() {
                return Ok(Some(attrs));
            }
        }
    }

    /// Resets the pipeline onto the safe head of the engine after a reset, ready to derive from it.
    async fn reset_onto(&mut self, state: EngineState) {
        self.engine_resets = state.resets();
        self.safe_head = state.safe_head();
        // Blocks derived before the reset may no longer be canonical.
        self.finalizer.clear();
        self.signal(Signal::Reset(ResetSignal { l2_safe_head: self.safe_head })).await;
        self.deriving = true;
    }

    /// Resets onto the engine's state if the engine reset since derivation last reset onto it, and
    /// returns whether it did.
    async fn follow_reset(&mut self, state: EngineState) -> bool {
        if state.resets() <= self.engine_resets {
            return false;
        }
        warn!(target: "derivation", ?state, "The engine reset; resetting derivation onto it");
        self.reset_onto(state).await;
        true
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

    /// Finalizes the highest derived L2 block whose L1 origin the finalized L1 head covers.
    async fn finalize(&mut self, finalized: BlockInfo) -> Result<(), ActorError> {
        if let Some(l2_block_number) = self.finalizer.try_finalize_next(finalized) {
            // Finalize by number against the engine's canonical chain.
            self.engine.finalize(FinalizeBlockId::ByNumber(l2_block_number)).await?;
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{EngineClientError, derivation::engine::MockEngine, engine::HeadsUpdate};
    use alloy_primitives::B256;
    use async_trait::async_trait;
    use kona_derive::PipelineResult;
    use kona_genesis::{HardForkConfig, RollupConfig, SystemConfig};
    use kona_protocol::{BlockInfo, L2BlockInfo};
    use rstest::rstest;
    use std::{collections::VecDeque, sync::Arc};

    /// A pipeline stub that can advance, prepare attributes, yield, or report one L1 reorg, and
    /// records the signals it receives.
    #[derive(Debug)]
    struct TestPipeline {
        rollup_config: Arc<RollupConfig>,
        origin: Option<BlockInfo>,
        advance_to: Option<BlockInfo>,
        reorg: bool,
        prepared: VecDeque<OpAttributesWithParent>,
        signals: Vec<Signal>,
    }

    impl TestPipeline {
        fn new(origin: Option<BlockInfo>) -> Self {
            Self {
                rollup_config: Arc::new(RollupConfig::default()),
                origin,
                advance_to: None,
                reorg: false,
                prepared: VecDeque::new(),
                signals: Vec::new(),
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

        async fn step(&mut self, _: L2BlockInfo) -> StepResult {
            if let Some(origin) = self.advance_to.take() {
                self.origin = Some(origin);
                StepResult::AdvancedOrigin
            } else if std::mem::take(&mut self.reorg) {
                StepResult::StepFailed(PipelineErrorKind::Reset(ResetError::ReorgDetected(
                    B256::ZERO,
                    B256::repeat_byte(1),
                )))
            } else if !self.prepared.is_empty() {
                StepResult::PreparedAttributes
            } else {
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

    /// The publishers an actor reads from.
    struct Publishers {
        engine_state: watch::Sender<EngineState>,
        heads: watch::Sender<BlockInfo>,
        finalized: watch::Sender<BlockInfo>,
    }

    /// Creates an actor that publishes to its own status channel, with the publishers it reads.
    fn actor(
        engine: MockEngine,
        pipeline: TestPipeline,
        finalized: BlockInfo,
    ) -> (Actor<MockEngine, TestPipeline>, Publishers) {
        let (engine_state, engine_state_rx) = watch::channel(EngineState::default());
        let (heads, head_rx) = watch::channel(BlockInfo::default());
        let (finalized, finalized_rx) = watch::channel(finalized);
        let (status, _) = watch::channel(DerivationStatus::default());
        let actor = Actor::new(engine, engine_state_rx, status, pipeline, head_rx, finalized_rx);
        (actor, Publishers { engine_state, heads, finalized })
    }

    /// Steps the actor until the pipeline runs out of data.
    async fn derive(actor: &mut Actor<MockEngine, TestPipeline>) {
        for _ in 0..100 {
            if !actor.deriving {
                return;
            }
            actor.step().await.unwrap();
        }
        panic!("derivation did not run out of data");
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
        let (_engine_state, engine_state) = watch::channel(EngineState::default());
        let (_heads, head) = watch::channel(BlockInfo::default());
        let (_finalized, finalized) = watch::channel(BlockInfo::default());
        let task = builder.build(
            MockEngine::new(),
            engine_state,
            TestPipeline::new(Some(origin)),
            head,
            finalized,
        );
        assert_eq!(handle.status().borrow().current_l1, Some(origin));
        drop(task);
    }

    /// Derivation starts by resetting the engine, asking again while the execution layer syncs,
    /// and resets its pipeline onto the engine's safe head.
    #[tokio::test(start_paused = true)]
    async fn starts_from_the_engine_reset_once_the_el_is_synced() {
        let origin = BlockInfo { number: 4, ..Default::default() };
        let mut engine = MockEngine::new();
        let mut attempts = 0;
        engine.expect_reset().times(2).returning(move || {
            attempts += 1;
            if attempts == 1 {
                Err(EngineClientError::ElSyncing)
            } else {
                Ok(engine_state(safe_head(origin), 1))
            }
        });
        let (mut actor, _publishers) = actor(engine, TestPipeline::new(None), BlockInfo::default());

        actor.start().await.unwrap();
        derive(&mut actor).await;

        assert!(matches!(actor.pipeline.signals[..], [Signal::Reset(_)]));
        assert_eq!(actor.status.borrow().current_l1, Some(origin));
        assert_eq!(actor.safe_head, safe_head(origin));
        assert_eq!(actor.engine_resets, 1);
        assert!(!actor.deriving);
    }

    /// A pipeline-driven reset resets the engine, asking again while the execution layer syncs,
    /// then resets the pipeline onto the engine's new safe head, and derivation continues.
    #[rstest]
    #[case::interop_inactive(None)]
    #[case::interop_active(Some(0))]
    #[tokio::test(start_paused = true)]
    async fn pipeline_reset_resets_the_engine_and_the_pipeline(#[case] lagoon_time: Option<u64>) {
        let reset_origin = BlockInfo { number: 3, ..Default::default() };
        let mut engine = MockEngine::new();
        let mut attempts = 0;
        engine.expect_reset().times(3).returning(move || {
            attempts += 1;
            match attempts {
                1 => Ok(engine_state(L2BlockInfo::default(), 1)),
                2 => Err(EngineClientError::ElSyncing),
                _ => Ok(engine_state(safe_head(reset_origin), 2)),
            }
        });
        let mut pipeline = TestPipeline::new(Some(BlockInfo::default()));
        pipeline.rollup_config = Arc::new(RollupConfig {
            hardforks: HardForkConfig { lagoon_time, ..Default::default() },
            ..Default::default()
        });
        pipeline.reorg = true;
        let (mut actor, _publishers) = actor(engine, pipeline, BlockInfo::default());

        actor.start().await.unwrap();
        derive(&mut actor).await;

        assert!(matches!(actor.pipeline.signals[..], [Signal::Reset(_), Signal::Reset(_)]));
        assert_eq!(actor.safe_head, safe_head(reset_origin));
        assert_eq!(actor.engine_resets, 2);
        assert!(!actor.deriving);
    }

    /// Derivation acts on the engine's reply to a consolidation: it flushes the channel of a
    /// replaced block, or resets onto the safe head of an engine that reset.
    #[rstest]
    #[case::consolidated(1, false, None)]
    #[case::flushed(1, true, Some(Signal::FlushChannel))]
    #[case::reset(2, false, Some(Signal::Reset(ResetSignal::default())))]
    #[tokio::test]
    async fn acts_on_the_consolidation_reply(
        #[case] resets: u64,
        #[case] flush: bool,
        #[case] expected_signal: Option<Signal>,
    ) {
        let new_safe_head = L2BlockInfo {
            block_info: BlockInfo { number: 1, ..Default::default() },
            ..Default::default()
        };
        let mut engine = MockEngine::new();
        engine.expect_reset().once().returning(|| Ok(engine_state(L2BlockInfo::default(), 1)));
        engine.expect_consolidate().once().returning(move |_| {
            Ok(Consolidation { state: engine_state(new_safe_head, resets), flush })
        });
        let mut pipeline = TestPipeline::new(Some(BlockInfo::default()));
        pipeline.prepared.push_back(attributes(L2BlockInfo::default()));
        let (mut actor, _publishers) = actor(engine, pipeline, BlockInfo::default());

        actor.start().await.unwrap();
        derive(&mut actor).await;

        // The first signal is the reset that started derivation.
        let signals = &actor.pipeline.signals[1..];
        match expected_signal {
            None => assert!(signals.is_empty()),
            Some(Signal::FlushChannel) => assert!(matches!(signals, [Signal::FlushChannel])),
            Some(_) => assert!(
                matches!(signals, [Signal::Reset(reset)] if reset.l2_safe_head == new_safe_head)
            ),
        }
        assert_eq!(actor.safe_head, new_safe_head);
        assert_eq!(actor.engine_resets, resets);
        assert!(!actor.deriving);
    }

    /// When another actor resets the engine, derivation resets its pipeline onto the engine's
    /// published safe head, even if the reset kept the safe head. A published state without a new
    /// reset changes nothing.
    #[tokio::test]
    async fn follows_resets_made_by_other_actors() {
        let mut engine = MockEngine::new();
        engine.expect_reset().once().returning(|| Ok(engine_state(L2BlockInfo::default(), 1)));
        let (mut actor, publishers) =
            actor(engine, TestPipeline::new(Some(BlockInfo::default())), BlockInfo::default());
        actor.start().await.unwrap();
        derive(&mut actor).await;

        publishers.engine_state.send_replace(engine_state(L2BlockInfo::default(), 1));
        actor.step().await.unwrap();
        assert_eq!(actor.pipeline.signals.len(), 1);
        assert!(!actor.deriving);

        let reset_origin = BlockInfo { number: 4, ..Default::default() };
        for resets in [2, 3] {
            publishers.engine_state.send_replace(engine_state(safe_head(reset_origin), resets));
            actor.step().await.unwrap();
            assert!(matches!(
                actor.pipeline.signals[(resets - 1) as usize..],
                [Signal::Reset(reset)] if reset.l2_safe_head == safe_head(reset_origin)
            ));
            // Derivation resumes from the engine's safe head.
            assert!(actor.deriving);
            derive(&mut actor).await;
        }
        assert_eq!(actor.status.borrow().current_l1, Some(reset_origin));
        assert_eq!(actor.safe_head, safe_head(reset_origin));
    }

    /// A reset another actor makes while derivation consolidates is followed before deriving the
    /// next attributes.
    #[tokio::test]
    async fn follows_a_reset_made_during_a_consolidation() {
        let reset_origin = BlockInfo { number: 4, ..Default::default() };
        let (engine_state_tx, engine_state_rx) = watch::channel(EngineState::default());
        let mut engine = MockEngine::new();
        engine.expect_reset().once().returning(|| Ok(engine_state(L2BlockInfo::default(), 1)));
        engine.expect_consolidate().once().returning(move |attributes| {
            let consolidated = engine_state(attributes.parent, 1);
            engine_state_tx.send_replace(engine_state(safe_head(reset_origin), 2));
            Ok(Consolidation { state: consolidated, flush: false })
        });
        let mut pipeline = TestPipeline::new(Some(BlockInfo::default()));
        pipeline.prepared.push_back(attributes(L2BlockInfo::default()));
        let (mut actor, _publishers) = actor(engine, pipeline, BlockInfo::default());
        actor.engine_state = engine_state_rx;

        actor.start().await.unwrap();
        derive(&mut actor).await;

        assert!(matches!(
            actor.pipeline.signals[..],
            [Signal::Reset(_), Signal::Reset(reset)] if reset.l2_safe_head == safe_head(reset_origin)
        ));
        assert_eq!(actor.engine_resets, 2);
    }

    /// Derivation handles a finalized L1 head between consolidations, rather than once the pipeline
    /// runs out of data.
    #[tokio::test]
    async fn finalizes_between_consolidations() {
        let block = |number| L2BlockInfo {
            block_info: BlockInfo { number, ..Default::default() },
            ..Default::default()
        };
        let consolidated = move |attributes: OpAttributesWithParent| {
            Ok(Consolidation {
                state: engine_state(block(attributes.block_number()), 1),
                flush: false,
            })
        };
        let mut engine = MockEngine::new();
        let mut sequence = mockall::Sequence::new();
        engine
            .expect_reset()
            .once()
            .in_sequence(&mut sequence)
            .returning(|| Ok(engine_state(L2BlockInfo::default(), 1)));
        engine.expect_consolidate().once().in_sequence(&mut sequence).returning(consolidated);
        engine
            .expect_finalize()
            .withf(|id| matches!(id, FinalizeBlockId::ByNumber(1)))
            .once()
            .in_sequence(&mut sequence)
            .returning(|_| Ok(()));
        engine.expect_consolidate().once().in_sequence(&mut sequence).returning(consolidated);
        let mut pipeline = TestPipeline::new(Some(BlockInfo::default()));
        pipeline.prepared.push_back(attributes(block(0)));
        pipeline.prepared.push_back(attributes(block(1)));
        let (mut actor, publishers) = actor(engine, pipeline, BlockInfo::default());

        actor.start().await.unwrap();
        actor.step().await.unwrap();
        publishers.finalized.send_replace(BlockInfo { number: 1, ..Default::default() });
        derive(&mut actor).await;

        assert_eq!(actor.safe_head, block(2));
    }

    #[tokio::test]
    async fn head_changes_wake_derivation_and_coalesce_reorgs() {
        let mut engine = MockEngine::new();
        engine.expect_reset().once().returning(|| Ok(engine_state(L2BlockInfo::default(), 1)));
        let (mut actor, publishers) =
            actor(engine, TestPipeline::new(Some(BlockInfo::default())), BlockInfo::default());
        actor.start().await.unwrap();
        derive(&mut actor).await;
        assert!(!actor.deriving);
        for (number, hash) in [(7, 1), (7, 2), (6, 3)] {
            let block = BlockInfo { number, hash: B256::repeat_byte(hash), ..Default::default() };
            actor.pipeline.advance_to = Some(block);
            publishers.heads.send_replace(block);
            actor.step().await.unwrap();
            derive(&mut actor).await;
            assert_eq!(actor.status.borrow().current_l1, Some(block));
        }
        publishers.heads.send_replace(BlockInfo { number: 8, ..Default::default() });
        let latest = BlockInfo { number: 9, ..Default::default() };
        publishers.heads.send_replace(latest);
        actor.pipeline.advance_to = Some(latest);
        actor.step().await.unwrap();
        derive(&mut actor).await;
        assert_eq!(actor.status.borrow().current_l1, Some(latest));
        assert!(!actor.head.has_changed().unwrap());
    }

    /// An L1 head that arrives while deriving is left for after the pipeline runs out of data, when
    /// it resumes derivation.
    #[tokio::test(start_paused = true)]
    async fn head_during_derivation_resumes_it_afterwards() {
        let mut engine = MockEngine::new();
        engine.expect_reset().once().returning(|| Ok(engine_state(L2BlockInfo::default(), 1)));
        engine.expect_consolidate().once().returning(|_| {
            Ok(Consolidation { state: engine_state(L2BlockInfo::default(), 1), flush: false })
        });
        let mut pipeline = TestPipeline::new(Some(BlockInfo::default()));
        pipeline.prepared.push_back(attributes(L2BlockInfo::default()));
        let (mut actor, publishers) = actor(engine, pipeline, BlockInfo::default());
        actor.start().await.unwrap();

        publishers.heads.send_replace(BlockInfo { number: 1, ..Default::default() });
        derive(&mut actor).await;
        tokio::time::timeout(tokio::time::Duration::from_secs(1), actor.step())
            .await
            .expect("the L1 head was left for afterwards")
            .unwrap();

        assert!(actor.deriving);
    }

    /// A reset drops the blocks awaiting finalization, which may no longer be canonical.
    #[tokio::test]
    async fn reset_drops_blocks_awaiting_finalization() {
        let mut engine = MockEngine::new();
        engine.expect_reset().once().returning(|| Ok(engine_state(L2BlockInfo::default(), 1)));
        engine.expect_finalize().never();
        let (mut actor, publishers) =
            actor(engine, TestPipeline::new(Some(BlockInfo::default())), BlockInfo::default());
        actor.start().await.unwrap();
        derive(&mut actor).await;
        actor.finalizer.enqueue_for_finalization(&attributes(L2BlockInfo::default()));

        publishers.engine_state.send_replace(engine_state(L2BlockInfo::default(), 2));
        actor.step().await.unwrap();
        derive(&mut actor).await;
        publishers.finalized.send_replace(BlockInfo { number: 1, ..Default::default() });
        actor.step().await.unwrap();
    }

    #[tokio::test]
    async fn finality_changes_finalize_without_new_heads() {
        let mut engine = MockEngine::new();
        engine
            .expect_finalize()
            .withf(|id| matches!(id, FinalizeBlockId::ByNumber(17)))
            .once()
            .returning(|_| Ok(()));
        let (mut actor, publishers) =
            actor(engine, TestPipeline::new(None), BlockInfo { number: 99, ..Default::default() });
        actor.finalizer.enqueue_for_finalization(&OpAttributesWithParent {
            attributes: Default::default(),
            parent: L2BlockInfo {
                block_info: BlockInfo { number: 16, ..Default::default() },
                ..Default::default()
            },
            derived_from: Some(BlockInfo { number: 7, ..Default::default() }),
            is_last_in_span: false,
        });
        // Initial snapshots do not act as block events.
        tokio::select! {
            biased;
            result = actor.step() => panic!("processed an initial snapshot: {result:?}"),
            _ = std::future::ready(()) => {}
        }
        publishers.finalized.send_replace(BlockInfo { number: 6, ..Default::default() });
        actor.step().await.unwrap();
        publishers.finalized.send_replace(BlockInfo { number: 7, ..Default::default() });
        publishers.finalized.send_replace(BlockInfo { number: 8, ..Default::default() });
        actor.step().await.unwrap();
    }

    #[rstest]
    #[case::head(0)]
    #[case::finalized(1)]
    #[case::engine_state(2)]
    #[tokio::test]
    async fn closed_publisher_is_fatal(#[case] closed: usize) {
        let (mut actor, publishers) =
            actor(MockEngine::new(), TestPipeline::new(None), BlockInfo::default());
        let Publishers { engine_state, heads, finalized } = publishers;
        match closed {
            0 => drop(heads),
            1 => drop(finalized),
            _ => drop(engine_state),
        }
        let result = actor.step().await;
        if closed == 2 {
            assert!(matches!(result, Err(ActorError::EngineStateClosed)));
        } else {
            assert!(matches!(result, Err(ActorError::L1ReceiveFailed)));
        }
    }
}
