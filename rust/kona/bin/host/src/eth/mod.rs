//! Ethereum utilities for the host binary.

use alloy_provider::{Network, RootProvider};
use alloy_rpc_client::BuiltInConnectionString;

mod precompiles;
pub(crate) use precompiles::execute;

/// Returns an RPC provider for the given URL.
///
/// Uses Reqwest explicitly for HTTP while preserving the configured `WebSocket` and IPC transports.
pub async fn rpc_provider<N: Network>(url: &str) -> RootProvider<N> {
    let connection = url.parse::<BuiltInConnectionString>().unwrap();
    if let BuiltInConnectionString::Http(url) = connection {
        RootProvider::new_http(url)
    } else {
        RootProvider::connect_with(connection).await.unwrap()
    }
}

#[cfg(test)]
mod tests {
    use super::rpc_provider;
    use alloy_provider::Provider;
    use futures_util::{SinkExt, StreamExt};
    use op_alloy_network::Optimism;
    use serde_json::{Value, json};
    use tokio::net::TcpListener;
    use tokio_tungstenite::{accept_async, tungstenite::Message};

    #[tokio::test]
    async fn supports_websocket_rpc_urls() {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let address = listener.local_addr().unwrap();
        let server = tokio::spawn(async move {
            let (stream, _) = listener.accept().await.unwrap();
            let mut socket = accept_async(stream).await.unwrap();
            let request = socket.next().await.unwrap().unwrap();
            let request: Value = serde_json::from_str(request.to_text().unwrap()).unwrap();
            let response = json!({
                "jsonrpc": "2.0",
                "id": request["id"],
                "result": "0xa",
            });
            socket.send(Message::Text(response.to_string().into())).await.unwrap();
        });

        let provider = rpc_provider::<Optimism>(&format!("ws://{address}")).await;
        assert_eq!(provider.get_chain_id().await.unwrap(), 10);
        server.await.unwrap();
    }
}
