//! The finality actor, which finalizes L2 blocks once the L1 blocks they were derived from
//! finalize.
//!
//! It follows the engine's published state to learn the L1 block each safe head was derived from,
//! and asks the engine to finalize the highest safe block whose source the finalized L1 block
//! covers. Only derivation from L1 runs it: delegated derivation finalizes the delegate's blocks.

mod actor;
pub use actor::build;

use crate::{EngineClientError, EngineClientResult, engine::FinalizeBlockId};
use async_trait::async_trait;
use std::fmt::Debug;
use thiserror::Error;

/// The engine, as the finality actor uses it.
#[async_trait]
pub trait Engine: Debug + Send + Sync {
    /// Asks the engine to finalize the L2 block identified by the provided [`FinalizeBlockId`],
    /// without waiting for the engine to do so.
    async fn finalize(&self, block_id: FinalizeBlockId) -> EngineClientResult<()>;
}

/// An error that stops the finality actor.
#[derive(Error, Debug)]
pub enum ActorError {
    /// A request to the engine actor failed.
    #[error("Failed to send a request to the engine actor: {0}")]
    Engine(#[from] EngineClientError),
    /// The engine's state publisher closed.
    #[error("engine state channel closed")]
    EngineStateClosed,
    /// The engine's reset publisher closed.
    #[error("engine reset channel closed")]
    EngineResetsClosed,
    /// The finalized L1 block publisher closed.
    #[error("finalized l1 channel closed")]
    FinalizedClosed,
}
