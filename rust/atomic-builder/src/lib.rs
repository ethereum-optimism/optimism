//! Builder-only suspended OP EVM execution.
//!
//! A pause retains the interpreter stack, memory, gas, journal and warm accesses.
//! It is not a transaction boundary and does not commit state. Speculative remote
//! replies must pass an ordinary final replay before a candidate can be accepted.
//! This crate is not installed in the canonical execution or proof paths.

mod observation;
pub mod router;
pub use observation::{Application, Observation, Route};

use op_revm::{
    OpBuilder, OpContext, OpHaltReason,
    api::{builder::DefaultOpEvm, exec::OpError},
    handler::OpHandler,
};
use revm::{
    Database, ExecuteEvm,
    context_interface::{Cfg, ContextTr, JournalTr, Transaction, result::ExecutionResult},
    handler::{EthFrame, EvmTr, FrameResult, Handler, ItemOrResult},
    interpreter::{
        CallInput, CallOutcome, CallScheme, Gas, InitialAndFloorGas, InstructionResult,
        InterpreterResult, SharedMemory,
        interpreter::EthInterpreter,
        interpreter_action::{FrameInit, FrameInput},
    },
    primitives::{Address, Bytes, Log, U256, constants::CALL_STACK_LIMIT},
    state::EvmState,
};

type Evm<DB> = DefaultOpEvm<OpContext<DB>>;
type HandlerFor<DB> = OpHandler<Evm<DB>, OpError<OpContext<DB>>, EthFrame<EthInterpreter>>;

/// A finite discovery budget, additional to the transaction's EVM gas limit.
#[derive(Clone, Copy, Debug)]
pub struct Limits {
    /// Maximum number of intercepted calls, including repeated calls to one chain.
    pub calls: usize,
    /// Maximum input or output length at an intercepted call.
    pub bytes_per_call: usize,
}

/// An execution or discovery failure. No speculative state is committed on error.
#[derive(Debug, thiserror::Error)]
pub enum Error<DB: Database> {
    /// Normal OP EVM validation, execution or database error.
    #[error("EVM: {0}")]
    Evm(OpError<OpContext<DB>>),
    /// Synthetic/system envelopes require a different execution lifecycle.
    #[error("only ordinary user transactions can be suspended")]
    UnsupportedTransaction,
    /// The discovery budget was exhausted.
    #[error("atomic discovery limit exceeded")]
    Limit,
    /// A remote reply would manufacture gas or unsupported local side effects.
    #[error("invalid speculative call reply")]
    InvalidReply,
    /// Final replay diverged. The builder must discard this candidate, not iterate.
    #[error("canonical replay diverged from discovery")]
    Diverged,
}

/// A copied request; it does not borrow the suspended interpreter's memory.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Request {
    /// The caller of the intercepted local endpoint.
    pub caller: Address,
    /// The local endpoint's address, not the remote chain's target address.
    pub endpoint: Address,
    /// Input to the endpoint.
    pub input: Bytes,
    /// Gas forwarded to this local call after the CALL opcode's costs and cap.
    pub gas_limit: u64,
}

/// An untrusted discovery response for a zero-value CALL or STATICCALL.
///
/// `gas_used` is the cost of the local response-handling endpoint. Remote chain
/// execution has its own gas budget and must not be charged/refunded through it.
#[derive(Clone, Debug)]
pub struct Reply {
    /// Whether the local endpoint returns or reverts.
    pub success: bool,
    /// Raw return or revert data.
    pub output: Bytes,
    /// Local endpoint gas consumed; no gas refunds can be manufactured here.
    pub gas_used: u64,
}

/// A completed speculative transaction, which is never committed by this crate.
#[derive(Debug)]
pub struct Candidate {
    result: ExecutionResult<OpHaltReason>,
    state: EvmState,
    observations: Vec<Observation>,
    routes: Vec<Route>,
}

impl Candidate {
    /// Actual router calls, including router-level failures caught by an envelope.
    pub fn routes(&self) -> &[Route] {
        &self.routes
    }

    /// Application effects captured independently of envelope/preload accounting.
    pub fn observations(&self) -> &[Observation] {
        &self.observations
    }

    /// Consume a canonical run after the coordinator validates the entire bundle.
    /// This does not assert cross-chain validity by itself.
    pub(crate) fn into_result(self) -> Verified {
        Verified { result: self.result, state: self.state }
    }

    /// Inspect the speculative receipt, logs and return data.
    pub const fn result(&self) -> &ExecutionResult<OpHaltReason> {
        &self.result
    }

    /// Replay exactly once using ordinary OP execution and accept only an exact match.
    ///
    /// The caller supplies the canonical transaction, witnesses and access list
    /// against the pinned block-prefix state. This checks local execution only;
    /// cross-chain message matching and full block validation remain mandatory.
    /// This method performs no retry or fixed-point iteration.
    pub fn verify<DB: Database>(self, context: OpContext<DB>) -> Result<Verified, Error<DB>> {
        if !matches!(context.tx().tx_type(), 0..=2 | 4) {
            return Err(Error::UnsupportedTransaction);
        }
        let canonical = context.build_op().replay().map_err(Error::Evm)?;
        if self.result != canonical.result || self.state != canonical.state {
            return Err(Error::Diverged);
        }
        Ok(Verified { result: canonical.result, state: canonical.state })
    }
}

/// A locally replayed result. Cross-chain and block validation are still required.
#[derive(Debug)]
pub struct Verified {
    /// Canonical transaction result.
    pub result: ExecutionResult<OpHaltReason>,
    /// Canonical state changes, for the caller's isolated candidate block.
    pub state: EvmState,
}

/// Execution either needs one remote response or has finished discovery.
#[derive(Debug)]
pub enum Progress<DB: Database> {
    /// Owns the live EVM; dropping it cancels discovery without committing state.
    Paused(Box<Paused<DB>>),
    /// The transaction finished and still requires canonical replay.
    Complete(Candidate),
}

/// A single-use continuation retaining all active frames and the transaction journal.
#[derive(Debug)]
pub struct Paused<DB: Database> {
    machine: Machine<DB>,
    next: FrameInit,
    request: Request,
}

impl<DB: Database> Paused<DB> {
    /// Logs emitted so far, including the request that caused this pause.
    pub fn logs(&self) -> &[Log] {
        self.machine.evm.ctx_ref().journal().logs()
    }

    /// Warm a discovered inbox checksum before running the real validation call.
    /// Canonical replay must supply it in the actual transaction access list.
    pub fn warm_slot(&mut self, address: Address, slot: U256) {
        self.machine.evm.ctx().journal_mut().warm_access_list(
            std::iter::once((address, std::iter::once(slot).collect())).collect(),
        );
    }

    /// Execute only this read-only getter in a supplied, isolated tape snapshot.
    /// No user prefix is repeated. The returned local gas cost is measured by REVM.
    pub fn quote(&self, mut context: OpContext<DB>) -> Result<Reply, Error<DB>> {
        let FrameInput::Call(original) = &self.next.frame_input else { unreachable!() };
        if original.scheme != CallScheme::StaticCall {
            return Err(Error::InvalidReply);
        }
        let mut call = original.clone();
        call.input = CallInput::Bytes(self.request.input.clone());
        context
            .journal_mut()
            .load_account_with_code(call.caller)
            .map_err(|e| Error::Evm(e.into()))?;
        context
            .journal_mut()
            .load_account_with_code(call.target_address)
            .map_err(|e| Error::Evm(e.into()))?;
        let spec = (*context.cfg().spec()).into();
        context.journal_mut().set_spec_id(spec);
        let mut evm = context.build_op();
        let result = HandlerFor::<DB>::new()
            .run_exec_loop(
                &mut evm,
                FrameInit {
                    depth: 0,
                    memory: SharedMemory::new(),
                    frame_input: FrameInput::Call(call),
                },
            )
            .map_err(Error::Evm)?;
        if !result.instruction_result().is_ok() {
            return Err(Error::InvalidReply);
        }
        Ok(Reply {
            success: true,
            output: result.interpreter_result().output.clone(),
            gas_used: result.gas().total_gas_spent(),
        })
    }

    /// The endpoint call where execution yielded.
    pub const fn request(&self) -> &Request {
        &self.request
    }

    /// Resume by executing the original endpoint normally, without substitutions.
    pub fn resume(self) -> Result<Progress<DB>, Error<DB>> {
        let Self { mut machine, next, .. } = self;
        let next = machine.evm.frame_init(next).map_err(|e| Error::Evm(e.into()))?.map_item(|_| ());
        machine.drive(next)
    }

    /// Inject an untrusted discovery reply and resume the waiting caller.
    ///
    /// REVM performs the normal return-data copy, success-stack update and gas
    /// return. The skipped local call makes no storage writes or logs. Any such
    /// omitted effects, wrong gas charges or fabricated replies fail final replay.
    pub fn resolve(self, reply: Reply) -> Result<Progress<DB>, Error<DB>> {
        let Self { mut machine, next, .. } = self;
        let FrameInput::Call(call) = next.frame_input else {
            unreachable!("only calls can be paused");
        };
        if reply.gas_used > call.gas_limit || reply.output.len() > machine.limits.bytes_per_call {
            return Err(Error::InvalidReply);
        }
        let mut gas = Gas::new(call.gas_limit);
        gas.set_reservoir(call.reservoir);
        if !gas.record_regular_cost(reply.gas_used) {
            return Err(Error::InvalidReply);
        }
        // Normal successful CALL initialization touches the destination even for
        // a zero-value transfer. On REVERT that touch would be rolled back.
        if reply.success &&
            machine
                .evm
                .ctx()
                .journal_mut()
                .transfer_loaded(call.caller, call.target_address, U256::ZERO)
                .is_some()
        {
            return Err(Error::InvalidReply);
        }
        let status =
            if reply.success { InstructionResult::Return } else { InstructionResult::Revert };
        let result = InterpreterResult::new(status, reply.output, gas);
        machine.drive(ItemOrResult::Result(FrameResult::Call(CallOutcome::new(
            result,
            call.return_memory_offset,
        ))))
    }
}

#[derive(Debug)]
struct Machine<DB: Database> {
    evm: Evm<DB>,
    endpoints: Vec<Endpoint>,
    application: Option<Application>,
    active: Option<observation::Active>,
    observations: Vec<Observation>,
    route: Option<(usize, Request)>,
    routes: Vec<Route>,
    limits: Limits,
    calls: usize,
    initial_gas: InitialAndFloorGas,
    authorization_refund: i64,
    reservoir: u64,
}

/// A local address and optional selector at which discovery may yield.
#[derive(Clone, Debug)]
pub struct Endpoint {
    /// Local contract address.
    pub address: Address,
    /// None intercepts all eligible calls to the address.
    pub selector: Option<[u8; 4]>,
}

/// Start an ordinary user transaction once and stop at registered local endpoints.
///
/// Only zero-value CALLs and STATICCALLs are intercepted. All other calls and creates
/// execute normally. Use isolated database snapshots: although this runner never
/// invokes `DatabaseCommit`, a database implementation can itself have side effects.
pub fn discover<DB: Database>(
    context: OpContext<DB>,
    endpoints: Vec<Address>,
    limits: Limits,
) -> Result<Progress<DB>, Error<DB>> {
    discover_with(
        context,
        endpoints.into_iter().map(|address| Endpoint { address, selector: None }).collect(),
        None,
        limits,
    )
}

/// Start discovery with selector filters and optional application observation.
/// With no endpoints this runs canonical bytecode without substitutions.
pub fn discover_with<DB: Database>(
    context: OpContext<DB>,
    endpoints: Vec<Endpoint>,
    application: Option<Application>,
    limits: Limits,
) -> Result<Progress<DB>, Error<DB>> {
    // Deposits and PostExec are deliberately outside this builder-only lifecycle.
    if !matches!(context.tx().tx_type(), 0..=2 | 4) {
        return Err(Error::UnsupportedTransaction);
    }
    let mut evm = context.build_op();
    let mut handler = HandlerFor::<DB>::new();
    let mut initial_gas = handler.validate(&mut evm).map_err(Error::Evm)?;
    let authorization_refund =
        handler.pre_execution(&mut evm, &mut initial_gas).map_err(Error::Evm)? as i64;
    let (gas_limit, reservoir) = initial_gas
        .initial_gas_and_reservoir(evm.ctx().tx().gas_limit(), evm.ctx().cfg().tx_gas_limit_cap());
    let first = handler.first_frame_input(&mut evm, gas_limit, reservoir).map_err(Error::Evm)?;
    let mut machine = Machine {
        evm,
        endpoints,
        application,
        active: None,
        observations: Vec::new(),
        route: None,
        routes: Vec::new(),
        limits,
        calls: 0,
        initial_gas,
        authorization_refund,
        reservoir,
    };
    machine.observe_init(&first);
    let next = machine.evm.frame_init(first).map_err(|e| Error::Evm(e.into()))?.map_item(|_| ());
    if let ItemOrResult::Result(result) = &next {
        machine.observe_return(0, result)?;
    }
    machine.drive(next)
}

impl<DB: Database> Machine<DB> {
    fn drive(mut self, mut next: ItemOrResult<(), FrameResult>) -> Result<Progress<DB>, Error<DB>> {
        loop {
            if let ItemOrResult::Result(result) = next {
                let completed = if self.evm.frame_stack().index().is_none() {
                    Some(result)
                } else {
                    self.evm.frame_return_result(result).map_err(|e| Error::Evm(e.into()))?
                };
                if let Some(mut result) = completed {
                    let mut handler = HandlerFor::<DB>::new();
                    handler
                        .last_frame_result(&mut self.evm, self.reservoir, &mut result)
                        .map_err(Error::Evm)?;
                    let gas = handler
                        .post_execution(
                            &mut self.evm,
                            &mut result,
                            self.initial_gas,
                            self.authorization_refund,
                        )
                        .map_err(Error::Evm)?;
                    let result =
                        handler.execution_result(&mut self.evm, result, gas).map_err(Error::Evm)?;
                    let state = self.evm.ctx().journal_mut().finalize();
                    return Ok(Progress::Complete(Candidate {
                        result,
                        state,
                        observations: self.observations,
                        routes: self.routes,
                    }));
                }
            }
            next = match self.evm.frame_run().map_err(|e| Error::Evm(e.into()))? {
                ItemOrResult::Result(result) => {
                    let depth = self.evm.frame_stack().get().depth;
                    self.observe_return(depth, &result)?;
                    ItemOrResult::Result(result)
                }
                ItemOrResult::Item(init) => {
                    if let FrameInput::Call(call) = &init.frame_input &&
                        self.endpoints.iter().any(|endpoint| {
                            endpoint.address == call.bytecode_address &&
                                endpoint.selector.is_none_or(|selector| {
                                    call.input.bytes(self.evm.ctx_ref()).starts_with(&selector)
                                })
                        }) &&
                        matches!(call.scheme, CallScheme::Call | CallScheme::StaticCall) &&
                        !call.transfers_value() &&
                        !call.charged_new_account_state_gas &&
                        init.depth <= CALL_STACK_LIMIT as usize
                    {
                        if self.calls >= self.limits.calls ||
                            call.input.len() > self.limits.bytes_per_call
                        {
                            return Err(Error::Limit);
                        }
                        self.calls += 1;
                        let request = Request {
                            caller: call.caller,
                            endpoint: call.bytecode_address,
                            input: call.input.bytes(self.evm.ctx()),
                            gas_limit: call.gas_limit,
                        };
                        return Ok(Progress::Paused(Box::new(Paused {
                            machine: self,
                            next: init,
                            request,
                        })));
                    }
                    let depth = init.depth;
                    self.observe_init(&init);
                    let next = self
                        .evm
                        .frame_init(init)
                        .map_err(|e| Error::Evm(e.into()))?
                        .map_item(|_| ());
                    if let ItemOrResult::Result(result) = &next {
                        self.observe_return(depth, result)?;
                    }
                    next
                }
            };
        }
    }
}

impl<DB: Database> Machine<DB> {
    fn observe_init(&mut self, init: &FrameInit) {
        if let FrameInput::Call(call) = &init.frame_input {
            if self.active.is_none() &&
                self.application.as_ref().is_some_and(|app| app.matches(call))
            {
                self.active = Some(observation::Active::new(init.depth, call, self.evm.ctx_ref()));
            }
            if self.route.is_none() &&
                self.application.as_ref().is_some_and(|app| {
                    call.caller == app.entry_caller && call.target_address == app.router
                })
            {
                self.route = Some((
                    init.depth,
                    Request {
                        caller: call.caller,
                        endpoint: call.target_address,
                        input: call.input.bytes(self.evm.ctx_ref()),
                        gas_limit: call.gas_limit,
                    },
                ));
            }
        }
    }
    fn observe_return(&mut self, depth: usize, result: &FrameResult) -> Result<(), Error<DB>> {
        if self.route.as_ref().is_some_and(|(at, _)| *at == depth) {
            let (_, request) = self.route.take().expect("checked");
            self.routes.push(Route { request, result: result.interpreter_result().clone() });
        }
        if self.active.as_ref().is_some_and(|active| active.depth == depth) {
            let active = self.active.take().expect("checked");
            self.observations.push(active.finish(result, self.evm.ctx())?);
        }
        Ok(())
    }
}
