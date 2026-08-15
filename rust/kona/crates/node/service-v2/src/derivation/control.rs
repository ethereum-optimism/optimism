//! Derivation-owned administrative reset control.

use tokio::sync::{mpsc, oneshot};

const CONTROL_CAPACITY: usize = 32;

#[derive(Debug)]
pub(super) struct ResetRequest {
    pub response: oneshot::Sender<Result<(), DerivationControlError>>,
}

/// Narrow RPC-only capability for derivation-pipeline reset.
#[derive(Debug, Clone)]
pub struct DerivationAdminAdapter {
    command_tx: mpsc::Sender<ResetRequest>,
}

impl DerivationAdminAdapter {
    pub(super) fn channel() -> (Self, mpsc::Receiver<ResetRequest>) {
        let (command_tx, command_rx) = mpsc::channel(CONTROL_CAPACITY);
        (Self { command_tx }, command_rx)
    }

    /// Resets only derivation state, preserving the current unsafe chain.
    pub async fn reset(&self) -> Result<(), DerivationControlError> {
        let (response, result) = oneshot::channel();
        self.command_tx
            .send(ResetRequest { response })
            .await
            .map_err(|_| DerivationControlError::Unavailable)?;
        result.await.map_err(|_| DerivationControlError::ResponseDropped)?
    }
}

/// Derivation administration failure.
#[derive(Debug, thiserror::Error, Clone, PartialEq, Eq)]
pub enum DerivationControlError {
    /// Derivation is unavailable.
    #[error("Derivation service is unavailable")]
    Unavailable,
    /// A reset may have completed but its response was dropped.
    #[error("Derivation reset response was dropped")]
    ResponseDropped,
    /// The pipeline rejected the reset.
    #[error("Derivation reset failed: {0}")]
    Reset(String),
}
