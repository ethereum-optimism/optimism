//! Prometheus gauges for the proposer.

use kona_sp1_host_utils::metrics::MetricsGauge;
use strum::EnumMessage;
use strum_macros::{Display, EnumIter};

/// All proposer metrics gauges.
#[derive(Debug, Clone, Copy, Display, EnumIter, EnumMessage)]
pub enum ProposerGauge {
    // Proposer metrics
    /// Highest super-root timestamp proposable under the configured safety level.
    #[strum(
        serialize = "kona_sp1_proposer_max_proposable_sequence_number",
        message = "Highest super-root timestamp proposable under the configured safety level"
    )]
    MaxProposableSequenceNumber,
    /// Super-root timestamp of the latest game created by the proposer.
    #[strum(
        serialize = "kona_sp1_proposer_latest_game_l2_sequence_number",
        message = "Latest game L2 sequence number (super-root timestamp)"
    )]
    LatestGameL2SequenceNumber,
    /// Super-root timestamp of the current anchor game.
    #[strum(
        serialize = "kona_sp1_proposer_anchor_game_l2_sequence_number",
        message = "Anchor game L2 sequence number (super-root timestamp)"
    )]
    AnchorGameL2SequenceNumber,
    /// Factory index of the canonical head game (-1 when cleared).
    #[strum(
        serialize = "kona_sp1_proposer_canonical_head_game_index",
        message = "Canonical head game index (-1 when cleared)"
    )]
    CanonicalHeadGameIndex,
    /// Factory index of the current anchor game (-1 when cleared).
    #[strum(
        serialize = "kona_sp1_proposer_anchor_game_index",
        message = "Anchor game index (-1 when cleared)"
    )]
    AnchorGameIndex,
    /// Total number of games created by the proposer.
    #[strum(
        serialize = "kona_sp1_proposer_games_created",
        message = "Total number of games created by the proposer"
    )]
    GamesCreated,
    /// Total number of games resolved by the proposer.
    #[strum(
        serialize = "kona_sp1_proposer_games_resolved",
        message = "Total number of games resolved by the proposer"
    )]
    GamesResolved,
    /// Total number of games whose bonds were claimed by the proposer.
    #[strum(
        serialize = "kona_sp1_proposer_games_bonds_claimed",
        message = "Total number of games that bonds were claimed by the proposer"
    )]
    GamesBondsClaimed,
    // Error metrics
    /// Total number of game creation errors encountered by the proposer.
    #[strum(
        serialize = "kona_sp1_proposer_game_creation_error",
        message = "Total number of game creation errors encountered by the proposer"
    )]
    GameCreationError,
    /// Total number of game resolution errors encountered by the proposer.
    #[strum(
        serialize = "kona_sp1_proposer_game_resolution_error",
        message = "Total number of game resolution errors encountered by the proposer"
    )]
    GameResolutionError,
    /// Total number of bond claiming errors encountered by the proposer.
    #[strum(
        serialize = "kona_sp1_proposer_bond_claiming_error",
        message = "Total number of bond claiming errors encountered by the proposer"
    )]
    BondClaimingError,
    /// Total number of metrics reporting errors encountered by the proposer.
    #[strum(
        serialize = "kona_sp1_proposer_metrics_error",
        message = "Total number of metrics errors encountered by the proposer"
    )]
    MetricsError,
    /// Total number of creation cycles skipped because the registered
    /// prestate is not in the known set.
    #[strum(
        serialize = "kona_sp1_proposer_unknown_registered_prestate",
        message = "Total number of creation cycles skipped on an unknown registered prestate"
    )]
    UnknownRegisteredPrestate,
    /// Total number of times a game's super-root data could not be fetched
    /// or validated, either during sync (game held as pending) or during a
    /// defense task (task fails and retries next cycle).
    #[strum(
        serialize = "kona_sp1_proposer_super_root_unavailable",
        message = "Total number of times a game's super-root data was unavailable (sync or defense)"
    )]
    SuperRootUnavailable,
    /// Total number of per-game sync failures. A discovery (fetch) failure
    /// aborts the sync cycle; status-read failures are contained to the
    /// affected game.
    #[strum(
        serialize = "kona_sp1_proposer_game_sync_error",
        message = "Total number of per-game sync failures"
    )]
    GameSyncError,
    /// Highest factory index processed by the sync walk (-1 before the
    /// first complete walk). A flat value while the factory's latest index
    /// grows means discovery is stuck (see `game_sync_error`).
    #[strum(
        serialize = "kona_sp1_proposer_sync_cursor",
        message = "Highest factory index processed by the sync walk (-1 when unset)"
    )]
    SyncCursor,
    /// Latest game index observed on the factory at the pinned block (-1
    /// when the factory has no games).
    #[strum(
        serialize = "kona_sp1_proposer_factory_latest_game_index",
        message = "Latest game index on the factory at the pinned block (-1 when empty)"
    )]
    FactoryLatestGameIndex,
    // Defense metrics
    /// Total number of games proven by the proposer.
    #[strum(
        serialize = "kona_sp1_proposer_games_proven",
        message = "Total number of games proven by the proposer"
    )]
    GamesProven,
    /// Total number of defense proving tasks spawned.
    #[strum(
        serialize = "kona_sp1_proposer_games_defense_spawned",
        message = "Total number of defense proving tasks spawned"
    )]
    GamesDefenseSpawned,
    /// Duration of the most recent successful game proving run, in seconds.
    #[strum(
        serialize = "kona_sp1_proposer_proving_duration_seconds",
        message = "Duration of the most recent successful game proving run in seconds"
    )]
    ProvingDurationSeconds,
    /// Total number of times a prove deadline was observed approaching
    /// (within half of `maxProveDuration`).
    #[strum(
        serialize = "kona_sp1_proposer_deadline_approaching",
        message = "Total number of approaching-deadline observations"
    )]
    DeadlineApproaching,
    // Defense error metrics. In network mode a persistently failing game
    // re-purchases its full proof set on every retry until the prove
    // deadline expires: a sustained non-zero rate on these gauges is a
    // spend alarm, not a transient.
    /// Total number of game proving task failures.
    #[strum(
        serialize = "kona_sp1_proposer_game_proving_error",
        message = "Total number of game proving task failures"
    )]
    GameProvingError,
    /// Total number of proof requests abandoned after exceeding the overall
    /// proving timeout.
    #[strum(
        serialize = "kona_sp1_proposer_proving_timeout_error",
        message = "Total number of proof requests that exceeded the proving timeout"
    )]
    ProvingTimeoutError,
    /// Total number of proof requests cancelled because no prover picked
    /// them up within the auction timeout (mainnet only).
    #[strum(
        serialize = "kona_sp1_proposer_auction_timeout_error",
        message = "Total number of proof requests cancelled on auction timeout"
    )]
    AuctionTimeoutError,
    /// Total number of proof requests that exceeded their server-side
    /// deadline.
    #[strum(
        serialize = "kona_sp1_proposer_deadline_exceeded_error",
        message = "Total number of proof requests past their server-side deadline"
    )]
    DeadlineExceededError,
    /// Total number of SP1 network API calls that hit the per-call timeout.
    #[strum(
        serialize = "kona_sp1_proposer_network_call_timeout",
        message = "Total number of SP1 network API call timeouts"
    )]
    NetworkCallTimeout,
    /// Total number of challenged games skipped because their prestate is
    /// unknown (artifacts not loadable) or poisoned.
    #[strum(
        serialize = "kona_sp1_proposer_unknown_prestate_challenged",
        message = "Total number of challenged games with an unknown or poisoned prestate"
    )]
    UnknownPrestateChallenged,
    /// Total number of prestates whose aggregation ELF failed verification
    /// against the on-chain prestate hash during proving-key setup.
    #[strum(
        serialize = "kona_sp1_proposer_prestate_vkey_mismatch",
        message = "Total number of prestates failing vkey verification"
    )]
    PrestateVkeyMismatch,
    /// Total number of games found permanently unprovable (claim data
    /// diverged or required L1 beyond the game's L1 head).
    #[strum(
        serialize = "kona_sp1_proposer_game_unprovable",
        message = "Total number of permanently unprovable games"
    )]
    GameUnprovable,
}

impl MetricsGauge for ProposerGauge {}
