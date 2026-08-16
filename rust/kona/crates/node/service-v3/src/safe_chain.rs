//! Safe-chain derivation task scaffold.

use crate::{ControlError, Engine};
use std::sync::Arc;
use thiserror::Error;
use tokio::sync::{Mutex, mpsc, oneshot};

const CONTROL_CAPACITY: usize = 8;

#[derive(Debug)]
enum SafeChainCommand {
    Reset(oneshot::Sender<Result<(), ControlError>>),
    Shutdown(oneshot::Sender<Result<(), ControlError>>),
}

/// Cloneable control capability for the safe-chain task.
#[derive(Debug, Clone)]
pub struct SafeChainBuilderHandle {
    control_tx: mpsc::Sender<SafeChainCommand>,
}

impl SafeChainBuilderHandle {
    /// Requests a derivation-pipeline reset.
    ///
    /// This is currently a lifecycle stub. Pipeline reset behavior will be implemented with the
    /// safe-chain workflow.
    pub async fn reset(&self) -> Result<(), ControlError> {
        self.request(SafeChainCommand::Reset).await
    }

    /// Requests clean task shutdown and waits for acknowledgement.
    pub async fn shutdown(&self) -> Result<(), ControlError> {
        self.request(SafeChainCommand::Shutdown).await
    }

    async fn request(
        &self,
        command: impl FnOnce(oneshot::Sender<Result<(), ControlError>>) -> SafeChainCommand,
    ) -> Result<(), ControlError> {
        let (response, result) = oneshot::channel();
        self.control_tx.send(command(response)).await.map_err(|_| ControlError::Unavailable)?;
        result.await.map_err(|_| ControlError::ResponseDropped)?
    }
}

/// Long-running owner of derivation, safe advancement, finality, and L1 reorg handling.
#[derive(Debug)]
pub struct SafeChainBuilder<L1Client, L2Client, EngineClient> {
    l1: L1Client,
    l2_el: L2Client,
    engine: Arc<Mutex<Engine<EngineClient>>>,
    control_rx: mpsc::Receiver<SafeChainCommand>,
}

impl<L1Client, L2Client, EngineClient> SafeChainBuilder<L1Client, L2Client, EngineClient> {
    /// Creates the safe-chain task and its control handle without spawning it.
    pub(crate) fn new(
        l1: L1Client,
        l2_el: L2Client,
        engine: Arc<Mutex<Engine<EngineClient>>>,
    ) -> (Self, SafeChainBuilderHandle) {
        let (control_tx, control_rx) = mpsc::channel(CONTROL_CAPACITY);
        (Self { l1, l2_el, engine, control_rx }, SafeChainBuilderHandle { control_tx })
    }

    /// Runs the safe-chain task until explicitly shut down.
    ///
    /// The current scaffold owns its dependencies and lifecycle. Derivation events and semantic
    /// Engine operations will be added here without changing the task boundary.
    pub async fn run(self) -> Result<(), SafeChainBuilderError> {
        let Self { l1, l2_el, engine, mut control_rx } = self;
        let _owned_resources = (l1, l2_el, engine);

        loop {
            let command =
                control_rx.recv().await.ok_or(SafeChainBuilderError::ControlChannelClosed)?;
            match command {
                SafeChainCommand::Reset(response) => {
                    // The derivation pipeline will be reset here once it is introduced.
                    let _ = response.send(Ok(()));
                }
                SafeChainCommand::Shutdown(response) => {
                    let _ = response.send(Ok(()));
                    return Ok(());
                }
            }
        }
    }
}

/// A terminal safe-chain task failure.
#[derive(Debug, Error, Clone, PartialEq, Eq)]
pub enum SafeChainBuilderError {
    /// Every safe-chain control handle was dropped unexpectedly.
    #[error("safe-chain control channel closed")]
    ControlChannelClosed,
}
