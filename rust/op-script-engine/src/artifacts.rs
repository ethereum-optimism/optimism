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

    /// Reads an artifact from a `file:contract` or `file` spec, mirroring the forge
    /// `getDeployedCode("ScriptExample.s.sol:NonceGetter")` convention.
    pub fn read_spec(&self, spec: &str) -> Result<Artifact, ArtifactError> {
        let (file, contract) = match spec.split_once(':') {
            Some((f, c)) => (f.to_string(), c.to_string()),
            None => {
                // "Foo.sol" -> contract "Foo"
                let stem = Path::new(spec)
                    .file_name()
                    .and_then(|s| s.to_str())
                    .and_then(|s| s.split('.').next())
                    .ok_or_else(|| ArtifactError::BadSpec(spec.to_string()))?;
                (spec.to_string(), stem.to_string())
            }
        };
        self.read(&file, &contract)
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

fn extract_object(v: &serde_json::Value, key: &str) -> Option<Bytes> {
    let s = v.get(key)?.get("object")?.as_str()?;
    let s = s.strip_prefix("0x").unwrap_or(s);
    let bytes = alloy_primitives::hex::decode(s).ok()?;
    Some(Bytes::from(bytes))
}
