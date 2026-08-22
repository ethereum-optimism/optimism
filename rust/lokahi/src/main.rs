#![doc = include_str!("../README.md")]
#![doc(issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/")]
#![cfg_attr(docsrs, feature(doc_cfg))]

mod cli;
mod config;
mod metrics;
mod supernode;
mod version;

fn main() {
    use clap::Parser;

    kona_cli::sigsegv_handler::install();
    kona_cli::backtrace::enable();

    if let Err(err) = cli::Cli::parse().run() {
        eprintln!("Error: {err:?}");
        std::process::exit(1);
    }
}
