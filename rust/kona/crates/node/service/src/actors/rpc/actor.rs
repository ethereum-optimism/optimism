//! RPC Server Actor

use crate::{NodeActor, RpcActorError, actors::rpc::launcher::RpcServerHandle};
use async_trait::async_trait;

/// An actor that treats an unexpected RPC server stop as a fatal error.
#[derive(Debug)]
pub struct RpcActor<Handle: RpcServerHandle> {
    /// The running server handle.
    handle: Handle,
}

impl<Handle: RpcServerHandle> RpcActor<Handle> {
    /// Takes ownership of a running RPC server handle.
    pub const fn new(handle: Handle) -> Self {
        Self { handle }
    }
}

#[async_trait]
impl<Handle: RpcServerHandle> NodeActor for RpcActor<Handle> {
    type Error = RpcActorError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        self.handle.stopped().await;
        Err(RpcActorError::ServerStopped)
    }
}
