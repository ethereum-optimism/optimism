//! Standard node RPC transports. Deadlines include receiving the complete response body.

use alloy_primitives::Bytes;
use alloy_provider::RootProvider;
use alloy_rpc_client::{ClientBuilder, RpcClient};
use alloy_rpc_types_engine::JwtSecret;
use alloy_transport::{TransportErrorKind, TransportFut};
use alloy_transport_http::{
    AuthLayer, AuthService, Http, HyperClient,
    hyper_util::{
        client::legacy::{Client, connect::HttpConnector},
        rt::TokioExecutor,
    },
};
use http_body_util::Full;
use op_alloy_network::Optimism;
use std::time::Duration;
use tower::{ServiceBuilder, util::MapFutureLayer};
use url::Url;

/// A Hyper HTTP client with a JWT authentication layer.
pub type HyperAuthClient<B = Full<Bytes>> = HyperClient<B, AuthService<Client<HttpConnector, B>>>;

// The HTTP client completes at response headers; the alloy transport also reads the body.
fn deadline(
    timeout: Duration,
) -> MapFutureLayer<impl Fn(TransportFut<'static>) -> TransportFut<'static> + Clone> {
    MapFutureLayer::new(move |request: TransportFut<'static>| -> TransportFut<'static> {
        Box::pin(async move {
            tokio::time::timeout(timeout, request)
                .await
                .map_err(|_| TransportErrorKind::custom_str("RPC request timed out"))?
        })
    })
}

/// Creates an HTTP RPC client whose complete responses are bounded by `timeout`.
pub fn rpc_client(url: Url, timeout: Duration) -> RpcClient {
    ClientBuilder::default().layer(deadline(timeout)).http(url)
}

/// Creates an L1 provider whose individual requests are bounded by `timeout`.
pub fn l1_provider(url: Url, timeout: Duration) -> RootProvider {
    RootProvider::new(rpc_client(url, timeout))
}

/// Creates an authenticated L2 provider whose individual requests are bounded by `timeout`.
///
/// The Engine API, ordinary execution reads and capability exchange share this connection.
pub fn l2_provider(url: Url, jwt: JwtSecret, timeout: Duration) -> RootProvider<Optimism> {
    let hyper_client = Client::builder(TokioExecutor::new()).build_http::<Full<Bytes>>();
    let service = ServiceBuilder::new().layer(AuthLayer::new(jwt)).service(hyper_client);
    let http = Http::with_client(HyperClient::with_service(service), url);
    RootProvider::new(ClientBuilder::default().layer(deadline(timeout)).transport(http, false))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{EngineClient, OpEngineClient};
    use alloy_eips::{BlockId, BlockNumberOrTag};
    use alloy_provider::Provider;
    use kona_genesis::RollupConfig;
    use std::sync::Arc;
    use tokio::{
        io::{AsyncReadExt, AsyncWriteExt},
        net::TcpListener,
        sync::mpsc,
        time,
    };

    // Hold the connection open after either no response or headers without a complete body.
    async fn stalled_endpoint(response: &'static [u8], answered: mpsc::UnboundedSender<()>) -> Url {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let url = format!("http://{}", listener.local_addr().unwrap()).parse().unwrap();
        tokio::spawn(async move {
            while let Ok((mut stream, _)) = listener.accept().await {
                let answered = answered.clone();
                tokio::spawn(async move {
                    let mut request = [0; 4096];
                    let _ = stream.read(&mut request).await;
                    stream.write_all(response).await.unwrap();
                    answered.send(()).unwrap();
                    std::future::pending::<()>().await
                });
            }
        });
        url
    }

    async fn fails_at_deadline(
        call: impl Future<Output = bool> + Send + 'static,
        timeout: Duration,
        answered: &mut mpsc::UnboundedReceiver<()>,
    ) {
        let call = tokio::spawn(call);
        // Do real socket I/O before pausing the clock.
        answered.recv().await.unwrap();
        time::pause();
        let paused_at = time::Instant::now();
        assert!(time::timeout(timeout * 2, call).await.expect("request has no deadline").unwrap());
        let waited = paused_at.elapsed();
        assert!(waited > timeout - Duration::from_secs(1));
        assert!(waited <= timeout + Duration::from_millis(1));
        time::resume();
    }

    #[rstest::rstest]
    #[case::no_response(b"")]
    #[case::stalled_body(
        b"HTTP/1.1 200 OK\r\ncontent-type: application/json\r\ncontent-length: 64\r\n\r\n"
    )]
    #[tokio::test]
    async fn stalled_engine_and_l1_requests_have_a_deadline(#[case] response: &'static [u8]) {
        let timeout = Duration::from_secs(10);
        let (answered_tx, mut answered) = mpsc::unbounded_channel();
        let client = OpEngineClient::new(
            l1_provider(stalled_endpoint(response, answered_tx.clone()).await, timeout),
            l2_provider(
                stalled_endpoint(response, answered_tx).await,
                JwtSecret::random(),
                timeout,
            ),
            Arc::new(RollupConfig::default()),
        );
        let engine = client.clone();
        fails_at_deadline(
            async move {
                engine.get_l2_block(BlockNumberOrTag::Latest.into()).full().await.is_err()
            },
            timeout,
            &mut answered,
        )
        .await;
        fails_at_deadline(
            async move { EngineClient::get_l1_block(&client, BlockId::latest()).await.is_err() },
            timeout,
            &mut answered,
        )
        .await;
    }

    #[rstest::rstest]
    #[case::short_no_response(2, b"")]
    #[case::long_no_response(20, b"")]
    #[case::short_stalled_body(
        2,
        b"HTTP/1.1 200 OK\r\ncontent-type: application/json\r\ncontent-length: 64\r\n\r\n"
    )]
    #[case::long_stalled_body(
        20,
        b"HTTP/1.1 200 OK\r\ncontent-type: application/json\r\ncontent-length: 64\r\n\r\n"
    )]
    #[tokio::test]
    async fn configured_l2_deadline_replaces_the_default(
        #[case] seconds: u64,
        #[case] response: &'static [u8],
    ) {
        let timeout = Duration::from_secs(seconds);
        let (answered_tx, mut answered) = mpsc::unbounded_channel();
        let l2 = l2_provider(
            stalled_endpoint(response, answered_tx).await,
            JwtSecret::random(),
            timeout,
        );
        fails_at_deadline(
            async move { l2.get_block_number().await.is_err() },
            timeout,
            &mut answered,
        )
        .await;
    }

    #[tokio::test]
    async fn l2_requests_use_the_supplied_jwt() {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let url = format!("http://{}", listener.local_addr().unwrap()).parse().unwrap();
        let request = tokio::spawn(async move {
            let (mut stream, _) = listener.accept().await.unwrap();
            let mut bytes = Vec::new();
            let header_end = loop {
                let mut chunk = [0; 4096];
                let n = stream.read(&mut chunk).await.unwrap();
                assert_ne!(n, 0);
                bytes.extend_from_slice(&chunk[..n]);
                if let Some(end) = bytes.windows(4).position(|window| window == b"\r\n\r\n") {
                    break end + 4;
                }
            };
            let headers = std::str::from_utf8(&bytes[..header_end]).unwrap().to_owned();
            let content_length: usize = headers
                .lines()
                .find_map(|line| {
                    let (name, value) = line.split_once(':')?;
                    name.eq_ignore_ascii_case("content-length")
                        .then(|| value.trim().parse().unwrap())
                })
                .expect("request content length");
            let body_end = header_end + content_length;
            while bytes.len() < body_end {
                let mut chunk = [0; 4096];
                let n = stream.read(&mut chunk).await.unwrap();
                assert_ne!(n, 0);
                bytes.extend_from_slice(&chunk[..n]);
            }
            let request: serde_json::Value =
                serde_json::from_slice(&bytes[header_end..body_end]).unwrap();
            let body = serde_json::json!({"jsonrpc":"2.0", "id":request["id"], "result":"0x7"})
                .to_string();
            stream.write_all(format!("HTTP/1.1 200 OK\r\ncontent-type: application/json\r\ncontent-length: {}\r\n\r\n{}", body.len(), body).as_bytes()).await.unwrap();
            headers
        });
        let jwt = JwtSecret::random();
        let provider = l2_provider(url, jwt, Duration::from_secs(10));
        assert_eq!(provider.get_block_number().await.unwrap(), 7);
        let headers = request.await.unwrap();
        let authorization = headers
            .lines()
            .find_map(|line| {
                let (name, value) = line.split_once(':')?;
                name.eq_ignore_ascii_case("authorization").then_some(value.trim())
            })
            .expect("JWT authorization header");
        jwt.validate(authorization.strip_prefix("Bearer ").unwrap()).unwrap();
    }
}
