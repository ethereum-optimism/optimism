use std::{
    sync::Arc,
    time::{Duration, SystemTime, UNIX_EPOCH},
};

use alloy_rpc_types_eth::BlockNumberOrTag;
use parking_lot::Mutex;
use tokio::{
    task::JoinHandle,
    time::{Instant, sleep_until},
};
use tracing::warn;

use crate::{EngineApiExt, ExecutionMode, Health, Probes};

pub struct HealthHandle {
    pub probes: Arc<Probes>,
    pub execution_mode: Arc<Mutex<ExecutionMode>>,
    pub l2_client: Arc<dyn EngineApiExt>,
    pub builder_client: Arc<dyn EngineApiExt>,
    pub health_check_interval: Duration,
    pub max_unsafe_interval: u64,
}

impl HealthHandle {
    /// Creates a new instance of [`HealthHandle`].
    pub fn new(
        probes: Arc<Probes>,
        execution_mode: Arc<Mutex<ExecutionMode>>,
        l2_client: Arc<dyn EngineApiExt>,
        builder_client: Arc<dyn EngineApiExt>,
        health_check_interval: Duration,
        max_unsafe_interval: u64,
    ) -> Self {
        Self {
            probes,
            execution_mode,
            l2_client,
            builder_client,
            health_check_interval,
            max_unsafe_interval,
        }
    }

    /// Periodically checks that the latest unsafe block timestamp is not older than the
    /// the current time minus the max_unsafe_interval.
    pub fn spawn(self) -> JoinHandle<()> {
        tokio::spawn(async move {
            let mut timestamp = MonotonicTimestamp::new();

            loop {
                let t = timestamp.tick();

                // Check L2 client health. If its unhealthy, set the health status to ServiceUnavailable
                // If in disabled or dry run execution mode, set the health status to Healthy if the l2 client is healthy
                match self
                    .l2_client
                    .get_block_by_number(BlockNumberOrTag::Latest, false)
                    .await
                {
                    Ok(block) => {
                        if t.saturating_sub(block.header.timestamp)
                            .gt(&self.max_unsafe_interval)
                        {
                            warn!(target: "rollup_boost::health", curr_unix = %t, unsafe_unix = %block.header.timestamp, "L2 client - unsafe block timestamp is too old, updating health status to ServiceUnavailable");
                            self.probes.set_health(Health::ServiceUnavailable);
                            sleep_until(Instant::now() + self.health_check_interval).await;
                            continue;
                        } else if self.execution_mode.lock().is_disabled()
                            || self.execution_mode.lock().is_dry_run()
                        {
                            self.probes.set_health(Health::Healthy);
                            sleep_until(Instant::now() + self.health_check_interval).await;
                            continue;
                        }
                    }
                    Err(e) => {
                        warn!(target: "rollup_boost::health", "L2 client - Failed to get unsafe block {} - updating health status", e);
                        self.probes.set_health(Health::ServiceUnavailable);
                        sleep_until(Instant::now() + self.health_check_interval).await;
                        continue;
                    }
                };

                if self.execution_mode.lock().is_enabled() {
                    // Only check builder client health if execution mode is enabled
                    // If its unhealthy, set the health status to PartialContent
                    match self
                        .builder_client
                        .get_block_by_number(BlockNumberOrTag::Latest, false)
                        .await
                    {
                        Ok(block) => {
                            if t.saturating_sub(block.header.timestamp)
                                .gt(&self.max_unsafe_interval)
                            {
                                warn!(target: "rollup_boost::health", curr_unix = %t, unsafe_unix = %block.header.timestamp, "Builder client - unsafe block timestamp is too old updating health status");
                                self.probes.set_health(Health::PartialContent);
                            } else {
                                self.probes.set_health(Health::Healthy);
                            }
                        }
                        Err(e) => {
                            warn!(target: "rollup_boost::health", "Builder client - Failed to get unsafe block {} - updating health status", e);
                            self.probes.set_health(Health::PartialContent);
                        }
                    };
                }
                sleep_until(Instant::now() + self.health_check_interval).await;
            }
        })
    }
}

/// A monotonic wall-clock timestamp tracker that resists system clock changes.
///
/// This struct provides a way to generate wall-clock-like timestamps that are
/// guaranteed to be monotonic (i.e., never go backward), even if the system
/// time is adjusted (e.g., via NTP, manual clock changes, or suspend/resume).
///
/// - It tracks elapsed time using `Instant` to ensure monotonic progression.
/// - It produces a synthetic wall-clock timestamp that won't regress.
pub struct MonotonicTimestamp {
    /// The last known UNIX timestamp in seconds.
    pub last_unix: u64,

    /// The last monotonic time reference.
    pub last_instant: Instant,
}

impl Default for MonotonicTimestamp {
    fn default() -> Self {
        Self::new()
    }
}

impl MonotonicTimestamp {
    pub fn new() -> Self {
        let last_unix = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("Time went backwards")
            .as_secs();
        let last_instant = Instant::now();
        Self {
            last_unix,
            last_instant,
        }
    }

    fn tick(&mut self) -> u64 {
        let elapsed = self.last_instant.elapsed().as_secs();
        self.last_unix += elapsed;
        self.last_instant = Instant::now();
        self.last_unix
    }
}

#[cfg(test)]
mod tests {
    use std::net::SocketAddr;

    use alloy_consensus::Header;
    use alloy_rpc_types_eth::{Block, Header as EthHeader, Transaction};

    use crate::RpcClient;
    use http::Uri;
    use http_body_util::BodyExt;
    use hyper::service::service_fn;
    use hyper_util::rt::TokioIo;
    use reth_rpc_layer::JwtSecret;
    use serde_json::json;
    use tokio::net::TcpListener;

    use super::*;
    use crate::Probes;
    use crate::payload::PayloadSource;
    use serial_test::serial;

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
                let listener = TcpListener::bind("127.0.0.1:0").await?;
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

    #[serial]
    #[tokio::test]
    async fn test_health_check_healthy() -> eyre::Result<()> {
        let probes = Arc::new(Probes::default());
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("Time went backwards")
            .as_secs();

        let l2 = MockHttpServer::serve(handler, now).await.unwrap();
        let l2_client = Arc::new(RpcClient::new(
            format!("http://{}", l2.addr).parse::<Uri>()?,
            JwtSecret::random(),
            100,
            PayloadSource::L2,
        )?);

        let builder = MockHttpServer::serve(handler, now).await.unwrap();
        let builder_client = Arc::new(RpcClient::new(
            format!("http://{}", builder.addr).parse::<Uri>()?,
            JwtSecret::random(),
            100,
            PayloadSource::Builder,
        )?);

        let health_handle = HealthHandle {
            probes: probes.clone(),
            execution_mode: Arc::new(Mutex::new(ExecutionMode::Enabled)),
            l2_client: l2_client.clone(),
            builder_client: builder_client.clone(),
            health_check_interval: Duration::from_secs(60),
            max_unsafe_interval: 5,
        };

        health_handle.spawn();
        tokio::time::sleep(Duration::from_secs(2)).await;
        assert!(matches!(probes.health(), Health::Healthy));
        Ok(())
    }

    #[serial]
    #[tokio::test]
    async fn test_health_check_builder_exceeds_max_unsafe_interval() -> eyre::Result<()> {
        let probes = Arc::new(Probes::default());
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("Time went backwards")
            .as_secs();

        // L2 healthy
        let l2 = MockHttpServer::serve(handler, now).await.unwrap();
        let l2_client = Arc::new(RpcClient::new(
            format!("http://{}", l2.addr).parse::<Uri>()?,
            JwtSecret::random(),
            100,
            PayloadSource::L2,
        )?);

        // Builder unhealthy
        let builder = MockHttpServer::serve(handler, now - 10).await.unwrap();
        let builder_client = Arc::new(RpcClient::new(
            format!("http://{}", builder.addr).parse::<Uri>()?,
            JwtSecret::random(),
            100,
            PayloadSource::Builder,
        )?);

        let health_handle = HealthHandle {
            probes: probes.clone(),
            execution_mode: Arc::new(Mutex::new(ExecutionMode::Enabled)),
            l2_client: l2_client.clone(),
            builder_client: builder_client.clone(),
            health_check_interval: Duration::from_secs(60),
            max_unsafe_interval: 5,
        };

        health_handle.spawn();
        tokio::time::sleep(Duration::from_secs(2)).await;
        assert!(matches!(probes.health(), Health::PartialContent));
        Ok(())
    }

    #[serial]
    #[tokio::test]
    async fn test_health_check_l2_exceeds_max_unsafe_interval() -> eyre::Result<()> {
        let probes = Arc::new(Probes::default());
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("Time went backwards")
            .as_secs();

        // L2 healthy unhealth
        let l2 = MockHttpServer::serve(handler, now - 10).await.unwrap();
        let l2_client = Arc::new(RpcClient::new(
            format!("http://{}", l2.addr).parse::<Uri>()?,
            JwtSecret::random(),
            100,
            PayloadSource::L2,
        )?);

        // Builder healthy
        let builder = MockHttpServer::serve(handler, now).await.unwrap();
        let builder_client = Arc::new(RpcClient::new(
            format!("http://{}", builder.addr).parse::<Uri>()?,
            JwtSecret::random(),
            100,
            PayloadSource::Builder,
        )?);

        let health_handle = HealthHandle {
            probes: probes.clone(),
            execution_mode: Arc::new(Mutex::new(ExecutionMode::Enabled)),
            l2_client: l2_client.clone(),
            builder_client: builder_client.clone(),
            health_check_interval: Duration::from_secs(60),
            max_unsafe_interval: 5,
        };

        health_handle.spawn();
        tokio::time::sleep(Duration::from_secs(2)).await;
        assert!(matches!(probes.health(), Health::ServiceUnavailable));
        Ok(())
    }

    #[serial]
    #[tokio::test]
    async fn test_health_check_exceeds_max_unsafe_interval_execution_mode_disabled()
    -> eyre::Result<()> {
        let probes = Arc::new(Probes::default());
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("Time went backwards")
            .as_secs();
        // L2 healthy
        let l2 = MockHttpServer::serve(handler, now).await.unwrap();
        let l2_client = Arc::new(RpcClient::new(
            format!("http://{}", l2.addr).parse::<Uri>()?,
            JwtSecret::random(),
            100,
            PayloadSource::L2,
        )?);
        let builder = MockHttpServer::serve(handler, now - 10).await.unwrap();

        let builder_client = Arc::new(RpcClient::new(
            format!("http://{}", builder.addr).parse::<Uri>()?,
            JwtSecret::random(),
            100,
            PayloadSource::Builder,
        )?);

        let health_handle = HealthHandle {
            probes: probes.clone(),
            execution_mode: Arc::new(Mutex::new(ExecutionMode::Disabled)),
            l2_client: l2_client.clone(),
            builder_client: builder_client.clone(),
            health_check_interval: Duration::from_secs(60),
            max_unsafe_interval: 5,
        };

        health_handle.spawn();
        tokio::time::sleep(Duration::from_secs(2)).await;
        assert!(matches!(probes.health(), Health::Healthy));
        Ok(())
    }

    #[serial]
    #[tokio::test]
    async fn test_health_check_exceeds_max_unsafe_interval_execution_mode_dryrun()
    -> eyre::Result<()> {
        let probes = Arc::new(Probes::default());
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("Time went backwards")
            .as_secs();
        // L2 healthy
        let l2 = MockHttpServer::serve(handler, now).await.unwrap();
        let l2_client = Arc::new(RpcClient::new(
            format!("http://{}", l2.addr).parse::<Uri>()?,
            JwtSecret::random(),
            100,
            PayloadSource::L2,
        )?);
        let builder = MockHttpServer::serve(handler, now - 10).await.unwrap();

        let builder_client = Arc::new(RpcClient::new(
            format!("http://{}", builder.addr).parse::<Uri>()?,
            JwtSecret::random(),
            100,
            PayloadSource::Builder,
        )?);

        let health_handle = HealthHandle {
            probes: probes.clone(),
            execution_mode: Arc::new(Mutex::new(ExecutionMode::DryRun)),
            l2_client: l2_client.clone(),
            builder_client: builder_client.clone(),
            health_check_interval: Duration::from_secs(60),
            max_unsafe_interval: 5,
        };

        health_handle.spawn();
        tokio::time::sleep(Duration::from_secs(2)).await;
        assert!(matches!(probes.health(), Health::Healthy));
        Ok(())
    }

    #[serial]
    #[tokio::test]
    async fn test_health_check_service_builder_unavailable() -> eyre::Result<()> {
        let probes = Arc::new(Probes::default());
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("Time went backwards")
            .as_secs();
        // L2 healthy
        let l2 = MockHttpServer::serve(handler, now).await.unwrap();
        let l2_client = Arc::new(RpcClient::new(
            format!("http://{}", l2.addr).parse::<Uri>()?,
            JwtSecret::random(),
            100,
            PayloadSource::L2,
        )?);

        // Builder unhealthy
        let builder_client = Arc::new(RpcClient::new(
            "http://127.0.0.1:6000".parse::<Uri>()?,
            JwtSecret::random(),
            100,
            PayloadSource::Builder,
        )?);

        let health_handle = HealthHandle {
            probes: probes.clone(),
            execution_mode: Arc::new(Mutex::new(ExecutionMode::Enabled)),
            l2_client: l2_client.clone(),
            builder_client: builder_client.clone(),
            health_check_interval: Duration::from_secs(60),
            max_unsafe_interval: 5,
        };

        health_handle.spawn();
        tokio::time::sleep(Duration::from_secs(2)).await;
        assert!(matches!(probes.health(), Health::PartialContent));
        Ok(())
    }

    #[serial]
    #[tokio::test]
    async fn test_health_check_service_l2_unavailable() -> eyre::Result<()> {
        let probes = Arc::new(Probes::default());
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("Time went backwards")
            .as_secs();

        // L2 returns an error
        let l2_client = Arc::new(RpcClient::new(
            "http://127.0.0.1:6000".parse::<Uri>()?,
            JwtSecret::random(),
            100,
            PayloadSource::L2,
        )?);

        // Builder healthy
        let builder = MockHttpServer::serve(handler, now).await.unwrap();
        let builder_client = Arc::new(RpcClient::new(
            format!("http://{}", builder.addr).parse::<Uri>()?,
            JwtSecret::random(),
            100,
            PayloadSource::Builder,
        )?);

        let health_handle = HealthHandle {
            probes: probes.clone(),
            execution_mode: Arc::new(Mutex::new(ExecutionMode::Enabled)),
            l2_client: l2_client.clone(),
            builder_client: builder_client.clone(),
            health_check_interval: Duration::from_secs(60),
            max_unsafe_interval: 5,
        };

        health_handle.spawn();
        tokio::time::sleep(Duration::from_secs(2)).await;
        assert!(matches!(probes.health(), Health::ServiceUnavailable));
        Ok(())
    }

    #[serial]
    #[tokio::test]
    async fn tick_advances_after_sleep() {
        let mut ts: MonotonicTimestamp = MonotonicTimestamp::new();
        let t1 = ts.tick();
        tokio::time::sleep(Duration::from_secs(1)).await;
        let t2 = ts.tick();

        assert!(t2 >= t1 + 1,);
    }

    #[serial]
    #[tokio::test]
    async fn tick_matches_system_clock() {
        let mut ts = MonotonicTimestamp::new();
        let unix = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("Time went backwards")
            .as_secs();

        assert_eq!(ts.last_unix, unix);

        std::thread::sleep(Duration::from_secs(5));

        let t1 = ts.tick();
        let unix = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("Time went backwards")
            .as_secs();
        assert_eq!(t1, unix);
    }
}
