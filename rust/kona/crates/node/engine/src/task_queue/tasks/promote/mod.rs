//! Task and its associated types for promoting the cross-safe head.

mod task;
pub use task::PromoteCrossSafeTask;

mod error;
pub use error::PromoteCrossSafeTaskError;

#[cfg(test)]
mod task_test;
