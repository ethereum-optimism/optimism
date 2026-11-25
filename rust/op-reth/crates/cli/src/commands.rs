use crate::chainspec::OpChainSpecParser;
use clap::Subcommand;
use reth_cli_commands::{
    config_cmd, db, dump_genesis, init_cmd,
    node::{self, NoArgs},
    p2p, prune, re_execute, stage,
};
use reth_optimism_chainspec::OpChainSpec;
use std::{fmt, sync::Arc};

pub mod init_state;
pub mod op_proofs;
mod slot_preimages_seed;

#[cfg(feature = "dev")]
pub mod test_vectors;

/// Commands to be executed
#[derive(Debug, Subcommand)]
pub enum Commands<Ext: clap::Args + fmt::Debug = NoArgs> {
    /// Start the node
    #[command(name = "node")]
    Node(Box<node::NodeCommand<OpChainSpecParser, Ext>>),
    /// Initialize the database from a genesis file.
    #[command(name = "init")]
    Init(init_cmd::InitCommand<OpChainSpecParser>),
    /// Initialize the database from a state dump file.
    #[command(name = "init-state")]
    InitState(init_state::InitStateCommandOp),
    /// Dumps genesis block JSON configuration to stdout.
    DumpGenesis(dump_genesis::DumpGenesisCommand<OpChainSpecParser>),
    /// Database debugging utilities
    #[command(name = "db")]
    Db(db::Command<OpChainSpecParser>),
    /// Manipulate individual stages.
    #[command(name = "stage")]
    Stage(Box<stage::Command<OpChainSpecParser>>),
    /// P2P Debugging utilities
    #[command(name = "p2p")]
    P2P(Box<p2p::Command<OpChainSpecParser>>),
    /// Write config to stdout
    #[command(name = "config")]
    Config(config_cmd::Command),
    /// Prune according to the configuration without any limits
    #[command(name = "prune")]
    Prune(prune::PruneCommand<OpChainSpecParser>),
    /// Generate Test Vectors
    #[cfg(feature = "dev")]
    #[command(name = "test-vectors")]
    TestVectors(test_vectors::Command),
    /// Re-execute blocks in parallel to verify historical sync correctness.
    #[command(name = "re-execute")]
    ReExecute(re_execute::Command<OpChainSpecParser>),
    /// Manage storage of historical proofs in expanded trie db in fault proof window.
    #[command(name = "proofs")]
    OpProofs(op_proofs::Command),
}

impl<Ext: clap::Args + fmt::Debug> Commands<Ext> {
    /// Returns the underlying chain being used for commands
    pub fn chain_spec(&self) -> Option<&Arc<OpChainSpec>> {
        match self {
            Self::Node(cmd) => cmd.chain_spec(),
            Self::Init(cmd) => cmd.chain_spec(),
            Self::InitState(cmd) => cmd.chain_spec(),
            Self::DumpGenesis(cmd) => cmd.chain_spec(),
            Self::Db(cmd) => cmd.chain_spec(),
            Self::Stage(cmd) => cmd.chain_spec(),
            Self::P2P(cmd) => cmd.chain_spec(),
            Self::Config(_) => None,
            Self::Prune(cmd) => cmd.chain_spec(),
            #[cfg(feature = "dev")]
            Self::TestVectors(_) => None,
            Self::ReExecute(cmd) => cmd.chain_spec(),
            Self::OpProofs(cmd) => cmd.chain_spec(),
        }
    }
}
