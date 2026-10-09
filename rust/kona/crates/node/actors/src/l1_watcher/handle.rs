use super::State;
use tokio::sync::watch;

/// Reads the latest L1 observations.
#[derive(Debug, Clone)]
pub struct Handle {
    state: watch::Receiver<State>,
}

impl Handle {
    pub(super) const fn new(state: watch::Receiver<State>) -> Self {
        Self { state }
    }

    /// Subscribes to the latest observations.
    pub fn state_receiver(&self) -> watch::Receiver<State> {
        self.state.clone()
    }
}
