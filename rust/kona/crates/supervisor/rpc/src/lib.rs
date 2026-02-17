#![doc = include_str!("../README.md")]

#[cfg(feature = "jsonrpsee")]
pub mod jsonrpsee;
#[cfg(all(feature = "jsonrpsee", feature = "client"))]
pub use jsonrpsee::{ManagedModeApiClient, SupervisorAdminApiClient, SupervisorApiClient};
#[cfg(feature = "jsonrpsee")]
pub use jsonrpsee::{SupervisorAdminApiServer, SupervisorApiServer};

#[cfg(feature = "server")]
pub mod config;
#[cfg(feature = "server")]
pub use config::SupervisorRpcConfig;

#[cfg(feature = "server")]
pub mod server;
#[cfg(feature = "server")]
pub use server::SupervisorRpcServer;

#[cfg(feature = "reqwest")]
pub mod reqwest;
#[cfg(feature = "reqwest")]
pub use reqwest::{CheckAccessListClient, SupervisorClient, SupervisorClientError};

pub mod response;
pub use response::{
    ChainRootInfoRpc, SuperRootOutputRpc, SupervisorChainSyncStatus, SupervisorSyncStatus,
};

pub use kona_protocol::BlockInfo;
