//! An Engine API Client.

use crate::Metrics;
use alloy_eips::{BlockId, eip1898::BlockNumberOrTag};
use alloy_json_rpc::{RequestPacket, ResponsePacket};
use alloy_network::{Ethereum, Network};
use alloy_primitives::Bytes;
use alloy_provider::{EthGetBlock, Provider, RootProvider, ext::EngineApi};
use alloy_rpc_client::ClientBuilder;
use alloy_rpc_types_engine::{ExecutionPayloadV1, JwtSecret, PayloadStatus};
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
use kona_protocol::{FromBlockError, L2BlockInfo, OutputRoot, Predeploys};
use op_alloy_network::Optimism;
use op_alloy_rpc_types::Transaction;
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

/// An error that occurred in the [`EngineClient`] or [`EngineQueryClient`].
#[derive(Error, Debug)]
pub enum EngineClientError {
    /// An RPC error occurred
    #[error("An RPC error occurred: {0}")]
    RpcError(#[from] RpcError<TransportErrorKind>),

    /// An error occurred while decoding the payload
    #[error("An error occurred while decoding the payload: {0}")]
    BlockInfoDecodeError(#[from] FromBlockError),

    /// No L2 block was found for the requested number or tag.
    #[error("No L2 block found for block number or tag: {0}")]
    NoL2BlockFound(BlockNumberOrTag),

    /// The block has no withdrawals root while Isthmus is active.
    #[error("No block withdrawals root while Isthmus is active")]
    NoWithdrawalsRoot,
}
/// Client for L1 reads and L2 Engine API calls. Providers erase their transport type, so tests
/// use the same client with a mock transport. Task helpers select the Engine API version.
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
        self.engine.new_payload_v1(payload).await
    }

    /// Fetches an L2 block with full transactions.
    pub async fn l2_block_by_label(
        &self,
        numtag: BlockNumberOrTag,
    ) -> Result<Option<Block<Transaction>>, EngineClientError> {
        Ok(self.engine.get_block_by_number(numtag).full().await?)
    }

    /// Returns a restricted handle sharing this client's L2 connection.
    pub fn query_client(&self) -> EngineQueryClient {
        EngineQueryClient { engine: self.engine.clone(), cfg: self.cfg.clone() }
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

/// Read-only execution-layer handle that computes L2 outputs. Its provider is private and it
/// exposes no Engine API mutations.
#[derive(Clone, Debug)]
pub struct EngineQueryClient {
    engine: RootProvider<Optimism>,
    cfg: Arc<RollupConfig>,
}

impl EngineQueryClient {
    /// Reads the L2 block and computes its output root using the active fork rules.
    pub async fn output_at_block(
        &self,
        block: BlockNumberOrTag,
    ) -> Result<(L2BlockInfo, OutputRoot), EngineClientError> {
        let output_block = self.engine.get_block_by_number(block).full().await?;
        let output_block = output_block.ok_or(EngineClientError::NoL2BlockFound(block))?;
        // Cloning the l2 block below is cheaper than sending a network request to get the
        // l2 block info. Querying the `L2BlockInfo` from the client ends up
        // fetching the full l2 block again.
        let consensus_block = output_block.clone().into_consensus();
        let output_block_info =
            L2BlockInfo::from_block_and_genesis::<op_alloy_consensus::OpTxEnvelope>(
                &consensus_block.map_transactions(|tx| tx.inner.inner.into_inner()),
                &self.cfg.genesis,
            )?;

        let state_root = output_block.header.state_root;

        let message_passer_storage_root =
            if self.cfg.is_isthmus_active(output_block.header.timestamp) {
                output_block.header.withdrawals_root.ok_or(EngineClientError::NoWithdrawalsRoot)?
            } else {
                // Fetch the storage root for the L2 head block.
                self.engine
                    .get_proof(Predeploys::L2_TO_L1_MESSAGE_PASSER, Default::default())
                    .block_id(block.into())
                    .await?
                    .storage_hash
            };

        let output_response_v0 = OutputRoot::from_parts(
            state_root,
            message_passer_storage_root,
            output_block.header.hash,
        );
        Ok((output_block_info, output_response_v0))
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
    use alloy_primitives::B256;
    use alloy_rpc_types_eth::EIP1186AccountProofResponse;
    use serde_json::json;

    #[tokio::test]
    async fn query_handle_shares_l2_connection_and_computes_output_roots() {
        for isthmus in [false, true] {
            let mut block = Block::<Transaction>::default();
            block.header.inner.state_root = B256::repeat_byte(1);
            block.header.inner.withdrawals_root = Some(B256::repeat_byte(2));
            block.header.hash = block.header.inner.hash_slow();
            let mut config = RollupConfig::default();
            config.genesis.l2.hash = block.header.hash;
            config.hardforks.isthmus_time = isthmus.then_some(0);
            let config = Arc::new(config);
            let (client, l1, l2) = test_engine_client(config);
            let query = client.query_client();
            l1.expect_params(
                "eth_getBlockByNumber",
                json!(["latest", false]),
                Option::<Block>::None,
            );
            l2.expect_params(
                "eth_getBlockByNumber",
                json!(["latest", false]),
                Option::<Block<Transaction>>::None,
            );
            l2.expect_params("eth_getBlockByNumber", json!(["0x0", true]), &block);
            let storage_hash = if isthmus { B256::repeat_byte(2) } else { B256::repeat_byte(3) };
            if !isthmus {
                l2.expect_params(
                    "eth_getProof",
                    json!([Predeploys::L2_TO_L1_MESSAGE_PASSER, [], "0x0"]),
                    EIP1186AccountProofResponse { storage_hash, ..Default::default() },
                );
            }
            assert!(client.get_l1_block(BlockId::latest()).await.unwrap().is_none());
            assert!(client.get_l2_block(BlockId::latest()).await.unwrap().is_none());
            let (info, root) = query.output_at_block(BlockNumberOrTag::Number(0)).await.unwrap();
            assert_eq!(info.block_info.hash, block.header.hash);
            assert_eq!(
                root.hash(),
                OutputRoot::from_parts(block.header.state_root, storage_hash, block.header.hash)
                    .hash()
            );
            l1.assert_finished();
            l2.assert_finished();
        }
    }
}
