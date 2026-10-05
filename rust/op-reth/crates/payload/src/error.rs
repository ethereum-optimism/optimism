//! Error type

/// Optimism specific payload building errors.
#[derive(Debug, thiserror::Error)]
pub enum OpPayloadBuilderError {
    /// Thrown when a transaction fails to convert to a
    /// [`alloy_consensus::transaction::Recovered`].
    #[error("failed to convert deposit transaction to RecoveredTx")]
    TransactionEcRecoverFailed,
    /// Thrown when the L1 block info could not be parsed from the calldata of the
    /// first transaction supplied in the payload attributes.
    #[error("failed to parse L1 block info from L1 info tx calldata")]
    L1BlockInfoParseFailed,
    /// Thrown when a database account could not be loaded.
    #[error("failed to load account {0}")]
    AccountLoadFailed(alloy_primitives::Address),
    /// Thrown when force deploy of create2deployer code fails.
    #[error("failed to force create2deployer account code")]
    ForceCreate2DeployerFail,
    /// Thrown when a blob transaction is included in a sequencer's block.
    #[error("blob transaction included in sequencer block")]
    BlobTransactionRejected,
}

/// Error returned by [`crate::builder::validate_derived_attributes`].
#[derive(Debug, thiserror::Error)]
pub enum DerivedAttributesError {
    /// The transactions make the payload invalid.
    #[error("invalid derived payload attributes: {0}")]
    InvalidPayload(reth_payload_builder_primitives::PayloadBuilderError),
    /// Validation failed for a node-local reason.
    #[error("failed to validate derived payload attributes: {0}")]
    Other(reth_payload_builder_primitives::PayloadBuilderError),
}
