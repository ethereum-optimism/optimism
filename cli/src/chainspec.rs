use reth_cli::chainspec::{parse_genesis, ChainSpecParser};
use reth_optimism_chainspec::{generated_chain_value_parser, OpChainSpec, SUPPORTED_CHAINS};
use std::sync::Arc;

/// Optimism chain specification parser.
#[derive(Debug, Clone, Default)]
#[non_exhaustive]
pub struct OpChainSpecParser;

impl ChainSpecParser for OpChainSpecParser {
    type ChainSpec = OpChainSpec;

    const SUPPORTED_CHAINS: &'static [&'static str] = SUPPORTED_CHAINS;

    fn parse(s: &str) -> eyre::Result<Arc<Self::ChainSpec>> {
        chain_value_parser(s)
    }
}

/// Clap value parser for [`OpChainSpec`]s.
///
/// The value parser matches either a known chain, the path
/// to a json file, or a json formatted string in-memory. The json needs to be a Genesis struct.
pub fn chain_value_parser(s: &str) -> eyre::Result<Arc<OpChainSpec>, eyre::Error> {
    if let Some(op_chain_spec) = generated_chain_value_parser(s) {
        Ok(op_chain_spec)
    } else {
        Ok(Arc::new(parse_genesis(s)?.into()))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_known_chain_spec() {
        for &chain in OpChainSpecParser::SUPPORTED_CHAINS {
            assert!(
                <OpChainSpecParser as ChainSpecParser>::parse(chain).is_ok(),
                "Failed to parse {chain}"
            );
        }
    }
}
