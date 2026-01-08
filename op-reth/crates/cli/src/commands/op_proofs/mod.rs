//! OP Proofs management commands

use clap::{Parser, Subcommand};
use reth_cli::chainspec::ChainSpecParser;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_primitives::OpPrimitives;
use std::sync::Arc;

pub mod init;
pub mod prune;
pub mod unwind;

/// `op-reth op-proofs` command
#[derive(Debug, Parser)]
pub struct Command<C: ChainSpecParser> {
    #[command(subcommand)]
    command: Subcommands<C>,
}

impl<C: ChainSpecParser<ChainSpec = OpChainSpec>> Command<C> {
    /// Execute `op-proofs` command
    pub async fn execute<
        N: reth_cli_commands::common::CliNodeTypes<
            ChainSpec = C::ChainSpec,
            Primitives = OpPrimitives,
        >,
    >(
        self,
    ) -> eyre::Result<()> {
        match self.command {
            Subcommands::Init(cmd) => cmd.execute::<N>().await,
            Subcommands::Prune(cmd) => cmd.execute::<N>().await,
            Subcommands::Unwind(cmd) => cmd.execute::<N>().await,
        }
    }
}

impl<C: ChainSpecParser> Command<C> {
    /// Returns the underlying chain being used to run this command
    pub const fn chain_spec(&self) -> Option<&Arc<C::ChainSpec>> {
        match &self.command {
            Subcommands::Init(cmd) => cmd.chain_spec(),
            Subcommands::Prune(cmd) => cmd.chain_spec(),
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
    /// Prune old proof history to reclaim space
    #[command(name = "prune")]
    Prune(prune::PruneCommand<C>),
    /// Unwind the proofs storage to a specific block
    #[command(name = "unwind")]
    Unwind(unwind::UnwindCommand<C>),
}
