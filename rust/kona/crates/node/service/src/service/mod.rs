//! Core [`RollupNode`] service, composing the available [`NodeActor`]s into various modes of
//! operation.
//!
//! [`NodeActor`]: crate::NodeActor

mod adapters;

mod block_sink;
pub(crate) use block_sink::BufferImportedBlocks;

mod clients;
pub use clients::{RollupNodeClients, RollupNodeClientsError};

mod config;
pub use config::{DerivationDelegateConfig, L1ClientConfig, RollupNodeConfig};

mod middleware;

pub use kona_node_actors::NodeMode;

mod node;
pub use node::RollupNode;

mod supervisor;
use supervisor::Supervisor;
