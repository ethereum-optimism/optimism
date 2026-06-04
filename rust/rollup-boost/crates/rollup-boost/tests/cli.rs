use assert_cmd::Command;
use predicates::prelude::*;

#[test]
fn test_invalid_args() {
    let mut cmd = Command::new(env!("CARGO_BIN_EXE_rollup-boost"));
    cmd.arg("--invalid-arg");

    cmd.assert().failure().stderr(predicate::str::contains(
        "error: unexpected argument '--invalid-arg' found",
    ));
}
