//! The layered `lokahi` configuration: global settings plus one overlay per chain.
//!
//! An operator running N chains states what the chains have in common once and only the
//! differences per chain, so a 10-chain supernode is not 10 copies of the same node
//! configuration. [`LokahiConfig`] is the file as written; [`ResolvedConfig`] is what the overlay
//! resolves to, with every per-chain value decided and checked. Resolution is pure — it reads no
//! files and opens no sockets — so the rules below are covered by unit tests rather than by
//! starting a node.
//!
//! The chain set is fixed at startup: it is what the file lists, and there is no path that adds a
//! chain to a running supernode. Interop safety is decided over a known set of chains, so a set
//! that changes underneath the node would change the meaning of an answer it has already given.

use kona_node_service::NodeMode;
use serde::{Deserialize, Deserializer, de::Error as _};
use std::{
    collections::HashMap,
    net::{IpAddr, Ipv4Addr, SocketAddr},
    path::{Path, PathBuf},
};
use url::Url;

/// The `lokahi` configuration file.
///
/// The `[l1]` table and the `[defaults]` table are the global layer; each `[[chains]]` entry is an
/// overlay on `[defaults]`.
#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields, rename_all = "kebab-case")]
pub(crate) struct LokahiConfig {
    /// The L1 every chain in this supernode derives from.
    pub(crate) l1: L1Settings,
    /// The per-chain settings that apply to every chain unless the chain overrides them.
    #[serde(default)]
    pub(crate) defaults: ChainSettings,
    /// The chains this supernode runs, one entry each.
    ///
    /// Defaulted rather than required so that a file listing no chain is reported as "no chains
    /// configured" rather than as a missing field.
    #[serde(default)]
    pub(crate) chains: Vec<ChainSettings>,
    /// The socket the supernode serves its RPC on, when the file asks for one.
    ///
    /// Absent means the supernode exposes no RPC at all: the chains are served on this socket too,
    /// so there is no second place for them to go. A supernode configured this way follows its
    /// chains and gossips, and answers nothing.
    pub(crate) admin: Option<AdminSettings>,
    /// The process-wide interop settings, when the file overrides any of them.
    pub(crate) interop: Option<InteropSettings>,
}

/// The process-wide interop settings.
///
/// Interop is a property of the whole hosted set rather than of one chain -- rounds are lockstep
/// across it -- so this is a single table rather than a per-chain field.
#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields, rename_all = "kebab-case")]
pub(crate) struct InteropSettings {
    /// Activates interop verification at this timestamp instead of deriving it from the chains'
    /// Lagoon time.
    ///
    /// Absent is the normal case and the default everywhere outside a test harness: the
    /// activation is read from each chain's rollup config, which is also where the message rules
    /// the verifier applies read it, so the two cannot disagree.
    ///
    /// Present, it supplies an activation for chains whose rollup config schedules no Lagoon at
    /// all -- which is the only thing it can do that reading the fork cannot. It does not
    /// override a fork that is scheduled: a chain that schedules Lagoon must schedule it at the
    /// same block this names, or the set is refused. See `InteropActor::activation`.
    pub(crate) activation_timestamp: Option<u64>,
}

/// The supernode's RPC socket.
///
/// One socket serves everything: the supernode's own namespaces at `/`, and each hosted chain's
/// node RPC at `/<l2-chain-id>`. So this is not only the admin surface's address — it is the
/// address of the whole process, and the one a caller has to be told.
#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields, rename_all = "kebab-case")]
pub(crate) struct AdminSettings {
    /// The address the supernode listens on. Loopback by default: the admin surface is a control
    /// surface, so it does not become reachable off the host by leaving a field out.
    pub(crate) rpc_addr: Option<IpAddr>,
    /// The port the supernode listens on. `0` lets the OS choose, and the chosen port is logged.
    pub(crate) rpc_port: u16,
}

/// The L1 settings, shared by every chain.
///
/// There is one L1 per supernode rather than one per chain: the L1 is followed once for the whole
/// process, and the chains of an interop cluster share an L1 by definition.
#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields, rename_all = "kebab-case")]
pub(crate) struct L1Settings {
    /// The L1 execution-layer RPC URL.
    pub(crate) eth_rpc: Url,
    /// The L1 beacon API URL.
    pub(crate) beacon: Url,
    /// Whether to trust the L1 RPC's responses without verifying them.
    #[serde(default)]
    pub(crate) trust_rpc: bool,
    /// An L1 chain config or genesis file, overriding the registry entry for the L1 chain id the
    /// chains' rollup configs name.
    pub(crate) chain_config: Option<PathBuf>,
    /// A fixed L1 slot duration, for when the beacon API's configuration is not available.
    pub(crate) slot_duration_override: Option<u64>,
    /// How often, in seconds, to poll the L1 for epoch updates — the finalized-block changes L2
    /// finality is driven from.
    ///
    /// op-node's `--l1.epoch-poll-interval`, shared by every chain because the L1 is followed once
    /// for the whole process. Unset keeps kona's default of one poll a minute, which suits a
    /// production L1; a devnet whose L1 finalizes in seconds sets this to match, or first finality
    /// is only observed up to a minute late.
    pub(crate) epoch_poll_interval: Option<u64>,
}

/// The subdirectory of the default data directory holding the process-wide interop stores.
pub(crate) const INTEROP_DIR: &str = "interop";

/// How many L1 blocks a sequencer keeps away from the L1 head by default when it picks an L1
/// origin.
///
/// kona-node's `--sequencer.l1-confs` default, so a chain sequences the same distance behind the
/// L1 head under a supernode as it does under a single-chain node.
const DEFAULT_SEQUENCER_L1_CONFS: u64 = 4;

/// One chain's settings, as written in the file.
///
/// Every field is optional: the same type is both the `[defaults]` layer and a `[[chains]]`
/// overlay, and which fields a chain must end up with is decided in [`LokahiConfig::resolve`]
/// rather than by the deserializer. That way a missing value is reported as "chain 901 has no
/// engine-rpc" instead of as a parse error against a table the operator did not write.
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(deny_unknown_fields, rename_all = "kebab-case")]
pub(crate) struct ChainSettings {
    /// The L2 chain id. Required per chain; meaningless in `[defaults]`.
    pub(crate) l2_chain_id: Option<u64>,
    /// A rollup config file for this chain, overriding its superchain registry entry.
    pub(crate) rollup_config: Option<PathBuf>,
    /// The chain's authenticated engine API URL.
    pub(crate) engine_rpc: Option<Url>,
    /// The JWT secret file for the engine API.
    pub(crate) jwt_secret: Option<PathBuf>,
    /// Whether to trust the chain's L2 RPC responses without verifying them.
    pub(crate) trust_l2_rpc: Option<bool>,
    /// Whether this chain is sequenced or validated by this supernode.
    #[serde(default, deserialize_with = "deserialize_mode")]
    pub(crate) mode: Option<NodeMode>,
    /// The file holding the key this chain signs the blocks it gossips with.
    ///
    /// Required once the chain sequences, and rejected on a chain that does not: see
    /// [`SequencerSettings`].
    ///
    /// A path rather than the key itself. kona-node also accepts `--p2p.sequencer.key` because a
    /// command line has nowhere else to put it; a supernode already configures itself from a file,
    /// and inlining the keys would make that one file the secret of every chain it hosts.
    ///
    /// A key is generated at this path when no file is there yet, as everywhere else in the stack
    /// that names a key by a path — so a mistyped path yields a working sequencer whose blocks no
    /// peer accepts, caught only by the unsafe-block-signer warning at startup.
    pub(crate) sequencer_key_path: Option<PathBuf>,
    /// Whether this chain's sequencer starts stopped, to be started over the admin API.
    pub(crate) sequencer_stopped: Option<bool>,
    /// How many L1 blocks this chain's sequencer keeps away from the L1 head when it picks an L1
    /// origin.
    pub(crate) sequencer_l1_confs: Option<u64>,
    /// Whether this chain's sequencer runs in recovery mode, building empty blocks that strictly
    /// advance the L1 origin.
    pub(crate) sequencer_recover: Option<bool>,
    /// The conductor RPC this chain's sequencer coordinates leadership through, if any.
    pub(crate) conductor_rpc: Option<Url>,
    /// The directory this chain's state lives under.
    ///
    /// A chain that names it directly gets exactly that directory; otherwise the chain gets
    /// `<datadir>/<l2-chain-id>`, so one `[defaults]` entry gives every chain its own directory.
    pub(crate) datadir: Option<PathBuf>,
    /// Whether to expose the admin namespace on this chain's RPC route.
    pub(crate) rpc_enable_admin: Option<bool>,
    /// Whether to expose the experimental `opstack` block-building namespace on this chain's RPC
    /// route.
    ///
    /// op-node's `--experimental.sequencer-api`, which op-supernode's devstack turns on for every
    /// virtual node it hosts: the op-test-sequencer drives block building through these methods.
    pub(crate) experimental_opstack_api: Option<bool>,
    /// The address this chain's P2P stack listens on.
    pub(crate) p2p_listen_ip: Option<IpAddr>,
    /// The TCP port this chain's gossip listens on.
    pub(crate) p2p_tcp_port: Option<u16>,
    /// The UDP port this chain's discovery listens on.
    pub(crate) p2p_udp_port: Option<u16>,
    /// The chain's unsafe block signer, overriding the registry entry for the chain.
    pub(crate) unsafe_block_signer: Option<alloy_primitives::Address>,
    /// The bootnodes this chain's discovery starts from.
    pub(crate) bootnodes: Option<Vec<String>>,
    /// A dependency-set file for this chain, required once the chain schedules interop.
    pub(crate) interop_dependency_set: Option<PathBuf>,
}

/// A resolved configuration: the L1 as written, and every chain with its values decided.
#[derive(Debug, Clone)]
pub(crate) struct ResolvedConfig {
    /// The L1 every chain derives from.
    pub(crate) l1: L1Settings,
    /// The chains this supernode runs, in the order the file lists them.
    ///
    /// A boxed slice rather than a `Vec`: the chain set is fixed once the file is read, and a
    /// collection that cannot grow says so in the type.
    pub(crate) chains: Box<[ResolvedChain]>,
    /// The socket the supernode serves its RPC on, if one is configured.
    pub(crate) admin_rpc: Option<SocketAddr>,
    /// The directory the process-wide interop stores live under, when one can be named.
    ///
    /// The interop verifier's frontier is a statement about the whole chain set, so it cannot
    /// live in any one chain's directory: whichever chain got it would be the one whose state
    /// could not be cleared independently. It is therefore `<[defaults] datadir>/interop`, and
    /// [`None`] when no default directory was given — a configuration that names only per-chain
    /// directories has nowhere process-wide to put it, which is a startup error once interop is
    /// actually scheduled rather than a reason to reject a validator that will never need it.
    pub(crate) interop_datadir: Option<PathBuf>,
    /// An interop activation timestamp from the file, overriding what the forks say.
    ///
    /// See [`InteropSettings::activation_timestamp`] for what it can and cannot do.
    pub(crate) interop_activation_override: Option<u64>,
}

/// One chain's resolved settings.
#[derive(Debug, Clone)]
pub(crate) struct ResolvedChain {
    /// The L2 chain id.
    pub(crate) l2_chain_id: u64,
    /// A rollup config file for this chain, or `None` to use its superchain registry entry.
    pub(crate) rollup_config: Option<PathBuf>,
    /// The chain's authenticated engine API URL.
    pub(crate) engine_rpc: Url,
    /// The JWT secret file for the engine API.
    pub(crate) jwt_secret: PathBuf,
    /// Whether to trust the chain's L2 RPC responses without verifying them.
    pub(crate) trust_l2_rpc: bool,
    /// How this chain sequences, or [`None`] when this supernode only validates it.
    ///
    /// This [`Option`] *is* the chain's mode: [`Self::mode`] reads it rather than a second field
    /// that could disagree with it. A sequencing chain has to have a key to sign the blocks it
    /// gossips, so "sequences" and "has the settings sequencing needs" are one state, decided
    /// while the file is being resolved, rather than a mode flag that the compose step then has to
    /// re-check.
    pub(crate) sequencer: Option<SequencerSettings>,
    /// The directory this chain's state lives under.
    pub(crate) datadir: PathBuf,
    /// Whether to expose the admin namespace on this chain's RPC route.
    pub(crate) rpc_enable_admin: bool,
    /// Whether to expose the experimental `opstack` block-building namespace on this chain's RPC
    /// route.
    pub(crate) experimental_opstack_api: bool,
    /// The address this chain's P2P stack listens on.
    pub(crate) p2p_listen_ip: IpAddr,
    /// The TCP port this chain's gossip listens on.
    pub(crate) p2p_tcp_port: u16,
    /// The UDP port this chain's discovery listens on.
    pub(crate) p2p_udp_port: u16,
    /// The chain's unsafe block signer, or `None` to use its superchain registry entry.
    pub(crate) unsafe_block_signer: Option<alloy_primitives::Address>,
    /// The bootnodes this chain's discovery starts from.
    pub(crate) bootnodes: Vec<String>,
    /// A dependency-set file for this chain, if one is configured.
    pub(crate) interop_dependency_set: Option<PathBuf>,
}

impl ResolvedChain {
    /// Whether this supernode sequences this chain or validates it.
    ///
    /// Derived from [`Self::sequencer`] rather than stored: there is one place a chain's mode is
    /// decided, so there is no combination of fields that says both.
    pub(crate) const fn mode(&self) -> NodeMode {
        match self.sequencer {
            Some(_) => NodeMode::Sequencer,
            None => NodeMode::Validator,
        }
    }
}

/// What one chain's sequencer needs, resolved.
///
/// Present exactly on the chains this supernode sequences. The fields mirror kona-node's
/// sequencer flags, so a chain sequences under a supernode the way it sequences under a
/// single-chain node; what is per-chain here is a flag per chain there.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct SequencerSettings {
    /// The file holding the key this chain signs the blocks it gossips with.
    ///
    /// Still a path at this point: resolution touches no files. Loading the key — or generating
    /// one, when the file is not there yet — happens where the signer is built, and a file that
    /// exists but does not hold a key is a startup error raised there.
    pub(crate) key_path: PathBuf,
    /// Whether the sequencer starts stopped, to be started over the admin API.
    pub(crate) stopped: bool,
    /// How many L1 blocks the sequencer keeps away from the L1 head when it picks an L1 origin.
    pub(crate) l1_confs: u64,
    /// Whether the sequencer runs in recovery mode.
    pub(crate) recover: bool,
    /// The conductor RPC the sequencer coordinates leadership through, if any.
    pub(crate) conductor_rpc: Option<Url>,
}

/// The ways a configuration file can fail to describe a runnable supernode.
///
/// Every variant names the chain it is about: with N chains, "no engine-rpc" on its own does not
/// tell an operator which entry to fix.
#[derive(Debug, thiserror::Error, PartialEq, Eq)]
pub(crate) enum ConfigError {
    /// The file lists no chains.
    #[error("no chains configured: a supernode runs at least one chain")]
    NoChains,
    /// A chain entry has no `l2-chain-id`.
    #[error("the chain at index {index} has no l2-chain-id")]
    MissingChainId {
        /// The chain's position in the `chains` array.
        index: usize,
    },
    /// A chain is missing a setting that has no default.
    #[error("chain {chain_id} has no {field}, and [defaults] does not set one either")]
    MissingSetting {
        /// The chain the setting is missing from.
        chain_id: u64,
        /// The name of the missing setting, as written in the file.
        field: &'static str,
    },
    /// Two chain entries name the same chain id.
    #[error("chain {chain_id} is configured twice")]
    DuplicateChain {
        /// The repeated chain id.
        chain_id: u64,
    },
    /// Two chains would listen on the same address.
    #[error(
        "chains {first} and {second} both listen on {what} {address}: give each chain its own port"
    )]
    AddressCollision {
        /// The first chain to claim the address.
        first: u64,
        /// The chain that claimed it again.
        second: u64,
        /// Which listener collided.
        what: &'static str,
        /// The address both chains claimed.
        address: String,
    },
    /// A chain is configured to sequence but has no key to sign with.
    #[error(
        "chain {chain_id} is configured to sequence but has no sequencer-key-path: a sequencing \
         chain signs the blocks it gossips, and a peer drops a block it cannot attribute"
    )]
    SequencerWithoutKey {
        /// The sequencing chain with no key.
        chain_id: u64,
    },
    /// A chain that does not sequence states a setting only a sequencer uses.
    #[error(
        "chain {chain_id} sets {field} but does not sequence: set mode = \"sequencer\" on the \
         chain, or drop the setting"
    )]
    SequencerSettingWithoutSequencing {
        /// The chain that stated the setting.
        chain_id: u64,
        /// The name of the setting, as written in the file.
        field: &'static str,
    },
    /// Two chains would keep their state in the same directory.
    #[error("chains {first} and {second} both use the data directory {path}")]
    DataDirCollision {
        /// The first chain to claim the directory.
        first: u64,
        /// The chain that claimed it again.
        second: u64,
        /// The directory both chains claimed.
        path: PathBuf,
    },
}

impl LokahiConfig {
    /// Parses a configuration from TOML.
    pub(crate) fn parse(toml: &str) -> Result<Self, toml::de::Error> {
        toml::from_str(toml)
    }

    /// Resolves every chain's settings against `[defaults]` and checks that the result is
    /// runnable.
    ///
    /// A chain's own value wins over the default; a setting with no value from either layer is an
    /// error unless the node can do without it. The checks are what N chains in one process makes
    /// possible to get wrong and a single-chain node cannot: the same chain configured twice, two
    /// chains sharing a P2P port, two chains sharing a data directory. There is no RPC port among
    /// them, because a chain does not have one: the whole process has one socket and a chain is a
    /// route on it.
    pub(crate) fn resolve(self) -> Result<ResolvedConfig, ConfigError> {
        let Self { l1, defaults, chains, admin, interop } = self;

        if chains.is_empty() {
            return Err(ConfigError::NoChains);
        }

        let resolved = chains
            .into_iter()
            .enumerate()
            .map(|(index, chain)| Self::resolve_chain(index, chain, &defaults))
            .collect::<Result<Vec<_>, _>>()?;

        check_unique(&resolved)?;

        let admin_rpc = admin.map(|admin| {
            SocketAddr::new(
                admin.rpc_addr.unwrap_or_else(|| IpAddr::from(Ipv4Addr::LOCALHOST)),
                admin.rpc_port,
            )
        });

        let interop_datadir = defaults.datadir.as_ref().map(|dir| dir.join(INTEROP_DIR));
        let interop_activation_override = interop.and_then(|interop| interop.activation_timestamp);

        Ok(ResolvedConfig {
            l1,
            chains: resolved.into_boxed_slice(),
            admin_rpc,
            interop_datadir,
            interop_activation_override,
        })
    }

    /// Resolves one chain's settings against the defaults.
    fn resolve_chain(
        index: usize,
        chain: ChainSettings,
        defaults: &ChainSettings,
    ) -> Result<ResolvedChain, ConfigError> {
        let l2_chain_id = chain.l2_chain_id.ok_or(ConfigError::MissingChainId { index })?;

        let required = |present: bool, field: &'static str| {
            present
                .then_some(())
                .ok_or(ConfigError::MissingSetting { chain_id: l2_chain_id, field })
        };

        let engine_rpc = chain.engine_rpc.clone().or_else(|| defaults.engine_rpc.clone());
        required(engine_rpc.is_some(), "engine-rpc")?;
        let jwt_secret = chain.jwt_secret.clone().or_else(|| defaults.jwt_secret.clone());
        required(jwt_secret.is_some(), "jwt-secret")?;
        let p2p_tcp_port = chain.p2p_tcp_port.or(defaults.p2p_tcp_port);
        required(p2p_tcp_port.is_some(), "p2p-tcp-port")?;
        let p2p_udp_port = chain.p2p_udp_port.or(defaults.p2p_udp_port);
        required(p2p_udp_port.is_some(), "p2p-udp-port")?;

        // A chain that names its own directory gets it; a shared default directory is split per
        // chain, so `datadir = "/var/lib/lokahi"` in `[defaults]` is not every chain writing over
        // every other chain's state.
        let datadir = chain.datadir.clone().map_or_else(
            || {
                defaults
                    .datadir
                    .clone()
                    .unwrap_or_else(|| PathBuf::from("."))
                    .join(l2_chain_id.to_string())
            },
            PathBuf::from,
        );

        let mode = chain.mode.or(defaults.mode).unwrap_or(NodeMode::Validator);
        let sequencer = Self::resolve_sequencer(l2_chain_id, mode, &chain, defaults)?;

        Ok(ResolvedChain {
            l2_chain_id,
            rollup_config: chain.rollup_config.clone().or_else(|| defaults.rollup_config.clone()),
            engine_rpc: engine_rpc.expect("checked above"),
            jwt_secret: jwt_secret.expect("checked above"),
            trust_l2_rpc: chain.trust_l2_rpc.or(defaults.trust_l2_rpc).unwrap_or_default(),
            sequencer,
            datadir,
            rpc_enable_admin: chain
                .rpc_enable_admin
                .or(defaults.rpc_enable_admin)
                .unwrap_or_default(),
            experimental_opstack_api: chain
                .experimental_opstack_api
                .or(defaults.experimental_opstack_api)
                .unwrap_or_default(),
            p2p_listen_ip: chain
                .p2p_listen_ip
                .or(defaults.p2p_listen_ip)
                .unwrap_or_else(|| IpAddr::from(Ipv4Addr::UNSPECIFIED)),
            p2p_tcp_port: p2p_tcp_port.expect("checked above"),
            p2p_udp_port: p2p_udp_port.expect("checked above"),
            unsafe_block_signer: chain.unsafe_block_signer.or(defaults.unsafe_block_signer),
            bootnodes: chain.bootnodes.or_else(|| defaults.bootnodes.clone()).unwrap_or_default(),
            interop_dependency_set: chain
                .interop_dependency_set
                .or_else(|| defaults.interop_dependency_set.clone()),
        })
    }

    /// Resolves the sequencer settings of a chain in `mode`, and checks that the two agree.
    ///
    /// The two ways they can disagree are both operator mistakes that would otherwise surface long
    /// after startup: a chain told to sequence with no key would build blocks and gossip none of
    /// them, and a chain given sequencer settings that does not sequence would run as a validator
    /// while its configuration says otherwise.
    ///
    /// Only settings the chain's *own* entry states make the second case an error. `[defaults]` is
    /// a base rather than a statement about any one chain — the same reason a shared `datadir`
    /// there is split per chain — so a mixed set can put `sequencer-l1-confs` in `[defaults]` and
    /// still have chains that only validate.
    fn resolve_sequencer(
        chain_id: u64,
        mode: NodeMode,
        chain: &ChainSettings,
        defaults: &ChainSettings,
    ) -> Result<Option<SequencerSettings>, ConfigError> {
        if mode.is_validator() {
            if let Some(field) = chain.stated_sequencer_setting() {
                return Err(ConfigError::SequencerSettingWithoutSequencing { chain_id, field });
            }
            return Ok(None);
        }

        let key_path = chain
            .sequencer_key_path
            .clone()
            .or_else(|| defaults.sequencer_key_path.clone())
            .ok_or(ConfigError::SequencerWithoutKey { chain_id })?;

        Ok(Some(SequencerSettings {
            key_path,
            stopped: chain.sequencer_stopped.or(defaults.sequencer_stopped).unwrap_or_default(),
            l1_confs: chain
                .sequencer_l1_confs
                .or(defaults.sequencer_l1_confs)
                .unwrap_or(DEFAULT_SEQUENCER_L1_CONFS),
            recover: chain.sequencer_recover.or(defaults.sequencer_recover).unwrap_or_default(),
            conductor_rpc: chain.conductor_rpc.clone().or_else(|| defaults.conductor_rpc.clone()),
        }))
    }
}

impl ChainSettings {
    /// The name of a sequencer-only setting this entry states, if it states one.
    ///
    /// Used to reject the settings on a chain that does not sequence; which one is named does not
    /// matter beyond pointing the operator at the line to look at.
    fn stated_sequencer_setting(&self) -> Option<&'static str> {
        [
            ("sequencer-key-path", self.sequencer_key_path.is_some()),
            ("sequencer-stopped", self.sequencer_stopped.is_some()),
            ("sequencer-l1-confs", self.sequencer_l1_confs.is_some()),
            ("sequencer-recover", self.sequencer_recover.is_some()),
            ("conductor-rpc", self.conductor_rpc.is_some()),
        ]
        .into_iter()
        .find_map(|(field, stated)| stated.then_some(field))
    }
}

/// Reports the first chain id, listener or data directory that two chains share.
///
/// Sharing any of them is a misconfiguration that a single-chain node cannot express and that
/// otherwise surfaces long after startup — as an address-in-use from one chain's RPC server, or as
/// two chains writing over each other's state.
fn check_unique(chains: &[ResolvedChain]) -> Result<(), ConfigError> {
    let mut ids = HashMap::new();
    let mut sockets: HashMap<(&'static str, String), u64> = HashMap::new();
    let mut datadirs: HashMap<&Path, u64> = HashMap::new();

    for chain in chains {
        let id = chain.l2_chain_id;

        if ids.insert(id, id).is_some() {
            return Err(ConfigError::DuplicateChain { chain_id: id });
        }

        let listeners =
            [("the p2p tcp port", chain.p2p_tcp_port), ("the p2p udp port", chain.p2p_udp_port)];
        for (what, port) in listeners {
            // Port 0 is a request for an ephemeral port, not an address: the kernel hands
            // every bind its own port, so two chains both saying 0 cannot collide. This is
            // how the devstack configures lokahi's P2P listeners, the same way it runs
            // kona-node.
            if port == 0 {
                continue;
            }
            let address = SocketAddr::new(chain.p2p_listen_ip, port).to_string();
            if let Some(&first) = sockets.get(&(what, address.clone())) {
                return Err(ConfigError::AddressCollision { first, second: id, what, address });
            }
            sockets.insert((what, address), id);
        }

        if let Some(&first) = datadirs.get(chain.datadir.as_path()) {
            return Err(ConfigError::DataDirCollision {
                first,
                second: id,
                path: chain.datadir.clone(),
            });
        }
        datadirs.insert(chain.datadir.as_path(), id);
    }

    Ok(())
}

/// Deserializes a [`NodeMode`] from its name.
///
/// [`NodeMode`] is a CLI type without a `Deserialize` implementation, and its `FromStr` is what
/// kona-node's `--mode` flag already accepts, so a mode reads the same in a lokahi file as it does
/// on a kona-node command line.
fn deserialize_mode<'de, D>(deserializer: D) -> Result<Option<NodeMode>, D::Error>
where
    D: Deserializer<'de>,
{
    let Some(name) = Option::<String>::deserialize(deserializer)? else {
        return Ok(None);
    };

    name.parse().map(Some).map_err(D::Error::custom)
}

#[cfg(test)]
mod tests {
    use super::*;

    /// A configuration with `chains`, sharing everything a chain needs through `[defaults]`.
    fn config(chains: &str) -> LokahiConfig {
        LokahiConfig::parse(&format!(
            r#"
            [l1]
            eth-rpc = "http://localhost:8545"
            beacon = "http://localhost:5052"

            [defaults]
            datadir = "/var/lib/lokahi"
            jwt-secret = "/etc/lokahi/jwt.hex"
            engine-rpc = "http://localhost:9551"
            p2p-tcp-port = 9222
            p2p-udp-port = 9222

            {chains}
            "#
        ))
        .expect("parses")
    }

    /// The interop table is optional, and its absence is what leaves lokahi reading each chain's
    /// Lagoon time -- the default for every node that is not a test harness.
    #[test]
    fn a_file_without_an_interop_table_has_no_activation_override() {
        let resolved = config(
            r#"
            [[chains]]
            l2-chain-id = 901
            "#,
        )
        .resolve()
        .expect("resolves");
        assert_eq!(resolved.interop_activation_override, None);
    }

    /// The activation override as the devstack writes it, in its own table because interop is a
    /// property of the whole hosted set rather than of one chain.
    #[test]
    fn an_interop_table_supplies_the_activation_override() {
        let config = LokahiConfig::parse(
            r#"
            [l1]
            eth-rpc = "http://localhost:8545"
            beacon = "http://localhost:5052"

            [interop]
            activation-timestamp = 1787335282

            [defaults]
            datadir = "/var/lib/lokahi"
            jwt-secret = "/etc/lokahi/jwt.hex"
            engine-rpc = "http://localhost:9551"
            p2p-tcp-port = 9222
            p2p-udp-port = 9222

            [[chains]]
            l2-chain-id = 901
            "#,
        )
        .expect("parses");

        let resolved = config.resolve().expect("resolves");
        assert_eq!(resolved.interop_activation_override, Some(1_787_335_282));
    }

    /// `deny_unknown_fields` covers the new table too, so a misspelled key is a parse error
    /// rather than a silently ignored activation.
    #[test]
    fn an_unknown_interop_key_is_refused() {
        let err = LokahiConfig::parse(
            r#"
            [l1]
            eth-rpc = "http://localhost:8545"
            beacon = "http://localhost:5052"

            [interop]
            activation-time = 1787335282

            [[chains]]
            l2-chain-id = 901
            "#,
        )
        .expect_err("an unknown interop key is not accepted");
        assert!(err.to_string().contains("activation-time"), "{err}");
    }

    /// The L1 epoch poll interval is an `[l1]` setting — the L1 is followed once for the whole
    /// process — and is optional: unset keeps kona's production default. The devstack sets it to
    /// the 2 seconds it hands op-node as `L1EpochPollInterval`, so lokahi observes the devnet
    /// L1's fast finality on the same cadence op-node does.
    #[test]
    fn the_l1_epoch_poll_interval_is_optional_and_read_from_the_l1_table() {
        let unset = config(&chain("l2-chain-id = 901")).resolve().expect("resolves");
        assert_eq!(unset.l1.epoch_poll_interval, None);

        let resolved = LokahiConfig::parse(
            r#"
            [l1]
            eth-rpc = "http://localhost:8545"
            beacon = "http://localhost:5052"
            epoch-poll-interval = 2

            [defaults]
            jwt-secret = "/etc/lokahi/jwt.hex"
            engine-rpc = "http://localhost:9551"
            p2p-tcp-port = 9222
            p2p-udp-port = 9222

            [[chains]]
            l2-chain-id = 901
            "#,
        )
        .expect("parses")
        .resolve()
        .expect("resolves");
        assert_eq!(resolved.l1.epoch_poll_interval, Some(2));
    }

    /// A chain entry naming only what the test is about.
    fn chain(fields: &str) -> String {
        format!("[[chains]]\n{fields}\n")
    }

    #[test]
    fn defaults_fill_what_a_chain_does_not_say() {
        let resolved = config(&chain("l2-chain-id = 901")).resolve().expect("resolves");

        let chain = &resolved.chains[0];
        assert_eq!(chain.l2_chain_id, 901);
        assert_eq!(chain.engine_rpc.as_str(), "http://localhost:9551/");
        assert_eq!(chain.jwt_secret, PathBuf::from("/etc/lokahi/jwt.hex"));
        assert_eq!(chain.mode(), NodeMode::Validator);
        assert_eq!(chain.sequencer, None);
    }

    #[test]
    fn a_chain_overrides_the_default() {
        let resolved = config(&chain(
            r#"
            l2-chain-id = 901
            engine-rpc = "http://localhost:7777"
            mode = "sequencer"
            sequencer-key-path = "/etc/lokahi/901-sequencer.key"
            "#,
        ))
        .resolve()
        .expect("resolves");

        let chain = &resolved.chains[0];
        assert_eq!(chain.engine_rpc.as_str(), "http://localhost:7777/");
        assert_eq!(chain.mode(), NodeMode::Sequencer);
    }

    /// A sequencing chain gets kona-node's sequencer defaults, so it sequences under a supernode
    /// the way it sequences under a single-chain node.
    #[test]
    fn a_sequencing_chain_takes_kona_nodes_sequencer_defaults() {
        let resolved = config(&chain(
            r#"
            l2-chain-id = 901
            mode = "sequencer"
            sequencer-key-path = "/etc/lokahi/901-sequencer.key"
            "#,
        ))
        .resolve()
        .expect("resolves");

        assert_eq!(
            resolved.chains[0].sequencer,
            Some(SequencerSettings {
                key_path: PathBuf::from("/etc/lokahi/901-sequencer.key"),
                stopped: false,
                l1_confs: DEFAULT_SEQUENCER_L1_CONFS,
                recover: false,
                conductor_rpc: None,
            })
        );
    }

    /// The mistake that would otherwise surface as a chain building blocks and gossiping none of
    /// them: a sequencer with nothing to sign with.
    #[test]
    fn a_sequencing_chain_needs_a_key() {
        assert_eq!(
            config(&chain("l2-chain-id = 901\nmode = \"sequencer\""))
                .resolve()
                .expect_err("no key"),
            ConfigError::SequencerWithoutKey { chain_id: 901 }
        );
    }

    /// The other direction: settings that only a sequencer reads, on a chain that only validates.
    /// Left to run, the chain would validate while its configuration says it sequences.
    #[test]
    fn a_validating_chain_cannot_state_a_sequencer_setting() {
        assert_eq!(
            config(&chain("l2-chain-id = 901\nsequencer-l1-confs = 2"))
                .resolve()
                .expect_err("sequencer setting on a validator"),
            ConfigError::SequencerSettingWithoutSequencing {
                chain_id: 901,
                field: "sequencer-l1-confs",
            }
        );
    }

    /// One supernode can sequence one chain and validate another, which is the arrangement a
    /// mixed cluster runs: the sequencer settings shared in `[defaults]` reach the chain that
    /// sequences and are ignored by the chain that does not.
    #[test]
    fn one_supernode_can_sequence_one_chain_and_validate_another() {
        let file = format!(
            r#"
            [l1]
            eth-rpc = "http://localhost:8545"
            beacon = "http://localhost:5052"

            [defaults]
            datadir = "/var/lib/lokahi"
            jwt-secret = "/etc/lokahi/jwt.hex"
            engine-rpc = "http://localhost:9551"
            p2p-udp-port = 9222
            sequencer-key-path = "/etc/lokahi/sequencer.key"
            sequencer-l1-confs = 2

            {}{}
            "#,
            chain("l2-chain-id = 901\nmode = \"sequencer\"\np2p-tcp-port = 9222"),
            chain("l2-chain-id = 902\np2p-tcp-port = 9232\np2p-udp-port = 9232"),
        );

        let resolved = LokahiConfig::parse(&file).expect("parses").resolve().expect("resolves");

        assert_eq!(resolved.chains[0].mode(), NodeMode::Sequencer);
        assert_eq!(
            resolved.chains[0].sequencer.as_ref().map(|sequencer| sequencer.l1_confs),
            Some(2)
        );
        assert_eq!(resolved.chains[1].mode(), NodeMode::Validator);
        assert_eq!(resolved.chains[1].sequencer, None);
    }

    #[test]
    fn a_shared_data_directory_is_split_per_chain() {
        let resolved = config(&format!(
            "{}{}",
            chain("l2-chain-id = 901\np2p-tcp-port = 9222\np2p-udp-port = 9222"),
            chain("l2-chain-id = 902\np2p-tcp-port = 9232\np2p-udp-port = 9232"),
        ))
        .resolve()
        .expect("resolves");

        assert_eq!(resolved.chains[0].datadir, PathBuf::from("/var/lib/lokahi/901"));
        assert_eq!(resolved.chains[1].datadir, PathBuf::from("/var/lib/lokahi/902"));
    }

    #[test]
    fn a_chain_can_name_its_own_data_directory() {
        let resolved = config(&chain("l2-chain-id = 901\ndatadir = \"/srv/901\""))
            .resolve()
            .expect("resolves");

        assert_eq!(resolved.chains[0].datadir, PathBuf::from("/srv/901"));
    }

    #[test]
    fn a_supernode_needs_a_chain() {
        assert_eq!(config("").resolve().expect_err("no chains"), ConfigError::NoChains);
    }

    #[test]
    fn a_chain_entry_needs_a_chain_id() {
        assert_eq!(
            config(&chain("p2p-tcp-port = 9222")).resolve().expect_err("no chain id"),
            ConfigError::MissingChainId { index: 0 }
        );
    }

    /// A setting with no value in either layer names the chain that is missing it: with N chains,
    /// "no engine-rpc" alone does not say which entry to fix.
    #[test]
    fn a_missing_setting_names_its_chain() {
        let toml = r#"
            [l1]
            eth-rpc = "http://localhost:8545"
            beacon = "http://localhost:5052"

            [[chains]]
            l2-chain-id = 901
            jwt-secret = "/etc/lokahi/jwt.hex"
            p2p-tcp-port = 9222
            p2p-udp-port = 9222
        "#;

        assert_eq!(
            LokahiConfig::parse(toml).expect("parses").resolve().expect_err("no engine rpc"),
            ConfigError::MissingSetting { chain_id: 901, field: "engine-rpc" }
        );
    }

    #[test]
    fn the_same_chain_cannot_be_configured_twice() {
        let entries = format!(
            "{}{}",
            chain("l2-chain-id = 901"),
            chain("l2-chain-id = 901\np2p-tcp-port = 9232\np2p-udp-port = 9232"),
        );

        assert_eq!(
            config(&entries).resolve().expect_err("duplicate chain"),
            ConfigError::DuplicateChain { chain_id: 901 }
        );
    }

    /// Two chains inheriting the same port from `[defaults]` is the mistake the overlay makes easy,
    /// so it is caught at startup rather than as an address-in-use from one chain's gossip. The
    /// RPC is not among the listeners a chain can collide on any more: the process has one socket
    /// and a chain is a route on it.
    #[test]
    fn two_chains_cannot_share_a_listener() {
        let entries = format!("{}{}", chain("l2-chain-id = 901"), chain("l2-chain-id = 902"));

        let error = config(&entries).resolve().expect_err("port collision");
        assert_eq!(
            error,
            ConfigError::AddressCollision {
                first: 901,
                second: 902,
                what: "the p2p tcp port",
                address: "0.0.0.0:9222".to_string(),
            }
        );
    }

    /// Port 0 asks the kernel for an ephemeral port, so every chain saying 0 gets its own
    /// socket — the address they appear to share is not one either of them will listen on.
    /// This is how the devstack runs lokahi (and kona-node) under parallel tests: a concrete
    /// port picked ahead of time can be taken by another process before the node binds it.
    #[test]
    fn ephemeral_p2p_ports_are_not_a_collision() {
        let entries = format!(
            "{}{}",
            chain("l2-chain-id = 901\np2p-tcp-port = 0\np2p-udp-port = 0"),
            chain("l2-chain-id = 902\np2p-tcp-port = 0\np2p-udp-port = 0"),
        );

        let resolved = config(&entries).resolve().expect("port 0 is ephemeral, not shared");
        assert_eq!(resolved.chains.len(), 2);
    }

    #[test]
    fn two_chains_cannot_share_a_data_directory() {
        let entries = format!(
            "{}{}",
            chain("l2-chain-id = 901\ndatadir = \"/srv/shared\""),
            chain(
                "l2-chain-id = 902\ndatadir = \"/srv/shared\"\np2p-tcp-port = 9232\np2p-udp-port = 9232"
            ),
        );

        assert_eq!(
            config(&entries).resolve().expect_err("data dir collision"),
            ConfigError::DataDirCollision {
                first: 901,
                second: 902,
                path: PathBuf::from("/srv/shared"),
            }
        );
    }

    /// A misspelled setting is an error rather than a silently ignored line, so a chain does not
    /// run with a setting the operator believes they set.
    #[test]
    fn an_unknown_setting_is_rejected() {
        let error = LokahiConfig::parse(
            r#"
            [l1]
            eth-rpc = "http://localhost:8545"
            beacon = "http://localhost:5052"

            [[chains]]
            l2-chain-id = 901
            rpc-prot = 9545
            "#,
        )
        .expect_err("unknown field");

        assert!(error.to_string().contains("rpc-prot"), "unexpected error: {error}");
    }

    /// A chain does not have a port: the supernode has one socket and a chain is a route on it. A
    /// file still naming `rpc-port` per chain is rejected rather than quietly ignored, so an
    /// operator carrying an old file over is told the setting is gone instead of getting a chain
    /// on an address nothing serves.
    #[test]
    fn a_chain_may_not_name_an_rpc_port() {
        let error = LokahiConfig::parse(
            r#"
            [l1]
            eth-rpc = "http://localhost:8545"
            beacon = "http://localhost:5052"

            [[chains]]
            l2-chain-id = 901
            rpc-port = 9545
            "#,
        )
        .expect_err("rpc-port is not a chain setting");

        assert!(error.to_string().contains("rpc-port"), "unexpected error: {error}");
    }

    /// The documented example is parsed here so it cannot drift from what the code accepts.
    #[test]
    fn the_example_configuration_resolves() {
        let resolved = LokahiConfig::parse(include_str!("../config.example.toml"))
            .expect("parses")
            .resolve()
            .expect("resolves");

        assert_eq!(resolved.chains.len(), 2);
        assert_eq!(resolved.chains[0].l2_chain_id, 901);
        assert_eq!(resolved.chains[1].l2_chain_id, 902);
        assert!(resolved.chains.iter().all(|chain| chain.rpc_enable_admin));
        assert_eq!(resolved.chains[0].mode(), NodeMode::Validator);
        assert_eq!(resolved.chains[1].mode(), NodeMode::Sequencer);
    }

    /// The experimental opstack namespace is off unless asked for, follows `[defaults]` like the
    /// admin flag, and a chain's own entry overrides the default — per chain, because op-node's
    /// `--experimental.sequencer-api` is per node.
    #[test]
    fn the_opstack_namespace_is_off_by_default_and_resolved_per_chain() {
        let resolved = LokahiConfig::parse(
            r#"
            [l1]
            eth-rpc = "http://localhost:8545"
            beacon = "http://localhost:5052"

            [defaults]
            engine-rpc = "http://localhost:8551"
            jwt-secret = "/tmp/jwt.hex"
            p2p-tcp-port = 9222
            p2p-udp-port = 9223
            experimental-opstack-api = true

            [[chains]]
            l2-chain-id = 901

            [[chains]]
            l2-chain-id = 902
            p2p-tcp-port = 9224
            p2p-udp-port = 9225
            experimental-opstack-api = false
            "#,
        )
        .expect("parses")
        .resolve()
        .expect("resolves");

        assert!(resolved.chains[0].experimental_opstack_api, "the default applies to chain 901");
        assert!(!resolved.chains[1].experimental_opstack_api, "chain 902 overrides the default");

        let unstated = LokahiConfig::parse(
            r#"
            [l1]
            eth-rpc = "http://localhost:8545"
            beacon = "http://localhost:5052"

            [[chains]]
            l2-chain-id = 901
            engine-rpc = "http://localhost:8551"
            jwt-secret = "/tmp/jwt.hex"
            p2p-tcp-port = 9222
            p2p-udp-port = 9223
            "#,
        )
        .expect("parses")
        .resolve()
        .expect("resolves");
        assert!(
            !unstated.chains[0].experimental_opstack_api,
            "an experimental namespace stays off unless asked for"
        );
    }

    /// A file that says nothing about `[admin]` gets no RPC at all — the chains are served on that
    /// socket too, so there is no second place for them to go.
    #[test]
    fn the_admin_rpc_is_off_unless_it_is_asked_for() {
        let resolved = config(&chain("l2-chain-id = 901")).resolve().expect("resolves");
        assert_eq!(resolved.admin_rpc, None);
    }

    /// A control surface does not become reachable off the host by leaving a field out.
    #[test]
    fn the_admin_rpc_listens_on_loopback_by_default() {
        let mut file = config(&chain("l2-chain-id = 901"));
        file.admin = Some(AdminSettings { rpc_addr: None, rpc_port: 9600 });

        let resolved = file.resolve().expect("resolves");
        assert_eq!(resolved.admin_rpc, Some("127.0.0.1:9600".parse().unwrap()));
    }

    /// Port 0 asks the OS for a free port, which is how a harness starts a supernode without
    /// having to pick one: the port it got is logged.
    #[test]
    fn the_admin_rpc_may_ask_the_os_for_a_port() {
        let mut file = config(&chain("l2-chain-id = 901"));
        file.admin = Some(AdminSettings { rpc_addr: None, rpc_port: 0 });

        let resolved = file.resolve().expect("resolves");
        assert_eq!(resolved.admin_rpc, Some("127.0.0.1:0".parse().unwrap()));
    }
}
