use axum::body::Bytes;
use std::{error::Error, net::SocketAddr, time::Duration};
use tdx_quote_provider::server::{Server, ServerConfig};
use tokio::net::TcpListener;

struct TestHarness {
    server: Server,
    server_addr: SocketAddr,
}

impl TestHarness {
    async fn alloc_port() -> SocketAddr {
        let address = SocketAddr::from(([127, 0, 0, 1], 0));
        let listener = TcpListener::bind(&address).await.unwrap();
        listener.local_addr().unwrap()
    }

    fn new(addr: SocketAddr) -> TestHarness {
        let path = format!("{}/tests/test_data/quote.bin", env!("CARGO_MANIFEST_DIR"));
        Self {
            server: Server::new(ServerConfig {
                listen_addr: addr,
                use_mock: true,
                mock_attestation_path: path,
            }),
            server_addr: addr,
        }
    }

    async fn healthcheck(&self) -> Result<(), Box<dyn Error>> {
        let url = format!("http://{}/healthz", self.server_addr);
        let response = reqwest::get(url).await?;
        match response.error_for_status() {
            Ok(_) => Ok(()),
            Err(e) => Err(e.into()),
        }
    }

    async fn attest(&self, app_data: String) -> Result<Bytes, Box<dyn Error>> {
        let url = format!("http://{}/attest/{}", self.server_addr, app_data);
        let response = reqwest::get(url).await?;
        match response.error_for_status() {
            Ok(response) => {
                let body = response.bytes().await?;
                Ok(body)
            }
            Err(e) => Err(e.into()),
        }
    }

    async fn start_server(&mut self) {
        let mut healthy = true;

        let server = self.server.clone();
        let _server_handle = tokio::spawn(async move {
            _ = server.listen().await;
        });

        for _ in 0..5 {
            let resp = self.healthcheck().await;
            match resp {
                Ok(_) => {
                    healthy = true;
                    break;
                }
                Err(_) => {
                    tokio::time::sleep(Duration::from_millis(25)).await;
                }
            }
        }

        assert!(healthy);
    }
}

#[tokio::test]
async fn test_healthcheck() {
    let addr = TestHarness::alloc_port().await;
    let mut harness = TestHarness::new(addr);
    assert!(harness.healthcheck().await.is_err());
    harness.start_server().await;
    assert!(harness.healthcheck().await.is_ok());
}

#[tokio::test]
async fn test_mock_attest() {
    let addr = TestHarness::alloc_port().await;

    let mut harness = TestHarness::new(addr);
    harness.start_server().await;

    let report_data = hex::encode(vec![0; 64]);
    let response = harness.attest(report_data).await;
    assert!(response.is_ok());
    let body = response.unwrap();
    assert_eq!(
        body,
        Bytes::from_static(include_bytes!("./test_data/quote.bin"))
    );
}

#[tokio::test]
async fn test_attest_invalid_report_data() {
    let addr = TestHarness::alloc_port().await;

    let mut harness = TestHarness::new(addr);
    harness.start_server().await;

    let response = harness.attest("invalid".to_string()).await;
    assert!(response.is_err());

    let response = harness.attest("aede".to_string()).await;
    assert!(response.is_err());
}
