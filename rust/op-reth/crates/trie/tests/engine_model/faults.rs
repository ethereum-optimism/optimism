//! A proofs store whose next write transaction can be made to fail, the way a full disk or a
//! database error fails a save.

use reth_db::DatabaseError;
use reth_optimism_trie::{OpProofsStorageResult, OpProofsStore};
use std::{
    fmt,
    sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    },
};

#[derive(Clone)]
pub(crate) struct StoreWithSaveFaults<S> {
    fail_next_write: Arc<AtomicBool>,
    inner: S,
}

impl<S> StoreWithSaveFaults<S> {
    pub(crate) fn new(inner: S) -> Self {
        Self { fail_next_write: Arc::default(), inner }
    }

    pub(crate) fn fail_next_write(&self) {
        self.fail_next_write.store(true, Ordering::SeqCst);
    }

    /// Withdraws a failure that no write has consumed yet; true if there was one.
    pub(crate) fn disarm(&self) -> bool {
        self.fail_next_write.swap(false, Ordering::SeqCst)
    }
}

impl<S: fmt::Debug> fmt::Debug for StoreWithSaveFaults<S> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("StoreWithSaveFaults").field("inner", &self.inner).finish_non_exhaustive()
    }
}

impl<S: OpProofsStore> OpProofsStore for StoreWithSaveFaults<S> {
    type ProviderRO<'a>
        = S::ProviderRO<'a>
    where
        Self: 'a;

    type ProviderRw<'a>
        = S::ProviderRw<'a>
    where
        Self: 'a;

    type Initializer<'a>
        = S::Initializer<'a>
    where
        Self: 'a;

    fn provider_ro<'a>(&'a self) -> OpProofsStorageResult<Self::ProviderRO<'a>> {
        self.inner.provider_ro()
    }

    fn provider_rw<'a>(&'a self) -> OpProofsStorageResult<Self::ProviderRw<'a>> {
        if self.fail_next_write.swap(false, Ordering::SeqCst) {
            return Err(DatabaseError::Other("injected write failure".to_owned()).into());
        }
        self.inner.provider_rw()
    }

    fn initialization_provider<'a>(&'a self) -> OpProofsStorageResult<Self::Initializer<'a>> {
        self.inner.initialization_provider()
    }
}
