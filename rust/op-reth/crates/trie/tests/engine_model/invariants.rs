//! What the harness checks after every step and at the end of a case.

use super::{
    harness::{Harness, STEP_TIMEOUT},
    store_view::BlockView,
};
use alloy_eips::NumHash;
use reth_optimism_trie::{OpProofsProviderRO, OpProofsStore, ProofWindowRange};
use std::{
    thread,
    time::{Duration, Instant},
};

const POLL_INTERVAL: Duration = Duration::from_millis(2);

/// Flushes a settle may need before the store stops changing.
const MAX_SETTLE_FLUSHES: usize = 64;

impl Harness {
    pub(super) fn check_invariants(&mut self) -> eyre::Result<()> {
        eyre::ensure!(
            self.model.num_hash(self.model.head()) == self.reth.head(),
            "the model's head {:?} is not reth's {:?}",
            self.model.num_hash(self.model.head()),
            self.reth.head()
        );
        let tip = self.settle_engine()?;
        let (window, views) = self.read_window_and_views(self.last_fork)?;
        self.ensure_canonical("earliest", window.earliest)?;
        self.ensure_canonical("latest", window.latest)?;
        eyre::ensure!(
            window.latest.number <= tip.number,
            "the store's latest block {} is above the engine's tip {}",
            window.latest.number,
            tip.number
        );
        eyre::ensure!(
            window.latest.number >= self.model.persisted_at_least(),
            "the store's latest block {} is below block {}, which it had persisted",
            window.latest.number,
            self.model.persisted_at_least()
        );
        let earliest_bounds = self.model.earliest_bounds(window.latest.number);
        eyre::ensure!(
            earliest_bounds.contains(&window.earliest.number),
            "the store holds blocks {}..={}, whose earliest block lies outside {:?}, what its \
             window of {} allows with reth persisted through {} before block {} was committed",
            window.earliest.number,
            window.latest.number,
            earliest_bounds,
            self.model.window(),
            self.model.block(window.latest.number).persisted_before_commit,
            window.latest.number
        );
        self.model.note_persisted(window.latest.number);
        for (number, view) in views {
            view.verify_proofs(number, self.model.block(number).state_root)?;
        }
        self.log(format!(
            "  engine tip {}, store blocks {}..={}, reth head {}",
            tip.number,
            window.earliest.number,
            window.latest.number,
            self.reth.head().number
        ));
        Ok(())
    }

    /// Flushes until a flush changes nothing, which leaves fewer blocks buffered than start a
    /// save.
    pub(super) fn settle_store(&mut self) -> eyre::Result<()> {
        let mut latest = self.window()?.latest;
        for _ in 0..MAX_SETTLE_FLUSHES {
            self.flush()?;
            let now = self.window()?.latest;
            if now == latest {
                return Ok(());
            }
            latest = now;
        }
        eyre::bail!("the store kept changing over {MAX_SETTLE_FLUSHES} flushes")
    }

    pub(super) fn check_settled(&mut self) -> eyre::Result<()> {
        let tip = self.settle_engine()?;
        let latest = self.window()?.latest.number;
        let unsaved = tip.number.checked_sub(latest).ok_or_else(|| {
            eyre::eyre!(
                "the store's latest block {latest} is above the engine's tip {}",
                tip.number
            )
        })?;
        eyre::ensure!(
            unsaved < self.config.cadence.persistence,
            "{unsaved} blocks stay buffered above the store's latest block {latest} after \
             settling, enough to start a save at the persistence threshold {}",
            self.config.cadence.persistence
        );
        self.log(format!("settled: engine tip {}, store latest {latest}", tip.number));
        Ok(())
    }

    /// Waits for the engine's tip to reach one the model predicts, and narrows the model to it.
    /// A tip off every path the model allows fails at once.
    fn settle_engine(&mut self) -> eyre::Result<NumHash> {
        let expected = self.model.settled_tips();
        let deadline = Instant::now() + STEP_TIMEOUT;
        loop {
            let tip = self.call("tip", |engine| engine.tip())??;
            if expected.contains(&tip) {
                self.model.observe_settled(tip)?;
                return Ok(tip);
            }
            eyre::ensure!(
                self.model.on_path(tip),
                "the engine's tip {tip:?} is on no path the model allows: {:?}",
                self.model.candidates()
            );
            if Instant::now() >= deadline {
                let window = self.window()?;
                eyre::bail!(
                    "the engine did not settle within {STEP_TIMEOUT:?}: its tip is {tip:?}, the \
                     model expects one of {expected:?}, the store holds blocks {}..={}",
                    window.earliest.number,
                    window.latest.number
                );
            }
            thread::sleep(POLL_INTERVAL);
        }
    }

    /// The store's window and, read in the same transaction so that a save landing meanwhile
    /// cannot prune a height from under the read, its views at the window's ends and at `fork`
    /// when the window holds it.
    fn read_window_and_views(
        &self,
        fork: Option<u64>,
    ) -> eyre::Result<(ProofWindowRange, Vec<(u64, BlockView)>)> {
        let provider = self.store().provider_ro()?;
        let window = provider.get_proof_window()?;
        let mut heights = vec![window.earliest.number, window.latest.number];
        if let Some(fork) = fork &&
            (window.earliest.number..=window.latest.number).contains(&fork)
        {
            heights.push(fork);
        }
        heights.sort_unstable();
        heights.dedup();
        let views = heights
            .into_iter()
            .map(|number| Ok((number, BlockView::read_from(&provider, number, &self.targets)?)))
            .collect::<eyre::Result<_>>()?;
        Ok((window, views))
    }

    fn describe_canonical(&self, number: u64) -> String {
        if number <= self.model.head() {
            format!("{}", self.model.block(number).hash)
        } else {
            format!("beyond the canonical head {}", self.model.head())
        }
    }

    fn ensure_canonical(&self, what: &str, block: NumHash) -> eyre::Result<()> {
        eyre::ensure!(
            self.model.is_canonical(block),
            "the store's {what} block {}:{} is not on the canonical chain, whose block {} is {}",
            block.number,
            block.hash,
            block.number,
            self.describe_canonical(block.number)
        );
        Ok(())
    }
}
