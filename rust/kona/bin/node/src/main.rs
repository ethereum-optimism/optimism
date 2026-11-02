//! The kona-node binary.

fn main() {
    kona_cli::sigsegv_handler::install();
    kona_cli::backtrace::enable();

    if let Err(err) = kona_node::cli::Cli::parse_args().run() {
        eprintln!("Error: {err:?}");
        std::process::exit(1);
    }
}
