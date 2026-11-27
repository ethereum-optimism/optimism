//! Task and its associated types for moving the safe label to a verifier's head.

mod head;
pub use head::{VerifierHead, VerifierStatus};

mod finality;
pub use finality::{VerifierFinality, verifier_activation_timestamp};

mod task;
pub use task::{PromoteTask, l1_block_is_canonical};

mod error;
pub use error::PromoteTaskError;

#[cfg(test)]
mod task_test;
