//! P2P discovery and gossip transport construction.
//!
//! The rollup-node runtime moves the started handler into Engine; this module exposes no running
//! network-service capability to other node domains.

mod builder;
pub use builder::NetworkBuilder;

mod config;
pub use config::NetworkConfig;

mod error;
pub use error::NetworkBuilderError;

mod handler;
pub use handler::NetworkHandler;

mod stack;
pub use stack::{NetworkStartError, UnstartedNetwork};

mod standalone;
pub use standalone::{StandaloneNetwork, StandaloneNetworkError};
