//! Core proposer: state sync, canonical-head selection, and the
//! creation/resolution/bond-claim/proving task scheduler.
//!
//! Ported from op-succinct's `fault-proof/src/proposer.rs` (@ 13716c2c),
//! adapted for the super-root `ZKDisputeGame`: supernode-sourced claims,
//! `parentIndex || superRootProof` extraData, prestate-based ownership, and
//! two-phase `DelayedWETH` bond claiming. The proving scheduler defends
//! challenged games and, in fast finality mode, proves owned games while
//! they are still unchallenged, via [`crate::proving`].

use std::{
    collections::{HashMap, HashSet},
    num::NonZeroU64,
    sync::{
        Arc,
        atomic::{AtomicBool, AtomicU64, Ordering},
    },
    time::{Duration, Instant},
};

use alloy_eips::BlockId;
use alloy_primitives::{Address, B256, U256};
use alloy_provider::ProviderBuilder;
use alloy_transport_http::reqwest::Url;
use anyhow::{Context, Result, anyhow, bail};
use kona_sp1_host_utils::metrics::MetricsGauge;
use kona_sp1_super_range_executor::HostInputs;
use sp1_sdk::HashableKey;
use tokio::{
    sync::{Mutex, OnceCell, RwLock},
    time,
};
use tracing::Instrument;

use crate::{
    L1Provider, TxErrorExt, ZK_GAME_TYPE,
    adapters::{
        ProductionActionExecutor, ProductionL1View, ProductionProofEngine,
        ProductionSuperRootSource, SystemQueryTime,
    },
    config::{PrestatePrograms, ProofProviderKind, ProposalSafety, ProposerConfig, load_prestate},
    contract::{DisputeGameFactory::DisputeGameFactoryInstance, GameStatus, ProposalStatus},
    metrics::ProposerGauge,
    ports::{
        ActionExecutor, BondState, ClaimPreflight, GameLifecycle, L1View, ProofEngine, ProofInputs,
        QueryTime, SuperRootSource,
    },
    prover::{ProofKeys, ProofProvider, setup_proof_keys},
    proving::{GameProofInputs, fetch_span_responses, is_unprovable, response_trusted},
    signer::{FeeCaps, SignerLock},
    superroot::{SuperrootClient, zk_extra_data},
};

#[cfg(test)]
mod scenario;

/// Max allowed time (secs) between a game's deadline and the anchor game's deadline.
///
/// Games beyond this threshold are skipped during incremental syncs to cut startup latency and
/// avoid caching stale data.
///
/// The 14-day window is chosen with a 7-day challenge period in mind, plus a 7-day buffer,
/// ensuring all actionable games are included under normal conditions.
pub const MAX_GAME_DEADLINE_LAG: u64 = 60 * 60 * 24 * 14; // 14 days

/// Nonzero identifier assigned to a proposer task.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub(crate) struct TaskId(NonZeroU64);

impl TaskId {
    fn allocate(next_task_id: &AtomicU64) -> Self {
        let task_id = next_task_id
            .fetch_update(Ordering::Relaxed, Ordering::Relaxed, |task_id| {
                if task_id == 0 { None } else { task_id.checked_add(1) }
            })
            .expect("task ID counter must be nonzero and not exhausted");
        Self(NonZeroU64::new(task_id).expect("task ID was validated before allocation"))
    }

    const fn get(self) -> u64 {
        self.0.get()
    }
}

type TaskHandle = tokio::task::JoinHandle<Result<TaskSuccess>>;
type TaskMap = HashMap<TaskId, (TaskHandle, OperationSummary)>;

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub(crate) enum TaskClass {
    Creation,
    Proving,
    Resolution,
    Claim,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub(crate) enum ProvingPurpose {
    Defense,
    FastFinality,
}

#[derive(Clone, Debug, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub(crate) enum OperationTarget {
    Creation { sequence_number: u64, parent_game_index: u32 },
    Game { factory_index: U256, address: Address },
    AllGames,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
enum TaskDeduplicationKey {
    Creation,
    Proving(Address),
    Resolution,
    Claim,
}

#[derive(Clone, Debug, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub(crate) enum OperationSummary {
    ProposeGame { sequence_number: u64, parent_game_index: u32 },
    ReconcileCreation { sequence_number: u64, parent_game_index: u32 },
    ProveGame { factory_index: U256, address: Address, purpose: ProvingPurpose },
    ResolutionSweep,
    ClaimSweep,
}

impl OperationSummary {
    const fn class(&self) -> TaskClass {
        match self {
            Self::ProposeGame { .. } | Self::ReconcileCreation { .. } => TaskClass::Creation,
            Self::ProveGame { .. } => TaskClass::Proving,
            Self::ResolutionSweep => TaskClass::Resolution,
            Self::ClaimSweep => TaskClass::Claim,
        }
    }

    const fn target(&self) -> OperationTarget {
        match self {
            Self::ProposeGame { sequence_number, parent_game_index } |
            Self::ReconcileCreation { sequence_number, parent_game_index } => {
                OperationTarget::Creation {
                    sequence_number: *sequence_number,
                    parent_game_index: *parent_game_index,
                }
            }
            Self::ProveGame { factory_index, address, .. } => {
                OperationTarget::Game { factory_index: *factory_index, address: *address }
            }
            Self::ResolutionSweep | Self::ClaimSweep => OperationTarget::AllGames,
        }
    }

    const fn deduplication_key(&self) -> TaskDeduplicationKey {
        match self {
            Self::ProposeGame { .. } | Self::ReconcileCreation { .. } => {
                TaskDeduplicationKey::Creation
            }
            Self::ProveGame { address, .. } => TaskDeduplicationKey::Proving(*address),
            Self::ResolutionSweep => TaskDeduplicationKey::Resolution,
            Self::ClaimSweep => TaskDeduplicationKey::Claim,
        }
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum TaskSuccess {
    Completed,
    TerminallyUnprovable,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) enum TaskFailureClass {
    ReturnedError(String),
    Panicked,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) enum TaskCompletionOutcome {
    Success,
    TerminallyUnprovable,
    Failed(TaskFailureClass),
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) struct TaskCompletion {
    pub(crate) task_id: TaskId,
    pub(crate) class: TaskClass,
    pub(crate) target: OperationTarget,
    pub(crate) outcome: TaskCompletionOutcome,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) struct ActiveTaskSummary {
    pub(crate) task_id: TaskId,
    pub(crate) class: TaskClass,
    pub(crate) target: OperationTarget,
    pub(crate) operation: OperationSummary,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) struct ScheduledOperation {
    pub(crate) task_id: TaskId,
    pub(crate) operation: OperationSummary,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum SyncDisposition {
    Advanced,
    UnchangedConfirmedHead,
    ConfirmedHeadRegressed { observed_number: u64 },
    ConfirmedBlockUnavailable,
}

#[derive(Clone, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub(crate) struct CompactGameSummary {
    pub(crate) factory_index: U256,
    pub(crate) address: Address,
    pub(crate) parent_index: u32,
    pub(crate) sequence_number: u64,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) struct InFlightCreationSummary {
    pub(crate) sequence_number: u64,
    pub(crate) parent_game_index: u32,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) struct CycleSnapshot {
    pub(crate) last_successful_pinned_l1: Option<crate::ports::L1BlockRef>,
    pub(crate) sync_disposition: SyncDisposition,
    pub(crate) anchor: Option<CompactGameSummary>,
    pub(crate) canonical_head_index: Option<U256>,
    pub(crate) canonical_head_sequence_number: Option<u64>,
    pub(crate) pending_games: Vec<CompactGameSummary>,
    pub(crate) in_flight_creation: Option<InFlightCreationSummary>,
    pub(crate) active_tasks: Vec<ActiveTaskSummary>,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) struct CycleResult {
    pub(crate) snapshot: CycleSnapshot,
    pub(crate) completions: Vec<TaskCompletion>,
    pub(crate) scheduled: Vec<ScheduledOperation>,
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
#[derive(Clone, Debug, PartialEq, Eq)]
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
    /// The game's creator address (`gameCreator()`), used to limit fast-finality
    /// spend to games created by this proposer's signer.
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
    /// Returns true when the proposer owns this game's defense, resolution, and
    /// claim lifecycle: its `absolutePrestate()` is in the known-prestates set
    /// (artifacts loadable from `KONA_SP1_PROPOSER_PRESTATES_URL`, proving keys not poisoned).
    ///
    /// Lifecycle ownership is prestate-based, not creator-based. Claims stay
    /// credit-driven, so iterating foreign games costs nothing where the proposer
    /// holds no credit. The fast-finality scan applies a separate creator filter
    /// before spending on an unchallenged game. Games created before a prestate
    /// rotation stay owned because sync loads the prestate of every cached game,
    /// not just the registered one.
    pub fn is_owned(&self, known_prestates: &HashSet<B256>) -> bool {
        known_prestates.contains(&self.absolute_prestate)
    }
}

impl From<&Game> for CompactGameSummary {
    fn from(game: &Game) -> Self {
        Self {
            factory_index: game.index,
            address: game.address,
            parent_index: game.parent_index,
            sequence_number: game.l2_sequence_number,
        }
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

    /// Marks a game and every cached descendant as terminally invalid, removes
    /// the subtree, and returns the removed game addresses.
    #[must_use]
    fn invalidate_subtree(&mut self, root_index: U256) -> Vec<Address> {
        let invalid_subtree = self.descendants_of(root_index);
        self.invalid_games.extend(invalid_subtree.iter().copied());
        invalid_subtree
            .into_iter()
            .filter_map(|index| {
                tracing::info!(?index, "Removing invalid game from cache");
                self.games.remove(&index).map(|game| game.address)
            })
            .collect()
    }

    /// Challenged, in-progress games as defense candidates, sorted by prove
    /// deadline ascending (closest to expiring first). Ownership and
    /// per-game dedup are applied by the defense scan.
    fn challenged_candidates(&self) -> Vec<(U256, Address, u64, B256, Address)> {
        let mut candidates = self
            .games
            .values()
            .filter(|game| game.status == GameStatus::InProgress)
            .filter(|game| matches!(game.proposal_status, ProposalStatus::Challenged))
            .map(|game| {
                (
                    game.index,
                    game.address,
                    game.deadline,
                    game.absolute_prestate,
                    game.anchor_state_registry,
                )
            })
            .collect::<Vec<_>>();
        candidates.sort_unstable_by_key(|(_, _, deadline, _, _)| *deadline);
        candidates
    }

    /// Unchallenged, in-progress games as fast-finality candidates, sorted
    /// by challenge deadline ascending (closest to expiring first). Creator,
    /// ownership, standing, and per-game dedup checks are applied by the scan.
    fn fast_finality_candidates(&self) -> Vec<(U256, Address, u64, B256, Address, Address)> {
        let mut candidates = self
            .games
            .values()
            .filter(|game| game.status == GameStatus::InProgress)
            .filter(|game| matches!(game.proposal_status, ProposalStatus::Unchallenged))
            .map(|game| {
                (
                    game.index,
                    game.address,
                    game.deadline,
                    game.absolute_prestate,
                    game.creator,
                    game.anchor_state_registry,
                )
            })
            .collect::<Vec<_>>();
        candidates.sort_unstable_by_key(|(_, _, deadline, _, _, _)| *deadline);
        candidates
    }

    /// Drops all cached state tied to the prior factory history and returns
    /// the removed game addresses.
    #[must_use]
    fn reset_factory_cache(&mut self) -> Vec<Address> {
        self.anchor_game = None;
        self.canonical_head_index = None;
        self.cursor = Cursor::none();
        let removed_addresses = self.games.drain().map(|(_, game)| game.address).collect();
        self.invalid_games.clear();
        removed_addresses
    }

    /// Selects the canonical head: the highest-L2-timestamp game on the best valid chain.
    ///
    /// With no anchor, the head is simply the highest game in the cache. With an anchor, the head
    /// is the highest descendant of the anchor, unless a higher chain branches off earlier
    /// (genesis-rooted, or a lower parent index than the anchor head); that alternative chain is
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
        // than stopping at the root; otherwise the head would pin to the root of a genesis-rooted
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

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum ClaimPreflightDecision {
    Submit,
    AlreadyClaimed,
    AwaitMaturity { withdrawal_timestamp: U256 },
}

fn classify_claim_preflight(preflight: &ClaimPreflight) -> ClaimPreflightDecision {
    let (Ok(credit), Ok(withdrawal)) = (&preflight.credit, &preflight.withdrawal) else {
        return ClaimPreflightDecision::Submit;
    };
    if *credit != U256::ZERO {
        return ClaimPreflightDecision::Submit;
    }
    if withdrawal.amount == U256::ZERO {
        return ClaimPreflightDecision::AlreadyClaimed;
    }
    ClaimPreflightDecision::AwaitMaturity { withdrawal_timestamp: withdrawal.timestamp }
}

struct GameDiscovery {
    anchor_deadline: Option<u64>,
    invalid_game_ids: Vec<U256>,
    newly_pending: Vec<CompactGameSummary>,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
struct GameSyncTarget {
    index: U256,
    address: Address,
    weth: Address,
    anchor_state_registry: Address,
    absolute_prestate: B256,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum GameSyncFacts {
    InProgress {
        index: U256,
        lifecycle: GameLifecycle,
        parent_resolved: bool,
        owned: bool,
    },
    DefenderWins {
        index: U256,
        game_address: Address,
        lifecycle: GameLifecycle,
        bond: BondState,
        canonical_head_index: Option<U256>,
    },
    ChallengerWins {
        index: U256,
    },
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum GameSyncRetention {
    CanonicalHead,
    Anchor,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum GameSyncAction {
    Update {
        index: U256,
        lifecycle: GameLifecycle,
        should_attempt_to_resolve: bool,
        should_attempt_to_claim_bond: bool,
        retention: Option<GameSyncRetention>,
    },
    Remove(U256),
    RemoveSubtree(U256),
}

fn classify_game_sync(
    facts: GameSyncFacts,
    now_ts: u64,
    anchor_address: Address,
) -> Result<GameSyncAction> {
    match facts {
        GameSyncFacts::InProgress { index, lifecycle, parent_resolved, owned } => {
            let is_game_over = match lifecycle.proposal_status {
                ProposalStatus::Unchallenged => now_ts > lifecycle.deadline,
                ProposalStatus::UnchallengedAndValidProofProvided |
                ProposalStatus::ChallengedAndValidProofProvided => true,
                ProposalStatus::Challenged | ProposalStatus::Resolved => false,
            };
            Ok(GameSyncAction::Update {
                index,
                lifecycle,
                should_attempt_to_resolve: parent_resolved && is_game_over && owned,
                should_attempt_to_claim_bond: false,
                retention: None,
            })
        }
        GameSyncFacts::DefenderWins {
            index,
            game_address,
            lifecycle,
            bond,
            canonical_head_index,
        } => {
            let withdrawal_ts = bond.withdrawal_timestamp.try_into().unwrap_or(u64::MAX);
            let weth_delay = bond.delay.try_into().context("DelayedWETH delay exceeds u64")?;
            let bond_action = bond_claim_action(
                lifecycle.is_finalized,
                bond.credit,
                bond.withdrawal_amount,
                withdrawal_ts,
                weth_delay,
                now_ts,
            );
            let (should_attempt_to_claim_bond, retention) = match bond_action {
                BondClaimAction::Unlock | BondClaimAction::Payout => (true, None),
                BondClaimAction::Wait => (false, None),
                BondClaimAction::Done if canonical_head_index == Some(index) => {
                    (false, Some(GameSyncRetention::CanonicalHead))
                }
                BondClaimAction::Done if anchor_address == game_address => {
                    (false, Some(GameSyncRetention::Anchor))
                }
                BondClaimAction::Done => return Ok(GameSyncAction::Remove(index)),
            };
            Ok(GameSyncAction::Update {
                index,
                lifecycle,
                should_attempt_to_resolve: false,
                should_attempt_to_claim_bond,
                retention,
            })
        }
        GameSyncFacts::ChallengerWins { index } => Ok(GameSyncAction::RemoveSubtree(index)),
    }
}
/// Core proposer service: syncs the on-chain game DAG, creates and defends games,
/// resolves finished ones, and claims bonds.
#[derive(Clone)]
pub struct Proposer {
    /// Proposer configuration loaded at startup.
    pub config: ProposerConfig,
    /// Cached proposer address used by policy and proof public inputs.
    proposer_address: Address,
    /// Read-only L1 observations consumed by proposer policy.
    l1_view: Arc<dyn L1View>,
    /// Host query time used for current super-root requests.
    query_time: Arc<dyn QueryTime>,
    /// Super-root observations used by proposal and proof policy.
    superroot_source: Arc<dyn SuperRootSource>,
    /// Witness collection and SP1 proof execution.
    proof_engine: Arc<dyn ProofEngine>,
    /// Serialized transaction submission and receipt decoding.
    action_executor: Arc<dyn ActionExecutor>,
    /// Prestate program cache: loaded ELFs keyed by `absolutePrestate()`
    /// hash, fetched from `KONA_SP1_PROPOSER_PRESTATES_URL` on demand (see [`PrestateCache`]).
    pub prestates: Arc<PrestateCache>,
    tasks: Arc<tokio::sync::Mutex<TaskMap>>,
    next_task_id: Arc<AtomicU64>,
    peak_concurrent_defense_tasks: Arc<AtomicU64>,
    state: Arc<RwLock<ProposerState>>,
    /// Proposer identity for foreign-game filtering and hardfork safety.
    pub identity: ProposerIdentity,
    /// Full semantic observation from the last successful pinned sync.
    last_successful_pinned_l1: Arc<RwLock<Option<crate::ports::L1BlockRef>>>,
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
    pending_games: Arc<RwLock<HashMap<U256, CompactGameSummary>>>,
    /// The registered game args' `maxProveDuration`, read once during
    /// `try_init`. Used for the deadline-approaching warning tier only;
    /// per-game deadlines come from `claimData` each sync.
    max_prove_duration: Arc<OnceCell<u64>>,
    /// The registered game args' `maxChallengeDuration`, read once during
    /// `try_init`. Used for the deadline-approaching warning tier of
    /// fast-finality proving only; per-game deadlines come from `claimData`
    /// each sync.
    max_challenge_duration: Arc<OnceCell<u64>>,
    /// Games found permanently unprovable (claim diverged from the
    /// supernode view, or required L1 beyond the game's L1 head). Skipped
    /// by the proving scans without re-fetching their spans. In-memory
    /// only: a restart re-evaluates (upstream-parity statelessness).
    undefendable: Arc<Mutex<HashSet<Address>>>,
}

impl std::fmt::Debug for Proposer {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Proposer")
            .field("config", &self.config)
            .field("identity", &self.identity)
            .finish_non_exhaustive()
    }
}

impl Proposer {
    /// Creates a proposer from resolved configuration and dependencies.
    pub async fn new(
        config: ProposerConfig,
        signer: SignerLock,
        factory: DisputeGameFactoryInstance<L1Provider>,
        proof_provider: ProofProvider,
    ) -> Result<Self> {
        let l1_provider = ProviderBuilder::default().connect_http(config.l1_rpc.clone());
        let prestates = Arc::new(PrestateCache::new(config.prestates_url.clone()));
        let host_inputs = Arc::new(HostInputs {
            l1_node_address: config.l1_rpc.to_string(),
            l1_beacon_address: config.l1_beacon_rpc.to_string(),
            l2_node_addresses: config.l2_rpcs.iter().map(Url::to_string).collect(),
            rollup_config_paths: config.rollup_config_paths.clone(),
            l1_config_path: config.l1_config_path.clone(),
            dependency_set_path: config.dependency_set_path.clone(),
        });
        let l1_view = Arc::new(ProductionL1View::new(
            l1_provider.clone(),
            factory.clone(),
            config.l1_rpc.clone(),
        ));
        let proposer_address = signer.address();
        let superroot_source =
            Arc::new(ProductionSuperRootSource::new(SuperrootClient::new(&config.superroot_rpcs)?));
        let proof_engine = Arc::new(ProductionProofEngine::new(
            proof_provider,
            host_inputs,
            config.range_split_count,
            config.max_concurrent_range_proofs,
        ));
        let action_executor = Arc::new(ProductionActionExecutor::new(
            signer,
            l1_provider,
            factory,
            config.l1_rpc.clone(),
            config.tx_confirmation_timeout,
            Self::fee_caps_for(&config),
        ));
        Self::new_with_dependencies(
            config,
            proposer_address,
            l1_view,
            Arc::new(SystemQueryTime),
            superroot_source,
            proof_engine,
            action_executor,
            prestates,
        )
        .await
    }

    /// Creates a proposer with crate-private observation dependencies.
    #[allow(clippy::too_many_arguments)]
    pub(crate) async fn new_with_dependencies(
        config: ProposerConfig,
        proposer_address: Address,
        l1_view: Arc<dyn L1View>,
        query_time: Arc<dyn QueryTime>,
        superroot_source: Arc<dyn SuperRootSource>,
        proof_engine: Arc<dyn ProofEngine>,
        action_executor: Arc<dyn ActionExecutor>,
        prestates: Arc<PrestateCache>,
    ) -> Result<Self> {
        let identity = ProposerIdentity::new();
        identity.log_startup_info(&config.prestates_url);

        Ok(Self {
            config,
            proposer_address,
            l1_view,
            query_time,
            superroot_source,
            proof_engine,
            action_executor,
            prestates,
            tasks: Arc::new(tokio::sync::Mutex::new(HashMap::new())),
            next_task_id: Arc::new(AtomicU64::new(1)),
            peak_concurrent_defense_tasks: Arc::new(AtomicU64::new(0)),
            state: Arc::new(RwLock::new(ProposerState::default())),
            identity,
            last_successful_pinned_l1: Arc::new(RwLock::new(None)),
            last_created_game_l2_sequence_number: Arc::new(AtomicU64::new(0)),
            last_created_game_address: Arc::new(tokio::sync::Mutex::new(Address::ZERO)),
            in_flight_creation: Arc::new(tokio::sync::Mutex::new(None)),
            pending_games: Arc::new(RwLock::new(HashMap::new())),
            max_prove_duration: Arc::new(OnceCell::new()),
            max_challenge_duration: Arc::new(OnceCell::new()),
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

            if let Err(e) = self.cycle().await {
                tracing::warn!("Failed to sync proposer state: {:?}", e);
                continue;
            }
            self.log_task_stats().await;
        }
    }

    /// Runs one no-sleep proposer cycle.
    pub(crate) async fn cycle(&self) -> Result<CycleResult> {
        let sync_disposition = self.sync_state().await?;
        let completions = self.reap_completed_tasks().await;
        let planned = self.determine_pending_operations().await;
        let snapshot = self.cycle_snapshot(sync_disposition).await;
        let scheduled = self.spawn_planned_operations(planned).await;
        Ok(CycleResult { snapshot, completions, scheduled })
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
        let game_args = self.l1_view.registered_game_args(BlockId::latest()).await?;
        if !self.prestates.ensure_loaded(game_args.absolute_prestate).await {
            // Not fatal: creation stays paused until the artifacts appear
            // under KONA_SP1_PROPOSER_PRESTATES_URL (PrestateCache::ensure_loaded logged why).
            tracing::warn!(
                registered = %game_args.absolute_prestate,
                "registered prestate programs unavailable; creation will pause"
            );
        }

        // Record the registered durations for the proving scans'
        // deadline-approaching warning tiers (per-game deadlines themselves
        // come from claimData each sync): maxProveDuration keys defense,
        // maxChallengeDuration keys fast finality.
        let _ = self.max_prove_duration.set(game_args.max_prove_duration);
        let _ = self.max_challenge_duration.set(game_args.max_challenge_duration);

        // Fetch and validate the anchor root from the currently registered registry.
        let anchor =
            self.l1_view.anchor_root(game_args.anchor_state_registry, BlockId::latest()).await?;
        anyhow::ensure!(
            anchor.root != B256::ZERO,
            "anchor state registry has no anchor root (game creation would revert)"
        );
        anchor.sequence_number.try_into().context("anchor sequence number exceeds u64")
    }

    /// Synchronizes the proposer's cached view of the dispute-game tree with the on-chain state.
    ///
    /// Steps run in order:
    /// 1. `sync_games` pulls newly created games and refreshes cached metadata.
    /// 2. `sync_anchor_game` aligns the cached anchor pointer with the registry contract.
    /// 3. `compute_canonical_head` recomputes the head game used for proposal selection.
    pub(crate) async fn sync_state(&self) -> Result<SyncDisposition> {
        // Pin one L1 block for the entire sync cycle so every state read sees a consistent
        // snapshot. Without this, load-balanced RPCs or a reorg can make related reads resolve
        // against different L1 states (e.g. credit vs anchorGame).
        // Ref: https://github.com/celo-org/op-succinct/issues/132
        let latest_block =
            self.l1_view.latest_head().await?.context("Failed to fetch latest L1 block")?;
        let confirmed_number =
            latest_block.number.saturating_sub(self.config.sync_l1_confirmations);

        // If L1 hasn't advanced past the last synced block, all on-chain state is identical.
        //
        // A lower confirmed number indicates backend regression from a load-balanced RPC, or a
        // deep L1 reorg past `sync_l1_confirmations`. This case is logged at WARN; equality stays
        // at DEBUG since it is the normal "L1 hasn't ticked" path.
        let previous_pin = *self.last_successful_pinned_l1.read().await;
        if let Some(previous_pin) = previous_pin.filter(|pin| confirmed_number <= pin.number) {
            if confirmed_number < previous_pin.number {
                tracing::warn!(
                    confirmed_number,
                    last_synced = previous_pin.number,
                    "L1 confirmed head moved backwards (backend regression or deep reorg), skipping sync"
                );
                return Ok(SyncDisposition::ConfirmedHeadRegressed {
                    observed_number: confirmed_number,
                });
            }
            tracing::debug!(
                confirmed_number,
                last_synced = previous_pin.number,
                "L1 head unchanged, skipping sync"
            );
            return Ok(SyncDisposition::UnchangedConfirmedHead);
        }

        // When no confirmation offset, use the latest block directly (single RPC response).
        // When offset > 0, fetch the confirmed block separately; if the backend hasn't
        // caught up, skip this cycle rather than pinning forward.
        let pinned_l1 = if self.config.sync_l1_confirmations == 0 {
            latest_block
        } else {
            match self.l1_view.block_ref(confirmed_number).await? {
                Some(block) => block,
                None => {
                    tracing::warn!(
                        confirmed_number,
                        "Confirmed block not available on this backend, skipping sync cycle"
                    );
                    return Ok(SyncDisposition::ConfirmedBlockUnavailable);
                }
            }
        };
        let pinned_block = BlockId::hash(pinned_l1.hash);
        let pinned_timestamp = pinned_l1.timestamp;

        // Pull new games and synchronize cached game statuses.
        self.sync_games(pinned_block, pinned_timestamp).await?;

        // Align anchor information after the cached game statuses have been synchronized.
        self.sync_anchor_game(pinned_block).await?;

        // With the cached game statuses and anchor synchronized, recompute the canonical head.
        self.compute_canonical_head().await;

        *self.last_successful_pinned_l1.write().await = Some(pinned_l1);

        Ok(SyncDisposition::Advanced)
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
        let pinned_latest_index = self.l1_view.latest_game_index(pinned_block).await?;
        ProposerGauge::FactoryLatestGameIndex
            .set(pinned_latest_index.map_or(-1.0, |i| i.to::<u64>() as f64));

        let Some(latest_index) = pinned_latest_index else {
            let removed_addresses = self.state.write().await.reset_factory_cache();
            for address in removed_addresses {
                self.proof_engine.clear(address);
            }
            self.pending_games.write().await.clear();
            ProposerGauge::SyncCursor.set(-1.0);
            return Ok(());
        };
        let latest_index = Cursor::from(latest_index);
        let anchor_address = self.l1_view.registered_anchor_game(pinned_block).await?;

        let discovery =
            self.discover_new_games(latest_index.clone(), anchor_address, pinned_block).await?;
        ProposerGauge::SyncCursor.set(latest_index.index().map_or(-1.0, |i| i.to::<u64>() as f64));

        self.prune_invalid_games(discovery.invalid_game_ids).await;
        self.revalidate_pending_games(
            &discovery.newly_pending,
            discovery.anchor_deadline,
            pinned_block,
        )
        .await;

        let (targets, known_prestates) = self.game_sync_targets().await;
        if targets.is_empty() {
            return Ok(());
        }

        let mut actions = Vec::with_capacity(targets.len());
        for target in targets {
            let index = target.index;
            let action = self
                .observe_game_sync(target, &known_prestates, pinned_block)
                .await
                .and_then(|facts| classify_game_sync(facts, pinned_timestamp, anchor_address));
            match action {
                Ok(action) => actions.push(action),
                Err(error) => {
                    tracing::warn!(
                        game_index = %index,
                        error = %error,
                        "Game status sync failed; skipping this game for this cycle"
                    );
                    ProposerGauge::GameSyncError.increment(1.0);
                }
            }
        }
        self.apply_game_sync_actions(actions).await;

        Ok(())
    }

    /// Walks new factory entries and advances the cursor only when no fetch fails.
    async fn discover_new_games(
        &self,
        latest_index: Cursor,
        anchor_address: Address,
        pinned_block: BlockId,
    ) -> Result<GameDiscovery> {
        let (cursor, removed_addresses) = {
            let mut state = self.state.write().await;
            let current_cursor = state.cursor.clone();
            if latest_index < current_cursor {
                tracing::warn!(
                    latest_index = %latest_index,
                    current_cursor = %current_cursor,
                    "Factory reset suspected; resetting cursor to 0"
                );
                (Cursor::none(), Some(state.reset_factory_cache()))
            } else {
                (current_cursor, None)
            }
        };
        if let Some(removed_addresses) = removed_addresses {
            for address in removed_addresses {
                self.proof_engine.clear(address);
            }
            self.pending_games.write().await.clear();
        }

        let mut index = latest_index.clone();
        let mut anchor_deadline = None;
        let mut invalid_game_ids = Vec::new();
        let mut newly_pending = Vec::new();
        while index != cursor {
            let i = index.index().expect("must have an index here");
            let fetch_result = match self.fetch_game(i, pinned_block).await {
                Ok(result) => result,
                Err(error) => {
                    tracing::warn!(
                        game_index = %index,
                        error = %error,
                        "Game fetch failed; aborting the sync cycle"
                    );
                    ProposerGauge::GameSyncError.increment(1.0);
                    return Err(error);
                }
            };

            match fetch_result {
                GameFetchResult::ValidGame { game_address, deadline } => {
                    if game_address == anchor_address {
                        anchor_deadline = Some(deadline);
                    }
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
                    if game_address == anchor_address {
                        break;
                    }
                }
                GameFetchResult::AlreadyExists => {}
                GameFetchResult::InvalidGame { index } => {
                    invalid_game_ids.push(index);
                }
                GameFetchResult::Pending {
                    index,
                    game_address,
                    parent_index,
                    sequence_number,
                    ..
                } => {
                    newly_pending.push(CompactGameSummary {
                        factory_index: index,
                        address: game_address,
                        parent_index,
                        sequence_number,
                    });
                }
            }

            index.step_back();
        }

        self.state.write().await.cursor = latest_index;
        Ok(GameDiscovery { anchor_deadline, invalid_game_ids, newly_pending })
    }

    async fn prune_invalid_games(&self, invalid_game_ids: Vec<U256>) {
        if invalid_game_ids.is_empty() {
            return;
        }
        let mut state = self.state.write().await;
        let mut removed_addresses = Vec::new();
        for index in invalid_game_ids {
            tracing::warn!(
                game_index = %index,
                "Removing invalid game and its subtree from cache"
            );
            removed_addresses.extend(state.invalidate_subtree(index));
        }
        drop(state);
        for address in removed_addresses {
            self.proof_engine.clear(address);
        }
    }

    /// Rechecks prior pending games; games first seen this cycle wait until the next sync.
    /// Loads each pending prestate before applying the eviction cutoff.
    async fn revalidate_pending_games(
        &self,
        newly_pending: &[CompactGameSummary],
        discovered_anchor_deadline: Option<u64>,
        pinned_block: BlockId,
    ) {
        let previously_pending = {
            let pending = self.pending_games.read().await;
            pending
                .keys()
                .copied()
                .filter(|index| !newly_pending.iter().any(|game| game.factory_index == *index))
                .collect::<Vec<_>>()
        };
        self.pending_games
            .write()
            .await
            .extend(newly_pending.iter().cloned().map(|game| (game.factory_index, game)));
        let anchor_deadline = match discovered_anchor_deadline {
            Some(deadline) => Some(deadline),
            None => self.state.read().await.anchor_game.as_ref().map(|game| game.deadline),
        };

        for index in previously_pending {
            match self.fetch_game(index, pinned_block).await {
                Ok(GameFetchResult::Pending {
                    index,
                    game_address,
                    parent_index,
                    sequence_number,
                    deadline,
                    prestate,
                }) => {
                    self.pending_games.write().await.insert(
                        index,
                        CompactGameSummary {
                            factory_index: index,
                            address: game_address,
                            parent_index,
                            sequence_number,
                        },
                    );
                    let _ = self.prestates.ensure_loaded(prestate).await;
                    let owned = self.prestates.known_prestates().await.contains(&prestate);
                    if owned {
                        tracing::debug!(
                            game_index = %index,
                            "Keeping pending owned game re-checkable (eviction exempt)"
                        );
                    } else if let Some(anchor_deadline) = anchor_deadline &&
                        pending_evictable(anchor_deadline, deadline)
                    {
                        tracing::warn!(
                            game_index = %index,
                            game_deadline = deadline,
                            anchor_deadline,
                            "Evicting pending game whose deadline fell behind the anchor beyond the lag cutoff"
                        );
                        self.pending_games.write().await.remove(&index);
                    }
                }
                Ok(GameFetchResult::InvalidGame { index }) => {
                    self.pending_games.write().await.remove(&index);
                    let removed_addresses = self.state.write().await.invalidate_subtree(index);
                    for address in removed_addresses {
                        self.proof_engine.clear(address);
                    }
                }
                Ok(_) => {
                    self.pending_games.write().await.remove(&index);
                }
                Err(error) => {
                    tracing::warn!(
                        game_index = %index,
                        error = %error,
                        "Pending game re-validation failed; retrying next cycle"
                    );
                    ProposerGauge::GameSyncError.increment(1.0);
                }
            }
        }
    }

    async fn game_sync_targets(&self) -> (Vec<GameSyncTarget>, HashSet<B256>) {
        let targets = {
            let state = self.state.read().await;
            state
                .games
                .values()
                .map(|game| GameSyncTarget {
                    index: game.index,
                    address: game.address,
                    weth: game.weth,
                    anchor_state_registry: game.anchor_state_registry,
                    absolute_prestate: game.absolute_prestate,
                })
                .collect::<Vec<_>>()
        };
        if targets.is_empty() {
            return (targets, HashSet::new());
        }

        let mut prestates =
            targets.iter().map(|target| target.absolute_prestate).collect::<Vec<_>>();
        prestates.sort_unstable();
        prestates.dedup();
        for prestate in prestates {
            let _ = self.prestates.ensure_loaded(prestate).await;
        }
        (targets, self.prestates.known_prestates().await)
    }

    /// Reads only the status-specific facts needed to classify one game.
    async fn observe_game_sync(
        &self,
        target: GameSyncTarget,
        known_prestates: &HashSet<B256>,
        pinned_block: BlockId,
    ) -> Result<GameSyncFacts> {
        let lifecycle = self
            .l1_view
            .game_lifecycle(target.address, target.anchor_state_registry, pinned_block)
            .await?;
        match lifecycle.status {
            GameStatus::InProgress => {
                let parent_resolved = if lifecycle.parent_index == u32::MAX {
                    true
                } else {
                    GameStatus::try_from(
                        self.l1_view
                            .parent_game_status(lifecycle.parent_index, pinned_block)
                            .await?,
                    )
                    .context("invalid parent game status")? ==
                        GameStatus::DefenderWins
                };
                Ok(GameSyncFacts::InProgress {
                    index: target.index,
                    lifecycle,
                    parent_resolved,
                    owned: known_prestates.contains(&target.absolute_prestate),
                })
            }
            GameStatus::DefenderWins => {
                let bond = self
                    .l1_view
                    .bond_state(target.address, target.weth, self.proposer_address, pinned_block)
                    .await?;
                let canonical_head_index = self.state.read().await.canonical_head_index;
                Ok(GameSyncFacts::DefenderWins {
                    index: target.index,
                    game_address: target.address,
                    lifecycle,
                    bond,
                    canonical_head_index,
                })
            }
            GameStatus::ChallengerWins => Ok(GameSyncFacts::ChallengerWins { index: target.index }),
        }
    }

    /// Applies the completed action batch under one state write lock.
    async fn apply_game_sync_actions(&self, actions: Vec<GameSyncAction>) {
        let mut state = self.state.write().await;
        let mut progress_addresses_to_clear = Vec::new();
        for action in actions {
            match action {
                GameSyncAction::Update {
                    index,
                    lifecycle,
                    should_attempt_to_resolve,
                    should_attempt_to_claim_bond,
                    retention,
                } => {
                    match retention {
                        Some(GameSyncRetention::CanonicalHead) => {
                            tracing::debug!(game_index = %index, "Retaining game: canonical head");
                        }
                        Some(GameSyncRetention::Anchor) => {
                            tracing::debug!(game_index = %index, "Retaining game: anchor game");
                        }
                        None => {}
                    }
                    let terminal = lifecycle.status != GameStatus::InProgress ||
                        !awaiting_proof(lifecycle.proposal_status);
                    if let Some(game) = state.games.get_mut(&index) {
                        game.status = lifecycle.status;
                        game.proposal_status = lifecycle.proposal_status;
                        game.deadline = lifecycle.deadline;
                        game.should_attempt_to_resolve = should_attempt_to_resolve;
                        game.should_attempt_to_claim_bond = should_attempt_to_claim_bond;
                        if terminal {
                            progress_addresses_to_clear.push(game.address);
                        }
                    }
                }
                GameSyncAction::Remove(index) => {
                    if let Some(game) = state.games.remove(&index) {
                        progress_addresses_to_clear.push(game.address);
                    }
                    tracing::debug!(game_index = %index, "Removed game from cache");
                }
                GameSyncAction::RemoveSubtree(index) => {
                    let subtree = state.descendants_of(index);
                    let guarded_addr = *self.last_created_game_address.lock().await;
                    if guarded_addr != Address::ZERO {
                        let guard_in_subtree = subtree.iter().any(|idx| {
                            state.games.get(idx).is_some_and(|game| game.address == guarded_addr)
                        });
                        if guard_in_subtree {
                            self.last_created_game_l2_sequence_number.store(0, Ordering::Relaxed);
                            *self.last_created_game_address.lock().await = Address::ZERO;
                            tracing::info!(
                                ?guarded_addr,
                                root_index = %index,
                                "Reset creation guard: tracked game removed by ChallengerWins"
                            );
                        }
                    }
                    progress_addresses_to_clear.extend(state.invalidate_subtree(index));
                }
            }
        }
        drop(state);
        for address in progress_addresses_to_clear {
            self.proof_engine.clear(address);
        }
    }

    /// Synchronizes the anchor game from the registry.
    async fn sync_anchor_game(&self, pinned_block: BlockId) -> Result<()> {
        let anchor_address = self.l1_view.registered_anchor_game(pinned_block).await?;

        let mut state = self.state.write().await;

        if anchor_address == Address::ZERO {
            state.anchor_game = None;
        } else if let Some((_, anchor_game)) =
            state.games.iter().find(|(_, game)| game.address == anchor_address)
        {
            state.anchor_game = Some(anchor_game.clone());
            tracing::debug!(?anchor_address, "Anchor game updated in cache");
        } else {
            // Anchor not in cache (pruned or not yet fetched); clear to prevent
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
            // Preserve canonical_head_sequence_number as the anchor baseline for
            // the first proposal. Clearing it would block creation on fresh
            // deployments or pinned snapshots without games.
            // canonical_head_index = -1 reports the no-head state.
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
        let args = self.l1_view.registered_game_args(BlockId::latest()).await?;
        Ok(self.prestate_usable_for_creation(args.absolute_prestate).await)
    }

    /// Returns whether game creation may proceed for `prestate`. Artifacts
    /// must load, and network mode also requires key setup and vkey
    /// verification before a game is bonded on the prestate.
    ///
    /// Key setup takes tens of seconds per ELF, so this scheduler path starts
    /// it in a background task and keeps creation paused until completion. A
    /// failed setup poisons the entry until corrected artifacts are published.
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
    const fn fee_caps_for(config: &ProposerConfig) -> FeeCaps {
        FeeCaps {
            max_fee_per_gas: config.max_fee_per_gas,
            max_priority_fee_per_gas: config.max_priority_fee_per_gas,
        }
    }

    /// Creates a new game with the given parameters.
    ///
    /// `root_claim`: the super-root hash we are proposing.
    /// `extra_data`: `parentIndex (4B BE) || superRootProof`.
    pub async fn create_game(&self, root_claim: B256, extra_data: Vec<u8>) -> Result<Address> {
        // Read at creation time rather than startup: the factory's init bond
        // can change, and a stale value would revert every create.
        let init_bond = self.l1_view.init_bond().await?;
        let receipt = self.action_executor.create_game(root_claim, extra_data, init_bond).await?;

        tracing::info!(
            game_address = ?receipt.game_address,
            tx_hash = ?receipt.transaction_hash,
            "Game created successfully"
        );

        Ok(receipt.game_address)
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
            match self.l1_view.game_status(game.address).await {
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
                    Err(error) => {
                        tracing::warn!(
                            game_address = ?game.address,
                            ?error,
                            "Invalid game status on chain, proceeding with resolve"
                        );
                    }
                    _ => {}
                },
                Err(error) => {
                    tracing::warn!(
                        game_address = ?game.address,
                        ?error,
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

        let proposer_address = self.proposer_address;
        for game in candidates {
            // Pre-flight on-chain claim-state check at `latest`. The cached
            // `should_attempt_to_claim_bond` is derived from the pinned (lagged)
            // snapshot; re-check so we neither re-submit a finished claim nor
            // submit a phase-2 payout before the WETH delay matures.
            let preflight =
                self.l1_view.claim_preflight(game.address, game.weth, proposer_address).await;
            let is_payout = match classify_claim_preflight(&preflight) {
                ClaimPreflightDecision::Submit => false,
                ClaimPreflightDecision::AlreadyClaimed => {
                    tracing::info!(
                        game_index = %game.index,
                        game_address = ?game.address,
                        "Skipping claim: bond already claimed on chain"
                    );
                    continue;
                }
                ClaimPreflightDecision::AwaitMaturity { withdrawal_timestamp } => {
                    // Phase 2: only submit once the WETH delay has elapsed in
                    // CHAIN time. DelayedWETH enforces
                    // `timestamp + DELAY_SECONDS <= block.timestamp`; wall
                    // clock and L1 time diverge under devstack time travel
                    // (and can drift in production).
                    let weth_delay: u64 = self
                        .l1_view
                        .weth_delay(game.weth)
                        .await?
                        .try_into()
                        .context("DelayedWETH delay exceeds u64")?;
                    let l1_now = self
                        .l1_view
                        .latest_l1_timestamp()
                        .await
                        .context("failed to fetch latest L1 block for claim maturity")?;
                    let matured = withdrawal_timestamp
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
                    true
                }
            };

            let transaction_hash = match self
                .action_executor
                .claim_credit(game.address, proposer_address)
                .instrument(tracing::info_span!("[[Claiming Proposer Bonds]]"))
                .await
            {
                Ok(transaction_hash) => transaction_hash,
                Err(error) => {
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
            };

            tracing::info!(
                game_index = %game.index,
                game_address = ?game.address,
                l2_sequence_end = %game.l2_sequence_number,
                tx_hash = ?transaction_hash,
                "Bond claimed successfully"
            );

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
        let transaction_hash = self.action_executor.resolve_game(game.address).await?;

        tracing::info!(
            game_index = %game.index,
            game_address = ?game.address,
            l2_sequence_end = %game.l2_sequence_number,
            tx_hash = ?transaction_hash,
            "Game resolved successfully"
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

        let factory_game = self.l1_view.factory_game(index, pinned_block).await?;
        let game_address = factory_game.address;
        let game_type = factory_game.game_type;

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

        let claim_data = self.l1_view.game_claim(game_address, pinned_block).await?;
        let parent_index = claim_data.parent_index;

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
        let identity = self.l1_view.game_identity(game_address, pinned_block).await?;
        let game_asr = identity.anchor_state_registry;
        let game_weth = identity.weth;
        let creator = identity.creator;

        let sequence_number = identity.sequence_number;
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
        let validity = self.l1_view.game_validity(game_address, pinned_block).await?;
        let claim = validity.root_claim;
        let was_respected = validity.was_respected;
        let status = validity.status;
        let absolute_prestate = validity.absolute_prestate;

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
        let super_root_at =
            match self.superroot_source.super_root_at_timestamp(sequence_number).await {
                Ok(super_root_at) => super_root_at,
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
                        game_address,
                        parent_index,
                        sequence_number,
                        deadline,
                        prestate: absolute_prestate,
                    });
                }
            };
        match super_root_at.root {
            None => {
                // Not yet safe from this node's view. Far-future timestamps beyond
                // the validation horizon are terminal (bonded spam); anything
                // nearer is pending and re-validated next sync.
                let local_safe = super_root_at.response.current_local_safe_timestamp;
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
                    game_address,
                    parent_index,
                    sequence_number,
                    deadline,
                    prestate: absolute_prestate,
                });
            }
            Some(super_root) if super_root.super_root != claim => {
                if !response_trusted(&super_root_at.response) {
                    tracing::warn!(
                        game_index = %index,
                        ?game_address,
                        ?claim,
                        canonical_super_root = ?super_root.super_root,
                        "Root mismatch from untrusted supernode response; deferring validation"
                    );
                    return Ok(GameFetchResult::Pending {
                        index,
                        game_address,
                        parent_index,
                        sequence_number,
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

        if game.creator != self.proposer_address {
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
            let super_root_at =
                self.superroot_source.super_root_at_timestamp(sequence_number).await?;
            let Some(super_root) = super_root_at.root else {
                // Transient: the chosen timestamp is not yet safe from this
                // node's view. Bail and retry on a later tick.
                bail!("no canonical super root at timestamp {sequence_number} yet");
            };
            let extra_data = zk_extra_data(parent_game_index, &super_root.proof_bytes);
            let existing_game =
                self.l1_view.game_by_uuid(super_root.super_root, extra_data.clone()).await?;

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

                        // Record the sequence number and address so creation planning skips a
                        // duplicate while the pinned cache has not caught up to this game.
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
            let existing_creator = self.l1_view.game_creator(existing_game).await?;
            if existing_creator == self.proposer_address {
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
    /// clears and normal proposals resume on the next tick.
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

        let nonces = self.l1_view.nonce_state(self.proposer_address).await?;
        if nonces.pending > nonces.latest {
            tracing::info!(
                sequence_number = record.sequence_number,
                parent_game_index = record.parent_game_index,
                pending_nonce = nonces.pending,
                latest_nonce = nonces.latest,
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
        let existing_game =
            self.l1_view.game_by_uuid(record.root_claim, record.extra_data.clone()).await?;
        if existing_game == Address::ZERO {
            return Ok(false);
        }

        let existing_creator = self.l1_view.game_creator(existing_game).await?;
        if existing_creator == self.proposer_address {
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

    async fn reap_completed_tasks(&self) -> Vec<TaskCompletion> {
        let mut completed = {
            let tasks = self.tasks.lock().await;
            tasks
                .iter()
                .filter_map(|(id, (handle, _))| handle.is_finished().then_some(*id))
                .collect::<Vec<_>>()
        };
        completed.sort_unstable();
        self.finalize_tasks(&completed, None).await.expect("production reaping has no watchdog")
    }

    async fn finalize_tasks(
        &self,
        task_ids: &[TaskId],
        watchdog: Option<Duration>,
    ) -> std::result::Result<Vec<TaskCompletion>, Vec<TaskCompletion>> {
        let mut selected = {
            let mut tasks = self.tasks.lock().await;
            task_ids
                .iter()
                .filter_map(|task_id| {
                    tasks.remove(task_id).map(|(handle, info)| (*task_id, handle, info))
                })
                .collect::<Vec<_>>()
        };
        selected.sort_unstable_by_key(|(task_id, _, _)| *task_id);

        let mut completions = Vec::with_capacity(selected.len());
        let watchdog_deadline = watchdog.map(|duration| time::Instant::now() + duration);
        let mut selected = selected.into_iter();
        while let Some((task_id, mut handle, operation)) = selected.next() {
            let joined = if let Some(deadline) = watchdog_deadline {
                match time::timeout_at(deadline, &mut handle).await {
                    Ok(joined) => joined,
                    Err(_) => {
                        let mut tasks = self.tasks.lock().await;
                        tasks.insert(task_id, (handle, operation));
                        tasks.extend(
                            selected.map(|(id, handle, operation)| (id, (handle, operation))),
                        );
                        return Err(completions);
                    }
                }
            } else {
                handle.await
            };
            let outcome = match joined {
                Ok(Ok(TaskSuccess::Completed)) => {
                    tracing::info!("Task {:?} completed successfully", operation);
                    TaskCompletionOutcome::Success
                }
                Ok(Ok(TaskSuccess::TerminallyUnprovable)) => {
                    tracing::info!("Task {:?} completed as terminally unprovable", operation);
                    TaskCompletionOutcome::TerminallyUnprovable
                }
                Ok(Err(error)) => {
                    tracing::warn!("Task {:?} failed: {:?}", operation, error);
                    let error = error.to_string();
                    self.handle_task_failure(&operation);
                    TaskCompletionOutcome::Failed(TaskFailureClass::ReturnedError(error))
                }
                Err(panic) => {
                    tracing::error!("Task {:?} panicked: {:?}", operation, panic);
                    TaskCompletionOutcome::Failed(TaskFailureClass::Panicked)
                }
            };
            completions.push(TaskCompletion {
                task_id,
                class: operation.class(),
                target: operation.target(),
                outcome,
            });
        }
        Ok(completions)
    }

    fn handle_task_failure(&self, operation: &OperationSummary) {
        match operation.class() {
            TaskClass::Creation => {
                ProposerGauge::GameCreationError.increment(1.0);
            }
            TaskClass::Resolution => {
                ProposerGauge::GameResolutionError.increment(1.0);
            }
            TaskClass::Proving => {
                ProposerGauge::GameProvingError.increment(1.0);
            }
            TaskClass::Claim => {
                ProposerGauge::BondClaimingError.increment(1.0);
            }
        }
    }

    async fn determine_pending_operations(&self) -> Vec<OperationSummary> {
        let mut deduplicated = {
            let tasks = self.tasks.lock().await;
            tasks
                .values()
                .map(|(_, operation)| operation.deduplication_key())
                .collect::<HashSet<_>>()
        };
        let mut planned = Vec::new();

        if deduplicated.contains(&TaskDeduplicationKey::Creation) {
            tracing::info!("Game creation task already active");
        } else {
            match self.plan_game_creation(&mut planned, &mut deduplicated).await {
                Ok(true) => tracing::info!("Successfully planned game creation task"),
                Ok(false) => {
                    tracing::debug!("No game creation needed - proposal interval not elapsed")
                }
                Err(e) => tracing::warn!("Failed to plan game creation task: {:?}", e),
            }
        }

        match self.plan_game_defense_tasks(&mut planned, &mut deduplicated).await {
            Ok(true) => tracing::info!("Successfully planned game defense tasks"),
            Ok(false) => tracing::debug!("No games need defense or defense is at capacity"),
            Err(e) => tracing::warn!("Failed to plan game defense tasks: {:?}", e),
        }

        if deduplicated.insert(TaskDeduplicationKey::Resolution) {
            planned.push(OperationSummary::ResolutionSweep);
            tracing::info!("Successfully planned game resolution task");
        }

        if deduplicated.insert(TaskDeduplicationKey::Claim) {
            planned.push(OperationSummary::ClaimSweep);
            tracing::info!("Successfully planned bond claim task");
        } else {
            tracing::info!("Bond claim task already active");
        }

        planned
    }

    async fn cycle_snapshot(&self, sync_disposition: SyncDisposition) -> CycleSnapshot {
        let (anchor, canonical_head_index, canonical_head_sequence_number) = {
            let state = self.state.read().await;
            (
                state.anchor_game.as_ref().map(CompactGameSummary::from),
                state.canonical_head_index,
                state.canonical_head_sequence_number,
            )
        };
        let mut pending_games =
            self.pending_games.read().await.values().cloned().collect::<Vec<_>>();
        pending_games.sort_unstable();
        let in_flight_creation =
            self.in_flight_creation.lock().await.as_ref().map(|creation| InFlightCreationSummary {
                sequence_number: creation.sequence_number,
                parent_game_index: creation.parent_game_index,
            });
        let mut active_tasks = {
            let tasks = self.tasks.lock().await;
            tasks
                .iter()
                .map(|(task_id, (_, operation))| ActiveTaskSummary {
                    task_id: *task_id,
                    class: operation.class(),
                    target: operation.target(),
                    operation: operation.clone(),
                })
                .collect::<Vec<_>>()
        };
        active_tasks.sort_unstable_by(|left, right| {
            (&left.class, &left.target, left.task_id).cmp(&(
                &right.class,
                &right.target,
                right.task_id,
            ))
        });

        CycleSnapshot {
            last_successful_pinned_l1: *self.last_successful_pinned_l1.read().await,
            sync_disposition,
            anchor,
            canonical_head_index,
            canonical_head_sequence_number,
            pending_games,
            in_flight_creation,
            active_tasks,
        }
    }

    /// Log current task statistics
    async fn log_task_stats(&self) {
        let tasks = self.tasks.lock().await;
        let active_count = tasks.len();
        if active_count > 0 {
            let mut task_counts: HashMap<&str, usize> = HashMap::new();

            for (_, operation) in tasks.values() {
                let task_type = match operation.class() {
                    TaskClass::Creation => "GameCreation",
                    TaskClass::Resolution => "GameResolution",
                    TaskClass::Proving => "GameProving",
                    TaskClass::Claim => "BondClaim",
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

    async fn plan_game_creation(
        &self,
        planned: &mut Vec<OperationSummary>,
        deduplicated: &mut HashSet<TaskDeduplicationKey>,
    ) -> Result<bool> {
        // An unresolved create takes precedence over new proposals: hold
        // them until its uuid is adopted or provably dead, so a
        // stuck-then-included original can never be joined by a sibling at
        // a fresh timestamp.
        let in_flight_sequence_number = self
            .in_flight_creation
            .lock()
            .await
            .as_ref()
            .map(|record| (record.sequence_number, record.parent_game_index));
        if let Some((sequence_number, parent_game_index)) = in_flight_sequence_number {
            planned
                .push(OperationSummary::ReconcileCreation { sequence_number, parent_game_index });
            deduplicated.insert(TaskDeduplicationKey::Creation);
            return Ok(true);
        }

        let (should_create, next_sequence_number, parent_game_index) =
            self.plan_game_creation_decision(planned, deduplicated).await?;
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
            // Standing must be current, not pinned: retirement/blacklisting
            // is retroactive and the child binds the registry at create time.
            let registry =
                self.l1_view.registered_game_args(BlockId::latest()).await?.anchor_state_registry;
            let standing = self.l1_view.parent_standing(parent_address, registry).await?;
            if standing.disallowed() {
                tracing::warn!(
                    parent_index = parent_game_index,
                    ?parent_address,
                    blacklisted = standing.blacklisted,
                    retired = standing.retired,
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
                let removed_addresses = state.invalidate_subtree(root_index);
                drop(state);
                for address in removed_addresses {
                    self.proof_engine.clear(address);
                }
                return Ok(false);
            }
        }

        planned.push(OperationSummary::ProposeGame {
            sequence_number: next_sequence_number,
            parent_game_index,
        });
        deduplicated.insert(TaskDeduplicationKey::Creation);
        Ok(true)
    }

    /// Check if we should create a game.
    ///
    /// In fast finality mode this first selects proving for owned
    /// unchallenged games without one (up to `KONA_SP1_PROPOSER_FAST_FINALITY_PROVING_LIMIT`,
    /// counting ALL in-flight proving tasks), then skips creation while at
    /// that capacity: never create games faster than they can be proven.
    ///
    /// Compares the next proposal sequence number against the highest
    /// timestamp proposable under the configured safety level.
    ///
    /// Returns whether a game should be created, the sequence number to
    /// propose at, and the parent game index (dummy values when false).
    async fn plan_game_creation_decision(
        &self,
        planned: &mut Vec<OperationSummary>,
        deduplicated: &mut HashSet<TaskDeduplicationKey>,
    ) -> Result<(bool, u64, u32)> {
        if self.config.fast_finality_mode {
            let mut active_proving = deduplicated
                .iter()
                .filter(|key| matches!(key, TaskDeduplicationKey::Proving(_)))
                .count() as u64;
            if active_proving < self.config.fast_finality_proving_limit.get() {
                let known_prestates = self.prestates.known_prestates().await;
                let candidates = self.state.read().await.fast_finality_candidates();
                for (index, game_address, deadline, prestate, creator, registry_address) in
                    candidates
                {
                    if active_proving >= self.config.fast_finality_proving_limit.get() {
                        tracing::debug!(
                            active = active_proving,
                            "Reached fast finality proving capacity while scanning"
                        );
                        break;
                    }
                    if creator != self.proposer_address {
                        tracing::debug!(
                            game_address = ?game_address,
                            creator = ?creator,
                            "Skipping fast finality: game created by another proposer"
                        );
                        continue;
                    }
                    if deduplicated.contains(&TaskDeduplicationKey::Proving(game_address)) {
                        continue;
                    }
                    if !known_prestates.contains(&prestate) {
                        // Not ours to prove (prestate-based ownership, #22111);
                        // an unchallenged foreign game is unremarkable.
                        tracing::debug!(
                            game_address = ?game_address,
                            "Skipping fast finality: unknown prestate"
                        );
                        continue;
                    }
                    if self.should_skip_proving(game_address, deadline, false).await? {
                        continue;
                    }
                    if self
                        .l1_view
                        .game_standing(game_address, registry_address)
                        .await?
                        .disallowed()
                    {
                        tracing::warn!(
                            game_index = %index,
                            ?game_address,
                            "Skipping fast finality: game is blacklisted or retired"
                        );
                        self.proof_engine.clear(game_address);
                        continue;
                    }
                    planned.push(OperationSummary::ProveGame {
                        factory_index: index,
                        address: game_address,
                        purpose: ProvingPurpose::FastFinality,
                    });
                    deduplicated.insert(TaskDeduplicationKey::Proving(game_address));
                    tracing::info!(
                        game_address = ?game_address,
                        game_index = %index,
                        "Planned fast finality proving"
                    );
                    active_proving += 1;
                }
            }
            if active_proving >= self.config.fast_finality_proving_limit.get() {
                tracing::info!(
                    active = active_proving,
                    limit = %self.config.fast_finality_proving_limit,
                    "Skipping game creation: at fast finality proving capacity"
                );
                return Ok((false, 0, u32::MAX));
            }
        }

        // Check if our game type matches the current respected game type.
        // The proposer should only create games when its type is the respected type.
        let respected_game_type = self.l1_view.respected_game_type(BlockId::latest()).await?;
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
        let now = self.query_time.unix_timestamp()?;
        let horizon = self.superroot_source.proposal_horizon(now).await?;
        Ok(match self.config.proposal_safety {
            ProposalSafety::Safe => horizon.safe_timestamp,
            ProposalSafety::Finalized => horizon.finalized_timestamp,
        })
    }

    async fn spawn_planned_operations(
        &self,
        planned: Vec<OperationSummary>,
    ) -> Vec<ScheduledOperation> {
        let mut scheduled = Vec::with_capacity(planned.len());
        let planned_defense_tasks = planned
            .iter()
            .filter(|operation| {
                matches!(
                    operation,
                    OperationSummary::ProveGame { purpose: ProvingPurpose::Defense, .. }
                )
            })
            .count() as u64;
        if planned_defense_tasks > 0 {
            let active_defense_tasks = self
                .tasks
                .lock()
                .await
                .values()
                .filter(|(handle, operation)| {
                    !handle.is_finished() &&
                        matches!(
                            operation,
                            OperationSummary::ProveGame { purpose: ProvingPurpose::Defense, .. }
                        )
                })
                .count() as u64;
            let admitted_defense_tasks = active_defense_tasks + planned_defense_tasks;
            let previous_peak = self
                .peak_concurrent_defense_tasks
                .fetch_max(admitted_defense_tasks, Ordering::Relaxed);
            if admitted_defense_tasks > previous_peak {
                ProposerGauge::PeakConcurrentDefenseTasks.set(admitted_defense_tasks as f64);
            }
        }

        for operation in planned {
            let task_id = TaskId::allocate(&self.next_task_id);
            let proposer = self.clone();
            let operation_for_task = operation.clone();
            let handle = tokio::spawn(async move {
                match operation_for_task {
                    OperationSummary::ProposeGame { sequence_number, parent_game_index } => {
                        proposer.handle_game_creation(sequence_number, parent_game_index).await?;
                        Ok(TaskSuccess::Completed)
                    }
                    OperationSummary::ReconcileCreation { .. } => {
                        proposer.resolve_in_flight_creation().await?;
                        Ok(TaskSuccess::Completed)
                    }
                    OperationSummary::ProveGame { factory_index, address, .. } => {
                        let result = proposer.prove_game(address).await;
                        proposer.handle_game_proving_result(factory_index, address, result).await
                    }
                    OperationSummary::ResolutionSweep => {
                        proposer.resolve_games().await?;
                        Ok(TaskSuccess::Completed)
                    }
                    OperationSummary::ClaimSweep => {
                        proposer.claim_bonds().await?;
                        Ok(TaskSuccess::Completed)
                    }
                }
            });
            self.tasks.lock().await.insert(task_id, (handle, operation.clone()));
            if let OperationSummary::ProveGame { purpose, .. } = &operation {
                match purpose {
                    ProvingPurpose::Defense => ProposerGauge::GamesDefenseSpawned.increment(1.0),
                    ProvingPurpose::FastFinality => {
                        ProposerGauge::GamesFastFinalitySpawned.increment(1.0);
                    }
                }
            }
            tracing::info!(task_id = task_id.get(), ?operation, "Spawned proposer task");
            scheduled.push(ScheduledOperation { task_id, operation });
        }
        scheduled.sort_unstable_by(|left, right| {
            left.operation.cmp(&right.operation).then_with(|| left.task_id.cmp(&right.task_id))
        });
        scheduled
    }

    /// Selects owned challenged games by ascending deadline, up to the configured limit.
    #[tracing::instrument(name = "[[Defending]]", skip(self))]
    async fn plan_game_defense_tasks(
        &self,
        planned: &mut Vec<OperationSummary>,
        deduplicated: &mut HashSet<TaskDeduplicationKey>,
    ) -> Result<bool> {
        let known_prestates = self.prestates.known_prestates().await;
        let candidates = self.state.read().await.challenged_candidates();

        let mut active_defense_tasks = {
            let tasks = self.tasks.lock().await;
            tasks
                .values()
                .filter(|(_, operation)| {
                    matches!(
                        operation,
                        OperationSummary::ProveGame { purpose: ProvingPurpose::Defense, .. }
                    )
                })
                .count() as u64
        } + planned
            .iter()
            .filter(|operation| {
                matches!(
                    operation,
                    OperationSummary::ProveGame { purpose: ProvingPurpose::Defense, .. }
                )
            })
            .count() as u64;
        let max_concurrent = self.config.max_concurrent_defense_tasks.get();
        let mut tasks_planned = false;

        for (index, game_address, deadline, prestate, registry_address) in candidates {
            if active_defense_tasks >= max_concurrent {
                tracing::debug!(
                    "The max concurrent defense tasks count ({}) has been reached",
                    max_concurrent
                );
                break;
            }

            if deduplicated.contains(&TaskDeduplicationKey::Proving(game_address)) {
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

            if self.should_skip_proving(game_address, deadline, true).await? {
                continue;
            }

            if self.l1_view.game_standing(game_address, registry_address).await?.disallowed() {
                tracing::warn!(
                    game_index = %index,
                    ?game_address,
                    "Skipping defense: game is blacklisted or retired"
                );
                self.proof_engine.clear(game_address);
                continue;
            }

            planned.push(OperationSummary::ProveGame {
                factory_index: index,
                address: game_address,
                purpose: ProvingPurpose::Defense,
            });
            deduplicated.insert(TaskDeduplicationKey::Proving(game_address));
            tracing::info!(
                game_address = ?game_address,
                game_index = %index,
                "Planned defense for challenged game"
            );
            active_defense_tasks += 1;
            tasks_planned = true;
        }

        Ok(tasks_planned)
    }

    /// Check if proving should be skipped for any reason:
    /// - The game was found permanently unprovable earlier.
    /// - It is already proven or resolved on chain (pre-flight at `latest`; the cached status is
    ///   read at the pinned, lagged block, so a recently confirmed `prove()` or `resolve()` may not
    ///   be reflected yet - this avoids an expensive proof regeneration that could only revert on
    ///   submission).
    /// - Its deadline has passed (with a warning tier when it is approaching). For defense the
    ///   deadline is the prove window and the warning tier is keyed to `maxProveDuration`; for fast
    ///   finality it is the challenge window, keyed to `maxChallengeDuration`.
    async fn should_skip_proving(
        &self,
        game_address: Address,
        deadline: u64,
        is_defense: bool,
    ) -> Result<bool> {
        if self.undefendable.lock().await.contains(&game_address) {
            tracing::debug!(?game_address, "Skipping proving: game is permanently unprovable");
            return Ok(true);
        }

        match self.l1_view.proof_status(game_address).await {
            Ok(status) => match ProposalStatus::try_from(status) {
                Ok(
                    ProposalStatus::UnchallengedAndValidProofProvided |
                    ProposalStatus::ChallengedAndValidProofProvided |
                    ProposalStatus::Resolved,
                ) => {
                    tracing::info!(
                        ?game_address,
                        "Skipping proving: game already proven or resolved on chain"
                    );
                    self.proof_engine.clear(game_address);
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

        let now = self
            .l1_view
            .latest_l1_timestamp()
            .await
            .context("failed to fetch latest L1 block for game deadline")?;
        let max_duration = if is_defense {
            *self.max_prove_duration.get().context("max_prove_duration must be set via try_init")?
        } else {
            *self
                .max_challenge_duration
                .get()
                .context("max_challenge_duration must be set via try_init")?
        };
        match check_deadline_status(now, deadline, max_duration) {
            DeadlineStatus::Passed => {
                tracing::error!(
                    game_address = ?game_address,
                    deadline,
                    now,
                    "Game proving deadline passed, cannot prove"
                );
                self.proof_engine.clear(game_address);
                return Ok(true);
            }
            DeadlineStatus::Approaching { hours_remaining } => {
                tracing::warn!(
                    game_address = ?game_address,
                    "Game proving deadline approaching, {:.1} hours remaining",
                    hours_remaining
                );
                ProposerGauge::DeadlineApproaching.increment(1.0);
            }
            DeadlineStatus::Ok => {}
        }
        Ok(false)
    }

    async fn handle_game_proving_result(
        &self,
        game_index: U256,
        game_address: Address,
        result: Result<()>,
    ) -> Result<TaskSuccess> {
        match result {
            Ok(()) => Ok(TaskSuccess::Completed),
            Err(err) if is_unprovable(&err) => {
                tracing::error!(
                    ?game_address,
                    error = %err,
                    "Game is permanently unprovable; giving up on proving it"
                );
                ProposerGauge::GameUnprovable.increment(1.0);
                let mut state = self.state.write().await;
                let progress_addresses_to_clear = if state
                    .games
                    .get(&game_index)
                    .is_some_and(|game| game.address == game_address)
                {
                    let guarded_addr = *self.last_created_game_address.lock().await;
                    if guarded_addr != Address::ZERO &&
                        state.descendants_of(game_index).iter().any(|index| {
                            state.games.get(index).is_some_and(|game| game.address == guarded_addr)
                        })
                    {
                        self.last_created_game_l2_sequence_number.store(0, Ordering::Relaxed);
                        *self.last_created_game_address.lock().await = Address::ZERO;
                        tracing::info!(
                            ?guarded_addr,
                            root_index = %game_index,
                            "Reset creation guard: tracked game removed with unprovable subtree"
                        );
                    }
                    state.invalidate_subtree(game_index)
                } else {
                    vec![game_address]
                };
                drop(state);
                for address in progress_addresses_to_clear {
                    self.proof_engine.clear(address);
                }
                self.undefendable.lock().await.insert(game_address);
                Ok(TaskSuccess::TerminallyUnprovable)
            }
            Err(err) => {
                let can_retry =
                    self.state.read().await.games.get(&game_index).is_some_and(|game| {
                        game.address == game_address &&
                            game.status == GameStatus::InProgress &&
                            awaiting_proof(game.proposal_status)
                    });
                if !can_retry {
                    self.proof_engine.clear(game_address);
                }
                Err(err)
            }
        }
    }

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
            tracing::info!(?game_address, "Game no longer tracked; abandoning its proving");
            return Ok(());
        };

        let proof_inputs = self.l1_view.proof_inputs(game_address).await?;
        let keys = match self.config.proof_provider {
            ProofProviderKind::Network => {
                Some(self.prestates.proof_keys(prestate, self.config.proof_provider).await?)
            }
            ProofProviderKind::Mock => None,
        };

        let inputs = self.game_proof_inputs(proof_inputs, prestate);
        let responses = fetch_span_responses(self.superroot_source.as_ref(), &inputs).await?;
        let proof_bytes = self.proof_engine.prove(game_address, keys, inputs, responses).await?;

        // Pre-submit re-check: proving can take long; avoid a guaranteed
        // revert when the game was proven by someone else, resolved, hit
        // its deadline, or was evicted (parent lost) meanwhile.
        if !self.pre_submit_checks(game_address).await? {
            self.proof_engine.clear(game_address);
            return Ok(());
        }

        let transaction_hash = self.action_executor.prove_game(game_address, proof_bytes).await?;
        self.proof_engine.clear(game_address);

        ProposerGauge::GamesProven.increment(1.0);
        ProposerGauge::ProvingDurationSeconds.set(start_time.elapsed().as_secs_f64());
        tracing::info!(
            game_address = ?game_address,
            tx_hash = ?transaction_hash,
            duration_s = start_time.elapsed().as_secs_f64(),
            "Game proven successfully"
        );
        Ok(())
    }

    const fn game_proof_inputs(
        &self,
        proof_inputs: ProofInputs,
        prestate: B256,
    ) -> GameProofInputs {
        GameProofInputs {
            l1_head: proof_inputs.l1_head,
            l1_head_number: proof_inputs.l1_head_number,
            starting_root: proof_inputs.starting_root,
            starting_ts: proof_inputs.starting_sequence_number,
            root_claim: proof_inputs.root_claim,
            claim_ts: proof_inputs.sequence_number,
            prestate,
            prover: self.proposer_address,
        }
    }

    /// Final checks before submitting a `prove()` transaction. Returns false
    /// when submission should be skipped (all three legs log why):
    /// 1. the game is still tracked (subtree removal on a lost parent evicts descendants, and
    ///    `prove()` reverts `InvalidParentGame`; a residual on-chain-but-unsynced parent loss still
    ///    reverts harmlessly and is caught by the tx status check);
    /// 2. `claimData` at `latest` still awaits a proof (`Unchallenged` or `Challenged`; a challenge
    ///    landing mid-proof does not invalidate the proof - the public values bind the signer, not
    ///    the status - and the reverse reorg is equally fine);
    /// 3. the game's deadline has not passed (the challenge deadline while `Unchallenged`, the
    ///    prove deadline once challenged; `claimData` at `latest` reflects the rewrite).
    async fn pre_submit_checks(&self, game_address: Address) -> Result<bool> {
        let tracked = {
            let state = self.state.read().await;
            state.games.values().any(|game| game.address == game_address)
        };
        if !tracked {
            tracing::info!(?game_address, "Skipping prove(): game evicted mid-proving");
            return Ok(false);
        }

        let claim = self.l1_view.game_claim(game_address, BlockId::latest()).await?;
        let status = ProposalStatus::try_from(claim.status)?;
        if !awaiting_proof(status) {
            tracing::info!(
                ?game_address,
                ?status,
                "Skipping prove(): game no longer awaiting a proof"
            );
            return Ok(false);
        }

        let registry = self.l1_view.anchor_state_registry(game_address).await?;
        if self.l1_view.game_standing(game_address, registry).await?.disallowed() {
            tracing::warn!(
                ?game_address,
                "Skipping prove(): game became blacklisted or retired while proving"
            );
            return Ok(false);
        }

        let now = self
            .l1_view
            .latest_l1_timestamp()
            .await
            .context("failed to fetch latest L1 block for game deadline")?;
        if now > claim.deadline {
            tracing::warn!(
                ?game_address,
                deadline = claim.deadline,
                now,
                "Skipping prove(): game deadline passed mid-proving"
            );
            return Ok(false);
        }

        Ok(true)
    }
}

/// Warn when less than `max_duration / DEADLINE_WARNING_DIVISOR` remains
/// before a game's proving deadline (the prove window for defense, the
/// challenge window for fast finality).
pub const DEADLINE_WARNING_DIVISOR: u64 = 2;

/// Status of a game's proving deadline.
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

/// Classifies a prove deadline; equality has not passed.
pub fn check_deadline_status(now: u64, deadline: u64, max_duration: u64) -> DeadlineStatus {
    if now > deadline {
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

/// A game still awaiting a proof: unchallenged (fast finality) or
/// challenged (defense). Proof-provided and `Resolved` games are past
/// proving.
pub(crate) const fn awaiting_proof(status: ProposalStatus) -> bool {
    matches!(status, ProposalStatus::Unchallenged | ProposalStatus::Challenged)
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
    /// The game's timestamp is not yet safe from this node's view, so it
    /// cannot be validated. It remains outside the DAG and is re-validated
    /// each sync until data appears or the horizon expires.
    Pending {
        /// Factory index of the pending game.
        index: U256,
        /// Address of the pending game.
        game_address: Address,
        /// Factory index of the parent game.
        parent_index: u32,
        /// Super-root timestamp of the pending game.
        sequence_number: u64,
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

/// Returns whether a pending game's deadline is more than the maximum lag
/// behind the anchor deadline. This is one-sided: a game ahead of a stalled
/// anchor may still have an open challenge window and remains re-checkable.
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
/// attempt. Bounds `KONA_SP1_PROPOSER_PRESTATES_URL` traffic and log noise for genuinely
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
/// fetched from the base `KONA_SP1_PROPOSER_PRESTATES_URL` on demand, plus per-prestate SP1
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

    /// Creates an empty cache with the given negative-cache retry window.
    pub fn with_retry_window(url: Url, unknown_retry: Duration) -> Self {
        Self {
            programs: RwLock::new(HashMap::new()),
            misses: RwLock::new(HashMap::new()),
            url,
            unknown_retry,
        }
    }

    /// Loads program ELFs unless a recent miss is cached. An unpoisoned
    /// loaded entry returns true even when network proving keys are not
    /// initialized; `prestate_usable_for_creation` applies that separate
    /// creation gate.
    ///
    /// Poisoned entries are re-fetched on the negative-cache cadence and
    /// replaced only when published bytes change. Load failures and unchanged
    /// poisoned artifacts follow [`UNKNOWN_PRESTATE_POLICY`].
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
                        let prestates_url = crate::env_var("PRESTATES_URL");
                        tracing::warn!(
                            prestate = %prestate,
                            "Prestate stays poisoned: published artifacts are unchanged \
                             (publish corrected artifacts under {prestates_url} to heal)"
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
                let prestates_url = crate::env_var("PRESTATES_URL");
                tracing::warn!(
                    prestate = %prestate,
                    error = %e,
                    retry_seconds = self.unknown_retry.as_secs(),
                    "Failed to load prestate programs \
                     (publish the artifacts under {prestates_url} if this is a hardfork)"
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

    /// Returns the proving keys for `prestate`, running SP1 key setup on first
    /// use in network mode.
    ///
    /// Setup takes tens of seconds per ELF. The entry is cloned before setup
    /// so no map guard blocks cache readers or writers. The aggregation
    /// verifying key must hash to `prestate`; otherwise the entry is poisoned
    /// and excluded from [`Self::known_prestates`].
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
    use std::{
        collections::HashSet,
        sync::{
            Arc, Mutex as StdMutex,
            atomic::{AtomicU64, Ordering as AtomicOrdering},
        },
    };

    use alloy_eips::BlockId;
    use alloy_primitives::{Address, B256, U256};
    use alloy_provider::{ProviderBuilder, mock::Asserter};
    use alloy_rpc_types_eth::Block;
    use alloy_signer_local::PrivateKeySigner;
    use async_trait::async_trait;
    use kona_sp1_host_utils::metrics::MetricsListen;
    use kona_sp1_super_range_executor::{
        BlockId as SuperBlockId, SuperRootAtTimestampResponse, SuperRootResponseData, SuperV1,
    };

    use super::{
        ClaimPreflightDecision, CompactGameSummary, Cursor, DEADLINE_WARNING_DIVISOR,
        DeadlineStatus, Game, GameFetchResult, GameSyncAction, GameSyncFacts, GameSyncRetention,
        MAX_GAME_DEADLINE_LAG, OperationSummary, PrestateCache, Proposer, ProposerState,
        ProvingPurpose, SyncDisposition, TaskDeduplicationKey, TaskId, TaskSuccess, awaiting_proof,
        check_deadline_status, classify_claim_preflight, classify_game_sync,
        next_proposal_timestamp,
    };
    use crate::{
        ZK_GAME_TYPE,
        adapters::ProductionL1View,
        config::{
            PrestatePrograms, ProofProviderConfig, ProofProviderKind, ProposalSafety,
            ProposerConfig, RangeSplitCount,
        },
        contract::{DisputeGameFactory, GameStatus, ProposalStatus, ZKGameArgs},
        ports::{
            ActionExecutor, AnchorRoot, BondState, ClaimPreflight, FactoryGame, GameClaim,
            GameCreationReceipt, GameIdentity, GameLifecycle, GameStanding, GameValidity,
            L1BlockRef, L1View, NonceState, ProofEngine, ProofInputs, ProposalHorizon, QueryTime,
            SuperRootAtTimestamp, SuperRootSource, WithdrawalState,
        },
        prover::{MockProofProvider, ProofKeys, ProofProvider},
        proving::GameProofInputs,
        signer::{Signer, SignerLock},
    };

    struct TestQueryTime(u64);

    impl QueryTime for TestQueryTime {
        fn unix_timestamp(&self) -> anyhow::Result<u64> {
            Ok(self.0)
        }
    }

    struct ErrorQueryTime;

    impl QueryTime for ErrorQueryTime {
        fn unix_timestamp(&self) -> anyhow::Result<u64> {
            anyhow::bail!("query time unavailable")
        }
    }

    struct UnavailableSuperRootSource;

    #[async_trait]
    impl SuperRootSource for UnavailableSuperRootSource {
        async fn proposal_horizon(&self, _timestamp: u64) -> anyhow::Result<ProposalHorizon> {
            anyhow::bail!("super-root unavailable")
        }

        async fn super_root_at_timestamp(
            &self,
            _timestamp: u64,
        ) -> anyhow::Result<SuperRootAtTimestamp> {
            anyhow::bail!("super-root unavailable")
        }
    }

    struct ScriptedSuperRootSource {
        horizon: ProposalHorizon,
        roots: Vec<(u64, SuperRootAtTimestamp)>,
    }

    #[async_trait]
    impl SuperRootSource for ScriptedSuperRootSource {
        async fn proposal_horizon(&self, _timestamp: u64) -> anyhow::Result<ProposalHorizon> {
            Ok(self.horizon)
        }

        async fn super_root_at_timestamp(
            &self,
            timestamp: u64,
        ) -> anyhow::Result<SuperRootAtTimestamp> {
            self.roots
                .iter()
                .find(|(observed_timestamp, _)| *observed_timestamp == timestamp)
                .map(|(_, super_root_at)| super_root_at.clone())
                .ok_or_else(|| anyhow::anyhow!("missing super root at timestamp {timestamp}"))
        }
    }

    #[derive(Default)]
    struct RecordingProofEngine {
        calls: StdMutex<Vec<(GameProofInputs, Vec<SuperRootAtTimestampResponse>)>>,
        proof: Vec<u8>,
        fail: bool,
        cleared: StdMutex<Vec<Address>>,
        cached: StdMutex<Option<Vec<u8>>>,
        generations: std::sync::atomic::AtomicUsize,
    }

    #[async_trait]
    impl ProofEngine for RecordingProofEngine {
        async fn prove(
            &self,
            _game_address: Address,
            _keys: Option<Arc<ProofKeys>>,
            game: GameProofInputs,
            responses: Vec<SuperRootAtTimestampResponse>,
        ) -> anyhow::Result<Vec<u8>> {
            self.calls.lock().unwrap().push((game, responses));
            if self.fail {
                anyhow::bail!("proof execution failed")
            }
            let mut cached = self.cached.lock().unwrap();
            if let Some(proof) = cached.as_ref() {
                return Ok(proof.clone());
            }
            self.generations.fetch_add(1, AtomicOrdering::SeqCst);
            *cached = Some(self.proof.clone());
            Ok(self.proof.clone())
        }

        fn clear(&self, game_address: Address) {
            *self.cached.lock().unwrap() = None;
            self.cleared.lock().unwrap().push(game_address);
        }
    }

    #[derive(Clone, Debug, PartialEq, Eq)]
    enum ActionCall {
        Create { root_claim: B256, extra_data: Vec<u8>, init_bond: U256 },
        Prove { game: Address, proof: Vec<u8> },
        Resolve(Address),
        ClaimCredit { game: Address, recipient: Address },
    }

    #[derive(Clone, Copy)]
    enum CreateFailure {
        Reverted,
        Uncertain,
    }

    #[derive(Default)]
    struct RecordingActionExecutor {
        calls: StdMutex<Vec<ActionCall>>,
        create_failure: Option<CreateFailure>,
        prove_failures: StdMutex<usize>,
    }

    #[async_trait]
    impl ActionExecutor for RecordingActionExecutor {
        async fn create_game(
            &self,
            root_claim: B256,
            extra_data: Vec<u8>,
            init_bond: U256,
        ) -> anyhow::Result<GameCreationReceipt> {
            match self.create_failure {
                Some(CreateFailure::Reverted) => {
                    anyhow::bail!("{} receipt", crate::TX_REVERTED_PREFIX)
                }
                Some(CreateFailure::Uncertain) => {
                    anyhow::bail!("transaction submission uncertain")
                }
                None => {}
            }
            self.calls.lock().unwrap().push(ActionCall::Create {
                root_claim,
                extra_data,
                init_bond,
            });
            Ok(GameCreationReceipt {
                game_address: Address::left_padding_from(&[0xc1]),
                transaction_hash: B256::left_padding_from(&[0xc2]),
            })
        }

        async fn prove_game(&self, game: Address, proof: Vec<u8>) -> anyhow::Result<B256> {
            self.calls.lock().unwrap().push(ActionCall::Prove { game, proof });
            let mut failures = self.prove_failures.lock().unwrap();
            if *failures > 0 {
                *failures -= 1;
                anyhow::bail!("prove transaction failed");
            }
            Ok(B256::left_padding_from(&[0xc3]))
        }

        async fn resolve_game(&self, game: Address) -> anyhow::Result<B256> {
            self.calls.lock().unwrap().push(ActionCall::Resolve(game));
            Ok(B256::left_padding_from(&[0xd1]))
        }

        async fn claim_credit(&self, game: Address, recipient: Address) -> anyhow::Result<B256> {
            self.calls.lock().unwrap().push(ActionCall::ClaimCredit { game, recipient });
            Ok(B256::left_padding_from(&[0xf1]))
        }
    }

    struct RecordingL1View {
        calls: StdMutex<Vec<&'static str>>,
        block_calls: StdMutex<Vec<(&'static str, BlockId)>>,
        anchor_targets: StdMutex<Vec<(Address, BlockId)>>,
        bond_targets: StdMutex<Vec<(Address, Address, Address, BlockId)>>,
        lifecycle_targets: StdMutex<Vec<(Address, Address, BlockId)>>,
        fail_on: Option<&'static str>,
        latest_head: Option<L1BlockRef>,
        block_ref: Option<L1BlockRef>,
        latest_game_index: Option<U256>,
        anchor_game: Address,
        registered_args: ZKGameArgs,
        anchor_root: AnchorRoot,
        factory_game: FactoryGame,
        game_claim: GameClaim,
        game_identity: GameIdentity,
        game_validity: GameValidity,
        lifecycle: GameLifecycle,
        parent_status: u8,
        bond_state: BondState,
        claim_preflight: Option<(U256, WithdrawalState)>,
        init_bond: U256,
        game_by_uuid: Address,
        proof_inputs: ProofInputs,
        proof_standing: GameStanding,
        proof_registry: Address,
        latest_l1_timestamp: u64,
    }

    impl Default for RecordingL1View {
        fn default() -> Self {
            Self {
                calls: StdMutex::new(Vec::new()),
                block_calls: StdMutex::new(Vec::new()),
                anchor_targets: StdMutex::new(Vec::new()),
                bond_targets: StdMutex::new(Vec::new()),
                lifecycle_targets: StdMutex::new(Vec::new()),
                fail_on: None,
                latest_head: Some(L1BlockRef { hash: B256::ZERO, number: 1, timestamp: 1_000 }),
                block_ref: None,
                latest_game_index: None,
                anchor_game: Address::ZERO,
                registered_args: ZKGameArgs {
                    absolute_prestate: B256::ZERO,
                    verifier: Address::ZERO,
                    max_challenge_duration: 1,
                    max_prove_duration: 1,
                    challenger_bond: U256::ZERO,
                    anchor_state_registry: Address::ZERO,
                    weth: Address::ZERO,
                },
                anchor_root: AnchorRoot {
                    root: B256::left_padding_from(&[1]),
                    sequence_number: U256::ZERO,
                },
                factory_game: FactoryGame { address: Address::ZERO, game_type: ZK_GAME_TYPE },
                game_claim: GameClaim {
                    status: ProposalStatus::Unchallenged as u8,
                    deadline: 2_000,
                    parent_index: u32::MAX,
                },
                game_identity: GameIdentity {
                    anchor_state_registry: Address::ZERO,
                    weth: Address::ZERO,
                    creator: Address::ZERO,
                    sequence_number: U256::ZERO,
                },
                game_validity: GameValidity {
                    root_claim: B256::ZERO,
                    was_respected: true,
                    status: GameStatus::InProgress,
                    absolute_prestate: B256::ZERO,
                },
                lifecycle: GameLifecycle {
                    proposal_status: ProposalStatus::Unchallenged,
                    deadline: 2_000,
                    parent_index: u32::MAX,
                    status: GameStatus::InProgress,
                    is_finalized: false,
                },
                parent_status: GameStatus::DefenderWins as u8,
                bond_state: BondState {
                    credit: U256::from(1),
                    withdrawal_amount: U256::ZERO,
                    withdrawal_timestamp: U256::ZERO,
                    delay: U256::ZERO,
                },
                claim_preflight: None,
                init_bond: U256::ZERO,
                game_by_uuid: Address::ZERO,
                proof_inputs: ProofInputs::default(),
                proof_standing: GameStanding { blacklisted: false, retired: false },
                proof_registry: Address::ZERO,
                latest_l1_timestamp: 1_000,
            }
        }
    }

    impl RecordingL1View {
        fn record(&self, call: &'static str) {
            self.calls.lock().unwrap().push(call);
        }

        fn calls(&self) -> Vec<&'static str> {
            self.calls.lock().unwrap().clone()
        }

        fn block_calls(&self) -> Vec<(&'static str, BlockId)> {
            self.block_calls.lock().unwrap().clone()
        }

        fn record_block(&self, call: &'static str, block: BlockId) {
            self.block_calls.lock().unwrap().push((call, block));
        }

        fn fail_if_configured(&self, call: &'static str) -> anyhow::Result<()> {
            if self.fail_on == Some(call) {
                anyhow::bail!("{call} unavailable")
            }
            Ok(())
        }
    }

    #[async_trait]
    impl L1View for RecordingL1View {
        async fn latest_head(&self) -> anyhow::Result<Option<L1BlockRef>> {
            self.record("latest_head");
            self.fail_if_configured("latest_head")?;
            Ok(self.latest_head)
        }

        async fn block_ref(&self, _number: u64) -> anyhow::Result<Option<L1BlockRef>> {
            self.record("block_ref");
            Ok(self.block_ref)
        }

        async fn registered_game_args(&self, block: BlockId) -> anyhow::Result<ZKGameArgs> {
            self.record("registered_game_args");
            self.record_block("registered_game_args", block);
            Ok(self.registered_args.clone())
        }

        async fn anchor_root(
            &self,
            registry: Address,
            block: BlockId,
        ) -> anyhow::Result<AnchorRoot> {
            self.record("anchor_root");
            self.record_block("anchor_root", block);
            self.anchor_targets.lock().unwrap().push((registry, block));
            Ok(self.anchor_root)
        }

        async fn latest_game_index(&self, block: BlockId) -> anyhow::Result<Option<U256>> {
            self.record("latest_game_index");
            self.record_block("latest_game_index", block);
            self.fail_if_configured("latest_game_index")?;
            Ok(self.latest_game_index)
        }

        async fn registered_anchor_game(&self, block: BlockId) -> anyhow::Result<Address> {
            self.record("registered_anchor_game");
            self.record_block("registered_anchor_game", block);
            self.fail_if_configured("registered_anchor_game")?;
            Ok(self.anchor_game)
        }

        async fn factory_game(&self, _index: U256, block: BlockId) -> anyhow::Result<FactoryGame> {
            self.record("factory_game");
            self.record_block("factory_game", block);
            self.fail_if_configured("factory_game")?;
            Ok(self.factory_game)
        }

        async fn game_claim(&self, _game: Address, block: BlockId) -> anyhow::Result<GameClaim> {
            self.record("game_claim");
            self.record_block("game_claim", block);
            self.fail_if_configured("game_claim")?;
            Ok(self.game_claim)
        }

        async fn game_identity(
            &self,
            _game: Address,
            block: BlockId,
        ) -> anyhow::Result<GameIdentity> {
            self.record("game_identity");
            self.record_block("game_identity", block);
            Ok(self.game_identity)
        }

        async fn game_validity(
            &self,
            _game: Address,
            block: BlockId,
        ) -> anyhow::Result<GameValidity> {
            self.record("game_validity");
            self.record_block("game_validity", block);
            Ok(self.game_validity)
        }

        async fn game_lifecycle(
            &self,
            game: Address,
            registry: Address,
            block: BlockId,
        ) -> anyhow::Result<GameLifecycle> {
            self.record("game_lifecycle");
            self.record_block("game_lifecycle", block);
            self.lifecycle_targets.lock().unwrap().push((game, registry, block));
            self.fail_if_configured("game_lifecycle")?;
            Ok(self.lifecycle)
        }

        async fn parent_game_status(
            &self,
            _parent_index: u32,
            block: BlockId,
        ) -> anyhow::Result<u8> {
            self.record("parent_game_status");
            self.record_block("parent_game_status", block);
            Ok(self.parent_status)
        }

        async fn bond_state(
            &self,
            game: Address,
            weth: Address,
            proposer: Address,
            block: BlockId,
        ) -> anyhow::Result<BondState> {
            self.record("bond_state");
            self.record_block("bond_state", block);
            self.bond_targets.lock().unwrap().push((game, weth, proposer, block));
            Ok(self.bond_state)
        }

        async fn init_bond(&self) -> anyhow::Result<U256> {
            self.record("init_bond");
            Ok(self.init_bond)
        }

        async fn game_status(&self, _game: Address) -> anyhow::Result<u8> {
            panic!("unexpected L1 call: game_status")
        }

        async fn claim_preflight(
            &self,
            _game: Address,
            _weth: Address,
            _proposer: Address,
        ) -> ClaimPreflight {
            self.record("claim_preflight");
            let (credit, withdrawal) =
                self.claim_preflight.expect("unexpected L1 call: claim_preflight");
            ClaimPreflight { credit: Ok(credit), withdrawal: Ok(withdrawal) }
        }

        async fn weth_delay(&self, _weth: Address) -> anyhow::Result<U256> {
            panic!("unexpected L1 call: weth_delay")
        }

        async fn game_by_uuid(
            &self,
            _root_claim: B256,
            _extra_data: Vec<u8>,
        ) -> anyhow::Result<Address> {
            self.record("game_by_uuid");
            Ok(self.game_by_uuid)
        }

        async fn game_creator(&self, _game: Address) -> anyhow::Result<Address> {
            panic!("unexpected L1 call: game_creator")
        }

        async fn nonce_state(&self, _proposer: Address) -> anyhow::Result<NonceState> {
            panic!("unexpected L1 call: nonce_state")
        }

        async fn respected_game_type(&self, _block: BlockId) -> anyhow::Result<u32> {
            panic!("unexpected L1 call: respected_game_type")
        }

        async fn parent_standing(
            &self,
            _game: Address,
            _registry: Address,
        ) -> anyhow::Result<GameStanding> {
            panic!("unexpected L1 call: parent_standing")
        }

        async fn game_standing(
            &self,
            _game: Address,
            _registry: Address,
        ) -> anyhow::Result<GameStanding> {
            self.record("game_standing");
            Ok(GameStanding {
                blacklisted: self.proof_standing.blacklisted,
                retired: self.proof_standing.retired,
            })
        }

        async fn proof_status(&self, _game: Address) -> anyhow::Result<u8> {
            panic!("unexpected L1 call: proof_status")
        }

        async fn proof_inputs(&self, _game: Address) -> anyhow::Result<ProofInputs> {
            self.record("proof_inputs");
            Ok(self.proof_inputs)
        }

        async fn anchor_state_registry(&self, _game: Address) -> anyhow::Result<Address> {
            self.record("anchor_state_registry");
            Ok(self.proof_registry)
        }

        async fn latest_l1_timestamp(&self) -> anyhow::Result<u64> {
            self.record("latest_l1_timestamp");
            Ok(self.latest_l1_timestamp)
        }
    }

    fn super_root_at_timestamp(
        timestamp: u64,
        root: B256,
        current_l1: u64,
        required_l1: u64,
    ) -> SuperRootAtTimestamp {
        SuperRootAtTimestamp {
            response: SuperRootAtTimestampResponse {
                current_l1: SuperBlockId { number: current_l1, ..Default::default() },
                current_safe_timestamp: timestamp,
                current_local_safe_timestamp: timestamp,
                current_finalized_timestamp: timestamp,
                optimistic_at_timestamp: Default::default(),
                chain_ids: Vec::new(),
                data: Some(SuperRootResponseData {
                    verified_required_l1: SuperBlockId {
                        number: required_l1,
                        ..Default::default()
                    },
                    super_v1: SuperV1 { timestamp, chains: Vec::new() },
                    super_root: root,
                }),
            },
            root: Some(crate::superroot::SuperRootAt {
                proof_bytes: vec![timestamp as u8],
                super_root: root,
            }),
        }
    }

    fn absent_super_root_at_timestamp(local_safe: u64) -> SuperRootAtTimestamp {
        SuperRootAtTimestamp {
            response: SuperRootAtTimestampResponse {
                current_l1: SuperBlockId::default(),
                current_safe_timestamp: local_safe,
                current_local_safe_timestamp: local_safe,
                current_finalized_timestamp: local_safe,
                optimistic_at_timestamp: Default::default(),
                chain_ids: Vec::new(),
                data: None,
            },
            root: None,
        }
    }

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

    fn write_test_prestate_artifacts(name: &str, prestate: B256) -> std::path::PathBuf {
        let dir = std::env::temp_dir()
            .join(format!("kona-sp1-proposer-sync-test-{}-{name}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        for suffix in [".agg.bin.gz", ".range.bin.gz"] {
            let mut gz = flate2::write::GzEncoder::new(Vec::new(), flate2::Compression::default());
            std::io::Write::write_all(&mut gz, b"elf").unwrap();
            std::fs::write(dir.join(format!("{prestate}{suffix}")), gz.finish().unwrap()).unwrap();
        }
        dir
    }

    fn test_config() -> ProposerConfig {
        ProposerConfig {
            l1_rpc: "http://127.0.0.1:1".parse().unwrap(),
            superroot_rpcs: vec!["http://127.0.0.1:1".parse().unwrap()],
            factory_address: Address::ZERO,
            prestates_url: "file:///nonexistent".parse().unwrap(),
            proposal_interval_seconds: 3600,
            proposal_safety: ProposalSafety::Finalized,
            fetch_interval: 30,
            metrics_listen: MetricsListen::Disabled,
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
            fast_finality_mode: false,
            fast_finality_proving_limit: std::num::NonZeroU64::MIN,
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
                max_price_per_pgu: Some(std::num::NonZeroU64::MIN),
                min_auction_period: 1,
            },
        }
    }

    /// A proposer whose RPC endpoints are unreachable: everything that
    /// needs no chain read is exercisable (task accounting, undefendable
    /// set, deadline arms after a failed pre-flight read).
    async fn test_proposer() -> Proposer {
        test_proposer_with(test_config()).await
    }

    /// The same stub proposer with a caller-supplied config.
    async fn test_proposer_with(config: ProposerConfig) -> Proposer {
        let provider =
            ProviderBuilder::default().connect_http("http://127.0.0.1:1".parse().unwrap());
        test_proposer_with_config_and_provider(config, provider).await
    }

    async fn test_proposer_with_provider(provider: crate::L1Provider) -> Proposer {
        test_proposer_with_config_and_provider(test_config(), provider).await
    }

    async fn test_proposer_with_config_and_provider(
        config: ProposerConfig,
        provider: crate::L1Provider,
    ) -> Proposer {
        let signer = SignerLock::new(Signer::LocalSigner(PrivateKeySigner::random()));
        let factory = DisputeGameFactory::new(Address::ZERO, provider.clone());
        let l1_view =
            ProductionL1View::new(provider.clone(), factory.clone(), config.l1_rpc.clone());
        let mut proposer =
            Proposer::new(config, signer, factory, ProofProvider::Mock(MockProofProvider))
                .await
                .unwrap();
        proposer.l1_view = Arc::new(l1_view);
        proposer.query_time = Arc::new(TestQueryTime(1_000));
        proposer
    }

    fn push_claim_error_and_default_head(asserter: &Asserter) {
        push_claim_error_and_head(asserter, 1_000);
    }

    fn push_claim_error_and_head(asserter: &Asserter, timestamp: u64) {
        asserter.push_failure_msg("claimData unavailable");
        let mut block = Block::<alloy_rpc_types_eth::Transaction>::default();
        block.header.timestamp = timestamp;
        asserter.push_success(&block);
    }

    async fn insert_task(proposer: &Proposer, operation: OperationSummary) {
        let task_id = TaskId::allocate(&proposer.next_task_id);
        let handle = tokio::spawn(async { Ok(TaskSuccess::Completed) });
        proposer.tasks.lock().await.insert(task_id, (handle, operation));
    }

    #[test]
    fn task_id_allocation_rejects_exhaustion_without_wrapping() {
        let next_task_id = AtomicU64::new(u64::MAX);

        assert!(std::panic::catch_unwind(|| TaskId::allocate(&next_task_id)).is_err());
        assert_eq!(next_task_id.load(AtomicOrdering::Relaxed), u64::MAX);
    }

    async fn active_task_keys(proposer: &Proposer) -> HashSet<TaskDeduplicationKey> {
        proposer
            .tasks
            .lock()
            .await
            .values()
            .map(|(_, operation)| operation.deduplication_key())
            .collect()
    }

    #[tokio::test]
    async fn sync_state_pins_reads_to_selected_l1_hash() {
        let latest_hash = B256::repeat_byte(0x11);
        let confirmed_hash = B256::repeat_byte(0x22);
        let cases = [
            (0, L1BlockRef { hash: latest_hash, number: 5, timestamp: 1_000 }, None, latest_hash),
            (
                2,
                L1BlockRef { hash: latest_hash, number: 7, timestamp: 1_000 },
                Some(L1BlockRef { hash: confirmed_hash, number: 5, timestamp: 998 }),
                confirmed_hash,
            ),
        ];

        for (confirmations, latest_head, block_ref, expected_hash) in cases {
            let mut config = test_config();
            config.sync_l1_confirmations = confirmations;
            let mut proposer = test_proposer_with(config).await;
            let view = Arc::new(RecordingL1View {
                fail_on: Some("latest_game_index"),
                latest_head: Some(latest_head),
                block_ref,
                ..Default::default()
            });
            proposer.l1_view = view.clone();

            assert!(proposer.sync_state().await.is_err());
            assert_eq!(
                view.block_calls(),
                vec![("latest_game_index", BlockId::hash(expected_hash))]
            );
        }
    }

    #[tokio::test]
    async fn sync_head_failures_and_no_advance_outcomes_preserve_the_last_pin() {
        let cases = [
            (Some("latest_head"), 1, 3, 0, None, &["latest_head"] as &[_]),
            (None, 5, 5, 0, Some(SyncDisposition::UnchangedConfirmedHead), &["latest_head"]),
            (
                None,
                4,
                5,
                0,
                Some(SyncDisposition::ConfirmedHeadRegressed { observed_number: 4 }),
                &["latest_head"],
            ),
            (
                None,
                10,
                3,
                2,
                Some(SyncDisposition::ConfirmedBlockUnavailable),
                &["latest_head", "block_ref"],
            ),
        ];

        for (fail_on, head, previous_pin, confirmations, expected, expected_calls) in cases {
            let mut config = test_config();
            config.sync_l1_confirmations = confirmations;
            let mut proposer = test_proposer_with(config).await;
            let view = Arc::new(RecordingL1View {
                fail_on,
                latest_head: Some(L1BlockRef { hash: B256::ZERO, number: head, timestamp: 1_000 }),
                ..Default::default()
            });
            proposer.l1_view = view.clone();
            *proposer.last_successful_pinned_l1.write().await =
                Some(L1BlockRef { hash: B256::ZERO, number: previous_pin, timestamp: 1_000 });
            match expected {
                Some(expected) => assert_eq!(proposer.sync_state().await.unwrap(), expected),
                None => assert!(proposer.sync_state().await.is_err()),
            }
            assert_eq!(
                *proposer.last_successful_pinned_l1.read().await,
                Some(L1BlockRef { hash: B256::ZERO, number: previous_pin, timestamp: 1_000 })
            );
            assert_eq!(view.calls(), expected_calls);
        }
    }

    #[tokio::test]
    async fn sync_state_advances_only_once_when_head_is_block_zero() {
        let mut proposer = test_proposer().await;
        proposer.l1_view = Arc::new(RecordingL1View {
            latest_head: Some(L1BlockRef { hash: B256::ZERO, number: 0, timestamp: 1_000 }),
            ..Default::default()
        });

        assert_eq!(proposer.sync_state().await.unwrap(), SyncDisposition::Advanced);
        assert_eq!(proposer.sync_state().await.unwrap(), SyncDisposition::UnchangedConfirmedHead);
    }

    #[tokio::test]
    async fn factory_failure_does_not_advance_the_successful_pin() {
        let view = Arc::new(RecordingL1View {
            latest_head: Some(L1BlockRef { hash: B256::ZERO, number: 5, timestamp: 1_000 }),
            fail_on: Some("latest_game_index"),
            ..Default::default()
        });
        let mut proposer = test_proposer().await;
        proposer.l1_view = view.clone();
        *proposer.last_successful_pinned_l1.write().await =
            Some(L1BlockRef { hash: B256::ZERO, number: 3, timestamp: 900 });
        let cursor = Cursor::from(U256::from(7));
        proposer.state.write().await.cursor = cursor.clone();

        assert!(proposer.sync_state().await.is_err());
        assert_eq!(
            *proposer.last_successful_pinned_l1.read().await,
            Some(L1BlockRef { hash: B256::ZERO, number: 3, timestamp: 900 })
        );
        assert_eq!(proposer.state.read().await.cursor, cursor);
        assert_eq!(view.calls(), vec!["latest_head", "latest_game_index"]);
    }
    #[tokio::test]
    async fn sync_factory_history_resets() {
        for (latest_game_index, cursor, expected_cursor) in [
            (None, Cursor::from(U256::from(7)), Cursor::none()),
            (Some(U256::ZERO), Cursor::from(U256::from(7)), Cursor::from(U256::ZERO)),
        ] {
            let view = Arc::new(RecordingL1View {
                latest_game_index,
                factory_game: FactoryGame {
                    address: Address::left_padding_from(&[0x44]),
                    game_type: ZK_GAME_TYPE + 1,
                },
                ..Default::default()
            });
            let mut proposer = test_proposer().await;
            proposer.l1_view = view;
            let proof_engine = Arc::new(RecordingProofEngine::default());
            proposer.proof_engine = proof_engine.clone();
            let cached = game_with(4, u32::MAX, 100);
            let cached_address = cached.address;
            {
                let mut state = proposer.state.write().await;
                state.anchor_game = Some(cached.clone());
                state.canonical_head_index = Some(cached.index);
                state.canonical_head_sequence_number = Some(123);
                state.cursor = cursor;
                state.games.insert(cached.index, cached);
                state.invalid_games.insert(U256::from(3));
            }
            proposer.pending_games.write().await.insert(
                U256::from(9),
                CompactGameSummary {
                    factory_index: U256::from(9),
                    address: Address::ZERO,
                    parent_index: u32::MAX,
                    sequence_number: 0,
                },
            );

            proposer.sync_games(BlockId::number(1), 1_000).await.unwrap();

            let state = proposer.state.read().await;
            assert!(state.anchor_game.is_none());
            assert_eq!(state.canonical_head_index, None);
            assert_eq!(state.canonical_head_sequence_number, Some(123));
            assert_eq!(state.cursor, expected_cursor);
            assert!(state.games.is_empty());
            assert!(state.invalid_games.is_empty());
            assert!(proposer.pending_games.read().await.is_empty());
            assert_eq!(*proof_engine.cleared.lock().unwrap(), vec![cached_address]);
        }
    }

    #[tokio::test]
    async fn sync_failure_boundaries_preserve_cursor_and_isolate_existing_games() {
        let anchor_failure = Arc::new(RecordingL1View {
            latest_head: Some(L1BlockRef { hash: B256::ZERO, number: 5, timestamp: 1_000 }),
            latest_game_index: Some(U256::ZERO),
            fail_on: Some("registered_anchor_game"),
            ..Default::default()
        });
        let mut proposer = test_proposer().await;
        proposer.l1_view = anchor_failure.clone();
        *proposer.last_successful_pinned_l1.write().await =
            Some(L1BlockRef { hash: B256::ZERO, number: 3, timestamp: 900 });
        let cursor = Cursor::from(U256::ZERO);
        proposer.state.write().await.cursor = cursor.clone();
        assert!(proposer.sync_state().await.is_err());
        assert_eq!(
            *proposer.last_successful_pinned_l1.read().await,
            Some(L1BlockRef { hash: B256::ZERO, number: 3, timestamp: 900 })
        );
        assert_eq!(proposer.state.read().await.cursor, cursor);
        assert_eq!(
            anchor_failure.calls(),
            vec!["latest_head", "latest_game_index", "registered_anchor_game"]
        );

        let discovery_failure = Arc::new(RecordingL1View {
            latest_head: Some(L1BlockRef { hash: B256::ZERO, number: 5, timestamp: 1_000 }),
            latest_game_index: Some(U256::ZERO),
            fail_on: Some("factory_game"),
            ..Default::default()
        });
        let mut proposer = test_proposer().await;
        proposer.l1_view = discovery_failure.clone();
        *proposer.last_successful_pinned_l1.write().await =
            Some(L1BlockRef { hash: B256::ZERO, number: 3, timestamp: 900 });
        assert!(proposer.sync_state().await.is_err());
        assert_eq!(
            *proposer.last_successful_pinned_l1.read().await,
            Some(L1BlockRef { hash: B256::ZERO, number: 3, timestamp: 900 })
        );
        assert_eq!(proposer.state.read().await.cursor, Cursor::none());
        assert_eq!(
            discovery_failure.calls(),
            vec!["latest_head", "latest_game_index", "registered_anchor_game", "factory_game",]
        );

        let lifecycle_failure = Arc::new(RecordingL1View {
            latest_game_index: Some(U256::ZERO),
            fail_on: Some("game_lifecycle"),
            ..Default::default()
        });
        let mut proposer = test_proposer().await;
        proposer.l1_view = lifecycle_failure.clone();
        let game = Game {
            status: GameStatus::DefenderWins,
            proposal_status: ProposalStatus::Challenged,
            deadline: 777,
            should_attempt_to_resolve: true,
            should_attempt_to_claim_bond: true,
            ..game_with(0, u32::MAX, 100)
        };
        {
            let mut state = proposer.state.write().await;
            state.cursor = Cursor::from(U256::ZERO);
            state.games.insert(game.index, game.clone());
        }
        proposer.sync_games(BlockId::number(1), 1_000).await.unwrap();
        let cached = proposer.state.read().await.games.get(&game.index).unwrap().clone();
        assert_eq!(cached, game);
        assert_eq!(
            lifecycle_failure.calls(),
            vec!["latest_game_index", "registered_anchor_game", "game_lifecycle"]
        );

        let pending_failure = Arc::new(RecordingL1View {
            latest_game_index: Some(U256::ZERO),
            fail_on: Some("factory_game"),
            ..Default::default()
        });
        let mut proposer = test_proposer().await;
        proposer.l1_view = pending_failure.clone();
        proposer.state.write().await.cursor = Cursor::from(U256::ZERO);
        proposer.pending_games.write().await.insert(
            U256::ZERO,
            CompactGameSummary {
                factory_index: U256::ZERO,
                address: Address::ZERO,
                parent_index: u32::MAX,
                sequence_number: 0,
            },
        );
        proposer.sync_games(BlockId::number(1), 1_000).await.unwrap();
        assert!(proposer.pending_games.read().await.contains_key(&U256::ZERO));
        assert_eq!(
            pending_failure.calls(),
            vec!["latest_game_index", "registered_anchor_game", "factory_game"]
        );
    }

    #[tokio::test]
    async fn lifecycle_refresh_only_reads_the_active_status_branch() {
        for (status, parent_index, expected_branch) in [
            (GameStatus::InProgress, 0, Some("parent_game_status")),
            (GameStatus::InProgress, u32::MAX, None),
            (GameStatus::DefenderWins, 0, Some("bond_state")),
            (GameStatus::ChallengerWins, 0, None),
        ] {
            let prestate = B256::left_padding_from(&[0x77]);
            let game = Game {
                address: Address::left_padding_from(&[0x70]),
                absolute_prestate: prestate,
                anchor_state_registry: Address::left_padding_from(&[0xa1]),
                weth: Address::left_padding_from(&[0xb1]),
                ..game_with(0, parent_index, 100)
            };
            let view = Arc::new(RecordingL1View {
                latest_game_index: Some(U256::ZERO),
                anchor_game: game.address,
                lifecycle: GameLifecycle {
                    proposal_status: ProposalStatus::Unchallenged,
                    deadline: 2_000,
                    parent_index,
                    status,
                    is_finalized: false,
                },
                ..Default::default()
            });
            let mut proposer = test_proposer().await;
            let expected_lifecycle_target =
                (game.address, game.anchor_state_registry, BlockId::number(1));
            let expected_bond_target =
                (game.address, game.weth, proposer.proposer_address, BlockId::number(1));
            proposer.l1_view = view.clone();
            proposer
                .prestates
                .insert_for_tests(
                    prestate,
                    PrestatePrograms { aggregation_elf: vec![1], range_elf: vec![1] },
                )
                .await;
            if status == GameStatus::ChallengerWins {
                proposer.last_created_game_l2_sequence_number.store(100, AtomicOrdering::Relaxed);
                *proposer.last_created_game_address.lock().await = game.address;
            }
            {
                let mut state = proposer.state.write().await;
                state.cursor = Cursor::from(U256::ZERO);
                state.games.insert(game.index, game);
            }

            proposer.sync_games(BlockId::number(1), 1_000).await.unwrap();

            let mut expected =
                vec!["latest_game_index", "registered_anchor_game", "game_lifecycle"];
            if let Some(branch) = expected_branch {
                expected.push(branch);
            }
            assert_eq!(
                view.block_calls(),
                expected.iter().map(|call| (*call, BlockId::number(1))).collect::<Vec<_>>(),
                "status {status:?}, parent {parent_index}"
            );
            assert_eq!(
                *view.lifecycle_targets.lock().unwrap(),
                vec![expected_lifecycle_target],
                "status {status:?}, parent {parent_index}"
            );
            if status == GameStatus::DefenderWins {
                assert_eq!(*view.bond_targets.lock().unwrap(), vec![expected_bond_target]);
            }

            let state = proposer.state.read().await;
            if status == GameStatus::ChallengerWins {
                assert!(!state.games.contains_key(&U256::ZERO));
                assert!(state.invalid_games.contains(&U256::ZERO));
                assert_eq!(
                    proposer.last_created_game_l2_sequence_number.load(AtomicOrdering::Relaxed),
                    0
                );
                assert_eq!(*proposer.last_created_game_address.lock().await, Address::ZERO);
            } else {
                let cached = state.games.get(&U256::ZERO).unwrap();
                assert_eq!(cached.status, status);
                assert_eq!(cached.proposal_status, ProposalStatus::Unchallenged);
                assert_eq!(cached.deadline, 2_000);
                assert!(!cached.should_attempt_to_resolve);
                assert!(!cached.should_attempt_to_claim_bond);
            }
        }
    }

    #[tokio::test]
    async fn lifecycle_refresh_loads_cached_prestates() {
        let prestate = B256::left_padding_from(&[0x66]);
        let dir = write_test_prestate_artifacts("lifecycle", prestate);
        let mut config = test_config();
        config.prestates_url =
            alloy_transport_http::reqwest::Url::from_directory_path(&dir).unwrap();
        let mut proposer = test_proposer_with(config).await;
        let game = Game { absolute_prestate: prestate, ..game_with(0, u32::MAX, 100) };
        proposer.l1_view = Arc::new(RecordingL1View {
            latest_game_index: Some(U256::ZERO),
            lifecycle: GameLifecycle {
                proposal_status: ProposalStatus::Unchallenged,
                deadline: 999,
                parent_index: u32::MAX,
                status: GameStatus::InProgress,
                is_finalized: false,
            },
            ..Default::default()
        });
        {
            let mut state = proposer.state.write().await;
            state.cursor = Cursor::from(U256::ZERO);
            state.games.insert(game.index, game);
        }
        assert!(proposer.prestates.known_prestates().await.is_empty());

        proposer.sync_games(BlockId::number(1), 1_000).await.unwrap();

        assert!(proposer.prestates.known_prestates().await.contains(&prestate));
        assert!(
            proposer.state.read().await.games.get(&U256::ZERO).unwrap().should_attempt_to_resolve
        );
        std::fs::remove_dir_all(dir).unwrap();
    }
    #[tokio::test]
    async fn sync_games_applies_lifecycle_actions() {
        let prestate = B256::left_padding_from(&[0x55]);
        let retained = Game {
            absolute_prestate: prestate,
            should_attempt_to_resolve: true,
            should_attempt_to_claim_bond: true,
            ..game_with(0, u32::MAX, 100)
        };
        let completed = Game { absolute_prestate: prestate, ..game_with(1, 0, 101) };
        let retained_address = retained.address;
        let completed_address = completed.address;
        let mut proposer = test_proposer().await;
        let proof_engine = Arc::new(RecordingProofEngine::default());
        proposer.proof_engine = proof_engine.clone();
        proposer.l1_view = Arc::new(RecordingL1View {
            latest_game_index: Some(U256::ZERO),
            anchor_game: retained.address,
            lifecycle: GameLifecycle {
                proposal_status: ProposalStatus::Resolved,
                deadline: 321,
                parent_index: u32::MAX,
                status: GameStatus::DefenderWins,
                is_finalized: true,
            },
            bond_state: BondState {
                credit: U256::ZERO,
                withdrawal_amount: U256::ZERO,
                withdrawal_timestamp: U256::ZERO,
                delay: U256::ZERO,
            },
            ..Default::default()
        });
        proposer
            .prestates
            .insert_for_tests(
                prestate,
                PrestatePrograms { aggregation_elf: vec![1], range_elf: vec![1] },
            )
            .await;
        {
            let mut state = proposer.state.write().await;
            state.cursor = Cursor::from(U256::ZERO);
            state.canonical_head_index = Some(retained.index);
            state.games.insert(retained.index, retained);
            state.games.insert(completed.index, completed);
        }

        proposer.sync_games(BlockId::number(1), 1_000).await.unwrap();

        let state = proposer.state.read().await;
        let cached = state.games.get(&U256::ZERO).unwrap();
        assert_eq!(cached.status, GameStatus::DefenderWins);
        assert_eq!(cached.proposal_status, ProposalStatus::Resolved);
        assert_eq!(cached.deadline, 321);
        assert!(!cached.should_attempt_to_resolve);
        assert!(!cached.should_attempt_to_claim_bond);
        assert!(!state.games.contains_key(&U256::ONE));
        let cleared = proof_engine.cleared.lock().unwrap().iter().copied().collect::<HashSet<_>>();
        assert_eq!(cleared, HashSet::from([retained_address, completed_address]));
    }

    #[tokio::test]
    async fn removing_subtree_clears_its_proof_progress() {
        let root = game_with(7, u32::MAX, 100);
        let child = game_with(8, 7, 101);
        let sibling = game_with(9, u32::MAX, 102);
        let removed_addresses = HashSet::from([root.address, child.address]);
        let sibling_index = sibling.index;
        let mut proposer = test_proposer().await;
        let proof_engine = Arc::new(RecordingProofEngine::default());
        proposer.proof_engine = proof_engine.clone();
        {
            let mut state = proposer.state.write().await;
            state.games.insert(root.index, root);
            state.games.insert(child.index, child);
            state.games.insert(sibling.index, sibling);
        }

        proposer.apply_game_sync_actions(vec![GameSyncAction::RemoveSubtree(U256::from(7))]).await;

        let state = proposer.state.read().await;
        assert_eq!(state.games.len(), 1);
        assert!(state.games.contains_key(&sibling_index));
        let cleared = proof_engine.cleared.lock().unwrap().iter().copied().collect::<HashSet<_>>();
        assert_eq!(cleared, removed_addresses);
    }

    mod game_sync_classification {
        use super::*;

        fn lifecycle(status: GameStatus) -> GameLifecycle {
            GameLifecycle {
                proposal_status: ProposalStatus::Unchallenged,
                deadline: 100,
                parent_index: u32::MAX,
                status,
                is_finalized: false,
            }
        }

        fn in_progress(
            lifecycle: GameLifecycle,
            parent_resolved: bool,
            owned: bool,
        ) -> GameSyncFacts {
            GameSyncFacts::InProgress { index: U256::from(7), lifecycle, parent_resolved, owned }
        }

        fn defender_wins(
            lifecycle: GameLifecycle,
            bond: BondState,
            canonical_head_index: Option<U256>,
        ) -> GameSyncFacts {
            GameSyncFacts::DefenderWins {
                index: U256::from(7),
                game_address: Address::left_padding_from(&[0x77]),
                lifecycle,
                bond,
                canonical_head_index,
            }
        }

        #[test]
        fn classifies_in_progress_resolution_from_immutable_facts() {
            let facts = in_progress(lifecycle(GameStatus::InProgress), true, true);
            assert_eq!(
                classify_game_sync(facts, 100, Address::ZERO).unwrap(),
                GameSyncAction::Update {
                    index: U256::from(7),
                    lifecycle: lifecycle(GameStatus::InProgress),
                    should_attempt_to_resolve: false,
                    should_attempt_to_claim_bond: false,
                    retention: None,
                }
            );
            assert_eq!(
                classify_game_sync(facts, 101, Address::ZERO).unwrap(),
                GameSyncAction::Update {
                    index: U256::from(7),
                    lifecycle: lifecycle(GameStatus::InProgress),
                    should_attempt_to_resolve: true,
                    should_attempt_to_claim_bond: false,
                    retention: None,
                }
            );

            let proof_facts = in_progress(
                GameLifecycle {
                    proposal_status: ProposalStatus::ChallengedAndValidProofProvided,
                    ..lifecycle(GameStatus::InProgress)
                },
                true,
                true,
            );
            assert!(matches!(
                classify_game_sync(proof_facts, 0, Address::ZERO).unwrap(),
                GameSyncAction::Update { should_attempt_to_resolve: true, .. }
            ));
        }

        #[test]
        fn classifies_defender_wins_claims_and_retention() {
            let index = U256::from(7);
            let address = Address::left_padding_from(&[0x77]);
            let finalized =
                GameLifecycle { is_finalized: true, ..lifecycle(GameStatus::DefenderWins) };
            let done = BondState {
                credit: U256::ZERO,
                withdrawal_amount: U256::ZERO,
                withdrawal_timestamp: U256::ZERO,
                delay: U256::ZERO,
            };
            assert!(matches!(
                classify_game_sync(defender_wins(finalized, done, Some(index)), 100, address)
                    .unwrap(),
                GameSyncAction::Update { retention: Some(GameSyncRetention::CanonicalHead), .. }
            ));
            assert!(matches!(
                classify_game_sync(defender_wins(finalized, done, None), 100, address).unwrap(),
                GameSyncAction::Update { retention: Some(GameSyncRetention::Anchor), .. }
            ));
            assert_eq!(
                classify_game_sync(defender_wins(finalized, done, None), 100, Address::ZERO)
                    .unwrap(),
                GameSyncAction::Remove(index)
            );

            let unlock = BondState { credit: U256::ONE, ..done };
            assert!(matches!(
                classify_game_sync(defender_wins(finalized, unlock, None), 100, Address::ZERO)
                    .unwrap(),
                GameSyncAction::Update { should_attempt_to_claim_bond: true, retention: None, .. }
            ));
            assert!(matches!(
                classify_game_sync(
                    defender_wins(lifecycle(GameStatus::DefenderWins), unlock, None,),
                    100,
                    Address::ZERO
                )
                .unwrap(),
                GameSyncAction::Update { should_attempt_to_claim_bond: false, retention: None, .. }
            ));
        }

        #[test]
        fn preserves_bond_conversion_boundaries() {
            let bond = BondState {
                credit: U256::ZERO,
                withdrawal_amount: U256::ONE,
                withdrawal_timestamp: U256::MAX,
                delay: U256::ZERO,
            };
            assert!(matches!(
                classify_game_sync(
                    defender_wins(lifecycle(GameStatus::DefenderWins), bond, None),
                    u64::MAX,
                    Address::ZERO
                )
                .unwrap(),
                GameSyncAction::Update { should_attempt_to_claim_bond: true, .. }
            ));

            let error = classify_game_sync(
                defender_wins(
                    lifecycle(GameStatus::DefenderWins),
                    BondState { delay: U256::from(u64::MAX) + U256::ONE, ..bond },
                    None,
                ),
                u64::MAX,
                Address::ZERO,
            )
            .unwrap_err();
            assert!(error.to_string().contains("DelayedWETH delay exceeds u64"));
        }

        #[test]
        fn challenger_wins_removes_the_subtree() {
            assert_eq!(
                classify_game_sync(
                    GameSyncFacts::ChallengerWins { index: U256::from(7) },
                    100,
                    Address::ZERO,
                )
                .unwrap(),
                GameSyncAction::RemoveSubtree(U256::from(7))
            );
        }
    }

    mod pending_revalidation_transitions {
        use super::*;

        fn pending_view(prestate: B256) -> RecordingL1View {
            RecordingL1View {
                factory_game: FactoryGame {
                    address: Address::left_padding_from(&[0x44]),
                    game_type: ZK_GAME_TYPE,
                },
                game_claim: GameClaim {
                    status: ProposalStatus::Unchallenged as u8,
                    deadline: 0,
                    parent_index: u32::MAX,
                },
                game_identity: GameIdentity {
                    sequence_number: U256::from(100),
                    ..Default::default()
                },
                game_validity: GameValidity {
                    root_claim: B256::ZERO,
                    was_respected: true,
                    status: GameStatus::InProgress,
                    absolute_prestate: prestate,
                },
                ..Default::default()
            }
        }

        #[tokio::test]
        async fn loads_pending_prestate_before_eviction_check() {
            let prestate = B256::left_padding_from(&[0x11]);
            let dir = write_test_prestate_artifacts("pending-owned", prestate);
            let mut config = test_config();
            config.prestates_url =
                alloy_transport_http::reqwest::Url::from_directory_path(&dir).unwrap();
            let view = Arc::new(RecordingL1View {
                latest_game_index: Some(U256::ZERO),
                ..pending_view(prestate)
            });
            let mut proposer = test_proposer_with(config).await;
            proposer.l1_view = view;
            proposer.superroot_source = Arc::new(UnavailableSuperRootSource);
            let mut anchor = game_with(9, u32::MAX, 100);
            anchor.deadline = MAX_GAME_DEADLINE_LAG + 10;
            {
                let mut state = proposer.state.write().await;
                state.cursor = Cursor::from(U256::ZERO);
                state.anchor_game = Some(anchor);
            }
            proposer.pending_games.write().await.insert(
                U256::ZERO,
                CompactGameSummary {
                    factory_index: U256::ZERO,
                    address: Address::left_padding_from(&[0x44]),
                    parent_index: u32::MAX,
                    sequence_number: 100,
                },
            );
            assert!(proposer.prestates.known_prestates().await.is_empty());

            proposer.sync_games(BlockId::number(1), 1_000).await.unwrap();

            assert!(proposer.pending_games.read().await.contains_key(&U256::ZERO));
            assert!(proposer.prestates.known_prestates().await.contains(&prestate));
            std::fs::remove_dir_all(dir).unwrap();
        }

        #[tokio::test]
        async fn newly_discovered_pending_game_is_not_revalidated_same_cycle() {
            let prestate = B256::left_padding_from(&[0x33]);
            let view = Arc::new(RecordingL1View {
                latest_game_index: Some(U256::ZERO),
                ..pending_view(prestate)
            });
            let mut proposer = test_proposer().await;
            proposer.l1_view = view.clone();
            proposer.superroot_source = Arc::new(UnavailableSuperRootSource);

            proposer.sync_games(BlockId::number(1), 1_000).await.unwrap();

            assert!(proposer.pending_games.read().await.contains_key(&U256::ZERO));
            assert_eq!(view.calls().into_iter().filter(|call| *call == "factory_game").count(), 1);
        }
    }

    #[tokio::test]
    async fn discovery_stops_after_each_terminal_classification_boundary() {
        let game_address = Address::left_padding_from(&[0x44]);
        let unsupported = Arc::new(RecordingL1View {
            factory_game: FactoryGame { address: game_address, game_type: ZK_GAME_TYPE + 1 },
            ..Default::default()
        });
        let mut proposer = test_proposer().await;
        proposer.l1_view = unsupported.clone();
        assert!(matches!(
            proposer.fetch_game(U256::ZERO, BlockId::number(1)).await.unwrap(),
            GameFetchResult::UnsupportedType { game_address: address } if address == game_address
        ));
        assert_eq!(unsupported.calls(), vec!["factory_game"]);

        let invalid_parent = Arc::new(RecordingL1View {
            factory_game: FactoryGame { address: game_address, game_type: ZK_GAME_TYPE },
            game_claim: GameClaim {
                status: ProposalStatus::Unchallenged as u8,
                deadline: 2_000,
                parent_index: 7,
            },
            ..Default::default()
        });
        let mut proposer = test_proposer().await;
        proposer.l1_view = invalid_parent.clone();
        proposer.state.write().await.invalid_games.insert(U256::from(7));
        assert!(matches!(
            proposer.fetch_game(U256::ZERO, BlockId::number(1)).await.unwrap(),
            GameFetchResult::InvalidGame { .. }
        ));
        assert_eq!(invalid_parent.calls(), vec!["factory_game", "game_claim"]);

        let sequence_overflow = Arc::new(RecordingL1View {
            factory_game: FactoryGame { address: game_address, game_type: ZK_GAME_TYPE },
            game_identity: GameIdentity { sequence_number: U256::MAX, ..Default::default() },
            ..Default::default()
        });
        let mut proposer = test_proposer().await;
        proposer.l1_view = sequence_overflow.clone();
        assert!(matches!(
            proposer.fetch_game(U256::ZERO, BlockId::number(1)).await.unwrap(),
            GameFetchResult::InvalidGame { .. }
        ));
        assert_eq!(
            sequence_overflow.block_calls(),
            vec![
                ("factory_game", BlockId::number(1)),
                ("game_claim", BlockId::number(1)),
                ("game_identity", BlockId::number(1)),
            ]
        );
    }

    #[tokio::test]
    async fn discovery_defers_game_when_super_root_rpc_is_unavailable() {
        let prestate = B256::left_padding_from(&[0x77]);
        let view = Arc::new(RecordingL1View {
            factory_game: FactoryGame {
                address: Address::left_padding_from(&[0x44]),
                game_type: ZK_GAME_TYPE,
            },
            game_identity: GameIdentity { sequence_number: U256::from(100), ..Default::default() },
            game_validity: GameValidity {
                root_claim: B256::ZERO,
                was_respected: true,
                status: GameStatus::InProgress,
                absolute_prestate: prestate,
            },
            ..Default::default()
        });
        let mut proposer = test_proposer().await;
        proposer.l1_view = view.clone();
        proposer.superroot_source = Arc::new(UnavailableSuperRootSource);

        assert!(matches!(
            proposer.fetch_game(U256::ZERO, BlockId::number(1)).await.unwrap(),
            GameFetchResult::Pending { index, deadline: 2_000, prestate: observed, .. }
                if index == U256::ZERO && observed == prestate
        ));
        assert_eq!(
            view.block_calls(),
            vec![
                ("factory_game", BlockId::number(1)),
                ("game_claim", BlockId::number(1)),
                ("game_identity", BlockId::number(1)),
                ("game_validity", BlockId::number(1)),
            ]
        );
    }

    #[tokio::test]
    async fn discovery_classifies_super_root_at_timestamp_results() {
        enum Expected {
            Pending,
            Invalid,
            Valid,
        }

        let canonical = B256::repeat_byte(0x11);
        let cases = [
            (100, absent_super_root_at_timestamp(99), canonical, Expected::Pending),
            (
                super::MAX_GAME_DEADLINE_LAG + 101,
                absent_super_root_at_timestamp(100),
                canonical,
                Expected::Invalid,
            ),
            (100, super_root_at_timestamp(100, canonical, 12, 11), canonical, Expected::Valid),
            (
                100,
                super_root_at_timestamp(100, canonical, 12, 11),
                B256::repeat_byte(0x22),
                Expected::Invalid,
            ),
            (
                100,
                super_root_at_timestamp(100, canonical, 11, 11),
                B256::repeat_byte(0x22),
                Expected::Pending,
            ),
        ];

        for (sequence_number, super_root_at, claim, expected) in cases {
            let game_address = Address::repeat_byte(0x44);
            let view = Arc::new(RecordingL1View {
                factory_game: FactoryGame { address: game_address, game_type: ZK_GAME_TYPE },
                game_identity: GameIdentity {
                    sequence_number: U256::from(sequence_number),
                    ..Default::default()
                },
                game_validity: GameValidity {
                    root_claim: claim,
                    was_respected: true,
                    status: GameStatus::InProgress,
                    absolute_prestate: B256::ZERO,
                },
                ..Default::default()
            });
            let mut proposer = test_proposer().await;
            proposer.l1_view = view;
            proposer.superroot_source = Arc::new(ScriptedSuperRootSource {
                horizon: ProposalHorizon { safe_timestamp: 100, finalized_timestamp: 100 },
                roots: vec![(sequence_number, super_root_at)],
            });

            let result = proposer.fetch_game(U256::ZERO, BlockId::number(1)).await.unwrap();
            assert!(match expected {
                Expected::Pending => matches!(result, GameFetchResult::Pending { .. }),
                Expected::Invalid => matches!(result, GameFetchResult::InvalidGame { .. }),
                Expected::Valid => matches!(
                    result,
                    GameFetchResult::ValidGame { game_address: observed, .. }
                        if observed == game_address
                ),
            });
        }
    }

    #[tokio::test]
    async fn initialization_snapshots_proving_durations_across_registry_rotation() {
        fn registered_view(
            prestate: B256,
            registry: Address,
            weth: Address,
            durations: (u64, u64),
            sequence_number: u64,
        ) -> Arc<RecordingL1View> {
            Arc::new(RecordingL1View {
                registered_args: ZKGameArgs {
                    absolute_prestate: prestate,
                    verifier: Address::ZERO,
                    max_challenge_duration: durations.0,
                    max_prove_duration: durations.1,
                    challenger_bond: U256::ZERO,
                    anchor_state_registry: registry,
                    weth,
                },
                anchor_root: AnchorRoot {
                    root: B256::left_padding_from(&[sequence_number as u8]),
                    sequence_number: U256::from(sequence_number),
                },
                ..Default::default()
            })
        }

        let first_prestate = B256::left_padding_from(&[0x11]);
        let second_prestate = B256::left_padding_from(&[0x22]);
        let first_registry = Address::left_padding_from(&[0xa1]);
        let second_registry = Address::left_padding_from(&[0xa2]);
        let first = registered_view(
            first_prestate,
            first_registry,
            Address::left_padding_from(&[0xb1]),
            (10, 20),
            100,
        );
        let second = registered_view(
            second_prestate,
            second_registry,
            Address::left_padding_from(&[0xb2]),
            (30, 40),
            200,
        );
        let mut proposer = test_proposer().await;
        for prestate in [first_prestate, second_prestate] {
            proposer
                .prestates
                .insert_for_tests(
                    prestate,
                    PrestatePrograms { aggregation_elf: vec![1], range_elf: vec![1] },
                )
                .await;
        }

        proposer.l1_view = first.clone();
        assert_eq!(proposer.startup_validations().await.unwrap(), 100);
        proposer.l1_view = second.clone();
        assert_eq!(proposer.startup_validations().await.unwrap(), 200);

        assert_eq!(proposer.max_challenge_duration.get(), Some(&10));
        assert_eq!(proposer.max_prove_duration.get(), Some(&20));
        assert_eq!(
            *first.anchor_targets.lock().unwrap(),
            vec![(first_registry, BlockId::latest())]
        );
        assert_eq!(
            *second.anchor_targets.lock().unwrap(),
            vec![(second_registry, BlockId::latest())]
        );
        let expected_blocks =
            vec![("registered_game_args", BlockId::latest()), ("anchor_root", BlockId::latest())];
        assert_eq!(first.block_calls(), expected_blocks);
        assert_eq!(second.block_calls(), expected_blocks);
    }

    #[tokio::test]
    async fn query_time_failure_stops_before_the_superroot_request() {
        let mut proposer = test_proposer().await;
        proposer.query_time = Arc::new(ErrorQueryTime);

        let err = proposer.max_proposable_timestamp().await.unwrap_err();
        assert!(err.to_string().contains("query time unavailable"));
    }

    #[tokio::test]
    async fn proposal_safety_selects_the_injected_horizon() {
        for (safety, expected) in [(ProposalSafety::Safe, 120), (ProposalSafety::Finalized, 110)] {
            let mut config = test_config();
            config.proposal_safety = safety;
            let mut proposer = test_proposer_with(config).await;
            proposer.superroot_source = Arc::new(ScriptedSuperRootSource {
                horizon: ProposalHorizon { safe_timestamp: 120, finalized_timestamp: 110 },
                roots: Vec::new(),
            });

            assert_eq!(proposer.max_proposable_timestamp().await.unwrap(), expected);
        }
    }

    #[tokio::test]
    async fn semantic_action_ports_receive_policy_inputs() {
        let actions = Arc::new(RecordingActionExecutor::default());
        let mut proposer = test_proposer().await;
        proposer.l1_view = Arc::new(RecordingL1View {
            claim_preflight: Some((
                U256::from(1),
                WithdrawalState { amount: U256::ZERO, timestamp: U256::ZERO },
            )),
            ..Default::default()
        });
        proposer.action_executor = actions.clone();
        let root_claim = B256::left_padding_from(&[0x11]);
        let extra_data = vec![0x22, 0x33];
        let prestate = B256::left_padding_from(&[0x44]);
        let mut game = game_with(7, u32::MAX, 100);
        game.absolute_prestate = prestate;
        game.should_attempt_to_claim_bond = true;
        proposer
            .prestates
            .insert_for_tests(
                prestate,
                PrestatePrograms { aggregation_elf: vec![1], range_elf: vec![1] },
            )
            .await;
        proposer.state.write().await.games.insert(game.index, game.clone());

        assert_eq!(
            proposer.create_game(root_claim, extra_data.clone()).await.unwrap(),
            Address::left_padding_from(&[0xc1])
        );
        proposer.submit_resolution_transaction(&game).await.unwrap();
        proposer.claim_bonds().await.unwrap();

        assert_eq!(
            *actions.calls.lock().unwrap(),
            vec![
                ActionCall::Create { root_claim, extra_data, init_bond: U256::ZERO },
                ActionCall::Resolve(game.address),
                ActionCall::ClaimCredit {
                    game: game.address,
                    recipient: proposer.proposer_address,
                },
            ]
        );
    }

    #[tokio::test]
    async fn l1_submission_retry_reuses_cached_proof() {
        let starting_root = B256::repeat_byte(0x11);
        let root_claim = B256::repeat_byte(0x22);
        let proof = vec![0xaa, 0xbb];
        let game = game_with(7, u32::MAX, 101);
        let view = Arc::new(RecordingL1View {
            proof_inputs: ProofInputs {
                l1_head: B256::repeat_byte(0x33),
                l1_head_number: 20,
                starting_root,
                starting_sequence_number: 100,
                root_claim,
                sequence_number: 101,
            },
            ..Default::default()
        });
        let engine = Arc::new(RecordingProofEngine {
            calls: StdMutex::new(Vec::new()),
            proof: proof.clone(),
            fail: false,
            cleared: StdMutex::new(Vec::new()),
            cached: StdMutex::new(None),
            generations: std::sync::atomic::AtomicUsize::new(0),
        });
        let actions = Arc::new(RecordingActionExecutor {
            prove_failures: StdMutex::new(1),
            ..Default::default()
        });
        let mut proposer = test_proposer().await;
        proposer.l1_view = view;
        proposer.superroot_source = Arc::new(ScriptedSuperRootSource {
            horizon: ProposalHorizon { safe_timestamp: 101, finalized_timestamp: 101 },
            roots: vec![
                (100, super_root_at_timestamp(100, starting_root, 12, 10)),
                (101, super_root_at_timestamp(101, root_claim, 12, 10)),
            ],
        });
        proposer.proof_engine = engine.clone();
        proposer.action_executor = actions.clone();
        proposer.state.write().await.games.insert(game.index, game.clone());

        assert!(proposer.prove_game(game.address).await.is_err());
        assert_eq!(engine.generations.load(AtomicOrdering::SeqCst), 1);
        assert!(engine.cleared.lock().unwrap().is_empty());

        proposer.prove_game(game.address).await.unwrap();

        let calls = engine.calls.lock().unwrap();
        assert_eq!(calls.len(), 2);
        assert_eq!(calls[0].0.starting_root, starting_root);
        assert_eq!(calls[0].0.root_claim, root_claim);
        assert_eq!(calls[0].0.prover, proposer.proposer_address);
        assert_eq!(calls[0].1.len(), 2);
        drop(calls);
        assert_eq!(engine.generations.load(AtomicOrdering::SeqCst), 1);
        assert_eq!(
            *actions.calls.lock().unwrap(),
            vec![
                ActionCall::Prove { game: game.address, proof: proof.clone() },
                ActionCall::Prove { game: game.address, proof },
            ]
        );
        assert_eq!(*engine.cleared.lock().unwrap(), vec![game.address]);
    }

    #[tokio::test]
    async fn proof_failure_and_post_proof_skip_do_not_submit() {
        for (fail, status) in [
            (true, ProposalStatus::Unchallenged),
            (false, ProposalStatus::UnchallengedAndValidProofProvided),
        ] {
            let starting_root = B256::repeat_byte(0x11);
            let root_claim = B256::repeat_byte(0x22);
            let game = game_with(7, u32::MAX, 101);
            let view = Arc::new(RecordingL1View {
                game_claim: GameClaim {
                    status: status as u8,
                    deadline: 2_000,
                    parent_index: u32::MAX,
                },
                proof_inputs: ProofInputs {
                    l1_head_number: 20,
                    starting_root,
                    starting_sequence_number: 100,
                    root_claim,
                    sequence_number: 101,
                    ..Default::default()
                },
                ..Default::default()
            });
            let engine = Arc::new(RecordingProofEngine {
                calls: StdMutex::new(Vec::new()),
                proof: vec![0xaa],
                fail,
                cleared: StdMutex::new(Vec::new()),
                cached: StdMutex::new(None),
                generations: std::sync::atomic::AtomicUsize::new(0),
            });
            let actions = Arc::new(RecordingActionExecutor::default());
            let mut proposer = test_proposer().await;
            proposer.l1_view = view;
            proposer.superroot_source = Arc::new(ScriptedSuperRootSource {
                horizon: ProposalHorizon { safe_timestamp: 101, finalized_timestamp: 101 },
                roots: vec![
                    (100, super_root_at_timestamp(100, starting_root, 12, 10)),
                    (101, super_root_at_timestamp(101, root_claim, 12, 10)),
                ],
            });
            proposer.proof_engine = engine.clone();
            proposer.action_executor = actions.clone();
            proposer.state.write().await.games.insert(game.index, game.clone());

            let result = proposer.prove_game(game.address).await;
            assert_eq!(result.is_err(), fail);
            assert!(actions.calls.lock().unwrap().is_empty());
            let expected_clears = if fail { Vec::new() } else { vec![game.address] };
            assert_eq!(*engine.cleared.lock().unwrap(), expected_clears);
        }
    }

    #[tokio::test]
    async fn create_revert_clears_in_flight_while_uncertainty_retains_it() {
        for (failure, should_retain) in
            [(CreateFailure::Reverted, false), (CreateFailure::Uncertain, true)]
        {
            let root = B256::repeat_byte(0x11);
            let mut proposer = test_proposer().await;
            proposer.l1_view = Arc::new(RecordingL1View::default());
            proposer.superroot_source = Arc::new(ScriptedSuperRootSource {
                horizon: ProposalHorizon { safe_timestamp: 100, finalized_timestamp: 100 },
                roots: vec![(100, super_root_at_timestamp(100, root, 12, 11))],
            });
            proposer.action_executor = Arc::new(RecordingActionExecutor {
                create_failure: Some(failure),
                ..Default::default()
            });

            assert!(proposer.handle_game_creation(100, u32::MAX).await.is_err());
            assert_eq!(proposer.in_flight_creation.lock().await.is_some(), should_retain);
        }
    }

    #[tokio::test]
    async fn proof_pre_submit_claim_failure_is_fatal_and_stops_later_reads() {
        let view = Arc::new(RecordingL1View { fail_on: Some("game_claim"), ..Default::default() });
        let mut proposer = test_proposer().await;
        proposer.l1_view = view.clone();
        let game = game_with(1, u32::MAX, 100);
        proposer.state.write().await.games.insert(game.index, game.clone());

        assert!(proposer.pre_submit_checks(game.address).await.is_err());
        assert_eq!(view.block_calls(), vec![("game_claim", BlockId::latest())]);
    }

    #[tokio::test]
    async fn private_constructor_preserves_injected_identity_cache_and_ports() {
        let config = test_config();
        let signer = SignerLock::new(Signer::LocalSigner(PrivateKeySigner::random()));
        let injected_address = Address::left_padding_from(&[0xaa]);
        assert_ne!(injected_address, signer.address());
        let prestates = Arc::new(PrestateCache::new(config.prestates_url.clone()));

        let proposer = Proposer::new_with_dependencies(
            config,
            injected_address,
            Arc::new(RecordingL1View::default()),
            Arc::new(TestQueryTime(1_000)),
            Arc::new(UnavailableSuperRootSource),
            Arc::new(RecordingProofEngine::default()),
            Arc::new(RecordingActionExecutor::default()),
            prestates.clone(),
        )
        .await
        .unwrap();

        assert_eq!(proposer.proposer_address, injected_address);
        assert!(Arc::ptr_eq(&proposer.prestates, &prestates));
        let proof_inputs = proposer.game_proof_inputs(ProofInputs::default(), B256::ZERO);
        assert_eq!(proof_inputs.prover, injected_address);
    }

    #[test]
    fn claim_preflight_degradation_preserves_submit_and_skip_decisions() {
        let withdrawal = WithdrawalState { amount: U256::from(1), timestamp: U256::from(2) };
        let credit_error = ClaimPreflight {
            credit: Err(anyhow::anyhow!("credit unavailable")),
            withdrawal: Ok(withdrawal),
        };
        assert_eq!(classify_claim_preflight(&credit_error), ClaimPreflightDecision::Submit);
        let done = ClaimPreflight {
            credit: Ok(U256::ZERO),
            withdrawal: Ok(WithdrawalState { amount: U256::ZERO, timestamp: U256::ZERO }),
        };
        assert_eq!(classify_claim_preflight(&done), ClaimPreflightDecision::AlreadyClaimed);
        let payout = ClaimPreflight { credit: Ok(U256::ZERO), withdrawal: Ok(withdrawal) };
        assert_eq!(
            classify_claim_preflight(&payout),
            ClaimPreflightDecision::AwaitMaturity { withdrawal_timestamp: U256::from(2) }
        );
    }

    mod ownership {
        use std::collections::HashSet;

        use super::*;

        #[test]
        fn rotation_keeps_old_prestate_games_owned() {
            // Defense, resolution, and claim ownership persists while the
            // old prestate artifacts remain published, regardless of creator.
            let old_prestate = B256::left_padding_from(&[0xde, 0xad]);
            let mut game = game_with(1, u32::MAX, 100);
            game.creator = Address::left_padding_from(&[0xbb]);
            game.absolute_prestate = old_prestate;
            let known = HashSet::from([old_prestate, B256::left_padding_from(&[0x01])]);
            assert!(game.is_owned(&known));
        }

        #[test]
        fn unknown_prestate_is_not_owned_even_for_own_creations() {
            // A signer-created game without loadable prestate artifacts is not
            // owned for defense, resolution, or claims.
            let us = Address::left_padding_from(&[0xaa]);
            let mut game = game_with(1, u32::MAX, 100);
            game.creator = us;
            game.absolute_prestate = B256::left_padding_from(&[0xde, 0xad]);
            assert!(!game.is_owned(&HashSet::new()));
        }
    }

    mod defense {

        use std::sync::atomic::Ordering;

        use alloy_primitives::Bytes;
        use alloy_sol_types::SolValue;

        use super::*;
        use crate::proving::GameUnprovable;

        pub(super) async fn plan_defenses(
            proposer: &Proposer,
        ) -> anyhow::Result<Vec<OperationSummary>> {
            let mut planned = Vec::new();
            let mut active = active_task_keys(proposer).await;
            proposer.plan_game_defense_tasks(&mut planned, &mut active).await?;
            Ok(planned)
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
        async fn should_skip_proving_honors_undefendable_set() {
            let proposer = test_proposer().await;
            let game = Address::left_padding_from(&[0xcc]);
            proposer.undefendable.lock().await.insert(game);
            // Arm (e) fires before any chain read: no RPC needed.
            assert!(proposer.should_skip_proving(game, u64::MAX, true).await.unwrap());
        }

        #[tokio::test]
        async fn game_standing_rejects_blacklisted_or_retired() {
            let asserter = Asserter::new();
            let yes: Bytes = true.abi_encode().into();
            let no: Bytes = false.abi_encode().into();
            asserter.push_success(&yes);
            asserter.push_success(&no);
            let provider = ProviderBuilder::default().connect_mocked_client(asserter.clone());
            let factory = DisputeGameFactory::new(Address::ZERO, provider.clone());
            let view =
                ProductionL1View::new(provider, factory, "http://127.0.0.1:1".parse().unwrap());
            let game_address = Address::left_padding_from(&[0x11]);
            let registry_address = Address::left_padding_from(&[0x22]);

            assert!(view.game_standing(game_address, registry_address).await.unwrap().disallowed());

            asserter.push_success(&no);
            asserter.push_success(&yes);
            assert!(view.game_standing(game_address, registry_address).await.unwrap().disallowed());

            asserter.push_success(&no);
            asserter.push_success(&no);
            assert!(
                !view.game_standing(game_address, registry_address).await.unwrap().disallowed()
            );
        }

        #[tokio::test]
        async fn latest_l1_timestamp_comes_from_head() {
            let asserter = Asserter::new();
            let mut block = Block::<alloy_rpc_types_eth::Transaction>::default();
            block.header.timestamp = 1_234;
            asserter.push_success(&block);
            let provider = ProviderBuilder::default().connect_mocked_client(asserter);
            let factory = DisputeGameFactory::new(Address::ZERO, provider.clone());
            let view =
                ProductionL1View::new(provider, factory, "http://127.0.0.1:1".parse().unwrap());

            assert_eq!(view.latest_l1_timestamp().await.unwrap(), 1_234);
        }

        #[tokio::test]
        async fn unprovable_result_invalidates_cached_subtree() {
            let mut proposer = test_proposer().await;
            let proof_engine = Arc::new(RecordingProofEngine::default());
            proposer.proof_engine = proof_engine.clone();
            let root = game_with(1, u32::MAX, 100);
            let child = game_with(2, 1, 200);
            let sibling = game_with(3, u32::MAX, 150);
            {
                let mut state = proposer.state.write().await;
                state.games = [root.clone(), child.clone(), sibling.clone()]
                    .into_iter()
                    .map(|game| (game.index, game))
                    .collect();
            }
            proposer
                .last_created_game_l2_sequence_number
                .store(child.l2_sequence_number, Ordering::Relaxed);
            *proposer.last_created_game_address.lock().await = child.address;

            let result = Err(anyhow::Error::new(GameUnprovable("trusted mismatch".into())));
            proposer.handle_game_proving_result(root.index, root.address, result).await.unwrap();

            assert!(proposer.undefendable.lock().await.contains(&root.address));
            let state = proposer.state.read().await;
            assert_eq!(state.invalid_games, HashSet::from([root.index, child.index]));
            assert!(!state.games.contains_key(&root.index));
            assert!(!state.games.contains_key(&child.index));
            assert!(state.games.contains_key(&sibling.index));
            drop(state);
            assert_eq!(proposer.last_created_game_l2_sequence_number.load(Ordering::Relaxed), 0);
            assert_eq!(*proposer.last_created_game_address.lock().await, Address::ZERO);
            let cleared =
                proof_engine.cleared.lock().unwrap().iter().copied().collect::<HashSet<_>>();
            assert_eq!(cleared, HashSet::from([root.address, child.address]));
        }

        #[tokio::test]
        async fn stale_unprovable_result_does_not_invalidate_replacement() {
            let mut proposer = test_proposer().await;
            let proof_engine = Arc::new(RecordingProofEngine::default());
            proposer.proof_engine = proof_engine.clone();
            let old_game = game_with(1, u32::MAX, 100);
            let mut replacement = game_with(1, u32::MAX, 101);
            replacement.address = Address::left_padding_from(&[0xaa]);
            let child = game_with(2, 1, 200);
            {
                let mut state = proposer.state.write().await;
                state.games = [replacement.clone(), child.clone()]
                    .into_iter()
                    .map(|game| (game.index, game))
                    .collect();
            }

            let result = Err(anyhow::Error::new(GameUnprovable("trusted mismatch".into())));
            proposer
                .handle_game_proving_result(old_game.index, old_game.address, result)
                .await
                .unwrap();

            let state = proposer.state.read().await;
            assert!(state.games.contains_key(&replacement.index));
            assert!(state.games.contains_key(&child.index));
            assert!(state.invalid_games.is_empty());
            assert_eq!(*proof_engine.cleared.lock().unwrap(), vec![old_game.address]);
        }

        #[tokio::test]
        async fn transient_error_retains_progress_only_while_game_can_retry() {
            let mut proposer = test_proposer().await;
            let proof_engine = Arc::new(RecordingProofEngine::default());
            proposer.proof_engine = proof_engine.clone();
            let active = game_with(1, u32::MAX, 100);
            proposer.state.write().await.games.insert(active.index, active.clone());

            let result = Err(anyhow::anyhow!("transient proving error"));
            assert!(
                proposer
                    .handle_game_proving_result(active.index, active.address, result)
                    .await
                    .is_err()
            );
            assert!(proof_engine.cleared.lock().unwrap().is_empty());

            proposer.state.write().await.games.remove(&active.index);
            let result = Err(anyhow::anyhow!("transient proving error"));
            assert!(
                proposer
                    .handle_game_proving_result(active.index, active.address, result)
                    .await
                    .is_err()
            );

            let mut terminal = game_with(2, u32::MAX, 101);
            terminal.proposal_status = ProposalStatus::Resolved;
            proposer.state.write().await.games.insert(terminal.index, terminal.clone());
            let result = Err(anyhow::anyhow!("transient proving error"));
            assert!(
                proposer
                    .handle_game_proving_result(terminal.index, terminal.address, result)
                    .await
                    .is_err()
            );
            assert_eq!(
                *proof_engine.cleared.lock().unwrap(),
                vec![active.address, terminal.address]
            );
        }

        #[tokio::test]
        async fn should_skip_proving_uses_l1_time_and_strict_expiry() {
            let asserter = Asserter::new();
            let provider = ProviderBuilder::default().connect_mocked_client(asserter.clone());
            let mut proposer = test_proposer_with_provider(provider).await;
            let proof_engine = Arc::new(RecordingProofEngine::default());
            proposer.proof_engine = proof_engine.clone();
            proposer.max_prove_duration.set(7200).unwrap();
            let game = Address::left_padding_from(&[0xdd]);

            push_claim_error_and_head(&asserter, 1_000);
            assert!(
                !proposer.should_skip_proving(game, 1_000, true).await.unwrap(),
                "deadline equality is not expired"
            );

            push_claim_error_and_head(&asserter, 1_001);
            assert!(
                proposer.should_skip_proving(game, 1_000, true).await.unwrap(),
                "L1 timestamp past the deadline must skip"
            );
            assert_eq!(*proof_engine.cleared.lock().unwrap(), vec![game]);

            push_claim_error_and_head(&asserter, 1_000);
            assert!(
                !proposer.should_skip_proving(game, 100_000, true).await.unwrap(),
                "distant deadline must proceed"
            );
        }

        #[tokio::test]
        async fn pre_submit_checks_skip_evicted_games() {
            let proposer = test_proposer().await;
            // A game evicted after a parent loss is never submitted.
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
                (1000, 1000, DeadlineStatus::Approaching { hours_remaining: 0.0 }),
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
        async fn defense_planning_skips_unknown_prestate_games() {
            let proposer = test_proposer().await;
            let mut challenged = game_with(1, u32::MAX, 100);
            challenged.proposal_status = ProposalStatus::Challenged;
            challenged.deadline = u64::MAX;
            proposer.state.write().await.games.insert(challenged.index, challenged);

            assert!(plan_defenses(&proposer).await.unwrap().is_empty());
        }

        #[tokio::test]
        async fn defense_planning_skips_blacklisted_games() {
            let asserter = Asserter::new();
            let provider = ProviderBuilder::default().connect_mocked_client(asserter.clone());
            let proposer = test_proposer_with_provider(provider).await;
            proposer.max_prove_duration.set(7200).unwrap();
            let prestate = B256::left_padding_from(&[0x77]);
            proposer
                .prestates
                .insert_for_tests(
                    prestate,
                    PrestatePrograms { aggregation_elf: vec![1], range_elf: vec![1] },
                )
                .await;
            let mut game = game_with(1, u32::MAX, 100);
            game.proposal_status = ProposalStatus::Challenged;
            game.deadline = u64::MAX;
            game.absolute_prestate = prestate;
            proposer.state.write().await.games.insert(game.index, game);

            push_claim_error_and_default_head(&asserter);
            let yes: Bytes = true.abi_encode().into();
            let no: Bytes = false.abi_encode().into();
            asserter.push_success(&yes);
            asserter.push_success(&no);

            assert!(plan_defenses(&proposer).await.unwrap().is_empty());
        }

        /// Network mode keeps creation closed until key setup verifies the
        /// prestate. Failed setup poisons the entry and leaves the gate closed.
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

        /// Prestate rotation pauses creation until the new artifacts exist
        /// while games on the old prestate remain owned.
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

    mod fast_finality {
        use super::*;
        use alloy_primitives::Bytes;
        use alloy_sol_types::SolValue;

        fn ff_config(limit: u64) -> ProposerConfig {
            let mut config = test_config();
            config.fast_finality_mode = true;
            config.fast_finality_proving_limit = std::num::NonZeroU64::new(limit).unwrap();
            config
        }

        async fn insert_prestate(proposer: &Proposer, prestate: B256) {
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
        }

        async fn plan_creation(
            proposer: &Proposer,
        ) -> (anyhow::Result<(bool, u64, u32)>, Vec<OperationSummary>) {
            let mut planned = Vec::new();
            let mut active = active_task_keys(proposer).await;
            let decision = proposer.plan_game_creation_decision(&mut planned, &mut active).await;
            (decision, planned)
        }

        #[test]
        fn fast_finality_candidates_filters_and_sorts() {
            let mut unchallenged_late = game_with(1, u32::MAX, 100);
            unchallenged_late.deadline = 500;
            let mut unchallenged_early = game_with(2, 1, 200);
            unchallenged_early.deadline = 100;
            // Excluded: challenged, proven, and resolved games.
            let mut challenged = game_with(3, 2, 300);
            challenged.proposal_status = ProposalStatus::Challenged;
            let mut proven = game_with(4, 3, 400);
            proven.proposal_status = ProposalStatus::UnchallengedAndValidProofProvided;
            let mut resolved = game_with(5, 4, 500);
            resolved.status = GameStatus::DefenderWins;

            let state = ProposerState {
                games: [unchallenged_late, unchallenged_early, challenged, proven, resolved]
                    .into_iter()
                    .map(|game| (game.index, game))
                    .collect(),
                ..Default::default()
            };

            let candidates = state.fast_finality_candidates();
            // Deadline-ascending: the game closest to its challenge deadline first.
            assert_eq!(
                candidates.iter().map(|(index, ..)| *index).collect::<Vec<_>>(),
                vec![U256::from(2), U256::from(1)]
            );
        }

        #[tokio::test]
        async fn fast_finality_capacity_gate_skips_creation() {
            for purpose in [ProvingPurpose::Defense, ProvingPurpose::FastFinality] {
                let proposer = test_proposer_with(ff_config(1)).await;
                insert_task(
                    &proposer,
                    OperationSummary::ProveGame {
                        factory_index: U256::ZERO,
                        address: Address::left_padding_from(&[0xaa]),
                        purpose,
                    },
                )
                .await;

                let (decision, planned) = plan_creation(&proposer).await;
                let (should_create, _, _) = decision.unwrap();

                assert!(!should_create, "purpose: {purpose:?}");
                assert!(planned.is_empty(), "purpose: {purpose:?}");
            }
        }

        #[tokio::test]
        async fn fast_finality_plans_nearest_deadlines_up_to_limit() {
            let asserter = Asserter::new();
            let provider = ProviderBuilder::default().connect_mocked_client(asserter.clone());
            let proposer = test_proposer_with_config_and_provider(ff_config(3), provider).await;
            proposer.max_challenge_duration.set(7200).unwrap();
            let prestate = B256::left_padding_from(&[0x88]);
            insert_prestate(&proposer, prestate).await;

            for (index, deadline) in [(1u64, 30_000), (2, 10_000), (3, 40_000), (4, 20_000)] {
                let mut game = game_with(index, u32::MAX, 100 * index);
                game.deadline = deadline;
                game.absolute_prestate = prestate;
                game.creator = proposer.proposer_address;
                proposer.state.write().await.games.insert(game.index, game);
            }

            let no: Bytes = false.abi_encode().into();
            for _ in 0..3 {
                push_claim_error_and_default_head(&asserter);
                asserter.push_success(&no);
                asserter.push_success(&no);
            }

            let (decision, planned) = plan_creation(&proposer).await;
            let (should_create, _, _) = decision.unwrap();
            assert!(!should_create);
            let proving = planned
                .iter()
                .filter_map(|operation| match operation {
                    OperationSummary::ProveGame {
                        address,
                        purpose: ProvingPurpose::FastFinality,
                        ..
                    } => Some(*address),
                    _ => None,
                })
                .collect::<HashSet<_>>();
            let expected = [2u8, 4, 1]
                .map(|index| Address::left_padding_from(&[index]))
                .into_iter()
                .collect::<HashSet<_>>();
            assert_eq!(proving, expected);
        }

        #[tokio::test]
        async fn fast_finality_planning_skips_foreign_creator() {
            let asserter = Asserter::new();
            let provider = ProviderBuilder::default().connect_mocked_client(asserter.clone());
            let proposer = test_proposer_with_config_and_provider(ff_config(1), provider).await;
            proposer.max_challenge_duration.set(7200).unwrap();
            let prestate = B256::left_padding_from(&[0x88]);
            insert_prestate(&proposer, prestate).await;

            let mut game = game_with(1, u32::MAX, 100);
            game.deadline = 100_000;
            game.absolute_prestate = prestate;
            game.creator = Address::left_padding_from(&[0xff]);
            proposer.state.write().await.games.insert(game.index, game);

            push_claim_error_and_default_head(&asserter);
            let (_, planned) = plan_creation(&proposer).await;
            assert!(planned.is_empty());
        }

        #[tokio::test]
        async fn fast_finality_planning_skips_blacklisted_game() {
            let asserter = Asserter::new();
            let provider = ProviderBuilder::default().connect_mocked_client(asserter.clone());
            let proposer = test_proposer_with_config_and_provider(ff_config(1), provider).await;
            proposer.max_challenge_duration.set(7200).unwrap();
            let prestate = B256::left_padding_from(&[0x88]);
            insert_prestate(&proposer, prestate).await;

            let mut game = game_with(1, u32::MAX, 100);
            game.deadline = 100_000;
            game.absolute_prestate = prestate;
            game.creator = proposer.proposer_address;
            proposer.state.write().await.games.insert(game.index, game);

            push_claim_error_and_default_head(&asserter);
            let yes: Bytes = true.abi_encode().into();
            let no: Bytes = false.abi_encode().into();
            asserter.push_success(&yes);
            asserter.push_success(&no);
            let (_, planned) = plan_creation(&proposer).await;
            assert!(planned.is_empty());
        }

        #[tokio::test]
        async fn fast_finality_planning_filters_candidates() {
            let asserter = Asserter::new();
            let provider = ProviderBuilder::default().connect_mocked_client(asserter.clone());
            let proposer = test_proposer_with_config_and_provider(ff_config(8), provider).await;
            proposer.max_challenge_duration.set(7200).unwrap();
            let prestate = B256::left_padding_from(&[0x88]);
            insert_prestate(&proposer, prestate).await;

            // Unknown prestate: not ours to prove.
            let mut foreign_prestate = game_with(1, u32::MAX, 100);
            foreign_prestate.deadline = u64::MAX;
            foreign_prestate.absolute_prestate = B256::left_padding_from(&[0x99]);
            foreign_prestate.creator = proposer.proposer_address;
            // An in-flight proving task exercises per-game dedup.
            let mut in_flight = game_with(2, 1, 200);
            in_flight.deadline = u64::MAX;
            in_flight.absolute_prestate = prestate;
            in_flight.creator = proposer.proposer_address;
            // Challenged games belong to the defense scan.
            let mut challenged = game_with(3, 2, 300);
            challenged.proposal_status = ProposalStatus::Challenged;
            challenged.deadline = u64::MAX;
            challenged.absolute_prestate = prestate;
            challenged.creator = proposer.proposer_address;
            let mut eligible = game_with(4, 3, 400);
            eligible.deadline = u64::MAX;
            eligible.absolute_prestate = prestate;
            eligible.creator = proposer.proposer_address;

            let in_flight_address = in_flight.address;
            let eligible_address = eligible.address;
            for game in [foreign_prestate, in_flight, challenged, eligible] {
                proposer.state.write().await.games.insert(game.index, game);
            }
            insert_task(
                &proposer,
                OperationSummary::ProveGame {
                    factory_index: U256::ZERO,
                    address: in_flight_address,
                    purpose: ProvingPurpose::FastFinality,
                },
            )
            .await;

            push_claim_error_and_default_head(&asserter);
            let no: Bytes = false.abi_encode().into();
            asserter.push_success(&no);
            asserter.push_success(&no);
            let (_, planned) = plan_creation(&proposer).await;

            let proving = planned
                .iter()
                .filter_map(|operation| match operation {
                    OperationSummary::ProveGame { address, .. } => Some(*address),
                    _ => None,
                })
                .collect::<HashSet<_>>();
            assert_eq!(proving, HashSet::from([eligible_address]));
        }

        #[test]
        fn awaiting_proof_status_table() {
            assert!(awaiting_proof(ProposalStatus::Unchallenged));
            assert!(awaiting_proof(ProposalStatus::Challenged));
            assert!(!awaiting_proof(ProposalStatus::UnchallengedAndValidProofProvided));
            assert!(!awaiting_proof(ProposalStatus::ChallengedAndValidProofProvided));
            assert!(!awaiting_proof(ProposalStatus::Resolved));
        }

        #[tokio::test]
        async fn should_skip_proving_selects_duration_by_task_kind() {
            let asserter = Asserter::new();
            let provider = ProviderBuilder::default().connect_mocked_client(asserter.clone());
            let proposer = test_proposer_with_provider(provider).await;
            // Only the fast-finality duration is set: the defense arm must
            // error on its missing cell, proving the selection wiring.
            proposer.max_challenge_duration.set(7200).unwrap();
            let game = Address::left_padding_from(&[0xab]);
            push_claim_error_and_default_head(&asserter);
            assert!(!proposer.should_skip_proving(game, 100_000, false).await.unwrap());
            push_claim_error_and_default_head(&asserter);
            assert!(proposer.should_skip_proving(game, 100_000, true).await.is_err());
        }

        #[tokio::test]
        async fn defense_planning_skips_game_with_inflight_fast_finality_task() {
            let proposer = test_proposer().await;
            proposer.max_prove_duration.set(7200).unwrap();
            let prestate = B256::left_padding_from(&[0x88]);
            insert_prestate(&proposer, prestate).await;
            // A challenged game with a far deadline and a known prestate:
            // the per-game dedup is the ONLY guard between it and a defense
            // spawn (a deadline of 0 would deadline-skip and make this pin
            // vacuous).
            let mut challenged = game_with(1, u32::MAX, 100);
            challenged.proposal_status = ProposalStatus::Challenged;
            challenged.deadline = u64::MAX;
            challenged.absolute_prestate = prestate;
            let game_address = challenged.address;
            proposer.state.write().await.games.insert(challenged.index, challenged);
            insert_task(
                &proposer,
                OperationSummary::ProveGame {
                    factory_index: U256::ZERO,
                    address: game_address,
                    purpose: ProvingPurpose::FastFinality,
                },
            )
            .await;

            assert!(super::defense::plan_defenses(&proposer).await.unwrap().is_empty());
            assert_eq!(proposer.tasks.lock().await.len(), 1, "no second task for the game");
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
            // A zero retry window isolates immediate self-healing from
            // negative-cache pacing.
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

            // The retry window prevents a refetch after artifacts are published.
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

        /// With `KONA_SP1_ELF_DIR` set, verifies the built aggregation vkey
        /// against the canonical prestate in `vkeys.toml`. Skipped when the
        /// environment variable is absent.
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

            // Identical published bytes do not clear the poisoned verdict.
            assert!(!cache.ensure_loaded(hash).await);
            assert!(!cache.known_prestates().await.contains(&hash));

            // Changed artifacts replace the entry and re-arm key setup.
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

        let removed = s.invalidate_subtree(U256::from(1)).into_iter().collect::<HashSet<_>>();

        assert_eq!(s.invalid_games, HashSet::from([U256::from(1), U256::from(2)]));
        assert!(s.games.contains_key(&U256::from(0)));
        assert!(s.games.contains_key(&U256::from(3)));
        assert!(!s.games.contains_key(&U256::from(1)));
        assert!(!s.games.contains_key(&U256::from(2)));
        assert_eq!(
            removed,
            HashSet::from([Address::left_padding_from(&[1]), Address::left_padding_from(&[2]),])
        );
    }

    #[test]
    fn factory_cache_reset_forgets_prior_history() {
        let anchor = game_with(0, u32::MAX, 100);
        let mut s = state(vec![anchor.clone(), game_with(1, 0, 200)], Some(anchor));
        s.cursor = Cursor::from(U256::from(1));
        s.canonical_head_index = Some(U256::from(1));
        s.canonical_head_sequence_number = Some(200);
        s.invalid_games.insert(U256::from(1));

        let removed = s.reset_factory_cache().into_iter().collect::<HashSet<_>>();

        assert_eq!(s.cursor, Cursor::none());
        assert!(s.games.is_empty());
        assert!(s.invalid_games.is_empty());
        assert!(s.anchor_game.is_none());
        assert!(s.canonical_head_index.is_none());
        assert_eq!(s.canonical_head_sequence_number, Some(200));
        assert_eq!(
            removed,
            HashSet::from([Address::left_padding_from(&[0]), Address::left_padding_from(&[1]),])
        );
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
