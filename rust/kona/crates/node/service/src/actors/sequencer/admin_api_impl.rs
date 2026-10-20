use super::{SequencerActor, handle::Message};
use crate::{Conductor, OriginSelector, SequencerActorError, UnsafePayloadGossipClient};
use alloy_primitives::B256;
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
    /// Verify leadership before inspecting active state, and read the head after that await.
    async fn start(&mut self, expected: B256) -> Result<(), SequencerAdminAPIError> {
        if let Some(conductor) = &self.conductor {
            let leader = conductor
                .leader()
                .await
                .map_err(|error| SequencerAdminAPIError::LeaderCheckFailed(error.to_string()))?;
            if !leader {
                return Err(SequencerAdminAPIError::NotLeader);
            }
        }
        if self.state.active {
            return Err(SequencerAdminAPIError::AlreadyRunning);
        }
        let head = self.engine_client.state().sync_state.unsafe_head();
        if head.hash() == B256::ZERO {
            return Err(SequencerAdminAPIError::NoPrestate(expected));
        }
        if head.hash() != expected {
            return Err(SequencerAdminAPIError::HeadMismatch { head, received: expected });
        }
        self.state.active = true;
        self.build_ticker.reset_immediately();
        self.publish();
        Ok(())
    }

    /// Complete accepted mutations even when their caller has stopped waiting.
    pub(super) async fn handle_command(
        &mut self,
        command: Message,
    ) -> Result<(), SequencerActorError> {
        match command {
            Message::Start(expected, reply) => {
                let result = self.start(expected).await;
                let _ = reply.send(result);
            }
            Message::Stop(reply) => {
                if !self.state.active {
                    let _ = reply.send(Err(SequencerAdminAPIError::NotRunning));
                    return Ok(());
                }
                // Keep the public active state until settlement completes. No build can run here.
                self.state.active = false;
                let result = self.settle_committed().await;
                self.build_ticker.reset();
                self.last_seal_duration = std::time::Duration::ZERO;
                self.publish();
                let head = self.engine_client.state().sync_state.unsafe_head().hash();
                if let Err(error) = result {
                    let _ =
                        reply.send(Err(SequencerAdminAPIError::RequestError(error.to_string())));
                    return Err(error);
                }
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
        Ok(())
    }
}
