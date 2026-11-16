//! Runs in its own process, so no seed is installed.

#[test]
#[should_panic(expected = "global hash seed not installed")]
fn random_state_before_install_panics() {
    let _ = foldhash::fast::RandomState::default();
}
