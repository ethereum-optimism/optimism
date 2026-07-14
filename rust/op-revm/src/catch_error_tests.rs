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
use revm::{
    context::{Context, TxEnv},
    context_interface::result::{EVMError, ExecutionResult, InvalidHeader, InvalidTransaction},
    handler::{EthFrame, Handler},
    interpreter::interpreter::EthInterpreter,
    primitives::B256,
};
use rstest::rstest;

/// A non-deposit OP custom transaction type (the SDM post-exec `0x7D` tx). Exercises the
/// `TransactionType::Custom` arm without hitting the deposit guard.
const POST_EXEC_TX_TYPE: u8 = 0x7D;

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
