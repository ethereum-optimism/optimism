//! Chain Processor Module
//! This module implements the Chain Processor, which manages the nodes and process events per
//! chain. It provides a structured way to handle tasks, manage chains, and process blocks
//! in a supervisor environment.
mod error;
pub use error::ChainProcessorError;

mod chain;
pub use chain::ChainProcessor;

mod metrics;
pub(crate) use metrics::Metrics;

mod state;
pub use state::ProcessorState;

pub mod handlers;
