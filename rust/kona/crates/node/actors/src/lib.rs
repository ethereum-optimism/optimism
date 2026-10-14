#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

#[macro_use]
extern crate tracing;

pub mod engine;
pub use engine::{EngineClientError, EngineClientResult};

pub mod derivation;
pub use derivation::{
    DerivationDelegateClient, DerivationDelegateClientError, DerivationDelegateProvider,
};

pub mod discovery;

pub mod finality;

pub mod l1_signer_updater;

pub mod l1;

pub mod signer;

pub mod network;
pub use network::NetworkConfig;

pub mod sequencer;

pub use sequencer::{
    Conductor, ConductorClient, ConductorError, DelayedL1OriginSelectorProvider, L1OriginSelector,
    L1OriginSelectorError, L1OriginSelectorProvider, OriginSelector, SequencerConfig,
    SequencerEngineClient,
};

#[cfg(test)]
pub use sequencer::{MockConductor, MockOriginSelector, MockSequencerEngineClient};

mod metrics;
pub use metrics::Metrics;

pub use derivation::DerivationStatus;

/// Drops the value a [`TrySendError`](tokio::sync::mpsc::error::TrySendError) returns, for
/// handles whose callers have no use for it.
pub(crate) fn without_value<T>(
    error: tokio::sync::mpsc::error::TrySendError<T>,
) -> tokio::sync::mpsc::error::TrySendError<()> {
    use tokio::sync::mpsc::error::TrySendError;
    match error {
        TrySendError::Full(_) => TrySendError::Full(()),
        TrySendError::Closed(_) => TrySendError::Closed(()),
    }
}

/// An empty unsafe payload with the given block number, for tests.
#[cfg(test)]
pub(crate) fn test_payload(number: u64) -> op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope {
    let mut block = alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default();
    block.header.number = number;
    op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope::V1(
        alloy_rpc_types_engine::ExecutionPayloadV1::from_block_slow(&block),
    )
}
