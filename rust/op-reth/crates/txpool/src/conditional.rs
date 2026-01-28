//! Additional support for pooled transactions with [`TransactionConditional`]

use alloy_consensus::conditional::BlockConditionalAttributes;
use alloy_rpc_types_eth::erc4337::TransactionConditional;

/// Helper trait that allows attaching a [`TransactionConditional`].
pub trait MaybeConditionalTransaction {
    /// Attach a [`TransactionConditional`].
    fn set_conditional(&mut self, conditional: TransactionConditional);

    /// Get attached [`TransactionConditional`] if any.
    fn conditional(&self) -> Option<&TransactionConditional>;

    /// Check if the conditional has exceeded the block attributes.
    fn has_exceeded_block_attributes(&self, block_attr: &BlockConditionalAttributes) -> bool {
        self.conditional().map(|tc| tc.has_exceeded_block_attributes(block_attr)).unwrap_or(false)
    }

    /// Helper that sets the conditional and returns the instance again
    fn with_conditional(mut self, conditional: TransactionConditional) -> Self
    where
        Self: Sized,
    {
        self.set_conditional(conditional);
        self
    }
}
