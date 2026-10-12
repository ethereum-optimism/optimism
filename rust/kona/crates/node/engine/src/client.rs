//! An Engine API Client.

use crate::{EngineForkchoiceVersion, EngineGetPayloadVersion, Metrics};
use alloy_eips::{BlockId, eip1898::BlockNumberOrTag};
use alloy_json_rpc::{RequestPacket, ResponsePacket};
use alloy_network::{Ethereum, Network};
use alloy_primitives::Bytes;
use alloy_provider::{EthGetBlock, Provider, RootProvider};
use alloy_rpc_client::ClientBuilder;
use alloy_rpc_types_engine::{
    ExecutionPayloadFieldV2, ExecutionPayloadInputV2, ExecutionPayloadV1, ForkchoiceState,
    ForkchoiceUpdated, JwtSecret, PayloadId, PayloadStatus,
};
use alloy_rpc_types_eth::Block;
use alloy_transport::{
    RpcError, TransportError, TransportErrorKind, TransportFut, TransportResult,
};
use alloy_transport_http::{
    AuthLayer, Http, HyperClient,
    hyper_util::{client::legacy::Client, rt::TokioExecutor},
};
use http_body_util::Full;
use kona_genesis::RollupConfig;
use op_alloy_network::Optimism;
use op_alloy_provider::ext::engine::OpEngineApi;
use op_alloy_rpc_types::Transaction;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadAttributes};
use std::{
    sync::Arc,
    task::{Context, Poll},
    time::{Duration, Instant},
};
use thiserror::Error;
use tower::{Layer, Service, ServiceBuilder, util::MapFutureLayer};
use url::Url;

/// Deadline for each request the engine client sends, to the Engine API or to L1.
///
/// It bounds one request, never an operation made of many requests, such as the block traversal
/// of a reset, so a long but healthy operation is not cut off.
const RPC_TIMEOUT: Duration = Duration::from_secs(10);

/// Bounds one request on an alloy transport by [`RPC_TIMEOUT`].
///
/// This wraps the transport's future rather than the HTTP client below it, because the HTTP
/// client's future completes when the response headers arrive and the transport reads the body
/// after that.
fn with_deadline(request: TransportFut<'static>) -> TransportFut<'static> {
    Box::pin(async move {
        tokio::time::timeout(RPC_TIMEOUT, request)
            .await
            .map_err(|_| TransportErrorKind::custom_str("RPC request timed out"))?
    })
}

/// Records how long each request to the L2 execution layer takes, labeled by its JSON-RPC method,
/// in the [`Metrics::ENGINE_METHOD_REQUEST_DURATION`] histogram. Failed requests are recorded too.
#[derive(Debug, Clone, Copy)]
struct RequestDurationLayer;

impl<S> Layer<S> for RequestDurationLayer {
    type Service = RequestDuration<S>;

    fn layer(&self, inner: S) -> Self::Service {
        RequestDuration(inner)
    }
}

/// The transport service that a [`RequestDurationLayer`] wraps.
#[derive(Debug, Clone)]
struct RequestDuration<S>(S);

impl<S> Service<RequestPacket> for RequestDuration<S>
where
    S: Service<RequestPacket, Response = ResponsePacket, Error = TransportError>,
    S::Future: Send + 'static,
{
    type Response = ResponsePacket;
    type Error = TransportError;
    type Future = TransportFut<'static>;

    fn poll_ready(&mut self, cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        self.0.poll_ready(cx)
    }

    fn call(&mut self, request: RequestPacket) -> Self::Future {
        let method = match &request {
            RequestPacket::Single(request) => request.method().to_string(),
            RequestPacket::Batch(_) => "batch".to_string(),
        };
        let response = self.0.call(request);
        Box::pin(async move {
            let start = Instant::now();
            let response = response.await;
            metrics::histogram!(Metrics::ENGINE_METHOD_REQUEST_DURATION, "method" => method)
                .record(start.elapsed().as_secs_f64());
            response
        })
    }
}

/// An error that occurred in the [`EngineClient`].
#[derive(Error, Debug)]
pub enum EngineClientError {
    /// An RPC error occurred
    #[error("An RPC error occurred: {0}")]
    RpcError(#[from] RpcError<TransportErrorKind>),
}
/// Client for L1 reads and L2 Engine API calls. Providers erase their transport type, so tests
/// use the same client with a mock transport. [`Self::forkchoice_updated`], [`Self::get_payload`]
/// and [`Self::new_payload`] select the Engine API method version.
#[derive(Clone, Debug)]
pub struct EngineClient {
    engine: RootProvider<Optimism>,
    l1_provider: RootProvider,
    cfg: Arc<RollupConfig>,
}

impl EngineClient {
    /// Creates a client from existing providers.
    pub const fn new(
        l1_provider: RootProvider,
        engine: RootProvider<Optimism>,
        cfg: Arc<RollupConfig>,
    ) -> Self {
        Self { engine, l1_provider, cfg }
    }

    /// Returns the rollup configuration.
    pub fn cfg(&self) -> &RollupConfig {
        &self.cfg
    }

    /// Fetches an L1 block.
    pub fn get_l1_block(
        &self,
        block: BlockId,
    ) -> EthGetBlock<<Ethereum as Network>::BlockResponse> {
        self.l1_provider.get_block(block)
    }

    /// Fetches an L2 block.
    pub fn get_l2_block(
        &self,
        block: BlockId,
    ) -> EthGetBlock<<Optimism as Network>::BlockResponse> {
        self.engine.get_block(block)
    }

    /// Submits a Paris payload, whose method is provided by Alloy's Ethereum Engine API extension.
    pub async fn new_payload_v1(
        &self,
        payload: ExecutionPayloadV1,
    ) -> TransportResult<PayloadStatus> {
        // OP Alloy's extension has no V1 method, and importing Alloy's Ethereum extension would
        // make the other versions' methods ambiguous.
        alloy_provider::ext::EngineApi::new_payload_v1(&self.engine, payload).await
    }

    /// Sends `engine_forkchoiceUpdated`, selecting the method version for the payload attributes'
    /// timestamp.
    ///
    /// Without payload attributes, the call does not depend on the version, so V3 is used.
    pub async fn forkchoice_updated(
        &self,
        state: ForkchoiceState,
        attributes: Option<OpPayloadAttributes>,
    ) -> TransportResult<ForkchoiceUpdated> {
        let version = attributes.as_ref().map_or(EngineForkchoiceVersion::V3, |attributes| {
            EngineForkchoiceVersion::from_cfg(&self.cfg, attributes.payload_attributes.timestamp)
        });
        match version {
            EngineForkchoiceVersion::V3 => self.fork_choice_updated_v3(state, attributes).await,
            EngineForkchoiceVersion::V2 => self.fork_choice_updated_v2(state, attributes).await,
        }
    }

    /// Fetches a built payload with `engine_getPayload`, selecting the method version for the
    /// payload's timestamp.
    ///
    /// - `engine_getPayloadV2` is used for payloads with a timestamp before the Ecotone fork.
    /// - `engine_getPayloadV3` is used for payloads with a timestamp after the Ecotone fork.
    /// - `engine_getPayloadV4` is used for payloads with a timestamp after the Isthmus fork.
    /// - `engine_getPayloadV5` is used for payloads with a timestamp after the Karst fork.
    pub async fn get_payload(
        &self,
        payload_id: PayloadId,
        timestamp: u64,
    ) -> TransportResult<OpExecutionPayloadEnvelope> {
        Ok(match EngineGetPayloadVersion::from_cfg(&self.cfg, timestamp) {
            EngineGetPayloadVersion::V5 => {
                // Osaka (Karst) reuses the V4-shaped envelope; only the engine method bumps to V5.
                let payload = self.get_payload_v5(payload_id).await?;
                OpExecutionPayloadEnvelope::V4 {
                    parent_beacon_block_root: payload.parent_beacon_block_root,
                    payload: payload.execution_payload,
                }
            }
            EngineGetPayloadVersion::V4 => {
                let payload = self.get_payload_v4(payload_id).await?;
                OpExecutionPayloadEnvelope::V4 {
                    parent_beacon_block_root: payload.parent_beacon_block_root,
                    payload: payload.execution_payload,
                }
            }
            EngineGetPayloadVersion::V3 => {
                let payload = self.get_payload_v3(payload_id).await?;
                OpExecutionPayloadEnvelope::V3 {
                    parent_beacon_block_root: payload.parent_beacon_block_root,
                    payload: payload.execution_payload,
                }
            }
            EngineGetPayloadVersion::V2 => {
                let payload = self.get_payload_v2(payload_id).await?;
                match payload.execution_payload {
                    ExecutionPayloadFieldV2::V1(payload) => OpExecutionPayloadEnvelope::V1(payload),
                    ExecutionPayloadFieldV2::V2(payload) => OpExecutionPayloadEnvelope::V2(payload),
                }
            }
        })
    }

    /// Submits a payload with `engine_newPayload`, selecting the method version for the payload's
    /// envelope.
    pub async fn new_payload(
        &self,
        envelope: OpExecutionPayloadEnvelope,
    ) -> TransportResult<PayloadStatus> {
        match envelope {
            OpExecutionPayloadEnvelope::V1(payload) => self.new_payload_v1(payload).await,
            OpExecutionPayloadEnvelope::V2(payload) => {
                let payload_input = ExecutionPayloadInputV2 {
                    execution_payload: payload.payload_inner,
                    withdrawals: Some(payload.withdrawals),
                };
                self.new_payload_v2(payload_input).await
            }
            OpExecutionPayloadEnvelope::V3 { payload, parent_beacon_block_root } => {
                self.new_payload_v3(payload, parent_beacon_block_root).await
            }
            OpExecutionPayloadEnvelope::V4 { payload, parent_beacon_block_root } => {
                self.new_payload_v4(payload, parent_beacon_block_root).await
            }
        }
    }

    /// Fetches an L2 block with full transactions.
    pub async fn l2_block_by_label(
        &self,
        numtag: BlockNumberOrTag,
    ) -> Result<Option<Block<Transaction>>, EngineClientError> {
        Ok(self.engine.get_block_by_number(numtag).full().await?)
    }

    /// Creates an authenticated L2 provider with request timing and a deadline.
    pub fn rpc_client(addr: Url, jwt: JwtSecret) -> RootProvider<Optimism> {
        let hyper_client = Client::builder(TokioExecutor::new()).build_http::<Full<Bytes>>();
        let service = ServiceBuilder::new().layer(AuthLayer::new(jwt)).service(hyper_client);
        let http_hyper = Http::with_client(HyperClient::with_service(service), addr);
        let rpc_client = ClientBuilder::default()
            .layer(RequestDurationLayer)
            .layer(MapFutureLayer::new(with_deadline))
            .transport(http_hyper, false);
        RootProvider::new(rpc_client)
    }
}

/// Engine API calls use the L2 provider through OP Alloy's blanket implementation.
impl Provider<Optimism> for EngineClient {
    fn root(&self) -> &RootProvider<Optimism> {
        &self.engine
    }
}

/// Connection settings for an [`EngineClient`].
#[derive(Debug, Clone)]
pub struct EngineClientBuilder {
    /// The L2 Engine API endpoint URL.
    pub l2: Url,
    /// The L2 JWT secret.
    pub l2_jwt: JwtSecret,
    /// The L1 RPC URL.
    pub l1_rpc: Url,
    /// The rollup configuration.
    pub cfg: Arc<RollupConfig>,
}

impl EngineClientBuilder {
    /// Connects to the authenticated L2 engine and unauthenticated L1 RPC. Each request has a
    /// deadline.
    pub fn build(self) -> EngineClient {
        let engine = EngineClient::rpc_client(self.l2, self.l2_jwt);
        let l1_provider = RootProvider::new(
            ClientBuilder::default().layer(MapFutureLayer::new(with_deadline)).http(self.l1_rpc),
        );
        EngineClient::new(l1_provider, engine, self.cfg)
    }
}

#[cfg(test)]
mod request_duration_tests {
    use super::*;
    use alloy_json_rpc::{Id, Request, Response, ResponsePayload};
    use metrics_exporter_prometheus::PrometheusBuilder;
    use serde_json::value::RawValue;

    /// Sends one request for `method` through a [`RequestDurationLayer`] over a transport that
    /// answers with `response`.
    fn send(method: &'static str, response: Result<ResponsePacket, TransportError>) {
        let request = Request::new(method, Id::Number(1), ()).serialize().unwrap();
        let mut response = Some(response);
        let mut service = RequestDurationLayer.layer(tower::service_fn(move |_| {
            let response = response.take().expect("the transport is called once");
            async move { response }
        }));
        let call = service.call(RequestPacket::Single(request));
        let _ = tokio::runtime::Builder::new_current_thread().build().unwrap().block_on(call);
    }

    #[test]
    fn records_successful_and_failed_requests_by_method() {
        let recorder = PrometheusBuilder::new().build_recorder();
        let metrics = recorder.handle();
        metrics::with_local_recorder(&recorder, || {
            send(
                "engine_newPayloadV3",
                Ok(ResponsePacket::Single(Response {
                    id: Id::Number(1),
                    payload: ResponsePayload::Success(
                        RawValue::from_string("null".into()).unwrap(),
                    ),
                })),
            );
            send("engine_getPayloadV3", Err(TransportErrorKind::custom_str("unreachable")));
        });

        let rendered = metrics.render();
        for method in ["engine_newPayloadV3", "engine_getPayloadV3"] {
            let count = format!(
                "{}_count{{method=\"{method}\"}} 1",
                Metrics::ENGINE_METHOD_REQUEST_DURATION
            );
            assert!(rendered.contains(&count), "missing {count} in:\n{rendered}");
        }
    }
}

#[cfg(test)]
mod deadline_tests {
    use super::*;
    use tokio::{
        io::{AsyncReadExt, AsyncWriteExt},
        net::TcpListener,
        sync::mpsc,
        time::{self, Duration},
    };

    /// Reads each request, writes `response`, signals `answered`, then holds the connection open
    /// without finishing the response.
    async fn stalled_endpoint(response: &'static [u8], answered: mpsc::UnboundedSender<()>) -> Url {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let url = format!("http://{}", listener.local_addr().unwrap()).parse().unwrap();
        tokio::spawn(async move {
            while let Ok((mut stream, _)) = listener.accept().await {
                let answered = answered.clone();
                tokio::spawn(async move {
                    let mut request = [0; 4096];
                    let _ = stream.read(&mut request).await;
                    stream.write_all(response).await.unwrap();
                    answered.send(()).unwrap();
                    std::future::pending::<()>().await
                });
            }
        });
        url
    }

    /// Runs `call` in real time until the endpoint has answered, so whatever it sent is already in
    /// the socket buffer, then pauses the clock and expects `call` to fail at its deadline.
    async fn fails_at_deadline(
        call: impl Future<Output = bool> + Send + 'static,
        answered: &mut mpsc::UnboundedReceiver<()>,
    ) {
        let call = tokio::spawn(call);
        answered.recv().await.unwrap();
        time::pause();
        let paused_at = time::Instant::now();
        let failed = time::timeout(RPC_TIMEOUT * 2, call).await.expect("request has no deadline");
        assert!(failed.unwrap());
        // The deadline started before the clock was paused, and only the clock moved since. Timers
        // round up to the next millisecond.
        let waited = paused_at.elapsed();
        assert!(waited > RPC_TIMEOUT - Duration::from_secs(1));
        assert!(waited <= RPC_TIMEOUT + Duration::from_millis(1));
        time::resume();
    }

    #[rstest::rstest]
    #[case::no_response(b"")]
    #[case::stalled_body(
        b"HTTP/1.1 200 OK\r\ncontent-type: application/json\r\ncontent-length: 64\r\n\r\n"
    )]
    #[tokio::test]
    async fn stalled_engine_and_l1_requests_have_a_deadline(#[case] response: &'static [u8]) {
        let (answered_tx, mut answered) = mpsc::unbounded_channel();
        let client = Arc::new(
            EngineClientBuilder {
                l2: stalled_endpoint(response, answered_tx.clone()).await,
                l2_jwt: JwtSecret::random(),
                l1_rpc: stalled_endpoint(response, answered_tx).await,
                cfg: Arc::new(RollupConfig::default()),
            }
            .build(),
        );

        let engine = client.clone();
        fails_at_deadline(
            async move { engine.l2_block_by_label(BlockNumberOrTag::Latest).await.is_err() },
            &mut answered,
        )
        .await;
        fails_at_deadline(
            async move { client.get_l1_block(BlockId::latest()).await.is_err() },
            &mut answered,
        )
        .await;
    }
}

#[cfg(test)]
mod provider_tests {
    use super::*;
    use crate::test_utils::test_engine_client;
    use serde_json::json;

    #[tokio::test]
    async fn reads_l1_and_l2_blocks_from_their_providers() {
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        l1.expect_params("eth_getBlockByNumber", json!(["latest", false]), Option::<Block>::None);
        l2.expect_params(
            "eth_getBlockByNumber",
            json!(["latest", false]),
            Option::<Block<Transaction>>::None,
        );
        assert!(client.get_l1_block(BlockId::latest()).await.unwrap().is_none());
        assert!(client.get_l2_block(BlockId::latest()).await.unwrap().is_none());
        l1.assert_finished();
        l2.assert_finished();
    }
}

#[cfg(test)]
mod version_tests {
    use super::*;
    use crate::test_utils::{TestAttributesBuilder, test_engine_client};
    use alloy_primitives::B256;
    use alloy_rpc_types_engine::{
        ExecutionPayloadEnvelopeV2, ExecutionPayloadV2, ExecutionPayloadV3, PayloadStatusEnum,
    };
    use op_alloy_rpc_types_engine::{
        OpExecutionPayloadEnvelopeV3, OpExecutionPayloadEnvelopeV4, OpExecutionPayloadV4,
    };
    use rstest::rstest;
    use serde_json::json;

    /// Returns a configuration that activates Ecotone at 10, Isthmus at 20, and Karst at 30.
    fn config() -> Arc<RollupConfig> {
        let mut cfg = RollupConfig::default();
        cfg.hardforks.ecotone_time = Some(10);
        cfg.hardforks.isthmus_time = Some(20);
        cfg.hardforks.karst_time = Some(30);
        Arc::new(cfg)
    }

    fn valid() -> PayloadStatus {
        PayloadStatus::from_status(PayloadStatusEnum::Valid)
    }

    #[rstest]
    #[case::before_ecotone(Some(5), "engine_forkchoiceUpdatedV2")]
    #[case::after_ecotone(Some(15), "engine_forkchoiceUpdatedV3")]
    #[case::without_attributes(None, "engine_forkchoiceUpdatedV3")]
    #[tokio::test]
    async fn forkchoice_update_selects_version(
        #[case] timestamp: Option<u64>,
        #[case] method: &'static str,
    ) {
        let (client, l1, l2) = test_engine_client(config());
        let state = ForkchoiceState::default();
        let attributes = timestamp.map(|timestamp| {
            TestAttributesBuilder::new().with_timestamp(timestamp).build().attributes
        });
        l2.expect_params(method, json!([state, attributes]), ForkchoiceUpdated::new(valid()));

        let response = client.forkchoice_updated(state, attributes).await.unwrap();
        assert_eq!(response.payload_status, valid());
        l1.assert_finished();
        l2.assert_finished();
    }

    fn block() -> alloy_consensus::Block<op_alloy_consensus::OpTxEnvelope> {
        alloy_consensus::Block::default()
    }

    /// The parent beacon block root, distinct from the withdrawals root.
    const BEACON_ROOT: B256 = B256::repeat_byte(0x42);
    /// The Isthmus withdrawals root, distinct from the parent beacon block root.
    const WITHDRAWALS_ROOT: B256 = B256::repeat_byte(0x43);

    /// Returns a payload envelope, its `engine_newPayload` method, and the parameters it sends.
    fn v1() -> (OpExecutionPayloadEnvelope, &'static str, serde_json::Value) {
        let payload = ExecutionPayloadV1::from_block_slow(&block());
        (OpExecutionPayloadEnvelope::V1(payload.clone()), "engine_newPayloadV1", json!([payload]))
    }

    fn v2() -> (OpExecutionPayloadEnvelope, &'static str, serde_json::Value) {
        let payload = ExecutionPayloadV2::from_block_slow(&block());
        (OpExecutionPayloadEnvelope::V2(payload.clone()), "engine_newPayloadV2", json!([payload]))
    }

    fn v3() -> (OpExecutionPayloadEnvelope, &'static str, serde_json::Value) {
        let payload = ExecutionPayloadV3::from_block_slow(&block());
        (
            OpExecutionPayloadEnvelope::V3 {
                payload: payload.clone(),
                parent_beacon_block_root: BEACON_ROOT,
            },
            "engine_newPayloadV3",
            json!([payload, [], BEACON_ROOT]),
        )
    }

    fn v4() -> (OpExecutionPayloadEnvelope, &'static str, serde_json::Value) {
        let payload = OpExecutionPayloadV4::from_v3_with_withdrawals_root(
            ExecutionPayloadV3::from_block_slow(&block()),
            WITHDRAWALS_ROOT,
        );
        (
            OpExecutionPayloadEnvelope::V4 {
                payload: payload.clone(),
                parent_beacon_block_root: BEACON_ROOT,
            },
            "engine_newPayloadV4",
            json!([payload, [], BEACON_ROOT, []]),
        )
    }

    #[rstest]
    #[case::v1(v1())]
    #[case::v2(v2())]
    #[case::v3(v3())]
    #[case::v4(v4())]
    #[tokio::test]
    async fn new_payload_selects_version_by_envelope(
        #[case] call: (OpExecutionPayloadEnvelope, &'static str, serde_json::Value),
    ) {
        let (envelope, method, params) = call;
        let (client, l1, l2) = test_engine_client(config());
        l2.expect_params(method, params, valid());

        assert_eq!(client.new_payload(envelope).await.unwrap(), valid());
        l1.assert_finished();
        l2.assert_finished();
    }

    /// Exercise version selection and decoding through the production client and OP Alloy
    /// extension.
    #[rstest]
    #[case(4, "engine_getPayloadV2")]
    #[case(5, "engine_getPayloadV2")]
    #[case(15, "engine_getPayloadV3")]
    #[case(25, "engine_getPayloadV4")]
    #[case(35, "engine_getPayloadV5")]
    #[tokio::test]
    async fn payload_fetch_selects_version_and_decodes_reply(
        #[case] timestamp: u64,
        #[case] method: &'static str,
    ) {
        let (client, l1, l2) = test_engine_client(config());
        let id = PayloadId::new([1; 8]);
        let params = json!([id]);
        let root = B256::repeat_byte(0x42);
        let block = alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default();
        let expected = if timestamp < 5 {
            // A reply without withdrawals is a pre-Canyon V1 payload.
            let payload = ExecutionPayloadV1::from_block_slow(&block);
            l2.expect_params(
                method,
                params,
                ExecutionPayloadEnvelopeV2 {
                    execution_payload: ExecutionPayloadFieldV2::V1(payload.clone()),
                    block_value: Default::default(),
                },
            );
            OpExecutionPayloadEnvelope::V1(payload)
        } else if timestamp < 10 {
            let payload = ExecutionPayloadV2::from_block_slow(&block);
            l2.expect_params(
                method,
                params,
                ExecutionPayloadEnvelopeV2 {
                    execution_payload: ExecutionPayloadFieldV2::V2(payload.clone()),
                    block_value: Default::default(),
                },
            );
            OpExecutionPayloadEnvelope::V2(payload)
        } else if timestamp < 20 {
            let payload = ExecutionPayloadV3::from_block_slow(&block);
            l2.expect_params(
                method,
                params,
                OpExecutionPayloadEnvelopeV3 {
                    execution_payload: payload.clone(),
                    block_value: Default::default(),
                    blobs_bundle: Default::default(),
                    should_override_builder: false,
                    parent_beacon_block_root: root,
                },
            );
            OpExecutionPayloadEnvelope::V3 { payload, parent_beacon_block_root: root }
        } else {
            let payload = OpExecutionPayloadV4::from_v3_with_withdrawals_root(
                ExecutionPayloadV3::from_block_slow(&block),
                root,
            );
            l2.expect_params(
                method,
                params,
                OpExecutionPayloadEnvelopeV4 {
                    execution_payload: payload.clone(),
                    block_value: Default::default(),
                    blobs_bundle: Default::default(),
                    should_override_builder: false,
                    parent_beacon_block_root: root,
                    execution_requests: vec![],
                },
            );
            OpExecutionPayloadEnvelope::V4 { payload, parent_beacon_block_root: root }
        };
        assert_eq!(client.get_payload(id, timestamp).await.unwrap(), expected);
        l1.assert_finished();
        l2.assert_finished();
    }
}
