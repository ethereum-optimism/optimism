//! Two chains, one `lokahi` process, one socket.
//!
//! This is the smoke test for the thing a supernode is for, run against the real binary: one
//! process is given two chains, and each of them ends up answering under its own route on the one
//! socket the process opened, with its own rollup config, while the root of that same socket
//! reports the set it was asked to host.
//!
//! Deriving blocks needs a live L1 and a live execution layer per chain, which is what the
//! devstack's lokahi component exists to provide (`op-devstack/sysgo/l2_cl_lokahi.go`). What the
//! stubbed L1 here establishes is the part that needs no chain to be moving and that no unit test
//! can reach: that two chains compose in one process without colliding, that a request lands on
//! the chain whose id is in the path it was sent to, that a chain the process does not host is a
//! 404, and that one chain's execution layer being unreachable neither stops the other chain nor
//! takes the process down.

use jsonrpsee::{
    core::client::ClientT,
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

/// Chains taken from the superchain registry, so the test needs no rollup-config fixtures of its
/// own and exercises the registry path a chain without a `rollup-config` resolves through.
const CHAIN_A: u64 = 10;
const CHAIN_B: u64 = 130;

/// How each chain's id comes back out of `optimism_rollupConfig`.
///
/// `RollupConfig::l2_chain_id` is an `alloy_chains::Chain`, which serializes a registered id as its
/// name rather than as a number, so this is what identifies the answering chain on the wire.
const CHAIN_A_NAME: &str = "optimism";
const CHAIN_B_NAME: &str = "unichain";

/// How long a chain gets to bind its RPC server. Composing a chain builds a P2P stack and reaches
/// out to the L1, so this is generous; a faster chain does not make the test wait for it.
const READY_TIMEOUT: Duration = Duration::from_secs(60);

#[tokio::test(flavor = "multi_thread")]
async fn two_chains_answer_on_one_socket_under_their_own_routes() {
    let stub = Stub::start();
    let dir = tempfile::tempdir().expect("temp dir");
    let ports = Ports::allocate();
    let toml = config(dir.path(), &stub, &ports, &stub.url(), &stub.url());

    let node = Node::start(&dir.path().join("cfg.toml"), &toml);

    // The root of the socket reports the set the process resolved from its configuration, which is
    // what turns a mistake in that configuration into a failed assertion here rather than into a
    // chain that quietly never started. `rpcPath` is the whole answer to *where is chain N*: the
    // caller is already talking to the socket.
    let hosted = node.admin().request::<Value, _>("lokahi_chains", rpc_params![]).await;
    let hosted = hosted.expect("the admin rpc did not answer lokahi_chains");
    let hosted = hosted.as_array().expect("lokahi_chains returns an array").clone();
    assert_eq!(hosted.len(), 2, "expected two hosted chains: {hosted:?}");
    assert_eq!(hosted[0]["chainId"], json!(CHAIN_A));
    assert_eq!(hosted[1]["chainId"], json!(CHAIN_B));
    assert_eq!(hosted[0]["rpcPath"], json!(format!("/{CHAIN_A}")));
    assert_eq!(hosted[1]["rpcPath"], json!(format!("/{CHAIN_B}")));

    // Each chain answers for itself: the chain id in the path decides which chain replies, and
    // both paths are on the one address the process logged. That is op-supernode's addressing,
    // segment for segment, so a client is pointed at either implementation with the same URL.
    for chain_id in [CHAIN_A, CHAIN_B] {
        let name = if chain_id == CHAIN_A { CHAIN_A_NAME } else { CHAIN_B_NAME };
        let config = await_rollup_config(&node.chain_url(chain_id)).await;
        assert_eq!(
            config["l2_chain_id"],
            json!(name),
            "the route of chain {chain_id} answered for another chain: {config}"
        );
    }

    // A chain id this process does not host is a 404, as it is in Go's router — not a hang and not
    // an empty JSON-RPC error, so a caller that mistyped a chain id can tell.
    let status = get_status(&format!("{}/999999", node.base())).await;
    assert_eq!(status, 404, "an unhosted chain id must be a 404");

    // The two supernode query methods answer on the same socket. Nothing here derives, so what
    // this establishes is the wiring no unit test can reach: that both methods are registered
    // under the namespaces their consumers call, and that they answer over the whole chain set
    // rather than for one chain.
    let status = await_sync_status(&node).await;
    assert_eq!(
        status["chain_ids"],
        json!([CHAIN_A.to_string(), CHAIN_B.to_string()]),
        "supernode_syncStatus must report the chain set as decimal strings: {status}"
    );
    assert!(
        status["chains"].get(CHAIN_A.to_string()).is_some(),
        "chain {CHAIN_A} is missing its own sync status: {status}"
    );

    // The timestamp is `hexutil.Uint64`, which is what every Go consumer sends, and it is far
    // enough ahead that no chain can have derived it. So no chain contributes an optimistic
    // output and there is no super root to state — which is a successful response with `data`
    // absent, not an error.
    let superroot = node
        .admin()
        .request::<Value, _>("superroot_atTimestamp", rpc_params!["0x7fffffff"])
        .await
        .expect("the admin rpc did not answer superroot_atTimestamp");
    assert_eq!(
        superroot["optimistic_at_timestamp"],
        json!({}),
        "no chain has derived that timestamp: {superroot}"
    );
    assert!(
        superroot.get("data").is_none(),
        "an absent super root omits `data` rather than sending null: {superroot}"
    );
    assert_eq!(superroot["chain_ids"], json!([CHAIN_A.to_string(), CHAIN_B.to_string()]));
}

/// One chain's execution layer being unreachable is that chain's problem.
///
/// A supernode that exits, or that stops answering for its other chains, because one execution
/// layer is down would be strictly worse than running N single-chain nodes. Chain A's engine points
/// at a closed port here; both chains must still serve their own RPC, and the process must live.
#[tokio::test(flavor = "multi_thread")]
async fn an_unreachable_execution_layer_does_not_stop_the_other_chain() {
    let stub = Stub::start();
    let dir = tempfile::tempdir().expect("temp dir");
    let ports = Ports::allocate();
    let closed = format!("http://127.0.0.1:{}", ports.closed);
    let toml = config(dir.path(), &stub, &ports, &closed, &stub.url());

    let mut node = Node::start(&dir.path().join("cfg.toml"), &toml);

    for (chain_id, name) in [(CHAIN_A, CHAIN_A_NAME), (CHAIN_B, CHAIN_B_NAME)] {
        let config = await_rollup_config(&node.chain_url(chain_id)).await;
        assert_eq!(config["l2_chain_id"], json!(name));
    }

    assert!(node.is_running(), "the supernode exited when one execution layer was unreachable");
}

/// Renders a two-chain configuration file.
///
/// The one socket asks for port 0 and no chain names a port at all: a chain is a route on that
/// socket, so the address the process logs is the address of every chain it hosts.
fn config(dir: &Path, stub: &Stub, ports: &Ports, engine_a: &str, engine_b: &str) -> String {
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
rpc-addr = "127.0.0.1"
rpc-enable-admin = true
p2p-listen-ip = "127.0.0.1"

[admin]
rpc-port = 0

[[chains]]
l2-chain-id = {CHAIN_A}
engine-rpc = "{engine_a}"
p2p-tcp-port = {tcp_a}
p2p-udp-port = {udp_a}
unsafe-block-signer = "0x1111111111111111111111111111111111111111"

[[chains]]
l2-chain-id = {CHAIN_B}
engine-rpc = "{engine_b}"
p2p-tcp-port = {tcp_b}
p2p-udp-port = {udp_b}
unsafe-block-signer = "0x2222222222222222222222222222222222222222"
"#,
        stub = stub.url(),
        datadir = dir.join("data").display(),
        jwt = jwt.display(),
        tcp_a = ports.tcp_a,
        tcp_b = ports.tcp_b,
        udp_a = ports.udp_a,
        udp_b = ports.udp_b,
    )
}

/// The ports one run needs, taken from the OS so concurrent test binaries do not collide.
///
/// P2P only: the RPC is one socket on port 0, whose address the process logs. All of them are
/// reserved at once and only then released: allocating them one at a time would hand out the same
/// port twice, because each is free again before the next is asked for.
struct Ports {
    tcp_a: u16,
    tcp_b: u16,
    udp_a: u16,
    udp_b: u16,
    /// A port nothing listens on, for the chain whose execution layer must be unreachable.
    closed: u16,
}

impl Ports {
    fn allocate() -> Self {
        let reserved: Vec<TcpListener> = (0..5)
            .map(|_| TcpListener::bind("127.0.0.1:0").expect("reserve an ephemeral port"))
            .collect();
        let ports: Vec<u16> =
            reserved.iter().map(|l| l.local_addr().expect("local addr").port()).collect();

        Self {
            tcp_a: ports[0],
            tcp_b: ports[1],
            udp_a: ports[2],
            udp_b: ports[3],
            closed: ports[4],
        }
    }
}

/// The L1, the beacon API and both engine APIs, on one socket.
///
/// None of them has to answer usefully for the chains to compose and serve RPC, with one exception:
/// kona panics at startup if it cannot read the genesis time and the slot interval from the beacon
/// API, so those two responses are real. Everything else answers `null`, which the actors read as a
/// chain that is not moving.
struct Stub {
    addr: SocketAddr,
}

impl Stub {
    /// The URL of the stub, for whichever of its roles a chain is being pointed at.
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

    /// The two beacon endpoints kona reads before it will compose a chain.
    fn beacon(request: &str) -> Value {
        if request.contains("/eth/v1/beacon/genesis") {
            json!({"data": {"genesis_time": "0"}})
        } else if request.contains("/eth/v1/config/spec") {
            json!({"data": {"SECONDS_PER_SLOT": "12"}})
        } else {
            json!({"data": {}})
        }
    }

    /// A null result for every call, as a batch when the request was one.
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
struct Node {
    child: Child,
    /// The one address the process serves everything on.
    base: String,
    admin: HttpClient,
}

impl Node {
    /// Writes `config` to `path`, starts the binary over it, and waits for the admin RPC's address.
    ///
    /// Waiting on the logged address rather than on a port the caller chose is the handshake an
    /// out-of-process launch needs, and it is the one the devstack component performs.
    fn start(path: &Path, config: &str) -> Self {
        std::fs::write(path, config).expect("write the config");

        let mut child = Command::new(env!("CARGO_BIN_EXE_lokahi"))
            .args(["node", "--config", path.to_str().expect("utf-8 path")])
            .env("KONA_LOG_STDOUT_FORMAT", "json")
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()
            .expect("start lokahi");

        let stdout = child.stdout.take().expect("piped stdout");
        let (tx, rx) = mpsc::channel();
        thread::spawn(move || {
            // Every line is read, not only the one being waited for: a full pipe buffer would
            // otherwise block the process under test. They are echoed so that a failure here comes
            // with the node's own account of what it did.
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

    /// The one address the process serves everything on.
    fn base(&self) -> &str {
        &self.base
    }

    /// Where chain `chain_id` answers: the supernode's address and the chain id as a path segment,
    /// which is `op-supernode`'s addressing too (`multichain_supernode_runtime.go`).
    fn chain_url(&self, chain_id: u64) -> String {
        format!("{}/{chain_id}", self.base)
    }

    fn is_running(&mut self) -> bool {
        matches!(self.child.try_wait(), Ok(None))
    }
}

impl Drop for Node {
    fn drop(&mut self) {
        // Killed on the way out, including when the test panics: a leaked supernode would keep the
        // ports it was given and outlive the run.
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

/// Polls one chain's own RPC until it answers `optimism_rollupConfig`.
/// Polls `supernode_syncStatus` until it answers.
///
/// The process-wide RPC is bound before the chains are composed, so that the address it logs is
/// available to a caller that has to wait for something. A call that arrives in that window is
/// refused with "the supernode is still starting" rather than answered for an empty chain set, so
/// this retries. A namespace that is not registered at all fails here instead, with the method
/// name in the error.
async fn await_sync_status(node: &Node) -> Value {
    let deadline = Instant::now() + READY_TIMEOUT;

    loop {
        match node.admin().request::<Value, _>("supernode_syncStatus", rpc_params![]).await {
            Ok(status) => return status,
            Err(err) => {
                assert!(
                    Instant::now() < deadline,
                    "the admin rpc never answered supernode_syncStatus: {err}"
                );
                sleep(Duration::from_millis(200)).await;
            }
        }
    }
}

async fn await_rollup_config(url: &str) -> Value {
    let client = HttpClientBuilder::default().build(url).expect("build a chain rpc client");
    let deadline = Instant::now() + READY_TIMEOUT;

    loop {
        match client.request::<Value, _>("optimism_rollupConfig", rpc_params![]).await {
            Ok(config) => return config,
            Err(err) => {
                assert!(Instant::now() < deadline, "the route {url} never answered: {err}");
                sleep(Duration::from_millis(200)).await;
            }
        }
    }
}

/// The status code of a bare POST to `url`, over a hand-rolled request so that the router's answer
/// is read as HTTP rather than through a JSON-RPC client that would hide it.
async fn get_status(url: &str) -> u16 {
    let url = url.strip_prefix("http://").expect("an http url");
    let (authority, path) = url.split_once('/').expect("a path to ask for");
    let mut stream = TcpStream::connect(authority).expect("connect to the supernode");
    let body = br#"{"jsonrpc":"2.0","id":1,"method":"optimism_rollupConfig","params":[]}"#;
    let request = format!(
        "POST /{path} HTTP/1.1\r\nhost: {authority}\r\ncontent-type: application/json\r\n\
         content-length: {}\r\nconnection: close\r\n\r\n",
        body.len()
    );
    stream.write_all(request.as_bytes()).expect("write the request head");
    stream.write_all(body).expect("write the request body");

    let mut status = String::new();
    BufReader::new(stream).read_line(&mut status).expect("read the status line");
    status
        .split_whitespace()
        .nth(1)
        .and_then(|code| code.parse().ok())
        .unwrap_or_else(|| panic!("unexpected status line: {status}"))
}
