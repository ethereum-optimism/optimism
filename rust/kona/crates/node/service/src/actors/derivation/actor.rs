//! [`NodeActor`] implementation for the derivation sub-routine.

use crate::{
    DerivationActorRequest, DerivationEngineClient, DerivationState, DerivationStateMachine,
    DerivationStateTransitionError, DerivationStateUpdate, Metrics, NodeActor,
};
use alloy_eips::BlockNumHash;
use async_trait::async_trait;
use kona_chainview::{ChainViewClient, ChainViewSnapshot, Fact, L1StatusKind, L2SafeFact};
use kona_derive::{
    ActivationSignal, Pipeline, PipelineError, PipelineErrorKind, ResetError, Signal,
    SignalReceiver, StepResult,
};
use kona_engine::FinalizeBlockId;
use kona_protocol::{BlockInfo, OpAttributesWithParent};
use thiserror::Error;
use tokio::sync::{mpsc, watch};

/// Attributes sent to the engine and awaiting its confirmation, paired with the L1 block they
/// were derived from.
#[derive(Debug, Clone, Copy)]
struct PendingAttributes {
    /// The L2 block number the attributes build.
    number: u64,
    /// The pipeline origin when the attributes were produced.
    derived_from: BlockInfo,
}

/// The [`NodeActor`] for the derivation sub-routine.
///
/// This actor is responsible for receiving messages from [`NodeActor`]s and stepping the
/// derivation pipeline forward to produce new payload attributes. The actor then sends the payload
/// to the [`NodeActor`] responsible for the execution sub-routine.
#[derive(Debug)]
pub struct DerivationActor<DerivationEngineClient_, PipelineSignalReceiver>
where
    DerivationEngineClient_: DerivationEngineClient,
    PipelineSignalReceiver: Pipeline + SignalReceiver,
{
    /// The channel on which all inbound requests are received by the [`DerivationActor`].
    inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
    /// The Engine client used to interact with the engine.
    engine_client: DerivationEngineClient_,

    /// The derivation pipeline.
    pipeline: PipelineSignalReceiver,
    /// The state machine controlling when derivation can occur.
    derivation_state_machine: DerivationStateMachine,
    /// The chain view: derived blocks become its facts, and finalization follows its
    /// `finalized_l2` view.
    chainview: ChainViewClient,
    /// Wakes the actor when the chain view publishes a new snapshot.
    snapshots: watch::Receiver<ChainViewSnapshot>,
    /// The attributes awaiting engine confirmation.
    pending_derived_from: Option<PendingAttributes>,
    /// Monotone sequence number for `l2_safe_blocks` rows.
    derived_seq: u64,
    /// The last block a finalize command was sent for, so each is sent once and never lower.
    last_finalize_sent: Option<BlockNumHash>,
}

impl<DerivationEngineClient_, PipelineSignalReceiver>
    DerivationActor<DerivationEngineClient_, PipelineSignalReceiver>
where
    DerivationEngineClient_: DerivationEngineClient,
    PipelineSignalReceiver: Pipeline + SignalReceiver,
{
    /// Creates a new instance of the [`DerivationActor`].
    pub fn new(
        engine_client: DerivationEngineClient_,
        inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
        pipeline: PipelineSignalReceiver,
        chainview: ChainViewClient,
    ) -> Self {
        Self {
            pipeline,
            inbound_request_rx,
            engine_client,
            derivation_state_machine: DerivationStateMachine::default(),
            snapshots: chainview.subscribe(),
            chainview,
            pending_derived_from: None,
            derived_seq: 0,
            last_finalize_sent: None,
        }
    }

    /// Pushes a fact to the chain view.
    async fn push_fact(&self, fact: Fact) -> Result<(), DerivationError> {
        self.chainview.push(fact).await.map_err(DerivationError::ChainView)
    }

    /// Records the derived-from L1 block of the attributes just sent to the engine.
    fn note_pending_attributes(&mut self, attrs: &OpAttributesWithParent) {
        match attrs.derived_from {
            Some(derived_from) => {
                self.pending_derived_from =
                    Some(PendingAttributes { number: attrs.block_number(), derived_from });
            }
            None => warn!(
                target: "derivation",
                l2_block = attrs.block_number(),
                "derived attributes carry no L1 origin; the chain view will not see this block"
            ),
        }
    }

    /// Pairs an engine-confirmed safe head with the pending attributes' derived-from block and
    /// pushes the result to the chain view.
    async fn record_confirmed_safe_head(
        &mut self,
        safe_head: &kona_protocol::L2BlockInfo,
    ) -> Result<(), DerivationError> {
        let Some(PendingAttributes { number, derived_from }) = self.pending_derived_from else {
            return Ok(());
        };
        if safe_head.block_info.number < number {
            return Ok(());
        }
        self.pending_derived_from = None;
        if safe_head.block_info.number != number {
            warn!(
                target: "derivation",
                expected = number,
                confirmed = safe_head.block_info.number,
                "engine confirmed a different block than the pending attributes"
            );
            return Ok(());
        }
        self.derived_seq += 1;
        self.push_fact(Fact::L2Safe(L2SafeFact {
            seq: self.derived_seq,
            block: *safe_head,
            derived_from,
        }))
        .await
    }

    /// Publishes an L1 origin the pipeline advanced to: the block itself and the `current`
    /// status.
    async fn note_origin(&self, origin: BlockInfo) -> Result<(), DerivationError> {
        self.push_fact(Fact::L1Origin(origin)).await?;
        self.push_fact(Fact::L1Status { kind: L1StatusKind::Current, block: origin }).await
    }

    /// After a reset, publishes the origin the pipeline rewound to; the blocks above it are
    /// re-walked and replace what the view holds at their heights if the L1 changed.
    async fn note_reset_origin(&self) -> Result<(), DerivationError> {
        match self.pipeline.origin() {
            Some(origin) => self.note_origin(origin).await,
            None => Ok(()),
        }
    }

    /// Retracts derived blocks above the reset safe head, and any block at its height with
    /// another hash, from the chain view.
    async fn note_reset(
        &mut self,
        l2_safe_head: &kona_protocol::L2BlockInfo,
    ) -> Result<(), DerivationError> {
        self.pending_derived_from = None;
        self.push_fact(Fact::L2SafeRetractAbove {
            l2_number: l2_safe_head.block_info.number,
            l2_hash: l2_safe_head.block_info.hash,
        })
        .await
    }

    /// Sends a finalize command for the chain view's `finalized_l2` block when it is new,
    /// higher than the last one sent, and not above the confirmed safe head.
    async fn react_to_snapshot(&mut self) -> Result<(), DerivationError> {
        let Some(finalized) = self.snapshots.borrow_and_update().finalized_l2 else {
            return Ok(());
        };
        let id = finalized.id;
        if self.last_finalize_sent.is_some_and(|last| id == last || id.number <= last.number) {
            return Ok(());
        }
        let safe = self.derivation_state_machine.last_confirmed_safe_head().block_info;
        if id.number > safe.number {
            debug!(target: "derivation", finalized = id.number, safe = safe.number, "chain view finality is ahead of the confirmed safe head; waiting");
            return Ok(());
        }
        // A finalize by hash of a non-canonical block is a critical engine error: never send
        // a block at the safe height whose hash is not the safe head's.
        if id.number == safe.number && id.hash != safe.hash {
            warn!(target: "derivation", finalized = ?id.hash, safe = ?safe.hash, number = id.number, "chain view names another block at the safe height; not finalizing it");
            return Ok(());
        }
        info!(target: "derivation", l2_block = id.number, l1_block = finalized.derived_from.number, "finalizing from the chain view");
        self.engine_client
            .send_finalized_l2_block(FinalizeBlockId::ByHash(id))
            .await
            .map_err(|e| DerivationError::Sender(Box::new(e)))?;
        self.last_finalize_sent = Some(id);
        Ok(())
    }

    /// Handles a [`Signal`] received over the derivation signal receiver channel.
    async fn signal(&mut self, signal: Signal) -> Result<(), DerivationError> {
        if let Signal::Reset(reset) = &signal {
            self.note_reset(&reset.l2_safe_head).await?;
        }

        match self.pipeline.signal(signal).await {
            Ok(_) => {
                info!(target: "derivation", ?signal, "[SIGNAL] Executed Successfully");
                if matches!(signal, Signal::Reset(_)) {
                    self.note_reset_origin().await?;
                }
            }
            Err(e) => {
                error!(target: "derivation", ?e, ?signal, "Failed to signal derivation pipeline")
            }
        }
        Ok(())
    }

    /// Attempts to step the derivation pipeline forward as much as possible in order to produce the
    /// next safe payload.
    async fn produce_next_attributes(&mut self) -> Result<OpAttributesWithParent, DerivationError> {
        // As we start the safe head at the disputed block's parent, we step the pipeline until the
        // first attributes are produced. All batches at and before the safe head will be
        // dropped, so the first payload will always be the disputed one.
        loop {
            match self.pipeline.step(self.derivation_state_machine.last_confirmed_safe_head()).await
            {
                StepResult::PreparedAttributes => { /* continue; attributes will be sent off. */ }
                StepResult::AdvancedOrigin => {
                    let origin =
                        self.pipeline.origin().ok_or(PipelineError::MissingOrigin.crit())?;

                    kona_macros::set!(counter, Metrics::DERIVATION_L1_ORIGIN, origin.number);
                    debug!(target: "derivation", l1_block = origin.number, "Advanced L1 origin");
                    self.note_origin(origin).await?;
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
                            return Err(DerivationError::Yield);
                        }
                        PipelineErrorKind::Reset(e) => {
                            warn!(target: "derivation", "Derivation pipeline is being reset: {e}");

                            if matches!(e, ResetError::HoloceneActivation) {
                                self.pipeline
                                    .signal(Signal::Activation(ActivationSignal {
                                        l2_safe_head: self
                                            .derivation_state_machine
                                            .last_confirmed_safe_head(),
                                    }))
                                    .await?;
                            } else {
                                if let ResetError::ReorgDetected(expected, new) = e {
                                    warn!(
                                        target: "derivation",
                                        "L1 reorg detected! Expected: {expected} | New: {new}"
                                    );

                                    kona_macros::inc!(counter, Metrics::L1_REORG_COUNT);
                                }
                                self.engine_client.reset_engine_forkchoice().await.map_err(|e| {
                                    error!(target: "derivation", ?e, "Failed to send reset request");
                                    DerivationError::Sender(Box::new(e))
                                })?;
                                self.derivation_state_machine
                                    .update(&DerivationStateUpdate::SignalNeeded)?;
                                return Err(DerivationError::Yield);
                            }
                        }
                        PipelineErrorKind::Critical(_) => {
                            error!(target: "derivation", "Critical derivation error: {e}");
                            kona_macros::inc!(counter, Metrics::DERIVATION_CRITICAL_ERROR);
                            return Err(e.into());
                        }
                    }
                }
            }

            // If there are any new attributes, send them to the execution actor.
            if let Some(attrs) = self.pipeline.next() {
                return Ok(attrs);
            }
        }
    }

    async fn handle_derivation_actor_request(
        &mut self,
        request_type: DerivationActorRequest,
    ) -> Result<(), DerivationError> {
        match request_type {
            DerivationActorRequest::ProcessEngineSignalRequest(signal) => {
                self.signal(*signal).await?;
                self.derivation_state_machine.update(&DerivationStateUpdate::SignalProcessed)?;
            }
            DerivationActorRequest::ProcessL1HeadUpdateRequest(l1_head) => {
                info!(target: "derivation", l1_head = ?*l1_head, "Processing l1 head update");

                self.derivation_state_machine.update(&DerivationStateUpdate::L1DataReceived)?;

                self.attempt_derivation().await?;
            }
            DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(safe_head) => {
                info!(target: "derivation", safe_head = ?*safe_head, "Received safe head from engine.");
                self.record_confirmed_safe_head(&safe_head).await?;
                self.derivation_state_machine
                    .update(&DerivationStateUpdate::NewAttributesConfirmed(safe_head))?;

                self.attempt_derivation().await?;
            }
            DerivationActorRequest::ProcessEngineSyncCompletionRequest(safe_head) => {
                info!(target: "derivation", "Engine finished syncing, starting derivation.");
                self.derivation_state_machine
                    .update(&DerivationStateUpdate::ELSyncCompleted(safe_head))?;

                self.attempt_derivation().await?;
            }
        }

        Ok(())
    }

    /// Attempts to process the next payload attributes.
    async fn attempt_derivation(&mut self) -> Result<(), DerivationError> {
        if self.derivation_state_machine.current_state() != DerivationState::Deriving {
            info!(target: "derivation", derivation_state=?self.derivation_state_machine, "Skipping derivation.");
            return Ok(());
        }

        info!(target: "derivation", derivation_state=?self.derivation_state_machine, "Attempting derivation.");

        // Advance the pipeline as much as possible, new data may be available or there still may be
        // payloads in the attributes queue.
        let payload_attributes = match self.produce_next_attributes().await {
            Ok(attrs) => attrs,
            Err(DerivationError::Yield) => {
                info!(target: "derivation", "Yielding derivation until more data is available.");
                self.derivation_state_machine.update(&DerivationStateUpdate::MoreDataNeeded)?;
                return Ok(());
            }
            Err(e) => {
                return Err(e);
            }
        };
        trace!(target: "derivation", ?payload_attributes, "Produced payload attributes.");

        self.derivation_state_machine.update(&DerivationStateUpdate::NewAttributesDerived(
            Box::new(payload_attributes.clone()),
        ))?;

        // Remember the derived-from L1 block until the engine confirms the block.
        self.note_pending_attributes(&payload_attributes);

        // Send payload attributes out for processing.
        self.engine_client
            .send_safe_l2_signal(payload_attributes.into())
            .await
            .map_err(|e| DerivationError::Sender(Box::new(e)))?;

        Ok(())
    }
}

#[async_trait]
impl<DerivationEngineClient_, PipelineSignalReceiver> NodeActor
    for DerivationActor<DerivationEngineClient_, PipelineSignalReceiver>
where
    DerivationEngineClient_: DerivationEngineClient + 'static,
    PipelineSignalReceiver: Pipeline + SignalReceiver + Send + Sync + 'static,
{
    type Error = DerivationError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        // Select only on which event fired; handling happens after the select so that no
        // non-`Send` error value is ever held across an await inside it.
        let wake = tokio::select! {
            biased;
            request = self.inbound_request_rx.recv() => Wake::Request(request),
            changed = self.snapshots.changed() => Wake::Snapshot { open: changed.is_ok() },
        };
        match wake {
            Wake::Request(request) => {
                let request = request.ok_or_else(|| {
                    error!(
                        target: "derivation",
                        "DerivationActor inbound request receiver closed unexpectedly",
                    );
                    DerivationError::RequestReceiveFailed
                })?;
                self.handle_derivation_actor_request(request).await
            }
            Wake::Snapshot { open: false } => Err(DerivationError::chain_view_closed()),
            Wake::Snapshot { open: true } => self.on_snapshot_changed().await,
        }
    }
}

/// What woke the derivation actor.
enum Wake {
    /// An inbound request (or the channel closed).
    Request(Option<DerivationActorRequest>),
    /// The chain view published a snapshot (`open == false`: it went away).
    Snapshot {
        /// Whether the chain view is still running.
        open: bool,
    },
}

impl<DerivationEngineClient_, PipelineSignalReceiver>
    DerivationActor<DerivationEngineClient_, PipelineSignalReceiver>
where
    DerivationEngineClient_: DerivationEngineClient,
    PipelineSignalReceiver: Pipeline + SignalReceiver,
{
    async fn on_snapshot_changed(&mut self) -> Result<(), DerivationError> {
        self.react_to_snapshot().await
    }
}

/// An error from the [`DerivationActor`].
#[derive(Error, Debug)]
pub enum DerivationError {
    /// An error originating from the derivation pipeline.
    #[error(transparent)]
    Pipeline(#[from] PipelineErrorKind),
    /// Waiting for more data to be available.
    #[error("Waiting for more data to be available")]
    Yield,
    /// An error originating from the broadcast sender.
    #[error("Failed to send event to broadcast sender: {0}")]
    Sender(Box<dyn std::error::Error>),
    /// Failed to receive inbound request
    #[error("Failed to receive inbound request")]
    RequestReceiveFailed,
    /// An invalid state transition occurred.
    #[error(transparent)]
    StateTransitionError(#[from] DerivationStateTransitionError),
    /// The chain view rejected a fact or went away.
    #[error("chain view error: {0}")]
    ChainView(kona_chainview::ChainViewError),
}

impl DerivationError {
    /// The chain view's snapshot channel closed.
    const fn chain_view_closed() -> Self {
        Self::ChainView(kona_chainview::ChainViewError::Closed)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::actors::derivation::engine_client::MockDerivationEngineClient;
    use alloy_primitives::B256;
    use kona_derive::PipelineResult;
    use kona_genesis::{HardForkConfig, RollupConfig, SystemConfig};
    use kona_protocol::{BlockInfo, L2BlockInfo};
    use rstest::rstest;
    use std::sync::Arc;

    /// A pipeline stub whose every step reports an L1 reorg, forcing the reset branch of
    /// [`DerivationActor::produce_next_attributes`].
    #[derive(Debug)]
    struct ReorgingPipeline {
        rollup_config: Arc<RollupConfig>,
    }

    impl Iterator for ReorgingPipeline {
        type Item = OpAttributesWithParent;

        fn next(&mut self) -> Option<Self::Item> {
            None
        }
    }

    impl kona_derive::OriginProvider for ReorgingPipeline {
        fn origin(&self) -> Option<BlockInfo> {
            Some(BlockInfo::default())
        }
    }

    #[async_trait]
    impl SignalReceiver for ReorgingPipeline {
        async fn signal(&mut self, _: Signal) -> PipelineResult<()> {
            Ok(())
        }
    }

    #[async_trait]
    impl Pipeline for ReorgingPipeline {
        fn peek(&self) -> Option<&OpAttributesWithParent> {
            None
        }

        async fn step(&mut self, _: L2BlockInfo) -> StepResult {
            StepResult::StepFailed(PipelineErrorKind::Reset(ResetError::ReorgDetected(
                B256::ZERO,
                B256::repeat_byte(1),
            )))
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

    /// A pipeline-driven reset must always reach the engine actor. The engine's reset is what
    /// sends the pipeline its [`Signal`] back; without it the actor parks in
    /// [`DerivationState::AwaitingSignal`] forever.
    #[rstest]
    #[case::interop_inactive(None)]
    #[case::interop_active(Some(0))]
    #[tokio::test]
    async fn test_pipeline_reset_always_resets_engine(#[case] lagoon_time: Option<u64>) {
        let rollup_config = Arc::new(RollupConfig {
            hardforks: HardForkConfig { lagoon_time, ..Default::default() },
            ..Default::default()
        });

        let mut engine_client = MockDerivationEngineClient::new();
        engine_client.expect_reset_engine_forkchoice().times(1).returning(|| Ok(()));

        let chainview =
            kona_chainview::spawn(kona_chainview::ChainViewConfig::default()).expect("chain view");
        let (request_tx, request_rx) = mpsc::channel(1);
        let mut actor = DerivationActor::new(
            engine_client,
            request_rx,
            ReorgingPipeline { rollup_config: rollup_config.clone() },
            chainview.client.clone(),
        );

        // Complete EL sync so the actor starts deriving, then let it hit the reorg.
        request_tx
            .send(DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::default()))
            .await
            .unwrap();
        actor.step().await.unwrap();

        assert_eq!(actor.derivation_state_machine.current_state(), DerivationState::AwaitingSignal);
    }
}

#[cfg(test)]
mod chainview_tests {
    use super::*;
    use crate::actors::derivation::engine_client::MockDerivationEngineClient;
    use alloy_primitives::B256;
    use kona_chainview::{ChainViewConfig, FinalizedL2, spawn};
    use kona_derive::{PipelineError, PipelineResult};
    use kona_genesis::{RollupConfig, SystemConfig};
    use kona_protocol::L2BlockInfo;
    use std::sync::Arc;

    /// A pipeline the tests never step; they call the actor's chain view hooks directly.
    #[derive(Debug)]
    struct IdlePipeline {
        rollup_config: Arc<RollupConfig>,
    }

    impl Iterator for IdlePipeline {
        type Item = OpAttributesWithParent;

        fn next(&mut self) -> Option<Self::Item> {
            None
        }
    }

    impl kona_derive::OriginProvider for IdlePipeline {
        fn origin(&self) -> Option<BlockInfo> {
            Some(BlockInfo::default())
        }
    }

    #[async_trait]
    impl SignalReceiver for IdlePipeline {
        async fn signal(&mut self, _: Signal) -> PipelineResult<()> {
            Ok(())
        }
    }

    #[async_trait]
    impl Pipeline for IdlePipeline {
        fn peek(&self) -> Option<&OpAttributesWithParent> {
            None
        }

        async fn step(&mut self, _: L2BlockInfo) -> StepResult {
            StepResult::StepFailed(PipelineErrorKind::Temporary(PipelineError::NotEnoughData))
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

    fn l1_block(number: u64) -> BlockInfo {
        BlockInfo { hash: B256::repeat_byte(0xa0 + number as u8), number, ..Default::default() }
    }

    fn l2(number: u64, tag: u8) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo {
                hash: B256::repeat_byte(tag),
                number,
                parent_hash: B256::ZERO,
                timestamp: 2 * number,
            },
            l1_origin: BlockNumHash::default(),
            seq_num: 0,
        }
    }

    fn snapshot_with_finalized(number: u64, tag: u8) -> ChainViewSnapshot {
        ChainViewSnapshot {
            finalized_l2: Some(FinalizedL2 {
                id: BlockNumHash { number, hash: B256::repeat_byte(tag) },
                derived_from: BlockNumHash::default(),
            }),
            ..Default::default()
        }
    }

    /// The finalize command follows `finalized_l2` exactly once per block, never above the
    /// confirmed safe head, never for another hash at the safe height, and never lower than
    /// the last one sent.
    #[tokio::test]
    async fn react_to_snapshot_finalizes_once_and_only_at_or_below_the_safe_head() {
        let mut engine_client = MockDerivationEngineClient::new();
        engine_client
            .expect_send_finalized_l2_block()
            .times(1)
            .withf(|id| {
                matches!(id, FinalizeBlockId::ByHash(h) if h.number == 8 && h.hash == B256::repeat_byte(8))
            })
            .returning(|_| Ok(()));
        let handle = spawn(ChainViewConfig::default()).expect("chain view");
        let (_request_tx, request_rx) = mpsc::channel(1);
        let (snapshot_tx, snapshot_rx) = watch::channel(ChainViewSnapshot::default());
        let mut actor = DerivationActor::new(
            engine_client,
            request_rx,
            IdlePipeline { rollup_config: Arc::default() },
            handle.client.clone(),
        );
        // Drive the guard from a snapshot channel the test controls.
        actor.snapshots = snapshot_rx;
        actor
            .derivation_state_machine
            .update(&DerivationStateUpdate::ELSyncCompleted(Box::new(l2(10, 10))))
            .unwrap();

        // Above the confirmed safe head: wait.
        snapshot_tx.send_replace(snapshot_with_finalized(12, 12));
        actor.react_to_snapshot().await.unwrap();
        // At the safe height with another hash: refuse.
        snapshot_tx.send_replace(snapshot_with_finalized(10, 99));
        actor.react_to_snapshot().await.unwrap();
        // At or below: send exactly once, even when published again.
        snapshot_tx.send_replace(snapshot_with_finalized(8, 8));
        actor.react_to_snapshot().await.unwrap();
        snapshot_tx.send_replace(snapshot_with_finalized(8, 8));
        actor.react_to_snapshot().await.unwrap();
        // Never lower than the last one sent.
        snapshot_tx.send_replace(snapshot_with_finalized(7, 7));
        actor.react_to_snapshot().await.unwrap();
        // Nothing to do when the view is empty.
        snapshot_tx.send_replace(ChainViewSnapshot::default());
        actor.react_to_snapshot().await.unwrap();
        assert_eq!(actor.last_finalize_sent.map(|id| id.number), Some(8));
    }

    /// An engine confirmation becomes a derived-block fact only when it names the block the
    /// pending attributes build; a lower confirmation keeps the pairing, a different block
    /// drops it.
    #[tokio::test]
    async fn record_confirmed_safe_head_pairs_the_pending_attributes() {
        let handle = spawn(ChainViewConfig::default()).expect("chain view");
        let (_request_tx, request_rx) = mpsc::channel(1);
        let mut actor = DerivationActor::new(
            MockDerivationEngineClient::new(),
            request_rx,
            IdlePipeline { rollup_config: Arc::default() },
            handle.client.clone(),
        );

        // Attributes building block 5, derived from L1 block 3.
        let attrs =
            OpAttributesWithParent::new(Default::default(), l2(4, 4), Some(l1_block(3)), false);
        actor.note_pending_attributes(&attrs);
        assert_eq!(actor.pending_derived_from.map(|p| p.number), Some(5));

        // A lower confirmation (a replay of an older head) leaves the pairing in place.
        actor.record_confirmed_safe_head(&l2(4, 4)).await.unwrap();
        assert!(actor.pending_derived_from.is_some());
        assert_eq!(actor.derived_seq, 0);

        // The matching confirmation is asserted to the chain view.
        actor.record_confirmed_safe_head(&l2(5, 5)).await.unwrap();
        assert!(actor.pending_derived_from.is_none());
        assert_eq!(actor.derived_seq, 1);
        handle.client.sync().await.unwrap();
        assert_eq!(handle.client.snapshot().history_len, 1);

        // A confirmation of another block than the pending one is not asserted.
        let attrs =
            OpAttributesWithParent::new(Default::default(), l2(5, 5), Some(l1_block(3)), false);
        actor.note_pending_attributes(&attrs);
        actor.record_confirmed_safe_head(&l2(7, 7)).await.unwrap();
        assert!(actor.pending_derived_from.is_none());
        assert_eq!(actor.derived_seq, 1);

        handle.shutdown();
    }

    /// An advanced origin becomes an L1 block of the view and its `current` status.
    #[tokio::test]
    async fn advanced_origins_become_l1_blocks() {
        let handle = spawn(ChainViewConfig::default()).expect("chain view");
        let (_request_tx, request_rx) = mpsc::channel(1);
        let actor = DerivationActor::new(
            MockDerivationEngineClient::new(),
            request_rx,
            IdlePipeline { rollup_config: Arc::default() },
            handle.client.clone(),
        );

        actor.note_origin(l1_block(3)).await.unwrap();
        actor.note_origin(l1_block(4)).await.unwrap();
        handle.client.sync().await.unwrap();
        let snapshot = handle.client.snapshot();
        assert_eq!(snapshot.l1.current, Some(l1_block(4)));
        assert_eq!(snapshot.l1_window_len, 2);

        handle.shutdown();
    }
}
