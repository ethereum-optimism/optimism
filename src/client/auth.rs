// From reth_rpc_layer
use alloy_rpc_types_engine::{Claims, JwtSecret};
use http::{HeaderValue, header::AUTHORIZATION};
use std::{
    iter::once,
    task::{Context, Poll},
    time::{Duration, SystemTime, UNIX_EPOCH},
};
use tower::{Layer, Service};
use tower_http::sensitive_headers::{SetSensitiveRequestHeaders, SetSensitiveRequestHeadersLayer};

pub type Auth<S> = AuthService<SetSensitiveRequestHeaders<S>>;

/// A layer that adds a new JWT token to every request using `AuthClientService`.
#[derive(Clone, Debug)]
pub struct AuthLayer {
    secret: JwtSecret,
}

impl AuthLayer {
    /// Create a new `AuthClientLayer` with the given `secret`.
    pub const fn new(secret: JwtSecret) -> Self {
        Self { secret }
    }
}

impl<S> Layer<S> for AuthLayer {
    type Service = AuthService<SetSensitiveRequestHeaders<S>>;

    fn layer(&self, inner: S) -> Self::Service {
        let inner = SetSensitiveRequestHeadersLayer::new(once(AUTHORIZATION)).layer(inner);
        AuthService::new(self.secret, inner)
    }
}

/// Automatically authenticates every client request with the given `secret`.
#[derive(Debug, Clone)]
pub struct AuthService<S> {
    secret: JwtSecret,
    inner: S,
}

impl<S> AuthService<S> {
    const fn new(secret: JwtSecret, inner: S) -> Self {
        Self { secret, inner }
    }
}

impl<S, B> Service<http::Request<B>> for AuthService<S>
where
    S: Service<http::Request<B>>,
    B: std::fmt::Debug,
{
    type Response = S::Response;
    type Error = S::Error;
    type Future = S::Future;

    fn poll_ready(&mut self, cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        self.inner.poll_ready(cx)
    }

    fn call(&mut self, mut request: http::Request<B>) -> Self::Future {
        request
            .headers_mut()
            .insert(AUTHORIZATION, secret_to_bearer_header(&self.secret));
        self.inner.call(request)
    }
}

/// Helper function to convert a secret into a Bearer auth header value with claims according to
/// <https://github.com/ethereum/execution-apis/blob/main/src/engine/authentication.md#jwt-claims>.
/// The token is valid for 60 seconds.
pub fn secret_to_bearer_header(secret: &JwtSecret) -> HeaderValue {
    format!(
        "Bearer {}",
        secret
            .encode(&Claims {
                iat: (SystemTime::now().duration_since(UNIX_EPOCH).unwrap()
                    + Duration::from_secs(60))
                .as_secs(),
                exp: None,
            })
            .unwrap()
    )
    .parse()
    .unwrap()
}
