use std::{
    pin::Pin,
    sync::Arc,
    task::{Context, Poll},
};

use futures::FutureExt as _;
use jsonrpsee::{
    core::BoxError,
    http_client::{HttpRequest, HttpResponse},
    server::HttpBody,
};
use parking_lot::Mutex;
use tower::{Layer, Service};
use tracing::info;

use crate::{Request, Response};

#[derive(Copy, Clone, Debug, Default)]
pub enum Health {
    /// Indicates that the builder is building blocks
    #[default]
    Healthy,
    /// Indicates that the l2 is building blocks, but the builder is not
    PartialContent,
    /// Indicates that blocks are not being built by either the l2 or the builder
    ///
    /// Service starts out unavailable until the first blocks are built
    ServiceUnavailable,
}

impl From<Health> for Response {
    fn from(health: Health) -> Self {
        match health {
            Health::Healthy => ok(),
            Health::PartialContent => partial_content(),
            Health::ServiceUnavailable => service_unavailable(),
        }
    }
}

#[derive(Debug, Default)]
pub struct Probes {
    health: Mutex<Health>,
}

impl Probes {
    pub fn set_health(&self, value: Health) {
        info!(target: "rollup_boost::probe", "Updating health probe to to {:?}", value);
        *self.health.lock() = value;
    }

    pub fn health(&self) -> Health {
        *self.health.lock()
    }
}

/// A [`Layer`] that adds probe endpoints to a service.
#[derive(Clone, Debug)]
pub struct ProbeLayer {
    probes: Arc<Probes>,
}

impl ProbeLayer {
    pub fn new() -> (Self, Arc<Probes>) {
        let probes = Arc::new(Probes::default());
        (
            Self {
                probes: probes.clone(),
            },
            probes,
        )
    }
}

impl<S> Layer<S> for ProbeLayer {
    type Service = ProbeService<S>;

    fn layer(&self, inner: S) -> Self::Service {
        ProbeService {
            inner,
            probes: self.probes.clone(),
        }
    }
}

#[derive(Clone, Debug)]
pub struct ProbeService<S> {
    inner: S,
    probes: Arc<Probes>,
}

impl<S> Service<Request> for ProbeService<S>
where
    S: Service<Request, Response = Response> + Send + Sync + Clone + 'static,
    S::Response: 'static,
    S::Error: Into<BoxError> + 'static,
    S::Future: Send + 'static,
{
    type Response = Response;
    type Error = BoxError;
    type Future =
        Pin<Box<dyn Future<Output = Result<Self::Response, Self::Error>> + Send + 'static>>;

    fn poll_ready(&mut self, cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        self.inner.poll_ready(cx).map_err(Into::into)
    }

    fn call(&mut self, request: HttpRequest) -> Self::Future {
        // See https://github.com/tower-rs/tower/blob/abb375d08cf0ba34c1fe76f66f1aba3dc4341013/tower-service/src/lib.rs#L276
        // for an explanation of this pattern
        let mut service = self.clone();
        service.inner = std::mem::replace(&mut self.inner, service.inner);

        async move {
            match request.uri().path() {
                // Return health status
                "/healthz" => Ok(service.probes.health().into()),
                // Service is responding, and therefor ready
                "/readyz" => Ok(ok()),
                // Service is responding, and therefor live
                "/livez" => Ok(ok()),
                // Forward the request to the inner service
                _ => service.inner.call(request).await.map_err(|e| e.into()),
            }
        }
        .boxed()
    }
}

fn ok() -> Response {
    HttpResponse::builder()
        .status(200)
        .body(HttpBody::from("OK"))
        .expect("Failed to create OK reponse")
}

fn partial_content() -> Response {
    HttpResponse::builder()
        .status(206)
        .body(HttpBody::from("Partial Content"))
        .expect("Failed to create partial content response")
}

fn service_unavailable() -> Response {
    HttpResponse::builder()
        .status(503)
        .body(HttpBody::from("Service Unavailable"))
        .expect("Failed to create service unavailable response")
}
