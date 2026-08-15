//! Node composition and structured task supervision.

mod error;
pub use error::NodeError;

mod follower;
pub use follower::FollowerNode;

#[cfg(test)]
mod tests;
