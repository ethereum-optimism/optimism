#![allow(dead_code)]
use alloy_eips::Encodable2718;
use alloy_primitives::{B256, Bytes, TxKind, U256, address, hex};
use alloy_rpc_types_engine::{ExecutionPayload, JwtSecret};
use alloy_rpc_types_engine::{
    ForkchoiceState, ForkchoiceUpdated, PayloadAttributes, PayloadId, PayloadStatus,
    PayloadStatusEnum,
};
use alloy_rpc_types_eth::BlockNumberOrTag;
use bytes::BytesMut;
use futures::FutureExt;
use futures::future::BoxFuture;
use jsonrpsee::http_client::{HttpClient, transport::HttpBackend};
use jsonrpsee::proc_macros::rpc;
use op_alloy_consensus::TxDeposit;
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use parking_lot::Mutex;
use proxy::{ProxyHandler, start_proxy_server};
use rollup_boost::DebugClient;
use rollup_boost::{AuthClientLayer, AuthClientService};
use rollup_boost::{EngineApiClient, OpExecutionPayloadEnvelope, Version};
use rollup_boost::{NewPayload, PayloadSource};
use services::op_reth::{AUTH_RPC_PORT, OpRethConfig, OpRethImage, OpRethMehods, P2P_PORT};
use services::rollup_boost::{RollupBoost, RollupBoostConfig};
use std::collections::HashSet;
use std::net::TcpListener;
use std::path::PathBuf;
use std::str::FromStr;
use std::sync::{Arc, LazyLock};
use std::time::SystemTime;
use testcontainers::core::ContainerPort;
use testcontainers::core::client::docker_client_instance;
use testcontainers::core::logs::LogFrame;
use testcontainers::core::logs::consumer::LogConsumer;
use testcontainers::runners::AsyncRunner;
use testcontainers::{ContainerAsync, ImageExt};
use time::{OffsetDateTime, format_description};
use tokio::io::AsyncWriteExt as _;

/// Default JWT token for testing purposes
pub const JWT_SECRET: &str = "688f5d737bad920bdfb2fc2f488d6b6209eebda1dae949a8de91398d932c517a";
pub const L2_P2P_ENODE: &str = "3479db4d9217fb5d7a8ed4d61ac36e120b05d36c2eefb795dc42ff2e971f251a2315f5649ea1833271e020b9adc98d5db9973c7ed92d6b2f1f2223088c3d852f";
pub static TEST_DATA: LazyLock<String> =
    LazyLock::new(|| format!("{}/tests/common/test_data", env!("CARGO_MANIFEST_DIR")));

pub mod proxy;
pub mod services;

pub struct LoggingConsumer {
    target: String,
    log_file: tokio::sync::Mutex<tokio::fs::File>,
}

impl LogConsumer for LoggingConsumer {
    fn accept<'a>(&'a self, record: &'a LogFrame) -> BoxFuture<'a, ()> {
        async move {
            match record {
                testcontainers::core::logs::LogFrame::StdOut(bytes) => {
                    self.log_file.lock().await.write_all(bytes).await.unwrap();
                }
                testcontainers::core::logs::LogFrame::StdErr(bytes) => {
                    self.log_file.lock().await.write_all(bytes).await.unwrap();
                }
            }
        }
        .boxed()
    }
}

pub struct EngineApi {
    pub engine_api_client: HttpClient<AuthClientService<HttpBackend>>,
}

// TODO: Use client/rpc.rs instead
impl EngineApi {
    pub fn new(url: &str, secret: &str) -> eyre::Result<Self> {
        let secret_layer = AuthClientLayer::new(JwtSecret::from_str(secret)?);
        let middleware = tower::ServiceBuilder::default().layer(secret_layer);
        let client = jsonrpsee::http_client::HttpClientBuilder::default()
            .set_http_middleware(middleware)
            .build(url)
            .expect("Failed to create http client");

        Ok(Self {
            engine_api_client: client,
        })
    }

    pub async fn get_payload(
        &self,
        version: Version,
        payload_id: PayloadId,
    ) -> eyre::Result<OpExecutionPayloadEnvelope> {
        match version {
            Version::V3 => Ok(OpExecutionPayloadEnvelope::V3(
                EngineApiClient::get_payload_v3(&self.engine_api_client, payload_id).await?,
            )),
            Version::V4 => Ok(OpExecutionPayloadEnvelope::V4(
                EngineApiClient::get_payload_v4(&self.engine_api_client, payload_id).await?,
            )),
        }
    }

    pub async fn new_payload(&self, payload: NewPayload) -> eyre::Result<PayloadStatus> {
        match payload {
            NewPayload::V3(new_payload) => Ok(EngineApiClient::new_payload_v3(
                &self.engine_api_client,
                new_payload.payload,
                new_payload.versioned_hashes,
                new_payload.parent_beacon_block_root,
            )
            .await?),
            NewPayload::V4(new_payload) => Ok(EngineApiClient::new_payload_v4(
                &self.engine_api_client,
                new_payload.payload,
                new_payload.versioned_hashes,
                new_payload.parent_beacon_block_root,
                new_payload.execution_requests,
            )
            .await?),
        }
    }

    pub async fn update_forkchoice(
        &self,
        current_head: B256,
        new_head: B256,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> eyre::Result<ForkchoiceUpdated> {
        Ok(EngineApiClient::fork_choice_updated_v3(
            &self.engine_api_client,
            ForkchoiceState {
                head_block_hash: new_head,
                safe_block_hash: current_head,
                finalized_block_hash: current_head,
            },
            payload_attributes,
        )
        .await?)
    }

    pub async fn latest(&self) -> eyre::Result<Option<alloy_rpc_types_eth::Block>> {
        Ok(BlockApiClient::get_block_by_number(
            &self.engine_api_client,
            BlockNumberOrTag::Latest,
            false,
        )
        .await?)
    }
}

#[rpc(client, namespace = "eth")]
pub trait BlockApi {
    #[method(name = "getBlockByNumber")]
    async fn get_block_by_number(
        &self,
        block_number: BlockNumberOrTag,
        include_txs: bool,
    ) -> RpcResult<Option<alloy_rpc_types_eth::Block>>;
}

/// Test flavor that sets up one Rollup-boost instance connected to two Reth nodes
pub struct RollupBoostTestHarness {
    pub l2: ContainerAsync<OpRethImage>,
    pub builder: ContainerAsync<OpRethImage>,
    pub rollup_boost: RollupBoost,
}

pub struct RollupBoostTestHarnessBuilder {
    test_name: String,
    proxy_handler: Option<Arc<dyn ProxyHandler>>,
}

impl RollupBoostTestHarnessBuilder {
    pub fn new(test_name: &str) -> Self {
        Self {
            test_name: test_name.to_string(),
            proxy_handler: None,
        }
    }

    pub fn file_path(&self, service_name: &str) -> eyre::Result<PathBuf> {
        let dt: OffsetDateTime = SystemTime::now().into();
        let format = format_description::parse("[year]_[month]_[day]_[hour]_[minute]_[second]")?;
        let timestamp = dt.format(&format)?;

        let dir = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .join("integration_logs")
            .join(self.test_name.clone())
            .join(timestamp);
        std::fs::create_dir_all(&dir)?;

        let file_name = format!("{service_name}.log");
        Ok(dir.join(file_name))
    }

    pub async fn async_log_file(&self, service_name: &str) -> eyre::Result<tokio::fs::File> {
        let file_path = self.file_path(service_name)?;
        Ok(tokio::fs::OpenOptions::new()
            .append(true)
            .create(true)
            .open(file_path)
            .await?)
    }

    pub async fn log_consumer(&self, service_name: &str) -> eyre::Result<LoggingConsumer> {
        let file = self.async_log_file(service_name).await?;
        Ok(LoggingConsumer {
            target: service_name.to_string(),
            log_file: tokio::sync::Mutex::new(file),
        })
    }

    pub fn proxy_handler(mut self, proxy_handler: Arc<dyn ProxyHandler>) -> Self {
        self.proxy_handler = Some(proxy_handler);
        self
    }

    pub async fn build(self) -> eyre::Result<RollupBoostTestHarness> {
        let network = rand::random::<u16>().to_string();
        let l2_log_consumer = self.log_consumer("l2").await?;
        let builder_log_consumer = self.log_consumer("builder").await?;
        let rollup_boost_log_file_path = self.file_path("rollup_boost")?;

        let l2_p2p_port = get_available_port();
        let l2 = OpRethConfig::default()
            .set_p2p_secret(Some(PathBuf::from(format!(
                "{}/p2p_secret.hex",
                *TEST_DATA
            ))))
            .build()?
            .with_mapped_port(l2_p2p_port, ContainerPort::Tcp(P2P_PORT))
            .with_mapped_port(l2_p2p_port, ContainerPort::Udp(P2P_PORT))
            .with_mapped_port(get_available_port(), ContainerPort::Tcp(AUTH_RPC_PORT))
            .with_network(&network)
            .with_log_consumer(l2_log_consumer)
            .start()
            .await?;

        let client = docker_client_instance().await?;
        let res = client.inspect_container(l2.id(), None).await?;
        let name = res.name.unwrap()[1..].to_string(); // remove the leading '/'

        let l2_enode = format!("enode://{}@{}:{}", L2_P2P_ENODE, name, P2P_PORT);

        let builder_p2p_port = get_available_port();
        let builder = OpRethConfig::default()
            .set_trusted_peers(vec![l2_enode])
            .build()?
            .with_mapped_port(builder_p2p_port, ContainerPort::Tcp(P2P_PORT))
            .with_mapped_port(builder_p2p_port, ContainerPort::Udp(P2P_PORT))
            .with_mapped_port(get_available_port(), ContainerPort::Tcp(AUTH_RPC_PORT))
            .with_network(&network)
            .with_log_consumer(builder_log_consumer)
            .start()
            .await?;

        println!("l2 authrpc: {}", l2.auth_rpc().await?);
        println!("builder authrpc: {}", builder.auth_rpc().await?);

        // run a proxy in between the builder and the rollup-boost if the proxy_handler is set
        let mut builder_authrpc_port = builder.auth_rpc_port().await?;
        if let Some(proxy_handler) = self.proxy_handler {
            println!("starting proxy server");
            let proxy_port = get_available_port();
            start_proxy_server(proxy_handler, proxy_port, builder_authrpc_port).await?;
            builder_authrpc_port = proxy_port
        };
        let builder_url = format!("http://localhost:{}/", builder_authrpc_port);
        println!("proxy authrpc: {}", builder_url);

        // Start Rollup-boost instance
        let mut rollup_boost = RollupBoostConfig::default();
        rollup_boost.args.l2_client.l2_url = l2.auth_rpc().await?;
        rollup_boost.args.builder.builder_url = builder_url.try_into().unwrap();
        rollup_boost.args.log_file = Some(rollup_boost_log_file_path);
        let rollup_boost = rollup_boost.start().await;
        println!("rollup-boost authrpc: {}", rollup_boost.rpc_endpoint());
        println!("rollup-boost metrics: {}", rollup_boost.metrics_endpoint());

        Ok(RollupBoostTestHarness {
            l2,
            builder,
            rollup_boost,
        })
    }
}

impl RollupBoostTestHarness {
    pub async fn block_generator(&self) -> eyre::Result<SimpleBlockGenerator> {
        let validator =
            BlockBuilderCreatorValidator::new(self.rollup_boost.args().log_file.clone().unwrap());

        let engine_api = EngineApi::new(&self.rollup_boost.rpc_endpoint(), JWT_SECRET)?;

        let mut block_creator = SimpleBlockGenerator::new(validator, engine_api);
        block_creator.init().await?;
        Ok(block_creator)
    }

    pub async fn debug_client(&self) -> DebugClient {
        DebugClient::new(&self.rollup_boost.debug_endpoint()).unwrap()
    }
}

/// A simple system that continuously generates empty blocks using the engine API
pub struct SimpleBlockGenerator {
    validator: BlockBuilderCreatorValidator,
    engine_api: EngineApi,
    latest_hash: B256,
    timestamp: u64,
    version: Version,
}

impl SimpleBlockGenerator {
    pub fn new(validator: BlockBuilderCreatorValidator, engine_api: EngineApi) -> Self {
        Self {
            validator,
            engine_api,
            latest_hash: B256::ZERO, // temporary value
            timestamp: 0,            // temporary value
            version: Version::V3,
        }
    }

    /// Initialize the block generator by fetching the latest block
    pub async fn init(&mut self) -> eyre::Result<()> {
        let latest_block = self.engine_api.latest().await?.expect("block not found");
        self.latest_hash = latest_block.header.hash;
        self.timestamp = latest_block.header.timestamp;
        Ok(())
    }

    /// Generate a single new block and return its hash
    pub async fn generate_block(
        &mut self,
        empty_blocks: bool,
    ) -> eyre::Result<(B256, PayloadSource)> {
        let txns = match self.version {
            Version::V4 => {
                let tx = create_deposit_tx();
                Some(vec![tx])
            }
            _ => None,
        };

        // Submit forkchoice update with payload attributes for the next block
        let result = self
            .engine_api
            .update_forkchoice(
                self.latest_hash,
                self.latest_hash,
                Some(OpPayloadAttributes {
                    payload_attributes: PayloadAttributes {
                        withdrawals: Some(vec![]),
                        parent_beacon_block_root: Some(B256::ZERO),
                        timestamp: self.timestamp + 1000, // 1 second later
                        prev_randao: B256::ZERO,
                        suggested_fee_recipient: Default::default(),
                    },
                    transactions: txns,
                    no_tx_pool: Some(empty_blocks),
                    gas_limit: Some(10000000000),
                    eip_1559_params: None,
                }),
            )
            .await?;

        let payload_id = result.payload_id.expect("missing payload id");

        if !empty_blocks {
            tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
        }

        let payload = self
            .engine_api
            .get_payload(self.version, payload_id)
            .await?;

        // Submit the new payload to the node
        let validation_status = self
            .engine_api
            .new_payload(NewPayload::from(payload.clone()))
            .await?;

        if validation_status.status != PayloadStatusEnum::Valid {
            return Err(eyre::eyre!("Invalid payload status"));
        }

        let execution_payload = ExecutionPayload::from(payload);
        let new_block_hash = execution_payload.block_hash();

        // Update the chain's head
        self.engine_api
            .update_forkchoice(self.latest_hash, new_block_hash, None)
            .await?;

        // Update internal state
        self.latest_hash = new_block_hash;
        self.timestamp = execution_payload.timestamp();

        // Check who built the block in the rollup-boost logs
        let block_creator = self
            .validator
            .get_block_creator(new_block_hash)
            .await?
            .expect("block creator not found");

        Ok((new_block_hash, block_creator))
    }
}

pub struct BlockBuilderCreatorValidator {
    file: PathBuf,
}

impl BlockBuilderCreatorValidator {
    pub fn new(file: PathBuf) -> Self {
        Self { file }
    }
}

impl BlockBuilderCreatorValidator {
    pub async fn get_block_creator(&self, block_hash: B256) -> eyre::Result<Option<PayloadSource>> {
        let contents = std::fs::read_to_string(&self.file)?;

        let search_query = format!("returning block hash={:#x}", block_hash);

        // Find the log line containing the block hash
        for line in contents.lines() {
            if line.contains(&search_query) {
                // Extract the context=X part
                if let Some(context_start) = line.find("context=") {
                    let context = line[context_start..]
                        .split_whitespace()
                        .next()
                        .ok_or(eyre::eyre!("no context found"))?
                        .split('=')
                        .nth(1)
                        .ok_or(eyre::eyre!("no context found"))?;

                    match context {
                        "builder" => return Ok(Some(PayloadSource::Builder)),
                        "l2" => return Ok(Some(PayloadSource::L2)),
                        _ => panic!("Unknown context: {}", context),
                    }
                } else {
                    panic!("no context found");
                }
            }
        }

        Ok(None)
    }
}

fn create_deposit_tx() -> Bytes {
    const ISTHMUS_DATA: &[u8] = &hex!(
        "098999be00000558000c5fc500000000000000030000000067a9f765000000000000002900000000000000000000000000000000000000000000000000000000006a6d09000000000000000000000000000000000000000000000000000000000000000172fcc8e8886636bdbe96ba0e4baab67ea7e7811633f52b52e8cf7a5123213b6f000000000000000000000000d3f2c5afb2d76f5579f326b0cd7da5f5a4126c3500004e2000000000000001f4"
    );

    let deposit_tx = TxDeposit {
        source_hash: B256::default(),
        from: address!("DeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001"),
        to: TxKind::Call(address!("4200000000000000000000000000000000000015")),
        mint: None,
        value: U256::default(),
        gas_limit: 210000,
        is_system_transaction: true,
        input: ISTHMUS_DATA.into(),
    };

    let mut buffer_without_header = BytesMut::new();
    deposit_tx.encode_2718(&mut buffer_without_header);

    buffer_without_header.to_vec().into()
}

pub fn get_available_port() -> u16 {
    static CLAIMED_PORTS: LazyLock<Mutex<HashSet<u16>>> =
        LazyLock::new(|| Mutex::new(HashSet::new()));
    loop {
        let port: u16 = rand::random_range(1000..20000);
        if TcpListener::bind(("127.0.0.1", port)).is_ok() && CLAIMED_PORTS.lock().insert(port) {
            return port;
        }
    }
}
