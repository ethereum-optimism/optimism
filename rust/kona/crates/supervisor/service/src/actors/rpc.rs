use std::{io, net::SocketAddr};

use async_trait::async_trait;
use derive_more::Constructor;
use jsonrpsee::{RpcModule, server::ServerBuilder};
use thiserror::Error;
use tokio_util::sync::CancellationToken;
use tracing::{error, info};

use crate::SupervisorActor;

#[derive(Debug, Constructor)]
pub struct SupervisorRpcActor<D> {
    rpc_addr: SocketAddr,
    rpc_module: RpcModule<D>,
    cancel_token: CancellationToken,
}

#[async_trait]
impl<D> SupervisorActor for SupervisorRpcActor<D>
where
    D: Send + Sync + 'static,
{
    type InboundEvent = ();
    type Error = SupervisorRpcActorError;

    async fn start(mut self) -> Result<(), Self::Error> {
        info!(
          target: "supervisor::rpc_actor",
          addr = %self.rpc_addr,
          "RPC server bound to address",
        );

        // let supervisor_rpc = SupervisorRpc::new(self.supervisor.clone());
        let server = ServerBuilder::default().build(self.rpc_addr).await?;
        // let mut root = supervisor_rpc.into_rpc();
        let handle = server.start(self.rpc_module);

        let stopped = handle.clone().stopped();
        let cancelled = self.cancel_token.cancelled();

        tokio::select! {
            _ = stopped => {
                error!(target: "supervisor::rpc_actor", "RPC server stopped unexpectedly");
                return Err(SupervisorRpcActorError::ServerStopped);
            }
            _ = cancelled => {
                match handle.stop() {
                    Ok(_) => info!(target: "supervisor::rpc_actor", "RPC server stopped gracefully"),
                    Err(e) => {
                        error!(target: "supervisor::rpc_actor", %e, "Failed to stop RPC server gracefully");
                        return Err(SupervisorRpcActorError::StopFailed);
                    }
                }
                info!(target: "supervisor::rpc_actor", "Cancellation requested, stopping RPC server...");
            }
        }

        Ok(())
    }
}

#[derive(Debug, Error)]
pub enum SupervisorRpcActorError {
    /// Failed to build the RPC server.
    #[error(transparent)]
    BuildFailed(#[from] io::Error),

    /// Indicates that the RPC server failed to start.
    #[error("rpc server stopped unexpectedly")]
    ServerStopped,

    /// Indicates that the RPC server failed to stop gracefully.
    #[error("failed to stop the RPC server")]
    StopFailed,
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_eips::BlockNumHash;
    use alloy_primitives::{B256, ChainId};
    use async_trait::async_trait;
    use kona_interop::{DependencySet, ExecutingDescriptor, SafetyLevel};
    use kona_protocol::BlockInfo;
    use kona_supervisor_core::{SupervisorError, SupervisorService};
    use kona_supervisor_rpc::{SuperRootOutputRpc, SupervisorApiServer};
    use kona_supervisor_types::SuperHead;
    use mockall::mock;
    use std::{
        net::{Ipv4Addr, SocketAddr},
        sync::Arc,
    };
    use tokio_util::sync::CancellationToken;

    // Mock SupervisorService
    mock!(
        #[derive(Debug)]
        pub SupervisorService {}

        #[async_trait]
        impl SupervisorService for SupervisorService {
            fn chain_ids(&self) -> impl Iterator<Item = ChainId>;
            fn dependency_set(&self) -> &DependencySet;
            fn super_head(&self, chain: ChainId) -> Result<SuperHead, SupervisorError>;
            fn latest_block_from(&self, l1_block: BlockNumHash, chain: ChainId) -> Result<BlockInfo, SupervisorError>;
            fn derived_to_source_block(&self, chain: ChainId, derived: BlockNumHash) -> Result<BlockInfo, SupervisorError>;
            fn local_unsafe(&self, chain: ChainId) -> Result<BlockInfo, SupervisorError>;
            fn local_safe(&self, chain: ChainId) -> Result<BlockInfo, SupervisorError>;
            fn cross_safe(&self, chain: ChainId) -> Result<BlockInfo, SupervisorError>;
            fn finalized(&self, chain: ChainId) -> Result<BlockInfo, SupervisorError>;
            fn finalized_l1(&self) -> Result<BlockInfo, SupervisorError>;
            fn check_access_list(&self, inbox_entries: Vec<B256>, min_safety: SafetyLevel, executing_descriptor: ExecutingDescriptor) -> Result<(), SupervisorError>;
            async fn super_root_at_timestamp(&self, timestamp: u64) -> Result<SuperRootOutputRpc, SupervisorError>;
        }
    );

    #[tokio::test]
    async fn test_supervisor_rpc_actor_stops_on_cancel() {
        let addr = SocketAddr::from((Ipv4Addr::LOCALHOST, 0));
        let supervisor = Arc::new(MockSupervisorService::new());
        let cancel_token = CancellationToken::new();

        let supervisor_rpc = kona_supervisor_core::rpc::SupervisorRpc::new(supervisor.clone());
        let rpc_module = supervisor_rpc.into_rpc();
        let actor = SupervisorRpcActor::new(addr, rpc_module, cancel_token.clone());

        let handle = tokio::spawn(actor.start());

        // Give the server a moment to start
        tokio::time::sleep(std::time::Duration::from_millis(100)).await;

        // Trigger cancellation
        cancel_token.cancel();

        // Await the actor and ensure it stops gracefully
        let result = handle.await.unwrap();
        assert!(result.is_ok() || matches!(result, Err(SupervisorRpcActorError::StopFailed)));
    }
}
