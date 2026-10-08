use super::SequencerActor;
use crate::{Conductor, OriginSelector, SequencerEngineClient, UnsafePayloadGossipClient};
use alloy_primitives::B256;
use kona_derive::AttributesBuilder;
use kona_rpc::{SequencerAdminAPIError, SequencerAdminCommand};

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
    /// Execute an admin command and acknowledge its result.
    pub(super) async fn handle_admin_command(&mut self, command: SequencerAdminCommand) {
        // A dropped RPC response receiver does not cancel an accepted command.
        match command {
            SequencerAdminCommand::StartSequencer(tx) => {
                let _ = tx.send(self.start_sequencer().await);
            }
            SequencerAdminCommand::StopSequencer(tx) => {
                let _ = tx.send(self.stop_sequencer().await);
            }
            SequencerAdminCommand::SetRecoveryMode(mode, tx) => {
                let _ = tx.send(self.set_recovery_mode(mode).await);
            }
            SequencerAdminCommand::OverrideLeader(tx) => {
                let _ = tx.send(self.override_leader().await);
            }
            SequencerAdminCommand::ResetDerivationPipeline(tx) => {
                let _ = tx.send(self.reset_derivation_pipeline().await);
            }
        }
    }

    /// Starts the sequencer in an idempotent fashion.
    pub(super) async fn start_sequencer(&self) -> Result<(), SequencerAdminAPIError> {
        if self.state().active {
            info!(target: "sequencer", "received request to start sequencer, but it is already started");
            return Ok(());
        }

        info!(target: "sequencer", "Starting sequencer");
        self.update_state(|state| state.active = true);

        Ok(())
    }

    /// Stops the sequencer in an idempotent fashion.
    pub(super) async fn stop_sequencer(&self) -> Result<B256, SequencerAdminAPIError> {
        info!(target: "sequencer", "Stopping sequencer");
        // Publish before awaiting the unsafe head: sequencing is stopped even if that read fails.
        self.update_state(|state| state.active = false);

        self.engine_client.get_unsafe_head().await
            .map(|h| h.hash())
            .map_err(|e| {
                error!(target: "sequencer", err=?e, "Error fetching unsafe head after stopping sequencer, which should never happen.");
                SequencerAdminAPIError::ErrorAfterSequencerWasStopped("current unsafe hash is unavailable.".to_string())
            })
    }

    /// Sets the recovery mode of the sequencer in an idempotent fashion.
    pub(super) async fn set_recovery_mode(&self, mode: bool) -> Result<(), SequencerAdminAPIError> {
        self.update_state(|state| state.recovery_mode = mode);
        info!(target: "sequencer", is_active = mode, "Updated recovery mode");

        Ok(())
    }

    /// Overrides the leader, if the conductor is enabled.
    /// If not, an error will be returned.
    pub(super) async fn override_leader(&mut self) -> Result<(), SequencerAdminAPIError> {
        let Some(conductor) = self.conductor.as_mut() else {
            return Err(SequencerAdminAPIError::LeaderOverrideError(
                "No conductor configured".to_string(),
            ));
        };

        if let Err(e) = conductor.override_leader().await {
            error!(target: "sequencer::rpc", "Failed to override leader: {}", e);
            return Err(SequencerAdminAPIError::LeaderOverrideError(e.to_string()));
        }
        info!(target: "sequencer", "Overrode leader via the conductor service");

        Ok(())
    }

    pub(super) async fn reset_derivation_pipeline(&self) -> Result<(), SequencerAdminAPIError> {
        info!(target: "sequencer", "Resetting derivation pipeline");
        self.engine_client.reset_engine_forkchoice().await.map_err(|e| {
            error!(target: "sequencer", err=?e, "Failed to reset engine forkchoice");
            SequencerAdminAPIError::RequestError(format!("Failed to reset engine: {e}"))
        })
    }
}
