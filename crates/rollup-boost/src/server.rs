use crate::debug_api::ExecutionMode;
use crate::{BlockSelectionPolicy, EngineApiExt};
use crate::{
    client::rpc::RpcClient,
    debug_api::DebugServer,
    health::HealthHandle,
    payload::{
        NewPayload, NewPayloadV3, NewPayloadV4, OpExecutionPayloadEnvelope, PayloadSource,
        PayloadTraceContext, PayloadVersion,
    },
    probe::{Health, Probes},
};
use alloy_primitives::{B256, Bytes, bytes};
use alloy_rpc_types_engine::{
    ExecutionPayload, ExecutionPayloadV3, ForkchoiceState, ForkchoiceUpdated, PayloadId,
    PayloadStatus,
};
use alloy_rpc_types_eth::{Block, BlockNumberOrTag};
use http_body_util::{BodyExt, Full};
use jsonrpsee::RpcModule;
use jsonrpsee::core::BoxError;
use jsonrpsee::core::{RegisterMethodError, RpcResult, async_trait};
use jsonrpsee::proc_macros::rpc;
use jsonrpsee::server::HttpBody;
use jsonrpsee::server::HttpRequest;
use jsonrpsee::server::HttpResponse;
use jsonrpsee::types::ErrorObject;
use jsonrpsee::types::error::INVALID_REQUEST_CODE;
use metrics::counter;
use op_alloy_rpc_types_engine::{
    OpExecutionPayloadEnvelopeV3, OpExecutionPayloadEnvelopeV4, OpExecutionPayloadV4,
    OpPayloadAttributes,
};
use opentelemetry::trace::SpanKind;
use parking_lot::Mutex;
use std::sync::Arc;
use std::time::Duration;
use tracing::{error, info, instrument};

pub type Request = HttpRequest;
pub type Response = HttpResponse;
pub type BufferedRequest = http::Request<Full<bytes::Bytes>>;
pub type BufferedResponse = http::Response<Full<bytes::Bytes>>;

#[derive(Clone)]
pub struct RollupBoostServer {
    pub l2_client: Arc<RpcClient>,
    pub builder_client: Arc<dyn EngineApiExt>,
    pub payload_trace_context: Arc<PayloadTraceContext>,
    block_selection_policy: Option<BlockSelectionPolicy>,
    execution_mode: Arc<Mutex<ExecutionMode>>,
    probes: Arc<Probes>,
}

impl RollupBoostServer {
    pub fn new(
        l2_client: RpcClient,
        builder_client: Arc<dyn EngineApiExt>,
        initial_execution_mode: Arc<Mutex<ExecutionMode>>,
        block_selection_policy: Option<BlockSelectionPolicy>,
        probes: Arc<Probes>,
        health_check_interval: u64,
        max_unsafe_interval: u64,
    ) -> Self {
        HealthHandle {
            probes: probes.clone(),
            builder_client: builder_client.clone(),
            health_check_interval: Duration::from_secs(health_check_interval),
            max_unsafe_interval,
        }
        .spawn();

        Self {
            l2_client: Arc::new(l2_client),
            builder_client,
            block_selection_policy,
            payload_trace_context: Arc::new(PayloadTraceContext::new()),
            execution_mode: initial_execution_mode,
            probes,
        }
    }

    pub async fn start_debug_server(&self, debug_addr: &str) -> eyre::Result<()> {
        let server = DebugServer::new(self.execution_mode.clone());
        server.run(debug_addr).await?;
        Ok(())
    }

    pub fn execution_mode(&self) -> ExecutionMode {
        *self.execution_mode.lock()
    }

    async fn new_payload(&self, new_payload: NewPayload) -> RpcResult<PayloadStatus> {
        let execution_payload = ExecutionPayload::from(new_payload.clone());
        let block_hash = execution_payload.block_hash();
        let parent_hash = execution_payload.parent_hash();
        info!(message = "received new_payload", "block_hash" = %block_hash, "version" = new_payload.version().as_str());

        if let Some(causes) = self
            .payload_trace_context
            .trace_ids_from_parent_hash(&parent_hash)
            .await
        {
            causes.iter().for_each(|cause| {
                tracing::Span::current().follows_from(cause);
            });
        }

        self.payload_trace_context
            .remove_by_parent_hash(&parent_hash)
            .await;

        // async call to builder to sync the builder node
        if !self.execution_mode().is_disabled() {
            let builder = self.builder_client.clone();
            let new_payload_clone = new_payload.clone();
            tokio::spawn(async move { builder.new_payload(new_payload_clone).await });
        }
        Ok(self.l2_client.new_payload(new_payload).await?)
    }

    async fn get_payload(
        &self,
        payload_id: PayloadId,
        version: PayloadVersion,
    ) -> RpcResult<OpExecutionPayloadEnvelope> {
        let l2_fut = self.l2_client.get_payload(payload_id, version);

        // If execution mode is disabled, return the l2 payload without sending
        // the request to the builder
        if self.execution_mode().is_disabled() {
            return match l2_fut.await {
                Ok(payload) => {
                    self.probes.set_health(Health::Healthy);
                    let context = PayloadSource::L2;
                    tracing::Span::current().record("payload_source", context.to_string());
                    counter!("rpc.blocks_created", "source" => context.to_string()).increment(1);

                    let execution_payload = ExecutionPayload::from(payload.clone());
                    info!(
                        message = "returning block",
                        "hash" = %execution_payload.block_hash(),
                        "number" = %execution_payload.block_number(),
                        %context,
                        %payload_id,
                        // Add an extra label to know that this is the disabled execution mode path
                        "execution_mode" = "disabled",
                    );

                    Ok(payload)
                }

                Err(e) => {
                    self.probes.set_health(Health::ServiceUnavailable);
                    Err(e.into())
                }
            };
        }

        // Forward the get payload request to the builder
        let builder_fut = async {
            if let Some(cause) = self.payload_trace_context.trace_id(&payload_id).await {
                tracing::Span::current().follows_from(cause);
            }
            if !self
                .payload_trace_context
                .has_builder_payload(&payload_id)
                .await
            {
                info!(message = "builder has no payload, skipping get_payload call to builder");
                tracing::Span::current().record("builder_has_payload", false);
                return RpcResult::Ok(None);
            }

            // Get payload and validate with the local l2 client
            tracing::Span::current().record("builder_has_payload", true);
            info!(message = "builder has payload, calling get_payload on builder");

            let payload = self.builder_client.get_payload(payload_id, version).await?;
            let _ = self
                .l2_client
                .new_payload(NewPayload::from(payload.clone()))
                .await?;

            Ok(Some(payload))
        };

        let (l2_payload, builder_payload) = tokio::join!(l2_fut, builder_fut);

        // Evaluate the builder and l2 response and select the final payload
        let (payload, context) = {
            let l2_payload =
                l2_payload.inspect_err(|_| self.probes.set_health(Health::ServiceUnavailable))?;
            self.probes.set_health(Health::Healthy);

            // Convert Result<Option<Payload>> to Option<Payload> by extracting the inner Option.
            // If there's an error, log it and return None instead.
            let builder_payload = builder_payload
                .map_err(|e| {
                    error!(message = "error getting payload from builder", error = %e);
                    e
                })
                .unwrap_or(None);

            if let Some(builder_payload) = builder_payload {
                // Record the delta (gas and txn) between the builder and l2 payload
                let span = tracing::Span::current();
                span.record(
                    "gas_delta",
                    (builder_payload.gas_used() - l2_payload.gas_used()).to_string(),
                );
                span.record(
                    "tx_count_delta",
                    (builder_payload.tx_count() - l2_payload.tx_count()).to_string(),
                );

                // If execution mode is set to DryRun, fallback to the l2_payload,
                // otherwise prefer the builder payload
                if self.execution_mode().is_dry_run() {
                    (l2_payload, PayloadSource::L2)
                } else if let Some(selection_policy) = &self.block_selection_policy {
                    selection_policy.select_block(builder_payload, l2_payload)
                } else {
                    (builder_payload, PayloadSource::Builder)
                }
            } else {
                // Only update the health status if the builder payload fails
                // and execution mode is not set to DryRun
                if !self.execution_mode().is_dry_run() {
                    self.probes.set_health(Health::PartialContent);
                }
                (l2_payload, PayloadSource::L2)
            }
        };

        tracing::Span::current().record("payload_source", context.to_string());
        // To maintain backwards compatibility with old metrics, we need to record blocks built
        // This is temporary until we migrate to the new metrics
        counter!("rpc.blocks_created", "source" => context.to_string()).increment(1);

        let inner_payload = ExecutionPayload::from(payload.clone());
        let block_hash = inner_payload.block_hash();
        let block_number = inner_payload.block_number();

        // Note: This log message is used by integration tests to track payload context.
        // While not ideal to rely on log parsing, it provides a reliable way to verify behavior.
        // Happy to consider an alternative approach later on.
        info!(
            message = "returning block",
            "hash" = %block_hash,
            "number" = %block_number,
            %context,
            %payload_id,
        );
        Ok(payload)
    }
}

impl TryInto<RpcModule<()>> for RollupBoostServer {
    type Error = RegisterMethodError;

    fn try_into(self) -> Result<RpcModule<()>, Self::Error> {
        let mut module: RpcModule<()> = RpcModule::new(());
        module.merge(EngineApiServer::into_rpc(self))?;

        for method in module.method_names() {
            info!(?method, "method registered");
        }

        Ok(module)
    }
}

#[rpc(server, client)]
pub trait EngineApi {
    #[method(name = "engine_forkchoiceUpdatedV3")]
    async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> RpcResult<ForkchoiceUpdated>;

    #[method(name = "engine_getPayloadV3")]
    async fn get_payload_v3(
        &self,
        payload_id: PayloadId,
    ) -> RpcResult<OpExecutionPayloadEnvelopeV3>;

    #[method(name = "engine_newPayloadV3")]
    async fn new_payload_v3(
        &self,
        payload: ExecutionPayloadV3,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
    ) -> RpcResult<PayloadStatus>;

    #[method(name = "engine_getPayloadV4")]
    async fn get_payload_v4(
        &self,
        payload_id: PayloadId,
    ) -> RpcResult<OpExecutionPayloadEnvelopeV4>;

    #[method(name = "engine_newPayloadV4")]
    async fn new_payload_v4(
        &self,
        payload: OpExecutionPayloadV4,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
        execution_requests: Vec<Bytes>,
    ) -> RpcResult<PayloadStatus>;

    #[method(name = "eth_getBlockByNumber")]
    async fn get_block_by_number(&self, number: BlockNumberOrTag, full: bool) -> RpcResult<Block>;
}

#[async_trait]
impl EngineApiServer for RollupBoostServer {
    #[instrument(
        skip_all,
        err,
        fields(
            otel.kind = ?SpanKind::Server,
            head_block_hash = %fork_choice_state.head_block_hash,
            timestamp = ?payload_attributes.as_ref().map(|attrs| attrs.payload_attributes.timestamp),
            payload_id
        )
    )]
    async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> RpcResult<ForkchoiceUpdated> {
        // Send the FCU to the default l2 client
        let l2_fut = self
            .l2_client
            .fork_choice_updated_v3(fork_choice_state, payload_attributes.clone());

        // If execution mode is disabled, return the l2 client response immediately
        if self.execution_mode().is_disabled() {
            return Ok(l2_fut.await?);
        }

        let span = tracing::Span::current();
        // If the fcu contains payload attributes and the tx pool is disabled,
        // only forward the FCU to the default l2 client
        if let Some(attrs) = payload_attributes.as_ref() {
            if attrs.no_tx_pool.unwrap_or_default() {
                let l2_response = l2_fut.await?;
                if let Some(payload_id) = l2_response.payload_id {
                    info!(
                        message = "block building started",
                        "payload_id" = %payload_id,
                        "builder_building" = false,
                    );

                    self.payload_trace_context
                        .store(
                            payload_id,
                            fork_choice_state.head_block_hash,
                            false,
                            span.id(),
                        )
                        .await;
                }

                // We always return the value from the l2 client
                return Ok(l2_response);
            } else {
                // If the tx pool is enabled, forward the fcu
                // to both the builder and the default l2 client
                let builder_fut = self
                    .builder_client
                    .fork_choice_updated_v3(fork_choice_state, payload_attributes);

                let (l2_result, builder_result) = tokio::join!(l2_fut, builder_fut);
                let l2_response = l2_result?;

                if let Some(payload_id) = l2_response.payload_id {
                    info!(
                        message = "block building started",
                        "payload_id" = %payload_id,
                        "builder_building" = builder_result.is_ok(),
                    );

                    self.payload_trace_context
                        .store(
                            payload_id,
                            fork_choice_state.head_block_hash,
                            builder_result.is_ok(),
                            span.id(),
                        )
                        .await;
                }

                return Ok(l2_response);
            }
        } else {
            // If the FCU does not contain payload attributes
            // forward the fcu to the builder to keep it synced and immediately return the l2
            // response without awaiting the builder
            let builder_client = self.builder_client.clone();
            tokio::spawn(async move {
                // It is not critical to wait for the builder response here
                // During moments of high load, Op-node can send hundreds of FCU requests
                // and we want to ensure that we don't block the main thread in those scenarios
                builder_client
                    .fork_choice_updated_v3(fork_choice_state, payload_attributes)
                    .await
            });
            return Ok(l2_fut.await?);
        }
    }

    #[instrument(
        skip_all,
        err,
        fields(
            otel.kind = ?SpanKind::Server,
            %payload_id,
            payload_source,
            gas_delta,
            tx_count_delta,
            builder_has_payload,
        )
    )]
    async fn get_payload_v3(
        &self,
        payload_id: PayloadId,
    ) -> RpcResult<OpExecutionPayloadEnvelopeV3> {
        info!("received get_payload_v3");

        match self.get_payload(payload_id, PayloadVersion::V3).await? {
            OpExecutionPayloadEnvelope::V3(v3) => Ok(v3),
            OpExecutionPayloadEnvelope::V4(_) => Err(ErrorObject::owned(
                INVALID_REQUEST_CODE,
                "Payload version 4 not supported",
                None::<String>,
            )),
        }
    }

    #[instrument(
        skip_all,
        err,
        fields(
            otel.kind = ?SpanKind::Server,
        )
    )]
    async fn new_payload_v3(
        &self,
        payload: ExecutionPayloadV3,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
    ) -> RpcResult<PayloadStatus> {
        info!("received new_payload_v3");

        self.new_payload(NewPayload::V3(NewPayloadV3 {
            payload,
            versioned_hashes,
            parent_beacon_block_root,
        }))
        .await
    }

    #[instrument(
        skip_all,
        err,
        fields(
            otel.kind = ?SpanKind::Server,
            %payload_id,
            payload_source,
            gas_delta,
            tx_count_delta,
        )
    )]
    async fn get_payload_v4(
        &self,
        payload_id: PayloadId,
    ) -> RpcResult<OpExecutionPayloadEnvelopeV4> {
        info!("received get_payload_v4");

        match self.get_payload(payload_id, PayloadVersion::V4).await? {
            OpExecutionPayloadEnvelope::V4(v4) => Ok(v4),
            OpExecutionPayloadEnvelope::V3(_) => Err(ErrorObject::owned(
                INVALID_REQUEST_CODE,
                "Payload version 4 not supported",
                None::<String>,
            )),
        }
    }

    #[instrument(
        skip_all,
        err,
        fields(
            otel.kind = ?SpanKind::Server,
        )
    )]
    async fn new_payload_v4(
        &self,
        payload: OpExecutionPayloadV4,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
        execution_requests: Vec<Bytes>,
    ) -> RpcResult<PayloadStatus> {
        info!("received new_payload_v4");

        self.new_payload(NewPayload::V4(NewPayloadV4 {
            payload,
            versioned_hashes,
            parent_beacon_block_root,
            execution_requests,
        }))
        .await
    }

    async fn get_block_by_number(&self, number: BlockNumberOrTag, full: bool) -> RpcResult<Block> {
        Ok(self.l2_client.get_block_by_number(number, full).await?)
    }
}

pub async fn into_buffered_request(req: HttpRequest) -> Result<BufferedRequest, BoxError> {
    let (parts, body) = req.into_parts();
    let bytes = body.collect().await?.to_bytes();
    let full = Full::<bytes::Bytes>::from(bytes.clone());
    Ok(http::Request::from_parts(parts, full))
}

pub fn from_buffered_request(req: BufferedRequest) -> HttpRequest {
    req.map(HttpBody::new)
}

#[cfg(test)]
#[allow(clippy::complexity)]
pub mod tests {
    use super::*;
    use crate::probe::ProbeLayer;
    use crate::proxy::ProxyLayer;
    use alloy_primitives::hex;
    use alloy_primitives::{FixedBytes, U256};
    use alloy_rpc_types_engine::JwtSecret;
    use alloy_rpc_types_engine::{
        BlobsBundleV1, ExecutionPayloadV1, ExecutionPayloadV2, PayloadStatusEnum,
    };
    use http::{StatusCode, Uri};
    use jsonrpsee::RpcModule;
    use jsonrpsee::http_client::HttpClient;
    use jsonrpsee::server::{Server, ServerBuilder, ServerHandle};
    use parking_lot::Mutex;
    use std::net::SocketAddr;
    use std::str::FromStr;
    use std::sync::Arc;
    use tokio::time::sleep;

    #[derive(Debug, Clone)]
    pub struct MockEngineServer {
        fcu_requests: Arc<Mutex<Vec<(ForkchoiceState, Option<OpPayloadAttributes>)>>>,
        pub get_payload_requests: Arc<Mutex<Vec<PayloadId>>>,
        new_payload_requests: Arc<Mutex<Vec<(ExecutionPayloadV3, Vec<B256>, B256)>>>,
        fcu_response: RpcResult<ForkchoiceUpdated>,
        get_payload_response: RpcResult<OpExecutionPayloadEnvelopeV3>,
        new_payload_response: RpcResult<PayloadStatus>,

        pub override_payload_id: Option<PayloadId>,
    }

    impl MockEngineServer {
        pub fn new() -> Self {
            Self {
                fcu_requests: Arc::new(Mutex::new(vec![])),
                get_payload_requests: Arc::new(Mutex::new(vec![])),
                new_payload_requests: Arc::new(Mutex::new(vec![])),
                fcu_response: Ok(ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid))),
                get_payload_response: Ok(OpExecutionPayloadEnvelopeV3{
                    execution_payload: ExecutionPayloadV3 {
                            payload_inner: ExecutionPayloadV2 {
                                payload_inner: ExecutionPayloadV1 {
                                    base_fee_per_gas:  U256::from(7u64),
                                    block_number: 0xa946u64,
                                    block_hash: hex!("a5ddd3f286f429458a39cafc13ffe89295a7efa8eb363cf89a1a4887dbcf272b").into(),
                                    logs_bloom: hex!("00200004000000000000000080000000000200000000000000000000000000000000200000000000000000000000000000000000800000000200000000000000000000000000000000000008000000200000000000000000000001000000000000000000000000000000800000000000000000000100000000000030000000000000000040000000000000000000000000000000000800080080404000000000000008000000000008200000000000200000000000000000000000000000000000000002000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000100000000000000000000").into(),
                                    extra_data: hex!("d883010d03846765746888676f312e32312e31856c696e7578").into(),
                                    gas_limit: 0x1c9c380,
                                    gas_used: 0x1f4a9,
                                    timestamp: 0x651f35b8,
                                    fee_recipient: hex!("f97e180c050e5ab072211ad2c213eb5aee4df134").into(),
                                    parent_hash: hex!("d829192799c73ef28a7332313b3c03af1f2d5da2c36f8ecfafe7a83a3bfb8d1e").into(),
                                    prev_randao: hex!("753888cc4adfbeb9e24e01c84233f9d204f4a9e1273f0e29b43c4c148b2b8b7e").into(),
                                    receipts_root: hex!("4cbc48e87389399a0ea0b382b1c46962c4b8e398014bf0cc610f9c672bee3155").into(),
                                    state_root: hex!("017d7fa2b5adb480f5e05b2c95cb4186e12062eed893fc8822798eed134329d1").into(),
                                    transactions: vec![],
                                },
                                withdrawals: vec![],
                            },
                            blob_gas_used: 0xc0000,
                        excess_blob_gas: 0x580000,
                    },
                    block_value: U256::from(0),
                    blobs_bundle: BlobsBundleV1{
                        commitments: vec![],
                        proofs: vec![],
                        blobs: vec![],
                    },
                should_override_builder: false,
                parent_beacon_block_root: B256::ZERO,
            }),
            override_payload_id: None,
            new_payload_response: Ok(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
        }
        }
    }

    struct TestHarness {
        l2_server: ServerHandle,
        l2_mock: MockEngineServer,
        builder_server: ServerHandle,
        builder_mock: MockEngineServer,
        server: ServerHandle,
        server_addr: SocketAddr,
        rpc_client: HttpClient,
        http_client: reqwest::Client,
    }

    impl TestHarness {
        async fn new(
            l2_mock: Option<MockEngineServer>,
            builder_mock: Option<MockEngineServer>,
        ) -> Self {
            let jwt_secret = JwtSecret::random();

            let l2_mock = l2_mock.unwrap_or(MockEngineServer::new());
            let builder_mock = builder_mock.unwrap_or(MockEngineServer::new());
            let (l2_server, l2_server_addr) = spawn_server(l2_mock.clone()).await;
            let (builder_server, builder_server_addr) = spawn_server(builder_mock.clone()).await;

            let l2_auth_rpc = Uri::from_str(&format!("http://{l2_server_addr}")).unwrap();
            let l2_client =
                RpcClient::new(l2_auth_rpc.clone(), jwt_secret, 2000, PayloadSource::L2).unwrap();

            let builder_auth_rpc = Uri::from_str(&format!("http://{builder_server_addr}")).unwrap();
            let builder_client = Arc::new(
                RpcClient::new(
                    builder_auth_rpc.clone(),
                    jwt_secret,
                    2000,
                    PayloadSource::Builder,
                )
                .unwrap(),
            );

            let (probe_layer, probes) = ProbeLayer::new();
            let execution_mode = Arc::new(Mutex::new(ExecutionMode::Enabled));

            let rollup_boost = RollupBoostServer::new(
                l2_client,
                builder_client,
                execution_mode.clone(),
                None,
                probes.clone(),
                60,
                5,
            );

            let module: RpcModule<()> = rollup_boost.try_into().unwrap();

            let http_middleware =
                tower::ServiceBuilder::new()
                    .layer(probe_layer)
                    .layer(ProxyLayer::new(
                        l2_auth_rpc,
                        jwt_secret,
                        1,
                        builder_auth_rpc,
                        jwt_secret,
                        1,
                        probes,
                        execution_mode.clone(),
                    ));

            let server = Server::builder()
                .set_http_middleware(http_middleware)
                .build("127.0.0.1:0".parse::<SocketAddr>().unwrap())
                .await
                .unwrap();

            let server_addr = server.local_addr().expect("missing server address");

            let server = server.start(module);

            let rpc_client = HttpClient::builder()
                .build(format!("http://{server_addr}"))
                .unwrap();
            let http_client = reqwest::Client::new();

            TestHarness {
                l2_server,
                l2_mock,
                builder_server,
                builder_mock,
                server,
                server_addr,
                rpc_client,
                http_client,
            }
        }

        async fn get(&self, path: &str) -> reqwest::Response {
            self.http_client
                .get(format!("http://{}/{}", self.server_addr, path))
                .send()
                .await
                .unwrap()
        }

        async fn cleanup(self) {
            self.l2_server.stop().unwrap();
            self.l2_server.stopped().await;
            self.builder_server.stop().unwrap();
            self.builder_server.stopped().await;
            self.server.stop().unwrap();
            self.server.stopped().await;
        }
    }

    #[tokio::test]
    async fn engine_success() {
        let test_harness = TestHarness::new(None, None).await;

        // Since no blocks have been created, the service should be unavailable
        let health = test_harness.get("healthz").await;
        assert_eq!(health.status(), StatusCode::OK);

        // test fork_choice_updated_v3 success
        let fcu = ForkchoiceState {
            head_block_hash: FixedBytes::random(),
            safe_block_hash: FixedBytes::random(),
            finalized_block_hash: FixedBytes::random(),
        };
        let fcu_response = test_harness
            .rpc_client
            .fork_choice_updated_v3(fcu, None)
            .await;
        assert!(fcu_response.is_ok());
        let fcu_requests = test_harness.l2_mock.fcu_requests.clone();
        {
            let fcu_requests_mu = fcu_requests.lock();
            let fcu_requests_builder = test_harness.builder_mock.fcu_requests.clone();
            let fcu_requests_builder_mu = fcu_requests_builder.lock();
            assert_eq!(fcu_requests_mu.len(), 1);
            assert_eq!(fcu_requests_builder_mu.len(), 1);
            let req: &(ForkchoiceState, Option<OpPayloadAttributes>) =
                fcu_requests_mu.first().unwrap();
            assert_eq!(req.0, fcu);
            assert_eq!(req.1, None);
        }

        // test new_payload_v3 success
        let new_payload_response = test_harness
            .rpc_client
            .new_payload_v3(
                test_harness
                    .l2_mock
                    .get_payload_response
                    .clone()
                    .unwrap()
                    .execution_payload
                    .clone(),
                vec![],
                B256::ZERO,
            )
            .await;
        assert!(new_payload_response.is_ok());
        let new_payload_requests = test_harness.l2_mock.new_payload_requests.clone();
        {
            let new_payload_requests_mu = new_payload_requests.lock();
            let new_payload_requests_builder =
                test_harness.builder_mock.new_payload_requests.clone();
            let new_payload_requests_builder_mu = new_payload_requests_builder.lock();
            assert_eq!(new_payload_requests_mu.len(), 1);
            assert_eq!(new_payload_requests_builder_mu.len(), 1);
            let req: &(ExecutionPayloadV3, Vec<FixedBytes<32>>, B256) =
                new_payload_requests_mu.first().unwrap();
            assert_eq!(
                req.0,
                test_harness
                    .l2_mock
                    .get_payload_response
                    .clone()
                    .unwrap()
                    .execution_payload
                    .clone()
            );
            assert_eq!(req.1, Vec::<FixedBytes<32>>::new());
            assert_eq!(req.2, B256::ZERO);
        }

        // test get_payload_v3 success
        let get_payload_response = test_harness
            .rpc_client
            .get_payload_v3(PayloadId::new([0, 0, 0, 0, 0, 0, 0, 1]))
            .await;
        assert!(get_payload_response.is_ok());
        let get_payload_requests = test_harness.l2_mock.get_payload_requests.clone();
        {
            let get_payload_requests_mu = get_payload_requests.lock();
            let get_payload_requests_builder =
                test_harness.builder_mock.get_payload_requests.clone();
            let get_payload_requests_builder_mu = get_payload_requests_builder.lock();
            let new_payload_requests = test_harness.l2_mock.new_payload_requests.clone();
            let new_payload_requests_mu = new_payload_requests.lock();
            assert_eq!(get_payload_requests_builder_mu.len(), 0);
            assert_eq!(get_payload_requests_mu.len(), 1);
            assert_eq!(new_payload_requests_mu.len(), 1);
            let req: &PayloadId = get_payload_requests_mu.first().unwrap();
            assert_eq!(*req, PayloadId::new([0, 0, 0, 0, 0, 0, 0, 1]));
        }

        // Now that a block has been produced by the l2 but not the builder
        // the health status should be Partial Content
        let health = test_harness.get("healthz").await;
        assert_eq!(health.status(), StatusCode::PARTIAL_CONTENT);

        test_harness.cleanup().await;
    }

    #[tokio::test]
    async fn builder_payload_err() {
        let mut l2_mock = MockEngineServer::new();
        l2_mock.new_payload_response = l2_mock.new_payload_response.clone().map(|mut status| {
            status.status = PayloadStatusEnum::Invalid {
                validation_error: "test".to_string(),
            };
            status
        });
        l2_mock.get_payload_response = l2_mock.get_payload_response.clone().map(|mut payload| {
            payload.block_value = U256::from(10);
            payload
        });
        let test_harness = TestHarness::new(Some(l2_mock), None).await;

        // test get_payload_v3 return l2 payload if builder payload is invalid
        let get_payload_response = test_harness
            .rpc_client
            .get_payload_v3(PayloadId::new([0, 0, 0, 0, 0, 0, 0, 0]))
            .await;
        assert!(get_payload_response.is_ok());
        assert_eq!(get_payload_response.unwrap().block_value, U256::from(10));

        test_harness.cleanup().await;
    }

    pub async fn spawn_server(mock_engine_server: MockEngineServer) -> (ServerHandle, SocketAddr) {
        let server = ServerBuilder::default().build("127.0.0.1:0").await.unwrap();
        let server_addr = server.local_addr().expect("Missing local address");

        let mut module: RpcModule<()> = RpcModule::new(());

        module
            .register_method("engine_forkchoiceUpdatedV3", move |params, _, _| {
                let params: (ForkchoiceState, Option<OpPayloadAttributes>) = params.parse()?;
                let mut fcu_requests = mock_engine_server.fcu_requests.lock();
                fcu_requests.push(params);

                let mut response = mock_engine_server.fcu_response.clone();
                if let Ok(ref mut fcu_response) = response {
                    if let Some(override_id) = mock_engine_server.override_payload_id {
                        fcu_response.payload_id = Some(override_id);
                    }
                }

                response
            })
            .unwrap();

        module
            .register_method("engine_getPayloadV3", move |params, _, _| {
                let params: (PayloadId,) = params.parse()?;
                let mut get_payload_requests = mock_engine_server.get_payload_requests.lock();
                get_payload_requests.push(params.0);

                mock_engine_server.get_payload_response.clone()
            })
            .unwrap();

        module
            .register_method("engine_newPayloadV3", move |params, _, _| {
                let params: (ExecutionPayloadV3, Vec<B256>, B256) = params.parse()?;
                let mut new_payload_requests = mock_engine_server.new_payload_requests.lock();
                new_payload_requests.push(params);

                mock_engine_server.new_payload_response.clone()
            })
            .unwrap();

        (server.start(module), server_addr)
    }

    #[tokio::test]
    async fn test_local_external_payload_ids_same() {
        let same_id: PayloadId = PayloadId::new([0, 0, 0, 0, 0, 0, 0, 42]);

        let mut l2_mock = MockEngineServer::new();
        l2_mock.fcu_response = Ok(ForkchoiceUpdated::new(PayloadStatus::from_status(
            PayloadStatusEnum::Valid,
        ))
        .with_payload_id(same_id));

        let mut builder_mock = MockEngineServer::new();
        builder_mock.override_payload_id = Some(same_id);

        let test_harness =
            TestHarness::new(Some(l2_mock.clone()), Some(builder_mock.clone())).await;

        // Test FCU call
        let fcu = ForkchoiceState {
            head_block_hash: FixedBytes::random(),
            safe_block_hash: FixedBytes::random(),
            finalized_block_hash: FixedBytes::random(),
        };
        let fcu_response = test_harness
            .rpc_client
            .fork_choice_updated_v3(fcu, None)
            .await;
        assert!(fcu_response.is_ok());

        // wait for builder to observe the FCU call
        sleep(std::time::Duration::from_millis(100)).await;

        {
            let builder_fcu_req = builder_mock.fcu_requests.lock();
            assert_eq!(builder_fcu_req.len(), 1);
            assert_eq!(l2_mock.fcu_requests.lock().len(), 1);
        }

        // Test getPayload call
        let get_res = test_harness.rpc_client.get_payload_v3(same_id).await;
        assert!(get_res.is_ok());

        // wait for builder to observe the getPayload call
        sleep(std::time::Duration::from_millis(100)).await;

        {
            let builder_gp_reqs = builder_mock.get_payload_requests.lock();
            assert_eq!(builder_gp_reqs.len(), 0);
        }

        {
            let local_gp_reqs = l2_mock.get_payload_requests.lock();
            assert_eq!(local_gp_reqs.len(), 1);
            assert_eq!(local_gp_reqs[0], same_id);
        }

        test_harness.cleanup().await;
    }

    #[tokio::test]
    async fn has_builder_payload() {
        let payload_id: PayloadId = PayloadId::new([0, 0, 0, 0, 0, 0, 0, 42]);
        let mut l2_mock = MockEngineServer::new();
        l2_mock.fcu_response = Ok(ForkchoiceUpdated::new(PayloadStatus::from_status(
            PayloadStatusEnum::Valid,
        ))
        .with_payload_id(payload_id));
        l2_mock.get_payload_response = l2_mock.get_payload_response.clone().map(|mut payload| {
            payload.block_value = U256::from(10);
            payload
        });

        let mut builder_mock = MockEngineServer::new();
        builder_mock.fcu_response = Ok(ForkchoiceUpdated::new(PayloadStatus::from_status(
            PayloadStatusEnum::Syncing,
        ))
        .with_payload_id(payload_id));
        builder_mock.get_payload_response =
            builder_mock
                .get_payload_response
                .clone()
                .map(|mut payload| {
                    payload.block_value = U256::from(15);
                    payload
                });

        let test_harness = TestHarness::new(Some(l2_mock), Some(builder_mock)).await;
        let fcu = ForkchoiceState {
            head_block_hash: FixedBytes::random(),
            safe_block_hash: FixedBytes::random(),
            finalized_block_hash: FixedBytes::random(),
        };
        let mut payload_attributes = OpPayloadAttributes {
            gas_limit: Some(1000000),
            ..Default::default()
        };
        let fcu_response = test_harness
            .rpc_client
            .fork_choice_updated_v3(fcu, Some(payload_attributes.clone()))
            .await;
        assert!(fcu_response.is_ok());

        // no tx pool is false so should return the builder payload
        let get_payload_response = test_harness.rpc_client.get_payload_v3(payload_id).await;
        assert!(get_payload_response.is_ok());
        assert_eq!(get_payload_response.unwrap().block_value, U256::from(15));

        payload_attributes.no_tx_pool = Some(true);
        let fcu_response = test_harness
            .rpc_client
            .fork_choice_updated_v3(fcu, Some(payload_attributes))
            .await;
        assert!(fcu_response.is_ok());

        // no tx pool is true so should return the l2 payload
        let get_payload_response = test_harness.rpc_client.get_payload_v3(payload_id).await;
        assert!(get_payload_response.is_ok());
        assert_eq!(get_payload_response.unwrap().block_value, U256::from(10));

        test_harness.cleanup().await;
    }

    #[tokio::test]
    async fn l2_client_fails_fcu() {
        // If the canonical l2 client fails the FCU call, it does not matter what the builder returns
        // the FCU call should fail
        let mut l2_mock = MockEngineServer::new();
        l2_mock.fcu_response = Err(ErrorObject::owned(
            INVALID_REQUEST_CODE,
            "Payload version 4 not supported",
            None::<String>,
        ));

        let test_harness = TestHarness::new(Some(l2_mock), None).await;

        let fcu = ForkchoiceState {
            head_block_hash: FixedBytes::random(),
            safe_block_hash: FixedBytes::random(),
            finalized_block_hash: FixedBytes::random(),
        };
        let fcu_response = test_harness
            .rpc_client
            .fork_choice_updated_v3(fcu, None)
            .await;
        assert!(fcu_response.is_err());

        let payload_attributes = OpPayloadAttributes {
            gas_limit: Some(1000000),
            ..Default::default()
        };
        let fcu_response = test_harness
            .rpc_client
            .fork_choice_updated_v3(fcu, Some(payload_attributes))
            .await;
        assert!(fcu_response.is_err());
    }
}
