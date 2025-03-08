//! Error types for the Optimism EVM module.

use reth_evm::execute::BlockExecutionError;

/// L1 Block Info specific errors
#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
pub enum L1BlockInfoError {
    /// Could not find L1 block info transaction in the L2 block
    #[error("could not find l1 block info tx in the L2 block")]
    MissingTransaction,
    /// Invalid L1 block info transaction calldata
    #[error("invalid l1 block info transaction calldata in the L2 block")]
    InvalidCalldata,
    /// Unexpected L1 block info transaction calldata length
    #[error("unexpected l1 block info tx calldata length found")]
    UnexpectedCalldataLength,
    /// Base fee conversion error
    #[error("could not convert l1 base fee")]
    BaseFeeConversion,
    /// Fee overhead conversion error
    #[error("could not convert l1 fee overhead")]
    FeeOverheadConversion,
    /// Fee scalar conversion error
    #[error("could not convert l1 fee scalar")]
    FeeScalarConversion,
    /// Base Fee Scalar conversion error
    #[error("could not convert base fee scalar")]
    BaseFeeScalarConversion,
    /// Blob base fee conversion error
    #[error("could not convert l1 blob base fee")]
    BlobBaseFeeConversion,
    /// Blob base fee scalar conversion error
    #[error("could not convert l1 blob base fee scalar")]
    BlobBaseFeeScalarConversion,
    /// Operator fee scalar conversion error
    #[error("could not convert operator fee scalar")]
    OperatorFeeScalarConversion,
    /// Operator fee constant conversion error
    #[error("could not convert operator fee constant")]
    OperatorFeeConstantConversion,
    /// Optimism hardforks not active
    #[error("Optimism hardforks are not active")]
    HardforksNotActive,
}

/// Optimism Block Executor Errors
#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
pub enum OpBlockExecutionError {
    /// Error when trying to parse L1 block info
    #[error(transparent)]
    L1BlockInfo(#[from] L1BlockInfoError),
    /// Thrown when force deploy of create2deployer code fails.
    #[error("failed to force create2deployer account code")]
    ForceCreate2DeployerFail,
    /// Thrown when a database account could not be loaded.
    #[error("failed to load account {_0}")]
    AccountLoadFailed(alloy_primitives::Address),
}

impl From<OpBlockExecutionError> for BlockExecutionError {
    fn from(err: OpBlockExecutionError) -> Self {
        Self::other(err)
    }
}
