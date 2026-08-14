//! Linear asynchronous sequencing workflow.

use crate::{
    Conductor, SequencerEngineClient, UnsafePayloadGossipClient,
    actors::{
        engine::EngineClientError,
        sequencer::{
            error::SequencerActorError,
            metrics::{
                update_attributes_build_duration_metrics, update_block_build_duration_metrics,
                update_conductor_commitment_duration_metrics, update_seal_duration_metrics,
                update_total_transactions_sequenced,
            },
            origin_selector::{L1OriginSelectorError, OriginSelector},
        },
    },
};
use alloy_rpc_types_engine::PayloadId;
use kona_derive::{AttributesBuilder, PipelineErrorKind};
use kona_engine::{InsertTaskError, SealTaskError, SynchronizeTaskError};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadAttributes};
use std::{sync::Arc, time::Duration};
use tokio::time::Instant;

/// How long to wait between retries that must retain the same payload.
#[cfg(not(test))]
const DISTRIBUTION_RETRY_DELAY: Duration = Duration::from_secs(1);
#[cfg(test)]
const DISTRIBUTION_RETRY_DELAY: Duration = Duration::from_millis(1);
/// Upper bound for one conductor commit attempt.
const CONDUCTOR_COMMIT_TIMEOUT: Duration = Duration::from_secs(30);
/// Delay for temporary planning failures, avoiding an immediate retry loop.
const PLANNING_RETRY_DELAY: Duration = Duration::from_millis(200);

/// Handle to an execution-layer payload build.
#[derive(Debug)]
pub(super) struct BuildHandle {
    /// Identifier returned by `engine_forkchoiceUpdated` when the build was started.
    pub payload_id: PayloadId,
    /// Attributes and expected parent used to start the build.
    pub attributes: OpAttributesWithParent,
}

/// A payload retrieved from the execution layer but not yet authorized for publication.
#[derive(Debug)]
pub(super) struct SealedCandidate {
    /// Sealed execution payload.
    payload: OpExecutionPayloadEnvelope,
    /// Attributes and expected parent used to build the payload.
    attributes: OpAttributesWithParent,
    /// Time at which block distribution began.
    distribution_started: Instant,
}

/// Outcome of one block-sequencing action.
#[derive(Debug)]
pub(super) enum BlockSequenceOutcome {
    /// A block was published and made canonical locally.
    Canonicalized(L2BlockInfo),
    /// Planning or sealing failed temporarily; plan a fresh block after the next control boundary.
    Replan,
    /// Conductor authorization failed or timed out; retry this exact sealed payload after the next
    /// control boundary.
    Retry(Box<SealedCandidate>),
}

/// A sealed payload that has passed the configured publication gate.
///
/// This wrapper is intentionally the only input accepted by publication and canonicalization
/// helpers in this module.
#[derive(Debug)]
struct CommittedCandidate {
    sealed: Box<SealedCandidate>,
}

/// Gate that grants permission to publish a sealed candidate.
#[derive(Debug)]
enum PublicationGate<Conductor_> {
    /// No HA conductor is configured.
    Standalone,
    /// Publication requires a successful conductor commit.
    Conductor(Arc<Conductor_>),
}

impl<Conductor_: Conductor> PublicationGate<Conductor_> {
    async fn authorize(
        &self,
        payload: &OpExecutionPayloadEnvelope,
    ) -> Result<(), crate::ConductorError> {
        match self {
            Self::Standalone => Ok(()),
            Self::Conductor(conductor) => conductor.commit_unsafe_payload(payload).await,
        }
    }
}

/// State and dependencies used by the sequencing workflow.
///
/// This type is owned and polled directly by the sequencer service. It is not a separately spawned
/// worker and has no control or status channels.
#[derive(Debug)]
pub(super) struct SequencingWorkflow<
    AttributesBuilder_,
    Conductor_,
    OriginSelector_,
    SequencerEngineClient_,
    UnsafePayloadGossipClient_,
> where
    AttributesBuilder_: AttributesBuilder,
    Conductor_: Conductor,
    OriginSelector_: OriginSelector,
    SequencerEngineClient_: SequencerEngineClient,
    UnsafePayloadGossipClient_: UnsafePayloadGossipClient,
{
    attributes_builder: AttributesBuilder_,
    publication_gate: PublicationGate<Conductor_>,
    engine_client: Arc<SequencerEngineClient_>,
    last_distribution_duration: Duration,
    origin_selector: OriginSelector_,
    rollup_config: Arc<RollupConfig>,
    unsafe_payload_gossip_client: UnsafePayloadGossipClient_,
}

impl<
    AttributesBuilder_,
    Conductor_,
    OriginSelector_,
    SequencerEngineClient_,
    UnsafePayloadGossipClient_,
>
    SequencingWorkflow<
        AttributesBuilder_,
        Conductor_,
        OriginSelector_,
        SequencerEngineClient_,
        UnsafePayloadGossipClient_,
    >
where
    AttributesBuilder_: AttributesBuilder,
    Conductor_: Conductor,
    OriginSelector_: OriginSelector,
    SequencerEngineClient_: SequencerEngineClient,
    UnsafePayloadGossipClient_: UnsafePayloadGossipClient,
{
    /// Creates the sequencing workflow.
    #[allow(clippy::too_many_arguments)]
    pub(super) fn new(
        attributes_builder: AttributesBuilder_,
        conductor: Option<Arc<Conductor_>>,
        engine_client: Arc<SequencerEngineClient_>,
        origin_selector: OriginSelector_,
        rollup_config: Arc<RollupConfig>,
        unsafe_payload_gossip_client: UnsafePayloadGossipClient_,
    ) -> Self {
        Self {
            attributes_builder,
            engine_client,
            publication_gate: conductor
                .map_or_else(|| PublicationGate::Standalone, PublicationGate::Conductor),
            last_distribution_duration: Duration::ZERO,
            origin_selector,
            rollup_config,
            unsafe_payload_gossip_client,
        }
    }

    /// Runs one block-sequencing action without polling control requests.
    ///
    /// A retry candidate skips planning and sealing so conductor failures always retry the exact
    /// payload. The caller regains control between conductor attempts.
    pub(super) async fn sequence_one_block(
        &mut self,
        recovery_mode: bool,
        retry_candidate: Option<Box<SealedCandidate>>,
    ) -> Result<BlockSequenceOutcome, SequencerActorError> {
        let sealed = match retry_candidate {
            Some(sealed) => sealed,
            None => {
                let Some(sealed) = self.prepare_candidate(recovery_mode).await? else {
                    tokio::time::sleep(PLANNING_RETRY_DELAY).await;
                    return Ok(BlockSequenceOutcome::Replan);
                };
                sealed
            }
        };

        let committed = match self.commit_once(sealed).await {
            Ok(committed) => committed,
            Err(sealed) => {
                tokio::time::sleep(DISTRIBUTION_RETRY_DELAY).await;
                return Ok(BlockSequenceOutcome::Retry(sealed));
            }
        };

        self.publish(&committed).await?;
        let head = self.canonicalize_until_done(&committed).await?;

        update_total_transactions_sequenced(committed.sealed.attributes.count_transactions());
        self.last_distribution_duration = committed.sealed.distribution_started.elapsed();

        Ok(BlockSequenceOutcome::Canonicalized(head))
    }

    /// Plans, builds, waits for, and seals one unpublished block candidate.
    async fn prepare_candidate(
        &mut self,
        recovery_mode: bool,
    ) -> Result<Option<Box<SealedCandidate>>, SequencerActorError> {
        let Some(build) = self.build_payload(recovery_mode).await? else {
            return Ok(None);
        };

        self.wait_until_seal_time(&build).await;

        let distribution_started = Instant::now();
        let seal_start = Instant::now();
        let payload = match self
            .engine_client
            .seal_block(build.payload_id, build.attributes.clone())
            .await
        {
            Ok(payload) => payload,
            Err(EngineClientError::SealError(err)) if !is_seal_task_err_fatal(&err) => {
                warn!(target: "sequencer", ?err, "Discarding uncommitted block attempt after seal failure");
                return Ok(None);
            }
            Err(err) => return Err(err.into()),
        };
        update_seal_duration_metrics(seal_start.elapsed());

        Ok(Some(Box::new(SealedCandidate {
            payload,
            attributes: build.attributes,
            distribution_started,
        })))
    }

    /// Builds payload attributes and starts an execution-layer build job.
    pub(super) async fn build_payload(
        &mut self,
        recovery_mode: bool,
    ) -> Result<Option<BuildHandle>, SequencerActorError> {
        let unsafe_head = self.engine_client.get_unsafe_head().await?;

        let Some(l1_origin) = self.get_next_payload_l1_origin(unsafe_head, recovery_mode).await?
        else {
            return Ok(None);
        };

        info!(
            target: "sequencer",
            parent_num = unsafe_head.block_info.number,
            l1_origin_num = l1_origin.number,
            "Started sequencing new block"
        );

        let attributes_build_start = Instant::now();
        let Some(attributes) = self.build_attributes(unsafe_head, l1_origin, recovery_mode).await?
        else {
            return Ok(None);
        };
        update_attributes_build_duration_metrics(attributes_build_start.elapsed());

        let build_request_start = Instant::now();
        let payload_id = self.engine_client.start_build_block(attributes.clone()).await?;
        update_block_build_duration_metrics(build_request_start.elapsed());

        Ok(Some(BuildHandle { payload_id, attributes }))
    }

    /// Waits until the payload timestamp.
    async fn wait_until_seal_time(&self, build: &BuildHandle) {
        let block_timestamp = build
            .attributes
            .parent()
            .block_info
            .timestamp
            .saturating_add(self.rollup_config.block_time);
        let target = std::time::UNIX_EPOCH + Duration::from_secs(block_timestamp) -
            self.last_distribution_duration;
        let delay = target.duration_since(std::time::SystemTime::now()).unwrap_or_default();
        tokio::time::sleep(delay).await;
    }

    /// Attempts to commit one exact sealed payload once.
    ///
    /// A timeout is ambiguous, so failures return ownership of the candidate for an exact retry.
    async fn commit_once(
        &self,
        sealed: Box<SealedCandidate>,
    ) -> Result<CommittedCandidate, Box<SealedCandidate>> {
        let started = Instant::now();
        let result = tokio::time::timeout(
            CONDUCTOR_COMMIT_TIMEOUT,
            self.publication_gate.authorize(&sealed.payload),
        )
        .await;
        update_conductor_commitment_duration_metrics(started.elapsed());

        match result {
            Ok(Ok(())) => Ok(CommittedCandidate { sealed }),
            Ok(Err(err)) => {
                error!(target: "sequencer", ?err, "Conductor commit failed; retaining sealed payload");
                Err(sealed)
            }
            Err(_) => {
                error!(target: "sequencer", "Conductor commit timed out; retaining sealed payload");
                Err(sealed)
            }
        }
    }

    /// Queues only an authorized payload for network publication.
    async fn publish(&self, committed: &CommittedCandidate) -> Result<(), SequencerActorError> {
        self.unsafe_payload_gossip_client
            .schedule_execution_payload_gossip(committed.sealed.payload.clone())
            .await
            .map_err(Into::into)
    }

    /// Retries local canonicalization of a committed payload without ever rebuilding it.
    async fn canonicalize_until_done(
        &self,
        committed: &CommittedCandidate,
    ) -> Result<L2BlockInfo, SequencerActorError> {
        loop {
            match self
                .engine_client
                .canonicalize_block(
                    committed.sealed.payload.clone(),
                    committed.sealed.attributes.clone(),
                )
                .await
            {
                Ok(head) => return Ok(head),
                Err(EngineClientError::SealError(err)) if !is_seal_task_err_fatal(&err) => {
                    warn!(target: "sequencer", ?err, "Canonicalization failed after commit; retaining payload");
                    tokio::time::sleep(DISTRIBUTION_RETRY_DELAY).await;
                }
                Err(err) => return Err(err.into()),
            }
        }
    }

    /// Determines and validates the next L1 origin.
    async fn get_next_payload_l1_origin(
        &mut self,
        unsafe_head: L2BlockInfo,
        recovery_mode: bool,
    ) -> Result<Option<BlockInfo>, SequencerActorError> {
        let l1_origin = match self.origin_selector.next_l1_origin(unsafe_head, recovery_mode).await
        {
            Ok(l1_origin) => l1_origin,
            Err(L1OriginSelectorError::OriginNotFound(hash)) => {
                warn!(target: "sequencer", %hash, "L1 origin block not found, resetting engine");
                self.engine_client.reset_engine_forkchoice().await?;
                return Ok(None);
            }
            Err(err) => {
                warn!(target: "sequencer", ?err, "Temporary error selecting L1 origin");
                return Ok(None);
            }
        };

        if unsafe_head.l1_origin.hash != l1_origin.parent_hash &&
            unsafe_head.l1_origin.hash != l1_origin.hash
        {
            warn!(
                target: "sequencer",
                l1_origin = ?l1_origin,
                unsafe_head_l1_origin = ?unsafe_head.l1_origin,
                "Cannot build on inconsistent L1 origin, resetting engine"
            );
            self.engine_client.reset_engine_forkchoice().await?;
            return Ok(None);
        }

        Ok(Some(l1_origin))
    }

    /// Builds payload attributes for the next block.
    async fn build_attributes(
        &mut self,
        unsafe_head: L2BlockInfo,
        l1_origin: BlockInfo,
        recovery_mode: bool,
    ) -> Result<Option<OpAttributesWithParent>, SequencerActorError> {
        let mut attributes = match self
            .attributes_builder
            .prepare_payload_attributes(unsafe_head, l1_origin.id())
            .await
        {
            Ok(attributes) => attributes,
            Err(PipelineErrorKind::Temporary(_)) => return Ok(None),
            Err(PipelineErrorKind::Reset(_)) => {
                self.engine_client.reset_engine_forkchoice().await?;
                warn!(target: "sequencer", "Reset engine after attributes-builder reset error");
                return Ok(None);
            }
            Err(err @ PipelineErrorKind::Critical(_)) => return Err(err.into()),
        };

        attributes.no_tx_pool =
            Some(!self.should_use_tx_pool(l1_origin, &attributes, recovery_mode));
        Ok(Some(OpAttributesWithParent::new(attributes, unsafe_head, None, false)))
    }

    /// Determines whether the execution layer may include transaction-pool transactions.
    fn should_use_tx_pool(
        &self,
        l1_origin: BlockInfo,
        attributes: &OpPayloadAttributes,
        recovery_mode: bool,
    ) -> bool {
        if recovery_mode {
            warn!(target: "sequencer", "Recovery mode active; producing empty block");
            return false;
        }

        let timestamp = attributes.payload_attributes.timestamp;
        if timestamp >
            l1_origin.timestamp + self.rollup_config.max_sequencer_drift(l1_origin.timestamp)
        {
            return false;
        }

        // Upgrade blocks must not contain transaction-pool transactions.
        !(self.rollup_config.is_first_ecotone_block(timestamp) ||
            self.rollup_config.is_first_fjord_block(timestamp) ||
            self.rollup_config.is_first_granite_block(timestamp) ||
            self.rollup_config.is_first_holocene_block(timestamp) ||
            self.rollup_config.is_first_isthmus_block(timestamp) ||
            self.rollup_config.is_first_jovian_block(timestamp) ||
            self.rollup_config.is_first_karst_block(timestamp) ||
            self.rollup_config.is_first_interop_block(timestamp))
    }
}

/// Determines whether a seal/canonicalization error violates an invariant rather than representing
/// a retryable execution-layer failure.
fn is_seal_task_err_fatal(err: &SealTaskError) -> bool {
    match err {
        SealTaskError::PayloadInsertionFailed(insert_err) => match &**insert_err {
            InsertTaskError::ForkchoiceUpdateFailed(synchronize_error) => match synchronize_error {
                SynchronizeTaskError::FinalizedAheadOfUnsafe(_, _) => true,
                SynchronizeTaskError::ForkchoiceUpdateFailed(_) |
                SynchronizeTaskError::InvalidForkchoiceState |
                SynchronizeTaskError::UnexpectedPayloadStatus(_) => false,
            },
            InsertTaskError::FromBlockError(_) | InsertTaskError::L2BlockInfoConstruction(_) => {
                true
            }
            InsertTaskError::InsertFailed(_) | InsertTaskError::UnexpectedPayloadStatus(_) => false,
        },
        SealTaskError::GetPayloadFailed(_) |
        SealTaskError::HoloceneInvalidFlush |
        SealTaskError::UnsafeHeadChangedSinceBuild => false,
        SealTaskError::DepositOnlyPayloadFailed |
        SealTaskError::DepositOnlyPayloadReattemptFailed |
        SealTaskError::MpscSend(_) |
        SealTaskError::ClockWentBackwards => true,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        ConductorError, MockConductor, MockOriginSelector, MockSequencerEngineClient,
        MockUnsafePayloadGossipClient,
    };
    use alloy_consensus::Block;
    use alloy_rpc_types_engine::ExecutionPayloadV1;
    use alloy_transport::RpcError;
    use kona_derive::test_utils::TestAttributesBuilder;
    use mockall::Sequence;
    use op_alloy_consensus::OpTxEnvelope;

    #[tokio::test]
    async fn conductor_commit_gates_gossip_and_canonicalization() {
        let unsafe_head = L2BlockInfo::default();
        let canonical_head = L2BlockInfo {
            block_info: BlockInfo { number: 1, ..Default::default() },
            ..Default::default()
        };
        let payload =
            OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(&Block::<
                OpTxEnvelope,
            >::default(
            )));
        let payload_id = PayloadId::default();

        let mut sequence = Sequence::new();
        let mut engine = MockSequencerEngineClient::new();
        engine
            .expect_get_unsafe_head()
            .times(1)
            .in_sequence(&mut sequence)
            .return_once(move || Ok(unsafe_head));
        engine
            .expect_start_build_block()
            .times(1)
            .in_sequence(&mut sequence)
            .return_once(move |_| Ok(payload_id));
        let sealed_payload = payload.clone();
        engine
            .expect_seal_block()
            .times(1)
            .in_sequence(&mut sequence)
            .return_once(move |_, _| Ok(sealed_payload));

        let mut conductor = MockConductor::new();
        conductor.expect_commit_unsafe_payload().times(1).in_sequence(&mut sequence).return_once(
            |_| Err(ConductorError::Rpc(RpcError::local_usage_str("commit unavailable"))),
        );
        conductor
            .expect_commit_unsafe_payload()
            .times(1)
            .in_sequence(&mut sequence)
            .return_once(|_| Ok(()));

        let mut gossip = MockUnsafePayloadGossipClient::new();
        gossip
            .expect_schedule_execution_payload_gossip()
            .times(1)
            .in_sequence(&mut sequence)
            .return_once(|_| Ok(()));

        engine
            .expect_canonicalize_block()
            .times(1)
            .in_sequence(&mut sequence)
            .return_once(move |_, _| Ok(canonical_head));

        let mut origin_selector = MockOriginSelector::new();
        origin_selector
            .expect_next_l1_origin()
            .times(1)
            .return_once(|_, _| Ok(BlockInfo::default()));

        let attributes_builder =
            TestAttributesBuilder { attributes: vec![Ok(OpPayloadAttributes::default())] };
        let mut workflow = SequencingWorkflow::new(
            attributes_builder,
            Some(Arc::new(conductor)),
            Arc::new(engine),
            origin_selector,
            Arc::new(RollupConfig { block_time: 2, ..Default::default() }),
            gossip,
        );

        let candidate = match workflow.sequence_one_block(false, None).await.unwrap() {
            BlockSequenceOutcome::Retry(candidate) => candidate,
            outcome => panic!("expected exact conductor retry, got {outcome:?}"),
        };
        let outcome = workflow.sequence_one_block(false, Some(candidate)).await.unwrap();
        assert!(matches!(
            outcome,
            BlockSequenceOutcome::Canonicalized(head) if head == canonical_head
        ));
    }
}
