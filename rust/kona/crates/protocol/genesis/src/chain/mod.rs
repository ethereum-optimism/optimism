//! Module containing the chain config.

/// OP Mainnet chain ID.
pub const OP_MAINNET_CHAIN_ID: u64 = 10;

/// OP Sepolia chain ID.
pub const OP_SEPOLIA_CHAIN_ID: u64 = 11155420;

/// Base Mainnet chain ID.
pub const BASE_MAINNET_CHAIN_ID: u64 = 8453;

/// Base Sepolia chain ID.
pub const BASE_SEPOLIA_CHAIN_ID: u64 = 84532;

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
