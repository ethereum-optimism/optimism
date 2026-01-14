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
    InteropMode, L1Config, L1ConfigBuilder, NodeMode, RollupNode, RollupNodeBuilder,
};

mod actors;
pub use actors::{
    BlockBuildingClient, BlockEngineError, BlockEngineResult, BlockStream, BuildRequest,
    CancellableContext, Conductor, ConductorClient, ConductorError,
    DelayedL1OriginSelectorProvider, DerivationActor, DerivationBuilder, DerivationContext,
    DerivationError, DerivationInboundChannels, DerivationState, EngineActor, EngineConfig,
    EngineContext, EngineError, EngineInboundData, InboundDerivationMessage, L1OriginSelector,
    L1OriginSelectorError, L1OriginSelectorProvider, L1WatcherActor, L1WatcherActorError,
    L2Finalizer, NetworkActor, NetworkActorError, NetworkBuilder, NetworkBuilderError,
    NetworkConfig, NetworkContext, NetworkDriver, NetworkDriverError, NetworkHandler,
    NetworkInboundData, NodeActor, OriginSelector, PipelineBuilder, QueuedBlockBuildingClient,
    QueuedSequencerAdminAPIClient, QueuedUnsafePayloadGossipClient, ResetRequest, RpcActor,
    RpcActorError, RpcContext, SealRequest, SequencerActor, SequencerActorError,
    SequencerAdminQuery, SequencerConfig, UnsafePayloadGossipClient,
    UnsafePayloadGossipClientError,
};

mod metrics;
pub use metrics::Metrics;

#[cfg(test)]
pub use actors::{
    MockBlockBuildingClient, MockConductor, MockOriginSelector, MockUnsafePayloadGossipClient,
};
