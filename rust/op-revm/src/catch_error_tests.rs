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

/// The EIP-4844 (blob) type ID. OP has no `OpTxType::Eip4844` variant, so this value exercises the
/// `OpTxType::try_from` *failure* fold in `catch_error_tx_error` (the unsupported-type path) rather
/// than a match arm.
const EIP4844_TX_TYPE: u8 = 3;

/// An account unrelated to the failing transaction, journal-seeded with [`SEEDED_BALANCE`] before
/// `catch_error` runs. Its balance afterwards is the discard signal: zero when the failed tx's
/// journal was discarded, still [`SEEDED_BALANCE`] when it leaked.
const SEEDED_ADDR: Address = Address::new([0x5E; 20]);
const SEEDED_BALANCE: u64 = 7;

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

/// Runs `catch_error` for the given `(tx type, error kind)` cell and returns its result together
/// with [`SEEDED_ADDR`]'s post-`catch_error` balance, seeded via a journaled balance change
/// beforehand to stand in for the failed tx's partial state.
fn run_catch_error(
    tx_type: u8,
    is_tx_error: bool,
) -> (Result<ExecutionResult<OpHaltReason>, TestError>, U256) {
    let mut evm = Context::op().with_tx(op_tx(tx_type)).build_op();
    evm.ctx()
        .journal_mut()
        .balance_incr(SEEDED_ADDR, U256::from(SEEDED_BALANCE))
        .expect("seeding the journal cannot fail on the default in-memory DB");
    let handler = OpHandler::<_, TestError, EthFrame<EthInterpreter>>::new();
    let result = handler.catch_error(&mut evm, error(is_tx_error));
    let seeded_balance_after = evm
        .ctx()
        .journal_mut()
        .evm_state()
        .get(&SEEDED_ADDR)
        .map(|account| account.info.balance)
        .unwrap_or_default();
    (result, seeded_balance_after)
}

#[rstest]
fn catch_error_over_tx_types_and_error_kinds(
    #[values(0, 1, 2, EIP4844_TX_TYPE, 4, POST_EXEC_TX_TYPE, DEPOSIT_TRANSACTION_TYPE)] tx_type: u8,
    #[values(true, false)] is_tx_error: bool,
) {
    let (result, seeded_balance_after) = run_catch_error(tx_type, is_tx_error);

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

    // Every cell must also discard the failed tx's journal — the deposit arm via
    // `checkpoint_revert`, every other arm via `discard_tx`. A surviving seeded balance means the
    // failed tx's partial state (and its EIP-2929 warm stamps) would leak into the next tx.
    assert_eq!(
        seeded_balance_after,
        U256::ZERO,
        "catch_error must discard the failed tx's journal changes \
         (tx_type={tx_type:#x}, is_tx_error={is_tx_error})"
    );
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
