//! Requests sent to the node engine actor and their replies.

use crate::{BuildTaskError, CanonicalizeTaskError, FinalizeBlockId, SealTaskError};
use alloy_rpc_types_engine::PayloadId;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use thiserror::Error;
use tokio::sync::mpsc;

/// A request handled by the node engine actor.
#[derive(Debug)]
pub enum EngineActorRequest {
    /// Request to start building a block.
    Build(Box<BuildRequest>),
    /// Request to consolidate attributes from the derivation pipeline.
    ProcessDerivedAttributes {
        /// The derived attributes.
        attributes: Box<OpAttributesWithParent>,
        /// How many engine resets derivation had handled when it derived the attributes. The
        /// engine drops attributes derived before its latest reset.
        reset_epoch: u64,
    },
    /// Request to finalize an L2 block that derivation recorded as safe.
    ProcessDerivedFinalization {
        /// The L2 block number to finalize.
        number: u64,
        /// How many engine resets derivation had handled when it recorded the block. The engine
        /// drops finalizations recorded before its latest reset.
        reset_epoch: u64,
    },
    /// Request to adopt a safe block reported by a derivation delegate.
    ProcessDelegatedSafeHead(Box<L2BlockInfo>),
    /// Request to process the finalized L2 block.
    ProcessFinalizedL2Block(Box<FinalizeBlockId>),
    /// Request to process a received unsafe L2 block.
    ProcessUnsafeL2Block(Box<OpExecutionPayloadEnvelope>),
    /// Request to reset the forkchoice and signal derivation to reset.
    Reset(Box<ResetRequest>),
    /// Request to seal a block.
    Seal(Box<SealRequest>),
    /// Request to canonicalize a committed sequenced block.
    Canonicalize(Box<CanonicalizeRequest>),
}

/// The result of a request to the node engine actor.
pub type EngineRequestResult<T> = Result<T, EngineRequestError>;

/// Error making requests to the node engine actor.
#[derive(Debug, Error)]
pub enum EngineRequestError {
    /// Error making a request to the engine. The request never made it there.
    #[error("Error making a request to the engine: {0}.")]
    RequestError(String),

    /// Error receiving response from the engine.
    /// This means the request may or may not have succeeded.
    #[error("Error receiving response from the engine: {0}.")]
    ResponseError(String),

    /// An error occurred starting to build a block.
    #[error(transparent)]
    StartBuildError(#[from] BuildTaskError),

    /// An error occurred sealing a block.
    #[error(transparent)]
    SealError(#[from] SealTaskError),

    /// An error occurred canonicalizing a sequenced block.
    #[error(transparent)]
    CanonicalizeError(#[from] CanonicalizeTaskError),

    /// An error occurred performing the reset.
    #[error("An error occurred performing the reset: {0}.")]
    ResetForkchoiceError(String),
}

/// A request to build a payload.
/// Contains the attributes to build and a channel to send back the resulting `PayloadId`.
#[derive(Debug)]
pub struct BuildRequest {
    /// The [`OpAttributesWithParent`] from which the block build should be started.
    pub attributes: OpAttributesWithParent,
    /// The channel on which the result, successful or not, will be sent.
    pub result_tx: mpsc::Sender<Result<PayloadId, BuildTaskError>>,
}

/// A request to reset the engine forkchoice.
/// Contains a channel to acknowledge successful processing to the caller.
#[derive(Debug)]
pub struct ResetRequest {
    /// The channel on which the reset result will be sent.
    pub result_tx: mpsc::Sender<EngineRequestResult<()>>,
}

/// A request to fetch a sealed payload without changing forkchoice.
/// Contains the `PayloadId`, attributes, and a channel to send back the result.
#[derive(Debug)]
pub struct SealRequest {
    /// The `PayloadId` to seal.
    pub payload_id: PayloadId,
    /// The attributes necessary for the seal operation.
    pub attributes: OpAttributesWithParent,
    /// The channel on which the result, successful or not, will be sent.
    pub result_tx: mpsc::Sender<Result<OpExecutionPayloadEnvelope, SealTaskError>>,
}

/// A request to canonicalize a sequenced payload after its conductor commit succeeds.
#[derive(Debug)]
pub struct CanonicalizeRequest {
    /// The committed payload to import.
    pub payload: OpExecutionPayloadEnvelope,
    /// The unsafe head on which the build started.
    pub parent: kona_protocol::L2BlockInfo,
    /// The response, including any import error, is relayed to the sequencer.
    pub result_tx: mpsc::Sender<Result<(), CanonicalizeTaskError>>,
}
