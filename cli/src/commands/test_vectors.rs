//! Command for generating test vectors.

use clap::{Parser, Subcommand};
use op_alloy_consensus::TxDeposit;
use proptest::test_runner::TestRunner;
use reth_chainspec::ChainSpec;
use reth_cli_commands::{
    compact_types,
    test_vectors::{
        compact,
        compact::{
            generate_vector, read_vector, GENERATE_VECTORS as ETH_GENERATE_VECTORS,
            READ_VECTORS as ETH_READ_VECTORS,
        },
        tables,
    },
};
use std::sync::Arc;

/// Generate test-vectors for different data types.
#[derive(Debug, Parser)]
pub struct Command {
    #[command(subcommand)]
    command: Subcommands,
}

#[derive(Subcommand, Debug)]
/// `reth test-vectors` subcommands
pub enum Subcommands {
    /// Generates test vectors for specified tables. If no table is specified, generate for all.
    Tables {
        /// List of table names. Case-sensitive.
        names: Vec<String>,
    },
    /// Generates test vectors for `Compact` types with `--write`. Reads and checks generated
    /// vectors with `--read`.
    #[group(multiple = false, required = true)]
    Compact {
        /// Write test vectors to a file.
        #[arg(long)]
        write: bool,

        /// Read test vectors from a file.
        #[arg(long)]
        read: bool,
    },
}

impl Command {
    /// Execute the command
    pub async fn execute(self) -> eyre::Result<()> {
        match self.command {
            Subcommands::Tables { names } => {
                tables::generate_vectors(names)?;
            }
            Subcommands::Compact { write, .. } => {
                compact_types!(
                    regular: [
                        TxDeposit
                    ], identifier: []
                );

                if write {
                    compact::generate_vectors_with(ETH_GENERATE_VECTORS)?;
                    compact::generate_vectors_with(GENERATE_VECTORS)?;
                } else {
                    compact::read_vectors_with(ETH_READ_VECTORS)?;
                    compact::read_vectors_with(READ_VECTORS)?;
                }
            }
        }
        Ok(())
    }
    /// Returns the underlying chain being used to run this command
    pub const fn chain_spec(&self) -> Option<&Arc<ChainSpec>> {
        None
    }
}
