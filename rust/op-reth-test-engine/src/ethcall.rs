//! `eth_call` / `eth_estimateGas` over historical chain state.
//!
//! The estimator reproduces op-geth's `eth/gasestimator.Estimate` control flow (initial bounds,
//! balance allowance, plain-transfer shortcut, optimistic limit, `lo*2`-clamped bisection with the
//! 1.5% error ratio) rather than reth's variant: the estimate lands in the signed transaction's
//! gas field, so any deviation from what the in-process geth backend would return changes the
//! transaction — and hence block — hashes across the two execution layers.

use alloy_evm::rpc::TryIntoTxEnv;
use alloy_op_evm::error::OpTxError;
use alloy_primitives::{B256, Bytes, TxKind, U256};
use alloy_rpc_types_eth::TransactionRequest;
use op_revm::{OpHaltReason, OpTransactionError, constants::L1_BLOCK_CONTRACT};
use reth_evm::{ConfigureEvm, Evm, EvmEnv};
use reth_optimism_evm::{OpEvmConfig, OpRethReceiptBuilder, OpTx};
use reth_revm::{
    context::result::{EVMError, ExecutionResult, InvalidTransaction, ResultAndState},
    database::StateProviderDatabase,
    db::State,
};
use reth_storage_api::StateProvider;

use crate::{EphemeralChain, Error};

/// geth's default RPC gas cap (`ethconfig.Defaults.RPCGasCap`) — the in-process geth backend the
/// action tests compare against runs with this default.
const RPC_GAS_CAP: u64 = 50_000_000;
/// geth `params.TxGas`: the intrinsic cost of a plain transfer.
const TX_GAS: u64 = 21_000;
/// geth `params.CallStipend`.
const CALL_STIPEND: u64 = 2_300;
/// geth `internal/ethapi.estimateGasErrorRatio`: allowed estimation overshoot before the binary
/// search terminates.
const ESTIMATE_GAS_ERROR_RATIO: f64 = 0.015;

/// The outcome of one gas-probed execution, collapsing geth's `(failed, result, err)` triple:
/// gas-related failures (out-of-gas halts, reverts, intrinsic gas above the limit) steer the
/// binary search instead of erroring.
enum Probe {
    /// Execution succeeded within the probed gas limit.
    Succeeded {
        /// Gas charged to the transaction (geth's `result.UsedGas`).
        gas_used: u64,
        /// Gas refunded at the end of execution (geth's `result.RefundedGas`).
        refunded: u64,
    },
    /// Execution reverted (may succeed with more gas, e.g. an inner out-of-gas).
    Reverted(Bytes),
    /// Execution failed on gas grounds: an out-of-gas (or other) halt, or intrinsic gas
    /// exceeding the probed limit.
    OutOfGas,
}

/// Interpret a raw EVM `transact` outcome as a [`Probe`]: geth's `ErrIntrinsicGas` /
/// `ErrFloorDataGas` equivalents count as gas failures; any other transaction-validation or
/// database error is terminal.
fn probe<DBError: core::fmt::Display>(
    outcome: Result<ResultAndState<OpHaltReason>, EVMError<DBError, OpTxError>>,
) -> crate::Result<Probe> {
    match outcome {
        Ok(ResultAndState { result, .. }) => Ok(match result {
            ExecutionResult::Success { gas, .. } => {
                Probe::Succeeded { gas_used: gas.tx_gas_used(), refunded: gas.final_refunded() }
            }
            ExecutionResult::Revert { output, .. } => Probe::Reverted(output),
            ExecutionResult::Halt { .. } => Probe::OutOfGas,
        }),
        Err(EVMError::Transaction(OpTxError(OpTransactionError::Base(
            InvalidTransaction::CallGasCostMoreThanGasLimit { .. } |
            InvalidTransaction::GasFloorMoreThanGasLimit { .. },
        )))) => Ok(Probe::OutOfGas),
        Err(other) => Err(Error::Execution(format!("call execution failed: {other}"))),
    }
}

/// Execute `request` read-only at the state of block `block_hash` (`eth_call`).
///
/// Mirrors geth's `doCall` environment: no base-fee or EIP-3607 enforcement, gas defaulting to the
/// RPC gas cap, and a zeroed base fee when the request carries no gas price. Returns the call
/// output; a revert surfaces as [`Error::Revert`].
pub(crate) fn call(
    chain: &EphemeralChain,
    block_hash: B256,
    mut request: TransactionRequest,
) -> crate::Result<Bytes> {
    // geth caps the gas for eth_call at the RPC gas cap, defaulting to it when unset.
    request.gas = Some(request.gas.map_or(RPC_GAS_CAP, |gas| gas.min(RPC_GAS_CAP)));

    let (mut evm_env, state) = call_env(chain, block_hash)?;
    let tx = tx_env(&state, &evm_env, request)?;
    // geth zeroes the base fee for gasprice-less calls so the EVM invariant `basefee <= gasprice`
    // holds (`internal/ethapi.DoCall`).
    if tx.0.base.gas_price == 0 {
        evm_env.block_env.basefee = 0;
    }

    let mut db = State::builder().with_database(StateProviderDatabase::new(&state)).build();
    db.load_cache_account(L1_BLOCK_CONTRACT).map_err(|e| Error::Execution(e.to_string()))?;
    let mut evm = evm_config(chain).evm_with_env(&mut db, evm_env);

    match evm.transact(tx) {
        Ok(ResultAndState { result, .. }) => match result {
            ExecutionResult::Success { output, .. } => Ok(output.into_data()),
            ExecutionResult::Revert { output, .. } => Err(Error::Revert(output)),
            ExecutionResult::Halt { reason, .. } => {
                Err(Error::Execution(format!("call halted: {reason:?}")))
            }
        },
        Err(err) => Err(Error::Execution(format!("call execution failed: {err}"))),
    }
}

/// Estimate the lowest gas limit that lets `request` succeed at the state of block `block_hash`
/// (`eth_estimateGas`), reproducing op-geth's `gasestimator.Estimate` step for step.
pub(crate) fn estimate_gas(
    chain: &EphemeralChain,
    block_hash: B256,
    request: TransactionRequest,
) -> crate::Result<u64> {
    let (evm_env, state) = call_env(chain, block_hash)?;

    // Highest gas limit to search under: the request's explicit gas (if it is at least a plain
    // transfer's worth), else the block gas limit.
    let mut hi = match request.gas {
        Some(gas) if gas >= TX_GAS => gas,
        _ => evm_env.block_env.gas_limit,
    };

    // Recap by the sender's balance when a fee price is given: (balance - value) / feeCap.
    let fee_cap = request.max_fee_per_gas.or(request.gas_price).unwrap_or_default();
    if fee_cap != 0 {
        let from = request.from.unwrap_or_default();
        let balance = state.account_balance(&from)?.unwrap_or_default();
        let value = request.value.unwrap_or_default();
        if value >= balance {
            return Err(Error::Execution("insufficient funds for transfer".to_string()));
        }
        let allowance = (balance - value) / U256::from(fee_cap);
        if let Ok(allowance) = u64::try_from(allowance) &&
            hi > allowance
        {
            hi = allowance;
        }
    }
    // Recap by the RPC gas cap.
    hi = hi.min(RPC_GAS_CAP);

    let tx = tx_env(&state, &evm_env, request)?;

    let mut db = State::builder().with_database(StateProviderDatabase::new(&state)).build();
    db.load_cache_account(L1_BLOCK_CONTRACT).map_err(|e| Error::Execution(e.to_string()))?;
    let mut evm = evm_config(chain).evm_with_env(&mut db, evm_env);

    let mut execute = |gas_limit: u64| -> crate::Result<Probe> {
        let mut probe_tx = tx.clone();
        probe_tx.0.base.gas_limit = gas_limit;
        probe(evm.transact(probe_tx))
    };

    // Plain value transfer to an account without code: short-circuit by probing exactly 21000.
    if tx.0.base.data.is_empty() &&
        let TxKind::Call(to) = tx.0.base.kind &&
        state.account_code(&to)?.is_none_or(|code| code.is_empty()) &&
        matches!(execute(TX_GAS)?, Probe::Succeeded { .. })
    {
        return Ok(TX_GAS);
    }

    // Execute once at the highest allowable limit: a failure here is terminal.
    let (gas_used, refunded) = match execute(hi)? {
        Probe::Succeeded { gas_used, refunded } => (gas_used, refunded),
        Probe::Reverted(output) => return Err(Error::Revert(output)),
        Probe::OutOfGas => {
            return Err(Error::Execution(format!("gas required exceeds allowance ({hi})")));
        }
    };
    let mut lo = gas_used - 1;

    // Optimistic first bound: used + refunded + stipend, scaled by 64/63 for call forwarding.
    let optimistic = (gas_used + refunded + CALL_STIPEND) * 64 / 63;
    if optimistic < hi {
        match execute(optimistic)? {
            Probe::Succeeded { .. } => hi = optimistic,
            _ => lo = optimistic,
        }
    }

    // Bisect, skewed low (geth caps the midpoint at `lo*2`), stopping within 1.5% of `hi`.
    while lo + 1 < hi {
        if ((hi - lo) as f64) / (hi as f64) < ESTIMATE_GAS_ERROR_RATIO {
            break;
        }
        let mut mid = lo + (hi - lo) / 2;
        if mid > lo * 2 {
            mid = lo * 2;
        }
        match execute(mid)? {
            Probe::Succeeded { .. } => hi = mid,
            _ => lo = mid,
        }
    }
    Ok(hi)
}

/// The EVM configuration used for calls (same construction as the block-building paths).
fn evm_config(chain: &EphemeralChain) -> OpEvmConfig {
    OpEvmConfig::new(chain.chain_spec(), OpRethReceiptBuilder::default())
}

/// The EVM environment and state provider for executing calls at block `block_hash`, with the
/// geth call-context relaxations applied: no EIP-3607 sender-code check, no base-fee charge or
/// enforcement, no EIP-7825 per-tx gas cap, and no block-gas-limit check (call gas may exceed it,
/// since the RPC gas cap — not the block limit — bounds it).
fn call_env(
    chain: &EphemeralChain,
    block_hash: B256,
) -> crate::Result<(EvmEnv<op_revm::OpSpecId>, Box<dyn StateProvider>)> {
    let header = chain
        .sealed_header(block_hash)?
        .ok_or_else(|| Error::Execution(format!("block {block_hash} is unknown")))?;
    let state = chain
        .state_at(block_hash)?
        .ok_or_else(|| Error::Execution(format!("no state for block {block_hash}")))?;
    let mut evm_env = evm_config(chain)
        .evm_env(header.header())
        .map_err(|e| Error::Execution(format!("build evm env: {e}")))?;
    evm_env.cfg_env.disable_eip3607 = true;
    evm_env.cfg_env.disable_base_fee = true;
    evm_env.cfg_env.disable_fee_charge = true;
    evm_env.cfg_env.disable_block_gas_limit = true;
    evm_env.cfg_env.tx_gas_limit_cap = Some(u64::MAX);
    Ok((evm_env, state))
}

/// Build the OP transaction environment for a call request, defaulting the nonce to the sender's
/// current one (call messages skip nonce checks in geth; filling the live nonce is equivalent).
fn tx_env(
    state: &dyn StateProvider,
    evm_env: &EvmEnv<op_revm::OpSpecId>,
    mut request: TransactionRequest,
) -> crate::Result<OpTx> {
    if request.nonce.is_none() {
        let from = request.from.unwrap_or_default();
        request.nonce = Some(state.account_nonce(&from)?.unwrap_or_default());
    }
    let base = request
        .try_into_tx_env(evm_env)
        .map_err(|e| Error::Execution(format!("invalid call request: {e}")))?;
    // An empty enveloped tx is what reth's RPC call path uses: it makes the L1-fee charge for this
    // simulated (non-deposit) transaction zero.
    Ok(OpTx(op_revm::OpTransaction {
        base,
        enveloped_tx: Some(Bytes::new()),
        deposit: Default::default(),
    }))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::testsupport::{test_engine_with_accounts, user_sender};
    use alloy_genesis::GenesisAccount;
    use alloy_primitives::{Address, address, bytes};

    /// Returns the 32-byte word 0x2a: `PUSH1 0x2a PUSH1 0 MSTORE PUSH1 0x20 PUSH1 0 RETURN`.
    const RETURNER: Address = address!("0x00000000000000000000000000000000000000aa");
    /// Always reverts with empty output: `PUSH1 0 PUSH1 0 REVERT`.
    const REVERTER: Address = address!("0x00000000000000000000000000000000000000bb");

    fn engine_with_contracts() -> crate::TestEngine {
        test_engine_with_accounts(
            user_sender(),
            [
                (
                    RETURNER,
                    GenesisAccount {
                        code: Some(bytes!("0x602a60005260206000f3")),
                        ..Default::default()
                    },
                ),
                (
                    REVERTER,
                    GenesisAccount { code: Some(bytes!("0x60006000fd")), ..Default::default() },
                ),
            ],
        )
    }

    fn request(to: Address) -> TransactionRequest {
        TransactionRequest {
            from: Some(user_sender()),
            to: Some(TxKind::Call(to)),
            ..Default::default()
        }
    }

    #[test]
    fn estimate_plain_transfer_is_21000() {
        let engine = engine_with_contracts();
        let genesis = engine.chain.genesis_hash();
        let mut req = request(Address::repeat_byte(0x11));
        req.value = Some(U256::from(1234u64));
        // With an explicit fee cap the balance-allowance path runs too.
        req.max_fee_per_gas = Some(1_000_000_000);
        assert_eq!(estimate_gas(&engine.chain, genesis, req).expect("estimate"), TX_GAS);
    }

    #[test]
    fn estimate_calldata_transfer_converges_above_intrinsic() {
        let engine = engine_with_contracts();
        let genesis = engine.chain.genesis_hash();
        let mut req = request(Address::repeat_byte(0x11));
        req.input = bytes!("0xdeadbeef00ff").into();
        let est = estimate_gas(&engine.chain, genesis, req).expect("estimate");
        // Above a plain transfer (calldata is not free), and terminates within geth's error ratio
        // of the true requirement (well under 2x).
        assert!(est > TX_GAS, "estimate {est} should exceed the plain-transfer cost");
        assert!(est < 2 * TX_GAS, "estimate {est} should stay near the intrinsic cost");
    }

    #[test]
    fn estimate_contract_call_exceeds_plain_transfer() {
        let engine = engine_with_contracts();
        let genesis = engine.chain.genesis_hash();
        let est = estimate_gas(&engine.chain, genesis, request(RETURNER)).expect("estimate");
        // Executes real code (memory write + return), so it must cost more than 21000 even with
        // empty calldata — the plain-transfer shortcut must not trigger for code-bearing targets.
        assert!(est > TX_GAS, "estimate {est} should exceed the plain-transfer cost");
    }

    #[test]
    fn call_returns_contract_output() {
        let engine = engine_with_contracts();
        let genesis = engine.chain.genesis_hash();
        let out = call(&engine.chain, genesis, request(RETURNER)).expect("call");
        assert_eq!(out, Bytes::from(U256::from(42u64).to_be_bytes::<32>()));
    }

    #[test]
    fn call_and_estimate_surface_reverts() {
        let engine = engine_with_contracts();
        let genesis = engine.chain.genesis_hash();
        assert!(matches!(
            call(&engine.chain, genesis, request(REVERTER)),
            Err(Error::Revert(output)) if output.is_empty()
        ));
        assert!(matches!(
            estimate_gas(&engine.chain, genesis, request(REVERTER)),
            Err(Error::Revert(_))
        ));
    }
}
