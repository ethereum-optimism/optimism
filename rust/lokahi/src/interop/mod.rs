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

use alloy_primitives::ChainId;
use alloy_provider::RootProvider;
use anyhow::{Context, Result, bail};
use kona_engine::CrossSafePromoter;
use kona_genesis::RollupConfig;
use kona_node_service::{ChainControllerRequest, ChainControllerRpcRequest};
use kona_safedb::{SafeDatabase, SharedSafeDb};
use lokahi_interop::{
    InteropChain, LogStores, RocksKv, Verifier, VerifierConfig, open_log_store, open_verified_store,
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
pub(crate) use chain::{L1Provider, NodeChain};

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
    /// The chain controller's request channel, which applies promotions.
    pub(crate) requests: mpsc::Sender<ChainControllerRequest>,
    /// The capability to promote this chain's cross-safe head.
    pub(crate) promoter: CrossSafePromoter,
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
    pub(crate) fn activation(chains: &[HostedChain<'_>]) -> Result<Option<u64>> {
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
                chain.safe_db,
                chain.el,
            ));
            observed.push(node_chain.clone());
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
            VerifierConfig::new(activation_timestamp),
        )
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

    #[test]
    fn a_set_that_schedules_nothing_leaves_interop_off() {
        let (a, b) = (config(None), config(None));
        assert_eq!(InteropActor::activation(&[hosted(901, &a), hosted(902, &b)]).unwrap(), None);
    }

    #[test]
    fn a_set_that_agrees_activates_at_that_timestamp() {
        let (a, b) = (config(Some(1_700)), config(Some(1_700)));
        assert_eq!(
            InteropActor::activation(&[hosted(901, &a), hosted(902, &b)]).unwrap(),
            Some(1_700)
        );
    }

    /// One activation timestamp is the verifier's whole notion of when the cluster's rules turn
    /// on, so two disagreeing chains have no answer to give it.
    #[test]
    fn a_set_that_disagrees_on_the_time_is_rejected() {
        let (a, b) = (config(Some(1_700)), config(Some(1_800)));
        let err =
            InteropActor::activation(&[hosted(901, &a), hosted(902, &b)]).unwrap_err().to_string();
        assert!(err.contains("901"), "{err}");
        assert!(err.contains("902"), "{err}");
    }

    /// Rounds are lockstep across the whole hosted set, so a chain that never activates interop
    /// would still be one the cluster waited for.
    #[test]
    fn a_partially_scheduled_set_is_rejected() {
        let (a, b) = (config(Some(1_700)), config(None));
        let err =
            InteropActor::activation(&[hosted(901, &a), hosted(902, &b)]).unwrap_err().to_string();
        assert!(err.contains("do not schedule it at all"), "{err}");
    }

    /// A single-chain supernode is a legitimate interop deployment: the cluster is just small.
    #[test]
    fn a_single_scheduled_chain_activates() {
        let a = config(Some(42));
        assert_eq!(InteropActor::activation(&[hosted(901, &a)]).unwrap(), Some(42));
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
        let (requests, _requests_rx) = mpsc::channel(1);
        Ok(ChainInterop {
            chain_id,
            safe_db: ChainInterop::open_safe_db(&datadir)?,
            datadir,
            rollup_config: RollupConfig {
                block_time: 2,
                hardforks: HardForkConfig { lagoon_time: Some(ACTIVATION), ..Default::default() },
                l2_chain_id: chain_id.into(),
                ..Default::default()
            },
            el: RootProvider::<Optimism>::new_http(url::Url::parse("http://127.0.0.1:1/").unwrap()),
            queries,
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
        let actor = InteropActor::build(Some(&interop_dir), &l1, ACTIVATION, chains);
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
        let err = InteropActor::build(None, &l1, ACTIVATION, chains).expect_err("must be rejected");
        assert!(err.to_string().contains("`[defaults]`"), "{err}");
    }
}
