use super::*;

#[test]
fn pre_refund_limit_uses_evm_gas_not_canonical_gas() {
    let mut info: ExecutionInfo = ExecutionInfo::with_capacity(0);
    info.cumulative_gas_used = 50;
    info.cumulative_evm_gas_used = 90;

    assert!(
        info.is_tx_over_limits(0, 100, None, None, 10, None, None)
            .is_ok(),
        "tx exactly filling the remaining pre-refund budget should fit"
    );

    match info.is_tx_over_limits(0, 100, None, None, 11, None, None) {
        Err(TxnExecutionResult::PreRefundGasLimitExceeded(
            cumulative_evm_gas_used,
            tx_gas_limit,
            block_gas_limit,
        )) => {
            assert_eq!(cumulative_evm_gas_used, 90);
            assert_eq!(tx_gas_limit, 11);
            assert_eq!(block_gas_limit, 100);
        }
        other => panic!("expected pre-refund gas limit error, got {other:?}"),
    }
}
