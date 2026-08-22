//! Interop: running the cross-chain verification round loop against the chains this process hosts.
//!
//! The verifier itself lives in `lokahi-interop`, which knows nothing about actors, kona or
//! rocksdb paths. This module is the join: it decides *whether* interop is on, opens the stores,
//! implements the two observation seams over the running chains, and drives the loop as one actor
//! alongside them.
//!
//! ## Whether interop is on
//!
//! Interop is on when every chain in the set schedules the Lagoon hardfork at the same timestamp,
//! and off when none of them does. Anything in between is a configuration error rather than a
//! partially-interop supernode: the verifier's rounds are lockstep across the whole set, so a
//! chain that is not part of the cluster cannot be a chain the cluster waits for. See
//! [`InteropActor::activation`].

mod actor;
mod chain;
mod query;
mod test_api;

use alloy_primitives::ChainId;
use alloy_provider::RootProvider;
use anyhow::{Context, Result, bail};
use kona_engine::CrossSafePromoter;
use kona_genesis::RollupConfig;
use kona_node_service::{ChainControllerRequest, ChainControllerRpcRequest};
use kona_rpc::L1WatcherQuerySender;
use kona_safedb::{SafeDatabase, SharedSafeDb};
use lokahi_interop::{
    ChainReplacement, InteropChain, LogStores, RewindableChain, RocksKv, RocksOutputArchive,
    Verifier, VerifierConfig, open_log_store, open_output_archive, open_verified_store,
};
use op_alloy_network::Optimism;
use std::{
    path::{Path, PathBuf},
    sync::Arc,
};
use tokio::sync::mpsc;
use tracing::info;
use url::Url;

pub(crate) use actor::InteropActor;
pub(crate) use chain::{ArchiveDenyList, ChainRewindRoute, L1Provider, NodeChain};
pub(crate) use query::{
    InteropQueryError, InteropReader, InteropStatus, SealedBlocks, Verdict, VerifiedAt,
};
pub(crate) use test_api::InteropTestHandle;

use actor::ChainRoute;

/// The name of the safe-head database inside a chain's data directory.
pub(crate) const SAFE_DB_DIR: &str = "safedb";

/// One chain a supernode hosts, as the interop decision reads it.
///
/// Named rather than a `(ChainId, &RollupConfig)` pair: which half is which matters at every call
/// site, and a chain whose config was loaded from a file cannot have its id recovered from that
/// config.
#[derive(Debug, Clone, Copy)]
pub(crate) struct HostedChain<'a> {
    /// The chain's id.
    pub(crate) chain_id: ChainId,
    /// The chain's rollup config.
    pub(crate) rollup_config: &'a RollupConfig,
}

/// Everything one composed chain contributes to the interop actor.
pub(crate) struct ChainInterop {
    /// The chain's id.
    pub(crate) chain_id: ChainId,
    /// The chain's own data directory, which its log store lives in.
    pub(crate) datadir: PathBuf,
    /// The chain's rollup config.
    pub(crate) rollup_config: RollupConfig,
    /// The safe-head database its controller records into, which the verifier reads history from.
    pub(crate) safe_db: SharedSafeDb,
    /// A read-only, JWT-authenticated execution-layer provider over the chain's engine endpoint.
    pub(crate) el: RootProvider<Optimism>,
    /// The chain controller's read-only query channel.
    pub(crate) queries: mpsc::Sender<ChainControllerRpcRequest>,
    /// The L1 watcher's query channel for this chain, which answers its derivation L1 progress.
    pub(crate) l1_queries: L1WatcherQuerySender,
    /// The chain controller's request channel, which applies promotions and rewinds.
    pub(crate) requests: mpsc::Sender<ChainControllerRequest>,
    /// The capability to promote this chain's cross-safe head.
    pub(crate) promoter: CrossSafePromoter,
    /// The chain's invalidated-output archive, which its engine reads as the deny list.
    ///
    /// The same handle the chain controller was composed with: the verifier writes an
    /// invalidation into it and the engine's deny checks read it, so the two cannot be looking at
    /// different stores.
    pub(crate) archive: Arc<RocksOutputArchive>,
}

impl ChainInterop {
    /// Opens the safe-head database in one chain's data directory.
    ///
    /// One database per chain, in that chain's own directory, so clearing a chain's state clears
    /// the history recorded about it and nothing else.
    pub(crate) fn open_safe_db(datadir: &Path) -> Result<SharedSafeDb> {
        let path = datadir.join(SAFE_DB_DIR);
        let db = SafeDatabase::new(&path)
            .with_context(|| format!("failed to open the safe-head database {}", path.display()))?;
        Ok(Arc::new(db))
    }

    /// Opens the invalidated-output archive in one chain's data directory.
    ///
    /// Per chain like the safe-head database — the deny list it doubles as is a per-chain
    /// question — but unlike everything else in the chain's directory it is NOT re-derivable:
    /// the output preimages it holds die with the replaced blocks. Its directory is separable
    /// for backup on purpose.
    pub(crate) fn open_archive(datadir: &Path) -> Result<Arc<RocksOutputArchive>> {
        let archive = open_output_archive(datadir).with_context(|| {
            format!("failed to open the invalidated-output archive under {}", datadir.display())
        })?;
        Ok(Arc::new(archive))
    }
}

impl InteropActor {
    /// The interop activation timestamp the hosted set shares, or [`None`] when interop is off.
    ///
    /// The verifier takes one cluster-wide activation timestamp, while the message rules it
    /// applies read each chain's own config. Requiring the set to agree here is what keeps those
    /// two the same number, so the rule the verifier applies and the rule the proof applies
    /// cannot diverge.
    ///
    /// A set where some chains schedule Lagoon and others do not is rejected rather than reduced
    /// to the chains that do. Rounds are lockstep across the set: a chain outside the cluster
    /// would still be one the cluster waited for, and its blocks would be verified against rules
    /// it never activated.
    ///
    /// `override_at` governs outright when it is set, and the scheduled forks are not consulted
    /// at all -- op-supernode's `resolveInteropActivationTimestamp` returns the override before it
    /// reads any chain's Lagoon time (`op-supernode/supernode/supernode.go:164`), and lokahi is
    /// meant to behave the way op-supernode behaves.
    ///
    /// That is not the rule this function started with. It first required a chain that scheduled
    /// Lagoon to agree with the override, on the reasoning that the message rules the verifier
    /// applies read the fork, so activating elsewhere would put the verifier and the proof on
    /// different rules. The acceptance suites then refused four tests over it --
    /// `TestInteropFaultProofs_ActivationBoundary`, `_PreForkActivation` and `TestPostInbox`,
    /// which schedule Lagoon well after the timestamp their preset supplies. op-supernode runs
    /// them,
    /// so the divergence was lokahi's, and mirroring op-supernode is the target that decides it.
    pub(crate) fn activation(
        chains: &[HostedChain<'_>],
        override_at: Option<u64>,
    ) -> Result<Option<u64>> {
        let Some(override_at) = override_at else { return Self::scheduled_activation(chains) };

        // The override wins, and the scheduled forks are not consulted -- neither to agree with
        // nor to disagree with. This mirrors op-supernode, which returns the override before it
        // looks at any rollup config, and which therefore also skips the cross-chain checks the
        // scheduled path applies.
        //
        // op-supernode says nothing when the two disagree. Reporting it is lokahi's own addition:
        // taking one number over another that the operator also wrote down is worth a line in the
        // log, and a log line cannot change what the verifier does. It is INFO rather than WARN
        // because the disagreement is the normal case for the presets that pass an override.
        for chain in chains {
            let Some(scheduled) = chain.rollup_config.hardforks.lagoon_time else { continue };
            let genesis = chain.rollup_config.genesis.l2_time;
            if Self::activation_point(scheduled, genesis) !=
                Self::activation_point(override_at, genesis)
            {
                info!(
                    chain = %chain.chain_id,
                    scheduled,
                    override_at,
                    "interop activation overridden away from this chain's scheduled Lagoon time"
                );
            }
        }

        Ok(Some(override_at))
    }

    /// The activation the hosted set's own rollup configs agree on.
    ///
    /// This is the path every node that is not handed an override takes, and it is unchanged by
    /// the existence of one: a set that cannot form a cluster is still refused here.
    fn scheduled_activation(chains: &[HostedChain<'_>]) -> Result<Option<u64>> {
        let mut scheduled = chains.iter().filter_map(|chain| {
            chain.rollup_config.hardforks.lagoon_time.map(|time| (chain.chain_id, time))
        });

        let Some((first_id, activation)) = scheduled.next() else { return Ok(None) };

        if let Some((other_id, other_time)) = scheduled.find(|(_, time)| *time != activation) {
            bail!(
                "chain {first_id} schedules Lagoon at {activation} but chain {other_id} schedules \
                 it at {other_time}: one supernode verifies one interop cluster, whose chains \
                 activate together"
            );
        }

        let unscheduled: Vec<ChainId> = chains
            .iter()
            .filter(|chain| chain.rollup_config.hardforks.lagoon_time.is_none())
            .map(|chain| chain.chain_id)
            .collect();
        if !unscheduled.is_empty() {
            bail!(
                "chain {first_id} schedules Lagoon at {activation} but chains {unscheduled:?} do \
                 not schedule it at all: a supernode verifies its whole chain set together, so a \
                 chain outside the interop cluster cannot be hosted alongside one inside it"
            );
        }

        Ok(Some(activation))
    }

    /// The first block timestamp a fork scheduled at `scheduled` applies to on a chain whose
    /// first block is at `genesis`.
    ///
    /// Anything at or before genesis is the same activation point, since there is no earlier
    /// block for two such numbers to disagree about. Comparing the raw numbers instead would
    /// refuse a set where one side spells "from the first block" as 0 and the other as the
    /// genesis timestamp, which is how the devstack's presets and op-supernode each spell it.
    const fn activation_point(scheduled: u64, genesis: u64) -> u64 {
        if scheduled > genesis { scheduled } else { genesis }
    }

    /// Builds the interop actor over the composed chains.
    ///
    /// The stores are opened here, at startup, and not lazily on the first round: a data
    /// directory that cannot be opened is a configuration problem, and a supernode that reports
    /// it as a startup failure is far easier to operate than one that starts, serves its chains,
    /// and never promotes anything.
    pub(crate) fn build(
        interop_datadir: Option<&Path>,
        l1_eth_rpc: &Url,
        activation_timestamp: u64,
        message_expiry_window: Option<u64>,
        chains: Vec<ChainInterop>,
    ) -> Result<Self> {
        let interop_datadir = interop_datadir.ok_or_else(|| {
            anyhow::anyhow!(
                "the chain set schedules interop at {activation_timestamp}, whose verifier keeps \
                 process-wide state, but the configuration names no `datadir` in `[defaults]` to put \
                 it under"
            )
        })?;

        std::fs::create_dir_all(interop_datadir).with_context(|| {
            format!("failed to create the interop data directory {}", interop_datadir.display())
        })?;

        let verified = open_verified_store(interop_datadir).with_context(|| {
            format!("failed to open the interop verified store under {}", interop_datadir.display())
        })?;

        let mut stores: LogStores = LogStores::new();
        let mut routes = Vec::with_capacity(chains.len());
        let mut observed: Vec<Arc<dyn InteropChain>> = Vec::with_capacity(chains.len());
        let mut replacements = std::collections::BTreeMap::new();

        for chain in chains {
            let chain_id = chain.chain_id;

            // The log store lives in the chain's own directory, next to the safe-head database:
            // both are re-derivable from that chain alone, and clearing one chain's
            // logs must not touch another's.
            let store = open_log_store(&chain.datadir, chain_id).with_context(|| {
                format!(
                    "failed to open the interop log store of chain {chain_id} under {}",
                    chain.datadir.display()
                )
            })?;
            stores.insert(chain_id, Arc::new(store));

            let node_chain = Arc::new(NodeChain::new(
                chain_id,
                chain.rollup_config,
                chain.queries,
                chain.l1_queries,
                chain.safe_db,
                chain.el,
            ));
            observed.push(node_chain.clone());
            // The invalidation route: the archive the chain's engine already reads as its deny
            // list, and a rewind seam writing through the same controller queue promotions go
            // through.
            replacements.insert(
                chain_id,
                ChainReplacement {
                    archive: chain.archive,
                    chain: Arc::new(ChainRewindRoute::new(
                        node_chain.clone(),
                        chain.requests.clone(),
                    )) as Arc<dyn RewindableChain>,
                },
            );
            routes.push(ChainRoute {
                chain: node_chain,
                promoter: chain.promoter,
                requests: chain.requests,
            });
        }

        let l1 = Arc::new(L1Provider::new(RootProvider::new_http(l1_eth_rpc.clone())));
        let verifier = Verifier::<RocksKv>::new(
            observed,
            l1,
            verified,
            stores,
            verifier_config(activation_timestamp, message_expiry_window),
        )
        .and_then(|verifier| verifier.with_replacements(replacements))
        .map_err(|err| anyhow::anyhow!("failed to build the interop verifier: {err}"))?;

        info!(
            target: "lokahi",
            chains = routes.len(),
            activation = activation_timestamp,
            datadir = %interop_datadir.display(),
            "Interop verification enabled"
        );

        Ok(Self::new(verifier, routes))
    }
}

/// The verifier's configuration for an activation timestamp and the dependency set's
/// message-expiry window.
///
/// The window comes from the dependency set — op-supernode reads it off the first virtual-node
/// config that has one (`op-supernode/supernode/supernode.go:121-127`), and its `Interop`
/// substitutes the protocol default for an absent or zero value
/// (`op-supernode/supernode/activity/interop/interop.go:279-281`); kona's
/// `DependencySet::get_message_expiry_window` applies the same substitution, so the [`None`] here
/// is only a set with no dependency set at all. The backfill depth deliberately does *not* follow
/// an override: op-supernode's is an independent setting whose default stays the protocol window
/// (`interop.go:34-37`), and backfilling more than a shrunken window costs nothing while
/// backfilling less would leave sealed history short of what a resumed verifier checks for.
const fn verifier_config(
    activation_timestamp: u64,
    message_expiry_window: Option<u64>,
) -> VerifierConfig {
    let mut config = VerifierConfig::new(activation_timestamp);
    if let Some(window) = message_expiry_window {
        config.message_expiry_window = window;
    }
    config
}

#[cfg(test)]
mod tests {
    use super::*;
    use kona_genesis::HardForkConfig;

    fn config(lagoon_time: Option<u64>) -> RollupConfig {
        RollupConfig {
            hardforks: HardForkConfig { lagoon_time, ..Default::default() },
            ..Default::default()
        }
    }

    fn hosted(chain_id: ChainId, rollup_config: &RollupConfig) -> HostedChain<'_> {
        HostedChain { chain_id, rollup_config }
    }

    /// A rollup config with a genesis L2 time, for the cases that turn on where the first block
    /// is rather than only on what the fork says.
    fn config_from_genesis(lagoon_time: Option<u64>, genesis_l2_time: u64) -> RollupConfig {
        let mut config = config(lagoon_time);
        config.genesis.l2_time = genesis_l2_time;
        config
    }

    #[test]
    fn a_set_that_schedules_nothing_leaves_interop_off() {
        let (a, b) = (config(None), config(None));
        assert_eq!(
            InteropActor::activation(&[hosted(901, &a), hosted(902, &b)], None).unwrap(),
            None
        );
    }

    #[test]
    fn a_set_that_agrees_activates_at_that_timestamp() {
        let (a, b) = (config(Some(1_700)), config(Some(1_700)));
        assert_eq!(
            InteropActor::activation(&[hosted(901, &a), hosted(902, &b)], None).unwrap(),
            Some(1_700)
        );
    }

    /// One activation timestamp is the verifier's whole notion of when the cluster's rules turn
    /// on, so two disagreeing chains have no answer to give it.
    #[test]
    fn a_set_that_disagrees_on_the_time_is_rejected() {
        let (a, b) = (config(Some(1_700)), config(Some(1_800)));
        let err = InteropActor::activation(&[hosted(901, &a), hosted(902, &b)], None)
            .unwrap_err()
            .to_string();
        assert!(err.contains("901"), "{err}");
        assert!(err.contains("902"), "{err}");
    }

    /// Rounds are lockstep across the whole hosted set, so a chain that never activates interop
    /// would still be one the cluster waited for.
    #[test]
    fn a_partially_scheduled_set_is_rejected() {
        let (a, b) = (config(Some(1_700)), config(None));
        let err = InteropActor::activation(&[hosted(901, &a), hosted(902, &b)], None)
            .unwrap_err()
            .to_string();
        assert!(err.contains("do not schedule it at all"), "{err}");
    }

    /// A single-chain supernode is a legitimate interop deployment: the cluster is just small.
    #[test]
    fn a_single_scheduled_chain_activates() {
        let a = config(Some(42));
        assert_eq!(InteropActor::activation(&[hosted(901, &a)], None).unwrap(), Some(42));
    }

    // ---------------------------------------------------------------------------
    // The activation override
    //
    // The devstack's simple-interop presets hand op-supernode an activation timestamp and write
    // rollup configs with no Lagoon time, so reading the fork finds nothing to activate on. The
    // override is what lets lokahi be configured the way op-supernode already is.
    // ---------------------------------------------------------------------------

    /// The case the override exists for, and the one that fails without it: no chain schedules
    /// Lagoon, so `scheduled_activation` leaves interop off and the verifier never runs. With the
    /// override the same set activates at the timestamp it was given.
    #[test]
    fn an_override_activates_a_set_whose_configs_schedule_no_lagoon() {
        let (a, b) = (config(None), config(None));
        let chains = [hosted(901, &a), hosted(902, &b)];

        // Without it: interop off, which is what left these presets unrunnable.
        assert_eq!(InteropActor::activation(&chains, None).unwrap(), None);
        // With it: on, at the timestamp the caller knows.
        assert_eq!(
            InteropActor::activation(&chains, Some(1_787_335_282)).unwrap(),
            Some(1_787_335_282)
        );
    }

    /// An override alongside a fork scheduled at the same block: the devstack passes both,
    /// because op-supernode needs the timestamp and the chains carry the fork. Agreement is no
    /// longer required, but it is still the common case and still yields that timestamp.
    #[test]
    fn an_override_matching_the_scheduled_fork_is_accepted() {
        let a = config(Some(1_700));
        let b = config(Some(1_700));
        assert_eq!(
            InteropActor::activation(&[hosted(901, &a), hosted(902, &b)], Some(1_700)).unwrap(),
            Some(1_700)
        );
    }

    /// The two spellings of "from the first block": a fork at genesis is written as 0, while the
    /// timestamp handed to a supernode is the genesis timestamp. Both select the same blocks. The
    /// clamp survives the refusal it was written for, because it is what keeps the log line below
    /// quiet for a configuration that is actually consistent.
    #[test]
    fn an_override_at_genesis_time_matches_a_fork_scheduled_at_zero() {
        const GENESIS: u64 = 1_787_335_282;
        let a = config_from_genesis(Some(0), GENESIS);
        let b = config_from_genesis(Some(0), GENESIS);
        assert_eq!(
            InteropActor::activation(&[hosted(901, &a), hosted(902, &b)], Some(GENESIS)).unwrap(),
            Some(GENESIS)
        );
    }

    /// An override moves an activation away from a fork the chain schedules, and that is allowed:
    /// op-supernode returns the override without reading the rollup configs at all
    /// (`op-supernode/supernode/supernode.go:164`). This is the case the acceptance suites need --
    /// `TestInteropFaultProofs_ActivationBoundary` and `_PreForkActivation` schedule Lagoon well
    /// after the timestamp their preset supplies -- and the case an earlier version of this
    /// function refused.
    #[test]
    fn an_override_governs_a_fork_scheduled_somewhere_else() {
        const GENESIS: u64 = 1_787_335_282;
        let a = config_from_genesis(Some(GENESIS + 50), GENESIS);
        assert_eq!(
            InteropActor::activation(&[hosted(901, &a)], Some(GENESIS + 100)).unwrap(),
            Some(GENESIS + 100),
            "the override governs, not the scheduled fork"
        );

        // Including backwards, to before the scheduled fork: the acceptance tests that exercise
        // pre-activation behaviour need an activation earlier than the fork their chain carries.
        assert_eq!(
            InteropActor::activation(&[hosted(901, &a)], Some(GENESIS + 10)).unwrap(),
            Some(GENESIS + 10)
        );
    }

    /// A mixed set: one chain schedules the fork, another does not. The override supplies the
    /// missing one and agrees with the scheduled one, which is exactly what it is for -- and is
    /// refused without it, because a set cannot be half in the cluster.
    #[test]
    fn an_override_fills_in_the_chain_that_schedules_nothing() {
        let a = config(Some(1_700));
        let b = config(None);
        let chains = [hosted(901, &a), hosted(902, &b)];

        let err = InteropActor::activation(&chains, None).unwrap_err().to_string();
        assert!(err.contains("do not schedule it at all"), "{err}");

        assert_eq!(InteropActor::activation(&chains, Some(1_700)).unwrap(), Some(1_700));
    }

    /// An override also skips the cross-chain checks, because op-supernode skips them: the
    /// override returns before the loop that compares chains to each other, so a set that the
    /// scheduled path refuses as "not one cluster" is accepted when an activation is supplied.
    ///
    /// This is a real widening and worth stating plainly. It is not a judgement that the check
    /// was wrong -- the same set is still refused when no override is given, immediately below --
    /// but op-supernode's behaviour is the target, and op-supernode accepts this.
    #[test]
    fn an_override_skips_the_cross_chain_checks_as_op_supernode_does() {
        let a = config(Some(1_700));
        let b = config(Some(2_000));
        let chains = [hosted(901, &a), hosted(902, &b)];

        assert_eq!(
            InteropActor::activation(&chains, Some(1_700)).unwrap(),
            Some(1_700),
            "an override is taken without comparing the chains to each other"
        );

        // Unchanged without one: the scheduled path still refuses a set that is not one cluster.
        let err = InteropActor::activation(&chains, None)
            .expect_err("two different scheduled forks are not one cluster")
            .to_string();
        assert!(err.contains("activate together"), "{err}");
    }
}

#[cfg(test)]
mod store_tests {
    use super::*;
    use alloy_provider::RootProvider;
    use kona_engine::{Engine, EngineState, OpEngineClient};
    use kona_genesis::HardForkConfig;
    use lokahi_interop::{LOGS_DIR, VERIFIED_DIR};
    use tokio::sync::watch;

    /// The activation timestamp these tests build chains around.
    const ACTIVATION: u64 = 1_000;

    /// A promoter, obtained the only way one can be: from an engine built to be fed externally.
    ///
    /// The engine is never stepped and its client type is never used, so this is a real promoter
    /// without a real engine behind it — which is exactly the point of the capability being
    /// unforgeable.
    fn promoter() -> CrossSafePromoter {
        let (state_tx, _state_rx) = watch::channel(EngineState::default());
        let (len_tx, _len_rx) = watch::channel(0usize);
        Engine::<OpEngineClient<RootProvider, RootProvider<Optimism>>>::with_external_cross_safe(
            EngineState::default(),
            state_tx,
            len_tx,
        )
        .1
    }

    /// One chain's inputs, with its state under `datadir`.
    fn chain(chain_id: ChainId, datadir: PathBuf) -> Result<ChainInterop> {
        std::fs::create_dir_all(&datadir)?;
        let (queries, _queries_rx) = mpsc::channel(1);
        let (l1_queries, _l1_queries_rx) = mpsc::channel(1);
        let (requests, _requests_rx) = mpsc::channel(1);
        Ok(ChainInterop {
            chain_id,
            safe_db: ChainInterop::open_safe_db(&datadir)?,
            archive: ChainInterop::open_archive(&datadir)?,
            datadir,
            rollup_config: RollupConfig {
                block_time: 2,
                hardforks: HardForkConfig { lagoon_time: Some(ACTIVATION), ..Default::default() },
                l2_chain_id: chain_id.into(),
                ..Default::default()
            },
            el: RootProvider::<Optimism>::new_http(url::Url::parse("http://127.0.0.1:1/").unwrap()),
            queries,
            l1_queries,
            requests,
            promoter: promoter(),
        })
    }

    /// The stores are opened at startup and where the configuration says, so that a directory
    /// that cannot be opened is a startup failure rather than a supernode that runs and never
    /// promotes anything.
    ///
    /// The split is asserted as well as the success: the process-wide frontier goes under the
    /// interop directory, and each chain's log store under that chain's own, so one chain's state
    /// can be cleared without discarding the frontier or another chain's logs.
    #[tokio::test]
    async fn the_stores_open_where_the_configuration_puts_them() {
        let root = tempfile::tempdir().expect("temp dir");
        let interop_dir = root.path().join("interop");
        let chains = vec![
            chain(901, root.path().join("901")).expect("chain 901"),
            chain(902, root.path().join("902")).expect("chain 902"),
        ];

        let l1 = url::Url::parse("http://127.0.0.1:1/").unwrap();
        let actor = InteropActor::build(Some(&interop_dir), &l1, ACTIVATION, None, chains);
        assert!(actor.is_ok(), "{:?}", actor.err());

        assert!(interop_dir.join(VERIFIED_DIR).is_dir(), "the verified store is process-wide");
        for chain_id in [901, 902] {
            assert!(
                root.path()
                    .join(chain_id.to_string())
                    .join(LOGS_DIR)
                    .join(format!("chain-{chain_id}"))
                    .is_dir(),
                "chain {chain_id}'s log store lives in its own directory"
            );
        }
    }

    /// The verifier's frontier is a statement about the whole set, so it needs somewhere
    /// process-wide to live. A configuration that names only per-chain directories has nowhere,
    /// and has to be told so at startup rather than at the first round.
    #[tokio::test]
    async fn a_configuration_with_no_process_wide_directory_is_rejected() {
        let root = tempfile::tempdir().expect("temp dir");
        let chains = vec![chain(901, root.path().join("901")).expect("chain 901")];

        let l1 = url::Url::parse("http://127.0.0.1:1/").unwrap();
        let err =
            InteropActor::build(None, &l1, ACTIVATION, None, chains).expect_err("must be rejected");
        assert!(err.to_string().contains("`[defaults]`"), "{err}");
    }

    /// The dependency set's message-expiry window reaches the verifier's rules — that is where a
    /// devstack override like `WithMessageExpiryWindow(12)` travels — while the backfill depth
    /// keeps the protocol default, which is op-supernode's independent setting
    /// (`op-supernode/supernode/activity/interop/interop.go:34-37`).
    #[test]
    fn the_dependency_sets_expiry_window_reaches_the_verifier() {
        let defaults = verifier_config(ACTIVATION, None);
        assert_eq!(defaults, VerifierConfig::new(ACTIVATION));

        let overridden = verifier_config(ACTIVATION, Some(12));
        assert_eq!(overridden.message_expiry_window, 12);
        assert_eq!(overridden.activation_timestamp, ACTIVATION);
        assert_eq!(
            overridden.log_backfill_depth,
            VerifierConfig::new(ACTIVATION).log_backfill_depth,
            "the backfill depth is an independent setting and keeps its default"
        );
    }
}
