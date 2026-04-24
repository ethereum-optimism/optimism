// Pedantic/nursery lints from workspace-level clippy config are largely stylistic;
// reth upstream maintains its own curated lint set. Allow the cosmetic/architectural-cost
// categories here so real issues stay visible.
#![allow(clippy::cast_lossless)]
#![allow(clippy::cast_possible_truncation)]
#![allow(clippy::cast_possible_wrap)]
#![allow(clippy::cast_precision_loss)]
#![allow(clippy::cast_sign_loss)]
#![allow(clippy::default_trait_access)]
#![allow(clippy::doc_markdown)]
#![allow(clippy::elidable_lifetime_names)]
#![allow(clippy::fallible_impl_from)]
#![allow(clippy::float_cmp)]
#![allow(clippy::future_not_send)]
#![allow(clippy::ignore_without_reason)]
#![allow(clippy::ignored_unit_patterns)]
#![allow(clippy::inconsistent_struct_constructor)]
#![allow(clippy::inline_always)]
#![allow(clippy::items_after_statements)]
#![allow(clippy::large_futures)]
#![allow(clippy::large_stack_arrays)]
#![allow(clippy::large_stack_frames)]
#![allow(clippy::manual_let_else)]
#![allow(clippy::map_unwrap_or)]
#![allow(clippy::match_wildcard_for_single_variants)]
#![allow(clippy::mismatching_type_param_order)]
#![allow(clippy::missing_const_for_fn)]
#![allow(clippy::missing_errors_doc)]
#![allow(clippy::missing_fields_in_debug)]
#![allow(clippy::missing_panics_doc)]
#![allow(clippy::must_use_candidate)]
#![allow(clippy::needless_pass_by_value)]
#![allow(clippy::needless_raw_string_hashes)]
#![allow(clippy::non_std_lazy_statics)]
#![allow(clippy::redundant_closure_for_method_calls)]
#![allow(clippy::redundant_pub_crate)]
#![allow(clippy::ref_option)]
#![allow(clippy::return_self_not_must_use)]
#![allow(clippy::semicolon_if_nothing_returned)]
#![allow(clippy::significant_drop_tightening)]
#![allow(clippy::similar_names)]
#![allow(clippy::single_match_else)]
#![allow(clippy::struct_excessive_bools)]
#![allow(clippy::struct_field_names)]
#![allow(clippy::too_long_first_doc_paragraph)]
#![allow(clippy::too_many_lines)]
#![allow(clippy::unchecked_time_subtraction)]
#![allow(clippy::uninlined_format_args)]
#![allow(clippy::unnecessary_semicolon)]
#![allow(clippy::unnecessary_wraps)]
#![allow(clippy::unreadable_literal)]
#![allow(clippy::unused_async)]
#![allow(clippy::unused_self)]
#![allow(clippy::use_self)]
#![allow(clippy::used_underscore_binding)]
#![allow(clippy::wildcard_imports)]

//! # reth-proof-bench
//!
//! A benchmarking tool for measuring the performance of historical state proofs
//! retrieval using the `eth_getProof` RPC method.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]

mod args;
mod report;
mod rpc;
mod utils;

use anyhow::Result;
use clap::Parser;
use futures::stream::{self, StreamExt};
use std::time::Instant;

use crate::{
    args::Args,
    report::{BenchMetrics, BenchSummary, Reporter},
    rpc::run_proof,
    utils::get_addresses,
};
use reth_cli_runner::CliRunner;

#[allow(missing_docs)]
fn main() -> Result<()> {
    let args = Args::parse();

    if args.from > args.to {
        anyhow::bail!("--from must be less than or equal to --to");
    }

    let runner = CliRunner::try_default_runtime()?;
    runner.run_command_until_exit(|_| run(args))
}

async fn run(args: Args) -> Result<()> {
    let client = reqwest::Client::new();
    let addresses = get_addresses();

    // Use the reporter for output
    Reporter::print_header();

    let start_time = Instant::now();
    let mut current_block = args.from;

    // Initialize Summary
    let mut summary = BenchSummary::new();

    while current_block <= args.to {
        let block_start = Instant::now();

        let target_block = current_block;

        let work_items = (0..args.reqs).map(|i| {
            let addr = addresses[i % addresses.len()];
            let client = client.clone();
            let rpc_url = args.rpc.clone();
            (i, addr, client, rpc_url, target_block)
        });

        let mut stream = stream::iter(work_items)
            .map(|(attempt, addr, client, url, block)| async move {
                run_proof(client, url, block, attempt, addr).await
            })
            .buffer_unordered(args.workers);

        let mut samples = Vec::with_capacity(args.reqs);
        while let Some(sample) = stream.next().await {
            summary.add(&sample);
            samples.push(sample);
        }

        let block_duration = block_start.elapsed().as_secs_f64();

        // Clean logic: Create metrics -> Report metrics
        let metrics = BenchMetrics::new(current_block, &samples, block_duration);
        Reporter::print_metrics(&metrics);

        current_block += args.step;
    }

    let total_duration = start_time.elapsed().as_secs_f64();
    Reporter::print_summary(&summary, total_duration);

    Ok(())
}
