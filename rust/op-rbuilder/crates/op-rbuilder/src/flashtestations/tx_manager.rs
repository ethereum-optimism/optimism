use alloy_json_rpc::RpcError;
use alloy_network::ReceiptResponse;
use alloy_primitives::{Address, B256, Bytes, TxHash, TxKind, U256};
use alloy_rpc_types_eth::TransactionRequest;
use alloy_sol_types::SolCall;
use alloy_transport::{TransportError, TransportErrorKind, TransportResult};
use k256::ecdsa;
use std::time::Duration;

use alloy_provider::{
    PendingTransactionBuilder, PendingTransactionError, Provider, ProviderBuilder,
};
use alloy_signer_local::PrivateKeySigner;
use op_alloy_network::Optimism;
use tracing::{debug, info, warn};

use crate::{
    flashtestations::{
        IERC20Permit::{self},
        IFlashtestationRegistry,
    },
    tx_signer::Signer,
};

#[derive(Debug, thiserror::Error)]
pub enum TxManagerError {
    #[error("rpc error: {0}")]
    RpcError(#[from] TransportError),
    #[error("tx reverted: {0}")]
    TxReverted(TxHash),
    #[error("error checking tx confirmation: {0}")]
    TxConfirmationError(PendingTransactionError),
    #[error("tx rpc error: {0}")]
    TxRpcError(RpcError<TransportErrorKind>),
    #[error("signer error: {0}")]
    SignerError(ecdsa::Error),
    #[error("error signing message: {0}")]
    SignatureError(secp256k1::Error),
}

#[derive(Debug, Clone)]
pub struct TxManager {
    tee_service_signer: Signer,
    builder_signer: Signer,
    rpc_url: String,
    registry_address: Address,
}

impl TxManager {
    pub fn new(
        tee_service_signer: Signer,
        builder_signer: Signer,
        rpc_url: String,
        registry_address: Address,
    ) -> Self {
        Self {
            tee_service_signer,
            builder_signer,
            rpc_url,
            registry_address,
        }
    }

    pub async fn register_tee_service(
        &self,
        attestation: Vec<u8>,
        extra_registration_data: Bytes,
    ) -> Result<(), TxManagerError> {
        info!(target: "flashtestations", "funding TEE address at {}", self.tee_service_signer.address);
        let quote_bytes = Bytes::from(attestation);
        let wallet =
            PrivateKeySigner::from_bytes(&self.builder_signer.secret.secret_bytes().into())
                .map_err(TxManagerError::SignerError)?;
        let provider = ProviderBuilder::new()
            .disable_recommended_fillers()
            .fetch_chain_id()
            .with_gas_estimation()
            .with_cached_nonce_management()
            .wallet(wallet)
            .network::<Optimism>()
            .connect(self.rpc_url.as_str())
            .await?;

        info!(target: "flashtestations", "submitting quote to registry at {}", self.registry_address);

        // Get permit nonce
        let nonce_call = IERC20Permit::noncesCall {
            owner: self.tee_service_signer.address,
        };
        let nonce_tx = TransactionRequest {
            to: Some(TxKind::Call(self.registry_address)),
            input: nonce_call.abi_encode().into(),
            ..Default::default()
        };
        let nonce = U256::from_be_slice(provider.call(nonce_tx.into()).await?.as_ref());

        // Set deadline 1 hour from now
        let deadline = U256::from(
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs()
                + 3600,
        );

        // Call computeStructHash to get the struct hash
        let struct_hash_call = IFlashtestationRegistry::computeStructHashCall {
            rawQuote: quote_bytes.clone(),
            extendedRegistrationData: extra_registration_data.clone(),
            nonce,
            deadline,
        };
        let struct_hash_tx = TransactionRequest {
            to: Some(TxKind::Call(self.registry_address)),
            input: struct_hash_call.abi_encode().into(),
            ..Default::default()
        };
        let struct_hash = B256::from_slice(provider.call(struct_hash_tx.into()).await?.as_ref());

        // Get typed data hash
        let typed_hash_call = IFlashtestationRegistry::hashTypedDataV4Call {
            structHash: struct_hash,
        };
        let typed_hash_tx = TransactionRequest {
            to: Some(TxKind::Call(self.registry_address)),
            input: typed_hash_call.abi_encode().into(),
            ..Default::default()
        };
        let message_hash = B256::from_slice(provider.call(typed_hash_tx.into()).await?.as_ref());

        // Sign the hash
        let signature = self
            .tee_service_signer
            .sign_message(message_hash)
            .map_err(TxManagerError::SignatureError)?;

        let calldata = IFlashtestationRegistry::permitRegisterTEEServiceCall {
            rawQuote: quote_bytes,
            extendedRegistrationData: extra_registration_data,
            nonce,
            deadline,
            signature: signature.as_bytes().into(),
        }
        .abi_encode();
        let tx = TransactionRequest {
            from: Some(self.tee_service_signer.address),
            to: Some(TxKind::Call(self.registry_address)),
            input: calldata.into(),
            ..Default::default()
        };
        match Self::process_pending_tx(provider.send_transaction(tx.into()).await).await {
            Ok(tx_hash) => {
                info!(target: "flashtestations", tx_hash = %tx_hash, "attestation transaction confirmed successfully");
                Ok(())
            }
            Err(e) => {
                warn!(target: "flashtestations", error = %e, "attestation transaction failed to be sent");
                Err(e)
            }
        }
    }

    /// Processes a pending transaction and logs whether the transaction succeeded or not
    async fn process_pending_tx(
        pending_tx_result: TransportResult<PendingTransactionBuilder<Optimism>>,
    ) -> Result<TxHash, TxManagerError> {
        match pending_tx_result {
            Ok(pending_tx) => {
                let tx_hash = *pending_tx.tx_hash();
                debug!(target: "flashtestations", tx_hash = %tx_hash, "transaction submitted");

                // Wait for funding transaction confirmation
                match pending_tx
                    .with_timeout(Some(Duration::from_secs(30)))
                    .get_receipt()
                    .await
                {
                    Ok(receipt) => {
                        if receipt.status() {
                            Ok(receipt.transaction_hash())
                        } else {
                            Err(TxManagerError::TxReverted(tx_hash))
                        }
                    }
                    Err(e) => Err(TxManagerError::TxConfirmationError(e)),
                }
            }
            Err(e) => Err(TxManagerError::TxRpcError(e)),
        }
    }
}
