//! Walks the superchain-registry submodule and rewrites `gen/`. No-op when the
//! submodule is absent (crates.io fallback uses committed `gen/`).

use std::{
    collections::{BTreeMap, BTreeSet},
    fs,
    io::Write,
    path::{Path, PathBuf},
    process::Command,
};

use serde::Serialize;
use serde_json::{Map, Value};

const SUBMODULE_REL: &str = "packages/contracts-bedrock/lib/superchain-registry";

fn main() {
    println!("cargo:rerun-if-changed=build.rs");

    let manifest_dir = PathBuf::from(env_or_panic("CARGO_MANIFEST_DIR"));
    let gen_dir = manifest_dir.join("gen");

    let submodule = match resolve_submodule() {
        Some(path) => path,
        None => {
            // Crates.io fallback: trust the committed `gen/` tree. `cargo:rerun`
            // declarations below tie the build to anything we'd regenerate.
            announce_gen_inputs(&gen_dir);
            return;
        }
    };

    let inputs = walk_inputs(&submodule);
    for path in &inputs.tracked_paths {
        println!("cargo:rerun-if-changed={}", path.display());
    }
    println!("cargo:rerun-if-changed={}", submodule.join("chainList.json").display());

    let dictionary =
        fs::read(submodule.join("superchain/extra/dictionary")).expect("read zstd dictionary");

    fs::create_dir_all(gen_dir.join("configs")).unwrap();
    fs::create_dir_all(gen_dir.join("genesis")).unwrap();

    let mut superchains: BTreeMap<String, SuperchainAgg> = BTreeMap::new();
    let mut supported: Vec<(String, String)> = Vec::new();
    let mut interop_chains: Vec<(u64, BTreeSet<u64>)> = Vec::new();

    // Pass 1: per-superchain (env-level) config.
    for env in &inputs.envs {
        let superchain_toml =
            submodule.join("superchain/configs").join(env).join("superchain.toml");
        if !superchain_toml.exists() {
            continue;
        }
        let raw = fs::read_to_string(&superchain_toml).expect("read superchain.toml");
        let value = toml_to_json_value(&raw);
        let out = gen_dir.join("configs").join(env).join("superchain.json");
        fs::create_dir_all(out.parent().unwrap()).unwrap();
        write_json_file(&out, &value);
        superchains.insert(
            env.clone(),
            SuperchainAgg { name: env.clone(), config: value, chains: Vec::new() },
        );
    }

    // Pass 2: per-chain configs + genesis.
    for (env, chain_name, chain_toml_path) in &inputs.chains {
        let raw = fs::read_to_string(chain_toml_path).expect("read chain toml");
        let value = toml_to_json_value(&raw);

        let chain_id = chain_id_from(&value)
            .unwrap_or_else(|| panic!("missing chain_id in {}", chain_toml_path.display()));

        // Per-chain JSON.
        let out = gen_dir.join("configs").join(env).join(format!("{chain_name}.json"));
        fs::create_dir_all(out.parent().unwrap()).unwrap();
        write_json_file(&out, &value);

        // Genesis: zstd-decode (with dict) → strip `.config` → zlib-encode.
        let genesis_src = submodule
            .join("superchain/extra/genesis")
            .join(env)
            .join(format!("{chain_name}.json.zst"));
        assert!(
            genesis_src.exists(),
            "missing genesis for {env}/{chain_name} at {}",
            genesis_src.display()
        );
        let zlib_bytes = reencode_genesis(&genesis_src, &dictionary);
        let genesis_out = gen_dir.join("genesis").join(env).join(format!("{chain_name}.json.zz"));
        fs::create_dir_all(genesis_out.parent().unwrap()).unwrap();
        fs::write(&genesis_out, &zlib_bytes).expect("write genesis");

        if let Some(deps) = interop_dependencies(&value) {
            interop_chains.push((chain_id, deps));
        }

        if let Some(agg) = superchains.get_mut(env) {
            agg.chains.push(value);
        }
        supported.push((chain_name.clone(), env.clone()));
    }

    // chainList.json — verbatim copy.
    fs::write(
        gen_dir.join("chainList.json"),
        fs::read(submodule.join("chainList.json")).expect("read chainList.json"),
    )
    .expect("write chainList.json");

    // Aggregate `superchains.json` for kona's eager-parse path.
    let mut sc_list: Vec<&SuperchainAgg> = superchains.values().collect();
    sc_list.sort_by(|a, b| a.name.cmp(&b.name));
    let aggregate = serde_json::json!({ "superchains": sc_list });
    write_json_file(&gen_dir.join("superchains.json"), &aggregate);

    // depsets.json.
    let depsets = aggregate_depsets(interop_chains);
    write_json_file(&gen_dir.join("depsets.json"), &depsets);

    // index.rs — generated lookup tables.
    supported.sort();
    write_index_rs(&gen_dir.join("index.rs"), &supported);
}

fn env_or_panic(key: &str) -> String {
    std::env::var(key).unwrap_or_else(|_| panic!("missing env var {key}"))
}

/// Locates the submodule via the monorepo's git toplevel. Returns `None` if the
/// directory is missing or empty (treated as a crates.io build).
fn resolve_submodule() -> Option<PathBuf> {
    let output = Command::new("git").args(["rev-parse", "--show-toplevel"]).output().ok()?;
    if !output.status.success() {
        return None;
    }
    let top = String::from_utf8(output.stdout).ok()?;
    let top = top.trim_end();
    let path = Path::new(top).join(SUBMODULE_REL);
    let populated = path.join("superchain").try_exists().unwrap_or(false);
    populated.then_some(path)
}

struct Inputs {
    /// Per-chain `(env, chain_name, toml_path)`.
    chains: Vec<(String, String, PathBuf)>,
    /// Sorted, unique env names (`mainnet`, `sepolia`, ...).
    envs: Vec<String>,
    /// All file paths we read — emitted as `cargo:rerun-if-changed` entries.
    tracked_paths: Vec<PathBuf>,
}

fn walk_inputs(submodule: &Path) -> Inputs {
    let configs = submodule.join("superchain/configs");
    let mut envs = BTreeSet::new();
    let mut chains: Vec<(String, String, PathBuf)> = Vec::new();
    let mut tracked: Vec<PathBuf> = Vec::new();

    for env_ent in fs::read_dir(&configs).expect("read configs dir") {
        let env_ent = env_ent.expect("read configs entry");
        if !env_ent.file_type().expect("file type").is_dir() {
            continue;
        }
        let env = env_ent.file_name().to_string_lossy().into_owned();
        envs.insert(env.clone());

        for chain_ent in fs::read_dir(env_ent.path()).expect("read env dir") {
            let chain_ent = chain_ent.expect("read chain entry");
            let path = chain_ent.path();
            let Some(ext) = path.extension() else { continue };
            if ext != "toml" {
                continue;
            }
            tracked.push(path.clone());
            let name = path.file_stem().unwrap().to_string_lossy().into_owned();
            if name == "superchain" {
                continue;
            }
            // Track genesis path too.
            tracked.push(
                submodule
                    .join("superchain/extra/genesis")
                    .join(&env)
                    .join(format!("{name}.json.zst")),
            );
            chains.push((env.clone(), name, path));
        }
    }
    chains.sort();
    Inputs { chains, envs: envs.into_iter().collect(), tracked_paths: tracked }
}

/// Emits `cargo:rerun-if-changed` for every file in `gen/`. Used in the
/// submodule-absent path so a developer who edits committed fallback content
/// still rebuilds.
fn announce_gen_inputs(gen_dir: &Path) {
    fn walk(p: &Path) {
        if !p.exists() {
            return;
        }
        if p.is_dir() {
            for entry in fs::read_dir(p).unwrap() {
                walk(&entry.unwrap().path());
            }
        } else {
            println!("cargo:rerun-if-changed={}", p.display());
        }
    }
    walk(gen_dir);
}

fn toml_to_json_value(raw: &str) -> Value {
    let table: toml::Value = toml::from_str(raw).expect("parse toml");
    toml_value_to_json(table)
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
            let mut map = Map::new();
            // toml::Value::Table preserves insertion order; preserve_order on
            // serde_json keeps it through serialization.
            for (k, v) in t {
                map.insert(k, toml_value_to_json(v));
            }
            Value::Object(map)
        }
    }
}

fn chain_id_from(v: &Value) -> Option<u64> {
    v.as_object()?.get("chain_id")?.as_u64()
}

/// Returns the set of dependency chain ids from a parsed chain config's
/// `[interop.dependencies.<id>]` block. Each `<id>` is a TOML table key —
/// preserved as a string in JSON.
fn interop_dependencies(v: &Value) -> Option<BTreeSet<u64>> {
    let deps = v.as_object()?.get("interop")?.as_object()?.get("dependencies")?.as_object()?;
    Some(deps.keys().filter_map(|k| k.parse::<u64>().ok()).collect())
}

#[derive(Serialize)]
struct SuperchainAgg {
    name: String,
    config: Value,
    chains: Vec<Value>,
}

/// Replicates `kona_genesis::aggregate_clusters` without typed parsing.
/// Each cluster's canonical identifier is its set of member chain ids; all
/// members must declare the same set. Returns a serializable JSON array matching
/// kona's `Vec<DependencySet>` schema (camelCase fields).
fn aggregate_depsets(chains: Vec<(u64, BTreeSet<u64>)>) -> Value {
    let by_chain: BTreeMap<u64, BTreeSet<u64>> = chains.into_iter().collect();

    for (chain_id, deps) in &by_chain {
        for dep in deps {
            assert!(
                by_chain.contains_key(dep),
                "interop dangling dependency: chain {chain_id} → {dep}"
            );
        }
    }
    for (chain_id, deps) in &by_chain {
        for dep in deps {
            let peer = by_chain.get(dep).expect("dangling check above");
            assert!(
                peer == deps,
                "interop inconsistent cluster: chain {chain_id} and {dep} declare different dependency sets"
            );
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
            let mut deps = Map::new();
            for id in c {
                deps.insert(id.to_string(), Value::Object(Map::new()));
            }
            let mut obj = Map::new();
            obj.insert("dependencies".to_string(), Value::Object(deps));
            obj.insert("overrideMessageExpiryWindow".to_string(), Value::Null);
            Value::Object(obj)
        })
        .collect();
    Value::Array(arr)
}

fn write_json_file(path: &Path, value: &Value) {
    // pretty-print with a trailing newline for clean diffs.
    let mut bytes = serde_json::to_vec_pretty(value).expect("serialize json");
    bytes.push(b'\n');
    fs::write(path, bytes).unwrap_or_else(|e| panic!("write {}: {e}", path.display()));
}

fn reencode_genesis(src: &Path, dictionary: &[u8]) -> Vec<u8> {
    let zst_bytes = fs::read(src).expect("read zst genesis");
    let mut decoder = zstd::stream::Decoder::with_dictionary(&zst_bytes[..], dictionary)
        .expect("init zstd decoder with dictionary");
    let mut decoded = Vec::with_capacity(zst_bytes.len() * 8);
    std::io::copy(&mut decoder, &mut decoded).expect("zstd decode");

    // Strip `.config` per upstream issue #901; the field isn't consistently
    // populated and consumers reconstruct it from chain metadata at runtime.
    let mut genesis: Value = serde_json::from_slice(&decoded).expect("parse genesis json");
    if let Some(obj) = genesis.as_object_mut() {
        obj.remove("config");
    }
    let stripped = serde_json::to_vec(&genesis).expect("serialize genesis");

    miniz_oxide::deflate::compress_to_vec_zlib(&stripped, 9)
}

fn write_index_rs(path: &Path, supported: &[(String, String)]) {
    let mut out = String::new();
    out.push_str("// Generated by build.rs — do not edit.\n\n");
    out.push_str("/// All `(name, environment)` pairs shipped by this crate.\n");
    out.push_str("pub const fn supported_chains() -> &'static [(&'static str, &'static str)] {\n");
    out.push_str("    &[\n");
    for (name, env) in supported {
        out.push_str(&format!("        (\"{name}\", \"{env}\"),\n"));
    }
    out.push_str("    ]\n}\n\n");

    out.push_str("/// Per-chain config JSON (TOML→JSON conversion of the source `<name>.toml`).\n");
    out.push_str("pub fn config_str(name: &str, environment: &str) -> Option<&'static str> {\n");
    out.push_str("    match (name, environment) {\n");
    for (name, env) in supported {
        out.push_str(&format!(
            "        (\"{name}\", \"{env}\") => Some(include_str!(\"../gen/configs/{env}/{name}.json\")),\n"
        ));
    }
    out.push_str("        _ => None,\n");
    out.push_str("    }\n}\n\n");

    out.push_str("/// Per-chain zlib-compressed genesis bytes (`.config` field stripped).\n");
    out.push_str(
        "pub fn genesis_bytes(name: &str, environment: &str) -> Option<&'static [u8]> {\n",
    );
    out.push_str("    match (name, environment) {\n");
    for (name, env) in supported {
        out.push_str(&format!(
            "        (\"{name}\", \"{env}\") => Some(include_bytes!(\"../gen/genesis/{env}/{name}.json.zz\")),\n"
        ));
    }
    out.push_str("        _ => None,\n");
    out.push_str("    }\n}\n");

    let mut f = fs::File::create(path).unwrap();
    f.write_all(out.as_bytes()).unwrap();
}
