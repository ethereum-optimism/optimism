use alloy_rpc_types_engine::PayloadId;
use kona_engine::{BuildTaskError, ConsolidateInput, EngineQueries, SealTaskError};
use kona_protocol::OpAttributesWithParent;
use kona_rpc::{RollupBoostAdminQuery, RollupBoostHealthQuery};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use thiserror::Error;
use tokio::sync::mpsc;

/// The result of an Engine client call.
pub type EngineClientResult<T> = Result<T, EngineClientError>;

/// Error making requests to the BlockEngine.
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

/// Inbound requests that the [`crate::EngineActor`] can process.
#[derive(Debug)]
pub enum EngineActorRequest {
    /// Request to build.
    BuildRequest(Box<BuildRequest>),
    /// Request to consolidate using a safe L2 signal from attributes or delegated safe-block
    /// derivation
    ProcessSafeL2SignalRequest(ConsolidateInput),
    /// Request to finalize the L2 block at the provided block number.
    ProcessFinalizedL2BlockNumberRequest(Box<u64>),
    /// Request to insert the provided unsafe block.
    ProcessUnsafeL2BlockRequest(Box<OpExecutionPayloadEnvelope>),
    /// Request to reset engine forkchoice.
    ResetRequest(Box<ResetRequest>),
    /// Request for the engine to process the provided RPC request.
    RpcRequest(Box<EngineRpcRequest>),
    /// Request to seal the block with the provided details.
    SealRequest(Box<SealRequest>),
}

/// RPC Request for the engine to handle.
#[derive(Debug)]
pub enum EngineRpcRequest {
    /// Engine RPC query.
    EngineQuery(Box<EngineQueries>),
    /// Rollup boost admin request.
    RollupBoostAdminRequest(Box<RollupBoostAdminQuery>),
    /// Rollup boost health request.
    RollupBoostHealthRequest(Box<RollupBoostHealthQuery>),
}

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
    /// The channel on which the result, successful or not, will be sent.
    pub result_tx: mpsc::Sender<Result<OpExecutionPayloadEnvelope, SealTaskError>>,
}
