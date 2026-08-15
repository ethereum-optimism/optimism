//! Node composition root and structured service supervision.

use crate::{
    engine::{EngineClient, EngineConfig, EngineService},
    l1::{L1Client, L1Service},
    network::{NetworkBuilder, NetworkClient, NetworkConfig, NetworkService},
    node::{DerivationDelegateConfig, InteropMode, L1Config},
    rpc::RpcService,
    safe_chain::{
        DelegatedSafeChainService, DerivationDelegateClient, SafeChainHandle, SafeChainService,
        ServicePipeline,
    },
    unsafe_chain::{
        ConductorClient, DelayedL1OriginSelectorProvider, L1OriginSelector, SequencerConfig,
        SequencerHandle, SequencingWorkflow, SequencingWorkflowFactory, UnsafeChainService,
    },
};
use alloy_provider::RootProvider;
use kona_derive::StatefulAttributesBuilder;
use kona_genesis::RollupConfig;
use kona_interop::DependencySet;
use kona_providers_alloy::{AlloyChainProvider, AlloyL2ChainProvider, OnlineBlobProvider};
use kona_rpc::RpcBuilder;
use op_alloy_network::Optimism;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{future::Future, pin::Pin, sync::Arc};
use tokio::{sync::mpsc, task::JoinHandle};
use tokio_util::sync::CancellationToken;

const PROVIDER_CACHE_SIZE: usize = 1024;
const UNSAFE_PAYLOAD_CAPACITY: usize = 256;
const SIGNER_UPDATE_CAPACITY: usize = 16;

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
    pub(crate) interop_mode: InteropMode,
    pub(crate) delegate_config: Option<DerivationDelegateConfig>,
    pub(crate) dependency_set: Option<Arc<DependencySet>>,
}

impl RollupNode {
    /// Constructs all services, runs them concurrently, and shuts them down in dependency order.
    pub async fn start(&self) -> Result<(), String> {
        let chain_shutdown = CancellationToken::new();
        let engine_shutdown = CancellationToken::new();
        let transport_shutdown = CancellationToken::new();
        let rpc_shutdown = CancellationToken::new();

        let raw_engine = Arc::new(self.engine_config.clone().build_client());
        let (engine_service, engine) = EngineService::new(raw_engine, self.config.clone());

        let (signer_tx, signer_rx) = mpsc::channel(SIGNER_UPDATE_CAPACITY);
        let l1_chain_provider = AlloyChainProvider::new_with_trust(
            self.l1.provider.clone(),
            PROVIDER_CACHE_SIZE,
            self.l1.trust_rpc,
        );
        let blob_provider = OnlineBlobProvider::init(self.l1.beacon_client.clone()).await;
        let (l1_service, l1) = L1Service::new(
            self.config.clone(),
            self.l1.provider.clone(),
            l1_chain_provider,
            blob_provider,
            signer_tx,
        );

        let handler = NetworkBuilder::from(self.network_config.clone())
            .build()
            .map_err(|error| format!("failed to build P2P network: {error}"))?
            .start()
            .await
            .map_err(|error| format!("failed to start P2P network: {error}"))?;
        let (payload_tx, payload_rx) =
            mpsc::channel::<OpExecutionPayloadEnvelope>(UNSAFE_PAYLOAD_CAPACITY);
        let (network_service, network) = NetworkService::new(handler, signer_rx, payload_tx);

        let (unsafe_service, sequencer) =
            self.build_unsafe_chain(engine.clone(), network.clone(), l1.clone(), payload_rx);

        let (safe_future, safe_chain) = self
            .build_safe_chain(engine.clone(), l1.clone(), sequencer.clone(), chain_shutdown.clone())
            .await?;

        let rpc_service = self
            .rpc_config
            .clone()
            .map(|config| {
                RpcService::new(
                    config,
                    engine.clone(),
                    l1.clone(),
                    network.clone(),
                    sequencer.clone(),
                    safe_chain.clone(),
                )
                .map_err(|error| error.to_string())
            })
            .transpose()?;

        // Keep control capabilities alive even when RPC is disabled.
        let _control_keepalive: (
            SafeChainHandle,
            Option<SequencerHandle>,
            EngineClient,
            NetworkClient,
        ) = (safe_chain, sequencer, engine.clone(), network.clone());

        let (terminated_tx, mut terminated_rx) =
            mpsc::unbounded_channel::<(&'static str, Result<(), String>)>();

        let engine_task_shutdown = engine_shutdown.clone();
        let engine_task = spawn_service(
            "engine",
            async move {
                engine_service.run(engine_task_shutdown).await.map_err(|error| error.to_string())
            },
            terminated_tx.clone(),
        );
        let unsafe_task_shutdown = chain_shutdown.clone();
        let unsafe_task = spawn_service(
            "unsafe_chain",
            async move {
                unsafe_service.run(unsafe_task_shutdown).await.map_err(|error| error.to_string())
            },
            terminated_tx.clone(),
        );
        let safe_task = spawn_service("safe_chain", safe_future, terminated_tx.clone());
        let l1_task_shutdown = transport_shutdown.clone();
        let l1_task = spawn_service(
            "l1",
            async move { l1_service.run(l1_task_shutdown).await.map_err(|error| error.to_string()) },
            terminated_tx.clone(),
        );
        let network_task_shutdown = transport_shutdown.clone();
        let network_task = spawn_service(
            "network",
            async move {
                network_service.run(network_task_shutdown).await.map_err(|error| error.to_string())
            },
            terminated_tx.clone(),
        );
        let rpc_task_shutdown = rpc_shutdown.clone();
        let mut rpc_task =
            rpc_service.map(|service| {
                spawn_service(
                    "rpc",
                    async move {
                        service.run(rpc_task_shutdown).await.map_err(|error| error.to_string())
                    },
                    terminated_tx,
                )
            });

        let cause = tokio::select! {
            () = shutdown_signal() => None,
            terminated = terminated_rx.recv() => terminated,
        };

        info!(target: "rollup_node", ?cause, "Shutting down node services");

        let mut cleanup_error = None;

        // Reject new administrative work first.
        rpc_shutdown.cancel();
        if let Some(task) = rpc_task.take() {
            await_service_record("rpc", task, &mut cleanup_error).await;
        }

        // Local production observes cancellation at a block boundary; safe and unsafe workflows
        // finish any operation already accepted by the engine before returning.
        chain_shutdown.cancel();
        await_service_record("unsafe_chain", unsafe_task, &mut cleanup_error).await;
        await_service_record("safe_chain", safe_task, &mut cleanup_error).await;

        // No chain workflow can issue another Engine API operation after both have drained.
        engine_shutdown.cancel();
        await_service_record("engine", engine_task, &mut cleanup_error).await;

        // Shared transports remain available while dependent workflows drain.
        transport_shutdown.cancel();
        await_service_record("network", network_task, &mut cleanup_error).await;
        await_service_record("l1", l1_task, &mut cleanup_error).await;

        match cause {
            Some((name, Ok(()))) => Err(format!("{name} service terminated unexpectedly")),
            Some((name, Err(error))) => Err(format!("{name} service failed: {error}")),
            None => cleanup_error.map_or(Ok(()), Err),
        }
    }

    fn build_unsafe_chain(
        &self,
        engine: EngineClient,
        network: NetworkClient,
        l1: L1Client,
        payload_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
    ) -> (UnsafeChainService, Option<SequencerHandle>) {
        if !self.engine_config.mode.is_sequencer() {
            return (UnsafeChainService::follower(engine, payload_rx), None);
        }

        let conductor = self
            .sequencer_config
            .conductor_rpc_url
            .clone()
            .map(ConductorClient::new_http)
            .map(|client| Arc::new(client) as Arc<dyn crate::unsafe_chain::Conductor>);
        let factory_conductor = conductor.clone();
        let config = self.config.clone();
        let l1_chain_config = self.l1.chain_config.clone();
        let l1_head = l1.subscribe_head();
        let l2_provider = self.l2_provider.clone();
        let l2_trust_rpc = self.l2_trust_rpc;
        let dependency_set = self.dependency_set.clone();
        let confirmation_depth = self.sequencer_config.l1_conf_delay;
        let workflow_engine = engine.clone();
        let workflow_network = network;
        let factory = SequencingWorkflowFactory::new(
            move || {
                let delayed_l1 = DelayedL1OriginSelectorProvider::new(
                    l1.clone(),
                    l1_head.clone(),
                    confirmation_depth,
                );
                let origin_selector = L1OriginSelector::new(config.clone(), delayed_l1);
                let attributes_builder = StatefulAttributesBuilder::new(
                    config.clone(),
                    l1_chain_config.clone(),
                    AlloyL2ChainProvider::new_with_trust(
                        l2_provider.clone(),
                        config.clone(),
                        PROVIDER_CACHE_SIZE,
                        l2_trust_rpc,
                    ),
                    l1.clone(),
                    dependency_set.clone(),
                );
                SequencingWorkflow::new(
                    Box::new(attributes_builder),
                    conductor.clone(),
                    workflow_engine.clone(),
                    workflow_network.clone(),
                    Box::new(origin_selector),
                    config.clone(),
                )
            },
            factory_conductor,
        );
        let (service, handle) = UnsafeChainService::sequencer(
            engine,
            payload_rx,
            factory,
            !self.sequencer_config.sequencer_stopped,
            self.sequencer_config.sequencer_recovery_mode,
        );
        (service, Some(handle))
    }

    async fn build_safe_chain(
        &self,
        engine: EngineClient,
        l1: crate::l1::L1Client,
        sequencer: Option<SequencerHandle>,
        shutdown: CancellationToken,
    ) -> Result<(ServiceFuture, SafeChainHandle), String> {
        if let Some(delegate) = self.delegate_config.clone() {
            let delegate = DerivationDelegateClient::new(delegate.l2_cl_url)
                .map_err(|error| error.to_string())?;
            let (service, handle) = DelegatedSafeChainService::new(delegate, l1, engine, sequencer);
            return Ok((
                Box::pin(
                    async move { service.run(shutdown).await.map_err(|error| error.to_string()) },
                ),
                handle,
            ));
        }

        let pipeline = self.create_pipeline(l1.clone());
        let (service, handle) = SafeChainService::new(
            Box::new(pipeline),
            engine,
            l1.subscribe_head(),
            l1.subscribe_finalized(),
            sequencer,
        );
        Ok((
            Box::pin(async move { service.run(shutdown).await.map_err(|error| error.to_string()) }),
            handle,
        ))
    }

    fn create_pipeline(&self, l1: L1Client) -> ServicePipeline {
        let l2 = AlloyL2ChainProvider::new_with_trust(
            self.l2_provider.clone(),
            self.config.clone(),
            PROVIDER_CACHE_SIZE,
            self.l2_trust_rpc,
        );
        match self.interop_mode {
            InteropMode::Polled => ServicePipeline::new_polled(
                self.config.clone(),
                self.l1.chain_config.clone(),
                l1,
                l2,
                self.dependency_set.clone(),
            ),
            InteropMode::Indexed => ServicePipeline::new_indexed(
                self.config.clone(),
                self.l1.chain_config.clone(),
                l1,
                l2,
                self.dependency_set.clone(),
            ),
        }
    }
}

fn spawn_service(
    name: &'static str,
    future: impl Future<Output = Result<(), String>> + Send + 'static,
    terminated: mpsc::UnboundedSender<(&'static str, Result<(), String>)>,
) -> ServiceTask {
    tokio::spawn(async move {
        let result = future.await;
        let _ = terminated.send((name, result.clone()));
        result
    })
}

async fn await_service_record(name: &str, task: ServiceTask, first_error: &mut Option<String>) {
    let error = match task.await {
        Ok(Ok(())) => return,
        Ok(Err(error)) => format!("{name} service failed during shutdown: {error}"),
        Err(error) => format!("{name} service task panicked: {error}"),
    };
    error!(target: "rollup_node", %error, "Service shutdown failed");
    if first_error.is_none() {
        *first_error = Some(error);
    }
}

async fn shutdown_signal() {
    let ctrl_c = async {
        tokio::signal::ctrl_c().await.expect("failed to install Ctrl+C handler");
    };
    #[cfg(unix)]
    let terminate = async {
        tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
            .expect("failed to install SIGTERM handler")
            .recv()
            .await;
    };
    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {}
        _ = terminate => {}
    }
}
