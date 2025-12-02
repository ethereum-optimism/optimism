//! This crate contains the core logic for the Optimism Supervisor component.

pub mod chain_processor;
pub use chain_processor::{ChainProcessor, ChainProcessorError, ProcessorState};

pub mod error;
pub use error::{SpecError, SupervisorError};

/// Contains the main Supervisor struct and its implementation.
mod supervisor;
pub use supervisor::{Supervisor, SupervisorService};

mod logindexer;
pub use logindexer::{
    LogIndexer, LogIndexerError, log_to_log_hash, log_to_message_payload, payload_hash_to_log_hash,
};

pub mod rpc;

pub mod config;
pub mod event;
pub mod l1_watcher;
pub mod syncnode;

pub mod safety_checker;
pub use safety_checker::{CrossSafetyCheckerJob, CrossSafetyError};

mod reorg;
pub use reorg::{ReorgHandler, ReorgHandlerError};
