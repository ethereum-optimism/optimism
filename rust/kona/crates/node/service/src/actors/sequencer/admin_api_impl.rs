use super::{SequencerActor, actor::BoundaryAction, handle::SequencerCommand};
use crate::{Conductor, OriginSelector, SequencerEngineClient, UnsafePayloadGossipClient};
use kona_derive::AttributesBuilder;
use kona_rpc::SequencerAdminAPIError;

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
    /// Handles one encapsulated control command at a block boundary.
    pub(super) async fn handle_command(
        &mut self,
        command: SequencerCommand,
        active: bool,
    ) -> BoundaryAction {
        match command {
            SequencerCommand::Active(response) => {
                if response.send(Ok(active)).is_err() {
                    warn!(target: "sequencer", "Failed to send active-state response");
                }
            }
            SequencerCommand::Start(response) => {
                if active {
                    info!(target: "sequencer", "Received start request while already active");
                } else {
                    info!(target: "sequencer", "Starting sequencer");
                    super::metrics::update_state_metrics(true, self.in_recovery_mode);
                }
                if response.send(Ok(())).is_err() {
                    warn!(target: "sequencer", "Failed to send start response");
                }
                return if active { BoundaryAction::Continue } else { BoundaryAction::Build };
            }
            SequencerCommand::Stop(response) => {
                let result = self.engine_client.get_unsafe_head().await.map(|head| head.hash()).map_err(
                    |err| {
                        error!(target: "sequencer", ?err, "Error fetching unsafe head while stopping sequencer");
                        SequencerAdminAPIError::ErrorAfterSequencerWasStopped(
                            "current unsafe hash is unavailable.".to_string(),
                        )
                    },
                );
                if active {
                    info!(target: "sequencer", "Stopping sequencer at block boundary");
                    super::metrics::update_state_metrics(false, self.in_recovery_mode);
                }
                if response.send(result).is_err() {
                    warn!(target: "sequencer", "Failed to send stop response");
                }
                return if active { BoundaryAction::Stop } else { BoundaryAction::Continue };
            }
            SequencerCommand::ConductorEnabled(response) => {
                if response.send(Ok(self.conductor.is_some())).is_err() {
                    warn!(target: "sequencer", "Failed to send conductor-enabled response");
                }
            }
            SequencerCommand::RecoveryMode(response) => {
                if response.send(Ok(self.in_recovery_mode)).is_err() {
                    warn!(target: "sequencer", "Failed to send recovery-mode response");
                }
            }
            SequencerCommand::SetRecoveryMode(mode, response) => {
                self.in_recovery_mode = mode;
                info!(target: "sequencer", mode, "Updated recovery mode");
                super::metrics::update_state_metrics(active, mode);
                if response.send(Ok(())).is_err() {
                    warn!(target: "sequencer", "Failed to send set-recovery-mode response");
                }
            }
            SequencerCommand::OverrideLeader(response) => {
                let result = match self.conductor.as_deref() {
                    Some(conductor) => conductor.override_leader().await.map_err(|err| {
                        error!(target: "sequencer::rpc", "Failed to override leader: {err}");
                        SequencerAdminAPIError::LeaderOverrideError(err.to_string())
                    }),
                    None => Err(SequencerAdminAPIError::LeaderOverrideError(
                        "No conductor configured".to_string(),
                    )),
                };
                if result.is_ok() {
                    info!(target: "sequencer", "Overrode leader via conductor service");
                }
                if response.send(result).is_err() {
                    warn!(target: "sequencer", "Failed to send override-leader response");
                }
            }
            SequencerCommand::ResetDerivationPipeline(response) => {
                info!(target: "sequencer", "Resetting derivation pipeline");
                let result = self.engine_client.reset_engine_forkchoice().await.map_err(|err| {
                    error!(target: "sequencer", ?err, "Failed to reset engine forkchoice");
                    SequencerAdminAPIError::RequestError(format!("Failed to reset engine: {err}"))
                });
                if response.send(result).is_err() {
                    warn!(target: "sequencer", "Failed to send reset response");
                }
            }
        }

        BoundaryAction::Continue
    }
}
