use alloy::providers::ProviderBuilder;
use clap::Parser;
use env_logger;
use log::info;
mod extra;
mod gpo;
mod l1block;

#[derive(Parser, Debug)]
#[command(version, about, long_about = None)]
struct Args {
    #[arg(short, long, default_value = "http://localhost:9545")]
    l2: String,

    #[arg(short, long)]
    secret_key: Option<String>,
}

#[tokio::main(flavor = "current_thread")]
async fn main() {
    env_logger::init();
    let args = Args::parse();
    println!("l2: {}", args.l2);
    println!("secret_key: {:?}", args.secret_key);

    info!("Constructing provider for {}...", args.l2);
    let url = args.l2;
    let provider = ProviderBuilder::new().connect(&url).await;
    let provider = match provider {
        Ok(provider) => provider,
        Err(error) => panic!("Could not construct provider: {error:?}"),
    };

    match gpo::check(&provider).await {
        Ok(()) => (),
        Err(error) => panic!("Could not check GPO: {error:?}"),
    };

    match l1block::check(&provider).await {
        Ok(()) => (),
        Err(error) => panic!("Could not check GPO: {error:?}"),
    };

    match extra::check(&provider).await {
        Ok(()) => (),
        Err(error) => panic!("Could not check extra data: {error:?}"),
    };
}
