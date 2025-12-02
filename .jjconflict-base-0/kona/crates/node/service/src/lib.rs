#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/op-rs/kona/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]

#[macro_use]
extern crate tracing;

mod service;
pub use service::{
    DerivationDelegateConfig, InteropMode, L1Config, L1ConfigBuilder, NodeMode, RollupNode,
    RollupNodeBuilder,
};

mod actors;
pub use actors::{
    BlockStream, BuildRequest, CancellableContext, Conductor, ConductorClient, ConductorError,
    DelayedL1OriginSelectorProvider, DelegateDerivationActor, DerivationActor,
    DerivationActorRequest, DerivationClientError, DerivationClientResult,
    DerivationDelegateClient, DerivationDelegateClientError, DerivationEngineClient,
    DerivationError, DerivationState, DerivationStateMachine, DerivationStateTransitionError,
    DerivationStateUpdate, EngineActor, EngineActorRequest, EngineClientError, EngineClientResult,
    EngineConfig, EngineDerivationClient, EngineError, EngineProcessingRequest, EngineProcessor,
    EngineRequestReceiver, EngineRpcProcessor, EngineRpcRequest, EngineRpcRequestReceiver,
    L1OriginSelector, L1OriginSelectorError, L1OriginSelectorProvider, L1WatcherActor,
    L1WatcherActorError, L1WatcherDerivationClient, NetworkActor, NetworkActorError,
    NetworkBuilder, NetworkBuilderError, NetworkConfig, NetworkDriver, NetworkDriverError,
    NetworkEngineClient, NetworkHandler, NetworkInboundData, NodeActor, OriginSelector,
    QueuedDerivationEngineClient, QueuedEngineDerivationClient, QueuedEngineRpcClient,
    QueuedL1WatcherDerivationClient, QueuedNetworkEngineClient, QueuedSequencerAdminAPIClient,
    QueuedSequencerEngineClient, QueuedUnsafePayloadGossipClient, ResetRequest,
    RollupBoostAdminApiClient, RollupBoostHealthRpcClient, RpcActor, RpcActorError, RpcContext,
    SealRequest, SequencerActor, SequencerActorError, SequencerAdminQuery, SequencerConfig,
    SequencerEngineClient, UnsafePayloadGossipClient, UnsafePayloadGossipClientError,
};

mod metrics;
pub use metrics::Metrics;

#[cfg(test)]
pub use actors::{
    MockConductor, MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient,
};
