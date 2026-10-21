//! [`NodeActor`] trait.

use async_trait::async_trait;

/// The [`NodeActor`] is an actor-like service for the node.
///
/// An actor's work is split in two phases: [`Self::next`] waits for the next inbound request,
/// event, or tick, and [`Self::handle`] processes it. Callers may call [`Self::step`] to wait for
/// and handle a single input.
///
/// The split lets the node choose how far shutdown interrupts an actor without changing any actor:
/// racing only [`Self::next`] against cancellation, and waiting for the actors to stop, would let
/// an actor finish handling its current input. The node currently races the whole [`Self::step`]
/// and does not wait, so shutdown is crash-only.
#[async_trait]
pub trait NodeActor: Send + 'static {
    /// The error type for the actor.
    type Error: std::fmt::Debug;

    /// A unit of work received by [`Self::next`] and processed by [`Self::handle`].
    type Input: Send;

    /// Wait for the next inbound request, event, or scheduled tick.
    ///
    /// This should do no work that must not be interrupted, which belongs in [`Self::handle`], so
    /// that the node could race only this against cancellation if it ever wanted actors to finish
    /// handling their current input. See the trait docs.
    ///
    /// Returning `Err(_)` indicates the actor has encountered a fatal
    /// condition and should not be stepped further.
    async fn next(&mut self) -> Result<Self::Input, Self::Error>;

    /// Handle an input returned by [`Self::next`].
    ///
    /// Shutdown may interrupt this at any `.await`, so any state it changes outside the process
    /// must be recoverable after a restart.
    ///
    /// Returning `Ok(())` indicates the actor is ready to be stepped again.
    /// Returning `Err(_)` indicates the actor has encountered a fatal
    /// condition and should not be stepped further.
    async fn handle(&mut self, input: Self::Input) -> Result<(), Self::Error>;

    /// Wait for the next input and handle it.
    ///
    /// Cancelling this can interrupt [`Self::handle`], which the node only does when it discards
    /// the actor afterwards.
    async fn step(&mut self) -> Result<(), Self::Error> {
        let input = self.next().await?;
        self.handle(input).await
    }
}
