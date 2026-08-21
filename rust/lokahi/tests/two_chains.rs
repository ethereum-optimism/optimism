//! Two chains, one `lokahi` process.
//!
//! This is the smoke test for the thing a supernode is for, run against the real binary: one
//! process is given two chains, and each of them ends up answering on its own RPC socket with its
//! own rollup config, while the process-wide admin RPC reports the set it was asked to host.
//!
//! Deriving blocks needs a live L1 and a live execution layer per chain, which is what the
//! devstack's lokahi component exists to provide (`op-devstack/sysgo/l2_cl_lokahi.go`). What the
//! stubbed L1 here establishes is the part that needs no chain to be moving and that no unit test
//! can reach: that two chains compose in one process without colliding, that a request lands on
//! the chain whose port it was sent to, and that one chain's execution layer being unreachable
//! neither stops the other chain nor takes the process down.

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
async fn two_chains_answer_on_their_own_rpc_sockets() {
    let stub = Stub::start();
    let dir = tempfile::tempdir().expect("temp dir");
    let ports = Ports::allocate();
    let toml = config(dir.path(), &stub, &ports, &stub.url(), &stub.url());

    let node = Node::start(&dir.path().join("cfg.toml"), &toml);

    // The admin RPC reports the set the process resolved from its configuration, which is what
    // turns a mistake in that configuration into a failed assertion here rather than into a chain
    // that quietly never started.
    let hosted = node.admin().request::<Value, _>("lokahi_chains", rpc_params![]).await;
    let hosted = hosted.expect("the admin rpc did not answer lokahi_chains");
    let hosted = hosted.as_array().expect("lokahi_chains returns an array").clone();
    assert_eq!(hosted.len(), 2, "expected two hosted chains: {hosted:?}");
    assert_eq!(hosted[0]["chainId"], json!(CHAIN_A));
    assert_eq!(hosted[1]["chainId"], json!(CHAIN_B));
    assert_eq!(hosted[0]["rpcAddr"], json!(format!("127.0.0.1:{}", ports.rpc_a)));
    assert_eq!(hosted[1]["rpcAddr"], json!(format!("127.0.0.1:{}", ports.rpc_b)));

    // Each chain answers for itself: the port a request goes to decides which chain replies, which
    // is the whole addressing scheme of a lokahi supernode.
    for (port, chain) in [(ports.rpc_a, CHAIN_A_NAME), (ports.rpc_b, CHAIN_B_NAME)] {
        let config = await_rollup_config(port).await;
        assert_eq!(
            config["l2_chain_id"],
            json!(chain),
            "the server on port {port} answered for another chain: {config}"
        );
    }
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

    for (port, chain) in [(ports.rpc_a, CHAIN_A_NAME), (ports.rpc_b, CHAIN_B_NAME)] {
        let config = await_rollup_config(port).await;
        assert_eq!(config["l2_chain_id"], json!(chain));
    }

    assert!(node.is_running(), "the supernode exited when one execution layer was unreachable");
}

/// Renders a two-chain configuration file.
///
/// The admin RPC asks for port 0 and the chains name concrete ports: kona binds each chain's server
/// itself and never reports the address back, so a chain on port 0 would be unaddressable, whereas
/// the admin RPC logs the port it was given.
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
rpc-port = {rpc_a}
p2p-tcp-port = {tcp_a}
p2p-udp-port = {udp_a}
unsafe-block-signer = "0x1111111111111111111111111111111111111111"

[[chains]]
l2-chain-id = {CHAIN_B}
engine-rpc = "{engine_b}"
rpc-port = {rpc_b}
p2p-tcp-port = {tcp_b}
p2p-udp-port = {udp_b}
unsafe-block-signer = "0x2222222222222222222222222222222222222222"
"#,
        stub = stub.url(),
        datadir = dir.join("data").display(),
        jwt = jwt.display(),
        rpc_a = ports.rpc_a,
        rpc_b = ports.rpc_b,
        tcp_a = ports.tcp_a,
        tcp_b = ports.tcp_b,
        udp_a = ports.udp_a,
        udp_b = ports.udp_b,
    )
}

/// The ports one run needs, taken from the OS so concurrent test binaries do not collide.
///
/// All of them are reserved at once and only then released: allocating them one at a time would
/// hand out the same port twice, because each is free again before the next is asked for.
struct Ports {
    rpc_a: u16,
    rpc_b: u16,
    tcp_a: u16,
    tcp_b: u16,
    udp_a: u16,
    udp_b: u16,
    /// A port nothing listens on, for the chain whose execution layer must be unreachable.
    closed: u16,
}

impl Ports {
    fn allocate() -> Self {
        let reserved: Vec<TcpListener> = (0..7)
            .map(|_| TcpListener::bind("127.0.0.1:0").expect("reserve an ephemeral port"))
            .collect();
        let ports: Vec<u16> =
            reserved.iter().map(|l| l.local_addr().expect("local addr").port()).collect();

        Self {
            rpc_a: ports[0],
            rpc_b: ports[1],
            tcp_a: ports[2],
            tcp_b: ports[3],
            udp_a: ports[4],
            udp_b: ports[5],
            closed: ports[6],
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

        let admin =
            rx.recv_timeout(READY_TIMEOUT).expect("lokahi did not log its admin RPC address");
        let admin = HttpClientBuilder::default()
            .build(format!("http://{admin}"))
            .expect("build the admin rpc client");

        Self { child, admin }
    }

    const fn admin(&self) -> &HttpClient {
        &self.admin
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
async fn await_rollup_config(port: u16) -> Value {
    let client = HttpClientBuilder::default()
        .build(format!("http://127.0.0.1:{port}"))
        .expect("build a chain rpc client");
    let deadline = Instant::now() + READY_TIMEOUT;

    loop {
        match client.request::<Value, _>("optimism_rollupConfig", rpc_params![]).await {
            Ok(config) => return config,
            Err(err) => {
                assert!(
                    Instant::now() < deadline,
                    "the rpc server on port {port} never answered: {err}"
                );
                sleep(Duration::from_millis(200)).await;
            }
        }
    }
}
