use std::{
    borrow::Cow,
    collections::HashMap,
    fs::File,
    io::BufReader,
    path::PathBuf,
    sync::LazyLock,
    time::{SystemTime, UNIX_EPOCH},
};

use http::Uri;
use serde_json::Value;
use testcontainers::{
    ContainerAsync, CopyToContainer, Image,
    core::{ContainerPort, WaitFor},
};

use crate::common::TEST_DATA;

const NAME: &str = "ghcr.io/paradigmxyz/op-reth";
const TAG: &str = "v1.3.4";

pub const AUTH_RPC_PORT: u16 = 8551;
pub const P2P_PORT: u16 = 30303;

// Genesis should only be created once and reused for
// each instance
static GENESIS: LazyLock<String> = LazyLock::new(|| {
    let file = File::open(PathBuf::from(format!("{}/genesis.json", *TEST_DATA))).unwrap();
    let reader = BufReader::new(file);
    let mut genesis: Value = serde_json::from_reader(reader).unwrap();

    // Update the timestamp field
    let timestamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs();

    if let Some(config) = genesis.as_object_mut() {
        // Assuming timestamp is at the root level - adjust path as needed
        config["timestamp"] = Value::String(format!("0x{:x}", timestamp));
    }

    serde_json::to_string_pretty(&genesis).unwrap()
});

#[derive(Debug, Clone)]
pub struct OpRethConfig {
    jwt_secret: PathBuf,
    p2p_secret: Option<PathBuf>,
    pub trusted_peers: Vec<String>,
    pub color: String,
    pub ipcdisable: bool,
    pub env_vars: HashMap<String, String>,
}

impl Default for OpRethConfig {
    fn default() -> Self {
        Self {
            jwt_secret: PathBuf::from(format!("{}/jwt_secret.hex", *TEST_DATA)),
            p2p_secret: None,
            trusted_peers: vec![],
            color: "never".to_string(),
            ipcdisable: true,
            env_vars: Default::default(),
        }
    }
}

impl OpRethConfig {
    pub fn set_trusted_peers(mut self, trusted_peers: Vec<String>) -> Self {
        self.trusted_peers = trusted_peers;
        self
    }

    pub fn set_jwt_secret(mut self, jwt_secret: PathBuf) -> Self {
        self.jwt_secret = jwt_secret;
        self
    }

    pub fn set_p2p_secret(mut self, p2p_secret: Option<PathBuf>) -> Self {
        self.p2p_secret = p2p_secret;
        self
    }

    pub fn build(self) -> eyre::Result<OpRethImage> {
        let mut copy_to_sources = vec![
            CopyToContainer::new(
                std::fs::read_to_string(&self.jwt_secret)?.into_bytes(),
                "/jwt_secret.hex".to_string(),
            ),
            CopyToContainer::new(GENESIS.clone().into_bytes(), "/genesis.json".to_string()),
        ];

        if let Some(p2p_secret) = &self.p2p_secret {
            let p2p_string = std::fs::read_to_string(p2p_secret)
                .unwrap()
                .replace("\n", "");
            copy_to_sources.push(CopyToContainer::new(
                p2p_string.into_bytes(),
                "/p2p_secret.hex".to_string(),
            ));
        }

        let expose_ports = vec![];

        Ok(OpRethImage {
            config: self,
            copy_to_sources,
            expose_ports,
        })
    }
}

impl OpRethImage {
    pub fn config(&self) -> &OpRethConfig {
        &self.config
    }
}

#[derive(Debug, Clone)]
pub struct OpRethImage {
    config: OpRethConfig,
    copy_to_sources: Vec<CopyToContainer>,
    expose_ports: Vec<ContainerPort>,
}

impl Image for OpRethImage {
    fn name(&self) -> &str {
        NAME
    }

    fn tag(&self) -> &str {
        TAG
    }

    fn ready_conditions(&self) -> Vec<WaitFor> {
        vec![WaitFor::message_on_stdout("Starting consensus")]
    }

    fn env_vars(
        &self,
    ) -> impl IntoIterator<Item = (impl Into<Cow<'_, str>>, impl Into<Cow<'_, str>>)> {
        &self.config.env_vars
    }

    fn copy_to_sources(&self) -> impl IntoIterator<Item = &CopyToContainer> {
        self.copy_to_sources.iter()
    }

    fn cmd(&self) -> impl IntoIterator<Item = impl Into<std::borrow::Cow<'_, str>>> {
        let mut cmd = vec![
            "node".to_string(),
            "--port=30303".to_string(),
            "--addr=0.0.0.0".to_string(),
            "--http".to_string(),
            "--http.addr=0.0.0.0".to_string(),
            "--authrpc.port=8551".to_string(),
            "--authrpc.addr=0.0.0.0".to_string(),
            "--authrpc.jwtsecret=/jwt_secret.hex".to_string(),
            "--chain=/genesis.json".to_string(),
            "--log.stdout.filter=trace".to_string(),
            "-vvvvv".to_string(),
            "--disable-discovery".to_string(),
            "--color".to_string(),
            self.config.color.clone(),
        ];
        if self.config.p2p_secret.is_some() {
            cmd.push("--p2p-secret-key=/p2p_secret.hex".to_string());
        }
        if !self.config.trusted_peers.is_empty() {
            println!("Trusted peers: {:?}", self.config.trusted_peers);
            cmd.extend([
                "--trusted-peers".to_string(),
                self.config.trusted_peers.join(","),
            ]);
        }
        if self.config.ipcdisable {
            cmd.push("--ipcdisable".to_string());
        }
        cmd
    }

    fn expose_ports(&self) -> &[ContainerPort] {
        &self.expose_ports
    }
}

pub trait OpRethMehods {
    async fn auth_rpc(&self) -> eyre::Result<Uri>;
    async fn auth_rpc_port(&self) -> eyre::Result<u16>;
}

impl OpRethMehods for ContainerAsync<OpRethImage> {
    async fn auth_rpc_port(&self) -> eyre::Result<u16> {
        Ok(self.get_host_port_ipv4(AUTH_RPC_PORT).await?)
    }

    async fn auth_rpc(&self) -> eyre::Result<Uri> {
        Ok(format!(
            "http://{}:{}",
            self.get_host().await?,
            self.get_host_port_ipv4(AUTH_RPC_PORT).await?
        )
        .parse()?)
    }
}
