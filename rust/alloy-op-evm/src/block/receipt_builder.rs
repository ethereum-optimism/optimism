//! Abstraction over receipt building logic to allow plugging different primitive types into
//! [`super::OpBlockExecutor`].

use alloy_consensus::{Eip658Value, TransactionEnvelope, TxReceipt};
use alloy_evm::{Evm, eth::receipt_builder::ReceiptBuilderCtx};
use alloy_primitives::Log;
use core::fmt::Debug;
use op_alloy::consensus::{OpDepositReceipt, OpReceiptEnvelope, OpTxEnvelope, OpTxType};

/// Type that knows how to build a receipt based on execution result.
#[auto_impl::auto_impl(&, Arc)]
pub trait OpReceiptBuilder: Debug {
    /// Transaction type.
    ///
    /// `TxType: Send + 'static` is required so that `OpTxResult<H, T>` can satisfy the
    /// upstream `TxResult` trait bound (`Self: Send + 'static`).
    type Transaction: TransactionEnvelope<TxType: Send + 'static>;
    /// Receipt type.
    ///
    /// `TxReceipt<Log = Log> + Clone` is the bundle every downstream impl already satisfies, so
    /// requiring it here lets `R::Receipt` bound lists drop those bounds. Sites that additionally
    /// encode receipts (e.g. the receipts-root computation) keep `Encodable2718` as a call-site
    /// bound — not every receipt builder's `Receipt` implements it.
    type Receipt: TxReceipt<Log = Log> + Clone;

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

    /// Strips the deposit nonce from `receipt` if the receipt is a deposit variant that carries
    /// one.
    ///
    /// Used by the executor's receipts-root computation to reproduce the op-geth/op-erigon
    /// Regolith-era encoding bug, in which the deposit nonce is omitted from the receipts-trie
    /// encoding from Regolith activation up to (but not including) Canyon activation. OP Stack
    /// implementations clear the nonce; chains whose deposit receipt has no nonce field
    /// implement this as a no-op.
    fn strip_deposit_nonce(&self, receipt: &mut Self::Receipt);
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
                    OpTxType::PostExec => OpReceiptEnvelope::PostExec(receipt),
                    OpTxType::Deposit => unreachable!(),
                })
            }
        }
    }

    fn build_deposit_receipt(&self, inner: OpDepositReceipt) -> Self::Receipt {
        OpReceiptEnvelope::Deposit(inner.with_bloom())
    }

    fn strip_deposit_nonce(&self, receipt: &mut Self::Receipt) {
        if let OpReceiptEnvelope::Deposit(d) = receipt {
            d.receipt.deposit_nonce = None;
        }
    }
}
