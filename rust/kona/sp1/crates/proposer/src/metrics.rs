//! Prometheus gauge definitions for the proposer.
//!
//! Trimmed from op-succinct's `fault-proof/src/prometheus.rs` (@ 13716c2c):
//! proving/defense gauges arrive with the defend path (#21463); the
//! challenger gauges are not ported (op-challenger owns that role). Gauge
//! names are renamed `kona_sp1_proposer_*` and speak super-root sequence
//! numbers instead of L2 block numbers.

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
    /// Total number of games whose super-root data could not be fetched or
    /// validated during sync (held as pending, re-checked next cycle).
    #[strum(
        serialize = "kona_sp1_proposer_super_root_unavailable",
        message = "Total number of games whose super-root data was unavailable during sync"
    )]
    SuperRootUnavailable,
    /// Total number of per-game sync failures (fetch or status read) that
    /// were contained without aborting the sync cycle.
    #[strum(
        serialize = "kona_sp1_proposer_game_sync_error",
        message = "Total number of contained per-game sync failures"
    )]
    GameSyncError,
}

impl MetricsGauge for ProposerGauge {}
