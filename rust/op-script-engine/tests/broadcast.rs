//! Rust-side reproduction of `TestScriptBroadcast` (script_test.go:79): runs `runBroadcast()`
//! and checks the captured broadcasts against the Go golden's hardcoded GasUsed/Nonce.

use std::path::PathBuf;

use alloy_primitives::{Address, Bytes, address};
use op_script_engine::{HostConfig, ScriptHost};

const SENDER: Address = address!("0x0000000000000000000000000000000000badc0d");

fn artifacts_dir() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("../../op-chain-ops/script/testdata/test-artifacts")
}

#[test]
fn broadcast_leg() {
    let mut host = ScriptHost::new(HostConfig {
        chain_id: 1337,
        no_max_code_size: false,
        use_create2_deployer: true,
        artifacts_dir: Some(artifacts_dir()),
        ..Default::default()
    });

    let addr = host
        .load_contract("ScriptExample.s.sol", "ScriptExample", ScriptHost::default_sender())
        .expect("deploy");
    host.allow_cheatcodes(addr);
    eprintln!("scriptAddr = {addr:?}");

    // runBroadcast()
    host.call(SENDER, addr, Bytes::from(vec![0xbe, 0xf0, 0x3a, 0xbc])).expect("runBroadcast");

    let bcasts = host.take_broadcasts();
    eprintln!("captured {} broadcasts:", bcasts.len());
    for b in &bcasts {
        eprintln!(
            "  from={} to={} type={} gasUsed={} nonce={} salt={} inputlen={}",
            b.from,
            b.to,
            b.kind,
            b.gas_used,
            b.nonce,
            b.salt,
            b.input.len()
        );
    }

    // Expected from the Go golden (script_test.go:95-160).
    let expected: [(&str, u64, u64); 7] = [
        ("call", 23421, 1),
        ("call", 1521, 0),
        ("call", 1565, 1),
        ("call", 2763, 0),
        ("create", 39112, 0),
        ("create2", 39112, 0),
        ("create", 39112, 2),
    ];
    assert_eq!(bcasts.len(), expected.len(), "broadcast count");
    for (i, (kind, gas, nonce)) in expected.iter().enumerate() {
        assert_eq!(&bcasts[i].kind, kind, "broadcast[{i}] kind");
        assert_eq!(bcasts[i].gas_used, *gas, "broadcast[{i}] gasUsed");
        assert_eq!(bcasts[i].nonce, *nonce, "broadcast[{i}] nonce");
    }

    // Final nonce assertions from the Go golden (script_test.go:187-192).
    assert_eq!(host.get_nonce(SENDER), 0, "sender nonce");
    assert_eq!(host.get_nonce(addr), 3, "script nonce");
    assert_eq!(
        host.get_nonce(address!("0x0000000000000000000000000000000000c0ffee")),
        2,
        "0xc0ffee nonce"
    );
    assert_eq!(host.get_nonce(address!("0x000000000000000000000000000000000000cafe")), 1, "0xcafe nonce");
}
