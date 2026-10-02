//! Immutable committed overlays and worker-local parent readers.

use crate::SpeculationError;
use imbl::HashMap;
use revm::{
    Database,
    database::{State, states::CacheAccount},
    database_interface::OnStateHook,
    primitives::{Address, B256, U256},
    state::{AccountInfo, Bytecode, EvmState},
};
use std::{
    collections::{BTreeMap, BTreeSet},
    fmt,
    sync::{Arc, Mutex, Weak},
    time::Instant,
};

/// A factory must reproduce the exact execution base, including branch-specific ancestors.
/// Readers are opened, used and dropped on the worker; they need not be Send or Sync.
pub trait StateReadFactory: Send + Sync + 'static {
    /// Opens an independent reader of the fixed base state.
    fn open(&self) -> Result<Box<dyn Database<Error = SpeculationError>>, SpeculationError>;
}

impl<F> StateReadFactory for F
where
    F: Fn() -> Result<Box<dyn Database<Error = SpeculationError>>, SpeculationError>
        + Send
        + Sync
        + 'static,
{
    fn open(&self) -> Result<Box<dyn Database<Error = SpeculationError>>, SpeculationError> {
        self()
    }
}

#[derive(Clone, Debug)]
struct SnapshotAccount {
    info: Option<AccountInfo>,
    storage: HashMap<U256, U256>,
    storage_known: bool,
}

/// An immutable logical view. Clones share persistent map nodes, including storage maps.
#[derive(Clone, Debug, Default)]
pub struct CommittedSnapshot {
    accounts: HashMap<Address, SnapshotAccount>,
    code: HashMap<B256, Bytecode>,
    hashes: HashMap<u64, B256>,
    bytes: usize,
}

impl CommittedSnapshot {
    /// Estimated retained logical data, conservatively charged even for shared bytecode.
    pub const fn size_bytes(&self) -> usize {
        self.bytes
    }

    pub(crate) fn basic(&self, address: Address) -> Option<Option<AccountInfo>> {
        self.accounts.get(&address).map(|account| account.info.clone())
    }

    pub(crate) fn storage(&self, address: Address, slot: U256) -> Option<U256> {
        self.accounts.get(&address).and_then(|account| {
            account
                .storage
                .get(&slot)
                .copied()
                .or_else(|| (account.storage_known || account.info.is_none()).then_some(U256::ZERO))
        })
    }

    pub(crate) fn code(&self, hash: B256) -> Option<Bytecode> {
        self.code.get(&hash).cloned()
    }
    pub(crate) fn block_hash(&self, number: u64) -> Option<B256> {
        self.hashes.get(&number).copied()
    }

    fn insert_code(&mut self, hash: B256, code: Bytecode, limit: usize) {
        let charge = self
            .code
            .get(&hash)
            .map_or(128 + code.len(), |old| code.len().saturating_sub(old.len()));
        if self.bytes.saturating_add(charge) > limit {
            self.bytes = limit.saturating_add(1);
            return;
        }
        if let Some(old) = self.code.insert(hash, code.clone()) {
            self.bytes = self.bytes.saturating_sub(old.len());
        } else {
            self.bytes = self.bytes.saturating_add(128);
        }
        self.bytes = self.bytes.saturating_add(code.len());
    }

    fn import_account(&mut self, address: Address, account: &CacheAccount, limit: usize) {
        self.import_parts(
            address,
            account.account_info(),
            account.status.is_storage_known(),
            account.account.as_ref().map_or(0, |a| a.storage.len()),
            account.account.iter().flat_map(|a| a.storage.iter().map(|(k, v)| (*k, *v))),
            limit,
        );
    }

    fn import_parts(
        &mut self,
        address: Address,
        info: Option<AccountInfo>,
        storage_known: bool,
        slots: usize,
        storage: impl Iterator<Item = (U256, U256)>,
        limit: usize,
    ) {
        let old_bytes = self
            .accounts
            .get(&address)
            .map_or(0, |old| 256 + old.storage.len().saturating_mul(128));
        let bytes =
            self.bytes.saturating_sub(old_bytes).saturating_add(256 + slots.saturating_mul(128));
        if bytes > limit {
            self.bytes = limit.saturating_add(1);
            return;
        }
        self.bytes = bytes;
        if let Some(info) = &info &&
            let Some(code) = &info.code
        {
            self.insert_code(info.code_hash, code.clone(), limit);
        }
        if self.bytes > limit {
            return;
        }
        self.accounts
            .insert(address, SnapshotAccount { info, storage: storage.collect(), storage_known });
    }

    fn refresh_account(
        &mut self,
        address: Address,
        account: &CacheAccount,
        dirty: DirtyAccount,
        limit: usize,
    ) {
        let existed = self.accounts.contains_key(&address);
        let mut entry = match self.accounts.get(&address).cloned() {
            Some(entry) => entry,
            None => {
                self.bytes = self.bytes.saturating_add(256);
                if self.bytes > limit {
                    return;
                }
                SnapshotAccount { info: None, storage: HashMap::new(), storage_known: false }
            }
        };
        entry.info = account.account_info();
        let storage_known = account.status.is_storage_known();
        // Becoming fully known does not itself clear existing storage (e.g. a previously
        // loaded account without nonce/code). Import that lifecycle transition from State.
        if storage_known && !entry.storage_known && !dirty.reset && entry.info.is_some() {
            if !existed {
                self.bytes = self.bytes.saturating_sub(256);
            }
            self.import_account(address, account, limit);
            return;
        }
        if dirty.reset || entry.info.is_none() {
            self.bytes = self.bytes.saturating_sub(entry.storage.len().saturating_mul(128));
            entry.storage.clear();
        }
        entry.storage_known = storage_known;
        for slot in &dirty.slots {
            // Read revm's committed cache, never a worker's pre-settlement journal.
            if let Some(value) = account.storage_slot(*slot) {
                if !entry.storage.contains_key(slot) && self.bytes.saturating_add(128) > limit {
                    self.bytes = limit.saturating_add(1);
                    return;
                }
                if entry.storage.insert(*slot, value).is_none() {
                    self.bytes = self.bytes.saturating_add(128);
                }
            }
        }
        if let Some(info) = &entry.info &&
            let Some(code) = &info.code
        {
            self.insert_code(info.code_hash, code.clone(), limit);
        }
        self.accounts.insert(address, entry);
    }
}

#[derive(Debug, Default)]
struct DirtyAccount {
    slots: BTreeSet<U256>,
    reset: bool,
}

#[derive(Default)]
struct SessionState {
    snapshot: CommittedSnapshot,
    current: Option<Arc<SnapshotVersion>>,
    leases: Arc<Mutex<BTreeMap<u64, usize>>>,
    next_version: u64,
    dirty: BTreeMap<Address, DirtyAccount>,
    dirty_bytes: usize,
    attached: bool,
    disabled: bool,
}

/// A distinct immutable root is charged once, until its last owner releases it.
#[derive(Debug)]
pub(crate) struct SnapshotVersion {
    snapshot: CommittedSnapshot,
    id: u64,
    leases: Weak<Mutex<BTreeMap<u64, usize>>>,
}

impl core::ops::Deref for SnapshotVersion {
    type Target = CommittedSnapshot;
    fn deref(&self) -> &Self::Target {
        &self.snapshot
    }
}

impl Drop for SnapshotVersion {
    fn drop(&mut self) {
        if let Some(leases) = self.leases.upgrade() {
            leases.lock().unwrap_or_else(std::sync::PoisonError::into_inner).remove(&self.id);
        }
    }
}

impl SessionState {
    fn retained_bytes(&self) -> usize {
        self.leases
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
            .values()
            .fold(0usize, |sum, bytes| sum.saturating_add(*bytes))
    }

    fn estimated_bytes(&self) -> usize {
        self.retained_bytes()
            .saturating_add(self.dirty_bytes)
            .saturating_add(if self.current.is_none() { self.snapshot.bytes } else { 0 })
    }
}

/// Read source and committed state belonging to one build or historical batch.
pub struct SnapshotSession {
    generation: crate::ExecutionGeneration,
    factory: Arc<dyn StateReadFactory>,
    state: Arc<Mutex<SessionState>>,
    limit: usize,
    public_hook: Arc<Mutex<Option<Box<dyn OnStateHook>>>>,
}

impl fmt::Debug for SnapshotSession {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("SnapshotSession").field("limit", &self.limit).finish_non_exhaustive()
    }
}

impl SnapshotSession {
    /// Creates an operation-local session. Do not share it across independent builds.
    pub fn new(factory: Arc<dyn StateReadFactory>, limit: usize) -> Self {
        Self {
            generation: crate::ExecutionGeneration::default(),
            factory,
            state: Arc::new(Mutex::new(SessionState::default())),
            limit,
            public_hook: Arc::default(),
        }
    }

    /// Estimated logical bytes retained by the committed overlay and pending dirty entries.
    pub fn estimated_bytes(&self) -> usize {
        let state = self.state.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
        state.estimated_bytes()
    }

    /// Identity shared by snapshots throughout this historical batch or payload build.
    pub const fn generation(&self) -> &crate::ExecutionGeneration {
        &self.generation
    }

    /// Whether a source or resource failure has disabled independent reads.
    pub fn is_disabled(&self) -> bool {
        self.state.lock().unwrap_or_else(std::sync::PoisonError::into_inner).disabled
    }

    /// Disables independent reads until a new session is created.
    pub fn disable(&self, reason: &'static str) {
        let mut state = self.state.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
        if !state.disabled {
            metrics::counter!("optimism_parallel.source_fallbacks", "reason" => reason)
                .increment(1);
        }
        state.disabled = true;
        state.current = None;
        state.snapshot = CommittedSnapshot::default();
        state.dirty.clear();
        state.dirty_bytes = 0;
    }

    /// Replaces only the public hook, retaining committed change tracking across batch blocks.
    pub fn set_public_hook<DB: Database>(
        &self,
        canonical: &mut State<DB>,
        hook: Option<Box<dyn OnStateHook>>,
    ) {
        let _ = self.capture(canonical);
        let attached =
            self.state.lock().unwrap_or_else(std::sync::PoisonError::into_inner).attached;
        if attached {
            *self.public_hook.lock().unwrap_or_else(std::sync::PoisonError::into_inner) = hook;
        } else {
            canonical.set_state_hook(hook);
        }
    }

    /// Captures only committed data from the canonical revm State. Public state hooks are
    /// composed, not replaced. If a caller replaces a hook, its Drop invalidates the attachment
    /// and the next capture imports the actual state again rather than using an incomplete view.
    pub fn capture<DB: Database>(&self, canonical: &mut State<DB>) -> Option<ReadWindow> {
        let started = Instant::now();
        let mut state = self.state.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
        if state.disabled || self.generation.is_cancelled() {
            return None;
        }
        let hashes: HashMap<_, _> = canonical.block_hashes.iter().collect();
        if state.attached &&
            state.dirty.is_empty() &&
            hashes == state.snapshot.hashes &&
            let Some(snapshot) = &state.current
        {
            return Some(ReadWindow {
                snapshot: snapshot.clone(),
                factory: self.factory.clone(),
                generation: self.generation.clone(),
            });
        }
        // Drop the canonical reference before charging the replacement. Worker references keep
        // old versions charged; structurally shared versions are conservatively counted in full.
        state.current = None;
        let limit = self.limit.saturating_sub(state.retained_bytes());
        if state.attached {
            let dirty = core::mem::take(&mut state.dirty);
            state.dirty_bytes = 0;
            for (address, dirty) in dirty {
                let Some(account) = canonical.cache.accounts.get(&address) else {
                    state.disabled = true;
                    break;
                };
                state.snapshot.refresh_account(address, account, dirty, limit);
                if state.snapshot.bytes > limit {
                    break;
                }
            }
        } else {
            state.snapshot = CommittedSnapshot::default();
            state.dirty.clear();
            state.dirty_bytes = 0;
            if canonical.use_preloaded_bundle {
                for (address, account) in &canonical.bundle_state.state {
                    state.snapshot.import_parts(
                        *address,
                        account.info.clone(),
                        account.status.is_storage_known(),
                        account.storage.len(),
                        account.storage.iter().map(|(k, v)| (*k, v.present_value())),
                        limit,
                    );
                    if state.snapshot.bytes > limit {
                        break;
                    }
                }
                for (hash, code) in &canonical.bundle_state.contracts {
                    if state.snapshot.bytes > limit {
                        break;
                    }
                    state.snapshot.insert_code(*hash, code.clone(), limit);
                }
            }
            for (address, account) in &canonical.cache.accounts {
                if state.snapshot.bytes > limit {
                    break;
                }
                state.snapshot.import_account(*address, account, limit);
            }
            for (hash, code) in &canonical.cache.contracts {
                if state.snapshot.bytes > limit {
                    break;
                }
                state.snapshot.insert_code(*hash, code.clone(), limit);
            }
            state.attached = true;
            *self.public_hook.lock().unwrap_or_else(std::sync::PoisonError::into_inner) =
                canonical.state_hook.take();
            canonical.set_state_hook(Some(Box::new(ChangeCollector {
                state: Arc::downgrade(&self.state),
                public_hook: self.public_hook.clone(),
                limit: self.limit,
            })));
        }
        // BLOCKHASH overrides are bounded by revm's history cache.
        state.snapshot.bytes =
            state.snapshot.bytes.saturating_sub(state.snapshot.hashes.len() * 96);
        state.snapshot.hashes = hashes;
        state.snapshot.bytes =
            state.snapshot.bytes.saturating_add(state.snapshot.hashes.len() * 96);
        if state.disabled || state.snapshot.bytes > limit {
            drop(state);
            self.disable("snapshot_limit_or_invalidated_cache");
            return None;
        }
        state.next_version += 1;
        let snapshot = Arc::new(SnapshotVersion {
            snapshot: state.snapshot.clone(),
            id: state.next_version,
            leases: Arc::downgrade(&state.leases),
        });
        state
            .leases
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
            .insert(snapshot.id, snapshot.bytes);
        state.current = Some(snapshot.clone());
        metrics::histogram!("optimism_parallel.snapshot_bytes")
            .record(state.estimated_bytes() as f64);
        metrics::histogram!("optimism_parallel.snapshot_versions").record(
            state.leases.lock().unwrap_or_else(std::sync::PoisonError::into_inner).len() as f64,
        );
        metrics::histogram!("optimism_parallel.snapshot_update_seconds")
            .record(started.elapsed().as_secs_f64());
        Some(ReadWindow {
            snapshot,
            factory: self.factory.clone(),
            generation: self.generation.clone(),
        })
    }
}

struct ChangeCollector {
    state: Weak<Mutex<SessionState>>,
    public_hook: Arc<Mutex<Option<Box<dyn OnStateHook>>>>,
    limit: usize,
}

impl OnStateHook for ChangeCollector {
    fn on_state(&mut self, changes: EvmState) {
        if let Some(state) = self.state.upgrade() {
            let mut state = state.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
            if !state.disabled {
                for (address, account) in &changes {
                    if !account.is_touched() {
                        continue;
                    }
                    let remaining = self.limit.saturating_sub(state.estimated_bytes());
                    let mut bytes = if state.dirty.contains_key(address) { 0 } else { 256 };
                    let mut overflow = bytes > remaining;
                    if !overflow {
                        let dirty = state.dirty.entry(*address).or_default();
                        dirty.reset |= account.is_created() || account.is_selfdestructed();
                        for (slot, _) in account.changed_storage_slots() {
                            if !dirty.slots.contains(slot) {
                                bytes += 128;
                                if bytes > remaining {
                                    overflow = true;
                                    break;
                                }
                                dirty.slots.insert(*slot);
                            }
                        }
                    }
                    state.dirty_bytes = state.dirty_bytes.saturating_add(bytes);
                    if overflow {
                        state.disabled = true;
                        state.current = None;
                        state.snapshot = CommittedSnapshot::default();
                        state.dirty.clear();
                        state.dirty_bytes = 0;
                        metrics::counter!("optimism_parallel.source_fallbacks", "reason" => "dirty_limit").increment(1);
                        break;
                    }
                }
            }
        }
        if let Some(hook) =
            &mut *self.public_hook.lock().unwrap_or_else(std::sync::PoisonError::into_inner)
        {
            hook.on_state(changes);
        }
    }
}

impl Drop for ChangeCollector {
    fn drop(&mut self) {
        if let Some(state) = self.state.upgrade() {
            state.lock().unwrap_or_else(std::sync::PoisonError::into_inner).attached = false;
        }
    }
}

/// Immutable committed version shared by a window or by rolling transaction attempts.
#[derive(Clone)]
pub struct ReadWindow {
    pub(crate) generation: crate::ExecutionGeneration,
    pub(crate) snapshot: Arc<SnapshotVersion>,
    pub(crate) factory: Arc<dyn StateReadFactory>,
}

impl fmt::Debug for ReadWindow {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("ReadWindow").field("snapshot", &self.snapshot).finish_non_exhaustive()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use revm::{
        DatabaseCommit,
        database::{EmptyDBTyped, InMemoryDB, states::bundle_state::BundleRetention},
        state::{Account, EvmStorageSlot},
    };
    use std::sync::atomic::{AtomicUsize, Ordering};

    fn session(limit: usize) -> SnapshotSession {
        SnapshotSession::new(
            Arc::new(|| {
                Ok(Box::new(EmptyDBTyped::default()) as Box<dyn Database<Error = SpeculationError>>)
            }),
            limit,
        )
    }
    fn change(
        state: &mut State<InMemoryDB>,
        address: Address,
        slot: u64,
        value: u64,
        create: bool,
        destroy: bool,
    ) {
        let info = state.basic(address).unwrap().unwrap_or_default();
        let original = state.storage(address, U256::from(slot)).unwrap();
        let mut account = Account::from(info);
        account.mark_touch();
        if create {
            account.mark_created();
        }
        if destroy {
            account.mark_selfdestruct();
        }
        account.storage.insert(
            U256::from(slot),
            EvmStorageSlot::new_changed(original, U256::from(value), Default::default()),
        );
        state.commit(std::iter::once((address, account)).collect());
    }

    #[test]
    fn snapshots_follow_canonical_lifecycle_and_keep_old_versions_immutable() {
        let address = Address::repeat_byte(1);
        let missing = Address::repeat_byte(2);
        let mut base = InMemoryDB::default();
        base.insert_account_info(address, AccountInfo::default().with_balance(U256::from(7)));
        base.insert_account_storage(address, U256::ZERO, U256::from(123)).unwrap();
        let mut state = State::builder().with_database(base).with_bundle_update().build();
        state.basic(address).unwrap();
        state.basic(missing).unwrap();
        let session = session(65536);
        let first = session.capture(&mut state).unwrap().snapshot;
        assert_eq!(first.basic(missing), Some(None));
        assert_eq!(first.storage(missing, U256::ZERO), Some(U256::ZERO));
        assert_eq!(first.storage(address, U256::ZERO), None, "unknown slot must use base");
        change(&mut state, address, 0, 0, false, false);
        let zero = session.capture(&mut state).unwrap().snapshot;
        assert_eq!(zero.storage(address, U256::ZERO), Some(U256::ZERO));
        assert_eq!(first.storage(address, U256::ZERO), None);
        change(&mut state, address, 1, 99, false, false);
        let before_delete = session.capture(&mut state).unwrap().snapshot;
        change(&mut state, address, 1, 99, false, true);
        let deleted = session.capture(&mut state).unwrap().snapshot;
        assert_eq!(deleted.basic(address), Some(None));
        assert_eq!(deleted.storage(address, U256::from(1)), Some(U256::ZERO));
        change(&mut state, address, 2, 42, true, false);
        let recreated = session.capture(&mut state).unwrap().snapshot;
        assert_eq!(recreated.storage(address, U256::ZERO), Some(U256::ZERO));
        assert_eq!(recreated.storage(address, U256::from(1)), Some(U256::ZERO));
        assert_eq!(recreated.storage(address, U256::from(2)), Some(U256::from(42)));
        assert_eq!(before_delete.storage(address, U256::from(1)), Some(U256::from(99)));
    }

    #[test]
    fn hooks_and_unpersisted_blocks_retain_tracking_without_reimporting_state() {
        let address = Address::repeat_byte(3);
        let mut state =
            State::builder().with_database(InMemoryDB::default()).with_bundle_update().build();
        state.insert_account(address, AccountInfo::default().with_balance(U256::from(1)));
        let session = session(65536);
        let calls = Arc::new(AtomicUsize::new(0));
        let count = calls.clone();
        session.set_public_hook(
            &mut state,
            Some(Box::new(move |_state| {
                count.fetch_add(1, Ordering::Relaxed);
            })),
        );
        for block in 1..=3 {
            change(&mut state, address, 0, block, false, false);
            state.merge_transitions(BundleRetention::Reverts);
            let snapshot = session.capture(&mut state).unwrap().snapshot;
            assert_eq!(snapshot.storage(address, U256::ZERO), Some(U256::from(block)));
            let again = session.capture(&mut state).unwrap().snapshot;
            assert!(snapshot.accounts.ptr_eq(&again.accounts), "unchanged captures share roots");
            session.set_public_hook(&mut state, None);
        }
        assert_eq!(calls.load(Ordering::Relaxed), 1);
        assert!(session.state.lock().unwrap().attached);
        // A fresh State imports actual preloaded bundle changes on top of the same base.
        let bundle = state.take_bundle();
        drop(state);
        let mut continued = State::builder()
            .with_database(InMemoryDB::default())
            .with_bundle_prestate(bundle)
            .build();
        let snapshot = session.capture(&mut continued).unwrap().snapshot;
        assert_eq!(snapshot.storage(address, U256::ZERO), Some(U256::from(3)));
        assert!(snapshot.basic(address).unwrap().is_some());
    }

    #[test]
    fn budget_exhaustion_releases_roots_and_is_sticky() {
        let mut state = State::builder().with_database(InMemoryDB::default()).build();
        let address = Address::repeat_byte(4);
        state.insert_account_with_storage(
            address,
            AccountInfo::default(),
            (0..10000).map(|slot| (U256::from(slot), U256::from(slot))).collect(),
        );
        let large = session(512);
        assert!(large.capture(&mut state).is_none());
        assert_eq!(large.state.lock().unwrap().snapshot.size_bytes(), 0);
        state.cache.accounts.clear();
        assert!(large.capture(&mut state).is_none());

        let small = session(512);
        state.insert_account(address, AccountInfo::default().with_balance(U256::from(1)));
        let calls = Arc::new(AtomicUsize::new(0));
        let observed = calls.clone();
        small.set_public_hook(
            &mut state,
            Some(Box::new(move |_| {
                observed.fetch_add(1, Ordering::Relaxed);
            })),
        );
        change(&mut state, address, 0, 7, false, false);
        assert!(small.capture(&mut state).is_none(), "dirty tracking is also bounded");
        assert_eq!(small.estimated_bytes(), 0);
        assert_eq!(state.storage(address, U256::ZERO).unwrap(), U256::from(7));
        assert_eq!(calls.load(Ordering::Relaxed), 1, "fallback retains the public hook");
    }

    #[test]
    fn initial_cache_takes_precedence_over_bundle_and_base() {
        let first = Address::repeat_byte(5);
        let second = Address::repeat_byte(6);
        let mut base = InMemoryDB::default();
        for address in [first, second] {
            base.insert_account_info(address, AccountInfo::default().with_balance(U256::from(1)));
            base.insert_account_storage(address, U256::ZERO, U256::from(9)).unwrap();
        }
        let mut previous =
            State::builder().with_database(base.clone()).with_bundle_update().build();
        change(&mut previous, first, 0, 20, false, false);
        change(&mut previous, second, 0, 0, false, false);
        previous.merge_transitions(BundleRetention::Reverts);
        let mut state = State::builder()
            .with_database(base)
            .with_bundle_prestate(previous.take_bundle())
            .build();
        state.insert_account_with_storage(
            first,
            AccountInfo::default().with_balance(U256::from(3)),
            std::iter::once((U256::ZERO, U256::from(30))).collect(),
        );
        let session = session(65536);
        let snapshot = session.capture(&mut state).unwrap().snapshot;
        assert_eq!(snapshot.basic(first), Some(state.basic(first).unwrap()));
        assert_eq!(snapshot.storage(first, U256::ZERO), Some(U256::from(30)));
        assert_eq!(snapshot.storage(second, U256::ZERO), Some(U256::ZERO));
        assert_eq!(state.storage(second, U256::ZERO).unwrap(), U256::ZERO);
    }

    #[test]
    fn updating_one_slot_shares_all_other_account_storage() {
        let mut state = State::builder().with_database(InMemoryDB::default()).build();
        for byte in 1..=16 {
            state.insert_account_with_storage(
                Address::repeat_byte(byte),
                AccountInfo::default().with_balance(U256::from(1)),
                (0..128).map(|slot| (U256::from(slot), U256::from(slot))).collect(),
            );
        }
        let session = session(1024 * 1024);
        let first = session.capture(&mut state).unwrap().snapshot;
        change(&mut state, Address::repeat_byte(1), 64, 99, false, false);
        let second = session.capture(&mut state).unwrap().snapshot;
        for byte in 2..=16 {
            let address = Address::repeat_byte(byte);
            assert!(first.accounts[&address].storage.ptr_eq(&second.accounts[&address].storage));
        }
        assert_eq!(first.size_bytes(), second.size_bytes());
        assert_eq!(first.storage(Address::repeat_byte(1), U256::from(64)), Some(U256::from(64)));
        assert_eq!(second.storage(Address::repeat_byte(1), U256::from(64)), Some(U256::from(99)));
    }
    #[test]
    fn retained_versions_are_charged_once_and_overflow_disables_the_session() {
        let mut state = State::builder().with_database(InMemoryDB::default()).build();
        let address = Address::repeat_byte(10);
        state.insert_account_with_storage(
            address,
            AccountInfo::default().with_balance(U256::from(1)),
            (0..3).map(|slot| (U256::from(slot), U256::from(7))).collect(),
        );
        let session = session(2200);
        let first = session.capture(&mut state).unwrap();
        let duplicate = session.capture(&mut state).unwrap();
        assert!(Arc::ptr_eq(&first.snapshot, &duplicate.snapshot));
        let root_bytes = first.snapshot.size_bytes();
        assert_eq!(session.estimated_bytes(), root_bytes);
        change(&mut state, address, 0, 9, false, false);
        let second = session.capture(&mut state).unwrap();
        assert_eq!(session.estimated_bytes(), 2 * root_bytes);
        change(&mut state, address, 0, 10, false, false);
        assert!(!session.is_disabled(), "the dirty keys fit before publishing another version");
        assert!(session.capture(&mut state).is_none(), "three live versions exceed the budget");
        assert!(session.is_disabled());
        assert_eq!(first.snapshot.storage(address, U256::ZERO), Some(U256::from(7)));
        assert_eq!(second.snapshot.storage(address, U256::ZERO), Some(U256::from(9)));
        assert_eq!(
            session.estimated_bytes(),
            2 * root_bytes,
            "disabled sessions still account for worker leases"
        );
        drop(first);
        assert_eq!(session.estimated_bytes(), 2 * root_bytes);
        drop(duplicate);
        assert_eq!(session.estimated_bytes(), root_bytes);
        drop(second);
        assert_eq!(session.estimated_bytes(), 0);
        assert!(session.capture(&mut state).is_none());
    }
}
