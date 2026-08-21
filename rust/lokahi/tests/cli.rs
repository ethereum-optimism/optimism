//! End-to-end specification for the `lokahi` binary.

use std::{
    io::Write,
    process::{Command, Output},
};

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

/// Asserts the command failed and returns its stderr.
fn stderr_of(output: &Output) -> &str {
    assert!(!output.status.success(), "expected a failing exit status");
    std::str::from_utf8(&output.stderr).expect("stderr is not utf-8")
}

/// Writes `toml` to a temporary file and runs the node subcommand over it.
fn run_node_with_config(toml: &str) -> Output {
    let mut file = tempfile::NamedTempFile::new().expect("temp file");
    file.write_all(toml.as_bytes()).expect("write config");
    run(&["node", "--config", file.path().to_str().expect("utf-8 path")])
}

#[test]
fn running_the_node_needs_a_configuration_file() {
    let output = run(&["node"]);
    let stderr = stderr_of(&output);
    assert!(stderr.contains("--config"), "unexpected error: {stderr}");
}

/// A supernode is a set of chains, so a file that lists none is rejected before any actor starts.
#[test]
fn a_configuration_without_chains_is_rejected() {
    let output = run_node_with_config(
        r#"
        [l1]
        eth-rpc = "http://localhost:8545"
        beacon = "http://localhost:5052"
        "#,
    );

    let stderr = stderr_of(&output);
    assert!(stderr.contains("no chains configured"), "unexpected error: {stderr}");
}

/// Two chains inheriting one P2P port from `[defaults]` fails at startup, naming both chains,
/// rather than as an address-in-use once one chain's gossip is already listening.
#[test]
fn two_chains_sharing_a_port_are_rejected() {
    let output = run_node_with_config(
        r#"
        [l1]
        eth-rpc = "http://localhost:8545"
        beacon = "http://localhost:5052"

        [defaults]
        engine-rpc = "http://localhost:9551"
        jwt-secret = "/etc/lokahi/jwt.hex"
        p2p-tcp-port = 9222
        p2p-udp-port = 9222

        [[chains]]
        l2-chain-id = 901

        [[chains]]
        l2-chain-id = 902
        "#,
    );

    let stderr = stderr_of(&output);
    assert!(stderr.contains("901"), "the first chain is not named: {stderr}");
    assert!(stderr.contains("902"), "the second chain is not named: {stderr}");
}

/// A chain told to sequence with nothing to sign with is rejected before any actor starts, rather
/// than running as a sequencer whose every block is dropped by its peers.
#[test]
fn a_sequencing_chain_without_a_key_is_rejected() {
    let output = run_node_with_config(
        r#"
        [l1]
        eth-rpc = "http://localhost:8545"
        beacon = "http://localhost:5052"

        [defaults]
        engine-rpc = "http://localhost:9551"
        jwt-secret = "/etc/lokahi/jwt.hex"
        p2p-tcp-port = 9222
        p2p-udp-port = 9222

        [[chains]]
        l2-chain-id = 901
        mode = "sequencer"
        "#,
    );

    let stderr = stderr_of(&output);
    assert!(stderr.contains("sequencer-key-path"), "unexpected error: {stderr}");
}

/// The other half of the same rule: a chain that only validates may not carry the settings only a
/// sequencer reads, so a configuration cannot say two things about one chain.
#[test]
fn a_validating_chain_with_sequencer_settings_is_rejected() {
    let output = run_node_with_config(
        r#"
        [l1]
        eth-rpc = "http://localhost:8545"
        beacon = "http://localhost:5052"

        [defaults]
        engine-rpc = "http://localhost:9551"
        jwt-secret = "/etc/lokahi/jwt.hex"
        p2p-tcp-port = 9222
        p2p-udp-port = 9222

        [[chains]]
        l2-chain-id = 901
        sequencer-key-path = "/etc/lokahi/901.key"
        "#,
    );

    let stderr = stderr_of(&output);
    assert!(stderr.contains("does not sequence"), "unexpected error: {stderr}");
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
