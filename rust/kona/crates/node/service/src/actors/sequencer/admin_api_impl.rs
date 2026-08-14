use super::SequencerActor;
use crate::{Conductor, OriginSelector, SequencerEngineClient, UnsafePayloadGossipClient};
use alloy_primitives::B256;
use kona_derive::AttributesBuilder;
use kona_rpc::SequencerAdminAPIError;
use std::sync::Arc;
use tokio::sync::oneshot;

/// The query types to the sequencer actor for the admin api.
#[derive(Debug)]
pub enum SequencerAdminQuery {
    /// A query to check if the sequencer is active.
    SequencerActive(oneshot::Sender<Result<bool, SequencerAdminAPIError>>),
    /// A query to start the sequencer.
    StartSequencer(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
    /// A query to stop the sequencer.
    StopSequencer(oneshot::Sender<Result<B256, SequencerAdminAPIError>>),
    /// A query to check if the conductor is enabled.
    ConductorEnabled(oneshot::Sender<Result<bool, SequencerAdminAPIError>>),
    /// A query to check if the sequencer is in recovery mode.
    RecoveryMode(oneshot::Sender<Result<bool, SequencerAdminAPIError>>),
    /// A query to set the recovery mode.
    SetRecoveryMode(bool, oneshot::Sender<Result<(), SequencerAdminAPIError>>),
    /// A query to override the leader.
    OverrideLeader(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
    /// A query to reset the derivation pipeline.
    ResetDerivationPipeline(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
}

/// Handler for the Sequencer Admin API.
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
    /// Handles the provided [`SequencerAdminQuery`], sending the response via its sender.
    pub(super) async fn handle_admin_query(&mut self, query: SequencerAdminQuery) {
        Self::handle_admin_query_parts(
            &mut self.is_active,
            &mut self.in_recovery_mode,
            self.conductor.as_ref(),
            &self.engine_client,
            query,
        )
        .await;
    }

    /// Handles an admin request without borrowing the sequencing workflow.
    ///
    /// Keeping this operation on the control-plane fields lets the service continue polling a
    /// pinned preparation or distribution future while it responds to compatible admin requests.
    pub(super) async fn handle_admin_query_parts(
        is_active: &mut bool,
        in_recovery_mode: &mut bool,
        conductor: Option<&Arc<Conductor_>>,
        engine_client: &Arc<SequencerEngineClient_>,
        query: SequencerAdminQuery,
    ) {
        match query {
            SequencerAdminQuery::SequencerActive(tx) => {
                if tx.send(Ok(*is_active)).is_err() {
                    warn!(target: "sequencer", "Failed to send response for is_sequencer_active query");
                }
            }
            SequencerAdminQuery::StartSequencer(tx) => {
                let result = start_sequencer(is_active, *in_recovery_mode);
                if tx.send(result).is_err() {
                    warn!(target: "sequencer", "Failed to send response for start_sequencer query");
                }
            }
            SequencerAdminQuery::StopSequencer(tx) => {
                let result =
                    stop_sequencer(is_active, *in_recovery_mode, engine_client.as_ref()).await;
                if tx.send(result).is_err() {
                    warn!(target: "sequencer", "Failed to send response for stop_sequencer query");
                }
            }
            SequencerAdminQuery::ConductorEnabled(tx) => {
                if tx.send(Ok(conductor.is_some())).is_err() {
                    warn!(target: "sequencer", "Failed to send response for is_conductor_enabled query");
                }
            }
            SequencerAdminQuery::RecoveryMode(tx) => {
                if tx.send(Ok(*in_recovery_mode)).is_err() {
                    warn!(target: "sequencer", "Failed to send response for in_recovery_mode query");
                }
            }
            SequencerAdminQuery::SetRecoveryMode(is_recovery_active, tx) => {
                set_recovery_mode(in_recovery_mode, is_recovery_active, *is_active);
                if tx.send(Ok(())).is_err() {
                    warn!(target: "sequencer", is_active = is_recovery_active, "Failed to send response for set_recovery_mode query");
                }
            }
            SequencerAdminQuery::OverrideLeader(tx) => {
                let result = override_leader(conductor.map(Arc::as_ref)).await;
                if tx.send(result).is_err() {
                    warn!(target: "sequencer", "Failed to send response for override_leader query");
                }
            }
            SequencerAdminQuery::ResetDerivationPipeline(tx) => {
                let result = reset_derivation_pipeline(engine_client.as_ref()).await;
                if tx.send(result).is_err() {
                    warn!(target: "sequencer", "Failed to send response for reset_derivation_pipeline query");
                }
            }
        }
    }

    /// Returns whether the sequencer is active.
    #[cfg(test)]
    pub(super) async fn is_sequencer_active(&self) -> Result<bool, SequencerAdminAPIError> {
        Ok(self.is_active)
    }

    /// Returns whether the conductor is enabled.
    #[cfg(test)]
    pub(super) async fn is_conductor_enabled(&self) -> Result<bool, SequencerAdminAPIError> {
        Ok(self.conductor.is_some())
    }

    /// Returns whether the node is in recovery mode.
    #[cfg(test)]
    pub(super) async fn in_recovery_mode(&self) -> Result<bool, SequencerAdminAPIError> {
        Ok(self.in_recovery_mode)
    }

    /// Starts the sequencer in an idempotent fashion.
    #[cfg(test)]
    pub(super) async fn start_sequencer(&mut self) -> Result<(), SequencerAdminAPIError> {
        start_sequencer(&mut self.is_active, self.in_recovery_mode)
    }

    /// Stops the sequencer in an idempotent fashion.
    #[cfg(test)]
    pub(super) async fn stop_sequencer(&mut self) -> Result<B256, SequencerAdminAPIError> {
        stop_sequencer(&mut self.is_active, self.in_recovery_mode, self.engine_client.as_ref())
            .await
    }

    /// Sets the recovery mode of the sequencer in an idempotent fashion.
    #[cfg(test)]
    pub(super) async fn set_recovery_mode(
        &mut self,
        is_active: bool,
    ) -> Result<(), SequencerAdminAPIError> {
        set_recovery_mode(&mut self.in_recovery_mode, is_active, self.is_active);
        Ok(())
    }

    /// Overrides the leader, if the conductor is enabled.
    #[cfg(test)]
    pub(super) async fn override_leader(&self) -> Result<(), SequencerAdminAPIError> {
        override_leader(self.conductor.as_deref()).await
    }

    /// Resets the engine forkchoice to the derivation pipeline's view.
    pub(super) async fn reset_derivation_pipeline(&self) -> Result<(), SequencerAdminAPIError> {
        reset_derivation_pipeline(self.engine_client.as_ref()).await
    }
}

fn start_sequencer(
    is_active: &mut bool,
    in_recovery_mode: bool,
) -> Result<(), SequencerAdminAPIError> {
    if *is_active {
        info!(target: "sequencer", "received request to start sequencer, but it is already started");
        return Ok(());
    }

    info!(target: "sequencer", "Starting sequencer");
    *is_active = true;
    super::metrics::update_state_metrics(*is_active, in_recovery_mode);
    Ok(())
}

async fn stop_sequencer<SequencerEngineClient_: SequencerEngineClient>(
    is_active: &mut bool,
    in_recovery_mode: bool,
    engine_client: &SequencerEngineClient_,
) -> Result<B256, SequencerAdminAPIError> {
    info!(target: "sequencer", "Stopping sequencer");
    *is_active = false;
    super::metrics::update_state_metrics(*is_active, in_recovery_mode);

    engine_client.get_unsafe_head().await.map(|head| head.hash()).map_err(|err| {
        error!(target: "sequencer", ?err, "Error fetching unsafe head after stopping sequencer");
        SequencerAdminAPIError::ErrorAfterSequencerWasStopped(
            "current unsafe hash is unavailable.".to_string(),
        )
    })
}

fn set_recovery_mode(in_recovery_mode: &mut bool, is_active: bool, sequencer_active: bool) {
    *in_recovery_mode = is_active;
    info!(target: "sequencer", is_active, "Updated recovery mode");
    super::metrics::update_state_metrics(sequencer_active, *in_recovery_mode);
}

async fn override_leader<Conductor_: Conductor>(
    conductor: Option<&Conductor_>,
) -> Result<(), SequencerAdminAPIError> {
    let Some(conductor) = conductor else {
        return Err(SequencerAdminAPIError::LeaderOverrideError(
            "No conductor configured".to_string(),
        ));
    };

    if let Err(err) = conductor.override_leader().await {
        error!(target: "sequencer::rpc", "Failed to override leader: {err}");
        return Err(SequencerAdminAPIError::LeaderOverrideError(err.to_string()));
    }
    info!(target: "sequencer", "Overrode leader via the conductor service");
    Ok(())
}

async fn reset_derivation_pipeline<SequencerEngineClient_: SequencerEngineClient>(
    engine_client: &SequencerEngineClient_,
) -> Result<(), SequencerAdminAPIError> {
    info!(target: "sequencer", "Resetting derivation pipeline");
    engine_client.reset_engine_forkchoice().await.map_err(|err| {
        error!(target: "sequencer", ?err, "Failed to reset engine forkchoice");
        SequencerAdminAPIError::RequestError(format!("Failed to reset engine: {err}"))
    })
}
