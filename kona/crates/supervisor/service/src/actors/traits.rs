/// The [`SupervisorActor`] trait is an actor-like service for the supervisor.
use async_trait::async_trait;
#[async_trait]
pub trait SupervisorActor {
    /// The event type received by the actor.
    type InboundEvent;
    /// The error type for the actor.
    type Error: std::fmt::Debug;
    /// Starts the actor.
    async fn start(mut self) -> Result<(), Self::Error>;
}
