use alloy_signer_local::PrivateKeySigner;
use derivation_tests::{config::PREFUNDED_ACCOUNT_KEY, harness::DerivationTest};

/// Create a `DerivationTest` with default config.
#[allow(dead_code)]
pub(crate) fn default_test() -> DerivationTest {
    DerivationTest::new()
}

/// Return a signer for the prefunded test account.
#[allow(dead_code)]
pub(crate) fn funded_signer() -> PrivateKeySigner {
    PrivateKeySigner::from_bytes(&PREFUNDED_ACCOUNT_KEY).expect("valid prefunded key")
}

/// Skip the current test if op-program or kona-host binaries are not available.
///
/// Prints a skip message and returns early when the required env vars are missing.
#[macro_export]
macro_rules! skip_without_programs {
    () => {
        if std::env::var("OP_PROGRAM_PATH").is_err() || std::env::var("KONA_HOST_PATH").is_err() {
            eprintln!(
                "SKIP: OP_PROGRAM_PATH and/or KONA_HOST_PATH not set, skipping program test"
            );
            return;
        }
    };
}
