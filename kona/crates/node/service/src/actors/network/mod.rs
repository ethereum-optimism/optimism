//! Network Actor

mod actor;
pub use actor::{NetworkActor, NetworkActorError, NetworkContext, NetworkInboundData};

mod builder;
pub use builder::NetworkBuilder;

mod driver;
pub use driver::{NetworkDriver, NetworkDriverError};

mod error;
pub use error::NetworkBuilderError;

mod handler;
pub use handler::NetworkHandler;

mod config;
mod gossip;
pub use gossip::{
    QueuedUnsafePayloadGossipClient, UnsafePayloadGossipClient, UnsafePayloadGossipClientError,
};

pub use config::NetworkConfig;

#[cfg(test)]
pub use gossip::MockUnsafePayloadGossipClient;
