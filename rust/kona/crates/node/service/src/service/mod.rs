//! Core [`RollupNode`] service, composing the available [`NodeActor`]s into various modes of
//! operation.
//!
//! [`NodeActor`]: crate::NodeActor

mod builder;
pub use builder::{DerivationDelegateConfig, L1ConfigBuilder, RollupNodeBuilder};

mod mode;
pub use mode::{InteropMode, NodeMode};

mod node;
pub use node::{L1Config, RollupNode};

// `util` hosts process-lifecycle helpers shared across actor modules; marking it
// `pub(crate)` mirrors its scope requirements. Making it plain `pub` would trigger
// `unreachable_pub`.
#[allow(clippy::redundant_pub_crate)]
pub(crate) mod util;
#[allow(clippy::redundant_pub_crate)]
pub(crate) use util::{shutdown_signal, spawn_and_wait};
