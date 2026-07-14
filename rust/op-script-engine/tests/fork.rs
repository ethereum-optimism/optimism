//! Fork-mode unit tests against an in-crate mock HTTP JSON-RPC server (no network, no secret, no
//! anvil — runnable anywhere, incl. the required go-tests-short CI job).
//!
//! These exercise the load-bearing fork machinery directly on `ScriptHost`: lazy RPC read-through,
//! block-hash pinning, the copy-on-write overlay + diff write-log, cache memoization,
//! absent-account semantics, the locally-served (persistent/excluded) set, and the createSelectFork
//! / stateDump guards. The ~5-method fork RPC surface is served by `MockL1` below.

use std::{
    collections::HashMap,
    io::{BufRead, BufReader, Read, Write},
    net::TcpListener,
    sync::{Arc, Mutex},
};

use alloy_primitives::{Address, B256, Bytes, U256, address, hex};
use op_script_engine::host::{HostConfig, ScriptHost};

// -------------------------------------------------------------------------------------------------
// Mock L1 archive: a minimal HTTP JSON-RPC server serving the 5 read-only fork methods, all pinned
// to one immutable block. Requests are counted per method so cache behavior is observable.
// -------------------------------------------------------------------------------------------------

#[derive(Default)]
struct MockState {
    block_number: u64,
    block_hash: B256,
    state_root: B256,
    // addr -> (nonce, balance, code)
    accounts: HashMap<Address, (u64, U256, Vec<u8>)>,
    // (addr, slot) -> value
    storage: HashMap<(Address, U256), B256>,
    counts: HashMap<String, usize>,
    // Per-address getCode / getStorageAt fetch counts (for asserting per-account memoization,
    // independent of unrelated loads like the zero-address coinbase).
    code_fetches: HashMap<Address, usize>,
    storage_fetches: HashMap<Address, usize>,
    // Every raw request body seen (for asserting the block-hash pin is on the wire).
    bodies: Vec<String>,
}

#[derive(Clone)]
struct MockL1 {
    url: String,
    state: Arc<Mutex<MockState>>,
}

impl MockL1 {
    fn start() -> Self {
        let listener = TcpListener::bind("127.0.0.1:0").expect("bind mock l1");
        let port = listener.local_addr().unwrap().port();
        let state = Arc::new(Mutex::new(MockState {
            block_number: 8_000_000,
            block_hash: B256::repeat_byte(0xB1),
            state_root: B256::repeat_byte(0x57),
            ..Default::default()
        }));
        let st = state.clone();
        std::thread::spawn(move || {
            for stream in listener.incoming() {
                let Ok(stream) = stream else { continue };
                let st = st.clone();
                std::thread::spawn(move || handle_conn(stream, st));
            }
        });
        Self { url: format!("http://127.0.0.1:{port}/"), state }
    }

    fn set_account(&self, addr: Address, nonce: u64, balance: U256, code: &[u8]) {
        self.state.lock().unwrap().accounts.insert(addr, (nonce, balance, code.to_vec()));
    }

    fn set_storage(&self, addr: Address, slot: U256, value: B256) {
        self.state.lock().unwrap().storage.insert((addr, slot), value);
    }

    fn count(&self, method: &str) -> usize {
        *self.state.lock().unwrap().counts.get(method).unwrap_or(&0)
    }

    fn code_fetches(&self, addr: Address) -> usize {
        *self.state.lock().unwrap().code_fetches.get(&addr).unwrap_or(&0)
    }

    fn storage_fetches(&self, addr: Address) -> usize {
        *self.state.lock().unwrap().storage_fetches.get(&addr).unwrap_or(&0)
    }

    fn block_hash(&self) -> B256 {
        self.state.lock().unwrap().block_hash
    }

    fn any_body_pins_block_hash(&self) -> bool {
        let s = self.state.lock().unwrap();
        let h = hex::encode(s.block_hash);
        // Account reads (not the block lookup itself) must carry the pinned block hash.
        s.bodies.iter().filter(|b| !b.contains("eth_getBlockBy")).any(|b| b.contains(&h))
    }
}

fn handle_conn(stream: std::net::TcpStream, state: Arc<Mutex<MockState>>) {
    let mut reader = BufReader::new(stream);
    loop {
        // Read request line + headers.
        let mut content_length = 0usize;
        let mut line = String::new();
        if reader.read_line(&mut line).unwrap_or(0) == 0 {
            return; // connection closed
        }
        loop {
            let mut h = String::new();
            if reader.read_line(&mut h).unwrap_or(0) == 0 {
                return;
            }
            if h == "\r\n" || h == "\n" {
                break;
            }
            let lower = h.to_ascii_lowercase();
            if let Some(v) = lower.strip_prefix("content-length:") {
                content_length = v.trim().parse().unwrap_or(0);
            }
        }
        let mut body = vec![0u8; content_length];
        if reader.read_exact(&mut body).is_err() {
            return;
        }
        let body_str = String::from_utf8_lossy(&body).into_owned();
        let resp = dispatch(&body_str, &state);
        let out = format!(
            "HTTP/1.1 200 OK\r\ncontent-type: application/json\r\ncontent-length: {}\r\nconnection: keep-alive\r\n\r\n{}",
            resp.len(),
            resp
        );
        if reader.get_mut().write_all(out.as_bytes()).is_err() {
            return;
        }
    }
}

fn parse_addr(v: &serde_json::Value) -> Address {
    v.as_str().unwrap().parse().unwrap()
}

fn parse_u256_hex(v: &serde_json::Value) -> U256 {
    let s = v.as_str().unwrap();
    U256::from_str_radix(s.strip_prefix("0x").unwrap_or(s), 16).unwrap()
}

fn dispatch(body: &str, state: &Arc<Mutex<MockState>>) -> String {
    let req: serde_json::Value = serde_json::from_str(body).expect("valid json-rpc request");
    let id = req.get("id").cloned().unwrap_or_else(|| serde_json::json!(1));
    let method = req.get("method").and_then(|m| m.as_str()).unwrap_or("");
    let params = req.get("params").cloned().unwrap_or(serde_json::json!([]));
    let p = params.as_array().cloned().unwrap_or_default();

    let mut st = state.lock().unwrap();
    *st.counts.entry(method.to_string()).or_insert(0) += 1;
    st.bodies.push(body.to_string());
    if std::env::var("MOCK_DEBUG").is_ok() {
        eprintln!("[mock] {} {}", method, params);
    }

    let result: serde_json::Value = match method {
        "eth_chainId" => serde_json::json!("0x1"),
        "eth_getBlockByNumber" | "eth_getBlockByHash" => block_json(&st),
        "eth_getTransactionCount" => {
            let a = parse_addr(&p[0]);
            let n = st.accounts.get(&a).map(|x| x.0).unwrap_or(0);
            serde_json::json!(format!("0x{n:x}"))
        }
        "eth_getBalance" => {
            let a = parse_addr(&p[0]);
            let b = st.accounts.get(&a).map(|x| x.1).unwrap_or(U256::ZERO);
            serde_json::json!(format!("0x{b:x}"))
        }
        "eth_getCode" => {
            let a = parse_addr(&p[0]);
            *st.code_fetches.entry(a).or_insert(0) += 1;
            let c = st.accounts.get(&a).map(|x| x.2.clone()).unwrap_or_default();
            serde_json::json!(format!("0x{}", hex::encode(c)))
        }
        "eth_getStorageAt" => {
            let a = parse_addr(&p[0]);
            let slot = parse_u256_hex(&p[1]);
            *st.storage_fetches.entry(a).or_insert(0) += 1;
            let v = st.storage.get(&(a, slot)).copied().unwrap_or(B256::ZERO);
            serde_json::json!(format!("0x{v:x}"))
        }
        other => {
            return serde_json::json!({
                "jsonrpc": "2.0", "id": id,
                "error": {"code": -32601, "message": format!("mock: unexpected method {other}")}
            })
            .to_string();
        }
    };
    serde_json::json!({"jsonrpc": "2.0", "id": id, "result": result}).to_string()
}

fn block_json(st: &MockState) -> serde_json::Value {
    let empty_trie = "0x56e81f171bcac1a9e0d40e6ea86e421fac7d3fdedaf1e56d9c1f31f9dcb0d7e2"; // unused root ok
    serde_json::json!({
        "hash": format!("0x{:x}", st.block_hash),
        "parentHash": format!("0x{:x}", B256::repeat_byte(0xA0)),
        "sha3Uncles": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
        "miner": "0x0000000000000000000000000000000000000000",
        "stateRoot": format!("0x{:x}", st.state_root),
        "transactionsRoot": empty_trie,
        "receiptsRoot": empty_trie,
        "logsBloom": format!("0x{}", "0".repeat(512)),
        "difficulty": "0x0",
        "number": format!("0x{:x}", st.block_number),
        "gasLimit": "0x1c9c380",
        "gasUsed": "0x0",
        "timestamp": "0x64000000",
        "extraData": "0x",
        "mixHash": format!("0x{:x}", B256::ZERO),
        "nonce": "0x0000000000000000",
        "baseFeePerGas": "0x1",
        "transactions": [],
        "uncles": []
    })
}

// -------------------------------------------------------------------------------------------------
// Test scaffolding
// -------------------------------------------------------------------------------------------------

const DEFAULT_SENDER: Address = address!("0x1804c8ab1f12e6bbf3894d4083f33e07309d1f38");

fn runtime() -> tokio::runtime::Runtime {
    tokio::runtime::Builder::new_multi_thread().enable_all().build().unwrap()
}

fn forked_host(handle: tokio::runtime::Handle, mock: &MockL1) -> ScriptHost {
    let mut host = ScriptHost::new(HostConfig {
        chain_id: 1337,
        runtime_handle: Some(handle),
        ..Default::default()
    });
    let block = mock.state.lock().unwrap().block_number;
    host.create_select_fork(&mock.url, Some(block)).expect("createSelectFork");
    host
}

/// Runtime `SLOAD(0); return` — reads storage slot 0 and returns it (32 bytes).
fn sload_return_bytecode() -> Vec<u8> {
    hex::decode("60005460005260206000f3").unwrap()
}

/// Runtime `EXTCODESIZE(target); return` — returns the code size of `target` (32 bytes).
fn extcodesize_bytecode(target: Address) -> Vec<u8> {
    let mut c = vec![0x73]; // PUSH20
    c.extend_from_slice(target.as_slice());
    c.extend_from_slice(&hex::decode("3b60005260206000f3").unwrap()); // EXTCODESIZE; return
    c
}

/// Runtime `SSTORE(slot, val); stop`.
fn sstore_bytecode(slot: u8, val: u8) -> Vec<u8> {
    vec![0x60, val, 0x60, slot, 0x55, 0x00]
}

// -------------------------------------------------------------------------------------------------
// Tests
// -------------------------------------------------------------------------------------------------

#[test]
fn fork_read_through_and_hash_pinning() {
    let rt = runtime();
    let mock = MockL1::start();
    let c = address!("0x00000000000000000000000000000000000c0de0");
    mock.set_account(c, 1, U256::ZERO, &sload_return_bytecode());
    mock.set_storage(c, U256::ZERO, B256::from(U256::from(0x2a)));

    let mut host = forked_host(rt.handle().clone(), &mock);
    let out = host.call(DEFAULT_SENDER, c, Bytes::new()).expect("call fork contract");
    assert_eq!(
        U256::from_be_slice(&out),
        U256::from(0x2a),
        "read-through SLOAD returns fork value"
    );

    // Every account/storage read must be pinned to the fork block HASH, matching Go RPCSource.
    assert!(mock.any_body_pins_block_hash(), "reads must carry the pinned block hash");
    assert_ne!(mock.block_hash(), B256::ZERO);
}

#[test]
fn fork_reads_are_cached() {
    let rt = runtime();
    let mock = MockL1::start();
    let c = address!("0x00000000000000000000000000000000000c0de1");
    mock.set_account(c, 1, U256::ZERO, &sload_return_bytecode());
    mock.set_storage(c, U256::ZERO, B256::from(U256::from(7)));

    let mut host = forked_host(rt.handle().clone(), &mock);
    for _ in 0..3 {
        let out = host.call(DEFAULT_SENDER, c, Bytes::new()).expect("call");
        assert_eq!(U256::from_be_slice(&out), U256::from(7));
    }
    // CacheDB memoizes: the contract's code + slot are each fetched exactly once across the 3 calls
    // (checked per-address so unrelated loads, e.g. the zero-address coinbase, don't interfere).
    assert_eq!(mock.code_fetches(c), 1, "contract code fetched once");
    assert_eq!(mock.storage_fetches(c), 1, "contract slot fetched once");
    // The pin block is resolved exactly once, at createSelectFork.
    assert_eq!(mock.count("eth_getBlockByNumber"), 1, "block resolved once");
}

#[test]
fn fork_absent_account_reads_as_empty() {
    let rt = runtime();
    let mock = MockL1::start();
    // A contract that reports EXTCODESIZE of an account the fork does NOT know about.
    let absent = address!("0x000000000000000000000000000000000000dead");
    let probe = address!("0x00000000000000000000000000000000000c0de2");
    mock.set_account(probe, 1, U256::ZERO, &extcodesize_bytecode(absent));
    // `absent` is deliberately not registered -> mock returns nonce/balance 0, code "0x".

    let mut host = forked_host(rt.handle().clone(), &mock);
    let out = host.call(DEFAULT_SENDER, probe, Bytes::new()).expect("call");
    assert_eq!(U256::from_be_slice(&out), U256::ZERO, "absent account has code size 0");
}

#[test]
fn locally_served_default_sender_shadows_fork() {
    let rt = runtime();
    let mock = MockL1::start();
    // The fork claims DEFAULT_SENDER has code — if the engine read it from the fork, EXTCODESIZE
    // would be nonzero. It must instead be served from the local overlay (excluded set), size 0.
    mock.set_account(DEFAULT_SENDER, 9, U256::from(1234), &hex::decode("60006000").unwrap());
    let probe = address!("0x00000000000000000000000000000000000c0de3");
    mock.set_account(probe, 1, U256::ZERO, &extcodesize_bytecode(DEFAULT_SENDER));

    let mut host = forked_host(rt.handle().clone(), &mock);
    let out = host.call(DEFAULT_SENDER, probe, Bytes::new()).expect("call");
    assert_eq!(
        U256::from_be_slice(&out),
        U256::ZERO,
        "DEFAULT_SENDER is served locally (excluded), not from the fork"
    );
}

#[test]
fn fork_diff_records_writes_not_reads() {
    let rt = runtime();
    let mock = MockL1::start();
    // Writer contract SSTOREs slot 1 = 0x2a; reader contract only SLOADs.
    let writer = address!("0x00000000000000000000000000000000000c0de4");
    let reader = address!("0x00000000000000000000000000000000000c0de5");
    mock.set_account(writer, 1, U256::ZERO, &sstore_bytecode(1, 0x2a));
    mock.set_account(reader, 1, U256::ZERO, &sload_return_bytecode());
    mock.set_storage(reader, U256::ZERO, B256::from(U256::from(99)));

    let mut host = forked_host(rt.handle().clone(), &mock);
    assert!(!host.fork_diff_any(), "diff empty before any execution");
    host.call(DEFAULT_SENDER, reader, Bytes::new()).expect("read-only call");
    assert!(!host.fork_diff_any(), "a pure SLOAD must not enter the diff");
    host.call(DEFAULT_SENDER, writer, Bytes::new()).expect("write call");
    assert!(host.fork_diff_any(), "the SSTORE must enter the diff");

    let diff = host.fork_diff().expect("fork diff");
    let acc = &diff["account"][format!("0x{writer:x}")];
    // Storage value is a 32-byte hex; balance is a decimal string; nonce is a number.
    assert_eq!(
        acc["storage"][format!("0x{:x}", B256::from(U256::from(1)))],
        serde_json::json!(format!("0x{:x}", B256::from(U256::from(0x2a)))),
    );
    assert!(acc["balance"].is_string(), "balance serializes as a decimal string");
    assert!(acc["nonce"].is_number(), "nonce serializes as a number");
    // The read-only `reader` account is absent from the diff.
    assert!(diff["account"].get(format!("0x{reader:x}")).is_none());
    // DEFAULT_SENDER (excluded) never enters the diff.
    assert!(diff["account"].get(format!("0x{DEFAULT_SENDER:x}")).is_none());
}

#[test]
fn second_create_select_fork_errors() {
    let rt = runtime();
    let mock = MockL1::start();
    let mut host = forked_host(rt.handle().clone(), &mock);
    let err = host.create_select_fork(&mock.url, Some(mock.state.lock().unwrap().block_number));
    assert!(err.is_err(), "a second createSelectFork must error");
    assert!(err.unwrap_err().to_string().contains("already installed"));
}

#[test]
fn create_select_fork_after_execution_errors() {
    let rt = runtime();
    let mock = MockL1::start();
    let mut host = ScriptHost::new(HostConfig {
        chain_id: 1337,
        runtime_handle: Some(rt.handle().clone()),
        ..Default::default()
    });
    // Execute something first (a bare CREATE of empty init code).
    host.create(DEFAULT_SENDER, Bytes::new()).expect("create");
    let err = host.create_select_fork(&mock.url, Some(8_000_000));
    assert!(err.is_err(), "createSelectFork after execution must error");
    assert!(err.unwrap_err().to_string().contains("after a script has executed"));
}

#[test]
fn fork_requires_runtime_handle() {
    // No runtime handle -> fork mode unavailable, loud error (never a silent fallback).
    let mut host = ScriptHost::new(HostConfig { chain_id: 1337, ..Default::default() });
    let err = host.create_select_fork("http://127.0.0.1:1/", Some(1));
    assert!(err.is_err());
    assert!(err.unwrap_err().to_string().contains("tokio runtime handle"));
}
