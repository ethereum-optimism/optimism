use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadId;
use kona_engine::{BuildTaskError, EngineQueries, SealTaskError, SealedPayload};
use kona_protocol::{BlockInfo, OpAttributesWithParent};
use thiserror::Error;
use tokio::{sync::mpsc, time::Instant};

/// The result of an Engine client call.
pub type EngineClientResult<T> = Result<T, EngineClientError>;

/// Error making requests to the `BlockEngine`.
#[derive(Debug, Error)]
pub enum EngineClientError {
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

    /// An error occurred performing the reset.
    #[error("An error occurred performing the reset: {0}.")]
    ResetForkchoiceError(String),
}

/// RPC Request for the engine to handle.
#[derive(Debug)]
pub struct EngineRpcRequest(pub Box<EngineQueries>);

/// A request to build a payload.
/// Contains the attributes to build and a channel to send back the resulting `PayloadId`.
#[derive(Debug)]
pub struct BuildRequest {
    /// The [`OpAttributesWithParent`] from which the block build should be started.
    pub attributes: OpAttributesWithParent,
    /// The channel on which the result, successful or not, will be sent.
    pub result_tx: mpsc::Sender<PayloadId>,
}

/// A request to reset the engine forkchoice.
/// Optionally contains a channel to send back the response if the caller would like to know that
/// the request was successfully processed.
#[derive(Debug)]
pub struct ResetRequest {
    /// response will be sent to this channel, if `Some`.
    pub result_tx: mpsc::Sender<EngineClientResult<()>>,
}

/// A request to seal and canonicalize a payload.
/// Contains the `PayloadId`, attributes, and a channel to send back the result.
#[derive(Debug)]
pub struct SealRequest {
    /// The `PayloadId` to seal and canonicalize.
    pub payload_id: PayloadId,
    /// The attributes necessary for the seal operation.
    pub attributes: OpAttributesWithParent,
    /// When set, the payload is fetched as soon as the execution layer reports it worth sealing,
    /// and at this instant at the latest.
    pub ready_deadline: Option<Instant>,
    /// The channel on which the result, successful or not, will be sent.
    pub result_tx: mpsc::Sender<Result<SealedPayload, SealTaskError>>,
}

/// A request for the [`BlockInfo`] of an L2 block, by hash.
#[derive(Debug)]
pub struct L2BlockInfoRequest {
    /// The hash of the L2 block to look up.
    pub hash: B256,
    /// The channel on which the result, successful or not, will be sent. `Ok(None)` means the
    /// execution layer does not have the block.
    pub result_tx: mpsc::Sender<EngineClientResult<Option<BlockInfo>>>,
}
