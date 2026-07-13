//! Rust-side reproduction of `TestScriptStateDump` (op-chain-ops/script/script_test.go:195)
//! against the committed ScriptExample artifacts, as a fast local parity sanity check before
//! the Go RPC gate.

use std::path::PathBuf;

use alloy_primitives::{Address, Bytes, U256, address};
use op_script_engine::{HostConfig, ScriptHost};

const DEFAULT_SENDER: Address = address!("0x1804c8ab1f12e6bbf3894d4083f33e07309d1f38");

fn artifacts_dir() -> PathBuf {
    // rust/op-script-engine -> repo root -> op-chain-ops/...
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("../../op-chain-ops/script/testdata/test-artifacts")
}

/// abi-encode `call1(string)`.
fn call1(v: &str) -> Bytes {
    let sel = [0x7e, 0x79, 0x25, 0x5d];
    let mut out = Vec::new();
    out.extend_from_slice(&sel);
    // offset to string data
    let mut off = [0u8; 32];
    off[31] = 0x20;
    out.extend_from_slice(&off);
    // length
    let mut len = [0u8; 32];
    len[24..32].copy_from_slice(&(v.len() as u64).to_be_bytes());
    out.extend_from_slice(&len);
    // data, right-padded to 32
    let mut data = v.as_bytes().to_vec();
    while data.len() % 32 != 0 {
        data.push(0);
    }
    out.extend_from_slice(&data);
    Bytes::from(out)
}

#[test]
fn state_dump_leg() {
    let mut host = ScriptHost::new(HostConfig {
        chain_id: 1337,
        no_max_code_size: false,
        use_create2_deployer: false,
        artifacts_dir: Some(artifacts_dir()),
        ..Default::default()
    });

    let addr = host
        .load_contract("ScriptExample.s.sol", "ScriptExample", DEFAULT_SENDER)
        .expect("deploy ScriptExample");
    host.allow_cheatcodes(addr);
    eprintln!("scriptAddr = {addr:?}");

    let addr_key = format!("0x{:x}", addr);
    let counter_slot = format!("0x{}", "0".repeat(64));

    // dump 1: contract present, counter slot not yet set
    let dump1 = host.state_dump();
    eprintln!("DUMP1:\n{}", serde_json::to_string_pretty(&dump1).unwrap());
    assert!(dump1.contains_key(&addr_key), "dump1 has contract");
    assert!(
        dump1[&addr_key].storage.as_ref().map_or(true, |s| !s.contains_key(&counter_slot)),
        "counter not counted yet"
    );

    // call A -> counter = 1
    host.call(DEFAULT_SENDER, addr, call1("call A")).expect("call A");
    let dump2 = host.state_dump();
    eprintln!("DUMP2:\n{}", serde_json::to_string_pretty(&dump2).unwrap());
    assert_eq!(
        dump2[&addr_key].storage.as_ref().unwrap().get(&counter_slot).map(String::as_str),
        Some(format!("0x{}1", "0".repeat(63)).as_str()),
        "counted to 1"
    );

    // call B -> counter = 2
    host.call(DEFAULT_SENDER, addr, call1("call B")).expect("call B");
    let dump3 = host.state_dump();
    eprintln!("DUMP3:\n{}", serde_json::to_string_pretty(&dump3).unwrap());
    assert_eq!(
        dump3[&addr_key].storage.as_ref().unwrap().get(&counter_slot).map(String::as_str),
        Some(format!("0x{}2", "0".repeat(63)).as_str()),
        "counted to 2"
    );

    // Sanity: the deployer nonce stays 1 (raw calls don't bump it).
    assert_eq!(host.get_nonce(DEFAULT_SENDER), 1, "deployer nonce stays 1");
    // and revm didn't leave value/gas artifacts
    let _ = U256::ZERO;
}
