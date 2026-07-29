use std::sync::Arc;

use alloy_primitives::{Address, B256, ChainId, SignatureError};
use alloy_rpc_client::RpcClient;
use alloy_signer::Signature;
use notify::RecommendedWatcher;
use op_alloy_rpc_types_engine::PayloadHash;
use serde::{Deserialize, Serialize};
use thiserror::Error;
use tokio::sync::RwLock;

/// Request parameters for signing a block payload
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct BlockPayloadArgs {
    domain: B256,
    chain_id: u64,
    payload_hash: B256,
    sender_address: Address,
}

/// Response from the remote signer
#[derive(Debug, Deserialize)]
struct SignResponse {
    signature: String,
}

/// Remote signer that communicates with an external signing service via JSON-RPC
#[derive(Debug)]
pub struct RemoteSignerHandler {
    /// The JSON-RPC client.
    pub(super) client: Arc<RwLock<RpcClient>>,
    /// The address of the signer.
    pub(super) address: Address,
    /// The watcher handle for certificate watching.
    pub(super) watcher_handle: Option<RecommendedWatcher>,
}

/// Errors that can occur when using the remote signer
#[derive(Debug, Error)]
pub enum RemoteSignerError {
    /// JSON-RPC transport error
    #[error("JSON-RPC transport error: {0}")]
    SigningRPCError(#[from] alloy_transport::TransportError),
    /// JSON serialization error
    #[error("JSON serialization error: {0}")]
    JsonError(#[from] serde_json::Error),
    /// Failed to ping signer
    #[error("Failed to ping signer: {0}")]
    PingError(alloy_transport::TransportError),
    /// Invalid signature hex encoding
    #[error("Invalid signature hex encoding: {0}")]
    InvalidSignatureHex(alloy_primitives::hex::FromHexError),
    /// Invalid signature length
    #[error("Invalid signature length, expected 65 bytes, got {0}")]
    InvalidSignatureLength(usize),
    /// Signature error
    #[error("Signature error: {0}")]
    SignatureError(#[from] SignatureError),
    /// Invalid address
    #[error(
        "Unsafe block signer address does not match remote signer address: {unsafe_block_signer} != {remote_signer}"
    )]
    InvalidAddress {
        /// The unsafe block signer address.
        unsafe_block_signer: Address,
        /// The remote signer address.
        remote_signer: Address,
    },
}

impl RemoteSignerHandler {
    /// Returns true if certificate watching is enabled
    pub const fn is_certificate_watching_enabled(&self) -> bool {
        self.watcher_handle.is_some()
    }

    /// Signs a block payload hash using the remote signer via JSON-RPC
    pub async fn sign_block_v1(
        &self,
        payload_hash: PayloadHash,
        chain_id: ChainId,
        sender_address: Address,
    ) -> Result<Signature, RemoteSignerError> {
        if sender_address != self.address {
            return Err(RemoteSignerError::InvalidAddress {
                unsafe_block_signer: sender_address,
                remote_signer: self.address,
            });
        }

        let params = BlockPayloadArgs {
            // For v1 payloads, the domain is always zero
            domain: B256::ZERO,
            chain_id,
            payload_hash: payload_hash.0,
            sender_address,
        };

        // Make JSON-RPC call to the custom method
        let response: SignResponse = {
            self.client
                .read()
                .await
                .request("opsigner_signBlockPayload", &params)
                .await
                .map_err(RemoteSignerError::SigningRPCError)?
        };

        // Parse the hex signature
        let signature_bytes =
            alloy_primitives::hex::decode(response.signature.trim_start_matches("0x"))
                .map_err(RemoteSignerError::InvalidSignatureHex)?;

        if signature_bytes.len() != 65 {
            return Err(RemoteSignerError::InvalidSignatureLength(signature_bytes.len()));
        }

        let signature = Signature::from_raw(signature_bytes.as_slice())
            .map_err(RemoteSignerError::SignatureError)?;

        Ok(signature)
    }
}
