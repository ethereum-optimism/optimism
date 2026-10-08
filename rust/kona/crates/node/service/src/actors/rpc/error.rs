/// An error returned by the [`crate::RpcActor`].
#[derive(Debug, thiserror::Error)]
pub enum RpcActorError {
    /// The [`crate::RpcActor`]'s RPC server stopped unexpectedly.
    #[error("RPC server stopped unexpectedly")]
    ServerStopped,
}
