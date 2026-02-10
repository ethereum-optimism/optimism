//! RPC Server Actor

use crate::{NodeActor, RpcActorError};
use async_trait::async_trait;
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
use std::{marker::PhantomData, time::Duration};
use tokio::sync::mpsc;

/// Builder for the [`RpcActor`]. Folds in what was previously `RpcContext`.
#[derive(Debug)]
pub struct RpcActorBuilder<
    EngineRpcClient_,
    RollupBoostAdminClient_,
    RollupBoostHealth_,
    SeqAdminClient_,
> where
    EngineRpcClient_: EngineRpcClient,
    RollupBoostAdminClient_: RollupBoostAdminClient,
    RollupBoostHealth_: RollupBoostHealthzApiServer,
    SeqAdminClient_: SequencerAdminAPIClient,
{
    /// A launcher for the rpc.
    pub config: RpcBuilder,
    /// The engine RPC client.
    pub engine_rpc_client: EngineRpcClient_,
    /// The rollup boost admin RPC client.
    pub rollup_boost_admin_rpc_client: RollupBoostAdminClient_,
    /// The rollup boost health RPC client.
    pub rollup_boost_health_rpc_client: RollupBoostHealth_,
    /// The optional sequencer admin RPC client.
    pub sequencer_admin_rpc_client: Option<SeqAdminClient_>,
    /// The network p2p rpc sender.
    pub p2p_network: mpsc::Sender<P2pRpcRequest>,
    /// The network admin rpc sender.
    pub network_admin: mpsc::Sender<NetworkAdminQuery>,
    /// The l1 watcher queries sender.
    pub l1_watcher_queries: mpsc::Sender<L1WatcherQueries>,
}

/// An actor that handles the RPC server for the rollup node.
///
/// The generic parameters are consumed during [`init`](NodeActor::init) when the RPC modules are
/// built. At runtime, the actor only holds the config, modules, and server handle.
#[derive(Debug)]
pub struct RpcActor<
    EngineRpcClient_: EngineRpcClient,
    RollupBoostAdminClient_: RollupBoostAdminClient,
    RollupBoostHealth_: RollupBoostHealthzApiServer,
    SeqAdminClient_: SequencerAdminAPIClient,
> {
    /// A launcher for the rpc.
    config: RpcBuilder,
    /// The RPC modules.
    modules: RpcModule<()>,
    /// The server handle.
    handle: ServerHandle,
    /// Maximum number of restarts.
    max_restarts: u32,
    /// Current restart count.
    current_restarts: u32,
    /// Phantom data for generic types consumed during init.
    _phantom: PhantomData<(
        EngineRpcClient_,
        RollupBoostAdminClient_,
        RollupBoostHealth_,
        SeqAdminClient_,
    )>,
}

/// Launches the jsonrpsee [`Server`].
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
impl<EngineRpcClient_, RollupBoostAdminClient_, RollupBoostHealth_, SeqAdminClient_> NodeActor
    for RpcActor<EngineRpcClient_, RollupBoostAdminClient_, RollupBoostHealth_, SeqAdminClient_>
where
    EngineRpcClient_: EngineRpcClient + 'static,
    RollupBoostAdminClient_: RollupBoostAdminClient + 'static,
    RollupBoostHealth_: RollupBoostHealthzApiServer + 'static,
    SeqAdminClient_: SequencerAdminAPIClient + 'static,
{
    type Error = RpcActorError;
    type Builder = RpcActorBuilder<
        EngineRpcClient_,
        RollupBoostAdminClient_,
        RollupBoostHealth_,
        SeqAdminClient_,
    >;
    type InboundData = ();

    async fn init(builder: Self::Builder) -> Result<(Self::InboundData, Self), Self::Error> {
        let mut modules = RpcModule::new(());

        modules.merge(HealthzApiServer::into_rpc(HealthzRpc {}))?;
        modules
            .merge(RollupBoostHealthzApiServer::into_rpc(builder.rollup_boost_health_rpc_client))?;

        // Build the p2p rpc module.
        modules.merge(P2pRpc::new(builder.p2p_network).into_rpc())?;

        // Build the admin rpc module.
        modules.merge(
            AdminRpc::new(
                builder.sequencer_admin_rpc_client,
                builder.network_admin,
                Some(builder.rollup_boost_admin_rpc_client),
            )
            .into_rpc(),
        )?;

        // Create context for communication between actors.
        let rollup_rpc =
            RollupRpc::new(builder.engine_rpc_client.clone(), builder.l1_watcher_queries);
        modules.merge(rollup_rpc.into_rpc())?;

        // Add development RPC module for engine state introspection if enabled
        if builder.config.dev_enabled() {
            let dev_rpc = DevEngineRpc::new(builder.engine_rpc_client.clone());
            modules.merge(dev_rpc.into_rpc())?;
        }

        if builder.config.ws_enabled() {
            modules.merge(WsRPC::new(builder.engine_rpc_client.clone()).into_rpc())?;
        }

        let max_restarts = builder.config.restart_count();
        let handle = launch(&builder.config, modules.clone()).await?;

        let actor = Self {
            config: builder.config,
            modules,
            handle,
            max_restarts,
            current_restarts: 0,
            _phantom: PhantomData,
        };
        Ok(((), actor))
    }

    async fn step(&mut self) -> Result<(), Self::Error> {
        // Wait until the server stops.
        self.handle.clone().stopped().await;

        // Attempt to restart.
        if self.current_restarts >= self.max_restarts {
            return Err(RpcActorError::ServerStopped);
        }

        self.current_restarts += 1;
        match launch(&self.config, self.modules.clone()).await {
            Ok(h) => {
                self.handle = h;
                Ok(())
            }
            Err(err) => {
                error!(target: "rpc", ?err, "Failed to launch rpc server");
                Err(RpcActorError::ServerStopped)
            }
        }
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
