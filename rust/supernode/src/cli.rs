//! The `rust-supernode` CLI.

use crate::version;
use clap::Parser;

/// The greeting the CLI prints until the supernode has behaviour of its own.
const GREETING: &str = "Hello Rust Supernode";

/// The `rust-supernode` CLI.
#[derive(Parser, Clone, Debug)]
#[command(
    author,
    version = version::short_version(),
    long_version = version::long_version(),
    about,
    long_about = None
)]
pub(crate) struct Cli {}

impl Cli {
    /// Runs the CLI.
    pub(crate) fn run(self) {
        println!("{GREETING}");
    }
}
