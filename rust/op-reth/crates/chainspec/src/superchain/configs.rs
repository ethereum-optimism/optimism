use crate::superchain::chain_metadata::{ChainMetadata, to_genesis_chain_config};
use alloc::{format, string::String};
use alloy_genesis::Genesis;
use miniz_oxide::inflate::decompress_to_vec_zlib_with_limit;

/// A genesis file can be up to 100MiB. This is a reasonable limit for the genesis file size.
const MAX_GENESIS_SIZE: usize = 100 * 1024 * 1024; // 100MiB

#[derive(Debug, thiserror::Error)]
pub(crate) enum SuperchainConfigError {
    #[error("Error deserializing JSON: {0}")]
    JsonError(#[from] serde_json::Error),
    #[error("No genesis bytes registered for `{0}-{1}`")]
    NoGenesis(String, String),
    #[error("No config JSON registered for `{0}-{1}`")]
    NoConfig(String, String),
    #[error("Error decompressing genesis: {0}")]
    DecompressError(String),
}

/// Reads the [`Genesis`] from the embedded op-superchain artifacts.
/// For example, `read_superchain_genesis("unichain", "mainnet")`.
pub(crate) fn read_superchain_genesis(
    name: &str,
    environment: &str,
) -> Result<Genesis, SuperchainConfigError> {
    let compressed = op_superchain::genesis_bytes(name, environment)
        .ok_or_else(|| SuperchainConfigError::NoGenesis(name.into(), environment.into()))?;
    let genesis_bytes = decompress_to_vec_zlib_with_limit(compressed, MAX_GENESIS_SIZE)
        .map_err(|e| SuperchainConfigError::DecompressError(format!("{e}")))?;

    let mut genesis: Genesis = serde_json::from_slice(&genesis_bytes)?;

    // The "config" field is stripped from the genesis file because it is not always populated.
    // For that reason, we read the config from the chain metadata file.
    // See: https://github.com/ethereum-optimism/superchain-registry/issues/901
    genesis.config = to_genesis_chain_config(&read_superchain_metadata(name, environment)?);

    Ok(genesis)
}

/// Reads the [`ChainMetadata`] from the embedded op-superchain artifacts.
/// For example, `read_superchain_config("unichain", "mainnet")`.
fn read_superchain_metadata(
    name: &str,
    environment: &str,
) -> Result<ChainMetadata, SuperchainConfigError> {
    let json = op_superchain::config_str(name, environment)
        .ok_or_else(|| SuperchainConfigError::NoConfig(name.into(), environment.into()))?;
    let chain_config: ChainMetadata = serde_json::from_str(json)?;
    Ok(chain_config)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{SUPPORTED_CHAINS, generated_chain_value_parser, superchain::Superchain};
    use alloy_chains::NamedChain;
    use alloy_op_hardforks::{
        BASE_MAINNET_CANYON_TIMESTAMP, BASE_MAINNET_ECOTONE_TIMESTAMP,
        BASE_MAINNET_ISTHMUS_TIMESTAMP, BASE_MAINNET_JOVIAN_TIMESTAMP,
        BASE_SEPOLIA_CANYON_TIMESTAMP, BASE_SEPOLIA_ECOTONE_TIMESTAMP,
        BASE_SEPOLIA_ISTHMUS_TIMESTAMP, BASE_SEPOLIA_JOVIAN_TIMESTAMP, OP_MAINNET_CANYON_TIMESTAMP,
        OP_MAINNET_ECOTONE_TIMESTAMP, OP_MAINNET_ISTHMUS_TIMESTAMP, OP_MAINNET_JOVIAN_TIMESTAMP,
        OP_SEPOLIA_CANYON_TIMESTAMP, OP_SEPOLIA_ECOTONE_TIMESTAMP, OP_SEPOLIA_ISTHMUS_TIMESTAMP,
        OP_SEPOLIA_JOVIAN_TIMESTAMP, OpHardfork,
    };
    use reth_optimism_primitives::L2_TO_L1_MESSAGE_PASSER_ADDRESS;

    #[test]
    fn test_read_superchain_genesis() {
        let genesis = read_superchain_genesis("unichain", "mainnet").unwrap();
        assert_eq!(genesis.config.chain_id, 130);
        assert_eq!(genesis.timestamp, 1730748359);
        assert!(genesis.alloc.contains_key(&L2_TO_L1_MESSAGE_PASSER_ADDRESS));
    }

    #[test]
    fn test_read_superchain_genesis_with_workaround() {
        let genesis = read_superchain_genesis("funki", "mainnet").unwrap();
        assert_eq!(genesis.config.chain_id, 33979);
        assert_eq!(genesis.timestamp, 1721211095);
        assert!(genesis.alloc.contains_key(&L2_TO_L1_MESSAGE_PASSER_ADDRESS));
    }

    #[test]
    fn test_read_superchain_metadata() {
        let chain_config = read_superchain_metadata("funki", "mainnet").unwrap();
        assert_eq!(chain_config.chain_id, 33979);
    }

    #[test]
    fn test_read_all_supported_chains() {
        // Every chain listed by op_superchain must have both config and genesis bytes.
        for (name, env) in op_superchain::supported_chains() {
            assert!(
                op_superchain::config_str(name, env).is_some(),
                "config missing for {name}-{env}"
            );
            assert!(
                op_superchain::genesis_bytes(name, env).is_some(),
                "genesis missing for {name}-{env}"
            );
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

    #[test]
    fn test_hardfork_timestamps() {
        for &chain in SUPPORTED_CHAINS {
            let metadata = generated_chain_value_parser(chain).unwrap();

            match metadata.chain().named() {
                Some(NamedChain::Optimism) => {
                    assert_eq!(
                        metadata.hardforks.get(OpHardfork::Jovian).unwrap().as_timestamp().unwrap(),
                        OP_MAINNET_JOVIAN_TIMESTAMP
                    );

                    assert_eq!(
                        metadata
                            .hardforks
                            .get(OpHardfork::Isthmus)
                            .unwrap()
                            .as_timestamp()
                            .unwrap(),
                        OP_MAINNET_ISTHMUS_TIMESTAMP
                    );

                    assert_eq!(
                        metadata.hardforks.get(OpHardfork::Canyon).unwrap().as_timestamp().unwrap(),
                        OP_MAINNET_CANYON_TIMESTAMP
                    );

                    assert_eq!(
                        metadata
                            .hardforks
                            .get(OpHardfork::Ecotone)
                            .unwrap()
                            .as_timestamp()
                            .unwrap(),
                        OP_MAINNET_ECOTONE_TIMESTAMP
                    );
                }
                Some(NamedChain::OptimismSepolia) => {
                    assert_eq!(
                        metadata.hardforks.get(OpHardfork::Jovian).unwrap().as_timestamp().unwrap(),
                        OP_SEPOLIA_JOVIAN_TIMESTAMP
                    );

                    assert_eq!(
                        metadata
                            .hardforks
                            .get(OpHardfork::Isthmus)
                            .unwrap()
                            .as_timestamp()
                            .unwrap(),
                        OP_SEPOLIA_ISTHMUS_TIMESTAMP
                    );

                    assert_eq!(
                        metadata.hardforks.get(OpHardfork::Canyon).unwrap().as_timestamp().unwrap(),
                        OP_SEPOLIA_CANYON_TIMESTAMP
                    );

                    assert_eq!(
                        metadata
                            .hardforks
                            .get(OpHardfork::Ecotone)
                            .unwrap()
                            .as_timestamp()
                            .unwrap(),
                        OP_SEPOLIA_ECOTONE_TIMESTAMP
                    );
                }
                Some(NamedChain::Base) => {
                    assert_eq!(
                        metadata.hardforks.get(OpHardfork::Jovian).unwrap().as_timestamp().unwrap(),
                        BASE_MAINNET_JOVIAN_TIMESTAMP
                    );

                    assert_eq!(
                        metadata
                            .hardforks
                            .get(OpHardfork::Isthmus)
                            .unwrap()
                            .as_timestamp()
                            .unwrap(),
                        BASE_MAINNET_ISTHMUS_TIMESTAMP
                    );

                    assert_eq!(
                        metadata.hardforks.get(OpHardfork::Canyon).unwrap().as_timestamp().unwrap(),
                        BASE_MAINNET_CANYON_TIMESTAMP
                    );

                    assert_eq!(
                        metadata
                            .hardforks
                            .get(OpHardfork::Ecotone)
                            .unwrap()
                            .as_timestamp()
                            .unwrap(),
                        BASE_MAINNET_ECOTONE_TIMESTAMP
                    );
                }
                Some(NamedChain::BaseSepolia) => {
                    assert_eq!(
                        metadata.hardforks.get(OpHardfork::Jovian).unwrap().as_timestamp().unwrap(),
                        BASE_SEPOLIA_JOVIAN_TIMESTAMP
                    );

                    assert_eq!(
                        metadata
                            .hardforks
                            .get(OpHardfork::Isthmus)
                            .unwrap()
                            .as_timestamp()
                            .unwrap(),
                        BASE_SEPOLIA_ISTHMUS_TIMESTAMP
                    );

                    assert_eq!(
                        metadata.hardforks.get(OpHardfork::Canyon).unwrap().as_timestamp().unwrap(),
                        BASE_SEPOLIA_CANYON_TIMESTAMP
                    );

                    assert_eq!(
                        metadata
                            .hardforks
                            .get(OpHardfork::Ecotone)
                            .unwrap()
                            .as_timestamp()
                            .unwrap(),
                        BASE_SEPOLIA_ECOTONE_TIMESTAMP
                    );
                }
                _ => {}
            }
        }
    }
}
