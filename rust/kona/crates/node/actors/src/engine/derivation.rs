//! What decides the engine's safe and finalized heads.

use async_trait::async_trait;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};

/// Decides the engine's safe head: derivation from L1, or a trusted source.
///
/// The engine drives it from its first reset: it asks what to make safe next on top of its safe
/// head, makes it safe, and asks again. It reports each of its resets, and each derived block it
/// replaced with a deposits-only block.
#[async_trait]
pub trait Safety: Send + Sync + 'static {
    /// Why safety stopped. The engine stops with it.
    type Error: Send;

    /// Waits for what to make safe next on top of `safe_head`, the engine's safe head.
    ///
    /// Asking again with the same safe head returns the same result, so dropping the wait loses
    /// nothing.
    async fn next(&mut self, safe_head: L2BlockInfo) -> Result<Derived, Self::Error>;

    /// Restarts from `safe`, the engine's safe head after it reset.
    async fn reset(&mut self, safe: L2BlockInfo) -> Result<(), Self::Error>;

    /// Drops the data of the last block made safe, which the engine replaced with a deposits-only
    /// block.
    async fn flush(&mut self) -> Result<(), Self::Error>;
}

/// Decides the engine's finalized head: from the L1 blocks its safe heads were derived from, or a
/// trusted source.
///
/// The engine drives it, from its first reset, with each safe head whose source it knows.
#[async_trait]
pub trait Finality: Send + Sync + 'static {
    /// Why finality stopped. The engine stops with it.
    type Error: Send;

    /// Waits for a block to finalize, given `safe`, the engine's latest safe head.
    ///
    /// Asking again with the same safe head loses nothing.
    async fn next(&mut self, safe: Safe) -> Result<L2BlockInfo, Self::Error>;
}

/// What to make safe next.
#[derive(Debug, Clone, PartialEq, Eq)]
#[allow(clippy::large_enum_variant)]
pub enum Derived {
    /// Attributes no block of the unsafe chain matches: the engine builds the block from them.
    Attributes(OpAttributesWithParent),
    /// A block the execution layer has.
    Safe(Safe),
}

/// A safe block, and the L1 block it was derived from.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Safe {
    /// The safe block.
    pub head: L2BlockInfo,
    /// The L1 block `head` was derived from.
    pub source: BlockInfo,
}
