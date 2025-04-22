use alloc::{string::String, sync::Arc, vec::Vec};
use core::convert::Into;
use reth_primitives_traits::sync::LazyLock;
use serde::Deserialize;

/// Contains the info about an available chain genesis config e.g. "unichain" and "mainnet"
#[derive(Debug, Clone, Deserialize)]
pub struct AvailableSuperchain {
    /// The name of the superchain e.g. "unichain"
    pub name: String,
    /// The environment of the superchain e.g. "mainnet"
    pub environment: String,
}

/// All available superchain genesis configs
pub static AVAILABLE_CHAINS: LazyLock<Arc<Vec<AvailableSuperchain>>> = LazyLock::new(|| {
    serde_json::from_str::<Vec<AvailableSuperchain>>(include_str!(
        "../../res/available-chains.json"
    ))
    .expect("Can't deserialize available-chains.json")
    .into()
});

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn test_deserialize_available_superchain() {
        let s = r#"
        {
          "name": "funki",
          "environment": "mainnet"
        }
        "#;
        let available_superchain: AvailableSuperchain = serde_json::from_str(s).unwrap();
        assert_eq!(available_superchain.name, "funki");
        assert_eq!(available_superchain.environment, "mainnet");
    }

    #[test]
    fn test_read_available_superchain() {
        let available_superchain = AVAILABLE_CHAINS.iter().find(|&chain| chain.name == "funki");
        assert_eq!(available_superchain.unwrap().name, "funki");
        assert_eq!(available_superchain.unwrap().environment, "mainnet");
    }
}
