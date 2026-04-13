//! OP transaction environment types.

pub use alloy_op_evm::OpTx;

/// Converter that builds an [`OpTx`] from an
/// [`OpTransactionRequest`](op_alloy_rpc_types::OpTransactionRequest).
///
/// Implements `TxEnvConverter` for use in `RpcConverter`, bypassing the orphan rule
/// issue with `TryIntoTxEnv` (neither `OpTransactionRequest` nor the trait is local).
#[derive(Debug, Clone, Copy, Default)]
pub struct OpTxEnvConverter;

#[cfg(feature = "rpc")]
mod rpc_impl {
    use super::*;
    use alloy_evm::rpc::{EthTxEnvError, TryIntoTxEnv};
    use op_alloy_rpc_types::OpTransactionRequest;
    use reth_evm::ConfigureEvm;
    use reth_rpc_eth_api::transaction::TxEnvConverter;

    impl<Evm> TxEnvConverter<OpTransactionRequest, Evm> for OpTxEnvConverter
    where
        Evm: ConfigureEvm,
        reth_evm::TxEnvFor<Evm>: From<OpTx>,
    {
        type Error = EthTxEnvError;

        fn convert_tx_env(
            &self,
            req: OpTransactionRequest,
            evm_env: &reth_evm::EvmEnvFor<Evm>,
        ) -> Result<reth_evm::TxEnvFor<Evm>, Self::Error> {
            let base: revm::context::TxEnv = req.as_ref().clone().try_into_tx_env(evm_env)?;
            let op_tx = OpTx(op_revm::OpTransaction {
                base,
                enveloped_tx: Some(alloy_primitives::Bytes::new()),
                deposit: Default::default(),
            });
            Ok(op_tx.into())
        }
    }
}
