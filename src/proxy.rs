use hyper::Response;
use hyper_util::client::legacy::connect::HttpConnector;
use hyper_util::client::legacy::Client;
use hyper_util::rt::TokioExecutor;
use jsonrpsee::core::BoxError;
use jsonrpsee::http_client::{HttpBody, HttpRequest, HttpResponse};
use std::task::{Context, Poll};
use std::{future::Future, pin::Pin};
use tower::{Layer, Service};
use tracing::error;

#[derive(Debug, Clone)]
pub struct ProxyLayer {
    target_url: String,
}

impl ProxyLayer {
    pub fn new(target_url: String) -> Self {
        ProxyLayer { target_url }
    }
}

impl<S> Layer<S> for ProxyLayer {
    type Service = ProxyService<S>;

    fn layer(&self, inner: S) -> Self::Service {
        ProxyService {
            inner,
            client: Client::builder(TokioExecutor::new()).build_http(),
            target_url: self.target_url.clone(),
        }
    }
}

#[derive(Debug, Clone)]
pub struct ProxyService<S> {
    inner: S,
    client: Client<HttpConnector, HttpBody>,
    target_url: String,
}

impl<S> Service<HttpRequest<HttpBody>> for ProxyService<S>
where
    S: Service<HttpRequest<HttpBody>, Response = Response<HttpBody>>,
    S::Response: 'static,
    S::Error: Into<BoxError> + 'static,
    S::Future: Send + 'static,
{
    type Response = HttpResponse<hyper::body::Incoming>;
    type Error = BoxError;
    type Future =
        Pin<Box<dyn Future<Output = Result<Self::Response, Self::Error>> + Send + 'static>>;

    fn poll_ready(&mut self, cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        self.inner.poll_ready(cx).map_err(Into::into)
    }

    fn call(&mut self, mut req: HttpRequest<HttpBody>) -> Self::Future {
        let target_url = self.target_url.clone();
        let client = self.client.clone();
        let fut = async move {
            *req.uri_mut() = target_url.parse().unwrap();
            client.request(req).await.map_err(|e| {
                error!("Error proxying request: {}", e);
                Box::new(e) as Box<dyn std::error::Error + Send + Sync>
            })
        };
        Box::pin(fut)
    }
}
