//! The clients a [`RollupNode`](crate::RollupNode) talks to other services with.

use crate::{DerivationDelegateClient, DerivationDelegateClientError};
use alloy_primitives::Bytes;
use alloy_provider::RootProvider;
use alloy_rpc_client::RpcClient;
use alloy_rpc_types_engine::JwtSecret;
use alloy_transport_http::{
    AuthLayer, Http, HyperClient,
    hyper_util::{client::legacy::Client, rt::TokioExecutor},
    reqwest,
};
use http_body_util::Full;
use kona_engine::EngineClient;
use kona_providers_alloy::OnlineBeaconClient;
use op_alloy_network::Optimism;
use std::time::Duration;
use tower::ServiceBuilder;
use url::Url;

/// Where the node's L1 and L2 endpoints are, which [`RollupNodeClients::connect`] builds the
/// standard clients for.
#[derive(Debug, Clone)]
pub struct RollupNodeEndpoints {
    /// The L1 execution layer's RPC.
    pub l1_rpc: Url,
    /// The L1 beacon API.
    pub l1_beacon: Url,
    /// The L2 execution layer's authenticated RPC, which also serves the Engine API.
    pub l2_engine: Url,
    /// The JWT secret the L2 execution layer authenticates the node with.
    pub l2_jwt_secret: JwtSecret,
    /// The consensus layer to follow instead of deriving, if any. It must expose the
    /// `optimism_syncStatus` RPC method.
    pub derivation_delegate: Option<Url>,
}

/// The clients a [`RollupNode`](crate::RollupNode) talks to the L1 and L2 chains with.
///
/// [`Self::connect`] builds the standard clients for a set of [`RollupNodeEndpoints`]. Embedders
/// and tests can build their own instead and pass them to
/// [`RollupNode::new`](crate::RollupNode::new).
#[derive(Debug, Clone)]
pub struct RollupNodeClients {
    /// The L1 execution layer's RPC.
    pub l1: RootProvider,
    /// The L1 beacon API.
    pub l1_beacon: OnlineBeaconClient,
    /// The L2 execution layer's RPC, authenticated with the engine's JWT secret. Derivation reads
    /// L2 blocks through it.
    pub l2: RootProvider<Optimism>,
    /// The L2 execution layer's Engine API, authenticated with the engine's JWT secret. It must
    /// reach the same execution layer as [`Self::l2`].
    ///
    /// [`RollupNode::new`](crate::RollupNode::new) builds the node's [`EngineClient`] from it,
    /// [`Self::l1`] and the config's rollup config, so the engine follows the same chain as the
    /// rest of the node.
    pub engine: RootProvider<Optimism>,
    /// The consensus layer the node follows instead of deriving. The node derives the safe chain
    /// itself when this is `None`.
    pub derivation_delegate: Option<DerivationDelegateClient>,
}

impl RollupNodeClients {
    /// Builds the standard clients for `endpoints`.
    ///
    /// This only builds the clients and does no network I/O, so it does not check that the
    /// endpoints are reachable.
    pub fn connect(endpoints: &RollupNodeEndpoints) -> Result<Self, RollupNodeClientsError> {
        let l1_http = reqwest::Client::builder()
            .timeout(Duration::from_secs(10))
            .build()
            .map_err(RollupNodeClientsError::L1HttpClient)?;
        let l1 =
            RootProvider::new(RpcClient::new_http_with_client(l1_http, endpoints.l1_rpc.clone()));
        let l1_beacon = OnlineBeaconClient::new_http(endpoints.l1_beacon.to_string());

        let hyper_client = Client::builder(TokioExecutor::new()).build_http::<Full<Bytes>>();
        let service = ServiceBuilder::new()
            .layer(AuthLayer::new(endpoints.l2_jwt_secret))
            .service(hyper_client);
        let l2 = RootProvider::<Optimism>::new(RpcClient::new(
            Http::with_client(HyperClient::with_service(service), endpoints.l2_engine.clone()),
            false,
        ));

        let engine = EngineClient::rpc_client(endpoints.l2_engine.clone(), endpoints.l2_jwt_secret);

        let derivation_delegate =
            endpoints.derivation_delegate.clone().map(DerivationDelegateClient::new).transpose()?;

        Ok(Self { l1, l1_beacon, l2, engine, derivation_delegate })
    }
}

/// An error building a [`RollupNodeClients`].
#[derive(Debug, thiserror::Error)]
pub enum RollupNodeClientsError {
    /// The L1 RPC's HTTP client could not be built.
    #[error("failed to build the L1 RPC's HTTP client")]
    L1HttpClient(#[source] reqwest::Error),
    /// The derivation delegate's client could not be built.
    #[error("failed to build the derivation delegate's client")]
    DerivationDelegate(#[from] DerivationDelegateClientError),
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::service::test_utils::{endpoints, network_config, node_config};

    #[tokio::test]
    async fn connect_builds_the_delegate_client_only_when_configured() {
        let clients = RollupNodeClients::connect(&endpoints(None)).unwrap();
        assert!(clients.derivation_delegate.is_none());

        let delegate = "http://127.0.0.1:4".parse().unwrap();
        let clients = RollupNodeClients::connect(&endpoints(Some(delegate))).unwrap();
        assert!(clients.derivation_delegate.is_some());
    }

    #[tokio::test]
    async fn the_engine_client_uses_the_configs_rollup_config() {
        let config = node_config(network_config(0, 0));
        let rollup_config = config.chain.rollup_config.clone();
        let clients = RollupNodeClients::connect(&endpoints(None)).unwrap();

        let node = crate::RollupNode::new(config, clients);

        assert!(std::ptr::eq(node.engine_client.cfg(), rollup_config.as_ref()));
    }
}
