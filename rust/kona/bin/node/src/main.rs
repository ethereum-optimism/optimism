#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
// The workspace opts into a curated subset of pedantic/nursery lints via
// `[workspace.lints.clippy]`. The lints below are either stylistic,
// documentation-only, or would require architectural changes we intentionally
// avoid, so we allow them at the crate level.
#![allow(
    clippy::missing_errors_doc,
    clippy::missing_panics_doc,
    clippy::must_use_candidate,
    clippy::return_self_not_must_use,
    clippy::module_name_repetitions,
    clippy::redundant_pub_crate,
    clippy::too_many_lines,
    clippy::items_after_statements,
    clippy::cast_possible_truncation,
    clippy::cast_possible_wrap,
    clippy::cast_precision_loss,
    clippy::cast_sign_loss,
    clippy::cast_lossless,
    clippy::used_underscore_binding,
    clippy::unused_async,
    clippy::future_not_send,
    clippy::significant_drop_tightening,
    clippy::struct_field_names,
    clippy::similar_names,
    clippy::needless_pass_by_value,
    clippy::unused_self,
    clippy::too_long_first_doc_paragraph,
    clippy::struct_excessive_bools,
    clippy::inline_always,
    clippy::unnecessary_box_returns
)]

pub mod cli;
pub mod commands;
pub mod flags;
pub mod metrics;

pub(crate) mod version;

fn main() {
    use clap::Parser;

    kona_cli::sigsegv_handler::install();
    kona_cli::backtrace::enable();

    if let Err(err) = cli::Cli::parse().run() {
        eprintln!("Error: {err:?}");
        std::process::exit(1);
    }
}
