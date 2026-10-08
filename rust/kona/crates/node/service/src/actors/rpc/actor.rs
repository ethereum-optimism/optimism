//! RPC Server Actor

use crate::{NodeActor, RpcActorError};
use async_trait::async_trait;
use jsonrpsee::server::ServerHandle;

/// An actor that treats an unexpected RPC server stop as a fatal error.
#[derive(Debug)]
pub struct RpcActor {
    /// The running server handle.
    handle: ServerHandle,
}

impl RpcActor {
    /// Takes ownership of a running RPC server handle.
    pub const fn new(handle: ServerHandle) -> Self {
        Self { handle }
    }
}

#[async_trait]
impl NodeActor for RpcActor {
    type Error = RpcActorError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        self.handle.clone().stopped().await;
        Err(RpcActorError::ServerStopped)
    }
}
