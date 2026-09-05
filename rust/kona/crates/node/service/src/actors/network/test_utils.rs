//! Test-only fixtures for the [`NetworkActor`](super::NetworkActor).
//!
//! Binds no ports: the swarm is built but never started, so `gossip.next()` stays pending and
//! nothing on the publish path misses it. Signer behaviour is injected at the JSON-RPC boundary
//! because `BlockSigner` is a concrete enum with no seam to mock.

use std::net::{IpAddr, Ipv4Addr};

use alloy_chains::Chain;
use alloy_primitives::{Address, B256};
use alloy_signer::SignerSync;
use alloy_signer_local::PrivateKeySigner;
use discv5::{ConfigBuilder, Enr, ListenConfig, enr::CombinedKey};
use jsonrpsee::{
    RpcModule,
    server::{Server, ServerHandle},
};
use kona_disc::{Discv5Handler, HandlerRequest, LocalNode};
use kona_genesis::RollupConfig;
use kona_gossip::{P2pRpcRequest, PEER_SCORE_INSPECT_FREQUENCY};
use kona_rpc::NetworkAdminQuery;
use kona_sources::{BlockSigner, RemoteSigner};
use libp2p::{Multiaddr, identity::Keypair};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use tokio::sync::mpsc;
use url::Url;

use crate::actors::network::{
    NetworkActor, NetworkBuilder, NetworkHandler, engine_client::MockNetworkEngineClient,
};

/// The chain id the fixture builds actors for.
const CHAIN_ID: u64 = 10;

/// How the mock signer answers `opsigner_signBlockPayload`. `health_status` is always answered,
/// since [`RemoteSigner::start`] pings it first.
#[derive(Debug, Clone, Copy)]
pub(crate) enum SignerBehaviour {
    /// Return a well-formed signature.
    Sign,
    /// Return a JSON-RPC error.
    Error,
    /// Accept the request and never answer it.
    Hang,
}

/// A running mock op-signer.
#[derive(Debug)]
pub(crate) struct MockSigner {
    /// The endpoint to point a [`RemoteSigner`] at.
    pub(crate) url: Url,
    /// The address the signer claims. A mismatched sender address is rejected before transport.
    pub(crate) address: Address,
    /// Dropping this stops the server.
    _handle: ServerHandle,
}

/// Starts a mock op-signer on an ephemeral port.
pub(crate) async fn spawn_mock_signer(behaviour: SignerBehaviour) -> MockSigner {
    let key = PrivateKeySigner::random();
    let address = key.address();

    // Reused for every request: the actor does not verify it, only forwards it.
    let canned = key.sign_hash_sync(&B256::random()).expect("signing a hash cannot fail");
    let canned = format!("0x{}", alloy_primitives::hex::encode(canned.as_bytes()));

    let server = Server::builder()
        .build((IpAddr::V4(Ipv4Addr::LOCALHOST), 0))
        .await
        .expect("mock signer failed to bind");
    let addr = server.local_addr().expect("mock signer has no local address");

    let mut module = RpcModule::new(());
    module
        .register_async_method("health_status", |_, _, _| async move { "mock".to_string() })
        .expect("failed to register health_status");
    module
        .register_async_method("opsigner_signBlockPayload", move |_, _, _| {
            let canned = canned.clone();
            async move {
                match behaviour {
                    SignerBehaviour::Sign => Ok(serde_json::json!({ "signature": canned })),
                    SignerBehaviour::Error => Err(jsonrpsee::types::ErrorObjectOwned::owned(
                        -32000,
                        "signer unavailable",
                        None::<()>,
                    )),
                    SignerBehaviour::Hang => {
                        std::future::pending::<()>().await;
                        unreachable!("pending never resolves")
                    }
                }
            }
        })
        .expect("failed to register opsigner_signBlockPayload");

    let handle = server.start(module);
    let url = Url::parse(&format!("http://{addr}")).expect("mock signer url is well formed");

    MockSigner { url, address, _handle: handle }
}

/// A [`NetworkActor`] wired for testing, holding every sender alive: the `unsafe_block_signer`
/// and `enr` receivers have no closed-channel guard, so a dropped sender makes `step()` fail for
/// the wrong reason.
#[derive(Debug)]
pub(crate) struct TestActor {
    /// The actor under test.
    pub(crate) actor: NetworkActor<MockNetworkEngineClient>,
    /// Schedules payloads for signing and publishing.
    pub(crate) publish_tx: mpsc::Sender<OpExecutionPayloadEnvelope>,
    /// Admin queries, including `PostUnsafePayload`.
    #[expect(dead_code, reason = "held so the actor's arm stays enabled, as in production")]
    pub(crate) admin_tx: mpsc::Sender<NetworkAdminQuery>,
    #[expect(dead_code, reason = "held so the actor's receiver never sees a closed channel")]
    pub(crate) unsafe_block_signer_tx: mpsc::Sender<Address>,
    #[expect(dead_code, reason = "held so the actor's receiver never sees a closed channel")]
    pub(crate) p2p_rpc_tx: mpsc::Sender<P2pRpcRequest>,
    #[expect(dead_code, reason = "held so the actor's receiver never sees a closed channel")]
    pub(crate) enr_tx: mpsc::Sender<Enr>,
    #[expect(dead_code, reason = "held so the discovery handler's sender stays usable")]
    pub(crate) discovery_rx: mpsc::Receiver<HandlerRequest>,
}

/// Builds a [`TestActor`] against `signer`, with `engine_client` standing in for the engine actor.
pub(crate) async fn test_actor(
    signer: &MockSigner,
    engine_client: MockNetworkEngineClient,
) -> TestActor {
    test_actor_with_unsafe_signer(signer, engine_client, signer.address).await
}

/// As [`test_actor`], but with the chain's unsafe block signer set to `unsafe_block_signer`.
///
/// Passing anything other than the mock's own address reproduces a signer key rotation: the
/// remote signer rejects the request locally, before any transport work.
pub(crate) async fn test_actor_with_unsafe_signer(
    signer: &MockSigner,
    engine_client: MockNetworkEngineClient,
    unsafe_block_signer: Address,
) -> TestActor {
    let remote = RemoteSigner {
        endpoint: signer.url.clone(),
        address: signer.address,
        client_cert: None,
        ca_cert: None,
        headers: Default::default(),
    };

    let gossip_addr: Multiaddr =
        "/ip4/127.0.0.1/tcp/0".parse().expect("gossip multiaddr is well formed");
    let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
        unreachable!("generate_secp256k1 returns a secp256k1 key")
    };

    let driver = NetworkBuilder::new(
        RollupConfig { l2_chain_id: Chain::from_id(CHAIN_ID), ..Default::default() },
        // Presented as `sender_address`.
        unsafe_block_signer,
        gossip_addr,
        Keypair::generate_secp256k1(),
        LocalNode::new(secret_key, IpAddr::V4(Ipv4Addr::LOCALHOST), 0, 0),
        ConfigBuilder::new(ListenConfig::from_ip(IpAddr::V4(Ipv4Addr::LOCALHOST), 0)).build(),
        Some(BlockSigner::Remote(remote)),
    )
    // Explicit, not incidental: `tokio::time::interval`'s first tick completes immediately, so a
    // configured monitor would make the peer-score arm ready on the first poll and race the
    // publish arm in `select!`.
    .with_peer_monitoring(None)
    .build()
    .expect("network builder failed");

    // `build()` only, never `start()`. This mirrors the field list of `NetworkDriver::start`;
    // if that gains a step, this fixture has to follow.
    let signer_handler = match driver.signer {
        Some(s) => Some(s.start().await.expect("mock signer refused the health check")),
        None => None,
    };

    let (discovery_tx, discovery_rx) = mpsc::channel::<HandlerRequest>(16);
    let (enr_tx, enr_receiver) = mpsc::channel::<Enr>(16);

    let handler = NetworkHandler {
        gossip: driver.gossip,
        discovery: Discv5Handler::new(CHAIN_ID, discovery_tx),
        enr_receiver,
        unsafe_block_signer_sender: driver.unsafe_block_signer_sender,
        peer_score_inspector: tokio::time::interval(*PEER_SCORE_INSPECT_FREQUENCY),
        signer: signer_handler,
    };

    let (unsafe_block_signer_tx, unsafe_block_signer_rx) = mpsc::channel::<Address>(16);
    let (p2p_rpc_tx, p2p_rpc_rx) = mpsc::channel::<P2pRpcRequest>(16);
    let (admin_tx, admin_rx) = mpsc::channel::<NetworkAdminQuery>(16);
    let (publish_tx, publish_rx) = mpsc::channel::<OpExecutionPayloadEnvelope>(16);

    TestActor {
        actor: NetworkActor::new(
            engine_client,
            handler,
            unsafe_block_signer_rx,
            p2p_rpc_rx,
            admin_rx,
            publish_rx,
        ),
        publish_tx,
        admin_tx,
        unsafe_block_signer_tx,
        p2p_rpc_tx,
        enr_tx,
        discovery_rx,
    }
}
