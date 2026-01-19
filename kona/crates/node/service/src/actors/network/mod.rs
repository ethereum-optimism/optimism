//! Network Actor

mod actor;
pub use actor::{NetworkActor, NetworkActorError, NetworkInboundData};

mod builder;
pub use builder::NetworkBuilder;

mod config;
pub use config::NetworkConfig;

mod driver;
pub use driver::{NetworkDriver, NetworkDriverError};

mod engine_client;
pub use engine_client::{NetworkEngineClient, QueuedNetworkEngineClient};

mod error;
pub use error::NetworkBuilderError;

mod gossip;
pub use gossip::{
    QueuedUnsafePayloadGossipClient, UnsafePayloadGossipClient, UnsafePayloadGossipClientError,
};

mod handler;
pub use handler::NetworkHandler;

#[cfg(test)]
pub use gossip::MockUnsafePayloadGossipClient;
