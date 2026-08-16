//! Unsafe-chain following and local sequencing task scaffold.

use crate::{ControlError, SharedEngine};
use thiserror::Error;
use tokio::sync::{mpsc, oneshot, watch};

const CONTROL_CAPACITY: usize = 8;

/// The unsafe-chain task's current data-plane mode.
#[derive(Debug, Default, Clone, Copy, PartialEq, Eq)]
pub enum UnsafeMode {
    /// Import unsafe payloads received from P2P.
    #[default]
    Following,
    /// Produce and distribute local unsafe payloads.
    Sequencing,
}

#[derive(Debug)]
enum UnsafeChainCommand {
    StartSequencer(oneshot::Sender<Result<(), ControlError>>),
    StopSequencer(oneshot::Sender<Result<(), ControlError>>),
    Shutdown(oneshot::Sender<Result<(), ControlError>>),
}

/// Cloneable control capability for the unsafe-chain task.
#[derive(Debug, Clone)]
pub struct UnsafeChainBuilderHandle {
    control_tx: mpsc::Sender<UnsafeChainCommand>,
    mode_rx: watch::Receiver<UnsafeMode>,
}

impl UnsafeChainBuilderHandle {
    /// Returns the latest published unsafe-chain mode.
    pub fn mode(&self) -> UnsafeMode {
        *self.mode_rx.borrow()
    }

    /// Starts local sequencing at the next safe workflow boundary.
    pub async fn start_sequencer(&self) -> Result<(), ControlError> {
        self.request(UnsafeChainCommand::StartSequencer).await
    }

    /// Stops local sequencing and resumes P2P following.
    pub async fn stop_sequencer(&self) -> Result<(), ControlError> {
        self.request(UnsafeChainCommand::StopSequencer).await
    }

    /// Requests clean task shutdown and waits for acknowledgement.
    pub async fn shutdown(&self) -> Result<(), ControlError> {
        self.request(UnsafeChainCommand::Shutdown).await
    }

    async fn request(
        &self,
        command: impl FnOnce(oneshot::Sender<Result<(), ControlError>>) -> UnsafeChainCommand,
    ) -> Result<(), ControlError> {
        let (response, result) = oneshot::channel();
        self.control_tx.send(command(response)).await.map_err(|_| ControlError::Unavailable)?;
        result.await.map_err(|_| ControlError::ResponseDropped)?
    }
}

/// Long-running owner of P2P following, local sequencing, conductor use, and gossip publication.
#[derive(Debug)]
pub struct UnsafeChainBuilder<L1Client, EngineClient, Network, Conductor> {
    l1: L1Client,
    engine: SharedEngine<EngineClient>,
    network: Network,
    conductor: Option<Conductor>,
    control_rx: mpsc::Receiver<UnsafeChainCommand>,
    mode_tx: watch::Sender<UnsafeMode>,
}

impl<L1Client, EngineClient, Network, Conductor>
    UnsafeChainBuilder<L1Client, EngineClient, Network, Conductor>
{
    /// Creates the unsafe-chain task and its control handle without spawning it.
    pub fn new(
        l1: L1Client,
        engine: SharedEngine<EngineClient>,
        network: Network,
        conductor: Option<Conductor>,
    ) -> (Self, UnsafeChainBuilderHandle) {
        let (control_tx, control_rx) = mpsc::channel(CONTROL_CAPACITY);
        let (mode_tx, mode_rx) = watch::channel(UnsafeMode::Following);
        (
            Self { l1, engine, network, conductor, control_rx, mode_tx },
            UnsafeChainBuilderHandle { control_tx, mode_rx },
        )
    }

    /// Runs the unsafe-chain task until explicitly shut down.
    ///
    /// The current scaffold applies administrative mode transitions. P2P intake and local block
    /// production will be introduced into this same loop.
    pub async fn run(self) -> Result<(), UnsafeChainBuilderError> {
        let Self { l1, engine, network, conductor, mut control_rx, mode_tx } = self;
        let _owned_resources = (l1, engine, network, conductor);

        loop {
            let command =
                control_rx.recv().await.ok_or(UnsafeChainBuilderError::ControlChannelClosed)?;
            match command {
                UnsafeChainCommand::StartSequencer(response) => {
                    mode_tx.send_replace(UnsafeMode::Sequencing);
                    let _ = response.send(Ok(()));
                }
                UnsafeChainCommand::StopSequencer(response) => {
                    mode_tx.send_replace(UnsafeMode::Following);
                    let _ = response.send(Ok(()));
                }
                UnsafeChainCommand::Shutdown(response) => {
                    // Shutdown leaves the service in follower mode after any future in-flight
                    // sequencing action has been drained.
                    mode_tx.send_replace(UnsafeMode::Following);
                    let _ = response.send(Ok(()));
                    return Ok(());
                }
            }
        }
    }
}

/// A terminal unsafe-chain task failure.
#[derive(Debug, Error, Clone, PartialEq, Eq)]
pub enum UnsafeChainBuilderError {
    /// Every unsafe-chain control handle was dropped unexpectedly.
    #[error("unsafe-chain control channel closed")]
    ControlChannelClosed,
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::Engine;
    use kona_genesis::RollupConfig;
    use std::sync::Arc;

    #[tokio::test]
    async fn unsafe_chain_starts_in_following_mode() {
        let engine = Engine::new(Arc::new(()), Arc::new(RollupConfig::default())).shared();
        let (service, handle) = UnsafeChainBuilder::new((), engine, (), None::<()>);
        let task = tokio::spawn(service.run());

        assert_eq!(handle.mode(), UnsafeMode::Following);
        handle.start_sequencer().await.unwrap();
        assert_eq!(handle.mode(), UnsafeMode::Sequencing);
        handle.stop_sequencer().await.unwrap();
        assert_eq!(handle.mode(), UnsafeMode::Following);
        handle.shutdown().await.unwrap();
        assert_eq!(task.await.unwrap(), Ok(()));
    }
}
