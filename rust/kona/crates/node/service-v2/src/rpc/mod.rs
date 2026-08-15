//! JSON-RPC transport service and subsystem-owned administration routing.

use crate::{
    engine::EngineClient, l1::L1Client, network::NetworkClient, safe_chain::SafeChainHandle,
    unsafe_chain::SequencerHandle,
};
use alloy_primitives::B256;
use async_trait::async_trait;
use jsonrpsee::{
    RpcModule,
    server::{Server, ServerHandle, middleware::http::ProxyGetRequestLayer},
};
use kona_rpc::{
    AdminApiServer, AdminRpc, DevEngineApiServer, DevEngineRpc, HealthzApiServer, HealthzRpc,
    OpP2PApiServer, P2pRpc, RollupNodeApiServer, RollupRpc, RpcBuilder, SequencerAdminAPIClient,
    SequencerAdminAPIError, WsRPC, WsServer,
};
use std::time::Duration;
use thiserror::Error;
use tokio_util::sync::CancellationToken;

/// Routes the legacy admin RPC interface to the subsystem that owns each operation.
#[derive(Debug, Clone)]
pub struct AdminControl {
    sequencer: Option<SequencerHandle>,
    safe_chain: SafeChainHandle,
}

impl AdminControl {
    /// Creates subsystem-owned admin routing.
    pub const fn new(sequencer: Option<SequencerHandle>, safe_chain: SafeChainHandle) -> Self {
        Self { sequencer, safe_chain }
    }

    fn sequencer(&self) -> Result<&SequencerHandle, SequencerAdminAPIError> {
        self.sequencer.as_ref().ok_or_else(|| {
            SequencerAdminAPIError::RequestError("local sequencing is not configured".to_string())
        })
    }
}

#[async_trait]
impl SequencerAdminAPIClient for AdminControl {
    async fn is_sequencer_active(&self) -> Result<bool, SequencerAdminAPIError> {
        Ok(self.sequencer()?.status().active)
    }

    async fn is_conductor_enabled(&self) -> Result<bool, SequencerAdminAPIError> {
        Ok(self.sequencer()?.status().conductor_enabled)
    }

    async fn is_recovery_mode(&self) -> Result<bool, SequencerAdminAPIError> {
        Ok(self.sequencer()?.status().recovery_mode)
    }

    async fn start_sequencer(&self) -> Result<(), SequencerAdminAPIError> {
        self.sequencer()?.start().await
    }

    async fn stop_sequencer(&self) -> Result<B256, SequencerAdminAPIError> {
        self.sequencer()?.stop().await
    }

    async fn set_recovery_mode(&self, mode: bool) -> Result<(), SequencerAdminAPIError> {
        self.sequencer()?.set_recovery_mode(mode).await
    }

    async fn override_leader(&self) -> Result<(), SequencerAdminAPIError> {
        self.sequencer()?.override_leader().await
    }

    async fn reset_derivation_pipeline(&self) -> Result<(), SequencerAdminAPIError> {
        self.safe_chain
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
    /// Builds all configured RPC namespaces from narrow subsystem clients.
    pub fn new(
        config: RpcBuilder,
        engine: EngineClient,
        l1: L1Client,
        network: NetworkClient,
        sequencer: Option<SequencerHandle>,
        safe_chain: SafeChainHandle,
    ) -> Result<Self, RpcServiceError> {
        let mut modules = RpcModule::new(());
        modules
            .merge(HealthzApiServer::into_rpc(HealthzRpc {}))
            .map_err(Self::registration_error)?;
        modules
            .merge(P2pRpc::new(network.p2p_sender()).into_rpc())
            .map_err(Self::registration_error)?;
        modules
            .merge(RollupRpc::new(engine.clone(), l1.query_sender()).into_rpc())
            .map_err(Self::registration_error)?;

        if config.enable_admin() {
            modules
                .merge(
                    AdminRpc::new(
                        Some(AdminControl::new(sequencer, safe_chain)),
                        network.admin_sender(),
                    )
                    .into_rpc(),
                )
                .map_err(Self::registration_error)?;
        }
        if config.dev_enabled() {
            modules
                .merge(DevEngineRpc::new(engine.clone()).into_rpc())
                .map_err(Self::registration_error)?;
        }
        if config.ws_enabled() {
            modules.merge(WsRPC::new(engine).into_rpc()).map_err(Self::registration_error)?;
        }

        Ok(Self { config, modules })
    }

    /// Runs the server and applies its configured restart budget.
    pub async fn run(self, shutdown: CancellationToken) -> Result<(), RpcServiceError> {
        let mut restarts = self.config.restart_count();
        loop {
            let handle = self.launch().await?;
            tokio::select! {
                biased;
                _ = shutdown.cancelled() => {
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
