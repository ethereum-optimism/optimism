//! Core proposer: state sync, canonical-head selection, and the
//! creation/resolution/bond-claim/defense task scheduler.
//!
//! Ported from op-succinct's `fault-proof/src/proposer.rs` (@ 13716c2c),
//! adapted for the super-root `ZKDisputeGame`: supernode-sourced claims,
//! `parentIndex || superRootProof` extraData, prestate-based ownership, and
//! two-phase `DelayedWETH` bond claiming. The defense scheduler proves
//! challenged games in the owned set via [`crate::proving`].

use std::{
    collections::{HashMap, HashSet},
    sync::{
        Arc,
        atomic::{AtomicBool, AtomicU64, Ordering},
    },
    time::{Duration, Instant},
};

use alloy_eips::{BlockId, BlockNumberOrTag};
use alloy_primitives::{Address, B256, U256};
use alloy_provider::{Provider, ProviderBuilder};
use alloy_sol_types::SolEvent;
use alloy_transport_http::reqwest::Url;
use anyhow::{Context, Result, anyhow, bail};
use kona_sp1_host_utils::metrics::MetricsGauge;
use kona_sp1_super_range_executor::HostInputs;
use sp1_sdk::HashableKey;
use tokio::{
    sync::{Mutex, OnceCell, RwLock},
    time,
};

use crate::{
    FactoryTrait, L1Provider, TX_REVERTED_PREFIX, TxErrorExt, ZK_GAME_TYPE,
    config::{PrestatePrograms, ProofProviderKind, ProposerConfig, load_prestate},
    contract::{
        AnchorStateRegistry::{self, AnchorStateRegistryInstance},
        DelayedWETH,
        DisputeGameFactory::{DisputeGameCreated, DisputeGameFactoryInstance},
        GameStatus, ProposalStatus, ZKDisputeGame, ZKGameArgs,
    },
    is_parent_resolved,
    metrics::ProposerGauge,
    prover::{ProofKeys, ProofProvider, setup_proof_keys},
    proving::{GameProofInputs, fetch_span_responses, is_unprovable, prove_game_inner},
    signer::{FeeCaps, SignerLock},
    superroot::{SuperrootClient, zk_extra_data},
};

/// Max allowed time (secs) between a game's deadline and the anchor game's deadline.
///
/// Games beyond this threshold are skipped during incremental syncs to cut startup latency and
/// avoid caching stale data.
///
/// The 14-day window is chosen with a 7-day challenge period in mind, plus a 7-day buffer,
/// ensuring all actionable games are included under normal conditions.
pub const MAX_GAME_DEADLINE_LAG: u64 = 60 * 60 * 24 * 14; // 14 days

/// Type alias for task ID
pub type TaskId = u64;

/// Type alias for task handles
pub type TaskHandle = tokio::task::JoinHandle<Result<()>>;

/// Type alias for a map of task IDs to their join handles and associated task info
pub type TaskMap = HashMap<TaskId, (TaskHandle, TaskInfo)>;

/// Information about a running task.
///
/// `GameCreation`, `GameResolution`, and `BondClaim` are singletons: at
/// most one task per variant runs at a time (see
/// `has_active_task_of_type`). `GameProving` is exempt from that rule and
/// deduplicated PER GAME instead: several games can be defended
/// concurrently, bounded by `MAX_CONCURRENT_DEFENSE_TASKS`.
#[derive(Clone, Debug)]
pub enum TaskInfo {
    /// Task creating a new game at the given super-root timestamp.
    GameCreation {
        /// Super-root timestamp (`l2SequenceNumber`) of the game being created.
        sequence_number: u64,
    },
    /// Task proving a game (defense today; the `is_defense` flag is the
    /// seam a future prove-at-creation mode reuses, see #22112).
    GameProving {
        /// Address of the game being proven.
        game_address: Address,
        /// Whether the proof was triggered by a challenge.
        is_defense: bool,
    },
    /// Task resolving finished games.
    GameResolution,
    /// Task unlocking and claiming game bonds.
    BondClaim,
}

/// Proposer identity for version tracking and startup logging.
#[derive(Clone, Debug)]
pub struct ProposerIdentity {
    /// Crate version string.
    pub version: String,
}

impl ProposerIdentity {
    /// Creates a new `ProposerIdentity`.
    pub fn new() -> Self {
        Self { version: env!("CARGO_PKG_VERSION").to_string() }
    }

    /// Logs the proposer identity and prestate artifact source at startup.
    pub fn log_startup_info(&self, prestates_url: &Url) {
        tracing::info!(
            version = %self.version,
            prestates_url = %crate::config::redacted_url(prestates_url),
            "proposer identity",
        );
    }
}

impl Default for ProposerIdentity {
    fn default() -> Self {
        Self::new()
    }
}

/// Represents a dispute game in the on-chain game DAG.
///
/// Games form a directed acyclic graph where each game builds upon a parent game, extending the
/// chain with a new proposed output root. The proposer tracks these games to determine when to
/// propose new games, defend existing ones, resolve completed games and claim bonds.
#[derive(Clone, Debug)]
pub struct Game {
    /// Index of the game in the `DisputeGameFactory`.
    pub index: U256,
    /// On-chain address of the game contract.
    pub address: Address,
    /// Factory index of the parent game this game extends (`u32::MAX` for anchor-rooted games).
    pub parent_index: u32,
    /// Super-root timestamp of the proposal (`l2SequenceNumber()`).
    pub l2_sequence_number: u64,
    /// On-chain lifecycle status of the game (in progress / resolved).
    pub status: GameStatus,
    /// ZK dispute game proposal status (unchallenged, challenged, proven, etc.).
    pub proposal_status: ProposalStatus,
    /// Claim deadline as an L1 timestamp in seconds.
    pub deadline: u64,
    /// Whether the proposer should try to resolve this game next resolution cycle.
    pub should_attempt_to_resolve: bool,
    /// Whether the proposer should try to claim this game's bond next claim cycle.
    pub should_attempt_to_claim_bond: bool,
    /// The game implementation's `absolutePrestate()` (the program vkey).
    /// Informational for the create path; the defend path selects the
    /// proving program by it.
    pub absolute_prestate: B256,
    /// The game's creator address (`gameCreator()`).
    pub creator: Address,
    /// The game's own `DelayedWETH` address, from its immutable args. Bond
    /// reads and claims bind this address, not the currently registered one:
    /// gameArgs can rotate across upgrades and each game holds its bond in
    /// the WETH it was created with.
    pub weth: Address,
    /// The game's own `AnchorStateRegistry` address, from its immutable
    /// args. Finality checks for this game bind this address.
    pub anchor_state_registry: Address,
}

impl Game {
    /// Returns true when the proposer can prove this game: its
    /// `absolutePrestate()` is in the known-prestates set (artifacts
    /// loadable from `PRESTATES_URL`, proving keys not poisoned).
    ///
    /// Ownership is prestate-based, not creator-based: the prove, resolve,
    /// and claim sets are the same set, and the creator is irrelevant to
    /// all three. Claims stay credit-driven, so iterating foreign games costs
    /// nothing where the proposer holds no credit. Games created before a
    /// prestate rotation stay owned because sync loads the prestate of
    /// every cached game, not just the registered one.
    pub fn is_owned(&self, known_prestates: &HashSet<B256>) -> bool {
        known_prestates.contains(&self.absolute_prestate)
    }
}

/// Central cache of the proposer's view of dispute games.
///
/// Tracks:
/// - `anchor_game`: the latest anchor fetched from the registry
/// - `canonical_head_index`/`canonical_head_sequence_number`: the best known game for scheduling
///   work
/// - `cursor`: Highest factory index processed in the prior sync. Each incremental sync walks
///   backward from the latest index to this value, then sets it to the new latest index.
/// - `games`: cached metadata for every tracked game keyed by index
/// - `invalid_games`: factory indices whose claims or ancestry are known to be invalid
#[derive(Default)]
struct ProposerState {
    anchor_game: Option<Game>,
    canonical_head_index: Option<U256>,
    canonical_head_sequence_number: Option<u64>,
    cursor: Cursor,
    games: HashMap<U256, Game>,
    invalid_games: HashSet<U256>,
}

impl ProposerState {
    /// Returns all game indices reachable from `root_index`, including the root.
    ///
    /// If this becomes hot, consider maintaining an adjacency index
    /// `parent -> Vec<child>`.
    fn descendants_of(&self, root_index: U256) -> HashSet<U256> {
        let mut reachable: HashSet<U256> = HashSet::new();
        let mut stack = vec![root_index];

        while let Some(index) = stack.pop() {
            if reachable.insert(index) {
                stack.extend(
                    self.games
                        .values()
                        .filter(|game| U256::from(game.parent_index) == index)
                        .map(|game| game.index),
                );
            }
        }

        reachable
    }

    /// Mark a game and every cached descendant as terminally invalid, then
    /// remove the subtree from the parent-eligible game cache.
    fn invalidate_subtree(&mut self, root_index: U256) {
        let invalid_subtree = self.descendants_of(root_index);
        self.invalid_games.extend(invalid_subtree.iter().copied());
        for index in invalid_subtree {
            tracing::info!(?index, "Removing invalid game from cache");
            self.games.remove(&index);
        }
    }

    /// Challenged, in-progress games as defense candidates, sorted by prove
    /// deadline ascending (closest to expiring first). Ownership and
    /// per-game dedup are applied by the defense scan.
    fn challenged_candidates(&self) -> Vec<(U256, Address, u64, B256)> {
        let mut candidates = self
            .games
            .values()
            .filter(|game| game.status == GameStatus::InProgress)
            .filter(|game| matches!(game.proposal_status, ProposalStatus::Challenged))
            .map(|game| (game.index, game.address, game.deadline, game.absolute_prestate))
            .collect::<Vec<_>>();
        candidates.sort_unstable_by_key(|(_, _, deadline, _)| *deadline);
        candidates
    }

    /// Drop all cached state tied to the prior factory history
    fn reset_factory_cache(&mut self) {
        self.anchor_game = None;
        self.canonical_head_index = None;
        self.cursor = Cursor::none();
        self.games.clear();
        self.invalid_games.clear();
    }

    /// Selects the canonical head: the highest-L2-timestamp game on the best valid chain.
    ///
    /// With no anchor, the head is simply the highest game in the cache. With an anchor, the head
    /// is the highest descendant of the anchor, unless a higher chain branches off earlier
    /// (genesis-rooted, or a lower parent index than the anchor head) — that alternative chain is
    /// then followed to its own tip.
    fn select_canonical_head(&self) -> Option<Game> {
        let Some(anchor_game) = self.anchor_game.as_ref() else {
            return self.games.values().max_by_key(|g| g.l2_sequence_number).cloned();
        };

        let reachable = self.descendants_of(anchor_game.index);

        // Best among the anchor's descendants.
        let anchor_head = self
            .games
            .values()
            .filter(|g| reachable.contains(&g.index))
            .max_by_key(|g| g.l2_sequence_number)
            .cloned();

        // Override with a higher non-descendant chain that branches off earlier than the anchor
        // head (genesis-rooted, or a lower parent index). Such a chain's root sits outside the
        // anchor's subtree, so we follow each qualifying root to its own highest-block tip rather
        // than stopping at the root — otherwise the head would pin to the root of a genesis-rooted
        // catch-up chain and stall instead of tracking its tip.
        let override_head = anchor_head.as_ref().and_then(|anchor| {
            let roots: Vec<U256> = self
                .games
                .values()
                .filter(|g| {
                    !reachable.contains(&g.index) &&
                        g.l2_sequence_number > anchor.l2_sequence_number &&
                        (g.parent_index == u32::MAX || g.parent_index < anchor.parent_index)
                })
                .map(|g| g.index)
                .collect();

            roots
                .into_iter()
                .flat_map(|root| self.descendants_of(root))
                .filter_map(|idx| self.games.get(&idx))
                .max_by_key(|g| g.l2_sequence_number)
                .cloned()
        });

        // Log when the head comes from a catch-up chain so operators can tell the proposer is
        // recovering along a chain outside the anchor subtree.
        if let Some(head) = override_head.as_ref() {
            tracing::debug!(
                head_index = %head.index,
                head_l2_block = %head.l2_sequence_number,
                "Canonical head selected from a catch-up chain outside the anchor subtree"
            );
        }

        override_head.or(anchor_head)
    }
}

/// An unresolved game creation: the exact bytes of a `create` whose
/// confirmation timed out, so the transaction may still land. While a
/// record is set, new proposals are held; each tick the factory is
/// checked for the uuid (adopting our landed game) and the record clears
/// once the signer's pool holds no transactions, i.e. nothing of ours can
/// land anymore. The hold plus the factory's dedup on (gameType,
/// rootClaim, extraData) keep a stuck-then-included original from being
/// joined by a sibling at a fresh timestamp. In-memory only: a restart
/// with a create in flight keeps the pre-existing documented
/// double-submit-once risk.
#[derive(Clone, Debug)]
struct InFlightCreation {
    /// Root claim of the unresolved create.
    root_claim: B256,
    /// Exact extraData bytes sent (pins parent, timestamp, and proof).
    extra_data: Vec<u8>,
    /// Super-root timestamp of the unresolved create (guard arming).
    sequence_number: u64,
    /// Parent game index, for logging.
    parent_game_index: u32,
}
/// Core proposer service: syncs the on-chain game DAG, creates and defends games,
/// resolves finished ones, and claims bonds.
#[derive(Clone)]
pub struct Proposer<P>
where
    P: Provider + Clone + Send + Sync + 'static,
{
    /// Proposer configuration loaded at startup.
    pub config: ProposerConfig,
    /// Shared transaction signer, serialized behind a lock.
    pub signer: SignerLock,
    /// L1 execution-layer provider used for contract reads and transactions.
    pub l1_provider: L1Provider,
    /// Supernode client used to fetch super roots and safe-head data.
    pub superroot_client: SuperrootClient,
    /// `DisputeGameFactory` contract instance used to create and enumerate games.
    pub factory: Arc<DisputeGameFactoryInstance<P>>,
    /// Prestate program cache: loaded ELFs keyed by `absolutePrestate()`
    /// hash, fetched from `PRESTATES_URL` on demand (see [`PrestateCache`]).
    pub prestates: Arc<PrestateCache>,
    tasks: Arc<tokio::sync::Mutex<TaskMap>>,
    next_task_id: Arc<AtomicU64>,
    state: Arc<RwLock<ProposerState>>,
    /// Proposer identity for foreign-game filtering and hardfork safety.
    pub identity: ProposerIdentity,
    /// L1 block number used in the last successful sync cycle. Sync is skipped when the
    /// pinned block hasn't advanced past this value.
    last_synced_l1_block: Arc<AtomicU64>,
    /// Sequence number of the most recently created game. Used to prevent duplicate
    /// game creation when the pinned sync cache lags behind the chain tip.
    last_created_game_l2_sequence_number: Arc<AtomicU64>,
    /// Address of the most recently created game. Used to precisely identify
    /// the guarded game for `ChallengerWins` subtree removal.
    last_created_game_address: Arc<tokio::sync::Mutex<Address>>,
    /// Exact bytes of a create whose confirmation timed out and whose fate
    /// is unknown (see [`InFlightCreation`]).
    in_flight_creation: Arc<tokio::sync::Mutex<Option<InFlightCreation>>>,
    /// Games seen on-chain whose super-root data is not yet obtainable from
    /// this node - the timestamp is not yet safe, or the query failed
    /// (including permanently, e.g. timestamps predating the node's recorded
    /// safe history) - and our own games whose claim contradicts the current
    /// supernode answer. Excluded from the DAG (and parent eligibility) but
    /// re-validated each sync - unlike terminal invalidity, pending games
    /// must not be dropped by the cursor (see `fetch_game`).
    pending_games: Arc<RwLock<HashSet<U256>>>,
    /// The proof provider defending challenged games.
    pub proof_provider: ProofProvider,
    /// Deployment-scoped witness endpoints and config paths for the
    /// `InteropHost` (built once from the config).
    host_inputs: Arc<HostInputs>,
    /// The registered game args' `maxProveDuration`, read once during
    /// `try_init`. Used for the deadline-approaching warning tier only;
    /// per-game deadlines come from `claimData` each sync.
    max_prove_duration: Arc<OnceCell<u64>>,
    /// Games found permanently unprovable (claim diverged from the
    /// supernode view, or required L1 beyond the game's L1 head). Skipped
    /// by the defense scan without re-fetching their spans. In-memory
    /// only: a restart re-evaluates (upstream-parity statelessness).
    undefendable: Arc<Mutex<HashSet<Address>>>,
}

impl<P> std::fmt::Debug for Proposer<P>
where
    P: Provider + Clone + Send + Sync + 'static,
{
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Proposer")
            .field("config", &self.config)
            .field("identity", &self.identity)
            .finish_non_exhaustive()
    }
}

impl<P> Proposer<P>
where
    P: Provider + Clone + Send + Sync + 'static,
{
    /// Creates a new proposer instance with the provided signer, factory
    /// contract instance, and proof provider.
    pub async fn new(
        config: ProposerConfig,
        signer: SignerLock,
        factory: DisputeGameFactoryInstance<P>,
        proof_provider: ProofProvider,
    ) -> Result<Self> {
        let identity = ProposerIdentity::new();
        identity.log_startup_info(&config.prestates_url);

        let l1_provider = ProviderBuilder::default().connect_http(config.l1_rpc.clone());
        let superroot_client = SuperrootClient::new(config.supernode_rpc.as_str())?;

        let prestates = Arc::new(PrestateCache::new(config.prestates_url.clone()));
        let host_inputs = Arc::new(HostInputs {
            l1_node_address: config.l1_rpc.to_string(),
            l1_beacon_address: config.l1_beacon_rpc.to_string(),
            l2_node_addresses: config.l2_rpcs.iter().map(Url::to_string).collect(),
            rollup_config_paths: config.rollup_config_paths.clone(),
            l1_config_path: config.l1_config_path.clone(),
            dependency_set_path: config.dependency_set_path.clone(),
        });
        Ok(Self {
            config,
            signer,
            l1_provider,
            superroot_client,
            factory: Arc::new(factory),
            prestates,
            tasks: Arc::new(tokio::sync::Mutex::new(HashMap::new())),
            next_task_id: Arc::new(AtomicU64::new(1)),
            state: Arc::new(RwLock::new(ProposerState::default())),
            identity,
            last_synced_l1_block: Arc::new(AtomicU64::new(0)),
            last_created_game_l2_sequence_number: Arc::new(AtomicU64::new(0)),
            last_created_game_address: Arc::new(tokio::sync::Mutex::new(Address::ZERO)),
            in_flight_creation: Arc::new(tokio::sync::Mutex::new(None)),
            pending_games: Arc::new(RwLock::new(HashSet::new())),
            proof_provider,
            host_inputs,
            max_prove_duration: Arc::new(OnceCell::new()),
            undefendable: Arc::new(Mutex::new(HashSet::new())),
        })
    }

    /// Runs the proposer indefinitely.
    pub async fn run(self: Arc<Self>) -> Result<()> {
        self.try_init().await?;

        // Spawn a dedicated task for continuous metrics collection
        self.spawn_metrics_collector();

        let mut interval = time::interval(Duration::from_secs(self.config.fetch_interval));
        loop {
            interval.tick().await;

            // 1. Synchronize cached dispute state before scheduling work.
            if let Err(e) = self.sync_state().await {
                tracing::warn!("Failed to sync proposer state: {:?}", e);
                continue;
            }

            // 2. Handle completed tasks.
            if let Err(e) = self.handle_completed_tasks().await {
                tracing::warn!("Failed to handle completed tasks: {:?}", e);
            }

            // 3. Spawn new work (non-blocking).
            if let Err(e) = self.spawn_pending_operations().await {
                tracing::warn!("Failed to spawn pending operations: {:?}", e);
            }

            // 4. Log task statistics.
            self.log_task_stats().await;
        }
    }

    /// Runs startup validations with retries before entering main loop.
    pub async fn try_init(&self) -> Result<()> {
        let mut interval = time::interval(Duration::from_secs(self.config.fetch_interval));
        let mut retry_count = 0u32;

        loop {
            match self.validate_and_init().await {
                Ok(()) => break,
                Err(e) => {
                    retry_count += 1;
                    if retry_count == 1 {
                        tracing::error!(attempt = retry_count, error = %e, "Startup validations failed");
                    } else {
                        tracing::warn!(
                            attempt = retry_count,
                            error = %e,
                            "Startup validations still pending, retrying..."
                        );
                    }
                    interval.tick().await;
                }
            }
        }

        Ok(())
    }

    /// Validates startup and initializes state.
    pub async fn validate_and_init(&self) -> Result<()> {
        let anchor_sequence_number = self.startup_validations().await?;
        self.state.write().await.canonical_head_sequence_number = Some(anchor_sequence_number);
        Ok(())
    }

    /// Runs one-time startup validations before the proposer begins normal operations.
    /// Returns the validated anchor sequence number.
    async fn startup_validations(&self) -> Result<u64> {
        // Validate the registered game args decode and load the registered
        // prestate's programs.
        let game_args = self.registered_game_args(BlockId::latest()).await?;
        if !self.prestates.ensure_loaded(game_args.absolute_prestate).await {
            // Not fatal: creation stays paused until the artifacts appear
            // under PRESTATES_URL (PrestateCache::ensure_loaded logged why).
            tracing::warn!(
                registered = %game_args.absolute_prestate,
                "registered prestate programs unavailable; creation will pause"
            );
        }

        // Record the registered maxProveDuration for the defense scheduler's
        // deadline-approaching warning tier (per-game deadlines themselves
        // come from claimData each sync).
        let _ = self.max_prove_duration.set(game_args.max_prove_duration);

        // Fetch and validate the anchor root from the currently registered registry.
        let registry =
            AnchorStateRegistry::new(game_args.anchor_state_registry, self.l1_provider.clone());
        let anchor = registry.getAnchorRoot().call().await?;
        let (anchor_root, anchor_sequence_number) = (anchor._0, anchor._1);
        anyhow::ensure!(
            anchor_root != B256::ZERO,
            "anchor state registry has no anchor root (game creation would revert)"
        );
        anchor_sequence_number.try_into().context("anchor sequence number exceeds u64")
    }

    /// Fetches and decodes the currently registered game args at `block`.
    ///
    /// Read per use rather than pinned at startup: gameArgs can rotate across
    /// upgrades, and the addresses inside (`AnchorStateRegistry`, `DelayedWETH`)
    /// are only authoritative for games created under them. Interactions with
    /// a specific game bind that game's own args instead (see `Game`).
    async fn registered_game_args(&self, block: BlockId) -> Result<ZKGameArgs> {
        let raw = self.factory.gameArgs(ZK_GAME_TYPE).block(block).call().await?;
        ZKGameArgs::decode(&raw)
            .with_context(|| format!("game type {ZK_GAME_TYPE} has no valid registered game args"))
    }

    /// Binds the currently registered `AnchorStateRegistry` at `block`.
    async fn registered_anchor_state_registry(
        &self,
        block: BlockId,
    ) -> Result<AnchorStateRegistryInstance<L1Provider>> {
        let args = self.registered_game_args(block).await?;
        Ok(AnchorStateRegistry::new(args.anchor_state_registry, self.l1_provider.clone()))
    }

    /// Synchronizes the proposer's cached view of the dispute-game tree with the on-chain state.
    ///
    /// Steps run in order:
    /// 1. `sync_games` pulls newly created games and refreshes cached metadata.
    /// 2. `sync_anchor_game` aligns the cached anchor pointer with the registry contract.
    /// 3. `compute_canonical_head` recomputes the head game used for proposal selection.
    pub async fn sync_state(&self) -> Result<()> {
        // Pin L1 block for the entire sync cycle so all state reads see a consistent
        // snapshot. Without this, load-balanced RPCs can return data from different block
        // heights, breaking atomicity between related reads (e.g. credit vs anchorGame).
        // Ref: https://github.com/celo-org/op-succinct/issues/132
        let latest_block = self
            .l1_provider
            .get_block_by_number(BlockNumberOrTag::Latest)
            .await?
            .context("Failed to fetch latest L1 block")?;
        let confirmed_number =
            latest_block.header.number.saturating_sub(self.config.sync_l1_confirmations);

        // If L1 hasn't advanced past the last synced block, all on-chain state is identical.
        //
        // `confirmed_number < prev` indicates backend regression from a load-balanced RPC, or a
        // deep L1 reorg past `sync_l1_confirmations`. This case should be logged at WARN so
        // operators can detect unhealthy backends or L1 reorg; the equal case stays at DEBUG since
        // it's the normal "L1 hasn't ticked" path.
        let prev = self.last_synced_l1_block.load(Ordering::Relaxed);
        if confirmed_number > 0 && confirmed_number <= prev {
            if confirmed_number < prev {
                tracing::warn!(
                    confirmed_number,
                    last_synced = prev,
                    "L1 confirmed head moved backwards (backend regression or deep reorg), skipping sync"
                );
            } else {
                tracing::debug!(
                    confirmed_number,
                    last_synced = prev,
                    "L1 head unchanged, skipping sync"
                );
            }
            return Ok(());
        }

        // When no confirmation offset, use the latest block directly (single RPC response).
        // When offset > 0, fetch the confirmed block separately; if the backend hasn't
        // caught up, skip this cycle rather than pinning forward.
        let (pinned_block, pinned_timestamp) = if self.config.sync_l1_confirmations == 0 {
            (BlockId::number(latest_block.header.number), latest_block.header.timestamp)
        } else {
            match self
                .l1_provider
                .get_block_by_number(BlockNumberOrTag::Number(confirmed_number))
                .await?
            {
                Some(block) => (BlockId::number(block.header.number), block.header.timestamp),
                None => {
                    tracing::warn!(
                        confirmed_number,
                        "Confirmed block not available on this backend, skipping sync cycle"
                    );
                    return Ok(());
                }
            }
        };

        // Pull new games and synchronize cached game statuses.
        self.sync_games(pinned_block, pinned_timestamp).await?;

        // Align anchor information after the cached game statuses have been synchronized.
        self.sync_anchor_game(pinned_block).await?;

        // With the cached game statuses and anchor synchronized, recompute the canonical head.
        self.compute_canonical_head().await;

        self.last_synced_l1_block.store(confirmed_number, Ordering::Relaxed);

        Ok(())
    }

    /// Synchronizes the game cache.
    ///
    /// 1. Discover new games: walk the factory backwards from the latest game to the cursor,
    ///    classifying each as valid / unsupported / invalid / pending, and stopping early once past
    ///    the anchor's deadline-lag cutoff. A fetch failure aborts the sync cycle (the cursor is
    ///    not advanced, so the range is re-walked next cycle).
    /// 2. Remove invalid games and their subtrees.
    /// 3. Re-validate pending games (timestamps not yet safe from this node's view, unavailable
    ///    super-root data, or an own-game claim mismatch); entries still pending past the anchor's
    ///    deadline-lag cutoff are evicted.
    /// 4. Synchronize the status of all cached games and apply actions: mark own games for
    ///    resolution (parent resolved in the defender's favor, game over), mark `DefenderWins`
    ///    games for bond claiming (finalized with credit, or a matured withdrawal), remove finished
    ///    games (keeping the anchor and the canonical head), and remove the entire subtree of a
    ///    `ChallengerWins` game (resetting the duplicate-creation guard when the tracked game is
    ///    inside it). Per-game read failures skip only that game for the cycle.
    pub async fn sync_games(&self, pinned_block: BlockId, pinned_timestamp: u64) -> Result<()> {
        let pinned_latest_index = self.factory.fetch_latest_game_index(pinned_block).await?;
        ProposerGauge::FactoryLatestGameIndex
            .set(pinned_latest_index.map_or(-1.0, |i| i.to::<u64>() as f64));

        // 1. Load new games.
        let latest_index = if let Some(index) = pinned_latest_index {
            Cursor::from(index)
        } else {
            // No games at the pinned block. Drop index-keyed caches so future
            // cycles cannot reuse entries from a prior factory history.
            {
                let mut state = self.state.write().await;
                state.reset_factory_cache();
            }
            self.pending_games.write().await.clear();
            ProposerGauge::SyncCursor.set(-1.0);
            return Ok(());
        };

        let anchor_address = self
            .registered_anchor_state_registry(pinned_block)
            .await?
            .anchorGame()
            .block(pinned_block)
            .call()
            .await?;

        let (cursor, factory_reset) = {
            let mut state = self.state.write().await;
            let current_cursor = state.cursor.clone();

            // This should never/rarely happen but in a case where the factory is redeployed/reset
            // while the proposer keeps running, the cursor is reset to zero to avoid skipping
            // any games.
            if latest_index < current_cursor {
                tracing::warn!(
                    latest_index = %latest_index,
                    current_cursor = %current_cursor,
                    "Factory reset suspected; resetting cursor to 0"
                );
                state.reset_factory_cache();
                (Cursor::none(), true)
            } else {
                (current_cursor, false)
            }
        };

        if factory_reset {
            self.pending_games.write().await.clear();
        }

        let mut index = latest_index.clone();
        let mut anchor_deadline: Option<u64> = None;
        let mut invalid_game_ids = Vec::new();
        let mut newly_pending: Vec<U256> = Vec::new();
        loop {
            if index == cursor {
                break;
            }

            let i = index.index().expect("must have an index here");
            let fetch_result = match self.fetch_game(i, pinned_block).await {
                Ok(result) => result,
                Err(e) => {
                    // A failed fetch aborts the whole cycle: acting on a
                    // partially discovered topology could select a parent
                    // whose unfetched ancestry is invalid. The cursor stays
                    // put so the range is re-walked next cycle; persistent
                    // failure is visible via the warn, the game_sync_error
                    // counter, and a flat sync cursor.
                    tracing::warn!(
                        game_index = %index,
                        error = %e,
                        "Game fetch failed; aborting the sync cycle"
                    );
                    ProposerGauge::GameSyncError.increment(1.0);
                    return Err(e);
                }
            };

            match fetch_result {
                GameFetchResult::ValidGame { game_address, deadline } => {
                    // First time we hit the anchor, record its deadline
                    if game_address == anchor_address {
                        anchor_deadline = Some(deadline);
                    }

                    // Once we know the anchor deadline, enforce the lag constraint.
                    if let Some(anchor_d) = anchor_deadline &&
                        beyond_deadline_lag(anchor_d, deadline)
                    {
                        tracing::debug!(
                            game_index = %index,
                            game_address = ?game_address,
                            game_deadline = %deadline,
                            anchor_deadline = %anchor_d,
                            "Game deadline exceeds max lag from anchor: stopping incremental fetch"
                        );
                        break;
                    }
                }
                GameFetchResult::UnsupportedType { game_address } => {
                    // Stop fetching once we find the anchor on an unsupported game.
                    if game_address == anchor_address {
                        break;
                    }
                }
                GameFetchResult::AlreadyExists => {}
                GameFetchResult::InvalidGame { index } => {
                    invalid_game_ids.push(index);
                }
                GameFetchResult::Pending { index, .. } => {
                    newly_pending.push(index);
                }
            }

            index.step_back();
        }

        // The loop only completes on a full walk (a fetch failure returns
        // above), so the cursor advance is unconditional; re-walked
        // processed games hit `AlreadyExists`, which is idempotent.
        ProposerGauge::SyncCursor.set(latest_index.index().map_or(-1.0, |i| i.to::<u64>() as f64));
        {
            let mut state = self.state.write().await;
            state.cursor = latest_index;
        }

        if !invalid_game_ids.is_empty() {
            let mut state = self.state.write().await;
            for idx in invalid_game_ids {
                tracing::warn!(
                    game_index = %idx,
                    "Removing invalid game and its subtree from cache"
                );
                state.invalidate_subtree(idx);
            }
        }

        // Re-validate previously pending games and record new ones
        // (FlowGaps #1): a pending game sits outside the DAG until its
        // timestamp becomes safe from this node's view; the verdict must not
        // be made permanent by the cursor.
        {
            let previously_pending: Vec<U256> = {
                let pending = self.pending_games.read().await;
                pending.iter().copied().filter(|idx| !newly_pending.contains(idx)).collect()
            };
            self.pending_games.write().await.extend(newly_pending.iter().copied());
            // Anchor deadline for the eviction cutoff: prefer the walk's
            // observation this cycle, fall back to the cached anchor game.
            // Neither known -> skip eviction this cycle (it is an
            // optimization; nothing is lost by waiting).
            let anchor_deadline_for_eviction = match anchor_deadline {
                Some(deadline) => Some(deadline),
                None => self.state.read().await.anchor_game.as_ref().map(|g| g.deadline),
            };
            for idx in previously_pending {
                match self.fetch_game(idx, pinned_block).await {
                    Ok(GameFetchResult::Pending { deadline, prestate, .. }) => {
                        // Owned games (prestate in the usable set) are
                        // exempt from eviction: an evicted own game loses
                        // defense, resolution, and bond tracking until a
                        // restart after the supernode heals. The
                        // ensure_loaded attempt is bounded by the
                        // negative-cache window for unknown prestates.
                        let owned = {
                            let _ = self.prestates.ensure_loaded(prestate).await;
                            self.prestates.known_prestates().await.contains(&prestate)
                        };
                        if owned {
                            tracing::debug!(
                                game_index = %idx,
                                "Keeping pending owned game re-checkable (eviction exempt)"
                            );
                        } else if let Some(anchor_d) = anchor_deadline_for_eviction &&
                            pending_evictable(anchor_d, deadline)
                        {
                            tracing::warn!(
                                game_index = %idx,
                                game_deadline = deadline,
                                anchor_deadline = anchor_d,
                                "Evicting pending game whose deadline fell behind the anchor beyond the lag cutoff"
                            );
                            self.pending_games.write().await.remove(&idx);
                        }
                    }
                    Ok(GameFetchResult::InvalidGame { index }) => {
                        self.pending_games.write().await.remove(&index);
                        self.state.write().await.invalidate_subtree(index);
                    }
                    Ok(_) => {
                        self.pending_games.write().await.remove(&idx);
                    }
                    Err(e) => {
                        tracing::warn!(
                            game_index = %idx,
                            error = %e,
                            "Pending game re-validation failed; retrying next cycle"
                        );
                        ProposerGauge::GameSyncError.increment(1.0);
                    }
                }
            }
        }

        // 2. Synchronize the status of all cached games.
        let games = {
            let state = self.state.read().await;
            state
                .games
                .values()
                .map(|game| {
                    (
                        game.index,
                        game.address,
                        game.weth,
                        game.anchor_state_registry,
                        game.absolute_prestate,
                    )
                })
                .collect::<Vec<_>>()
        };

        if !games.is_empty() {
            // Load the prestate programs of every cached game, not just the
            // registered one, so games created before a prestate rotation
            // keep their ownership (prove = resolve = claim set). The
            // negative cache bounds re-fetch attempts for unknown
            // prestates.
            let mut prestates: Vec<B256> =
                games.iter().map(|(_, _, _, _, prestate)| *prestate).collect();
            prestates.sort_unstable();
            prestates.dedup();
            for prestate in prestates {
                let _ = self.prestates.ensure_loaded(prestate).await;
            }
            let known_prestates = self.prestates.known_prestates().await;

            let now_ts = pinned_timestamp;
            let signer_address = self.signer.address();

            enum GameSyncAction {
                Update {
                    index: U256,
                    status: GameStatus,
                    proposal_status: ProposalStatus,
                    deadline: u64,
                    should_attempt_to_resolve: bool,
                    should_attempt_to_claim_bond: bool,
                },
                Remove(U256),
                RemoveSubtree(U256),
            }

            let mut actions = Vec::with_capacity(games.len());

            for (index, game_address, game_weth, game_asr, absolute_prestate) in games {
                let synced: Result<()> = async {
                let contract = ZKDisputeGame::new(game_address, self.l1_provider.clone());
                let claim_data = contract.claimData().block(pinned_block).call().await?;
                // Enums are uint8 in the ABI; convert once at the read boundary.
                let proposal_status = ProposalStatus::try_from(claim_data.status)?;
                let status =
                    GameStatus::try_from(contract.status().block(pinned_block).call().await?)?;
                let deadline = claim_data.deadline;
                let parent_index = claim_data.parentIndex;

                // Bind the game's own registry: finality is defined by the
                // registry the game was created under, which can differ from
                // the currently registered one across upgrades.
                let is_finalized = AnchorStateRegistry::new(game_asr, self.l1_provider.clone())
                    .isGameFinalized(game_address)
                    .block(pinned_block)
                    .call()
                    .await?;

                match status {
                    GameStatus::InProgress => {
                        let parent_resolved =
                            is_parent_resolved(parent_index, self.factory.as_ref(), pinned_block)
                                .await?;
                        // Proof-provided statuses (whether set by us or by a
                        // third party, since prove() is permissionless) make
                        // the game "over" for resolution purposes.
                        let is_game_over = match proposal_status {
                            ProposalStatus::Unchallenged => now_ts > deadline,
                            ProposalStatus::UnchallengedAndValidProofProvided |
                            ProposalStatus::ChallengedAndValidProofProvided => true,
                            _ => false,
                        };

                        // Cached games already passed game-type validation
                        // in fetch_game. Ownership is prestate-based: the
                        // resolve set equals the willing-to-prove set.
                        let should_attempt_to_resolve = parent_resolved &&
                            is_game_over &&
                            known_prestates.contains(&absolute_prestate);

                        actions.push(GameSyncAction::Update {
                            index,
                            status,
                            proposal_status,
                            deadline,
                            should_attempt_to_resolve,
                            should_attempt_to_claim_bond: false,
                        });
                    }
                    GameStatus::DefenderWins => {
                        let credit =
                            contract.credit(signer_address).block(pinned_block).call().await?;
                        // Bind the game's own DelayedWETH: its bond lives in
                        // the WETH it was created with.
                        let weth = DelayedWETH::new(game_weth, self.l1_provider.clone());
                        let withdrawal = weth
                            .withdrawals(game_address, signer_address)
                            .block(pinned_block)
                            .call()
                            .await?;
                        let withdrawal_amount = withdrawal.amount;
                        let withdrawal_ts: u64 =
                            withdrawal.timestamp.try_into().unwrap_or(u64::MAX);
                        let weth_delay: u64 = weth
                            .delay()
                            .block(pinned_block)
                            .call()
                            .await?
                            .try_into()
                            .context("DelayedWETH delay exceeds u64")?;

                        let action = bond_claim_action(
                            is_finalized,
                            credit,
                            withdrawal_amount,
                            withdrawal_ts,
                            weth_delay,
                            now_ts,
                        );
                        let done = action == BondClaimAction::Done;

                        if done {
                            // Game removal policy:
                            // - Canonical head games are retained even with zero credit to maintain
                            //   chain consistency.
                            // - Anchor games are retained as they serve as the root of the dispute
                            //   game tree.
                            // - All other games with bonds already claimed are removed to free
                            //   cache memory.
                            let canonical_head_index = {
                                let state = self.state.read().await;
                                state.canonical_head_index
                            };

                            let should_remove = if canonical_head_index == Some(index) {
                                tracing::debug!(game_index = %index, "Retaining game: canonical head");
                                false
                            } else if anchor_address == game_address {
                                tracing::debug!(game_index = %index, "Retaining game: anchor game");
                                false
                            } else {
                                true
                            };

                            if should_remove {
                                actions.push(GameSyncAction::Remove(index));
                            } else {
                                actions.push(GameSyncAction::Update {
                                    index,
                                    status,
                                    proposal_status,
                                    deadline,
                                    should_attempt_to_resolve: false,
                                    should_attempt_to_claim_bond: false,
                                });
                            }
                        } else {
                            actions.push(GameSyncAction::Update {
                                index,
                                status,
                                proposal_status,
                                deadline,
                                should_attempt_to_resolve: false,
                                should_attempt_to_claim_bond: matches!(
                                    action,
                                    BondClaimAction::Unlock | BondClaimAction::Payout
                                ),
                            });
                        }
                    }
                    GameStatus::ChallengerWins => {
                        actions.push(GameSyncAction::RemoveSubtree(index));
                    }
                }
                Ok(())
                }
                .await;
                if let Err(e) = synced {
                    tracing::warn!(
                        game_index = %index,
                        error = %e,
                        "Game status sync failed; skipping this game for this cycle"
                    );
                    ProposerGauge::GameSyncError.increment(1.0);
                }
            }

            let mut state = self.state.write().await;
            for action in actions {
                match action {
                    GameSyncAction::Update {
                        index,
                        status,
                        proposal_status,
                        deadline,
                        should_attempt_to_resolve,
                        should_attempt_to_claim_bond,
                    } => {
                        if let Some(game) = state.games.get_mut(&index) {
                            game.status = status;
                            game.proposal_status = proposal_status;
                            game.deadline = deadline;
                            game.should_attempt_to_resolve = should_attempt_to_resolve;
                            game.should_attempt_to_claim_bond = should_attempt_to_claim_bond;
                        }
                    }
                    GameSyncAction::Remove(index) => {
                        state.games.remove(&index);
                        tracing::debug!(game_index = %index, "Removed game from cache");
                    }
                    GameSyncAction::RemoveSubtree(index) => {
                        // Reset the duplicate-creation guard if the subtree being
                        // removed contains the exact game we created (matched by
                        // address, which is globally unique and race-free).
                        let guarded_addr = *self.last_created_game_address.lock().await;
                        if guarded_addr != Address::ZERO {
                            let subtree = state.descendants_of(index);
                            let guard_in_subtree =
                                std::iter::once(&index).chain(subtree.iter()).any(|idx| {
                                    state.games.get(idx).is_some_and(|g| g.address == guarded_addr)
                                });
                            if guard_in_subtree {
                                self.last_created_game_l2_sequence_number
                                    .store(0, Ordering::Relaxed);
                                *self.last_created_game_address.lock().await = Address::ZERO;
                                tracing::info!(
                                    ?guarded_addr,
                                    root_index = %index,
                                    "Reset creation guard: tracked game removed by ChallengerWins"
                                );
                            }
                        }
                        state.invalidate_subtree(index);
                    }
                }
            }
        }

        Ok(())
    }

    /// Synchronizes the anchor game from the registry.
    async fn sync_anchor_game(&self, pinned_block: BlockId) -> Result<()> {
        let anchor_address = self
            .registered_anchor_state_registry(pinned_block)
            .await?
            .anchorGame()
            .block(pinned_block)
            .call()
            .await?;

        let mut state = self.state.write().await;

        if anchor_address == Address::ZERO {
            state.anchor_game = None;
        } else if let Some((_, anchor_game)) =
            state.games.iter().find(|(_, game)| game.address == anchor_address)
        {
            state.anchor_game = Some(anchor_game.clone());
            tracing::debug!(?anchor_address, "Anchor game updated in cache");
        } else {
            // Anchor not in cache (pruned or not yet fetched) — clear to prevent
            // compute_canonical_head from following a stale subtree.
            state.anchor_game = None;
            tracing::debug!(?anchor_address, "Anchor game not in cache, clearing");
        }

        Ok(())
    }

    /// Computes and stores the canonical head used to schedule new proposals, logging on change.
    async fn compute_canonical_head(&self) {
        let mut state = self.state.write().await;

        let canonical_head = state.select_canonical_head();

        let previous_canonical_index = state.canonical_head_index;

        if let Some(canonical_head) = canonical_head {
            state.canonical_head_index = Some(canonical_head.index);
            state.canonical_head_sequence_number = Some(canonical_head.l2_sequence_number);

            if previous_canonical_index != state.canonical_head_index {
                tracing::info!(
                    previous_canonical_index = ?previous_canonical_index,
                    new_canonical_index = %canonical_head.index,
                    l2_sequence_number = %canonical_head.l2_sequence_number,
                    total_games = state.games.len(),
                    "Canonical head updated"
                );
            }
        } else {
            // Clear stale canonical head index when no valid games exist.
            // canonical_head_sequence_number is intentionally preserved — it serves as the anchor
            // baseline for should_create_game() to propose the first game. Clearing it
            // would permanently block proposals on fresh deployments or when the pinned
            // snapshot has no games. The new canonical_head_index gauge (-1) provides
            // observability for the "no head" state.
            state.canonical_head_index = None;

            if previous_canonical_index.is_some() {
                tracing::info!(
                    previous_canonical_index = ?previous_canonical_index,
                    total_games = state.games.len(),
                    "Canonical head cleared: no valid games in cache"
                );
            }
        }
    }

    /// Returns true if game creation may proceed for the currently registered
    /// game implementation's prestate (see [`Self::prestate_usable_for_creation`]).
    async fn registered_prestate_known(&self) -> Result<bool> {
        let args = self.registered_game_args(BlockId::latest()).await?;
        Ok(self.prestate_usable_for_creation(args.absolute_prestate).await)
    }

    /// Returns whether game creation may proceed for `prestate`: the program
    /// artifacts must load ([`PrestateCache::ensure_loaded`]) and, in network
    /// mode, the SP1 proving keys must be set up and vkey-verified BEFORE any
    /// game is bonded on the prestate. Loading alone would admit artifacts
    /// whose keys were never initialized, deferring verification to the first
    /// defense - after a bond is already at stake.
    ///
    /// Key setup takes tens of seconds per ELF and this gate runs inline on
    /// the scheduler path each tick, so setup is NEVER awaited here: the
    /// first call kicks it off on a background task and creation stays
    /// paused until the verdict lands. A failed setup poisons the entry,
    /// which keeps this gate closed (via [`PrestateCache::ensure_loaded`])
    /// until corrected artifacts are published.
    async fn prestate_usable_for_creation(&self, prestate: B256) -> bool {
        if !self.prestates.ensure_loaded(prestate).await {
            return false;
        }
        if self.config.proof_provider != ProofProviderKind::Network {
            return true;
        }
        match self.prestates.key_verification_state(prestate).await {
            Some(true) => true,
            // Poisoned: ensure_loaded owns healing and log pacing; it only
            // reports usable again once changed artifacts replace the entry.
            Some(false) => false,
            None => {
                if self.prestates.try_kick_key_setup(prestate).await {
                    let prestates = self.prestates.clone();
                    let kind = self.config.proof_provider;
                    tokio::spawn(async move {
                        if let Err(err) = prestates.proof_keys(prestate, kind).await {
                            tracing::warn!(
                                prestate = %prestate,
                                error = %err,
                                "registered prestate proving keys unusable; creation stays paused"
                            );
                        }
                    });
                    tracing::info!(
                        prestate = %prestate,
                        "Started SP1 proving-key setup; creation waits for the verdict"
                    );
                }
                false
            }
        }
    }

    /// Returns the loaded program ELFs for `prestate`, if present in the
    /// cache. The defend path proves with these.
    pub async fn prestate_programs(&self, prestate: B256) -> Option<Arc<PrestatePrograms>> {
        self.prestates.programs(prestate).await
    }

    /// Operator fee caps applied to every submitted transaction.
    const fn fee_caps(&self) -> FeeCaps {
        FeeCaps {
            max_fee_per_gas: self.config.max_fee_per_gas,
            max_priority_fee_per_gas: self.config.max_priority_fee_per_gas,
        }
    }

    /// Creates a new game with the given parameters.
    ///
    /// `root_claim`: the super-root hash we are proposing.
    /// `extra_data`: `parentIndex (4B BE) || superRootProof`.
    pub async fn create_game(&self, root_claim: B256, extra_data: Vec<u8>) -> Result<Address> {
        // Read at creation time rather than startup: the factory's init bond
        // can change, and a stale value would revert every create.
        let init_bond = self.factory.fetch_init_bond(ZK_GAME_TYPE).await?;
        let transaction_request = self
            .factory
            .create(ZK_GAME_TYPE, root_claim, extra_data.into())
            .value(init_bond)
            .into_transaction_request();

        let receipt = self
            .signer
            .send_transaction_request_with_timeout(
                self.config.l1_rpc.clone(),
                transaction_request,
                self.config.tx_confirmation_timeout,
                self.fee_caps(),
            )
            .await?;

        if !receipt.status() {
            bail!("{TX_REVERTED_PREFIX} {receipt:?}");
        }

        let game_address = receipt
            .inner
            .logs()
            .iter()
            .find_map(|log| {
                DisputeGameCreated::decode_log(&log.inner).ok().map(|event| event.disputeProxy)
            })
            .context("Could not find DisputeGameCreated event in transaction receipt logs")?;

        tracing::info!(
            game_address = ?game_address,
            tx_hash = ?receipt.transaction_hash,
            "Game created successfully"
        );

        Ok(game_address)
    }

    async fn resolve_games(&self) -> Result<()> {
        // Ownership is prestate-based: the resolve set equals the
        // willing-to-prove set.
        let known_prestates = self.prestates.known_prestates().await;
        let candidates = {
            let state = self.state.read().await;
            state
                .games
                .values()
                .filter(|game| game.is_owned(&known_prestates))
                .filter(|game| game.should_attempt_to_resolve)
                .cloned()
                .collect::<Vec<_>>()
        };

        for game in candidates {
            // Pre-flight on-chain status check at `latest`. The cached `should_attempt_to_resolve`
            // is derived from the pinned (lagged) snapshot, so a recently confirmed `resolve()` tx
            // may not yet be reflected. Querying at `latest` avoids re-submitting a resolution
            // that would only revert on chain.
            let contract = ZKDisputeGame::new(game.address, self.l1_provider.clone());
            match contract.status().call().await {
                Ok(status) => match GameStatus::try_from(status) {
                    Ok(status) if status != GameStatus::InProgress => {
                        tracing::info!(
                            game_index = %game.index,
                            game_address = ?game.address,
                            ?status,
                            "Skipping resolve: game already resolved on chain"
                        );
                        continue;
                    }
                    Err(e) => {
                        tracing::warn!(
                            game_address = ?game.address,
                            error = ?e,
                            "Invalid game status on chain, proceeding with resolve"
                        );
                    }
                    _ => {}
                },
                Err(e) => {
                    tracing::warn!(
                        game_address = ?game.address,
                        error = ?e,
                        "Pre-flight status check failed, proceeding with resolve"
                    );
                }
            }

            if let Err(error) = self.submit_resolution_transaction(&game).await {
                if error.is_revert() {
                    tracing::error!(
                        game_index = %game.index,
                        game_address = ?game.address,
                        l2_sequence_end = %game.l2_sequence_number,
                        ?error,
                        "Resolution tx included but reverted on-chain"
                    );
                } else {
                    tracing::warn!(
                        game_index = %game.index,
                        game_address = ?game.address,
                        l2_sequence_end = %game.l2_sequence_number,
                        ?error,
                        "Resolution tx unconfirmed (may be on-chain), will verify next cycle"
                    );
                }
                ProposerGauge::GameResolutionError.increment(1.0);
                continue;
            }

            ProposerGauge::GamesResolved.increment(1.0);
        }

        Ok(())
    }

    /// Attempt to claim proposer bonds for any games flagged for claiming
    async fn claim_bonds(&self) -> Result<()> {
        // Same ownership set as proving and resolution. Claims are
        // credit-driven: iterating a foreign game where the proposer holds
        // no credit is a no-op (the pre-flight reads classify it as done).
        let known_prestates = self.prestates.known_prestates().await;
        let candidates = {
            let state = self.state.read().await;
            state
                .games
                .values()
                .filter(|game| game.is_owned(&known_prestates))
                .filter(|game| game.should_attempt_to_claim_bond)
                .cloned()
                .collect::<Vec<_>>()
        };

        let signer_address = self.signer.address();
        for game in candidates {
            // Pre-flight on-chain claim-state check at `latest`. The cached
            // `should_attempt_to_claim_bond` is derived from the pinned (lagged)
            // snapshot; re-check so we neither re-submit a finished claim nor
            // submit a phase-2 payout before the WETH delay matures.
            let contract = ZKDisputeGame::new(game.address, self.l1_provider.clone());
            let credit = contract.credit(signer_address).call().await;
            // Bind the game's own DelayedWETH: its bond lives in the WETH it
            // was created with, which can differ from the currently
            // registered one across upgrades.
            let weth = DelayedWETH::new(game.weth, self.l1_provider.clone());
            let withdrawal = weth.withdrawals(game.address, signer_address).call().await;
            let mut is_payout = false;
            if let (Ok(credit), Ok(withdrawal)) = (&credit, &withdrawal) {
                // Phase 2 = credit already unlocked (zero) with a recorded
                // withdrawal; only a successful payout claims the bond - the
                // phase-1 unlock is bookkeeping and must not count.
                is_payout = *credit == U256::ZERO && withdrawal.amount > U256::ZERO;
                if *credit == U256::ZERO && withdrawal.amount == U256::ZERO {
                    tracing::info!(
                        game_index = %game.index,
                        game_address = ?game.address,
                        "Skipping claim: bond already claimed on chain"
                    );
                    continue;
                }
                if *credit == U256::ZERO && withdrawal.amount > U256::ZERO {
                    // Phase 2: only submit once the WETH delay has elapsed in
                    // CHAIN time. DelayedWETH enforces
                    // `timestamp + DELAY_SECONDS <= block.timestamp`; wall
                    // clock and L1 time diverge under devstack time travel
                    // (and can drift in production).
                    let weth_delay: u64 = weth
                        .delay()
                        .call()
                        .await?
                        .try_into()
                        .context("DelayedWETH delay exceeds u64")?;
                    let l1_now = self
                        .l1_provider
                        .get_block_by_number(BlockNumberOrTag::Latest)
                        .await?
                        .context("failed to fetch latest L1 block for claim maturity")?
                        .header
                        .timestamp;
                    let matured = withdrawal
                        .timestamp
                        .try_into()
                        .map(|ts: u64| withdrawal_matured(ts, weth_delay, l1_now))
                        .unwrap_or(false);
                    if !matured {
                        tracing::debug!(
                            game_index = %game.index,
                            game_address = ?game.address,
                            "Skipping claim: WETH withdrawal delay not yet elapsed"
                        );
                        continue;
                    }
                }
            }

            if let Err(error) = self.submit_bond_claim_transaction(&game).await {
                if error.is_revert() {
                    tracing::error!(
                        game_index = %game.index,
                        game_address = ?game.address,
                        l2_sequence_end = %game.l2_sequence_number,
                        ?error,
                        "Bond claim tx included but reverted on-chain"
                    );
                } else {
                    tracing::warn!(
                        game_index = %game.index,
                        game_address = ?game.address,
                        l2_sequence_end = %game.l2_sequence_number,
                        ?error,
                        "Bond claim tx unconfirmed (may be on-chain), will verify next cycle"
                    );
                }
                ProposerGauge::BondClaimingError.increment(1.0);
                continue;
            }

            // If the pre-flight reads failed we cannot know the phase; skip
            // the count (under-count in a degraded state, never a double-count).
            if is_payout {
                ProposerGauge::GamesBondsClaimed.increment(1.0);
            }
        }

        Ok(())
    }

    /// Submits a `resolve()` transaction for the game and bails if it reverted.
    pub async fn submit_resolution_transaction(&self, game: &Game) -> Result<()> {
        let contract = ZKDisputeGame::new(game.address, self.l1_provider.clone());
        let transaction_request = contract.resolve().into_transaction_request();
        let receipt = self
            .signer
            .send_transaction_request_with_timeout(
                self.config.l1_rpc.clone(),
                transaction_request,
                self.config.tx_confirmation_timeout,
                self.fee_caps(),
            )
            .await?;

        if !receipt.status() {
            bail!("{TX_REVERTED_PREFIX} {receipt:?}");
        }

        tracing::info!(
            game_index = %game.index,
            game_address = ?game.address,
            l2_sequence_end = %game.l2_sequence_number,
            tx_hash = ?receipt.transaction_hash,
            "Game resolved successfully"
        );

        Ok(())
    }

    /// Submit the on-chain transaction to claim the proposer's bond for a given game.
    #[tracing::instrument(name = "[[Claiming Proposer Bonds]]", skip(self, game))]
    pub async fn submit_bond_claim_transaction(&self, game: &Game) -> Result<()> {
        let contract = ZKDisputeGame::new(game.address, self.l1_provider.clone());
        // No explicit gas limit: claimCredit's implicit closeGame (anchor
        // update + WETH ops) exceeds op-succinct's hardcoded 200k.
        let transaction_request =
            contract.claimCredit(self.signer.address()).into_transaction_request();
        let receipt = self
            .signer
            .send_transaction_request_with_timeout(
                self.config.l1_rpc.clone(),
                transaction_request,
                self.config.tx_confirmation_timeout,
                self.fee_caps(),
            )
            .await?;

        if !receipt.status() {
            bail!("{TX_REVERTED_PREFIX} {receipt:?}");
        }

        tracing::info!(
            game_index = %game.index,
            game_address = ?game.address,
            l2_sequence_end = %game.l2_sequence_number,
            tx_hash = ?receipt.transaction_hash,
            "Bond claimed successfully"
        );

        Ok(())
    }

    /// Fetch game from the factory.
    ///
    /// Terminal drops: unsupported game type, mismatched anchor state
    /// registry, disrespected game type at creation, an `l2SequenceNumber`
    /// exceeding `u64`, or another proposer's
    /// claim contradicting the canonical super root (our OWN game's claim
    /// mismatch is held pending instead of terminally dropped, since bad
    /// supernode data is the likelier cause). A timestamp not yet safe from
    /// this node's view yields `Pending` instead: excluded from the DAG but
    /// re-validated on later syncs.
    pub async fn fetch_game(&self, index: U256, pinned_block: BlockId) -> Result<GameFetchResult> {
        {
            let state = self.state.read().await;

            if state.games.contains_key(&index) {
                return Ok(GameFetchResult::AlreadyExists);
            }
        }

        let game = self.factory.gameAtIndex(index).block(pinned_block).call().await?;
        let game_address = game.proxy_;
        let game_type = game.gameType_;

        // Drop unsupported game types.
        if game_type != ZK_GAME_TYPE {
            tracing::warn!(
                game_index = %index,
                ?game_address,
                game_type,
                expected_game_type = ZK_GAME_TYPE,
                "Unsupported game type"
            );
            return Ok(GameFetchResult::UnsupportedType { game_address });
        }

        let contract = ZKDisputeGame::new(game_address, self.l1_provider.clone());
        let claim_data = contract.claimData().block(pinned_block).call().await?;
        let parent_index = claim_data.parentIndex;

        if parent_index != u32::MAX &&
            self.state.read().await.invalid_games.contains(&U256::from(parent_index))
        {
            tracing::warn!(
                game_index = %index,
                ?game_address,
                parent_index,
                "Invalid game: parent belongs to a rejected game subtree"
            );
            return Ok(GameFetchResult::InvalidGame { index });
        }

        // Capture the game's own immutable args: bond claims bind its WETH
        // and finality checks bind its registry, which can differ from the
        // currently registered ones across upgrades. Games under an older
        // registry stay in the DAG; the factory re-validates parents at
        // create time.
        let game_asr = contract.anchorStateRegistry().block(pinned_block).call().await?;
        let game_weth = contract.weth().block(pinned_block).call().await?;
        let creator = contract.gameCreator().block(pinned_block).call().await?;

        let sequence_number = contract.l2SequenceNumber().block(pinned_block).call().await?;
        // Unreachable for games created through the current contract (the
        // field is sourced from an 8-byte extraData slot), kept as a
        // classification rather than an error so a malformed game can never
        // wedge the walk: only transient errors may bubble from discovery.
        let Ok(sequence_number) = u64::try_from(sequence_number) else {
            tracing::warn!(
                game_index = %index,
                ?game_address,
                "Invalid game: l2SequenceNumber exceeds u64"
            );
            return Ok(GameFetchResult::InvalidGame { index });
        };
        let claim = contract.rootClaim().block(pinned_block).call().await?;
        let was_respected =
            contract.wasRespectedGameTypeWhenCreated().block(pinned_block).call().await?;
        let status = GameStatus::try_from(contract.status().block(pinned_block).call().await?)?;
        let absolute_prestate = contract.absolutePrestate().block(pinned_block).call().await?;

        // Enums are uint8 in the ABI; convert once at the read boundary.
        let (proposal_status, deadline) =
            (ProposalStatus::try_from(claim_data.status)?, claim_data.deadline);

        // A CHALLENGER_WINS game is terminal: children can never
        // initialize on it (InvalidParentGame), it can never anchor, and
        // the proposer holds no claimable credit in it. It never enters
        // the DAG or the pending set.
        if status == GameStatus::ChallengerWins {
            tracing::info!(
                game_index = %index,
                ?game_address,
                "Invalid game: resolved CHALLENGER_WINS (terminal)"
            );
            return Ok(GameFetchResult::InvalidGame { index });
        }

        // Drop games whose type does not respect the expected type.
        if !was_respected {
            tracing::warn!(
                game_index = %index,
                ?game_address, game_type,
                expected_game_type = ZK_GAME_TYPE,
                "Invalid game: game type was not respected when created"
            );
            return Ok(GameFetchResult::InvalidGame { index });
        }

        // Validate the claim against the canonical super root at the game's timestamp.
        //
        // A data-source failure here must not abort the factory walk: the
        // super-root query has a permanent failure class (the supernode maps
        // timestamps predating its recorded safe history to an error), so
        // propagating it would stall the cursor forever behind a single
        // un-fetchable game - a state one bonded spam game at anchor+1 could
        // force. Hold the game as pending instead: it stays outside the DAG
        // (never parent-eligible) and is re-checked each sync, bounded to one
        // query per cycle.
        let response = match self.superroot_client.superroot_at_timestamp(sequence_number).await {
            Ok(response) => response,
            Err(e) => {
                tracing::warn!(
                    game_index = %index,
                    ?game_address,
                    sequence_number,
                    error = %e,
                    "Super-root data unavailable for game; deferring validation"
                );
                ProposerGauge::SuperRootUnavailable.increment(1.0);
                return Ok(GameFetchResult::Pending {
                    index,
                    deadline,
                    prestate: absolute_prestate,
                });
            }
        };
        let super_root = match SuperrootClient::super_root_at(&response, sequence_number) {
            Ok(super_root) => super_root,
            Err(e) => {
                tracing::warn!(
                    game_index = %index,
                    ?game_address,
                    sequence_number,
                    error = %e,
                    "Super-root response failed validation for game; deferring"
                );
                ProposerGauge::SuperRootUnavailable.increment(1.0);
                return Ok(GameFetchResult::Pending {
                    index,
                    deadline,
                    prestate: absolute_prestate,
                });
            }
        };
        match super_root {
            None => {
                // Not yet safe from this node's view. Far-future timestamps beyond
                // the validation horizon are terminal (bonded spam); anything
                // nearer is pending and re-validated next sync.
                let local_safe = response.current_local_safe_timestamp;
                if local_safe > 0 &&
                    sequence_number > local_safe.saturating_add(MAX_GAME_DEADLINE_LAG)
                {
                    tracing::warn!(
                        game_index = %index,
                        ?game_address,
                        sequence_number,
                        local_safe,
                        "Invalid game: timestamp beyond validation horizon"
                    );
                    return Ok(GameFetchResult::InvalidGame { index });
                }
                tracing::info!(
                    game_index = %index,
                    ?game_address,
                    sequence_number,
                    "Game timestamp not yet safe from this node's view; deferring validation"
                );
                return Ok(GameFetchResult::Pending {
                    index,
                    deadline,
                    prestate: absolute_prestate,
                });
            }
            Some(super_root) if super_root.super_root != claim => {
                if creator == self.signer.address() {
                    // Our own game contradicting the supernode is far more
                    // likely bad supernode data than a bad claim we
                    // validated at creation. Hold it re-checkable instead of
                    // terminally dropping it (and its subtree and our init
                    // bond) on one wrong answer. The game is never
                    // parent-eligible while pending; if the claim really is
                    // bad the bond is lost on chain regardless, and the lost
                    // game is dropped at re-validation once it resolves
                    // CHALLENGER_WINS (owned pending entries are otherwise
                    // exempt from the eviction cutoff).
                    tracing::warn!(
                        game_index = %index,
                        ?game_address,
                        ?claim,
                        canonical_super_root = ?super_root.super_root,
                        "Own game contradicts canonical super root; holding as pending"
                    );
                    return Ok(GameFetchResult::Pending {
                        index,
                        deadline,
                        prestate: absolute_prestate,
                    });
                }
                tracing::warn!(
                    game_index = %index,
                    ?game_address,
                    ?claim,
                    canonical_super_root = ?super_root.super_root,
                    "Invalid game: root claim does not match canonical super root"
                );
                return Ok(GameFetchResult::InvalidGame { index });
            }
            Some(_) => {}
        }

        tracing::info!(
            game_index = %index,
            ?game_type,
            ?game_address,
            parent_index = %parent_index,
            sequence_number,
            ?status,
            ?proposal_status,
            deadline = %deadline,
            "Valid game: adding to cache"
        );

        let game = Game {
            index,
            address: game_address,
            parent_index,
            l2_sequence_number: sequence_number,
            status,
            proposal_status,
            deadline,
            should_attempt_to_resolve: false,
            should_attempt_to_claim_bond: false,
            absolute_prestate,
            creator,
            weth: game_weth,
            anchor_state_registry: game_asr,
        };

        if game.creator != self.signer.address() {
            tracing::info!(
                game_index = %index,
                creator = %game.creator,
                "Discovered game created by another proposer; defense, resolution, and claims follow the prestate-based ownership set"
            );
        }

        let mut state = self.state.write().await;
        state.games.insert(index, game);

        Ok(GameFetchResult::ValidGame { game_address, deadline })
    }

    /// Handles the creation of a new game if conditions are met.
    #[tracing::instrument(name = "[[Proposing]]", skip(self))]
    pub async fn handle_game_creation(
        &self,
        mut sequence_number: u64,
        parent_game_index: u32,
    ) -> Result<()> {
        let max_proposable = self.max_proposable_timestamp().await?;

        loop {
            let response = self.superroot_client.superroot_at_timestamp(sequence_number).await?;
            let Some(super_root) = SuperrootClient::super_root_at(&response, sequence_number)?
            else {
                // Transient: the chosen timestamp is not yet safe from this
                // node's view. Bail and retry on a later tick.
                bail!("no canonical super root at timestamp {sequence_number} yet");
            };
            let extra_data = zk_extra_data(parent_game_index, &super_root.proof_bytes);
            let existing_game = self
                .factory
                .games(ZK_GAME_TYPE, super_root.super_root, extra_data.clone().into())
                .call()
                .await?
                .proxy_;

            if existing_game == Address::ZERO {
                tracing::info!(
                    sequence_number,
                    parent_game_index,
                    root_claim = %super_root.super_root,
                    "Creating game"
                );
                // Record the exact bytes before sending: if confirmation
                // times out the tx may still land, and the record holds
                // proposals until it is adopted or provably dead instead
                // of proposing a sibling at a fresh timestamp.
                *self.in_flight_creation.lock().await = Some(InFlightCreation {
                    root_claim: super_root.super_root,
                    extra_data: extra_data.clone(),
                    sequence_number,
                    parent_game_index,
                });
                match self.create_game(super_root.super_root, extra_data).await {
                    Ok(game_address) => {
                        *self.in_flight_creation.lock().await = None;

                        // Record the sequence number and address so should_create_game() skips
                        // duplicate creation while the pinned cache hasn't caught up to this
                        // game.
                        self.last_created_game_l2_sequence_number
                            .store(sequence_number, Ordering::Relaxed);
                        *self.last_created_game_address.lock().await = game_address;
                        ProposerGauge::GamesCreated.increment(1.0);
                        return Ok(());
                    }
                    Err(e) => {
                        if e.is_revert() {
                            // A landed revert consumed the transaction: nothing is in
                            // flight, so there is no uuid to pin.
                            *self.in_flight_creation.lock().await = None;
                        }
                        return Err(e);
                    }
                }
            }

            // A game with identical parameters already exists (UUID
            // collision). If WE created it - a previous create tx landed
            // after its confirmation timed out, leaving the guard unarmed -
            // adopt it instead of advancing the timestamp, which would bond
            // a second game on the same parent. Only a third-party
            // collision advances.
            let existing_creator = ZKDisputeGame::new(existing_game, self.l1_provider.clone())
                .gameCreator()
                .call()
                .await?;
            if existing_creator == self.signer.address() {
                tracing::info!(
                    sequence_number,
                    parent_game_index,
                    game_address = ?existing_game,
                    "Adopting own existing game after create-tx uncertainty"
                );
                self.last_created_game_l2_sequence_number.store(sequence_number, Ordering::Relaxed);
                *self.last_created_game_address.lock().await = existing_game;
                return Ok(());
            }
            // Third-party collision: advance the timestamp - bounded by the
            // safety limit - and refetch a fresh super root (the proof
            // embeds the timestamp). On reaching the bound, defer: the next
            // sync adopts the existing game as canonical head.
            match advance_collision_timestamp(sequence_number, max_proposable) {
                Some(next) => {
                    tracing::debug!(
                        collided_at = sequence_number,
                        next,
                        "Game UUID collision, advancing timestamp"
                    );
                    sequence_number = next;
                }
                None => {
                    tracing::info!(
                        sequence_number,
                        max_proposable,
                        "Game UUID collision at the safety bound; deferring to next sync"
                    );
                    return Ok(());
                }
            }
        }
    }

    /// Resolves an in-flight creation left by a confirmation timeout.
    ///
    /// Looks the recorded uuid up on the factory first: if the game exists
    /// and is ours, adopt it (arm the duplicate-creation guard); if it is
    /// another proposer's, the uuid is spent and no duplicate is possible.
    /// Otherwise consult the signer's pool: while the pending transaction
    /// count exceeds the latest, something of ours is still floating and
    /// may land, so the record is kept and proposals stay held. Once the
    /// pool drains, nothing of ours can land anymore - the original was
    /// mined (the lookup above adopts it) or dropped - so the record
    /// clears and normal proposals resume next tick through
    /// `should_create_game`'s checks.
    ///
    /// The pool view is this node's: a transaction evicted here but alive
    /// in another pool can land after the clear. That residual re-creates
    /// the sibling scenario this record exists to narrow, which is benign
    /// and self-healing - the sibling is a valid own game whose bond is
    /// recovered by the normal resolution and claiming flow.
    async fn resolve_in_flight_creation(&self) -> Result<()> {
        let Some(record) = self.in_flight_creation.lock().await.clone() else {
            return Ok(());
        };

        if self.try_adopt_recorded_uuid(&record).await? {
            return Ok(());
        }

        let signer_address = self.signer.address();
        let pending_nonce =
            self.l1_provider.get_transaction_count(signer_address).pending().await?;
        let latest_nonce = self.l1_provider.get_transaction_count(signer_address).latest().await?;
        if pending_nonce > latest_nonce {
            tracing::info!(
                sequence_number = record.sequence_number,
                parent_game_index = record.parent_game_index,
                pending_nonce,
                latest_nonce,
                "In-flight transaction still in the pool; holding proposals"
            );
            return Ok(());
        }

        tracing::info!(
            sequence_number = record.sequence_number,
            parent_game_index = record.parent_game_index,
            "In-flight creation left the pool without landing; clearing the record"
        );
        *self.in_flight_creation.lock().await = None;
        Ok(())
    }

    /// Resolves the record when its uuid already exists on the factory:
    /// our own game is adopted (duplicate-creation guard armed), a foreign
    /// copy spends the uuid either way. Returns whether the record was
    /// resolved. Read errors keep the record and bubble (retried next
    /// tick).
    async fn try_adopt_recorded_uuid(&self, record: &InFlightCreation) -> Result<bool> {
        let existing_game = self
            .factory
            .games(ZK_GAME_TYPE, record.root_claim, record.extra_data.clone().into())
            .call()
            .await?
            .proxy_;
        if existing_game == Address::ZERO {
            return Ok(false);
        }

        let existing_creator = ZKDisputeGame::new(existing_game, self.l1_provider.clone())
            .gameCreator()
            .call()
            .await?;
        if existing_creator == self.signer.address() {
            // Our stuck create landed after all: adopt it. Not counted
            // in GamesCreated, consistent with the collision-adoption
            // path (accepted under-count).
            tracing::info!(
                sequence_number = record.sequence_number,
                parent_game_index = record.parent_game_index,
                game_address = ?existing_game,
                "Adopting in-flight game that landed after its confirmation timeout"
            );
            self.last_created_game_l2_sequence_number
                .store(record.sequence_number, Ordering::Relaxed);
            *self.last_created_game_address.lock().await = existing_game;
        } else {
            tracing::info!(
                sequence_number = record.sequence_number,
                game_address = ?existing_game,
                "In-flight uuid was created by another proposer; no duplicate possible"
            );
        }
        *self.in_flight_creation.lock().await = None;
        Ok(true)
    }

    /// Fetch the proposer metrics.
    async fn fetch_proposer_metrics(&self) -> Result<()> {
        let (canonical_head_sequence_number, canonical_head_index, anchor_game) = {
            let state = self.state.read().await;
            (
                state.canonical_head_sequence_number,
                state.canonical_head_index,
                state.anchor_game.clone(),
            )
        };

        // Index-based metrics use -1 as sentinel for "cleared/absent" since index 0 is valid.
        ProposerGauge::CanonicalHeadGameIndex
            .set(canonical_head_index.map_or(-1.0, |idx| idx.to::<u64>() as f64));
        ProposerGauge::AnchorGameIndex
            .set(anchor_game.as_ref().map_or(-1.0, |g| g.index.to::<u64>() as f64));

        if let Some(head) = canonical_head_sequence_number {
            ProposerGauge::LatestGameL2SequenceNumber.set(head as f64);
        }
        if let Some(anchor_game) = anchor_game {
            ProposerGauge::AnchorGameL2SequenceNumber.set(anchor_game.l2_sequence_number as f64);
        }

        // Highest proposable super-root timestamp under the configured safety level.
        let max_proposable = self.max_proposable_timestamp().await?;
        ProposerGauge::MaxProposableSequenceNumber.set(max_proposable as f64);

        Ok(())
    }

    /// Spawn a dedicated metrics collection task
    fn spawn_metrics_collector(&self) {
        let proposer_metrics = self.clone();
        tokio::spawn(async move {
            let mut metrics_timer = time::interval(Duration::from_secs(15));
            loop {
                metrics_timer.tick().await;
                if let Err(e) = proposer_metrics.fetch_proposer_metrics().await {
                    tracing::warn!("Failed to fetch metrics: {:?}", e);
                    ProposerGauge::MetricsError.increment(1.0);
                }
            }
        });
    }

    /// Handle completed tasks and clean them up
    async fn handle_completed_tasks(&self) -> Result<()> {
        let mut tasks = self.tasks.lock().await;
        let mut completed = Vec::new();

        // Find completed tasks
        for (id, (handle, _)) in tasks.iter() {
            if handle.is_finished() {
                completed.push(*id);
            }
        }

        // Process completed tasks
        for id in completed {
            if let Some((handle, info)) = tasks.remove(&id) {
                match handle.await {
                    Ok(Ok(())) => {
                        tracing::info!("Task {:?} completed successfully", info);
                    }
                    Ok(Err(e)) => {
                        tracing::warn!("Task {:?} failed: {:?}", info, e);
                        // Handle task failure based on type
                        self.handle_task_failure(&info, e).await?;
                    }
                    Err(panic) => {
                        tracing::error!("Task {:?} panicked: {:?}", info, panic);
                    }
                }
            }
        }

        Ok(())
    }

    /// Handle task failure based on task type
    async fn handle_task_failure(&self, info: &TaskInfo, _error: anyhow::Error) -> Result<()> {
        match info {
            TaskInfo::GameCreation { .. } => {
                ProposerGauge::GameCreationError.increment(1.0);
            }
            TaskInfo::GameResolution => {
                ProposerGauge::GameResolutionError.increment(1.0);
            }
            TaskInfo::GameProving { .. } => {
                ProposerGauge::GameProvingError.increment(1.0);
            }
            TaskInfo::BondClaim => {
                ProposerGauge::BondClaimingError.increment(1.0);
            }
        }
        Ok(())
    }

    /// Spawn pending operations if not already running
    async fn spawn_pending_operations(&self) -> Result<()> {
        // Check if we should create a game and spawn task if needed
        if self.has_active_task_of_type(&TaskInfo::GameCreation { sequence_number: 0 }).await {
            tracing::info!("Game creation task already active");
        } else {
            match self.spawn_game_creation_task().await {
                Ok(true) => tracing::info!("Successfully spawned game creation task"),
                Ok(false) => {
                    tracing::debug!("No game creation needed - proposal interval not elapsed")
                }
                Err(e) => tracing::warn!("Failed to spawn game creation task: {:?}", e),
            }
        }

        // Spawn defense tasks for challenged games in the owned set
        // (per-game, capped by MAX_CONCURRENT_DEFENSE_TASKS).
        match self.spawn_game_defense_tasks().await {
            Ok(true) => tracing::info!("Successfully spawned game defense tasks"),
            Ok(false) => tracing::debug!("No games need defense or defense is at capacity"),
            Err(e) => tracing::warn!("Failed to spawn game defense tasks: {:?}", e),
        }

        // Spawn game resolution task (only operates on owned games via is_owned() filter)
        if !self.has_active_task_of_type(&TaskInfo::GameResolution).await {
            if let Err(e) = self.spawn_game_resolution_task().await {
                tracing::warn!("Failed to spawn game resolution task: {:?}", e);
            } else {
                tracing::info!("Successfully spawned game resolution task");
            }
        }

        // Spawn bond claim task (only operates on owned games via is_owned() filter)
        if self.has_active_task_of_type(&TaskInfo::BondClaim).await {
            tracing::info!("Bond claim task already active");
        } else {
            if let Err(e) = self.spawn_bond_claim_task().await {
                tracing::warn!("Failed to spawn bond claim task: {:?}", e);
            } else {
                tracing::info!("Successfully spawned bond claim task");
            }
        }

        Ok(())
    }

    /// Check if there's an active task of the given type
    async fn has_active_task_of_type(&self, task_type: &TaskInfo) -> bool {
        let tasks = self.tasks.lock().await;
        tasks
            .values()
            .any(|(_, info)| std::mem::discriminant(info) == std::mem::discriminant(task_type))
    }

    /// Log current task statistics
    async fn log_task_stats(&self) {
        let tasks = self.tasks.lock().await;
        let active_count = tasks.len();
        if active_count > 0 {
            let mut task_counts: HashMap<&str, usize> = HashMap::new();

            for (_, info) in tasks.values() {
                let task_type = match info {
                    TaskInfo::GameCreation { .. } => "GameCreation",
                    TaskInfo::GameResolution => "GameResolution",
                    TaskInfo::GameProving { .. } => "GameProving",
                    TaskInfo::BondClaim => "BondClaim",
                };
                *task_counts.entry(task_type).or_insert(0) += 1;
            }

            let task_types: Vec<String> = task_counts
                .into_iter()
                .map(|(type_name, count)| format!("{type_name}: {count}"))
                .collect();

            tracing::info!("Active tasks: {} ({})", active_count, task_types.join(", "));
        }
    }

    /// Spawn a game creation task if conditions are met
    ///
    /// Returns:
    /// - Ok(true): Task was successfully spawned
    /// - Ok(false): No work needed (proposal interval not elapsed or no finalized blocks)
    /// - Err: Actual error occurred during task spawning
    async fn spawn_game_creation_task(&self) -> Result<bool> {
        // An unresolved create takes precedence over new proposals: hold
        // them until its uuid is adopted or provably dead, so a
        // stuck-then-included original can never be joined by a sibling at
        // a fresh timestamp.
        let in_flight_sequence_number =
            self.in_flight_creation.lock().await.as_ref().map(|record| record.sequence_number);
        if let Some(sequence_number) = in_flight_sequence_number {
            let proposer = self.clone();
            let task_id = self.next_task_id.fetch_add(1, Ordering::Relaxed);

            let handle = tokio::spawn(async move {
                if let Err(e) = proposer.resolve_in_flight_creation().await {
                    tracing::warn!("Failed to resolve in-flight game creation: {:?}", e);
                    return Err(e);
                }

                Ok(())
            });

            let task_info = TaskInfo::GameCreation { sequence_number };

            self.tasks.lock().await.insert(task_id, (handle, task_info));
            tracing::info!(
                "Spawned in-flight creation resolution task {} for sequence number {}",
                task_id,
                sequence_number
            );
            return Ok(true);
        }

        // First check if we should create a game
        let (should_create, next_sequence_number, parent_game_index) =
            self.should_create_game().await?;
        if !should_create {
            return Ok(false);
        }

        // A retired or blacklisted parent reverts child creation forever
        // (ZKDisputeGame.initialize consults the registered registry's
        // isGameBlacklisted/isGameRetired at create time; nothing at resolve
        // time ever prunes such a parent). Check the chosen parent against
        // the registry a new child would bind - the currently registered
        // one, not the parent's own (they differ across ASR swaps) - and
        // drop its subtree so head selection falls back next sync. Bare
        // isGameProper would be wrong here: its paused() clause is true for
        // every game during a superchain pause and would mass-evict the
        // cache.
        if parent_game_index != u32::MAX {
            let parent_address = {
                let state = self.state.read().await;
                state.games.get(&U256::from(parent_game_index)).map(|game| game.address)
            };
            let Some(parent_address) = parent_address else {
                // A non-MAX parent index that is absent from the cache is a
                // dangling head reference (e.g. right after this block removed
                // its subtree, before the next full sync re-picks the head).
                // Creating against it would be exactly the doomed creation
                // this check exists to prevent, so defer instead.
                tracing::debug!(
                    parent_index = parent_game_index,
                    "Chosen parent not in cache (dangling head after subtree removal); deferring creation until the next sync"
                );
                return Ok(false);
            };
            let registry = self.registered_anchor_state_registry(BlockId::latest()).await?;
            // Standing must be current, not pinned: retirement/blacklisting
            // is retroactive and the child binds the registry at create time.
            let blacklisted = registry.isGameBlacklisted(parent_address).call().await?;
            let retired = registry.isGameRetired(parent_address).call().await?;
            if blacklisted || retired {
                tracing::warn!(
                    parent_index = parent_game_index,
                    ?parent_address,
                    blacklisted,
                    retired,
                    "Chosen parent can no longer be built on; dropping its subtree"
                );
                let root_index = U256::from(parent_game_index);
                let mut state = self.state.write().await;
                // Mirror sync_games' RemoveSubtree handling: reset the
                // duplicate-creation guard if the removed subtree contains
                // the game it tracks (descendants_of includes the root).
                let guarded_addr = *self.last_created_game_address.lock().await;
                if guarded_addr != Address::ZERO {
                    let guard_in_subtree = state
                        .descendants_of(root_index)
                        .iter()
                        .any(|idx| state.games.get(idx).is_some_and(|g| g.address == guarded_addr));
                    if guard_in_subtree {
                        self.last_created_game_l2_sequence_number.store(0, Ordering::Relaxed);
                        *self.last_created_game_address.lock().await = Address::ZERO;
                        tracing::info!(
                            ?guarded_addr,
                            root_index = parent_game_index,
                            "Reset creation guard: tracked game removed with a retired/blacklisted ancestor"
                        );
                    }
                }
                state.invalidate_subtree(root_index);
                return Ok(false);
            }
        }

        let proposer = self.clone();
        let task_id = self.next_task_id.fetch_add(1, Ordering::Relaxed);

        let handle = tokio::spawn(async move {
            if let Err(e) =
                proposer.handle_game_creation(next_sequence_number, parent_game_index).await
            {
                tracing::warn!("Failed to handle game creation: {:?}", e);
                return Err(e);
            }

            Ok(())
        });

        let task_info = TaskInfo::GameCreation { sequence_number: next_sequence_number };

        self.tasks.lock().await.insert(task_id, (handle, task_info));
        tracing::info!(
            "Spawned game creation task {} for sequence number {}",
            task_id,
            next_sequence_number
        );
        Ok(true)
    }

    /// Check if we should create a game.
    ///
    /// Compares the next proposal sequence number against the highest
    /// timestamp proposable under the configured safety level.
    ///
    /// Returns whether a game should be created, the sequence number to
    /// propose at, and the parent game index (dummy values when false).
    pub async fn should_create_game(&self) -> Result<(bool, u64, u32)> {
        // Check if our game type matches the current respected game type.
        // The proposer should only create games when its type is the respected type.
        let respected_game_type = self
            .registered_anchor_state_registry(BlockId::latest())
            .await?
            .respectedGameType()
            .call()
            .await?;
        if ZK_GAME_TYPE != respected_game_type {
            tracing::warn!(
                proposer_game_type = ZK_GAME_TYPE,
                ?respected_game_type,
                "Skipping game creation, game type does not match respected type"
            );
            return Ok((false, 0, u32::MAX));
        }

        // Skip creation if the registered prestate is not in the known set.
        if !self.registered_prestate_known().await? {
            return Ok((false, 0, u32::MAX));
        }

        let (canonical_head_sequence_number, parent_game_index) = {
            let state = self.state.read().await;

            let Some(canonical_head_sequence_number) = state.canonical_head_sequence_number else {
                tracing::info!("No canonical head; skipping game creation");
                return Ok((false, 0, u32::MAX));
            };

            // When the canonical head IS the anchor game, use u32::MAX (anchor path) instead of
            // referencing it by index. The contract requires parent.l2SeqNum > anchor.l2SeqNum,
            // so the anchor itself cannot be used as a parent via index.
            let anchor_index = state.anchor_game.as_ref().map(|a| a.index);
            let parent_game_index = state
                .canonical_head_index
                .filter(|&idx| anchor_index != Some(idx))
                .map(|index| index.to::<u32>())
                .unwrap_or(u32::MAX);

            (canonical_head_sequence_number, parent_game_index)
        };

        let max_proposable = self.max_proposable_timestamp().await?;
        let Some(next_sequence_number) = next_proposal_timestamp(
            canonical_head_sequence_number,
            self.config.proposal_interval_seconds,
            max_proposable,
        ) else {
            tracing::debug!(
                head = canonical_head_sequence_number,
                max_proposable,
                "Skipping game creation: proposal interval not elapsed under safety bound"
            );
            return Ok((false, 0, u32::MAX));
        };

        // Guard against duplicate creation when the pinned cache lags behind the tip.
        // If we recently created a game at or beyond this sequence number, skip until the
        // cache catches up and advances canonical_head_sequence_number.
        let last_created = self.last_created_game_l2_sequence_number.load(Ordering::Relaxed);
        if last_created > 0 && next_sequence_number <= last_created {
            tracing::debug!(
                next_sequence_number,
                last_created,
                "Skipping game creation: recently created game not yet visible in pinned cache"
            );
            return Ok((false, 0, u32::MAX));
        }

        Ok((true, next_sequence_number, parent_game_index))
    }

    /// Returns the highest timestamp currently proposable under the
    /// configured safety level, from a fresh supernode response.
    async fn max_proposable_timestamp(&self) -> Result<u64> {
        let now = std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH)?.as_secs();
        let response = self.superroot_client.superroot_at_timestamp(now).await?;
        Ok(SuperrootClient::max_proposable_timestamp(&response, self.config.proposal_safety))
    }

    /// Spawn a game resolution task
    #[tracing::instrument(name = "[[Proposer Resolving]]", skip(self))]
    async fn spawn_game_resolution_task(&self) -> Result<()> {
        let proposer = self.clone();
        let task_id = self.next_task_id.fetch_add(1, Ordering::Relaxed);

        let handle = tokio::spawn(async move { proposer.resolve_games().await });

        let task_info = TaskInfo::GameResolution;
        self.tasks.lock().await.insert(task_id, (handle, task_info));
        tracing::info!("Spawned game resolution task {}", task_id);
        Ok(())
    }

    /// Spawn a bond claim task
    async fn spawn_bond_claim_task(&self) -> Result<()> {
        let proposer = self.clone();
        let task_id = self.next_task_id.fetch_add(1, Ordering::Relaxed);

        let handle = tokio::spawn(async move { proposer.claim_bonds().await });

        let task_info = TaskInfo::BondClaim;
        self.tasks.lock().await.insert(task_id, (handle, task_info));
        tracing::info!("Spawned bond claim task {}", task_id);
        Ok(())
    }

    /// Count active defense proving tasks.
    async fn count_active_defense_tasks(&self) -> u64 {
        let tasks = self.tasks.lock().await;
        tasks
            .values()
            .filter(|(_, info)| matches!(info, TaskInfo::GameProving { is_defense: true, .. }))
            .count() as u64
    }

    /// Check if there's an active proving task for a specific game.
    async fn has_active_proving_for_game(&self, game_address: Address) -> bool {
        let tasks = self.tasks.lock().await;
        tasks.values().any(|(_, info)| {
            matches!(info, TaskInfo::GameProving { game_address: addr, .. } if *addr == game_address)
        })
    }

    /// Spawns defense proving tasks for challenged games in the owned set.
    ///
    /// Candidates are sorted by deadline ascending (closest to expiring
    /// first) and capped by `MAX_CONCURRENT_DEFENSE_TASKS`; each game gets
    /// at most one live proving task. Retry is emergent: a failed task is
    /// removed by `handle_completed_tasks`, and the still-Challenged game
    /// is re-detected here next cycle.
    ///
    /// Returns `Ok(true)` if any task was spawned.
    #[tracing::instrument(name = "[[Defending]]", skip(self))]
    async fn spawn_game_defense_tasks(&self) -> Result<bool> {
        let known_prestates = self.prestates.known_prestates().await;
        let candidates = self.state.read().await.challenged_candidates();

        let mut active_defense_tasks = self.count_active_defense_tasks().await;
        let max_concurrent = self.config.max_concurrent_defense_tasks.get();
        let mut tasks_spawned = false;

        for (index, game_address, deadline, prestate) in candidates {
            if active_defense_tasks >= max_concurrent {
                tracing::debug!(
                    "The max concurrent defense tasks count ({}) has been reached",
                    max_concurrent
                );
                break;
            }

            if self.has_active_proving_for_game(game_address).await {
                continue;
            }

            if !known_prestates.contains(&prestate) {
                // A challenged game this proposer cannot prove is a state
                // operators must notice: the bond is lost when the prove
                // deadline expires. Re-fetch attempts for the prestate are
                // bounded by the cache's negative-cache window.
                tracing::warn!(
                    game_index = %index,
                    game_address = ?game_address,
                    prestate = %prestate,
                    "Challenged game has an unknown or unusable prestate; cannot defend"
                );
                ProposerGauge::UnknownPrestateChallenged.increment(1.0);
                continue;
            }

            if self.should_skip_proving(game_address, deadline).await? {
                continue;
            }

            self.spawn_game_proving_task(game_address, true).await?;
            tracing::info!(
                game_address = ?game_address,
                game_index = %index,
                "Spawned defense for challenged game"
            );
            ProposerGauge::GamesDefenseSpawned.increment(1.0);
            active_defense_tasks += 1;
            tasks_spawned = true;
        }

        Ok(tasks_spawned)
    }

    /// Check if proving should be skipped for any reason:
    /// - The game was found permanently unprovable earlier.
    /// - It is already proven or resolved on chain (pre-flight at `latest`; the cached status is
    ///   read at the pinned, lagged block, so a recently confirmed `prove()` or `resolve()` may not
    ///   be reflected yet - this avoids an expensive proof regeneration that could only revert on
    ///   submission).
    /// - Its prove deadline has passed (with a warning tier when it is approaching).
    async fn should_skip_proving(&self, game_address: Address, deadline: u64) -> Result<bool> {
        if self.undefendable.lock().await.contains(&game_address) {
            tracing::debug!(?game_address, "Skipping proving: game is permanently unprovable");
            return Ok(true);
        }

        let contract = ZKDisputeGame::new(game_address, self.l1_provider.clone());
        match contract.claimData().call().await {
            Ok(claim_data) => match ProposalStatus::try_from(claim_data.status) {
                Ok(
                    ProposalStatus::UnchallengedAndValidProofProvided |
                    ProposalStatus::ChallengedAndValidProofProvided |
                    ProposalStatus::Resolved,
                ) => {
                    tracing::info!(
                        ?game_address,
                        "Skipping proving: game already proven or resolved on chain"
                    );
                    return Ok(true);
                }
                Ok(_) => {}
                Err(e) => {
                    tracing::warn!(
                        ?game_address,
                        error = %e,
                        "Pre-flight proposal status decode failed, proceeding with proving"
                    );
                }
            },
            Err(e) => {
                tracing::warn!(
                    ?game_address,
                    error = ?e,
                    "Pre-flight proposal status check failed, proceeding with proving"
                );
            }
        }

        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .context("system time before Unix epoch")?
            .as_secs();
        let max_prove_duration = *self
            .max_prove_duration
            .get()
            .context("max_prove_duration must be set via try_init")?;
        match check_deadline_status(now, deadline, max_prove_duration) {
            DeadlineStatus::Passed => {
                tracing::error!(
                    game_address = ?game_address,
                    deadline,
                    now,
                    "Game prove deadline passed, cannot prove"
                );
                Ok(true)
            }
            DeadlineStatus::Approaching { hours_remaining } => {
                tracing::warn!(
                    game_address = ?game_address,
                    "Game prove deadline approaching, {:.1} hours remaining",
                    hours_remaining
                );
                ProposerGauge::DeadlineApproaching.increment(1.0);
                Ok(false)
            }
            DeadlineStatus::Ok => Ok(false),
        }
    }

    /// Spawns a tracked proving task for the game.
    ///
    /// The task runs the full pipeline (span fetch, witness collection,
    /// proving, `prove()` submission). A [`crate::proving::GameUnprovable`]
    /// outcome is
    /// terminal: the game joins the `undefendable` set and the task
    /// completes cleanly (no error-gauge retry churn). Any other failure
    /// bubbles to `handle_task_failure` and retries emergently.
    async fn spawn_game_proving_task(&self, game_address: Address, is_defense: bool) -> Result<()> {
        let proposer = self.clone();
        let task_id = self.next_task_id.fetch_add(1, Ordering::Relaxed);

        let handle = tokio::spawn(async move {
            match proposer.prove_game(game_address).await {
                Ok(()) => Ok(()),
                Err(err) if is_unprovable(&err) => {
                    tracing::error!(
                        ?game_address,
                        error = %err,
                        "Game is permanently unprovable; giving up on its defense"
                    );
                    ProposerGauge::GameUnprovable.increment(1.0);
                    proposer.undefendable.lock().await.insert(game_address);
                    Ok(())
                }
                Err(err) => Err(err),
            }
        });

        let task_info = TaskInfo::GameProving { game_address, is_defense };
        self.tasks.lock().await.insert(task_id, (handle, task_info));
        tracing::info!("Spawned game proving task {} for game {:?}", task_id, game_address);
        Ok(())
    }

    /// Proves one game end to end and submits `prove()`.
    #[tracing::instrument(name = "[[Proving]]", skip(self), fields(game_address = ?game_address))]
    async fn prove_game(&self, game_address: Address) -> Result<()> {
        let start_time = Instant::now();

        // The game's prestate selects the proving programs; a game that
        // left the cache mid-flight (e.g. subtree removal) is not proven.
        let prestate = {
            let state = self.state.read().await;
            state
                .games
                .values()
                .find(|game| game.address == game_address)
                .map(|game| game.absolute_prestate)
        };
        let Some(prestate) = prestate else {
            tracing::info!(?game_address, "Game no longer tracked; abandoning its defense");
            return Ok(());
        };

        // On-chain facts the proof binds.
        let contract = ZKDisputeGame::new(game_address, self.l1_provider.clone());
        let l1_head = contract.l1Head().call().await?;
        let starting = contract.startingProposal().call().await?;
        let root_claim = contract.rootClaim().call().await?;
        let claim_ts: u64 = contract
            .l2SequenceNumber()
            .call()
            .await?
            .try_into()
            .context("l2SequenceNumber exceeds u64")?;
        let starting_ts: u64 = starting
            .l2SequenceNumber
            .try_into()
            .context("starting l2SequenceNumber exceeds u64")?;
        let l1_head_number = kona_sp1_super_range_executor::fetch_l1_head_number(
            self.config.l1_rpc.as_str(),
            l1_head,
        )
        .await?;

        let keys = match self.config.proof_provider {
            ProofProviderKind::Network => {
                Some(self.prestates.proof_keys(prestate, self.config.proof_provider).await?)
            }
            ProofProviderKind::Mock => None,
        };

        let inputs = GameProofInputs {
            l1_head,
            l1_head_number,
            starting_root: starting.root,
            starting_ts,
            root_claim,
            claim_ts,
            prestate,
            prover: self.signer.address(),
        };
        let responses = fetch_span_responses(&self.superroot_client, &inputs).await?;
        let proof_bytes = prove_game_inner(
            &self.proof_provider,
            keys.as_deref(),
            &self.host_inputs,
            &inputs,
            &responses,
            self.config.range_split_count,
            self.config.max_concurrent_range_proofs,
        )
        .await?;

        // Pre-submit re-check: proving can take long; avoid a guaranteed
        // revert when the game was proven by someone else, resolved, hit
        // its deadline, or was evicted (parent lost) meanwhile.
        if !self.pre_submit_checks(game_address).await? {
            return Ok(());
        }

        let transaction_request = contract.prove(proof_bytes.into()).into_transaction_request();
        let receipt = self
            .signer
            .send_transaction_request_with_timeout(
                self.config.l1_rpc.clone(),
                transaction_request,
                self.config.tx_confirmation_timeout,
                self.fee_caps(),
            )
            .await?;

        if !receipt.status() {
            bail!("{TX_REVERTED_PREFIX} {receipt:?}");
        }

        ProposerGauge::GamesProven.increment(1.0);
        ProposerGauge::ProvingDurationSeconds.set(start_time.elapsed().as_secs_f64());
        tracing::info!(
            game_address = ?game_address,
            tx_hash = ?receipt.transaction_hash,
            duration_s = start_time.elapsed().as_secs_f64(),
            "Game proven successfully"
        );
        Ok(())
    }

    /// Final checks before submitting a `prove()` transaction. Returns false
    /// when submission should be skipped (all three legs log why):
    /// 1. the game is still tracked (subtree removal on a lost parent evicts descendants, and
    ///    `prove()` reverts `InvalidParentGame`; a residual on-chain-but-unsynced parent loss still
    ///    reverts harmlessly and is caught by the tx status check);
    /// 2. `claimData` at `latest` is still `Challenged`;
    /// 3. the prove deadline has not passed.
    async fn pre_submit_checks(&self, game_address: Address) -> Result<bool> {
        let tracked = {
            let state = self.state.read().await;
            state.games.values().any(|game| game.address == game_address)
        };
        if !tracked {
            tracing::info!(?game_address, "Skipping prove(): game evicted mid-proving");
            return Ok(false);
        }

        let contract = ZKDisputeGame::new(game_address, self.l1_provider.clone());
        let claim_data = contract.claimData().call().await?;
        let status = ProposalStatus::try_from(claim_data.status)?;
        if status != ProposalStatus::Challenged {
            tracing::info!(
                ?game_address,
                ?status,
                "Skipping prove(): game no longer awaiting a proof"
            );
            return Ok(false);
        }

        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .context("system time before Unix epoch")?
            .as_secs();
        if now >= claim_data.deadline {
            tracing::warn!(
                ?game_address,
                deadline = claim_data.deadline,
                now,
                "Skipping prove(): prove deadline passed mid-proving"
            );
            return Ok(false);
        }

        Ok(true)
    }
}

/// Warn when less than `max_duration / DEADLINE_WARNING_DIVISOR` remains
/// before a game's prove deadline.
pub const DEADLINE_WARNING_DIVISOR: u64 = 2;

/// Status of a game's prove deadline.
#[derive(Debug, Clone, PartialEq)]
pub enum DeadlineStatus {
    /// Deadline has passed.
    Passed,
    /// Deadline is approaching (within the warning window).
    Approaching {
        /// Hours remaining until the deadline.
        hours_remaining: f64,
    },
    /// Plenty of time remains.
    Ok,
}

/// Check the deadline status for a game (ported from upstream op-succinct).
pub fn check_deadline_status(now: u64, deadline: u64, max_duration: u64) -> DeadlineStatus {
    if now >= deadline {
        return DeadlineStatus::Passed;
    }

    let time_remaining = deadline.saturating_sub(now);
    if time_remaining < max_duration / DEADLINE_WARNING_DIVISOR {
        let hours_remaining = time_remaining as f64 / 3600.0;
        DeadlineStatus::Approaching { hours_remaining }
    } else {
        DeadlineStatus::Ok
    }
}

/// Result of fetching a game from the factory.
///
/// Games can either be added to the cache or dropped based on validation criteria.
#[derive(Debug)]
pub enum GameFetchResult {
    /// Game was successfully validated and added to cache
    ValidGame {
        /// Address of the validated game.
        game_address: Address,
        /// Claim deadline of the game (L1 timestamp, seconds).
        deadline: u64,
    },
    /// Game type is unsupported
    UnsupportedType {
        /// Address of the rejected game.
        game_address: Address,
    },
    /// Game is invalid
    InvalidGame {
        /// Factory index of the invalid game.
        index: U256,
    },
    /// The game's timestamp is not yet safe from this node's view: it cannot
    /// be validated yet. Kept OUT of the DAG (never parent-eligible) but
    /// re-validated each sync until data appears or the horizon expires -
    /// unlike terminal invalidity, this verdict is not permanent (the
    /// upstream design could drop terminally because its data source errored
    /// on unavailable blocks; ours reports absence as `data: None`).
    Pending {
        /// Factory index of the pending game.
        index: U256,
        /// Claim deadline of the game (L1 timestamp, seconds), used by the
        /// pending eviction cutoff.
        deadline: u64,
        /// The game's `absolutePrestate()`: owned (usable-prestate) games
        /// are exempt from pending eviction.
        prestate: B256,
    },
    /// Game was already present in the cache
    AlreadyExists,
}

/// Cursor that tracks dispute-game indices, representing the current position in the ordered
/// factory sequence.
///
/// Wraps `Option<U256>`:
/// - `Some(i)`: concrete position within the factory's ordered game sequence.
/// - `None`: sentinel meaning "no position" (before first game / past zero / uninitialized).
#[derive(Default, Debug, Clone, PartialEq, Eq, PartialOrd, Ord)]
pub struct Cursor {
    index: Option<U256>,
}

impl Cursor {
    /// Create a cursor with no index.
    pub const fn none() -> Self {
        Self { index: None }
    }

    /// Get the current index of the cursor.
    pub const fn index(&self) -> Option<U256> {
        self.index
    }

    /// Step the cursor back by one. If the cursor is at zero, it becomes `None`.
    pub fn step_back(&mut self) {
        if let Some(idx) = self.index {
            if idx > U256::ZERO {
                self.index = Some(idx.saturating_sub(U256::ONE));
            } else {
                self.index = None;
            }
        }
    }
}

impl std::fmt::Display for Cursor {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match &self.index {
            Some(idx) => write!(f, "{idx}"),
            None => write!(f, "None"),
        }
    }
}

impl From<U256> for Cursor {
    fn from(idx: U256) -> Self {
        Self { index: Some(idx) }
    }
}

/// Picks the sequence number for the next proposal, or `None` when the
/// proposal interval has not elapsed under the safety bound.
///
/// Normally `head + interval` (fixed grid); when the head lags the bound by
/// two or more intervals, jumps to `max_proposable` directly (catch-up), so a
/// restart after downtime - or a devstack time-travel warp - yields one game
/// per tick instead of a crawl across the gap.
pub fn next_proposal_timestamp(head: u64, interval: u64, max_proposable: u64) -> Option<u64> {
    let next = head.checked_add(interval)?;
    if next > max_proposable {
        return None;
    }
    if max_proposable.saturating_sub(head) >= interval.saturating_mul(2) {
        Some(max_proposable)
    } else {
        Some(next)
    }
}

/// Advances the UUID-collision walk by one second, bounded by the safety
/// limit. `None` means the walk reached the bound and creation must defer to
/// the next sync (an identical valid game already exists at every candidate
/// timestamp up to the bound).
pub fn advance_collision_timestamp(current: u64, max_proposable: u64) -> Option<u64> {
    let next = current.checked_add(1)?;
    (next <= max_proposable).then_some(next)
}

/// Action to take for an owned `DefenderWins` game in the two-phase
/// `DelayedWETH` claim flow.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum BondClaimAction {
    /// Phase (a): credit remains and the game is finalized - call
    /// `claimCredit` to zero the credit and unlock the WETH withdrawal    /// (claimCredit's
    /// implicit closeGame reverts before finality).
    Unlock,
    /// Phase (b): a withdrawal is recorded and the WETH delay has elapsed
    /// in CHAIN time - call `claimCredit` again for the payout.
    Payout,
    /// Nothing actionable yet (unfinalized credit, or immature withdrawal).
    Wait,
    /// No credit and no pending withdrawal - the game can be evicted.
    Done,
}

/// Two-phase `DelayedWETH` claim decision. `l1_now` MUST be an L1 block
/// timestamp: `DelayedWETH.withdraw` enforces
/// `timestamp + DELAY_SECONDS <= block.timestamp`, and wall-clock time
/// diverges from chain time under devstack time travel.
pub fn bond_claim_action(
    is_finalized: bool,
    credit: U256,
    withdrawal_amount: U256,
    withdrawal_ts: u64,
    weth_delay: u64,
    l1_now: u64,
) -> BondClaimAction {
    if is_finalized && credit > U256::ZERO {
        return BondClaimAction::Unlock;
    }
    if withdrawal_amount > U256::ZERO && withdrawal_matured(withdrawal_ts, weth_delay, l1_now) {
        return BondClaimAction::Payout;
    }
    if is_finalized && credit == U256::ZERO && withdrawal_amount == U256::ZERO {
        return BondClaimAction::Done;
    }
    BondClaimAction::Wait
}

/// Returns whether a recorded `DelayedWETH` withdrawal has matured at the
/// given L1 timestamp.
pub fn withdrawal_matured(withdrawal_ts: u64, weth_delay: u64, l1_now: u64) -> bool {
    withdrawal_ts != 0 &&
        withdrawal_ts.checked_add(weth_delay).is_some_and(|deadline| l1_now >= deadline)
}

/// Returns whether a game deadline is beyond the maximum allowed lag from
/// the anchor deadline (the walk stop condition and the pending-eviction
/// cutoff share this rule).
pub const fn beyond_deadline_lag(anchor_deadline: u64, game_deadline: u64) -> bool {
    anchor_deadline.abs_diff(game_deadline) > MAX_GAME_DEADLINE_LAG
}

/// Returns whether a pending game may be evicted from re-validation: its
/// deadline fell BEHIND the anchor deadline by more than the maximum lag.
/// Deliberately one-sided, unlike [`beyond_deadline_lag`]: a game that far
/// behind the anchor can never become parent-eligible, so nothing is lost by
/// dropping it, while a game AHEAD of a stalled anchor is fresh (its
/// challenge window is still open) and must stay re-checkable.
pub const fn pending_evictable(anchor_deadline: u64, game_deadline: u64) -> bool {
    game_deadline.saturating_add(MAX_GAME_DEADLINE_LAG) < anchor_deadline
}

/// Policy for game creation when the registered prestate's programs cannot
/// be loaded.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum UnknownPrestatePolicy {
    /// Skip creation until the programs become loadable. The conservative
    /// default: never bond a game we could not defend.
    Pause,
    /// Warn and create anyway (liveness-first).
    Continue,
}

/// The active policy.
pub const UNKNOWN_PRESTATE_POLICY: UnknownPrestatePolicy = UnknownPrestatePolicy::Pause;

/// How long a failed prestate artifact load is cached before the next fetch
/// attempt. Bounds `PRESTATES_URL` traffic and log noise for genuinely
/// unknown prestates while keeping them retryable: an operator can publish
/// artifacts later and the proposer self-heals without a restart.
pub const UNKNOWN_PRESTATE_RETRY: Duration = Duration::from_secs(60);

/// A cached prestate: its program ELFs plus lazily initialized proving keys.
#[derive(Debug)]
pub struct PrestateEntry {
    programs: Arc<PrestatePrograms>,
    /// Lazily initialized proving keys and the vkey-verification verdict
    /// (network mode only; mock mode never initializes this). A stored
    /// `Err` poisons the entry: setup already ran and the aggregation ELF
    /// does not hash to the prestate (or setup itself failed), so the
    /// outcome is deterministic and never retried.
    keys: OnceCell<Result<Arc<ProofKeys>, PrestateKeyError>>,
    /// One-shot latch claiming the right to start background key setup for
    /// this entry (see `try_kick_key_setup`).
    setup_kicked: AtomicBool,
}

/// Why a prestate's proving keys are unusable. Cloneable so the poisoned
/// verdict can be stored once and returned to every later caller.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum PrestateKeyError {
    /// The aggregation ELF's verifying key does not hash to the on-chain
    /// prestate: the published artifacts cannot prove games with this
    /// prestate.
    VkeyMismatch {
        /// The on-chain prestate the artifacts are keyed by.
        expected: B256,
        /// The verifying-key hash the aggregation ELF actually has.
        actual: B256,
    },
    /// SP1 proving-key setup failed.
    Setup(String),
}

impl std::fmt::Display for PrestateKeyError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::VkeyMismatch { expected, actual } => write!(
                f,
                "aggregation ELF verifying key {actual} does not hash to prestate {expected}"
            ),
            Self::Setup(err) => write!(f, "SP1 proving-key setup failed: {err}"),
        }
    }
}

impl std::error::Error for PrestateKeyError {}

/// Prestate program cache: loaded ELFs keyed by `absolutePrestate()` hash,
/// fetched from the base `PRESTATES_URL` on demand, plus per-prestate SP1
/// proving keys for the defend path.
///
/// The artifact directory is consulted live on every cache miss (subject to
/// the negative-cache retry window), so a hardfork that rotates the
/// registered prestate only requires publishing the new program artifacts,
/// not restarting the proposer.
#[derive(Debug)]
pub struct PrestateCache {
    programs: RwLock<HashMap<B256, Arc<PrestateEntry>>>,
    /// Negative cache: prestates whose last artifact load failed, with the
    /// time of that attempt. No re-fetch (or re-log) happens until
    /// `unknown_retry` elapses.
    misses: RwLock<HashMap<B256, Instant>>,
    url: Url,
    unknown_retry: Duration,
}

impl PrestateCache {
    /// Creates an empty cache backed by `url` with the default
    /// [`UNKNOWN_PRESTATE_RETRY`] window.
    pub fn new(url: Url) -> Self {
        Self::with_retry_window(url, UNKNOWN_PRESTATE_RETRY)
    }

    /// Creates an empty cache with an explicit negative-cache retry window
    /// (tests use a zero window to pin the immediate self-heal property).
    pub fn with_retry_window(url: Url, unknown_retry: Duration) -> Self {
        Self {
            programs: RwLock::new(HashMap::new()),
            misses: RwLock::new(HashMap::new()),
            url,
            unknown_retry,
        }
    }

    /// Ensures the program ELFs for `prestate` are loaded AND usable,
    /// fetching them on a miss (see [`load_prestate`]) unless a recent
    /// failed attempt is still inside the negative-cache window. Returns
    /// whether game creation may proceed for that prestate. When loading
    /// fails, the result follows [`UNKNOWN_PRESTATE_POLICY`].
    ///
    /// A POISONED entry (its proving keys failed setup or vkey
    /// verification) always blocks game creation: bonding a game the
    /// proposer has already proven it cannot defend hands the bond to the
    /// first challenger. Poisoned entries re-fetch on the negative-cache
    /// cadence and heal only when the published artifacts actually
    /// changed, so an operator can fix bad artifacts without a restart
    /// while identical bad artifacts stay poisoned (no setup churn).
    pub async fn ensure_loaded(&self, prestate: B256) -> bool {
        let poisoned_programs = match self.programs.read().await.get(&prestate) {
            Some(entry) if !matches!(entry.keys.get(), Some(Err(_))) => return true,
            Some(entry) => Some(entry.programs.clone()),
            None => None,
        };
        let unknown = match UNKNOWN_PRESTATE_POLICY {
            UnknownPrestatePolicy::Pause => false,
            UnknownPrestatePolicy::Continue => true,
        };
        if let Some(last_attempt) = self.misses.read().await.get(&prestate) &&
            last_attempt.elapsed() < self.unknown_retry
        {
            return unknown;
        }
        match load_prestate(&self.url, prestate).await {
            Ok(programs) => {
                if let Some(poisoned) = poisoned_programs {
                    if *poisoned == programs {
                        // Identical artifacts would poison again; keep the
                        // stored verdict and re-check on the next window.
                        tracing::warn!(
                            prestate = %prestate,
                            "Prestate stays poisoned: published artifacts are unchanged \
                             (publish corrected artifacts under PRESTATES_URL to heal)"
                        );
                        self.misses.write().await.insert(prestate, Instant::now());
                        return unknown;
                    }
                    tracing::info!(
                        prestate = %prestate,
                        "Published artifacts changed; replacing poisoned prestate entry"
                    );
                }
                tracing::info!(
                    prestate = %prestate,
                    aggregation_elf_bytes = programs.aggregation_elf.len(),
                    range_elf_bytes = programs.range_elf.len(),
                    "Loaded prestate programs"
                );
                self.misses.write().await.remove(&prestate);
                self.programs.write().await.insert(
                    prestate,
                    Arc::new(PrestateEntry {
                        programs: Arc::new(programs),
                        keys: OnceCell::new(),
                        setup_kicked: AtomicBool::new(false),
                    }),
                );
                true
            }
            Err(e) => {
                tracing::warn!(
                    prestate = %prestate,
                    error = %e,
                    retry_seconds = self.unknown_retry.as_secs(),
                    "Failed to load prestate programs \
                     (publish the artifacts under PRESTATES_URL if this is a hardfork)"
                );
                ProposerGauge::UnknownRegisteredPrestate.increment(1.0);
                self.misses.write().await.insert(prestate, Instant::now());
                unknown
            }
        }
    }

    /// The loaded programs for `prestate`, if present. The defend path
    /// proves with these.
    pub async fn programs(&self, prestate: B256) -> Option<Arc<PrestatePrograms>> {
        self.programs.read().await.get(&prestate).map(|entry| entry.programs.clone())
    }

    /// Test-only: inserts programs for `prestate` without fetching.
    #[cfg(test)]
    pub(crate) async fn insert_for_tests(&self, prestate: B256, programs: PrestatePrograms) {
        self.programs.write().await.insert(
            prestate,
            Arc::new(PrestateEntry {
                programs: Arc::new(programs),
                keys: OnceCell::new(),
                setup_kicked: AtomicBool::new(false),
            }),
        );
    }

    /// Snapshot of the prestates the proposer can currently prove: loaded
    /// artifacts whose proving keys are not poisoned. This set defines
    /// game ownership (prove = resolve = claim set).
    pub async fn known_prestates(&self) -> HashSet<B256> {
        self.programs
            .read()
            .await
            .iter()
            .filter(|(_, entry)| !matches!(entry.keys.get(), Some(Err(_))))
            .map(|(prestate, _)| *prestate)
            .collect()
    }

    /// Non-blocking view of the proving-key verdict for `prestate`:
    /// `Some(true)` = set up and vkey-verified, `Some(false)` = poisoned,
    /// `None` = no verdict yet (setup pending or never started) or the
    /// prestate is not loaded.
    pub async fn key_verification_state(&self, prestate: B256) -> Option<bool> {
        self.programs
            .read()
            .await
            .get(&prestate)
            .and_then(|entry| entry.keys.get())
            .map(|outcome| outcome.is_ok())
    }

    /// Claims the one-shot right to start key setup for `prestate`'s entry.
    /// Returns true exactly once per entry; healing replaces the entry and
    /// re-arms the claim for the corrected artifacts.
    pub async fn try_kick_key_setup(&self, prestate: B256) -> bool {
        self.programs
            .read()
            .await
            .get(&prestate)
            .is_some_and(|entry| !entry.setup_kicked.swap(true, Ordering::Relaxed))
    }

    /// The proving keys for `prestate`, running SP1 key setup on first use
    /// (network mode only; the mock provider never needs keys).
    ///
    /// Setup runs on the blocking pool and takes tens of seconds per ELF;
    /// the entry is cloned OUT of the map first so no map guard is held
    /// across it (a held guard would queue writers and stall every reader
    /// for the whole setup). The deferred PR-1 MUST-DO lands here: after
    /// setup, the aggregation verifying key must hash to `prestate`, else
    /// the entry is poisoned and excluded from [`Self::known_prestates`].
    pub async fn proof_keys(
        &self,
        prestate: B256,
        kind: ProofProviderKind,
    ) -> Result<Arc<ProofKeys>> {
        anyhow::ensure!(
            kind == ProofProviderKind::Network,
            "proving keys are unavailable in mock mode"
        );
        let entry = self
            .programs
            .read()
            .await
            .get(&prestate)
            .cloned()
            .ok_or_else(|| anyhow!("prestate {prestate} is not loaded"))?;
        let programs = entry.programs.clone();
        let outcome = entry
            .keys
            .get_or_init(|| async move {
                match setup_proof_keys(&programs).await {
                    Ok(keys) => {
                        let actual = B256::from(keys.agg_vk.bytes32_raw());
                        if actual == prestate {
                            Ok(Arc::new(keys))
                        } else {
                            tracing::error!(
                                prestate = %prestate,
                                vkey = %actual,
                                "Aggregation ELF does not hash to the on-chain prestate; \
                                 games with this prestate cannot be defended with the \
                                 published artifacts"
                            );
                            ProposerGauge::PrestateVkeyMismatch.increment(1.0);
                            Err(PrestateKeyError::VkeyMismatch { expected: prestate, actual })
                        }
                    }
                    Err(err) => Err(PrestateKeyError::Setup(err.to_string())),
                }
            })
            .await;
        outcome.clone().map_err(Into::into)
    }
}

#[cfg(test)]
mod tests {
    use super::{Cursor, Game, ProposerState, next_proposal_timestamp};
    use crate::contract::{GameStatus, ProposalStatus};
    use alloy_primitives::{Address, B256, U256};
    use std::collections::HashSet;

    fn game_with(index: u64, parent_index: u32, l2_sequence_number: u64) -> Game {
        Game {
            index: U256::from(index),
            address: Address::left_padding_from(&[index as u8]),
            parent_index,
            l2_sequence_number,
            status: GameStatus::InProgress,
            proposal_status: ProposalStatus::Unchallenged,
            deadline: 0,
            should_attempt_to_resolve: false,
            should_attempt_to_claim_bond: false,
            absolute_prestate: B256::ZERO,
            creator: Address::ZERO,
            weth: Address::ZERO,
            anchor_state_registry: Address::ZERO,
        }
    }

    mod ownership {
        use std::collections::HashSet;

        use super::*;

        // Ownership moved from creator-based to prestate-based with the
        // defend path: the prove, resolve, and claim sets are one set,
        // keyed by whether the game's prestate programs are loadable.

        #[test]
        fn rotation_keeps_old_prestate_games_owned() {
            // Rotation scenario: a game created before a prestate upgrade
            // stays owned as long as its (old) prestate artifacts remain
            // published, regardless of who created it.
            let old_prestate = B256::left_padding_from(&[0xde, 0xad]);
            let mut game = game_with(1, u32::MAX, 100);
            game.creator = Address::left_padding_from(&[0xbb]);
            game.absolute_prestate = old_prestate;
            let known = HashSet::from([old_prestate, B256::left_padding_from(&[0x01])]);
            assert!(game.is_owned(&known));
        }

        #[test]
        fn unknown_prestate_is_not_owned_even_for_own_creations() {
            // The creator is irrelevant: a game whose prestate artifacts
            // are not loadable cannot be proven, so it is not resolved or
            // claimed either.
            let us = Address::left_padding_from(&[0xaa]);
            let mut game = game_with(1, u32::MAX, 100);
            game.creator = us;
            game.absolute_prestate = B256::left_padding_from(&[0xde, 0xad]);
            assert!(!game.is_owned(&HashSet::new()));
        }
    }

    mod defense {

        use alloy_provider::ProviderBuilder;
        use alloy_signer_local::PrivateKeySigner;

        use super::*;
        use crate::{
            config::{
                PrestatePrograms, ProofProviderConfig, ProofProviderKind, ProposalSafety,
                ProposerConfig, RangeSplitCount,
            },
            contract::DisputeGameFactory,
            proposer::{
                DEADLINE_WARNING_DIVISOR, DeadlineStatus, Proposer, TaskInfo, check_deadline_status,
            },
            prover::{MockProofProvider, ProofProvider},
            signer::{Signer, SignerLock},
        };

        fn test_config() -> ProposerConfig {
            ProposerConfig {
                l1_rpc: "http://127.0.0.1:1".parse().unwrap(),
                supernode_rpc: "http://127.0.0.1:1".parse().unwrap(),
                factory_address: Address::ZERO,
                prestates_url: "file:///nonexistent".parse().unwrap(),
                proposal_interval_seconds: 3600,
                proposal_safety: ProposalSafety::Finalized,
                fetch_interval: 30,
                metrics_port: 0,
                sync_l1_confirmations: 0,
                tx_confirmation_timeout: 60,
                max_fee_per_gas: None,
                max_priority_fee_per_gas: None,
                proof_provider: ProofProviderKind::Mock,
                l1_beacon_rpc: "http://127.0.0.1:1".parse().unwrap(),
                l2_rpcs: vec!["http://127.0.0.1:1".parse().unwrap()],
                rollup_config_paths: None,
                l1_config_path: None,
                dependency_set_path: None,
                range_split_count: RangeSplitCount::one(),
                max_concurrent_range_proofs: std::num::NonZeroUsize::MIN,
                max_concurrent_defense_tasks: std::num::NonZeroU64::new(8).unwrap(),
                proof_provider_config: ProofProviderConfig {
                    timeout: 14_400,
                    network_calls_timeout: 15,
                    auction_timeout: 60,
                    range_proof_strategy: sp1_sdk::network::FulfillmentStrategy::Reserved,
                    agg_proof_strategy: sp1_sdk::network::FulfillmentStrategy::Reserved,
                    range_cycle_limit: 1,
                    range_gas_limit: 1,
                    agg_cycle_limit: 1,
                    agg_gas_limit: 1,
                    max_price_per_pgu: 1,
                    min_auction_period: 1,
                },
            }
        }

        /// A proposer whose RPC endpoints are unreachable: everything that
        /// needs no chain read is exercisable (task accounting, undefendable
        /// set, deadline arms after a failed pre-flight read).
        async fn test_proposer() -> Proposer<crate::L1Provider> {
            let signer = SignerLock::new(Signer::LocalSigner(PrivateKeySigner::random()));
            let provider =
                ProviderBuilder::default().connect_http("http://127.0.0.1:1".parse().unwrap());
            let factory = DisputeGameFactory::new(Address::ZERO, provider);
            Proposer::new(test_config(), signer, factory, ProofProvider::Mock(MockProofProvider))
                .await
                .unwrap()
        }

        async fn insert_task(proposer: &Proposer<crate::L1Provider>, info: TaskInfo) {
            let task_id = proposer.next_task_id.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
            let handle = tokio::spawn(async { Ok(()) });
            proposer.tasks.lock().await.insert(task_id, (handle, info));
        }

        #[test]
        fn challenged_candidates_sorted_and_filtered() {
            let mut challenged_late = game_with(1, u32::MAX, 100);
            challenged_late.proposal_status = ProposalStatus::Challenged;
            challenged_late.deadline = 500;
            let mut challenged_early = game_with(2, 1, 200);
            challenged_early.proposal_status = ProposalStatus::Challenged;
            challenged_early.deadline = 100;
            // Excluded: unchallenged, proven, and resolved games.
            let unchallenged = game_with(3, 2, 300);
            let mut proven = game_with(4, 3, 400);
            proven.proposal_status = ProposalStatus::ChallengedAndValidProofProvided;
            let mut resolved = game_with(5, 4, 500);
            resolved.proposal_status = ProposalStatus::Challenged;
            resolved.status = GameStatus::DefenderWins;

            let state = ProposerState {
                games: [challenged_late, challenged_early, unchallenged, proven, resolved]
                    .into_iter()
                    .map(|game| (game.index, game))
                    .collect(),
                ..Default::default()
            };

            let candidates = state.challenged_candidates();
            // Deadline-ascending: the game closest to expiry first.
            assert_eq!(
                candidates.iter().map(|(index, ..)| *index).collect::<Vec<_>>(),
                vec![U256::from(2), U256::from(1)]
            );
        }

        #[tokio::test]
        async fn defense_task_accounting_dedups_per_game() {
            let proposer = test_proposer().await;
            let game_a = Address::left_padding_from(&[0xa1]);
            let game_b = Address::left_padding_from(&[0xb1]);

            insert_task(
                &proposer,
                TaskInfo::GameProving { game_address: game_a, is_defense: true },
            )
            .await;
            insert_task(
                &proposer,
                TaskInfo::GameProving { game_address: game_b, is_defense: false },
            )
            .await;
            insert_task(&proposer, TaskInfo::GameResolution).await;

            assert!(proposer.has_active_proving_for_game(game_a).await);
            assert!(proposer.has_active_proving_for_game(game_b).await);
            assert!(!proposer.has_active_proving_for_game(Address::ZERO).await);
            // Only defense-triggered proving counts against the defense cap.
            assert_eq!(proposer.count_active_defense_tasks().await, 1);
        }

        #[tokio::test]
        async fn should_skip_proving_honors_undefendable_set() {
            let proposer = test_proposer().await;
            let game = Address::left_padding_from(&[0xcc]);
            proposer.undefendable.lock().await.insert(game);
            // Arm (e) fires before any chain read: no RPC needed.
            assert!(proposer.should_skip_proving(game, u64::MAX).await.unwrap());
        }

        #[tokio::test]
        async fn should_skip_proving_deadline_arms() {
            let proposer = test_proposer().await;
            proposer.max_prove_duration.set(7200).unwrap();
            let game = Address::left_padding_from(&[0xdd]);
            let now = std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs();

            // The pre-flight claimData read fails (unreachable RPC) and
            // proceeds by design; the deadline arm then decides.
            assert!(
                proposer.should_skip_proving(game, now.saturating_sub(10)).await.unwrap(),
                "passed deadline must skip"
            );
            assert!(
                !proposer.should_skip_proving(game, now + 100_000).await.unwrap(),
                "distant deadline must proceed"
            );
            assert!(
                !proposer.should_skip_proving(game, now + 600).await.unwrap(),
                "approaching deadline warns but proceeds"
            );
        }

        #[tokio::test]
        async fn pre_submit_checks_skip_evicted_games() {
            let proposer = test_proposer().await;
            // Leg 1: a game absent from state.games (evicted mid-proving,
            // e.g. subtree removal after a parent loss) is never submitted.
            let evicted = Address::left_padding_from(&[0xee]);
            assert!(!proposer.pre_submit_checks(evicted).await.unwrap());
        }

        #[test]
        fn deadline_status_tiers() {
            const MAX_DURATION: u64 = 4 * 3600;
            const HOUR: u64 = 3600;

            // (now, deadline, expected)
            let cases = [
                (1000, 900, DeadlineStatus::Passed),
                (1000, 1000, DeadlineStatus::Passed),
                (1000, 1000 + 5 * HOUR, DeadlineStatus::Ok),
                // Exactly at the threshold is still Ok (strict less-than).
                (1000, 1000 + MAX_DURATION / DEADLINE_WARNING_DIVISOR, DeadlineStatus::Ok),
            ];
            for (now, deadline, expected) in cases {
                assert_eq!(
                    check_deadline_status(now, deadline, MAX_DURATION),
                    expected,
                    "now={now} deadline={deadline}"
                );
            }

            // Inside the warning window: Approaching with the right hours.
            match check_deadline_status(1000, 1000 + HOUR, MAX_DURATION) {
                DeadlineStatus::Approaching { hours_remaining } => {
                    assert!((hours_remaining - 1.0).abs() < 0.01);
                }
                other => panic!("Expected Approaching, got {other:?}"),
            }
        }

        #[tokio::test]
        async fn defense_scan_skips_unknown_prestate_games() {
            let proposer = test_proposer().await;
            let mut challenged = game_with(1, u32::MAX, 100);
            challenged.proposal_status = ProposalStatus::Challenged;
            challenged.deadline = u64::MAX;
            proposer.state.write().await.games.insert(challenged.index, challenged);

            // The game's prestate is not loadable (empty cache): the scan
            // must not spawn a defense for it.
            assert!(!proposer.spawn_game_defense_tasks().await.unwrap());
            assert_eq!(proposer.tasks.lock().await.len(), 0);
        }

        #[tokio::test]
        async fn defense_scan_enforces_concurrency_cap() {
            let mut config = test_config();
            config.max_concurrent_defense_tasks = std::num::NonZeroU64::MIN;
            let signer = SignerLock::new(Signer::LocalSigner(PrivateKeySigner::random()));
            let provider =
                ProviderBuilder::default().connect_http("http://127.0.0.1:1".parse().unwrap());
            let factory = DisputeGameFactory::new(Address::ZERO, provider);
            let proposer =
                Proposer::new(config, signer, factory, ProofProvider::Mock(MockProofProvider))
                    .await
                    .unwrap();
            proposer.max_prove_duration.set(7200).unwrap();

            // Two challenged games with a known prestate.
            let prestate = B256::left_padding_from(&[0x77]);
            proposer
                .prestates
                .insert_for_tests(
                    prestate,
                    crate::config::PrestatePrograms {
                        aggregation_elf: vec![1],
                        range_elf: vec![1],
                    },
                )
                .await;
            for index in [1u64, 2] {
                let mut game = game_with(index, u32::MAX, 100 * index);
                game.proposal_status = ProposalStatus::Challenged;
                game.deadline = u64::MAX;
                game.absolute_prestate = prestate;
                proposer.state.write().await.games.insert(game.index, game);
            }

            // The scan spawns exactly one task (cap), not two. The spawned
            // task itself fails later against the unreachable RPC; only the
            // scheduling arithmetic is under test here.
            assert!(proposer.spawn_game_defense_tasks().await.unwrap());
            let tasks = proposer.tasks.lock().await;
            let proving_tasks = tasks
                .values()
                .filter(|(_, info)| matches!(info, TaskInfo::GameProving { .. }))
                .count();
            assert_eq!(proving_tasks, 1);
        }

        /// Network mode must verify a prestate's proving keys BEFORE any
        /// game is bonded on it: loaded-but-unverified artifacts do not open
        /// the creation gate, and a failed setup poisons the entry so the
        /// gate stays closed afterwards. Without this, `ensure_loaded` alone
        /// would admit the prestate and defer verification to the first
        /// defense - after a bond is already at stake.
        #[tokio::test]
        async fn network_creation_gate_requires_verified_keys() {
            let mut config = test_config();
            config.proof_provider = ProofProviderKind::Network;
            let signer = SignerLock::new(Signer::LocalSigner(PrivateKeySigner::random()));
            let provider =
                ProviderBuilder::default().connect_http("http://127.0.0.1:1".parse().unwrap());
            let factory = DisputeGameFactory::new(Address::ZERO, provider);
            let proposer =
                Proposer::new(config, signer, factory, ProofProvider::Mock(MockProofProvider))
                    .await
                    .unwrap();

            let prestate = B256::left_padding_from(&[0x77]);
            proposer
                .prestates
                .insert_for_tests(
                    prestate,
                    PrestatePrograms {
                        aggregation_elf: b"not an elf".to_vec(),
                        range_elf: b"not an elf".to_vec(),
                    },
                )
                .await;

            // Artifacts load, but the keys are unverified: creation must
            // pause immediately, and the gate kicks setup off the scheduler
            // path onto a background task.
            assert!(!proposer.prestate_usable_for_creation(prestate).await);

            // Setup on garbage ELFs fails and poisons the entry; wait for
            // the background verdict to land.
            let deadline = std::time::Instant::now() + std::time::Duration::from_secs(120);
            while proposer.prestates.key_verification_state(prestate).await.is_none() {
                assert!(std::time::Instant::now() < deadline, "key setup verdict did not land");
                tokio::time::sleep(std::time::Duration::from_millis(100)).await;
            }

            // Poisoned: the gate stays closed on the cheap paths too.
            assert!(!proposer.prestate_usable_for_creation(prestate).await);
            assert!(!proposer.prestates.ensure_loaded(prestate).await);
        }

        /// A prestate rotation (upgrade registering a NEW prestate) pauses
        /// creation until the new artifacts are published, while games on
        /// the old prestate stay owned (defense, resolution, and claims
        /// continue). Publish-heals-the-gate is pinned at the cache level in
        /// `prestate_gate::missing_programs_hard_pause`.
        #[tokio::test]
        async fn prestate_rotation_pauses_creation_and_keeps_old_games_owned() {
            let proposer = test_proposer().await;
            let old = B256::left_padding_from(&[0x0a]);
            let new = B256::left_padding_from(&[0x0b]);
            proposer
                .prestates
                .insert_for_tests(
                    old,
                    PrestatePrograms { aggregation_elf: vec![1], range_elf: vec![1] },
                )
                .await;

            // Old prestate: usable (mock mode never initializes keys).
            assert!(proposer.prestate_usable_for_creation(old).await);
            // Rotated-in prestate with unpublished artifacts: creation
            // pauses...
            assert!(!proposer.prestate_usable_for_creation(new).await);
            // ...while the old prestate stays in the owned set, so existing
            // games keep being defended, resolved, and claimed.
            assert!(proposer.prestates.known_prestates().await.contains(&old));

            // Operator publishes the new artifacts: creation resumes.
            proposer
                .prestates
                .insert_for_tests(
                    new,
                    PrestatePrograms { aggregation_elf: vec![2], range_elf: vec![2] },
                )
                .await;
            assert!(proposer.prestate_usable_for_creation(new).await);
        }
    }

    mod prestate_gate {
        use std::collections::HashSet;

        use alloy_primitives::B256;
        use alloy_transport_http::reqwest::Url;

        use super::super::{PrestateCache, PrestateKeyError};
        use crate::config::{PrestatePrograms, ProofProviderKind};

        const HASH_A: &str = "0x0101010101010101010101010101010101010101010101010101010101010101";

        fn artifact_dir(name: &str) -> std::path::PathBuf {
            let dir = std::env::temp_dir()
                .join(format!("kona-sp1-gate-test-{}-{name}", std::process::id()));
            std::fs::create_dir_all(&dir).unwrap();
            dir
        }

        fn write_artifacts(dir: &std::path::Path) {
            write_artifacts_with(dir, b"elf");
        }

        fn write_artifacts_with(dir: &std::path::Path, contents: &[u8]) {
            for s in [".agg.bin.gz", ".range.bin.gz"] {
                let mut gz =
                    flate2::write::GzEncoder::new(Vec::new(), flate2::Compression::default());
                std::io::Write::write_all(&mut gz, contents).unwrap();
                std::fs::write(dir.join(format!("{HASH_A}{s}")), gz.finish().unwrap()).unwrap();
            }
        }

        #[tokio::test]
        async fn loadable_programs_allow_creation_and_fill_cache() {
            let dir = artifact_dir("available");
            let hash: B256 = HASH_A.parse().unwrap();
            write_artifacts(&dir);
            let base = Url::from_directory_path(&dir).unwrap();
            let cache = PrestateCache::new(base);
            assert!(cache.ensure_loaded(hash).await);
            let programs = cache.programs(hash).await.expect("cached");
            assert_eq!(programs.aggregation_elf, b"elf");
            assert_eq!(programs.range_elf, b"elf");

            // Cache hit: succeeds even if the directory disappears.
            std::fs::remove_dir_all(&dir).unwrap();
            assert!(cache.ensure_loaded(hash).await);
        }

        #[tokio::test]
        async fn missing_programs_hard_pause() {
            let dir = artifact_dir("pause");
            let hash: B256 = HASH_A.parse().unwrap();
            let base = Url::from_directory_path(&dir).unwrap();
            // Zero retry window: this test pins the SELF-HEAL property, not
            // the negative-cache pacing (covered separately below).
            let cache = PrestateCache::with_retry_window(base, std::time::Duration::from_secs(0));
            assert!(!cache.ensure_loaded(hash).await);
            assert!(
                cache.programs(hash).await.is_none(),
                "failed load must not populate the cache"
            );

            // Hardfork flow: publishing the artifacts self-heals the gate
            // without a restart.
            write_artifacts(&dir);
            assert!(cache.ensure_loaded(hash).await);
        }

        #[tokio::test]
        async fn unknown_prestate_negative_cache_gates_refetch() {
            let dir = artifact_dir("negative-cache");
            let hash: B256 = HASH_A.parse().unwrap();
            let base = Url::from_directory_path(&dir).unwrap();
            let cache =
                PrestateCache::with_retry_window(base, std::time::Duration::from_secs(3600));
            assert!(!cache.ensure_loaded(hash).await);

            // Publishing the artifacts inside the retry window does NOT
            // self-heal yet: the miss is cached and no re-fetch happens.
            write_artifacts(&dir);
            assert!(!cache.ensure_loaded(hash).await);
            assert!(cache.programs(hash).await.is_none());
            assert!(!cache.known_prestates().await.contains(&hash));
        }

        #[tokio::test]
        async fn known_prestates_tracks_loaded_entries() {
            let dir = artifact_dir("known-set");
            let hash: B256 = HASH_A.parse().unwrap();
            write_artifacts(&dir);
            let base = Url::from_directory_path(&dir).unwrap();
            let cache = PrestateCache::new(base);
            assert!(cache.known_prestates().await.is_empty());
            assert!(cache.ensure_loaded(hash).await);
            assert_eq!(cache.known_prestates().await, HashSet::from([hash]));
        }

        #[tokio::test]
        async fn failed_check_hard_pauses() {
            let hash: B256 = HASH_A.parse().unwrap();
            let base = Url::parse("ftp://example.com/prestates").unwrap();
            assert!(!PrestateCache::new(base).ensure_loaded(hash).await);
        }

        #[tokio::test]
        async fn proof_keys_unavailable_in_mock_mode() {
            let cache = PrestateCache::new(Url::parse("file:///nonexistent").unwrap());
            let hash: B256 = HASH_A.parse().unwrap();
            cache
                .insert_for_tests(
                    hash,
                    PrestatePrograms { aggregation_elf: vec![1], range_elf: vec![1] },
                )
                .await;
            let err = cache.proof_keys(hash, ProofProviderKind::Mock).await.unwrap_err();
            assert!(err.to_string().contains("mock mode"), "unexpected error: {err}");
        }

        /// Network-mode key setup on artifacts that are not valid ELFs (the
        /// stub-artifact scenario) must poison the entry: the prestate
        /// leaves the known set, and later calls return the stored verdict
        /// without re-running setup.
        #[tokio::test]
        async fn invalid_elf_setup_poisons_and_excludes() {
            let cache = PrestateCache::new(Url::parse("file:///nonexistent").unwrap());
            let hash: B256 = HASH_A.parse().unwrap();
            cache
                .insert_for_tests(
                    hash,
                    PrestatePrograms {
                        aggregation_elf: b"not an elf".to_vec(),
                        range_elf: b"not an elf".to_vec(),
                    },
                )
                .await;
            assert!(cache.known_prestates().await.contains(&hash));

            let err = cache.proof_keys(hash, ProofProviderKind::Network).await.unwrap_err();
            let verdict = err.to_string();
            assert!(
                matches!(err.downcast_ref::<PrestateKeyError>(), Some(PrestateKeyError::Setup(_))),
                "expected a Setup poisoning, got: {verdict}"
            );
            assert!(!cache.known_prestates().await.contains(&hash));

            // The poisoned verdict is stored: same error, no re-setup.
            let again = cache.proof_keys(hash, ProofProviderKind::Network).await.unwrap_err();
            assert_eq!(again.to_string(), verdict);
        }

        /// With real artifacts (`KONA_SP1_ELF_DIR` pointing at the built
        /// ELFs and their `vkeys.toml`), key setup must ACCEPT the canonical
        /// prestate: the aggregation vkey's `bytes32_raw` - the bn254
        /// packing `cargo prove vkey` writes into `vkeys.toml`, which
        /// deployments use as `absolutePrestate()`. Skipped without
        /// the env var; run locally after `just build-elfs` or
        /// in the real-ELF lane.
        #[tokio::test]
        async fn real_elf_agg_vkey_matches_vkeys_toml_prestate() {
            let Ok(elf_dir) = std::env::var("KONA_SP1_ELF_DIR") else {
                return;
            };
            let dir = std::path::PathBuf::from(&elf_dir);
            let manifest = std::fs::read_to_string(dir.join("vkeys.toml")).unwrap();
            let prestate: B256 = manifest
                .lines()
                .find_map(|line| line.strip_prefix("super-aggregation = \""))
                .and_then(|rest| rest.strip_suffix('"'))
                .expect("vkeys.toml lacks a super-aggregation entry")
                .parse()
                .unwrap();

            let cache = PrestateCache::new(Url::parse("file:///nonexistent").unwrap());
            cache
                .insert_for_tests(
                    prestate,
                    PrestatePrograms {
                        aggregation_elf: std::fs::read(dir.join("super-aggregation-elf")).unwrap(),
                        range_elf: std::fs::read(dir.join("super-range-elf")).unwrap(),
                    },
                )
                .await;
            cache.proof_keys(prestate, ProofProviderKind::Network).await.expect(
                "real aggregation vkey must match the vkeys.toml prestate (bytes32 encoding)",
            );
            assert!(cache.known_prestates().await.contains(&prestate));
        }

        /// A poisoned prestate must close the creation gate (never bond a
        /// game the proposer has proven it cannot defend) and must heal
        /// only when the published artifacts actually change.
        #[tokio::test]
        async fn poisoned_prestate_closes_creation_gate_until_artifacts_change() {
            let dir = artifact_dir("poison-gate");
            let hash: B256 = HASH_A.parse().unwrap();
            write_artifacts_with(&dir, b"not an elf");
            let base = Url::from_directory_path(&dir).unwrap();
            let cache = PrestateCache::with_retry_window(base, std::time::Duration::from_secs(0));

            // Loads fine (existence only), then key setup poisons it.
            assert!(cache.ensure_loaded(hash).await);
            cache.proof_keys(hash, ProofProviderKind::Network).await.unwrap_err();
            assert!(!cache.known_prestates().await.contains(&hash));

            // The creation gate is closed while the same bad artifacts are
            // published: refetching identical bytes must NOT clear the
            // poisoned verdict.
            assert!(!cache.ensure_loaded(hash).await);
            assert!(!cache.known_prestates().await.contains(&hash));

            // Publishing CHANGED artifacts heals the entry without a
            // restart: the gate reopens and the prestate re-enters the
            // owned set (key setup will re-run on the next defense).
            write_artifacts_with(&dir, b"corrected elf");
            assert!(cache.ensure_loaded(hash).await);
            assert!(cache.known_prestates().await.contains(&hash));
            assert_eq!(cache.programs(hash).await.unwrap().aggregation_elf, b"corrected elf");
        }
    }

    fn state(games: Vec<Game>, anchor: Option<Game>) -> ProposerState {
        ProposerState {
            games: games.into_iter().map(|g| (g.index, g)).collect(),
            anchor_game: anchor,
            ..Default::default()
        }
    }

    #[test]
    fn invalidating_game_remembers_entire_cached_subtree() {
        let mut s = state(
            vec![
                game_with(0, u32::MAX, 100),
                game_with(1, 0, 200),
                game_with(2, 1, 300),
                game_with(3, 0, 250),
            ],
            None,
        );

        s.invalidate_subtree(U256::from(1));

        assert_eq!(s.invalid_games, HashSet::from([U256::from(1), U256::from(2)]));
        assert!(s.games.contains_key(&U256::from(0)));
        assert!(s.games.contains_key(&U256::from(3)));
        assert!(!s.games.contains_key(&U256::from(1)));
        assert!(!s.games.contains_key(&U256::from(2)));
    }

    #[test]
    fn factory_cache_reset_forgets_prior_history() {
        let anchor = game_with(0, u32::MAX, 100);
        let mut s = state(vec![anchor.clone(), game_with(1, 0, 200)], Some(anchor));
        s.cursor = Cursor::from(U256::from(1));
        s.canonical_head_index = Some(U256::from(1));
        s.canonical_head_sequence_number = Some(200);
        s.invalid_games.insert(U256::from(1));

        s.reset_factory_cache();

        assert_eq!(s.cursor, Cursor::none());
        assert!(s.games.is_empty());
        assert!(s.invalid_games.is_empty());
        assert!(s.anchor_game.is_none());
        assert!(s.canonical_head_index.is_none());
        assert_eq!(s.canonical_head_sequence_number, Some(200));
    }

    mod canonical_head {
        use super::*;

        #[test]
        fn no_anchor_selects_highest_sequence_game() {
            let s = state(vec![game_with(0, u32::MAX, 100), game_with(1, 0, 200)], None);
            assert_eq!(s.select_canonical_head().unwrap().index, U256::from(1));
        }

        #[test]
        fn anchor_subtree_selects_its_tip() {
            let anchor = game_with(5, 4, 100);
            let s = state(
                vec![anchor.clone(), game_with(6, 5, 200), game_with(7, 6, 300)],
                Some(anchor),
            );
            assert_eq!(s.select_canonical_head().unwrap().index, U256::from(7));
        }

        #[test]
        fn genesis_rooted_catchup_chain_follows_to_tip() {
            // Anchor 5 was the canonical head; the proposer could not parent on it (the contract
            // reverts when the parent is the anchor), so it built a fresh anchor-rooted chain
            // 6 <- 7 above it. The head must track the chain tip (7), not its root (6).
            let anchor = game_with(5, 4, 100);
            let s = state(
                vec![anchor.clone(), game_with(6, u32::MAX, 200), game_with(7, 6, 300)],
                Some(anchor),
            );
            assert_eq!(s.select_canonical_head().unwrap().index, U256::from(7));
        }

        #[test]
        fn genesis_rooted_deep_chain_follows_to_tip() {
            let anchor = game_with(5, 4, 100);
            let s = state(
                vec![
                    anchor.clone(),
                    game_with(6, u32::MAX, 200),
                    game_with(7, 6, 300),
                    game_with(8, 7, 400),
                ],
                Some(anchor),
            );
            assert_eq!(s.select_canonical_head().unwrap().index, U256::from(8));
        }

        #[test]
        fn earlier_lineage_override_follows_to_tip() {
            // A chain branching off an earlier lineage than the anchor (parent 2 < anchor parent 4)
            // must also be followed to its tip.
            let anchor = game_with(5, 4, 100);
            let s = state(
                vec![anchor.clone(), game_with(6, 2, 200), game_with(7, 6, 300)],
                Some(anchor),
            );
            assert_eq!(s.select_canonical_head().unwrap().index, U256::from(7));
        }

        #[test]
        fn multiple_catchup_chains_select_highest_tip() {
            // Repeated stall/recovery cycles can leave several anchor-rooted catch-up chains in
            // the cache at once. The head must be the highest-sequence tip across all qualifying
            // chains pooled together.
            let anchor = game_with(5, 4, 100);
            let s = state(
                vec![
                    anchor.clone(),
                    game_with(6, u32::MAX, 200),
                    game_with(7, 6, 300),
                    game_with(8, u32::MAX, 250),
                    game_with(9, 8, 400),
                ],
                Some(anchor),
            );
            assert_eq!(s.select_canonical_head().unwrap().index, U256::from(9));
        }
    }

    mod proposal_timing {
        use super::next_proposal_timestamp;

        /// `FlowGaps` #3 regression: a head lagging by many intervals yields one
        /// proposal at `max_proposable`, not a crawl across the gap.
        #[test]
        fn catch_up_proposes_at_max() {
            assert_eq!(next_proposal_timestamp(100, 6, 700_000), Some(700_000));
        }

        #[test]
        fn steady_state_proposes_at_fixed_step() {
            // Gap below two intervals: keep the fixed grid.
            assert_eq!(next_proposal_timestamp(100, 6, 107), Some(106));
            assert_eq!(next_proposal_timestamp(100, 6, 111), Some(106));
        }

        #[test]
        fn interval_not_elapsed_skips() {
            assert_eq!(next_proposal_timestamp(100, 6, 105), None);
            assert_eq!(next_proposal_timestamp(100, 6, 100), None);
        }

        #[test]
        fn catch_up_boundary_is_two_intervals() {
            // Exactly two intervals of lag jumps to the tip.
            assert_eq!(next_proposal_timestamp(100, 6, 112), Some(112));
        }

        #[test]
        fn overflow_is_none() {
            assert_eq!(next_proposal_timestamp(u64::MAX, 1, u64::MAX), None);
        }
    }

    mod collision_walk {
        use super::super::advance_collision_timestamp;

        /// `FlowGaps` #2 regression: the UUID-collision walk never advances past
        /// the safety bound.
        #[test]
        fn collision_walk_bounded() {
            assert_eq!(advance_collision_timestamp(100, 102), Some(101));
            assert_eq!(advance_collision_timestamp(101, 102), Some(102));
            assert_eq!(advance_collision_timestamp(102, 102), None);
            assert_eq!(advance_collision_timestamp(u64::MAX, u64::MAX), None);
        }
    }
    mod claim_state_machine {
        use super::super::{BondClaimAction, bond_claim_action, withdrawal_matured};
        use alloy_primitives::U256;

        const DELAY: u64 = 302_400;

        /// Full two-phase transition: unlock -> wait for delay -> payout -> done.
        #[test]
        fn claim_state_machine() {
            let credit = U256::from(1_000);
            let amount = U256::from(1_000);

            // Resolved but not finalized: nothing actionable.
            assert_eq!(
                bond_claim_action(false, credit, U256::ZERO, 0, DELAY, 100),
                BondClaimAction::Wait
            );
            // Finalized with credit: phase 1 unlock.
            assert_eq!(
                bond_claim_action(true, credit, U256::ZERO, 0, DELAY, 100),
                BondClaimAction::Unlock
            );
            // Unlocked, delay not elapsed: wait.
            assert_eq!(
                bond_claim_action(true, U256::ZERO, amount, 100, DELAY, 100 + DELAY - 1),
                BondClaimAction::Wait
            );
            // Delay elapsed at L1 time: phase 2 payout.
            assert_eq!(
                bond_claim_action(true, U256::ZERO, amount, 100, DELAY, 100 + DELAY),
                BondClaimAction::Payout
            );
            // Paid out: evict.
            assert_eq!(
                bond_claim_action(true, U256::ZERO, U256::ZERO, 0, DELAY, 100 + DELAY),
                BondClaimAction::Done
            );
        }

        /// The maturity check is against chain time; a zero timestamp means
        /// no withdrawal was recorded.
        #[test]
        fn withdrawal_maturity_uses_chain_time() {
            assert!(!withdrawal_matured(0, DELAY, u64::MAX));
            assert!(!withdrawal_matured(100, DELAY, 100 + DELAY - 1));
            assert!(withdrawal_matured(100, DELAY, 100 + DELAY));
            assert!(!withdrawal_matured(u64::MAX, DELAY, u64::MAX));
        }

        /// Unlock strictly precedes payout. The both-apply state is
        /// on-chain-unreachable for a consistent read (claimCredit zeroes
        /// credit atomically with the withdrawal-recording unlock); the test
        /// pins the branch order that keeps it safe anyway.
        #[test]
        fn unlock_precedes_payout_when_both_apply() {
            assert_eq!(
                bond_claim_action(
                    true,
                    U256::from(1_000),
                    U256::from(1_000),
                    100,
                    DELAY,
                    100 + DELAY
                ),
                BondClaimAction::Unlock
            );
        }

        /// Documents the branch order for an on-chain-unreachable state: a
        /// recorded withdrawal implies a prior claimCredit, which required
        /// finality - the function still pays out rather than waiting.
        #[test]
        fn unfinalized_matured_withdrawal_pays_out() {
            assert_eq!(
                bond_claim_action(false, U256::ZERO, U256::from(1_000), 100, DELAY, 100 + DELAY),
                BondClaimAction::Payout
            );
        }
    }

    mod deadline_lag {
        use super::super::{MAX_GAME_DEADLINE_LAG, beyond_deadline_lag, pending_evictable};

        /// The walk cutoff is exclusive at the boundary and symmetric: a game
        /// exactly `MAX_GAME_DEADLINE_LAG` away is still in; one second
        /// beyond, in either direction, is out.
        #[test]
        fn cutoff_is_exclusive_at_the_boundary() {
            assert!(!beyond_deadline_lag(1_000_000, 1_000_000));
            assert!(!beyond_deadline_lag(1_000_000, 1_000_000 + MAX_GAME_DEADLINE_LAG));
            assert!(beyond_deadline_lag(1_000_000, 1_000_000 + MAX_GAME_DEADLINE_LAG + 1));
            assert!(beyond_deadline_lag(1_000_000 + MAX_GAME_DEADLINE_LAG + 1, 1_000_000));
        }

        /// Pending eviction is one-sided: a game behind the anchor beyond
        /// the lag is evicted, one exactly at the lag is kept, and a game
        /// ahead of a stalled anchor is never evicted no matter how far.
        #[test]
        fn eviction_only_fires_behind_the_anchor() {
            let anchor = 1_000_000 + MAX_GAME_DEADLINE_LAG + 1;
            assert!(pending_evictable(anchor, 1_000_000));
            assert!(!pending_evictable(anchor - 1, 1_000_000));
            assert!(!pending_evictable(1_000_000, 1_000_000));
            assert!(!pending_evictable(1_000_000, 1_000_000 + MAX_GAME_DEADLINE_LAG + 1));
            assert!(!pending_evictable(1_000_000, u64::MAX));
        }
    }

    mod pending_games {
        use super::*;

        /// `FlowGaps` #1 regression (state level): pending games are kept out
        /// of `state.games`, and that exclusion is what makes them
        /// parent-ineligible - inserted, the far-future game would win head
        /// selection.
        #[test]
        fn pending_games_are_not_parent_eligible() {
            let validated = game_with(0, u32::MAX, 100);
            let far_future = game_with(1, u32::MAX, 10_000);

            let without_pending = state(vec![validated.clone()], None);
            assert_eq!(without_pending.select_canonical_head().unwrap().l2_sequence_number, 100);

            let with_pending = state(vec![validated, far_future], None);
            assert_eq!(with_pending.select_canonical_head().unwrap().l2_sequence_number, 10_000);
        }
    }
}
