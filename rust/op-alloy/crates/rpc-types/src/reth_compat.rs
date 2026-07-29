//! Implementations of `reth-rpc-traits` for OP types.
//!
//! Ported from reth v1.11.3 (`d6324d63e`), where they lived behind `cfg(feature = "op")`:
//! - `FromConsensusTx`, `TryIntoSimTx`: `crates/rpc/rpc-convert/src/transaction.rs`
//! - `SignableTxRequest`: `crates/rpc/rpc-convert/src/rpc.rs`
//!
//! The traits themselves moved from `reth-rpc-convert` to the published `reth-rpc-traits`
//! crate (v0.1.0), so the impls now target `reth-rpc-traits` types. The logic is identical
//! to upstream.

use alloy_consensus::SignableTransaction;
use alloy_primitives::Address;
use alloy_signer::Signature;
use core::convert::Infallible;
use op_alloy_consensus::{
    OpTxEnvelope, TxDeposit,
    transaction::{OpTransaction, OpTransactionInfo},
};
use reth_rpc_traits::{FromConsensusTx, SignTxRequestError, SignableTxRequest, TryIntoSimTx};

use crate::OpTransactionRequest;

impl<T: OpTransaction + alloy_consensus::Transaction> FromConsensusTx<T> for crate::Transaction<T> {
    type TxInfo = OpTransactionInfo;
    type Err = Infallible;

    fn from_consensus_tx(tx: T, signer: Address, tx_info: Self::TxInfo) -> Result<Self, Self::Err> {
        Ok(Self::from_transaction(
            alloy_consensus::transaction::Recovered::new_unchecked(tx, signer),
            tx_info,
        ))
    }
}

impl TryIntoSimTx<OpTxEnvelope> for OpTransactionRequest {
    fn try_into_sim_tx(self) -> Result<OpTxEnvelope, alloy_consensus::error::ValueError<Self>> {
        let tx = self.build_typed_tx().map_err(|request| {
            alloy_consensus::error::ValueError::new(request, "Required fields missing")
        })?;

        // Create an empty signature for the transaction.
        let signature = Signature::new(Default::default(), Default::default(), false);

        Ok(tx.into_signed(signature).into())
    }
}

impl SignableTxRequest<OpTxEnvelope> for OpTransactionRequest {
    async fn try_build_and_sign(
        self,
        signer: impl alloy_network::TxSigner<Signature> + Send,
    ) -> Result<OpTxEnvelope, SignTxRequestError> {
        let mut tx =
            self.build_typed_tx().map_err(|_| SignTxRequestError::InvalidTransactionRequest)?;

        // Deposit transactions must not be signed by the user.
        if matches!(tx, op_alloy_consensus::OpTypedTransaction::Deposit(TxDeposit { .. })) {
            return Err(SignTxRequestError::InvalidTransactionRequest);
        }

        let signature = signer.sign_transaction(&mut tx).await?;

        Ok(tx.into_signed(signature).into())
    }
}
