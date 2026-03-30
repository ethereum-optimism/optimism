//! Integration tests that run op-program and kona-host against test chains.
//!
//! Each scenario is defined with `run_all_programs!`, which automatically
//! generates one test per program implementation. To add a new program,
//! update the macro — all scenarios get it for free.
//!
//! These tests require external binaries and are excluded from `just test-derivation`.
//! Run with: `just test-derivation-integration` (from rust/)
//! Run a single program: `just test-derivation-op-program` or `just test-derivation-kona-host`

use std::{future::Future, pin::Pin, process::ExitStatus};

use alloy_primitives::{Address, U256};
use derivation_tests::{
    config::DeterministicConfig,
    harness::{
        BatchConfig, DerivationTest, RunConfig, run_config_from_test, run_kona_host, run_op_program,
    },
};

// ---------------------------------------------------------------------------
// Program abstraction
// ---------------------------------------------------------------------------

type ProgramRunFn = for<'a> fn(
    &'a RunConfig,
    &'a DeterministicConfig,
) -> Pin<
    Box<dyn Future<Output = Result<ExitStatus, Box<dyn std::error::Error>>> + 'a>,
>;

struct Program {
    name: &'static str,
    env_var: &'static str,
    run: ProgramRunFn,
}

const OP_PROGRAM: Program = Program {
    name: "op-program",
    env_var: "OP_PROGRAM_PATH",
    run: |config, det_config| Box::pin(run_op_program(config, det_config)),
};

const KONA_HOST: Program = Program {
    name: "kona-host",
    env_var: "KONA_HOST_PATH",
    run: |config, det_config| Box::pin(run_kona_host(config, det_config)),
};

async fn run_program(build: fn() -> DerivationTest, program: &Program) {
    assert!(
        std::env::var(program.env_var).is_ok(),
        "{} not set. Build the binary first (see just test-derivation-integration).",
        program.env_var
    );

    let test = build();
    let servers = test.serve().await.unwrap();
    let config = run_config_from_test(&test, &servers);

    let status = (program.run)(&config, &test.config)
        .await
        .unwrap_or_else(|e| panic!("{} failed to execute: {e}", program.name));
    servers.stop();

    assert!(status.success(), "{} should exit 0, got: {status}", program.name,);
}

/// Generates one `#[tokio::test]` per program for a given scenario.
///
/// Adding a new program means adding one line here — every scenario gets it
/// automatically.
macro_rules! run_all_programs {
    ($scenario:ident) => {
        paste::paste! {
            #[tokio::test]
            async fn [<test_op_program_ $scenario>]() {
                run_program($scenario, &OP_PROGRAM).await;
            }

            #[tokio::test]
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
    test.advance_l1(2);
    test.derive_empty_l2_block();
    test.submit_batch_with(BatchConfig::singular_calldata());
    test.finalize();
    test
}

/// A 1 ETH transfer submitted as a singular batch.
fn with_batch() -> DerivationTest {
    let mut test = DerivationTest::new();
    test.advance_l1(2);
    test.derive_l2_block()
        .with_funded_transfer(
            Address::with_last_byte(0x99),
            U256::from(1_000_000_000_000_000_000u64),
        )
        .build();
    test.submit_batch_with(BatchConfig::singular_calldata());
    test.finalize();
    test
}

// Generate tests: each line produces test_op_program_<name> + test_kona_host_<name>
run_all_programs!(empty_blocks);
run_all_programs!(with_batch);
