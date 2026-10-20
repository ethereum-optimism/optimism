use alloy_primitives::B256;
use async_trait::async_trait;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::fmt::Debug;
use thiserror::Error;

/// Engine-owned admin operations, independent of the node's sequencing mode.
#[async_trait]
pub trait EngineAdmin: Send + Sync {
    /// A failure from the engine owner.
    type Error: std::error::Error + Send + Sync + 'static;

    /// Waits for reliable payload admission, before execution. Submitted payloads are not gossiped.
    async fn post_unsafe_payload(
        &self,
        payload: OpExecutionPayloadEnvelope,
    ) -> Result<(), Self::Error>;

    /// Waits until the engine has completed and published its reset.
    async fn reset_derivation_pipeline(&self) -> Result<(), Self::Error>;
}

/// Client trait wrapping RPC implementation for the Sequencer admin endpoints.
#[async_trait]
pub trait SequencerAdminAPIClient: Send + Sync + Debug {
    /// Check if the sequencer is active.
    async fn is_sequencer_active(&self) -> Result<bool, SequencerAdminAPIError>;

    /// Check if the conductor is enabled.
    async fn is_conductor_enabled(&self) -> Result<bool, SequencerAdminAPIError>;

    /// Check if in recovery mode.
    async fn is_recovery_mode(&self) -> Result<bool, SequencerAdminAPIError>;

    /// Start the sequencer.
    async fn start_sequencer(&self, expected: B256) -> Result<(), SequencerAdminAPIError>;

    /// Stop the sequencer.
    async fn stop_sequencer(&self) -> Result<B256, SequencerAdminAPIError>;

    /// Set recovery mode.
    async fn set_recovery_mode(&self, mode: bool) -> Result<(), SequencerAdminAPIError>;

    /// Override the leader.
    async fn override_leader(&self) -> Result<(), SequencerAdminAPIError>;
}

/// Errors that can occur when using the sequencer admin API.
#[derive(Debug, Error)]
pub enum SequencerAdminAPIError {
    /// Activation requires a stopped sequencer.
    #[error("sequencer already running")]
    AlreadyRunning,
    /// A stopped sequencer has no stop operation to settle.
    #[error("sequencer not running")]
    NotRunning,
    /// Conductor leadership is required before activation.
    #[error("sequencer is not the leader, aborting")]
    NotLeader,
    /// The conductor could not establish leadership.
    #[error("sequencer leader check failed: {0}")]
    LeaderCheckFailed(String),
    /// Execution has not published a usable current head.
    #[error("no prestate, cannot determine if sequencer start at {0} is safe")]
    NoPrestate(B256),
    /// The operator's expected head differs from the current engine head.
    #[error("block hash does not match: head {hash}:{number}, received {received}", hash = .head.block_info.hash, number = .head.block_info.number)]
    HeadMismatch {
        /// Actual published unsafe head after the leadership check.
        head: kona_protocol::L2BlockInfo,
        /// Expected unsafe head supplied by the caller.
        received: B256,
    },

    /// Error sending request.
    #[error("Error sending request: {0}.")]
    RequestError(String),

    /// Error receiving response.
    /// Note: this error message is not future-proof, in that it may not be a safe assumption that
    /// communication is channel-based. If/when that changes the enum will likely need to be
    /// updated to take a parameter, so we can change it then.
    #[error("Error receiving response: response channel closed.")]
    ResponseError,

    /// Sequencer stopped successfully, followed by some error.
    #[error("Sequencer stopped successfully, followed by error: {0}.")]
    ErrorAfterSequencerWasStopped(String),
}

impl SequencerAdminAPIError {
    /// Domain rejections use the same server-error category as op-node.
    pub const fn is_domain_error(&self) -> bool {
        matches!(
            self,
            Self::AlreadyRunning |
                Self::NotRunning |
                Self::NotLeader |
                Self::LeaderCheckFailed(_) |
                Self::NoPrestate(_) |
                Self::HeadMismatch { .. }
        )
    }
}
