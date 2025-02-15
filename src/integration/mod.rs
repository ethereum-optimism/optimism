use crate::debug_api::DebugClient;
use crate::server::EngineApiClient;
use crate::server::PayloadCreator;
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
use alloy_rpc_types_engine::{
    ExecutionPayloadV3, ForkchoiceState, ForkchoiceUpdated, PayloadAttributes, PayloadId,
    PayloadStatus, PayloadStatusEnum,
};
use jsonrpsee::http_client::{transport::HttpBackend, HttpClient};
use jsonrpsee::proc_macros::rpc;
use lazy_static::lazy_static;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelopeV3;
use reth_optimism_payload_builder::OpPayloadAttributes;
use reth_rpc_layer::{AuthClientLayer, AuthClientService, JwtSecret};
use std::collections::{HashMap, HashSet};
use std::future::Future;
use std::path::{Path, PathBuf};
use std::str::FromStr;
use std::sync::Mutex;
use std::{
    fs::{File, OpenOptions},
    io,
    io::prelude::*,
    process::{Child, Command},
    time::{Duration, SystemTime},
};
use thiserror::Error;
use time::{format_description, OffsetDateTime};

/// Default JWT token for testing purposes
pub const DEFAULT_JWT_TOKEN: &str =
    "688f5d737bad920bdfb2fc2f488d6b6209eebda1dae949a8de91398d932c517a";

mod integration_test;
mod service_rb;
mod service_reth;

#[derive(Debug, Error)]
pub enum IntegrationError {
    #[error("Failed to spawn process")]
    SpawnError,
    #[error("Binary not found")]
    BinaryNotFound,
    #[error("Failed to setup integration framework")]
    SetupError,
    #[error("Log error")]
    LogError,
    #[error("Service already running")]
    ServiceAlreadyRunning,
    #[error("Service stopped")]
    ServiceStopped,
    #[error(transparent)]
    AddrParseError(#[from] std::net::AddrParseError),
}

#[derive(Debug, Clone)]
pub enum Arg {
    Port { name: String, preferred: u16 },
    Dir { name: String },
    Value(String),
}

impl From<String> for Arg {
    fn from(s: String) -> Self {
        Arg::Value(s)
    }
}

impl From<&str> for Arg {
    fn from(s: &str) -> Self {
        Arg::Value(s.to_string())
    }
}

impl From<PathBuf> for Arg {
    fn from(path: PathBuf) -> Self {
        Arg::Value(
            path.to_str()
                .expect("Failed to convert path to string")
                .to_string(),
        )
    }
}

impl From<&Path> for Arg {
    fn from(path: &Path) -> Self {
        Arg::Value(
            path.to_str()
                .expect("Failed to convert path to string")
                .to_string(),
        )
    }
}

impl From<&String> for Arg {
    fn from(s: &String) -> Self {
        Arg::Value(s.clone())
    }
}

impl From<&PathBuf> for Arg {
    fn from(path: &PathBuf) -> Self {
        Arg::Value(
            path.to_str()
                .expect("Failed to convert path to string")
                .to_string(),
        )
    }
}

pub struct ServiceCommand {
    program: String,
    args: Vec<Arg>,
}

impl ServiceCommand {
    pub fn new(program: impl Into<String>) -> Self {
        Self {
            program: program.into(),
            args: Vec::new(),
        }
    }

    pub fn arg(mut self, arg: impl Into<Arg>) -> Self {
        self.args.push(arg.into());
        self
    }
}

pub trait Service {
    fn command(&self) -> ServiceCommand;
    fn ready(
        &self,
        service_instance: &mut ServiceInstance,
    ) -> impl Future<Output = Result<(), IntegrationError>> + Send;
}

pub struct ServiceInstance {
    process: Option<Child>,
    pub log_path: PathBuf,
    allocated_ports: HashMap<String, u16>,
}

lazy_static! {
    static ref GLOBAL_ALLOCATED_PORTS: Mutex<HashSet<u16>> = Mutex::new(HashSet::new());
}

pub struct IntegrationFramework {
    test_dir: PathBuf,
    logs_dir: PathBuf,
    services: HashMap<String, ServiceInstance>,
}

impl ServiceInstance {
    pub fn new(name: String, logs_dir: PathBuf, allocated_ports: HashMap<String, u16>) -> Self {
        let log_path = logs_dir.join(format!("{}.log", name));
        Self {
            process: None,
            log_path,
            allocated_ports,
        }
    }

    pub fn start(&mut self, command: Command) -> Result<(), IntegrationError> {
        if self.process.is_some() {
            return Err(IntegrationError::ServiceAlreadyRunning);
        }

        let log = open_log_file(&self.log_path)?;
        let stdout = log.try_clone().map_err(|_| IntegrationError::LogError)?;
        let stderr = log.try_clone().map_err(|_| IntegrationError::LogError)?;

        let mut cmd = command;
        cmd.stdout(stdout).stderr(stderr);

        let child = match cmd.spawn() {
            Ok(child) => Ok(child),
            Err(e) => match e.kind() {
                io::ErrorKind::NotFound => Err(IntegrationError::BinaryNotFound),
                _ => Err(IntegrationError::SpawnError),
            },
        }?;

        self.process = Some(child);
        Ok(())
    }

    pub fn stop(&mut self) -> Result<(), IntegrationError> {
        if let Some(mut process) = self.process.take() {
            process.kill().map_err(|_| IntegrationError::SpawnError)?;
        }
        Ok(())
    }

    /// Start a service using its configuration and wait for it to be ready
    pub async fn start_with_config<T: Service>(
        &mut self,
        config: &T,
        system_command: Command,
    ) -> Result<(), IntegrationError> {
        self.start(system_command)?;
        config.ready(self).await?;
        Ok(())
    }

    pub fn get_port(&self, name: &str) -> u16 {
        *self.allocated_ports.get(name).unwrap_or_else(|| {
            panic!("Port for {} not found", name);
        })
    }

    pub fn get_endpoint(&self, name: &str) -> String {
        format!("http://localhost:{}", self.get_port(name))
    }

    pub fn wait_for_log(
        &mut self,
        pattern: &str,
        timeout: Duration,
    ) -> Result<(), IntegrationError> {
        let start = std::time::Instant::now();

        loop {
            // Check if process has stopped
            if let Some(ref mut process) = self.process {
                match process.try_wait() {
                    Ok(None) => {}
                    Ok(Some(_status)) => {
                        // Process has exited
                        return Err(IntegrationError::ServiceStopped);
                    }
                    Err(_) => {
                        return Err(IntegrationError::ServiceStopped);
                    }
                }
            }

            if start.elapsed() > timeout {
                return Err(IntegrationError::SpawnError);
            }

            let mut file = File::open(&self.log_path).map_err(|_| IntegrationError::LogError)?;
            let mut contents = String::new();
            file.read_to_string(&mut contents)
                .map_err(|_| IntegrationError::LogError)?;

            if contents.contains(pattern) {
                return Ok(());
            }

            std::thread::sleep(Duration::from_millis(100));
        }
    }
}

impl IntegrationFramework {
    pub fn new(test_name: &str) -> Result<Self, IntegrationError> {
        let dt: OffsetDateTime = SystemTime::now().into();
        let format = format_description::parse("[year]_[month]_[day]_[hour]_[minute]_[second]")
            .map_err(|_| IntegrationError::SetupError)?;

        let timestamp = dt
            .format(&format)
            .map_err(|_| IntegrationError::SetupError)?;

        let test_name = format!("{}_{}", timestamp, test_name);

        let mut test_dir = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
        test_dir.push("./integration_logs");
        test_dir.push(test_name);

        // Create logs subdirectory
        let logs_dir = test_dir.join("logs");
        std::fs::create_dir_all(&logs_dir).map_err(|_| IntegrationError::SetupError)?;

        Ok(Self {
            test_dir,
            logs_dir,
            services: HashMap::new(),
        })
    }

    fn build_command(
        &mut self,
        service_name: &str,
        cmd: ServiceCommand,
    ) -> Result<(HashMap<String, u16>, Command), IntegrationError> {
        let mut command = Command::new(cmd.program);
        let mut allocated_ports = HashMap::new();

        for arg in cmd.args {
            match arg {
                Arg::Port { name, preferred } => {
                    let port = self.find_available_port(preferred)?;
                    allocated_ports.insert(name, port);
                    command.arg(port.to_string());
                }
                Arg::Dir { name } => {
                    let dir_path = self.test_dir.join(service_name).join(name);
                    std::fs::create_dir_all(&dir_path).map_err(|_| IntegrationError::SetupError)?;
                    command.arg(dir_path.to_str().expect("Failed to convert path to string"));
                }
                Arg::Value(value) => {
                    command.arg(value);
                }
            }
        }

        Ok((allocated_ports, command))
    }

    fn find_available_port(&self, start: u16) -> Result<u16, IntegrationError> {
        let mut global_ports = GLOBAL_ALLOCATED_PORTS
            .lock()
            .expect("Failed to acquire lock");

        (start..start + 100)
            .find(|&port| {
                if global_ports.contains(&port) {
                    return false;
                }
                if std::net::TcpListener::bind(("127.0.0.1", port)).is_ok() {
                    global_ports.insert(port);
                    return true;
                }
                false
            })
            .ok_or(IntegrationError::SetupError)
    }

    pub async fn start<T: Service>(
        &mut self,
        name: &str,
        config: &T,
    ) -> Result<&mut ServiceInstance, IntegrationError> {
        let (allocated_ports, command) = self.build_command(name, config.command())?;

        // Store the service instance in the framework
        let service =
            ServiceInstance::new(name.to_string(), self.logs_dir.clone(), allocated_ports);
        self.services.insert(name.to_string(), service);
        let service = self.services.get_mut(name).unwrap();

        service.start_with_config(config, command).await?;
        Ok(service)
    }

    /// Writes content to a file in the test directory and returns its absolute path
    pub fn write_file(
        &self,
        name: &str,
        content: impl AsRef<[u8]>,
    ) -> Result<PathBuf, IntegrationError> {
        let file_path = self.test_dir.join(name);
        if let Some(parent) = file_path.parent() {
            std::fs::create_dir_all(parent).map_err(|_| IntegrationError::SetupError)?;
        }
        std::fs::write(&file_path, content).map_err(|_| IntegrationError::SetupError)?;
        Ok(file_path)
    }
}

fn open_log_file(path: &PathBuf) -> Result<File, IntegrationError> {
    let prefix = path.parent().unwrap();
    std::fs::create_dir_all(prefix).map_err(|_| IntegrationError::LogError)?;

    OpenOptions::new()
        .append(true)
        .create(true)
        .open(path)
        .map_err(|_| IntegrationError::LogError)
}

impl Drop for IntegrationFramework {
    fn drop(&mut self) {
        // Stop all services first
        for service in &mut self.services {
            let _ = service.1.stop();
        }

        // Release allocated ports from global registry
        let mut global_ports = GLOBAL_ALLOCATED_PORTS
            .lock()
            .expect("Failed to acquire lock");
        for service in &self.services {
            for port in service.1.allocated_ports.values() {
                global_ports.remove(port);
            }
        }
    }
}

pub struct EngineApi {
    pub engine_api_client: HttpClient<AuthClientService<HttpBackend>>,
}

impl EngineApi {
    pub fn new(url: &str, secret: &str) -> Result<Self, Box<dyn std::error::Error>> {
        let secret_layer = AuthClientLayer::new(JwtSecret::from_str(secret)?);
        let middleware = tower::ServiceBuilder::default().layer(secret_layer);
        let client = jsonrpsee::http_client::HttpClientBuilder::default()
            .set_http_middleware(middleware)
            .build(&url)
            .expect("Failed to create http client");

        Ok(Self {
            engine_api_client: client,
        })
    }

    pub async fn get_payload_v3(
        &self,
        payload_id: PayloadId,
    ) -> eyre::Result<OpExecutionPayloadEnvelopeV3> {
        Ok(EngineApiClient::get_payload_v3(&self.engine_api_client, payload_id).await?)
    }

    pub async fn new_payload(
        &self,
        payload: ExecutionPayloadV3,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
    ) -> eyre::Result<PayloadStatus> {
        Ok(EngineApiClient::new_payload_v3(
            &self.engine_api_client,
            payload,
            versioned_hashes,
            parent_beacon_block_root,
        )
        .await?)
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
    _framework: IntegrationFramework, // Keep framework alive to maintain service ownership
}

impl RollupBoostTestHarness {
    pub async fn new(test_name: &str) -> Result<Self, IntegrationError> {
        let mut framework = IntegrationFramework::new(test_name)?;

        let jwt_path = framework.write_file("jwt.hex", DEFAULT_JWT_TOKEN)?;

        let genesis_path =
            framework.write_file("genesis.json", include_str!("testdata/genesis.json"))?;

        // Start L2 Reth instance
        let l2_reth_config = service_reth::RethConfig::new()
            .jwt_secret_path(jwt_path.clone())
            .chain_config_path(genesis_path.clone());

        let l2_service = {
            let service = framework.start("l2-reth", &l2_reth_config).await?;
            service.get_endpoint("authrpc")
        };

        // Start Builder Reth instance
        let builder_reth_config = service_reth::RethConfig::new()
            .jwt_secret_path(jwt_path.clone())
            .chain_config_path(genesis_path);
        let builder_service = {
            let service = framework
                .start("builder-reth", &builder_reth_config)
                .await?;
            service.get_endpoint("authrpc")
        };

        // Start Rollup-boost instance
        let rb_config = service_rb::RollupBoostConfig::new()
            .jwt_path(jwt_path)
            .l2_url(l2_service)
            .builder_url(builder_service);

        let _ = framework.start("rollup-boost", &rb_config).await?;

        Ok(Self {
            _framework: framework,
        })
    }

    pub async fn get_block_generator(&self) -> SimpleBlockGenerator {
        let rb_service = self._framework.services.get("rollup-boost").unwrap();
        let validator = BlockBuilderCreatorValidator::new(rb_service.log_path.clone());

        let engine_api = EngineApi::new(&rb_service.get_endpoint("rpc"), DEFAULT_JWT_TOKEN)
            .map_err(|_| IntegrationError::SetupError)
            .unwrap();

        let mut block_creator = SimpleBlockGenerator::new(validator, engine_api);
        block_creator.init().await.unwrap();
        block_creator
    }

    pub async fn get_client(&self) -> DebugClient {
        let rb_service = self._framework.services.get("rollup-boost").unwrap();
        let endpoint = rb_service.get_endpoint("debug");

        DebugClient::new(&endpoint).unwrap()
    }
}

/// A simple system that continuously generates empty blocks using the engine API
pub struct SimpleBlockGenerator {
    validator: BlockBuilderCreatorValidator,
    engine_api: EngineApi,
    latest_hash: B256,
    timestamp: u64,
}

impl SimpleBlockGenerator {
    pub fn new(validator: BlockBuilderCreatorValidator, engine_api: EngineApi) -> Self {
        Self {
            validator,
            engine_api,
            latest_hash: B256::ZERO, // temporary value
            timestamp: 0,            // temporary value
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
    ) -> eyre::Result<(B256, PayloadCreator)> {
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
                        target_blobs_per_block: None,
                        max_blobs_per_block: None,
                    },
                    transactions: None,
                    no_tx_pool: Some(empty_blocks),
                    gas_limit: Some(10000000000),
                    eip_1559_params: None,
                }),
            )
            .await?;

        let payload_id = result.payload_id.expect("missing payload id");
        let payload = self.engine_api.get_payload_v3(payload_id).await?;

        // Submit the new payload to the node
        let validation_status = self
            .engine_api
            .new_payload(payload.execution_payload.clone(), vec![], B256::ZERO)
            .await?;

        if validation_status.status != PayloadStatusEnum::Valid {
            return Err(eyre::eyre!("Invalid payload status"));
        }

        let new_block_hash = payload
            .execution_payload
            .payload_inner
            .payload_inner
            .block_hash;

        // Update the chain's head
        self.engine_api
            .update_forkchoice(self.latest_hash, new_block_hash, None)
            .await?;

        // Update internal state
        self.latest_hash = new_block_hash;
        self.timestamp = payload
            .execution_payload
            .payload_inner
            .payload_inner
            .timestamp;

        // Check who built the block in the rollup-boost logs
        let block_creator = self
            .validator
            .get_block_creator(new_block_hash)?
            .expect("block creator not found");

        Ok((new_block_hash, block_creator))
    }
}

pub struct BlockBuilderCreatorValidator {
    log_path: PathBuf,
}

impl BlockBuilderCreatorValidator {
    pub fn new(log_path: PathBuf) -> Self {
        Self { log_path }
    }
}

impl BlockBuilderCreatorValidator {
    pub fn get_block_creator(&self, block_hash: B256) -> eyre::Result<Option<PayloadCreator>> {
        let mut file = File::open(&self.log_path).map_err(|_| IntegrationError::LogError)?;
        let mut contents = String::new();
        file.read_to_string(&mut contents)
            .map_err(|_| IntegrationError::LogError)?;

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
                        "builder" => return Ok(Some(PayloadCreator::Builder)),
                        "l2" => return Ok(Some(PayloadCreator::L2)),
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
