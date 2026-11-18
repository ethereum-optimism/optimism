//! Module containing the chain config.

use alloy_chains::NamedChain;

/// OP Mainnet chain ID.
pub const OP_MAINNET_CHAIN_ID: u64 = 10;

/// OP Sepolia chain ID.
pub const OP_SEPOLIA_CHAIN_ID: u64 = 11155420;

/// Chain ids of the L1s whose [`L1ChainConfig`] `kona-registry` embeds.
pub const REGISTRY_L1_CHAIN_IDS: &[u64] =
    &[NamedChain::Mainnet as u64, NamedChain::Sepolia as u64, NamedChain::Holesky as u64];

mod addresses;
pub use addresses::AddressList;

mod config;
pub use config::{ChainConfig, L1ChainConfig};

mod altda;
pub use altda::AltDAConfig;

mod hardfork;
pub use hardfork::HardForkConfig;

mod roles;
pub use roles::Roles;
