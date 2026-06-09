//! Heavily influenced by [reth](https://github.com/paradigmxyz/reth/blob/1e965caf5fa176f244a31c0d2662ba1b590938db/crates/optimism/payload/src/builder.rs#L570)
use alloy_primitives::{Address, U256};
use core::fmt::Debug;
use derive_more::Display;
use op_revm::OpTransactionError;
use reth_optimism_primitives::{OpReceipt, OpTransactionSigned};

#[derive(Debug, Display)]
pub enum TxnExecutionResult {
    TransactionDALimitExceeded,
    #[display("BlockDALimitExceeded: total_da_used={_0} tx_da_size={_1} block_da_limit={_2}")]
    BlockDALimitExceeded(u64, u64, u64),
    #[display("TransactionGasLimitExceeded: total_gas_used={_0} tx_gas_limit={_1}")]
    TransactionGasLimitExceeded(u64, u64, u64),
    #[display(
        "PreRefundGasLimitExceeded: cumulative_evm_gas_used={_0} tx_gas_limit={_1} block_gas_limit={_2}"
    )]
    PreRefundGasLimitExceeded(u64, u64, u64),
    SequencerTransaction,
    NonceTooLow,
    InteropFailed,
    #[display("InternalError({_0})")]
    InternalError(OpTransactionError),
    EvmError,
    Success,
    Reverted,
    RevertedAndExcluded,
    MaxGasUsageExceeded,
}

#[derive(Default, Debug)]
pub struct ExecutionInfo<Extra: Debug + Default = ()> {
    /// All executed transactions (unrecovered).
    pub executed_transactions: Vec<OpTransactionSigned>,
    /// The recovered senders for the executed transactions.
    pub executed_senders: Vec<Address>,
    /// The transaction receipts
    pub receipts: Vec<OpReceipt>,
    /// All gas used so far
    pub cumulative_gas_used: u64,
    /// All pre-refund EVM gas (real compute) so far, before any SDM/post-exec refund.
    ///
    /// Persists across flashblocks (each gets a fresh executor whose accumulator resets), so it is
    /// the binding quantity for the cross-flashblock pre-refund block-gas cap.
    pub cumulative_evm_gas_used: u64,
    /// Estimated DA size
    pub cumulative_da_bytes_used: u64,
    /// Tracks fees from executed mempool transactions
    pub total_fees: U256,
    /// Extra execution information that can be attached by individual builders.
    pub extra: Extra,
    /// DA Footprint Scalar for Jovian
    pub da_footprint_scalar: Option<u16>,
}

impl<T: Debug + Default> ExecutionInfo<T> {
    /// Create a new instance with allocated slots.
    pub fn with_capacity(capacity: usize) -> Self {
        Self {
            executed_transactions: Vec::with_capacity(capacity),
            executed_senders: Vec::with_capacity(capacity),
            receipts: Vec::with_capacity(capacity),
            cumulative_gas_used: 0,
            cumulative_evm_gas_used: 0,
            cumulative_da_bytes_used: 0,
            total_fees: U256::ZERO,
            extra: Default::default(),
            da_footprint_scalar: None,
        }
    }

    /// Returns true if the transaction would exceed the block limits:
    /// - block gas limit: ensures the transaction still fits into the block.
    /// - tx DA limit: if configured, ensures the tx does not exceed the maximum allowed DA limit
    ///   per tx.
    /// - block DA limit: if configured, ensures the transaction's DA size does not exceed the
    ///   maximum allowed DA limit per block.
    #[allow(clippy::too_many_arguments)]
    pub fn is_tx_over_limits(
        &self,
        tx_da_size: u64,
        block_gas_limit: u64,
        tx_data_limit: Option<u64>,
        block_data_limit: Option<u64>,
        tx_gas_limit: u64,
        da_footprint_gas_scalar: Option<u16>,
        block_da_footprint_limit: Option<u64>,
    ) -> Result<(), TxnExecutionResult> {
        if tx_data_limit.is_some_and(|da_limit| tx_da_size > da_limit) {
            return Err(TxnExecutionResult::TransactionDALimitExceeded);
        }
        let total_da_bytes_used = self.cumulative_da_bytes_used.saturating_add(tx_da_size);
        if block_data_limit.is_some_and(|da_limit| total_da_bytes_used > da_limit) {
            return Err(TxnExecutionResult::BlockDALimitExceeded(
                self.cumulative_da_bytes_used,
                tx_da_size,
                block_data_limit.unwrap_or_default(),
            ));
        }

        // Post Jovian: the tx DA footprint must be less than the block gas limit
        if let Some(da_footprint_gas_scalar) = da_footprint_gas_scalar {
            let tx_da_footprint =
                total_da_bytes_used.saturating_mul(da_footprint_gas_scalar as u64);
            if tx_da_footprint > block_da_footprint_limit.unwrap_or(block_gas_limit) {
                return Err(TxnExecutionResult::BlockDALimitExceeded(
                    total_da_bytes_used,
                    tx_da_size,
                    tx_da_footprint,
                ));
            }
        }

        // Canonical (post-refund) block-gas-limit check. The pre-refund check below subsumes the
        // *bound* (since `cumulative_evm_gas_used >= cumulative_gas_used`), but this is kept as a
        // distinct diagnostic: it fires only when even canonical gas overflows (the only case with
        // SDM off).
        if self.cumulative_gas_used.saturating_add(tx_gas_limit) > block_gas_limit {
            return Err(TxnExecutionResult::TransactionGasLimitExceeded(
                self.cumulative_gas_used,
                tx_gas_limit,
                block_gas_limit,
            ));
        }

        // Cap real compute (pre-refund EVM gas) at the block gas limit, regardless of SDM refunds.
        // Binds across flashblocks because `cumulative_evm_gas_used` persists while the per-
        // flashblock executor accumulator resets. With SDM on, this fires (where the canonical check
        // above did not) when refunds let canonical gas fit but real compute is the binding limit.
        if self.cumulative_evm_gas_used.saturating_add(tx_gas_limit) > block_gas_limit {
            return Err(TxnExecutionResult::PreRefundGasLimitExceeded(
                self.cumulative_evm_gas_used,
                tx_gas_limit,
                block_gas_limit,
            ));
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests;
