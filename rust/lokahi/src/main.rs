#![doc = include_str!("../README.md")]
#![doc(issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/")]
#![cfg_attr(docsrs, feature(doc_cfg))]

mod cli;
mod version;

fn main() {
    use clap::Parser;

    cli::Cli::parse().run();
}
