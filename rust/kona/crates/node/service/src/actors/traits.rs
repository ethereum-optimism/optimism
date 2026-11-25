//! [`NodeActor`] trait.

use async_trait::async_trait;

/// The [`NodeActor`] is an actor-like service for the node.
///
/// Callers may call [`Self::step`] to execute a single inbound request,
/// event, or tick.
#[async_trait]
pub trait NodeActor: Send + 'static {
    /// The error type for the actor.
    type Error: std::fmt::Debug;

    /// Handle the next inbound request, event, or scheduled tick.
    ///
    /// Returning `Ok(())` indicates the actor is ready to be stepped again.
    /// Returning `Err(_)` indicates the actor has encountered a fatal
    /// condition and should not be stepped further.
    async fn step(&mut self) -> Result<(), Self::Error>;
}
