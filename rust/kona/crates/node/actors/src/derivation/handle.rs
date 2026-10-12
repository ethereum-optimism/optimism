//! Reading the derivation actor's progress.

use super::{DerivationStatus, Handle};
use tokio::sync::watch;

impl Handle {
    /// Subscribes to the derivation progress the actor publishes.
    pub fn status(&self) -> watch::Receiver<DerivationStatus> {
        self.status.clone()
    }
}
