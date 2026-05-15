//! OP Proofs snapshot management commands.
//!
//! Exposes `op-reth op-proofs snapshot {init,drop}`.

use clap::{Parser, Subcommand};
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::common::CliNodeTypes;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_primitives::OpPrimitives;
use std::sync::Arc;

pub mod drop;
pub mod init;

/// `op-reth op-proofs snapshot` command.
#[derive(Debug, Parser)]
pub struct SnapshotCommand<C: ChainSpecParser> {
    #[command(subcommand)]
    command: SnapshotSubcommand<C>,
}

impl<C: ChainSpecParser<ChainSpec = OpChainSpec>> SnapshotCommand<C> {
    /// Execute the snapshot subcommand.
    pub async fn execute<N: CliNodeTypes<ChainSpec = C::ChainSpec, Primitives = OpPrimitives>>(
        self,
        runtime: reth_tasks::Runtime,
    ) -> eyre::Result<()> {
        match self.command {
            SnapshotSubcommand::Init(cmd) => cmd.execute::<N>(runtime).await,
            SnapshotSubcommand::Drop(cmd) => cmd.execute::<N>(runtime).await,
        }
    }
}

impl<C: ChainSpecParser> SnapshotCommand<C> {
    /// Returns the underlying chain being used to run this command.
    pub const fn chain_spec(&self) -> Option<&Arc<C::ChainSpec>> {
        match &self.command {
            SnapshotSubcommand::Init(cmd) => cmd.chain_spec(),
            SnapshotSubcommand::Drop(cmd) => cmd.chain_spec(),
        }
    }
}

/// `op-reth op-proofs snapshot` subcommands.
#[derive(Debug, Subcommand)]
pub enum SnapshotSubcommand<C: ChainSpecParser> {
    /// Build a snapshot at a target block and mark it Ready.
    #[command(name = "init")]
    Init(init::SnapshotInitCommand<C>),
    /// Drop the snapshot tables and meta row.
    #[command(name = "drop")]
    Drop(drop::SnapshotDropCommand<C>),
}
