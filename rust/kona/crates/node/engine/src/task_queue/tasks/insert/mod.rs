//! Tasks to insert payloads into the execution engine.

mod task;
pub use task::{CanonicalizeTask, InsertTask};

mod error;
pub use error::InsertTaskError;
