//! Walks the superchain-registry submodule and rewrites both committed copies
//! of the bundle zip:
//!
//! - `rust/op-superchain/data/superchain-configs.zip` (canonical for Rust)
//! - `op-core/superchain/superchain-configs.zip` (Go `//go:embed`)
//!
//! Run via `just sync-superchain`. The two outputs are byte-identical.

use std::{
    collections::{BTreeMap, BTreeSet},
    fs,
    io::Write,
    path::{Path, PathBuf},
    process::Command,
};

use serde::{Deserialize, Serialize};
use serde_json::Value;
use zip::{CompressionMethod, DateTime, ZipWriter, write::FileOptions};

const SUBMODULE_REL: &str = "packages/contracts-bedrock/lib/superchain-registry";

/// Skip-list: chains the embedded bundle must not include. Boba {Mainnet,
/// Sepolia} have non-standard genesis; Celo Mainnet is a converted L1 (not a
/// bedrock genesis), so neither the Go nor Rust readers can load them.
const SKIP_CHAIN_IDS: &[u64] = &[
    288,   // Boba Mainnet
    28882, // Boba Sepolia
    42220, // Celo Mainnet
];

#[derive(Deserialize)]
struct ChainConfigToml {
    chain_id: u64,
    #[serde(default)]
    interop: Option<InteropTable>,
}

#[derive(Deserialize)]
struct InteropTable {
    #[serde(default)]
    #[allow(clippy::zero_sized_map_values)] // matches kona's `ChainDependency`-as-unit shape.
    dependencies: BTreeMap<String, EmptyTable>,
}

#[derive(Deserialize)]
struct EmptyTable {}

#[derive(Serialize)]
struct ChainsIndexEntry<'a> {
    name: &'a str,
    network: &'a str,
}

fn main() {
    if let Err(err) = run() {
        eprintln!("regenerate: {err}");
        std::process::exit(1);
    }
}

fn run() -> Result<(), Box<dyn std::error::Error>> {
    let repo_root = resolve_repo_root()?;
    let submodule = repo_root.join(SUBMODULE_REL);
    let rust_target = repo_root.join("rust/op-superchain/data/superchain-configs.zip");
    let go_target = repo_root.join("op-core/superchain/superchain-configs.zip");

    if let Some(parent) = rust_target.parent() {
        fs::create_dir_all(parent)?;
    }
    if let Some(parent) = go_target.parent() {
        fs::create_dir_all(parent)?;
    }

    if submodule.join("superchain").exists() {
        // Full regenerate: walk the submodule and rewrite both copies.
        let commit = git_head(&submodule)?;
        let zip_bytes = build_zip(&submodule, &commit)?;
        fs::write(&rust_target, &zip_bytes)?;
        fs::write(&go_target, &zip_bytes)?;
        println!(
            "regenerate: wrote {} bytes (commit {commit}) to:\n  {}\n  {}",
            zip_bytes.len(),
            rust_target.display(),
            go_target.display()
        );
        Ok(())
    } else if rust_target.exists() {
        // Fallback for `just sync-superchain` (or a manual invocation) when
        // the submodule isn't initialised: the canonical Rust copy is in
        // git, so just propagate its bytes to the gitignored Go-side path
        // so `//go:embed` has the file at build time.
        let bytes = fs::copy(&rust_target, &go_target)?;
        println!(
            "regenerate: submodule absent; copied {bytes} bytes from {} to {}",
            rust_target.display(),
            go_target.display()
        );
        Ok(())
    } else {
        Err(format!(
            "superchain-registry submodule missing at {} AND canonical Rust copy missing at {}.\n\
             Run `git submodule update --init -- {SUBMODULE_REL}` and re-run.",
            submodule.display(),
            rust_target.display()
        )
        .into())
    }
}

fn resolve_repo_root() -> Result<PathBuf, Box<dyn std::error::Error>> {
    let output = Command::new("git").args(["rev-parse", "--show-toplevel"]).output()?;
    if !output.status.success() {
        return Err("`git rev-parse --show-toplevel` failed; run inside the monorepo".into());
    }
    Ok(PathBuf::from(String::from_utf8(output.stdout)?.trim()))
}

fn git_head(dir: &Path) -> Result<String, Box<dyn std::error::Error>> {
    let output = Command::new("git")
        .args(["-C", &dir.display().to_string(), "rev-parse", "HEAD"])
        .output()?;
    if !output.status.success() {
        return Err(format!("git rev-parse HEAD failed in {}", dir.display()).into());
    }
    Ok(String::from_utf8(output.stdout)?.trim().to_string())
}

/// Single file added to the deterministic zip.
struct ZipEntry {
    name: String,
    data: Vec<u8>,
}

fn build_zip(submodule: &Path, commit: &str) -> Result<Vec<u8>, Box<dyn std::error::Error>> {
    let configs_dir = submodule.join("superchain/configs");
    let genesis_dir = submodule.join("superchain/extra/genesis");
    let dict_path = submodule.join("superchain/extra/dictionary");
    let chain_list_path = submodule.join("chainList.json");

    let dictionary = fs::read(&dict_path)?;
    let chain_list_json = fs::read_to_string(&chain_list_path)?;

    let walked = walk_chains(&configs_dir, &genesis_dir)?;

    // Build chains.json (chain_id → {name, network}) for Go.
    let mut chains_index: BTreeMap<String, ChainsIndexEntry<'_>> = BTreeMap::new();
    for chain in &walked {
        chains_index.insert(
            chain.chain_id.to_string(),
            ChainsIndexEntry { name: &chain.name, network: &chain.environment },
        );
    }
    let chains_json = serialize_chains_json(&chains_index)?;

    // Build depsets.json from each chain's [interop.dependencies] block.
    let interop_chains: Vec<(u64, BTreeSet<u64>)> = walked
        .iter()
        .filter_map(|c| c.interop_deps.as_ref().map(|d| (c.chain_id, d.clone())))
        .collect();
    let depsets_json = aggregate_depsets_json(interop_chains)?;

    // Build superchains.json (aggregate Superchains struct, snake_case fields,
    // compatible with kona-genesis' ChainConfig via serde(alias = ...)).
    let superchains_json = build_superchains_json(&walked, submodule)?;

    // Assemble entries.
    let mut entries: Vec<ZipEntry> = Vec::new();
    entries.push(ZipEntry { name: "COMMIT".into(), data: format!("{commit}\n").into_bytes() });
    entries.push(ZipEntry { name: "dictionary".into(), data: dictionary });
    entries.push(ZipEntry { name: "chains.json".into(), data: chains_json.into_bytes() });
    entries.push(ZipEntry { name: "chainList.json".into(), data: chain_list_json.into_bytes() });
    entries.push(ZipEntry { name: "depsets.json".into(), data: depsets_json.into_bytes() });
    entries.push(ZipEntry { name: "superchains.json".into(), data: superchains_json.into_bytes() });

    // Per-superchain.toml files (raw TOML).
    for env in collect_envs(&configs_dir)? {
        let st = configs_dir.join(&env).join("superchain.toml");
        if st.exists() {
            entries.push(ZipEntry {
                name: format!("configs/{env}/superchain.toml"),
                data: fs::read(&st)?,
            });
        }
    }

    // Per-chain TOML configs + zstd-compressed genesis (raw from submodule).
    for chain in &walked {
        entries.push(ZipEntry {
            name: format!("configs/{}/{}.toml", chain.environment, chain.name),
            data: fs::read(&chain.toml_path)?,
        });
        entries.push(ZipEntry {
            name: format!("genesis/{}/{}.json.zst", chain.environment, chain.name),
            data: fs::read(&chain.genesis_path)?,
        });
    }

    entries.sort_by(|a, b| a.name.cmp(&b.name));

    write_deterministic_zip(&entries)
}

struct WalkedChain {
    chain_id: u64,
    name: String,
    environment: String,
    toml_path: PathBuf,
    genesis_path: PathBuf,
    interop_deps: Option<BTreeSet<u64>>,
}

fn collect_envs(configs_dir: &Path) -> Result<Vec<String>, Box<dyn std::error::Error>> {
    let mut envs: Vec<String> = Vec::new();
    for ent in fs::read_dir(configs_dir)? {
        let ent = ent?;
        if ent.file_type()?.is_dir() {
            envs.push(ent.file_name().to_string_lossy().into_owned());
        }
    }
    envs.sort();
    Ok(envs)
}

fn walk_chains(
    configs_dir: &Path,
    genesis_dir: &Path,
) -> Result<Vec<WalkedChain>, Box<dyn std::error::Error>> {
    let mut out: Vec<WalkedChain> = Vec::new();
    for env in collect_envs(configs_dir)? {
        let env_dir = configs_dir.join(&env);
        let mut entries: Vec<PathBuf> =
            fs::read_dir(&env_dir)?.filter_map(|e| e.ok().map(|e| e.path())).collect();
        entries.sort();
        for path in entries {
            let Some(ext) = path.extension() else { continue };
            if ext != "toml" {
                continue;
            }
            let stem = path.file_stem().unwrap().to_string_lossy().into_owned();
            if stem == "superchain" {
                continue;
            }
            let raw = fs::read_to_string(&path)?;
            let parsed: ChainConfigToml = toml::from_str(&raw)?;
            if SKIP_CHAIN_IDS.contains(&parsed.chain_id) || parsed.chain_id == 0 {
                continue;
            }
            let genesis_path = genesis_dir.join(&env).join(format!("{stem}.json.zst"));
            if !genesis_path.exists() {
                return Err(format!(
                    "missing genesis for {env}/{stem} (chain id {}) at {}",
                    parsed.chain_id,
                    genesis_path.display()
                )
                .into());
            }
            let interop_deps = parsed.interop.map(|i| {
                i.dependencies.keys().filter_map(|k| k.parse::<u64>().ok()).collect::<BTreeSet<_>>()
            });
            out.push(WalkedChain {
                chain_id: parsed.chain_id,
                name: stem,
                environment: env.clone(),
                toml_path: path,
                genesis_path,
                interop_deps,
            });
        }
    }
    out.sort_by(|a, b| {
        (a.environment.as_str(), a.name.as_str()).cmp(&(b.environment.as_str(), b.name.as_str()))
    });
    Ok(out)
}

fn serialize_chains_json(
    map: &BTreeMap<String, ChainsIndexEntry<'_>>,
) -> Result<String, Box<dyn std::error::Error>> {
    // Match the bash script's existing format: no trailing newline, no leading
    // whitespace, compact-ish. Sort by chain_id (lexicographic string sort over
    // numeric keys is fine for determinism — Go reads as a map regardless).
    let mut buf = String::from("{");
    let mut first = true;
    for (k, v) in map {
        if !first {
            buf.push(',');
        }
        first = false;
        buf.push_str(&serde_json::to_string(k)?);
        buf.push(':');
        buf.push_str(&serde_json::to_string(v)?);
    }
    buf.push('}');
    Ok(buf)
}

/// Replicates `kona_genesis::aggregate_clusters` semantics with no type deps:
/// each cluster is identified by its set of member chain ids; all members must
/// declare the same set. Output is a JSON array of `{ dependencies, overrideMessageExpiryWindow }`
/// matching kona's `DependencySet` schema.
fn aggregate_depsets_json(
    chains: Vec<(u64, BTreeSet<u64>)>,
) -> Result<String, Box<dyn std::error::Error>> {
    let by_chain: BTreeMap<u64, BTreeSet<u64>> = chains.into_iter().collect();

    for (chain_id, deps) in &by_chain {
        for dep in deps {
            if !by_chain.contains_key(dep) {
                return Err(format!("interop dangling dependency: chain {chain_id} → {dep}").into());
            }
        }
    }
    for (chain_id, deps) in &by_chain {
        for dep in deps {
            let peer = by_chain.get(dep).expect("dangling check above");
            if peer != deps {
                return Err(format!(
                    "interop inconsistent cluster: chain {chain_id} and {dep} declare different dependency sets"
                )
                .into());
            }
        }
    }

    let mut visited: BTreeSet<u64> = BTreeSet::new();
    let mut clusters: Vec<BTreeSet<u64>> = Vec::new();
    for (chain_id, deps) in &by_chain {
        if visited.contains(chain_id) || deps.is_empty() {
            visited.insert(*chain_id);
            continue;
        }
        for m in deps {
            visited.insert(*m);
        }
        clusters.push(deps.clone());
    }
    clusters.sort_by_key(|c| c.iter().copied().next().unwrap_or_default());

    let arr: Vec<Value> = clusters
        .into_iter()
        .map(|c| {
            let mut deps = serde_json::Map::new();
            for id in c {
                deps.insert(id.to_string(), Value::Object(serde_json::Map::new()));
            }
            let mut obj = serde_json::Map::new();
            obj.insert("dependencies".to_string(), Value::Object(deps));
            obj.insert("overrideMessageExpiryWindow".to_string(), Value::Null);
            Value::Object(obj)
        })
        .collect();

    let mut out = serde_json::to_string_pretty(&Value::Array(arr))?;
    out.push('\n');
    Ok(out)
}

/// Builds the aggregate `Superchains` JSON consumed by kona-registry's eager-
/// parse path. Each per-superchain group nests its `SuperchainConfig` (parsed
/// from `superchain.toml`) and the list of per-chain `ChainConfig`s (parsed
/// from each `<name>.toml`). Field names are the raw TOML `snake_case` keys;
/// kona's `ChainConfig` accepts them via `serde(alias = ...)`.
fn build_superchains_json(
    chains: &[WalkedChain],
    submodule: &Path,
) -> Result<String, Box<dyn std::error::Error>> {
    let configs_dir = submodule.join("superchain/configs");
    let mut by_env: BTreeMap<&str, Vec<&WalkedChain>> = BTreeMap::new();
    for c in chains {
        by_env.entry(&c.environment).or_default().push(c);
    }

    let mut superchains: Vec<Value> = Vec::new();
    for (env, chains) in by_env {
        let superchain_toml = configs_dir.join(env).join("superchain.toml");
        let config_value: Value = if superchain_toml.exists() {
            toml_to_json(&fs::read_to_string(&superchain_toml)?)?
        } else {
            Value::Object(serde_json::Map::new())
        };
        let mut chain_values: Vec<Value> = Vec::new();
        for c in chains {
            let raw = fs::read_to_string(&c.toml_path)?;
            chain_values.push(toml_to_json(&raw)?);
        }
        let mut entry = serde_json::Map::new();
        entry.insert("name".to_string(), Value::String(env.to_string()));
        entry.insert("config".to_string(), config_value);
        entry.insert("chains".to_string(), Value::Array(chain_values));
        superchains.push(Value::Object(entry));
    }

    let mut top = serde_json::Map::new();
    top.insert("superchains".to_string(), Value::Array(superchains));
    let mut out = serde_json::to_string_pretty(&Value::Object(top))?;
    out.push('\n');
    Ok(out)
}

fn toml_to_json(raw: &str) -> Result<Value, Box<dyn std::error::Error>> {
    let parsed: toml::Value = toml::from_str(raw)?;
    Ok(toml_value_to_json(parsed))
}

fn toml_value_to_json(value: toml::Value) -> Value {
    match value {
        toml::Value::String(s) => Value::String(s),
        toml::Value::Integer(i) => Value::from(i),
        toml::Value::Float(f) => {
            serde_json::Number::from_f64(f).map(Value::Number).unwrap_or(Value::Null)
        }
        toml::Value::Boolean(b) => Value::Bool(b),
        toml::Value::Datetime(dt) => Value::String(dt.to_string()),
        toml::Value::Array(arr) => Value::Array(arr.into_iter().map(toml_value_to_json).collect()),
        toml::Value::Table(t) => {
            let mut map = serde_json::Map::new();
            for (k, v) in t {
                map.insert(k, toml_value_to_json(v));
            }
            Value::Object(map)
        }
    }
}

fn write_deterministic_zip(entries: &[ZipEntry]) -> Result<Vec<u8>, Box<dyn std::error::Error>> {
    // 1980-01-01 00:00:00 -- the earliest representable zip timestamp,
    // guarantees byte-stable output across runs.
    let epoch_dos = DateTime::from_date_and_time(1980, 1, 1, 0, 0, 0)
        .expect("1980-01-01 is a valid DOS datetime");
    let mut buf: Vec<u8> = Vec::new();
    {
        let mut zw = ZipWriter::new(std::io::Cursor::new(&mut buf));
        let options = FileOptions::default()
            .compression_method(CompressionMethod::Deflated)
            .compression_level(Some(9))
            .last_modified_time(epoch_dos)
            .unix_permissions(0o755);
        for entry in entries {
            zw.start_file(&entry.name, options)?;
            zw.write_all(&entry.data)?;
        }
        zw.finish()?;
    }
    Ok(buf)
}
