//! Directory Manager downloads and manages files from the op-superchain-registry

use eyre::Context;
use reth_fs_util as fs;
use reth_fs_util::Result;
use serde_json::Value;
use std::path::{Path, PathBuf};
use tracing::{debug, trace};
use zstd::{dict::DecoderDictionary, stream::read::Decoder};

/// Directory manager that handles caching and downloading of genesis files
#[derive(Debug)]
pub struct SuperChainRegistryManager {
    base_path: PathBuf,
}

impl SuperChainRegistryManager {
    const DICT_URL: &'static str = "https://raw.githubusercontent.com/ethereum-optimism/superchain-registry/main/superchain/extra/dictionary";
    const GENESIS_BASE_URL: &'static str = "https://raw.githubusercontent.com/ethereum-optimism/superchain-registry/main/superchain/extra/genesis";

    /// Create a new registry manager with the given base path
    pub fn new(base_path: impl Into<PathBuf>) -> Result<Self> {
        let base_path = base_path.into();
        fs::create_dir_all(&base_path)?;
        Ok(Self { base_path })
    }

    /// Get the path to the dictionary file
    pub fn dictionary_path(&self) -> PathBuf {
        self.base_path.join("dictionary")
    }

    /// Get the path to a genesis file for the given network (`mainnet`, `base`).
    pub fn genesis_path(&self, network_type: &str, network: &str) -> PathBuf {
        self.base_path.join(network_type).join(format!("{network}.json.zst"))
    }

    /// Read file from the given path
    fn read_file(&self, path: &Path) -> Result<Vec<u8>> {
        fs::read(path)
    }

    /// Save data to the given path
    fn save_file(&self, path: &Path, data: &[u8]) -> Result<()> {
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent)?;
        }
        fs::write(path, data)
    }

    /// Download a file from the given URL
    fn download_file(&self, url: &str, path: &Path) -> eyre::Result<Vec<u8>> {
        if path.exists() {
            debug!(target: "reth::cli", path = ?path.display() ,"Reading from cache");
            return Ok(self.read_file(path)?);
        }

        trace!(target: "reth::cli", url = ?url ,"Downloading from URL");
        let response = reqwest::blocking::get(url).context("Failed to download file")?;

        if !response.status().is_success() {
            eyre::bail!("Failed to download: Status {}", response.status());
        }

        let bytes = response.bytes()?.to_vec();
        self.save_file(path, &bytes)?;

        Ok(bytes)
    }

    /// Download and update the dictionary
    fn update_dictionary(&self) -> eyre::Result<Vec<u8>> {
        let path = self.dictionary_path();
        self.download_file(Self::DICT_URL, &path)
    }

    /// Get genesis data for a network, downloading it if necessary
    pub fn get_genesis(&self, network_type: &str, network: &str) -> eyre::Result<Value> {
        let dict_bytes = self.update_dictionary()?;
        trace!(target: "reth::cli", bytes = ?dict_bytes.len(),"Got dictionary");

        let dictionary = DecoderDictionary::copy(&dict_bytes);

        let url = format!("{}/{}/{}.json.zst", Self::GENESIS_BASE_URL, network_type, network);
        let path = self.genesis_path(network_type, network);

        let compressed_bytes = self.download_file(&url, &path)?;
        trace!(target: "reth::cli", bytes = ?compressed_bytes.len(),"Got genesis file");

        let decoder = Decoder::with_prepared_dictionary(&compressed_bytes[..], &dictionary)
            .context("Failed to create decoder with dictionary")?;

        let json: Value = serde_json::from_reader(decoder)
            .with_context(|| format!("Failed to parse JSON: {path:?}"))?;

        Ok(json)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use eyre::Result;

    #[test]
    fn test_directory_manager() -> Result<()> {
        let dir = tempfile::tempdir()?;
        // Create a temporary directory for testing
        let manager = SuperChainRegistryManager::new(dir.path())?;

        assert!(!manager.genesis_path("mainnet", "base").exists());
        // Test downloading genesis data
        let json_data = manager.get_genesis("mainnet", "base")?;
        assert!(json_data.is_object(), "Parsed JSON should be an object");

        assert!(manager.genesis_path("mainnet", "base").exists());

        // Test using cached data
        let cached_json_data = manager.get_genesis("mainnet", "base")?;
        assert!(cached_json_data.is_object(), "Cached JSON should be an object");

        Ok(())
    }
}
