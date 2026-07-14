//! Drives the OPCM RunScriptSingle input/output-precompile mechanism end-to-end in-process,
//! against the compiled synthetic `OPCMExample.s.sol` script.

use std::path::PathBuf;

use alloy_primitives::{
    Address, Bytes, U256, address, keccak256,
    map::{HashMap, HashSet},
};
use op_script_engine::host::{HostConfig, ScriptHost};

const DEFAULT_SENDER: Address = address!("0x1804c8ab1f12e6bbf3894d4083f33e07309d1f38");
const TARGET: Address = address!("0x0000000000000000000000000000000000c0ffee");

fn sel(sig: &str) -> [u8; 4] {
    keccak256(sig.as_bytes())[..4].try_into().unwrap()
}

fn artifacts_dir() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("../../op-chain-ops/script/rustengine/testdata/test-artifacts")
}

fn abi_address(a: &Address) -> Bytes {
    let mut out = [0u8; 32];
    out[12..].copy_from_slice(a.as_slice());
    Bytes::from(out.to_vec())
}

fn abi_bytes(data: &[u8]) -> Bytes {
    let padded = data.len().div_ceil(32) * 32;
    let mut out = Vec::with_capacity(64 + padded);
    out.extend_from_slice(&U256::from(0x20).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(data.len()).to_be_bytes::<32>());
    out.extend_from_slice(data);
    out.resize(64 + padded, 0);
    Bytes::from(out)
}

#[test]
fn opcm_run_script_single() {
    let owner = address!("0x1111111111111111111111111111111111111111");
    let blob = b"hello-opcm-parity";

    let mut host = ScriptHost::new(HostConfig {
        chain_id: 1337,
        artifacts_dir: Some(artifacts_dir()),
        ..Default::default()
    });

    // Input snapshot: owner() and blob().
    let mut snapshot: HashMap<[u8; 4], Bytes> = HashMap::default();
    snapshot.insert(sel("owner()"), abi_address(&owner));
    snapshot.insert(sel("blob()"), abi_bytes(blob));
    let input_addr = host.install_input_precompile(snapshot);

    // Output: valid getter result().
    let mut getters: HashSet<[u8; 4]> = HashSet::default();
    getters.insert(sel("result()"));
    let output_addr = host.install_output_precompile(getters);

    // run(inputAddr, outputAddr)
    let mut calldata = Vec::new();
    calldata.extend_from_slice(&sel("run(address,address)"));
    calldata.extend_from_slice(abi_address(&input_addr).as_ref());
    calldata.extend_from_slice(abi_address(&output_addr).as_ref());

    let res =
        host.run_script("OPCMExample.s.sol", "OPCMExample", Bytes::from(calldata), DEFAULT_SENDER);
    let out = res.expect("run_script should succeed");
    println!("run output: 0x{}", alloy_primitives::hex::encode(&out));

    let sets = host.take_captured_sets(output_addr);
    assert_eq!(sets.len(), 1, "exactly one output set() captured");
    // set(bytes4,address) = selector(4) + fieldSel word[4..36] + value word[36..68].
    let s = &sets[0];
    assert_eq!(s.len(), 68, "set(bytes4,address) calldata length");
    assert_eq!(&s[0..4], &sel("set(bytes4,address)"));
    assert_eq!(&s[4..8], &sel("result()"), "field selector = result()");
    assert_eq!(Address::from_slice(&s[48..68]), TARGET, "captured value = TARGET");

    let dump = host.state_dump();
    let key = format!("0x{:x}", TARGET);
    let acc = dump.get(&key).expect("TARGET must be in the dump");
    assert_eq!(acc.balance, format!("0x{:x}", U256::from_be_slice(owner.as_slice())));
    assert!(acc.code.is_some(), "TARGET must have etched code");
    println!("TARGET account: {acc:?}");
}
