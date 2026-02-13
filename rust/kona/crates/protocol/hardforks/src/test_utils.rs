//! Test utilities for the `kona-hardforks` crate.

use alloy_eips::Encodable2718;
use alloy_primitives::{Address, B256, keccak256};
use op_alloy_consensus::{OpTxType, TxDeposit};
use op_revm::{DefaultOp, OpSpecId, transaction::deposit::DepositTransactionParts};
use revm::{
    Context, ExecuteCommitEvm, MainBuilder,
    context::{
        CfgEnv,
        result::{ExecutionResult, Output},
    },
    database::{CacheDB, EmptyDB},
    interpreter::Host,
};

/// Runs an upgrade deposit transaction that deploys a contract in an in-memory EVM, and checks that
/// the contract deploys to the expected address and with the expected codehash.
pub(crate) fn check_deployment_code(
    deployment_tx: TxDeposit,
    expected_address: Address,
    expected_code_hash: B256,
) {
    let ctx = Context::op()
        .with_cfg(CfgEnv::new_with_spec(OpSpecId::INTEROP))
        .modify_tx_chained(|tx| {
            // Deposit + OP meta
            tx.deposit = DepositTransactionParts {
                source_hash: deployment_tx.source_hash,
                mint: Some(deployment_tx.mint),
                is_system_transaction: deployment_tx.is_system_transaction,
            };
            tx.enveloped_tx = Some(deployment_tx.encoded_2718().into());

            // Base meta
            tx.base.tx_type = OpTxType::Deposit as u8;
            tx.base.caller = deployment_tx.from;
            tx.base.kind = deployment_tx.to;
            tx.base.value = deployment_tx.value;
            tx.base.gas_limit = deployment_tx.gas_limit;
            tx.base.data = deployment_tx.input;
        })
        .with_db(CacheDB::<EmptyDB>::default());
    let mut evm = ctx.build_mainnet();

    let res = evm.replay_commit().expect("Failed to run deployment transaction");

    let address = match res {
        ExecutionResult::Success { output: Output::Create(_, Some(address)), .. } => {
            assert_eq!(address, expected_address, "Contract deployed to an unexpected address");
            address
        }
        ExecutionResult::Success { output: Output::Create(_, None), .. } => {
            panic!("Contract deployed to the zero address");
        }
        res => panic!("Failed to deploy contract: {res:?}"),
    };

    let code = evm.load_account_code(address).expect("Account does not exist");
    assert_eq!(keccak256(code.as_ref()), expected_code_hash);
}
