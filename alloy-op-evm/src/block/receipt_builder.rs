//! Abstraction over receipt building logic to allow plugging different primitive types into
//! [`super::OpBlockExecutor`].

use alloy_consensus::{Eip658Value, TransactionEnvelope};
use alloy_evm::{eth::receipt_builder::ReceiptBuilderCtx, Evm};
use core::fmt::Debug;
use op_alloy::consensus::{OpDepositReceipt, OpReceiptEnvelope, OpTxEnvelope, OpTxType};

/// Type that knows how to build a receipt based on execution result.
#[auto_impl::auto_impl(&, Arc)]
pub trait OpReceiptBuilder: Debug {
    /// Transaction type.
    type Transaction: TransactionEnvelope;
    /// Receipt type.
    type Receipt;

    /// Builds a receipt given a transaction and the result of the execution.
    ///
    /// Note: this method should return `Err` if the transaction is a deposit transaction. In that
    /// case, the `build_deposit_receipt` method will be called.
    fn build_receipt<'a, E: Evm>(
        &self,
        ctx: ReceiptBuilderCtx<'a, <Self::Transaction as TransactionEnvelope>::TxType, E>,
    ) -> Result<
        Self::Receipt,
        ReceiptBuilderCtx<'a, <Self::Transaction as TransactionEnvelope>::TxType, E>,
    >;

    /// Builds receipt for a deposit transaction.
    fn build_deposit_receipt(&self, inner: OpDepositReceipt) -> Self::Receipt;
}

/// Receipt builder operating on op-alloy types.
#[derive(Debug, Default, Clone, Copy)]
#[non_exhaustive]
pub struct OpAlloyReceiptBuilder;

impl OpReceiptBuilder for OpAlloyReceiptBuilder {
    type Transaction = OpTxEnvelope;
    type Receipt = OpReceiptEnvelope;

    fn build_receipt<'a, E: Evm>(
        &self,
        ctx: ReceiptBuilderCtx<'a, OpTxType, E>,
    ) -> Result<Self::Receipt, ReceiptBuilderCtx<'a, OpTxType, E>> {
        match ctx.tx_type {
            OpTxType::Deposit => Err(ctx),
            ty => {
                let receipt = alloy_consensus::Receipt {
                    status: Eip658Value::Eip658(ctx.result.is_success()),
                    cumulative_gas_used: ctx.cumulative_gas_used,
                    logs: ctx.result.into_logs(),
                }
                .with_bloom();

                Ok(match ty {
                    OpTxType::Legacy => OpReceiptEnvelope::Legacy(receipt),
                    OpTxType::Eip2930 => OpReceiptEnvelope::Eip2930(receipt),
                    OpTxType::Eip1559 => OpReceiptEnvelope::Eip1559(receipt),
                    OpTxType::Eip7702 => OpReceiptEnvelope::Eip7702(receipt),
                    OpTxType::Deposit => unreachable!(),
                })
            }
        }
    }

    fn build_deposit_receipt(&self, inner: OpDepositReceipt) -> Self::Receipt {
        OpReceiptEnvelope::Deposit(inner.with_bloom())
    }
}
