//! Producer + readers for the OP Stack's committed superchain-registry zip.
//!
//! This crate ships a single committed binary at
//! `rust/op-superchain/data/superchain-configs.zip`. The same bytes also live
//! at `op-core/superchain/superchain-configs.zip` for Go's `//go:embed`. Both
//! copies are produced by the `regenerate` binary in this crate from the
//! `packages/contracts-bedrock/lib/superchain-registry` submodule.
//!
//! The zip contains, mirroring the OP Stack's existing on-disk layout:
//! - `COMMIT` — the superchain-registry commit hash, with trailing newline.
//! - `dictionary` — the zstd dictionary used for the per-chain genesis files.
//! - `chains.json` — `chain_id` → `{ name, network }` index used by Go.
//! - `chainList.json` — verbatim from the submodule (kona consumes this).
//! - `superchains.json` — aggregate `Superchains` JSON for kona's eager parse.
//! - `depsets.json` — `Vec<DependencySet>` aggregated from each chain's `[interop]` block.
//! - `configs/<env>/<name>.toml` + `configs/<env>/superchain.toml` — raw TOML.
//! - `genesis/<env>/<name>.json.zst` — zstd-compressed with the bundled dict.
//!
//! Consumers wire in via:
//! - **Go (`op-core/superchain`)**: `//go:embed superchain-configs.zip`, reads via `archive/zip` as
//!   `fs.FS` — same code path as before this crate existed.
//! - **kona-registry**: build.rs uses [`read_chain_list_json`], [`read_superchains_json`],
//!   [`read_depsets_json`] to materialise the three aggregate JSON files into `OUT_DIR`, then
//!   `include_str!`s them.
//! - **reth-optimism-chainspec**: build.rs uses [`supported_chains`], [`read_chain_config_toml`]
//!   (and a TOML→JSON conversion), plus [`read_chain_genesis_compressed`] + [`read_dictionary`] to
//!   materialise per-chain artifacts in `OUT_DIR`. Runtime decompresses via the `zstd` crate.

use std::io::Read;

/// The committed superchain-registry zip embedded into this crate.
pub const SUPERCHAIN_CONFIGS_ZIP: &[u8] = include_bytes!("../data/superchain-configs.zip");

/// Errors returned by the reader helpers.
#[derive(Debug, thiserror::Error)]
pub enum Error {
    /// The zip header could not be parsed.
    #[error("opening embedded superchain zip: {0}")]
    Zip(#[from] zip::result::ZipError),
    /// I/O failure while reading an entry.
    #[error("reading entry {entry}: {source}")]
    Read {
        /// The entry path being read.
        entry: String,
        /// The underlying I/O error.
        #[source]
        source: std::io::Error,
    },
    /// The bytes in the entry were not valid UTF-8.
    #[error("entry {entry} is not valid UTF-8: {source}")]
    Utf8 {
        /// The entry path being read.
        entry: String,
        /// The underlying UTF-8 error.
        #[source]
        source: std::string::FromUtf8Error,
    },
}

fn open() -> Result<zip::ZipArchive<std::io::Cursor<&'static [u8]>>, Error> {
    Ok(zip::ZipArchive::new(std::io::Cursor::new(SUPERCHAIN_CONFIGS_ZIP))?)
}

fn read_entry(name: &str) -> Result<Option<Vec<u8>>, Error> {
    let mut archive = open()?;
    let mut entry = match archive.by_name(name) {
        Ok(entry) => entry,
        Err(zip::result::ZipError::FileNotFound) => return Ok(None),
        Err(other) => return Err(other.into()),
    };
    let mut buf = Vec::with_capacity(entry.size() as usize);
    entry.read_to_end(&mut buf).map_err(|source| Error::Read { entry: name.into(), source })?;
    Ok(Some(buf))
}

fn read_entry_string(name: &str) -> Result<Option<String>, Error> {
    let Some(bytes) = read_entry(name)? else { return Ok(None) };
    Ok(Some(String::from_utf8(bytes).map_err(|source| Error::Utf8 { entry: name.into(), source })?))
}

/// Returns the contents of the zip's `COMMIT` entry (whitespace-trimmed).
pub fn embedded_commit() -> Result<String, Error> {
    Ok(read_entry_string("COMMIT")?.expect("zip is missing COMMIT entry").trim().to_string())
}

/// Returns the zstd dictionary bundled in the zip.
pub fn read_dictionary() -> Result<Vec<u8>, Error> {
    Ok(read_entry("dictionary")?.expect("zip is missing dictionary entry"))
}

/// Returns `chainList.json` verbatim (consumed by kona-registry).
pub fn read_chain_list_json() -> Result<String, Error> {
    Ok(read_entry_string("chainList.json")?.expect("zip is missing chainList.json"))
}

/// Returns the aggregate `superchains.json` consumed by kona-registry.
pub fn read_superchains_json() -> Result<String, Error> {
    Ok(read_entry_string("superchains.json")?.expect("zip is missing superchains.json"))
}

/// Returns the aggregated dependency-set JSON (`Vec<DependencySet>`) consumed
/// by kona-registry.
pub fn read_depsets_json() -> Result<String, Error> {
    Ok(read_entry_string("depsets.json")?.expect("zip is missing depsets.json"))
}

/// Returns the `chains.json` index (`chain_id` → `{name, network}`) used by
/// the Go side and the regenerate binary.
pub fn read_chains_json() -> Result<String, Error> {
    Ok(read_entry_string("chains.json")?.expect("zip is missing chains.json"))
}

/// Returns the per-chain TOML config (the raw TOML from the submodule).
pub fn read_chain_config_toml(name: &str, environment: &str) -> Result<Option<String>, Error> {
    read_entry_string(&format!("configs/{environment}/{name}.toml"))
}

/// Returns the per-superchain TOML config (the raw `superchain.toml` from the
/// submodule), if present for the given environment.
pub fn read_superchain_config_toml(environment: &str) -> Result<Option<String>, Error> {
    read_entry_string(&format!("configs/{environment}/superchain.toml"))
}

/// Returns the per-chain zstd-compressed genesis bytes (uses the bundled
/// dictionary returned by [`read_dictionary`]).
pub fn read_chain_genesis_compressed(
    name: &str,
    environment: &str,
) -> Result<Option<Vec<u8>>, Error> {
    read_entry(&format!("genesis/{environment}/{name}.json.zst"))
}

/// Returns the per-chain genesis JSON, decompressed using the bundled
/// dictionary. Convenience wrapper over [`read_chain_genesis_compressed`] +
/// [`read_dictionary`].
pub fn read_chain_genesis_decompressed(
    name: &str,
    environment: &str,
) -> Result<Option<Vec<u8>>, Error> {
    let Some(compressed) = read_chain_genesis_compressed(name, environment)? else {
        return Ok(None);
    };
    let dict = read_dictionary()?;
    let mut decoder =
        zstd::stream::Decoder::with_dictionary(&compressed[..], &dict).map_err(|source| {
            Error::Read { entry: format!("genesis/{environment}/{name}.json.zst"), source }
        })?;
    let mut out = Vec::new();
    decoder.read_to_end(&mut out).map_err(|source| Error::Read {
        entry: format!("genesis/{environment}/{name}.json.zst"),
        source,
    })?;
    Ok(Some(out))
}

/// Returns the `[(name, environment)]` table from `chains.json`, sorted for
/// stable iteration.
pub fn supported_chains() -> Result<Vec<(String, String)>, Error> {
    #[derive(serde::Deserialize)]
    struct Entry {
        name: String,
        network: String,
    }
    let raw = read_chains_json()?;
    let map: std::collections::BTreeMap<String, Entry> =
        serde_json::from_str(&raw).expect("chains.json deserializes");
    let mut out: Vec<(String, String)> = map.into_values().map(|e| (e.name, e.network)).collect();
    out.sort();
    Ok(out)
}
