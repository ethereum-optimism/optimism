//! [`NodeActor`] services for the node.
//!
//! [NodeActor]: super::NodeActor

mod traits;
pub use traits::NodeActor;

pub(crate) mod engine;
pub use engine::{
    BuildToken, Consolidated, EngineActor, EngineClientError, EngineClientResult, EngineConfig,
    EngineContext, EngineError, EngineHandle, Sealed, UnsafeAdmissionError,
};

pub(crate) mod rpc;
pub use rpc::{
    JsonrpseeServerLauncher, QueuedSequencerAdminAPIClient, RpcActor, RpcActorError,
    RpcServerHandle, RpcServerLauncher,
};

mod derivation;
pub use derivation::{
    DelegateDerivationActor, DerivationActor, DerivationActorRequest, DerivationClientError,
    DerivationClientResult, DerivationDelegateClient, DerivationDelegateClientError,
    DerivationDelegateProvider, DerivationError,
};

mod l1_watcher;
pub use l1_watcher::{
    BlockStream, L1WatcherActor, L1WatcherActorError, L1WatcherChain, L1WatcherDerivationClient,
    QueuedL1WatcherDerivationClient,
};

mod signer;
pub use signer::{SignedPayload, SignerActor, SignerActorError};

mod network;
pub use network::{
    NetworkActor, NetworkActorError, NetworkBuilder, NetworkBuilderError, NetworkConfig,
    NetworkDriver, NetworkDriverError, NetworkEngineClient, NetworkHandler,
    QueuedUnsafePayloadGossipClient, UnsafePayloadGossipClient, UnsafePayloadGossipClientError,
};

mod sequencer;

pub use sequencer::{
    Conductor, ConductorClient, ConductorError, DelayedL1OriginSelectorProvider, L1OriginSelector,
    L1OriginSelectorError, L1OriginSelectorProvider, OriginSelector, SequencerActor,
    SequencerActorError, SequencerAdminQuery, SequencerConfig,
};

#[cfg(test)]
pub use network::MockUnsafePayloadGossipClient;
#[cfg(test)]
pub use sequencer::{MockConductor, MockOriginSelector};
