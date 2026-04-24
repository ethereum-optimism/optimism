#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
// The workspace opts into a curated subset of pedantic/nursery lints via
// `[workspace.lints.clippy]`. The lints below are either stylistic,
// documentation-only, or would require architectural changes we intentionally
// avoid, so we allow them at the crate level.
#![allow(
    clippy::missing_errors_doc,
    clippy::missing_panics_doc,
    clippy::must_use_candidate,
    clippy::return_self_not_must_use,
    clippy::module_name_repetitions,
    clippy::redundant_pub_crate,
    clippy::too_many_lines,
    clippy::items_after_statements,
    clippy::cast_possible_truncation,
    clippy::cast_possible_wrap,
    clippy::cast_precision_loss,
    clippy::cast_sign_loss,
    clippy::cast_lossless,
    clippy::used_underscore_binding,
    clippy::unused_async,
    clippy::future_not_send,
    clippy::significant_drop_tightening,
    clippy::struct_field_names,
    clippy::similar_names,
    clippy::needless_pass_by_value,
    clippy::unused_self,
    clippy::too_long_first_doc_paragraph,
    clippy::struct_excessive_bools,
    clippy::inline_always,
    clippy::large_types_passed_by_value,
    clippy::ref_option
)]

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
    QueuedSequencerEngineClient, QueuedUnsafePayloadGossipClient, ResetRequest, RpcActor,
    RpcActorError, RpcContext, SealRequest, SequencerActor, SequencerActorError,
    SequencerAdminQuery, SequencerConfig, SequencerEngineClient, UnsafePayloadGossipClient,
    UnsafePayloadGossipClientError,
};

mod metrics;
pub use metrics::Metrics;

#[cfg(test)]
pub use actors::{
    MockConductor, MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient,
};
