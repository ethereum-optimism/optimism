//! The N-chain host: one actor group per configured chain, one L1 watcher, one process.

use crate::{
    admin,
    config::{L1Settings, ResolvedChain, ResolvedConfig, SequencerSettings},
    interop::{ChainInterop, HostedChain, InteropActor, InteropTestHandle},
    query::{QueryChain, QueryHandle},
};
use alloy_primitives::{Address, B256};
use alloy_provider::RootProvider;
use alloy_rpc_types_engine::JwtSecret;
use alloy_signer::Signer;
use alloy_signer_local::PrivateKeySigner;
use anyhow::{Context, Result, anyhow, bail};
use discv5::enr::k256;
use kona_disc::LocalNode;
use kona_engine::OpEngineClient;
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_node_service::{
    BoxedNodeActor, ComposedChain, EngineConfig, IntoBoxedNodeActor, L1Config, L1ConfigBuilder,
    L1WatcherPorts, NetworkConfig, QueuedEngineRpcClient, RollupNodeBuilder, SequencerConfig,
    label_chain, run_actors,
};
use kona_peers::{BootNode, BootStoreFile};
use kona_providers_alloy::OnlineBeaconClient;
use kona_rpc::RpcBuilder;
use kona_safedb::SharedSafeDb;
use kona_sources::BlockSigner;
use libp2p::identity::Keypair;
use op_alloy_network::Optimism;
use serde_json::{Value, from_reader, from_value};
use std::{
    fs::File,
    net::SocketAddr,
    path::{Path, PathBuf},
    sync::Arc,
};
use tokio_util::sync::CancellationToken;
use tracing::{info, warn};

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
    chains: Box<[OpenChain]>,
    /// The supernode-level admin RPC's socket, if one is configured.
    admin_rpc: Option<SocketAddr>,
    /// The chains as configured, kept for the admin RPC to answer from.
    configured: Box<[ResolvedChain]>,
    /// The directory the process-wide interop stores live under, when the configuration names
    /// one.
    interop_datadir: Option<PathBuf>,
    /// The interop activation timestamp the chain set shares, or [`None`] when interop is off.
    interop_activation: Option<u64>,
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

/// A chain whose on-disk state is open: the phase after [`Chain::load`], and the only form
/// [`OpenChain::compose`] exists on.
///
/// A separate type rather than more [`Option`]s on [`Chain`], because "not opened yet" and
/// "interop is off for this chain" are different states that a single `Option` would spell the
/// same way — and the safe-head database is now open in both of the latter's cases.
#[derive(Debug)]
struct OpenChain {
    /// The chain's resolved settings.
    settings: ResolvedChain,
    /// The chain's rollup config.
    rollup_config: RollupConfig,
    /// The chain's interop dependency set, when one is configured.
    dependency_set: Option<Arc<DependencySet>>,
    /// The safe-head database this chain's controller records local-safe advances into.
    ///
    /// Open on every chain, whether or not interop is scheduled, because two readers need it and
    /// only one of them is the interop verifier. The other is `superroot_atTimestamp`, which pairs
    /// a block behind the local-safe head with the L1 block that made it safe — and
    /// pre-activation that pairing *is* the super root, since the optimistic outputs are the
    /// canonical ones there. op-supernode has it unconditionally for the same reason: its chain
    /// container sets `SafeDBPath` on every virtual node it starts, with no interop gate
    /// (`op-supernode/supernode/chain_container/chain_container.go:236`).
    safe_db: SharedSafeDb,
    /// What interop needs from this chain beyond the safe-head history, present exactly when
    /// interop is on.
    interop: Option<ChainInteropState>,
}

/// The per-chain state the interop verifier needs, alongside the chain's own actors.
#[derive(Debug, Clone)]
struct ChainInteropState {
    /// A read-only execution-layer provider over the chain's engine endpoint.
    ///
    /// The engine endpoint is JWT-authenticated, so this is built through kona's own
    /// authenticated-client helper rather than as a plain HTTP provider: an unauthenticated
    /// request to that port is rejected, and the chain has no second, unauthenticated endpoint
    /// configured.
    el: RootProvider<Optimism>,
}

impl Supernode {
    /// Loads every chain's rollup config and dependency set, and the L1 chain config.
    ///
    /// This is where a misconfiguration still becomes a startup error rather than a running node
    /// with one broken chain: the whole set is loaded and checked before any actor exists.
    pub(crate) fn load(config: ResolvedConfig) -> Result<Self> {
        let ResolvedConfig { l1, chains, admin_rpc, interop_datadir, interop_activation_override } =
            config;

        let configured = chains.clone();
        let loaded = chains.into_vec().into_iter().map(Chain::load).collect::<Result<Vec<_>>>()?;

        // Whether interop is on is decided over the whole loaded set, before any actor exists,
        // so a set that cannot form one cluster fails to start rather than starting as a
        // supernode that verifies nothing.
        let interop_activation = InteropActor::activation(
            &loaded
                .iter()
                .map(|chain| HostedChain {
                    chain_id: chain.settings.l2_chain_id,
                    rollup_config: &chain.rollup_config,
                })
                .collect::<Vec<_>>(),
            interop_activation_override,
        )?;

        // A chain in the interop cluster needs the dependency set the verifier reads messages
        // against. `load_dependency_set` requires one from the fork alone, which an overridden
        // activation does not go through, so the requirement is restated here -- the one place
        // that knows interop is on for the whole set rather than for one chain's config.
        if interop_activation.is_some() &&
            let Some(chain) = loaded.iter().find(|chain| chain.dependency_set.is_none())
        {
            bail!(
                "chain {} has no interop-dependency-set, and interop activates at {:?} for the \
                 whole hosted set: the verifier reads every chain's messages against it",
                chain.settings.l2_chain_id,
                interop_activation,
            );
        }

        // One L1 watcher serves every chain, so the chains must agree on which L1 that is. Chains
        // on different L1s in one process would each be served the other's L1 blocks.
        let (first, rest) = loaded.split_first().expect("resolution rejects an empty chain set");
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

        // Last, because this is the first thing in `load` that touches the disk: a chain's data
        // directory and its safe-head database are created only once the whole set is known to be
        // runnable. Attached to the chain rather than opened later by the interop module, so that
        // the controller which records into a database and the readers of it — the verifier, and
        // the super-root query — are handed the same handle by construction.
        let chains = loaded
            .into_iter()
            .map(|chain| chain.open(interop_activation.is_some()))
            .collect::<Result<Vec<_>>>()?
            .into_boxed_slice();

        Ok(Self {
            l1,
            l1_chain_config,
            chains,
            admin_rpc,
            configured,
            interop_datadir,
            interop_activation,
        })
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
        let Self {
            l1,
            l1_chain_config,
            chains,
            admin_rpc,
            configured,
            interop_datadir,
            interop_activation,
        } = self;

        info!(
            target: "lokahi",
            chains = chains.len(),
            l1_chain_id = l1_chain_config.chain_id,
            "Starting supernode"
        );

        // Bound before the chains are composed, so the address it logs is available to a caller
        // that launched this process and has to wait for something. Composing a chain reaches out
        // to the L1 and to an execution layer and can take a while, or fail; a caller watching for
        // one line either has an admin RPC to ask or has a process that exited.
        //
        // Held until `run` returns: dropping the handle stops the server, so the admin RPC is up
        // for exactly as long as the supernode is.
        //
        // The query API is registered on this server with nothing behind it yet, and the handle is
        // filled in once every chain is composed. That ordering is what lets the address be logged
        // before composition without the API ever answering for a chain set that does not exist.
        let queries = QueryHandle::default();
        let interop_test = InteropTestHandle::default();
        let _admin = match admin_rpc {
            Some(socket) => Some(
                admin::serve(socket, &configured, queries.clone(), interop_test.clone()).await?,
            ),
            None => None,
        };

        let mut actors: Vec<BoxedNodeActor> = Vec::new();
        let mut watcher_ports: Vec<L1WatcherPorts> = Vec::with_capacity(chains.len());
        let mut interop_chains: Vec<ChainInterop> = Vec::with_capacity(chains.len());
        let mut query_chains: Vec<QueryChain> = Vec::with_capacity(chains.len());

        for chain in chains {
            let chain_id = chain.settings.l2_chain_id;
            let datadir = chain.settings.datadir.clone();
            let rollup_config = chain.rollup_config.clone();
            let interop_state = chain.interop.clone();
            let safe_db = chain.safe_db.clone();
            let ComposedChain {
                actors: chain_actors,
                l1_watcher_ports,
                controller_request_tx,
                controller_rpc_request_tx,
                l1_query_tx,
                cross_safe_promoter,
            } = chain
                .compose(&l1, &l1_chain_config)
                .await
                .with_context(|| format!("failed to compose the actors of chain {chain_id}"))?;

            // Every chain is queryable, whether or not interop is on, and both methods answer
            // from the same places in both regimes: `supernode_syncStatus` aggregates the set, and
            // `superroot_atTimestamp` reads this chain's safe-head history. Handing it that
            // history unconditionally is what lets it answer before interop activates, where the
            // verifier reports interop inactive, the optimistic outputs are the canonical ones,
            // and the super root is composed from them and the L1 blocks recorded here.
            query_chains.push(QueryChain::new(
                chain_id,
                Arc::new(rollup_config.clone()),
                QueuedEngineRpcClient::new(controller_rpc_request_tx.clone()),
                l1_query_tx,
                safe_db.clone(),
            ));

            // A promoter exists exactly when the chain was composed with an externally fed
            // cross-safe head, which is exactly when interop is on. Taking it here is what makes
            // the interop actor the chain's only cross-safe writer.
            if let (Some(promoter), Some(state)) = (cross_safe_promoter, interop_state) {
                interop_chains.push(ChainInterop {
                    chain_id,
                    datadir,
                    rollup_config,
                    safe_db,
                    el: state.el,
                    queries: controller_rpc_request_tx,
                    requests: controller_request_tx,
                    promoter,
                });
            }

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

        // One verifier for the whole process, in the same actor set as the chains it verifies: a
        // verifier that halts stops the supernode, rather than leaving it serving chains whose
        // cross-safe heads have silently stopped moving.
        //
        // The read handle for the query API is taken from the actor before it is boxed: the
        // verified store lives inside the actor, and rocksdb would refuse a second opener even if
        // the types allowed one, so the only way to read it is to ask the actor that owns it.
        let mut interop_reader = None;
        if let Some(activation) = interop_activation {
            let mut interop = InteropActor::build(
                interop_datadir.as_deref(),
                &l1.eth_rpc,
                activation,
                interop_chains,
            )?;
            let reader = interop.attach_queries(activation);
            // The test-control API reads and drives the verifier through the same queue the query
            // API reads it through, so a pause it sets and a frontier the query API reports cannot
            // disagree: both are one turn of the actor's loop.
            interop_test.attach(reader.clone());
            interop_reader = Some(reader);
            actors.push(interop.boxed());
        }

        // Past this point every chain exists, so the query API can answer for the set.
        queries.compose(query_chains, interop_reader);

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

    /// Opens this chain's on-disk state, and the interop state on top of it when interop is on.
    ///
    /// The safe-head database is opened either way. It exists so that a block behind the local-safe
    /// head can be paired with the L1 block that made it safe, and both readers of that pairing
    /// need it: the interop verifier, which only exists once interop is scheduled, and
    /// `superroot_atTimestamp`, which is asked about pre-interop timestamps by the proposer of a
    /// chain set that schedules no interop at all. Without it that query has no history to read
    /// and fails for every timestamp behind the head — where op-supernode answers from pre-interop
    /// consensus, because its chain container gives every virtual node a `SafeDBPath` with no
    /// interop gate (`op-supernode/supernode/chain_container/chain_container.go:236`).
    ///
    /// `enabled` therefore decides only what interop needs *in addition*: the authenticated
    /// execution-layer provider the verifier reads receipts through.
    fn open(self, enabled: bool) -> Result<OpenChain> {
        let Self { settings, rollup_config, dependency_set } = self;

        // The database lives in the chain's own directory, which composition creates; create it
        // here too, because this runs first.
        std::fs::create_dir_all(&settings.datadir).with_context(|| {
            format!("failed to create the data directory {}", settings.datadir.display())
        })?;
        let safe_db = ChainInterop::open_safe_db(&settings.datadir)?;

        let interop = if enabled {
            let jwt_secret = load_jwt_secret(&settings.jwt_secret)?;
            let el = OpEngineClient::<RootProvider, RootProvider<Optimism>>::rpc_client::<Optimism>(
                settings.engine_rpc.clone(),
                jwt_secret,
            );
            Some(ChainInteropState { el })
        } else {
            None
        };

        Ok(OpenChain { settings, rollup_config, dependency_set, safe_db, interop })
    }
}

impl OpenChain {
    /// Builds this chain's node and composes its actor group.
    async fn compose(
        self,
        l1: &L1Settings,
        l1_chain_config: &L1ChainConfig,
    ) -> Result<ComposedChain> {
        let Self { settings, rollup_config, dependency_set, safe_db, interop } = self;
        let external_cross_safe = interop.is_some();

        // Every chain keeps its state in its own directory: its P2P identity, its bootstore, and
        // the admin-API state its RPC server persists.
        std::fs::create_dir_all(&settings.datadir).with_context(|| {
            format!("failed to create the data directory {}", settings.datadir.display())
        })?;

        let jwt_secret = load_jwt_secret(&settings.jwt_secret)?;

        // A sequencing chain's key is read here rather than while the configuration is resolved:
        // resolution touches no files, so a key that is missing or malformed is a startup error
        // raised where the JWT secret and the rollup config are read.
        let sequencer = settings
            .sequencer
            .as_ref()
            .map(|sequencer| SequencerSigner::load(settings.l2_chain_id, sequencer))
            .transpose()?;
        let sequencer_config = settings.sequencer.as_ref().map(SequencerConfig::from);

        let engine_config = EngineConfig {
            config: Arc::new(rollup_config.clone()),
            l2_url: settings.engine_rpc.clone(),
            l2_jwt_secret: jwt_secret,
            l1_url: l1.eth_rpc.clone(),
            mode: settings.mode(),
        };

        let l1_config_builder = L1ConfigBuilder {
            chain_config: l1_chain_config.clone(),
            trust_rpc: l1.trust_rpc,
            beacon: l1.beacon.clone(),
            rpc_url: l1.eth_rpc.clone(),
            slot_duration_override: l1.slot_duration_override,
        };

        let p2p_config = network_config(&settings, &rollup_config, sequencer)?;

        info!(
            target: "lokahi",
            chain_id = settings.l2_chain_id,
            mode = %settings.mode(),
            rpc = %settings.rpc_socket,
            datadir = %settings.datadir.display(),
            "Configured chain"
        );

        let builder = RollupNodeBuilder::new(
            rollup_config,
            l1_config_builder,
            settings.trust_l2_rpc,
            engine_config,
            p2p_config,
            Some(rpc_builder(&settings)),
        )
        .with_dependency_set(dependency_set)
        .with_external_cross_safe(external_cross_safe)
        .with_safe_db(safe_db);

        // Left unset on a chain this supernode only validates. The builder then falls back to
        // `SequencerConfig::default()`, which is only ever read by the sequencer actor — and a
        // validator composes none.
        let builder = match sequencer_config {
            Some(config) => builder.with_sequencer_config(config),
            None => builder,
        };

        builder.build().compose().await.map_err(|e| anyhow!(e))
    }
}

/// The signer of one sequencing chain, and the address its signatures recover to.
///
/// The address is kept alongside the signer so that the chain's configured unsafe block signer can
/// be checked against it: they are two statements about the same key, and lokahi cannot read the
/// second one off the chain's `SystemConfig` the way kona-node does.
#[derive(Debug)]
struct SequencerSigner {
    /// The signer the network actor signs gossiped payloads with.
    signer: BlockSigner,
    /// The address that signer's signatures recover to.
    address: Address,
}

impl SequencerSigner {
    /// Loads a chain's sequencer key from the file its settings name, generating one when no file
    /// is there yet.
    ///
    /// Load-or-generate is what a key named by a path does across the stack: this goes through
    /// kona's [`kona_cli::SecretKeyLoader`], the loader behind kona-node's own
    /// `--p2p.sequencer.key.path`, and the Go stack writes a key the same way at
    /// `--p2p.priv.path` (`loadNetworkPrivKey`, `op-node/p2p/cli/load_config.go`). The one thing
    /// `SecretKeyLoader` leaves out is restricting what it writes, so a key it creates is
    /// tightened to owner-only afterwards — the mode the Go loader creates its key with. An
    /// existing file's mode is left as it is, which is what the Go loader does too.
    ///
    /// Nothing here can tell a generated key from the one the network expects: the only signal a
    /// mistyped path produces is the unsafe-block-signer comparison in [`network_config`], which
    /// warns and lets startup continue.
    fn load(chain_id: u64, sequencer: &SequencerSettings) -> Result<Self> {
        let path = sequencer.key_path.as_path();
        let existed = path.is_file();

        let keypair = kona_cli::SecretKeyLoader::load(path).map_err(|e| {
            anyhow!("failed to load the sequencer key {} of chain {chain_id}: {e}", path.display())
        })?;

        if !existed {
            restrict_to_owner(path).with_context(|| {
                format!(
                    "failed to restrict the new sequencer key {} of chain {chain_id}",
                    path.display()
                )
            })?;
            warn!(
                target: "lokahi",
                chain_id,
                path = %path.display(),
                "No sequencer key at this path: generated a new one. The blocks this chain signs \
                 with it are dropped by its peers until it is the chain's unsafe block signer"
            );
        }

        let secret = keypair
            .try_into_secp256k1()
            .map_err(|_| {
                anyhow!(
                    "the sequencer key {} of chain {chain_id} is not a secp256k1 key",
                    path.display()
                )
            })?
            .secret()
            .to_bytes();
        let signer = PrivateKeySigner::from_bytes(&B256::from_slice(&secret))
            .map_err(|e| {
                anyhow!(
                    "the sequencer key {} of chain {chain_id} is not a valid signing key: {e}",
                    path.display()
                )
            })?
            .with_chain_id(Some(chain_id));

        Ok(Self { address: signer.address(), signer: BlockSigner::Local(signer) })
    }
}

/// Restricts a file to owner read/write, the mode the Go stack's p2p key loader creates a key file
/// with. A no-op where file modes do not apply.
fn restrict_to_owner(path: &Path) -> std::io::Result<()> {
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o600))?;
    }
    #[cfg(not(unix))]
    let _ = path;
    Ok(())
}

impl From<&SequencerSettings> for SequencerConfig {
    fn from(sequencer: &SequencerSettings) -> Self {
        Self {
            sequencer_stopped: sequencer.stopped,
            sequencer_recovery_mode: sequencer.recover,
            conductor_rpc_url: sequencer.conductor_rpc.clone(),
            l1_conf_delay: sequencer.l1_confs,
        }
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
/// process do not share a peer id or overwrite each other's known peers. `sequencer` is the signer
/// of a chain this supernode sequences, and [`None`] on a chain it only validates: without one the
/// network actor has nothing to sign a built block with and gossips none of them.
fn network_config(
    settings: &ResolvedChain,
    rollup_config: &RollupConfig,
    sequencer: Option<SequencerSigner>,
) -> Result<NetworkConfig> {
    let keypair =
        kona_cli::SecretKeyLoader::load(&settings.datadir.join(P2P_KEY_FILE)).map_err(|e| {
            anyhow!("failed to load the p2p key of chain {}: {e}", settings.l2_chain_id)
        })?;
    let local_node = local_node(&keypair, settings)?;

    let mut gossip_address = libp2p::Multiaddr::from(settings.p2p_listen_ip);
    gossip_address.push(libp2p::multiaddr::Protocol::Tcp(settings.p2p_tcp_port));

    let unsafe_block_signer = unsafe_block_signer(settings)?;

    // Two statements about the same key: what this chain signs its blocks with, and whose
    // signature its peers accept. lokahi takes the second from the operator rather than from the
    // chain's `SystemConfig`, so a disagreement is a configuration mistake it can see but cannot
    // resolve — and one whose only other symptom is every peer silently dropping this chain's
    // blocks. Not fatal: a key being rotated into the `SystemConfig` legitimately differs from the
    // address currently recorded there.
    if let Some(sequencer) = &sequencer &&
        sequencer.address != unsafe_block_signer
    {
        warn!(
            target: "lokahi",
            chain_id = settings.l2_chain_id,
            sequencer = %sequencer.address,
            unsafe_block_signer = %unsafe_block_signer,
            "The sequencer key of this chain does not match its unsafe block signer: peers will \
             reject the blocks it gossips unless the signer is rotated"
        );
    }

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
        gossip_signer: sequencer.map(|sequencer| sequencer.signer),
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::interop::SAFE_DB_DIR;
    use alloy_primitives::address;
    use kona_genesis::HardForkConfig;
    use kona_interop::ChainDependency;
    use std::net::{IpAddr, Ipv4Addr};
    use url::Url;

    /// Interop activates here on every chain these tests build.
    const ACTIVATION: u64 = 1_000;

    /// Writes a rollup config naming `l1_chain_id`, scheduling interop when asked to.
    fn rollup_config_file(
        dir: &Path,
        l2_chain_id: u64,
        l1_chain_id: u64,
        interop: bool,
    ) -> PathBuf {
        let config = RollupConfig {
            l1_chain_id,
            l2_chain_id: l2_chain_id.into(),
            block_time: 2,
            hardforks: HardForkConfig {
                lagoon_time: interop.then_some(ACTIVATION),
                ..Default::default()
            },
            ..Default::default()
        };
        let path = dir.join(format!("rollup-{l2_chain_id}.json"));
        std::fs::write(&path, serde_json::to_vec(&config).expect("serialize the rollup config"))
            .expect("write the rollup config");
        path
    }

    /// Writes a dependency set covering `chain_ids`, which an interop-scheduled chain must have.
    fn dependency_set_file(dir: &Path, chain_ids: &[u64]) -> PathBuf {
        let dependency_set = DependencySet {
            dependencies: chain_ids.iter().map(|id| (*id, ChainDependency {})).collect(),
            override_message_expiry_window: None,
        };
        let path = dir.join("depset.json");
        std::fs::write(&path, serde_json::to_vec(&dependency_set).expect("serialize the depset"))
            .expect("write the dependency set");
        path
    }

    /// A resolved chain whose files are written under `dir`, scheduling interop.
    fn chain(dir: &Path, l2_chain_id: u64, l1_chain_id: u64, port: u16) -> ResolvedChain {
        chain_with_interop(dir, l2_chain_id, l1_chain_id, port, true)
    }

    /// A resolved chain that schedules no interop, and so has no dependency set either.
    fn chain_without_interop(
        dir: &Path,
        l2_chain_id: u64,
        l1_chain_id: u64,
        port: u16,
    ) -> ResolvedChain {
        chain_with_interop(dir, l2_chain_id, l1_chain_id, port, false)
    }

    /// A resolved chain whose files are written under `dir`.
    ///
    /// `interop` is one decision, not two: a chain that schedules Lagoon must have a dependency
    /// set — `load_dependency_set` refuses one that does not — and a chain that does not schedule
    /// it has nothing to put in one. Taking both from one flag is also what keeps the rollup config
    /// and the dependency set from being written by two calls that disagree, since both land at the
    /// same path.
    fn chain_with_interop(
        dir: &Path,
        l2_chain_id: u64,
        l1_chain_id: u64,
        port: u16,
        interop: bool,
    ) -> ResolvedChain {
        let jwt = dir.join("jwt.hex");
        std::fs::write(&jwt, format!("0x{}", "11".repeat(32))).expect("write the jwt secret");
        ResolvedChain {
            l2_chain_id,
            rollup_config: Some(rollup_config_file(dir, l2_chain_id, l1_chain_id, interop)),
            engine_rpc: Url::parse("http://127.0.0.1:1/").unwrap(),
            jwt_secret: jwt,
            trust_l2_rpc: false,
            sequencer: None,
            datadir: dir.join(l2_chain_id.to_string()),
            rpc_socket: SocketAddr::new(IpAddr::from(Ipv4Addr::LOCALHOST), port),
            rpc_enable_admin: false,
            p2p_listen_ip: IpAddr::from(Ipv4Addr::LOCALHOST),
            p2p_tcp_port: port + 1,
            p2p_udp_port: port + 1,
            unsafe_block_signer: Some(Address::repeat_byte(1)),
            bootnodes: Vec::new(),
            interop_dependency_set: interop.then(|| dependency_set_file(dir, &[l2_chain_id])),
        }
    }

    /// Sequencer settings naming `key_path`, with kona-node's defaults for everything else.
    fn sequencer_settings(key_path: PathBuf) -> SequencerSettings {
        SequencerSettings {
            key_path,
            stopped: false,
            l1_confs: 4,
            recover: false,
            conductor_rpc: None,
        }
    }

    /// A configuration over `chains`, with its interop state under `dir`.
    fn config(dir: &Path, chains: Vec<ResolvedChain>) -> ResolvedConfig {
        ResolvedConfig {
            l1: L1Settings {
                eth_rpc: Url::parse("http://127.0.0.1:1/").unwrap(),
                beacon: Url::parse("http://127.0.0.1:1/").unwrap(),
                trust_rpc: false,
                chain_config: None,
                slot_duration_override: None,
            },
            chains: chains.into_boxed_slice(),
            admin_rpc: None,
            interop_datadir: Some(dir.join("interop")),
            interop_activation_override: None,
        }
    }

    /// Loading is all checks and then all side effects, in that order.
    ///
    /// A set whose chains derive from different L1s cannot run, and `load` says so. The point of
    /// this test is what it must *not* have done on the way: opening a chain's safe-head database
    /// creates its directory and a rocksdb LOCK, and doing that before the set is known to be
    /// runnable leaves half-initialised state behind for every failed start — state a later run,
    /// or an operator reading the data directory, would take for a chain this node once served.
    #[test]
    fn a_rejected_chain_set_leaves_nothing_on_disk() {
        let dir = tempfile::tempdir().expect("temp dir");
        // 1 and 11155111 are mainnet and sepolia: two chains no supernode can follow at once.
        let chains =
            vec![chain(dir.path(), 901, 1, 9545), chain(dir.path(), 902, 11_155_111, 9555)];

        let err = Supernode::load(config(dir.path(), chains)).expect_err("must be rejected");
        assert!(err.to_string().contains("one supernode follows a single L1"), "{err:?}");

        for chain_id in [901, 902] {
            let datadir = dir.path().join(chain_id.to_string());
            assert!(
                !datadir.join(SAFE_DB_DIR).exists(),
                "chain {chain_id}'s safe-head database must not be created before the chain set is \
                 known to be runnable"
            );
        }
    }

    /// The counterpart, so the test above cannot pass by never opening a database at all: an
    /// agreeing set does open one, in the chain's own directory.
    #[test]
    fn an_accepted_chain_set_opens_each_chain_a_safe_head_database() {
        let dir = tempfile::tempdir().expect("temp dir");
        let chains = vec![chain(dir.path(), 901, 1, 9565), chain(dir.path(), 902, 1, 9575)];

        let supernode = Supernode::load(config(dir.path(), chains)).expect("must load");
        assert_eq!(supernode.interop_activation, Some(ACTIVATION));

        for chain_id in [901, 902] {
            assert!(
                dir.path().join(chain_id.to_string()).join(SAFE_DB_DIR).is_dir(),
                "chain {chain_id}'s safe-head database lives in its own directory"
            );
        }
    }

    /// A chain set that schedules no interop still gets a safe-head database per chain.
    ///
    /// This is what makes `superroot_atTimestamp` answerable before interop activates. The query
    /// pairs the block at a timestamp with the L1 block that made it safe, and for any timestamp
    /// behind the local-safe head that pairing only exists in this database — so a chain set
    /// without one fails the whole call rather than answering from pre-interop consensus, which is
    /// what op-supernode does there. op-supernode has no interop gate on it either: its chain
    /// container sets `SafeDBPath` on every virtual node it starts.
    #[test]
    fn a_chain_set_without_interop_still_opens_a_safe_head_database() {
        let dir = tempfile::tempdir().expect("temp dir");
        let chains = vec![
            chain_without_interop(dir.path(), 901, 1, 9585),
            chain_without_interop(dir.path(), 902, 1, 9595),
        ];

        let supernode = Supernode::load(config(dir.path(), chains)).expect("must load");
        assert_eq!(
            supernode.interop_activation, None,
            "this set schedules no interop, or the test is not testing what it says"
        );

        for chain in &supernode.chains {
            assert!(
                chain.interop.is_none(),
                "chain {} must have no interop state without interop scheduled",
                chain.settings.l2_chain_id
            );
            assert!(
                dir.path().join(chain.settings.l2_chain_id.to_string()).join(SAFE_DB_DIR).is_dir(),
                "chain {} must record safe-head history whether or not interop is scheduled",
                chain.settings.l2_chain_id
            );
        }
    }

    /// A sequencing chain's key is taken from the file its settings name when one is there, and
    /// the address it signs with is the one its peers have to be expecting.
    #[test]
    fn a_sequencer_key_is_read_from_its_file() {
        let dir = tempfile::tempdir().expect("temp dir");
        let path = dir.path().join("sequencer.key");
        std::fs::write(&path, format!("0x{}\n", "00".repeat(31) + "01"))
            .expect("write the sequencer key");

        let sequencer = SequencerSigner::load(901, &sequencer_settings(path)).expect("must load");
        // The address of the secp256k1 key `1`.
        assert_eq!(sequencer.address, address!("0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf"));
    }

    /// A key file that is not there yet is generated rather than refused, the same as every other
    /// key the stack names by a path — and it is written owner-only, the mode the Go
    /// `loadNetworkPrivKey` creates its key with. Generated once: the second load reads back the
    /// key the first one wrote, so a restart keeps the identity its peers have started accepting.
    #[test]
    fn a_missing_sequencer_key_is_generated_owner_only_and_then_reused() {
        let dir = tempfile::tempdir().expect("temp dir");
        let path = dir.path().join("nested").join("absent.key");

        let sequencer =
            SequencerSigner::load(901, &sequencer_settings(path.clone())).expect("must generate");
        assert!(path.is_file(), "a missing sequencer key file is created");

        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            let mode = std::fs::metadata(&path).expect("metadata").permissions().mode() & 0o777;
            assert_eq!(mode, 0o600, "a generated sequencer key is readable only by its owner");
        }

        let reloaded =
            SequencerSigner::load(901, &sequencer_settings(path)).expect("must load what it wrote");
        assert_eq!(reloaded.address, sequencer.address);
    }
}
