//! Unsafe chain acquisition through local sequencing or network following.

mod error;
pub use error::{UnsafeChainError, UnsafePayloadIngressError};

mod follower;
pub use follower::{DEFAULT_UNSAFE_PAYLOAD_CAPACITY, FollowerService, UnsafePayloadIngress};

#[cfg(test)]
mod tests;
