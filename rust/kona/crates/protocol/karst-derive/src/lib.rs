//! # karst-derive
//!
//! A flat, inlined OP Stack derivation pipeline that only supports Holocene+.
//! This replaces the staged, trait-composed `kona-derive` pipeline with a single
//! `Pipeline` struct that holds all intermediate state as fields and runs all
//! derivation steps as direct method calls.

extern crate alloc;

mod pipeline;
mod traversal;
mod frames;
mod channel;
mod reader;
mod batch;

pub use pipeline::Pipeline;

// Re-export key types and traits from kona-derive that consumers need.
pub use kona_derive::{
    AttributesBuilder, ChainProvider, DataAvailabilityProvider, L2ChainProvider, PipelineError,
    PipelineErrorKind, PipelineResult,
};
