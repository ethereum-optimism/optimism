//! Getting a log store to the point where a round can be verified against it, and keeping it
//! there.
//!
//! A round can only answer existence questions for timestamps its log stores already hold, so
//! before the first round every chain's store has to cover the message expiry window behind the
//! timestamp the verifier will start at. That is what backfill does. After that, each advancing
//! round seals exactly one more block per chain, through [`seal_indexed`].
//!
//! Everything here is idempotent. A store that already holds the block being sealed is left alone
//! rather than written again, which is what makes a crash between a write-ahead log entry and its
//! last side effect recoverable by replaying the entry.

use crate::{
    chain::{ChainError, InteropChain, RoundError},
    error::StoreError,
    logs::LogsDb,
    verify::FrontierBlock,
};
use alloy_eips::BlockNumHash;
use alloy_primitives::ChainId;
use tracing::{info, warn};

/// Seals an indexed block into `store`, or leaves the store alone when it already holds it.
///
/// The store enforces contiguity and parent-hash chaining itself; this adds the two things it
/// cannot decide for itself — whether an already-present block is the same block (a replay, which
/// is fine) or a different one (a reorg the store has not been told about yet), and whether the
/// gap to the block being sealed is one the store can bridge.
pub fn seal_indexed(store: &dyn LogsDb, block: &FrontierBlock) -> Result<(), StoreError> {
    let id = block.block.id();

    if let Some(latest) = store.latest_sealed_block() {
        if latest.number >= id.number {
            let seal = store.find_sealed_block(id.number)?;
            return if seal.hash == id.hash {
                // Already sealed, identically: a replayed transition, or a round whose timestamp
                // fell inside a block the previous round already sealed.
                Ok(())
            } else {
                Err(StoreError::Conflict("store holds a different block at that height"))
            };
        }
        if id.number > latest.number + 1 {
            return Err(StoreError::OutOfOrder("sealing would leave a hole in the log store"));
        }
    }

    let parent_hash = block.block.parent_hash;
    let parent = BlockNumHash { number: id.number.saturating_sub(1), hash: parent_hash };
    for (log_index, log_hash) in block.log_hashes.iter().enumerate() {
        let log_index = log_index as u32;
        store.add_log(
            *log_hash,
            parent,
            log_index,
            block.executing_messages.get(&log_index).map(|message| message.stored),
        )?;
    }
    store.seal_block(parent_hash, id, block.block.timestamp)
}

/// Fetches `block`'s logs and seals them into `store`.
pub async fn fetch_and_seal(
    chain: &dyn InteropChain,
    store: &dyn LogsDb,
    block: BlockNumHash,
) -> Result<(), RoundError> {
    let chain_id = chain.chain_id();
    let logs =
        chain.block_logs(block).await.map_err(|source| RoundError::chain(chain_id, source))?;
    if logs.id() != block {
        return Err(RoundError::Invariant(format!(
            "chain {chain_id}: asked for the logs of {block:?} and was given {:?}",
            logs.id()
        )));
    }
    seal_indexed(store, &FrontierBlock::index(chain_id, &logs)).map_err(RoundError::Store)
}

/// Drops any tail of `store` whose blocks are no longer canonical on `chain`.
///
/// A reorg that happens while the verifier is not running leaves the store's tip on a chain that
/// no longer exists. Without trimming it, the first seal after restart can never satisfy the
/// store's parent-hash check and the node retries forever.
pub async fn reconcile_tail(
    chain: &dyn InteropChain,
    store: &dyn LogsDb,
) -> Result<(), RoundError> {
    let chain_id = chain.chain_id();
    let Some(latest) = store.latest_sealed_block() else { return Ok(()) };

    let canonical = |number: u64| async move {
        chain
            .output_at(number)
            .await
            .map(|output| output.block_hash)
            .map_err(|source| RoundError::chain(chain_id, source))
    };

    if canonical(latest.number).await? == latest.hash {
        return Ok(());
    }

    let first = store.first_sealed_block().map_err(RoundError::Store)?;
    for number in (first.number..latest.number).rev() {
        let seal = store.find_sealed_block(number).map_err(RoundError::Store)?;
        if seal.hash != canonical(number).await? {
            continue;
        }
        warn!(
            target: "lokahi_interop",
            chain_id,
            rewind_to = number,
            trimmed_tip = latest.number,
            "Log store tail diverged from canonical; rewinding"
        );
        return store.rewind(BlockNumHash { number, hash: seal.hash }).map_err(RoundError::Store);
    }

    warn!(
        target: "lokahi_interop",
        chain_id,
        first_sealed = first.number,
        latest_sealed = latest.number,
        "Log store diverges from canonical throughout; clearing"
    );
    store.clear().map_err(RoundError::Store)
}

/// The block range one chain's backfill has to cover.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct BackfillRange {
    /// The first block to seal.
    pub from: u64,
    /// The last block to seal, inclusive.
    pub to: u64,
}

/// Returns the timestamp window backfill covers before `verification_start`, or [`None`] when
/// there is nothing to cover.
///
/// The lower bound is the latest of three: the interop activation time, because no earlier block
/// can hold a message anyone may reference; the requested depth behind the start, because messages
/// older than the expiry window can no longer be executed; and — per chain, applied by the
/// caller — the chain's own genesis.
pub fn backfill_window(
    activation: u64,
    verification_start: u64,
    depth_seconds: u64,
) -> Option<(u64, u64)> {
    if depth_seconds == 0 {
        return None;
    }
    let end = verification_start.checked_sub(1)?;
    let start = end.saturating_sub(depth_seconds).max(activation);
    (start <= end).then_some((start, end))
}

/// Seals every canonical block of `chain` in `range` into `store`.
pub async fn backfill_chain(
    chain: &dyn InteropChain,
    store: &dyn LogsDb,
    range: BackfillRange,
) -> Result<(), RoundError> {
    let chain_id = chain.chain_id();
    reconcile_tail(chain, store).await?;

    // Resume behind whatever survived reconciliation rather than re-sealing it.
    let from = store.latest_sealed_block().map_or(range.from, |latest| latest.number + 1);
    if from > range.to {
        return Ok(());
    }

    info!(
        target: "lokahi_interop",
        chain_id,
        from,
        to = range.to,
        "Backfilling log store"
    );
    for number in from..=range.to {
        let output =
            chain.output_at(number).await.map_err(|source| RoundError::chain(chain_id, source))?;
        fetch_and_seal(chain, store, BlockNumHash { number, hash: output.block_hash }).await?;
    }
    Ok(())
}

/// Resolves the block range `chain` must backfill for a timestamp window, or [`None`] when the
/// window falls entirely before the chain existed.
pub async fn chain_backfill_range(
    chain: &dyn InteropChain,
    window: (u64, u64),
) -> Result<Option<BackfillRange>, RoundError> {
    let chain_id = chain.chain_id();
    let (start, end) = window;
    // A chain younger than the window starts at its own genesis: there is no earlier block of it
    // to seal, and asking for one is not an error to retry.
    let start = start.max(chain.rollup_config().genesis.l2_time);
    if start > end {
        return Ok(None);
    }

    let number_at = |timestamp: u64| async move {
        chain
            .block_number_at_timestamp(timestamp)
            .await
            .map_err(|source: ChainError| RoundError::chain(chain_id, source))
    };

    Ok(Some(BackfillRange { from: number_at(start).await?, to: number_at(end).await? }))
}

/// Returns the timestamp the verifier should start at: the latest of interop activation and every
/// chain's first recorded safe head.
///
/// A chain whose safe-head history begins later than another's sets the bound: the verifier cannot
/// verify a timestamp one of its chains has no safe head for.
pub fn verification_start(activation: u64, first_safe_heads: &[(ChainId, u64)]) -> u64 {
    first_safe_heads.iter().map(|&(_, timestamp)| timestamp).fold(activation, u64::max)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn a_zero_depth_disables_backfill() {
        assert_eq!(backfill_window(100, 200, 0), None);
    }

    #[test]
    fn the_window_ends_just_before_the_first_verified_timestamp() {
        assert_eq!(backfill_window(100, 200, 50), Some((149, 199)));
    }

    #[test]
    fn the_window_never_reaches_before_activation() {
        assert_eq!(backfill_window(180, 200, 50), Some((180, 199)));
    }

    #[test]
    fn a_window_entirely_before_activation_is_empty() {
        assert_eq!(backfill_window(500, 200, 50), None);
    }

    #[test]
    fn a_verification_start_at_genesis_has_no_window_behind_it() {
        assert_eq!(backfill_window(0, 0, 50), None);
    }

    #[test]
    fn the_verification_start_is_the_latest_bound() {
        assert_eq!(verification_start(1000, &[(901, 900), (902, 1100)]), 1100);
        assert_eq!(verification_start(1000, &[(901, 900), (902, 950)]), 1000);
        assert_eq!(verification_start(1000, &[]), 1000);
    }
}
