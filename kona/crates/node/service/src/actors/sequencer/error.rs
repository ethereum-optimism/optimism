use crate::{
    L1OriginSelectorError, UnsafePayloadGossipClientError, actors::engine::BlockEngineError,
};
use kona_derive::PipelineErrorKind;
use kona_engine::BuildTaskError;

/// An error produced by the [`crate::SequencerActor`].
#[derive(Debug, thiserror::Error)]
pub enum SequencerActorError {
    /// An error occurred while building payload attributes.
    #[error(transparent)]
    AttributesBuilder(#[from] PipelineErrorKind),
    /// A channel was unexpectedly closed.
    #[error("Channel closed unexpectedly")]
    ChannelClosed,
    /// An error occurred while selecting the next L1 origin.
    #[error(transparent)]
    L1OriginSelector(#[from] L1OriginSelectorError),
    /// An error occurred while attempting to seal a payload.
    #[error(transparent)]
    BlockEngine(#[from] BlockEngineError),
    /// An error occurred while attempting to build a payload.
    #[error(transparent)]
    BuildError(#[from] BuildTaskError),
    /// An error occurred while attempting to schedule unsafe payload gossip.
    #[error("An error occurred while attempting to schedule unsafe payload gossip: {0}")]
    PayloadGossip(#[from] UnsafePayloadGossipClientError),
}
