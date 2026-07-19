//! Canonical unsafe blocks stream client and actor.

mod actor;
pub use actor::{BlocksClientActor, BlocksClientActorError};

mod client;
pub use client::{BlocksClient, BlocksClientError, MAX_BLOCK_FRAME_SIZE};

#[cfg(test)]
mod tests;
