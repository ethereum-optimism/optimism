#![allow(missing_docs)]

use reth_optimism_chainspec::AVAILABLE_CHAINS;
use std::{fs::File, io::Write, path::Path};

// Generate Rust code to list supported chains and a function to parse chain spec based on chain
// name
fn main() {
    // For backwards compatibility, we need to support the old chain names
    let mut supported_chains: Vec<String> = vec![
        "dev".to_string(),
        "optimism".to_string(),
        "optimism_sepolia".to_string(),
        "optimism-sepolia".to_string(),
        "base".to_string(),
        "base_sepolia".to_string(),
        "base-sepolia".to_string(),
    ];

    for chain in AVAILABLE_CHAINS.iter() {
        if chain.environment == "mainnet" {
            supported_chains.push(chain.name.clone());
        } else {
            supported_chains.push(format!("{}-{}", chain.name, chain.environment));
        }
    }

    let out_dir = std::env::var("OUT_DIR").unwrap();
    let dest_path = Path::new(&out_dir).join("chainspec_generated.rs");
    let mut f = File::create(&dest_path).unwrap();

    write!(
        f,
        "pub(crate) const SUPPORTED_CHAINS: &[&str] = &{:?};\n\n{}",
        supported_chains,
        generate_chain_value_parser_fn()
    )
    .unwrap();
}

// Generate Rust match statement code to load chain spec based on chain name and environment
fn generate_chain_value_parser_fn() -> String {
    let mut match_arms: String = String::new();
    for chain in AVAILABLE_CHAINS.iter() {
        let chain_name = if chain.environment == "mainnet" {
            chain.name.clone()
        } else {
            format!("{}-{}", chain.name, chain.environment)
        };
        let chain_spec = format!(
            "Some(reth_optimism_chainspec::{}_{}.clone())",
            chain.name.to_uppercase(),
            chain.environment.to_uppercase()
        )
        .replace('-', "_");

        match_arms.push_str(&format!("        \"{}\" => {},\n", chain_name, chain_spec));
    }
    format!(
        "pub(crate) fn generated_chain_value_parser(s: &str) -> Option<Arc<OpChainSpec>> {{
    match s {{
        \"dev\" => Some(reth_optimism_chainspec::OP_DEV.clone()),
        \"optimism\" => Some(reth_optimism_chainspec::OP_MAINNET.clone()),
        \"optimism_sepolia\" | \"optimism-sepolia\" => Some(reth_optimism_chainspec::OP_SEPOLIA.clone()),
        \"base\" => Some(reth_optimism_chainspec::BASE_MAINNET.clone()),
        \"base_sepolia\" | \"base-sepolia\" => Some(reth_optimism_chainspec::BASE_SEPOLIA.clone()),
{}        _ => None,
    }}
}}",
        match_arms
    )
}
