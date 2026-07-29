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
