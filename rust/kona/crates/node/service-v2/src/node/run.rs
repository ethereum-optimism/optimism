//! Node composition root, startup handshake, and explicit task ownership.

use crate::{
    derivation::{
        DelegatedDerivationService, DerivationAdminAdapter, DerivationDelegateClient,
        DerivationService, ServicePipeline,
    },
    engine::{
        EngineAdminAdapter, EngineConfig, EngineController, EngineHandle, EngineRpcAdapter,
        EngineRuntimeConfig, EngineService, EngineStarted, SequencerConfig,
    },
    l1::{L1Access, L1Reader, L1Watcher},
    network::{NetworkBuilder, NetworkConfig},
    node::{DerivationDelegateConfig, L1Config},
    rpc::RpcService,
};
use alloy_provider::RootProvider;
use kona_genesis::RollupConfig;
use kona_interop::DependencySet;
use kona_providers_alloy::{AlloyChainProvider, AlloyL2ChainProvider, OnlineBlobProvider};
use kona_rpc::RpcBuilder;
use op_alloy_network::Optimism;
use std::{
    future::{Future, pending},
    pin::Pin,
    sync::Arc,
};
use tokio::{
    sync::oneshot,
    task::{JoinError, JoinHandle},
};

const PROVIDER_CACHE_SIZE: usize = 1024;

type ServiceFuture = Pin<Box<dyn Future<Output = Result<(), String>> + Send>>;
type ServiceTask = JoinHandle<Result<(), String>>;

/// Fully configured service-oriented rollup node.
#[derive(Debug)]
pub struct RollupNode {
    pub(crate) config: Arc<RollupConfig>,
    pub(crate) l1: L1Config,
    pub(crate) l2_provider: RootProvider<Optimism>,
    pub(crate) l2_trust_rpc: bool,
    pub(crate) engine_config: EngineConfig,
    pub(crate) network_config: NetworkConfig,
    pub(crate) rpc_config: Option<RpcBuilder>,
    pub(crate) sequencer_config: SequencerConfig,
    pub(crate) delegate_config: Option<DerivationDelegateConfig>,
    pub(crate) dependency_set: Option<Arc<DependencySet>>,
}

impl RollupNode {
    /// Starts Engine, Derivation, and RPC, then waits for a signal or critical task exit.
    pub async fn start(self) -> Result<(), String> {
        let Some(mut running) = self.launch().await? else {
            return Ok(());
        };
        let cause = tokio::select! {
            signal = shutdown_signal() => signal.err(),
            failure = running.wait_for_critical_exit() => Some(failure),
        };

        info!(target: "rollup_node", ?cause, "Shutting down node services");
        let cleanup = running.shutdown().await.err();
        cause.or(cleanup).map_or(Ok(()), Err)
    }

    async fn launch(self) -> Result<Option<RunningNode>, String> {
        // Perform all fallible construction possible before spawning a long-running task.
        let delegate = self
            .delegate_config
            .clone()
            .map(|config| DerivationDelegateClient::new(config.l2_cl_url))
            .transpose()
            .map_err(|error| error.to_string())?;
        let l1_chain_provider = AlloyChainProvider::new_with_trust(
            self.l1.provider.clone(),
            PROVIDER_CACHE_SIZE,
            self.l1.trust_rpc,
        );
        let blob_provider = OnlineBlobProvider::init(self.l1.beacon_client.clone()).await;
        let reader = L1Reader::new(self.l1.provider.clone(), l1_chain_provider, blob_provider);
        let (l1_watcher, l1) =
            L1Watcher::new(self.config.clone(), self.l1.provider.clone(), reader);
        let network = NetworkBuilder::from(self.network_config.clone())
            .build()
            .map_err(|error| format!("failed to build P2P network: {error}"))?
            .start()
            .await
            .map_err(|error| format!("failed to start P2P network: {error}"))?;

        let (l1_shutdown, l1_shutdown_rx) = oneshot::channel();
        let mut l1_task = tokio::spawn(async move {
            l1_watcher.run(l1_shutdown_rx).await.map_err(|error| error.to_string())
        });
        let raw_engine = Arc::new(self.engine_config.clone().build_client());
        let runtime = EngineRuntimeConfig {
            mode: self.engine_config.mode,
            network,
            l1: l1.reader(),
            l1_snapshots: l1.subscribe(),
            l1_chain_config: self.l1.chain_config.clone(),
            l2_provider: self.l2_provider.clone(),
            l2_trust_rpc: self.l2_trust_rpc,
            sequencer: self.sequencer_config.clone(),
            dependency_set: self.dependency_set.clone(),
        };
        let (engine_service, engine_controller) =
            EngineService::new(raw_engine, self.config.clone(), runtime);
        let (engine_started_tx, engine_started_rx) = oneshot::channel();
        let mut engine_task = tokio::spawn(async move {
            engine_service.run(engine_started_tx).await.map_err(|error| error.to_string())
        });

        enum Startup {
            Started(Result<EngineStarted, String>),
            EngineExited(String),
            L1Exited(String),
            Signal(Result<(), String>),
        }
        let startup = tokio::select! {
            started = engine_started_rx => Startup::Started(
                started.map_err(|_| "Engine stopped before startup synchronization completed".to_string())
            ),
            result = &mut engine_task => Startup::EngineExited(unexpected_exit("Engine", result)),
            result = &mut l1_task => Startup::L1Exited(unexpected_exit("L1 watcher", result)),
            signal = shutdown_signal() => Startup::Signal(signal),
        };

        let started = match startup {
            Startup::Started(Ok(started)) => started,
            Startup::Started(Err(error)) => {
                let _ = engine_controller.shutdown().await;
                let _ = engine_task.await;
                let _ = l1_shutdown.send(());
                let _ = l1_task.await;
                return Err(error);
            }
            Startup::EngineExited(error) => {
                let _ = l1_shutdown.send(());
                let _ = l1_task.await;
                return Err(error);
            }
            Startup::L1Exited(error) => {
                let _ = engine_controller.shutdown().await;
                let _ = engine_task.await;
                return Err(error);
            }
            Startup::Signal(signal) => {
                let _ = engine_controller.shutdown().await;
                let _ = engine_task.await;
                let _ = l1_shutdown.send(());
                let _ = l1_task.await;
                signal?;
                return Ok(None);
            }
        };
        let EngineStarted { handle, rpc, admin, safe_head } = started;

        let (derivation_shutdown, derivation_shutdown_rx) = oneshot::channel();
        let (derivation_future, derivation_admin) = self.build_derivation(
            delegate,
            handle.clone(),
            safe_head,
            l1.clone(),
            derivation_shutdown_rx,
        );
        let rpc_service = self
            .rpc_config
            .clone()
            .map(|config| {
                RpcService::new(
                    config,
                    rpc.clone(),
                    admin.clone(),
                    derivation_admin.clone(),
                    l1.query_sender(),
                )
                .map_err(|error| error.to_string())
            })
            .transpose();
        let rpc_service = match rpc_service {
            Ok(service) => service,
            Err(error) => {
                let _ = engine_controller.shutdown().await;
                let _ = engine_task.await;
                let _ = l1_shutdown.send(());
                let _ = l1_task.await;
                return Err(error);
            }
        };

        let derivation_task = tokio::spawn(derivation_future);
        let (rpc_shutdown, rpc_task) = rpc_service.map_or_else(
            || (None, None),
            |service| {
                let (shutdown, shutdown_rx) = oneshot::channel();
                let task = tokio::spawn(async move {
                    service.run(shutdown_rx).await.map_err(|error| error.to_string())
                });
                (Some(shutdown), Some(task))
            },
        );

        Ok(Some(RunningNode {
            rpc_shutdown,
            derivation_shutdown: Some(derivation_shutdown),
            l1_shutdown: Some(l1_shutdown),
            engine_controller,
            rpc_task,
            derivation_task: Some(derivation_task),
            engine_task: Some(engine_task),
            l1_task: Some(l1_task),
            // Keep every capability lane open until its owning service is explicitly stopped.
            _engine_handle: handle,
            _engine_rpc: rpc,
            _engine_admin: admin,
            _derivation_admin: derivation_admin,
            _l1: l1,
        }))
    }

    fn build_derivation(
        &self,
        delegate: Option<DerivationDelegateClient>,
        engine: EngineHandle,
        safe_head: kona_protocol::L2BlockInfo,
        l1: L1Access,
        shutdown: oneshot::Receiver<()>,
    ) -> (ServiceFuture, DerivationAdminAdapter) {
        if let Some(delegate) = delegate {
            let (service, admin) = DelegatedDerivationService::new(delegate, l1.reader(), engine);
            return (
                Box::pin(
                    async move { service.run(shutdown).await.map_err(|error| error.to_string()) },
                ),
                admin,
            );
        }

        let pipeline = self.create_pipeline(l1.reader());
        let (service, admin) = DerivationService::new(
            Box::new(pipeline),
            engine,
            safe_head,
            l1.reader(),
            l1.subscribe(),
        );
        (
            Box::pin(async move { service.run(shutdown).await.map_err(|error| error.to_string()) }),
            admin,
        )
    }

    fn create_pipeline(&self, l1: L1Reader) -> ServicePipeline {
        let l2 = AlloyL2ChainProvider::new_with_trust(
            self.l2_provider.clone(),
            self.config.clone(),
            PROVIDER_CACHE_SIZE,
            self.l2_trust_rpc,
        );
        ServicePipeline::new(
            self.config.clone(),
            self.l1.chain_config.clone(),
            l1,
            l2,
            self.dependency_set.clone(),
        )
    }
}

/// Owns all spawned top-level tasks and lifecycle senders.
struct RunningNode {
    rpc_shutdown: Option<oneshot::Sender<()>>,
    derivation_shutdown: Option<oneshot::Sender<()>>,
    l1_shutdown: Option<oneshot::Sender<()>>,
    engine_controller: EngineController,
    rpc_task: Option<ServiceTask>,
    derivation_task: Option<ServiceTask>,
    engine_task: Option<ServiceTask>,
    l1_task: Option<ServiceTask>,
    _engine_handle: EngineHandle,
    _engine_rpc: EngineRpcAdapter,
    _engine_admin: EngineAdminAdapter,
    _derivation_admin: DerivationAdminAdapter,
    _l1: L1Access,
}

impl RunningNode {
    async fn wait_for_critical_exit(&mut self) -> String {
        enum Service {
            Rpc,
            Derivation,
            Engine,
            L1,
        }
        let (service, result) = tokio::select! {
            result = poll_service(&mut self.rpc_task) => (Service::Rpc, result),
            result = poll_service(&mut self.derivation_task) => (Service::Derivation, result),
            result = poll_service(&mut self.engine_task) => (Service::Engine, result),
            result = poll_service(&mut self.l1_task) => (Service::L1, result),
        };
        let name = match service {
            Service::Rpc => {
                self.rpc_task.take();
                "RPC"
            }
            Service::Derivation => {
                self.derivation_task.take();
                "Derivation"
            }
            Service::Engine => {
                self.engine_task.take();
                "Engine"
            }
            Service::L1 => {
                self.l1_task.take();
                "L1 watcher"
            }
        };
        unexpected_exit(name, result)
    }

    /// Stops intake, drains dependencies in order, and joins every task.
    async fn shutdown(mut self) -> Result<(), String> {
        let mut first_error = None;

        if let Some(shutdown) = self.rpc_shutdown.take() {
            let _ = shutdown.send(());
        }
        join_service("RPC", self.rpc_task.take(), &mut first_error).await;

        if self.engine_task.is_some() &&
            let Err(error) = self.engine_controller.quiesce_unsafe().await
        {
            record_error(
                format!("failed to quiesce Engine unsafe processing: {error}"),
                &mut first_error,
            );
        }

        if let Some(shutdown) = self.derivation_shutdown.take() {
            let _ = shutdown.send(());
        }
        join_service("Derivation", self.derivation_task.take(), &mut first_error).await;

        if self.engine_task.is_some() &&
            let Err(error) = self.engine_controller.shutdown().await
        {
            record_error(format!("failed to shut down Engine: {error}"), &mut first_error);
        }
        join_service("Engine", self.engine_task.take(), &mut first_error).await;

        if let Some(shutdown) = self.l1_shutdown.take() {
            let _ = shutdown.send(());
        }
        join_service("L1 watcher", self.l1_task.take(), &mut first_error).await;

        first_error.map_or(Ok(()), Err)
    }
}

async fn poll_service(task: &mut Option<ServiceTask>) -> Result<Result<(), String>, JoinError> {
    match task {
        Some(task) => task.await,
        None => pending().await,
    }
}

fn unexpected_exit(name: &str, result: Result<Result<(), String>, JoinError>) -> String {
    match result {
        Ok(Ok(())) => format!("{name} service terminated unexpectedly"),
        Ok(Err(error)) => format!("{name} service failed: {error}"),
        Err(error) => format!("{name} service task panicked: {error}"),
    }
}

async fn join_service(name: &str, task: Option<ServiceTask>, first_error: &mut Option<String>) {
    let Some(task) = task else { return };
    let error = match task.await {
        Ok(Ok(())) => return,
        Ok(Err(error)) => format!("{name} service failed during shutdown: {error}"),
        Err(error) => format!("{name} service task panicked during shutdown: {error}"),
    };
    record_error(error, first_error);
}

fn record_error(error: String, first_error: &mut Option<String>) {
    error!(target: "rollup_node", %error, "Service shutdown failed");
    if first_error.is_none() {
        *first_error = Some(error);
    }
}

async fn shutdown_signal() -> Result<(), String> {
    let ctrl_c = async {
        tokio::signal::ctrl_c()
            .await
            .map_err(|error| format!("failed to install Ctrl-C handler: {error}"))
    };
    #[cfg(unix)]
    let terminate = async {
        let mut signal = tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
            .map_err(|error| format!("failed to install SIGTERM handler: {error}"))?;
        signal.recv().await;
        Ok(())
    };
    #[cfg(not(unix))]
    let terminate = pending::<Result<(), String>>();

    tokio::select! {
        result = ctrl_c => result,
        result = terminate => result,
    }
}
