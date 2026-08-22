//! The [`ChainController`], [`ChainControllerRpcActor`], and their components.

mod controller;
pub use controller::{ChainController, ChainControllerRequest};

#[cfg(test)]
mod controller_test;

mod derivation_client;
pub use derivation_client::{
    ChainControllerDerivationClient, QueuedChainControllerDerivationClient,
};

mod config;
pub use config::EngineConfig;

mod error;
pub use error::ChainControllerError;

mod request;
pub use request::{
    BuildRequest, ChainControllerClientError, ChainControllerClientResult,
    ChainControllerRpcRequest, ResetRequest, SealRequest,
};

mod rpc_actor;
pub use rpc_actor::ChainControllerRpcActor;
