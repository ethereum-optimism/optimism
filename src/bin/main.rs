use clap::Parser;
use rollup_boost::Args;

use dotenv::dotenv;

#[tokio::main]
async fn main() -> eyre::Result<()> {
    dotenv().ok();
    Args::parse().run().await
}
