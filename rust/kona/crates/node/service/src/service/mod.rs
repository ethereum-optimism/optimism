//! Core [`RollupNode`] service, composing the available [`NodeActor`]s into various modes of
//! operation.
//!
//! [`RollupNode::compose`] is the single-chain composition entry point: every host that runs a
//! chain — the single-chain [`RollupNode::start`] here, or a multi-chain host running one actor
//! group per chain — builds its actors there and runs them with [`run_actors`], so no second copy
//! of the wiring exists to drift.
//!
//! [`NodeActor`]: crate::NodeActor

mod block_sink;
pub(crate) use block_sink::BufferImportedBlocks;

mod builder;
pub use builder::{DerivationDelegateConfig, L1ConfigBuilder, RollupNodeBuilder};

mod composition;
pub use composition::{ComposedChain, L1WatcherPorts};

mod mode;
pub use mode::NodeMode;

mod node;
pub use node::{L1Config, RollupNode};

mod spawn;
pub use spawn::{
    BoxedNodeActor, ChainLabeledActor, ErasedNodeActor, IntoBoxedNodeActor, label_chain, run_actors,
};
