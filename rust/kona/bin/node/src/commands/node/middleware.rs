//! Metrics for the node's outgoing JSON-RPC requests, recorded in the transport.

use alloy_json_rpc::{RequestPacket, ResponsePacket};
use alloy_transport::{TransportError, TransportFut};
use std::{
    task::{Context, Poll},
    time::Instant,
};
use tower::{Layer, Service};

/// Identifier for the histogram of outgoing JSON-RPC request durations, labelled by `client`,
/// `method` and `outcome`. Its count also gives the number of requests and errors.
const RPC_CLIENT_REQUEST_DURATION: &str = "kona_rpc_client_request_duration_seconds";

/// Describes the metrics recorded by [`RpcMetricsLayer`].
pub(super) fn describe() {
    metrics::describe_histogram!(
        RPC_CLIENT_REQUEST_DURATION,
        metrics::Unit::Seconds,
        "Duration of outgoing JSON-RPC requests"
    );
}

/// Records the duration and outcome of every request sent through a JSON-RPC transport.
#[derive(Debug, Clone, Copy)]
pub(super) struct RpcMetricsLayer {
    /// The `client` label on the recorded metrics.
    client: &'static str,
}

impl RpcMetricsLayer {
    /// Creates a layer that labels its metrics with `client`.
    pub(super) const fn new(client: &'static str) -> Self {
        Self { client }
    }
}

impl<S> Layer<S> for RpcMetricsLayer {
    type Service = RpcMetrics<S>;

    fn layer(&self, inner: S) -> Self::Service {
        RpcMetrics { inner, client: self.client }
    }
}

/// The transport service that an [`RpcMetricsLayer`] wraps.
#[derive(Debug, Clone)]
pub(super) struct RpcMetrics<S> {
    inner: S,
    client: &'static str,
}

impl<S> Service<RequestPacket> for RpcMetrics<S>
where
    S: Service<RequestPacket, Response = ResponsePacket, Error = TransportError>,
    S::Future: Send + 'static,
{
    type Response = ResponsePacket;
    type Error = TransportError;
    type Future = TransportFut<'static>;

    fn poll_ready(&mut self, cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        self.inner.poll_ready(cx)
    }

    fn call(&mut self, request: RequestPacket) -> Self::Future {
        let client = self.client;
        let method = match &request {
            RequestPacket::Single(request) => request.method().to_string(),
            RequestPacket::Batch(_) => "batch".to_string(),
        };
        let start = Instant::now();
        let response = self.inner.call(request);
        Box::pin(async move {
            let response = response.await;
            let outcome = match &response {
                Ok(packet) if packet.is_error() => "rpc_error",
                Ok(_) => "ok",
                Err(_) => "transport_error",
            };
            metrics::histogram!(
                RPC_CLIENT_REQUEST_DURATION,
                "client" => client,
                "method" => method,
                "outcome" => outcome,
            )
            .record(start.elapsed().as_secs_f64());
            response
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_json_rpc::{Id, Request};
    use alloy_transport::TransportErrorKind;
    use metrics_exporter_prometheus::PrometheusBuilder;

    /// Sends one request for `method` through an [`RpcMetricsLayer`] over a transport that answers
    /// with `response`.
    fn send(method: &'static str, response: Result<ResponsePacket, TransportError>) {
        let request = Request::new(method, Id::Number(1), ()).serialize().unwrap();
        let mut response = Some(response);
        let mut service = RpcMetricsLayer::new("l1").layer(tower::service_fn(move |_| {
            let response = response.take().expect("the transport is called once");
            async move { response }
        }));
        let call = service.call(RequestPacket::Single(request));
        let _ = tokio::runtime::Builder::new_current_thread().build().unwrap().block_on(call);
    }

    #[test]
    fn records_each_request_by_client_method_and_outcome() {
        let recorder = PrometheusBuilder::new().build_recorder();
        let metrics = recorder.handle();
        metrics::with_local_recorder(&recorder, || {
            send(
                "eth_getBlockByHash",
                Ok(serde_json::from_str(r#"{"jsonrpc":"2.0","id":1,"result":null}"#).unwrap()),
            );
            send(
                "debug_getRawReceipts",
                Ok(serde_json::from_str(
                    r#"{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"method not found"}}"#,
                )
                .unwrap()),
            );
            send("eth_blockNumber", Err(TransportErrorKind::custom_str("unreachable")));
        });

        let rendered = metrics.render();
        for (method, outcome) in [
            ("eth_getBlockByHash", "ok"),
            ("debug_getRawReceipts", "rpc_error"),
            ("eth_blockNumber", "transport_error"),
        ] {
            let count = format!(
                r#"{RPC_CLIENT_REQUEST_DURATION}_count{{client="l1",method="{method}",outcome="{outcome}"}} 1"#
            );
            assert!(rendered.contains(&count), "missing {count} in:\n{rendered}");
        }
    }
}
