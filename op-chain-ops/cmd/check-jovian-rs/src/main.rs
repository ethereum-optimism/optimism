use clap::Parser;
use env_logger;
mod gpo;

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

    match gpo::check_gpo(&args.l2).await {
        Ok(()) => (),
        Err(error) => panic!("Problem opening the file: {error:?}"),
    };
}
