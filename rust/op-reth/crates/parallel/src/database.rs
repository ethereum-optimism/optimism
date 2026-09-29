use revm::{
    Database,
    database_interface::DBErrorMarker,
    primitives::{Address, B256, U256},
    state::{AccountInfo, Bytecode},
};
use std::{
    cell::Cell,
    collections::BTreeMap,
    sync::{Arc, mpsc},
    time::{Duration, Instant},
};

/// A speculative failure always falls back to canonical execution.
#[derive(Debug, Clone, thiserror::Error)]
pub enum SpeculationError {
    /// This output belongs to an invalidated build generation.
    #[error("speculative build generation cancelled")]
    Cancelled,
    /// A configured memory/work limit was reached.
    #[error("speculation resource limit reached")]
    Limit,
    /// Invalid runtime configuration.
    #[error("{0}")]
    Configuration(&'static str),
    /// A worker or provider failed; the original error is re-obtained on canonical execution.
    #[error("{0}")]
    Worker(String),
}
impl DBErrorMarker for SpeculationError {}

/// Logical inputs consumed by one execution attempt, including absence and cached reads.
#[derive(Debug, Default)]
pub struct Dependencies {
    accounts: BTreeMap<Address, Option<AccountInfo>>,
    storage: BTreeMap<(Address, U256), U256>,
    code: BTreeMap<B256, Bytecode>,
    hashes: BTreeMap<u64, B256>,
    bytes: usize,
}

impl Dependencies {
    /// Checks dependencies against the latest committed prefix. The provider must still refer
    /// to the same parent/build generation; the owning executor prevents cross-block reuse.
    pub fn validate<DB: Database>(&self, database: &mut DB) -> Result<bool, DB::Error> {
        for (address, expected) in &self.accounts {
            if database.basic(*address)? != *expected {
                return Ok(false);
            }
        }
        for ((address, slot), expected) in &self.storage {
            if database.storage(*address, *slot)? != *expected {
                return Ok(false);
            }
        }
        for (hash, expected) in &self.code {
            if database.code_by_hash(*hash)? != *expected {
                return Ok(false);
            }
        }
        for (number, expected) in &self.hashes {
            if database.block_hash(*number)? != *expected {
                return Ok(false);
            }
        }
        Ok(true)
    }

    /// Retained read-set size, including bytecode.
    pub const fn size_bytes(&self) -> usize {
        self.bytes
    }
}

pub(crate) enum Request {
    Account(Address, mpsc::SyncSender<Result<Option<AccountInfo>, SpeculationError>>),
    Storage(Address, U256, mpsc::SyncSender<Result<U256, SpeculationError>>),
    Code(B256, mpsc::SyncSender<Result<Bytecode, SpeculationError>>),
    Hash(u64, mpsc::SyncSender<Result<B256, SpeculationError>>),
}

impl Request {
    pub(crate) fn respond<DB: Database>(self, database: &mut DB) {
        fn error(error: impl core::fmt::Display) -> SpeculationError {
            SpeculationError::Worker(error.to_string())
        }
        match self {
            Self::Account(address, reply) => {
                let _ = reply.send(
                    database
                        .basic(address)
                        .map(|info| {
                            info.map(|mut info| {
                                info.account_id = None;
                                info
                            })
                        })
                        .map_err(error),
                );
            }
            Self::Storage(address, slot, reply) => {
                let _ = reply.send(database.storage(address, slot).map_err(error));
            }
            Self::Code(hash, reply) => {
                let _ = reply.send(database.code_by_hash(hash).map_err(error));
            }
            Self::Hash(number, reply) => {
                let _ = reply.send(database.block_hash(number).map_err(error));
            }
        }
    }
}

pub(crate) enum Event<Output> {
    Read(Request),
    Finished(usize, Result<(Output, Dependencies), SpeculationError>),
}

/// A transaction-local database recording logical reads above its cache.
pub struct SpeculativeDatabase {
    request: Arc<dyn Fn(Request) -> Result<(), SpeculationError> + Send + Sync>,
    reads: Dependencies,
    limit: usize,
    exceeded: bool,
    generation: crate::ExecutionGeneration,
    read_wait: Cell<Duration>,
}

impl core::fmt::Debug for SpeculativeDatabase {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.debug_struct("SpeculativeDatabase")
            .field("reads", &self.reads)
            .field("limit", &self.limit)
            .finish_non_exhaustive()
    }
}

impl SpeculativeDatabase {
    pub(crate) fn new<Output: Send + 'static>(
        events: mpsc::SyncSender<Event<Output>>,
        limit: usize,
        generation: crate::ExecutionGeneration,
    ) -> Self {
        Self {
            request: Arc::new(move |request| {
                events
                    .send(Event::Read(request))
                    .map_err(|_| SpeculationError::Worker("coordinator disconnected".into()))
            }),
            reads: Default::default(),
            limit,
            exceeded: false,
            generation,
            read_wait: Cell::new(Duration::ZERO),
        }
    }

    const fn reserve(&mut self, bytes: usize) -> Result<(), SpeculationError> {
        if self.reads.bytes.saturating_add(bytes) > self.limit {
            self.exceeded = true;
            return Err(SpeculationError::Limit);
        }
        self.reads.bytes += bytes;
        Ok(())
    }

    pub(crate) fn finish(self) -> Result<Dependencies, SpeculationError> {
        if self.generation.is_cancelled() {
            Err(SpeculationError::Cancelled)
        } else if self.exceeded {
            Err(SpeculationError::Limit)
        } else {
            Ok(self.reads)
        }
    }

    pub(crate) const fn read_wait(&self) -> Duration {
        self.read_wait.get()
    }

    fn read<T>(
        &self,
        request: impl FnOnce(mpsc::SyncSender<Result<T, SpeculationError>>) -> Request,
    ) -> Result<T, SpeculationError> {
        if self.generation.is_cancelled() {
            return Err(SpeculationError::Cancelled);
        }
        let started = Instant::now();
        let (reply, receive) = mpsc::sync_channel(1);
        let result = (self.request)(request(reply)).and_then(|()| {
            receive
                .recv()
                .map_err(|_| SpeculationError::Worker("state read disconnected".into()))?
        });
        self.read_wait.set(self.read_wait.get().saturating_add(started.elapsed()));
        result
    }
}

impl Database for SpeculativeDatabase {
    type Error = SpeculationError;

    fn basic(&mut self, address: Address) -> Result<Option<AccountInfo>, Self::Error> {
        if let Some(info) = self.reads.accounts.get(&address) {
            return Ok(info.clone());
        }
        let info = self.read(|reply| Request::Account(address, reply))?;
        let code_bytes = info.as_ref().and_then(|info| info.code.as_ref()).map_or(0, Bytecode::len);
        self.reserve(256usize.saturating_add(code_bytes))?;
        self.reads.accounts.insert(address, info.clone());
        Ok(info)
    }

    fn storage(&mut self, address: Address, slot: U256) -> Result<U256, Self::Error> {
        if let Some(value) = self.reads.storage.get(&(address, slot)) {
            return Ok(*value);
        }
        // State::storage requires its account to be loaded; also record the account's existence.
        let _ = self.basic(address)?;
        let value = self.read(|reply| Request::Storage(address, slot, reply))?;
        self.reserve(128)?;
        self.reads.storage.insert((address, slot), value);
        Ok(value)
    }

    fn code_by_hash(&mut self, hash: B256) -> Result<Bytecode, Self::Error> {
        if let Some(code) = self.reads.code.get(&hash) {
            return Ok(code.clone());
        }
        let code = self.read(|reply| Request::Code(hash, reply))?;
        self.reserve(128usize.saturating_add(code.len()))?;
        self.reads.code.insert(hash, code.clone());
        Ok(code)
    }

    fn block_hash(&mut self, number: u64) -> Result<B256, Self::Error> {
        if let Some(hash) = self.reads.hashes.get(&number) {
            return Ok(*hash);
        }
        let hash = self.read(|reply| Request::Hash(number, reply))?;
        self.reserve(96)?;
        self.reads.hashes.insert(number, hash);
        Ok(hash)
    }
}
