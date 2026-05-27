//! Build script that generates a `configs.json` file from the configs.

use std::{
    collections::{BTreeMap, BTreeSet, btree_map::Entry},
    fs,
    path::{Path, PathBuf},
};

use kona_genesis::{
    Chain, ChainConfig, ChainList, DependencySet, InteropConfig, Superchain, Superchains,
    aggregate_clusters,
};
use serde::de::DeserializeOwned;

fn main() {
    // Declared up-front so rustc doesn't warn about the unused cfg name when the
    // custom-configs path isn't active.
    println!("cargo:rustc-check-cfg=cfg(kona_custom_configs)");
    println!("cargo:rerun-if-env-changed=KONA_CUSTOM_CONFIGS");
    println!("cargo:rerun-if-env-changed=KONA_CUSTOM_CONFIGS_TEST");
    println!("cargo:rerun-if-env-changed=KONA_CUSTOM_CONFIGS_DIR");

    let kona_custom_configs =
        std::env::var("KONA_CUSTOM_CONFIGS").unwrap_or_else(|_| "false".to_string()) == "true";

    // Default path: lib.rs uses `op_superchain::*` constants directly via the
    // `inputs` module. No build-side work to do.
    if !kona_custom_configs {
        return;
    }

    // Custom-configs path: re-walk the submodule (typed parse via kona-genesis)
    // to produce the base chainList/configs/depsets in $OUT_DIR, then overlay
    // the custom configs on top. lib.rs's `cfg(kona_custom_configs)` arm picks
    // up the merged outputs from $OUT_DIR.
    println!("cargo:rustc-cfg=kona_custom_configs");

    let out_dir = PathBuf::from(std::env::var("OUT_DIR").expect("OUT_DIR set"));
    let chain_list_path = out_dir.join("chainList.json");
    let configs_path = out_dir.join("configs.json");
    let depsets_path = out_dir.join("depsets.json");

    let superchain_registry = resolve_submodule();
    println!("cargo:rerun-if-changed={}", superchain_registry.join("chainList.json").display());

    std::fs::copy(superchain_registry.join("chainList.json"), &chain_list_path)
        .expect("copy chainList.json");

    let configs_dir = superchain_registry.join("superchain/configs");
    let mut superchains = Superchains::default();
    for env_ent in std::fs::read_dir(&configs_dir).expect("read superchain/configs") {
        let env_ent = env_ent.expect("read configs entry");
        let env_path = env_ent.path();
        if !env_path.is_dir() {
            continue;
        }
        let superchain_name = env_ent.file_name().into_string().unwrap();
        let mut superchain =
            Superchain { name: superchain_name, chains: Vec::new(), ..Default::default() };

        for chain_ent in std::fs::read_dir(&env_path).expect("read env dir") {
            let chain_ent = chain_ent.expect("read chain entry");
            let chain_path = chain_ent.path();
            println!("cargo:rerun-if-changed={}", chain_path.display());
            let chain_name = chain_ent.file_name().into_string().unwrap();
            let raw = std::fs::read_to_string(&chain_path).expect("read chain toml");
            if chain_name == "superchain.toml" {
                superchain.config = toml::from_str(&raw).expect("parse superchain.toml");
                continue;
            }
            let cfg: ChainConfig = toml::from_str(&raw)
                .unwrap_or_else(|e| panic!("parse {}: {e}", chain_path.display()));
            superchain.chains.push(cfg);
        }
        superchains.superchains.push(superchain);
    }
    superchains.superchains.sort_by_key(|a| a.name.clone());
    for superchain in &mut superchains.superchains {
        superchain.chains.sort_by_key(|a| a.chain_id);
    }
    std::fs::write(&configs_path, serde_json::to_string_pretty(&superchains).unwrap())
        .expect("write configs.json");

    // Base depsets from the submodule.
    let interop_chains: Vec<(u64, &InteropConfig)> = superchains
        .superchains
        .iter()
        .flat_map(|sc| sc.chains.iter())
        .filter_map(|c| c.interop.as_ref().map(|i| (c.chain_id, i)))
        .collect();
    let depsets = aggregate_clusters(interop_chains.iter().map(|(id, cfg)| (*id, *cfg)))
        .unwrap_or_else(|e| {
            panic!("failed to aggregate interop clusters from superchain configs: {e}")
        });
    write_depsets(&depsets_path, &depsets);

    overlay_custom_configs(&chain_list_path, &configs_path, &depsets_path);
}

/// Resolves the monorepo root and returns the path to the superchain-registry
/// submodule. Panics with a directed message when the submodule is missing.
fn resolve_submodule() -> PathBuf {
    let output = std::process::Command::new("git")
        .args(["rev-parse", "--show-toplevel"])
        .output()
        .expect("failed to run `git rev-parse --show-toplevel`");
    assert!(output.status.success(), "`git rev-parse --show-toplevel` failed");
    let repo_root = String::from_utf8(output.stdout).unwrap();
    let repo_root = repo_root.trim_end();
    let path =
        PathBuf::from(format!("{repo_root}/packages/contracts-bedrock/lib/superchain-registry"));
    assert!(
        path.exists(),
        "superchain-registry submodule missing. Run `just source` to initialize it."
    );
    path
}

/// Layers the `KONA_CUSTOM_CONFIGS_DIR` overlay (chainList + configs + depsets)
/// on top of the base files at the given paths.
fn overlay_custom_configs(chain_list_path: &Path, configs_path: &Path, depsets_path: &Path) {
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
    // preserve ordering of chains in etc/chainList.json
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

fn write_depsets(target: &Path, depsets: &[DependencySet]) {
    let json = serde_json::to_string_pretty(depsets)
        .unwrap_or_else(|e| panic!("Failed to serialize {}: {e}", target.display()));
    fs::write(target, json).unwrap_or_else(|e| panic!("Failed to write {}: {e}", target.display()));
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
    write_depsets(target, &existing);
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
