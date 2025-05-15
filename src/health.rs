use std::{
    sync::Arc,
    time::{Duration, SystemTime, UNIX_EPOCH},
};

use alloy_rpc_types_eth::BlockNumberOrTag;
use tokio::{
    task::JoinHandle,
    time::{Instant, sleep_until},
};
use tracing::warn;

use crate::{Health, Probes, RpcClient};

pub struct HealthHandle {
    pub probes: Arc<Probes>,
    pub builder_client: Arc<RpcClient>,
    pub health_check_interval: u64,
    pub max_unsafe_interval: u64,
}

impl HealthHandle {
    /// Periodically checks that the latest unsafe block timestamp is not older than the
    /// the current time minus the max_unsafe_interval.
    pub fn spawn(self) -> JoinHandle<()> {
        tokio::spawn(async move {
            loop {
                let latest_unsafe = match self
                    .builder_client
                    .get_block_by_number(BlockNumberOrTag::Latest, false)
                    .await
                {
                    Ok(block) => block,
                    Err(e) => {
                        warn!(target: "rollup_boost::health", "Failed to get unsafe block from builder client: {} - updating health status", e);
                        self.probes.set_health(Health::PartialContent);
                        sleep_until(
                            Instant::now() + Duration::from_secs(self.health_check_interval),
                        )
                        .await;
                        continue;
                    }
                };

                let now = SystemTime::now()
                    .duration_since(UNIX_EPOCH)
                    .expect("Time went backwards")
                    .as_secs();

                if now - latest_unsafe.header.timestamp > self.max_unsafe_interval {
                    warn!(target: "rollup_boost::health", "Unsafe block timestamp is too old ({} seconds - updating health status)", now - latest_unsafe.header.timestamp);
                    self.probes.set_health(Health::PartialContent);
                } else {
                    self.probes.set_health(Health::Healthy);
                }

                sleep_until(Instant::now() + Duration::from_secs(self.health_check_interval)).await;
            }
        })
    }
}

#[cfg(test)]
mod tests {
    use std::net::SocketAddr;

    use alloy_consensus::Header;
    use alloy_rpc_types_eth::{Block, Header as EthHeader, Transaction};

    use http::Uri;
    use http_body_util::BodyExt;
    use hyper::service::service_fn;
    use hyper_util::rt::TokioIo;
    use reth_rpc_layer::JwtSecret;
    use serde_json::json;
    use tokio::net::TcpListener;

    use super::*;
    use crate::{PayloadSource, Probes};

    pub struct MockHttpServer {
        addr: SocketAddr,
        join_handle: JoinHandle<()>,
    }

    impl Drop for MockHttpServer {
        fn drop(&mut self) {
            self.join_handle.abort();
        }
    }

    impl MockHttpServer {
        async fn serve<S>(
            f: fn(hyper::Request<hyper::body::Incoming>, timestamp: u64) -> S,
            timestamp: u64,
        ) -> eyre::Result<Self>
        where
            S: Future<Output = Result<hyper::Response<String>, hyper::Error>>
                + Send
                + Sync
                + 'static,
        {
            {
                let listener = TcpListener::bind("0.0.0.0:0").await?;
                let addr = listener.local_addr()?;

                let handle = tokio::spawn(async move {
                    loop {
                        match listener.accept().await {
                            Ok((stream, _)) => {
                                let io = TokioIo::new(stream);
                                tokio::spawn(async move {
                                    if let Err(err) = hyper::server::conn::http1::Builder::new()
                                        .serve_connection(
                                            io,
                                            service_fn(move |req| f(req, timestamp)),
                                        )
                                        .await
                                    {
                                        eprintln!("Error serving connection: {}", err);
                                    }
                                });
                            }
                            Err(e) => eprintln!("Error accepting connection: {}", e),
                        }
                    }
                });

                Ok(Self {
                    addr,
                    join_handle: handle,
                })
            }
        }
    }

    async fn handler(
        req: hyper::Request<hyper::body::Incoming>,
        block_timstamp: u64,
    ) -> Result<hyper::Response<String>, hyper::Error> {
        let body_bytes = match req.into_body().collect().await {
            Ok(buf) => buf.to_bytes(),
            Err(_) => {
                let error_response = json!({
                    "jsonrpc": "2.0",
                    "error": { "code": -32700, "message": "Failed to read request body" },
                    "id": null
                });
                return Ok(hyper::Response::new(error_response.to_string()));
            }
        };

        let request_body: serde_json::Value = match serde_json::from_slice(&body_bytes) {
            Ok(json) => json,
            Err(_) => {
                let error_response = json!({
                    "jsonrpc": "2.0",
                    "error": { "code": -32700, "message": "Invalid JSON format" },
                    "id": null
                });
                return Ok(hyper::Response::new(error_response.to_string()));
            }
        };

        let method = request_body["method"].as_str().unwrap_or_default();

        let mock_block = Block::<Transaction> {
            header: EthHeader {
                inner: Header {
                    timestamp: block_timstamp,
                    ..Default::default()
                },
                ..Default::default()
            },
            ..Default::default()
        };

        let response = match method {
            "eth_getBlockByNumber" => json!({
                "jsonrpc": "2.0",
                "result": mock_block,
                "id": request_body["id"]
            }),
            _ => {
                let error_response = json!({
                    "jsonrpc": "2.0",
                    "error": { "code": -32601, "message": "Method not found" },
                    "id": request_body["id"]
                });
                return Ok(hyper::Response::new(error_response.to_string()));
            }
        };

        Ok(hyper::Response::new(response.to_string()))
    }

    #[tokio::test]
    async fn test_health_check_healthy() -> eyre::Result<()> {
        tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
        let probes = Arc::new(Probes::default());
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("Time went backwards")
            .as_secs();

        let builder = MockHttpServer::serve(handler, now).await.unwrap();
        let builder_client = Arc::new(RpcClient::new(
            format!("http://{}", builder.addr).parse::<Uri>()?,
            JwtSecret::random(),
            100,
            PayloadSource::Builder,
        )?);

        let health_handle = HealthHandle {
            probes: probes.clone(),
            builder_client: builder_client.clone(),
            health_check_interval: 60,
            max_unsafe_interval: 5,
        };

        health_handle.spawn();
        tokio::time::sleep(Duration::from_secs(2)).await;
        assert!(matches!(probes.health(), Health::Healthy));
        Ok(())
    }

    #[tokio::test]
    async fn test_health_check_exceeds_max_unsafe_interval() -> eyre::Result<()> {
        tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
        let probes = Arc::new(Probes::default());
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("Time went backwards")
            .as_secs();
        let builder = MockHttpServer::serve(handler, now - 10).await.unwrap();

        let builder_client = Arc::new(RpcClient::new(
            format!("http://{}", builder.addr).parse::<Uri>()?,
            JwtSecret::random(),
            100,
            PayloadSource::Builder,
        )?);

        let health_handle = HealthHandle {
            probes: probes.clone(),
            builder_client: builder_client.clone(),
            health_check_interval: 60,
            max_unsafe_interval: 5,
        };

        health_handle.spawn();
        tokio::time::sleep(Duration::from_secs(2)).await;
        assert!(matches!(probes.health(), Health::PartialContent));
        Ok(())
    }

    #[tokio::test]
    async fn test_health_check_service_unavailable() -> eyre::Result<()> {
        tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
        let probes = Arc::new(Probes::default());
        let builder_client = Arc::new(RpcClient::new(
            "http://127.0.0.1:6000".parse::<Uri>()?,
            JwtSecret::random(),
            100,
            PayloadSource::Builder,
        )?);

        let health_handle = HealthHandle {
            probes: probes.clone(),
            builder_client: builder_client.clone(),
            health_check_interval: 60,
            max_unsafe_interval: 5,
        };

        health_handle.spawn();
        tokio::time::sleep(Duration::from_secs(2)).await;
        assert!(matches!(probes.health(), Health::PartialContent));
        Ok(())
    }
}
