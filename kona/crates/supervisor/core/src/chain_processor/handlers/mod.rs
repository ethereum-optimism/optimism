//! This module contains various event handlers for processing different types of chain events.
mod cross_chain;
mod finalized;
mod invalidation;
mod origin;
mod safe_block;
mod unsafe_block;

pub use cross_chain::{CrossSafeHandler, CrossUnsafeHandler};
pub use finalized::FinalizedHandler;
pub use invalidation::{InvalidationHandler, ReplacementHandler};
pub use origin::OriginHandler;
pub use safe_block::SafeBlockHandler;
pub use unsafe_block::UnsafeBlockHandler;

use crate::{ChainProcessorError, ProcessorState};
use async_trait::async_trait;
use kona_protocol::BlockInfo;

/// [`EventHandler`] trait defines the interface for handling different types of events in the chain
/// processor. Each handler will implement this trait to process specific events like block updates,
/// invalidations, etc.
#[async_trait]
pub trait EventHandler<E> {
    /// Handle the event with the given state.
    async fn handle(
        &self,
        event: E,
        state: &mut ProcessorState,
    ) -> Result<BlockInfo, ChainProcessorError>;
}
