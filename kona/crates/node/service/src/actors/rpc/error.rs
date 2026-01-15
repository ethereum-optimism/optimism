use jsonrpsee::core::RegisterMethodError;

/// An error returned by the [`crate::RpcActor`].
#[derive(Debug, thiserror::Error)]
pub enum RpcActorError {
    /// Failed to register the healthz endpoint.
    #[error("Failed to register the healthz endpoint")]
    RegisterHealthz(#[from] RegisterMethodError),
    /// Failed to launch the RPC server.
    #[error(transparent)]
    LaunchFailed(#[from] std::io::Error),
    /// The [`crate::RpcActor`]'s RPC server stopped unexpectedly.
    #[error("RPC server stopped unexpectedly")]
    ServerStopped,
    /// Failed to stop the RPC server.
    #[error("Failed to stop the RPC server")]
    StopFailed,
}
