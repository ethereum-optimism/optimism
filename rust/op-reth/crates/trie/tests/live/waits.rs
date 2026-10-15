//! Bounded waits on the engine and on the store, so that a stall fails a case instead of
//! hanging it.

use reth_ethereum_primitives::Block;
use reth_optimism_trie::engine::EngineHandle;
use std::{
    sync::mpsc,
    thread,
    time::{Duration, Instant},
};
use thiserror::Error;

const POLL_INTERVAL: Duration = Duration::from_millis(10);

/// Why a bounded wait on an engine call ended without its result.
#[derive(Debug, Error)]
pub(super) enum CallError {
    #[error("{call_name} did not return within {bound:?}")]
    Hung { bound: Duration, call_name: &'static str },
    #[error("{call_name} panicked")]
    Panicked { call_name: &'static str },
}

/// An engine call running on its own thread.
pub(super) struct PendingCall<T> {
    call_name: &'static str,
    reply: mpsc::Receiver<T>,
    started: Instant,
}

impl<T: Send + 'static> PendingCall<T> {
    pub(super) fn spawn(
        engine: EngineHandle<Block>,
        call_name: &'static str,
        call: impl FnOnce(EngineHandle<Block>) -> T + Send + 'static,
    ) -> Self {
        let (reply_tx, reply_rx) = mpsc::channel();
        thread::spawn(move || {
            let _ = reply_tx.send(call(engine));
        });
        Self { call_name, reply: reply_rx, started: Instant::now() }
    }

    /// Fails if the call returns within `probe`, while it had to stay blocked.
    pub(super) fn ensure_blocked_for(&self, probe: Duration) -> eyre::Result<()> {
        match self.reply.recv_timeout(probe) {
            Err(mpsc::RecvTimeoutError::Timeout) => Ok(()),
            Ok(_) => eyre::bail!(
                "{} returned after {:?} while it had to stay blocked",
                self.call_name,
                self.started.elapsed()
            ),
            Err(mpsc::RecvTimeoutError::Disconnected) => {
                Err(CallError::Panicked { call_name: self.call_name }.into())
            }
        }
    }

    /// The call's result and how long it took, or why neither is there once `bound` has passed
    /// since the call started.
    pub(super) fn wait(self, bound: Duration) -> Result<(T, Duration), CallError> {
        let remaining = bound.saturating_sub(self.started.elapsed());
        match self.reply.recv_timeout(remaining) {
            Ok(value) => Ok((value, self.started.elapsed())),
            Err(mpsc::RecvTimeoutError::Timeout) => {
                Err(CallError::Hung { bound, call_name: self.call_name })
            }
            Err(mpsc::RecvTimeoutError::Disconnected) => {
                Err(CallError::Panicked { call_name: self.call_name })
            }
        }
    }
}

/// Polls `condition` until it holds, or fails once `bound` has passed; `what` names what is
/// waited for.
pub(super) fn wait_until(
    what: &str,
    bound: Duration,
    mut condition: impl FnMut() -> eyre::Result<bool>,
) -> eyre::Result<()> {
    let deadline = Instant::now() + bound;
    loop {
        if condition()? {
            return Ok(());
        }
        eyre::ensure!(Instant::now() < deadline, "{what} did not happen within {bound:?}");
        thread::sleep(POLL_INTERVAL);
    }
}
