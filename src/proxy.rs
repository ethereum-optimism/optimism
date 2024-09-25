use http::Uri;
use hyper_util::client::legacy::connect::HttpConnector;
use hyper_util::client::legacy::Client;
use hyper_util::rt::TokioExecutor;
use jsonrpsee::core::{http_helpers, BoxError};
use jsonrpsee::http_client::{HttpBody, HttpRequest, HttpResponse};
use std::task::{Context, Poll};
use std::{future::Future, pin::Pin};
use tower::{Layer, Service};
use tracing::debug;

#[derive(Debug, Clone)]
pub struct ProxyLayer {
    target_url: Uri,
}

impl ProxyLayer {
    pub fn new(target_url: Uri) -> Self {
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
    target_url: Uri,
}

impl<S> Service<HttpRequest<HttpBody>> for ProxyService<S>
where
    S: Service<HttpRequest<HttpBody>, Response = HttpResponse> + Send + Clone + 'static,
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

    fn call(&mut self, req: HttpRequest<HttpBody>) -> Self::Future {
        let target_url = self.target_url.clone();
        let client = self.client.clone();
        let mut inner = self.inner.clone();

        #[derive(serde::Deserialize, Debug)]
        struct RpcRequest<'a> {
            #[serde(borrow)]
            method: &'a str,
        }

        let fut = async move {
            let (parts, body) = req.into_parts();

            let (body, _is_single) =
                http_helpers::read_body(&parts.headers, body, u32::MAX).await?;
            // Deserialize the bytes to find the method
            let method: RpcRequest = serde_json::from_slice(&body)?;

            // Create a new body from the bytes
            let new_body = HttpBody::from(body.clone());

            // Reconstruct the request
            let mut req = HttpRequest::from_parts(parts, new_body);

            debug!(
                message = "received json rpc request for",
                method = method.method
            );
            if method.method.starts_with("engine_") {
                // let rpc server handle engine rpc requests
                let res = inner.call(req).await.map_err(|e| e.into())?;
                Ok(res)
            } else {
                // Modify the URI
                *req.uri_mut() = target_url;

                // Forward the request
                let res = client
                    .request(req)
                    .await
                    .map(|res| res.map(HttpBody::new))?;
                Ok(res)
            }
        };
        Box::pin(fut)
    }
}
