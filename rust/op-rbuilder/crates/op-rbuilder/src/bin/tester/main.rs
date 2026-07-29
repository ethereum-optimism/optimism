use alloy_primitives::Address;
use alloy_provider::{Identity, ProviderBuilder};
use clap::Parser;
use op_alloy_network::Optimism;
use op_rbuilder::tests::*;

/// CLI Commands
#[derive(Parser, Debug)]
#[clap(author, version, about, long_about = None)]
struct Cli {
    #[clap(subcommand)]
    command: Commands,
}

#[derive(Parser, Debug)]
enum Commands {
    /// Generate genesis configuration
    Genesis {
        #[clap(long, help = "Output path for genesis files")]
        output: Option<String>,
    },
    /// Run the testing system
    Run {
        #[clap(long, short, action)]
        validation: bool,

        #[clap(long, short, action, default_value = "false")]
        no_tx_pool: bool,

        #[clap(long, short, action, default_value = "1")]
        block_time_secs: u64,

        #[clap(long, short, action)]
        flashblocks_endpoint: Option<String>,

        #[clap(long, action, default_value = "false")]
        no_sleep: bool,
    },
    /// Deposit funds to the system
    Deposit {
        #[clap(long, help = "Address to deposit funds to")]
        address: Address,
        #[clap(long, help = "Amount to deposit")]
        amount: u128,
    },
}

#[tokio::main]
async fn main() -> eyre::Result<()> {
    let cli = Cli::parse();

    match cli.command {
        Commands::Genesis { output } => generate_genesis(output),
        Commands::Run { validation, .. } => run_system(validation).await,
        Commands::Deposit { address, amount } => {
            let engine_api = EngineApi::with_http("http://localhost:4444");
            let provider = ProviderBuilder::<Identity, Identity, Optimism>::default()
                .connect_http("http://localhost:2222".try_into()?);
            let driver = ChainDriver::<Http>::remote(provider, engine_api);
            let block_hash = driver.fund(address, amount).await?;
            println!("Deposit transaction included in block: {block_hash}");
            Ok(())
        }
    }
}

#[allow(dead_code)]
pub async fn run_system(validation: bool) -> eyre::Result<()> {
    println!("Validation node enabled: {validation}");

    let engine_api = EngineApi::with_http("http://localhost:4444");
    let provider = ProviderBuilder::<Identity, Identity, Optimism>::default()
        .connect_http("http://localhost:4444".try_into()?);
    let mut driver = ChainDriver::<Http>::remote(provider, engine_api);

    if validation {
        driver = driver
            .with_validation_node(ExternalNode::reth().await?)
            .await?;
    }

    // Infinite loop generating blocks
    loop {
        println!("Generating new block...");
        let block = driver.build_new_block().await?;
        println!("Generated block: {:?}", block.header.hash);
    }
}
