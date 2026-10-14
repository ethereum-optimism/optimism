//! Metrics and logging for the requests the node's JSON-RPC clients send.

use crate::Metrics;
use alloy_json_rpc::{ErrorPayload, RequestPacket, Response, ResponsePacket, ResponsePayload};
use alloy_transport::{RpcError, TransportError, TransportErrorKind, TransportFut};
use alloy_transport_http::reqwest;
use std::{
    borrow::Cow,
    task::{Context, Poll},
    time::Instant,
};
use tower::{Layer, Service};
use tracing::{debug, trace, warn};

/// The error a request fails with when it does not complete before its deadline.
#[derive(Debug, Clone, Copy, thiserror::Error)]
#[error("RPC request timed out")]
pub(crate) struct RequestTimedOut;

/// Records metrics and logs for every request a JSON-RPC client sends, labeled by the client's
/// name and the request's method. A batch is recorded once, under the method `batch`.
///
/// Per request it counts the request and the bytes of its params, times it, and counts the
/// response by its error, which is one of:
/// - `none` for a successful response, whose result bytes are counted;
/// - `rpc_<code>` for a JSON-RPC error response, or the first error in a batch;
/// - `http_<status>` when the server answers with an HTTP error status instead of a JSON-RPC
///   response, such as a `401` for a rejected JWT;
/// - `timeout` when the request misses a deadline set by a layer below this one, or the HTTP
///   client's timeout;
/// - `transport` when no response arrives for any other reason, or it cannot be read.
///
/// It logs successful requests at `trace`, JSON-RPC errors at `debug` with their message since
/// callers decide whether they matter, and HTTP errors and requests that got no response at
/// `warn`. Params and results are never logged.
#[derive(Debug, Clone, Copy)]
pub struct RpcClientLayer {
    client: &'static str,
}

impl RpcClientLayer {
    /// Creates the layer for the client named `client`, such as `l1`.
    pub const fn new(client: &'static str) -> Self {
        Self { client }
    }
}

impl<S> Layer<S> for RpcClientLayer {
    type Service = RpcClientService<S>;

    fn layer(&self, inner: S) -> Self::Service {
        RpcClientService { inner, client: self.client }
    }
}

/// The transport service that an [`RpcClientLayer`] wraps.
#[derive(Debug, Clone)]
pub struct RpcClientService<S> {
    inner: S,
    client: &'static str,
}

impl<S> Service<RequestPacket> for RpcClientService<S>
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
        // Reading the params parses the serialized request again, which is cheap next to sending
        // it.
        let (method, id, params_size): (Cow<'static, str>, _, usize) = match &request {
            RequestPacket::Single(request) => (
                request.method_clone(),
                Some(request.id().clone()),
                request.params().map_or(0, |params| params.get().len()),
            ),
            RequestPacket::Batch(requests) => (
                Cow::Borrowed("batch"),
                None,
                requests
                    .iter()
                    .filter_map(|request| request.params())
                    .map(|params| params.get().len())
                    .sum(),
            ),
        };
        let labels = [("client", Cow::Borrowed(client)), ("method", method.clone())];
        metrics::counter!(Metrics::RPC_CLIENT_REQUESTS, &labels).increment(1);
        metrics::counter!(Metrics::RPC_CLIENT_PARAMS_SIZE, &labels).increment(params_size as u64);

        let response = self.inner.call(request);
        Box::pin(async move {
            let start = Instant::now();
            let response = response.await;
            let elapsed = start.elapsed();
            metrics::histogram!(Metrics::RPC_CLIENT_REQUEST_DURATION, &labels)
                .record(elapsed.as_secs_f64());

            let outcome = Outcome::of(&response);
            let error = outcome.label();
            metrics::counter!(
                Metrics::RPC_CLIENT_RESPONSES,
                &[
                    ("client", Cow::Borrowed(client)),
                    ("method", method.clone()),
                    ("error", error.clone())
                ]
            )
            .increment(1);
            match outcome {
                Outcome::Success { results_size } => {
                    metrics::counter!(Metrics::RPC_CLIENT_RESULTS_SIZE, &labels)
                        .increment(results_size as u64);
                    trace!(target: "rpc_client", client, %method, ?id, ?elapsed, "RPC request succeeded");
                }
                Outcome::RpcError(payload) => {
                    debug!(target: "rpc_client", client, %method, ?id, ?elapsed, %error, message = %payload.message, "RPC request failed");
                }
                Outcome::Http(_) => {
                    warn!(target: "rpc_client", client, %method, ?id, ?elapsed, %error, "RPC request failed with an HTTP error");
                }
                Outcome::Timeout | Outcome::Transport => {
                    let reason = response.as_ref().err().map(ToString::to_string);
                    warn!(target: "rpc_client", client, %method, ?id, ?elapsed, %error, ?reason, "RPC request got no response");
                }
            }
            response
        })
    }
}

/// How a request ended, for its metrics and log.
#[derive(Debug, Clone, Copy)]
enum Outcome<'a> {
    /// Every response succeeded, with this many bytes of results.
    Success { results_size: usize },
    /// The response, or the first failed one in a batch, is this JSON-RPC error.
    RpcError(&'a ErrorPayload),
    /// The server answered with this HTTP error status instead of a JSON-RPC response.
    Http(u16),
    /// The request missed its deadline.
    Timeout,
    /// No response arrived for another reason, or it could not be read.
    Transport,
}

impl<'a> Outcome<'a> {
    fn of(response: &'a Result<ResponsePacket, TransportError>) -> Self {
        let responses: &[Response] = match response {
            Ok(ResponsePacket::Single(response)) => std::slice::from_ref(response),
            Ok(ResponsePacket::Batch(responses)) => responses,
            Err(RpcError::Transport(kind)) => {
                if let Some(error) = kind.as_http_error() {
                    return Self::Http(error.status);
                }
                return match kind {
                    TransportErrorKind::Custom(error)
                        if error.is::<RequestTimedOut>() ||
                            error
                                .downcast_ref::<reqwest::Error>()
                                .is_some_and(reqwest::Error::is_timeout) =>
                    {
                        Self::Timeout
                    }
                    _ => Self::Transport,
                };
            }
            Err(_) => return Self::Transport,
        };
        let mut results_size = 0;
        for response in responses {
            match &response.payload {
                ResponsePayload::Success(result) => results_size += result.get().len(),
                ResponsePayload::Failure(error) => return Self::RpcError(error),
            }
        }
        Self::Success { results_size }
    }

    /// The `error` label of the response.
    fn label(self) -> Cow<'static, str> {
        match self {
            Self::Success { .. } => Cow::Borrowed("none"),
            Self::RpcError(error) => Cow::Owned(format!("rpc_{}", error.code)),
            Self::Http(status) => Cow::Owned(format!("http_{status}")),
            Self::Timeout => Cow::Borrowed("timeout"),
            Self::Transport => Cow::Borrowed("transport"),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_json_rpc::{Id, Request};
    use metrics_exporter_prometheus::PrometheusBuilder;
    use serde_json::value::RawValue;

    /// Sends one request for `method` with `params` through an [`RpcClientLayer`] named `test`,
    /// over a transport that answers with `response`.
    fn send(
        method: &'static str,
        params: impl alloy_json_rpc::RpcSend,
        response: Result<ResponsePacket, TransportError>,
    ) {
        let request = Request::new(method, Id::Number(1), params).serialize().unwrap();
        let mut response = Some(response);
        let mut service = RpcClientLayer::new("test").layer(tower::service_fn(move |_| {
            let response = response.take().expect("the transport is called once");
            async move { response }
        }));
        let call = service.call(RequestPacket::Single(request));
        let _ = tokio::runtime::Builder::new_current_thread().build().unwrap().block_on(call);
    }

    fn reply(payload: ResponsePayload) -> Result<ResponsePacket, TransportError> {
        Ok(ResponsePacket::Single(Response { id: Id::Number(1), payload }))
    }

    #[test]
    fn records_each_request_by_client_method_and_error() {
        let recorder = PrometheusBuilder::new().build_recorder();
        let metrics = recorder.handle();
        metrics::with_local_recorder(&recorder, || {
            send(
                "eth_chainId",
                [1u8],
                reply(ResponsePayload::Success(RawValue::from_string("\"0xa\"".into()).unwrap())),
            );
            send(
                "eth_getBlockByNumber",
                (),
                reply(ResponsePayload::Failure(ErrorPayload {
                    code: -32000,
                    message: "header not found".into(),
                    data: None,
                })),
            );
            send("eth_blockNumber", (), Err(TransportErrorKind::custom(RequestTimedOut)));
            send("eth_syncing", (), Err(TransportErrorKind::http_error(401, String::new())));
            send("eth_gasPrice", (), Err(TransportErrorKind::custom_str("connection refused")));
        });

        let rendered = metrics.render();
        for expected in [
            format!("{}{{client=\"test\",method=\"eth_chainId\"}} 1", Metrics::RPC_CLIENT_REQUESTS),
            // The serialized params, `[1]`.
            format!(
                "{}{{client=\"test\",method=\"eth_chainId\"}} 3",
                Metrics::RPC_CLIENT_PARAMS_SIZE
            ),
            // The serialized result, `"0xa"`.
            format!(
                "{}{{client=\"test\",method=\"eth_chainId\"}} 5",
                Metrics::RPC_CLIENT_RESULTS_SIZE
            ),
            format!(
                "{}_count{{client=\"test\",method=\"eth_chainId\"}} 1",
                Metrics::RPC_CLIENT_REQUEST_DURATION
            ),
            format!(
                "{}{{client=\"test\",method=\"eth_chainId\",error=\"none\"}} 1",
                Metrics::RPC_CLIENT_RESPONSES
            ),
            format!(
                "{}{{client=\"test\",method=\"eth_getBlockByNumber\",error=\"rpc_-32000\"}} 1",
                Metrics::RPC_CLIENT_RESPONSES
            ),
            format!(
                "{}{{client=\"test\",method=\"eth_blockNumber\",error=\"timeout\"}} 1",
                Metrics::RPC_CLIENT_RESPONSES
            ),
            format!(
                "{}{{client=\"test\",method=\"eth_syncing\",error=\"http_401\"}} 1",
                Metrics::RPC_CLIENT_RESPONSES
            ),
            format!(
                "{}{{client=\"test\",method=\"eth_gasPrice\",error=\"transport\"}} 1",
                Metrics::RPC_CLIENT_RESPONSES
            ),
            format!(
                "{}_count{{client=\"test\",method=\"eth_gasPrice\"}} 1",
                Metrics::RPC_CLIENT_REQUEST_DURATION
            ),
        ] {
            assert!(rendered.contains(&expected), "missing {expected} in:\n{rendered}");
        }
    }

    #[test]
    fn an_http_client_timeout_is_a_timeout() {
        let recorder = PrometheusBuilder::new().build_recorder();
        let metrics = recorder.handle();
        let runtime = tokio::runtime::Builder::new_current_thread().enable_all().build().unwrap();
        metrics::with_local_recorder(&recorder, || {
            runtime.block_on(async {
                // Accepts connections and never answers.
                let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
                let url = format!("http://{}", listener.local_addr().unwrap()).parse().unwrap();
                tokio::spawn(async move {
                    while let Ok((stream, _)) = listener.accept().await {
                        tokio::spawn(async move {
                            let _stream = stream;
                            std::future::pending::<()>().await
                        });
                    }
                });
                let http = reqwest::Client::builder()
                    .timeout(std::time::Duration::from_millis(10))
                    .build()
                    .unwrap();
                let client = alloy_rpc_client::ClientBuilder::default()
                    .layer(RpcClientLayer::new("test"))
                    .transport(alloy_transport_http::Http::with_client(http, url), true);
                let response: Result<String, _> = client.request_noparams("eth_chainId").await;
                assert!(response.is_err());
            });
        });

        let expected = format!(
            "{}{{client=\"test\",method=\"eth_chainId\",error=\"timeout\"}} 1",
            Metrics::RPC_CLIENT_RESPONSES
        );
        let rendered = metrics.render();
        assert!(rendered.contains(&expected), "missing {expected} in:\n{rendered}");
    }

    #[test]
    fn a_batch_is_recorded_once_by_its_first_error() {
        let success = Response {
            id: Id::Number(1),
            payload: ResponsePayload::Success(RawValue::from_string("1".into()).unwrap()),
        };
        let failure = Response {
            id: Id::Number(2),
            payload: ResponsePayload::Failure(ErrorPayload {
                code: -32601,
                message: "method not found".into(),
                data: None,
            }),
        };
        let batch = Ok(ResponsePacket::Batch(vec![success.clone(), success.clone()]));
        assert!(matches!(Outcome::of(&batch), Outcome::Success { results_size: 2 }));
        let batch = Ok(ResponsePacket::Batch(vec![success, failure]));
        assert_eq!(Outcome::of(&batch).label(), "rpc_-32601");
    }
}
