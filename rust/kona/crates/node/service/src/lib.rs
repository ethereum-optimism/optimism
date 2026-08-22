#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

#[macro_use]
extern crate tracing;

mod service;
pub use service::{
    BoxedNodeActor, ComposedChain, DerivationDelegateConfig, ErasedNodeActor, IntoBoxedNodeActor,
    L1Config, L1ConfigBuilder, L1WatcherPorts, NodeMode, RollupNode, RollupNodeBuilder, run_actors,
};

mod actors;
pub use actors::{
    BlockStream, BuildRequest, ChainController, ChainControllerClientError,
    ChainControllerClientResult, ChainControllerDerivationClient, ChainControllerError,
    ChainControllerRequest, ChainControllerRpcActor, ChainControllerRpcRequest, Conductor,
    ConductorClient, ConductorError, DelayedL1OriginSelectorProvider, DelegateDerivationActor,
    DerivationActor, DerivationActorRequest, DerivationClientError, DerivationClientResult,
    DerivationDelegateClient, DerivationDelegateClientError, DerivationDelegateProvider,
    DerivationEngineClient, DerivationError, DerivationState, DerivationStateMachine,
    DerivationStateTransitionError, DerivationStateUpdate, EngineConfig, JsonrpseeServerLauncher,
    L1OriginSelector, L1OriginSelectorError, L1OriginSelectorProvider, L1WatcherActor,
    L1WatcherActorError, L1WatcherDerivationClient, NetworkActor, NetworkActorError,
    NetworkBuilder, NetworkBuilderError, NetworkConfig, NetworkDriver, NetworkDriverError,
    NetworkEngineClient, NetworkHandler, NodeActor, OriginSelector,
    QueuedChainControllerDerivationClient, QueuedDerivationEngineClient, QueuedEngineRpcClient,
    QueuedL1WatcherDerivationClient, QueuedNetworkEngineClient, QueuedSequencerAdminAPIClient,
    QueuedSequencerEngineClient, QueuedUnsafePayloadGossipClient, ResetRequest, RpcActor,
    RpcActorError, RpcServerHandle, RpcServerLauncher, SealRequest, SequencerActor,
    SequencerActorError, SequencerAdminQuery, SequencerConfig, SequencerEngineClient,
    UnsafePayloadGossipClient, UnsafePayloadGossipClientError,
};

mod metrics;
pub use metrics::Metrics;

#[cfg(test)]
pub use actors::{
    MockConductor, MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient,
};
