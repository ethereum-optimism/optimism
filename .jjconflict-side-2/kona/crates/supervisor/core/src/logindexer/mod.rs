//! Log indexing module for processing L2 receipts and extracting messages.
//!
//! This module provides functionality to extract and persist
//! [`ExecutingMessage`](kona_supervisor_types::ExecutingMessage)s and their corresponding
//! [`Log`](alloy_primitives::Log)s from L2 block receipts. It handles computing message payload
//! hashes and log hashes based on the interop messaging specification.
//!
//! # Modules
//!
//! - [`LogIndexer`] — main indexer that processes logs and persists them.
//! - [`LogIndexerError`] — error type for failures in fetching or storing logs.
//! - `util` — helper functions for computing payload and log hashes.
mod indexer;
pub use indexer::{LogIndexer, LogIndexerError};

mod util;
pub use util::{log_to_log_hash, log_to_message_payload, payload_hash_to_log_hash};
