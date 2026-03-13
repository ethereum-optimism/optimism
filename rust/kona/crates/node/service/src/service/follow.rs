//! Contains the [`FollowNode`] implementation — a lightweight node mode that polls a source L2
//! node and drives the local engine, bypassing full derivation and P2P networking.

use super::builder::{DerivationDelegateConfig, L1ConfigBuilder};
use crate::{
    DelegateDerivationActor, DerivationDelegateClient, EngineConfig, L1WatcherActor, NodeActor,
    QueuedDerivationEngineClient, QueuedEngineDerivationClient, QueuedEngineRpcClient,
    QueuedL1WatcherDerivationClient, QueuedSequencerAdminAPIClient, RollupBoostAdminApiClient,
    RollupBoostHealthRpcClient, RpcActor, RpcContext,
    actors::BlockStream,
    service::node::{L1Config, create_engine_actor},
};
use alloy_eips::BlockNumberOrTag;
use alloy_provider::RootProvider;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use kona_providers_alloy::{AlloyChainProvider, OnlineBeaconClient};
use kona_rpc::RpcBuilder;
use std::{sync::Arc, time::Duration};
use tokio::sync::{mpsc, watch};
use tokio_util::sync::CancellationToken;

const DERIVATION_PROVIDER_CACHE_SIZE: usize = 1024;
const HEAD_STREAM_POLL_INTERVAL: u64 = 4;
const FINALIZED_STREAM_POLL_INTERVAL: u64 = 60;

/// A lightweight node that follows a source L2 CL node via delegated derivation.
///
/// Unlike [`super::RollupNode`], this node does not run P2P networking, sequencing, or full
/// derivation. It spawns only:
/// - [`EngineActor`] — drives the local execution engine
/// - [`DelegateDerivationActor`] — polls `optimism_syncStatus` from the source CL
/// - [`L1WatcherActor`] — watches the L1 chain for system config updates
/// - [`RpcActor`] (optional) — serves RPC without P2P or admin endpoints
#[derive(Debug)]
pub struct FollowNode {
    /// The rollup configuration.
    config: Arc<RollupConfig>,
    /// The [`EngineConfig`] for the node.
    engine_config: EngineConfig,
    /// The derivation delegate client.
    derivation_delegate_provider: DerivationDelegateClient,
    /// The optional [`RpcBuilder`] for the node.
    rpc_builder: Option<RpcBuilder>,
    /// The L1 configuration.
    l1_config: L1Config,
}

impl FollowNode {
    /// Starts the follow node service.
    pub async fn start(&self) -> Result<(), String> {
        let cancellation = CancellationToken::new();

        let (derivation_actor_request_tx, derivation_actor_request_rx) = mpsc::channel(1024);
        let (engine_actor_request_tx, engine_actor_request_rx) = mpsc::channel(1024);
        let (unsafe_head_tx, _unsafe_head_rx) = watch::channel(L2BlockInfo::default());

        let engine_actor = create_engine_actor(
            &self.engine_config,
            self.config.clone(),
            false, // follow mode is never a sequencer
            cancellation.clone(),
            engine_actor_request_rx,
            QueuedEngineDerivationClient::new(derivation_actor_request_tx.clone()),
            unsafe_head_tx,
        )?;

        // L1 Provider for sanity checking Derivation Delegation
        let l1_provider = AlloyChainProvider::new(
            self.l1_config.engine_provider.clone(),
            DERIVATION_PROVIDER_CACHE_SIZE,
        );

        let derivation = DelegateDerivationActor::<_>::new(
            QueuedDerivationEngineClient {
                engine_actor_request_tx: engine_actor_request_tx.clone(),
            },
            cancellation.clone(),
            derivation_actor_request_rx,
            self.derivation_delegate_provider.clone(),
            l1_provider,
        );

        // Create the L1 Watcher actor
        let (l1_query_tx, l1_query_rx) = mpsc::channel(1024);
        let (l1_head_updates_tx, _l1_head_updates_rx) = watch::channel(None);

        let head_stream = BlockStream::new_as_stream(
            self.l1_config.engine_provider.clone(),
            BlockNumberOrTag::Latest,
            Duration::from_secs(HEAD_STREAM_POLL_INTERVAL),
        )?;
        let finalized_stream = BlockStream::new_as_stream(
            self.l1_config.engine_provider.clone(),
            BlockNumberOrTag::Finalized,
            Duration::from_secs(FINALIZED_STREAM_POLL_INTERVAL),
        )?;

        let l1_watcher = L1WatcherActor::new(
            self.config.clone(),
            self.l1_config.engine_provider.clone(),
            l1_query_rx,
            l1_head_updates_tx,
            QueuedL1WatcherDerivationClient { derivation_actor_request_tx },
            None, // No block signer sender — no network actor in follow mode
            cancellation.clone(),
            head_stream,
            finalized_stream,
        );

        // Create the RPC server actor (optional, no P2P or admin endpoints).
        let rpc = self.rpc_builder.clone().map(|b| {
            RpcActor::new(
                b,
                QueuedEngineRpcClient::new(engine_actor_request_tx.clone()),
                RollupBoostAdminApiClient {
                    engine_actor_request_tx: engine_actor_request_tx.clone(),
                },
                RollupBoostHealthRpcClient {
                    engine_actor_request_tx: engine_actor_request_tx.clone(),
                },
                None::<QueuedSequencerAdminAPIClient>, // No sequencer admin client
            )
        });

        crate::service::spawn_and_wait!(
            cancellation,
            actors = [
                rpc.map(|r| (
                    r,
                    RpcContext {
                        cancellation: cancellation.clone(),
                        p2p_network: None,
                        network_admin: None,
                        l1_watcher_queries: l1_query_tx,
                    }
                )),
                Some((l1_watcher, ())),
                Some((derivation, ())),
                Some((engine_actor, ())),
            ]
        );
        Ok(())
    }
}

/// Builder for the [`FollowNode`].
#[derive(Debug)]
pub struct FollowNodeBuilder {
    /// The rollup configuration.
    pub config: RollupConfig,
    /// The L1 chain configuration builder.
    pub l1_config_builder: L1ConfigBuilder,
    /// Engine builder configuration.
    pub engine_config: EngineConfig,
    /// An optional RPC Configuration.
    pub rpc_config: Option<RpcBuilder>,
    /// Configuration for Derivation Delegate mode (required for follow).
    pub derivation_delegate_config: DerivationDelegateConfig,
}

impl FollowNodeBuilder {
    /// Creates a new [`FollowNodeBuilder`].
    pub const fn new(
        config: RollupConfig,
        l1_config_builder: L1ConfigBuilder,
        engine_config: EngineConfig,
        rpc_config: Option<RpcBuilder>,
        derivation_delegate_config: DerivationDelegateConfig,
    ) -> Self {
        Self { config, l1_config_builder, engine_config, rpc_config, derivation_delegate_config }
    }

    /// Assembles the [`FollowNode`] service.
    pub fn build(self) -> FollowNode {
        let mut l1_beacon = OnlineBeaconClient::new_http(self.l1_config_builder.beacon.to_string());
        if let Some(l1_slot_duration) = self.l1_config_builder.slot_duration_override {
            l1_beacon = l1_beacon.with_l1_slot_duration_override(l1_slot_duration);
        }

        let l1_config = L1Config {
            chain_config: Arc::new(self.l1_config_builder.chain_config),
            trust_rpc: self.l1_config_builder.trust_rpc,
            beacon_client: l1_beacon,
            engine_provider: RootProvider::new_http(self.l1_config_builder.rpc_url.clone()),
        };

        let derivation_delegate_provider =
            DerivationDelegateClient::new(self.derivation_delegate_config.l2_cl_url.clone())
                .expect("Failed to create Derivation Delegate provider");

        FollowNode {
            config: Arc::new(self.config),
            engine_config: self.engine_config,
            derivation_delegate_provider,
            rpc_builder: self.rpc_config,
            l1_config,
        }
    }
}
