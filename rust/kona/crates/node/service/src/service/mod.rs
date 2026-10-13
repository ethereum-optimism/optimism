//! Core [`RollupNode`] service, composing the available [`NodeActor`]s into various modes of
//! operation.
//!
//! [`NodeActor`]: crate::NodeActor

mod adapters;

mod block_sink;
pub(crate) use block_sink::BufferImportedBlocks;

mod clients;
pub use clients::{RollupNodeClients, RollupNodeClientsError, RollupNodeEndpoints};

mod config;
pub use config::RollupNodeConfig;

mod error;
pub use error::{ListenProtocol, RollupNodeError};

mod middleware;

pub use kona_node_actors::NodeMode;

mod node;
pub use node::RollupNode;

mod spec;
pub use spec::ChainSpec;

mod startup;

mod supervisor;

#[cfg(test)]
pub(crate) mod test_utils;
use supervisor::Supervisor;
