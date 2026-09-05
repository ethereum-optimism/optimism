use alloy_consensus::{Eip658Value, Receipt};
use alloy_evm::eth::receipt_builder::ReceiptBuilderCtx;
use alloy_op_evm::block::receipt_builder::OpReceiptBuilder;
use op_alloy_consensus::{OpDepositReceipt, OpTxType};
use reth_evm::Evm;
use reth_optimism_primitives::{OpReceipt, OpTransactionSigned};

/// A builder that operates on op-reth primitive types, specifically [`OpTransactionSigned`] and
/// [`OpReceipt`].
#[derive(Debug, Default, Clone, Copy)]
#[non_exhaustive]
pub struct OpRethReceiptBuilder {
    suppress_deposit_logs: bool,
}

impl OpRethReceiptBuilder {
    /// Projection deposits retain execution and accounting, but cannot publish events. Only
    /// sequencer-published replay transactions contribute to the public message history.
    pub const fn for_public_projection() -> Self {
        Self { suppress_deposit_logs: true }
    }
}

impl OpReceiptBuilder for OpRethReceiptBuilder {
    type Transaction = OpTransactionSigned;
    type Receipt = OpReceipt;

    fn build_receipt<'a, E: Evm>(
        &self,
        ctx: ReceiptBuilderCtx<'a, OpTxType, E>,
    ) -> Result<Self::Receipt, ReceiptBuilderCtx<'a, OpTxType, E>> {
        match ctx.tx_type {
            OpTxType::Deposit => Err(ctx),
            ty => {
                let receipt = Receipt {
                    // Success flag was added in `EIP-658: Embedding transaction status code in
                    // receipts`.
                    status: Eip658Value::Eip658(ctx.result.is_success()),
                    cumulative_gas_used: ctx.cumulative_gas_used,
                    logs: ctx.result.into_logs(),
                };

                Ok(match ty {
                    OpTxType::Legacy => OpReceipt::Legacy(receipt),
                    OpTxType::Eip1559 => OpReceipt::Eip1559(receipt),
                    OpTxType::Eip2930 => OpReceipt::Eip2930(receipt),
                    OpTxType::Eip7702 => OpReceipt::Eip7702(receipt),
                    OpTxType::PostExec => OpReceipt::PostExec(receipt),
                    OpTxType::Deposit => unreachable!(),
                })
            }
        }
    }

    fn build_deposit_receipt(&self, mut inner: OpDepositReceipt) -> Self::Receipt {
        if self.suppress_deposit_logs {
            inner.inner.logs.clear();
        }
        OpReceipt::Deposit(inner)
    }

    fn strip_deposit_nonce(&self, receipt: &mut Self::Receipt) {
        if let OpReceipt::Deposit(d) = receipt {
            d.deposit_nonce = None;
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::vec;
    use alloy_primitives::{Address, Log};

    #[test]
    fn projection_deposits_keep_accounting_without_publishing_logs() {
        let deposit = OpDepositReceipt {
            inner: Receipt {
                status: Eip658Value::Eip658(true),
                cumulative_gas_used: 42_000,
                logs: vec![Log::new_unchecked(Address::ZERO, vec![], vec![1, 2].into())],
            },
            deposit_nonce: Some(7),
            deposit_receipt_version: Some(1),
        };
        assert_eq!(
            OpRethReceiptBuilder::default().build_deposit_receipt(deposit.clone()),
            OpReceipt::Deposit(deposit.clone())
        );
        let OpReceipt::Deposit(projected) =
            OpRethReceiptBuilder::for_public_projection().build_deposit_receipt(deposit.clone())
        else {
            panic!("deposit receipt must retain its type");
        };
        assert!(projected.inner.logs.is_empty());
        assert_eq!(projected.inner.status, deposit.inner.status);
        assert_eq!(projected.inner.cumulative_gas_used, deposit.inner.cumulative_gas_used);
        assert_eq!(projected.deposit_nonce, deposit.deposit_nonce);
        assert_eq!(projected.deposit_receipt_version, deposit.deposit_receipt_version);
    }
}
