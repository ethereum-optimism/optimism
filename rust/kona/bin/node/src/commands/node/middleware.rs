//! Metrics for the node's outgoing JSON-RPC requests, recorded in the transport.

use alloy_json_rpc::{RequestPacket, ResponsePacket, RpcError};
use alloy_transport::{TransportError, TransportErrorKind, TransportFut, TransportResult};
use jsonrpsee_types::error::{
    INTERNAL_ERROR_CODE, INVALID_PARAMS_CODE, INVALID_REQUEST_CODE, METHOD_NOT_FOUND_CODE,
    PARSE_ERROR_CODE,
};
use std::{
    borrow::Cow,
    task::{Context, Poll},
    time::Instant,
};
use tokio::time::error::Elapsed;
use tower::{Layer, Service};

/// Identifier for the histogram of outgoing JSON-RPC request durations, labelled by `client`,
/// `method`, `outcome` and `error`. Its count also gives the number of requests and errors.
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
            let (outcome, error) = classify(&response);
            metrics::histogram!(
                RPC_CLIENT_REQUEST_DURATION,
                "client" => client,
                "method" => method,
                "outcome" => outcome,
                "error" => error,
            )
            .record(start.elapsed().as_secs_f64());
            response
        })
    }
}

/// Classifies a transport's result as its `outcome` and `error` labels.
///
/// `outcome` is `ok`, `rpc_error` or `transport_error`, and `error` says which error, or `none`.
/// Every value comes from a fixed set, apart from the HTTP status, which HTTP bounds, so a
/// misbehaving endpoint cannot create unbounded label values.
fn classify(response: &TransportResult<ResponsePacket>) -> (&'static str, Cow<'static, str>) {
    match response {
        Ok(packet) => packet.first_error_code().map_or(("ok", Cow::Borrowed("none")), |code| {
            ("rpc_error", Cow::Borrowed(rpc_error(code)))
        }),
        Err(error) => ("transport_error", transport_error(error)),
    }
}

/// Names a JSON-RPC error code. The reserved codes are named individually, and the range reserved
/// for server errors, which includes geth's "header not found" and providers' rate limits, is
/// grouped.
fn rpc_error(code: i64) -> &'static str {
    // jsonrpsee's codes are `i32`, so a code outside that range is none of them.
    let Ok(code) = i32::try_from(code) else { return "other" };
    match code {
        PARSE_ERROR_CODE => "parse",
        INVALID_REQUEST_CODE => "invalid_request",
        METHOD_NOT_FOUND_CODE => "method_not_found",
        INVALID_PARAMS_CODE => "invalid_params",
        INTERNAL_ERROR_CODE => "internal",
        // The JSON-RPC 2.0 specification reserves this range for implementation-defined errors.
        -32099..=-32000 => "server",
        // The code that geth and other execution clients give a reverted `eth_call`.
        3 => "execution_reverted",
        _ => "other",
    }
}

/// Names a failure to get a JSON-RPC response at all.
fn transport_error(error: &TransportError) -> Cow<'static, str> {
    match error {
        RpcError::Transport(
            TransportErrorKind::HttpError(http) |
            TransportErrorKind::HttpErrorWithRetryAfter { error: http, .. },
        ) => Cow::Owned(format!("http_{}", http.status)),
        // The request deadline (`kona_engine::with_deadline`) fails with `Elapsed`.
        RpcError::Transport(TransportErrorKind::Custom(source)) if source.is::<Elapsed>() => {
            Cow::Borrowed("timeout")
        }
        RpcError::Transport(TransportErrorKind::Custom(source))
            if source.downcast_ref::<reqwest::Error>().is_some_and(reqwest::Error::is_connect) =>
        {
            Cow::Borrowed("connect")
        }
        RpcError::DeserError { .. } => Cow::Borrowed("decode"),
        _ => Cow::Borrowed("other"),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_json_rpc::{Id, Request};
    use metrics_exporter_prometheus::PrometheusBuilder;
    use std::time::Duration;
    use tower::util::MapFutureLayer;

    /// A request for `method`.
    fn request(method: &'static str) -> RequestPacket {
        RequestPacket::Single(Request::new(method, Id::Number(1), ()).serialize().unwrap())
    }

    /// A response packet parsed from `json`.
    fn packet(json: &str) -> ResponsePacket {
        serde_json::from_str(json).unwrap()
    }

    /// A response packet carrying a JSON-RPC error with `code`.
    fn rpc_error_packet(code: i64) -> ResponsePacket {
        packet(&format!(r#"{{"jsonrpc":"2.0","id":1,"error":{{"code":{code},"message":"x"}}}}"#))
    }

    /// Sends one request for `method` through an [`RpcMetricsLayer`] over a transport that answers
    /// with `response`.
    fn send(method: &'static str, response: TransportResult<ResponsePacket>) {
        let mut response = Some(response);
        let mut service = RpcMetricsLayer::new("l1").layer(tower::service_fn(move |_| {
            let response = response.take().expect("the transport is called once");
            async move { response }
        }));
        let call = service.call(request(method));
        let _ = tokio::runtime::Builder::new_current_thread().build().unwrap().block_on(call);
    }

    #[test]
    fn records_each_request_by_client_method_outcome_and_error() {
        let recorder = PrometheusBuilder::new().build_recorder();
        let metrics = recorder.handle();
        metrics::with_local_recorder(&recorder, || {
            send("eth_getBlockByHash", Ok(packet(r#"{"jsonrpc":"2.0","id":1,"result":null}"#)));
            send("debug_getRawReceipts", Ok(rpc_error_packet(-32601)));
            send("eth_blockNumber", Err(TransportErrorKind::http_error(503, String::new())));
        });

        let rendered = metrics.render();
        for (method, outcome, error) in [
            ("eth_getBlockByHash", "ok", "none"),
            ("debug_getRawReceipts", "rpc_error", "method_not_found"),
            ("eth_blockNumber", "transport_error", "http_503"),
        ] {
            let count = format!(
                r#"{RPC_CLIENT_REQUEST_DURATION}_count{{client="l1",method="{method}",outcome="{outcome}",error="{error}"}} 1"#
            );
            assert!(rendered.contains(&count), "missing {count} in:\n{rendered}");
        }
    }

    #[test]
    fn classifies_responses_and_rpc_errors() {
        assert_eq!(
            classify(&Ok(packet(r#"{"jsonrpc":"2.0","id":1,"result":"0x1"}"#))),
            ("ok", Cow::Borrowed("none"))
        );
        for (code, error) in [
            (-32700, "parse"),
            (-32600, "invalid_request"),
            (-32601, "method_not_found"),
            (-32602, "invalid_params"),
            (-32603, "internal"),
            (-32000, "server"),
            (-32005, "server"),
            (-32099, "server"),
            (3, "execution_reverted"),
            (-32100, "other"),
            (42, "other"),
            // Truncated to `i32`, this would be -32601.
            (-32601 - (1 << 32), "other"),
        ] {
            assert_eq!(
                classify(&Ok(rpc_error_packet(code))),
                ("rpc_error", Cow::Borrowed(error)),
                "code {code}"
            );
        }
    }

    #[test]
    fn classifies_transport_errors() {
        for (error, expected) in [
            (TransportErrorKind::http_error(429, String::new()), "http_429"),
            (
                TransportErrorKind::http_error_with_retry_after(
                    503,
                    String::new(),
                    Some(Duration::from_secs(1)),
                ),
                "http_503",
            ),
            (
                TransportError::deser_err(serde_json::from_str::<u8>("x").unwrap_err(), "x"),
                "decode",
            ),
            (TransportErrorKind::custom_str("unreachable"), "other"),
            (TransportErrorKind::backend_gone(), "other"),
        ] {
            assert_eq!(classify(&Err(error)), ("transport_error", Cow::Borrowed(expected)));
        }
    }

    #[tokio::test(start_paused = true)]
    async fn the_request_deadline_is_classified_as_a_timeout() {
        // A transport that never answers.
        let mut service = MapFutureLayer::new(kona_engine::with_deadline).layer(tower::service_fn(
            |_| -> TransportFut<'static> { Box::pin(std::future::pending()) },
        ));

        let response = service.call(request("eth_blockNumber")).await;

        assert_eq!(classify(&response), ("transport_error", Cow::Borrowed("timeout")));
    }

    #[tokio::test]
    async fn a_refused_connection_is_classified_as_a_connect_error() {
        // Port 1 is reserved and nothing listens on it, so the connection is refused. This goes
        // through alloy's real HTTP transport, so it also checks that the `reqwest` we downcast
        // to is the one alloy uses.
        let mut transport = alloy_transport_http::Http::new("http://127.0.0.1:1".parse().unwrap());

        let response = transport.call(request("eth_blockNumber")).await;

        assert_eq!(classify(&response), ("transport_error", Cow::Borrowed("connect")));
    }
}
