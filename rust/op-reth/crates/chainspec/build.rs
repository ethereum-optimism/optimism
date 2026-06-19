//! Materializes `res/superchain-configs.tar` from the `superchain-registry`
//! submodule, so the (large, non-text) archive doesn't need to be committed — only
//! its `res/superchain-configs.tar.sha256` is. The Rust port of the former
//! `res/fetch_superchain_config.sh`, using pinned, deterministic crates
//! (`toml`/`serde_json` for TOML→JSON, `zstd` to decode the dictionary-compressed
//! genesis, `miniz_oxide` to re-emit plain zlib the `no_std` runtime can inflate).
//!
//! Semantics mirror kona's `KONA_SYNC_SUPERCHAIN`, with one addition — op-reth's tar
//! is gitignored (not committed text), so it can be absent:
//! - **default** (env unset): if `res/superchain-configs.tar` is present (provided by `just
//!   sync-superchain-rust` or copied into the build context), use it as-is — no submodule needed —
//!   after checking it matches the committed `.sha256`. If it is absent, regenerate it from the
//!   submodule and assert it matches the committed pin.
//! - **`OP_RETH_SYNC_SUPERCHAIN=1`** (run by `just sync-superchain-rust`): always regenerate the
//!   tar, its `.sha256`, and `src/superchain/chain_specs.rs`.

use std::{
    fs,
    io::Read,
    path::{Path, PathBuf},
};

/// Committed file pinning the expected archive hash (sha256sum format).
const SHA_FILE: &str = "res/superchain-configs.tar.sha256";
const TAR_NAME: &str = "superchain-configs.tar";
/// Fixed tar entry timestamp (1980-01-01), for a reproducible archive.
const MTIME: u64 = 315_532_800;
/// Upper bound for a decompressed genesis file (matches the runtime limit).
const MAX_GENESIS_SIZE: u64 = 100 * 1024 * 1024;

fn main() {
    println!("cargo:rerun-if-env-changed=OP_RETH_SYNC_SUPERCHAIN");
    println!("cargo:rerun-if-changed={SHA_FILE}");

    // The tar is only consumed under the `superchain-configs` feature; skip the
    // (non-trivial) generation work entirely when it's off.
    if std::env::var_os("CARGO_FEATURE_SUPERCHAIN_CONFIGS").is_none() {
        return;
    }

    let manifest_dir = PathBuf::from(std::env::var("CARGO_MANIFEST_DIR").unwrap());
    let tar_path = manifest_dir.join("res").join(TAR_NAME);
    let sha_path = manifest_dir.join(SHA_FILE);
    println!("cargo:rerun-if-changed=res/{TAR_NAME}");

    let refresh = matches!(std::env::var("OP_RETH_SYNC_SUPERCHAIN").as_deref(), Ok("1" | "true"));
    let expected_sha = {
        let s = fs::read_to_string(&sha_path).unwrap_or_default();
        s.split_whitespace().next().unwrap_or_default().to_string()
    };

    // Fast path (default mode, artifact already provided): use the gitignored
    // res/superchain-configs.tar as-is — provided by `just sync-superchain-rust` or
    // copied into the build context — without touching the submodule. Just check it
    // matches the committed .sha256 so a stale artifact fails loudly.
    if !refresh && tar_path.exists() {
        let sha = sha256_hex(&fs::read(&tar_path).unwrap());
        assert_eq!(
            sha, expected_sha,
            "res/superchain-configs.tar is out of date with its .sha256 — run \
             `just sync-superchain-rust`."
        );
        return;
    }

    // Otherwise — refresh mode, or default mode with the tar absent — (re)generate it
    // from the superchain-registry submodule. Resolve the submodule relative to this
    // crate (rust/op-reth/crates/chainspec) rather than via git, so it also works in
    // contexts without a `.git` dir.
    let sr = manifest_dir.ancestors().nth(4).expect("repo root").join("superchain-registry");
    assert!(
        sr.join("superchain").is_dir(),
        "superchain-registry submodule not found at {}. \
         Run `just update-superchain-registry-submodule` from the repo root.",
        sr.display()
    );

    // (path-in-archive, bytes); sorted before tarring for a deterministic layout.
    let mut entries: Vec<(String, Vec<u8>)> = Vec::new();

    // Configs: TOML → JSON (the `no_std` runtime parses JSON, not TOML).
    let configs_dir = sr.join("superchain/configs");
    for toml_path in walk(&configs_dir, "toml") {
        let rel = toml_path.strip_prefix(&configs_dir).unwrap().to_str().unwrap();
        let json: serde_json::Value =
            toml::from_str(&fs::read_to_string(&toml_path).unwrap()).unwrap();
        let rel_json = format!("configs/{}.json", rel.strip_suffix(".toml").unwrap());
        entries.push((rel_json, serde_json::to_vec(&json).unwrap()));
    }

    // Genesis: dictionary-compressed `.json.zst` → plain zlib `.json.zz`. The SR
    // genesis carry `"config": null` (inconsistently populated), which would fail
    // `Genesis` deserialization, so strip it — the runtime re-derives the config
    // from the chain metadata. See superchain-registry#901.
    let dict = fs::read(sr.join("superchain/extra/dictionary")).unwrap();
    let genesis_dir = sr.join("superchain/extra/genesis");
    for zst_path in walk(&genesis_dir, "zst") {
        let rel = zst_path.strip_prefix(&genesis_dir).unwrap().to_str().unwrap();
        let raw = zstd_decompress_with_dict(&fs::read(&zst_path).unwrap(), &dict);
        let mut genesis: serde_json::Value = serde_json::from_slice(&raw).unwrap();
        if let Some(obj) = genesis.as_object_mut() {
            obj.remove("config");
        }
        let stripped = serde_json::to_vec(&genesis).unwrap();
        let zz = miniz_oxide::deflate::compress_to_vec_zlib(&stripped, 6);
        entries.push((format!("genesis/{}.zz", rel.strip_suffix(".zst").unwrap()), zz));
    }

    entries.sort_by(|a, b| a.0.cmp(&b.0));
    let tar_bytes = build_tar(&entries);
    fs::write(&tar_path, &tar_bytes).unwrap();
    let sha = sha256_hex(&tar_bytes);

    if refresh {
        // Update the committed pin + generated chain list.
        fs::write(&sha_path, format!("{sha}  {TAR_NAME}\n")).unwrap();
        write_chain_specs(&sr, &manifest_dir);
    } else {
        // Default-mode regeneration (the tar wasn't provided): the freshly built
        // archive must still match the committed pin, or the submodule has drifted.
        assert_eq!(
            sha, expected_sha,
            "regenerated res/superchain-configs.tar sha256 {sha} != committed \
             {expected_sha}; the submodule and the committed .sha256 disagree — run \
             `just sync-superchain-rust`."
        );
    }
}

/// Regenerates `src/superchain/chain_specs.rs` from `chainList.json`, sorted by
/// (environment, chain name). Chains without a genesis file in the archive are skipped.
fn write_chain_specs(sr: &Path, manifest_dir: &Path) {
    let list: serde_json::Value =
        serde_json::from_str(&fs::read_to_string(sr.join("chainList.json")).unwrap()).unwrap();
    let mut chains: Vec<(String, String)> = list
        .as_array()
        .unwrap()
        .iter()
        .map(|c| {
            let env = c["parent"]["chain"].as_str().unwrap().to_string();
            let name = c["identifier"].as_str().unwrap().split('/').nth(1).unwrap().to_string();
            (env, name)
        })
        .collect();
    chains.sort();

    let mut out = String::from(
        "// Generated by build.rs\nuse crate::create_superchain_specs;\n\ncreate_superchain_specs!(\n",
    );
    for (env, name) in &chains {
        if sr.join(format!("superchain/extra/genesis/{env}/{name}.json.zst")).exists() {
            out.push_str(&format!("    (\"{name}\", \"{env}\"),\n"));
        }
    }
    out.push_str(");\n");
    fs::write(manifest_dir.join("src/superchain/chain_specs.rs"), out).unwrap();
}

fn build_tar(entries: &[(String, Vec<u8>)]) -> Vec<u8> {
    let mut ar = tar::Builder::new(Vec::new());
    for (path, data) in entries {
        let mut header = tar::Header::new_gnu();
        header.set_size(data.len() as u64);
        header.set_mode(0o644);
        header.set_mtime(MTIME);
        header.set_uid(0);
        header.set_gid(0);
        header.set_entry_type(tar::EntryType::Regular);
        ar.append_data(&mut header, path, data.as_slice()).unwrap();
    }
    ar.into_inner().unwrap()
}

fn zstd_decompress_with_dict(compressed: &[u8], dict: &[u8]) -> Vec<u8> {
    let decoder =
        zstd::stream::read::Decoder::with_dictionary(std::io::Cursor::new(compressed), dict)
            .unwrap();
    let mut out = Vec::new();
    decoder.take(MAX_GENESIS_SIZE).read_to_end(&mut out).unwrap();
    out
}

fn sha256_hex(data: &[u8]) -> String {
    use sha2::{Digest, Sha256};
    Sha256::digest(data).iter().map(|b| format!("{b:02x}")).collect()
}

fn walk(dir: &Path, ext: &str) -> Vec<PathBuf> {
    fs::read_dir(dir)
        .unwrap()
        .flat_map(|entry| {
            let p = entry.unwrap().path();
            if p.is_dir() {
                walk(&p, ext)
            } else if p.extension().and_then(|s| s.to_str()) == Some(ext) {
                vec![p]
            } else {
                vec![]
            }
        })
        .collect()
}
