//! Optimism consensus errors

use alloy_primitives::B256;
use reth_consensus::ConsensusError;
use reth_storage_errors::provider::ProviderError;

/// Optimism consensus error.
#[derive(Debug, Clone, thiserror::Error)]
pub enum OpConsensusError {
    /// Block body has non-empty withdrawals list (l1 withdrawals).
    #[error("non-empty block body withdrawals list")]
    WithdrawalsNonEmpty,
    /// Failed to compute L2 withdrawals storage root.
    #[error("compute L2 withdrawals root failed: {_0}")]
    L2WithdrawalsRootCalculationFail(#[from] ProviderError),
    /// L2 withdrawals root missing in block header.
    #[error("L2 withdrawals root missing from block header")]
    L2WithdrawalsRootMissing,
    /// L2 withdrawals root in block header, doesn't match local storage root of predeploy.
    #[error("L2 withdrawals root mismatch, header: {header}, exec_res: {exec_res}")]
    L2WithdrawalsRootMismatch {
        /// Storage root of pre-deploy in block.
        header: B256,
        /// Storage root of pre-deploy loaded from local state.
        exec_res: B256,
    },
    /// L1 [`ConsensusError`], that also occurs on L2.
    #[error(transparent)]
    Eth(#[from] ConsensusError),
}
