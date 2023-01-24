use clap::Parser;
use ethers::providers::{Ipc, Middleware, Provider};
use eyre::Result;
use futures::future;
use rayon::prelude::*;
use serde::Serialize;
use std::sync::Arc;
use yansi::Paint;

#[derive(Parser)]
#[command(version, about)]
struct EofCrawler {
    /// The path to the IPC socket to connect to
    #[clap(short, long)]
    ipc: String,

    /// The block to start crawling from
    #[clap(short, long)]
    start: Option<u64>,

    /// The block to end crawling at
    #[clap(short, long)]
    end: Option<u64>,
}

#[derive(Serialize)]
struct EOFContract {
    address: String,
    deployment_block: u64,
    deployer: String,
    bytecode: String,
}

#[tokio::main]
#[allow(dead_code)]
async fn main() -> Result<()> {
    // Parse CLI args
    let args = EofCrawler::parse();

    // Create a new provider with a connection to an IPC socket
    let provider = Arc::new(Provider::<Ipc>::connect_ipc(&args.ipc).await?);

    // Unless a starting block was specified, begin at genesis.
    let start = args.start.unwrap_or(0);

    // Get the latest block or use the end specified in the CLI args
    let end = match args.end {
        Some(block) => block,
        None => provider.get_block_number().await?.as_u64(),
    };

    // Perform a parallel search over blocks [0, block] for contracts
    // that were deployed with the EOF prefix.
    println!(
        "{}",
        Paint::green(format!("Starting search on block range [{start}, {end}]"))
    );

    let tasks: Vec<_> = (start..=end)
        .into_iter()
        .map(|block_no| {
            // Clone the provider's reference counter to move into the task
            let provider = Arc::clone(&provider);

            // Spawn a task to search for EOF contracts in the block
            tokio::spawn(async move {
                // Grab the block at the current block number, panic if we were unable
                // to fetch it.
                let block = provider
                    .get_block_with_txs(block_no)
                    .await
                    .unwrap()
                    .unwrap();

                // Iterate through all transactions within the block in parallel to look
                // for creation txs.
                block.transactions.into_par_iter().for_each(|tx| {
                    // `tx.to` is `None` if the transaction is a contract creation
                    // TODO: Check if contract creation failed.
                    if tx.to.is_none() {
                        // TODO: Pull the runtime code from the creation code; store info
                        // about the contract.
                    }
                });
            })
        })
        .collect();

    // Wait on futures in parallel.
    future::join_all(tasks).await;

    Ok(())
}
