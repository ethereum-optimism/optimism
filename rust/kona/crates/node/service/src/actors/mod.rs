//! [`NodeActor`] services for the node.
//!
//! [NodeActor]: super::NodeActor

mod traits;
pub use traits::NodeActor;

mod engine;
pub use engine::{
    BuildRequest, EngineActor, EngineActorRequest, EngineClientError, EngineClientResult,
    EngineConfig, EngineDerivationClient, EngineError, EngineRpcActor, EngineRpcRequest,
    QueuedEngineDerivationClient, ResetRequest, SealRequest,
};

pub(crate) mod rpc;
pub use rpc::{
    JsonrpseeServerLauncher, QueuedEngineRpcClient, QueuedSequencerAdminAPIClient, RpcActor,
    RpcActorError, RpcServerHandle, RpcServerLauncher,
};

mod derivation;
pub use derivation::{
    DelegateDerivationActor, DerivationActor, DerivationActorRequest, DerivationClientError,
    DerivationClientResult, DerivationDelegateClient, DerivationDelegateClientError,
    DerivationDelegateProvider, DerivationEngineClient, DerivationError, DerivationState,
    DerivationStateMachine, DerivationStateTransitionError, DerivationStateUpdate,
    QueuedDerivationEngineClient,
};

mod l1_watcher;
pub use l1_watcher::{
    BlockStream, L1WatcherActor, L1WatcherActorError, L1WatcherDerivationClient,
    QueuedL1WatcherDerivationClient,
};

mod network;
pub use network::{
    NetworkActor, NetworkActorError, NetworkBuilder, NetworkBuilderError, NetworkConfig,
    NetworkDriver, NetworkDriverError, NetworkEngineClient, NetworkHandler,
    QueuedNetworkEngineClient, QueuedUnsafePayloadGossipClient, UnsafePayloadGossipClient,
    UnsafePayloadGossipClientError,
};

mod sequencer;

pub use sequencer::{
    Conductor, ConductorClient, ConductorError, DelayedL1OriginSelectorProvider, L1OriginSelector,
    L1OriginSelectorError, L1OriginSelectorProvider, OriginSelector, QueuedSequencerEngineClient,
    SequencerActor, SequencerActorError, SequencerAdminQuery, SequencerConfig,
    SequencerEngineClient,
};

#[cfg(test)]
pub use network::MockUnsafePayloadGossipClient;
#[cfg(test)]
pub use sequencer::{MockConductor, MockOriginSelector, MockSequencerEngineClient};
