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
    #[strum(
        serialize = "kona_sp1_proposer_max_proposable_sequence_number",
        message = "Highest super-root timestamp proposable under the configured safety level"
    )]
    MaxProposableSequenceNumber,
    #[strum(
        serialize = "kona_sp1_proposer_latest_game_l2_sequence_number",
        message = "Latest game L2 sequence number (super-root timestamp)"
    )]
    LatestGameL2SequenceNumber,
    #[strum(
        serialize = "kona_sp1_proposer_anchor_game_l2_sequence_number",
        message = "Anchor game L2 sequence number (super-root timestamp)"
    )]
    AnchorGameL2SequenceNumber,
    #[strum(
        serialize = "kona_sp1_proposer_canonical_head_game_index",
        message = "Canonical head game index (-1 when cleared)"
    )]
    CanonicalHeadGameIndex,
    #[strum(
        serialize = "kona_sp1_proposer_anchor_game_index",
        message = "Anchor game index (-1 when cleared)"
    )]
    AnchorGameIndex,
    #[strum(
        serialize = "kona_sp1_proposer_games_created",
        message = "Total number of games created by the proposer"
    )]
    GamesCreated,
    #[strum(
        serialize = "kona_sp1_proposer_games_resolved",
        message = "Total number of games resolved by the proposer"
    )]
    GamesResolved,
    #[strum(
        serialize = "kona_sp1_proposer_games_bonds_claimed",
        message = "Total number of games that bonds were claimed by the proposer"
    )]
    GamesBondsClaimed,
    // Error metrics
    #[strum(
        serialize = "kona_sp1_proposer_game_creation_error",
        message = "Total number of game creation errors encountered by the proposer"
    )]
    GameCreationError,
    #[strum(
        serialize = "kona_sp1_proposer_game_resolution_error",
        message = "Total number of game resolution errors encountered by the proposer"
    )]
    GameResolutionError,
    #[strum(
        serialize = "kona_sp1_proposer_bond_claiming_error",
        message = "Total number of bond claiming errors encountered by the proposer"
    )]
    BondClaimingError,
    #[strum(
        serialize = "kona_sp1_proposer_metrics_error",
        message = "Total number of metrics errors encountered by the proposer"
    )]
    MetricsError,
    // Backup metrics
    #[strum(
        serialize = "kona_sp1_proposer_backup_save_success",
        message = "Total number of successful backup saves"
    )]
    BackupSaveSuccess,
    #[strum(
        serialize = "kona_sp1_proposer_backup_save_error",
        message = "Total number of backup save errors"
    )]
    BackupSaveError,
    #[strum(
        serialize = "kona_sp1_proposer_backup_restore_success",
        message = "Total number of successful backup restores"
    )]
    BackupRestoreSuccess,
    #[strum(
        serialize = "kona_sp1_proposer_backup_restore_error",
        message = "Total number of backup restore errors"
    )]
    BackupRestoreError,
}

impl MetricsGauge for ProposerGauge {}
