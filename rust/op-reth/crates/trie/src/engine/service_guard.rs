//! Generic guard that joins a service thread on drop.

use std::{fmt, thread::JoinHandle};

/// Joins the wrapped thread when dropped. `None` is allowed for test/mock construction.
pub(super) struct ServiceGuard(Option<JoinHandle<()>>);

impl ServiceGuard {
    pub(super) const fn new(handle: JoinHandle<()>) -> Self {
        Self(Some(handle))
    }
}

impl fmt::Debug for ServiceGuard {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_tuple("ServiceGuard").field(&self.0.as_ref().map(|_| "...")).finish()
    }
}

impl Drop for ServiceGuard {
    fn drop(&mut self) {
        if let Some(join_handle) = self.0.take() {
            let _ = join_handle.join();
        }
    }
}
