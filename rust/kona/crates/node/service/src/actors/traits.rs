//! [`NodeActor`] trait.

use async_trait::async_trait;

/// The [`NodeActor`] is an actor-like service for the node.
///
/// An actor's work is split in two phases: [`Self::next`] waits for the next inbound request,
/// event, or tick, and [`Self::handle`] processes it. The node's actor loop races only
/// [`Self::next`] against shutdown, so once an input has been received, it is handled to
/// completion.
///
/// Callers may call [`Self::step`] to wait for and handle a single input.
#[async_trait]
pub trait NodeActor: Send + 'static {
    /// The error type for the actor.
    type Error: std::fmt::Debug;

    /// A unit of work received by [`Self::next`] and processed by [`Self::handle`].
    type Input: Send;

    /// Wait for the next inbound request, event, or scheduled tick.
    ///
    /// This must be cancellation safe: on shutdown, the future is dropped if it has not resolved,
    /// which must neither lose an input nor leave the actor in an inconsistent state. Any work
    /// that must not be interrupted belongs in [`Self::handle`].
    ///
    /// Returning `Err(_)` indicates the actor has encountered a fatal
    /// condition and should not be stepped further.
    async fn next(&mut self) -> Result<Self::Input, Self::Error>;

    /// Handle an input returned by [`Self::next`].
    ///
    /// This is never interrupted by shutdown.
    ///
    /// Returning `Ok(())` indicates the actor is ready to be stepped again.
    /// Returning `Err(_)` indicates the actor has encountered a fatal
    /// condition and should not be stepped further.
    async fn handle(&mut self, input: Self::Input) -> Result<(), Self::Error>;

    /// Wait for the next input and handle it.
    ///
    /// Unlike [`Self::next`], this is not cancellation safe.
    async fn step(&mut self) -> Result<(), Self::Error> {
        let input = self.next().await?;
        self.handle(input).await
    }
}
