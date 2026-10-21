//! [`NodeActor`] trait.

use async_trait::async_trait;
use std::convert::Infallible;

/// The [`NodeActor`] is an actor-like service for the node.
///
/// An actor's work is split in two phases: [`Self::next`] waits for the next inbound request,
/// event, or tick, and [`Self::handle`] processes it. The node calls them in turn, in a loop.
///
/// The split lets the node choose how far shutdown interrupts an actor without changing any actor:
/// stopping actors only while they wait in [`Self::next`], and waiting for them, would let an
/// actor finish handling its current input. The node currently stops actors wherever they are and
/// does not wait for them, so shutdown is crash-only.
#[async_trait]
pub trait NodeActor: Send + 'static {
    /// The error type for the actor.
    type Error: std::fmt::Debug;

    /// A unit of work received by [`Self::next`] and processed by [`Self::handle`].
    type Input: Send;

    /// Wait for the next inbound request, event, or scheduled tick.
    ///
    /// This should do no work that must not be interrupted, which belongs in [`Self::handle`], so
    /// that the node could stop actors only while they wait here if it ever wanted them to finish
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
    /// A convenience for driving an actor one input at a time, for example in tests. The node calls
    /// [`Self::next`] and [`Self::handle`] itself.
    async fn step(&mut self) -> Result<(), Self::Error> {
        let input = self.next().await?;
        self.handle(input).await
    }
}

/// Drives `actor` until it fails: waits for its next input with [`NodeActor::next`], then handles
/// it with [`NodeActor::handle`].
///
/// This is how the node runs its actors. It never returns `Ok`, so awaiting it yields the error
/// the actor failed with.
pub async fn run_actor<A: NodeActor>(mut actor: A) -> Result<Infallible, A::Error> {
    loop {
        let input = actor.next().await?;
        actor.handle(input).await?;
    }
}
