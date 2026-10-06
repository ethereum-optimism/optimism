use std::time::Duration;

use alloy_primitives::{Address, B256, ChainId, SignatureError};
use alloy_signer::Signature;
use alloy_transport::{RpcError, TransportError, TransportErrorKind};
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
    /// A transient failure is retried until the signer returns a signature: a network failure,
    /// or an HTTP status meaning the signer or a proxy in front of it is briefly unavailable. An
    /// op-signer JSON-RPC error, any other HTTP status, and any other error are returned after the
    /// first request, so any error this returns is one that retrying cannot fix.
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

/// Whether a failed signing request can succeed if it is sent again.
///
/// Network failures are transient: the connection fails or drops, or the signer does not answer
/// within the request timeout. The HTTP client reports these as [`TransportErrorKind::Custom`]. So
/// are the HTTP statuses that mean the signer or a proxy in front of it is briefly unavailable:
/// 408, 429, 502, 503 and 504. Anything else fails the same way every time: an op-signer JSON-RPC
/// error, whatever its HTTP status, any other HTTP status such as 401 or 403, a request or
/// response that cannot be encoded or decoded, and any other transport error.
///
/// alloy's own retry classification is not used: it does not retry a refused connection or a
/// timeout, which are the main outages of a remote signer.
const fn is_transient(err: &TransportError) -> bool {
    // Treat all op-signer JSON-RPC errors as fatal; retry only transient network failures.
    let RpcError::Transport(kind) = err else { return false };
    match kind {
        TransportErrorKind::Custom(_) => true,
        TransportErrorKind::HttpError(http) |
        TransportErrorKind::HttpErrorWithRetryAfter { error: http, .. } => {
            matches!(http.status, 408 | 429 | 502 | 503 | 504)
        }
        _ => false,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::RemoteSigner;
    use alloy_signer::SignerSync;
    use alloy_signer_local::PrivateKeySigner;
    use serde_json::{Value, json};
    use std::sync::{
        Arc,
        atomic::{AtomicUsize, Ordering},
    };
    use tokio::{
        io::{AsyncReadExt, AsyncWriteExt},
        net::{TcpListener, TcpStream},
    };

    const CHAIN_ID: u64 = 10;

    /// How the fake signer answers a signing request.
    #[derive(Debug, Clone, Copy)]
    enum Answer {
        /// Close the connection without a response.
        Drop,
        /// No response at all.
        Stall,
        /// An HTTP error status with a body that is not JSON-RPC.
        Status(u16),
        /// An op-signer JSON-RPC error response, sent with this HTTP status.
        RpcError(u16),
        /// A result that is not a signature.
        Garbage,
        /// A signature over the payload hash.
        Sign,
    }

    /// Starts a fake op-signer for `key` that answers its `n`th signing request (from zero) with
    /// `answer(n)`, and a handler connected to it. Also returns a counter of the signing requests
    /// the fake receives, which does not count `health_status`.
    async fn fake_signer(
        key: &PrivateKeySigner,
        answer: impl Fn(usize) -> Answer + Copy + Send + 'static,
    ) -> (RemoteSignerHandler, Arc<AtomicUsize>) {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let endpoint = format!("http://{}", listener.local_addr().unwrap()).parse().unwrap();
        let calls = Arc::new(AtomicUsize::new(0));
        let (signer_key, signer_calls) = (key.clone(), calls.clone());
        tokio::spawn(async move {
            while let Ok((stream, _)) = listener.accept().await {
                let (key, calls) = (signer_key.clone(), signer_calls.clone());
                tokio::spawn(serve(stream, key, calls, answer));
            }
        });
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
        (handler, calls)
    }

    /// Answers the one request on `stream`. Each response closes the connection, so every request
    /// arrives on a new one.
    async fn serve(
        mut stream: TcpStream,
        key: PrivateKeySigner,
        calls: Arc<AtomicUsize>,
        answer: impl Fn(usize) -> Answer,
    ) {
        let Some(request) = read_request(&mut stream).await else { return };
        let id = request["id"].clone();
        if request["method"] == "health_status" {
            return respond(
                &mut stream,
                200,
                json!({ "jsonrpc": "2.0", "id": id, "result": "ok" }),
            )
            .await;
        }
        let body = match answer(calls.fetch_add(1, Ordering::SeqCst)) {
            Answer::Drop => return,
            Answer::Stall => std::future::pending().await,
            Answer::Status(status) => {
                return respond(&mut stream, status, json!("unavailable")).await;
            }
            Answer::RpcError(status) => {
                let error = json!({ "code": -32000, "message": "refused" });
                return respond(
                    &mut stream,
                    status,
                    json!({ "jsonrpc": "2.0", "id": id, "error": error }),
                )
                .await;
            }
            Answer::Garbage => {
                json!({ "jsonrpc": "2.0", "id": id, "result": { "unexpected": true } })
            }
            Answer::Sign => {
                // The client sends the arguments as a single object.
                let hash: B256 =
                    serde_json::from_value(request["params"]["payloadHash"].clone()).unwrap();
                let signature = expected_signature(&key, PayloadHash(hash)).to_string();
                json!({ "jsonrpc": "2.0", "id": id, "result": { "signature": signature } })
            }
        };
        respond(&mut stream, 200, body).await;
    }

    /// Reads one HTTP request and returns its JSON body.
    async fn read_request(stream: &mut TcpStream) -> Option<Value> {
        let mut buf = Vec::new();
        let mut chunk = [0; 4096];
        let (head_len, body_len) = loop {
            if let Some(end) = buf.windows(4).position(|w| w == b"\r\n\r\n") {
                let head = String::from_utf8_lossy(&buf[..end]).to_ascii_lowercase();
                let body_len = head
                    .lines()
                    .find_map(|line| line.strip_prefix("content-length:"))
                    .map_or(0, |len| len.trim().parse().unwrap());
                break (end + 4, body_len);
            }
            let read = stream.read(&mut chunk).await.ok().filter(|&n| n > 0)?;
            buf.extend_from_slice(&chunk[..read]);
        };
        while buf.len() < head_len + body_len {
            let read = stream.read(&mut chunk).await.ok().filter(|&n| n > 0)?;
            buf.extend_from_slice(&chunk[..read]);
        }
        serde_json::from_slice(&buf[head_len..head_len + body_len]).ok()
    }

    async fn respond(stream: &mut TcpStream, status: u16, body: Value) {
        let body = body.to_string();
        let response = format!(
            "HTTP/1.1 {status} Fake\r\ncontent-type: application/json\r\n\
             content-length: {}\r\nconnection: close\r\n\r\n{body}",
            body.len()
        );
        let _ = stream.write_all(response.as_bytes()).await;
    }

    fn expected_signature(key: &PrivateKeySigner, hash: PayloadHash) -> Signature {
        key.sign_hash_sync(&hash.signature_message(CHAIN_ID)).unwrap()
    }

    async fn sign(
        handler: &RemoteSignerHandler,
        key: &PrivateKeySigner,
    ) -> Result<Signature, RemoteSignerError> {
        handler.sign_block_v1(PayloadHash(B256::repeat_byte(7)), CHAIN_ID, key.address()).await
    }

    #[tokio::test]
    async fn json_rpc_errors_are_fatal_after_one_request() {
        let key = PrivateKeySigner::random();
        // The error body decides, even when it comes with a status that is otherwise retried.
        for status in [200, 503] {
            let (handler, calls) = fake_signer(&key, move |_| Answer::RpcError(status)).await;
            let result = sign(&handler, &key).await;
            assert!(
                matches!(result, Err(RemoteSignerError::SigningRPCError(RpcError::ErrorResp(_)))),
                "status {status}: unexpected result {result:?}"
            );
            assert_eq!(calls.load(Ordering::SeqCst), 1, "status {status}");
        }
    }

    #[tokio::test]
    async fn dropped_connections_are_retried_until_the_signer_signs() {
        let key = PrivateKeySigner::random();
        let (handler, calls) =
            fake_signer(&key, |n| if n < 2 { Answer::Drop } else { Answer::Sign }).await;
        let signature = sign(&handler, &key).await.unwrap();
        assert_eq!(signature, expected_signature(&key, PayloadHash(B256::repeat_byte(7))));
        assert_eq!(calls.load(Ordering::SeqCst), 3);
    }

    #[tokio::test]
    async fn a_request_without_an_answer_times_out_and_is_retried() {
        let key = PrivateKeySigner::random();
        let (handler, calls) =
            fake_signer(&key, |n| if n == 0 { Answer::Stall } else { Answer::Sign }).await;
        let signature = sign(&handler, &key).await.unwrap();
        assert_eq!(signature, expected_signature(&key, PayloadHash(B256::repeat_byte(7))));
        assert_eq!(calls.load(Ordering::SeqCst), 2);
    }

    #[tokio::test]
    async fn unavailable_statuses_are_retried() {
        let key = PrivateKeySigner::random();
        for status in [408, 429, 502, 503, 504] {
            let (handler, calls) =
                fake_signer(
                    &key,
                    move |n| {
                        if n < 2 { Answer::Status(status) } else { Answer::Sign }
                    },
                )
                .await;
            let signature = sign(&handler, &key).await.unwrap();
            assert_eq!(signature, expected_signature(&key, PayloadHash(B256::repeat_byte(7))));
            assert_eq!(calls.load(Ordering::SeqCst), 3, "status {status}");
        }
    }

    #[tokio::test]
    async fn permanent_http_errors_are_fatal_after_one_request() {
        let key = PrivateKeySigner::random();
        for status in [401, 403] {
            let (handler, calls) = fake_signer(&key, move |_| Answer::Status(status)).await;
            let result = sign(&handler, &key).await;
            let Err(RemoteSignerError::SigningRPCError(RpcError::Transport(kind))) = result else {
                panic!("status {status}: unexpected result {result:?}");
            };
            assert_eq!(kind.as_http_error().map(|http| http.status), Some(status));
            assert_eq!(calls.load(Ordering::SeqCst), 1, "status {status}");
        }
    }

    #[tokio::test]
    async fn an_undecodable_response_is_not_retried() {
        let key = PrivateKeySigner::random();
        let (handler, calls) = fake_signer(&key, |_| Answer::Garbage).await;
        let result = sign(&handler, &key).await;
        assert!(matches!(
            result,
            Err(RemoteSignerError::SigningRPCError(RpcError::DeserError { .. }))
        ));
        assert_eq!(calls.load(Ordering::SeqCst), 1);
    }
}
