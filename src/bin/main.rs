use clap::Parser;
use rollup_boost::Args;

use dotenvy::dotenv;

#[tokio::main]
async fn main() -> eyre::Result<()> {
    dotenv().ok();
    Args::parse().run().await
}
