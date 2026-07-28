//! Core proposer: state sync, canonical-head selection, and the
//! creation/resolution/bond-claim task scheduler.
//!
//! Ported from op-succinct's `fault-proof/src/proposer.rs` (@ 13716c2c),
//! adapted for the super-root `ZKDisputeGame`: supernode-sourced claims,
//! `parentIndex || superRootProof` extraData, vkey-based identity, and
//! two-phase `DelayedWETH` bond claiming. Defense/proving scheduling is
//! intentionally absent (defend path, #21463).

use std::{
    collections::{HashMap, HashSet},
    sync::{
        Arc,
        atomic::{AtomicU64, Ordering},
    },
    time::Duration,
};

use alloy_eips::{BlockId, BlockNumberOrTag};
use alloy_primitives::{Address, B256, U256};
use alloy_provider::{Provider, ProviderBuilder};
use alloy_sol_types::SolEvent;
use anyhow::{Context, Result, bail};
use kona_sp1_host_utils::metrics::MetricsGauge;
use tokio::{sync::RwLock, time};

use crate::{
    FactoryTrait, L1Provider, TX_REVERTED_PREFIX, TxErrorExt, ZK_GAME_TYPE,
    config::{PrestatePrograms, ProposerConfig, load_prestate},
    contract::{
        AnchorStateRegistry::{self, AnchorStateRegistryInstance},
        DelayedWETH,
        DisputeGameFactory::{DisputeGameCreated, DisputeGameFactoryInstance},
        GameStatus, ProposalStatus, ZKDisputeGame, ZKGameArgs,
    },
    is_parent_resolved,
    metrics::ProposerGauge,
    signer::SignerLock,
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

/// Information about a running task
#[derive(Clone, Debug)]
pub enum TaskInfo {
    /// Task creating a new game at the given super-root timestamp.
    GameCreation {
        /// Super-root timestamp (`l2SequenceNumber`) of the game being created.
        sequence_number: u64,
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
    pub fn log_startup_info(&self, prestates_url: &alloy_transport_http::reqwest::Url) {
        tracing::info!(
            version = %self.version,
            prestates_url = %prestates_url,
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
#[derive(Clone, Debug, serde::Serialize, serde::Deserialize)]
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
    /// Returns true if the proposer created this game (owned = should
    /// resolve and claim its bond). Ownership is creator-based, not
    /// prestate-based: games created before a prestate rotation are still
    /// ours to resolve and claim.
    ///
    /// The defend path widens this: any game the proposer would prove when
    /// challenged (its own games and their ancestors, since `prove()` is
    /// permissionless and proving earns the challenger bond) is also a game
    /// it must resolve and claim. That set is settled with the proving
    /// logic (#21463).
    pub fn is_owned(&self, proposer_address: Address) -> bool {
        self.creator == proposer_address
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
#[derive(Default)]
struct ProposerState {
    anchor_game: Option<Game>,
    canonical_head_index: Option<U256>,
    canonical_head_sequence_number: Option<u64>,
    cursor: Cursor,
    games: HashMap<U256, Game>,
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

    /// Remove a game subtree from the cache.
    ///
    /// Used when a game is invalidated (i.e., `ChallengerWins`) and its entire subtree must be
    /// dropped.
    fn remove_subtree(&mut self, root_index: U256) {
        tracing::info!(?root_index, "Removing subtree from cache");
        for index in self.descendants_of(root_index) {
            tracing::info!(?index, "Removing game from cache");
            self.games.remove(&index);
        }
    }

    /// Selects the canonical head: the highest-L2-block game on the best valid chain.
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

/// Snapshot of the proposer's cached state for testing and monitoring.
#[derive(Clone, Debug)]
pub struct ProposerStateSnapshot {
    /// Factory index of the anchor game, if one is set.
    pub anchor_index: Option<U256>,
    /// Factory index of the current canonical head game, if any.
    pub canonical_head_index: Option<U256>,
    /// Super-root timestamp of the canonical head (0 when no head exists).
    pub canonical_head_sequence_number: u64,
    /// Cached games as `(factory index, game address)` pairs.
    pub games: Vec<(U256, Address)>,
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
    /// Loaded prestate programs keyed by `absolutePrestate()` hash, fetched
    /// from `PRESTATES_URL` on demand (see [`ensure_prestate_loaded`]).
    pub prestates: Arc<RwLock<HashMap<B256, Arc<PrestatePrograms>>>>,
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
    /// Games seen on-chain whose super-root data is not yet obtainable from
    /// this node - the timestamp is not yet safe, or the query failed
    /// (including permanently, e.g. timestamps predating the node's recorded
    /// safe history). Excluded from the DAG (and parent eligibility) but
    /// re-validated each sync - unlike terminal invalidity, pending games
    /// must not be dropped by the cursor (see `fetch_game`).
    pending_games: Arc<RwLock<HashSet<U256>>>,
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
    /// Creates a new proposer instance with the provided signer and factory
    /// contract instance.
    pub async fn new(
        config: ProposerConfig,
        signer: SignerLock,
        factory: DisputeGameFactoryInstance<P>,
    ) -> Result<Self> {
        let identity = ProposerIdentity::new();
        identity.log_startup_info(&config.prestates_url);

        let l1_provider = ProviderBuilder::default().connect_http(config.l1_rpc.clone());
        let superroot_client = SuperrootClient::new(config.supernode_rpc.as_str())?;

        Ok(Self {
            config,
            signer,
            l1_provider,
            superroot_client,
            factory: Arc::new(factory),
            prestates: Arc::new(RwLock::new(HashMap::new())),
            tasks: Arc::new(tokio::sync::Mutex::new(HashMap::new())),
            next_task_id: Arc::new(AtomicU64::new(1)),
            state: Arc::new(RwLock::new(ProposerState::default())),
            identity,
            last_synced_l1_block: Arc::new(AtomicU64::new(0)),
            last_created_game_l2_sequence_number: Arc::new(AtomicU64::new(0)),
            last_created_game_address: Arc::new(tokio::sync::Mutex::new(Address::ZERO)),
            pending_games: Arc::new(RwLock::new(HashSet::new())),
        })
    }

    /// Returns a lightweight snapshot of the proposer's cached state.
    pub async fn state_snapshot(&self) -> ProposerStateSnapshot {
        let state = self.state.read().await;
        ProposerStateSnapshot {
            anchor_index: state.anchor_game.as_ref().map(|game| game.index),
            canonical_head_index: state.canonical_head_index,
            canonical_head_sequence_number: state
                .canonical_head_sequence_number
                .unwrap_or_default(),
            games: state.games.values().map(|game| (game.index, game.address)).collect(),
        }
    }

    /// Returns a copy of a game's full internal state for testing.
    pub async fn get_game(&self, index: U256) -> Option<Game> {
        let state = self.state.read().await;
        state.games.get(&index).cloned()
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
        if !ensure_prestate_loaded(
            &self.prestates,
            &self.config.prestates_url,
            game_args.absolute_prestate,
        )
        .await
        {
            // Not fatal: creation stays paused until the artifacts appear
            // under PRESTATES_URL (ensure_prestate_loaded logged why).
            tracing::warn!(
                registered = %game_args.absolute_prestate,
                "registered prestate programs unavailable; creation will pause"
            );
        }

        // Fetch and validate the anchor root from the currently registered registry.
        let registry =
            AnchorStateRegistry::new(game_args.anchor_state_registry, self.l1_provider.clone());
        let anchor = registry.getAnchorRoot().call().await?;
        anyhow::ensure!(
            anchor._0 != B256::ZERO,
            "anchor state registry has no anchor root (game creation would revert)"
        );
        anchor._1.try_into().context("anchor sequence number exceeds u64")
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
    /// 1. Load new games.
    ///    - Incrementally fetch games from the factory, starting from the latest and working
    ///      backwards to the oldest unprocessed game, stopping at games exceeding the maximum
    ///      deadline lag from the anchor game (`MAX_GAME_DEADLINE_LAG`).
    ///    - Games are validated (correct type, valid output root) before being added.
    /// 2. Synchronize the status of all cached games.
    ///    - Games are removed (along with their subtree) if their parent is not in the cache.
    ///    - Games are marked for resolution if the parent is resolved, the game is over, and it's
    ///      own game.
    ///    - Games are marked for bond claim if they are finalized and there is credit to claim.
    /// 3. Evict games from the cache.
    ///    - Games that are finalized but there is no credit left to claim.
    ///    - The entire subtree of a `ChallengerWins` game.
    pub async fn sync_games(&self, pinned_block: BlockId, pinned_timestamp: u64) -> Result<()> {
        let pinned_latest_index = self.factory.fetch_latest_game_index(pinned_block).await?;

        // 1. Load new games.
        let latest_index = if let Some(index) = pinned_latest_index {
            Cursor::from(index)
        } else {
            // No games at pinned block. Reset cursor so future sync cycles can discover
            // games once they become confirmed.
            let mut state = self.state.write().await;
            state.cursor = Cursor::none();
            return Ok(());
        };

        let anchor_address = self
            .registered_anchor_state_registry(pinned_block)
            .await?
            .anchorGame()
            .block(pinned_block)
            .call()
            .await?;

        let cursor = {
            let state = self.state.read().await;
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
                Cursor::none()
            } else {
                current_cursor
            }
        };

        let mut index = latest_index.clone();
        let mut anchor_deadline: Option<u64> = None;
        let mut invalid_game_ids = Vec::new();
        let mut newly_pending: Vec<U256> = Vec::new();

        loop {
            if index == cursor {
                break;
            }

            let i = index.index().expect("must have an index here");
            let fetch_result = self.fetch_game(i, pinned_block).await?;

            match fetch_result {
                GameFetchResult::ValidGame { game_address, deadline } => {
                    // First time we hit the anchor, record its deadline
                    if game_address == anchor_address {
                        anchor_deadline = Some(deadline);
                    }

                    // Once we know the anchor deadline, enforce the lag constraint.
                    if let Some(anchor_d) = anchor_deadline &&
                        anchor_d.abs_diff(deadline) > MAX_GAME_DEADLINE_LAG
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
                GameFetchResult::Pending { index } => {
                    newly_pending.push(index);
                }
            }

            index.step_back();
        }

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
                state.remove_subtree(idx);
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
            for idx in previously_pending {
                match self.fetch_game(idx, pinned_block).await? {
                    GameFetchResult::Pending { .. } => {}
                    GameFetchResult::InvalidGame { index } => {
                        self.pending_games.write().await.remove(&index);
                    }
                    _ => {
                        self.pending_games.write().await.remove(&idx);
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
                .map(|game| (game.index, game.address, game.weth, game.anchor_state_registry))
                .collect::<Vec<_>>()
        };

        if !games.is_empty() {
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

            for (index, game_address, game_weth, game_asr) in games {
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
                        // The only proof-provided statuses reachable in the create-path
                        // proposer are those set by third parties; they still make the
                        // game "over" for resolution purposes.
                        let is_game_over = match proposal_status {
                            ProposalStatus::Unchallenged => now_ts > deadline,
                            ProposalStatus::UnchallengedAndValidProofProvided |
                            ProposalStatus::ChallengedAndValidProofProvided => true,
                            _ => false,
                        };
                        let creator = contract.gameCreator().block(pinned_block).call().await?;
                        let is_own_game = match proposal_status {
                            ProposalStatus::Unchallenged => creator == signer_address,
                            ProposalStatus::UnchallengedAndValidProofProvided |
                            ProposalStatus::ChallengedAndValidProofProvided => {
                                creator == signer_address || claim_data.prover == signer_address
                            }
                            _ => false,
                        };

                        // Cached games already passed game-type validation in fetch_game.
                        let should_attempt_to_resolve =
                            parent_resolved && is_game_over && is_own_game;

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
                        state.remove_subtree(index);
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
    /// game implementation's prestate (see [`ensure_prestate_loaded`]).
    async fn registered_prestate_known(&self) -> Result<bool> {
        let args = self.registered_game_args(BlockId::latest()).await?;
        Ok(ensure_prestate_loaded(
            &self.prestates,
            &self.config.prestates_url,
            args.absolute_prestate,
        )
        .await)
    }

    /// Returns the loaded program ELFs for `prestate`, if present in the
    /// cache. The defend path proves with these.
    pub async fn prestate_programs(&self, prestate: B256) -> Option<Arc<PrestatePrograms>> {
        self.prestates.read().await.get(&prestate).cloned()
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
        let candidates = {
            let state = self.state.read().await;
            state
                .games
                .values()
                // Only resolve owned games (created by this proposer).
                .filter(|game| game.is_owned(self.signer.address()))
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
        let candidates = {
            let state = self.state.read().await;
            state
                .games
                .values()
                // Only claim bonds for owned games (created by this proposer).
                .filter(|game| game.is_owned(self.signer.address()))
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
            if let (Ok(credit), Ok(withdrawal)) = (credit, withdrawal) {
                if credit == U256::ZERO && withdrawal.amount == U256::ZERO {
                    tracing::info!(
                        game_index = %game.index,
                        game_address = ?game.address,
                        "Skipping claim: bond already claimed on chain"
                    );
                    continue;
                }
                if credit == U256::ZERO && withdrawal.amount > U256::ZERO {
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

            ProposerGauge::GamesBondsClaimed.increment(1.0);
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
    /// registry, disrespected game type at creation, or a claim contradicting
    /// the canonical super root. A timestamp not yet safe from this node's
    /// view yields `Pending` instead: excluded from the DAG but re-validated
    /// on later syncs.
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

        // Capture the game's own immutable args: bond claims bind its WETH
        // and finality checks bind its registry, which can differ from the
        // currently registered ones across upgrades. Games under an older
        // registry stay in the DAG; the factory re-validates parents at
        // create time.
        let game_asr = contract.anchorStateRegistry().block(pinned_block).call().await?;
        let game_weth = contract.weth().block(pinned_block).call().await?;
        let creator = contract.gameCreator().block(pinned_block).call().await?;

        let sequence_number: u64 = contract
            .l2SequenceNumber()
            .block(pinned_block)
            .call()
            .await?
            .try_into()
            .context("game sequence number exceeds u64")?;
        let claim = contract.rootClaim().block(pinned_block).call().await?;
        let was_respected =
            contract.wasRespectedGameTypeWhenCreated().block(pinned_block).call().await?;
        let status = GameStatus::try_from(contract.status().block(pinned_block).call().await?)?;
        let claim_data = contract.claimData().block(pinned_block).call().await?;
        let absolute_prestate = contract.absolutePrestate().block(pinned_block).call().await?;

        // Enums are uint8 in the ABI; convert once at the read boundary.
        let (parent_index, proposal_status, deadline) = (
            claim_data.parentIndex,
            ProposalStatus::try_from(claim_data.status)?,
            claim_data.deadline,
        );

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
                return Ok(GameFetchResult::Pending { index });
            }
        };
        let super_root = match SuperrootClient::super_root_at(&response) {
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
                return Ok(GameFetchResult::Pending { index });
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
                return Ok(GameFetchResult::Pending { index });
            }
            Some(super_root) if super_root.super_root != claim => {
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

        if !game.is_owned(self.signer.address()) {
            tracing::info!(game_index = %index, "Discovered game created by another proposer - tracking for DAG but not resolving/claiming");
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
            let Some(super_root) = SuperrootClient::super_root_at(&response)? else {
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
                let game_address = self.create_game(super_root.super_root, extra_data).await?;

                // Record the sequence number and address so should_create_game() skips
                // duplicate creation while the pinned cache hasn't caught up to this game.
                self.last_created_game_l2_sequence_number.store(sequence_number, Ordering::Relaxed);
                *self.last_created_game_address.lock().await = game_address;
                return Ok(());
            }

            // An identical VALID game already exists (UUID collision). Advance the
            // timestamp - bounded by the safety limit - and refetch a fresh super
            // root (the proof embeds the timestamp). On reaching the bound, defer:
            // the next sync adopts the existing game as canonical head.
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
        // First check if we should create a game
        let (should_create, next_sequence_number, parent_game_index) =
            self.should_create_game().await?;
        if !should_create {
            return Ok(false);
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

            ProposerGauge::GamesCreated.increment(1.0);

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

/// Policy for game creation when the registered prestate's programs cannot
/// be loaded. Hard-pause (`false`) is the conservative default: never bond a
/// game we could not defend. Flip to `true` for warn-and-continue.
pub const CREATE_ON_UNKNOWN_PRESTATE: bool = false;

/// Ensures the program ELFs for `registered` are loaded into `cache`,
/// fetching them from `url` on a miss (see [`load_prestate`]). Returns
/// whether game creation may proceed for that prestate.
///
/// The artifact directory is consulted live on every cache miss, so a
/// hardfork that rotates the registered prestate only requires publishing
/// the new program artifacts, not restarting the proposer. When loading
/// fails, returns [`CREATE_ON_UNKNOWN_PRESTATE`].
pub async fn ensure_prestate_loaded(
    cache: &RwLock<HashMap<B256, Arc<PrestatePrograms>>>,
    url: &alloy_transport_http::reqwest::Url,
    registered: B256,
) -> bool {
    if cache.read().await.contains_key(&registered) {
        return true;
    }
    match load_prestate(url, registered).await {
        Ok(programs) => {
            tracing::info!(
                prestate = %registered,
                aggregation_elf_bytes = programs.aggregation_elf.len(),
                range_elf_bytes = programs.range_elf.len(),
                "Loaded prestate programs"
            );
            cache.write().await.insert(registered, Arc::new(programs));
            true
        }
        Err(e) => {
            tracing::warn!(
                registered = %registered,
                error = %e,
                "Failed to load the registered prestate's programs - skipping game creation \
                 (publish the artifacts under PRESTATES_URL if this is a hardfork)"
            );
            ProposerGauge::UnknownRegisteredPrestate.increment(1.0);
            CREATE_ON_UNKNOWN_PRESTATE
        }
    }
}

#[cfg(test)]
mod tests {
    use super::{Game, ProposerState, next_proposal_timestamp};
    use crate::contract::{GameStatus, ProposalStatus};
    use alloy_primitives::{Address, B256, U256};

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
        use super::*;

        #[test]
        fn owned_by_creator_regardless_of_prestate() {
            // Rotation scenario: a game created before a prestate upgrade
            // (its prestate differs from anything current) is still ours to
            // resolve and claim.
            let us = Address::left_padding_from(&[0xaa]);
            let mut game = game_with(1, u32::MAX, 100);
            game.creator = us;
            game.absolute_prestate = B256::left_padding_from(&[0xde, 0xad]);
            assert!(game.is_owned(us));
        }

        #[test]
        fn foreign_creator_is_not_owned_even_with_matching_prestate() {
            let us = Address::left_padding_from(&[0xaa]);
            let mut game = game_with(1, u32::MAX, 100);
            game.creator = Address::left_padding_from(&[0xbb]);
            game.absolute_prestate = B256::ZERO;
            assert!(!game.is_owned(us));
        }
    }

    mod prestate_gate {
        use super::super::ensure_prestate_loaded;
        use crate::config::PrestatePrograms;
        use alloy_primitives::B256;
        use alloy_transport_http::reqwest::Url;
        use std::{collections::HashMap, sync::Arc};
        use tokio::sync::RwLock;

        const HASH_A: &str = "0x0101010101010101010101010101010101010101010101010101010101010101";

        fn artifact_dir(name: &str) -> std::path::PathBuf {
            let dir = std::env::temp_dir()
                .join(format!("kona-sp1-gate-test-{}-{name}", std::process::id()));
            std::fs::create_dir_all(&dir).unwrap();
            dir
        }

        fn write_artifacts(dir: &std::path::Path) {
            for s in [".agg.bin.gz", ".range.bin.gz"] {
                let mut gz =
                    flate2::write::GzEncoder::new(Vec::new(), flate2::Compression::default());
                std::io::Write::write_all(&mut gz, b"elf").unwrap();
                std::fs::write(dir.join(format!("{HASH_A}{s}")), gz.finish().unwrap()).unwrap();
            }
        }

        fn empty_cache() -> RwLock<HashMap<B256, Arc<PrestatePrograms>>> {
            RwLock::new(HashMap::new())
        }

        #[tokio::test]
        async fn loadable_programs_allow_creation_and_fill_cache() {
            let dir = artifact_dir("available");
            let hash: B256 = HASH_A.parse().unwrap();
            write_artifacts(&dir);
            let base = Url::from_directory_path(&dir).unwrap();
            let cache = empty_cache();
            assert!(ensure_prestate_loaded(&cache, &base, hash).await);
            let programs = cache.read().await.get(&hash).cloned().expect("cached");
            assert_eq!(programs.aggregation_elf, b"elf");
            assert_eq!(programs.range_elf, b"elf");

            // Cache hit: succeeds even if the directory disappears.
            std::fs::remove_dir_all(&dir).unwrap();
            assert!(ensure_prestate_loaded(&cache, &base, hash).await);
        }

        #[tokio::test]
        async fn missing_programs_hard_pause() {
            let dir = artifact_dir("pause");
            let hash: B256 = HASH_A.parse().unwrap();
            let base = Url::from_directory_path(&dir).unwrap();
            let cache = empty_cache();
            assert!(!ensure_prestate_loaded(&cache, &base, hash).await);
            assert!(cache.read().await.is_empty(), "failed load must not populate the cache");

            // Hardfork flow: publishing the artifacts self-heals the gate
            // without a restart.
            write_artifacts(&dir);
            assert!(ensure_prestate_loaded(&cache, &base, hash).await);
        }

        #[tokio::test]
        async fn failed_check_hard_pauses() {
            let hash: B256 = HASH_A.parse().unwrap();
            let base = Url::parse("ftp://example.com/prestates").unwrap();
            assert!(!ensure_prestate_loaded(&empty_cache(), &base, hash).await);
        }
    }

    fn state(games: Vec<Game>, anchor: Option<Game>) -> ProposerState {
        ProposerState {
            games: games.into_iter().map(|g| (g.index, g)).collect(),
            anchor_game: anchor,
            ..Default::default()
        }
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
    }

    mod pending_games {
        use super::*;

        /// `FlowGaps` #1 regression (state level): a pending game is not in the
        /// DAG, so it can never be selected as canonical head / parent.
        #[test]
        fn pending_games_are_not_parent_eligible() {
            // Only the validated game is in the DAG; the far-future game at
            // sequence 10_000 was classified pending and never inserted.
            let s = state(vec![game_with(0, u32::MAX, 100)], None);
            assert_eq!(s.select_canonical_head().unwrap().l2_sequence_number, 100);
        }
    }
}
