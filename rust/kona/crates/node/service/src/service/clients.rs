//! The clients a [`RollupNode`](crate::RollupNode) talks to other services with.

use crate::{DerivationDelegateClient, DerivationDelegateClientError, RollupNodeConfig};
use alloy_primitives::Bytes;
use alloy_provider::RootProvider;
use alloy_rpc_client::RpcClient;
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

/// The clients a [`RollupNode`](crate::RollupNode) talks to the L1 and L2 chains with.
///
/// [`Self::connect`] builds the standard clients for a [`RollupNodeConfig`]. Embedders and tests
/// can build their own instead and pass them to [`RollupNode::new`](crate::RollupNode::new).
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
    /// The consensus layer the node follows instead of deriving, when it is configured with one.
    pub derivation_delegate: Option<DerivationDelegateClient>,
}

impl RollupNodeClients {
    /// Builds the standard clients for `config`.
    ///
    /// This only builds the clients and does no network I/O, so it does not check that the
    /// endpoints are reachable.
    pub fn connect(config: &RollupNodeConfig) -> Result<Self, RollupNodeClientsError> {
        let l1_http = reqwest::Client::builder()
            .timeout(Duration::from_secs(10))
            .build()
            .map_err(RollupNodeClientsError::L1HttpClient)?;
        let l1 = RootProvider::new(RpcClient::new_http_with_client(
            l1_http,
            config.l1_client.rpc_url.clone(),
        ));
        let l1_beacon = OnlineBeaconClient::new_http(config.l1_client.beacon.to_string());

        let hyper_client = Client::builder(TokioExecutor::new()).build_http::<Full<Bytes>>();
        let service = ServiceBuilder::new()
            .layer(AuthLayer::new(config.engine_config.l2_jwt_secret))
            .service(hyper_client);
        let l2 = RootProvider::<Optimism>::new(RpcClient::new(
            Http::with_client(
                HyperClient::with_service(service),
                config.engine_config.l2_url.clone(),
            ),
            false,
        ));

        let engine = EngineClient::rpc_client(
            config.engine_config.l2_url.clone(),
            config.engine_config.l2_jwt_secret,
        );

        let derivation_delegate = config
            .derivation_delegate_config
            .as_ref()
            .map(|delegate| DerivationDelegateClient::new(delegate.l2_cl_url.clone()))
            .transpose()?;

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
    use crate::{
        DerivationDelegateConfig, EngineConfig, L1ClientConfig, NetworkConfig, NodeMode,
        SequencerConfig,
    };
    use alloy_rpc_types_engine::JwtSecret;
    use kona_disc::LocalNode;
    use kona_genesis::{L1ChainConfig, RollupConfig};
    use std::{
        net::{IpAddr, Ipv4Addr},
        sync::Arc,
    };

    /// A node config whose endpoints nothing listens on.
    fn config(derivation_delegate_config: Option<DerivationDelegateConfig>) -> RollupNodeConfig {
        let rollup_config = Arc::new(RollupConfig::default());
        let local_node = LocalNode::new(
            enr::k256::ecdsa::SigningKey::from_bytes(&[1u8; 32].into()).unwrap(),
            IpAddr::V4(Ipv4Addr::LOCALHOST),
            0,
            0,
        );
        RollupNodeConfig {
            version: "test".to_owned(),
            rollup_config: rollup_config.clone(),
            l1_client: L1ClientConfig {
                chain_config: Arc::new(L1ChainConfig::default()),
                beacon: "http://127.0.0.1:1".parse().unwrap(),
                rpc_url: "http://127.0.0.1:2".parse().unwrap(),
            },
            engine_config: EngineConfig {
                l2_url: "http://127.0.0.1:3".parse().unwrap(),
                l2_jwt_secret: JwtSecret::from_hex(
                    "0xabababababababababababababababababababababababababababababababab",
                )
                .unwrap(),
                mode: NodeMode::Validator,
            },
            p2p_config: NetworkConfig::new(
                (*rollup_config).clone(),
                local_node,
                libp2p::Multiaddr::empty(),
            ),
            rpc_config: None,
            sequencer_config: SequencerConfig::default(),
            derivation_delegate_config,
            dependency_set: None,
        }
    }

    #[tokio::test]
    async fn connect_builds_the_delegate_client_only_when_configured() {
        let clients = RollupNodeClients::connect(&config(None)).unwrap();
        assert!(clients.derivation_delegate.is_none());

        let delegate =
            DerivationDelegateConfig { l2_cl_url: "http://127.0.0.1:4".parse().unwrap() };
        let clients = RollupNodeClients::connect(&config(Some(delegate))).unwrap();
        assert!(clients.derivation_delegate.is_some());
    }

    #[tokio::test]
    async fn the_engine_client_uses_the_configs_rollup_config() {
        let config = config(None);
        let rollup_config = config.rollup_config.clone();
        let clients = RollupNodeClients::connect(&config).unwrap();

        let node = crate::RollupNode::new(config, clients);

        assert!(std::ptr::eq(node.engine_client.cfg(), rollup_config.as_ref()));
    }
}
