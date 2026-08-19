//! Core [`RollupNode`] service, composing the available [`NodeActor`]s into various modes of
//! operation.
//!
//! [`NodeActor`]: crate::NodeActor

mod builder;
pub use builder::{DerivationDelegateConfig, L1ConfigBuilder, RollupNodeBuilder};

mod mode;
pub use mode::NodeMode;

mod node;
pub use node::{L1Config, RollupNode};

pub(crate) mod util;
pub(crate) use util::{shutdown_signal, spawn_and_wait};
