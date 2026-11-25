//! End-to-end specification for the `lokahi` binary.

use std::process::{Command, Output};

/// Runs the binary under test with `args` and returns its output.
fn run(args: &[&str]) -> Output {
    Command::new(env!("CARGO_BIN_EXE_lokahi"))
        .args(args)
        .output()
        .expect("failed to run the lokahi binary")
}

/// Asserts the command succeeded and returns its stdout.
fn stdout_of(output: &Output) -> &str {
    assert!(
        output.status.success(),
        "exited with {:?}, stderr: {}",
        output.status.code(),
        String::from_utf8_lossy(&output.stderr)
    );
    std::str::from_utf8(&output.stdout).expect("stdout is not utf-8")
}

#[test]
fn prints_the_greeting() {
    let output = run(&[]);
    assert_eq!(stdout_of(&output), "Hello Lokahi\n");
}

#[test]
fn short_version_flag_reports_one_line() {
    let output = run(&["-V"]);
    let stdout = stdout_of(&output);
    let version = stdout.strip_prefix("lokahi ").expect("version is prefixed by the name");
    assert_eq!(version.lines().count(), 1, "unexpected short version: {stdout}");
}

#[test]
fn long_version_flag_reports_build_metadata() {
    let output = run(&["--version"]);
    let stdout = stdout_of(&output);
    for field in ["Version:", "Commit SHA:", "Build Timestamp:", "Build Profile:"] {
        assert!(stdout.contains(field), "{field} missing from long version: {stdout}");
    }
}
