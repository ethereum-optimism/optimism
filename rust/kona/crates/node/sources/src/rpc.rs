//! The transport under the node's outgoing JSON-RPC clients.
//!
//! [`l1`] and [`engine`] build the node's two clients. Each bounds every request by [`TIMEOUT`]
//! and records it, under the client's label, in the [`REQUEST_DURATION`] histogram and a
//! completion log.

use alloy_json_rpc::{RequestPacket, ResponsePacket, RpcError};
use alloy_primitives::Bytes;
use alloy_provider::RootProvider;
use alloy_rpc_client::ClientBuilder;
use alloy_rpc_types_engine::JwtSecret;
use alloy_transport::{TransportError, TransportErrorKind, TransportFut, TransportResult};
use alloy_transport_http::{
    AuthLayer, Http, HyperClient, hyper_util,
    hyper_util::{client::legacy::Client, rt::TokioExecutor},
    reqwest,
};
use http_body_util::Full;
use op_alloy_network::Optimism;
use std::{
    borrow::Cow,
    task::{Context, Poll},
    time::{Duration, Instant},
};
use tokio::time::error::Elapsed;
use tower::{
    Layer, Service, ServiceBuilder,
    layer::util::{Identity, Stack},
    util::MapFutureLayer,
};
use url::Url;

/// Deadline for each request the node sends to an execution-layer RPC, on L1 or to the Engine API.
///
/// It bounds one request, never an operation made of many requests, such as the block traversal
/// of a reset, so a long but healthy operation is not cut off.
pub const TIMEOUT: Duration = Duration::from_secs(10);

/// Identifier for the histogram of outgoing JSON-RPC request durations, labeled by `client`,
/// `method`, `outcome` and `error`. Its count also gives the number of requests and errors.
pub const REQUEST_DURATION: &str = "kona_rpc_client_request_duration_seconds";

/// Describes [`REQUEST_DURATION`].
pub fn describe() {
    metrics::describe_histogram!(
        REQUEST_DURATION,
        metrics::Unit::Seconds,
        "Duration of outgoing JSON-RPC requests"
    );
}

/// Builds the node's L1 client for the HTTP endpoint at `url`, recorded as `l1`.
pub fn l1(url: Url) -> RootProvider {
    RootProvider::new(layered("l1").http(url))
}

/// Builds the node's L2 engine connection to the HTTP endpoint at `url`, signing each request
/// with `jwt`, recorded as `engine`. It carries L2 `eth_` reads, from derivation's providers and
/// the engine's own block lookups, as well as the Engine API calls.
pub fn engine(url: Url, jwt: JwtSecret) -> RootProvider<Optimism> {
    let hyper = Client::builder(TokioExecutor::new()).build_http::<Full<Bytes>>();
    let service = ServiceBuilder::new().layer(AuthLayer::new(jwt)).service(hyper);
    let transport = Http::with_client(HyperClient::with_service(service), url);
    RootProvider::new(layered("engine").transport(transport, false))
}

/// The layers under every client. [`Observe`] wraps the deadline, so a timeout is recorded as a
/// transport error; tower's [`Stack`] names the inner layer first.
type Layers = Stack<
    MapFutureLayer<fn(TransportFut<'static>) -> TransportFut<'static>>,
    Stack<Observe, Identity>,
>;

/// A client builder with [`Layers`] applied, recording as `client`.
fn layered(client: &'static str) -> ClientBuilder<Layers> {
    ClientBuilder::default().layer(Observe::new(client)).layer(MapFutureLayer::new(with_deadline))
}

/// Bounds one request on an alloy transport by [`TIMEOUT`].
///
/// This wraps the transport's future rather than the HTTP client below it, because the HTTP
/// client's future completes when the response headers arrive and the transport reads the body
/// after that.
///
/// A request that runs out of time fails with a [`TransportErrorKind::Custom`] holding a
/// [`tokio::time::error::Elapsed`], so outer layers can tell a timeout from other failures.
pub fn with_deadline(request: TransportFut<'static>) -> TransportFut<'static> {
    Box::pin(async move {
        tokio::time::timeout(TIMEOUT, request).await.map_err(TransportErrorKind::custom)?
    })
}

/// Records the duration and outcome of every request sent through a JSON-RPC transport, in
/// [`REQUEST_DURATION`] and a completion log under the `rpc_client` target, both labeled with
/// the client and the JSON-RPC method.
///
/// A failed request logs at warn. A successful Engine API call (`engine_*`) logs at info, and
/// any other successful request at debug: while the node catches up, derivation reads several
/// L1 and L2 blocks a second.
#[derive(Debug, Clone, Copy)]
pub struct Observe {
    /// The `client` label on the recorded metrics and logs.
    client: &'static str,
}

impl Observe {
    /// Creates a layer that records as `client`.
    pub const fn new(client: &'static str) -> Self {
        Self { client }
    }
}

impl<S> Layer<S> for Observe {
    type Service = Observed<S>;

    fn layer(&self, inner: S) -> Self::Service {
        Observed { inner, observe: *self }
    }
}

/// The transport service that an [`Observe`] layer wraps.
#[derive(Debug, Clone)]
pub struct Observed<S> {
    inner: S,
    observe: Observe,
}

impl<S> Service<RequestPacket> for Observed<S>
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
        let Observe { client } = self.observe;
        let (method, id, request_count) = match &request {
            RequestPacket::Single(request) => {
                (request.method().to_string(), Some(request.id().clone()), 1)
            }
            RequestPacket::Batch(requests) => ("batch".to_string(), None, requests.len()),
        };
        let start = Instant::now();
        let response = self.inner.call(request);
        Box::pin(async move {
            let response = response.await;
            let duration = start.elapsed();
            let (outcome, kind) = classify(&response);
            // A log level is fixed per call site, so each level gets its own.
            macro_rules! completed {
                ($level:ident, $($detail:tt)*) => {
                    tracing::$level!(
                        target: "rpc_client",
                        client,
                        method,
                        ?id,
                        request_count,
                        ?duration,
                        outcome,
                        $($detail)*
                        "JSON-RPC request completed"
                    )
                };
            }
            match &response {
                Ok(packet) if packet.is_error() => completed!(
                    warn,
                    code = packet.first_error_code(),
                    error = packet.first_error_message(),
                ),
                Ok(_) if method.starts_with("engine_") => completed!(info,),
                Ok(_) => completed!(debug,),
                Err(error) => completed!(warn, %error,),
            }
            metrics::histogram!(
                REQUEST_DURATION,
                "client" => client,
                "method" => method,
                "outcome" => outcome,
                "error" => kind,
            )
            .record(duration.as_secs_f64());
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
const fn rpc_error(code: i64) -> &'static str {
    match code {
        -32700 => "parse",
        -32600 => "invalid_request",
        -32601 => "method_not_found",
        -32602 => "invalid_params",
        -32603 => "internal",
        -32099..=-32000 => "server",
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
        // alloy's hyper transport and its JWT layer each wrap the error of the service below them
        // in a custom error of their own, so the error that names the failure may be nested.
        RpcError::Transport(TransportErrorKind::Custom(source)) => {
            match source.downcast_ref::<TransportError>() {
                Some(inner) => transport_error(inner),
                // The request deadline (`with_deadline`) fails with `Elapsed`.
                None if source.is::<Elapsed>() => Cow::Borrowed("timeout"),
                None if is_connect(source.as_ref()) => Cow::Borrowed("connect"),
                None => Cow::Borrowed("other"),
            }
        }
        RpcError::DeserError { .. } => Cow::Borrowed("decode"),
        _ => Cow::Borrowed("other"),
    }
}

/// Whether `source` is a failure to connect, from reqwest under [`l1`] or hyper under [`engine`].
fn is_connect(source: &(dyn std::error::Error + Send + Sync + 'static)) -> bool {
    source.downcast_ref::<reqwest::Error>().is_some_and(reqwest::Error::is_connect) ||
        source
            .downcast_ref::<hyper_util::client::legacy::Error>()
            .is_some_and(|error| error.is_connect())
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_json_rpc::{Id, Request, Response, ResponsePayload};
    use alloy_primitives::U64;
    use alloy_provider::Provider;
    use alloy_rpc_client::RpcClient;
    use metrics_exporter_prometheus::PrometheusBuilder;
    use rstest::rstest;
    use serde_json::value::RawValue;
    use std::{
        io,
        sync::{Arc, Mutex},
    };
    use tokio::{
        io::{AsyncReadExt, AsyncWriteExt},
        net::TcpListener,
        sync::mpsc,
        time,
    };
    use tracing::Level;

    #[derive(Clone, Default)]
    struct LogWriter(Arc<Mutex<Vec<u8>>>);

    impl io::Write for LogWriter {
        fn write(&mut self, bytes: &[u8]) -> io::Result<usize> {
            self.0.lock().unwrap().extend_from_slice(bytes);
            Ok(bytes.len())
        }

        fn flush(&mut self) -> io::Result<()> {
            Ok(())
        }
    }

    /// Captures every log at `level` and above while the returned guard lives.
    fn capture_logs(level: Level) -> (LogWriter, tracing::subscriber::DefaultGuard) {
        let logs = LogWriter::default();
        let writer = logs.clone();
        let subscriber = tracing_subscriber::fmt()
            .without_time()
            .with_ansi(false)
            .with_max_level(level)
            .with_writer(move || writer.clone())
            .finish();
        let guard = tracing::subscriber::set_default(subscriber);
        (logs, guard)
    }

    /// The captured `rpc_client` log lines.
    fn request_logs(logs: &LogWriter) -> Vec<String> {
        let output = String::from_utf8(logs.0.lock().unwrap().clone()).unwrap();
        output.lines().filter(|line| line.contains("rpc_client:")).map(str::to_string).collect()
    }

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

    /// Sends one request for `method` through `observe` over a transport that answers with
    /// `response`.
    fn send(observe: Observe, method: &'static str, response: TransportResult<ResponsePacket>) {
        let mut response = Some(response);
        let mut service = observe.layer(tower::service_fn(move |_| {
            let response = response.take().expect("the transport is called once");
            async move { response }
        }));
        let call = service.call(request(method));
        let _ = tokio::runtime::Builder::new_current_thread().build().unwrap().block_on(call);
    }

    /// A JSON-RPC client over the transport of the node's L1 client for `url`.
    fn l1_rpc(url: Url) -> RpcClient {
        RpcClient::new(l1(url).client().transport().clone(), false)
    }

    /// A JSON-RPC client over the transport of the node's engine connection for `url`.
    fn engine_rpc(url: Url) -> RpcClient {
        RpcClient::new(engine(url, JwtSecret::random()).client().transport().clone(), false)
    }

    /// Builds one of the node's clients against a URL.
    type Build = fn(Url) -> RpcClient;

    /// The node's clients, by label.
    const CLIENTS: [(&str, Build); 2] = [("l1", l1_rpc), ("engine", engine_rpc)];

    #[test]
    fn records_each_request_by_client_method_outcome_and_error() {
        let (logs, _guard) = capture_logs(Level::DEBUG);
        let observe = Observe::new("l1");
        let recorder = PrometheusBuilder::new().build_recorder();
        let metrics = recorder.handle();
        metrics::with_local_recorder(&recorder, || {
            send(
                observe,
                "eth_getBlockByHash",
                Ok(packet(r#"{"jsonrpc":"2.0","id":1,"result":null}"#)),
            );
            send(observe, "debug_getRawReceipts", Ok(rpc_error_packet(-32601)));
            send(
                observe,
                "eth_blockNumber",
                Err(TransportErrorKind::http_error(503, String::new())),
            );
        });

        let rendered = metrics.render();
        for (method, outcome, error) in [
            ("eth_getBlockByHash", "ok", "none"),
            ("debug_getRawReceipts", "rpc_error", "method_not_found"),
            ("eth_blockNumber", "transport_error", "http_503"),
        ] {
            let count = format!(
                r#"{REQUEST_DURATION}_count{{client="l1",method="{method}",outcome="{outcome}",error="{error}"}} 1"#
            );
            assert!(rendered.contains(&count), "missing {count} in:\n{rendered}");
        }
        let requests = request_logs(&logs);
        assert_eq!(requests.len(), 3, "{requests:?}");
        for (method, level) in [
            ("eth_getBlockByHash", "DEBUG"),
            ("debug_getRawReceipts", "WARN"),
            ("eth_blockNumber", "WARN"),
        ] {
            let log = requests.iter().find(|line| line.contains(method)).unwrap();
            assert!(log.contains(level), "{log}");
            assert!(log.contains("client=\"l1\""), "{log}");
        }
    }

    #[test]
    fn logs_each_request_with_its_outcome() {
        let (logs, _guard) = capture_logs(Level::DEBUG);
        let observe = Observe::new("engine");
        let recorder = PrometheusBuilder::new().build_recorder();
        let metrics = recorder.handle();
        metrics::with_local_recorder(&recorder, || {
            send(
                observe,
                "engine_newPayloadV3",
                Ok(ResponsePacket::Single(Response {
                    id: Id::Number(1),
                    payload: ResponsePayload::Success(
                        RawValue::from_string(
                            r#"{"status":"INVALID","validationError":"bad state"}"#.into(),
                        )
                        .unwrap(),
                    ),
                })),
            );
            send(
                observe,
                "engine_getPayloadV3",
                Err(TransportErrorKind::custom_str("unreachable")),
            );
            send(
                observe,
                "eth_blockNumber",
                Ok(packet(r#"{"jsonrpc":"2.0","id":1,"result":"0x1"}"#)),
            );
            send(observe, "engine_forkchoiceUpdatedV3", Ok(rpc_error_packet(-32602)));
        });

        let rendered = metrics.render();
        for method in [
            "engine_newPayloadV3",
            "engine_getPayloadV3",
            "engine_forkchoiceUpdatedV3",
            "eth_blockNumber",
        ] {
            let count = format!(r#"{REQUEST_DURATION}_count{{client="engine",method="{method}","#);
            assert!(rendered.contains(&count), "missing {count} in:\n{rendered}");
        }
        let output = String::from_utf8(logs.0.lock().unwrap().clone()).unwrap();
        assert!(!output.contains("bad state"), "method-specific payloads are opaque: {output}");
        assert!(!output.contains("payload_status="), "{output}");
        let requests = request_logs(&logs);
        assert_eq!(requests.len(), 4, "{output}");
        for (method, outcome, level, detail) in [
            ("engine_newPayloadV3", "ok", "INFO", "outcome=\"ok\""),
            ("engine_getPayloadV3", "transport_error", "WARN", "unreachable"),
            ("engine_forkchoiceUpdatedV3", "rpc_error", "WARN", "code=-32602"),
            // The engine connection also carries derivation's L2 reads, which log at debug.
            ("eth_blockNumber", "ok", "DEBUG", "outcome=\"ok\""),
        ] {
            let log = requests.iter().find(|line| line.contains(method)).unwrap();
            assert!(log.contains(&format!("outcome=\"{outcome}\"")), "{log}");
            assert!(log.contains("client=\"engine\""), "{log}");
            assert!(log.contains("duration="), "{log}");
            assert!(log.contains("id="), "{log}");
            assert!(log.contains(level), "{log}");
            assert!(log.contains(detail), "{log}");
        }
    }

    #[test]
    fn classifies_a_successful_response() {
        assert_eq!(
            classify(&Ok(packet(r#"{"jsonrpc":"2.0","id":1,"result":"0x1"}"#))),
            ("ok", Cow::Borrowed("none"))
        );
    }

    #[rstest]
    #[case::parse(-32700, "parse")]
    #[case::invalid_request(-32600, "invalid_request")]
    #[case::method_not_found(-32601, "method_not_found")]
    #[case::invalid_params(-32602, "invalid_params")]
    #[case::internal(-32603, "internal")]
    #[case::server_upper(-32000, "server")]
    #[case::server_inner(-32005, "server")]
    #[case::server_lower(-32099, "server")]
    #[case::execution_reverted(3, "execution_reverted")]
    #[case::below_server_range(-32100, "other")]
    #[case::unreserved(42, "other")]
    fn classifies_rpc_errors(#[case] code: i64, #[case] error: &str) {
        assert_eq!(classify(&Ok(rpc_error_packet(code))), ("rpc_error", Cow::Borrowed(error)));
    }

    #[rstest]
    #[case::http_status(TransportErrorKind::http_error(429, String::new()), "http_429")]
    #[case::http_status_with_retry_after(
        TransportErrorKind::http_error_with_retry_after(
            503,
            String::new(),
            Some(Duration::from_secs(1)),
        ),
        "http_503"
    )]
    #[case::decode(
        TransportError::deser_err(serde_json::from_str::<u8>("x").unwrap_err(), "x"),
        "decode"
    )]
    #[case::custom(TransportErrorKind::custom_str("unreachable"), "other")]
    #[case::backend_gone(TransportErrorKind::backend_gone(), "other")]
    // Errors nested the way alloy's hyper transport and JWT layer nest them.
    #[case::nested_http_status(
        TransportErrorKind::custom(TransportErrorKind::http_error(502, String::new())),
        "http_502"
    )]
    #[case::nested_custom(
        TransportErrorKind::custom(TransportErrorKind::custom(TransportErrorKind::custom_str(
            "x"
        ),)),
        "other"
    )]
    fn classifies_transport_errors(#[case] error: TransportError, #[case] expected: &str) {
        assert_eq!(classify(&Err(error)), ("transport_error", Cow::Borrowed(expected)));
    }

    #[tokio::test(start_paused = true)]
    async fn the_request_deadline_is_classified_as_a_timeout() {
        // A transport that never answers.
        let mut service = MapFutureLayer::new(with_deadline).layer(tower::service_fn(
            |_| -> TransportFut<'static> { Box::pin(std::future::pending()) },
        ));

        let response = service.call(request("eth_blockNumber")).await;

        assert_eq!(classify(&response), ("transport_error", Cow::Borrowed("timeout")));
    }

    #[rstest]
    #[tokio::test]
    async fn a_refused_connection_is_classified_as_a_connect_error(
        #[values(CLIENTS[0], CLIENTS[1])] client: (&str, Build),
    ) {
        let (_, build) = client;
        // Port 1 is reserved and nothing listens on it, so the connection is refused. This goes
        // through alloy's real transports, so it also checks that the `reqwest` and `hyper_util`
        // errors we downcast to are the ones alloy raises.
        let client = build("http://127.0.0.1:1".parse().unwrap());

        let response = client.transport().clone().call(request("eth_blockNumber")).await;

        assert_eq!(classify(&response), ("transport_error", Cow::Borrowed("connect")));
    }

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
        answered: &mut mpsc::UnboundedReceiver<()>,
    ) {
        let call = tokio::spawn(call);
        answered.recv().await.unwrap();
        time::pause();
        let paused_at = time::Instant::now();
        let failed = time::timeout(TIMEOUT * 2, call).await.expect("request has no deadline");
        assert!(failed.unwrap());
        // The deadline started before the clock was paused, and only the clock moved since. Timers
        // round up to the next millisecond.
        let waited = paused_at.elapsed();
        assert!(waited > TIMEOUT - Duration::from_secs(1));
        assert!(waited <= TIMEOUT + Duration::from_millis(1));
        time::resume();
    }

    #[rstest]
    #[case::no_response(b"")]
    #[case::stalled_body(
        b"HTTP/1.1 200 OK\r\ncontent-type: application/json\r\ncontent-length: 64\r\n\r\n"
    )]
    #[tokio::test]
    async fn stalled_requests_have_a_deadline_and_record_a_timeout(
        #[case] response: &'static [u8],
        #[values(CLIENTS[0], CLIENTS[1])] client: (&str, Build),
    ) {
        let (label, build) = client;
        let recorder = PrometheusBuilder::new().build_recorder();
        let metrics = recorder.handle();
        let _recorder = metrics::set_default_local_recorder(&recorder);
        let (answered_tx, mut answered) = mpsc::unbounded_channel();
        let client = build(stalled_endpoint(response, answered_tx).await);

        fails_at_deadline(
            async move { client.request::<_, U64>("eth_blockNumber", ()).await.is_err() },
            &mut answered,
        )
        .await;

        // The timeout is recorded, so the deadline sits inside the observing layer.
        let rendered = metrics.render();
        let timeout = format!(
            r#"{REQUEST_DURATION}_count{{client="{label}",method="eth_blockNumber",outcome="transport_error",error="timeout"}} 1"#
        );
        assert!(rendered.contains(&timeout), "missing {timeout} in:\n{rendered}");
    }
}
