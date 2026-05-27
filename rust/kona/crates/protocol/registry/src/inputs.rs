//! Compile-time data sources for the registry.
//!
//! Default path: forward `op_superchain::*` constants. When `KONA_CUSTOM_CONFIGS=true`
//! is set during the build, `build.rs` writes merged outputs (op-superchain base +
//! the custom-configs overlay) into `$OUT_DIR` and emits a `kona_custom_configs`
//! cfg; the conditional `include_str!`s below switch to that material.

#[cfg(kona_custom_configs)]
mod imp {
    pub(crate) const CHAIN_LIST_JSON: &str =
        include_str!(concat!(env!("OUT_DIR"), "/chainList.json"));
    pub(crate) const SUPERCHAINS_JSON: &str =
        include_str!(concat!(env!("OUT_DIR"), "/configs.json"));
    pub(crate) const DEPSETS_JSON: &str = include_str!(concat!(env!("OUT_DIR"), "/depsets.json"));
}

#[cfg(not(kona_custom_configs))]
mod imp {
    pub(crate) const CHAIN_LIST_JSON: &str = op_superchain::CHAIN_LIST_JSON;
    pub(crate) const SUPERCHAINS_JSON: &str = op_superchain::SUPERCHAINS_JSON;
    pub(crate) const DEPSETS_JSON: &str = op_superchain::DEPSETS_JSON;
}

pub(crate) use imp::{CHAIN_LIST_JSON, DEPSETS_JSON, SUPERCHAINS_JSON};
