//! Construction of Engine-private unsafe, network, and signer workflows.

use crate::{
    engine::{
        EngineAdminAdapter,
        api::EngineInternalHandle,
        network::{NetworkClient, NetworkService},
        signer::SignerTracker,
        unsafe_chain::{
            Conductor, ConductorClient, DelayedL1OriginSelectorProvider, L1OriginSelector,
            SequencerConfig, SequencingWorkflow, SequencingWorkflowFactory, UnsafeChainService,
            UnsafeLifecycleCommand,
        },
    },
    l1::{L1Reader, L1Snapshot},
    network::NetworkHandler,
    node::NodeMode,
};
use alloy_provider::RootProvider;
use kona_derive::StatefulAttributesBuilder;
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_providers_alloy::AlloyL2ChainProvider;
use op_alloy_network::Optimism;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;
use tokio::sync::{mpsc, watch};

const PROVIDER_CACHE_SIZE: usize = 1024;
const UNSAFE_PAYLOAD_CAPACITY: usize = 256;
const SIGNER_UPDATE_CAPACITY: usize = 16;

/// Dependencies used to construct Engine's private unsafe-processing graph.
#[derive(Debug)]
pub(crate) struct EngineRuntimeConfig {
    /// Rollup mode determines whether local production can be enabled.
    pub mode: NodeMode,
    /// Started P2P stack, owned from this point onward by Engine.
    pub network: NetworkHandler,
    /// Engine-owned direct L1 reader.
    pub l1: L1Reader,
    /// Canonical L1 targets observed independently by Engine.
    pub l1_snapshots: watch::Receiver<L1Snapshot>,
    /// L1 consensus configuration used for payload attributes.
    pub l1_chain_config: Arc<L1ChainConfig>,
    /// L2 provider used by payload-attribute construction.
    pub l2_provider: RootProvider<Optimism>,
    /// Whether L2 responses may be trusted.
    pub l2_trust_rpc: bool,
    /// Local production configuration.
    pub sequencer: SequencerConfig,
    /// Interop dependency set required by Lagoon attributes.
    pub dependency_set: Option<Arc<DependencySet>>,
}

/// Constructed private graph retained and supervised by [`super::EngineService`].
pub(super) struct EngineRuntime {
    pub(super) network: NetworkService,
    pub(super) signer: SignerTracker,
    pub(super) unsafe_chain: UnsafeChainService,
    pub(super) unsafe_lifecycle: mpsc::Sender<UnsafeLifecycleCommand>,
    pub(super) admin: EngineAdminAdapter,
}

impl core::fmt::Debug for EngineRuntime {
    fn fmt(&self, formatter: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        formatter.debug_struct("EngineRuntime").finish_non_exhaustive()
    }
}

impl EngineRuntimeConfig {
    pub(super) fn build(
        self,
        config: Arc<RollupConfig>,
        engine: EngineInternalHandle,
    ) -> EngineRuntime {
        let Self {
            mode,
            network: network_handler,
            l1,
            l1_snapshots,
            l1_chain_config,
            l2_provider,
            l2_trust_rpc,
            sequencer,
            dependency_set,
        } = self;
        let initial_signer = *network_handler.unsafe_block_signer_sender.borrow();
        let (signer_tx, signer_rx) = mpsc::channel(SIGNER_UPDATE_CAPACITY);
        let (payload_tx, payload_rx) =
            mpsc::channel::<OpExecutionPayloadEnvelope>(UNSAFE_PAYLOAD_CAPACITY);
        let (network, network_client) = NetworkService::new(network_handler, signer_rx, payload_tx);
        let signer = SignerTracker::new(
            config.clone(),
            l1.clone(),
            l1_snapshots.clone(),
            signer_tx,
            initial_signer,
        );

        let (unsafe_chain, sequencer_handle, unsafe_lifecycle) = if mode.is_sequencer() {
            Self::build_sequencer(
                config,
                engine,
                network_client.clone(),
                payload_rx,
                l1,
                l1_snapshots,
                l1_chain_config,
                l2_provider,
                l2_trust_rpc,
                sequencer,
                dependency_set,
            )
        } else {
            UnsafeChainService::follower(engine, payload_rx)
        };
        let admin = EngineAdminAdapter::new(sequencer_handle, network_client);
        EngineRuntime { network, signer, unsafe_chain, unsafe_lifecycle, admin }
    }

    #[allow(clippy::too_many_arguments)]
    fn build_sequencer(
        config: Arc<RollupConfig>,
        engine: EngineInternalHandle,
        network: NetworkClient,
        payload_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
        l1: L1Reader,
        l1_snapshots: watch::Receiver<L1Snapshot>,
        l1_chain_config: Arc<L1ChainConfig>,
        l2_provider: RootProvider<Optimism>,
        l2_trust_rpc: bool,
        sequencer: SequencerConfig,
        dependency_set: Option<Arc<DependencySet>>,
    ) -> (
        UnsafeChainService,
        super::unsafe_chain::SequencerHandle,
        mpsc::Sender<UnsafeLifecycleCommand>,
    ) {
        let conductor = sequencer
            .conductor_rpc_url
            .clone()
            .map(ConductorClient::new_http)
            .map(|client| Arc::new(client) as Arc<dyn Conductor>);
        let factory_conductor = conductor.clone();
        let confirmation_depth = sequencer.l1_conf_delay;
        let workflow_engine = engine.clone();
        let workflow_network = network;
        let workflow_config = config;
        let factory = SequencingWorkflowFactory::new(
            move || {
                let delayed_l1 = DelayedL1OriginSelectorProvider::new(
                    l1.clone(),
                    l1_snapshots.clone(),
                    confirmation_depth,
                );
                let origin_selector = L1OriginSelector::new(workflow_config.clone(), delayed_l1);
                let attributes_builder = StatefulAttributesBuilder::new(
                    workflow_config.clone(),
                    l1_chain_config.clone(),
                    AlloyL2ChainProvider::new_with_trust(
                        l2_provider.clone(),
                        workflow_config.clone(),
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
                    workflow_config.clone(),
                )
            },
            factory_conductor,
        );
        UnsafeChainService::sequencer(
            engine,
            payload_rx,
            factory,
            !sequencer.sequencer_stopped,
            sequencer.sequencer_recovery_mode,
        )
    }
}
