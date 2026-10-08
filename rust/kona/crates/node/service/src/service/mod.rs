//! Core [`RollupNode`] service, composing the available [`NodeActor`]s into various modes of
//! operation.
//!
//! [`NodeActor`]: crate::NodeActor

mod block_sink;
pub(crate) use block_sink::BufferImportedBlocks;

mod builder;
pub use builder::{DerivationDelegateConfig, L1ConfigBuilder, RollupNodeBuilder};

mod middleware;

mod mode;
pub use mode::NodeMode;

mod node;
pub use node::{L1Config, RollupNode};

mod supervisor;
use supervisor::{Supervisor, run_node_actor};
