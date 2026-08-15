//! Net Subcommand

use crate::flags::{GlobalArgs, P2PArgs, RpcArgs};
use clap::Parser;
use jsonrpsee::{RpcModule, server::Server};
use kona_cli::LogConfig;
use kona_gossip::P2pRpcRequest;
use kona_node_service_v2::{NetworkBuilder, network::NetworkService};
use kona_registry::scr_rollup_config_by_alloy_ident;
use kona_rpc::{OpP2PApiServer, P2pRpc, RpcBuilder};
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
use tracing::{info, warn};
use url::Url;

/// Runs the standalone Kona networking stack.
#[derive(Parser, Default, PartialEq, Eq, Debug, Clone)]
#[command(about = "Runs the networking stack for the kona-node.")]
pub struct NetCommand {
    /// URL of the L1 execution client RPC API used to load the unsafe block signer.
    #[arg(long, visible_alias = "l1", env = "L1_ETH_RPC")]
    pub l1_eth_rpc: Option<Url>,
    /// P2P CLI flags.
    #[command(flatten)]
    pub p2p: P2PArgs,
    /// RPC CLI flags.
    #[command(flatten)]
    pub rpc: RpcArgs,
}

impl NetCommand {
    /// Initializes logging.
    pub fn init_logs(&self, args: &GlobalArgs) -> anyhow::Result<()> {
        let filter = tracing_subscriber::EnvFilter::from_default_env()
            .add_directive("discv5=error".parse()?)
            .add_directive("bootstore=debug".parse()?);
        LogConfig::new(args.log_args.clone()).init_tracing_subscriber(Some(filter))?;
        Ok(())
    }

    /// Runs the standalone network service.
    pub async fn run(self, args: &GlobalArgs) -> anyhow::Result<()> {
        info!(target: "net", signer = ?args.genesis_signer()?, "Genesis block signer");
        let rpc_config = Option::<RpcBuilder>::from(self.rpc);
        let rollup_config =
            scr_rollup_config_by_alloy_ident(&args.l2_chain_id).ok_or_else(|| {
                anyhow::anyhow!("Rollup config not found for chain id: {}", args.l2_chain_id)
            })?;

        self.p2p.check_ports()?;
        let p2p_config = self.p2p.config(rollup_config, args, self.l1_eth_rpc).await?;
        let handler = NetworkBuilder::from(p2p_config).build()?.start().await?;

        let (payload_tx, mut payload_rx) = mpsc::channel(1024);
        let (signer_tx, signer_rx) = mpsc::channel(16);
        let (network, client) = NetworkService::new(handler, signer_rx, payload_tx);
        // The standalone command has no L1 service, so retain the update sender for service
        // lifetime while continuing to use the configured genesis signer.
        let _signer_keepalive = signer_tx;
        let rpc = client.p2p_sender();
        let shutdown = CancellationToken::new();
        let mut network_task = tokio::spawn(network.run(shutdown.clone()));

        let rpc_handle = if let Some(config) = rpc_config {
            let mut module = RpcModule::new(());
            module.merge(P2pRpc::new(rpc.clone()).into_rpc())?;
            let server = Server::builder().build(config.socket).await?;
            Some(server.start(module))
        } else {
            None
        };

        let mut interval = tokio::time::interval(tokio::time::Duration::from_secs(2));
        loop {
            tokio::select! {
                payload = payload_rx.recv() => {
                    let Some(payload) = payload else {
                        anyhow::bail!("network payload stream closed");
                    };
                    info!(target: "net", hash = %payload.block_hash(), "Received unsafe payload");
                }
                _ = interval.tick() => {
                    let (response, result) = tokio::sync::oneshot::channel();
                    rpc.send(P2pRpcRequest::PeerCount(response)).await?;
                    match tokio::time::timeout(tokio::time::Duration::from_secs(5), result).await {
                        Ok(Ok((discovery, gossip))) => {
                            info!(target: "net", discovery = discovery.unwrap_or_default(), gossip, "Peer counts");
                        }
                        Ok(Err(_)) => warn!(target: "net", "Peer-count response dropped"),
                        Err(_) => warn!(target: "net", "Peer-count request timed out"),
                    }
                }
                result = &mut network_task => {
                    result.map_err(|error| anyhow::anyhow!("network service panicked: {error}"))??;
                    anyhow::bail!("network service stopped unexpectedly");
                }
                _ = wait_for_rpc_stop(rpc_handle.clone()), if rpc_handle.is_some() => {
                    shutdown.cancel();
                    network_task.await??;
                    return Ok(());
                }
            }
        }
    }
}

async fn wait_for_rpc_stop(handle: Option<jsonrpsee::server::ServerHandle>) {
    if let Some(handle) = handle {
        handle.stopped().await;
    } else {
        std::future::pending::<()>().await;
    }
}
