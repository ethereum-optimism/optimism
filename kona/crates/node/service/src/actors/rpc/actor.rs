//! RPC Server Actor

use crate::{NodeActor, RpcActorError, actors::CancellableContext};
use async_trait::async_trait;
use derive_more::Constructor;
use jsonrpsee::{
    RpcModule,
    server::{Server, ServerHandle, middleware::http::ProxyGetRequestLayer},
};
use kona_gossip::P2pRpcRequest;
use kona_rpc::{
    AdminApiServer, AdminRpc, DevEngineApiServer, DevEngineRpc, EngineRpcClient, HealthzApiServer,
    HealthzRpc, L1WatcherQueries, NetworkAdminQuery, OpP2PApiServer, P2pRpc,
    RollupBoostAdminClient, RollupBoostHealthzApiServer, RollupNodeApiServer, RollupRpc,
    RpcBuilder, SequencerAdminAPIClient, WsRPC, WsServer,
};
use std::time::Duration;
use tokio::sync::mpsc;
use tokio_util::sync::{CancellationToken, WaitForCancellationFuture};

/// An actor that handles the RPC server for the rollup node.
#[derive(Constructor, Debug)]
pub struct RpcActor<
    EngineRpcClient_,
    RollupBoostAdminClient_,
    RollupBoostHealth,
    SequencerAdminApiClient_,
> where
    EngineRpcClient_: EngineRpcClient,
    RollupBoostAdminClient_: RollupBoostAdminClient,
    RollupBoostHealth: RollupBoostHealthzApiServer,
    SequencerAdminApiClient_: SequencerAdminAPIClient,
{
    /// A launcher for the rpc.
    config: RpcBuilder,

    engine_rpc_client: EngineRpcClient_,
    rollup_boost_admin_rpc_client: RollupBoostAdminClient_,
    rollup_boost_health_rpc_client: RollupBoostHealth,
    sequencer_admin_rpc_client: Option<SequencerAdminApiClient_>,
}

/// The communication context used by the RPC actor.
#[derive(Debug)]
pub struct RpcContext {
    /// The network p2p rpc sender.
    pub p2p_network: mpsc::Sender<P2pRpcRequest>,
    /// The network admin rpc sender.
    pub network_admin: mpsc::Sender<NetworkAdminQuery>,
    /// The l1 watcher queries sender.
    pub l1_watcher_queries: mpsc::Sender<L1WatcherQueries>,
    /// The cancellation token, shared between all tasks.
    pub cancellation: CancellationToken,
}

impl CancellableContext for RpcContext {
    fn cancelled(&self) -> WaitForCancellationFuture<'_> {
        self.cancellation.cancelled()
    }
}

/// Launches the jsonrpsee [`Server`].
///
/// If the RPC server is disabled, this will return `Ok(None)`.
///
/// ## Errors
///
/// - [`std::io::Error`] if the server fails to start.
async fn launch(
    config: &RpcBuilder,
    module: RpcModule<()>,
) -> Result<ServerHandle, std::io::Error> {
    let middleware = tower::ServiceBuilder::new()
        .layer(
            ProxyGetRequestLayer::new([
                ("/healthz", "healthz"),
                ("/kona-rollup-boost/healthz", "kona-rollup-boost_healthz"),
            ])
            .expect("Critical: Failed to build GET method proxy"),
        )
        .timeout(Duration::from_secs(2));
    let server = Server::builder().set_http_middleware(middleware).build(config.socket).await?;

    if let Ok(addr) = server.local_addr() {
        info!(target: "rpc", addr = ?addr, "RPC server bound to address");
    } else {
        error!(target: "rpc", "Failed to get local address for RPC server");
    }

    Ok(server.start(module))
}

#[async_trait]
impl<EngineRpcClient_, RollupBoostAdminClient_, RollupBoostHealth, SequencerAdminApiClient_>
    NodeActor
    for RpcActor<
        EngineRpcClient_,
        RollupBoostAdminClient_,
        RollupBoostHealth,
        SequencerAdminApiClient_,
    >
where
    EngineRpcClient_: EngineRpcClient + 'static,
    RollupBoostAdminClient_: RollupBoostAdminClient + 'static,
    RollupBoostHealth: RollupBoostHealthzApiServer + 'static,
    SequencerAdminApiClient_: SequencerAdminAPIClient + 'static,
{
    type Error = RpcActorError;
    type StartData = RpcContext;

    async fn start(
        mut self,
        RpcContext {
            cancellation,
            p2p_network,
            l1_watcher_queries,
            network_admin,
        }: Self::StartData,
    ) -> Result<(), Self::Error> {
        let mut modules = RpcModule::new(());

        modules.merge(HealthzApiServer::into_rpc(HealthzRpc {}))?;
        modules
            .merge(RollupBoostHealthzApiServer::into_rpc(self.rollup_boost_health_rpc_client))?;

        // Build the p2p rpc module.
        modules.merge(P2pRpc::new(p2p_network).into_rpc())?;

        // Build the admin rpc module.
        modules.merge(
            AdminRpc::new(
                self.sequencer_admin_rpc_client,
                network_admin,
                Some(self.rollup_boost_admin_rpc_client),
            )
            .into_rpc(),
        )?;

        // Create context for communication between actors.
        let rollup_rpc = RollupRpc::new(self.engine_rpc_client.clone(), l1_watcher_queries);
        modules.merge(rollup_rpc.into_rpc())?;

        // Add development RPC module for engine state introspection if enabled
        if self.config.dev_enabled() {
            let dev_rpc = DevEngineRpc::new(self.engine_rpc_client.clone());
            modules.merge(dev_rpc.into_rpc())?;
        }

        if self.config.ws_enabled() {
            modules.merge(WsRPC::new(self.engine_rpc_client.clone()).into_rpc())?;
        }

        let restarts = self.config.restart_count();

        let mut handle = launch(&self.config, modules.clone()).await?;

        for _ in 0..=restarts {
            tokio::select! {
                _ = handle.clone().stopped() => {
                    match launch(&self.config, modules.clone()).await {
                        Ok(h) => handle = h,
                        Err(err) => {
                            error!(target: "rpc", ?err, "Failed to launch rpc server");
                            cancellation.cancel();
                            return Err(RpcActorError::ServerStopped);
                        }
                    }
                }
                _ = cancellation.cancelled() => {
                    // The cancellation token has been triggered, so we should stop the server.
                    handle.stop().map_err(|_| RpcActorError::StopFailed)?;
                    // Since the RPC Server didn't originate the error, we should return Ok.
                    return Ok(());
                }
            }
        }

        // Stop the node if there has already been 3 rpc restarts.
        cancellation.cancel();
        return Err(RpcActorError::ServerStopped);
    }
}

#[cfg(test)]
mod tests {
    use std::net::SocketAddr;

    use super::*;

    #[tokio::test]
    async fn test_launch_no_modules() {
        let launcher = RpcBuilder {
            socket: SocketAddr::from(([127, 0, 0, 1], 8080)),
            no_restart: false,
            enable_admin: false,
            admin_persistence: None,
            ws_enabled: false,
            dev_enabled: false,
        };
        let result = launch(&launcher, RpcModule::new(())).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_launch_with_modules() {
        let launcher = RpcBuilder {
            socket: SocketAddr::from(([127, 0, 0, 1], 8081)),
            no_restart: false,
            enable_admin: false,
            admin_persistence: None,
            ws_enabled: false,
            dev_enabled: false,
        };
        let mut modules = RpcModule::new(());

        modules.merge(RpcModule::new(())).expect("module merge");
        modules.merge(RpcModule::new(())).expect("module merge");
        modules.merge(RpcModule::new(())).expect("module merge");

        let result = launch(&launcher, modules).await;
        assert!(result.is_ok());
    }
}
