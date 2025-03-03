use crate::chainspec::OpChainSpecParser;
use clap::Subcommand;
use import::ImportOpCommand;
use import_receipts::ImportReceiptsOpCommand;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::{
    config_cmd, db, dump_genesis, init_cmd,
    node::{self, NoArgs},
    p2p, prune, recover, stage,
};
use std::fmt;

pub mod import;
pub mod import_receipts;
pub mod init_state;

#[cfg(feature = "dev")]
pub mod test_vectors;

/// Commands to be executed
#[derive(Debug, Subcommand)]
#[allow(clippy::large_enum_variant)]
pub enum Commands<Spec: ChainSpecParser = OpChainSpecParser, Ext: clap::Args + fmt::Debug = NoArgs>
{
    /// Start the node
    #[command(name = "node")]
    Node(Box<node::NodeCommand<Spec, Ext>>),
    /// Initialize the database from a genesis file.
    #[command(name = "init")]
    Init(init_cmd::InitCommand<Spec>),
    /// Initialize the database from a state dump file.
    #[command(name = "init-state")]
    InitState(init_state::InitStateCommandOp<Spec>),
    /// This syncs RLP encoded OP blocks below Bedrock from a file, without executing.
    #[command(name = "import-op")]
    ImportOp(ImportOpCommand<Spec>),
    /// This imports RLP encoded receipts from a file.
    #[command(name = "import-receipts-op")]
    ImportReceiptsOp(ImportReceiptsOpCommand<Spec>),
    /// Dumps genesis block JSON configuration to stdout.
    DumpGenesis(dump_genesis::DumpGenesisCommand<Spec>),
    /// Database debugging utilities
    #[command(name = "db")]
    Db(db::Command<Spec>),
    /// Manipulate individual stages.
    #[command(name = "stage")]
    Stage(Box<stage::Command<Spec>>),
    /// P2P Debugging utilities
    #[command(name = "p2p")]
    P2P(p2p::Command<Spec>),
    /// Write config to stdout
    #[command(name = "config")]
    Config(config_cmd::Command),
    /// Scripts for node recovery
    #[command(name = "recover")]
    Recover(recover::Command<Spec>),
    /// Prune according to the configuration without any limits
    #[command(name = "prune")]
    Prune(prune::PruneCommand<Spec>),
    /// Generate Test Vectors
    #[cfg(feature = "dev")]
    #[command(name = "test-vectors")]
    TestVectors(test_vectors::Command),
}
