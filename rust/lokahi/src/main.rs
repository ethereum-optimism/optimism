#![doc = include_str!("../README.md")]
#![doc(issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/")]
#![cfg_attr(docsrs, feature(doc_cfg))]

mod cli;
mod config;
mod rpc;
mod source;
mod state;
mod store;
mod verifier;
mod version;

#[tokio::main]
async fn main() -> eyre::Result<()> {
    use clap::Parser;

    cli::Cli::parse().run().await
}
