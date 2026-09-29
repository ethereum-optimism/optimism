use super::{ParallelCandidate, RawTransactionOutput};
use crate::OpTx;
use alloc::{
    collections::{BTreeMap, BTreeSet},
    sync::Arc,
    vec::Vec,
};
use alloy_primitives::{Address, B256};
use reth_optimism_parallel::{
    Dependencies, ExecutionGeneration, ExecutionMode, ExecutionPipeline, ExecutionScheduler,
    ParallelRuntime, SpeculationError, SpeculativeDatabase, StateReads,
};
use revm::Database;

pub(super) type SnapshotCapture<DB> = (
    Arc<reth_optimism_parallel::SnapshotSession>,
    fn(
        &mut DB,
        &reth_optimism_parallel::SnapshotSession,
    ) -> Option<reth_optimism_parallel::ReadWindow>,
);

pub(super) type Worker<H> = dyn Fn(
        (OpTx, ExecutionContext),
        &mut SpeculativeDatabase,
    ) -> Result<WorkerOutput<H>, SpeculationError>
    + Send
    + Sync;

#[derive(Clone)]
pub(super) struct ExecutionContext {
    pub(super) l1_block_info: Option<op_revm::L1BlockInfo>,
    pub(super) producing: bool,
    pub(super) environment: alloy_evm::EvmEnv<op_revm::OpSpecId>,
}

pub(super) struct WorkerOutput<H> {
    pub(super) raw: RawTransactionOutput<H>,
    pub(super) environment: alloy_evm::EvmEnv<op_revm::OpSpecId>,
    pub(super) warm_precompiles: Vec<Address>,
    pub(super) call_targets: Vec<Address>,
}

/// Precompile implementations are not necessarily Send/Sync or reproducible by a factory.
/// Track every call (including reverted calls) so customized maps can fall back canonically.
#[derive(Default)]
pub(super) struct CallTargets {
    pub(super) addresses: BTreeSet<Address>,
    pub(super) exceeded: bool,
}

impl<CTX: revm::context_interface::ContextTr> revm::Inspector<CTX> for CallTargets {
    fn call(
        &mut self,
        _context: &mut CTX,
        inputs: &mut revm::interpreter::CallInputs,
    ) -> Option<revm::interpreter::CallOutcome> {
        if self.addresses.len() < 4096 {
            self.addresses.insert(inputs.bytecode_address);
        } else {
            self.exceeded = true;
        }
        None
    }
}

/// All speculative data is block-local. Dropping the executor invalidates the entire generation;
/// destruction drains rolling work, so workers cannot outlive their parent/build.
pub(crate) struct ParallelBlockState<H: Send + 'static> {
    pub(super) runtime: Arc<ParallelRuntime>,
    pub(super) worker: Arc<Worker<H>>,
    outputs: BTreeMap<B256, (OpTx, WorkerOutput<H>, Dependencies)>,
    window_senders: BTreeSet<Address>,
    serial_sender: Option<Address>,
    attempted: BTreeSet<B256>,
    remaining_gas: u64,
    generation: ExecutionGeneration,
    pipeline: Option<ExecutionPipeline<(OpTx, ExecutionContext), WorkerOutput<H>>>,
    session: Option<Arc<reth_optimism_parallel::SnapshotSession>>,
    identities: BTreeMap<B256, OpTx>,
    handled: BTreeSet<B256>,
    invalid_from: BTreeMap<Address, u64>,
    authoritative: Option<B256>,
    environment: Option<alloy_evm::EvmEnv<op_revm::OpSpecId>>,
    rolling_disabled: bool,
}

impl<H: Send + 'static> core::fmt::Debug for ParallelBlockState<H> {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.debug_struct("ParallelBlockState")
            .field("runtime", &self.runtime)
            .field("outputs", &self.outputs.len())
            .field("remaining_gas", &self.remaining_gas)
            .finish_non_exhaustive()
    }
}

impl<H: Send + 'static> ParallelBlockState<H> {
    pub(super) fn new(runtime: Arc<ParallelRuntime>, worker: Arc<Worker<H>>) -> Self {
        let remaining_gas = runtime.config().max_speculative_gas;
        Self {
            runtime,
            worker,
            outputs: BTreeMap::new(),
            window_senders: BTreeSet::new(),
            serial_sender: None,
            attempted: BTreeSet::new(),
            remaining_gas,
            generation: ExecutionGeneration::default(),
            pipeline: None,
            session: None,
            identities: BTreeMap::new(),
            handled: BTreeSet::new(),
            invalid_from: BTreeMap::new(),
            authoritative: None,
            environment: None,
            rolling_disabled: false,
        }
    }

    pub(crate) fn clear(&mut self) {
        self.outputs.clear();
        self.window_senders.clear();
        self.serial_sender = None;
        self.generation.cancel();
        self.drain_pipeline();
        self.identities.clear();
        self.authoritative = None;
        self.environment = None;
        self.generation = ExecutionGeneration::default();
    }

    fn drain_pipeline(&mut self) {
        if let Some(pipeline) = self.pipeline.take() {
            pipeline.cancel_and_drain();
            // Reader teardown can be the last event in a block/subblock. Preserve that failure
            // in the retained session even when no subsequent `take` polls pipeline health.
            if pipeline.source_failed() {
                if let Some(session) = &self.session {
                    session.disable("provider_failure");
                }
                self.rolling_disabled = true;
            }
        }
    }

    pub(crate) fn candidates_changed(&mut self, candidates: &[ParallelCandidate]) {
        self.serial_sender = None;
        let keep: BTreeSet<_> = candidates.iter().map(|candidate| candidate.hash).collect();
        self.handled.retain(|hash| keep.contains(hash));
        if let Some(pipeline) = &self.pipeline {
            for hash in self.identities.keys().filter(|hash| !keep.contains(*hash)) {
                if self.authoritative != Some(*hash) {
                    pipeline.retire(*hash);
                }
            }
        }
    }

    pub(crate) fn rejected(&mut self, hash: B256, descendants: Option<(Address, u64)>) {
        if self.handled.len() < 4096 {
            self.handled.insert(hash);
        }
        self.outputs.remove(&hash);
        if let Some((sender, nonce)) = descendants {
            if self.invalid_from.len() < 4096 {
                self.invalid_from
                    .entry(sender)
                    .and_modify(|old| *old = (*old).min(nonce))
                    .or_insert(nonce);
            }
            self.outputs
                .retain(|_, (tx, _, _)| tx.0.base.caller != sender || tx.0.base.nonce < nonce);
        }
        if let Some(pipeline) = &self.pipeline {
            pipeline.retire(hash);
            if let Some((sender, nonce)) = descendants {
                for (hash, tx) in &self.identities {
                    if tx.0.base.caller == sender && tx.0.base.nonce >= nonce {
                        pipeline.retire(*hash);
                    }
                }
            }
        }
        if self.authoritative == Some(hash) {
            self.authoritative = None;
        }
    }

    pub(super) fn retire_authoritative(&mut self) {
        if let Some(hash) = self.authoritative.take() {
            self.rejected(hash, None);
        }
    }

    pub(super) fn selected(&mut self, hash: B256) {
        self.retire_authoritative();
        self.authoritative = Some(hash);
    }

    fn rolling_healthy<DB: Database>(&mut self, capture: Option<&SnapshotCapture<DB>>) -> bool {
        let failed = self.pipeline.as_ref().is_some_and(ExecutionPipeline::source_failed);
        if failed {
            if let Some((session, _)) = capture {
                session.disable("provider_failure");
            }
        }
        if failed || capture.is_some_and(|(session, _)| session.is_disabled()) {
            self.drain_pipeline();
            self.identities.clear();
            self.rolling_disabled = true;
        }
        !self.rolling_disabled
    }

    /// Publish a fresh snapshot only after canonical settlement and state commit are complete.
    pub(super) fn committed<DB: Database>(
        &mut self,
        previews: &[ParallelCandidate],
        context: Option<ExecutionContext>,
        database: &mut DB,
        capture: Option<&SnapshotCapture<DB>>,
    ) {
        self.retire_authoritative();
        if self.runtime.config().scheduler != ExecutionScheduler::Rolling ||
            self.runtime.config().state_reads != StateReads::Auto ||
            self.serial_sender.is_some() ||
            !self.rolling_healthy(capture)
        {
            return;
        }
        if let Some(context) = context {
            if self.environment.as_ref().is_some_and(|old| *old != context.environment) {
                self.clear();
            }
            self.refill(None, previews, &context, database, capture);
        } else {
            self.clear();
        }
    }

    fn refill<DB: Database>(
        &mut self,
        selected: Option<(B256, &OpTx)>,
        previews: &[ParallelCandidate],
        context: &ExecutionContext,
        database: &mut DB,
        capture: Option<&SnapshotCapture<DB>>,
    ) {
        let Some((session, capture)) = capture else { return };
        if let Some(pipeline) = &self.pipeline {
            self.identities.retain(|hash, _| pipeline.contains(*hash));
        }
        let occupied = self.pipeline.as_ref().map_or(0, ExecutionPipeline::outstanding);
        let slots = self.runtime.config().max_in_flight.saturating_sub(occupied);
        if slots == 0 {
            return;
        }
        let mut senders: BTreeSet<_> =
            self.identities.values().map(|tx| tx.0.base.caller).collect();
        let first = selected.map(|(hash, tx)| ParallelCandidate { hash, transaction: tx.clone() });
        let mut jobs = Vec::new();
        let mut gas = self.remaining_gas;
        for candidate in first.iter().chain(previews) {
            // Hints preceding an already handled deposit are not a new boundary. An unhandled
            // structural transaction terminates lookahead even if other senders are eligible.
            if self.handled.contains(&candidate.hash) ||
                self.invalid_from
                    .get(&candidate.transaction.0.base.caller)
                    .is_some_and(|nonce| candidate.transaction.0.base.nonce >= *nonce)
            {
                continue;
            }
            if matches!(candidate.transaction.0.base.tx_type, 0x7e | 0x7d) {
                break;
            }
            if self.attempted.contains(&candidate.hash) ||
                !senders.insert(candidate.transaction.0.base.caller)
            {
                continue;
            }
            let budget = candidate.transaction.0.base.gas_limit.max(21_000);
            if budget > gas {
                break;
            }
            gas -= budget;
            jobs.push(candidate.clone());
            if jobs.len() == slots {
                break;
            }
        }
        if jobs.is_empty() {
            return;
        }
        if occupied == 0 && jobs.len() == 1 && self.runtime.config().mode == ExecutionMode::Parallel
        {
            if let Some((_, tx)) = selected {
                self.serial_sender = Some(tx.0.base.caller);
            }
            return;
        }
        let Some(reads) = capture(database, session) else {
            self.drain_pipeline();
            self.identities.clear();
            self.rolling_disabled = true;
            return;
        };
        self.session = Some(session.clone());
        let pipeline = self.pipeline.get_or_insert_with(|| {
            ExecutionPipeline::new(
                &self.runtime,
                self.worker.clone(),
                self.generation.clone(),
                &reads,
            )
        });
        let work = jobs
            .iter()
            .map(|candidate| {
                (candidate.hash, (candidate.transaction.clone(), context.clone()), reads.clone())
            })
            .collect();
        if pipeline.submit_batch(work).is_ok() {
            for candidate in jobs {
                self.remaining_gas -= candidate.transaction.0.base.gas_limit.max(21_000);
                self.attempted.insert(candidate.hash);
                self.identities.insert(candidate.hash, candidate.transaction);
            }
        }
        self.environment = Some(context.environment.clone());
    }

    #[cfg(feature = "metrics")]
    pub(super) fn attempted(&self, hash: B256) -> bool {
        self.attempted.contains(&hash)
    }

    pub(super) fn take<DB: Database>(
        &mut self,
        hash: B256,
        transaction: &OpTx,
        previews: &[ParallelCandidate],
        context: Option<ExecutionContext>,
        database: &mut DB,
        capture: Option<&SnapshotCapture<DB>>,
    ) -> Result<Option<WorkerOutput<H>>, DB::Error> {
        let Some(context) = context else {
            self.clear();
            return Ok(None);
        };
        if capture.is_some_and(|(session, _)| session.generation().is_cancelled()) {
            self.clear();
            return Ok(None);
        }
        if self.environment.as_ref().is_some_and(|environment| *environment != context.environment)
        {
            self.clear();
        }
        if self.runtime.config().scheduler == ExecutionScheduler::Rolling &&
            self.runtime.config().state_reads == StateReads::Auto &&
            capture.is_some() &&
            self.rolling_healthy(capture)
        {
            // A previous prepare without a commit is a rejected candidate, not committed state.
            self.selected(hash);
            if self.serial_sender == Some(transaction.0.base.caller) {
                return Ok(None);
            }
            self.serial_sender = None;
            // A selector can choose outside our lookahead. Never enqueue that new head behind
            // unrelated running hints: canonical execution makes progress immediately.
            if self
                .pipeline
                .as_ref()
                .is_some_and(|pipeline| pipeline.outstanding() > 0 && !pipeline.contains(hash))
            {
                return Ok(None);
            }
            self.refill(Some((hash, transaction)), previews, &context, database, capture);
            if !self.rolling_disabled {
                let result = self.pipeline.as_ref().and_then(|pipeline| pipeline.take(hash));
                if !self.rolling_healthy(capture) {
                    return Ok(None);
                }
                if let Some(Ok((output, dependencies))) = result {
                    if self
                        .identities
                        .get(&hash)
                        .is_some_and(|expected| expected.0 == transaction.0) &&
                        output.environment == context.environment &&
                        dependencies.validate(database)? &&
                        !self.generation.is_cancelled() &&
                        !capture.is_some_and(|(session, _)| session.generation().is_cancelled())
                    {
                        return Ok(Some(output));
                    }
                    self.runtime.record_conflict();
                }
                return Ok(None);
            }
        }
        if !self.outputs.contains_key(&hash) && !self.attempted.contains(&hash) {
            // Avoid rescanning the same lookahead for every transaction of a pure nonce chain.
            // A different selected sender or new candidate hints reopen speculation.
            if self.serial_sender == Some(transaction.0.base.caller) {
                return Ok(None);
            }
            self.serial_sender = None;
            // A descendant can run canonically while the window still holds reusable work for
            // other senders. Do not discard that work just to speculate on the descendant.
            if !self.outputs.is_empty() && self.window_senders.contains(&transaction.0.base.caller)
            {
                return Ok(None);
            }
            self.outputs.clear();
            self.window_senders.clear();
            let start = previews.iter().position(|candidate| candidate.hash == hash);
            let subsequent = start.map(|index| &previews[index + 1..]).unwrap_or(previews);
            let first = ParallelCandidate { hash, transaction: transaction.clone() };
            let mut jobs = Vec::new();
            let mut identities = Vec::new();
            let mut senders = BTreeSet::new();
            let mut remaining_gas = self.remaining_gas;
            for candidate in core::iter::once(&first).chain(subsequent) {
                let tx = &candidate.transaction;
                // No speculation across deposits or structural post-exec transactions.
                if matches!(tx.0.base.tx_type, 0x7e | 0x7d) {
                    break;
                }
                if jobs.len() == self.runtime.config().max_in_flight {
                    break;
                }
                if self.attempted.contains(&candidate.hash) {
                    continue;
                }
                // All jobs see the same committed nonce. Only the first candidate per sender
                // can be useful in this window; skipped hints consume no speculative budget.
                if senders.contains(&tx.0.base.caller) {
                    continue;
                }
                let budget = tx.0.base.gas_limit.max(21_000);
                if budget > remaining_gas {
                    break;
                }
                remaining_gas -= budget;
                senders.insert(tx.0.base.caller);
                identities.push((candidate.hash, tx.clone()));
                jobs.push((tx.clone(), context.clone()));
            }
            // A singleton cannot overlap EVM execution. Shadow mode still dispatches it to
            // compare against the reference, but operational parallel mode avoids the round trip.
            if jobs.is_empty() ||
                (jobs.len() == 1 && self.runtime.config().mode == ExecutionMode::Parallel)
            {
                if jobs.len() == 1 {
                    self.serial_sender = Some(transaction.0.base.caller);
                }
                return Ok(None);
            }
            self.remaining_gas = remaining_gas;
            self.window_senders = senders;
            self.attempted.extend(identities.iter().map(|(hash, _)| *hash));
            let reads = capture.and_then(|(session, capture)| capture(database, session));
            let outputs = self.runtime.execute_with_reads(
                database,
                jobs,
                self.worker.clone(),
                &self.generation,
                reads,
            );
            if outputs.iter().any(|output| matches!(output, Err(SpeculationError::Source(_)))) {
                if let Some((session, _)) = capture {
                    session.disable("provider_failure");
                }
            }
            for ((hash, tx), output) in identities.into_iter().zip(outputs) {
                if let Ok((raw, dependencies)) = output {
                    self.outputs.insert(hash, (tx, raw, dependencies));
                }
            }
        }
        let Some((expected, output, dependencies)) = self.outputs.remove(&hash) else {
            return Ok(None);
        };
        if expected.0 != transaction.0 ||
            output.environment != context.environment ||
            !dependencies.validate(database)?
        {
            self.runtime.record_conflict();
            return Ok(None);
        }
        Ok(Some(output))
    }
}

/// Compare execution changes, excluding transaction-local warmth and journal sequence numbers.
/// Only touched accounts and changed slots are published to State/DatabaseCommit.
pub(super) fn equal_state(a: &revm::state::EvmState, b: &revm::state::EvmState) -> bool {
    let relevant = |state: &revm::state::EvmState| {
        state.values().filter(|account| account.is_touched()).count()
    };
    relevant(a) == relevant(b) &&
        a.iter().filter(|(_, account)| account.is_touched()).all(|(address, account)| {
            let Some(other) = b.get(address).filter(|account| account.is_touched()) else {
                return false;
            };
            account.info == other.info &&
                account.info.code == other.info.code &&
                account.is_created() == other.is_created() &&
                account.is_selfdestructed() == other.is_selfdestructed() &&
                account.is_loaded_as_not_existing() == other.is_loaded_as_not_existing() &&
                account.changed_storage_slots().count() == other.changed_storage_slots().count() &&
                account.changed_storage_slots().all(|(slot, value)| {
                    other.storage.get(slot).is_some_and(|other| {
                        value.original_value == other.original_value &&
                            value.present_value == other.present_value
                    })
                })
        })
}

impl<H: Send + 'static> Drop for ParallelBlockState<H> {
    fn drop(&mut self) {
        self.generation.cancel();
        self.drain_pipeline();
    }
}
