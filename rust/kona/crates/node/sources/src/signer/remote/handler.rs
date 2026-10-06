use std::time::Duration;

use alloy_primitives::{Address, B256, ChainId, SignatureError};
use alloy_signer::Signature;
use alloy_transport::{RpcError, TransportError};
use backon::{ExponentialBuilder, Retryable};
use op_alloy_rpc_types_engine::PayloadHash;
use serde::{Deserialize, Serialize};
use thiserror::Error;

use crate::ReloadingRpcClient;

/// Delay before the first retry of a transient signing failure. It doubles up to
/// [`MAX_RETRY_DELAY`].
const MIN_RETRY_DELAY: Duration = Duration::from_millis(100);
/// Longest delay between signing attempts.
const MAX_RETRY_DELAY: Duration = Duration::from_secs(5);

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
    pub(super) client: ReloadingRpcClient,
    /// The address of the signer.
    pub(super) address: Address,
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
        self.client.is_watching()
    }

    /// Signs a block payload hash using the remote signer via JSON-RPC.
    ///
    /// A transient failure is retried until the signer returns a signature: the signer is
    /// unreachable, does not answer within the request timeout, or answers with an error. Any
    /// error this returns is one that retrying cannot fix.
    ///
    /// `sender_address` is checked once, before the first attempt. If the unsafe block signer
    /// rotates while a request is being retried, the signature returned is for the retired
    /// address, and the next call fails with [`RemoteSignerError::InvalidAddress`].
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

        let response = self.request_signature(&params).await?;

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

    /// Requests a signature, retrying transient failures until the signer returns one.
    async fn request_signature(
        &self,
        params: &BlockPayloadArgs,
    ) -> Result<SignResponse, RemoteSignerError> {
        let backoff = ExponentialBuilder::default()
            .with_min_delay(MIN_RETRY_DELAY)
            .with_max_delay(MAX_RETRY_DELAY)
            .without_max_times();
        // Take the current client on each attempt, so a client rebuilt for new certificates is
        // used next.
        (|| async { self.client.client().request("opsigner_signBlockPayload", params).await })
            .retry(backoff)
            .when(is_transient)
            .notify(|err, delay| {
                tracing::warn!(target: "signer", ?err, ?delay, "Signing request failed; retrying");
            })
            .await
            .map_err(RemoteSignerError::SigningRPCError)
    }
}

/// Whether a failed signing request can succeed if it is sent again: the signer was unreachable,
/// did not answer in time, or answered with an error. A request or response that cannot be
/// encoded or decoded fails the same way every time, and the remaining variants are not produced
/// by an HTTP transport.
///
/// alloy's own retry classification is narrower: it does not retry a refused connection or a
/// timeout, which are the main outages of a remote signer.
const fn is_transient(err: &TransportError) -> bool {
    matches!(err, RpcError::Transport(_) | RpcError::ErrorResp(_))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::RemoteSigner;
    use alloy_signer::SignerSync;
    use alloy_signer_local::PrivateKeySigner;
    use jsonrpsee::{
        RpcModule,
        server::{ServerBuilder, ServerHandle},
        types::ErrorObjectOwned,
    };
    use std::sync::{
        Arc,
        atomic::{AtomicUsize, Ordering},
    };

    const CHAIN_ID: u64 = 10;

    /// How the fake signer answers a signing request.
    #[derive(Debug, Clone, Copy)]
    enum Answer {
        /// A JSON-RPC error response.
        Error,
        /// No response at all.
        Stall,
        /// A result that is not a signature.
        Garbage,
        /// A signature over the payload hash.
        Sign,
    }

    /// Starts a fake op-signer for `key` that answers its `n`th signing request (from zero) with
    /// `answer(n)`, and a handler connected to it. The signer runs until the returned
    /// [`ServerHandle`] is dropped.
    async fn fake_signer(
        key: &PrivateKeySigner,
        answer: fn(usize) -> Answer,
    ) -> (RemoteSignerHandler, Arc<AtomicUsize>, ServerHandle) {
        let calls = Arc::new(AtomicUsize::new(0));
        let server = ServerBuilder::default().build("127.0.0.1:0").await.unwrap();
        let endpoint = format!("http://{}", server.local_addr().unwrap()).parse().unwrap();
        let mut module = RpcModule::new((calls.clone(), key.clone()));
        module.register_method("health_status", |_, _, _| "ok").unwrap();
        module
            .register_async_method("opsigner_signBlockPayload", move |params, ctx, _| async move {
                let (calls, key) = &*ctx;
                match answer(calls.fetch_add(1, Ordering::SeqCst)) {
                    Answer::Error => {
                        Err(ErrorObjectOwned::owned(-32000, "unavailable", None::<()>))
                    }
                    Answer::Stall => std::future::pending().await,
                    Answer::Garbage => Ok(serde_json::json!({ "unexpected": true })),
                    Answer::Sign => {
                        let args: serde_json::Value = params.parse().unwrap();
                        let hash: B256 =
                            serde_json::from_value(args["payloadHash"].clone()).unwrap();
                        let message = PayloadHash(hash).signature_message(CHAIN_ID);
                        let signature = key.sign_hash_sync(&message).unwrap();
                        Ok(serde_json::json!({ "signature": signature.to_string() }))
                    }
                }
            })
            .unwrap();
        let server = server.start(module);
        let handler = RemoteSigner {
            endpoint,
            address: key.address(),
            client_cert: None,
            ca_cert: None,
            headers: Default::default(),
        }
        .start()
        .await
        .unwrap();
        (handler, calls, server)
    }

    fn expected_signature(key: &PrivateKeySigner, hash: PayloadHash) -> Signature {
        key.sign_hash_sync(&hash.signature_message(CHAIN_ID)).unwrap()
    }

    #[tokio::test]
    async fn error_responses_are_retried_until_the_signer_signs() {
        let key = PrivateKeySigner::random();
        let (handler, calls, _server) =
            fake_signer(&key, |n| if n < 3 { Answer::Error } else { Answer::Sign }).await;
        let hash = PayloadHash(B256::repeat_byte(7));
        let signature = handler.sign_block_v1(hash, CHAIN_ID, key.address()).await.unwrap();
        assert_eq!(signature, expected_signature(&key, hash));
        assert_eq!(calls.load(Ordering::SeqCst), 4);
    }

    #[tokio::test]
    async fn a_request_without_an_answer_times_out_and_is_retried() {
        let key = PrivateKeySigner::random();
        let (handler, calls, _server) =
            fake_signer(&key, |n| if n == 0 { Answer::Stall } else { Answer::Sign }).await;
        let hash = PayloadHash(B256::repeat_byte(7));
        let signature = handler.sign_block_v1(hash, CHAIN_ID, key.address()).await.unwrap();
        assert_eq!(signature, expected_signature(&key, hash));
        assert_eq!(calls.load(Ordering::SeqCst), 2);
    }

    #[tokio::test]
    async fn an_undecodable_response_is_not_retried() {
        let key = PrivateKeySigner::random();
        let (handler, calls, _server) = fake_signer(&key, |_| Answer::Garbage).await;
        let result =
            handler.sign_block_v1(PayloadHash(B256::repeat_byte(7)), CHAIN_ID, key.address()).await;
        assert!(matches!(
            result,
            Err(RemoteSignerError::SigningRPCError(RpcError::DeserError { .. }))
        ));
        assert_eq!(calls.load(Ordering::SeqCst), 1);
    }
}
