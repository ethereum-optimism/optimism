use clap::Parser;
use eyre::Result;
use serde::{Deserialize, Serialize};
use std::{
    fs::File,
    io::{BufRead, BufReader},
};
use yansi::{Color, Paint};

#[derive(Parser)]
#[command(version, about)]
struct EofCrawler {
    /// The path to the geth snapshot file.
    #[clap(short, long)]
    snapshot_file: String,
}

#[derive(Serialize, Deserialize, Debug)]
struct SnapshotAccount {
    /// The key of the account in the global state trie
    key: String,
    /// The address of the account
    address: Option<String>,
    /// The bytecode of the account
    code: Option<String>,
}

fn main() -> Result<()> {
    let args = EofCrawler::parse();

    let mut eof_contracts: Vec<SnapshotAccount> = Vec::new();

    let file = File::open(&args.snapshot_file)?;
    let mut reader = BufReader::new(file);

    println!(
        "{}",
        Paint::wrapping(format!(
            "Starting EOF contract search in {}",
            Paint::yellow(&args.snapshot_file)
        ),)
        .fg(Color::Cyan)
    );

    let mut buf = String::new();
    // Ignore the first line of the snapshot, which contains the root of the snapshot.
    if let Ok(mut num_bytes) = reader.read_line(&mut buf) {
        while num_bytes > 0 {
            buf.clear();
            num_bytes = reader.read_line(&mut buf)?;
            if buf.is_empty() {
                break;
            }

            // Check if the account is a contract, and if it is, check if it has an EOF
            // prefix.
            let contract: SnapshotAccount = serde_json::from_str(&buf)?;
            if let Some(code) = contract.code.as_ref() {
                if &code[2..4].to_uppercase() == "EF" {
                    eof_contracts.push(contract);
                }
            }
        }
    }

    println!(
        "{}",
        Paint::wrapping(format!(
            "Found {} EOF contracts",
            Paint::yellow(eof_contracts.len())
        ))
        .fg(Color::Cyan)
    );
    std::fs::write(
        "eof_contracts.json",
        serde_json::to_string_pretty(&eof_contracts)?,
    )?;
    println!(
        "{}",
        Paint::wrapping(format!(
            "Wrote EOF contracts to {}",
            Paint::yellow("eof_contracts.json")
        ))
        .fg(Color::Green)
    );

    Ok(())
}
