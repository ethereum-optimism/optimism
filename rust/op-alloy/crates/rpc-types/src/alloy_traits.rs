//! Implementations of alloy-network conversion traits for OP RPC types.
//!
//! These impls are gated behind the `alloy-compat` feature because they
//! require `alloy-network` and `alloy-signer` dependencies.

use alloy_consensus::{
    SignableTransaction, error::ValueError, transaction::Recovered,
};
use alloy_network::convert::{
    FromConsensusTx, SignTxRequestError, SignableTxRequest, TryIntoSimTx,
};
use alloy_primitives::{Address, Signature};
use op_alloy_consensus::{
    OpTransaction, OpTxEnvelope,
    transaction::OpTransactionInfo,
};
use std::convert::Infallible;

use crate::{OpTransactionRequest, Transaction};

// ===== FromConsensusTx =====

impl<T: OpTransaction + alloy_consensus::Transaction> FromConsensusTx<T> for Transaction<T> {
    type TxInfo = OpTransactionInfo;
    type Err = Infallible;

    fn from_consensus_tx(
        tx: T,
        signer: Address,
        tx_info: Self::TxInfo,
    ) -> Result<Self, Self::Err> {
        Ok(Self::from_transaction(Recovered::new_unchecked(tx, signer), tx_info))
    }
}

// ===== TryIntoSimTx =====

impl TryIntoSimTx<OpTxEnvelope> for OpTransactionRequest {
    fn try_into_sim_tx(self) -> Result<OpTxEnvelope, ValueError<Self>> {
        let tx = self
            .build_typed_tx()
            .map_err(|request| ValueError::new(request, "Required fields missing"))?;

        // Create an empty signature for the transaction.
        let signature = Signature::new(Default::default(), Default::default(), false);

        Ok(tx.into_signed(signature).into())
    }
}

// ===== SignableTxRequest =====

impl SignableTxRequest<OpTxEnvelope> for OpTransactionRequest {
    async fn try_build_and_sign(
        self,
        signer: impl alloy_network::TxSigner<Signature> + Send,
    ) -> Result<OpTxEnvelope, SignTxRequestError> {
        let mut tx =
            self.build_typed_tx().map_err(|_| SignTxRequestError::InvalidTransactionRequest)?;

        // sanity check: deposit transactions must not be signed by the user
        if tx.is_deposit() {
            return Err(SignTxRequestError::InvalidTransactionRequest);
        }

        let signature = signer.sign_transaction(&mut tx).await?;

        Ok(tx.into_signed(signature).into())
    }
}
