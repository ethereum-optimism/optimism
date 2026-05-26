//! Automatic builder playground configuration.
//!
//! This module is used mostly for testing purposes. It allows op-rbuilder to
//! automatically configure itself to run against a running op-builder playground.
//!
//! To setup the playground, checkout this repository:
//!
//!   https://github.com/flashbots/builder-playground
//!
//! Then run the following command:
//!
//!   go run main.go cook opstack --external-builder http://host.docker.internal:4444
//!
//! Wait until the playground is up and running, then run the following command to build
//! op-rbuilder with flashblocks support:
//!
//!   cargo build --bin op-rbuilder -p op-rbuilder
//!
//! then run the following command to start op-rbuilder against the playground:
//!
//!   target/debug/op-rbuilder node --builder.playground
//!
//! This will automatically try to detect the playground configuration and apply
//! it to the op-rbuilder startup settings.
//!
//! Optionally you can specify the `--builder.playground` flag with a different
//! directory to use. This is useful for testing against different playground
//! configurations.

use alloy_primitives::hex;
use clap::{CommandFactory, parser::ValueSource};
use core::{
    net::{IpAddr, Ipv4Addr, SocketAddr},
    ops::Range,
    time::Duration,
};
use eyre::{Result, eyre};
use reth_cli::chainspec::ChainSpecParser;
use reth_network_peers::TrustedPeer;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_cli::{chainspec::OpChainSpecParser, commands::Commands};
use secp256k1::SecretKey;
use serde_json::Value;
use std::{
    fs::read_to_string,
    path::{Path, PathBuf},
    sync::Arc,
};
use url::{Host, Url};

use super::Cli;

pub(super) struct PlaygroundOptions {
    /// Sets node.chain in NodeCommand
    pub chain: Arc<OpChainSpec>,

    /// Sets node.rpc.http_port in NodeCommand
    pub http_port: u16,

    /// Sets node.rpc.auth_addr in NodeCommand
    pub authrpc_addr: IpAddr,

    /// Sets node.rpc.authrpc_port in NodeCommand
    pub authrpc_port: u16,

    /// Sets node.rpc.authrpc_jwtsecret in NodeCommand
    pub authrpc_jwtsecret: PathBuf,

    /// Sets node.network.port in NodeCommand
    pub port: u16,

    /// Sets the node.network.trusted_peers in NodeCommand
    pub trusted_peer: TrustedPeer,

    /// Sets node.ext.flashblock_block_time in NodeCommand
    pub chain_block_time: Duration,
}

impl PlaygroundOptions {
    /// Creates a new `PlaygroundOptions` instance with the specified genesis path.
    pub(super) fn new(path: &Path) -> Result<Self> {
        if !path.exists() {
            return Err(eyre!(
                "Playground data directory {} does not exist",
                path.display()
            ));
        }

        let chain = OpChainSpecParser::parse(&existing_path(path, "l2-genesis.json")?)?;

        let authrpc_addr = Ipv4Addr::UNSPECIFIED.into();
        let http_port = pick_preferred_port(2222, 3000..9999);
        let authrpc_jwtsecret = existing_path(path, "jwtsecret")?.into();
        let port = pick_preferred_port(30333, 30000..65535);
        let chain_block_time = extract_chain_block_time(path)?;
        let authrpc_port = extract_authrpc_port(path)?;
        let trusted_peer = TrustedPeer::from_secret_key(
            Host::Ipv4(Ipv4Addr::LOCALHOST),
            extract_trusted_peer_port(path)?,
            &extract_deterministic_p2p_key(path)?,
        );

        Ok(Self {
            chain,
            http_port,
            authrpc_addr,
            authrpc_port,
            authrpc_jwtsecret,
            port,
            trusted_peer,
            chain_block_time,
        })
    }

    pub(super) fn apply(self, cli: Cli) -> Cli {
        let mut cli = cli;
        let Commands::Node(ref mut node) = cli.command else {
            // playground defaults are only relevant if running the node commands.
            return cli;
        };

        if !node.network.trusted_peers.contains(&self.trusted_peer) {
            node.network.trusted_peers.push(self.trusted_peer);
        }

        // populate the command line arguments only if they were never set by the user
        // either via the command line or an environment variable. Otherwise, don't
        // override the user provided values.
        let matches = Cli::command().get_matches();
        let matches = matches
            .subcommand_matches("node")
            .expect("validated that we are in the node command");

        if matches.value_source("chain").is_default() {
            node.chain = self.chain;
        }

        if matches.value_source("http").is_default() {
            node.rpc.http = true;
        }

        if matches.value_source("http_port").is_default() {
            node.rpc.http_port = self.http_port;
        }

        if matches.value_source("port").is_default() {
            node.network.port = self.port;
        }

        if matches.value_source("auth_addr").is_default() {
            node.rpc.auth_addr = self.authrpc_addr;
        }

        if matches.value_source("auth_port").is_default() {
            node.rpc.auth_port = self.authrpc_port;
        }

        if matches.value_source("auth_jwtsecret").is_default() {
            node.rpc.auth_jwtsecret = Some(self.authrpc_jwtsecret);
        }

        if matches.value_source("disable_discovery").is_default() {
            node.network.discovery.disable_discovery = true;
        }

        if matches.value_source("chain_block_time").is_default() {
            node.ext.chain_block_time = self.chain_block_time.as_millis() as u64;
        }

        cli
    }
}

fn existing_path(base: &Path, relative: &str) -> Result<String> {
    let path = base.join(relative);
    if path.exists() {
        Ok(path.to_string_lossy().to_string())
    } else {
        Err(eyre::eyre!(
            "Expected file {relative} is not present in playground directory {}",
            base.display()
        ))
    }
}

fn pick_random_port(range: Range<u16>) -> u16 {
    use rand::Rng;
    let mut rng = rand::rng();

    loop {
        // Generate a random port number between 30000 and 65535
        let port = rng.random_range(range.clone());

        // Check if the port is already in use
        let socket = SocketAddr::new(IpAddr::V4(Ipv4Addr::LOCALHOST), port);
        if std::net::TcpListener::bind(socket).is_ok() {
            return port;
        }
    }
}

fn pick_preferred_port(preferred: u16, fallback_range: Range<u16>) -> u16 {
    if !is_port_free(preferred) {
        return pick_random_port(fallback_range);
    }

    preferred
}

fn is_port_free(port: u16) -> bool {
    let socket = SocketAddr::new(IpAddr::V4(Ipv4Addr::LOCALHOST), port);
    std::net::TcpListener::bind(socket).is_ok()
}

fn extract_chain_block_time(basepath: &Path) -> Result<Duration> {
    Ok(Duration::from_secs(
        serde_json::from_str::<Value>(&read_to_string(existing_path(basepath, "rollup.json")?)?)?
            .get("block_time")
            .and_then(|v| v.as_u64())
            .ok_or_else(|| eyre::eyre!("Missing chain_block_time in rollup.json"))?,
    ))
}

fn extract_deterministic_p2p_key(basepath: &Path) -> Result<SecretKey> {
    let key = read_to_string(existing_path(basepath, "enode-key-1.txt")?)?;
    Ok(SecretKey::from_slice(
        &hex::decode(key).map_err(|e| eyre!("Invalid hex key: {e}"))?,
    )?)
}

fn read_docker_compose(basepath: &Path) -> Result<serde_yaml::Value> {
    // this happens only once on statup so it's fine to read the file multiple times
    let docker_compose = read_to_string(existing_path(basepath, "docker-compose.yaml")?)?;
    serde_yaml::from_str(&docker_compose).map_err(|e| eyre!("Invalid docker-compose file: {e}"))
}

fn extract_service_command_flag(basepath: &Path, service: &str, flag: &str) -> Result<String> {
    let docker_compose = read_docker_compose(basepath)?;
    let args = docker_compose["services"][service]["command"]
        .as_sequence()
        .ok_or(eyre!(
            "docker-compose.yaml is missing command line arguments for {service}"
        ))?
        .iter()
        .map(|s| {
            s.as_str().ok_or_else(|| {
                eyre!("docker-compose.yaml service command line argument is not a string")
            })
        })
        .collect::<Result<Vec<_>>>()?;

    let index = args
        .iter()
        .position(|arg| *arg == flag)
        .ok_or_else(|| eyre!("docker_compose: {flag} not found on {service} service"))?;

    let value = args
        .get(index + 1)
        .ok_or_else(|| eyre!("docker_compose: {flag} value not found"))?;

    Ok(value.to_string())
}

fn extract_authrpc_port(basepath: &Path) -> Result<u16> {
    let builder_url = extract_service_command_flag(basepath, "rollup-boost", "--builder-url")?;
    let url = Url::parse(&builder_url).map_err(|e| eyre!("Invalid builder-url: {e}"))?;
    url.port().ok_or_else(|| eyre!("missing builder-url port"))
}

fn extract_trusted_peer_port(basepath: &Path) -> Result<u16> {
    let docker_compose = read_docker_compose(basepath)?;

    // first we need to find the internal port of the op-geth service from the docker-compose.yaml
    // command line arguments used to start the op-geth service

    let Some(opgeth_args) = docker_compose["services"]["op-geth"]["command"][1].as_str() else {
        return Err(eyre!(
            "docker-compose.yaml is missing command line arguments for op-geth"
        ));
    };

    let opgeth_args = opgeth_args.split_whitespace().collect::<Vec<_>>();
    let port_param_position = opgeth_args
        .iter()
        .position(|arg| *arg == "--port")
        .ok_or_else(|| eyre!("docker_compose: --port param not found on op-geth service"))?;

    let port_value = opgeth_args
        .get(port_param_position + 1)
        .ok_or_else(|| eyre!("docker_compose: --port value not found"))?;

    let port_value = port_value
        .parse::<u16>()
        .map_err(|e| eyre!("Invalid port value: {e}"))?;

    // now we need to find the external port of the op-geth service from the docker-compose.yaml
    // ports mapping used to start the op-geth service
    let Some(opgeth_ports) = docker_compose["services"]["op-geth"]["ports"].as_sequence() else {
        return Err(eyre!(
            "docker-compose.yaml is missing ports mapping for op-geth"
        ));
    };
    let ports_mapping = opgeth_ports
        .iter()
        .map(|s| {
            s.as_str().ok_or_else(|| {
                eyre!("docker-compose.yaml service ports mapping in op-geth is not a string")
            })
        })
        .collect::<Result<Vec<_>>>()?;

    // port mappings is in the format [..., "127.0.0.1:30304:30303", ...]
    // we need to find the mapping that contains the port value we found earlier
    // and extract the external port from it
    let port_mapping = ports_mapping
        .iter()
        .find(|mapping| mapping.contains(&format!(":{port_value}")))
        .ok_or_else(|| {
            eyre!("docker_compose: external port mapping not found for {port_value} for op-geth")
        })?;

    // extract the external port from the mapping
    let port_mapping = port_mapping
        .split(':')
        .nth(1)
        .ok_or_else(|| eyre!("docker_compose: external port mapping for op-geth is not valid"))?;

    port_mapping
        .parse::<u16>()
        .map_err(|e| eyre!("Invalid external port mapping value for op-geth: {e}"))
}

trait IsDefaultSource {
    fn is_default(&self) -> bool;
}

impl IsDefaultSource for Option<ValueSource> {
    fn is_default(&self) -> bool {
        matches!(self, Some(ValueSource::DefaultValue)) || self.is_none()
    }
}
