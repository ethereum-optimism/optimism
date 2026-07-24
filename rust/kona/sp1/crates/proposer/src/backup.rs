//! Simple file-based state persistence for proposer recovery.
//!
//! On restart, the proposer can restore its cursor and game cache from a backup file,
//! avoiding a full re-sync from the factory contract.

use std::{io::Write, path::Path};

use tempfile::NamedTempFile;

use alloy_primitives::{Address, U256};
use anyhow::{Context, Result, bail};
use serde::{Deserialize, Serialize};

use crate::proposer::Game;

/// Current backup format version. Increment when making breaking changes.
pub const BACKUP_VERSION: u32 = 1;

/// Serializable backup of the proposer state.
#[derive(Debug, Serialize, Deserialize)]
pub struct ProposerBackup {
    /// Backup format version this file was written with; must equal [`BACKUP_VERSION`] to load.
    pub version: u32,
    /// Factory game index the sync loop has processed up to, if any games were seen.
    pub cursor: Option<U256>,
    /// Cached games tracked by the proposer at backup time.
    pub games: Vec<Game>,
    /// Factory index of the anchor game the cache was seeded from, if known.
    pub anchor_game_index: Option<U256>,
    /// L2 block of the most recently created game. Prevents duplicate creation after
    /// restart when the pinned sync cache hasn't caught up. Defaults to 0 for backups
    /// created before this field existed.
    #[serde(default)]
    pub last_created_game_l2_sequence_number: u64,
    /// Address of the most recently created game. Used for precise `ChallengerWins`
    /// guard reset. Defaults to `Address::ZERO` (no guard) for old backups.
    #[serde(default)]
    pub last_created_game_address: Address,
    /// Games whose timestamps were not yet safe from this node's view at
    /// backup time. Must survive restarts: the restored cursor is already
    /// past them, so losing this set would make them permanently invisible.
    #[serde(default)]
    pub pending_games: Vec<U256>,
}

impl ProposerBackup {
    /// Create a new backup with the current version.
    pub const fn new(
        cursor: Option<U256>,
        games: Vec<Game>,
        anchor_game_index: Option<U256>,
    ) -> Self {
        Self {
            version: BACKUP_VERSION,
            cursor,
            games,
            anchor_game_index,
            last_created_game_l2_sequence_number: 0,
            last_created_game_address: Address::ZERO,
            pending_games: Vec::new(),
        }
    }

    /// Validate backup integrity. Rejects stale/corrupted backups but allows orphaned parent
    /// references, which are normal when anchor-based fetching or ASR filtering produce partial
    /// DAGs.
    pub fn validate(&self) -> Result<()> {
        // Cursor with no games indicates a stale or corrupted backup.
        if let Some(cursor) = self.cursor &&
            self.games.is_empty() &&
            cursor > U256::ZERO
        {
            bail!("cursor exists but no games");
        }

        // Anchor must reference a game that exists in the backup.
        if let Some(anchor_idx) = self.anchor_game_index &&
            !self.games.iter().any(|g| g.index == anchor_idx)
        {
            bail!("anchor game index references non-existent game");
        }

        Ok(())
    }

    /// Save the backup to a file as JSON (atomic via temp file + rename with fsync).
    pub fn save(&self, path: &Path) -> Result<()> {
        let json =
            serde_json::to_string_pretty(self).context("failed to serialize proposer backup")?;

        let dir = path.parent().unwrap_or_else(|| Path::new("."));
        let mut temp =
            NamedTempFile::new_in(dir).context("failed to create proposer backup temp file")?;
        temp.write_all(json.as_bytes()).context("failed to write proposer backup temp file")?;
        temp.as_file().sync_all().context("failed to sync proposer backup temp file")?;
        temp.persist(path).context("failed to persist proposer backup file")?;

        tracing::debug!(?path, games = self.games.len(), "Proposer state backed up");
        Ok(())
    }

    /// Load and validate a backup from file.
    ///
    /// Returns None and logs a warning if:
    /// - File doesn't exist or can't be read
    /// - JSON parsing fails
    /// - Version mismatch
    /// - Validation fails (stale/corrupted data)
    pub fn load(path: &Path) -> Option<Self> {
        let json = std::fs::read_to_string(path).ok()?;

        let backup = match serde_json::from_str::<Self>(&json) {
            Ok(b) => b,
            Err(e) => {
                tracing::warn!(?path, error = %e, "Failed to parse backup, starting fresh");
                return None;
            }
        };

        if backup.version != BACKUP_VERSION {
            tracing::warn!(
                ?path,
                backup_version = backup.version,
                current_version = BACKUP_VERSION,
                "Backup version mismatch, starting fresh"
            );
            return None;
        }

        if let Err(e) = backup.validate() {
            tracing::warn!(?path, error = %e, "Backup validation failed, starting fresh");
            return None;
        }

        tracing::info!(?path, games = backup.games.len(), "Proposer backup loaded");
        Some(backup)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Schema guard: if this test fails, you likely need to bump `BACKUP_VERSION`.
    /// This catches accidental schema changes that would break backup compatibility.
    #[test]
    fn backup_schema_guard() {
        use crate::contract::{GameStatus, ProposalStatus};
        use alloy_primitives::{Address, B256};

        // If Game fields change, this won't compile or the JSON keys will differ
        let game = Game {
            index: U256::ZERO,
            address: Address::ZERO,
            parent_index: 0,
            l2_sequence_number: 0,
            status: GameStatus::InProgress,
            proposal_status: ProposalStatus::Unchallenged,
            deadline: 0,
            should_attempt_to_resolve: false,
            should_attempt_to_claim_bond: false,
            absolute_prestate: B256::ZERO,
        };

        let json = serde_json::to_value(&game).unwrap();
        let mut keys: Vec<_> = json.as_object().unwrap().keys().cloned().collect();
        keys.sort();

        // If this assertion fails, Game schema changed - bump BACKUP_VERSION!
        assert_eq!(
            keys,
            vec![
                "absolute_prestate",
                "address",
                "deadline",
                "index",
                "l2_sequence_number",
                "parent_index",
                "proposal_status",
                "should_attempt_to_claim_bond",
                "should_attempt_to_resolve",
                "status",
            ],
            "Game schema changed! Bump BACKUP_VERSION in backup.rs"
        );

        // Check ProposerBackup fields
        let backup = ProposerBackup::new(None, vec![], None);
        let json = serde_json::to_value(&backup).unwrap();
        let mut keys: Vec<_> = json.as_object().unwrap().keys().cloned().collect();
        keys.sort();

        assert_eq!(
            keys,
            vec![
                "anchor_game_index",
                "cursor",
                "games",
                "last_created_game_address",
                "last_created_game_l2_sequence_number",
                "pending_games",
                "version"
            ],
            "ProposerBackup schema changed! Bump BACKUP_VERSION in backup.rs"
        );
    }
}
