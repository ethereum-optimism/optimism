//! Net Subcommand

use crate::flags::{GlobalArgs, P2PArgs, RpcArgs};
use clap::Parser;
use futures::future::OptionFuture;
use jsonrpsee::{RpcModule, server::Server};
use kona_cli::LogConfig;
use kona_gossip::P2pRpcRequest;
use kona_node_service::{
    NetworkActor, NetworkBuilder, NetworkContext, NetworkInboundData, NodeActor,
};
use kona_registry::scr_rollup_config_by_alloy_ident;
use kona_rpc::{OpP2PApiServer, P2pRpc, RpcBuilder};
use tokio_util::sync::CancellationToken;
use tracing::{info, warn};
use url::Url;

/// The `net` Subcommand
///
/// The `net` subcommand is used to run the networking stack for the `kona-node`.
///
/// # Usage
///
/// ```sh
/// kona-node net [FLAGS] [OPTIONS]
/// ```
#[derive(Parser, Default, PartialEq, Debug, Clone)]
#[command(about = "Runs the networking stack for the kona-node.")]
pub struct NetCommand {
    /// URL of the L1 execution client RPC API.
    /// This is used to load the unsafe block signer at startup.
    /// Without this, the rollup config unsafe block signer will be used which may be outdated.
    #[arg(long, visible_alias = "l1", env = "L1_ETH_RPC")]
    pub l1_eth_rpc: Option<Url>,
    /// P2P CLI Flags
    #[command(flatten)]
    pub p2p: P2PArgs,
    /// RPC CLI Flags
    #[command(flatten)]
    pub rpc: RpcArgs,
}

impl NetCommand {
    /// Initializes the logging system based on global arguments.
    pub fn init_logs(&self, args: &GlobalArgs) -> anyhow::Result<()> {
        // Filter out discovery warnings since they're very very noisy.
        let filter = tracing_subscriber::EnvFilter::from_default_env()
            .add_directive("discv5=error".parse()?)
            .add_directive("bootstore=debug".parse()?);

        // Initialize the telemetry stack.
        LogConfig::new(args.log_args.clone()).init_tracing_subscriber(Some(filter))?;
        Ok(())
    }

    /// Run the Net subcommand.
    pub async fn run(self, args: &GlobalArgs) -> anyhow::Result<()> {
        let signer = args.genesis_signer()?;
        info!(target: "net", "Genesis block signer: {:?}", signer);

        let rpc_config = Option::<RpcBuilder>::from(self.rpc);

        // Get the rollup config from the args
        let rollup_config = scr_rollup_config_by_alloy_ident(&args.l2_chain_id)
            .ok_or(anyhow::anyhow!("Rollup config not found for chain id: {}", args.l2_chain_id))?;

        // Start the Network Stack
        self.p2p.check_ports()?;
        let p2p_config = self.p2p.config(rollup_config, args, self.l1_eth_rpc).await?;

        let (NetworkInboundData { p2p_rpc: rpc, .. }, network) =
            NetworkActor::new(NetworkBuilder::from(p2p_config));

        let (blocks, mut blocks_rx) = tokio::sync::mpsc::channel(1024);
        network.start(NetworkContext { blocks, cancellation: CancellationToken::new() }).await?;

        info!(target: "net", "Network started, receiving blocks.");

        // On an interval, use the rpc tx to request stats about the p2p network.
        let mut interval = tokio::time::interval(tokio::time::Duration::from_secs(2));

        let handle = if let Some(config) = rpc_config {
            info!(target: "net", socket = ?config.socket, "Starting RPC server");

            // Setup the RPC server with the P2P RPC Module
            let mut launcher = RpcModule::new(());
            launcher.merge(P2pRpc::new(rpc.clone()).into_rpc())?;

            let server = Server::builder().build(config.socket).await?;
            Some(server.start(launcher))
        } else {
            info!(target: "net", "RPC server disabled");
            None
        };

        loop {
            tokio::select! {
                Some(payload) = blocks_rx.recv() => {
                    info!(target: "net", "Received unsafe payload: {:?}", payload.execution_payload.block_hash());
                }
                _ = interval.tick(), if !rpc.is_closed() => {
                    let (otx, mut orx) = tokio::sync::oneshot::channel();
                    if let Err(e) = rpc.send(P2pRpcRequest::PeerCount(otx)).await {
                        warn!(target: "net", "Failed to send network rpc request: {:?}", e);
                        continue;
                    }
                    tokio::time::timeout(tokio::time::Duration::from_secs(5), async move {
                        loop {
                            match orx.try_recv() {
                                Ok((d, g)) => {
                                    let d = d.unwrap_or_default();
                                    info!(target: "net", "Peer counts: Discovery={} | Swarm={}", d, g);
                                    break;
                                }
                                Err(tokio::sync::oneshot::error::TryRecvError::Empty) => {
                                    /* Keep trying to receive */
                                }
                                Err(tokio::sync::oneshot::error::TryRecvError::Closed) => {
                                    break;
                                }
                            }
                        }
                    }).await.unwrap();
                }
                _ = OptionFuture::from(handle.clone().map(|h| h.stopped())) => {
                    warn!(target: "net", "RPC server stopped");
                    return Ok(());
                }
            }
        }
    }
}
