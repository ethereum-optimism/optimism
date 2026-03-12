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
