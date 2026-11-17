//! Serves only `healthz` until the node's actors run.

use jsonrpsee::{
    server::middleware::rpc::{Batch, MethodResponse, Notification, Request, RpcServiceT},
    types::{ErrorCode, ErrorObject, Id},
};
use std::sync::{
    Arc,
    atomic::{AtomicBool, Ordering},
};
use tower::Layer;

/// The method that the gate always serves, also served at `GET /healthz`.
const HEALTHZ_METHOD: &str = "healthz";

/// Rejects every JSON-RPC method except `healthz` until [`Self::open`] is called.
///
/// The RPC server binds before the node waits on its execution layer and L1, so that health
/// checks pass while it waits. Until the actors run, nothing would answer the other methods, and a
/// command sent to an actor queue would run later even though its caller had timed out.
///
/// Place it inside the observability layer, which splits a batch into calls, so that each call in
/// a batch is gated on its own. A batch that reaches the gate whole is rejected until it opens.
#[derive(Clone, Debug, Default)]
pub(super) struct StartupGate(Arc<AtomicBool>);

impl StartupGate {
    /// Serves every method from now on.
    pub(super) fn open(&self) {
        self.0.store(true, Ordering::Release);
    }

    fn is_open(&self) -> bool {
        self.0.load(Ordering::Acquire)
    }

    fn starting_response(id: Id<'_>) -> MethodResponse {
        MethodResponse::error(
            id,
            ErrorObject::owned(ErrorCode::ServerIsBusy.code(), "The node is starting", None::<()>),
        )
    }
}

impl<S> Layer<S> for StartupGate {
    type Service = StartupGateService<S>;

    fn layer(&self, inner: S) -> Self::Service {
        StartupGateService { inner, gate: self.clone() }
    }
}

/// The service that [`StartupGate`] wraps around the RPC methods.
#[derive(Clone, Debug)]
pub(super) struct StartupGateService<S> {
    inner: S,
    gate: StartupGate,
}

impl<S> RpcServiceT for StartupGateService<S>
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
        let open = self.gate.is_open() || req.method_name() == HEALTHZ_METHOD;
        let inner = self.inner.clone();
        async move { if open { inner.call(req).await } else { StartupGate::starting_response(req.id) } }
    }

    fn notification<'a>(
        &self,
        notification: Notification<'a>,
    ) -> impl Future<Output = MethodResponse> + Send + 'a {
        // jsonrpsee does not execute notification handlers.
        self.inner.notification(notification)
    }

    fn batch<'a>(&self, batch: Batch<'a>) -> impl Future<Output = MethodResponse> + Send + 'a {
        let open = self.gate.is_open();
        let inner = self.inner.clone();
        async move {
            if open { inner.batch(batch).await } else { StartupGate::starting_response(Id::Null) }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::service::middleware::RpcObservabilityLayer;
    use jsonrpsee::{
        RpcModule,
        core::{
            client::{BatchResponse, ClientT},
            params::BatchRequestBuilder,
        },
        http_client::HttpClientBuilder,
        rpc_params,
        server::{
            Server,
            middleware::{http::ProxyGetRequestLayer, rpc::RpcServiceBuilder},
        },
        ws_client::WsClientBuilder,
    };
    use std::net::SocketAddr;
    use tokio::{
        io::{AsyncReadExt, AsyncWriteExt},
        net::TcpStream,
    };

    async fn get_healthz_status(addr: SocketAddr) -> String {
        let mut stream = TcpStream::connect(addr).await.unwrap();
        stream
            .write_all(b"GET /healthz HTTP/1.1\r\nhost: localhost\r\nconnection: close\r\n\r\n")
            .await
            .unwrap();
        let mut response = String::new();
        stream.read_to_string(&mut response).await.unwrap();
        response.lines().next().unwrap().to_owned()
    }

    fn successes(response: &BatchResponse<'_, u64>) -> Vec<bool> {
        response.iter().map(Result::is_ok).collect()
    }

    #[tokio::test]
    async fn serves_only_healthz_until_opened() {
        let mut module = RpcModule::new(());
        module.register_method(HEALTHZ_METHOD, |_, _, _| 1u64).unwrap();
        module.register_method("admin_startSequencer", |_, _, _| 2u64).unwrap();
        let gate = StartupGate::default();
        let server = Server::builder()
            .set_http_middleware(
                tower::ServiceBuilder::new()
                    .layer(ProxyGetRequestLayer::new([("/healthz", HEALTHZ_METHOD)]).unwrap()),
            )
            .set_rpc_middleware(
                RpcServiceBuilder::new()
                    .layer(RpcObservabilityLayer::new(module.method_names(), u32::MAX))
                    .layer(gate.clone()),
            )
            .build("127.0.0.1:0")
            .await
            .unwrap();
        let addr = server.local_addr().unwrap();
        let handle = server.start(module);
        let http = HttpClientBuilder::default().build(format!("http://{addr}")).unwrap();
        let ws = WsClientBuilder::default().build(format!("ws://{addr}")).await.unwrap();
        let batch = || {
            let mut batch = BatchRequestBuilder::new();
            batch.insert(HEALTHZ_METHOD, rpc_params![]).unwrap();
            batch.insert("admin_startSequencer", rpc_params![]).unwrap();
            batch
        };

        assert_eq!(get_healthz_status(addr).await, "HTTP/1.1 200 OK");
        assert_eq!(ws.request::<u64, _>(HEALTHZ_METHOD, rpc_params![]).await.unwrap(), 1);
        let starting = http.request::<u64, _>("admin_startSequencer", rpc_params![]).await;
        assert!(starting.unwrap_err().to_string().contains("The node is starting"));
        assert!(ws.request::<u64, _>("admin_startSequencer", rpc_params![]).await.is_err());
        assert_eq!(successes(&http.batch_request(batch()).await.unwrap()), [true, false]);

        gate.open();
        assert_eq!(http.request::<u64, _>("admin_startSequencer", rpc_params![]).await.unwrap(), 2);
        assert_eq!(ws.request::<u64, _>("admin_startSequencer", rpc_params![]).await.unwrap(), 2);
        assert_eq!(successes(&http.batch_request(batch()).await.unwrap()), [true, true]);

        handle.stop().unwrap();
        handle.stopped().await;
    }

    #[tokio::test]
    async fn rejects_a_whole_batch_until_opened() {
        #[derive(Clone)]
        struct Inner;

        impl RpcServiceT for Inner {
            type MethodResponse = MethodResponse;
            type BatchResponse = MethodResponse;
            type NotificationResponse = MethodResponse;

            fn call<'a>(&self, _: Request<'a>) -> impl Future<Output = MethodResponse> + Send + 'a {
                std::future::ready(MethodResponse::notification())
            }

            fn notification<'a>(
                &self,
                _: Notification<'a>,
            ) -> impl Future<Output = MethodResponse> + Send + 'a {
                std::future::ready(MethodResponse::notification())
            }

            fn batch<'a>(&self, _: Batch<'a>) -> impl Future<Output = MethodResponse> + Send + 'a {
                std::future::ready(MethodResponse::notification())
            }
        }

        let gate = StartupGate::default();
        let service = gate.layer(Inner);
        assert_eq!(
            service.batch(Batch::new()).await.as_error_code(),
            Some(ErrorCode::ServerIsBusy.code())
        );
        gate.open();
        assert!(service.batch(Batch::new()).await.is_notification());
    }
}
