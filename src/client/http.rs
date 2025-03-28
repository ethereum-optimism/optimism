use crate::server::PayloadSource;
use alloy_rpc_types_engine::JwtSecret;
use http::Uri;
use http_body_util::BodyExt;
use hyper_rustls::HttpsConnector;
use hyper_util::client::legacy::Client;
use hyper_util::client::legacy::connect::HttpConnector;
use hyper_util::rt::TokioExecutor;
use jsonrpsee::core::BoxError;
use jsonrpsee::http_client::HttpBody;
use opentelemetry::trace::SpanKind;
use tower::{Service as _, ServiceBuilder, ServiceExt};
use tower_http::decompression::{Decompression, DecompressionLayer};
use tracing::{debug, error, instrument};

use super::auth::{AuthClientLayer, AuthClientService};

#[derive(Clone, Debug)]
pub(crate) struct HttpClient {
    client: Decompression<AuthClientService<Client<HttpsConnector<HttpConnector>, HttpBody>>>,
    url: Uri,
    target: PayloadSource,
}

impl HttpClient {
    pub(crate) fn new(url: Uri, secret: JwtSecret, target: PayloadSource) -> Self {
        let connector = hyper_rustls::HttpsConnectorBuilder::new()
            .with_native_roots()
            .expect("no native root CA certificates found")
            .https_or_http()
            .enable_http1()
            .enable_http2()
            .build();

        let client = Client::builder(TokioExecutor::new()).build(connector);

        let client = ServiceBuilder::new()
            .layer(DecompressionLayer::new())
            .layer(AuthClientLayer::new(secret))
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
        fields(otel.kind = ?SpanKind::Client),
        err(Debug)
    )]
    pub async fn forward(
        &mut self,
        mut req: http::Request<HttpBody>,
        method: String,
    ) -> Result<http::Response<HttpBody>, BoxError> {
        debug!("forwarding {} to {}", method, self.target);
        *req.uri_mut() = self.url.clone();

        let res = self.client.ready().await?.call(req).await?;

        let (parts, body) = res.into_parts();
        let body_bytes = body.collect().await?.to_bytes().to_vec();

        if let Some(code) = parse_response_code(&body_bytes)? {
            error!(%code, "error in forwarded response");
        }

        Ok(http::Response::from_parts(
            parts,
            HttpBody::from(body_bytes),
        ))
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
