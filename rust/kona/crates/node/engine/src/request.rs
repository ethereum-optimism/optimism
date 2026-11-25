//! Requests sent to the node engine actor and their replies.

use crate::{
    BuildTaskError, CanonicalizeTaskError, ConsolidateInput, DeniedBlock, FinalizeBlockId,
    InsertTaskError, OpenBlockError, SealTaskError, VerifierHead,
};
use alloy_eips::BlockNumHash;
use alloy_rpc_types_engine::PayloadId;
use kona_protocol::OpAttributesWithParent;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadAttributes};
use thiserror::Error;
use tokio::sync::mpsc;

/// A request handled by the node engine actor.
#[derive(Debug)]
pub enum EngineActorRequest {
    /// Request to start building a block.
    Build(Box<BuildRequest>),
    /// Request to process derived attributes or delegated safe block information.
    ProcessSafeL2Signal(ConsolidateInput),
    /// Request to process the finalized L2 block. A node following a cross-chain verifier takes
    /// it as its own L1 finality, which bounds the verifier's finalized head.
    ProcessFinalizedL2Block(Box<FinalizeBlockId>),
    /// Request to move the safe label to a cross-chain verifier's head.
    ProcessVerifiedSafeHead(Box<VerifierHead>),
    /// Request to process a cross-chain verifier's finalized head, which the node's own L1
    /// finality bounds.
    ProcessVerifiedFinalizedHead(Box<VerifierHead>),
    /// Request to process a cross-chain verifier's denied blocks above the finalized head, with
    /// the L1 blocks the denials were decided on, or with `None`, when the verifier cannot be
    /// reached, to judge its last answer again on the node's L1 chain.
    ProcessDeniedBlocks(Option<Vec<DeniedBlock>>),
    /// Request to process a received unsafe L2 block.
    ProcessUnsafeL2Block(Box<OpExecutionPayloadEnvelope>),
    /// Request to reset the forkchoice and signal derivation to reset.
    Reset(Box<ResetRequest>),
    /// Request to seal a block.
    Seal(Box<SealRequest>),
    /// Request to canonicalize a committed sequenced block.
    Canonicalize(Box<CanonicalizeRequest>),
    /// Request to run a block-building step of the `opstack` RPC namespace.
    OpStack(Box<OpStackRequest>),
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

/// A block-building step of the `opstack` RPC namespace, through which a sequencer outside the
/// node, such as op-test-sequencer, builds blocks on it. Each step sends its outcome on its
/// channel.
#[derive(Debug, Clone)]
pub enum OpStackRequest {
    /// Starts building a block with the attributes on the parent, which can be any block the
    /// execution layer has.
    Open {
        /// The block to build on.
        parent: BlockNumHash,
        /// The attributes of the block to build.
        attributes: Box<OpPayloadAttributes>,
        /// The channel on which the build's payload ID, or the failure, is sent.
        result_tx: mpsc::Sender<Result<PayloadId, OpenBlockError>>,
    },
    /// Fetches the payload of a build, without importing it.
    GetPayload {
        /// The payload ID of the build.
        payload_id: PayloadId,
        /// The timestamp of the block being built, which selects the Engine API version.
        timestamp: u64,
        /// The channel on which the payload, or the failure, is sent.
        result_tx: mpsc::Sender<Result<OpExecutionPayloadEnvelope, SealTaskError>>,
    },
    /// Imports a payload and makes it the unsafe head.
    Commit {
        /// The payload to import.
        payload: Box<OpExecutionPayloadEnvelope>,
        /// The channel on which the outcome of the import is sent.
        result_tx: mpsc::Sender<Result<(), InsertTaskError>>,
    },
}
