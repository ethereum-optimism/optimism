//! Minimal forge-artifact loader: reads `<dir>/<file>/<contract>.json` and extracts
//! creation / deployed bytecode. Mirrors the subset of `op-chain-ops/foundry` that the
//! script engine needs (no source maps, no ABI wrangling).

use std::path::{Path, PathBuf};

use alloy_primitives::Bytes;

/// Reasons a forge artifact could not be loaded.
#[derive(Debug, thiserror::Error)]
pub enum ArtifactError {
    /// No file exists at the resolved artifact path.
    #[error("artifact not found: {0}")]
    NotFound(PathBuf),
    /// The artifact file could not be read.
    #[error("failed to read artifact {0}: {1}")]
    Io(PathBuf, std::io::Error),
    /// The artifact JSON could not be parsed.
    #[error("failed to parse artifact {0}: {1}")]
    Parse(PathBuf, serde_json::Error),
    /// The artifact JSON is missing a required field (e.g. bytecode object).
    #[error("artifact {0} missing field {1}")]
    MissingField(PathBuf, &'static str),
    /// The caller-supplied artifact path spec (`"File.sol:Contract"` etc.) is malformed.
    #[error("bad artifact path spec {0:?}")]
    BadSpec(String),
}

/// A resolved forge artifact.
#[derive(Debug, Clone)]
pub struct Artifact {
    /// Creation (constructor) bytecode, used for CREATE.
    pub bytecode: Bytes,
    /// Runtime bytecode as deployed on chain, used to etch code directly.
    pub deployed_bytecode: Bytes,
}

/// Artifacts filesystem rooted at a forge `out/` directory.
#[derive(Debug, Clone)]
pub struct Artifacts {
    root: PathBuf,
}

impl Artifacts {
    /// Roots an artifact loader at a forge `out/` directory.
    pub fn new(root: impl Into<PathBuf>) -> Self {
        Self { root: root.into() }
    }

    /// Reads the artifact for `file`/`contract`, e.g. ("ScriptExample.s.sol", "`ScriptExample`").
    pub fn read(&self, file: &str, contract: &str) -> Result<Artifact, ArtifactError> {
        // Jail the lookup to `root`, mirroring the Go host's `os.DirFS` + `fs.ValidPath`: reject
        // absolute paths and `..` traversal in either component so a script cannot read files
        // outside the artifacts directory.
        let rel = format!("{file}/{contract}.json");
        if !is_local_path(&rel) {
            return Err(ArtifactError::BadSpec(rel));
        }
        let path = self.root.join(file).join(format!("{contract}.json"));
        self.read_path(&path)
    }

    /// Resolves a `vm.getCode` / `vm.getDeployedCode` artifact-path argument to an artifact,
    /// byte-for-byte matching the Go host's `CheatCodesPrecompile.getArtifact`
    /// (`op-chain-ops/script/cheatcodes_external.go`):
    ///   - a `*.json` path resolves via the `forge-artifacts/`/`out/`-stripped dir + file stem;
    ///   - otherwise `"Foo"` -> file `Foo.sol` / contract `Foo`, and `"Foo.sol:Bar"` -> file
    ///     `Foo.sol` / contract `Bar`.
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
        let data = std::fs::read(path).map_err(|e| ArtifactError::Io(path.to_path_buf(), e))?;
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

/// Mirrors Go `fs.ValidPath`: true when `p` is a relative, forward-slash path with no empty,
/// `.`, or `..` segments — i.e. one that stays within the artifacts root. Backslashes are rejected
/// defensively so a Windows-style path cannot smuggle a separator past the segment check.
fn is_local_path(p: &str) -> bool {
    if p.is_empty() || p.starts_with('/') || p.contains('\\') {
        return false;
    }
    p.split('/').all(|seg| !seg.is_empty() && seg != "." && seg != "..")
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn read_rejects_path_traversal() {
        let a = Artifacts::new("/tmp/artifacts-root");
        // Absolute or `..`-escaping components in either argument must be rejected.
        assert!(matches!(a.read("/etc", "passwd"), Err(ArtifactError::BadSpec(_))));
        assert!(matches!(a.read("../../etc", "passwd"), Err(ArtifactError::BadSpec(_))));
        assert!(matches!(a.read("Foo.sol", "../../../etc/passwd"), Err(ArtifactError::BadSpec(_))));
        // A well-formed relative lookup passes the jail (NotFound => the jail allowed it through).
        assert!(matches!(a.read("Foo.sol", "Foo"), Err(ArtifactError::NotFound(_))));
    }

    #[test]
    fn is_local_path_matches_valid_path_semantics() {
        assert!(is_local_path("Foo.sol/Foo.json"));
        assert!(is_local_path("src/L1/Foo.sol/Foo.json"));
        assert!(!is_local_path("/abs/Foo.json"));
        assert!(!is_local_path("../escape/Foo.json"));
        assert!(!is_local_path("a/../b/Foo.json"));
        assert!(!is_local_path("a\\b.json"));
    }
}
