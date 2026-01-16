//! Build script that generates a `configs.json` file from the configs.

use std::{
    collections::{BTreeMap, BTreeSet, btree_map::Entry},
    fs,
    path::{Path, PathBuf},
};

use kona_genesis::{Chain, ChainConfig, ChainList, Superchain, SuperchainConfig, Superchains};
use serde::de::DeserializeOwned;

fn main() {
    // If the `KONA_BIND` environment variable is _not_ set, then return early.
    let kona_bind: bool =
        std::env::var("KONA_BIND").unwrap_or_else(|_| "false".to_string()) == "true";
    println!("cargo:rerun-if-env-changed=KONA_BIND");
    if !kona_bind {
        merge_custom_configs();
        return;
    }

    // Get the directory of this file from the environment
    let src_dir = std::env::var("CARGO_MANIFEST_DIR").unwrap();

    // Check if the `superchain-registry` directory exists
    let superchain_registry = format!("{src_dir}/superchain-registry");
    if !std::path::Path::new(&superchain_registry).exists() {
        panic!("Git Submodule missing. Please run `just source` to initialize the submodule.");
    }

    // Copy the `superchain-registry/chainList.json` file to `etc/chainList.json`
    let chain_list = format!("{src_dir}/superchain-registry/chainList.json");
    let etc_dir = std::path::Path::new("etc");
    if !etc_dir.exists() {
        std::fs::create_dir_all(etc_dir).unwrap();
    }
    std::fs::copy(chain_list, "etc/chainList.json").unwrap();

    // Get the `superchain-registry/superchain/configs` directory`
    let configs_dir = format!("{src_dir}/superchain-registry/superchain/configs");
    let configs = std::fs::read_dir(configs_dir).unwrap();

    // Get all the directories in the `configs` directory
    let mut superchains = Superchains::default();
    for config in configs {
        let config = config.unwrap();
        let config_path = config.path();
        let superchain_name = config.file_name().into_string().unwrap();
        let mut superchain =
            Superchain { name: superchain_name, chains: Vec::new(), ..Default::default() };
        if config_path.is_dir() {
            let config_files = std::fs::read_dir(&config_path).unwrap();
            for config_file in config_files {
                let config_file = config_file.unwrap();
                let config_file_path = config_file.path();

                // Read the `superchain.toml` as the `SuperchainConfig`
                let config_file_name = config_file.file_name().into_string().unwrap();
                if config_file_name == "superchain.toml" {
                    let config = std::fs::read_to_string(config_file_path).unwrap();
                    let config: SuperchainConfig = toml::from_str(&config).unwrap();
                    superchain.config = config;
                    continue;
                }

                // Read the config file as a `ChainConfig`
                let config = std::fs::read_to_string(config_file_path).unwrap();
                let config: ChainConfig = toml::from_str(&config).unwrap();
                superchain.chains.push(config);
            }
            superchains.superchains.push(superchain);
        }
    }

    // Sort the superchains by name.
    superchains.superchains.sort_by(|a, b| a.name.cmp(&b.name));

    // For each superchain, sort the list of chains by chain id.
    for superchain in superchains.superchains.iter_mut() {
        superchain.chains.sort_by(|a, b| a.chain_id.cmp(&b.chain_id));
    }

    let output_path = std::path::Path::new("etc/configs.json");
    std::fs::write(output_path, serde_json::to_string_pretty(&superchains).unwrap()).unwrap();
    merge_custom_configs();
}

fn merge_custom_configs() {
    let kona_custom_configs =
        std::env::var("KONA_CUSTOM_CONFIGS").unwrap_or_else(|_| "false".to_string()) == "true";
    println!("cargo:rerun-if-env-changed=KONA_CUSTOM_CONFIGS");
    println!("cargo:rerun-if-env-changed=KONA_CUSTOM_CONFIGS_TEST");

    // if we're running tests, bust the cache if the base etc configs are updated. This ensures that
    // the test build can be repeated after modifying the base configs
    if std::env::var("KONA_CUSTOM_CONFIGS_TEST") == Ok("true".to_string()) {
        println!("cargo:rerun-if-changed=etc/chainList.json");
        println!("cargo:rerun-if-changed=etc/configs.json");
    }

    if !kona_custom_configs {
        return;
    }

    let custom_configs_dir = std::env::var("KONA_CUSTOM_CONFIGS_DIR")
        .expect("KONA_CUSTOM_CONFIGS_DIR must be set when KONA_CUSTOM_CONFIGS is enabled");
    println!("cargo:rerun-if-env-changed=KONA_CUSTOM_CONFIGS_DIR");
    let custom_configs_dir = PathBuf::from(custom_configs_dir);
    if !custom_configs_dir.exists() {
        panic!("Custom configs directory {} does not exist", custom_configs_dir.display());
    }

    let custom_chain_list_path = custom_configs_dir.join("chainList.json");
    let custom_configs_path = custom_configs_dir.join("configs.json");

    println!("cargo:rerun-if-changed={}", custom_chain_list_path.display());
    println!("cargo:rerun-if-changed={}", custom_configs_path.display());

    let target_chain_list = Path::new("etc/chainList.json");
    let target_superchains = Path::new("etc/configs.json");

    validate_chain_configs(&custom_chain_list_path, &custom_configs_path);

    merge_chain_list(&custom_chain_list_path, target_chain_list);
    merge_superchain_configs(&custom_configs_path, target_superchains);
    validate_chain_configs(target_chain_list, target_superchains);
}

fn merge_chain_list(custom_path: &Path, target_path: &Path) {
    if !custom_path.exists() {
        panic!("Custom chain list {} does not exist", custom_path.display());
    }
    if !target_path.exists() {
        panic!("Target chain list {} does not exist", target_path.display());
    }

    let mut merged_chain_list: ChainList = read_json(target_path);
    let custom_chain_list: ChainList = read_json(custom_path);

    let mut chains_by_id: BTreeMap<u64, Chain> = BTreeMap::new();
    let mut identifiers: BTreeMap<String, Chain> = BTreeMap::new();

    for chain in merged_chain_list.chains.iter() {
        let ident_key = chain.identifier.to_ascii_lowercase();
        identifiers.insert(ident_key, chain.clone());
        chains_by_id.insert(chain.chain_id, chain.clone());
    }
    // preserve ordering of chains in etc/chainList.json
    for chain in custom_chain_list.chains.iter() {
        let ident_key = chain.identifier.to_ascii_lowercase();
        if let Some(existing_chain) = identifiers.get(&ident_key) {
            if existing_chain == chain {
                continue;
            } else {
                panic!(
                    "Chain identifier `{}` in {} already exists in the registry with a different config",
                    chain.identifier,
                    custom_path.display()
                );
            }
        }
        if let Some(existing_chain) = chains_by_id.get(&chain.chain_id) {
            if existing_chain == chain {
                continue;
            } else {
                panic!(
                    "Chain id {} in {} already exists in the registry with a different config for identifier `{}`",
                    chain.chain_id,
                    custom_path.display(),
                    existing_chain.identifier
                );
            }
        }
        identifiers.insert(ident_key, chain.clone());
        chains_by_id.insert(chain.chain_id, chain.clone());
        merged_chain_list.chains.push(chain.clone());
    }

    write_pretty_json(target_path, &merged_chain_list);
}

fn merge_superchain_configs(custom_path: &Path, target_path: &Path) {
    if !custom_path.exists() {
        panic!("Custom configs {} does not exist", custom_path.display());
    }
    if !target_path.exists() {
        panic!("Target configs {} does not exist", target_path.display());
    }

    let mut superchains: BTreeMap<String, Superchain> = read_json::<Superchains>(target_path)
        .superchains
        .into_iter()
        .map(|sc| (sc.name.clone(), sc))
        .collect();

    let custom_superchains: Superchains = read_json(custom_path);

    for custom in custom_superchains.superchains {
        match superchains.entry(custom.name.clone()) {
            Entry::Occupied(mut entry) => {
                println!(
                    "cargo:warning=debug: merging custom chains {}: [{}]",
                    custom.name,
                    custom.chains.iter().map(|c| c.name.as_str()).collect::<Vec<_>>().join(",")
                );
                let existing = entry.get_mut();
                *existing = merge_superchain_entry(std::mem::take(existing), custom);
            }
            Entry::Vacant(entry) => {
                println!(
                    "cargo:warning=debug: inserting new custom chain {}: [{}]",
                    custom.name,
                    custom.chains.iter().map(|c| c.name.as_str()).collect::<Vec<_>>().join(",")
                );
                entry.insert(custom);
            }
        }
    }

    let mut merged: Vec<Superchain> = superchains.into_values().collect();
    merged.sort_by(|a, b| a.name.cmp(&b.name));
    for superchain in merged.iter_mut() {
        superchain.chains.sort_by(|a, b| a.chain_id.cmp(&b.chain_id));
    }

    let merged = Superchains { superchains: merged };
    write_pretty_json(target_path, &merged);
}

/// Merges the custom chains to the chains in the superchain-registry, panicking on conflicts
fn merge_superchain_entry(base: Superchain, custom: Superchain) -> Superchain {
    let mut merged = base;

    // maintain the ordering of chains in base
    let mut chain_map: BTreeMap<u64, ChainConfig> =
        merged.chains.clone().into_iter().map(|chain| (chain.chain_id, chain)).collect();
    for chain in custom.chains {
        if let Some(existing_config) = chain_map.get(&chain.chain_id) {
            if existing_config == &chain {
                continue;
            } else {
                panic!(
                    "conflict merging superchain `{}`: chain id {} has differing configs",
                    merged.name, chain.chain_id
                );
            }
        }
        chain_map.insert(chain.chain_id, chain.clone());
        merged.chains.push(chain.clone());
    }
    merged
}

fn validate_chain_configs(chain_list_path: &Path, superchains_path: &Path) {
    if !chain_list_path.exists() || !superchains_path.exists() {
        return;
    }

    let chain_list: ChainList = read_json(chain_list_path);
    let superchains: Superchains = read_json(superchains_path);

    let mut list_chain_ids = BTreeSet::new();
    for chain in &chain_list.chains {
        if !list_chain_ids.insert(chain.chain_id) {
            panic!(
                "Duplicate chain id {} (identifier `{}`) detected in {}",
                chain.chain_id,
                chain.identifier,
                chain_list_path.display()
            );
        }
    }

    let mut config_chain_ids = BTreeSet::new();
    for superchain in &superchains.superchains {
        for chain in &superchain.chains {
            if !config_chain_ids.insert(chain.chain_id) {
                panic!(
                    "Duplicate chain id {} detected across superchain configs in {}",
                    chain.chain_id,
                    superchains_path.display()
                );
            }
        }
    }

    for chain_id in &config_chain_ids {
        if !list_chain_ids.contains(chain_id) {
            panic!(
                "Chain id {} present in {} but missing from {}",
                chain_id,
                superchains_path.display(),
                chain_list_path.display()
            );
        }
    }

    for chain in chain_list.chains {
        if !config_chain_ids.contains(&chain.chain_id) {
            panic!(
                "Chain `{}` (chain id {}) present in {} but missing from {}",
                chain.identifier,
                chain.chain_id,
                chain_list_path.display(),
                superchains_path.display()
            );
        }
    }
}

fn read_json<T: DeserializeOwned>(path: &Path) -> T {
    let contents = fs::read_to_string(path)
        .unwrap_or_else(|e| panic!("Failed to read {}: {e}", path.display()));
    serde_json::from_str(&contents)
        .unwrap_or_else(|e| panic!("Failed to parse {}: {e}", path.display()))
}

fn write_pretty_json<T: serde::Serialize>(path: &Path, value: &T) {
    fs::write(
        path,
        serde_json::to_string_pretty(value)
            .unwrap_or_else(|e| panic!("Failed to serialize {}: {e}", path.display())),
    )
    .unwrap_or_else(|e| panic!("Failed to write {}: {e}", path.display()));
}
