use crate::NodeMode;
use alloy_rpc_types_engine::JwtSecret;
use alloy_transport::{TransportError, TransportErrorKind};
use backon::{ExponentialBuilder, Retryable};
use kona_engine::{EngineClient, EngineClientBuilder};
use kona_genesis::RollupConfig;
use op_alloy_provider::ext::engine::OpEngineApi;
use std::{sync::Arc, time::Duration};
use url::Url;

/// Configuration for the Engine Actor.
#[derive(Debug, Clone)]
pub struct EngineConfig {
    /// The [`RollupConfig`].
    pub config: Arc<RollupConfig>,

    /// The engine rpc url.
    pub l2_url: Url,
    /// The engine jwt secret.
    pub l2_jwt_secret: JwtSecret,

    /// The L1 rpc url.
    pub l1_url: Url,

    /// The mode of operation for the node.
    /// When the node is in sequencer mode, the engine actor will receive requests to build blocks
    /// from the sequencer actor.
    pub mode: NodeMode,
}

/// The execution layer did not accept the engine API JWT secret.
#[derive(Debug, thiserror::Error)]
pub(crate) enum JwtValidationError {
    /// The execution layer rejected the JWT secret.
    #[error(
        "Engine API JWT secret differs from the one specified by \
         --l2.jwt-secret/--l2.jwt-secret-encoded (by default `jwt.hex` in the current directory)"
    )]
    InvalidSignature,
    /// The capability exchange failed for another reason.
    #[error("Failed to exchange capabilities with engine: {0}")]
    CapabilityExchange(TransportError),
}

/// The longest wait between attempts to reach the execution layer.
const MAX_HANDSHAKE_RETRY_DELAY: Duration = Duration::from_secs(10);

impl EngineConfig {
    /// Builds and returns the [`EngineClient`].
    pub fn build_engine_client(self) -> EngineClient {
        EngineClientBuilder {
            l2: self.l2_url,
            l2_jwt: self.l2_jwt_secret,
            l1_rpc: self.l1_url,
            cfg: self.config,
        }
        .build()
    }

    /// Checks that the execution layer accepts the JWT secret by exchanging capabilities with it.
    ///
    /// The execution layer often starts after the node, so this retries until it answers, but
    /// fails as soon as it rejects the secret.
    pub(crate) async fn validate_jwt(&self) -> Result<(), JwtValidationError> {
        let engine = EngineClient::rpc_client(self.l2_url.clone(), self.l2_jwt_secret);
        let exchange = || async {
            engine.exchange_capabilities(vec![]).await.map(drop).map_err(|e| {
                // The engine API answers 401 Unauthorized to a request with a bad JWT.
                let unauthorized = e
                    .as_transport_err()
                    .and_then(TransportErrorKind::as_http_error)
                    .is_some_and(|http| http.status == 401);
                if unauthorized {
                    JwtValidationError::InvalidSignature
                } else {
                    JwtValidationError::CapabilityExchange(e)
                }
            })
        };
        exchange
            .retry(
                ExponentialBuilder::default()
                    .with_max_delay(MAX_HANDSHAKE_RETRY_DELAY)
                    .without_max_times(),
            )
            .when(|e| !matches!(e, JwtValidationError::InvalidSignature))
            .notify(|error, delay| {
                warn!(target: "engine", %error, ?delay, "Engine capability handshake failed; retrying");
            })
            .await?;
        debug!(target: "engine", "Successfully exchanged capabilities with engine");
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::{AtomicUsize, Ordering};
    use tokio::{
        io::{AsyncReadExt, AsyncWriteExt},
        net::TcpListener,
    };

    /// Serves `responses` in turn, one per connection, and counts the requests served.
    async fn serve(responses: Vec<&'static str>) -> (EngineConfig, Arc<AtomicUsize>) {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        let requests = Arc::new(AtomicUsize::new(0));
        let served = requests.clone();
        tokio::spawn(async move {
            for response in responses {
                let (mut stream, _) = listener.accept().await.unwrap();
                served.fetch_add(1, Ordering::Relaxed);
                let _ = stream.read(&mut [0; 4096]).await.unwrap();
                stream.write_all(response.as_bytes()).await.unwrap();
            }
        });
        let config = EngineConfig {
            config: Arc::new(RollupConfig::default()),
            l2_url: format!("http://{addr}").parse().unwrap(),
            l2_jwt_secret: JwtSecret::random(),
            l1_url: format!("http://{addr}").parse().unwrap(),
            mode: NodeMode::Validator,
        };
        (config, requests)
    }

    #[tokio::test]
    async fn a_rejected_secret_is_not_retried() {
        let (config, requests) = serve(vec![
            "HTTP/1.1 401 Unauthorized\r\ncontent-length: 26\r\nconnection: close\r\n\r\n\
             token signature is invalid",
        ])
        .await;
        assert!(matches!(config.validate_jwt().await, Err(JwtValidationError::InvalidSignature)));
        assert_eq!(requests.load(Ordering::Relaxed), 1);
    }

    #[tokio::test]
    async fn an_unavailable_engine_is_retried() {
        let (config, requests) = serve(vec![
            "HTTP/1.1 503 Service Unavailable\r\ncontent-length: 0\r\nconnection: close\r\n\r\n",
            "HTTP/1.1 200 OK\r\ncontent-type: application/json\r\ncontent-length: 36\r\n\
             connection: close\r\n\r\n{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":[]}",
        ])
        .await;
        config.validate_jwt().await.unwrap();
        assert_eq!(requests.load(Ordering::Relaxed), 2);
    }
}
