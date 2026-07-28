//! Proposer service binary for the super-root ZK dispute game.
//!
//! Derived from op-succinct's `fault-proof/bin/proposer.rs` (@ 13716c2c).
//! Configuration is environment-only; see `ProposerConfig::from_env`.

use std::sync::Arc;

use alloy_provider::ProviderBuilder;
use anyhow::Result;
use kona_sp1_host_utils::{
    logger::setup_logger,
    metrics::{MetricsGauge, init_metrics},
};
use kona_sp1_proposer::{
    config::ProposerConfig,
    contract::DisputeGameFactory,
    metrics::ProposerGauge,
    proposer::Proposer,
    signer::{Signer, SignerLock},
};

#[tokio::main]
async fn main() -> Result<()> {
    setup_logger();

    let config = ProposerConfig::from_env()?;
    let signer = SignerLock::new(Signer::from_env().await?);

    let l1_provider = ProviderBuilder::default().connect_http(config.l1_rpc.clone());
    let factory = DisputeGameFactory::new(config.factory_address, l1_provider);

    // The AnchorStateRegistry and DelayedWETH are not pinned here: their
    // addresses come from gameArgs, which can rotate across upgrades. The
    // proposer binds the currently registered args per use and each game's
    // own args for game-specific reads.

    // Metrics: bind before the readiness log so the advertised address is live.
    // A failed bind is a startup error, not a degraded mode.
    let metrics_addr = if config.metrics_port != 0 {
        ProposerGauge::register_all();
        init_metrics(&config.metrics_port)?;
        ProposerGauge::init_all();
        Some(format!("0.0.0.0:{}", config.metrics_port))
    } else {
        None
    };

    let proposer = Proposer::new(config, signer, factory).await?;

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
