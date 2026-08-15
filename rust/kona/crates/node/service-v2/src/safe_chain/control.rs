//! Safe-chain reset control.

use tokio::sync::{mpsc, oneshot};

const CONTROL_CAPACITY: usize = 32;

#[derive(Debug)]
pub(super) struct ResetRequest {
    pub response: oneshot::Sender<Result<(), SafeChainControlError>>,
}

/// Cloneable capability for derivation-only reset.
#[derive(Debug, Clone)]
pub struct SafeChainHandle {
    command_tx: mpsc::Sender<ResetRequest>,
}

impl SafeChainHandle {
    pub(super) fn channel() -> (Self, mpsc::Receiver<ResetRequest>) {
        let (command_tx, command_rx) = mpsc::channel(CONTROL_CAPACITY);
        (Self { command_tx }, command_rx)
    }

    /// Resets derivation state while preserving the current unsafe chain.
    pub async fn reset(&self) -> Result<(), SafeChainControlError> {
        let (response, result) = oneshot::channel();
        self.command_tx
            .send(ResetRequest { response })
            .await
            .map_err(|_| SafeChainControlError::Unavailable)?;
        result.await.map_err(|_| SafeChainControlError::ResponseDropped)?
    }
}

/// Safe-chain control failure.
#[derive(Debug, thiserror::Error, Clone, PartialEq, Eq)]
pub enum SafeChainControlError {
    /// Safe-chain service is unavailable.
    #[error("safe-chain service is unavailable")]
    Unavailable,
    /// A reset may have completed but its response was dropped.
    #[error("safe-chain reset response was dropped")]
    ResponseDropped,
    /// Derivation pipeline rejected the reset.
    #[error("safe-chain reset failed: {0}")]
    Reset(String),
}
