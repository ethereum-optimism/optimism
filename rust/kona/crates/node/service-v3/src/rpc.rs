//! RPC task scaffold and domain-control routing.

use crate::{ControlError, SafeChainBuilderHandle, UnsafeChainBuilderHandle};
use thiserror::Error;
use tokio::sync::{mpsc, oneshot};

const CONTROL_CAPACITY: usize = 4;

#[derive(Debug)]
enum RpcCommand {
    Shutdown(oneshot::Sender<Result<(), ControlError>>),
}

/// Cloneable lifecycle capability for the RPC task.
#[derive(Debug, Clone)]
pub struct RpcHandle {
    control_tx: mpsc::Sender<RpcCommand>,
}

impl RpcHandle {
    /// Stops RPC intake and waits for the server task to acknowledge shutdown.
    pub async fn shutdown(&self) -> Result<(), ControlError> {
        let (response, result) = oneshot::channel();
        self.control_tx
            .send(RpcCommand::Shutdown(response))
            .await
            .map_err(|_| ControlError::Unavailable)?;
        result.await.map_err(|_| ControlError::ResponseDropped)?
    }
}

/// Long-running RPC transport owner.
///
/// RPC retains only narrow control handles. JSON-RPC server construction and method registration
/// will be added after the safe and unsafe core workflows exist.
#[derive(Debug)]
pub struct Rpc {
    safe_chain: SafeChainBuilderHandle,
    unsafe_chain: UnsafeChainBuilderHandle,
    control_rx: mpsc::Receiver<RpcCommand>,
}

impl Rpc {
    /// Creates the RPC task and its lifecycle handle without spawning it.
    pub fn new(
        safe_chain: SafeChainBuilderHandle,
        unsafe_chain: UnsafeChainBuilderHandle,
    ) -> (Self, RpcHandle) {
        let (control_tx, control_rx) = mpsc::channel(CONTROL_CAPACITY);
        (Self { safe_chain, unsafe_chain, control_rx }, RpcHandle { control_tx })
    }

    /// Returns the safe-chain control capability that RPC methods will use.
    pub const fn safe_chain_handle(&self) -> &SafeChainBuilderHandle {
        &self.safe_chain
    }

    /// Returns the unsafe-chain control capability that RPC methods will use.
    pub const fn unsafe_chain_handle(&self) -> &UnsafeChainBuilderHandle {
        &self.unsafe_chain
    }

    /// Runs the RPC task until explicitly shut down.
    pub async fn run(self) -> Result<(), RpcError> {
        let Self { safe_chain, unsafe_chain, mut control_rx } = self;
        let _domain_handles = (safe_chain, unsafe_chain);

        let command = control_rx.recv().await.ok_or(RpcError::ControlChannelClosed)?;
        match command {
            RpcCommand::Shutdown(response) => {
                // The concrete server will stop accepting requests and await connection shutdown
                // before acknowledging this command.
                let _ = response.send(Ok(()));
                Ok(())
            }
        }
    }
}

/// A terminal RPC task failure.
#[derive(Debug, Error, Clone, PartialEq, Eq)]
pub enum RpcError {
    /// Every RPC lifecycle handle was dropped unexpectedly.
    #[error("RPC control channel closed")]
    ControlChannelClosed,
}
