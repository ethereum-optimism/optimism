//! Combination tests for [`OpHandler::catch_error`](crate::handler::OpHandler) across every
//! transaction type and both error kinds.
//!
//! `catch_error` returns the dedicated `FailedDeposit` halt only when a *deposit* transaction fails
//! with a transaction-validity error; every other `(tx type, error kind)` combination surfaces the
//! original error. These tests pin that matrix.

use crate::{
    DefaultOp, OpBuilder, OpHaltReason, OpTransaction,
    handler::OpHandler,
    transaction::{OpTransactionError, deposit::DEPOSIT_TRANSACTION_TYPE},
};
use op_alloy_consensus::OpTxType;
use revm::{
    context::{Context, TxEnv},
    context_interface::{
        ContextTr, JournalTr,
        result::{EVMError, ExecutionResult, InvalidHeader, InvalidTransaction},
    },
    database::InMemoryDB,
    handler::{EthFrame, EvmTr, Handler},
    interpreter::interpreter::EthInterpreter,
    primitives::{Address, B256, U256},
    state::AccountInfo,
};
use rstest::rstest;

/// A non-deposit OP custom transaction type (the SDM post-exec tx). Exercises the
/// `OpTxType::PostExec` arm (a non-deposit type) rather than the deposit arm.
const POST_EXEC_TX_TYPE: u8 = OpTxType::PostExec as u8;

type TestError = EVMError<core::convert::Infallible, OpTransactionError>;

/// Builds an OP transaction whose `tx_type()` is `tx_type`. Deposits are identified by a non-zero
/// source hash (not the base type byte), matching how they are constructed in production.
fn op_tx(tx_type: u8) -> OpTransaction<TxEnv> {
    if tx_type == DEPOSIT_TRANSACTION_TYPE {
        OpTransaction::builder()
            .base(TxEnv::builder().gas_limit(21_000))
            .source_hash(B256::with_last_byte(1))
            .build_fill()
    } else {
        OpTransaction::builder()
            .base(TxEnv::builder().gas_limit(21_000).tx_type(Some(tx_type)))
            .build_fill()
    }
}

/// A transaction-validity error (`is_tx_error()` true) or a non-transaction error otherwise.
fn error(is_tx_error: bool) -> TestError {
    if is_tx_error {
        EVMError::Transaction(OpTransactionError::Base(
            InvalidTransaction::PriorityFeeGreaterThanMaxFee,
        ))
    } else {
        EVMError::Header(InvalidHeader::PrevrandaoNotSet)
    }
}

fn run_catch_error(
    tx_type: u8,
    is_tx_error: bool,
) -> Result<ExecutionResult<OpHaltReason>, TestError> {
    let mut evm = Context::op().with_tx(op_tx(tx_type)).build_op();
    let handler = OpHandler::<_, TestError, EthFrame<EthInterpreter>>::new();
    handler.catch_error(&mut evm, error(is_tx_error))
}

#[rstest]
fn catch_error_over_tx_types_and_error_kinds(
    #[values(0, 1, 2, 3, 4, POST_EXEC_TX_TYPE, DEPOSIT_TRANSACTION_TYPE)] tx_type: u8,
    #[values(true, false)] is_tx_error: bool,
) {
    let result = run_catch_error(tx_type, is_tx_error);

    // A deposit that fails a tx-validity check is the only combination that yields the dedicated
    // FailedDeposit halt; everything else surfaces the original error.
    if is_tx_error && tx_type == DEPOSIT_TRANSACTION_TYPE {
        assert!(
            matches!(result, Ok(ExecutionResult::Halt { reason: OpHaltReason::FailedDeposit, .. })),
            "expected FailedDeposit halt for a deposit tx-error, got {result:?}"
        );
    } else {
        assert!(
            result.is_err(),
            "expected the original error to surface \
             (tx_type={tx_type:#x}, is_tx_error={is_tx_error}), got {result:?}"
        );
    }
}

/// A failed deposit rolls the world state back to just after the initial mint, but still
/// increments the sender nonce by 1 and persists the mint — see the deposits spec:
/// <https://specs.optimism.io/protocol/deposits.html#nonce-handling>.
#[test]
fn failed_deposit_bumps_sender_nonce_and_persists_mint() {
    const CALLER: Address = Address::new([0x11; 20]);
    const START_NONCE: u64 = 7;
    const START_BALANCE: u64 = 500;
    const MINT: u128 = 1_000;

    let mut db = InMemoryDB::default();
    db.insert_account_info(
        CALLER,
        AccountInfo {
            nonce: START_NONCE,
            balance: U256::from(START_BALANCE),
            ..Default::default()
        },
    );

    // A deposit (non-zero source hash) from `CALLER` carrying a mint, failed via a tx-validity
    // error so it takes `catch_error`'s failed-deposit path.
    let tx = OpTransaction::builder()
        .base(TxEnv::builder().caller(CALLER).gas_limit(21_000))
        .source_hash(B256::with_last_byte(1))
        .mint(MINT)
        .build_fill();
    let mut evm = Context::op().with_db(db).with_tx(tx).build_op();
    let handler = OpHandler::<_, TestError, EthFrame<EthInterpreter>>::new();

    let result = handler.catch_error(
        &mut evm,
        EVMError::Transaction(OpTransactionError::Base(
            InvalidTransaction::PriorityFeeGreaterThanMaxFee,
        )),
    );
    assert!(
        matches!(result, Ok(ExecutionResult::Halt { reason: OpHaltReason::FailedDeposit, .. })),
        "a failed deposit must return the FailedDeposit halt, got {result:?}"
    );

    let caller = evm
        .ctx()
        .journal_mut()
        .evm_state()
        .get(&CALLER)
        .expect("caller account present after a failed deposit")
        .clone();
    assert_eq!(
        caller.info.nonce,
        START_NONCE + 1,
        "a failed deposit must increment the sender nonce by 1"
    );
    assert_eq!(
        caller.info.balance,
        U256::from(START_BALANCE) + U256::from(MINT),
        "a failed deposit must persist the minted ETH"
    );
}
