//! RPC Server Actor

use crate::NodeActor;
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

impl Drop for RpcActor {
    fn drop(&mut self) {
        // Stop even if another component still holds a handle to the server.
        let _ = self.handle.stop();
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

/// An error returned by the [`crate::RpcActor`].
#[derive(Debug, thiserror::Error)]
pub enum RpcActorError {
    /// The [`crate::RpcActor`]'s RPC server stopped unexpectedly.
    #[error("RPC server stopped unexpectedly")]
    ServerStopped,
}

#[cfg(test)]
mod tests {
    use super::*;
    use jsonrpsee::{RpcModule, server::Server};
    use std::time::Duration;
    use tokio::time::timeout;

    async fn running_actor() -> (RpcActor, ServerHandle) {
        let server = Server::builder().build("127.0.0.1:0").await.unwrap();
        let handle = server.start(RpcModule::new(()));
        (RpcActor::new(handle.clone()), handle)
    }

    #[tokio::test]
    async fn unexpected_server_stop_is_fatal() {
        let (mut actor, handle) = running_actor().await;
        let step = actor.step();
        tokio::pin!(step);
        assert!(futures::poll!(&mut step).is_pending(), "a running server must keep serving");
        handle.stop().unwrap();
        assert!(matches!(
            timeout(Duration::from_secs(1), step).await.unwrap(),
            Err(RpcActorError::ServerStopped)
        ));
    }

    #[tokio::test]
    async fn dropping_actor_stops_server_with_other_handles_alive() {
        let (actor, handle) = running_actor().await;
        drop(actor);
        timeout(Duration::from_secs(1), handle.stopped())
            .await
            .expect("dropping the actor must shut down its server");
    }
}
