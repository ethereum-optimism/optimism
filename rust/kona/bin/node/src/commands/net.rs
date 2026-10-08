//! Net Subcommand

use crate::flags::{GlobalArgs, P2PArgs, RpcArgs};
use clap::Parser;
use futures::future::OptionFuture;
use jsonrpsee::{RpcModule, core::async_trait, server::Server};
use kona_cli::LogConfig;
use kona_node_service::{
    EngineClientResult, NetworkActor, NetworkBuilder, NetworkEngineClient, NodeActor,
};
use kona_registry::scr_rollup_config_by_alloy_ident;
use kona_rpc::{OpP2PApiServer, P2pRpc, RpcBuilder};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use tokio::sync::mpsc;
use tracing::{error, info, warn};
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
#[derive(Parser, Default, PartialEq, Eq, Debug, Clone)]
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
        let rollup_config =
            scr_rollup_config_by_alloy_ident(&args.l2_chain_id).ok_or_else(|| {
                anyhow::anyhow!("Rollup config not found for chain id: {}", args.l2_chain_id)
            })?;

        // Start the Network Stack
        self.p2p.check_ports()?;
        let p2p_config = self.p2p.config(rollup_config, args, self.l1_eth_rpc).await?;

        let (block_tx, mut block_rx) = mpsc::channel(1024);
        let (gossip_command_tx, gossip_command_rx) = mpsc::channel(1024);
        let (admin_rpc_tx, admin_rpc_rx) = mpsc::channel(1024);
        let (gossip_payload_tx, gossip_payload_rx) = mpsc::channel(256);
        // admin_rpc_tx and gossip_payload_tx are not used by this single-purpose binary — they
        // exist solely to satisfy NetworkActor::new and are held to keep the channels open.
        let _unused_senders = (admin_rpc_tx, gossip_payload_tx);

        let handler = NetworkBuilder::from(p2p_config).build()?.start().await?;

        let discovery = handler.discovery.clone();
        let mut network = NetworkActor::new(
            ForwardingNetworkEngineClient { block_tx },
            handler,
            gossip_command_rx,
            admin_rpc_rx,
            gossip_payload_rx,
        );

        let rpc = P2pRpc::new(network.gossip_query_handle(), discovery, gossip_command_tx.clone());

        // Spawn the actor; the loop below polls the p2p RPC interface on an interval.
        tokio::spawn(async move {
            loop {
                if let Err(e) = network.step().await {
                    error!(target: "net", "Network actor error: {e:?}");
                    return;
                }
            }
        });

        info!(target: "net", "Network started, receiving blocks.");

        // On an interval, read gossip state and query discovery for peer counts.
        let mut interval = tokio::time::interval(tokio::time::Duration::from_secs(2));

        let handle = if let Some(config) = rpc_config {
            info!(target: "net", socket = ?config.socket, "Starting RPC server");

            // Setup the RPC server with the P2P RPC Module
            let mut launcher = RpcModule::new(());
            launcher.merge(rpc.clone().into_rpc())?;

            let server = Server::builder().build(config.socket).await?;
            Some(server.start(launcher))
        } else {
            info!(target: "net", "RPC server disabled");
            None
        };

        loop {
            tokio::select! {
                Some(payload) = block_rx.recv() => {
                    info!(target: "net", "Received unsafe payload: {:?}", payload.block_hash());
                }
                _ = interval.tick(), if !gossip_command_tx.is_closed() => {
                    match tokio::time::timeout(tokio::time::Duration::from_secs(5), rpc.opp2p_peer_count()).await? {
                        Ok(count) => {
                            info!(target: "net", "Peer counts: Discovery={} | Swarm={}", count.connected_discovery.unwrap_or_default(), count.connected_gossip);
                        }
                        Err(e) => warn!(target: "net", "Failed to query peer counts: {e:?}"),
                    }
                }
                _ = OptionFuture::from(handle.clone().map(|h| h.stopped())) => {
                    warn!(target: "net", "RPC server stopped");
                    return Ok(());
                }
            }
        }
    }
}

#[derive(Debug)]
struct ForwardingNetworkEngineClient {
    block_tx: mpsc::Sender<OpExecutionPayloadEnvelope>,
}

#[async_trait]
impl NetworkEngineClient for ForwardingNetworkEngineClient {
    async fn send_unsafe_block(&self, block: OpExecutionPayloadEnvelope) -> EngineClientResult<()> {
        let _ = self
            .block_tx
            .send(block)
            .await
            .inspect_err(|e| error!(target: "net", "Failed to send block: {:?}", e));

        Ok(())
    }
}
