//! Top-level JSON-RPC transport and narrow domain-capability routing.

use crate::{
    derivation::DerivationAdminAdapter,
    engine::{EngineAdminAdapter, EngineRpcAdapter},
};
use alloy_primitives::B256;
use async_trait::async_trait;
use jsonrpsee::{
    RpcModule,
    server::{Server, ServerHandle, middleware::http::ProxyGetRequestLayer},
};
use kona_rpc::{
    AdminApiServer, AdminRpc, DevEngineApiServer, DevEngineRpc, HealthzApiServer, HealthzRpc,
    L1WatcherQuerySender, OpP2PApiServer, P2pRpc, RollupNodeApiServer, RollupRpc, RpcBuilder,
    SequencerAdminAPIClient, SequencerAdminAPIError, WsRPC, WsServer,
};
use std::time::Duration;
use thiserror::Error;
use tokio::sync::oneshot;

/// Composes legacy admin RPC methods from the domain that owns each operation.
#[derive(Debug, Clone)]
pub struct AdminControl {
    engine: EngineAdminAdapter,
    derivation: DerivationAdminAdapter,
}

impl AdminControl {
    /// Creates the RPC-only compatibility router.
    pub const fn new(engine: EngineAdminAdapter, derivation: DerivationAdminAdapter) -> Self {
        Self { engine, derivation }
    }
}

#[async_trait]
impl SequencerAdminAPIClient for AdminControl {
    async fn is_sequencer_active(&self) -> Result<bool, SequencerAdminAPIError> {
        self.engine.is_sequencer_active()
    }

    async fn is_conductor_enabled(&self) -> Result<bool, SequencerAdminAPIError> {
        self.engine.is_conductor_enabled()
    }

    async fn is_recovery_mode(&self) -> Result<bool, SequencerAdminAPIError> {
        self.engine.is_recovery_mode()
    }

    async fn start_sequencer(&self) -> Result<(), SequencerAdminAPIError> {
        self.engine.start_sequencer().await
    }

    async fn stop_sequencer(&self) -> Result<B256, SequencerAdminAPIError> {
        self.engine.stop_sequencer().await
    }

    async fn set_recovery_mode(&self, mode: bool) -> Result<(), SequencerAdminAPIError> {
        self.engine.set_recovery_mode(mode).await
    }

    async fn override_leader(&self) -> Result<(), SequencerAdminAPIError> {
        self.engine.override_leader().await
    }

    async fn reset_derivation_pipeline(&self) -> Result<(), SequencerAdminAPIError> {
        self.derivation
            .reset()
            .await
            .map_err(|error| SequencerAdminAPIError::RequestError(error.to_string()))
    }
}

/// Long-running JSON-RPC server service.
#[derive(Debug)]
pub struct RpcService {
    config: RpcBuilder,
    modules: RpcModule<()>,
}

impl RpcService {
    /// Builds configured RPC namespaces from narrow domain adapters.
    pub fn new(
        config: RpcBuilder,
        engine_rpc: EngineRpcAdapter,
        engine_admin: EngineAdminAdapter,
        derivation_admin: DerivationAdminAdapter,
        l1_queries: L1WatcherQuerySender,
    ) -> Result<Self, RpcServiceError> {
        let mut modules = RpcModule::new(());
        modules
            .merge(HealthzApiServer::into_rpc(HealthzRpc {}))
            .map_err(Self::registration_error)?;
        modules
            .merge(P2pRpc::new(engine_admin.p2p_sender()).into_rpc())
            .map_err(Self::registration_error)?;
        modules
            .merge(RollupRpc::new(engine_rpc.clone(), l1_queries).into_rpc())
            .map_err(Self::registration_error)?;

        if config.enable_admin() {
            modules
                .merge(
                    AdminRpc::new(
                        Some(AdminControl::new(engine_admin.clone(), derivation_admin)),
                        engine_admin.network_admin_sender(),
                    )
                    .into_rpc(),
                )
                .map_err(Self::registration_error)?;
        }
        if config.dev_enabled() {
            modules
                .merge(DevEngineRpc::new(engine_rpc.clone()).into_rpc())
                .map_err(Self::registration_error)?;
        }
        if config.ws_enabled() {
            modules.merge(WsRPC::new(engine_rpc).into_rpc()).map_err(Self::registration_error)?;
        }

        Ok(Self { config, modules })
    }

    /// Runs until Node requests shutdown or the restart budget is exhausted.
    pub async fn run(self, mut shutdown: oneshot::Receiver<()>) -> Result<(), RpcServiceError> {
        let mut restarts = self.config.restart_count();
        loop {
            let handle = self.launch().await?;
            tokio::select! {
                biased;
                _ = &mut shutdown => {
                    let _ = handle.stop();
                    handle.stopped().await;
                    return Ok(());
                }
                () = handle.clone().stopped() => {}
            }

            if restarts == 0 {
                return Err(RpcServiceError::ServerStopped);
            }
            restarts = restarts.saturating_sub(1);
        }
    }

    async fn launch(&self) -> Result<ServerHandle, RpcServiceError> {
        let middleware = tower::ServiceBuilder::new()
            .layer(
                ProxyGetRequestLayer::new([("/healthz", "healthz")])
                    .expect("static health proxy configuration is valid"),
            )
            .timeout(Duration::from_secs(2));
        let server = Server::builder()
            .set_http_middleware(middleware)
            .build(self.config.socket)
            .await
            .map_err(RpcServiceError::Bind)?;
        match server.local_addr() {
            Ok(addr) => info!(target: "rpc", addr = ?addr, "RPC server bound to address"),
            Err(error) => warn!(target: "rpc", ?error, "Failed to read RPC server address"),
        }
        Ok(server.start(self.modules.clone()))
    }

    fn registration_error(error: impl core::fmt::Display) -> RpcServiceError {
        RpcServiceError::Registration(error.to_string())
    }
}

/// Terminal RPC service failure.
#[derive(Debug, Error)]
pub enum RpcServiceError {
    /// Namespace registration failed.
    #[error("RPC method registration failed: {0}")]
    Registration(String),
    /// Server bind failed.
    #[error("RPC server bind failed: {0}")]
    Bind(std::io::Error),
    /// Server stopped after exhausting its restart budget.
    #[error("RPC server stopped")]
    ServerStopped,
}
