use crate::{
    evm::CustomTxEnv,
    primitives::{CustomHeader, CustomTransaction},
};
use alloy_consensus::error::ValueError;
use alloy_network::TxSigner;
use op_alloy_consensus::OpTxEnvelope;
use op_alloy_rpc_types::{OpTransactionReceipt, OpTransactionRequest};
use reth_op::rpc::RpcTypes;
use reth_rpc_api::eth::{SignTxRequestError, SignableTxRequest, TryIntoSimTx};

#[derive(Debug, Clone, Copy, Default)]
#[non_exhaustive]
pub struct CustomRpcTypes;

impl RpcTypes for CustomRpcTypes {
    type Header = alloy_rpc_types_eth::Header<CustomHeader>;
    type Receipt = OpTransactionReceipt;
    type TransactionRequest = OpTransactionRequest;
    type TransactionResponse = op_alloy_rpc_types::Transaction<CustomTransaction>;
}

impl TryIntoSimTx<CustomTransaction> for OpTransactionRequest {
    fn try_into_sim_tx(self) -> Result<CustomTransaction, ValueError<Self>> {
        Ok(CustomTransaction::Op(self.try_into_sim_tx()?))
    }
}

/// Custom `TxEnvConverter` that converts [`OpTransactionRequest`] into [`CustomTxEnv`].
#[derive(Debug, Clone, Copy, Default)]
pub struct CustomTxEnvConverter;

impl<Evm> reth_rpc_api::eth::transaction::TxEnvConverter<OpTransactionRequest, Evm>
    for CustomTxEnvConverter
where
    Evm: reth_evm::ConfigureEvm,
    reth_evm::TxEnvFor<Evm>: From<CustomTxEnv>,
{
    type Error = alloy_evm::rpc::EthTxEnvError;

    fn convert_tx_env(
        &self,
        req: OpTransactionRequest,
        evm_env: &reth_evm::EvmEnvFor<Evm>,
    ) -> Result<reth_evm::TxEnvFor<Evm>, Self::Error> {
        use alloy_evm::rpc::TryIntoTxEnv;
        let base: revm::context::TxEnv = req.as_ref().clone().try_into_tx_env(evm_env)?;
        let op_tx = reth_optimism_evm::OpTx(op_revm::OpTransaction {
            base,
            enveloped_tx: Some(alloy_primitives::Bytes::new()),
            deposit: Default::default(),
        });
        Ok(CustomTxEnv::Op(op_tx).into())
    }
}

impl SignableTxRequest<CustomTransaction> for OpTransactionRequest {
    async fn try_build_and_sign(
        self,
        signer: impl TxSigner<alloy_primitives::Signature> + Send,
    ) -> Result<CustomTransaction, SignTxRequestError> {
        Ok(CustomTransaction::Op(
            SignableTxRequest::<OpTxEnvelope>::try_build_and_sign(self, signer).await?,
        ))
    }
}
