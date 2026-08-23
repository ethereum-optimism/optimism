//! The experimental `opstack` block-building namespace on the chain routes.
//!
//! op-supernode's virtual op-nodes serve `opstack_*` on each chain's route when the experimental
//! sequencer API is enabled (`op-node/node/node.go`, gated on `ExperimentalOPStackAPI`), and the
//! op-test-sequencer's standard builder, committer and publisher drive block building through
//! those methods. The devstack turns the flag on for every hosted chain.
//!
//! Building a real block needs a live execution layer, which is the devstack's job. What the
//! stubbed processes here establish is the wiring no unit test can reach: that the namespace is
//! registered on each chain's route exactly when the chain's configuration asks for it, that it
//! stays off the supernode's root, and that the methods answer with op-node's error surface — a
//! missing parent is refused the way op-node refuses it, a payload the engine does not know comes
//! back as the `-40120` unknown-payload build error's family, not as "Method not found".

use jsonrpsee::{
    core::{ClientError, client::ClientT},
    http_client::{HttpClient, HttpClientBuilder},
    rpc_params,
};
use serde_json::{Value, json};
use std::{
    io::{BufRead, BufReader, Read, Write},
    net::{SocketAddr, TcpListener, TcpStream},
    path::Path,
    process::{Child, Command, Stdio},
    sync::mpsc,
    thread,
    time::Duration,
};
use tokio::time::{Instant, sleep};

/// A chain from the superchain registry, so the test needs no rollup-config fixture.
const CHAIN: u64 = 10;

/// How long a chain gets to bind its RPC route. Composing a chain builds a P2P stack and reaches
/// out to the L1, so this is generous.
const READY_TIMEOUT: Duration = Duration::from_secs(60);

/// Without the flag, a chain's route refuses the namespace the way it refuses any method it does
/// not serve: `-32601`. This is what every route answered before the namespace existed, and what
/// a chain whose operator did not opt in keeps answering.
#[tokio::test(flavor = "multi_thread")]
async fn the_opstack_namespace_stays_off_without_its_flag() {
    let stub = Stub::start();
    let dir = tempfile::tempdir().expect("temp dir");
    let toml = config(dir.path(), &stub, false);

    let node = Node::start(&dir.path().join("cfg.toml"), &toml);
    let chain = node.chain_client(CHAIN);
    await_route(&chain).await;

    let err = open_block(&chain).await.expect_err("the namespace must be off");
    assert_eq!(error_code(&err), Some(-32601), "an unregistered namespace is Method not found");
}

/// With the flag, the chain's route serves the namespace with op-node's semantics: the methods
/// exist, a parent the engine does not have is refused with op-node's message, a build job the
/// engine does not know is a build error rather than a JSON-RPC default, and the supernode's root
/// still does not serve any of it — op-supernode's root does not either.
#[tokio::test(flavor = "multi_thread")]
async fn the_opstack_namespace_answers_on_the_chains_route() {
    let stub = Stub::start();
    let dir = tempfile::tempdir().expect("temp dir");
    let toml = config(dir.path(), &stub, true);

    let node = Node::start(&dir.path().join("cfg.toml"), &toml);
    let chain = node.chain_client(CHAIN);
    await_route(&chain).await;

    // openBlockV1: the stubbed engine has no blocks, so the parent lookup that op-node's
    // `OpenBlock` makes first fails with op-node's own refusal — proof the method is served and
    // reaches the engine, without an execution layer that can actually build.
    let err = open_block(&chain).await.expect_err("the stub engine has no parent block");
    assert_eq!(
        error_code(&err),
        Some(-32000),
        "an uncoded op-node error carries the default server error code: {err}"
    );
    assert!(
        err.to_string().contains("failed to retrieve parent block"),
        "the refusal is op-node's: {err}"
    );

    // sealBlockV1 / cancelBlockV1: the stub answers `null` where a payload envelope belongs, so
    // the seal fails inside `engine_getPayload` — as the `-40199` catch-all of op-node's
    // `SealBlock`, not as a missing method and not as a default server error.
    let seal: Result<Value, _> = chain
        .request(
            "opstack_sealBlockV1",
            rpc_params![json!({"id": "0x0102030405060708", "timestamp": 1234})],
        )
        .await;
    let err = seal.expect_err("the stub engine has no payload to seal");
    assert_eq!(error_code(&err), Some(-40199), "a seal failure is a build error: {err}");
    assert!(err.to_string().contains("failed to seal payload"), "{err}");

    // commitBlockV1: the envelope parses (the wire shape is Go's), the write reaches the chain
    // controller's queue, and the engine's refusal comes back to this caller — op-node's
    // "failed to insert payload" — rather than being dropped the way a bad gossip payload is.
    let commit: Result<Value, _> =
        chain.request("opstack_commitBlockV1", rpc_params![signed_envelope()]).await;
    let err = commit.expect_err("the stub engine cannot insert a payload");
    assert_eq!(error_code(&err), Some(-32000), "{err}");
    assert!(err.to_string().contains("failed to insert payload"), "{err}");

    // publishBlockV1: scheduling the signed payload onto the chain's gossip topic succeeds with
    // no peers to hear it; op-node's dsl callers ignore publish errors for the same reason.
    let published: Result<Option<Value>, _> =
        chain.request("opstack_publishBlockV1", rpc_params![signed_envelope()]).await;
    published.expect("publishing schedules the payload and answers");

    // The namespace belongs to the chains: the root keeps the supernode's own method set, as
    // op-supernode's root does.
    let err = open_block(node.admin()).await.expect_err("the root must not build blocks");
    assert_eq!(error_code(&err), Some(-32601), "the root does not serve opstack: {err}");
}

/// Calls `opstack_openBlockV1` with the JSON shapes Go's opstack client sends
/// (`op-service/sources/opstack_client.go`): an `eth.BlockID` and `eth.PayloadAttributes`.
async fn open_block(client: &HttpClient) -> Result<Value, ClientError> {
    let parent = json!({
        "hash": format!("0x{}", "11".repeat(32)),
        "number": 1,
    });
    let attrs = json!({
        "timestamp": "0x64",
        "prevRandao": format!("0x{}", "00".repeat(32)),
        "suggestedFeeRecipient": format!("0x{}", "00".repeat(20)),
        "withdrawals": null,
        "parentBeaconBlockRoot": null,
        "transactions": null,
        "noTxPool": true,
        "gasLimit": "0x1c9c380",
    });
    client.request("opstack_openBlockV1", rpc_params![parent, attrs]).await
}

/// A signed execution payload envelope in Go's wire shape
/// (`opsigner.SignedExecutionPayloadEnvelope`): a pre-Ecotone V1 payload and a 65-byte signature.
fn signed_envelope() -> Value {
    json!({
        "envelope": {
            "parentBeaconBlockRoot": null,
            "executionPayload": {
                "parentHash": format!("0x{}", "11".repeat(32)),
                "feeRecipient": format!("0x{}", "00".repeat(20)),
                "stateRoot": format!("0x{}", "22".repeat(32)),
                "receiptsRoot": format!("0x{}", "33".repeat(32)),
                "logsBloom": format!("0x{}", "00".repeat(256)),
                "prevRandao": format!("0x{}", "00".repeat(32)),
                "blockNumber": "0x1",
                "gasLimit": "0x1c9c380",
                "gasUsed": "0x0",
                "timestamp": "0x64",
                "extraData": "0x",
                "baseFeePerGas": "0x1",
                "blockHash": format!("0x{}", "44".repeat(32)),
                "transactions": [],
            },
        },
        // 65 bytes: r and s zero, v = 27, which is a parseable secp256k1 signature encoding.
        "signature": format!("0x{}1b", "00".repeat(64)),
    })
}

/// The JSON-RPC error code of a client error, if the server answered with one.
fn error_code(err: &ClientError) -> Option<i32> {
    match err {
        ClientError::Call(err) => Some(err.code()),
        _ => None,
    }
}

/// Polls the chain's route until it answers, so the assertions run against a composed chain.
async fn await_route(client: &HttpClient) {
    let deadline = Instant::now() + READY_TIMEOUT;
    loop {
        match client.request::<Value, _>("optimism_rollupConfig", rpc_params![]).await {
            Ok(_) => return,
            Err(err) => {
                assert!(Instant::now() < deadline, "the chain route never answered: {err}");
                sleep(Duration::from_millis(200)).await;
            }
        }
    }
}

/// Renders a one-chain configuration, with the opstack namespace on or off.
fn config(dir: &Path, stub: &Stub, opstack: bool) -> String {
    let jwt = dir.join("jwt.hex");
    std::fs::write(&jwt, format!("0x{}", "11".repeat(32))).expect("write the jwt secret");

    format!(
        r#"
[l1]
eth-rpc = "{stub}"
beacon = "{stub}"

[defaults]
datadir = "{datadir}"
jwt-secret = "{jwt}"
mode = "validator"
rpc-enable-admin = true
experimental-opstack-api = {opstack}
p2p-listen-ip = "127.0.0.1"

[admin]
rpc-addr = "127.0.0.1"
rpc-port = 0

[[chains]]
l2-chain-id = {CHAIN}
engine-rpc = "{stub}"
p2p-tcp-port = 0
p2p-udp-port = 0
unsafe-block-signer = "0x1111111111111111111111111111111111111111"
"#,
        stub = stub.url(),
        datadir = dir.join("data").display(),
        jwt = jwt.display(),
    )
}

/// The L1, the beacon API and the chain's engine API, on one socket, answering `null` to every
/// RPC call: enough for the chain to compose and serve its route, with no chain moving.
///
/// The same stub `two_chains.rs` runs against; see there for why the two beacon responses are
/// real.
struct Stub {
    addr: SocketAddr,
}

impl Stub {
    fn url(&self) -> String {
        format!("http://{}", self.addr)
    }

    fn start() -> Self {
        let listener = TcpListener::bind("127.0.0.1:0").expect("bind the stub");
        let addr = listener.local_addr().expect("stub addr");

        thread::spawn(move || {
            for stream in listener.incoming() {
                let Ok(stream) = stream else { break };
                thread::spawn(move || Self::serve(stream));
            }
        });

        Self { addr }
    }

    fn serve(mut stream: TcpStream) {
        let mut reader = BufReader::new(stream.try_clone().expect("clone the stub stream"));
        loop {
            let mut request = String::new();
            let mut length = 0usize;
            loop {
                let mut line = String::new();
                if reader.read_line(&mut line).unwrap_or(0) == 0 {
                    return;
                }
                if let Some(value) = line.to_ascii_lowercase().strip_prefix("content-length:") {
                    length = value.trim().parse().unwrap_or(0);
                }
                if line == "\r\n" || line == "\n" {
                    break;
                }
                request.push_str(&line);
            }

            let mut body = vec![0u8; length];
            if reader.read_exact(&mut body).is_err() {
                return;
            }

            let response =
                if request.starts_with("GET") { Self::beacon(&request) } else { Self::rpc(&body) }
                    .to_string();
            let head = format!(
                "HTTP/1.1 200 OK\r\ncontent-type: application/json\r\ncontent-length: {}\r\n\r\n",
                response.len()
            );
            if stream.write_all(head.as_bytes()).is_err() ||
                stream.write_all(response.as_bytes()).is_err()
            {
                return;
            }
        }
    }

    fn beacon(request: &str) -> Value {
        if request.contains("/eth/v1/beacon/genesis") {
            json!({"data": {"genesis_time": "0"}})
        } else if request.contains("/eth/v1/config/spec") {
            json!({"data": {"SECONDS_PER_SLOT": "12"}})
        } else {
            json!({"data": {}})
        }
    }

    fn rpc(body: &[u8]) -> Value {
        let request: Value = serde_json::from_slice(body).unwrap_or(Value::Null);
        let reply = |call: &Value| json!({"jsonrpc": "2.0", "id": call["id"].clone(), "result": Value::Null});
        match &request {
            Value::Array(calls) => Value::Array(calls.iter().map(reply).collect()),
            call => reply(call),
        }
    }
}

/// A running `lokahi` process, stopped when the test that started it ends.
///
/// The same launch-and-handshake `two_chains.rs` performs: start the binary, wait for the logged
/// address of the one socket everything answers on.
struct Node {
    child: Child,
    base: String,
    admin: HttpClient,
}

impl Node {
    fn start(path: &Path, config: &str) -> Self {
        std::fs::write(path, config).expect("write the config");

        let mut child = Command::new(env!("CARGO_BIN_EXE_lokahi"))
            .args(["node", "--config", path.to_str().expect("utf-8 path")])
            .env("KONA_LOG_STDOUT_FORMAT", "json")
            .stdout(Stdio::piped())
            .stderr(Stdio::inherit())
            .spawn()
            .expect("start lokahi");

        let stdout = child.stdout.take().expect("piped stdout");
        let (tx, rx) = mpsc::channel();
        thread::spawn(move || {
            for line in BufReader::new(stdout).lines().map_while(Result::ok) {
                eprintln!("[lokahi] {line}");
                if let Some(addr) = admin_addr(&line) {
                    let _ = tx.send(addr);
                }
            }
        });

        let addr =
            rx.recv_timeout(READY_TIMEOUT).expect("lokahi did not log its admin RPC address");
        let base = format!("http://{addr}");
        let admin = HttpClientBuilder::default().build(&base).expect("build the admin rpc client");

        Self { child, base, admin }
    }

    /// The supernode's own namespaces, at the root of its socket.
    const fn admin(&self) -> &HttpClient {
        &self.admin
    }

    /// A client on chain `chain_id`'s route.
    fn chain_client(&self, chain_id: u64) -> HttpClient {
        HttpClientBuilder::default()
            .build(format!("{}/{chain_id}", self.base))
            .expect("build a chain rpc client")
    }
}

impl Drop for Node {
    fn drop(&mut self) {
        let _ = self.child.kill();
        let _ = self.child.wait();
    }
}

/// Extracts the admin RPC address from a structured log line.
fn admin_addr(line: &str) -> Option<SocketAddr> {
    let entry: Value = serde_json::from_str(line).ok()?;
    let fields = entry.get("fields")?;
    (fields.get("message")?.as_str()? == "Admin RPC server bound to address")
        .then(|| fields.get("addr")?.as_str()?.parse().ok())
        .flatten()
}
