//! Registry Subcommand

// Internal CLI module: the `pub` items below are for inter-module use within the
// `kona-node` binary, not a library API, so exhaustive `# Errors` / `# Panics`
// sections would duplicate the call-site context.
#![allow(clippy::missing_errors_doc, clippy::missing_panics_doc)]
use crate::flags::GlobalArgs;
use clap::Parser;
use kona_cli::LogConfig;

/// The `registry` Subcommand
///
/// The `registry` subcommand lists the OP Stack chains available in the `superchain-registry`.
///
/// # Usage
///
/// ```sh
/// kona-node registry [FLAGS] [OPTIONS]
/// ```
#[derive(Parser, Default, PartialEq, Eq, Debug, Clone)]
#[command(about = "Lists the OP Stack chains available in the superchain-registry")]
pub struct RegistryCommand;

impl RegistryCommand {
    /// Initializes the logging system based on global arguments.
    pub fn init_logs(&self, args: &GlobalArgs) -> anyhow::Result<()> {
        LogConfig::new(args.log_args.clone()).init_tracing_subscriber(None)?;
        Ok(())
    }

    /// Runs the subcommand.
    pub fn run(self, _args: &GlobalArgs) -> anyhow::Result<()> {
        let chains = kona_registry::CHAINS.chains.clone();
        let mut table = tabled::Table::new(chains);
        table.with(tabled::settings::Style::modern());
        table.modify(
            tabled::settings::object::Columns::first(),
            tabled::settings::Alignment::right(),
        );
        println!("{table}");
        Ok(())
    }
}
