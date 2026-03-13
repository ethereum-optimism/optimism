//! Follow Subcommand — lightweight node mode that follows a source L2 CL node.

use crate::flags::{
    DerivationDelegateArgs, GlobalArgs, L1ClientArgs, L2ClientArgs, RollupBoostFlags, RpcArgs,
};
use anyhow::{Result, bail};
use clap::Parser;
use kona_cli::LogConfig;
use kona_node_service::{EngineConfig, FollowNodeBuilder, L1ConfigBuilder, NodeMode};
use std::{path::PathBuf, sync::Arc, time::Duration};
use tracing::{error, info};

use super::node::NodeCommand;

/// Command-line interface for running a lightweight follow node.
///
/// The follow node polls a source L2 CL node for sync status and drives the local engine,
/// bypassing full derivation and P2P networking. This is useful for nodes that want to follow
/// an existing CL without the overhead of running their own derivation pipeline or P2P stack.
#[derive(Parser, Debug, Clone, Default)]
#[command(about = "Runs a lightweight follow node that tracks a source L2 CL")]
pub struct FollowCommand {
    /// L1 RPC CLI arguments.
    #[clap(flatten)]
    pub l1_rpc_args: L1ClientArgs,

    /// L2 engine CLI arguments.
    #[clap(flatten)]
    pub l2_client_args: L2ClientArgs,

    /// Source L2 CL to follow (required).
    #[clap(flatten)]
    pub derivation_delegate_args: DerivationDelegateArgs,

    /// RPC CLI arguments (optional).
    #[command(flatten)]
    pub rpc_flags: RpcArgs,

    /// Path to a custom L2 rollup configuration file
    /// (overrides the default rollup configuration from the registry)
    #[arg(long, visible_alias = "rollup-cfg", env = "KONA_NODE_ROLLUP_CONFIG")]
    pub l2_config_file: Option<PathBuf>,

    /// Path to a custom L1 rollup configuration file
    /// (overrides the default rollup configuration from the registry)
    #[arg(long, visible_alias = "rollup-l1-cfg", env = "KONA_NODE_L1_CHAIN_CONFIG")]
    pub l1_config_file: Option<PathBuf>,
}

impl FollowCommand {
    /// Initializes the logging system based on global arguments.
    pub fn init_logs(&self, args: &GlobalArgs) -> anyhow::Result<()> {
        LogConfig::new(args.log_args.clone()).init_tracing_subscriber(None)?;
        Ok(())
    }

    /// Run the Follow subcommand.
    pub async fn run(self, args: &GlobalArgs) -> Result<()> {
        let cfg = NodeCommand::get_l2_config_from(&self.l2_config_file, args)?;

        info!(
            target: "follow_node",
            chain_id = cfg.l2_chain_id.id(),
            "Starting follow node services"
        );

        let l1_config = L1ConfigBuilder {
            chain_config: NodeCommand::get_l1_config_from(&self.l1_config_file, cfg.l1_chain_id)?,
            trust_rpc: self.l1_rpc_args.l1_trust_rpc,
            beacon: self.l1_rpc_args.l1_beacon.clone(),
            rpc_url: self.l1_rpc_args.l1_eth_rpc.clone(),
            slot_duration_override: self.l1_rpc_args.l1_slot_duration_override,
        };

        let Some(derivation_delegate_config) = self.derivation_delegate_args.config() else {
            bail!("Follow mode requires --l2.follow.source to specify the source L2 CL node");
        };

        let jwt_secret = NodeCommand::validate_jwt_with(&self.l2_client_args).await?;

        let engine_config = EngineConfig {
            config: Arc::new(cfg.clone()),
            builder_url: self.l2_client_args.l2_engine_rpc.clone(),
            builder_jwt_secret: jwt_secret,
            builder_timeout: Duration::from_millis(self.l2_client_args.l2_engine_timeout),
            l2_url: self.l2_client_args.l2_engine_rpc.clone(),
            l2_jwt_secret: jwt_secret,
            l2_timeout: Duration::from_millis(self.l2_client_args.l2_engine_timeout),
            l1_url: self.l1_rpc_args.l1_eth_rpc.clone(),
            mode: NodeMode::Validator,
            rollup_boost: RollupBoostFlags::default().as_rollup_boost_args(),
        };

        let rpc_config = self.rpc_flags.into();

        FollowNodeBuilder::new(
            cfg,
            l1_config,
            engine_config,
            rpc_config,
            derivation_delegate_config,
        )
        .build()
        .start()
        .await
        .map_err(|e| {
            error!(target: "follow_node", "Failed to start follow node service: {e}");
            anyhow::anyhow!("{e}")
        })?;

        Ok(())
    }
}
