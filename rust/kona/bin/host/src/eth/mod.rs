//! Ethereum utilities for the host binary.

use alloy_provider::{ConnectionConfig, Network, RootProvider, WebSocketConfig};
use alloy_rpc_client::{BuiltInConnectionString, ClientBuilder};
use alloy_transport::{TransportConnect, TransportResult};

mod precompiles;
pub(crate) use precompiles::execute;

/// Connects to the RPC endpoint at `url` over HTTP, websocket or IPC.
///
/// Uses Reqwest explicitly for HTTP while preserving the configured websocket and IPC transports.
/// Websocket messages and frames have no size limit, as over HTTP.
pub async fn connect<N: Network>(url: &str) -> TransportResult<RootProvider<N>> {
    let connection = url.parse::<BuiltInConnectionString>()?;
    if let BuiltInConnectionString::Http(url) = connection {
        return Ok(RootProvider::new_http(url));
    }
    let config = ConnectionConfig::new().with_ws_config(ws_config());
    let transport = connection.connect_boxed_with(config).await?;
    Ok(RootProvider::new(ClientBuilder::default().transport(transport, connection.is_local())))
}

/// Returns an RPC provider for the given URL, as [`connect`] does. Panics if it can't connect.
pub async fn rpc_provider<N: Network>(url: &str) -> RootProvider<N> {
    connect(url).await.unwrap()
}

/// The websocket settings of [`connect`]: messages and frames have no size limit, as over HTTP.
/// There should be no limit: don't add one.
fn ws_config() -> WebSocketConfig {
    WebSocketConfig::default().max_message_size(None).max_frame_size(None)
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
    use std::{net::SocketAddr, sync::Arc, time::Duration};

    /// How long a call in these tests may take. A local call in them takes well under a second.
    const CALL_DEADLINE: Duration = Duration::from_secs(30);

    /// The size of the response in [`large_responses_arrive_over_http_and_websocket`]: just above
    /// tungstenite's default message size, the larger of the two limits that [`ws_config`]
    /// removes.
    fn response_size() -> usize {
        WebSocketConfig::default().max_message_size.unwrap() + (1 << 20)
    }

    /// Serves `test_large` over HTTP and websocket on one address, answering with `size / 2` zero
    /// bytes, which hex encodes to `size` bytes. Returns the address, and the handle that keeps
    /// the server running.
    async fn large_response_server(size: usize) -> (SocketAddr, ServerHandle) {
        let config = ServerConfig::builder().max_response_body_size(u32::MAX).build();
        let server = Server::builder().set_config(config).build("127.0.0.1:0").await.unwrap();
        let addr = server.local_addr().unwrap();
        let result: Arc<str> = format!("0x{}", "00".repeat(size / 2)).into();
        let mut module = RpcModule::new(());
        module.register_method("test_large", move |_, _, _| result.to_string()).unwrap();
        (addr, server.start(module))
    }

    /// Checks that [`ws_config`] keeps no limit. A response size test can't catch a large limit.
    #[test]
    fn websocket_messages_and_frames_have_no_size_limit() {
        let config = ws_config();
        assert_eq!(config.max_message_size, None);
        assert_eq!(config.max_frame_size, None);
    }

    #[rstest]
    #[tokio::test]
    async fn large_responses_arrive_over_http_and_websocket(#[values("http", "ws")] scheme: &str) {
        let size = response_size();
        let (addr, _server) = large_response_server(size).await;
        let provider = rpc_provider::<Ethereum>(&format!("{scheme}://{addr}")).await;
        // Subscriptions stay available over websocket.
        assert_eq!(provider.client().pubsub_frontend().is_some(), scheme == "ws");
        let call = provider.client().request::<_, Bytes>("test_large", ());
        let result = tokio::time::timeout(CALL_DEADLINE, call)
            .await
            .expect("the call completes before its deadline")
            .unwrap();
        assert_eq!(result.len(), size / 2);
    }
}
