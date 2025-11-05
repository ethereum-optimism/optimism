use clap::Parser;
use dotenvy::dotenv;
use rollup_boost::RollupBoostServiceArgs;
use rollup_boost::init_tracing;

#[tokio::main]
async fn main() -> eyre::Result<()> {
    dotenv().ok();

    let args = RollupBoostServiceArgs::parse();
    init_tracing(&args)?;
    args.run().await
}
