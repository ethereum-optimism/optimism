use clap::Parser;

#[derive(Parser, Debug)]
#[command(version, about, long_about = None)]
struct Args {
    #[arg(short, long, default_value = "http://localhost:9545")]
    l2: String,

    #[arg(short, long)]
    secret_key: Option<String>,
}

fn main() {
    let args = Args::parse();
    println!("l2: {}", args.l2);
    println!("secret_key: {:?}", args.secret_key);
}
