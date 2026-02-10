//! [`NodeActor`] trait.

use async_trait::async_trait;

/// The [`NodeActor`] is an actor-like service for the node.
///
/// Actors may:
/// - Handle incoming messages.
///     - Perform background tasks.
/// - Emit new events for other actors to process.
///
/// ## Lifecycle
///
/// 1. **Build**: The caller assembles a `Self::Builder` with everything the actor needs.
/// 2. **Init**: [`NodeActor::init`] consumes the builder and returns `(Self::InboundData, Self)`.
///    `InboundData` contains sender endpoints that other actors use to communicate with this actor.
/// 3. **Step**: The event loop calls [`NodeActor::step`] repeatedly. Each call processes exactly
///    one event. Cancellation is handled externally by the caller (e.g. via `tokio::select!`).
#[async_trait]
pub trait NodeActor: Send + 'static {
    /// The error type for the actor.
    type Error: std::fmt::Debug;
    /// Everything the actor needs to initialize.
    type Builder: Sized + Send;
    /// Sender endpoints returned from init that other actors use to talk to this actor.
    /// Use `()` when the actor has no inbound channels or they are pre-created externally.
    type InboundData: Sized + Send;

    /// Initializes the actor from a builder, returning `(inbound_data, actor)`.
    async fn init(builder: Self::Builder) -> Result<(Self::InboundData, Self), Self::Error>
    where
        Self: Sized;

    /// Processes exactly one event. Called in a loop by the event-loop driver.
    async fn step(&mut self) -> Result<(), Self::Error>;
}
