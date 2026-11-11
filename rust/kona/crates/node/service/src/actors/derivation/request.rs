use kona_derive::Signal;
use kona_engine::{DelegatedRejection, DelegatedSafeHeads, EngineSyncState};
use kona_protocol::{BlockInfo, L2BlockInfo};
use thiserror::Error;

/// The result of an Engine client call.
pub type DerivationClientResult<T> = Result<T, DerivationClientError>;

/// Error making requests to the [`crate::DerivationActor`].
#[derive(Debug, Error)]
pub enum DerivationClientError {
    /// Error making a request to the [`crate::DerivationActor`]. The request never made it there.
    #[error("Error making a request to the derivation actor: {0}.")]
    RequestError(String),

    /// Error receiving response from the [`crate::DerivationActor`].
    /// This means the request may or may not have succeeded.
    #[error("Error receiving response from the derivation actor: {0}..")]
    ResponseError(String),
}

/// Inbound requests that the [`crate::DerivationActor`] can process.
#[derive(Debug)]
pub enum DerivationActorRequest {
    /// Request to process the fact that Engine sync has completed, along with the engine's sync
    /// state at that point.
    ProcessEngineSyncCompletionRequest(Box<EngineSyncState>),
    /// Request to process the provided L2 engine safe head update.
    ProcessEngineSafeHeadUpdateRequest(Box<L2BlockInfo>),
    /// A request containing a [`Signal`] to the derivation pipeline.
    /// This allows the Engine to send the `DerivationActor` signals (e.g. to Flush or Reset).
    ProcessEngineSignalRequest(Box<Signal>),
    /// A request to process the provided finalized L1 [`BlockInfo`].
    ProcessFinalizedL1Block(Box<BlockInfo>),
    /// Request to process the provided L1 head block update.
    ProcessL1HeadUpdateRequest(Box<BlockInfo>),
    /// The execution layer rejected a forkchoice update for these delegated safe heads, and would
    /// reject them again until the engine resets.
    ProcessRejectedDelegatedSafeHeads {
        /// The rejected delegated safe heads.
        heads: Box<DelegatedSafeHeads>,
        /// Why the execution layer rejected them.
        rejection: DelegatedRejection,
    },
}
