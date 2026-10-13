use discv5::Enr;
use tokio::sync::mpsc::{self, error::TrySendError};

/// The network that dials the peers discovery finds.
pub trait Network: Send + 'static {
    /// Asks the network to dial `enr` without waiting for room in its queue.
    ///
    /// Discovery drops a peer the network has no room for; it finds peers again.
    fn dial(&self, enr: Enr) -> Result<(), TrySendError<Enr>>;
}

impl Network for mpsc::Sender<Enr> {
    fn dial(&self, enr: Enr) -> Result<(), TrySendError<Enr>> {
        self.try_send(enr)
    }
}
