//! OP Proofs management commands

use clap::{Parser, Subcommand};
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::common::CliNodeTypes;
use reth_node_metrics::recorder::install_prometheus_recorder;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_primitives::OpPrimitives;
use std::sync::Arc;

pub mod backfill;
pub mod init;
pub mod prune;
pub mod snapshot;
pub mod unwind;

/// `op-reth op-proofs` command
#[derive(Debug, Parser)]
pub struct Command<C: ChainSpecParser> {
    #[command(subcommand)]
    command: Subcommands<C>,
}

impl<C: ChainSpecParser<ChainSpec = OpChainSpec>> Command<C> {
    /// Execute `op-proofs` command
    pub async fn execute<N: CliNodeTypes<ChainSpec = C::ChainSpec, Primitives = OpPrimitives>>(
        self,
        runtime: reth_tasks::Runtime,
    ) -> eyre::Result<()> {
        // Drain the Prometheus recorder's per-metric `AtomicBucket`s on a 5s tick.
        // The buckets are append-only; without this, `reth_trie::trie::StateRoot::calculate`
        // (run per backfilled/initialized block) leaks ~3 KB/block via histogram observations
        // and OOMs on chain-sized runs. See paradigmxyz/reth#12664. op-proofs subcommands have no endpoint, so
        // we spawn upkeep here directly. Idempotent — safe under repeated invocation.
        install_prometheus_recorder().spawn_upkeep();

        match self.command {
            Subcommands::Init(cmd) => cmd.execute::<N>(runtime).await,
            Subcommands::Backfill(cmd) => cmd.execute::<N>(runtime).await,
            Subcommands::Prune(cmd) => cmd.execute::<N>(runtime).await,
            Subcommands::Snapshot(cmd) => cmd.execute::<N>(runtime).await,
            Subcommands::Unwind(cmd) => cmd.execute::<N>(runtime).await,
        }
    }
}

impl<C: ChainSpecParser> Command<C> {
    /// Returns the underlying chain being used to run this command
    pub const fn chain_spec(&self) -> Option<&Arc<C::ChainSpec>> {
        match &self.command {
            Subcommands::Init(cmd) => cmd.chain_spec(),
            Subcommands::Backfill(cmd) => cmd.chain_spec(),
            Subcommands::Prune(cmd) => cmd.chain_spec(),
            Subcommands::Snapshot(cmd) => cmd.chain_spec(),
            Subcommands::Unwind(cmd) => cmd.chain_spec(),
        }
    }
}

/// `op-reth op-proofs` subcommands
#[derive(Debug, Subcommand)]
pub enum Subcommands<C: ChainSpecParser> {
    /// Initialize the proofs storage with the current state of the chain
    #[command(name = "init")]
    Init(init::InitCommand<C>),
    /// Backfill proofs history to an older earliest block
    #[command(name = "backfill")]
    Backfill(backfill::BackfillCommand<C>),
    /// Prune old proof history to reclaim space
    #[command(name = "prune")]
    Prune(prune::PruneCommand<C>),
    /// Build or drop the trie-state snapshot
    #[command(name = "snapshot")]
    Snapshot(snapshot::SnapshotCommand<C>),
    /// Unwind the proofs storage to a specific block
    #[command(name = "unwind")]
    Unwind(unwind::UnwindCommand<C>),
}
