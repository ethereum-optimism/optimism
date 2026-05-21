//! Integration tests for the `op-revm` crate.

use crate::TestdataConfig;
use op_revm::{
    DefaultOp, L1BlockInfo, OpBuilder, OpHaltReason, OpSpecId, OpTransaction,
    precompiles::bn254_pair::GRANITE_MAX_INPUT_SIZE,
};
use revm::{
    Context, ExecuteEvm, InspectEvm, Inspector, Journal, SystemCallEvm,
    bytecode::opcode,
    context::{
        BlockEnv, CfgEnv, TxEnv,
        result::{ExecutionResult, OutOfGasError, ResultGas},
    },
    context_interface::result::HaltReason,
    database::{BENCH_CALLER, BENCH_CALLER_BALANCE, BENCH_TARGET, BenchmarkDB, EmptyDB, State},
    handler::system_call::SYSTEM_ADDRESS,
    interpreter::{
        InterpreterTypes,
        gas::{InitialAndFloorGas, calculate_initial_tx_gas},
    },
    precompile::{bls12_381_const, bls12_381_utils, bn254, secp256r1, u64_to_address},
    primitives::{Address, Bytes, Log, TxKind, U256, address, bytes, eip7825},
    state::Bytecode,
};
use std::{path::PathBuf, vec::Vec};

// Re-export the constant for testdata directory path
const TESTS_TESTDATA: &str = "tests/op_revm_testdata";

fn op_revm_testdata_config() -> TestdataConfig {
    TestdataConfig { testdata_dir: PathBuf::from(TESTS_TESTDATA) }
}

fn compare_or_save_op_testdata<T>(filename: &str, output: &T)
where
    T: serde::Serialize + for<'a> serde::Deserialize<'a> + PartialEq + std::fmt::Debug,
{
    crate::compare_or_save_testdata_with_config(filename, output, op_revm_testdata_config());
}

#[test]
fn test_deposit_tx() {
    let ctx = Context::op()
        .with_tx(
            OpTransaction::builder()
                .enveloped_tx(None)
                .mint(100)
                .source_hash(revm::primitives::B256::from([1u8; 32]))
                .build_fill(),
        )
        .with_cfg(CfgEnv::new_with_spec(OpSpecId::HOLOCENE));

    let mut evm = ctx.build_op();

    let output = evm.replay().unwrap();

    // balance should be 100
    assert_eq!(
        output.state.get(&Address::default()).map(|a| a.info.balance),
        Some(U256::from(100))
    );
    compare_or_save_op_testdata("test_deposit_tx.json", &output);
}

#[test]
fn test_halted_deposit_tx() {
    let ctx = Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(TxEnv::builder().caller(BENCH_CALLER).kind(TxKind::Call(BENCH_TARGET)))
                .enveloped_tx(None)
                .mint(100)
                .source_hash(revm::primitives::B256::from([1u8; 32]))
                .build_fill(),
        )
        .with_cfg(CfgEnv::new_with_spec(OpSpecId::HOLOCENE))
        .with_db(BenchmarkDB::new_bytecode(Bytecode::new_legacy([opcode::POP].into())));

    // POP would return a halt.
    let mut evm = ctx.build_op();

    let output = evm.replay().unwrap();

    // balance should be 100 + previous balance
    assert_eq!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::FailedDeposit,
            gas: ResultGas::default().with_total_gas_spent(eip7825::TX_GAS_LIMIT_CAP),
            logs: vec![],
        }
    );
    assert_eq!(
        output.state.get(&BENCH_CALLER).map(|a| a.info.balance),
        Some(U256::from(100) + BENCH_CALLER_BALANCE)
    );

    compare_or_save_op_testdata("test_halted_deposit_tx.json", &output);
}

fn p256verify_test_tx()
-> Context<BlockEnv, OpTransaction<TxEnv>, CfgEnv<OpSpecId>, EmptyDB, Journal<EmptyDB>, L1BlockInfo>
{
    const SPEC_ID: OpSpecId = OpSpecId::FJORD;

    let InitialAndFloorGas { initial_total_gas, .. } =
        calculate_initial_tx_gas(SPEC_ID.into(), &[], false, 0, 0, 0);

    Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(u64_to_address(secp256r1::P256VERIFY_ADDRESS)))
                        .gas_limit(initial_total_gas + secp256r1::P256VERIFY_BASE_GAS_FEE),
                )
                .build_fill(),
        )
        .with_cfg(CfgEnv::new_with_spec(SPEC_ID))
}

#[test]
fn test_tx_call_p256verify() {
    let ctx = p256verify_test_tx();

    let mut evm = ctx.build_op();
    let output = evm.replay().unwrap();

    // assert successful call to P256VERIFY
    assert!(output.result.is_success());

    compare_or_save_op_testdata("test_tx_call_p256verify.json", &output);
}

#[test]
fn test_halted_tx_call_p256verify() {
    const SPEC_ID: OpSpecId = OpSpecId::FJORD;
    let InitialAndFloorGas { initial_total_gas, .. } =
        calculate_initial_tx_gas(SPEC_ID.into(), &[], false, 0, 0, 0);
    let original_gas_limit = initial_total_gas + secp256r1::P256VERIFY_BASE_GAS_FEE;

    let ctx = Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(u64_to_address(secp256r1::P256VERIFY_ADDRESS)))
                        .gas_limit(original_gas_limit - 1),
                )
                .build_fill(),
        )
        .with_cfg(CfgEnv::new_with_spec(SPEC_ID));

    let mut evm = ctx.build_op();
    let output = evm.replay().unwrap();

    // assert out of gas for P256VERIFY
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::OutOfGas(OutOfGasError::Precompile)),
            ..
        }
    ));

    compare_or_save_op_testdata("test_halted_tx_call_p256verify.json", &output);
}

fn bn254_pair_test_tx(
    spec: OpSpecId,
) -> Context<BlockEnv, OpTransaction<TxEnv>, CfgEnv<OpSpecId>, EmptyDB, Journal<EmptyDB>, L1BlockInfo>
{
    let input = Bytes::from([1; GRANITE_MAX_INPUT_SIZE + 2]);
    let InitialAndFloorGas { initial_total_gas, .. } =
        calculate_initial_tx_gas(spec.into(), &input[..], false, 0, 0, 0);

    Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(bn254::pair::ADDRESS))
                        .data(input)
                        .gas_limit(initial_total_gas),
                )
                .build_fill(),
        )
        .with_cfg(CfgEnv::new_with_spec(spec))
}

#[test]
fn test_halted_tx_call_bn254_pair_fjord() {
    let ctx = bn254_pair_test_tx(OpSpecId::FJORD);

    let mut evm = ctx.build_op();
    let output = evm.replay().unwrap();

    // assert out of gas
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::OutOfGas(OutOfGasError::Precompile)),
            ..
        }
    ));

    compare_or_save_op_testdata("test_halted_tx_call_bn254_pair_fjord.json", &output);
}

#[test]
fn test_halted_tx_call_bn254_pair_granite() {
    let ctx = bn254_pair_test_tx(OpSpecId::GRANITE);

    let mut evm = ctx.build_op();
    let output = evm.replay().unwrap();

    // assert bails early because input size too big
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::PrecompileErrorWithContext(ref msg)),
            ..
        } if msg == "bn254 invalid pair length"
    ));

    compare_or_save_op_testdata("test_halted_tx_call_bn254_pair_granite.json", &output);
}

#[test]
fn test_halted_tx_call_bls12_381_g1_add_out_of_gas() {
    let ctx = Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(bls12_381_const::G1_ADD_ADDRESS))
                        .gas_limit(21_000 + bls12_381_const::G1_ADD_BASE_GAS_FEE - 1),
                )
                .build_fill(),
        )
        .modify_chain_chained(|l1_block| {
            l1_block.operator_fee_constant = Some(U256::ZERO);
            l1_block.operator_fee_scalar = Some(U256::ZERO)
        })
        .with_cfg(CfgEnv::new_with_spec(OpSpecId::ISTHMUS));

    let mut evm = ctx.build_op();

    let output = evm.replay().unwrap();

    // assert out of gas
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::OutOfGas(OutOfGasError::Precompile)),
            ..
        }
    ));

    compare_or_save_op_testdata("test_halted_tx_call_bls12_381_g1_add_out_of_gas.json", &output);
}

#[test]
fn test_halted_tx_call_bls12_381_g1_add_input_wrong_size() {
    let ctx = Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(bls12_381_const::G1_ADD_ADDRESS))
                        .gas_limit(21_000 + bls12_381_const::G1_ADD_BASE_GAS_FEE),
                )
                .build_fill(),
        )
        .modify_chain_chained(|l1_block| {
            l1_block.operator_fee_constant = Some(U256::ZERO);
            l1_block.operator_fee_scalar = Some(U256::ZERO)
        })
        .with_cfg(CfgEnv::new_with_spec(OpSpecId::ISTHMUS));

    let mut evm = ctx.build_op();
    let output = evm.replay().unwrap();

    // assert fails post gas check, because input is wrong size
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::PrecompileErrorWithContext(ref msg)),
            ..
        } if msg == "bls12-381 g1 add input length error"
    ));

    compare_or_save_op_testdata(
        "test_halted_tx_call_bls12_381_g1_add_input_wrong_size.json",
        &output,
    );
}

fn g1_msm_test_tx()
-> Context<BlockEnv, OpTransaction<TxEnv>, CfgEnv<OpSpecId>, EmptyDB, Journal<EmptyDB>, L1BlockInfo>
{
    const SPEC_ID: OpSpecId = OpSpecId::ISTHMUS;

    let input = Bytes::from([1; bls12_381_const::G1_MSM_INPUT_LENGTH]);
    let InitialAndFloorGas { initial_total_gas, .. } =
        calculate_initial_tx_gas(SPEC_ID.into(), &input[..], false, 0, 0, 0);
    let gs1_msm_gas = bls12_381_utils::msm_required_gas(
        1,
        &bls12_381_const::DISCOUNT_TABLE_G1_MSM,
        bls12_381_const::G1_MSM_BASE_GAS_FEE,
    );

    Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(bls12_381_const::G1_MSM_ADDRESS))
                        .data(input)
                        .gas_limit(initial_total_gas + gs1_msm_gas),
                )
                .build_fill(),
        )
        .modify_chain_chained(|l1_block| {
            l1_block.operator_fee_constant = Some(U256::ZERO);
            l1_block.operator_fee_scalar = Some(U256::ZERO)
        })
        .with_cfg(CfgEnv::new_with_spec(SPEC_ID))
}

#[test]
fn test_halted_tx_call_bls12_381_g1_msm_input_wrong_size() {
    const SPEC_ID: OpSpecId = OpSpecId::ISTHMUS;
    let input = Bytes::from([1; bls12_381_const::G1_MSM_INPUT_LENGTH]);
    let InitialAndFloorGas { initial_total_gas, .. } =
        calculate_initial_tx_gas(SPEC_ID.into(), &input[..], false, 0, 0, 0);
    let gs1_msm_gas = bls12_381_utils::msm_required_gas(
        1,
        &bls12_381_const::DISCOUNT_TABLE_G1_MSM,
        bls12_381_const::G1_MSM_BASE_GAS_FEE,
    );

    let ctx = Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(bls12_381_const::G1_MSM_ADDRESS))
                        .data(input.slice(1..))
                        .gas_limit(initial_total_gas + gs1_msm_gas),
                )
                .build_fill(),
        )
        .modify_chain_chained(|l1_block| {
            l1_block.operator_fee_constant = Some(U256::ZERO);
            l1_block.operator_fee_scalar = Some(U256::ZERO)
        })
        .with_cfg(CfgEnv::new_with_spec(SPEC_ID));

    let mut evm = ctx.build_op();
    let output = evm.replay().unwrap();

    // assert fails pre gas check, because input is wrong size
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::PrecompileErrorWithContext(ref msg)),
            ..
        } if msg == "bls12-381 g1 msm input length error"
    ));

    compare_or_save_op_testdata(
        "test_halted_tx_call_bls12_381_g1_msm_input_wrong_size.json",
        &output,
    );
}

#[test]
fn test_halted_tx_call_bls12_381_g1_msm_out_of_gas() {
    const SPEC_ID: OpSpecId = OpSpecId::ISTHMUS;
    let input = Bytes::from([1; bls12_381_const::G1_MSM_INPUT_LENGTH]);
    let InitialAndFloorGas { initial_total_gas, .. } =
        calculate_initial_tx_gas(SPEC_ID.into(), &input[..], false, 0, 0, 0);
    let gs1_msm_gas = bls12_381_utils::msm_required_gas(
        1,
        &bls12_381_const::DISCOUNT_TABLE_G1_MSM,
        bls12_381_const::G1_MSM_BASE_GAS_FEE,
    );

    let ctx = Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(bls12_381_const::G1_MSM_ADDRESS))
                        .data(input)
                        .gas_limit(initial_total_gas + gs1_msm_gas - 1),
                )
                .build_fill(),
        )
        .modify_chain_chained(|l1_block| {
            l1_block.operator_fee_constant = Some(U256::ZERO);
            l1_block.operator_fee_scalar = Some(U256::ZERO)
        })
        .with_cfg(CfgEnv::new_with_spec(SPEC_ID));

    let mut evm = ctx.build_op();
    let output = evm.replay().unwrap();

    // assert out of gas
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::OutOfGas(OutOfGasError::Precompile)),
            ..
        }
    ));

    compare_or_save_op_testdata("test_halted_tx_call_bls12_381_g1_msm_out_of_gas.json", &output);
}

#[test]
fn test_halted_tx_call_bls12_381_g1_msm_wrong_input_layout() {
    let ctx = g1_msm_test_tx();

    let mut evm = ctx.build_op();
    let output = evm.replay().unwrap();

    // assert fails post gas check, because input is wrong layout
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::PrecompileErrorWithContext(ref msg)),
            ..
        } if msg == "bls12-381 fp 64 top bytes of input are not zero"
    ));

    compare_or_save_op_testdata(
        "test_halted_tx_call_bls12_381_g1_msm_wrong_input_layout.json",
        &output,
    );
}

#[test]
fn test_halted_tx_call_bls12_381_g2_add_out_of_gas() {
    let ctx = Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(bls12_381_const::G2_ADD_ADDRESS))
                        .gas_limit(21_000 + bls12_381_const::G2_ADD_BASE_GAS_FEE - 1),
                )
                .build_fill(),
        )
        .modify_chain_chained(|l1_block| {
            l1_block.operator_fee_constant = Some(U256::ZERO);
            l1_block.operator_fee_scalar = Some(U256::ZERO)
        })
        .with_cfg(CfgEnv::new_with_spec(OpSpecId::ISTHMUS));

    let mut evm = ctx.build_op();

    let output = evm.replay().unwrap();

    // assert out of gas
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::OutOfGas(OutOfGasError::Precompile)),
            ..
        }
    ));

    compare_or_save_op_testdata("test_halted_tx_call_bls12_381_g2_add_out_of_gas.json", &output);
}

#[test]
fn test_halted_tx_call_bls12_381_g2_add_input_wrong_size() {
    let ctx = Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(bls12_381_const::G2_ADD_ADDRESS))
                        .gas_limit(21_000 + bls12_381_const::G2_ADD_BASE_GAS_FEE),
                )
                .build_fill(),
        )
        .modify_chain_chained(|l1_block| {
            l1_block.operator_fee_constant = Some(U256::ZERO);
            l1_block.operator_fee_scalar = Some(U256::ZERO)
        })
        .with_cfg(CfgEnv::new_with_spec(OpSpecId::ISTHMUS));

    let mut evm = ctx.build_op();

    let output = evm.replay().unwrap();

    // assert fails post gas check, because input is wrong size
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::PrecompileErrorWithContext(ref msg)),
            ..
        } if msg == "bls12-381 g2 add input length error"
    ));

    compare_or_save_op_testdata(
        "test_halted_tx_call_bls12_381_g2_add_input_wrong_size.json",
        &output,
    );
}

fn g2_msm_test_tx()
-> Context<BlockEnv, OpTransaction<TxEnv>, CfgEnv<OpSpecId>, EmptyDB, Journal<EmptyDB>, L1BlockInfo>
{
    const SPEC_ID: OpSpecId = OpSpecId::ISTHMUS;

    let input = Bytes::from([1; bls12_381_const::G2_MSM_INPUT_LENGTH]);
    let InitialAndFloorGas { initial_total_gas, .. } =
        calculate_initial_tx_gas(SPEC_ID.into(), &input[..], false, 0, 0, 0);
    let gs2_msm_gas = bls12_381_utils::msm_required_gas(
        1,
        &bls12_381_const::DISCOUNT_TABLE_G2_MSM,
        bls12_381_const::G2_MSM_BASE_GAS_FEE,
    );

    Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(bls12_381_const::G2_MSM_ADDRESS))
                        .data(input)
                        .gas_limit(initial_total_gas + gs2_msm_gas),
                )
                .build_fill(),
        )
        .modify_chain_chained(|l1_block| {
            l1_block.operator_fee_constant = Some(U256::ZERO);
            l1_block.operator_fee_scalar = Some(U256::ZERO)
        })
        .with_cfg(CfgEnv::new_with_spec(SPEC_ID))
}

#[test]
fn test_halted_tx_call_bls12_381_g2_msm_input_wrong_size() {
    const SPEC_ID: OpSpecId = OpSpecId::ISTHMUS;
    let input = Bytes::from([1; bls12_381_const::G2_MSM_INPUT_LENGTH]);
    let InitialAndFloorGas { initial_total_gas, .. } =
        calculate_initial_tx_gas(SPEC_ID.into(), &input[..], false, 0, 0, 0);
    let gs2_msm_gas = bls12_381_utils::msm_required_gas(
        1,
        &bls12_381_const::DISCOUNT_TABLE_G2_MSM,
        bls12_381_const::G2_MSM_BASE_GAS_FEE,
    );

    let ctx = Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(bls12_381_const::G2_MSM_ADDRESS))
                        .data(input.slice(1..))
                        .gas_limit(initial_total_gas + gs2_msm_gas),
                )
                .build_fill(),
        )
        .modify_chain_chained(|l1_block| {
            l1_block.operator_fee_constant = Some(U256::ZERO);
            l1_block.operator_fee_scalar = Some(U256::ZERO)
        })
        .with_cfg(CfgEnv::new_with_spec(SPEC_ID));

    let mut evm = ctx.build_op();
    let output = evm.replay().unwrap();

    // assert fails pre gas check, because input is wrong size
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::PrecompileErrorWithContext(ref msg)),
            ..
        } if msg == "bls12-381 g2 msm input length error"
    ));

    compare_or_save_op_testdata(
        "test_halted_tx_call_bls12_381_g2_msm_input_wrong_size.json",
        &output,
    );
}

#[test]
fn test_halted_tx_call_bls12_381_g2_msm_out_of_gas() {
    const SPEC_ID: OpSpecId = OpSpecId::ISTHMUS;
    let input = Bytes::from([1; bls12_381_const::G2_MSM_INPUT_LENGTH]);
    let InitialAndFloorGas { initial_total_gas, .. } =
        calculate_initial_tx_gas(SPEC_ID.into(), &input[..], false, 0, 0, 0);
    let gs2_msm_gas = bls12_381_utils::msm_required_gas(
        1,
        &bls12_381_const::DISCOUNT_TABLE_G2_MSM,
        bls12_381_const::G2_MSM_BASE_GAS_FEE,
    );

    let ctx = Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(bls12_381_const::G2_MSM_ADDRESS))
                        .data(input)
                        .gas_limit(initial_total_gas + gs2_msm_gas - 1),
                )
                .build_fill(),
        )
        .modify_chain_chained(|l1_block| {
            l1_block.operator_fee_constant = Some(U256::ZERO);
            l1_block.operator_fee_scalar = Some(U256::ZERO)
        })
        .with_cfg(CfgEnv::new_with_spec(SPEC_ID));

    let mut evm = ctx.build_op();
    let output = evm.replay().unwrap();

    // assert out of gas
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::OutOfGas(OutOfGasError::Precompile)),
            ..
        }
    ));

    compare_or_save_op_testdata("test_halted_tx_call_bls12_381_g2_msm_out_of_gas.json", &output);
}

#[test]
fn test_halted_tx_call_bls12_381_g2_msm_wrong_input_layout() {
    let ctx = g2_msm_test_tx();

    let mut evm = ctx.build_op();
    let output = evm.replay().unwrap();

    // assert fails post gas check, because input is wrong layout
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::PrecompileErrorWithContext(ref msg)),
            ..
        } if msg == "bls12-381 fp 64 top bytes of input are not zero"
    ));

    compare_or_save_op_testdata(
        "test_halted_tx_call_bls12_381_g2_msm_wrong_input_layout.json",
        &output,
    );
}

fn bl12_381_pairing_test_tx()
-> Context<BlockEnv, OpTransaction<TxEnv>, CfgEnv<OpSpecId>, EmptyDB, Journal<EmptyDB>, L1BlockInfo>
{
    const SPEC_ID: OpSpecId = OpSpecId::ISTHMUS;

    let input = Bytes::from([1; bls12_381_const::PAIRING_INPUT_LENGTH]);
    let InitialAndFloorGas { initial_total_gas, .. } =
        calculate_initial_tx_gas(SPEC_ID.into(), &input[..], false, 0, 0, 0);

    let pairing_gas: u64 =
        bls12_381_const::PAIRING_MULTIPLIER_BASE + bls12_381_const::PAIRING_OFFSET_BASE;

    Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(bls12_381_const::PAIRING_ADDRESS))
                        .data(input)
                        .gas_limit(initial_total_gas + pairing_gas),
                )
                .build_fill(),
        )
        .modify_chain_chained(|l1_block| {
            l1_block.operator_fee_constant = Some(U256::ZERO);
            l1_block.operator_fee_scalar = Some(U256::ZERO)
        })
        .with_cfg(CfgEnv::new_with_spec(OpSpecId::ISTHMUS))
}

#[test]
fn test_halted_tx_call_bls12_381_pairing_input_wrong_size() {
    const SPEC_ID: OpSpecId = OpSpecId::ISTHMUS;
    let input = Bytes::from([1; bls12_381_const::PAIRING_INPUT_LENGTH]);
    let InitialAndFloorGas { initial_total_gas, .. } =
        calculate_initial_tx_gas(SPEC_ID.into(), &input[..], false, 0, 0, 0);
    let pairing_gas: u64 =
        bls12_381_const::PAIRING_MULTIPLIER_BASE + bls12_381_const::PAIRING_OFFSET_BASE;

    let ctx = Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(bls12_381_const::PAIRING_ADDRESS))
                        .data(input.slice(1..))
                        .gas_limit(initial_total_gas + pairing_gas),
                )
                .build_fill(),
        )
        .modify_chain_chained(|l1_block| {
            l1_block.operator_fee_constant = Some(U256::ZERO);
            l1_block.operator_fee_scalar = Some(U256::ZERO)
        })
        .with_cfg(CfgEnv::new_with_spec(OpSpecId::ISTHMUS));

    let mut evm = ctx.build_op();
    let output = evm.replay().unwrap();

    // assert fails pre gas check, because input is wrong size
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::PrecompileErrorWithContext(ref msg)),
            ..
        } if msg == "bls12-381 pairing input length error"
    ));

    compare_or_save_op_testdata(
        "test_halted_tx_call_bls12_381_pairing_input_wrong_size.json",
        &output,
    );
}

#[test]
fn test_halted_tx_call_bls12_381_pairing_out_of_gas() {
    const SPEC_ID: OpSpecId = OpSpecId::ISTHMUS;
    let input = Bytes::from([1; bls12_381_const::PAIRING_INPUT_LENGTH]);
    let InitialAndFloorGas { initial_total_gas, .. } =
        calculate_initial_tx_gas(SPEC_ID.into(), &input[..], false, 0, 0, 0);
    let pairing_gas: u64 =
        bls12_381_const::PAIRING_MULTIPLIER_BASE + bls12_381_const::PAIRING_OFFSET_BASE;

    let ctx = Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(bls12_381_const::PAIRING_ADDRESS))
                        .data(input)
                        .gas_limit(initial_total_gas + pairing_gas - 1),
                )
                .build_fill(),
        )
        .modify_chain_chained(|l1_block| {
            l1_block.operator_fee_constant = Some(U256::ZERO);
            l1_block.operator_fee_scalar = Some(U256::ZERO)
        })
        .with_cfg(CfgEnv::new_with_spec(OpSpecId::ISTHMUS));

    let mut evm = ctx.build_op();
    let output = evm.replay().unwrap();

    // assert out of gas
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::OutOfGas(OutOfGasError::Precompile)),
            ..
        }
    ));

    compare_or_save_op_testdata("test_halted_tx_call_bls12_381_pairing_out_of_gas.json", &output);
}

#[test]
fn test_tx_call_bls12_381_pairing_wrong_input_layout() {
    let ctx = bl12_381_pairing_test_tx();

    let mut evm = ctx.build_op();
    let output = evm.replay().unwrap();

    // assert fails post gas check, because input is wrong layout
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::PrecompileErrorWithContext(ref msg)),
            ..
        } if msg == "bls12-381 fp 64 top bytes of input are not zero"
    ));

    compare_or_save_op_testdata(
        "test_halted_tx_call_bls12_381_pairing_wrong_input_layout.json",
        &output,
    );
}

#[test]
fn test_halted_tx_call_bls12_381_map_fp_to_g1_out_of_gas() {
    const SPEC_ID: OpSpecId = OpSpecId::ISTHMUS;
    let input = Bytes::from([1; bls12_381_const::PADDED_FP_LENGTH]);
    let InitialAndFloorGas { initial_total_gas, .. } =
        calculate_initial_tx_gas(SPEC_ID.into(), &input[..], false, 0, 0, 0);

    let ctx = Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(bls12_381_const::MAP_FP_TO_G1_ADDRESS))
                        .data(input)
                        .gas_limit(
                            initial_total_gas + bls12_381_const::MAP_FP_TO_G1_BASE_GAS_FEE - 1,
                        ),
                )
                .build_fill(),
        )
        .modify_chain_chained(|l1_block| {
            l1_block.operator_fee_constant = Some(U256::ZERO);
            l1_block.operator_fee_scalar = Some(U256::ZERO)
        })
        .with_cfg(CfgEnv::new_with_spec(SPEC_ID));

    let mut evm = ctx.build_op();
    let output = evm.replay().unwrap();

    // assert out of gas
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::OutOfGas(OutOfGasError::Precompile)),
            ..
        }
    ));

    compare_or_save_op_testdata(
        "test_halted_tx_call_bls12_381_map_fp_to_g1_out_of_gas.json",
        &output,
    );
}

#[test]
fn test_halted_tx_call_bls12_381_map_fp_to_g1_input_wrong_size() {
    const SPEC_ID: OpSpecId = OpSpecId::ISTHMUS;
    let input = Bytes::from([1; bls12_381_const::PADDED_FP_LENGTH]);
    let InitialAndFloorGas { initial_total_gas, .. } =
        calculate_initial_tx_gas(SPEC_ID.into(), &input[..], false, 0, 0, 0);

    let ctx = Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(bls12_381_const::MAP_FP_TO_G1_ADDRESS))
                        .data(input.slice(1..))
                        .gas_limit(initial_total_gas + bls12_381_const::MAP_FP_TO_G1_BASE_GAS_FEE),
                )
                .build_fill(),
        )
        .modify_chain_chained(|l1_block| {
            l1_block.operator_fee_constant = Some(U256::ZERO);
            l1_block.operator_fee_scalar = Some(U256::ZERO)
        })
        .with_cfg(CfgEnv::new_with_spec(SPEC_ID));

    let mut evm = ctx.build_op();
    let output = evm.replay().unwrap();

    // assert fails post gas check, because input is wrong size
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::PrecompileErrorWithContext(ref msg)),
            ..
        } if msg == "bls12-381 map fp to g1 input length error"
    ));

    compare_or_save_op_testdata(
        "test_halted_tx_call_bls12_381_map_fp_to_g1_input_wrong_size.json",
        &output,
    );
}

#[test]
fn test_halted_tx_call_bls12_381_map_fp2_to_g2_out_of_gas() {
    const SPEC_ID: OpSpecId = OpSpecId::ISTHMUS;
    let input = Bytes::from([1; bls12_381_const::PADDED_FP2_LENGTH]);
    let InitialAndFloorGas { initial_total_gas, .. } =
        calculate_initial_tx_gas(SPEC_ID.into(), &input[..], false, 0, 0, 0);

    let ctx = Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(bls12_381_const::MAP_FP2_TO_G2_ADDRESS))
                        .data(input)
                        .gas_limit(
                            initial_total_gas + bls12_381_const::MAP_FP2_TO_G2_BASE_GAS_FEE - 1,
                        ),
                )
                .build_fill(),
        )
        .modify_chain_chained(|l1_block| {
            l1_block.operator_fee_constant = Some(U256::ZERO);
            l1_block.operator_fee_scalar = Some(U256::ZERO)
        })
        .with_cfg(CfgEnv::new_with_spec(SPEC_ID));

    let mut evm = ctx.build_op();
    let output = evm.replay().unwrap();

    // assert out of gas
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::OutOfGas(OutOfGasError::Precompile)),
            ..
        }
    ));

    compare_or_save_op_testdata(
        "test_halted_tx_call_bls12_381_map_fp2_to_g2_out_of_gas.json",
        &output,
    );
}

#[test]
fn test_l1block_load_for_pre_regolith() {
    const SPEC_ID: OpSpecId = OpSpecId::REGOLITH;

    let ctx = Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .caller(BENCH_CALLER)
                        .kind(TxKind::Call(address!("0x0000000000000000000000000000000000100000")))
                        .value(U256::from(1))
                        .gas_limit(100_000),
                )
                .build_fill(),
        )
        .modify_chain_chained(|l1_block| {
            l1_block.l2_block = None;
        })
        .with_cfg(CfgEnv::new_with_spec(SPEC_ID));

    let mut evm =
        ctx.with_db(State::builder().with_database(BenchmarkDB::default()).build()).build_op();
    let output = evm.replay().unwrap();

    // assert out of gas
    assert!(output.result.is_success());

    compare_or_save_op_testdata("test_l1block_load_for_pre_regolith.json", &output);
}

#[test]
fn test_halted_tx_call_bls12_381_map_fp2_to_g2_input_wrong_size() {
    const SPEC_ID: OpSpecId = OpSpecId::ISTHMUS;
    let input = Bytes::from([1; bls12_381_const::PADDED_FP2_LENGTH]);
    let InitialAndFloorGas { initial_total_gas, .. } =
        calculate_initial_tx_gas(SPEC_ID.into(), &input[..], false, 0, 0, 0);

    let ctx = Context::op()
        .with_tx(
            OpTransaction::builder()
                .base(
                    TxEnv::builder()
                        .kind(TxKind::Call(bls12_381_const::MAP_FP2_TO_G2_ADDRESS))
                        .data(input.slice(1..))
                        .gas_limit(initial_total_gas + bls12_381_const::MAP_FP2_TO_G2_BASE_GAS_FEE),
                )
                .build_fill(),
        )
        .modify_chain_chained(|l1_block| {
            l1_block.operator_fee_constant = Some(U256::ZERO);
            l1_block.operator_fee_scalar = Some(U256::ZERO)
        })
        .with_cfg(CfgEnv::new_with_spec(SPEC_ID));

    let mut evm = ctx.build_op();
    let output = evm.replay().unwrap();

    // assert fails post gas check, because input is wrong size
    assert!(matches!(
        output.result,
        ExecutionResult::Halt {
            reason: OpHaltReason::Base(HaltReason::PrecompileErrorWithContext(ref msg)),
            ..
        } if msg == "bls12-381 map fp2 to g2 input length error"
    ));

    compare_or_save_op_testdata(
        "test_halted_tx_call_bls12_381_map_fp2_to_g2_input_wrong_size.json",
        &output,
    );
}

#[test]
#[cfg(feature = "optional_balance_check")]
fn test_disable_balance_check() {
    const RETURN_CALLER_BALANCE_BYTECODE: &[u8] = &[
        opcode::CALLER,
        opcode::BALANCE,
        opcode::PUSH1,
        0x00,
        opcode::MSTORE,
        opcode::PUSH1,
        0x20,
        opcode::PUSH1,
        0x00,
        opcode::RETURN,
    ];

    let mut evm = Context::op()
        .modify_cfg_chained(|cfg| cfg.disable_balance_check = true)
        .with_db(BenchmarkDB::new_bytecode(Bytecode::new_legacy(
            RETURN_CALLER_BALANCE_BYTECODE.into(),
        )))
        .build_op();

    // Construct tx so that effective cost is more than caller balance.
    let gas_price = 1;
    let gas_limit = 100_000;
    // Make sure value doesn't consume all balance since we want to validate that all effective
    // cost is deducted.
    let tx_value = BENCH_CALLER_BALANCE - U256::from(1);

    let result = evm
        .transact_one(
            OpTransaction::builder()
                .base(
                    TxEnv::builder_for_bench()
                        .gas_price(gas_price)
                        .gas_limit(gas_limit)
                        .value(tx_value),
                )
                .build_fill(),
        )
        .unwrap();

    assert!(result.is_success());

    let returned_balance = U256::from_be_slice(result.output().unwrap().as_ref());
    let expected_balance = U256::ZERO;
    assert_eq!(returned_balance, expected_balance);
}

#[derive(Default, Debug)]
struct LogInspector {
    logs: Vec<Log>,
}

impl<CTX, INTR: InterpreterTypes> Inspector<CTX, INTR> for LogInspector {
    fn log(&mut self, _context: &mut CTX, log: Log) {
        self.logs.push(log);
    }
}

#[test]
fn test_log_inspector() {
    // simple yul contract emits a log in constructor

    /*object "Contract" {
        code {
            log0(0, 0)
        }
    }*/

    let contract_data: Bytes =
        Bytes::from([opcode::PUSH1, 0x00, opcode::DUP1, opcode::LOG0, opcode::STOP]);
    let bytecode = Bytecode::new_raw(contract_data);

    let ctx = Context::op().with_db(BenchmarkDB::new_bytecode(bytecode));

    let mut evm = ctx.build_op_with_inspector(LogInspector::default());

    let tx = OpTransaction::builder()
        .base(TxEnv::builder().caller(BENCH_CALLER).kind(TxKind::Call(BENCH_TARGET)))
        .build_fill();

    // Run evm.
    let output = evm.inspect_tx(tx).unwrap();

    let inspector = &evm.0.inspector;
    assert!(!inspector.logs.is_empty());

    compare_or_save_op_testdata("test_log_inspector.json", &output);
}

#[test]
fn test_system_call_inspection() {
    use revm::InspectSystemCallEvm;

    let ctx = Context::op();

    let mut evm = ctx.build_op_with_inspector(LogInspector::default());

    // Test system call inspection
    let result = evm.inspect_one_system_call(BENCH_TARGET, Bytes::default()).unwrap();

    // Should succeed
    assert!(result.is_success());

    // Test system call inspection with caller
    let custom_caller = Address::from([0x12; 20]);
    let result = evm
        .inspect_one_system_call_with_caller(custom_caller, BENCH_TARGET, Bytes::default())
        .unwrap();

    // Should also succeed
    assert!(result.is_success());

    // Test system call inspection with inspector
    let result = evm
        .inspect_one_system_call_with_inspector(
            BENCH_TARGET,
            Bytes::default(),
            LogInspector::default(),
        )
        .unwrap();

    // Should succeed
    assert!(result.is_success());
}

#[test]
fn test_system_call() {
    let ctx = Context::op();

    let mut evm = ctx.build_op();

    let _ = evm.system_call_one(BENCH_TARGET, bytes!("0x0001"));
    let state = evm.finalize();

    assert!(!state.contains_key(&SYSTEM_ADDRESS));
    assert!(state.get(&BENCH_TARGET).unwrap().is_touched());
}
