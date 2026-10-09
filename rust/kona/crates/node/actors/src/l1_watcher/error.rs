use thiserror::Error;

/// A fatal error from the L1 watcher actor.
#[derive(Error, Debug)]
pub enum ActorError {
    /// An observation stream ended.
    #[error("observation stream ended")]
    StreamEnded,
    /// A watch channel closed.
    #[error("channel closed")]
    ChannelClosed,
    /// Derivation could not receive an observation.
    #[error("{0}")]
    Derivation(String),
}
