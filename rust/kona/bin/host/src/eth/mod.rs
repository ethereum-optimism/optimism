//! Ethereum utilities for the host binary.

use alloy_provider::{Network, RootProvider};
use alloy_rpc_client::BuiltInConnectionString;

mod precompiles;
pub(crate) use precompiles::execute;

/// Returns an RPC provider for the given URL.
///
/// Uses Reqwest explicitly for HTTP while preserving the configured websocket and IPC transports.
pub async fn rpc_provider<N: Network>(url: &str) -> RootProvider<N> {
    let connection = url.parse::<BuiltInConnectionString>().unwrap();
    if let BuiltInConnectionString::Http(url) = connection {
        RootProvider::new_http(url)
    } else {
        RootProvider::connect_with(connection).await.unwrap()
    }
}
