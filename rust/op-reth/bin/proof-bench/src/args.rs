use clap::Parser;

#[derive(Parser, Debug)]
#[command(author, version, about = "Benchmark eth_getProof performance", long_about = None)]
pub struct Args {
    #[arg(long, default_value = "http://localhost:8545")]
    pub rpc: String,

    #[arg(long)]
    pub from: u64,

    #[arg(long)]
    pub to: u64,

    #[arg(long, default_value_t = 10000)]
    pub step: u64,

    #[arg(long, default_value_t = 10)]
    pub reqs: usize,

    #[arg(long, default_value_t = 2)]
    pub workers: usize,
}
