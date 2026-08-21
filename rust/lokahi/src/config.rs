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
    /// The supernode-level admin/test RPC, when the file asks for one.
    ///
    /// Absent means the supernode exposes no process-wide surface at all, which is what a
    /// production deployment wants: everything an operator needs is on the chains' own RPCs.
    pub(crate) admin: Option<AdminSettings>,
}

/// The supernode-level admin/test RPC's settings.
#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields, rename_all = "kebab-case")]
pub(crate) struct AdminSettings {
    /// The address the admin RPC listens on. Loopback by default: this is a control surface, so
    /// it does not become reachable off the host by leaving a field out.
    pub(crate) rpc_addr: Option<IpAddr>,
    /// The port the admin RPC listens on. `0` lets the OS choose, and the chosen port is logged.
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
    /// The address this chain's RPC server listens on.
    pub(crate) rpc_addr: Option<IpAddr>,
    /// The port this chain's RPC server listens on.
    pub(crate) rpc_port: Option<u16>,
    /// Whether to expose the admin namespace on this chain's RPC server.
    pub(crate) rpc_enable_admin: Option<bool>,
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
    /// The socket the supernode-level admin RPC listens on, if one is configured.
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
    /// The socket this chain's RPC server listens on.
    pub(crate) rpc_socket: SocketAddr,
    /// Whether to expose the admin namespace on this chain's RPC server.
    pub(crate) rpc_enable_admin: bool,
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
    /// Still a path at this point: resolution reads no files, so whether the key is present and
    /// parseable is a startup error raised where the signer is built.
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
    /// The admin RPC would listen on a socket a chain's RPC server already claimed.
    #[error(
        "the admin rpc would listen on {address}, which is chain {chain_id}'s rpc socket: give \
         the admin rpc its own port"
    )]
    AdminAddressCollision {
        /// The chain that claimed the socket.
        chain_id: u64,
        /// The socket both would bind.
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
    /// chains sharing a port, two chains sharing a data directory.
    pub(crate) fn resolve(self) -> Result<ResolvedConfig, ConfigError> {
        let Self { l1, defaults, chains, admin } = self;

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
        check_admin_socket(admin_rpc, &resolved)?;

        let interop_datadir = defaults.datadir.as_ref().map(|dir| dir.join(INTEROP_DIR));

        Ok(ResolvedConfig { l1, chains: resolved.into_boxed_slice(), admin_rpc, interop_datadir })
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
        let rpc_port = chain.rpc_port.or(defaults.rpc_port);
        required(rpc_port.is_some(), "rpc-port")?;
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

        let rpc_addr = chain
            .rpc_addr
            .or(defaults.rpc_addr)
            .unwrap_or_else(|| IpAddr::from(Ipv4Addr::LOCALHOST));

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
            rpc_socket: SocketAddr::new(rpc_addr, rpc_port.expect("checked above")),
            rpc_enable_admin: chain
                .rpc_enable_admin
                .or(defaults.rpc_enable_admin)
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

        let listeners = [
            ("the rpc socket", chain.rpc_socket.to_string()),
            (
                "the p2p tcp port",
                SocketAddr::new(chain.p2p_listen_ip, chain.p2p_tcp_port).to_string(),
            ),
            (
                "the p2p udp port",
                SocketAddr::new(chain.p2p_listen_ip, chain.p2p_udp_port).to_string(),
            ),
        ];
        for (what, address) in listeners {
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

/// Reports an admin RPC socket that a chain's RPC server has already claimed.
///
/// Same reason as the per-chain checks: two servers on one socket surfaces as an address-in-use
/// after one of them is already up, and which one wins depends on start order. Port 0 is exempt —
/// it is a request for any free port, not a claim on one.
fn check_admin_socket(
    admin_rpc: Option<SocketAddr>,
    chains: &[ResolvedChain],
) -> Result<(), ConfigError> {
    let Some(admin_rpc) = admin_rpc.filter(|socket| socket.port() != 0) else {
        return Ok(());
    };

    chains.iter().find(|chain| chain.rpc_socket == admin_rpc).map_or(Ok(()), |chain| {
        Err(ConfigError::AdminAddressCollision {
            chain_id: chain.l2_chain_id,
            address: admin_rpc.to_string(),
        })
    })
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
            rpc-port = 9545
            p2p-tcp-port = 9222
            p2p-udp-port = 9222

            {chains}
            "#
        ))
        .expect("parses")
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
        assert_eq!(chain.rpc_socket.port(), 9545);
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
            chain("l2-chain-id = 901\nmode = \"sequencer\"\nrpc-port = 9545\np2p-tcp-port = 9222"),
            chain("l2-chain-id = 902\nrpc-port = 9555\np2p-tcp-port = 9232\np2p-udp-port = 9232"),
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
            chain("l2-chain-id = 901\nrpc-port = 9545\np2p-tcp-port = 9222\np2p-udp-port = 9222"),
            chain("l2-chain-id = 902\nrpc-port = 9555\np2p-tcp-port = 9232\np2p-udp-port = 9232"),
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
            config(&chain("rpc-port = 9545")).resolve().expect_err("no chain id"),
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
            rpc-port = 9545
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
            chain("l2-chain-id = 901\nrpc-port = 9555\np2p-tcp-port = 9232\np2p-udp-port = 9232"),
        );

        assert_eq!(
            config(&entries).resolve().expect_err("duplicate chain"),
            ConfigError::DuplicateChain { chain_id: 901 }
        );
    }

    /// Two chains inheriting the same port from `[defaults]` is the mistake the overlay makes easy,
    /// so it is caught at startup rather than as an address-in-use from one chain's RPC server.
    #[test]
    fn two_chains_cannot_share_a_listener() {
        let entries = format!("{}{}", chain("l2-chain-id = 901"), chain("l2-chain-id = 902"));

        let error = config(&entries).resolve().expect_err("port collision");
        assert_eq!(
            error,
            ConfigError::AddressCollision {
                first: 901,
                second: 902,
                what: "the rpc socket",
                address: "127.0.0.1:9545".to_string(),
            }
        );
    }

    #[test]
    fn two_chains_cannot_share_a_data_directory() {
        let entries = format!(
            "{}{}",
            chain("l2-chain-id = 901\ndatadir = \"/srv/shared\""),
            chain(
                "l2-chain-id = 902\ndatadir = \"/srv/shared\"\nrpc-port = 9555\np2p-tcp-port = 9232\np2p-udp-port = 9232"
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

    /// A file that says nothing about `[admin]` gets no process-wide surface at all, which is what
    /// a production deployment wants: the chains' own RPCs are the operator interface.
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

    /// The same reason the per-chain listeners are checked: an overlap surfaces as an
    /// address-in-use after one of the two servers is already up.
    #[test]
    fn the_admin_rpc_cannot_take_a_chains_rpc_socket() {
        let mut file = config(&chain("l2-chain-id = 901"));
        file.admin = Some(AdminSettings { rpc_addr: None, rpc_port: 9545 });

        assert_eq!(
            file.resolve().expect_err("admin socket collision"),
            ConfigError::AdminAddressCollision {
                chain_id: 901,
                address: "127.0.0.1:9545".to_string(),
            }
        );
    }

    /// Port 0 asks the OS for a free port rather than claiming one, so it cannot collide with a
    /// chain even when a chain is also on 0.
    #[test]
    fn the_admin_rpc_may_ask_the_os_for_a_port() {
        let mut file = config(&chain("l2-chain-id = 901"));
        file.admin = Some(AdminSettings { rpc_addr: None, rpc_port: 0 });

        let resolved = file.resolve().expect("resolves");
        assert_eq!(resolved.admin_rpc, Some("127.0.0.1:0".parse().unwrap()));
    }
}
