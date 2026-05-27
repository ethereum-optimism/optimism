//! Extracts the three aggregate JSON files (`chainList.json`, `configs.json`,
//! `depsets.json`) from the committed `op-superchain` zip into `$OUT_DIR`.
//!
//! `lib.rs` and `superchain.rs` `include_str!` the `OUT_DIR` copies. When the
//! optional `KONA_CUSTOM_CONFIGS` env var is set, the existing merge logic
//! overlays a developer-supplied `KONA_CUSTOM_CONFIGS_DIR` on top of the
//! `OUT_DIR` files in place -- same fail-fast validation as before, with the
//! same on-disk schema.

use std::{
    collections::{BTreeMap, BTreeSet, btree_map::Entry},
    fs,
    path::{Path, PathBuf},
};

use kona_genesis::{Chain, ChainConfig, ChainList, DependencySet, Superchain, Superchains};
use serde::de::DeserializeOwned;

fn main() {
    println!("cargo:rerun-if-changed=build.rs");
    println!("cargo:rerun-if-env-changed=KONA_CUSTOM_CONFIGS");
    println!("cargo:rerun-if-env-changed=KONA_CUSTOM_CONFIGS_TEST");
    println!("cargo:rerun-if-env-changed=KONA_CUSTOM_CONFIGS_DIR");

    let out_dir = PathBuf::from(std::env::var("OUT_DIR").expect("OUT_DIR set"));
    let chain_list_path = out_dir.join("chainList.json");
    let configs_path = out_dir.join("configs.json");
    let depsets_path = out_dir.join("depsets.json");

    // Pull the three aggregates straight out of the committed zip in
    // op-superchain. include_bytes! handles its own rerun bookkeeping via
    // op-superchain's build.rs.
    fs::write(&chain_list_path, op_superchain::read_chain_list_json().expect("read chainList"))
        .expect("write chainList.json");
    fs::write(&configs_path, op_superchain::read_superchains_json().expect("read superchains"))
        .expect("write superchains.json");
    fs::write(&depsets_path, op_superchain::read_depsets_json().expect("read depsets"))
        .expect("write depsets.json");

    if std::env::var("KONA_CUSTOM_CONFIGS").as_deref() == Ok("true") {
        merge_custom_configs(&chain_list_path, &configs_path, &depsets_path);
    }
}

/// Layers the `KONA_CUSTOM_CONFIGS_DIR` overlay (chainList + configs + depsets)
/// on top of the `OUT_DIR` copies the default path wrote.
fn merge_custom_configs(chain_list_path: &Path, configs_path: &Path, depsets_path: &Path) {
    let custom_configs_dir = std::env::var("KONA_CUSTOM_CONFIGS_DIR")
        .expect("KONA_CUSTOM_CONFIGS_DIR must be set when KONA_CUSTOM_CONFIGS is enabled");
    let custom_configs_dir = PathBuf::from(custom_configs_dir);
    assert!(
        custom_configs_dir.exists(),
        "Custom configs directory {} does not exist",
        custom_configs_dir.display()
    );

    let custom_chain_list_path = custom_configs_dir.join("chainList.json");
    let custom_configs_path = custom_configs_dir.join("configs.json");

    println!("cargo:rerun-if-changed={}", custom_chain_list_path.display());
    println!("cargo:rerun-if-changed={}", custom_configs_path.display());

    validate_chain_configs(&custom_chain_list_path, &custom_configs_path);

    merge_chain_list(&custom_chain_list_path, chain_list_path);
    merge_superchain_configs(&custom_configs_path, configs_path);
    merge_custom_depsets(&custom_configs_dir, depsets_path);
    validate_chain_configs(chain_list_path, configs_path);
    validate_depsets(depsets_path, chain_list_path);
}

fn merge_chain_list(custom_path: &Path, target_path: &Path) {
    assert!(custom_path.exists(), "Custom chain list {} does not exist", custom_path.display());
    assert!(target_path.exists(), "Target chain list {} does not exist", target_path.display());

    let mut merged_chain_list: ChainList = read_json(target_path);
    let custom_chain_list: ChainList = read_json(custom_path);

    let mut chains_by_id: BTreeMap<u64, Chain> = BTreeMap::new();
    let mut identifiers: BTreeMap<String, Chain> = BTreeMap::new();

    for chain in &merged_chain_list.chains {
        let ident_key = chain.identifier.to_ascii_lowercase();
        identifiers.insert(ident_key, chain.clone());
        chains_by_id.insert(chain.chain_id, chain.clone());
    }
    // preserve ordering of chains in the base chainList.json
    for chain in &custom_chain_list.chains {
        let ident_key = chain.identifier.to_ascii_lowercase();
        if let Some(existing_chain) = identifiers.get(&ident_key) {
            if existing_chain == chain {
                continue;
            }
            panic!(
                "Chain identifier `{}` in {} already exists in the registry with a different config",
                chain.identifier,
                custom_path.display()
            );
        }
        if let Some(existing_chain) = chains_by_id.get(&chain.chain_id) {
            if existing_chain == chain {
                continue;
            }
            panic!(
                "Chain id {} in {} already exists in the registry with a different config for identifier `{}`",
                chain.chain_id,
                custom_path.display(),
                existing_chain.identifier
            );
        }
        identifiers.insert(ident_key, chain.clone());
        chains_by_id.insert(chain.chain_id, chain.clone());
        merged_chain_list.chains.push(chain.clone());
    }

    write_pretty_json(target_path, &merged_chain_list);
}

fn merge_superchain_configs(custom_path: &Path, target_path: &Path) {
    assert!(custom_path.exists(), "Custom configs {} does not exist", custom_path.display());
    assert!(target_path.exists(), "Target configs {} does not exist", target_path.display());

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
    merged.sort_by_key(|a| a.name.clone());
    for superchain in &mut merged {
        superchain.chains.sort_by_key(|a| a.chain_id);
    }

    let merged = Superchains { superchains: merged };
    write_pretty_json(target_path, &merged);
}

/// Merges the custom chains into a superchain, panicking on conflicts.
fn merge_superchain_entry(base: Superchain, custom: Superchain) -> Superchain {
    let mut merged = base;

    let mut chain_map: BTreeMap<u64, ChainConfig> =
        merged.chains.clone().into_iter().map(|chain| (chain.chain_id, chain)).collect();
    for chain in custom.chains {
        if let Some(existing_config) = chain_map.get(&chain.chain_id) {
            if existing_config == &chain {
                continue;
            }
            panic!(
                "conflict merging superchain `{}`: chain id {} has differing configs",
                merged.name, chain.chain_id
            );
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
        assert!(
            list_chain_ids.insert(chain.chain_id),
            "Duplicate chain id {} (identifier `{}`) detected in {}",
            chain.chain_id,
            chain.identifier,
            chain_list_path.display()
        );
    }

    let mut config_chain_ids = BTreeSet::new();
    for superchain in &superchains.superchains {
        for chain in &superchain.chains {
            assert!(
                config_chain_ids.insert(chain.chain_id),
                "Duplicate chain id {} detected across superchain configs in {}",
                chain.chain_id,
                superchains_path.display()
            );
        }
    }

    for chain_id in &config_chain_ids {
        assert!(
            list_chain_ids.contains(chain_id),
            "Chain id {} present in {} but missing from {}",
            chain_id,
            superchains_path.display(),
            chain_list_path.display()
        );
    }

    for chain in chain_list.chains {
        assert!(
            config_chain_ids.contains(&chain.chain_id),
            "Chain `{}` (chain id {}) present in {} but missing from {}",
            chain.identifier,
            chain.chain_id,
            chain_list_path.display(),
            superchains_path.display()
        );
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

fn merge_custom_depsets(custom_dir: &Path, target: &Path) {
    let path = custom_dir.join("depsets.json");
    println!("cargo:rerun-if-changed={}", path.display());
    if !path.exists() {
        return;
    }
    let custom: Vec<DependencySet> = read_json(&path);
    let mut existing: Vec<DependencySet> = read_json(target);
    for new_ds in custom {
        for existing_ds in &existing {
            let collisions: Vec<u64> = new_ds
                .dependencies
                .keys()
                .filter(|k| existing_ds.dependencies.contains_key(k))
                .copied()
                .collect();
            assert!(
                collisions.is_empty() || existing_ds == &new_ds,
                "Custom depset overlaps existing cluster on chain ids {collisions:?} but the cluster contents differ"
            );
        }
        if !existing.iter().any(|d| d == &new_ds) {
            existing.push(new_ds);
        }
    }
    let json = serde_json::to_string_pretty(&existing)
        .unwrap_or_else(|e| panic!("Failed to serialize {}: {e}", target.display()));
    fs::write(target, json).unwrap_or_else(|e| panic!("Failed to write {}: {e}", target.display()));
}

fn validate_depsets(target: &Path, chain_list_path: &Path) {
    let depsets: Vec<DependencySet> = read_json(target);
    let chain_list: ChainList = read_json(chain_list_path);
    let known: BTreeSet<u64> = chain_list.chains.iter().map(|c| c.chain_id).collect();
    for ds in &depsets {
        for id in ds.dependencies.keys() {
            assert!(
                known.contains(id),
                "Depset references chain id {id} which is not in {}",
                chain_list_path.display()
            );
        }
    }
}
