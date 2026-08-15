//! Node composition and structured service supervision.

mod builder;
pub use builder::{DerivationDelegateConfig, L1ConfigBuilder, RollupNodeBuilder};

mod config;
pub use config::L1Config;

mod mode;
pub use mode::{InteropMode, NodeMode};

mod run;
pub use run::RollupNode;

#[cfg(test)]
mod tests;
