//! Ethereum utilities for the host binary.

use alloy_provider::{Network, RootProvider};

mod precompiles;
pub(crate) use precompiles::execute;

/// Returns an HTTP provider for the given URL.
///
/// # Panics
///
/// Panics if internal invariants are violated.
pub async fn rpc_provider<N: Network>(url: &str) -> RootProvider<N> {
    RootProvider::connect(url).await.unwrap()
}
