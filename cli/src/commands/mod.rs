use crate::chainspec::OpChainSpecParser;
use clap::Subcommand;
use import::ImportOpCommand;
use import_receipts::ImportReceiptsOpCommand;
use reth_chainspec::{EthChainSpec, EthereumHardforks, Hardforks};
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::{
    config_cmd, db, dump_genesis, init_cmd,
    node::{self, NoArgs},
    p2p, prune, recover, stage,
};
use std::{fmt, sync::Arc};

pub mod import;
pub mod import_receipts;
pub mod init_state;

#[cfg(feature = "dev")]
pub mod test_vectors;

/// Commands to be executed
#[derive(Debug, Subcommand)]
#[expect(clippy::large_enum_variant)]
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

impl<
        C: ChainSpecParser<ChainSpec: EthChainSpec + Hardforks + EthereumHardforks>,
        Ext: clap::Args + fmt::Debug,
    > Commands<C, Ext>
{
    /// Returns the underlying chain being used for commands
    pub fn chain_spec(&self) -> Option<&Arc<C::ChainSpec>> {
        match self {
            Self::Node(cmd) => cmd.chain_spec(),
            Self::Init(cmd) => cmd.chain_spec(),
            Self::InitState(cmd) => cmd.chain_spec(),
            Self::DumpGenesis(cmd) => cmd.chain_spec(),
            Self::Db(cmd) => cmd.chain_spec(),
            Self::Stage(cmd) => cmd.chain_spec(),
            Self::P2P(cmd) => cmd.chain_spec(),
            Self::Config(_) => None,
            Self::Recover(cmd) => cmd.chain_spec(),
            Self::Prune(cmd) => cmd.chain_spec(),
            Self::ImportOp(cmd) => cmd.chain_spec(),
            Self::ImportReceiptsOp(cmd) => cmd.chain_spec(),
            #[cfg(feature = "dev")]
            Self::TestVectors(_) => None,
        }
    }
}
