use crate::derivation::DerivationClientResult;
use async_trait::async_trait;
use kona_derive::Signal;
use kona_protocol::L2BlockInfo;
use std::fmt::Debug;

/// The derivation actor, as the engine actor uses it.
#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait EngineDerivationClient: Debug + Send + Sync {
    /// Notifies derivation that the execution layer finished syncing, with the safe head to start
    /// from.
    /// Note: Does not wait for the derivation actor to process this message.
    async fn notify_sync_completed(&self, safe_head: L2BlockInfo) -> DerivationClientResult<()>;

    /// Sends derivation the engine's new `safe_head`.
    /// Note: Does not wait for the derivation actor to process this message.
    async fn send_new_engine_safe_head(&self, safe_head: L2BlockInfo)
    -> DerivationClientResult<()>;

    /// Sends the derivation pipeline the provided [`Signal`].
    /// Note: Does not wait for the derivation actor to process this message.
    async fn send_signal(&self, signal: Signal) -> DerivationClientResult<()>;
}
