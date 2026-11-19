//! Ethereum utilities for the host binary.

use alloy_provider::{Network, RootProvider, WebSocketConfig};
use alloy_rpc_client::{BuiltInConnectionString, ClientBuilder, ConnectionConfig};

mod precompiles;
pub(crate) use precompiles::execute;

/// Returns an RPC provider for the given URL.
///
/// Uses Reqwest explicitly for HTTP while preserving the configured websocket and IPC transports.
/// Websocket messages and frames have no size limit, to match HTTP.
pub async fn rpc_provider<N: Network>(url: &str) -> RootProvider<N> {
    let connection = url.parse::<BuiltInConnectionString>().unwrap();
    if let BuiltInConnectionString::Http(url) = connection {
        RootProvider::new_http(url)
    } else {
        let ws_config = WebSocketConfig::default().max_message_size(None).max_frame_size(None);
        let config = ConnectionConfig::new().with_ws_config(ws_config);
        RootProvider::new(ClientBuilder::default().connect_with_config(url, config).await.unwrap())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::Bytes;
    use alloy_provider::{Provider, network::Ethereum};
    use jsonrpsee::{
        RpcModule,
        server::{Server, ServerConfig, ServerHandle},
    };
    use rstest::rstest;
    use std::{sync::Arc, time::Duration};

    /// tungstenite's default maximum websocket frame size, which alloy's websocket transport
    /// uses.
    const TUNGSTENITE_FRAME_LIMIT: usize = 16 << 20;

    /// A response far larger than typical, and above tungstenite's default message size.
    const LARGE_RESPONSE: usize = 72 << 20;

    /// How long a call in these tests may take. A local call of [`LARGE_RESPONSE`] takes well
    /// under a second.
    const CALL_DEADLINE: Duration = Duration::from_secs(30);

    /// Serves `eth_call` over HTTP and websocket on one address, answering with `size / 2` zero
    /// bytes, which hex encodes to `size` bytes. Returns the HTTP and websocket URLs.
    async fn large_response_server(size: usize) -> (String, String, ServerHandle) {
        let config = ServerConfig::builder().max_response_body_size(u32::MAX).build();
        let server = Server::builder().set_config(config).build("127.0.0.1:0").await.unwrap();
        let addr = server.local_addr().unwrap();
        let result: Arc<str> = format!("0x{}", "00".repeat(size / 2)).into();
        let mut module = RpcModule::new(());
        module.register_method("eth_call", move |_, _, _| result.to_string()).unwrap();
        (format!("http://{addr}"), format!("ws://{addr}"), server.start(module))
    }

    #[rstest]
    #[case::above_tungstenite_frame_limit(TUNGSTENITE_FRAME_LIMIT + (1 << 20))]
    #[case::large(LARGE_RESPONSE)]
    #[tokio::test]
    async fn large_responses_arrive_over_http_and_websocket(
        #[case] size: usize,
        #[values("http", "ws")] transport: &str,
    ) {
        let (http, ws, _server) = large_response_server(size).await;
        let url = if transport == "http" { http } else { ws };
        let provider = rpc_provider::<Ethereum>(&url).await;
        let call = provider.client().request::<_, Bytes>("eth_call", ());
        let result = tokio::time::timeout(CALL_DEADLINE, call)
            .await
            .expect("the call completes before its deadline")
            .unwrap();
        assert_eq!(result.len(), size / 2);
    }
}
