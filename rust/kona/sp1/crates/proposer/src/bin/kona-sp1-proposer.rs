//! Proposer service binary for the super-root ZK dispute game.
//!
//! Derived from op-succinct's `fault-proof/bin/proposer.rs` (@ 13716c2c).
//! Configuration is environment-only; see `ProposerConfig::from_env`.

use std::sync::Arc;

use alloy_provider::ProviderBuilder;
use anyhow::Result;
use clap::Parser;
use kona_sp1_host_utils::{
    logger::setup_logger,
    metrics::{MetricsGauge, init_metrics},
    network::{build_network_prover_from_env, determine_network_mode},
};
use kona_sp1_proposer::{
    config::{ProofProviderKind, ProposerConfig, redacted_url},
    contract::DisputeGameFactory,
    metrics::ProposerGauge,
    proposer::Proposer,
    prover::{MockProofProvider, NetworkProofProvider, ProofProvider},
    signer::{Signer, SignerLock},
};

/// Command-line interface for process metadata; runtime configuration remains environment-only.
#[derive(Debug, Parser)]
#[command(version, about = env!("CARGO_PKG_DESCRIPTION"))]
struct Cli {}

#[tokio::main]
async fn main() -> Result<()> {
    Cli::parse();
    setup_logger();

    let config = ProposerConfig::from_env()?;

    tracing::info!(
        l1_rpc = %redacted_url(&config.l1_rpc),
        superroot_rpc = %redacted_url(&config.superroot_rpc),
        factory_address = %config.factory_address,
        prestates_url = %redacted_url(&config.prestates_url),
        proposal_interval_seconds = config.proposal_interval_seconds,
        proposal_safety = ?config.proposal_safety,
        fetch_interval = config.fetch_interval,
        metrics_listen = %config.metrics_listen,
        sync_l1_confirmations = config.sync_l1_confirmations,
        tx_confirmation_timeout = config.tx_confirmation_timeout,
        max_fee_per_gas = ?config.max_fee_per_gas,
        max_priority_fee_per_gas = ?config.max_priority_fee_per_gas,
        proof_provider = ?config.proof_provider,
        l1_beacon_rpc = %redacted_url(&config.l1_beacon_rpc),
        l2_rpcs = %config
            .l2_rpcs
            .iter()
            .map(redacted_url)
            .collect::<Vec<_>>()
            .join(","),
        rollup_config_paths = ?config.rollup_config_paths,
        l1_config_path = ?config.l1_config_path,
        dependency_set_path = ?config.dependency_set_path,
        range_split_count = ?config.range_split_count,
        max_concurrent_range_proofs = %config.max_concurrent_range_proofs,
        max_concurrent_defense_tasks = %config.max_concurrent_defense_tasks,
        fast_finality_mode = config.fast_finality_mode,
        fast_finality_proving_limit = %config.fast_finality_proving_limit,
        "Resolved proposer configuration"
    );

    // Read NETWORK_PRIVATE_KEY only in network mode; mock deployments
    // need no SPN credentials.
    let proof_provider = match config.proof_provider {
        ProofProviderKind::Network => {
            let provider_config = config.proof_provider_config.clone();
            let network_mode = determine_network_mode(
                provider_config.range_proof_strategy,
                provider_config.agg_proof_strategy,
            )?;
            let prover =
                build_network_prover_from_env(provider_config.range_proof_strategy).await?;
            ProofProvider::Network(NetworkProofProvider::new(
                Arc::new(prover),
                provider_config,
                network_mode,
            ))
        }
        ProofProviderKind::Mock => {
            tracing::warn!(
                "mock proof provider: prestate artifacts are not verified against on-chain \
                 prestates and submitted proofs are placeholder bytes (dev deployments only)"
            );
            ProofProvider::Mock(MockProofProvider)
        }
    };
    let signer = SignerLock::new(Signer::from_env().await?);

    let l1_provider = ProviderBuilder::default().connect_http(config.l1_rpc.clone());
    let factory = DisputeGameFactory::new(config.factory_address, l1_provider);

    // The AnchorStateRegistry and DelayedWETH are not pinned here: their
    // addresses come from gameArgs, which can rotate across upgrades. The
    // proposer binds the currently registered args per use and each game's
    // own args for game-specific reads.

    // Bind before readiness so the advertised address is live. Install the
    // recorder before register_all; describe_gauge! calls sent to the no-op
    // recorder lose their HELP lines.
    let metrics_addr = init_metrics(config.metrics_listen)?;
    if metrics_addr.is_some() {
        ProposerGauge::register_all();
        ProposerGauge::init_all();
    }

    let proposer = Proposer::new(config, signer, factory, proof_provider).await?;

    // Devstack readiness matches this message. Emit it before chain-dependent
    // initialization so a deriving supernode does not stall process readiness.
    match metrics_addr {
        Some(addr) => tracing::info!(metrics_addr = %addr, "kona-sp1-proposer started"),
        None => tracing::info!("kona-sp1-proposer started"),
    }

    Arc::new(proposer).run().await
}
