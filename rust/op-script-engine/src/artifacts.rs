//! Minimal forge-artifact loader: reads `<dir>/<file>/<contract>.json` and extracts
//! creation / deployed bytecode. Mirrors the subset of `op-chain-ops/foundry` that the
//! script engine needs (no source maps, no ABI wrangling).

use std::path::{Path, PathBuf};

use alloy_primitives::Bytes;

#[derive(Debug, thiserror::Error)]
pub enum ArtifactError {
    #[error("artifact not found: {0}")]
    NotFound(PathBuf),
    #[error("failed to read artifact {0}: {1}")]
    Io(PathBuf, std::io::Error),
    #[error("failed to parse artifact {0}: {1}")]
    Parse(PathBuf, serde_json::Error),
    #[error("artifact {0} missing field {1}")]
    MissingField(PathBuf, &'static str),
    #[error("bad artifact path spec {0:?}")]
    BadSpec(String),
}

/// A resolved forge artifact.
#[derive(Debug, Clone)]
pub struct Artifact {
    pub bytecode: Bytes,
    pub deployed_bytecode: Bytes,
}

/// Artifacts filesystem rooted at a forge `out/` directory.
#[derive(Debug, Clone)]
pub struct Artifacts {
    root: PathBuf,
}

impl Artifacts {
    pub fn new(root: impl Into<PathBuf>) -> Self {
        Self { root: root.into() }
    }

    /// Reads the artifact for `file`/`contract`, e.g. ("ScriptExample.s.sol", "ScriptExample").
    pub fn read(&self, file: &str, contract: &str) -> Result<Artifact, ArtifactError> {
        let path = self.root.join(file).join(format!("{contract}.json"));
        self.read_path(&path)
    }

    /// Resolves a `vm.getCode` / `vm.getDeployedCode` artifact-path argument to an artifact,
    /// byte-for-byte matching the Go host's `CheatCodesPrecompile.getArtifact`
    /// (`op-chain-ops/script/cheatcodes_external.go`):
    ///   - a `*.json` path resolves via the `forge-artifacts/`/`out/`-stripped dir + file stem;
    ///   - otherwise `"Foo"` -> file `Foo.sol` / contract `Foo`, and `"Foo.sol:Bar"` ->
    ///     file `Foo.sol` / contract `Bar`.
    pub fn read_spec(&self, input: &str) -> Result<Artifact, ArtifactError> {
        if let Some((name, contract)) = parse_artifact_path_input(input) {
            return self.read(&name, &contract);
        }
        // fetching by relative file path, or using a contract version, is not supported
        let (name, contract) = match input.split_once(':') {
            Some((f, c)) => (f.to_string(), c.to_string()),
            None => (format!("{input}.sol"), input.to_string()),
        };
        self.read(&name, &contract)
    }

    fn read_path(&self, path: &Path) -> Result<Artifact, ArtifactError> {
        if !path.exists() {
            return Err(ArtifactError::NotFound(path.to_path_buf()));
        }
        let data =
            std::fs::read(path).map_err(|e| ArtifactError::Io(path.to_path_buf(), e))?;
        let v: serde_json::Value = serde_json::from_slice(&data)
            .map_err(|e| ArtifactError::Parse(path.to_path_buf(), e))?;
        let bytecode = extract_object(&v, "bytecode")
            .ok_or_else(|| ArtifactError::MissingField(path.to_path_buf(), "bytecode.object"))?;
        let deployed = extract_object(&v, "deployedBytecode").ok_or_else(|| {
            ArtifactError::MissingField(path.to_path_buf(), "deployedBytecode.object")
        })?;
        Ok(Artifact { bytecode, deployed_bytecode: deployed })
    }
}

/// Mirrors the Go host's `parseArtifactPathInput`: resolves a `*.json` forge-artifact path into
/// its `(dir, contract)` pair, stripping a leading `forge-artifacts/` or `out/`. Returns `None`
/// for non-`.json` inputs (handled by the `file`/`file:contract` fallback).
fn parse_artifact_path_input(input: &str) -> Option<(String, String)> {
    let clean = clean_path(input.trim());
    let clean = clean.strip_prefix("./").unwrap_or(&clean);
    if !clean.ends_with(".json") {
        return None;
    }
    let clean = clean
        .strip_prefix("forge-artifacts/")
        .or_else(|| clean.strip_prefix("out/"))
        .unwrap_or(clean);
    let (dir, file) = match clean.rsplit_once('/') {
        Some((d, f)) => (d, f),
        None => ("", clean),
    };
    let contract = file.strip_suffix(".json").unwrap_or(file);
    if dir.is_empty() || contract.is_empty() {
        return None;
    }
    Some((dir.to_string(), contract.to_string()))
}

/// A minimal `path.Clean` sufficient for artifact-path inputs: collapses `//` and drops `.`
/// segments. Full lexical normalization is unnecessary for the forge-artifact paths we see.
fn clean_path(p: &str) -> String {
    let mut out: Vec<&str> = Vec::new();
    for seg in p.split('/') {
        if seg.is_empty() || seg == "." {
            continue;
        }
        out.push(seg);
    }
    let joined = out.join("/");
    if p.starts_with('/') { format!("/{joined}") } else { joined }
}

fn extract_object(v: &serde_json::Value, key: &str) -> Option<Bytes> {
    let s = v.get(key)?.get("object")?.as_str()?;
    let s = s.strip_prefix("0x").unwrap_or(s);
    let bytes = alloy_primitives::hex::decode(s).ok()?;
    Some(Bytes::from(bytes))
}
