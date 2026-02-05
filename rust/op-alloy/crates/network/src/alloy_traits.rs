//! Implementations of alloy-network conversion traits for OP network types.
//!
//! Only traits that use [`Optimism`] (a local type) can be implemented here
//! due to the orphan rule. The remaining conversion trait impls
//! ([`FromConsensusTx`], [`TryIntoSimTx`], [`SignableTxRequest`])
//! live in `op-alloy-rpc-types` behind the `alloy-compat` feature.

use alloy_network::convert::{TryFromReceiptResponse, TryFromTransactionResponse};
use op_alloy_consensus::OpTxEnvelope;
use std::convert::Infallible;

use crate::Optimism;

// ===== TryFromTransactionResponse =====

impl TryFromTransactionResponse<Optimism> for OpTxEnvelope {
    type Error = Infallible;

    fn from_transaction_response(
        transaction_response: op_alloy_rpc_types::Transaction,
    ) -> Result<Self, Self::Error> {
        Ok(transaction_response.inner.into_inner())
    }
}

// ===== TryFromReceiptResponse =====

impl TryFromReceiptResponse<Optimism> for op_alloy_consensus::OpReceipt {
    type Error = Infallible;

    fn from_receipt_response(
        receipt_response: op_alloy_rpc_types::OpTransactionReceipt,
    ) -> Result<Self, Self::Error> {
        Ok(receipt_response.inner.inner.into_components().0.map_logs(Into::into))
    }
}
