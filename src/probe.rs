use std::{
    pin::Pin,
    sync::Arc,
    task::{Context, Poll},
};

use futures::FutureExt as _;
use jsonrpsee::{
    core::BoxError,
    http_client::{HttpBody, HttpRequest, HttpResponse},
};
use parking_lot::Mutex;
use tower::{Layer, Service};

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

impl From<Health> for HttpResponse<HttpBody> {
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
    pub(crate) fn new() -> (Self, Arc<Probes>) {
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

impl<S> Service<HttpRequest<HttpBody>> for ProbeService<S>
where
    S: Service<HttpRequest<HttpBody>, Response = HttpResponse> + Send + Sync + Clone + 'static,
    S::Response: 'static,
    S::Error: Into<BoxError> + 'static,
    S::Future: Send + 'static,
{
    type Response = S::Response;
    type Error = BoxError;
    type Future =
        Pin<Box<dyn Future<Output = Result<Self::Response, Self::Error>> + Send + 'static>>;

    fn poll_ready(&mut self, cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        self.inner.poll_ready(cx).map_err(Into::into)
    }

    fn call(&mut self, request: HttpRequest<HttpBody>) -> Self::Future {
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

fn ok() -> HttpResponse<HttpBody> {
    HttpResponse::builder()
        .status(200)
        .body(HttpBody::from("OK"))
        .unwrap()
}

fn partial_content() -> HttpResponse<HttpBody> {
    HttpResponse::builder()
        .status(206)
        .body(HttpBody::from("Partial Content"))
        .unwrap()
}

fn service_unavailable() -> HttpResponse<HttpBody> {
    HttpResponse::builder()
        .status(503)
        .body(HttpBody::from("Service Unavailable"))
        .unwrap()
}
