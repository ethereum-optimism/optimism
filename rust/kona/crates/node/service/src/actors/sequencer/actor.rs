//! The long-running [`SequencerActor`] service.

use crate::{
    Conductor, SequencerAdminQuery, SequencerEngineClient, UnsafePayloadGossipClient,
    actors::sequencer::{
        error::SequencerActorError,
        origin_selector::OriginSelector,
        workflow::{SealedCandidate, SequencingWorkflow},
    },
};
use kona_derive::AttributesBuilder;
use kona_genesis::RollupConfig;
use kona_rpc::SequencerAdminAPIError;
use std::sync::Arc;
use tokio::sync::{mpsc, oneshot};
use tokio_util::sync::CancellationToken;

/// Result of racing cancellable block preparation against control-plane events.
#[derive(Debug)]
enum PreparationEvent {
    /// An unpublished candidate is ready to enter protected distribution.
    Candidate(Result<Box<SealedCandidate>, SequencerActorError>),
    /// An admin operation requires the unpublished preparation future to be cancelled first.
    Interrupt(SequencerAdminQuery),
    /// Node shutdown was requested before the publication boundary.
    Shutdown,
}

/// Admin operation deferred until protected payload distribution completes.
#[derive(Debug)]
enum DeferredAdminQuery {
    /// Stop response waiting for the final distributed head.
    Stop(oneshot::Sender<Result<alloy_primitives::B256, SequencerAdminAPIError>>),
    /// Engine reset that must not race payload canonicalization.
    Reset(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
}

/// The sequencer's control plane and linear block-production workflow.
///
/// Unlike the node's step-driven actors, the sequencer owns one long-running async event loop. It
/// races admin requests against cancellation-safe block preparation, then shields conductor commit,
/// gossip, and canonicalization from cancellation once publication may have begun.
#[derive(Debug)]
pub struct SequencerActor<
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
    /// Receiver for admin API requests.
    pub admin_api_rx: mpsc::Receiver<SequencerAdminQuery>,
    /// Optional conductor client shared with admin operations.
    pub conductor: Option<Arc<Conductor_>>,
    /// Engine client shared with admin operations.
    pub engine_client: Arc<SequencerEngineClient_>,
    /// Whether sequencing is desired.
    pub is_active: bool,
    /// Whether recovery-mode blocks are desired.
    pub in_recovery_mode: bool,

    pub(super) workflow: SequencingWorkflow<
        AttributesBuilder_,
        Conductor_,
        OriginSelector_,
        SequencerEngineClient_,
        UnsafePayloadGossipClient_,
    >,
}

impl<
    AttributesBuilder_,
    Conductor_,
    OriginSelector_,
    SequencerEngineClient_,
    UnsafePayloadGossipClient_,
>
    SequencerActor<
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
    /// Instantiates the sequencer service.
    #[allow(clippy::too_many_arguments)]
    pub fn new(
        admin_api_rx: mpsc::Receiver<SequencerAdminQuery>,
        attributes_builder: AttributesBuilder_,
        conductor: Option<Conductor_>,
        engine_client: SequencerEngineClient_,
        is_active: bool,
        in_recovery_mode: bool,
        origin_selector: OriginSelector_,
        rollup_config: Arc<RollupConfig>,
        unsafe_payload_gossip_client: UnsafePayloadGossipClient_,
    ) -> Self {
        let conductor = conductor.map(Arc::new);
        let engine_client = Arc::new(engine_client);
        let workflow = SequencingWorkflow::new(
            attributes_builder,
            conductor.clone(),
            engine_client.clone(),
            origin_selector,
            rollup_config,
            unsafe_payload_gossip_client,
        );

        Self { admin_api_rx, conductor, engine_client, is_active, in_recovery_mode, workflow }
    }

    /// Runs the sequencer until node shutdown or a critical sequencing error.
    ///
    /// Shutdown and admin stop requests cancel preparation by dropping its future. Once a conductor
    /// commit attempt may have begun, the distribution future is retained until the exact payload
    /// has been published and canonicalized.
    pub async fn run(mut self, shutdown: CancellationToken) -> Result<(), SequencerActorError> {
        tokio::select! {
            biased;
            _ = shutdown.cancelled() => return Ok(()),
            result = self.engine_client.reset_engine_forkchoice() => {
                result.map_err(|err| {
                    error!(target: "sequencer", ?err, "Failed to perform initial engine reset");
                    err
                })?;
            }
        }

        let mut admin_channel_open = true;

        loop {
            while !self.is_active {
                if !admin_channel_open {
                    shutdown.cancelled().await;
                    return Ok(());
                }

                tokio::select! {
                    biased;
                    _ = shutdown.cancelled() => return Ok(()),
                    query = self.admin_api_rx.recv() => {
                        match query {
                            Some(query) => self.handle_admin_query(query).await,
                            None => admin_channel_open = false,
                        }
                    }
                }
            }

            let recovery_mode = self.in_recovery_mode;
            let preparation_event = {
                let preparation = self.workflow.prepare_candidate(recovery_mode);
                tokio::pin!(preparation);

                loop {
                    tokio::select! {
                        biased;
                        query = self.admin_api_rx.recv(), if admin_channel_open => {
                            match query {
                                Some(query) if Self::interrupts_preparation(&query) => {
                                    break PreparationEvent::Interrupt(query);
                                }
                                Some(query) => {
                                    Self::handle_admin_query_parts(
                                        &mut self.is_active,
                                        &mut self.in_recovery_mode,
                                        self.conductor.as_ref(),
                                        &self.engine_client,
                                        query,
                                    ).await;
                                }
                                None => admin_channel_open = false,
                            }
                        }
                        _ = shutdown.cancelled() => break PreparationEvent::Shutdown,
                        result = &mut preparation => break PreparationEvent::Candidate(result),
                    }
                }
            };

            let candidate = match preparation_event {
                PreparationEvent::Candidate(result) => result?,
                PreparationEvent::Interrupt(query) => {
                    // The preparation future has been dropped before handling an operation that
                    // may inspect or reset the engine.
                    self.handle_admin_query(query).await;
                    continue;
                }
                PreparationEvent::Shutdown => return Ok(()),
            };

            let mut deferred_queries = Vec::<DeferredAdminQuery>::new();
            let mut shutdown_requested = false;

            let distribution_result = {
                let distribution = self.workflow.distribute_candidate(candidate);
                tokio::pin!(distribution);

                loop {
                    tokio::select! {
                        biased;
                        result = &mut distribution => break result,
                        query = self.admin_api_rx.recv(), if admin_channel_open => {
                            match query {
                                Some(SequencerAdminQuery::StopSequencer(response)) => {
                                    info!(target: "sequencer", "Stopping sequencer after protected payload distribution");
                                    self.is_active = false;
                                    super::metrics::update_state_metrics(
                                        self.is_active,
                                        self.in_recovery_mode,
                                    );
                                    deferred_queries.push(DeferredAdminQuery::Stop(response));
                                }
                                Some(SequencerAdminQuery::ResetDerivationPipeline(response)) => {
                                    // A reset may invalidate the payload being distributed. Defer it
                                    // until the committed payload has become canonical locally.
                                    deferred_queries.push(DeferredAdminQuery::Reset(response));
                                }
                                Some(query) => {
                                    Self::handle_admin_query_parts(
                                        &mut self.is_active,
                                        &mut self.in_recovery_mode,
                                        self.conductor.as_ref(),
                                        &self.engine_client,
                                        query,
                                    ).await;
                                }
                                None => admin_channel_open = false,
                            }
                        }
                        _ = shutdown.cancelled(), if !shutdown_requested => {
                            shutdown_requested = true;
                            self.is_active = false;
                        }
                    }
                }
            };

            let head = distribution_result?;
            debug!(target: "sequencer", head = ?head.block_info, "Sequencer advanced unsafe head");

            for query in deferred_queries {
                match query {
                    DeferredAdminQuery::Stop(response) => {
                        if response.send(Ok(head.hash())).is_err() {
                            warn!(target: "sequencer", "Failed to send response for stop_sequencer query");
                        }
                    }
                    DeferredAdminQuery::Reset(response) => {
                        if response.send(self.reset_derivation_pipeline().await).is_err() {
                            warn!(target: "sequencer", "Failed to send response for reset_derivation_pipeline query");
                        }
                    }
                }
            }

            if shutdown_requested {
                return Ok(());
            }
        }
    }

    /// Returns whether an admin operation must first cancel unpublished block preparation.
    const fn interrupts_preparation(query: &SequencerAdminQuery) -> bool {
        matches!(
            query,
            SequencerAdminQuery::StopSequencer(_) | SequencerAdminQuery::ResetDerivationPipeline(_)
        )
    }
}
