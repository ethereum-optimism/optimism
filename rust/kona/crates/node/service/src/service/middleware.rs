//! Metrics for parsed JSON-RPC calls and notifications, including batch entries.

use crate::Metrics;
use jsonrpsee::{
    core::server::BatchResponseBuilder,
    server::middleware::rpc::{
        Batch, BatchEntry, MethodResponse, Notification, Request, RpcServiceT,
    },
};
use std::{collections::HashSet, sync::Arc, time::Instant};
use tower::Layer;

/// Records all registered methods, grouping unregistered names under `unknown`.
#[derive(Clone, Debug)]
pub(super) struct RpcMetricsLayer {
    methods: Arc<HashSet<&'static str>>,
    max_response_body_size: u32,
}

impl RpcMetricsLayer {
    pub(super) fn new(
        methods: impl Iterator<Item = &'static str>,
        max_response_body_size: u32,
    ) -> Self {
        Self { methods: Arc::new(methods.collect()), max_response_body_size }
    }
}

impl<S> Layer<S> for RpcMetricsLayer {
    type Service = RpcMetricsService<S>;

    fn layer(&self, inner: S) -> Self::Service {
        RpcMetricsService { inner, metrics: self.clone() }
    }
}

#[derive(Clone, Debug)]
pub(super) struct RpcMetricsService<S> {
    inner: S,
    metrics: RpcMetricsLayer,
}

impl<S> RpcMetricsService<S> {
    fn method(&self, name: &str) -> &'static str {
        self.metrics.methods.get(name).copied().unwrap_or("unknown")
    }
}

impl<S> RpcServiceT for RpcMetricsService<S>
where
    S: RpcServiceT<
            MethodResponse = MethodResponse,
            BatchResponse = MethodResponse,
            NotificationResponse = MethodResponse,
        > + Clone
        + Send
        + Sync
        + 'static,
{
    type MethodResponse = MethodResponse;
    type BatchResponse = MethodResponse;
    type NotificationResponse = MethodResponse;

    fn call<'a>(&self, req: Request<'a>) -> impl Future<Output = MethodResponse> + Send + 'a {
        let method = self.method(req.method_name());
        let inner = self.inner.clone();
        async move {
            metrics::counter!(Metrics::RPC_REQUESTS, "method" => method, "kind" => "call")
                .increment(1);
            let start = Instant::now();
            let response = inner.call(req).await;
            let result = response.as_error_code().map_or("success", |code| {
                metrics::counter!(Metrics::RPC_ERRORS, "method" => method, "code" => code.to_string())
                    .increment(1);
                "error"
            });
            metrics::histogram!(Metrics::RPC_REQUEST_DURATION, "method" => method, "result" => result)
                .record(start.elapsed().as_secs_f64());
            response
        }
    }

    fn notification<'a>(
        &self,
        notification: Notification<'a>,
    ) -> impl Future<Output = MethodResponse> + Send + 'a {
        let method = self.method(&notification.method);
        let inner = self.inner.clone();
        async move {
            // jsonrpsee does not execute notification handlers. Count receipt separately from
            // calls, without reporting a successful method execution or a duration.
            metrics::counter!(Metrics::RPC_REQUESTS, "method" => method, "kind" => "notification")
                .increment(1);
            inner.notification(notification).await
        }
    }

    fn batch<'a>(&self, batch: Batch<'a>) -> impl Future<Output = MethodResponse> + Send + 'a {
        let service = self.clone();
        async move {
            // Forwarding to inner.batch would bypass call/notification middleware. Preserve
            // jsonrpsee's ordered dispatch, response limit, and notification-only response.
            let mut responses = BatchResponseBuilder::new_with_limit(
                service.metrics.max_response_body_size as usize,
            );
            let mut got_notification = false;
            for entry in batch {
                let response = match entry {
                    Ok(BatchEntry::Call(req)) => service.call(req).await,
                    Ok(BatchEntry::Notification(notification)) => {
                        got_notification = true;
                        service.notification(notification).await;
                        continue;
                    }
                    Err(err) => {
                        let (err, id) = err.into_parts();
                        MethodResponse::error(id, err)
                    }
                };
                if let Err(err) = responses.append(response) {
                    return err;
                }
            }
            if responses.is_empty() && got_notification {
                MethodResponse::notification()
            } else {
                MethodResponse::from_batch(responses.finish())
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use jsonrpsee::{
        RpcModule,
        core::{
            RpcResult, SubscriptionError,
            client::{ClientT, SubscriptionClientT, SubscriptionKind},
            params::BatchRequestBuilder,
            server::ResponsePayload,
        },
        http_client::HttpClientBuilder,
        rpc_params,
        server::{Server, middleware::rpc::RpcServiceBuilder},
        types::{ErrorCode, Id},
        ws_client::WsClientBuilder,
    };
    use metrics_exporter_prometheus::PrometheusBuilder;
    use std::sync::atomic::{AtomicUsize, Ordering};

    fn assert_sample(rendered: &str, name: &str, labels: &[&str], expected: u64) {
        let prefix = format!("{name}{{");
        let line = rendered
            .lines()
            .find(|line| {
                line.starts_with(&prefix) && labels.iter().all(|label| line.contains(label))
            })
            .unwrap_or_else(|| panic!("Missing {name} with {labels:?}:\n{rendered}"));
        assert_eq!(line.rsplit_once(' ').unwrap().1, expected.to_string(), "{line}");
    }

    #[test]
    fn records_http_batches_websocket_calls_and_subscription_setup() {
        let recorder = PrometheusBuilder::new().build_recorder();
        let metrics = recorder.handle();
        let runtime = tokio::runtime::Builder::new_current_thread().enable_all().build().unwrap();
        metrics::with_local_recorder(&recorder, || {
            runtime.block_on(async {
                let mut module = RpcModule::new(());
                module
                    .register_async_method("optimism_answer", |_, _, _| async {
                        tokio::task::yield_now().await;
                        42_u64
                    })
                    .unwrap();
                module
                    .register_method("admin_fail", |_, _, _| {
                        RpcResult::<u64>::Err(ErrorCode::InternalError.into())
                    })
                    .unwrap();
                module
                    .register_method("optimism_echo", |params, _, _| params.one::<u64>())
                    .unwrap();
                module
                    .register_subscription::<Result<(), SubscriptionError>, _, _>(
                        "ws_subscribe",
                        "ws_notice",
                        "ws_unsubscribe",
                        |_, pending, _, _| async {
                            let sink = pending.accept().await?;
                            sink.closed().await;
                            Ok(())
                        },
                    )
                    .unwrap();

                let rpc_middleware = RpcServiceBuilder::new().layer(RpcMetricsLayer::new(
                    module.method_names(),
                    jsonrpsee::core::TEN_MB_SIZE_BYTES,
                ));
                let server = Server::builder()
                    .set_rpc_middleware(rpc_middleware)
                    .build("127.0.0.1:0")
                    .await
                    .unwrap();
                let addr = server.local_addr().unwrap();
                let handle = server.start(module);
                let http = HttpClientBuilder::default().build(format!("http://{addr}")).unwrap();
                assert_eq!(
                    http.request::<u64, _>("optimism_answer", rpc_params![]).await.unwrap(),
                    42
                );
                assert!(http.request::<u64, _>("admin_fail", rpc_params![]).await.is_err());
                assert!(http.request::<u64, _>("optimism_echo", rpc_params!["bad"]).await.is_err());
                for method in ["missing_one", "missing_two"] {
                    assert!(http.request::<u64, _>(method, rpc_params![]).await.is_err());
                }
                http.notification("optimism_answer", rpc_params![]).await.unwrap();

                let mut batch = BatchRequestBuilder::new();
                batch.insert("optimism_answer", rpc_params![]).unwrap();
                batch.insert("admin_fail", rpc_params![]).unwrap();
                batch.insert("missing_three", rpc_params![]).unwrap();
                let response = http.batch_request::<u64>(batch).await.unwrap();
                assert_eq!(response.num_successful_calls(), 1);
                assert_eq!(response.num_failed_calls(), 2);

                let ws = WsClientBuilder::default().build(format!("ws://{addr}")).await.unwrap();
                assert_eq!(
                    ws.request::<u64, _>("optimism_answer", rpc_params![]).await.unwrap(),
                    42
                );
                let subscription = ws
                    .subscribe::<u64, _>("ws_subscribe", rpc_params![], "ws_unsubscribe")
                    .await
                    .unwrap();
                let id = match subscription.kind() {
                    SubscriptionKind::Subscription(id) => id.clone(),
                    _ => panic!("Expected a subscription ID"),
                };
                assert!(ws.request::<bool, _>("ws_unsubscribe", rpc_params![id]).await.unwrap());
                drop(ws);
                handle.stop().unwrap();
                handle.stopped().await;
                drop(subscription);
            });
        });

        let rendered = metrics.render();
        for (method, count) in [
            ("optimism_answer", 3),
            ("admin_fail", 2),
            ("optimism_echo", 1),
            ("unknown", 3),
            ("ws_subscribe", 1),
            ("ws_unsubscribe", 1),
        ] {
            let method_label = format!("method=\"{method}\"");
            assert_sample(
                &rendered,
                Metrics::RPC_REQUESTS,
                &[&method_label, "kind=\"call\""],
                count,
            );
            let result = if matches!(method, "admin_fail" | "optimism_echo" | "unknown") {
                "result=\"error\""
            } else {
                "result=\"success\""
            };
            assert_sample(
                &rendered,
                &format!("{}_count", Metrics::RPC_REQUEST_DURATION),
                &[&method_label, result],
                count,
            );
        }
        assert_sample(
            &rendered,
            Metrics::RPC_REQUESTS,
            &["method=\"optimism_answer\"", "kind=\"notification\""],
            1,
        );
        for (method, code, count) in
            [("admin_fail", -32603, 2), ("optimism_echo", -32602, 1), ("unknown", -32601, 3)]
        {
            assert_sample(
                &rendered,
                Metrics::RPC_ERRORS,
                &[&format!("method=\"{method}\""), &format!("code=\"{code}\"")],
                count,
            );
        }
        assert!(!rendered.contains("missing_one"));
        assert!(!rendered.contains("missing_two"));
        assert!(!rendered.contains("missing_three"));
    }

    #[derive(Clone)]
    struct TestService(Arc<AtomicUsize>);

    impl RpcServiceT for TestService {
        type MethodResponse = MethodResponse;
        type BatchResponse = MethodResponse;
        type NotificationResponse = MethodResponse;

        fn call<'a>(&self, req: Request<'a>) -> impl Future<Output = MethodResponse> + Send + 'a {
            self.0.fetch_add(1, Ordering::Relaxed);
            async move {
                MethodResponse::response(req.id, ResponsePayload::success(42), u32::MAX as usize)
            }
        }

        fn notification<'a>(
            &self,
            _: Notification<'a>,
        ) -> impl Future<Output = MethodResponse> + Send + 'a {
            std::future::ready(MethodResponse::notification())
        }

        fn batch<'a>(&self, _: Batch<'a>) -> impl Future<Output = MethodResponse> + Send + 'a {
            // A delegated batch would fail the response assertions below.
            std::future::ready(MethodResponse::error(Id::Null, ErrorCode::InternalError))
        }
    }

    #[tokio::test]
    async fn preserves_mixed_batches_notifications_and_response_limits() {
        use jsonrpsee::server::middleware::rpc::BatchEntryErr;

        let calls = Arc::new(AtomicUsize::new(0));
        let service =
            RpcMetricsLayer::new(["answer"].into_iter(), 1024).layer(TestService(calls.clone()));
        let call = || Request::owned("answer".into(), None, Id::Number(1));
        let notification = || Notification::new("answer".into(), None);
        let response = service
            .batch(Batch::from(vec![
                Ok(BatchEntry::Call(call())),
                Ok(BatchEntry::Notification(notification())),
                Err(BatchEntryErr::new(Id::Number(2), ErrorCode::InvalidRequest.into())),
            ]))
            .await;
        let json: serde_json::Value = serde_json::from_str(response.as_json().get()).unwrap();
        assert_eq!(json[0]["result"], 42);
        assert_eq!(json[1]["error"]["code"], -32600);
        assert_eq!(json.as_array().unwrap().len(), 2);
        assert!(
            service
                .batch(Batch::from(vec![Ok(BatchEntry::Notification(notification()))]))
                .await
                .is_notification()
        );
        let empty = service.batch(Batch::new()).await;
        let json: serde_json::Value = serde_json::from_str(empty.as_json().get()).unwrap();
        assert_eq!(json["error"]["code"], -32600);

        let limited =
            RpcMetricsLayer::new(["answer"].into_iter(), 1).layer(TestService(calls.clone()));
        let response = limited
            .batch(Batch::from(vec![Ok(BatchEntry::Call(call())), Ok(BatchEntry::Call(call()))]))
            .await;
        assert!(response.is_error());
        assert_eq!(
            calls.load(Ordering::Relaxed),
            2,
            "Stop dispatching once the response exceeds its limit"
        );
    }
}
