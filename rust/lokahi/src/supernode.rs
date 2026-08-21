//! The N-chain host: one actor group per configured chain, one L1 watcher, one process.

use crate::config::{L1Settings, ResolvedChain, ResolvedConfig};
use alloy_primitives::Address;
use alloy_provider::RootProvider;
use alloy_rpc_types_engine::JwtSecret;
use anyhow::{Context, Result, anyhow, bail};
use discv5::enr::k256;
use kona_disc::LocalNode;
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_node_service::{
    BoxedNodeActor, ComposedChain, EngineConfig, IntoBoxedNodeActor, L1Config, L1ConfigBuilder,
    L1WatcherPorts, NetworkConfig, RollupNodeBuilder, label_chain, run_actors,
};
use kona_peers::{BootNode, BootStoreFile};
use kona_providers_alloy::OnlineBeaconClient;
use kona_rpc::RpcBuilder;
use libp2p::identity::Keypair;
use serde_json::{Value, from_reader, from_value};
use std::{fs::File, net::SocketAddr, path::Path, sync::Arc};
use tokio_util::sync::CancellationToken;
use tracing::info;

/// The name of the P2P secret key file inside a chain's data directory.
const P2P_KEY_FILE: &str = "p2p_priv_key";

/// The name of the bootstore file inside a chain's data directory.
const BOOTSTORE_FILE: &str = "bootstore.json";

/// The name of the admin-API persistence file inside a chain's data directory.
const ADMIN_STATE_FILE: &str = "admin_state.json";

/// The supernode: the chains it runs, and the L1 they all derive from.
///
/// The chain set is fixed at construction. Growing it would mean starting another chain's actors
/// after the process has begun answering questions about cross-chain safety over the set it has,
/// so there is no method that adds one — [`run`](Self::run) consumes the supernode instead.
#[derive(Debug)]
pub(crate) struct Supernode {
    /// The L1 settings shared by every chain.
    l1: L1Settings,
    /// The L1 chain config every chain derives against.
    l1_chain_config: L1ChainConfig,
    /// The chains, with their rollup configs loaded. Fixed: a boxed slice cannot grow.
    chains: Box<[Chain]>,
}

/// One configured chain, with the files its settings named already loaded.
#[derive(Debug)]
struct Chain {
    /// The chain's resolved settings.
    settings: ResolvedChain,
    /// The chain's rollup config.
    rollup_config: RollupConfig,
    /// The chain's interop dependency set, when one is configured.
    dependency_set: Option<Arc<DependencySet>>,
}

impl Supernode {
    /// Loads every chain's rollup config and dependency set, and the L1 chain config.
    ///
    /// This is where a misconfiguration still becomes a startup error rather than a running node
    /// with one broken chain: the whole set is loaded and checked before any actor exists.
    pub(crate) fn load(config: ResolvedConfig) -> Result<Self> {
        let ResolvedConfig { l1, chains } = config;

        let chains = chains
            .into_vec()
            .into_iter()
            .map(Chain::load)
            .collect::<Result<Vec<_>>>()?
            .into_boxed_slice();

        // One L1 watcher serves every chain, so the chains must agree on which L1 that is. Chains
        // on different L1s in one process would each be served the other's L1 blocks.
        let (first, rest) = chains.split_first().expect("resolution rejects an empty chain set");
        let l1_chain_id = first.rollup_config.l1_chain_id;
        if let Some(other) = rest.iter().find(|c| c.rollup_config.l1_chain_id != l1_chain_id) {
            bail!(
                "chain {} derives from L1 chain {l1_chain_id} but chain {} derives from L1 chain {}: \
                 one supernode follows a single L1",
                first.settings.l2_chain_id,
                other.settings.l2_chain_id,
                other.rollup_config.l1_chain_id,
            );
        }

        let l1_chain_config = load_l1_chain_config(&l1, l1_chain_id)?;

        Ok(Self { l1, l1_chain_config, chains })
    }

    /// Runs every chain until one of its actors exits or the process is signalled.
    ///
    /// Each chain goes through [`RollupNode::compose`], the same entry point a single-chain
    /// kona-node uses, so the per-chain wiring here is not a second copy of it. The composed
    /// actors of all chains are then run as one set by [`run_actors`], which is also what makes
    /// the failure behaviour process-wide: a fatal error on any chain stops the supernode rather
    /// than leaving it serving N-1 chains and silently answering for the dead one. Transient
    /// conditions — an execution layer that is down, an L1 RPC that times out — are retried inside
    /// the actors and do not reach here.
    ///
    /// [`RollupNode::compose`]: kona_node_service::RollupNode::compose
    pub(crate) async fn run(self) -> Result<()> {
        let Self { l1, l1_chain_config, chains } = self;

        info!(
            target: "lokahi",
            chains = chains.len(),
            l1_chain_id = l1_chain_config.chain_id,
            "Starting supernode"
        );

        let mut actors: Vec<BoxedNodeActor> = Vec::new();
        let mut watcher_ports: Vec<L1WatcherPorts> = Vec::with_capacity(chains.len());

        for chain in chains {
            let chain_id = chain.settings.l2_chain_id;
            let ComposedChain { actors: chain_actors, l1_watcher_ports } = chain
                .compose(&l1, &l1_chain_config)
                .await
                .with_context(|| format!("failed to compose the actors of chain {chain_id}"))?;

            info!(
                target: "lokahi",
                chain_id,
                actors = chain_actors.len(),
                "Composed chain"
            );

            // Labelled as they are collected: past this point the actors of every chain are one
            // set, and a fatal error's `Debug` rendering does not say which chain it came from.
            actors.extend(label_chain(chain_id, chain_actors));
            watcher_ports.push(l1_watcher_ports);
        }

        // One watcher for the whole process: the L1 head and finalized streams are polled once and
        // fanned out to every chain, rather than every chain polling the same L1 for itself.
        let l1_config = L1Config {
            chain_config: Arc::new(l1_chain_config),
            trust_rpc: l1.trust_rpc,
            beacon_client: beacon_client(&l1),
            engine_provider: RootProvider::new_http(l1.eth_rpc.clone()),
        };
        actors.push(l1_config.l1_watcher(watcher_ports).map_err(|e| anyhow!(e))?.boxed());

        run_actors(CancellationToken::new(), actors).await.map_err(|e| anyhow!(e))
    }
}

impl Chain {
    /// Loads the files this chain's settings name.
    fn load(settings: ResolvedChain) -> Result<Self> {
        let rollup_config = load_rollup_config(&settings)?;
        let dependency_set = load_dependency_set(&settings, &rollup_config)?;
        Ok(Self { settings, rollup_config, dependency_set })
    }

    /// Builds this chain's node and composes its actor group.
    async fn compose(
        self,
        l1: &L1Settings,
        l1_chain_config: &L1ChainConfig,
    ) -> Result<ComposedChain> {
        let Self { settings, rollup_config, dependency_set } = self;

        // Every chain keeps its state in its own directory: its P2P identity, its bootstore, and
        // the admin-API state its RPC server persists.
        std::fs::create_dir_all(&settings.datadir).with_context(|| {
            format!("failed to create the data directory {}", settings.datadir.display())
        })?;

        let jwt_secret = load_jwt_secret(&settings.jwt_secret)?;

        let engine_config = EngineConfig {
            config: Arc::new(rollup_config.clone()),
            l2_url: settings.engine_rpc.clone(),
            l2_jwt_secret: jwt_secret,
            l1_url: l1.eth_rpc.clone(),
            mode: settings.mode,
        };

        let l1_config_builder = L1ConfigBuilder {
            chain_config: l1_chain_config.clone(),
            trust_rpc: l1.trust_rpc,
            beacon: l1.beacon.clone(),
            rpc_url: l1.eth_rpc.clone(),
            slot_duration_override: l1.slot_duration_override,
        };

        let p2p_config = network_config(&settings, &rollup_config)?;

        info!(
            target: "lokahi",
            chain_id = settings.l2_chain_id,
            mode = %settings.mode,
            rpc = %settings.rpc_socket,
            datadir = %settings.datadir.display(),
            "Configured chain"
        );

        RollupNodeBuilder::new(
            rollup_config,
            l1_config_builder,
            settings.trust_l2_rpc,
            engine_config,
            p2p_config,
            Some(rpc_builder(&settings)),
        )
        .with_dependency_set(dependency_set)
        .build()
        .compose()
        .await
        .map_err(|e| anyhow!(e))
    }
}

/// The RPC server configuration for one chain.
///
/// Each chain answers on its own socket, so a caller addresses a chain by the port it asks on and
/// every chain keeps the method names a single-chain node has. The admin API's persisted state
/// lives in the chain's own data directory for the same reason.
fn rpc_builder(settings: &ResolvedChain) -> RpcBuilder {
    RpcBuilder {
        no_restart: false,
        socket: settings.rpc_socket,
        enable_admin: settings.rpc_enable_admin,
        admin_persistence: Some(settings.datadir.join(ADMIN_STATE_FILE)),
        ws_enabled: false,
        dev_enabled: false,
    }
}

/// Builds one chain's P2P configuration.
///
/// The chain's identity and its bootstore live in its own data directory, so two chains in one
/// process do not share a peer id or overwrite each other's known peers.
fn network_config(settings: &ResolvedChain, rollup_config: &RollupConfig) -> Result<NetworkConfig> {
    let keypair =
        kona_cli::SecretKeyLoader::load(&settings.datadir.join(P2P_KEY_FILE)).map_err(|e| {
            anyhow!("failed to load the p2p key of chain {}: {e}", settings.l2_chain_id)
        })?;
    let local_node = local_node(&keypair, settings)?;

    let mut gossip_address = libp2p::Multiaddr::from(settings.p2p_listen_ip);
    gossip_address.push(libp2p::multiaddr::Protocol::Tcp(settings.p2p_tcp_port));

    let unsafe_block_signer = unsafe_block_signer(settings)?;

    let bootnodes = settings
        .bootnodes
        .iter()
        .map(|bootnode| BootNode::parse_bootnode(bootnode))
        .collect::<Vec<BootNode>>()
        .into();

    // Discovery listens on the chain's UDP port; the defaults `NetworkConfig::new` picks are
    // derived from the advertised record, which is the TCP port.
    let discovery_listen = SocketAddr::new(settings.p2p_listen_ip, settings.p2p_udp_port);

    Ok(NetworkConfig {
        discovery_config: NetworkConfig::discv5_config(discovery_listen.into(), false),
        bootstore: Some(BootStoreFile::Custom(settings.datadir.join(BOOTSTORE_FILE))),
        bootnodes,
        keypair,
        // `NetworkConfig::new` leaves this at `libp2p::gossipsub::Config::default()`, whose
        // `ValidationMode::Strict` is rejected outright by the `MessageAuthenticity::Anonymous`
        // the gossip behaviour is built with, so every chain fails to compose. kona-node's CLI
        // never hits that because it always overrides this field; take the same defaults it
        // starts from, including the per-chain message-id label.
        gossip_config: kona_gossip::default_config(settings.l2_chain_id),
        ..NetworkConfig::new(rollup_config.clone(), local_node, gossip_address, unsafe_block_signer)
    })
}

/// Reads the chain's unsafe block signer from its settings, falling back to the registry.
fn unsafe_block_signer(settings: &ResolvedChain) -> Result<Address> {
    if let Some(signer) = settings.unsafe_block_signer {
        return Ok(signer);
    }

    kona_registry::OPCHAINS
        .get(&settings.l2_chain_id)
        .and_then(|chain| chain.roles.as_ref())
        .and_then(|roles| roles.unsafe_block_signer)
        .ok_or_else(|| {
            anyhow!(
                "chain {} has no unsafe-block-signer configured and no registry entry to take one \
                 from",
                settings.l2_chain_id
            )
        })
}

/// Builds the advertised node record for one chain from its P2P keypair.
fn local_node(keypair: &Keypair, settings: &ResolvedChain) -> Result<LocalNode> {
    let secret = keypair
        .clone()
        .try_into_secp256k1()
        .map_err(|e| {
            anyhow!("the p2p keypair of chain {} is not secp256k1: {e}", settings.l2_chain_id)
        })?
        .secret()
        .to_bytes();
    let signing_key = k256::ecdsa::SigningKey::from_bytes(&secret.into()).map_err(|e| {
        anyhow!("the p2p key of chain {} is not a valid signing key: {e}", settings.l2_chain_id)
    })?;

    Ok(LocalNode::new(
        signing_key,
        settings.p2p_listen_ip,
        settings.p2p_tcp_port,
        settings.p2p_udp_port,
    ))
}

/// Reads a chain's rollup config from the file its settings name, or from the superchain registry.
fn load_rollup_config(settings: &ResolvedChain) -> Result<RollupConfig> {
    match &settings.rollup_config {
        Some(path) => {
            let file = File::open(path).with_context(|| {
                format!(
                    "failed to open the rollup config {} of chain {}",
                    path.display(),
                    settings.l2_chain_id
                )
            })?;
            let config: RollupConfig = from_reader(file)
                .with_context(|| format!("failed to parse the rollup config {}", path.display()))?;
            if config.l2_chain_id.id() != settings.l2_chain_id {
                bail!(
                    "the rollup config {} is for chain {} but is configured under chain {}",
                    path.display(),
                    config.l2_chain_id.id(),
                    settings.l2_chain_id
                );
            }
            Ok(config)
        }
        None => {
            kona_registry::ROLLUP_CONFIGS.get(&settings.l2_chain_id).cloned().ok_or_else(|| {
                anyhow!(
                    "chain {} is not in the superchain registry: set its rollup-config",
                    settings.l2_chain_id
                )
            })
        }
    }
}

/// Reads the L1 chain config from the file the L1 settings name, or from the registry.
fn load_l1_chain_config(l1: &L1Settings, l1_chain_id: u64) -> Result<L1ChainConfig> {
    match &l1.chain_config {
        Some(path) => {
            let file = File::open(path).with_context(|| {
                format!("failed to open the l1 chain config {}", path.display())
            })?;
            parse_l1_chain_config(file)
                .with_context(|| format!("failed to parse the l1 chain config {}", path.display()))
        }
        None => kona_registry::L1Config::get_l1_genesis(l1_chain_id)
            .map(Into::into)
            .map_err(|e| anyhow!("no known l1 chain config for chain {l1_chain_id}: {e}")),
    }
}

/// Parses an L1 chain config from either a chain config document or a genesis document.
fn parse_l1_chain_config(reader: impl std::io::Read) -> serde_json::Result<L1ChainConfig> {
    let mut value: Value = from_reader(reader)?;
    if let Value::Object(object) = &mut value &&
        let Some(config) = object.remove("config")
    {
        return from_value(config);
    }
    from_value(value)
}

/// Reads a chain's interop dependency set, and requires one once the chain schedules interop.
///
/// Same rule as kona-node: the attributes builder panics on an interop-scheduled chain without a
/// dependency set, so this turns that panic into a startup error.
fn load_dependency_set(
    settings: &ResolvedChain,
    rollup_config: &RollupConfig,
) -> Result<Option<Arc<DependencySet>>> {
    match &settings.interop_dependency_set {
        Some(path) => {
            let file = File::open(path).with_context(|| {
                format!(
                    "failed to open the dependency set {} of chain {}",
                    path.display(),
                    settings.l2_chain_id
                )
            })?;
            let dependency_set = from_reader(file).with_context(|| {
                format!("failed to parse the dependency set {}", path.display())
            })?;
            Ok(Some(Arc::new(dependency_set)))
        }
        None if rollup_config.hardforks.lagoon_time.is_some() => bail!(
            "chain {} schedules Lagoon at {:?} but has no interop-dependency-set",
            settings.l2_chain_id,
            rollup_config.hardforks.lagoon_time,
        ),
        None => Ok(None),
    }
}

/// Reads an engine API JWT secret from a hex file.
fn load_jwt_secret(path: &Path) -> Result<JwtSecret> {
    let hex = std::fs::read_to_string(path)
        .with_context(|| format!("failed to read the jwt secret {}", path.display()))?;
    JwtSecret::from_hex(hex.trim())
        .map_err(|e| anyhow!("failed to parse the jwt secret {}: {e}", path.display()))
}

/// Builds the shared L1 beacon client.
fn beacon_client(l1: &L1Settings) -> OnlineBeaconClient {
    let client = OnlineBeaconClient::new_http(l1.beacon.to_string());
    match l1.slot_duration_override {
        Some(duration) => client.with_l1_slot_duration_override(duration),
        None => client,
    }
}
