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
pub struct OpRethReceiptBuilder;

impl OpReceiptBuilder for OpRethReceiptBuilder {
    type Transaction = OpTransactionSigned;
    type Receipt = OpReceipt;

    fn build_receipt<'a, E: Evm>(
        &self,
        ctx: ReceiptBuilderCtx<'a, OpTransactionSigned, E>,
    ) -> Result<Self::Receipt, ReceiptBuilderCtx<'a, OpTransactionSigned, E>> {
        match ctx.tx.tx_type() {
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
                    OpTxType::Deposit => unreachable!(),
                })
            }
        }
    }

    fn build_deposit_receipt(&self, inner: OpDepositReceipt) -> Self::Receipt {
        OpReceipt::Deposit(inner)
    }
}
