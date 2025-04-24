use crate::superchain::chain_metadata::{to_genesis_chain_config, ChainMetadata};
use alloc::{
    format,
    string::{String, ToString},
    vec::Vec,
};
use alloy_genesis::Genesis;
use miniz_oxide::inflate::decompress_to_vec_zlib_with_limit;
use tar_no_std::{CorruptDataError, TarArchiveRef};

/// A genesis file can be up to 10MiB. This is a reasonable limit for the genesis file size.
const MAX_GENESIS_SIZE: usize = 16 * 1024 * 1024; // 16MiB

/// The tar file contains the chain configs and genesis files for all chains.
const SUPER_CHAIN_CONFIGS_TAR_BYTES: &[u8] = include_bytes!("../../res/superchain-configs.tar");

#[derive(Debug, thiserror::Error)]
pub(crate) enum SuperchainConfigError {
    #[error("Error reading archive due to corrupt data: {0}")]
    CorruptDataError(CorruptDataError),
    #[error("Error converting bytes to UTF-8 String: {0}")]
    FromUtf8Error(#[from] alloc::string::FromUtf8Error),
    #[error("Error reading file: {0}")]
    Utf8Error(#[from] core::str::Utf8Error),
    #[error("Error deserializing JSON: {0}")]
    JsonError(#[from] serde_json::Error),
    #[error("File {0} not found in archive")]
    FileNotFound(String),
    #[error("Error decompressing file: {0}")]
    DecompressError(String),
}

/// Reads the [`Genesis`] from the superchain config tar file for a superchain.
/// For example, `read_genesis_from_superchain_config("unichain", "mainnet")`.
pub(crate) fn read_superchain_genesis(
    name: &str,
    environment: &str,
) -> Result<Genesis, SuperchainConfigError> {
    // Open the archive.
    let archive = TarArchiveRef::new(SUPER_CHAIN_CONFIGS_TAR_BYTES)
        .map_err(SuperchainConfigError::CorruptDataError)?;
    // Read and decompress the genesis file.
    let compressed_genesis_file =
        read_file(&archive, &format!("genesis/{environment}/{name}.json.zz"))?;
    let genesis_file =
        decompress_to_vec_zlib_with_limit(&compressed_genesis_file, MAX_GENESIS_SIZE)
            .map_err(|e| SuperchainConfigError::DecompressError(format!("{e}")))?;

    // Load the genesis file.
    let mut genesis: Genesis = serde_json::from_slice(&genesis_file)?;

    // The "config" field is stripped (see fetch_superchain_config.sh) from the genesis file
    // because it is not always populated. For that reason, we read the config from the chain
    // metadata file. See: https://github.com/ethereum-optimism/superchain-registry/issues/901
    genesis.config =
        to_genesis_chain_config(&read_superchain_metadata(name, environment, &archive)?);

    Ok(genesis)
}

/// Reads the [`ChainMetadata`] from the superchain config tar file for a superchain.
/// For example, `read_superchain_config("unichain", "mainnet")`.
fn read_superchain_metadata(
    name: &str,
    environment: &str,
    archive: &TarArchiveRef<'_>,
) -> Result<ChainMetadata, SuperchainConfigError> {
    let config_file = read_file(archive, &format!("configs/{environment}/{name}.json"))?;
    let config_content = String::from_utf8(config_file)?;
    let chain_config: ChainMetadata = serde_json::from_str(&config_content)?;
    Ok(chain_config)
}

/// Reads a file from the tar archive. The file path is relative to the root of the tar archive.
fn read_file(
    archive: &TarArchiveRef<'_>,
    file_path: &str,
) -> Result<Vec<u8>, SuperchainConfigError> {
    for entry in archive.entries() {
        if entry.filename().as_str()? == file_path {
            return Ok(entry.data().to_vec())
        }
    }
    Err(SuperchainConfigError::FileNotFound(file_path.to_string()))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::superchain::Superchain;
    use reth_optimism_primitives::ADDRESS_L2_TO_L1_MESSAGE_PASSER;
    use tar_no_std::TarArchiveRef;

    #[test]
    fn test_read_superchain_genesis() {
        let genesis = read_superchain_genesis("unichain", "mainnet").unwrap();
        assert_eq!(genesis.config.chain_id, 130);
        assert_eq!(genesis.timestamp, 1730748359);
        assert!(genesis.alloc.contains_key(&ADDRESS_L2_TO_L1_MESSAGE_PASSER));
    }

    #[test]
    fn test_read_superchain_genesis_with_workaround() {
        let genesis = read_superchain_genesis("funki", "mainnet").unwrap();
        assert_eq!(genesis.config.chain_id, 33979);
        assert_eq!(genesis.timestamp, 1721211095);
        assert!(genesis.alloc.contains_key(&ADDRESS_L2_TO_L1_MESSAGE_PASSER));
    }

    #[test]
    fn test_read_superchain_metadata() {
        let archive = TarArchiveRef::new(SUPER_CHAIN_CONFIGS_TAR_BYTES).unwrap();
        let chain_config = read_superchain_metadata("funki", "mainnet", &archive).unwrap();
        assert_eq!(chain_config.chain_id, 33979);
    }

    #[test]
    fn test_read_all_genesis_files() {
        let archive = TarArchiveRef::new(SUPER_CHAIN_CONFIGS_TAR_BYTES).unwrap();
        // Check that all genesis files can be read without errors.
        for entry in archive.entries() {
            let filename = entry
                .filename()
                .as_str()
                .unwrap()
                .split('/')
                .map(|s| s.to_string())
                .collect::<Vec<String>>();
            if filename.first().unwrap().ne(&"genesis") {
                continue
            }
            read_superchain_metadata(
                &filename.get(2).unwrap().replace(".json.zz", ""),
                filename.get(1).unwrap(),
                &archive,
            )
            .unwrap();
        }
    }

    #[test]
    fn test_genesis_exists_for_all_available_chains() {
        for &chain in Superchain::ALL {
            let genesis = read_superchain_genesis(chain.name(), chain.environment());
            assert!(
                genesis.is_ok(),
                "Genesis not found for chain: {}-{}",
                chain.name(),
                chain.environment()
            );
        }
    }
}
