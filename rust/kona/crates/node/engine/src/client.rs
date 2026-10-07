//! An Engine API Client.

use crate::Metrics;
use alloy_eips::{BlockId, eip1898::BlockNumberOrTag};
use alloy_json_rpc::{RequestPacket, ResponsePacket};
use alloy_network::{Ethereum, Network};
use alloy_primitives::{Address, B256, Bytes, StorageKey};
use alloy_provider::{EthGetBlock, Provider, RootProvider, RpcWithBlock, ext::EngineApi};
use alloy_rpc_client::ClientBuilder;
use alloy_rpc_types_engine::{ExecutionPayloadV1, JwtSecret, PayloadStatus};
use alloy_rpc_types_eth::{Block, EIP1186AccountProofResponse};
use alloy_transport::{
    RpcError, TransportError, TransportErrorKind, TransportFut, TransportResult,
};
use alloy_transport_http::{
    AuthLayer, AuthService, Http, HyperClient,
    hyper_util::{
        client::legacy::{Client, connect::HttpConnector},
        rt::TokioExecutor,
    },
};
use async_trait::async_trait;
use http_body_util::Full;
use kona_genesis::RollupConfig;
use kona_protocol::FromBlockError;
use op_alloy_network::Optimism;
use op_alloy_provider::ext::engine::OpEngineApi;
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

/// An error that occurred in the [`EngineClient`].
#[derive(Error, Debug)]
pub enum EngineClientError {
    /// An RPC error occurred
    #[error("An RPC error occurred: {0}")]
    RpcError(#[from] RpcError<TransportErrorKind>),

    /// An error occurred while decoding the payload
    #[error("An error occurred while decoding the payload: {0}")]
    BlockInfoDecodeError(#[from] FromBlockError),
}
/// A Hyper HTTP client with a JWT authentication layer.
pub type HyperAuthClient<B = Full<Bytes>> = HyperClient<B, AuthService<Client<HttpConnector, B>>>;

/// Engine API client used to communicate with L1/L2 ELs.
/// `EngineClient` trait that is very coupled to its only implementation.
/// The main reason this exists is for mocking/unit testing.
#[async_trait]
pub trait EngineClient: OpEngineApi<Optimism, Http<HyperAuthClient>> + Send + Sync {
    /// Returns a reference to the inner [`RollupConfig`].
    fn cfg(&self) -> &RollupConfig;

    /// Fetches the L1 block with the provided `BlockId`.
    fn get_l1_block(&self, block: BlockId) -> EthGetBlock<<Ethereum as Network>::BlockResponse>;

    /// Fetches the L2 block with the provided `BlockId`.
    fn get_l2_block(&self, block: BlockId) -> EthGetBlock<<Optimism as Network>::BlockResponse>;

    /// Get the account and storage values of the specified account including the merkle proofs.
    /// This call can be used to verify that the data has not been tampered with.
    fn get_proof(
        &self,
        address: Address,
        keys: Vec<StorageKey>,
    ) -> RpcWithBlock<(Address, Vec<StorageKey>), EIP1186AccountProofResponse>;

    /// Sends the given payload to the execution layer client, as specified for the Paris fork.
    async fn new_payload_v1(&self, payload: ExecutionPayloadV1) -> TransportResult<PayloadStatus>;

    /// Fetches the [`Block<Transaction>`] for the given [`BlockNumberOrTag`].
    async fn l2_block_by_label(
        &self,
        numtag: BlockNumberOrTag,
    ) -> Result<Option<Block<Transaction>>, EngineClientError>;
}

/// Read-only subset of [`EngineClient`] used by the engine RPC actor.
///
/// Exposes only the methods required to serve [`crate::EngineQueries`] — fetching an L2 block by
/// label, and reading the L2-to-L1 message-passer storage hash. The engine RPC actor handles
/// queries only and must not have any way to call state-mutating Engine API methods; constraining
/// it to this trait prevents that at the type system level.
#[async_trait]
pub trait EngineRpcClient: Send + Sync {
    /// Fetches the [`Block<Transaction>`] for the given [`BlockNumberOrTag`].
    async fn l2_block_by_label(
        &self,
        numtag: BlockNumberOrTag,
    ) -> Result<Option<Block<Transaction>>, EngineClientError>;

    /// Returns the storage hash of `address` at the given block, used to compute the L2-to-L1
    /// message-passer storage root pre-Isthmus. This is a narrower projection of `get_proof`'s
    /// `storage_hash` field; callers needing the full account proof should not be using this
    /// trait.
    async fn get_storage_hash(
        &self,
        address: Address,
        block: BlockId,
    ) -> Result<B256, RpcError<TransportErrorKind>>;
}

#[async_trait]
impl<T: EngineClient + ?Sized> EngineRpcClient for T {
    async fn l2_block_by_label(
        &self,
        numtag: BlockNumberOrTag,
    ) -> Result<Option<Block<Transaction>>, EngineClientError> {
        EngineClient::l2_block_by_label(self, numtag).await
    }

    async fn get_storage_hash(
        &self,
        address: Address,
        block: BlockId,
    ) -> Result<B256, RpcError<TransportErrorKind>> {
        Ok(self.get_proof(address, Default::default()).block_id(block).await?.storage_hash)
    }
}

/// An Engine API client that provides authenticated HTTP communication with an execution layer.
///
/// The [`OpEngineClient`] handles JWT authentication and manages connections to both L1 and L2
/// execution layers. It automatically selects the appropriate Engine API version based on the
/// rollup configuration and block timestamps.
#[derive(Clone, Debug)]
pub struct OpEngineClient<L1Provider, L2Provider>
where
    L1Provider: Provider,
    L2Provider: Provider<Optimism>,
{
    /// The L2 engine provider for Engine API calls.
    engine: L2Provider,
    /// The L1 chain provider for reading L1 data.
    l1_provider: L1Provider,
    /// The [`RollupConfig`] for determining Engine API versions based on hardfork activations.
    cfg: Arc<RollupConfig>,
}

impl<L1Provider, L2Provider> OpEngineClient<L1Provider, L2Provider>
where
    L1Provider: Provider,
    L2Provider: Provider<Optimism>,
{
    /// Creates a new RPC client for the given address and JWT secret. Each request has a deadline.
    pub fn rpc_client<N: Network>(addr: Url, jwt: JwtSecret) -> RootProvider<N> {
        let hyper_client = Client::builder(TokioExecutor::new()).build_http::<Full<Bytes>>();
        let auth_layer = AuthLayer::new(jwt);
        let service = ServiceBuilder::new().layer(auth_layer).service(hyper_client);
        let layer_transport = HyperClient::with_service(service);
        let http_hyper = Http::with_client(layer_transport, addr);
        let client = ClientBuilder::default();
        // Time requests in full, including any that hit the deadline.
        let client = client.layer(RequestDurationLayer);
        let rpc_client =
            client.layer(MapFutureLayer::new(with_deadline)).transport(http_hyper, false);
        RootProvider::<N>::new(rpc_client)
    }
}

/// The builder for the [`OpEngineClient`].
#[derive(Debug, Clone)]
pub struct EngineClientBuilder {
    /// The L2 Engine API endpoint URL.
    pub l2: Url,
    /// The L2 JWT secret.
    pub l2_jwt: JwtSecret,
    /// The L1 RPC URL.
    pub l1_rpc: Url,
    /// The [`RollupConfig`] for determining Engine API versions based on hardfork activations.
    pub cfg: Arc<RollupConfig>,
}

impl EngineClientBuilder {
    /// Creates a new [`OpEngineClient`] with authenticated HTTP connections.
    ///
    /// Sets up a JWT-authenticated connection to the L2 Engine API endpoint
    /// along with an unauthenticated connection to the L1 chain. Each request on either
    /// connection has a deadline.
    pub fn build(self) -> OpEngineClient<RootProvider, RootProvider<Optimism>> {
        let engine = OpEngineClient::<RootProvider, RootProvider<Optimism>>::rpc_client::<Optimism>(
            self.l2,
            self.l2_jwt,
        );

        let l1_provider = RootProvider::new(
            ClientBuilder::default().layer(MapFutureLayer::new(with_deadline)).http(self.l1_rpc),
        );

        OpEngineClient { engine, l1_provider, cfg: self.cfg }
    }
}

#[async_trait]
impl<L1Provider, L2Provider> EngineClient for OpEngineClient<L1Provider, L2Provider>
where
    L1Provider: Provider,
    L2Provider: Provider<Optimism>,
{
    fn cfg(&self) -> &RollupConfig {
        self.cfg.as_ref()
    }

    fn get_l1_block(&self, block: BlockId) -> EthGetBlock<<Ethereum as Network>::BlockResponse> {
        self.l1_provider.get_block(block)
    }

    fn get_l2_block(&self, block: BlockId) -> EthGetBlock<<Optimism as Network>::BlockResponse> {
        self.engine.get_block(block)
    }

    fn get_proof(
        &self,
        address: Address,
        keys: Vec<StorageKey>,
    ) -> RpcWithBlock<(Address, Vec<StorageKey>), EIP1186AccountProofResponse> {
        self.engine.get_proof(address, keys)
    }

    async fn new_payload_v1(&self, payload: ExecutionPayloadV1) -> TransportResult<PayloadStatus> {
        self.engine.new_payload_v1(payload).await
    }

    async fn l2_block_by_label(
        &self,
        numtag: BlockNumberOrTag,
    ) -> Result<Option<Block<Transaction>>, EngineClientError> {
        Ok(self.engine.get_block_by_number(numtag).full().await?)
    }
}

/// Engine API calls on an [`OpEngineClient`] go to the L2 execution layer: with this, it gets
/// [`OpEngineApi`] from the implementation for every [`Provider`].
impl<L1Provider, L2Provider> Provider<Optimism> for OpEngineClient<L1Provider, L2Provider>
where
    L1Provider: Provider,
    L2Provider: Provider<Optimism>,
{
    fn root(&self) -> &RootProvider<Optimism> {
        self.engine.root()
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
            async move {
                EngineClient::l2_block_by_label(&*engine, BlockNumberOrTag::Latest).await.is_err()
            },
            &mut answered,
        )
        .await;
        fails_at_deadline(
            async move { EngineClient::get_l1_block(&*client, BlockId::latest()).await.is_err() },
            &mut answered,
        )
        .await;
    }
}
