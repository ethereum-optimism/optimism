use alloy_rpc_types_engine::PayloadId;
use kona_engine::{BuildTaskError, CommitBlockError, EngineQueries, SealTaskError};
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use thiserror::Error;
use tokio::sync::mpsc;

/// The result of an Engine client call.
pub type ChainControllerClientResult<T> = Result<T, ChainControllerClientError>;

/// Error making requests to the `BlockEngine`.
#[derive(Debug, Error)]
pub enum ChainControllerClientError {
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
pub struct ChainControllerRpcRequest(pub Box<EngineQueries>);

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
    pub result_tx: mpsc::Sender<ChainControllerClientResult<()>>,
}

/// A request to rewind the chain onto `parent`, disowning every block above it.
///
/// This is the engine half of applying a block invalidation: the interop verifier has decided the
/// block above `parent` is invalid, recorded it on the deny list, and now needs the chain off it so
/// derivation can rebuild the height — where the deny list turns the rebuild into a deposits-only
/// replacement. Unlike [`ResetRequest`], which discovers its landing point by walkback, the target
/// here is fixed by the caller: the parent of the invalidated block, which op-supernode's
/// `RewindEngine` likewise receives rather than derives
/// (`op-supernode/supernode/chain_container/chain_container.go:749`).
#[derive(Debug)]
pub struct RewindRequest {
    /// The block the chain must sit on once the rewind completes: the parent of the invalidated
    /// block.
    pub parent: L2BlockInfo,
    /// The channel on which the result, successful or not, will be sent.
    pub result_tx: mpsc::Sender<ChainControllerClientResult<()>>,
}

/// A request to commit an externally built payload as the chain's unsafe head, answering the
/// caller: `opstack_commitBlockV1`'s write.
///
/// This is the unsafe-block import the gossip path performs fire-and-forget
/// ([`ChainControllerRequest::ProcessUnsafeL2Block`]), with a result channel: op-node's
/// `CommitBlock` returns the `engine_newPayload` verdict to its caller, so this does too.
///
/// [`ChainControllerRequest::ProcessUnsafeL2Block`]: crate::ChainControllerRequest::ProcessUnsafeL2Block
#[derive(Debug)]
pub struct CommitRequest {
    /// The payload to commit.
    pub envelope: OpExecutionPayloadEnvelope,
    /// The channel on which the result, successful or not, will be sent.
    pub result_tx: mpsc::Sender<Result<L2BlockInfo, CommitBlockError>>,
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
