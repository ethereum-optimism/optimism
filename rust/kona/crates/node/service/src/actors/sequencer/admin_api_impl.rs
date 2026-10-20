use super::{SequencerActor, handle::Message};
use crate::{Conductor, OriginSelector, UnsafePayloadGossipClient};
use kona_derive::AttributesBuilder;
use kona_rpc::SequencerAdminAPIError;

impl<AttributesBuilder_, Conductor_, OriginSelector_, UnsafePayloadGossipClient_>
    SequencerActor<AttributesBuilder_, Conductor_, OriginSelector_, UnsafePayloadGossipClient_>
where
    AttributesBuilder_: AttributesBuilder,
    Conductor_: Conductor,
    OriginSelector_: OriginSelector,
    UnsafePayloadGossipClient_: UnsafePayloadGossipClient,
{
    /// Complete accepted mutations even when their caller has stopped waiting.
    pub(super) async fn handle_command(&mut self, command: Message) {
        match command {
            Message::Start(reply) => {
                if !self.state.active {
                    info!(target: "sequencer", "Starting sequencer");
                    self.state.active = true;
                    self.build_ticker.reset_immediately();
                }
                self.publish();
                let _ = reply.send(Ok(()));
            }
            Message::Stop(reply) => {
                info!(target: "sequencer", "Stopping sequencer");
                self.state.active = false;
                let head = self.engine_client.state().sync_state.unsafe_head().hash();
                self.publish();
                let _ = reply.send(Ok(head));
            }
            Message::SetRecoveryMode(mode, reply) => {
                self.state.recovery_mode = mode;
                info!(target: "sequencer", mode, "Updated recovery mode");
                self.publish();
                let _ = reply.send(Ok(()));
            }
            Message::OverrideLeader(reply) => {
                let result = match self.conductor.as_mut() {
                    Some(conductor) => conductor.override_leader().await.map_err(|error| {
                        error!(target: "sequencer::rpc", %error, "Failed to override leader");
                        SequencerAdminAPIError::LeaderOverrideError(error.to_string())
                    }),
                    None => Err(SequencerAdminAPIError::LeaderOverrideError(
                        "No conductor configured".into(),
                    )),
                };
                self.publish();
                let _ = reply.send(result);
            }
        }
    }
}
