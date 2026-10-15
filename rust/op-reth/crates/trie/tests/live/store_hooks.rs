//! A proofs store whose transactions a case can delay, hold, fail or make panic: the faults a
//! slow or broken disk puts in the engine's way, injected where the engine opens a read or a
//! write transaction.

use parking_lot::Mutex;
use reth_db::DatabaseError;
use reth_optimism_trie::{OpProofsStorageResult, OpProofsStore};
use std::{
    fmt,
    sync::{
        Arc,
        mpsc::{Receiver, SyncSender, sync_channel},
    },
    thread,
    time::Duration,
};

/// How long a case waits for the engine's persistence thread to reach a held write.
const HELD_WRITE_TIMEOUT: Duration = Duration::from_secs(30);

/// Runs on the thread opening the transaction, before the inner store opens it; an error is
/// returned in the transaction's place.
type StoreHook = Box<dyn FnOnce() -> OpProofsStorageResult<()> + Send>;

#[derive(Default)]
struct Hooks {
    next_read: Option<StoreHook>,
    next_write: Option<StoreHook>,
    write_delay: Duration,
}

/// How a held write goes on once released.
enum Release {
    Panic,
    Proceed,
}

#[derive(Clone)]
pub(super) struct StoreWithHooks<S> {
    /// Taken under the lock and run after it is released: a hook may block, and a case arming
    /// the next hook meanwhile would deadlock against the lock.
    hooks: Arc<Mutex<Hooks>>,
    inner: S,
}

impl<S> StoreWithHooks<S> {
    pub(super) fn new(inner: S) -> Self {
        Self { hooks: Arc::default(), inner }
    }

    /// The wrapped store, for reads that must not consume an armed hook.
    pub(super) const fn inner(&self) -> &S {
        &self.inner
    }

    /// Runs `hook` on the thread opening the next read transaction, right before it is opened.
    /// Any read consumes it, a case's own reads through this wrapper included.
    pub(super) fn before_next_read(
        &self,
        hook: impl FnOnce() -> OpProofsStorageResult<()> + Send + 'static,
    ) {
        self.hooks.lock().next_read = Some(Box::new(hook));
    }

    /// Runs `hook` on the thread opening the next write transaction, right before it is opened:
    /// the engine's persistence thread, at its next save or unwind.
    fn before_next_write(&self, hook: impl FnOnce() -> OpProofsStorageResult<()> + Send + 'static) {
        self.hooks.lock().next_write = Some(Box::new(hook));
    }

    pub(super) fn is_next_read_armed(&self) -> bool {
        self.hooks.lock().next_read.is_some()
    }

    /// Withdraws a write hook no write has consumed yet; true if there was one.
    pub(super) fn disarm_next_write(&self) -> bool {
        self.hooks.lock().next_write.take().is_some()
    }

    /// Every write transaction from now on opens `delay` late, as on a slow disk.
    pub(super) fn delay_writes(&self, delay: Duration) {
        self.hooks.lock().write_delay = delay;
    }

    /// The next write transaction fails to open, as on a full disk.
    pub(super) fn fail_next_write(&self) {
        self.before_next_write(|| {
            Err(DatabaseError::Other("injected write failure".to_owned()).into())
        });
    }

    /// The next write transaction blocks until the returned hold is released, which keeps a
    /// save of the engine in flight for as long as a case needs.
    pub(super) fn hold_next_write(&self) -> HeldWrite {
        let (entered_tx, entered_rx) = sync_channel(1);
        let (release_tx, release_rx) = sync_channel(1);
        self.before_next_write(move || {
            let _ = entered_tx.send(());
            match release_rx.recv() {
                Ok(Release::Panic) => panic!("injected panic in a held write transaction"),
                Ok(Release::Proceed) | Err(_) => Ok(()),
            }
        });
        HeldWrite { entered: entered_rx, release: release_tx }
    }

    /// The thread opening the next read transaction panics, as a bug in the store would.
    pub(super) fn panic_in_next_read(&self) {
        self.before_next_read(|| panic!("injected panic in a read transaction"));
    }

    /// The thread opening the next write transaction panics, as a bug in the store would.
    pub(super) fn panic_in_next_write(&self) {
        self.before_next_write(|| panic!("injected panic in a write transaction"));
    }
}

/// The case's side of a held write. Dropping it lets the write go on too.
pub(super) struct HeldWrite {
    entered: Receiver<()>,
    release: SyncSender<Release>,
}

impl HeldWrite {
    /// Blocks until the engine's persistence thread is inside the held write.
    pub(super) fn wait_until_entered(&self) {
        self.entered
            .recv_timeout(HELD_WRITE_TIMEOUT)
            .expect("the engine's persistence thread reaches the held write");
    }

    /// Lets the held write go on, or the next write through if none is held yet.
    pub(super) fn release(self) {
        let _ = self.release.send(Release::Proceed);
    }

    /// Lets the held write go on into a panic, as a store that dies mid-write would.
    pub(super) fn release_into_panic(self) {
        let _ = self.release.send(Release::Panic);
    }
}

impl<S: fmt::Debug> fmt::Debug for StoreWithHooks<S> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("StoreWithHooks").field("inner", &self.inner).finish_non_exhaustive()
    }
}

impl<S: OpProofsStore> OpProofsStore for StoreWithHooks<S> {
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
        let hook = self.hooks.lock().next_read.take();
        if let Some(hook) = hook {
            hook()?;
        }
        self.inner.provider_ro()
    }

    fn provider_rw<'a>(&'a self) -> OpProofsStorageResult<Self::ProviderRw<'a>> {
        let (hook, delay) = {
            let mut hooks = self.hooks.lock();
            (hooks.next_write.take(), hooks.write_delay)
        };
        if let Some(hook) = hook {
            hook()?;
        }
        thread::sleep(delay);
        self.inner.provider_rw()
    }

    fn initialization_provider<'a>(&'a self) -> OpProofsStorageResult<Self::Initializer<'a>> {
        self.inner.initialization_provider()
    }
}
