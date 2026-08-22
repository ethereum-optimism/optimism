//! Server implementation of the experimental `opstack` block-building namespace.
//!
//! Mirrors op-node's `opstackAPI` (`op-node/node/api.go`) method for method, delegating to the
//! same machinery op-node's does:
//!
//! - `openBlockV1` is op-node's `EngineController::OpenBlock` (`op-node/rollup/engine/api.go`): a
//!   direct `engine_forkchoiceUpdated` on the chain's execution layer, head at the given parent and
//!   the given attributes, returning the build job's
//!   [`PayloadId`](alloy_rpc_types_engine::PayloadId).
//! - `cancelBlockV1` and `sealBlockV1` are `CancelBlock`/`SealBlock`: a direct `engine_getPayload`,
//!   versioned by the job's timestamp. Sealing does *not* canonicalize the block — that is
//!   `commitBlockV1`'s job, exactly as in op-node.
//! - `commitBlockV1` is `CommitBlock`: `engine_newPayload` plus the unsafe-head move and the
//!   canonicalizing forkchoice update. Those writes move the node's own head state, which the
//!   [`ChainController`] owns, so this one goes through its request queue
//!   ([`ChainControllerRequest::CommitBlock`]) rather than straight to the execution layer.
//! - `publishBlockV1` is `PublishBlock` (`op-node/node/node.go`): the signed envelope goes out on
//!   the chain's gossip topic with the signature the caller supplied. The network actor owns the
//!   swarm, so the payload is scheduled onto its publish queue; op-node awaits the pubsub publish
//!   itself, which is the one behavioural difference — a gossip failure here is logged by the
//!   network actor rather than returned.
//!
//! Error codes are op-node's (`op-service/apis/opstack.go`): open/seal/cancel failures come back
//! as `-401xx` build errors, and the errors op-node returns as plain Go errors come back as the
//! default server error code, `-32000`, matching go-ethereum's encoding of uncoded errors.
//!
//! [`ChainController`]: crate::ChainController

use crate::{ChainControllerRequest, CommitRequest, actors::network::PayloadToPublish};
use alloy_primitives::Signature;
use alloy_rpc_types_engine::{ExecutionPayload, ForkchoiceState, PayloadStatusEnum};
use alloy_transport::TransportErrorKind;
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorObject, ErrorObjectOwned},
};
use kona_engine::{
    CommitBlockError, EngineClient, EngineForkchoiceVersion, EngineGetPayloadVersion,
};
use kona_genesis::RollupConfig;
use kona_rpc::{
    BUILD_ERR_CODE_INVALID_INPUT, BUILD_ERR_CODE_OTHER, BUILD_ERR_CODE_PRESTATE,
    BUILD_ERR_CODE_TEMPORARY, BUILD_ERR_CODE_UNKNOWN_PAYLOAD, EngineRpcClient, OpStackApiServer,
    OpStackBlockId, PayloadInfo, SignedExecutionPayloadEnvelope,
};
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadAttributes};
use std::sync::Arc;
use tokio::sync::mpsc;

/// go-ethereum's `defaultErrorCode`: what op-node's uncoded (plain `error`) returns encode to.
const DEFAULT_ERROR_CODE: i32 = -32000;

/// The engine-API error code an execution layer answers `engine_getPayload` with for a payload id
/// it does not know (`eth.UnknownPayload`).
const ENGINE_UNKNOWN_PAYLOAD: i64 = -38001;

/// `eth.InvalidForkchoiceState`.
const ENGINE_INVALID_FORKCHOICE_STATE: i64 = -38002;

/// `eth.InvalidPayloadAttributes`.
const ENGINE_INVALID_PAYLOAD_ATTRIBUTES: i64 = -38003;

/// The engine-API error code range (`eth.ErrorCode.IsEngineError`).
const ENGINE_ERROR_RANGE: std::ops::RangeInclusive<i64> = -38100..=-38000;

/// Serves the `opstack` namespace for one chain.
///
/// Holds its own [`EngineClient`] — op-node's opstack API shares the driver's engine client, but
/// what it does with it (a forkchoice update to start a build, `engine_getPayload`) never touches
/// op-node's head state either; the head-moving call, commit, goes through the
/// [`ChainController`]'s queue here just as every other head write does.
///
/// [`ChainController`]: crate::ChainController
#[derive(Debug)]
pub struct OpStackRpc<EngineClient_, EngineRpcClient_> {
    /// The chain's rollup config, which selects engine API versions by timestamp.
    cfg: Arc<RollupConfig>,
    /// A direct client on the chain's execution-layer engine API.
    engine: Arc<EngineClient_>,
    /// The read-only view of the engine state, for the safe and finalized forkchoice labels.
    engine_query: EngineRpcClient_,
    /// The chain controller's request queue, where the commit write goes.
    controller_request_tx: mpsc::Sender<ChainControllerRequest>,
    /// The network actor's publish queue, where signed payloads go out on gossip.
    publish_tx: mpsc::Sender<PayloadToPublish>,
}

impl<EngineClient_, EngineRpcClient_> OpStackRpc<EngineClient_, EngineRpcClient_> {
    /// Constructs a new [`OpStackRpc`].
    pub const fn new(
        cfg: Arc<RollupConfig>,
        engine: Arc<EngineClient_>,
        engine_query: EngineRpcClient_,
        controller_request_tx: mpsc::Sender<ChainControllerRequest>,
        publish_tx: mpsc::Sender<PayloadToPublish>,
    ) -> Self {
        Self { cfg, engine, engine_query, controller_request_tx, publish_tx }
    }
}

/// An uncoded op-node error: the default server error code with the given message.
fn plain_error(message: String) -> ErrorObjectOwned {
    ErrorObject::owned(DEFAULT_ERROR_CODE, message, None::<()>)
}

/// A `-401xx` build error, code and message as op-node's `opstack` namespace serves them.
fn build_error(code: i32, message: String) -> ErrorObjectOwned {
    ErrorObject::owned(code, message, None::<()>)
}

/// The engine-API error code carried by a transport error, if it is a JSON-RPC error response.
fn error_resp_code(err: &alloy_transport::RpcError<TransportErrorKind>) -> Option<i64> {
    err.as_error_resp().map(|resp| resp.code)
}

/// Maps an `engine_getPayload` failure the way op-node's `CancelBlock`/`SealBlock` do: the
/// execution layer not knowing the payload id is `-40120`, anything else is `-40199`.
fn get_payload_error(
    verb: &str,
    err: alloy_transport::RpcError<TransportErrorKind>,
) -> ErrorObjectOwned {
    if error_resp_code(&err) == Some(ENGINE_UNKNOWN_PAYLOAD) {
        return build_error(BUILD_ERR_CODE_UNKNOWN_PAYLOAD, "unknown payload".to_string());
    }
    build_error(BUILD_ERR_CODE_OTHER, format!("failed to {verb} payload: {err}"))
}

#[async_trait]
impl<EngineClient_, EngineRpcClient_> OpStackApiServer
    for OpStackRpc<EngineClient_, EngineRpcClient_>
where
    EngineClient_: EngineClient + 'static,
    EngineRpcClient_: EngineRpcClient + 'static,
{
    async fn open_block_v1(
        &self,
        parent: OpStackBlockId,
        attrs: OpPayloadAttributes,
    ) -> RpcResult<PayloadInfo> {
        // The parent must be a block the engine has, before its hash is advertised as the head to
        // build on: op-node's `OpenBlock` makes the same check with the same failure.
        let known = self
            .engine
            .get_l2_block(parent.hash.into())
            .await
            .map_err(|err| {
                plain_error(format!(
                    "failed to retrieve parent block {}:{} from engine: {err}",
                    parent.hash, parent.number
                ))
            })?
            .is_some();
        if !known {
            return Err(plain_error(format!(
                "failed to retrieve parent block {}:{} from engine: block not found",
                parent.hash, parent.number
            )));
        }

        // The safe and finalized labels come from the node's current view, the head from the
        // caller: op-node's `OpenBlock` builds the same forkchoice state from its controller.
        let state = self.engine_query.get_state().await?;
        let forkchoice = ForkchoiceState {
            head_block_hash: parent.hash,
            safe_block_hash: state.sync_state.cross_safe_head().block_info.hash,
            finalized_block_hash: state.sync_state.finalized_head().block_info.hash,
        };

        let timestamp = attrs.payload_attributes.timestamp;
        let version = EngineForkchoiceVersion::from_cfg(&self.cfg, timestamp);
        let update = match version {
            EngineForkchoiceVersion::V3 => {
                self.engine.fork_choice_updated_v3(forkchoice, Some(attrs)).await
            }
            EngineForkchoiceVersion::V2 => {
                self.engine.fork_choice_updated_v2(forkchoice, Some(attrs)).await
            }
        }
        .map_err(|err| {
            // op-node's `startPayload` error mapping, code for code.
            match error_resp_code(&err) {
                Some(ENGINE_INVALID_FORKCHOICE_STATE) => build_error(
                    BUILD_ERR_CODE_PRESTATE,
                    format!(
                        "need reset to resolve pre-state problem: pre-block-creation forkchoice \
                         update was inconsistent with engine, need reset to resolve: {err}"
                    ),
                ),
                Some(ENGINE_INVALID_PAYLOAD_ATTRIBUTES) => build_error(
                    BUILD_ERR_CODE_PRESTATE,
                    format!(
                        "invalid payload attributes: payload attributes are not valid, cannot \
                         build block: {err}"
                    ),
                ),
                Some(code) if ENGINE_ERROR_RANGE.contains(&code) => build_error(
                    BUILD_ERR_CODE_PRESTATE,
                    format!(
                        "need reset to resolve pre-state problem: unexpected engine error code \
                         in forkchoice-updated response: {err}"
                    ),
                ),
                _ => build_error(
                    BUILD_ERR_CODE_TEMPORARY,
                    format!(
                        "temporarily cannot insert new safe block: failed to create new block \
                         via forkchoice: {err}"
                    ),
                ),
            }
        })?;

        match update.payload_status.status {
            PayloadStatusEnum::Invalid { .. } => Err(build_error(
                BUILD_ERR_CODE_PRESTATE,
                format!("invalid payload attributes: {}", update.payload_status.status),
            )),
            PayloadStatusEnum::Valid => update.payload_id.map_or_else(
                || {
                    Err(build_error(
                        BUILD_ERR_CODE_TEMPORARY,
                        "temporarily cannot insert new safe block: nil id in forkchoice result \
                         when expecting a valid ID"
                            .to_string(),
                    ))
                },
                |id| Ok(PayloadInfo { id, timestamp }),
            ),
            status => Err(build_error(
                BUILD_ERR_CODE_TEMPORARY,
                format!("temporarily cannot insert new safe block: {status}"),
            )),
        }
    }

    async fn cancel_block_v1(&self, id: PayloadInfo) -> RpcResult<()> {
        // op-node's `CancelBlock`: fetch the payload and discard it. The engine API has no
        // explicit cancel; retrieving the payload completes the job.
        let version = EngineGetPayloadVersion::from_cfg(&self.cfg, id.timestamp);
        let result = match version {
            EngineGetPayloadVersion::V5 => self.engine.get_payload_v5(id.id).await.map(|_| ()),
            EngineGetPayloadVersion::V4 => self.engine.get_payload_v4(id.id).await.map(|_| ()),
            EngineGetPayloadVersion::V3 => self.engine.get_payload_v3(id.id).await.map(|_| ()),
            EngineGetPayloadVersion::V2 => self.engine.get_payload_v2(id.id).await.map(|_| ()),
        };
        result.map_err(|err| get_payload_error("cancel", err))
    }

    async fn seal_block_v1(&self, id: PayloadInfo) -> RpcResult<OpExecutionPayloadEnvelope> {
        // op-node's `SealBlock`: `engine_getPayload` versioned by the job's timestamp, and
        // nothing more — the block is not canonical until `commitBlockV1`.
        let version = EngineGetPayloadVersion::from_cfg(&self.cfg, id.timestamp);
        let envelope = match version {
            EngineGetPayloadVersion::V5 => {
                let payload = self
                    .engine
                    .get_payload_v5(id.id)
                    .await
                    .map_err(|e| get_payload_error("seal", e))?;
                OpExecutionPayloadEnvelope::V4 {
                    parent_beacon_block_root: payload.parent_beacon_block_root,
                    payload: payload.execution_payload,
                }
            }
            EngineGetPayloadVersion::V4 => {
                let payload = self
                    .engine
                    .get_payload_v4(id.id)
                    .await
                    .map_err(|e| get_payload_error("seal", e))?;
                OpExecutionPayloadEnvelope::V4 {
                    parent_beacon_block_root: payload.parent_beacon_block_root,
                    payload: payload.execution_payload,
                }
            }
            EngineGetPayloadVersion::V3 => {
                let payload = self
                    .engine
                    .get_payload_v3(id.id)
                    .await
                    .map_err(|e| get_payload_error("seal", e))?;
                OpExecutionPayloadEnvelope::V3 {
                    parent_beacon_block_root: payload.parent_beacon_block_root,
                    payload: payload.execution_payload,
                }
            }
            EngineGetPayloadVersion::V2 => {
                let payload = self
                    .engine
                    .get_payload_v2(id.id)
                    .await
                    .map_err(|e| get_payload_error("seal", e))?;
                match payload.execution_payload.into_payload() {
                    ExecutionPayload::V1(payload) => OpExecutionPayloadEnvelope::V1(payload),
                    ExecutionPayload::V2(payload) => OpExecutionPayloadEnvelope::V2(payload),
                    _ => {
                        return Err(build_error(
                            BUILD_ERR_CODE_OTHER,
                            "failed to seal payload: the engine answered getPayloadV2 with a \
                             post-V2 payload"
                                .to_string(),
                        ));
                    }
                }
            }
        };
        Ok(envelope)
    }

    async fn commit_block_v1(&self, signed: SignedExecutionPayloadEnvelope) -> RpcResult<()> {
        // op-node's `CommitBlock` never reads the signature either: committing makes the block
        // canonical locally, and the signature only matters to gossip peers.
        let (result_tx, mut result_rx) = mpsc::channel(1);
        self.controller_request_tx
            .send(ChainControllerRequest::CommitBlock(Box::new(CommitRequest {
                envelope: signed.envelope,
                result_tx,
            })))
            .await
            .map_err(|_| {
                build_error(
                    BUILD_ERR_CODE_OTHER,
                    "failed to commit block: the chain controller is gone".to_string(),
                )
            })?;

        let result = result_rx.recv().await.ok_or_else(|| {
            build_error(
                BUILD_ERR_CODE_OTHER,
                "failed to commit block: the chain controller dropped the request".to_string(),
            )
        })?;

        result.map(|_| ()).map_err(|err| match err {
            // op-node's `CommitBlock` answers an execution-invalid verdict with the invalid-input
            // build error; every other failure is a plain error.
            CommitBlockError::Insert(kona_engine::InsertTaskError::UnexpectedPayloadStatus(
                status,
            )) => build_error(BUILD_ERR_CODE_INVALID_INPUT, format!("execution invalid: {status}")),
            CommitBlockError::Insert(kona_engine::InsertTaskError::InsertFailed(err)) => {
                plain_error(format!("failed to insert payload: {err}"))
            }
            CommitBlockError::Insert(kona_engine::InsertTaskError::ForkchoiceUpdateFailed(err)) => {
                plain_error(format!("failed to update engine forkchoice: {err}"))
            }
            CommitBlockError::Insert(err) => plain_error(format!("invalid payload: {err}")),
            err @ CommitBlockError::DoesNotDescendFromLocalSafe => {
                plain_error(format!("failed to insert payload: {err}"))
            }
        })
    }

    async fn publish_block_v1(&self, signed: SignedExecutionPayloadEnvelope) -> RpcResult<()> {
        // The signature goes out exactly as given, as op-node's `PublishBlock` publishes it.
        let signature = Signature::from_raw(signed.signature.as_slice())
            .map_err(|err| plain_error(format!("invalid signature: {err}")))?;

        self.publish_tx
            .send(PayloadToPublish::Signed(signed.envelope, signature))
            .await
            .map_err(|_| plain_error("failed to publish payload: the network is gone".to_string()))
    }
}
