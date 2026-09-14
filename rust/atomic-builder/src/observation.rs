//! Application-boundary observations for envelopes whose witness prelude changes.
use crate::{Error, Request};
use op_revm::OpContext;
use revm::{
    Database,
    context_interface::{ContextTr, JournalTr},
    handler::FrameResult,
    interpreter::{CallInputs, InterpreterResult},
    primitives::{Address, B256, Log, U256},
    state::EvmState,
};
use std::collections::BTreeMap;

/// Select calls made by a router into application code.
#[derive(Clone, Debug)]
pub struct Application {
    /// Router whose outgoing calls delimit application execution.
    pub router: Address,
    /// Authenticated account expected to invoke the router entry point.
    pub entry_caller: Address,
    /// Infrastructure callees excluded from application observations.
    pub infrastructure: Vec<Address>,
}
impl Application {
    pub(crate) fn matches(&self, call: &CallInputs) -> bool {
        call.caller == self.router &&
            call.target_address != self.router &&
            !self.infrastructure.contains(&call.target_address)
    }
}

/// Exact application execution, excluding envelope fees and tape preload/cleanup.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Observation {
    /// Input and gas supplied at the application boundary.
    pub request: Request,
    /// Full status, return data, remaining gas and refund accounting.
    pub result: InterpreterResult,
    /// Application logs, including nested router/inbox logs.
    pub logs: Vec<Log>,
    effects: BTreeMap<Address, Effect>,
}
#[derive(Clone, Debug, PartialEq, Eq)]
struct Effect {
    balance_delta: U256,
    nonce_delta: u64,
    code: Option<B256>,
    created: bool,
    destroyed: bool,
    touched_empty: bool,
    storage: BTreeMap<U256, (U256, U256)>,
}
#[derive(Debug)]
pub(crate) struct Active {
    pub depth: usize,
    request: Request,
    state: EvmState,
    log_index: usize,
}
impl Active {
    pub(crate) fn new<DB: Database>(
        depth: usize,
        call: &CallInputs,
        context: &OpContext<DB>,
    ) -> Self {
        Self {
            depth,
            request: Request {
                caller: call.caller,
                endpoint: call.target_address,
                input: call.input.bytes(context),
                gas_limit: call.gas_limit,
            },
            state: context.journal().evm_state().clone(),
            log_index: context.journal().logs().len(),
        }
    }
    pub(crate) fn finish<DB: Database>(
        self,
        result: &FrameResult,
        context: &mut OpContext<DB>,
    ) -> Result<Observation, Error<DB>> {
        let mut effects = BTreeMap::new();
        let after = context.journal().evm_state().clone();
        for (address, account) in &after {
            let before = self.state.get(address);
            let info = if let Some(before) = before {
                before.info.clone()
            } else {
                context
                    .journaled_state
                    .database
                    .basic(*address)
                    .map_err(|e| {
                        Error::Evm(revm::context_interface::result::EVMError::Database(e))
                    })?
                    .unwrap_or_default()
            };
            let mut storage = BTreeMap::new();
            for (key, slot) in &account.storage {
                let old = before
                    .and_then(|a| a.storage.get(key))
                    .map_or_else(|| slot.original_value(), |s| s.present_value());
                if old != slot.present_value() {
                    storage.insert(*key, (old, slot.present_value()));
                }
            }
            let effect = Effect {
                balance_delta: account.info.balance.wrapping_sub(info.balance),
                nonce_delta: account.info.nonce.wrapping_sub(info.nonce),
                code: (account.info.code_hash != info.code_hash).then_some(account.info.code_hash),
                created: account.is_created() && !before.is_some_and(|a| a.is_created()),
                destroyed: account.is_selfdestructed() &&
                    !before.is_some_and(|a| a.is_selfdestructed()),
                touched_empty: account.is_touched() &&
                    account.info.is_empty() &&
                    !before.is_some_and(|a| a.is_touched() && a.info.is_empty()),
                storage,
            };
            if effect.balance_delta != U256::ZERO ||
                effect.nonce_delta != 0 ||
                effect.code.is_some() ||
                effect.created ||
                effect.destroyed ||
                effect.touched_empty ||
                !effect.storage.is_empty()
            {
                effects.insert(*address, effect);
            }
        }
        Ok(Observation {
            request: self.request,
            result: result.interpreter_result().clone(),
            logs: context.journal().logs()[self.log_index..].to_vec(),
            effects,
        })
    }
}

/// Observed router entry and exit, independently of an ERC-4337 wrapper's success.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Route {
    /// Actual router invocation, including materialized calldata.
    pub request: Request,
    /// Router status, output and gas. Wrapper gas is not an application invariant.
    pub result: InterpreterResult,
}
