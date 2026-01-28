//! An Engine API Client.

use crate::{Metrics, RollupBoostServerArgs, RollupBoostServerError};
use alloy_eips::{BlockId, eip1898::BlockNumberOrTag};
use alloy_network::{Ethereum, Network};
use alloy_primitives::{Address, B256, BlockHash, Bytes, StorageKey};
use alloy_provider::{EthGetBlock, Provider, RootProvider, RpcWithBlock, ext::EngineApi};
use alloy_rpc_client::RpcClient;
use alloy_rpc_types_engine::{
    ClientVersionV1, ExecutionPayloadBodiesV1, ExecutionPayloadEnvelopeV2, ExecutionPayloadInputV2,
    ExecutionPayloadV1, ExecutionPayloadV3, ForkchoiceState, ForkchoiceUpdated, JwtSecret,
    PayloadId, PayloadStatus,
};
use alloy_rpc_types_eth::{Block, EIP1186AccountProofResponse};
use alloy_transport::{RpcError, TransportErrorKind, TransportResult};
use alloy_transport_http::{
    AuthLayer, AuthService, Http, HyperClient,
    hyper_util::{
        client::legacy::{Client, connect::HttpConnector},
        rt::TokioExecutor,
    },
};
use async_trait::async_trait;
use http::uri::InvalidUri;
use http_body_util::Full;
use kona_genesis::RollupConfig;
use kona_protocol::{FromBlockError, L2BlockInfo};
use op_alloy_network::Optimism;
use op_alloy_provider::ext::engine::OpEngineApi;
use op_alloy_rpc_types::Transaction;
use op_alloy_rpc_types_engine::{
    OpExecutionPayloadEnvelopeV3, OpExecutionPayloadEnvelopeV4, OpExecutionPayloadV4,
    OpPayloadAttributes, ProtocolVersion,
};
use parking_lot::Mutex;
use rollup_boost::{
    EngineApiServer, Flashblocks, FlashblocksWebsocketConfig, Probes, RollupBoostServer,
    RpcClientError,
};
use rollup_boost_types::payload::PayloadSource;
use std::{
    future::Future,
    net::{AddrParseError, IpAddr, SocketAddr},
    str::FromStr,
    sync::Arc,
    time::{Duration, Instant},
};
use thiserror::Error;
use tower::ServiceBuilder;
use url::Url;

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

/// Engine API client used to communicate with L1/L2 ELs and optional rollup-boost.
/// EngineClient trait that is very coupled to its only implementation.
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

    /// Fetches the [L2BlockInfo] by [BlockNumberOrTag].
    async fn l2_block_info_by_label(
        &self,
        numtag: BlockNumberOrTag,
    ) -> Result<Option<L2BlockInfo>, EngineClientError>;
}

/// An Engine API client that provides authenticated HTTP communication with an execution layer.
///
/// The [`OpEngineClient`] handles JWT authentication and manages connections to both L1 and L2
/// execution layers. It automatically selects the appropriate Engine API version based on the
/// rollup configuration and block timestamps.
///
/// Engine API client used to communicate with L1/L2 ELs and optional rollup-boost.
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
    /// The rollup boost server
    pub rollup_boost: Arc<RollupBoostServer>,
}

impl<L1Provider, L2Provider> OpEngineClient<L1Provider, L2Provider>
where
    L1Provider: Provider,
    L2Provider: Provider<Optimism>,
{
    /// Creates a new RPC client for the given address and JWT secret.
    pub fn rpc_client<N: Network>(addr: Url, jwt: JwtSecret) -> RootProvider<N> {
        let hyper_client = Client::builder(TokioExecutor::new()).build_http::<Full<Bytes>>();
        let auth_layer = AuthLayer::new(jwt);
        let service = ServiceBuilder::new().layer(auth_layer).service(hyper_client);
        let layer_transport = HyperClient::with_service(service);
        let http_hyper = Http::with_client(layer_transport, addr);
        let rpc_client = RpcClient::new(http_hyper, false);
        RootProvider::<N>::new(rpc_client)
    }
}

/// The builder for the [`OpEngineClient`].
#[derive(Debug, Clone)]
pub struct EngineClientBuilder {
    /// The builder URL.
    pub builder: Url,
    /// The builder JWT secret.
    pub builder_jwt: JwtSecret,
    /// The builder timeout.
    pub builder_timeout: Duration,
    /// The L2 Engine API endpoint URL.
    pub l2: Url,
    /// The L2 JWT secret.
    pub l2_jwt: JwtSecret,
    /// The L2 timeout.
    pub l2_timeout: Duration,
    /// The L1 RPC URL.
    pub l1_rpc: Url,
    /// The [`RollupConfig`] for determining Engine API versions based on hardfork activations.
    pub cfg: Arc<RollupConfig>,
    /// The rollup boost arguments.
    pub rollup_boost: RollupBoostServerArgs,
}

/// An error that occurred in the [`EngineClientBuilder`].
#[derive(Error, Debug)]
pub enum EngineClientBuilderError {
    /// An error occurred while parsing the URL
    #[error("An error occurred while parsing the URL: {0}")]
    UrlParseError(#[from] InvalidUri),
    /// An error occurred while parsing the IP address
    #[error("An error occurred while parsing the IP address: {0}")]
    IpAddrParseError(#[from] AddrParseError),
    /// An error occurred while creating the RPC client
    #[error("An error occurred while creating the RPC client: {0}")]
    RpcClientError(#[from] RpcClientError),
    /// An error occurred while creating the Flashblocks service
    #[error("An error occurred while creating the Flashblocks service: {0}")]
    FlashblocksError(String),
}

impl EngineClientBuilder {
    /// Creates a new [`OpEngineClient`] with authenticated HTTP connections.
    ///
    /// Sets up JWT-authenticated connections to the Engine API endpoint through the rollup-boost
    /// server along with an unauthenticated connection to the L1 chain.
    ///
    /// # FIXME(@theochap, `<https://github.com/op-rs/kona/issues/3053>`, `<https://github.com/op-rs/kona/issues/3054>`):
    /// This method can be simplified/improved in a few ways:
    /// - Unify kona's and rollup-boost's RPC client creation
    /// - Removed the `dyn RollupBoostServerLike` type erasure.
    pub fn build(
        self,
    ) -> Result<OpEngineClient<RootProvider, RootProvider<Optimism>>, EngineClientBuilderError>
    {
        let probes = Arc::new(Probes::default());
        let l2_client = rollup_boost::RpcClient::new(
            http::Uri::from_str(self.l2.to_string().as_str())?,
            self.l2_jwt,
            self.l2_timeout.as_millis() as u64,
            PayloadSource::L2,
        )?;
        let builder_client = rollup_boost::RpcClient::new(
            http::Uri::from_str(self.builder.to_string().as_str())?,
            self.builder_jwt,
            self.builder_timeout.as_millis() as u64,
            PayloadSource::Builder,
        )?;

        let rollup_boost_server = match self.rollup_boost.flashblocks {
            Some(flashblocks) => {
                let inbound_url = flashblocks.flashblocks_builder_url;
                let outbound_addr = SocketAddr::new(
                    IpAddr::from_str(&flashblocks.flashblocks_host)?,
                    flashblocks.flashblocks_port,
                );

                let ws_config = flashblocks.flashblocks_ws_config;

                let builder_client = Arc::new(
                    Flashblocks::run(
                        builder_client,
                        inbound_url,
                        outbound_addr,
                        FlashblocksWebsocketConfig {
                            flashblock_builder_ws_initial_reconnect_ms: ws_config
                                .flashblock_builder_ws_initial_reconnect_ms,
                            flashblock_builder_ws_max_reconnect_ms: ws_config
                                .flashblock_builder_ws_max_reconnect_ms,
                            flashblock_builder_ws_connect_timeout_ms: ws_config
                                .flashblock_builder_ws_connect_timeout_ms,
                            flashblock_builder_ws_ping_interval_ms: ws_config
                                .flashblock_builder_ws_ping_interval_ms,
                            flashblock_builder_ws_pong_timeout_ms: ws_config
                                .flashblock_builder_ws_pong_timeout_ms,
                        },
                    )
                    .map_err(|e| EngineClientBuilderError::FlashblocksError(e.to_string()))?,
                );
                Arc::new(rollup_boost::RollupBoostServer::new(
                    l2_client,
                    builder_client,
                    Arc::new(Mutex::new(self.rollup_boost.initial_execution_mode)),
                    self.rollup_boost.block_selection_policy,
                    probes,
                    self.rollup_boost.external_state_root,
                    self.rollup_boost.ignore_unhealthy_builders,
                ))
            }
            None => Arc::new(rollup_boost::RollupBoostServer::new(
                l2_client,
                Arc::new(builder_client),
                Arc::new(Mutex::new(self.rollup_boost.initial_execution_mode)),
                self.rollup_boost.block_selection_policy,
                probes,
                self.rollup_boost.external_state_root,
                self.rollup_boost.ignore_unhealthy_builders,
            )),
        };

        // TODO(ethereum-optimism/optimism#18656): remove this client, upstream the remaining
        // EngineApiExt methods to the RollupBoostServer
        let engine = OpEngineClient::<RootProvider, RootProvider<Optimism>>::rpc_client::<Optimism>(
            self.l2,
            self.l2_jwt,
        );

        let l1_provider = RootProvider::new_http(self.l1_rpc);

        Ok(OpEngineClient { engine, l1_provider, cfg: self.cfg, rollup_boost: rollup_boost_server })
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

    async fn l2_block_info_by_label(
        &self,
        numtag: BlockNumberOrTag,
    ) -> Result<Option<L2BlockInfo>, EngineClientError> {
        let block = self.engine.get_block_by_number(numtag).full().await?;
        let Some(block) = block else {
            return Ok(None);
        };
        Ok(Some(L2BlockInfo::from_block_and_genesis(&block.into_consensus(), &self.cfg.genesis)?))
    }
}

#[async_trait::async_trait]
impl<L1Provider, L2Provider> OpEngineApi<Optimism, Http<HyperAuthClient>>
    for OpEngineClient<L1Provider, L2Provider>
where
    L1Provider: Provider,
    L2Provider: Provider<Optimism>,
{
    async fn new_payload_v2(
        &self,
        payload: ExecutionPayloadInputV2,
    ) -> TransportResult<PayloadStatus> {
        let call = <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::new_payload_v2(
            &self.engine,
            payload,
        );

        record_call_time(call, Metrics::NEW_PAYLOAD_METHOD).await
    }

    async fn new_payload_v3(
        &self,
        payload: ExecutionPayloadV3,
        parent_beacon_block_root: B256,
    ) -> TransportResult<PayloadStatus> {
        let call = self.rollup_boost.new_payload_v3(payload, vec![], parent_beacon_block_root);

        record_call_time(call, Metrics::NEW_PAYLOAD_METHOD)
            .await
            .map_err(|err| RollupBoostServerError::from(err).into())
    }

    async fn new_payload_v4(
        &self,
        payload: OpExecutionPayloadV4,
        parent_beacon_block_root: B256,
    ) -> TransportResult<PayloadStatus> {
        let call = self.rollup_boost.new_payload_v4(
            payload.clone(),
            vec![],
            parent_beacon_block_root,
            vec![],
        );

        record_call_time(call, Metrics::NEW_PAYLOAD_METHOD)
            .await
            .map_err(|err| RollupBoostServerError::from(err).into())
    }

    async fn fork_choice_updated_v2(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> TransportResult<ForkchoiceUpdated> {
        let call =
            <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::fork_choice_updated_v2(
                &self.engine,
                fork_choice_state,
                payload_attributes,
            );

        record_call_time(call, Metrics::FORKCHOICE_UPDATE_METHOD).await
    }

    async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> TransportResult<ForkchoiceUpdated> {
        let call = self.rollup_boost.fork_choice_updated_v3(fork_choice_state, payload_attributes);

        record_call_time(call, Metrics::FORKCHOICE_UPDATE_METHOD)
            .await
            .map_err(|err| RollupBoostServerError::from(err).into())
    }

    async fn get_payload_v2(
        &self,
        payload_id: PayloadId,
    ) -> TransportResult<ExecutionPayloadEnvelopeV2> {
        let call = <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::get_payload_v2(
            &self.engine,
            payload_id,
        );

        record_call_time(call, Metrics::GET_PAYLOAD_METHOD).await
    }

    async fn get_payload_v3(
        &self,
        payload_id: PayloadId,
    ) -> TransportResult<OpExecutionPayloadEnvelopeV3> {
        let call = self.rollup_boost.get_payload_v3(payload_id);

        record_call_time(call, Metrics::GET_PAYLOAD_METHOD)
            .await
            .map_err(|err| RollupBoostServerError::from(err).into())
    }

    async fn get_payload_v4(
        &self,
        payload_id: PayloadId,
    ) -> TransportResult<OpExecutionPayloadEnvelopeV4> {
        let call = self.rollup_boost.get_payload_v4(payload_id);

        record_call_time(call, Metrics::GET_PAYLOAD_METHOD)
            .await
            .map_err(|err| RollupBoostServerError::from(err).into())
    }

    async fn get_payload_bodies_by_hash_v1(
        &self,
        block_hashes: Vec<BlockHash>,
    ) -> TransportResult<ExecutionPayloadBodiesV1> {
        <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::get_payload_bodies_by_hash_v1(
            &self.engine,
            block_hashes,
        )
        .await
    }

    async fn get_payload_bodies_by_range_v1(
        &self,
        start: u64,
        count: u64,
    ) -> TransportResult<ExecutionPayloadBodiesV1> {
        <L2Provider as OpEngineApi<
            Optimism,
            Http<HyperAuthClient>,
        >>::get_payload_bodies_by_range_v1(&self.engine, start, count).await
    }

    async fn get_client_version_v1(
        &self,
        client_version: ClientVersionV1,
    ) -> TransportResult<Vec<ClientVersionV1>> {
        <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::get_client_version_v1(
            &self.engine,
            client_version,
        )
        .await
    }

    async fn signal_superchain_v1(
        &self,
        recommended: ProtocolVersion,
        required: ProtocolVersion,
    ) -> TransportResult<ProtocolVersion> {
        <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::signal_superchain_v1(
            &self.engine,
            recommended,
            required,
        )
        .await
    }

    async fn exchange_capabilities(
        &self,
        capabilities: Vec<String>,
    ) -> TransportResult<Vec<String>> {
        <L2Provider as OpEngineApi<Optimism, Http<HyperAuthClient>>>::exchange_capabilities(
            &self.engine,
            capabilities,
        )
        .await
    }
}

/// Wrapper to record the time taken for a call to the engine API and log the result as a metric.
async fn record_call_time<T, Err>(
    f: impl Future<Output = Result<T, Err>>,
    metric_label: &'static str,
) -> Result<T, Err> {
    // Await on the future and track its duration.
    let start = Instant::now();
    let result = f.await?;
    let duration = start.elapsed();

    // Record the call duration.
    kona_macros::record!(
        histogram,
        Metrics::ENGINE_METHOD_REQUEST_DURATION,
        "method",
        metric_label,
        duration.as_secs_f64()
    );
    Ok(result)
}
