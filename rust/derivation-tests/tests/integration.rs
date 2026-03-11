//! Integration tests that run op-program and kona-host against test chains.
//!
//! Each scenario is defined with `run_all_programs!`, which automatically
//! generates one test per program implementation. To add a new program,
//! update the macro — all scenarios get it for free.
//!
//! These tests require external binaries and are marked `#[ignore]`.
//! Run with: `just test-derivation-integration` (from rust/)

mod helpers;

use std::future::Future;
use std::pin::Pin;
use std::process::ExitStatus;

use derivation_tests::harness::{
    DerivationTest, RunConfig, run_config_from_test, run_kona_host, run_op_program,
};

// ---------------------------------------------------------------------------
// Program abstraction
// ---------------------------------------------------------------------------

struct Program {
    name: &'static str,
    env_var: &'static str,
    run: for<'a> fn(
        &'a RunConfig,
        &'a kona_genesis::RollupConfig,
        &'a alloy_genesis::ChainConfig,
    ) -> Pin<Box<dyn Future<Output = Result<ExitStatus, Box<dyn std::error::Error>>> + 'a>>,
}

const OP_PROGRAM: Program = Program {
    name: "op-program",
    env_var: "OP_PROGRAM_PATH",
    run: |config, rollup, l1_chain| Box::pin(run_op_program(config, rollup, l1_chain)),
};

const KONA_HOST: Program = Program {
    name: "kona-host",
    env_var: "KONA_HOST_PATH",
    run: |config, rollup, l1_chain| Box::pin(run_kona_host(config, rollup, l1_chain)),
};

async fn run_program(build: fn() -> DerivationTest, program: &Program) {
    if std::env::var(program.env_var).is_err() {
        eprintln!("SKIP: {} not set.", program.env_var);
        return;
    }

    let test = build();
    let servers = test.serve().await.unwrap();
    let config = run_config_from_test(&test, &servers);
    let rollup_config = test.config.rollup_config();
    let l1_chain_config = test.config.l1_chain_config();

    let status = (program.run)(&config, &rollup_config, &l1_chain_config)
        .await
        .unwrap_or_else(|e| panic!("{} failed to execute: {e}", program.name));
    servers.stop();

    assert!(
        status.success(),
        "{} should exit 0, got: {status}",
        program.name,
    );
}

/// Generates one `#[tokio::test] #[ignore]` per program for a given scenario.
///
/// Adding a new program means adding one line here — every scenario gets it
/// automatically.
macro_rules! run_all_programs {
    ($scenario:ident) => {
        paste::paste! {
            #[tokio::test]
            #[ignore]
            async fn [<test_op_program_ $scenario>]() {
                run_program($scenario, &OP_PROGRAM).await;
            }

            #[tokio::test]
            #[ignore]
            async fn [<test_kona_host_ $scenario>]() {
                run_program($scenario, &KONA_HOST).await;
            }
        }
    };
}

// ---------------------------------------------------------------------------
// Scenarios
// ---------------------------------------------------------------------------

/// Empty (deposit-only) L2 blocks derived from empty L1 blocks.
fn empty_blocks() -> DerivationTest {
    let mut test = DerivationTest::new();

    test.l1.emit_empty_block();
    test.l1.emit_empty_block();

    let l1_genesis = test.l1.block_at(0).unwrap().clone();
    test.l2.set_epoch(&l1_genesis);
    let block_ref = test.l2.build_empty_block().unwrap();

    let batch = test.singular_batch_calldata(&[block_ref], &l1_genesis);
    test.l1.emit_block_with_batches(vec![batch]);
    test.l1.emit_empty_block();

    test
}

/// A 1 ETH transfer submitted as a singular batch.
fn with_batch() -> DerivationTest {
    use alloy_consensus::{SignableTransaction, TxEip1559};
    use alloy_primitives::{Address, TxKind, U256};
    use alloy_signer::SignerSync;
    use op_alloy_consensus::OpTxEnvelope;

    let mut test = DerivationTest::new();

    test.l1.emit_empty_block();
    test.l1.emit_empty_block();

    let l1_genesis = test.l1.block_at(0).unwrap().clone();
    test.l2.set_epoch(&l1_genesis);

    let signer = helpers::funded_signer();
    let tx = TxEip1559 {
        chain_id: test.config.l2_chain_id,
        nonce: 0,
        gas_limit: 21_000,
        max_fee_per_gas: 1,
        max_priority_fee_per_gas: 0,
        to: TxKind::Call(Address::with_last_byte(0x99)),
        value: U256::from(1_000_000_000_000_000_000u64),
        ..Default::default()
    };
    let sig = signer
        .sign_hash_sync(&tx.signature_hash())
        .expect("signing works");
    let signed = tx.into_signed(sig);
    let eth_envelope = alloy_consensus::TxEnvelope::Eip1559(signed);
    let op_tx = OpTxEnvelope::try_from_eth_envelope(eth_envelope)
        .expect("should convert ETH envelope to OP envelope");

    let block_ref = test.l2.build_block(vec![op_tx]).unwrap();

    let batch = test.singular_batch_calldata(&[block_ref], &l1_genesis);
    test.l1.emit_block_with_batches(vec![batch]);
    test.l1.emit_empty_block();

    test
}

// Generate tests: each line produces test_op_program_<name> + test_kona_host_<name>
run_all_programs!(empty_blocks);
run_all_programs!(with_batch);
