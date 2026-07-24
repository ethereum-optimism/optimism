//! Proposer service binary for the super-root ZK dispute game.
//!
//! Derived from op-succinct's `fault-proof/bin/proposer.rs` (@ 13716c2c).

use std::sync::Arc;

use alloy_provider::ProviderBuilder;
use anyhow::Result;
use clap::Parser;
use kona_sp1_host_utils::{
    logger::setup_logger,
    metrics::{MetricsGauge, init_metrics},
};
use kona_sp1_proposer::{
    config::ProposerConfig,
    contract::{AnchorStateRegistry, DelayedWETH, DisputeGameFactory, ZKGameArgs},
    metrics::ProposerGauge,
    proposer::Proposer,
    signer::{Signer, SignerLock},
};

#[derive(Parser)]
struct Args {
    /// Path to the environment file to load configuration from.
    #[arg(long, default_value = ".env.proposer")]
    env_file: String,
}

#[tokio::main]
async fn main() -> Result<()> {
    let args = Args::parse();
    // Optional: absent env files are fine, the environment may be pre-set
    // (devstack passes every variable directly).
    let _ = dotenvy::from_filename(args.env_file);

    setup_logger();

    let config = ProposerConfig::from_env()?;
    let signer = SignerLock::new(Signer::from_env().await?);

    let l1_provider = ProviderBuilder::default().connect_http(config.l1_rpc.clone());
    let factory = DisputeGameFactory::new(config.factory_address, l1_provider.clone());

    // Resolve the anchor state registry and DelayedWETH from the registered
    // game args; a dedicated env var would only add a mismatch footgun.
    let game_args_bytes = factory.gameArgs(config.game_type).call().await?;
    let game_args = ZKGameArgs::decode(&game_args_bytes)?;
    let anchor_state_registry =
        AnchorStateRegistry::new(game_args.anchor_state_registry, l1_provider.clone());
    let weth = DelayedWETH::new(game_args.weth, l1_provider.clone());

    // Metrics: bind before the readiness log so the advertised address is live.
    let metrics_addr = (config.metrics_port != 0).then(|| {
        ProposerGauge::register_all();
        init_metrics(&config.metrics_port);
        ProposerGauge::init_all();
        format!("0.0.0.0:{}", config.metrics_port)
    });

    let proposer = Proposer::new(config, signer, anchor_state_registry, factory, weth).await?;

    // STARTUP LOG CONTRACT: devstack readiness matches this exact message,
    // and reads `metrics_addr` from the same entry when metrics are enabled.
    // Emitted before the chain-dependent init retry loop on purpose - a
    // supernode that is still deriving must not stall process readiness.
    match &metrics_addr {
        Some(addr) => tracing::info!(metrics_addr = %addr, "kona-sp1-proposer started"),
        None => tracing::info!("kona-sp1-proposer started"),
    }

    Arc::new(proposer).run().await
}
