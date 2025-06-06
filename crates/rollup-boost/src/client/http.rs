use std::time::Duration;

use crate::client::auth::AuthLayer;
use crate::payload::PayloadSource;
use alloy_primitives::bytes::Bytes;
use alloy_rpc_types_engine::JwtSecret;
use http::Uri;
use http_body_util::{BodyExt, Full};
use hyper::body::Body;
use hyper_rustls::HttpsConnector;
use hyper_util::client::legacy::Client;
use hyper_util::client::legacy::connect::HttpConnector;
use hyper_util::rt::TokioExecutor;
use jsonrpsee::core::BoxError;
use jsonrpsee::server::HttpBody;
use opentelemetry::trace::SpanKind;
use tower::{
    Service as _, ServiceBuilder, ServiceExt,
    timeout::{Timeout, TimeoutLayer},
};
use tower_http::decompression::{Decompression, DecompressionLayer};
use tracing::{debug, error, instrument};

use super::auth::Auth;

pub type HttpClientService =
    Timeout<Decompression<Auth<Client<HttpsConnector<HttpConnector>, HttpBody>>>>;

#[derive(Clone, Debug)]
pub struct HttpClient {
    client: HttpClientService,
    url: Uri,
    target: PayloadSource,
}

impl HttpClient {
    pub fn new(url: Uri, secret: JwtSecret, target: PayloadSource, timeout: u64) -> Self {
        let connector = hyper_rustls::HttpsConnectorBuilder::new()
            .with_native_roots()
            .expect("no native root CA certificates found")
            .https_or_http()
            .enable_http1()
            .enable_http2()
            .build();

        let client = Client::builder(TokioExecutor::new()).build(connector);

        let client = ServiceBuilder::new()
            .layer(TimeoutLayer::new(Duration::from_millis(timeout)))
            .layer(DecompressionLayer::new())
            .layer(AuthLayer::new(secret))
            .service(client);

        Self {
            client,
            url,
            target,
        }
    }

    /// Forwards an HTTP request to the `authrpc`, attaching the provided JWT authorization.
    #[instrument(
        skip(self, req),
        fields(
            otel.kind = ?SpanKind::Client,
            url = %self.url,
            method,
            code,
        ),
        err(Debug)
    )]
    pub async fn forward<B>(
        &mut self,
        mut req: http::Request<B>,
        method: String,
    ) -> Result<http::Response<Full<Bytes>>, BoxError>
    where
        B: Body<Data = Bytes, Error: Into<Box<dyn std::error::Error + Send + Sync>>>
            + Send
            + 'static,
    {
        debug!("forwarding {} to {}", method, self.target);
        tracing::Span::current().record("method", method);
        *req.uri_mut() = self.url.clone();

        let req = req.map(HttpBody::new);

        let res = self.client.ready().await?.call(req).await?;

        let (parts, body) = res.into_parts();
        let body_bytes = body.collect().await?.to_bytes();

        if let Some(code) = parse_response_code(&body_bytes)? {
            error!(%code, "error in forwarded response");
            tracing::Span::current().record("code", code);
        }

        Ok(http::Response::from_parts(parts, Full::from(body_bytes)))
    }
}

fn parse_response_code(body_bytes: &[u8]) -> eyre::Result<Option<i32>> {
    #[derive(serde::Deserialize, Debug)]
    struct RpcResponse {
        error: Option<JsonRpcError>,
    }

    #[derive(serde::Deserialize, Debug)]
    struct JsonRpcError {
        code: i32,
    }

    let res = serde_json::from_slice::<RpcResponse>(body_bytes)?;

    Ok(res.error.map(|e| e.code))
}
