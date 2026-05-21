#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/alloy-rs/core/main/assets/alloy.jpg",
    html_favicon_url = "https://raw.githubusercontent.com/alloy-rs/core/main/assets/favicon.ico"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg))]

pub use alloy_network::*;

use alloy_consensus::{ReceiptWithBloom, TxType};
use op_alloy_consensus::{OpReceipt, OpTxType, OpTypedTransaction};
use op_alloy_rpc_types::OpTransactionRequest;

/// Types for an Op-stack network.
#[derive(Clone, Copy, Debug)]
pub struct Optimism {
    _private: (),
}

impl Network for Optimism {
    type TxType = OpTxType;

    type TxEnvelope = op_alloy_consensus::OpTxEnvelope;

    type UnsignedTx = op_alloy_consensus::OpTypedTransaction;

    type ReceiptEnvelope = ReceiptWithBloom<OpReceipt>;

    type Header = alloy_consensus::Header;

    type TransactionRequest = op_alloy_rpc_types::OpTransactionRequest;

    type TransactionResponse = op_alloy_rpc_types::Transaction;

    type ReceiptResponse = op_alloy_rpc_types::OpTransactionReceipt;

    type HeaderResponse = alloy_rpc_types_eth::Header;

    type BlockResponse =
        alloy_rpc_types_eth::Block<Self::TransactionResponse, Self::HeaderResponse>;
}

impl NetworkTransactionBuilder<Optimism> for OpTransactionRequest {
    fn complete_type(&self, ty: OpTxType) -> Result<(), Vec<&'static str>> {
        match ty {
            OpTxType::Deposit => Err(vec!["not implemented for deposit tx"]),
            OpTxType::PostExec => Err(vec!["not implemented for post-exec tx"]),
            _ => {
                let ty = TxType::try_from(ty as u8).unwrap();
                NetworkTransactionBuilder::<Ethereum>::complete_type(self.as_ref(), ty)
            }
        }
    }

    fn can_submit(&self) -> bool {
        NetworkTransactionBuilder::<Ethereum>::can_submit(self.as_ref())
    }

    fn can_build(&self) -> bool {
        NetworkTransactionBuilder::<Ethereum>::can_build(self.as_ref())
    }

    #[doc(alias = "output_transaction_type")]
    fn output_tx_type(&self) -> OpTxType {
        match NetworkTransactionBuilder::<Ethereum>::output_tx_type(self.as_ref()) {
            TxType::Eip1559 | TxType::Eip4844 => OpTxType::Eip1559,
            TxType::Eip2930 => OpTxType::Eip2930,
            TxType::Eip7702 => OpTxType::Eip7702,
            TxType::Legacy => OpTxType::Legacy,
        }
    }

    #[doc(alias = "output_transaction_type_checked")]
    fn output_tx_type_checked(&self) -> Option<OpTxType> {
        NetworkTransactionBuilder::<Ethereum>::output_tx_type_checked(self.as_ref()).map(|tx_ty| {
            match tx_ty {
                TxType::Eip1559 | TxType::Eip4844 => OpTxType::Eip1559,
                TxType::Eip2930 => OpTxType::Eip2930,
                TxType::Eip7702 => OpTxType::Eip7702,
                TxType::Legacy => OpTxType::Legacy,
            }
        })
    }

    fn prep_for_submission(&mut self) {
        NetworkTransactionBuilder::<Ethereum>::prep_for_submission(self.as_mut());
    }

    fn build_unsigned(self) -> BuildResult<OpTypedTransaction, Optimism> {
        if let Err((tx_type, missing)) = self.as_ref().missing_keys() {
            let tx_type = OpTxType::try_from(tx_type as u8).unwrap();
            return Err(TransactionBuilderError::InvalidTransactionRequest(tx_type, missing)
                .into_unbuilt(self));
        }
        Ok(self.build_typed_tx().expect("checked by missing_keys"))
    }

    async fn build<W: NetworkWallet<Optimism>>(
        self,
        wallet: &W,
    ) -> Result<<Optimism as Network>::TxEnvelope, TransactionBuilderError<Optimism>> {
        Ok(wallet.sign_request(self).await?)
    }
}

use alloy_provider::fillers::{
    ChainIdFiller, GasFiller, JoinFill, NonceFiller, RecommendedFillers,
};

impl RecommendedFillers for Optimism {
    type RecommendedFillers = JoinFill<GasFiller, JoinFill<NonceFiller, ChainIdFiller>>;

    fn recommended_fillers() -> Self::RecommendedFillers {
        Default::default()
    }
}
