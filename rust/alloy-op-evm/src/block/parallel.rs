use super::{ParallelCandidate, RawTransactionOutput};
use crate::OpTx;
use alloc::{
    collections::{BTreeMap, BTreeSet},
    sync::Arc,
    vec::Vec,
};
use alloy_primitives::{Address, B256};
use reth_optimism_parallel::{
    Dependencies, ExecutionGeneration, ExecutionMode, ParallelRuntime, SpeculationError,
    SpeculativeDatabase,
};
use revm::Database;

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
/// a window is fully drained before returning, so workers cannot outlive its parent/build.
pub(crate) struct ParallelBlockState<H> {
    pub(super) runtime: Arc<ParallelRuntime>,
    pub(super) worker: Arc<Worker<H>>,
    outputs: BTreeMap<B256, (OpTx, WorkerOutput<H>, Dependencies)>,
    window_senders: BTreeSet<Address>,
    serial_sender: Option<Address>,
    attempted: BTreeSet<B256>,
    remaining_gas: u64,
    generation: ExecutionGeneration,
}

impl<H> core::fmt::Debug for ParallelBlockState<H> {
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
        }
    }

    pub(super) fn clear(&mut self) {
        self.outputs.clear();
        self.window_senders.clear();
        self.serial_sender = None;
        self.generation.cancel();
        self.generation = ExecutionGeneration::default();
    }

    pub(crate) const fn candidates_changed(&mut self) {
        self.serial_sender = None;
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
    ) -> Result<Option<WorkerOutput<H>>, DB::Error> {
        let Some(context) = context else {
            return Ok(None);
        };
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
            let outputs = self.runtime.execute_in_generation(
                database,
                jobs,
                self.worker.clone(),
                &self.generation,
            );
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

impl<H> Drop for ParallelBlockState<H> {
    fn drop(&mut self) {
        self.generation.cancel();
    }
}
