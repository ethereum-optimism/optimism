//! RPC transaction conversion for OP types.

use crate::OpTx;
use alloy_evm::{
    EvmEnv,
    env::BlockEnvironment,
    rpc::{EthTxEnvError, TryIntoTxEnv},
};
use alloy_primitives::Bytes;
use op_alloy::rpc_types::OpTransactionRequest;
use op_revm::OpTransaction;

impl<Block: BlockEnvironment> TryIntoTxEnv<OpTx, Block> for OpTransactionRequest {
    type Err = EthTxEnvError;

    fn try_into_tx_env<Spec>(
        self,
        evm_env: &EvmEnv<Spec, Block>,
    ) -> Result<OpTx, Self::Err> {
        Ok(OpTx(OpTransaction {
            base: self.as_ref().clone().try_into_tx_env(evm_env)?,
            enveloped_tx: Some(Bytes::new()),
            deposit: Default::default(),
        }))
    }
}
