#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

#[macro_use]
extern crate tracing;

/// Semantic execution-engine access and forkchoice reconciliation.
pub mod engine;
/// Shared L1 data access.
pub mod l1;
/// Network transport integration.
pub mod network;
/// Node composition and task supervision.
pub mod node;
/// RPC transport and subsystem control integration.
pub mod rpc;
/// Safe and finalized chain derivation from L1.
pub mod safe_chain;
/// Unsafe chain acquisition through local sequencing or network following.
pub mod unsafe_chain;

// The copied V1 runtime remains available while the V2 modules replace it one subsystem at a
// time. Keeping this implementation in the V2 crate provides a behaviorally equivalent binary for
// acceptance-test comparison without modifying the original service crate.
mod service;
pub use service::{
    DerivationDelegateConfig, InteropMode, L1Config, L1ConfigBuilder, NodeMode, RollupNode,
    RollupNodeBuilder,
};

mod actors;
pub use actors::{
    BlockStream, BuildRequest, Conductor, ConductorClient, ConductorError,
    DelayedL1OriginSelectorProvider, DelegateDerivationActor, DerivationActor,
    DerivationActorRequest, DerivationClientError, DerivationClientResult,
    DerivationDelegateClient, DerivationDelegateClientError, DerivationDelegateProvider,
    DerivationEngineClient, DerivationError, DerivationState, DerivationStateMachine,
    DerivationStateTransitionError, DerivationStateUpdate, EngineActor, EngineActorRequest,
    EngineClientError, EngineClientResult, EngineConfig, EngineDerivationClient, EngineError,
    EngineRpcActor, EngineRpcRequest, JsonrpseeServerLauncher, L1OriginSelector,
    L1OriginSelectorError, L1OriginSelectorProvider, L1WatcherActor, L1WatcherActorError,
    L1WatcherDerivationClient, NetworkActor, NetworkActorError, NetworkBuilder,
    NetworkBuilderError, NetworkConfig, NetworkDriver, NetworkDriverError, NetworkEngineClient,
    NetworkHandler, NodeActor, OriginSelector, QueuedDerivationEngineClient,
    QueuedEngineDerivationClient, QueuedEngineRpcClient, QueuedL1WatcherDerivationClient,
    QueuedNetworkEngineClient, QueuedSequencerAdminAPIClient, QueuedSequencerEngineClient,
    QueuedUnsafePayloadGossipClient, ResetRequest, RpcActor, RpcActorError, RpcServerHandle,
    RpcServerLauncher, SealRequest, SequencerActor, SequencerActorError, SequencerAdminQuery,
    SequencerConfig, SequencerEngineClient, UnsafePayloadGossipClient,
    UnsafePayloadGossipClientError,
};

mod metrics;
pub use metrics::Metrics;

#[cfg(test)]
pub use actors::{
    MockConductor, MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient,
};
