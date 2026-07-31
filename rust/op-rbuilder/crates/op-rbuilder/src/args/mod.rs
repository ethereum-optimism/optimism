use crate::{
    builders::BuilderMode,
    metrics::{LONG_VERSION, SHORT_VERSION},
};
use clap_builder::{CommandFactory, FromArgMatches};
pub use op::{FlashblocksArgs, OpRbuilderArgs, TelemetryArgs};
use playground::PlaygroundOptions;
use reth_optimism_cli::{chainspec::OpChainSpecParser, commands::Commands};

mod op;
mod playground;

/// This trait is used to extend Reth's CLI with additional functionality that
/// are specific to the OP builder, such as populating default values for CLI arguments
/// when running in the playground mode or checking the builder mode.
///
pub trait CliExt {
    /// Populates the default values for the CLI arguments when the user specifies
    /// the `--builder.playground` flag.
    fn populate_defaults(self) -> Self;

    /// Returns the builder mode that the node is started with.
    fn builder_mode(&self) -> BuilderMode;

    /// Returns the Cli instance with the parsed command line arguments
    /// and defaults populated if applicable.
    fn parsed() -> Self;

    /// Returns the Cli instance with the parsed command line arguments
    /// and replaces version, name, author, and about
    fn set_version() -> Self;
}

pub type Cli = reth_optimism_cli::Cli<OpChainSpecParser, OpRbuilderArgs>;

impl CliExt for Cli {
    /// Checks if the node is started with the `--builder.playground` flag,
    /// and if so, populates the default values for the CLI arguments from the
    /// playground configuration.
    ///
    /// The `--builder.playground` flag is used to populate the CLI arguments with
    /// default values for running the builder against the playground environment.
    ///
    /// The values are populated from the default directory of the playground
    /// configuration, which is `$HOME/.playground/devnet/` by default.
    ///
    /// Any manually specified CLI arguments by the user will override the defaults.
    fn populate_defaults(self) -> Self {
        let Commands::Node(ref node_command) = self.command else {
            // playground defaults are only relevant if running the node commands.
            return self;
        };

        let Some(ref playground_dir) = node_command.ext.playground else {
            // not running in playground mode.
            return self;
        };

        let options = PlaygroundOptions::new(playground_dir).unwrap_or_else(|e| exit(e));

        options.apply(self)
    }

    fn parsed() -> Self {
        Cli::set_version().populate_defaults()
    }

    /// Returns the type of builder implementation that the node is started with.
    /// Currently supports `Standard` and `Flashblocks` modes.
    fn builder_mode(&self) -> BuilderMode {
        if let Commands::Node(ref node_command) = self.command
            && node_command.ext.flashblocks.enabled
        {
            return BuilderMode::Flashblocks;
        }
        BuilderMode::Standard
    }

    /// Parses commands and overrides versions
    fn set_version() -> Self {
        let logs_dir = dirs_next::cache_dir()
            .map(|root| root.join("op-rbuilder/logs"))
            .unwrap()
            .into_os_string();
        let matches = Cli::command()
            .version(SHORT_VERSION)
            .long_version(LONG_VERSION)
            .about("Block builder designed for the Optimism stack")
            .author("Flashbots")
            .name("op-rbuilder")
            .mut_arg("log_file_directory", |arg| arg.default_value(logs_dir))
            .get_matches();
        Cli::from_arg_matches(&matches).expect("Parsing args")
    }
}

/// Following clap's convention, a failure to parse the command line arguments
/// will result in terminating the program with a non-zero exit code.
fn exit(error: eyre::Report) -> ! {
    eprintln!("{error}");
    std::process::exit(-1);
}
