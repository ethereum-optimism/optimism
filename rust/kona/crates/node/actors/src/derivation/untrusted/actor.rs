//! The actor that derives payload attributes from L1.

use super::{ActorError, L1, L2, Message, attributes::AttributesMatch};
use crate::{
    Metrics,
    engine::{Derived, Safe},
    l1::{Child, HandleError},
};
use kona_derive::{
    ActivationSignal, BuilderError, Pipeline, PipelineError, PipelineErrorKind, ResetError,
    ResetSignal, Signal, SignalReceiver, StepResult,
};
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use tokio::{
    sync::{mpsc, watch},
    time::{self, Duration},
};

/// How long to wait before trying again to reset the pipeline after a failure that is not
/// critical.
const PIPELINE_RESET_RETRY_INTERVAL: Duration = Duration::from_secs(1);

/// How long to wait before stepping the pipeline again after a temporary failure while L1 has
/// blocks after the pipeline's origin.
const STEP_RETRY_INTERVAL: Duration = Duration::from_secs(1);

/// How long to wait before reading the unsafe chain again after the read failed.
const L2_READ_RETRY_INTERVAL: Duration = Duration::from_secs(1);

/// Derives payload attributes from L1 on the safe head the engine gives it, and publishes them.
///
/// It derives nothing until the engine first resets it. A reset discards the attributes derived
/// before it, and the pipeline resets onto the reset's safe head before it derives again. The
/// actor derives the attributes on top of the safe head the engine last gave it, publishes them,
/// and waits for the engine to give it the next safe head.
#[derive(Debug)]
pub(super) struct Actor<Pipeline_, L1_, L2_> {
    /// The pipeline, and the safe head it derives on.
    deriver: Deriver<Pipeline_, L1_, L2_>,
    /// Receives requests from handles.
    messages_rx: mpsc::Receiver<Message>,
    /// The engine's safe head, which handles publish.
    safe_head_rx: watch::Receiver<Option<L2BlockInfo>>,
    /// Publishes what was derived on the safe head, with the safe head.
    derived_tx: watch::Sender<Option<(L2BlockInfo, Derived)>>,
}

/// The actor's end of the channels to its handles.
#[derive(Debug)]
pub(super) struct Channels {
    /// Receives requests from handles.
    pub(super) messages_rx: mpsc::Receiver<Message>,
    /// The engine's safe head, which handles publish.
    pub(super) safe_head_rx: watch::Receiver<Option<L2BlockInfo>>,
    /// Publishes what was derived on the safe head, with the safe head.
    pub(super) derived_tx: watch::Sender<Option<(L2BlockInfo, Derived)>>,
    /// Publishes the pipeline's origin.
    pub(super) current_l1_tx: watch::Sender<BlockInfo>,
}

impl<Pipeline_, L1_, L2_> Actor<Pipeline_, L1_, L2_>
where
    Pipeline_: Pipeline + SignalReceiver + Send + Sync,
    L1_: L1,
    L2_: L2,
{
    /// Creates the actor, and publishes the pipeline's origin.
    pub(super) fn new(pipeline: Pipeline_, l1: L1_, l2: L2_, channels: Channels) -> Self {
        let Channels { messages_rx, safe_head_rx, derived_tx, current_l1_tx } = channels;
        let deriver = Deriver::new(pipeline, l1, l2, current_l1_tx);
        Self { deriver, messages_rx, safe_head_rx, derived_tx }
    }

    pub(super) async fn run(&mut self) -> Result<(), ActorError> {
        // Whether the engine reset derivation, which starts it.
        let mut started = false;
        // Whether the pipeline must reset onto the safe head before it derives.
        let mut resetting = false;
        // Whether the attributes on the deriver's safe head are published.
        let mut derived = false;
        loop {
            tokio::select! {
                // Requests interrupt derivation in progress.
                biased;
                message = self.messages_rx.recv() => {
                    match message.ok_or(ActorError::HandlesDropped)? {
                        Message::Reset { safe, result_tx } => {
                            info!(target: "derivation", ?safe, "Resetting derivation onto the engine's safe head");
                            // Attributes derived before the reset are stale.
                            self.derived_tx.send_replace(None);
                            // A safe head published before the reset is stale too.
                            self.safe_head_rx.mark_unchanged();
                            self.deriver.safe_head = safe;
                            (started, resetting, derived) = (true, true, false);
                            let _ = result_tx.send(());
                        }
                        Message::Flush { result_tx } => {
                            // The engine flushes only after adopting the published attributes, and
                            // before it gives the next safe head, so nothing is being derived.
                            self.deriver.flush().await;
                            let _ = result_tx.send(());
                        }
                    }
                }
                changed = self.safe_head_rx.changed(), if derived => {
                    changed.map_err(|_| ActorError::HandlesDropped)?;
                    let safe_head = *self.safe_head_rx.borrow_and_update();
                    if let Some(safe_head) = safe_head.filter(|head| *head != self.deriver.safe_head) {
                        // The pipeline only steps on from the safe head it derived on, to its
                        // child. On any other safe head, it resets first.
                        resetting = safe_head.block_info.parent_hash != self.deriver.safe_head.block_info.hash;
                        self.deriver.safe_head = safe_head;
                        derived = false;
                    }
                }
                next = Self::derive(&mut self.deriver, resetting), if started && !derived => {
                    let next = next?;
                    trace!(target: "derivation", ?next, "Derived what to make safe next");
                    self.derived_tx.send_replace(Some((self.deriver.safe_head, next)));
                    (resetting, derived) = (false, true);
                }
            }
        }
    }

    /// Resets the pipeline onto the safe head if `resetting`, then derives the attributes on it,
    /// and returns what to make safe next.
    ///
    /// Cancelling it can leave a step half done, so only a pipeline reset may follow.
    async fn derive(
        deriver: &mut Deriver<Pipeline_, L1_, L2_>,
        resetting: bool,
    ) -> Result<Derived, ActorError> {
        if resetting {
            deriver.reset_onto(deriver.safe_head).await?;
        }
        let attributes = deriver.next_attributes().await?;
        Ok(deriver.compare_with_unsafe_chain(attributes).await)
    }
}

/// What stepping the pipeline produced.
#[derive(Debug)]
#[allow(clippy::large_enum_variant)]
enum Stepped {
    /// Attributes for the engine.
    Attributes(OpAttributesWithParent),
    /// The pipeline needs a reset, such as after an L1 reorg.
    ResetNeeded,
}

/// The derivation pipeline, the safe head it derives on, the L1 chain it waits on, and the L2
/// execution layer whose unsafe chain it compares attributes with.
#[derive(Debug)]
struct Deriver<Pipeline_, L1_, L2_> {
    /// The derivation pipeline.
    pipeline: Pipeline_,
    /// The engine's safe head, which the next attributes extend.
    safe_head: L2BlockInfo,
    /// The L1 chain, which wakes derivation once it extends the pipeline's origin.
    l1: L1_,
    /// The L2 execution layer.
    l2: L2_,
    /// Publishes the pipeline's origin to RPC readers.
    current_l1: watch::Sender<BlockInfo>,
}

impl<Pipeline_, L1_, L2_> Deriver<Pipeline_, L1_, L2_>
where
    Pipeline_: Pipeline + SignalReceiver + Send + Sync,
    L1_: L1,
    L2_: L2,
{
    /// Creates the deriver and publishes the pipeline's origin.
    fn new(pipeline: Pipeline_, l1: L1_, l2: L2_, current_l1: watch::Sender<BlockInfo>) -> Self {
        let deriver = Self { pipeline, safe_head: L2BlockInfo::default(), l1, l2, current_l1 };
        deriver.publish_current_l1();
        deriver
    }

    /// Returns what to make safe from `attributes`: the unsafe chain's block at their height, if
    /// the execution layer has one that matches them, and otherwise the attributes, for the engine
    /// to build the block from. A failed read is retried: building the block would truncate the
    /// unsafe chain after it.
    async fn compare_with_unsafe_chain(&self, attributes: OpAttributesWithParent) -> Derived {
        let Some(source) = attributes.derived_from else {
            return Derived::Attributes(attributes);
        };
        let block = loop {
            match self.l2.block(attributes.block_number()).await {
                Ok(block) => break block,
                Err(err) => {
                    warn!(target: "derivation", ?err, "Failed to read the unsafe block; retrying");
                    time::sleep(L2_READ_RETRY_INTERVAL).await;
                }
            }
        };
        let Some(block) = block else {
            return Derived::Attributes(attributes);
        };
        let config = self.pipeline.rollup_config();
        let matched = AttributesMatch::check(config, &attributes, &block);
        if !matched.is_match() {
            debug!(target: "derivation", ?matched, block_hash = %block.header.hash, "The unsafe block does not match the derived attributes");
            return Derived::Attributes(attributes);
        }
        let block = block.into_consensus().map_transactions(|tx| tx.inner.inner.into_inner());
        match L2BlockInfo::from_block_and_genesis(&block, &config.genesis) {
            Ok(head) => Derived::Safe(Safe { head, source }),
            Err(err) => {
                warn!(target: "derivation", ?err, "Failed to read the unsafe block's info; building the derived block");
                Derived::Attributes(attributes)
            }
        }
    }

    /// Derives the attributes on the safe head, resetting the pipeline onto the safe head
    /// whenever it needs a reset.
    ///
    /// Cancelling it can leave a step half done, so only a pipeline reset may follow.
    async fn next_attributes(&mut self) -> Result<OpAttributesWithParent, ActorError> {
        loop {
            match self.step().await? {
                Stepped::Attributes(attributes) => return Ok(attributes),
                Stepped::ResetNeeded => self.reset_onto(self.safe_head).await?,
            }
        }
    }

    /// Steps the pipeline on the safe head until it produces attributes or needs a reset, waiting
    /// after each temporary failure until stepping again may succeed.
    ///
    /// Cancelling it can leave a step half done, so only a pipeline reset may follow. The pipeline
    /// queues the attributes a step prepares, which a reset does not clear, but they are taken
    /// right after the step, with no wait between, so the queue is empty whenever it is cancelled.
    async fn step(&mut self) -> Result<Stepped, ActorError> {
        // As we start the safe head at the disputed block's parent, we step the pipeline until the
        // first attributes are produced. All batches at and before the safe head will be
        // dropped, so the first payload will always be the disputed one.
        loop {
            let result = self.pipeline.step(self.safe_head).await;
            self.publish_current_l1();
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
                    PipelineErrorKind::Temporary(err) => {
                        debug!(target: "derivation", ?err, "Derivation stopped temporarily");
                        if let Some(derived) = self.wait_to_retry().await? {
                            return Ok(derived);
                        }
                    }
                    PipelineErrorKind::Reset(ResetError::HoloceneActivation) => {
                        info!(target: "derivation", "Activating Holocene in the pipeline");
                        self.pipeline
                            .signal(Signal::Activation(ActivationSignal {
                                l2_safe_head: self.safe_head,
                            }))
                            .await?;
                        self.publish_current_l1();
                    }
                    PipelineErrorKind::Reset(ResetError::ReorgDetected(expected, new)) => {
                        warn!(target: "derivation", "L1 reorg detected! Expected: {expected} | New: {new}");
                        metrics::counter!(Metrics::L1_REORG_COUNT).increment(1);
                        return Ok(Stepped::ResetNeeded);
                    }
                    PipelineErrorKind::Reset(e) if is_inconsistent(&e) => {
                        error!(target: "derivation", "The pipeline is inconsistent with the safe head: {e}");
                        return Err(ActorError::Inconsistent(e));
                    }
                    PipelineErrorKind::Reset(e) => {
                        warn!(target: "derivation", "Derivation pipeline is being reset: {e}");
                        return Ok(Stepped::ResetNeeded);
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
                return Ok(Stepped::Attributes(attributes));
            }
        }
    }

    /// Waits, after a temporary failure, until stepping the pipeline again may succeed: until L1
    /// extends the pipeline's origin if it has not, or else for an interval, since something other
    /// than a lack of L1 data failed. Returns [`Stepped::ResetNeeded`] if L1 reorged out the
    /// origin instead.
    async fn wait_to_retry(&self) -> Result<Option<Stepped>, ActorError> {
        let origin = self.pipeline.origin().ok_or(PipelineError::MissingOrigin.crit())?;
        let l1_stopped = |_: HandleError| ActorError::L1Stopped;
        let child = match self.l1.try_child_of(origin.id()).await.map_err(l1_stopped)? {
            None => {
                debug!(target: "derivation", ?origin, "Waiting for L1 to extend the pipeline's origin");
                self.l1.child_of(origin.id()).await.map_err(l1_stopped)?
            }
            Some(child) => {
                if let Child::Block(_) = child {
                    time::sleep(STEP_RETRY_INTERVAL).await;
                }
                child
            }
        };
        match child {
            Child::Block(_) => Ok(None),
            Child::Reorged => {
                warn!(target: "derivation", ?origin, "L1 reorged out the pipeline's origin");
                metrics::counter!(Metrics::L1_REORG_COUNT).increment(1);
                Ok(Some(Stepped::ResetNeeded))
            }
        }
    }

    /// Resets the pipeline onto `safe_head`, retrying a failure that is not critical after an
    /// interval until the reset succeeds. A pipeline that cannot reset onto the safe head, such as
    /// one whose L1 origin an L1 reorg replaced, retries too: the engine resets for such a reorg,
    /// and then resets derivation onto its new safe head.
    async fn reset_onto(&mut self, safe_head: L2BlockInfo) -> Result<(), ActorError> {
        self.safe_head = safe_head;
        loop {
            let signal = Signal::Reset(ResetSignal { l2_safe_head: safe_head });
            let result = self.pipeline.signal(signal).await;
            self.publish_current_l1();
            match result {
                Ok(()) => {
                    info!(target: "derivation", ?signal, "[SIGNAL] Executed Successfully");
                    return Ok(());
                }
                Err(PipelineErrorKind::Temporary(err)) => {
                    warn!(target: "derivation", ?err, "Failed to reset the pipeline; retrying");
                }
                Err(PipelineErrorKind::Reset(err)) => {
                    warn!(target: "derivation", ?err, "The pipeline cannot reset onto the safe head; retrying");
                }
                Err(err) => {
                    error!(target: "derivation", ?err, "Failed to reset the pipeline");
                    return Err(err.into());
                }
            }
            time::sleep(PIPELINE_RESET_RETRY_INTERVAL).await;
        }
    }

    /// Flushes the channel of a block the engine replaced with a deposits-only block.
    async fn flush(&mut self) {
        warn!(target: "derivation", "Flushing the channel of a rejected block");
        let result = self.pipeline.signal(Signal::FlushChannel).await;
        self.publish_current_l1();
        if let Err(err) = result {
            error!(target: "derivation", ?err, "Failed to flush the pipeline's channel");
        }
    }

    fn publish_current_l1(&self) {
        if let Some(origin) = self.pipeline.origin() {
            self.current_l1
                .send_if_modified(|current| std::mem::replace(current, origin) != origin);
        }
    }
}

/// Returns whether a reset error means the pipeline is inconsistent with the safe head it derives
/// on: batch validation drops invalid batches before they reach these checks, and an L1 reorg
/// cannot cause them, so they are bugs, which a pipeline reset would only repeat.
const fn is_inconsistent(err: &ResetError) -> bool {
    matches!(
        err,
        ResetError::BadParentHash(..) |
            ResetError::BadTimestamp(..) |
            ResetError::L1OriginMismatch(..) |
            ResetError::AttributesBuilder(BuilderError::BrokenTimeInvariant(..))
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        derivation::untrusted::{Builder, Handle, actor::Channels},
        engine::Safety,
        l1::{
            self,
            test_utils::{TestChain, block as l1_block},
        },
    };
    use alloy_primitives::B256;
    use alloy_rpc_types_eth::{Block, BlockTransactions, Header as RpcHeader};
    use alloy_transport::{TransportError, TransportErrorKind};
    use async_trait::async_trait;
    use kona_derive::PipelineResult;
    use kona_genesis::{HardForkConfig, RollupConfig, SystemConfig};
    use kona_protocol::BlockInfo;
    use op_alloy_rpc_types::Transaction;
    use op_alloy_rpc_types_engine::OpPayloadAttributes;
    use rstest::rstest;
    use std::{
        collections::{HashMap, VecDeque},
        future::{self, Future},
        sync::{Arc, Mutex},
    };

    /// A pipeline stub that advances its origin along L1, prepares attributes, runs out of data,
    /// reports a reset or a temporary failure, or hangs, and records the signals it receives and
    /// the safe heads it steps on.
    #[derive(Debug)]
    struct TestPipeline {
        rollup_config: Arc<RollupConfig>,
        origin: Option<BlockInfo>,
        /// The L1 chain the origin advances along, one block a step.
        l1: Option<l1::Handle>,
        /// Fails the next step temporarily.
        temporary_failure: bool,
        /// Reports an L1 reorg on the next step.
        reorg: bool,
        /// Reports Holocene's activation on the next step.
        holocene_activation: bool,
        /// Never finishes the next step.
        hang: bool,
        /// Fails the next step with this error.
        failure: Option<PipelineErrorKind>,
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
                l1: None,
                temporary_failure: false,
                reorg: false,
                holocene_activation: false,
                hang: false,
                failure: None,
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
            let next_origin = match (&self.l1, self.origin) {
                (Some(l1), Some(origin)) => l1.canonical(origin.number + 1).await.unwrap(),
                _ => None,
            };
            if let Some(failure) = self.failure.take() {
                StepResult::StepFailed(failure)
            } else if std::mem::take(&mut self.temporary_failure) {
                StepResult::StepFailed(PipelineError::Provider("unavailable".into()).temp())
            } else if std::mem::take(&mut self.reorg) {
                StepResult::StepFailed(PipelineErrorKind::Reset(ResetError::ReorgDetected(
                    B256::ZERO,
                    B256::repeat_byte(1),
                )))
            } else if std::mem::take(&mut self.holocene_activation) {
                StepResult::StepFailed(PipelineErrorKind::Reset(ResetError::HoloceneActivation))
            } else if next_origin.is_some() {
                self.origin = next_origin;
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

    /// An L2 execution layer whose canonical blocks the test sets, and which fails as many reads
    /// as the test asks first.
    #[derive(Debug, Clone, Default)]
    struct TestL2(Arc<Mutex<HashMap<u64, Block<Transaction>>>>, Arc<Mutex<usize>>);

    #[async_trait]
    impl L2 for TestL2 {
        async fn block(&self, number: u64) -> Result<Option<Block<Transaction>>, TransportError> {
            let mut failures = self.1.lock().unwrap();
            if *failures > 0 {
                *failures -= 1;
                return Err(TransportErrorKind::custom_str("unavailable"));
            }
            Ok(self.0.lock().unwrap().get(&number).cloned())
        }
    }

    type TestActor = Actor<TestPipeline, l1::Handle, TestL2>;

    /// The L1 chain an actor follows, the L2 execution layer it reads, and the handle that drives
    /// it.
    struct Publishers {
        l1: TestChain,
        l2: TestL2,
        handle: Handle,
    }

    /// Creates an actor with the publishers it reads from. L1 holds only the default block, the
    /// origin of the test safe heads, so the pipeline has derived from all of L1. The L2 execution
    /// layer has no blocks.
    fn actor(pipeline: TestPipeline) -> (TestActor, Publishers) {
        let builder = Builder::new();
        let handle = builder.handle();
        let Builder { handle: _, messages_rx, safe_head_rx, derived_tx, current_l1_tx } = builder;
        let l1 = TestChain::new(&[BlockInfo::default()]);
        let l2 = TestL2::default();
        let channels = Channels { messages_rx, safe_head_rx, derived_tx, current_l1_tx };
        let actor = Actor::new(pipeline, l1.handle(), l2.clone(), channels);
        (actor, Publishers { l1, l2, handle })
    }

    /// Returns L1 block `number` of the chain `fork`, whose parent is block `number - 1` of the
    /// chain `parent_fork`, and block 0 the default block.
    fn l1_child(number: u64, fork: u8, parent_fork: u8) -> BlockInfo {
        let parent_hash = if number == 1 {
            B256::ZERO
        } else {
            l1_block(number - 1, parent_fork, parent_fork).hash
        };
        BlockInfo { parent_hash, ..l1_block(number, fork, parent_fork) }
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

    /// Resets the actor onto `safe_head`, and runs it until it waits for more L1 data, leaving time
    /// for any retries.
    async fn run_until_out_of_data(actor: &mut TestActor, handle: &Handle, safe_head: L2BlockInfo) {
        let script = async {
            handle.clone().reset(safe_head).await.unwrap();
            time::sleep(Duration::from_secs(60)).await;
        };
        assert!(run_during(actor, script).await.is_none(), "the actor stopped");
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

    /// Returns attributes derived from the default L1 block, whose parent is the given block.
    fn attributes(parent: L2BlockInfo) -> OpAttributesWithParent {
        OpAttributesWithParent::new(Default::default(), parent, Some(BlockInfo::default()), false)
    }

    /// Returns a reset signal onto the given safe head.
    const fn reset(l2_safe_head: L2BlockInfo) -> Signal {
        Signal::Reset(ResetSignal { l2_safe_head })
    }

    /// Building needs no runtime, and the handle shows the pipeline's origin as soon as the actor
    /// is built, before it runs.
    #[test]
    fn build_publishes_the_pipeline_origin_without_a_runtime() {
        let origin = BlockInfo { number: 5, ..Default::default() };
        let builder = Builder::new();
        let handle = builder.handle();
        let l1 = TestChain::new(&[BlockInfo::default()]);
        let pipeline = TestPipeline { origin: Some(origin), ..TestPipeline::new() };
        let task = builder.build(pipeline, l1.handle(), TestL2::default());
        assert_eq!(handle.current_l1(), origin);
        drop(task);
    }

    /// Derivation derives nothing until the engine resets it, and then resets the pipeline onto
    /// the safe head of each reset, including one that keeps the safe head.
    #[tokio::test(start_paused = true)]
    async fn resets_the_pipeline_whenever_the_engine_resets_it() {
        let origin = BlockInfo { number: 4, ..Default::default() };
        let later_origin = BlockInfo { number: 2, ..Default::default() };
        let (mut actor, publishers) = actor(TestPipeline::new());
        let mut handle = publishers.handle.clone();

        let stopped = run_during(&mut actor, async {
            settle().await;
            handle.reset(safe_head(origin)).await.unwrap();
            settle().await;
            assert_eq!(handle.current_l1(), origin);
            handle.reset(safe_head(later_origin)).await.unwrap();
            settle().await;
            assert_eq!(handle.current_l1(), later_origin);
            handle.reset(safe_head(later_origin)).await.unwrap();
            settle().await;
        })
        .await;

        assert!(stopped.is_none());
        assert_eq!(
            actor.deriver.pipeline.signals,
            [
                reset(safe_head(origin)),
                reset(safe_head(later_origin)),
                reset(safe_head(later_origin))
            ]
        );
    }

    /// A pipeline that asks for a reset resets onto the safe head it derives on.
    #[rstest]
    #[case::interop_inactive(None)]
    #[case::interop_active(Some(0))]
    #[tokio::test(start_paused = true)]
    async fn pipeline_reset_resets_the_pipeline_in_place(#[case] lagoon_time: Option<u64>) {
        let mut pipeline = TestPipeline::new();
        pipeline.rollup_config = Arc::new(RollupConfig {
            hardforks: HardForkConfig { lagoon_time, ..Default::default() },
            ..Default::default()
        });
        pipeline.reorg = true;
        let (mut actor, publishers) = actor(pipeline);

        run_until_out_of_data(&mut actor, &publishers.handle, block(0)).await;

        assert_eq!(actor.deriver.pipeline.signals, [reset(block(0)), reset(block(0))]);
        assert_eq!(actor.deriver.pipeline.stepped_on.last(), Some(&block(0)));
    }

    /// A reset error that only a bug can cause, since the pipeline is inconsistent with the safe
    /// head it derives on, stops the actor; any other resets the pipeline in place.
    #[rstest]
    #[case::bad_parent_hash(ResetError::BadParentHash(B256::ZERO, B256::ZERO), true)]
    #[case::bad_timestamp(ResetError::BadTimestamp(1, 2), true)]
    #[case::l1_origin_mismatch(ResetError::L1OriginMismatch(1, 2), true)]
    #[case::broken_time_invariant(
        ResetError::AttributesBuilder(BuilderError::BrokenTimeInvariant(
            Default::default(),
            0,
            Default::default(),
            0
        )),
        true
    )]
    #[case::block_not_found(ResetError::BlockNotFound(B256::ZERO.into()), false)]
    #[case::blobs_unavailable(ResetError::BlobsUnavailable(1), false)]
    #[tokio::test(start_paused = true)]
    async fn inconsistencies_stop_the_actor(#[case] error: ResetError, #[case] stops: bool) {
        let pipeline = TestPipeline { failure: Some(error.reset()), ..TestPipeline::new() };
        let (mut actor, publishers) = actor(pipeline);

        let stopped = run_during(&mut actor, async {
            publishers.handle.clone().reset(block(0)).await.unwrap();
            time::sleep(Duration::from_secs(60)).await;
        })
        .await;

        if stops {
            assert!(matches!(stopped, Some(ActorError::Inconsistent(_))));
        } else {
            assert!(stopped.is_none());
            assert_eq!(actor.deriver.pipeline.signals, [reset(block(0)), reset(block(0))]);
        }
    }

    /// A reset interrupts derivation in progress, which the pipeline reset then discards.
    #[tokio::test(start_paused = true)]
    async fn reset_interrupts_a_step_in_progress() {
        let pipeline = TestPipeline { hang: true, ..TestPipeline::new() };
        let (mut actor, publishers) = actor(pipeline);
        let mut handle = publishers.handle.clone();

        run_during(&mut actor, async {
            handle.reset(block(0)).await.unwrap();
            settle().await;
            handle.reset(block(3)).await.unwrap();
            settle().await;
        })
        .await;

        assert_eq!(actor.deriver.pipeline.signals, [reset(block(0)), reset(block(3))]);
        assert_eq!(actor.deriver.pipeline.stepped_on, [block(0), block(3)]);
        assert!(actor.deriver.pipeline.out_of_data);
    }

    /// A pipeline reset that fails, other than critically, is retried after an interval, and the
    /// engine never waits for it.
    #[rstest]
    #[case::temporary(PipelineError::Eof.temp())]
    #[case::cannot_reset(ResetError::BlockNotFound(B256::ZERO.into()).reset())]
    #[tokio::test(start_paused = true)]
    async fn failed_pipeline_reset_is_retried(#[case] error: PipelineErrorKind) {
        let pipeline =
            TestPipeline { reset_errors: VecDeque::from([error]), ..TestPipeline::new() };
        let (mut actor, publishers) = actor(pipeline);

        run_until_out_of_data(&mut actor, &publishers.handle, block(3)).await;

        assert_eq!(actor.deriver.pipeline.signals, [reset(block(3))]);
    }

    /// A pipeline reset that keeps failing is retried onto the safe head of a newer reset.
    #[tokio::test(start_paused = true)]
    async fn pipeline_reset_retries_onto_a_newer_reset() {
        let pipeline = TestPipeline {
            reset_errors: VecDeque::from([PipelineError::Eof.temp()]),
            ..TestPipeline::new()
        };
        let (mut actor, publishers) = actor(pipeline);
        let mut handle = publishers.handle.clone();

        run_during(&mut actor, async {
            handle.reset(block(3)).await.unwrap();
            time::sleep(PIPELINE_RESET_RETRY_INTERVAL / 2).await;
            handle.reset(block(5)).await.unwrap();
            settle().await;
        })
        .await;

        assert_eq!(actor.deriver.pipeline.signals, [reset(block(5))]);
    }

    /// A pipeline reset that fails critically stops the actor.
    #[tokio::test(start_paused = true)]
    async fn critical_pipeline_reset_failure_stops_the_actor() {
        let pipeline = TestPipeline {
            reset_errors: VecDeque::from([PipelineError::MissingOrigin.crit()]),
            ..TestPipeline::new()
        };
        let (mut actor, publishers) = actor(pipeline);

        let stopped = run_during(&mut actor, async {
            publishers.handle.clone().reset(block(3)).await.unwrap();
            future::pending().await
        })
        .await;

        assert!(matches!(stopped, Some(ActorError::Pipeline(_))));
    }

    /// Holocene's activation signals the pipeline with the safe head, and derivation continues.
    #[tokio::test(start_paused = true)]
    async fn holocene_activation_signals_the_pipeline() {
        let pipeline = TestPipeline { holocene_activation: true, ..TestPipeline::new() };
        let (mut actor, publishers) = actor(pipeline);

        run_until_out_of_data(&mut actor, &publishers.handle, block(3)).await;

        assert!(matches!(
            actor.deriver.pipeline.signals[..],
            [Signal::Reset(_), Signal::Activation(activation)] if activation.l2_safe_head == block(3)
        ));
    }

    /// Derivation publishes the attributes on the safe head it was given, and derives the next
    /// ones on the safe head the engine gives it next, after flushing the channel of a block the
    /// engine replaced.
    #[rstest]
    #[case::adopted(false)]
    #[case::replaced(true)]
    #[tokio::test(start_paused = true)]
    async fn derives_on_each_safe_head_the_engine_gives(#[case] replaced: bool) {
        let pipeline = TestPipeline::preparing([attributes(L2BlockInfo::default())]);
        let (mut actor, publishers) = actor(pipeline);
        let mut handle = publishers.handle.clone();

        let stopped = run_during(&mut actor, async {
            handle.reset(L2BlockInfo::default()).await.unwrap();
            let derived = handle.next(L2BlockInfo::default()).await.unwrap();
            assert_eq!(derived, Derived::Attributes(attributes(L2BlockInfo::default())));
            // Asking again returns the same attributes.
            assert_eq!(handle.next(L2BlockInfo::default()).await.unwrap(), derived);
            if replaced {
                handle.flush().await.unwrap();
            }
            // The engine adopted the attributes as block 1.
            let next = time::timeout(Duration::from_secs(10), handle.next(block(1))).await;
            assert!(next.is_err(), "the pipeline ran out of data");
        })
        .await;

        assert!(stopped.is_none());
        let flushed = if replaced { vec![Signal::FlushChannel] } else { vec![] };
        assert_eq!(actor.deriver.pipeline.signals[1..], flushed);
        assert_eq!(actor.deriver.pipeline.stepped_on.last(), Some(&block(1)));
    }

    /// How the unsafe chain's block at the derived attributes' height compares with them.
    #[derive(Debug, Clone, Copy)]
    enum Unsafe {
        /// The execution layer has no block there.
        Missing,
        /// Its block matches the attributes.
        Matching,
        /// Its block matches them, once reads stop failing.
        MatchingAfterFailures,
        /// Its block does not match them.
        Mismatching,
    }

    /// Derivation compares the attributes it derives with the unsafe chain: a block that matches
    /// them is made safe as it is, with the L1 block the attributes were derived from, and
    /// otherwise the engine builds the attributes. A failed read is retried rather than built,
    /// which would truncate the unsafe chain.
    #[rstest]
    #[case::missing(Unsafe::Missing)]
    #[case::matching(Unsafe::Matching)]
    #[case::matching_after_failures(Unsafe::MatchingAfterFailures)]
    #[case::mismatching(Unsafe::Mismatching)]
    #[tokio::test(start_paused = true)]
    async fn compares_attributes_with_the_unsafe_chain(#[case] unsafe_block: Unsafe) {
        let header = RpcHeader::new(alloy_consensus::Header::default());
        let block_hash = header.hash;
        let block: Block<Transaction> = Block::new(header, BlockTransactions::Full(vec![]));
        // Attributes that match the default block, which is genesis, so its info needs no L1-info
        // deposit.
        let source = BlockInfo { number: 7, ..Default::default() };
        let gas_limit = match unsafe_block {
            Unsafe::Mismatching => 1,
            Unsafe::Missing | Unsafe::Matching | Unsafe::MatchingAfterFailures => 0,
        };
        let derived = OpAttributesWithParent::new(
            OpPayloadAttributes { gas_limit: Some(gas_limit), ..Default::default() },
            L2BlockInfo::default(),
            Some(source),
            false,
        );
        let mut pipeline = TestPipeline::preparing([derived.clone()]);
        let mut config = RollupConfig::default();
        config.genesis.l2 = alloy_eips::BlockNumHash { number: 0, hash: block_hash };
        pipeline.rollup_config = Arc::new(config.clone());
        let (mut actor, publishers) = actor(pipeline);
        if !matches!(unsafe_block, Unsafe::Missing) {
            publishers.l2.0.lock().unwrap().insert(derived.block_number(), block.clone());
        }
        if matches!(unsafe_block, Unsafe::MatchingAfterFailures) {
            *publishers.l2.1.lock().unwrap() = 3;
        }
        let mut handle = publishers.handle.clone();

        run_during(&mut actor, async {
            handle.reset(L2BlockInfo::default()).await.unwrap();
            let next = handle.next(L2BlockInfo::default()).await.unwrap();
            match unsafe_block {
                Unsafe::Matching | Unsafe::MatchingAfterFailures => {
                    let block =
                        block.into_consensus().map_transactions(|tx| tx.inner.inner.into_inner());
                    let head =
                        L2BlockInfo::from_block_and_genesis(&block, &config.genesis).unwrap();
                    assert_eq!(next, Derived::Safe(Safe { head, source }));
                }
                Unsafe::Missing | Unsafe::Mismatching => {
                    assert_eq!(next, Derived::Attributes(derived));
                }
            }
        })
        .await;
    }

    /// A safe head the engine published before a reset is stale: derivation does not move onto it
    /// once it has derived on the reset's safe head.
    #[tokio::test(start_paused = true)]
    async fn a_safe_head_published_before_a_reset_is_stale() {
        let pipeline = TestPipeline {
            hang: true,
            ..TestPipeline::preparing([attributes(L2BlockInfo::default())])
        };
        let (mut actor, publishers) = actor(pipeline);
        let mut handle = publishers.handle.clone();

        run_during(&mut actor, async {
            handle.reset(block(0)).await.unwrap();
            // The engine asks on another safe head while the first step hangs, then resets.
            let stale = time::timeout(Duration::from_secs(1), handle.next(block(5))).await;
            assert!(stale.is_err());
            handle.reset(block(3)).await.unwrap();
            settle().await;
        })
        .await;

        assert_eq!(actor.deriver.pipeline.signals, [reset(block(0)), reset(block(3))]);
        assert!(!actor.deriver.pipeline.stepped_on.contains(&block(5)));
    }

    /// A reset discards the attributes derived before it.
    #[tokio::test(start_paused = true)]
    async fn reset_discards_attributes_derived_before_it() {
        let pipeline = TestPipeline::preparing([attributes(L2BlockInfo::default())]);
        let (mut actor, publishers) = actor(pipeline);
        let mut handle = publishers.handle.clone();

        run_during(&mut actor, async {
            handle.reset(L2BlockInfo::default()).await.unwrap();
            handle.next(L2BlockInfo::default()).await.unwrap();
            handle.reset(L2BlockInfo::default()).await.unwrap();
            // The pipeline prepared nothing more since.
            let derived =
                time::timeout(Duration::from_secs(10), handle.next(L2BlockInfo::default())).await;
            assert!(derived.is_err());
        })
        .await;
    }

    /// Each block L1 adds wakes derivation, which advances its origin along L1.
    #[tokio::test(start_paused = true)]
    async fn blocks_l1_adds_wake_derivation() {
        let (mut actor, publishers) = actor(TestPipeline::new());
        actor.deriver.pipeline.l1 = Some(publishers.l1.handle());
        let mut handle = publishers.handle.clone();

        let stopped = run_during(&mut actor, async {
            handle.reset(L2BlockInfo::default()).await.unwrap();
            settle().await;
            for number in 1..=3 {
                publishers.l1.set_head(&[l1_child(number, 1, 1)]);
                settle().await;
                assert_eq!(handle.current_l1(), l1_child(number, 1, 1));
            }
            // Blocks added together are derived from in turn.
            publishers.l1.set_head(&[l1_child(4, 1, 1), l1_child(5, 1, 1)]);
            settle().await;
            assert_eq!(handle.current_l1(), l1_child(5, 1, 1));
        })
        .await;

        assert!(stopped.is_none());
        assert!(actor.deriver.pipeline.out_of_data);
    }

    /// A block L1 adds while the engine adopts attributes makes derivation step the pipeline again
    /// once it has derived from all of L1.
    #[tokio::test(start_paused = true)]
    async fn a_block_added_during_derivation_resumes_it_afterwards() {
        let (mut actor, publishers) =
            actor(TestPipeline::preparing([attributes(L2BlockInfo::default())]));
        actor.deriver.pipeline.l1 = Some(publishers.l1.handle());
        let mut handle = publishers.handle.clone();

        run_during(&mut actor, async {
            handle.reset(L2BlockInfo::default()).await.unwrap();
            handle.next(L2BlockInfo::default()).await.unwrap();
            publishers.l1.set_head(&[l1_child(1, 1, 1)]);
            let next = time::timeout(Duration::from_secs(10), handle.next(block(1))).await;
            assert!(next.is_err(), "the pipeline ran out of data");
        })
        .await;

        assert_eq!(actor.deriver.pipeline.origin, Some(l1_child(1, 1, 1)));
    }

    /// L1 reorging out the pipeline's origin, while derivation waits on it, resets the pipeline
    /// onto the safe head it derives on.
    #[tokio::test(start_paused = true)]
    async fn a_reorged_origin_resets_the_pipeline() {
        let (mut actor, publishers) = actor(TestPipeline::new());
        actor.deriver.pipeline.l1 = Some(publishers.l1.handle());
        let mut handle = publishers.handle.clone();

        let stopped = run_during(&mut actor, async {
            handle.reset(L2BlockInfo::default()).await.unwrap();
            publishers.l1.set_head(&[l1_child(1, 1, 1)]);
            settle().await;
            publishers.l1.set_head(&[l1_child(1, 2, 2), l1_child(2, 2, 2)]);
            settle().await;
        })
        .await;

        assert!(stopped.is_none());
        assert_eq!(
            actor.deriver.pipeline.signals,
            [reset(L2BlockInfo::default()), reset(L2BlockInfo::default())]
        );
    }

    /// A temporary failure while L1 has blocks after the pipeline's origin is retried after an
    /// interval.
    #[tokio::test(start_paused = true)]
    async fn a_temporary_failure_with_l1_data_left_is_retried() {
        let pipeline = TestPipeline { temporary_failure: true, ..TestPipeline::new() };
        let (mut actor, publishers) = actor(pipeline);
        publishers.l1.set_head(&[l1_child(1, 1, 1)]);
        actor.deriver.pipeline.l1 = Some(publishers.l1.handle());

        run_until_out_of_data(&mut actor, &publishers.handle, L2BlockInfo::default()).await;

        // The step that failed, the one after the interval, and the one that ran out of data.
        assert_eq!(actor.deriver.pipeline.stepped_on.len(), 3);
        assert_eq!(actor.deriver.pipeline.origin, Some(l1_child(1, 1, 1)));
    }

    /// A temporary failure while L1 has no block after the pipeline's origin waits for one.
    #[tokio::test(start_paused = true)]
    async fn a_temporary_failure_without_l1_data_left_waits_for_l1() {
        let pipeline = TestPipeline { temporary_failure: true, ..TestPipeline::new() };
        let (mut actor, publishers) = actor(pipeline);
        actor.deriver.pipeline.l1 = Some(publishers.l1.handle());

        let stopped = run_during(&mut actor, async {
            publishers.handle.clone().reset(L2BlockInfo::default()).await.unwrap();
            time::sleep(STEP_RETRY_INTERVAL * 10).await;
        })
        .await;

        assert!(stopped.is_none());
        // Only the step that failed: L1 never extended the origin.
        assert_eq!(actor.deriver.pipeline.stepped_on.len(), 1);
    }

    #[tokio::test(start_paused = true)]
    async fn a_stopped_l1_actor_is_fatal() {
        let (mut actor, publishers) = actor(TestPipeline::new());
        let Publishers { l1, mut handle, .. } = publishers;
        drop(l1);

        let stopped = run_during(&mut actor, async {
            handle.reset(L2BlockInfo::default()).await.unwrap();
            future::pending().await
        })
        .await;

        assert!(matches!(stopped, Some(ActorError::L1Stopped)));
    }
}
