//! The providers the node reads L1 and L2 through.

use crate::{RpcClientLayer, rpc_client::RequestTimedOut};
use alloy_primitives::Bytes;
use alloy_provider::RootProvider;
use alloy_rpc_client::{ClientBuilder, RpcClient};
use alloy_rpc_types_engine::JwtSecret;
use alloy_transport::{TransportErrorKind, TransportFut};
use alloy_transport_http::{
    AuthLayer, Http, HyperClient,
    hyper_util::{client::legacy::Client, rt::TokioExecutor},
};
use http_body_util::Full;
use op_alloy_network::Optimism;
use std::time::Duration;
use tower::{ServiceBuilder, util::MapFutureLayer};
use url::Url;

/// Deadline for each request a provider sends.
///
/// It bounds one request, never an operation made of many requests, such as the block traversal
/// of a reset, so a long but healthy operation is not cut off.
const RPC_TIMEOUT: Duration = Duration::from_secs(10);

/// Bounds each request on an alloy transport by `timeout`.
///
/// This wraps the transport's future rather than the HTTP client below it, because the HTTP
/// client's future completes when the response headers arrive and the transport reads the body
/// after that.
fn deadline(
    timeout: Duration,
) -> MapFutureLayer<impl Fn(TransportFut<'static>) -> TransportFut<'static> + Clone> {
    MapFutureLayer::new(move |request: TransportFut<'static>| -> TransportFut<'static> {
        Box::pin(async move {
            tokio::time::timeout(timeout, request)
                .await
                .map_err(|_| TransportErrorKind::custom(RequestTimedOut))?
        })
    })
}

/// Returns a provider for the L1 execution layer's RPC at `url`. Each request has a deadline, and
/// [`RpcClientLayer`] records it as client `l1`.
pub fn l1_provider(url: Url) -> RootProvider {
    RootProvider::new(
        ClientBuilder::default()
            .layer(RpcClientLayer::new("l1"))
            .layer(deadline(RPC_TIMEOUT))
            .http(url),
    )
}

/// Returns a provider for the L2 execution layer's authenticated RPC at `url`, which also serves
/// the Engine API, authenticated with `jwt`. Each request has a deadline, and [`RpcClientLayer`]
/// records it as client `l2`.
pub fn l2_provider(url: Url, jwt: JwtSecret) -> RootProvider<Optimism> {
    let hyper_client = Client::builder(TokioExecutor::new()).build_http::<Full<Bytes>>();
    let service = ServiceBuilder::new().layer(AuthLayer::new(jwt)).service(hyper_client);
    let http_hyper = Http::with_client(HyperClient::with_service(service), url);
    let rpc_client = ClientBuilder::default()
        .layer(RpcClientLayer::new("l2"))
        .layer(deadline(RPC_TIMEOUT))
        .transport(http_hyper, false);
    RootProvider::new(rpc_client)
}

/// Returns an RPC client for the op-conductor at `url`. Each request fails after `timeout`, so an
/// unresponsive conductor cannot hold up the sequencer, and [`RpcClientLayer`] records it as client
/// `conductor`.
pub fn conductor_rpc(url: Url, timeout: Duration) -> RpcClient {
    ClientBuilder::default()
        .layer(RpcClientLayer::new("conductor"))
        .layer(deadline(timeout))
        .http(url)
}

#[cfg(test)]
mod deadline_tests {
    use super::*;
    use alloy_provider::Provider;
    use tokio::{
        io::{AsyncReadExt, AsyncWriteExt},
        net::TcpListener,
        sync::mpsc,
        time::{self, Duration},
    };

    /// Reads each request, writes `response`, signals `answered`, then holds the connection open
    /// without finishing the response.
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

    /// Runs `call` in real time until the endpoint has answered, so whatever it sent is already in
    /// the socket buffer, then pauses the clock and expects `call` to fail at its deadline.
    async fn fails_at_deadline(
        call: impl Future<Output = bool> + Send + 'static,
        deadline: Duration,
        answered: &mut mpsc::UnboundedReceiver<()>,
    ) {
        let call = tokio::spawn(call);
        answered.recv().await.unwrap();
        time::pause();
        let paused_at = time::Instant::now();
        let failed = time::timeout(deadline * 2, call).await.expect("request has no deadline");
        assert!(failed.unwrap());
        // The deadline started before the clock was paused, and only the clock moved since. Timers
        // round up to the next millisecond.
        let waited = paused_at.elapsed();
        assert!(waited > deadline - Duration::from_millis(500));
        assert!(waited <= deadline + Duration::from_millis(1));
        time::resume();
    }

    #[rstest::rstest]
    #[case::no_response(b"")]
    #[case::stalled_body(
        b"HTTP/1.1 200 OK\r\ncontent-type: application/json\r\ncontent-length: 64\r\n\r\n"
    )]
    #[tokio::test]
    async fn stalled_requests_have_a_deadline(#[case] response: &'static [u8]) {
        let (answered_tx, mut answered) = mpsc::unbounded_channel();
        let l1 = l1_provider(stalled_endpoint(response, answered_tx.clone()).await);
        let l2 =
            l2_provider(stalled_endpoint(response, answered_tx.clone()).await, JwtSecret::random());
        let conductor_timeout = Duration::from_secs(1);
        let conductor =
            conductor_rpc(stalled_endpoint(response, answered_tx).await, conductor_timeout);

        let l1_call = async move { l1.get_block_number().await.is_err() };
        fails_at_deadline(l1_call, RPC_TIMEOUT, &mut answered).await;
        let l2_call = async move { l2.get_block_number().await.is_err() };
        fails_at_deadline(l2_call, RPC_TIMEOUT, &mut answered).await;
        let conductor_call =
            async move { conductor.request_noparams::<bool>("conductor_leader").await.is_err() };
        fails_at_deadline(conductor_call, conductor_timeout, &mut answered).await;
    }
}
