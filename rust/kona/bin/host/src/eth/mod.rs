//! Ethereum utilities for the host binary.

use alloy_provider::{Network, RootProvider};

mod precompiles;
pub(crate) use precompiles::execute;

/// Returns an HTTP provider for the given URL.
///
/// Uses the explicit Reqwest constructor so workspace feature unification cannot change transports.
pub fn rpc_provider<N: Network>(url: &str) -> RootProvider<N> {
    RootProvider::new_http(url.parse().unwrap())
}
