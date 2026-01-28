#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/op-rs/kona/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]

pub mod cli;
pub mod flags;
pub mod metrics;
pub(crate) mod version;

use clap::Parser;

fn main() {
    kona_cli::sigsegv_handler::install();
    kona_cli::backtrace::enable();

    if let Err(err) = cli::Cli::parse().run() {
        eprintln!("Error: {err:?}");
        std::process::exit(1);
    }
}
