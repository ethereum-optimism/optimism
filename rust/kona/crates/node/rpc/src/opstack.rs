//! The experimental `opstack` block-building namespace, as op-node serves it.

use crate::OpStackApiServer;
use alloy_eips::BlockNumHash;
use alloy_primitives::{FixedBytes, Signature};
use alloy_rpc_types_engine::{INVALID_PAYLOAD_ATTRIBUTES_ERROR, PayloadId, PayloadStatusEnum};
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObjectOwned},
};
use kona_engine::{
    EngineActorRequest, EngineBuildError, InsertTaskError, OpStackRequest, OpenBlockError,
    SealTaskError,
};
use kona_gossip::SignedPayload;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadAttributes};
use serde::{Deserialize, Serialize};
use tokio::sync::mpsc;

/// The identifier of a block build: op-node's `eth.PayloadInfo`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub struct PayloadInfo {
    /// The payload ID the execution layer assigned to the build.
    pub id: PayloadId,
    /// The timestamp of the block being built.
    pub timestamp: u64,
}

/// A payload with its sequencer signature: op-node's `SignedExecutionPayloadEnvelope`.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct SignedExecutionPayloadEnvelope {
    /// The payload.
    pub envelope: OpExecutionPayloadEnvelope,
    /// The signature over the payload hash, as `r || s || v` with `v` the y-parity.
    pub signature: FixedBytes<65>,
}

/// The error codes of op-node's `opstack` namespace, from `op-service/apis/opstack.go`.
mod code {
    pub(super) const TEMPORARY: i32 = -40100;
    pub(super) const PRESTATE: i32 = -40101;
    pub(super) const INVALID_INPUT: i32 = -40110;
    pub(super) const UNKNOWN_PAYLOAD: i32 = -40120;
    pub(super) const OTHER: i32 = -40199;
    /// The code go-ethereum's RPC server answers a plain Go error with.
    pub(super) const SERVER_ERROR: i32 = -32000;
    /// The Engine API's code for a payload ID the execution layer does not know.
    pub(super) const ENGINE_UNKNOWN_PAYLOAD: i64 = -38001;
}

fn error(code: i32, message: impl Into<String>) -> ErrorObjectOwned {
    ErrorObjectOwned::owned(code, message, None::<()>)
}

/// The server of the `opstack` namespace. It runs every engine step through the engine actor, the
/// node's only writer to the execution layer, and gossips through the network actor.
#[derive(Debug, Clone)]
pub struct OpStackRpc {
    engine: mpsc::Sender<EngineActorRequest>,
    gossip: mpsc::Sender<SignedPayload>,
}

impl OpStackRpc {
    /// Constructs the server from the engine actor's and the network actor's request channels.
    pub const fn new(
        engine: mpsc::Sender<EngineActorRequest>,
        gossip: mpsc::Sender<SignedPayload>,
    ) -> Self {
        Self { engine, gossip }
    }

    /// Sends the request `make` builds around a reply channel to the engine actor, and waits for
    /// the reply.
    async fn run<T, E>(
        &self,
        make: impl FnOnce(mpsc::Sender<Result<T, E>>) -> OpStackRequest,
    ) -> RpcResult<Result<T, E>> {
        let (result_tx, mut result_rx) = mpsc::channel(1);
        self.engine
            .send(EngineActorRequest::OpStack(Box::new(make(result_tx))))
            .await
            .map_err(|_| error(code::SERVER_ERROR, "the engine stopped"))?;
        // The engine drops queued work when it resets.
        result_rx
            .recv()
            .await
            .ok_or_else(|| error(code::SERVER_ERROR, "the engine dropped the request"))
    }

    async fn payload(
        &self,
        id: PayloadInfo,
    ) -> RpcResult<Result<OpExecutionPayloadEnvelope, SealTaskError>> {
        self.run(|result_tx| OpStackRequest::GetPayload {
            payload_id: id.id,
            timestamp: id.timestamp,
            result_tx,
        })
        .await
    }
}

/// The error op-node's `OpenBlock` answers for `err`.
fn open_error(parent: BlockNumHash, err: OpenBlockError) -> ErrorObjectOwned {
    let build = match err {
        OpenBlockError::Build(build) => build,
        err => {
            return error(
                code::SERVER_ERROR,
                format!("failed to retrieve parent block {parent:?} from engine: {err}"),
            );
        }
    };
    let (code, message) = match &build {
        EngineBuildError::AttributesInsertionFailed(rpc) => {
            match rpc.as_error_resp().map(|resp| resp.code) {
                Some(engine) if engine == i64::from(INVALID_PAYLOAD_ATTRIBUTES_ERROR) => {
                    (code::PRESTATE, "invalid payload attributes")
                }
                // Any other Engine API error, such as an invalid forkchoice state.
                Some(engine) if -38100 < engine && engine <= -38000 => {
                    (code::PRESTATE, "need reset to resolve pre-state problem")
                }
                _ => (code::TEMPORARY, "temporarily cannot insert new safe block"),
            }
        }
        EngineBuildError::InvalidPayload(_) => (code::PRESTATE, "invalid payload attributes"),
        EngineBuildError::EngineSyncing |
        EngineBuildError::UnexpectedPayloadStatus(_) |
        EngineBuildError::MissingPayloadId => {
            (code::TEMPORARY, "temporarily cannot insert new safe block")
        }
        // op-node has no such check before a build.
        EngineBuildError::FinalizedAheadOfUnsafe(..) => (code::OTHER, "unknown error type"),
    };
    error(code, format!("{message}: {build}"))
}

/// The error op-node's `SealBlock` and `CancelBlock` answer for `err`.
fn payload_error(action: &str, err: SealTaskError) -> ErrorObjectOwned {
    if let SealTaskError::GetPayloadFailed(rpc) = &err &&
        rpc.as_error_resp().is_some_and(|resp| resp.code == code::ENGINE_UNKNOWN_PAYLOAD)
    {
        return error(code::UNKNOWN_PAYLOAD, "unknown payload");
    }
    error(code::OTHER, format!("failed to {action} payload: {err}"))
}

/// The error op-node's `CommitBlock` answers for `err`.
fn commit_error(err: InsertTaskError) -> ErrorObjectOwned {
    let (code, message) = match &err {
        InsertTaskError::UnexpectedPayloadStatus(PayloadStatusEnum::Invalid { .. }) => {
            (code::INVALID_INPUT, "execution invalid")
        }
        InsertTaskError::InsertFailed(_) | InsertTaskError::UnexpectedPayloadStatus(_) => {
            (code::SERVER_ERROR, "failed to insert payload")
        }
        InsertTaskError::ForkchoiceUpdateFailed(_) => {
            (code::SERVER_ERROR, "failed to update engine forkchoice")
        }
        InsertTaskError::FromBlockError(_) | InsertTaskError::L2BlockInfoConstruction(_) => {
            (code::SERVER_ERROR, "invalid payload")
        }
    };
    error(code, format!("{message}: {err}"))
}

#[async_trait]
impl OpStackApiServer for OpStackRpc {
    async fn open_block_v1(
        &self,
        parent: BlockNumHash,
        attributes: OpPayloadAttributes,
    ) -> RpcResult<PayloadInfo> {
        let timestamp = attributes.payload_attributes.timestamp;
        let result = self
            .run(|result_tx| OpStackRequest::Open {
                parent,
                attributes: Box::new(attributes),
                result_tx,
            })
            .await?;
        result.map(|id| PayloadInfo { id, timestamp }).map_err(|err| open_error(parent, err))
    }

    async fn cancel_block_v1(&self, id: PayloadInfo) -> RpcResult<()> {
        self.payload(id).await?.map(|_| ()).map_err(|err| payload_error("cancel", err))
    }

    async fn seal_block_v1(&self, id: PayloadInfo) -> RpcResult<OpExecutionPayloadEnvelope> {
        self.payload(id).await?.map_err(|err| payload_error("seal", err))
    }

    async fn commit_block_v1(&self, envelope: SignedExecutionPayloadEnvelope) -> RpcResult<()> {
        self.run(|result_tx| OpStackRequest::Commit {
            payload: Box::new(envelope.envelope),
            result_tx,
        })
        .await?
        .map_err(commit_error)
    }

    async fn publish_block_v1(&self, signed: SignedExecutionPayloadEnvelope) -> RpcResult<()> {
        let signature = Signature::from_raw(signed.signature.as_slice()).map_err(|err| {
            error(ErrorCode::InvalidParams.code(), format!("invalid signature: {err}"))
        })?;
        // Like the node's own blocks, the payload is gossiped on the network actor's turn, and a
        // failure to gossip it is logged there.
        self.gossip
            .send(SignedPayload { payload: signed.envelope, signature })
            .await
            .map_err(|_| error(code::SERVER_ERROR, "the network stopped"))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_json_rpc::ErrorPayload;
    use alloy_primitives::{B256, Bytes};
    use alloy_rpc_types_engine::ExecutionPayloadV1;
    use alloy_transport::RpcError;
    use jsonrpsee::{RpcModule, core::server::MethodsError};
    use serde_json::{Value, json};
    use std::sync::Arc;

    /// Serves the namespace with a stand-in engine actor that answers each request with what
    /// `answer` returns for it.
    fn serve(
        answer: impl Fn(OpStackRequest) + Send + 'static,
    ) -> (RpcModule<OpStackRpc>, mpsc::Receiver<SignedPayload>) {
        let (engine_tx, mut engine_rx) = mpsc::channel(1);
        let (gossip_tx, gossip_rx) = mpsc::channel(1);
        tokio::spawn(async move {
            while let Some(request) = engine_rx.recv().await {
                let EngineActorRequest::OpStack(request) = request else {
                    panic!("expected an opstack request, got {request:?}");
                };
                answer(*request);
            }
        });
        (OpStackRpc::new(engine_tx, gossip_tx).into_rpc(), gossip_rx)
    }

    async fn call(module: &RpcModule<OpStackRpc>, method: &str, params: Vec<Value>) -> Value {
        let mut rpc_params = jsonrpsee::core::params::ArrayParams::new();
        for param in params {
            rpc_params.insert(param).unwrap();
        }
        module.call(method, rpc_params).await.unwrap()
    }

    async fn call_error(
        module: &RpcModule<OpStackRpc>,
        method: &str,
        params: Vec<Value>,
    ) -> (i32, String) {
        let mut rpc_params = jsonrpsee::core::params::ArrayParams::new();
        for param in params {
            rpc_params.insert(param).unwrap();
        }
        match module.call::<_, Value>(method, rpc_params).await {
            Err(MethodsError::JsonRpc(error)) => (error.code(), error.message().to_owned()),
            outcome => panic!("expected a JSON-RPC error, got {outcome:?}"),
        }
    }

    fn engine_error(code: i64) -> RpcError<alloy_transport::TransportErrorKind> {
        RpcError::ErrorResp(ErrorPayload { code, message: "engine error".into(), data: None })
    }

    /// A V1 payload as op-node's JSON has it, without a parent beacon block root.
    fn go_envelope() -> (OpExecutionPayloadEnvelope, Value) {
        let payload = ExecutionPayloadV1 {
            parent_hash: B256::repeat_byte(1),
            fee_recipient: Default::default(),
            state_root: B256::repeat_byte(2),
            receipts_root: B256::repeat_byte(3),
            logs_bloom: Default::default(),
            prev_randao: B256::repeat_byte(4),
            block_number: 9,
            gas_limit: 30_000_000,
            gas_used: 0,
            timestamp: 18,
            extra_data: Bytes::new(),
            base_fee_per_gas: Default::default(),
            block_hash: B256::repeat_byte(5),
            transactions: vec![],
        };
        let envelope = OpExecutionPayloadEnvelope::V1(payload);
        let mut json = serde_json::to_value(&envelope).unwrap();
        json.as_object_mut().unwrap().remove("parentBeaconBlockRoot");
        (envelope, json)
    }

    /// `opstack_openBlockV1` takes op-node's block ID and Engine API attributes, and answers
    /// op-node's payload info.
    #[tokio::test]
    async fn open_block_takes_and_answers_op_node_json() {
        let parent = B256::repeat_byte(0xaa);
        let (module, _gossip) = serve(move |request| {
            let OpStackRequest::Open { parent: requested, attributes, result_tx } = request else {
                panic!("expected an open request");
            };
            assert_eq!(requested, BlockNumHash { number: 5, hash: parent });
            assert_eq!(attributes.payload_attributes.timestamp, 16);
            assert_eq!(attributes.no_tx_pool, Some(true));
            assert_eq!(attributes.gas_limit, Some(30_000_000));
            result_tx.try_send(Ok(PayloadId::new([7; 8]))).unwrap();
        });
        let attributes = json!({
            "timestamp": "0x10",
            "prevRandao": B256::ZERO,
            "suggestedFeeRecipient": "0x4200000000000000000000000000000000000011",
            "withdrawals": [],
            "noTxPool": true,
            "gasLimit": "0x1c9c380",
        });
        let info = call(
            &module,
            "opstack_openBlockV1",
            vec![json!({"hash": parent, "number": 5}), attributes],
        )
        .await;
        assert_eq!(info, json!({"id": "0x0707070707070707", "timestamp": 16}));
    }

    #[rstest::rstest]
    #[case::parent_not_found(OpenBlockError::ParentNotFound(B256::ZERO), code::SERVER_ERROR)]
    #[case::invalid_forkchoice(
        OpenBlockError::Build(EngineBuildError::AttributesInsertionFailed(Arc::new(engine_error(-38002)))),
        code::PRESTATE
    )]
    #[case::invalid_attributes(
        OpenBlockError::Build(EngineBuildError::AttributesInsertionFailed(Arc::new(engine_error(-38003)))),
        code::PRESTATE
    )]
    #[case::other_rpc_error(
        OpenBlockError::Build(EngineBuildError::AttributesInsertionFailed(Arc::new(engine_error(-32603)))),
        code::TEMPORARY
    )]
    #[case::invalid_payload(OpenBlockError::Build(EngineBuildError::InvalidPayload("bad".into())), code::PRESTATE)]
    #[case::syncing(OpenBlockError::Build(EngineBuildError::EngineSyncing), code::TEMPORARY)]
    #[case::missing_id(OpenBlockError::Build(EngineBuildError::MissingPayloadId), code::TEMPORARY)]
    #[tokio::test]
    async fn open_block_answers_op_node_error_codes(
        #[case] failure: OpenBlockError,
        #[case] expected: i32,
    ) {
        let failure = std::sync::Mutex::new(Some(failure));
        let (module, _gossip) = serve(move |request| {
            let OpStackRequest::Open { result_tx, .. } = request else {
                panic!("expected an open request");
            };
            result_tx.try_send(Err(failure.lock().unwrap().take().unwrap())).unwrap();
        });
        let attributes = serde_json::to_value(OpPayloadAttributes::default()).unwrap();
        let (code, _) = call_error(
            &module,
            "opstack_openBlockV1",
            vec![json!({"hash": B256::ZERO, "number": 0}), attributes],
        )
        .await;
        assert_eq!(code, expected);
    }

    /// Sealing answers the payload in op-node's JSON. An unknown payload ID is op-node's
    /// unknown-payload error, for sealing and for cancelling, which fetches the payload too.
    #[tokio::test]
    async fn seal_and_cancel_fetch_the_payload() {
        let (envelope, envelope_json) = go_envelope();
        let id = PayloadInfo { id: PayloadId::new([7; 8]), timestamp: 18 };
        let known = id.id;
        let (module, _gossip) = serve(move |request| {
            let OpStackRequest::GetPayload { payload_id, timestamp, result_tx } = request else {
                panic!("expected a getPayload request");
            };
            assert_eq!(timestamp, 18);
            let result = if payload_id == known {
                Ok(envelope.clone())
            } else {
                Err(SealTaskError::GetPayloadFailed(engine_error(code::ENGINE_UNKNOWN_PAYLOAD)))
            };
            result_tx.try_send(result).unwrap();
        });
        let id_json = json!({"id": "0x0707070707070707", "timestamp": 18});
        assert_eq!(serde_json::to_value(id).unwrap(), id_json);
        let sealed = call(&module, "opstack_sealBlockV1", vec![id_json.clone()]).await;
        assert_eq!(
            serde_json::from_value::<OpExecutionPayloadEnvelope>(sealed).unwrap().block_hash(),
            B256::repeat_byte(5)
        );
        assert_eq!(
            serde_json::from_value::<OpExecutionPayloadEnvelope>(envelope_json)
                .unwrap()
                .block_hash(),
            B256::repeat_byte(5)
        );
        assert_eq!(call(&module, "opstack_cancelBlockV1", vec![id_json]).await, Value::Null);

        let unknown = json!({"id": "0x0101010101010101", "timestamp": 18});
        for method in ["opstack_sealBlockV1", "opstack_cancelBlockV1"] {
            assert_eq!(
                call_error(&module, method, vec![unknown.clone()]).await,
                (code::UNKNOWN_PAYLOAD, "unknown payload".to_owned())
            );
        }
    }

    /// Committing imports the envelope of op-node's signed JSON.
    #[tokio::test]
    async fn commit_imports_the_signed_envelope() {
        let (envelope, envelope_json) = go_envelope();
        let (module, _gossip) = serve(move |request| {
            let OpStackRequest::Commit { payload, result_tx } = request else {
                panic!("expected a commit request");
            };
            assert_eq!(*payload, envelope);
            result_tx.try_send(Ok(())).unwrap();
        });
        let signed =
            json!({"envelope": envelope_json, "signature": format!("0x{}", "11".repeat(65))});
        assert_eq!(call(&module, "opstack_commitBlockV1", vec![signed]).await, Value::Null);
    }

    /// A payload the execution layer finds invalid is op-node's invalid-input error.
    #[tokio::test]
    async fn commit_of_an_invalid_payload_is_invalid_input() {
        let (_, envelope_json) = go_envelope();
        let (module, _gossip) = serve(|request| {
            let OpStackRequest::Commit { result_tx, .. } = request else {
                panic!("expected a commit request");
            };
            result_tx
                .try_send(Err(InsertTaskError::UnexpectedPayloadStatus(
                    PayloadStatusEnum::Invalid { validation_error: "bad".into() },
                )))
                .unwrap();
        });
        let signed =
            json!({"envelope": envelope_json, "signature": format!("0x{}", "11".repeat(65))});
        let (code, _) = call_error(&module, "opstack_commitBlockV1", vec![signed]).await;
        assert_eq!(code, code::INVALID_INPUT);
    }

    /// Publishing gossips the envelope with the caller's signature, whose `v` is the y-parity.
    #[tokio::test]
    async fn publish_gossips_the_signed_envelope() {
        let (envelope, envelope_json) = go_envelope();
        let (module, mut gossip) = serve(|request| panic!("unexpected engine request {request:?}"));
        let mut signature = [0x11u8; 65];
        signature[64] = 1;
        let signed =
            json!({"envelope": envelope_json, "signature": Bytes::from(signature.to_vec())});
        assert_eq!(call(&module, "opstack_publishBlockV1", vec![signed]).await, Value::Null);
        let published = gossip.recv().await.unwrap();
        assert_eq!(published.payload, envelope);
        assert_eq!(published.signature, Signature::from_raw_array(&signature).unwrap());
    }

    /// A request the engine drops, as it drops queued work on a reset, fails instead of hanging.
    #[tokio::test]
    async fn dropped_request_fails() {
        let (module, _gossip) = serve(drop);
        let id = json!({"id": "0x0707070707070707", "timestamp": 18});
        let (code, _) = call_error(&module, "opstack_sealBlockV1", vec![id]).await;
        assert_eq!(code, code::SERVER_ERROR);
    }
}
