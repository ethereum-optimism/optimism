//! [NodeActor] services for the node.
//!
//! [NodeActor]: super::NodeActor

mod traits;
pub use traits::{CancellableContext, NodeActor};

mod engine;
pub use engine::{
    BlockBuildingClient, BlockEngineError, BlockEngineResult, BuildRequest, EngineActor,
    EngineConfig, EngineContext, EngineError, EngineInboundData, L2Finalizer,
    QueuedBlockBuildingClient, ResetRequest, SealRequest,
};

mod rpc;
pub use rpc::{RpcActor, RpcActorError, RpcContext};

mod derivation;
pub use derivation::{
    DerivationActor, DerivationBuilder, DerivationContext, DerivationError,
    DerivationInboundChannels, DerivationState, InboundDerivationMessage, PipelineBuilder,
};

mod l1_watcher;
pub use l1_watcher::{BlockStream, L1WatcherActor, L1WatcherActorError};

mod network;
pub use network::{
    NetworkActor, NetworkActorError, NetworkBuilder, NetworkBuilderError, NetworkConfig,
    NetworkContext, NetworkDriver, NetworkDriverError, NetworkHandler, NetworkInboundData,
    QueuedUnsafePayloadGossipClient, UnsafePayloadGossipClient, UnsafePayloadGossipClientError,
};

mod sequencer;
pub use sequencer::{
    Conductor, ConductorClient, ConductorError, DelayedL1OriginSelectorProvider, L1OriginSelector,
    L1OriginSelectorError, L1OriginSelectorProvider, OriginSelector, QueuedSequencerAdminAPIClient,
    SequencerActor, SequencerActorError, SequencerAdminQuery, SequencerConfig,
};

#[cfg(test)]
pub use engine::MockBlockBuildingClient;
#[cfg(test)]
pub use network::MockUnsafePayloadGossipClient;
#[cfg(test)]
pub use sequencer::{MockConductor, MockOriginSelector};
