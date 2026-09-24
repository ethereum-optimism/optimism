//! Operator-side witness collection and proving for the native-check protocol v2
//! (spec-sound-profile §G.2). None of this I/O runs inside derivation or the guest.
//!
//! The request carries only span-bounded data (the public span, private data, configs and the
//! dependency set). Recovery blocks and every witness are fetched here over RPC, with bounded
//! concurrency, so the stdin cap does not depend on how long an outage lasted (§G.3a).
use alloy_consensus::{Header, Transaction};
use alloy_eips::{
    BlockNumHash, Encodable2718, eip2935::HISTORY_STORAGE_ADDRESS, eip4788::BEACON_ROOTS_ADDRESS,
};
use alloy_primitives::{Address, B256, Bytes, U256, hex, keccak256};
use alloy_rlp::Decodable;
use anyhow::{Context, Result, anyhow, bail, ensure};
use jsonrpsee::ws_client::{WsClient, WsClientBuilder};
use jsonrpsee_core::{client::ClientT, rpc_params, traits::ToRpcParams};
use jsonrpsee_http_client::{HttpClient, HttpClientBuilder};
use kona_genesis::RollupConfig;
use kona_protocol::{
    L1BlockInfoTx, SpanBatch, SpanBatchElement,
    projection::{
        self, EXECUTION_MOCK, Envelope, EnvelopeKind, PUBLIC_VALUES_LEN, SP1_PRIVATE_PROJECTION_V1,
    },
};
use kona_sp1_client_utils::private_projection::{
    ProjectionBlock, PublicValues, RelationInput, Witness, execute, execution_mock_digest,
    ordered_list_preimages, public_value_word,
};
use op_alloy_consensus::OpBlock;
use serde::{Deserialize, de::DeserializeOwned};
use sp1_sdk::{
    Elf, HashableKey, LightProver, ProveRequest, Prover, ProverClient, ProvingKey, SP1Proof,
    SP1ProofWithPublicValues, SP1Stdin,
};
use std::{
    collections::{BTreeMap, BTreeSet},
    future::Future,
    io::Read,
    path::Path,
    sync::Arc,
    time::{Duration, Instant},
};
use tokio::{sync::Semaphore, task::JoinSet};

/// Stdin cap. The request is span-bounded (§G.3a item 1), so this no longer grows with outages.
const MAX_REQUEST: usize = 128 * 1024 * 1024;
/// Concurrent witness requests (§G.3a item 3).
const FETCH_CONCURRENCY: usize = 8;
const MESSAGE_PASSER: &str = "0x4200000000000000000000000000000000000016";
/// Ring-buffer length of the EIP-4788 beacon-roots and EIP-2935 history system contracts.
const SYSTEM_HISTORY: u64 = 8191;
/// How long to wait for the private chain (`LightCL`) to adopt the canonical deposit-only recovery
/// blocks before failing. The batcher's proof timeout bounds the whole run.
const RECOVERY_ADOPTION_WAIT: Duration = Duration::from_secs(60);
const RECOVERY_ADOPTION_POLL: Duration = Duration::from_secs(1);

/// Prover selection of the v2 request (§G.2 table).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub(crate) enum ProverKind {
    /// Native relation only; legacy `execution-mock-v1` envelope.
    Native,
    /// Native relation; SP1 mock envelope built without the ELF.
    NativeMock,
    /// SP1 mock prover over the ELF.
    Mock,
    /// Local CPU Groth16 proving.
    Cpu,
    /// Succinct prover network Groth16 proving.
    Network,
}

#[derive(Deserialize)]
struct VersionProbe {
    version: Option<u64>,
}

/// The v2 publication request. Every field is span-bounded.
#[derive(Debug, Deserialize)]
pub(crate) struct Request {
    version: u64,
    prover: ProverKind,
    private_rpc: String,
    projection_rpc: String,
    l1_rpc: String,
    private_config: Bytes,
    l1_config: Bytes,
    projection_config: Bytes,
    dependency_set: Bytes,
    parent_hash: B256,
    anchor: BlockNumHash,
    anchor_output: B256,
    recovery_hash: B256,
    blocks: Vec<ProjectionBlock>,
    private_data: Bytes,
    l1_head: B256,
    expected_public_values: Bytes,
    #[serde(default)]
    expected_digest: Option<B256>,
}

pub(crate) fn parse_request(encoded: &[u8]) -> Result<Request> {
    let probe: VersionProbe = serde_json::from_slice(encoded).context("publication request")?;
    ensure!(
        probe.version == Some(2),
        "unsupported publication request version {:?}",
        probe.version
    );
    let r: Request = serde_json::from_slice(encoded).context("publication request v2")?;
    ensure!(r.version == 2, "unsupported publication request version");
    ensure!(
        r.expected_public_values.len() == PUBLIC_VALUES_LEN,
        "expected_public_values must be {PUBLIC_VALUES_LEN} bytes"
    );
    Ok(r)
}

enum Rpc {
    Http(HttpClient),
    Ws(WsClient),
}
impl Rpc {
    async fn connect(url: &str) -> Result<Arc<Self>> {
        Ok(Arc::new(if url.starts_with("ws://") || url.starts_with("wss://") {
            Self::Ws(
                WsClientBuilder::default()
                    .request_timeout(Duration::from_secs(60))
                    .max_response_size(100 * 1024 * 1024)
                    .max_concurrent_requests(4 * FETCH_CONCURRENCY)
                    .build(url)
                    .await?,
            )
        } else {
            Self::Http(
                HttpClientBuilder::default()
                    .request_timeout(Duration::from_secs(60))
                    .max_response_size(100 * 1024 * 1024)
                    .build(url)?,
            )
        }))
    }
    async fn request<R: DeserializeOwned, P: ToRpcParams + Send>(
        &self,
        method: &str,
        params: P,
    ) -> Result<R> {
        let out = match self {
            Self::Http(client) => client.request(method, params).await,
            Self::Ws(client) => client.request(method, params).await,
        };
        out.with_context(|| method.to_string())
    }
}

async fn block(client: &Rpc, number: u64) -> Result<OpBlock> {
    let value: alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction> =
        client.request("eth_getBlockByNumber", rpc_params![format!("{number:#x}"), true]).await?;
    Ok(value.into_consensus().map_transactions(|tx| tx.inner.inner.into_inner()))
}

async fn header_by_number(client: &Rpc, number: u64) -> Result<Header> {
    let value: alloy_rpc_types_eth::Block =
        client.request("eth_getBlockByNumber", rpc_params![format!("{number:#x}"), false]).await?;
    Ok(value.header.inner)
}

/// An L1 header by hash, re-encoded and checked against its hash.
async fn l1_header(client: &Rpc, hash: B256) -> Result<(Header, Bytes)> {
    let value: Option<alloy_rpc_types_eth::Block> =
        client.request("eth_getBlockByHash", rpc_params![hash, false]).await?;
    let header = value.ok_or_else(|| anyhow!("L1 block {hash} not found"))?.header.inner;
    let raw: Bytes = alloy_rlp::encode(&header).into();
    ensure!(keccak256(&raw) == hash, "L1 header {hash} does not re-encode to its hash");
    Ok((header, raw))
}

#[derive(Deserialize)]
struct ExecutionWitness {
    state: Vec<Bytes>,
    codes: Vec<Bytes>,
    #[serde(default)]
    headers: Vec<Bytes>,
}

/// Run `f` over `items` with at most [`FETCH_CONCURRENCY`] in flight. Results reach `sink` as
/// they arrive; the first error aborts the rest.
async fn fetch_all<K, T, F, Fut>(items: Vec<K>, f: F, mut sink: impl FnMut(K, T)) -> Result<()>
where
    K: Send + 'static,
    T: Send + 'static,
    F: Fn(K) -> Fut,
    Fut: Future<Output = (K, Result<T>)> + Send + 'static,
{
    let permits = Arc::new(Semaphore::new(FETCH_CONCURRENCY));
    let mut set = JoinSet::new();
    for item in items {
        let permit = Arc::clone(&permits).acquire_owned().await?;
        let fut = f(item);
        set.spawn(async move {
            let out = fut.await;
            drop(permit);
            out
        });
        while let Some(done) = set.try_join_next() {
            let (k, r) = done?;
            sink(k, r?);
        }
    }
    while let Some(done) = set.join_next().await {
        let (k, r) = done?;
        sink(k, r?);
    }
    Ok(())
}

struct PrivateBlockWitness {
    hash: B256,
    parent_hash: B256,
    preimages: Vec<Bytes>,
}

/// Parent-state proofs of the accounts and slots a block touches even when it carries no user
/// transaction: the message passer (read by the output root), the EIP-4788 beacon-roots slots and
/// the EIP-2935 history slot written by the pre-execution system calls. They duplicate what
/// `debug_executionWitness` should report, so the witness does not depend on it doing so.
fn system_proofs(header: &Header) -> [(Address, Vec<B256>); 3] {
    let slot = |n: u64| B256::from(U256::from(n));
    let ts = header.timestamp % SYSTEM_HISTORY;
    [
        (MESSAGE_PASSER.parse().expect("constant address"), vec![]),
        (BEACON_ROOTS_ADDRESS, vec![slot(ts), slot(ts + SYSTEM_HISTORY)]),
        (HISTORY_STORAGE_ADDRESS, vec![slot((header.number - 1) % SYSTEM_HISTORY)]),
    ]
}

async fn private_block_witness(private: Arc<Rpc>, number: u64) -> Result<PrivateBlockWitness> {
    let deadline = Instant::now() + RECOVERY_ADOPTION_WAIT;
    loop {
        let header = header_by_number(&private, number).await?;
        let w: ExecutionWitness = match private
            .request("debug_executionWitness", rpc_params![format!("{number:#x}")])
            .await
        {
            Ok(w) => w,
            // A freshly built or reorged head can briefly have no state ("no state found for
            // block number"); retry within the same budget rather than fail the proof.
            Err(e) if Instant::now() < deadline => {
                eprintln!("witness of private block {number} unavailable ({e:#}); retrying");
                tokio::time::sleep(RECOVERY_ADOPTION_POLL).await;
                continue;
            }
            Err(e) => return Err(e),
        };
        let parent_state = w.headers.iter().find_map(|raw| {
            let parent = Header::decode(&mut raw.as_ref()).ok()?;
            (parent.hash_slow() == header.parent_hash).then_some(parent.state_root)
        });
        // The execution witness itself must be over the parent state (checked before the proofs
        // below add their own copy of the root node).
        let over_parent = witness_describes_parent(parent_state, &w.state, &[]);
        let mut preimages: Vec<Bytes> = w.state;
        preimages.extend(w.codes);
        preimages.extend(w.headers);
        let mut proof_roots = Vec::new();
        for (address, slots) in system_proofs(&header) {
            // EIP-1898: pin the proofs to the parent by hash, not by number.
            let at = serde_json::json!({ "blockHash": header.parent_hash });
            let proof: alloy_rpc_types_eth::EIP1186AccountProofResponse =
                private.request("eth_getProof", rpc_params![address, slots, at]).await?;
            proof_roots.extend(proof.account_proof.first().map(keccak256));
            preimages.extend(proof.account_proof);
            preimages.extend(proof.storage_proof.into_iter().flat_map(|p| p.proof));
        }
        match over_parent
            .and_then(|()| witness_describes_parent(parent_state, &preimages, &proof_roots))
        {
            Ok(()) => {
                return Ok(PrivateBlockWitness {
                    hash: header.hash_slow(),
                    parent_hash: header.parent_hash,
                    preimages,
                });
            }
            Err(why) => {
                ensure!(
                    Instant::now() < deadline,
                    "private node keeps serving a witness of another state for block {number} \
                     ({why}) after {RECOVERY_ADOPTION_WAIT:?}"
                );
                eprintln!(
                    "witness of private block {number} is not over its parent ({why}); retrying"
                );
                tokio::time::sleep(RECOVERY_ADOPTION_POLL).await;
            }
        }
    }
}

/// Whether a block's witness is over its parent's state: it carries the parent header, the node of
/// the parent state root, and account proofs rooted there. Right after a private reorg (`LightCL`
/// replacing a span with deposit-only blocks) a node can serve witnesses built on the other
/// branch's persisted state, which would leave the relation without the nodes it reads, most
/// visibly in the first system call of the next block.
pub(crate) fn witness_describes_parent(
    parent_state: Option<B256>,
    preimages: &[Bytes],
    proof_roots: &[B256],
) -> Result<(), &'static str> {
    let root = parent_state.ok_or("no parent header")?;
    if !preimages.iter().any(|node| keccak256(node) == root) {
        return Err("no parent state root node");
    }
    if proof_roots.iter().any(|r| *r != root) {
        return Err("account proof of another state");
    }
    Ok(())
}

/// Whether `private` is the private counterpart of the canonical deposit-only recovery block
/// `public`: same height, time and L1 origin, and the same forced inputs (and trailing post-exec).
/// Only such a block's witness describes the state the relation executes; `LightCL` builds it after
/// the projection replaces a span, and until then the private chain still holds the block its
/// sequencer produced.
pub(crate) fn corresponds(private: &OpBlock, public: &OpBlock) -> bool {
    let (p, q) = (encoded_txs(private), encoded_txs(public));
    private.header.number == public.header.number &&
        private.header.timestamp == public.header.timestamp &&
        p.len() == q.len() &&
        !p.is_empty() &&
        p[1..] == q[1..] &&
        matches!((l1_origin(private), l1_origin(public)), (Ok(a), Ok(b)) if a == b)
}

/// Wait until the private chain holds the private counterpart of every recovery block, and return
/// those private blocks.
async fn await_recovery_adoption(
    private: &Arc<Rpc>,
    recovery: &BTreeMap<u64, OpBlock>,
) -> Result<BTreeMap<u64, OpBlock>> {
    let deadline = Instant::now() + RECOVERY_ADOPTION_WAIT;
    let mut adopted = BTreeMap::new();
    let Some(mut from) = recovery.keys().next().copied() else { return Ok(adopted) };
    loop {
        let pending: Vec<u64> = recovery.range(from..).map(|(n, _)| *n).collect();
        let mut stale = BTreeSet::new();
        fetch_all(
            pending,
            |n| {
                let private = Arc::clone(private);
                async move { (n, block(&private, n).await) }
            },
            |n, b: OpBlock| {
                if corresponds(&b, &recovery[&n]) {
                    adopted.insert(n, b);
                } else {
                    stale.insert(n);
                }
            },
        )
        .await?;
        let Some(first_stale) = stale.first().copied() else { return Ok(adopted) };
        ensure!(
            Instant::now() < deadline,
            "private chain has not adopted canonical recovery block {first_stale} within \
             {RECOVERY_ADOPTION_WAIT:?} ({} of {} recovery blocks pending); retry once LightCL \
             has replaced them",
            stale.len(),
            recovery.len()
        );
        eprintln!(
            "waiting for the private chain to adopt recovery blocks {first_stale}..: {} pending",
            stale.len()
        );
        from = first_stale;
        tokio::time::sleep(RECOVERY_ADOPTION_POLL).await;
    }
}

fn encoded_txs(block: &OpBlock) -> Vec<Bytes> {
    block.body.transactions.iter().map(|tx| tx.encoded_2718().into()).collect()
}

fn l1_origin(block: &OpBlock) -> Result<BlockNumHash> {
    let tx = block.body.transactions.first().ok_or_else(|| anyhow!("missing L1 info"))?;
    Ok(L1BlockInfoTx::decode_calldata(tx.input())?.id())
}

/// Collect the relation's input and witness for a request (§D.2 host), with the private chain's
/// counterparts of the recovery blocks (diagnostics only).
pub(crate) async fn collect(r: &Request) -> Result<(RelationInput, Witness, Vec<OpBlock>)> {
    let started = Instant::now();
    let private_cfg: RollupConfig = serde_json::from_slice(&r.private_config)?;
    let projection_cfg: RollupConfig = serde_json::from_slice(&r.projection_config)?;
    let span = SpanBatch {
        batches: r
            .blocks
            .iter()
            .map(|b| SpanBatchElement {
                timestamp: b.timestamp,
                epoch_num: b.epoch,
                transactions: vec![],
            })
            .collect(),
        ..Default::default()
    };
    let (first, last) = projection::range_bounds(&projection_cfg, &span)?;
    ensure!(r.anchor.number < first, "invalid anchor");
    let private = Rpc::connect(&r.private_rpc).await?;
    let public = Rpc::connect(&r.projection_rpc).await?;
    let l1 = Rpc::connect(&r.l1_rpc).await?;

    let anchor = block(&private, r.anchor.number).await?;
    let anchor_hash = anchor.header.hash_slow();
    let anchor_transactions = encoded_txs(&anchor);
    let anchor_epoch = if r.anchor.number == private_cfg.genesis.l2.number {
        private_cfg.genesis.l1.number
    } else {
        l1_origin(&anchor)?.number
    };

    // Canonical recovery blocks from the projection, checked against the preflight statement.
    let mut recovery = BTreeMap::new();
    fetch_all(
        (r.anchor.number + 1..first).collect(),
        |n| {
            let public = Arc::clone(&public);
            async move { (n, block(&public, n).await) }
        },
        |n, b: OpBlock| {
            recovery.insert(n, b);
        },
    )
    .await?;
    let recomputed = recovery
        .values()
        .rev()
        .fold(B256::ZERO, |h, b| projection::recovery_step(h, b, &encoded_txs(b)));
    ensure!(recomputed == r.recovery_hash, "canonical recovery blocks differ from recovery_hash");
    // Witnesses of private blocks that are not the recovery counterparts describe another state.
    let adopted = await_recovery_adoption(&private, &recovery).await?;

    // Private witnesses, FETCH_CONCURRENCY requests in flight.
    let mut preimages = BTreeMap::new();
    let mut chain = BTreeMap::new();
    let numbers: Vec<u64> = (r.anchor.number + 1..=last).collect();
    fetch_all(
        numbers.clone(),
        |n| {
            let private = Arc::clone(&private);
            async move { (n, private_block_witness(private, n).await) }
        },
        |n, w: PrivateBlockWitness| {
            for raw in w.preimages {
                preimages.insert(keccak256(&raw), raw);
            }
            chain.insert(n, (w.hash, w.parent_hash));
        },
    )
    .await?;
    let mut parent = anchor_hash;
    for (n, (hash, parent_hash)) in &chain {
        ensure!(
            *parent_hash == parent && adopted.get(n).is_none_or(|a| a.header.hash_slow() == *hash),
            "private chain changed during witness collection"
        );
        parent = *hash;
    }
    let recovery: Vec<OpBlock> = recovery.into_values().collect();

    // L1: headers from l1_head down to the smallest new-block epoch, recovery epoch headers, and
    // the receipts of every epoch that some block of the range opens.
    let min_epoch = r.blocks.iter().map(|b| b.epoch).min().expect("range_bounds: nonempty");
    let mut epochs = BTreeMap::new();
    let mut hash = r.l1_head;
    loop {
        let (header, raw) = l1_header(&l1, hash).await?;
        preimages.insert(hash, raw);
        epochs.insert(header.number, (hash, header.receipts_root));
        if header.number <= min_epoch {
            break;
        }
        hash = header.parent_hash;
    }
    let mut block_epochs = Vec::with_capacity(recovery.len() + r.blocks.len());
    for b in &recovery {
        block_epochs.push(l1_origin(b)?);
    }
    for b in &r.blocks {
        let (hash, _) = epochs
            .get(&b.epoch)
            .ok_or_else(|| anyhow!("L1 epoch {} not under l1_head", b.epoch))?;
        block_epochs.push(BlockNumHash { number: b.epoch, hash: *hash });
    }
    let missing: Vec<B256> = block_epochs
        .iter()
        .filter(|e| !epochs.contains_key(&e.number))
        .map(|e| e.hash)
        .collect::<BTreeSet<_>>()
        .into_iter()
        .collect();
    fetch_all(
        missing,
        |h| {
            let l1 = Arc::clone(&l1);
            async move { (h, l1_header(&l1, h).await) }
        },
        |h, (header, raw): (Header, Bytes)| {
            preimages.insert(h, raw);
            epochs.insert(header.number, (h, header.receipts_root));
        },
    )
    .await?;
    let mut opened = BTreeSet::new();
    let mut previous = anchor_epoch;
    for e in &block_epochs {
        if e.number != previous {
            opened.insert(e.number);
        }
        previous = e.number;
    }
    let receipts: Vec<(B256, B256)> = opened
        .into_iter()
        .map(|n| epochs.get(&n).copied().ok_or_else(|| anyhow!("L1 epoch {n} header")))
        .collect::<Result<_>>()?;
    let mut mismatched = Vec::new();
    fetch_all(
        receipts,
        |(h, root)| {
            let l1 = Arc::clone(&l1);
            async move {
                let out = l1.request::<Vec<Bytes>, _>("debug_getRawReceipts", rpc_params![h]).await;
                ((h, root), out)
            }
        },
        |(h, root), raw: Vec<Bytes>| {
            let (computed, nodes) = ordered_list_preimages(&raw);
            if computed == root {
                for node in nodes {
                    preimages.insert(keccak256(&node), node);
                }
            } else {
                mismatched.push(h);
            }
        },
    )
    .await?;
    ensure!(mismatched.is_empty(), "L1 receipts do not match their receipts roots: {mismatched:?}");

    // Number-based debug RPC must not silently mix witnesses from two branches.
    let mut changed = false;
    fetch_all(
        numbers,
        |n| {
            let private = Arc::clone(&private);
            async move { (n, header_by_number(&private, n).await) }
        },
        |n, header: Header| changed |= header.hash_slow() != chain[&n].0,
    )
    .await?;
    ensure!(!changed, "private witness reorged");
    eprintln!(
        "private projection witness: {} blocks ({} recovery), {} preimages in {:.1?}",
        last - r.anchor.number,
        recovery.len(),
        preimages.len(),
        started.elapsed()
    );
    Ok((
        RelationInput {
            private_config: r.private_config.clone(),
            l1_config: r.l1_config.clone(),
            projection_config: r.projection_config.clone(),
            dependency_set: r.dependency_set.clone(),
            parent_hash: r.parent_hash,
            anchor: r.anchor,
            anchor_output: r.anchor_output,
            recovery,
            blocks: r.blocks.clone(),
            l1_head: r.l1_head,
        },
        Witness {
            anchor_header: anchor.header,
            anchor_transactions,
            private_data: r.private_data.clone(),
            preimages,
        },
        adopted.into_values().collect(),
    ))
}

/// On a relation failure, print how the private recovery blocks relate to the canonical ones, and
/// write the relation's input to `$KONA_SP1_PRIVATE_PROJECTION_DUMP_DIR` if set. Stderr only;
/// never on stdout, which carries the envelope.
fn diagnose(input: &RelationInput, witness: &Witness, private_recovery: &[OpBlock]) {
    for (public, private) in input.recovery.iter().zip(private_recovery) {
        let (p, q) = (&private.header, &public.header);
        eprintln!(
            "recovery block {}: private {} state {} beacon {:?}/{:?} randao {}/{} tx0 {}",
            p.number,
            p.hash_slow(),
            p.state_root,
            p.parent_beacon_block_root,
            q.parent_beacon_block_root,
            p.mix_hash == q.mix_hash,
            p.beneficiary == q.beneficiary,
            if encoded_txs(private).first() == encoded_txs(public).first() {
                "same"
            } else {
                "differs"
            },
        );
    }
    if let Some(dir) = std::env::var_os("KONA_SP1_PRIVATE_PROJECTION_DUMP_DIR") {
        let path = Path::new(&dir).join(format!("failed-{}.json", input.anchor.number));
        let dump = serde_json::json!({
            "input": input, "witness": witness, "private_recovery": private_recovery,
        });
        match std::fs::write(&path, dump.to_string()) {
            Ok(()) => eprintln!("relation input written to {}", path.display()),
            Err(e) => eprintln!("could not write {}: {e}", path.display()),
        }
    }
}

/// Compare native public values with the preflight statement's. Under `execution-mock-v1` the
/// config pins `private_config_hash = 0` while the relation emits the real hash, so word 3 is
/// excluded there (§G.2 step 3).
pub(crate) fn check_expected(pv: &PublicValues, expected: &[u8], verifier: &str) -> Result<()> {
    ensure!(expected.len() == PUBLIC_VALUES_LEN, "expected public values length");
    for word in 0..PUBLIC_VALUES_LEN / 32 {
        if word == 3 && verifier == EXECUTION_MOCK {
            continue;
        }
        let got = public_value_word(pv, word);
        let want = B256::from_slice(&expected[word * 32..word * 32 + 32]);
        ensure!(got == want, "native public values differ at word {word}: {got} != {want}");
    }
    Ok(())
}

/// The bytes32 program vkey (`program_vkey`) of an ELF.
pub(crate) async fn program_vkey(elf: &Path) -> Result<B256> {
    let bytes = std::fs::read(elf).with_context(|| format!("read {}", elf.display()))?;
    let pk = LightProver::new()
        .await
        .setup(Elf::Dynamic(Arc::from(bytes)))
        .await
        .map_err(|e| anyhow!("ELF setup: {e}"))?;
    Ok(B256::from(pk.verifying_key().bytes32_raw()))
}

/// One SP1 public input string: decimal (`BigUint`), or a field element's debug form carrying
/// `0x…` hex (the mock prover's vkey hash prints as `FFBn254Fr(0x…)`).
fn public_input_word(input: &str) -> Result<[u8; 32]> {
    let value = input.find("0x").map_or_else(
        || U256::from_str_radix(input, 10),
        |i| {
            let hex: String = input[i + 2..].chars().take_while(char::is_ascii_hexdigit).collect();
            U256::from_str_radix(&hex, 16)
        },
    );
    Ok(value.map_err(|e| anyhow!("SP1 public input {input:?}: {e}"))?.to_be_bytes::<32>())
}

/// A kind-0x02 envelope from the SP1 mock proof's five public inputs.
fn mock_envelope(proof: &SP1ProofWithPublicValues, pv: &PublicValues) -> Result<Vec<u8>> {
    let SP1Proof::Groth16(p) = &proof.proof else {
        bail!("mock prover returned a non-Groth16 proof")
    };
    let mut words = Vec::with_capacity(projection::MOCK_PROOF_LEN);
    for input in &p.public_inputs {
        words.extend_from_slice(&public_input_word(input)?);
    }
    Ok(projection::encode_envelope(&Envelope {
        kind: EnvelopeKind::Mock,
        proof: words,
        public_values: *pv,
    }))
}

/// A Groth16 (or mock Groth16) proof of `elf` over `stdin`, with the program's vkey.
async fn prove_groth16<P: Prover>(
    client: &P,
    elf: Elf,
    stdin: SP1Stdin,
) -> Result<(B256, SP1ProofWithPublicValues)> {
    let pk = client.setup(elf).await.map_err(|e| anyhow!("setup: {e}"))?;
    let vkey = B256::from(pk.verifying_key().bytes32_raw());
    let proof = client.prove(&pk, stdin).groth16().await.map_err(|e| anyhow!("prove: {e}"))?;
    Ok((vkey, proof))
}

/// The claim envelope of `pv` from `prover` (§G.2 table). `program_vkey` is the consensus
/// value: SP1-produced envelopes must come from exactly that program.
pub(crate) async fn envelope(
    prover: ProverKind,
    program_vkey: B256,
    elf: Option<&Path>,
    stdin_payload: Vec<u8>,
    pv: &PublicValues,
) -> Result<Vec<u8>> {
    if prover == ProverKind::NativeMock {
        return Ok(projection::encode_envelope(&Envelope {
            kind: EnvelopeKind::Mock,
            proof: projection::mock_proof(program_vkey, pv),
            public_values: *pv,
        }));
    }
    let path =
        elf.ok_or_else(|| anyhow!("{prover:?} proving requires the private-projection ELF"))?;
    let bytes = std::fs::read(path).with_context(|| format!("read {}", path.display()))?;
    let elf = Elf::Dynamic(Arc::from(bytes));
    let mut stdin = SP1Stdin::new();
    stdin.write_vec(stdin_payload);
    let (vkey, proof) = match prover {
        ProverKind::Mock => {
            prove_groth16(&ProverClient::builder().mock().build().await, elf, stdin).await?
        }
        ProverKind::Cpu => {
            prove_groth16(&ProverClient::builder().cpu().build().await, elf, stdin).await?
        }
        ProverKind::Network => {
            prove_groth16(&ProverClient::builder().network().build().await, elf, stdin).await?
        }
        ProverKind::Native | ProverKind::NativeMock => unreachable!("not an SP1 prover"),
    };
    ensure!(
        vkey == program_vkey,
        "ELF vkey {vkey} is not the configured program_vkey {program_vkey}"
    );
    ensure!(
        proof.public_values.as_slice() == pv.as_slice(),
        "SP1 public values differ from native"
    );
    if prover == ProverKind::Mock {
        let out = mock_envelope(&proof, pv)?;
        let env = projection::decode_envelope(&out)?;
        projection::check_mock_proof(&env.proof, program_vkey, pv)?;
        return Ok(out);
    }
    let bytes = proof.bytes();
    // The same verifier admission runs: `sp1-verifier` over the pinned SP1 v6.1.0 circuit.
    projection::sp1::verify_groth16(&projection::sp1::circuit_v6_1_0(), &bytes, program_vkey, pv)?;
    let out = projection::encode_envelope(&Envelope {
        kind: EnvelopeKind::Groth16,
        proof: bytes,
        public_values: *pv,
    });
    projection::decode_envelope(&out)?;
    Ok(out)
}

/// Serve one v2 request: collect, run native `execute`, check the preflight statement, prove.
/// Private bytes stay in memory/stdin; stdout carries only the public envelope.
pub(super) async fn publish(elf: Option<&Path>) -> Result<()> {
    let mut encoded = Vec::new();
    std::io::stdin().take(MAX_REQUEST as u64 + 1).read_to_end(&mut encoded)?;
    ensure!(encoded.len() <= MAX_REQUEST, "publication request too large");
    let r = parse_request(&encoded)?;
    drop(encoded);
    let projection_cfg: RollupConfig = serde_json::from_slice(&r.projection_config)?;
    let profile = projection_cfg
        .private_projection
        .as_ref()
        .ok_or_else(|| anyhow!("projection config lacks private_projection"))?;
    let verifier = profile.verifier.as_str();
    let sp1 = verifier == SP1_PRIVATE_PROJECTION_V1;
    match r.prover {
        ProverKind::Native => {
            ensure!(verifier == EXECUTION_MOCK, "prover native requires {EXECUTION_MOCK}")
        }
        ProverKind::NativeMock | ProverKind::Mock => ensure!(
            sp1 && profile.mock_proofs,
            "prover {:?} requires {SP1_PRIVATE_PROJECTION_V1} with mock_proofs",
            r.prover
        ),
        ProverKind::Cpu | ProverKind::Network => {
            ensure!(sp1, "prover {:?} requires {SP1_PRIVATE_PROJECTION_V1}", r.prover)
        }
    }
    let (input, witness, private_recovery) = collect(&r).await?;
    let started = Instant::now();
    let pv =
        execute(&input, &witness).inspect_err(|_| diagnose(&input, &witness, &private_recovery))?;
    let blocks = input.recovery.len() + input.blocks.len();
    let elapsed = started.elapsed();
    eprintln!(
        "native relation: {blocks} blocks in {elapsed:.1?} ({:.0} blocks/s)",
        blocks as f64 / elapsed.as_secs_f64().max(1e-9)
    );
    check_expected(&pv, &r.expected_public_values, verifier)?;
    ensure!(public_value_word(&pv, 9) == r.recovery_hash, "recovery hash");
    let out = if r.prover == ProverKind::Native {
        let digest =
            r.expected_digest.ok_or_else(|| anyhow!("prover native requires expected_digest"))?;
        ensure!(
            execution_mock_digest(&pv) == digest,
            "native execution returned a different statement"
        );
        let mut proof = b"optimism.private-execution.mock.v1\0".to_vec();
        proof.extend_from_slice(digest.as_slice());
        proof
    } else {
        let default_elf = std::env::var_os("KONA_SP1_ELF_DIR")
            .map(|dir| Path::new(&dir).join("private-projection-elf"));
        let payload = serde_json::to_vec(&(&input, &witness))?;
        envelope(r.prover, profile.program_vkey, elf.or(default_elf.as_deref()), payload, &pv)
            .await?
    };
    println!("0x{}", hex::encode(out));
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use op_alloy_consensus::{OpTxEnvelope, TxDeposit};

    const SAMPLE: &str = r#"{ "version": 2, "prover": "native-mock",
      "private_rpc": "http://a", "projection_rpc": "http://b", "l1_rpc": "http://c",
      "private_config": "0x7b7d", "l1_config": "0x7b7d", "projection_config": "0x7b7d",
      "dependency_set": "0x7b7d",
      "parent_hash": "0x1111111111111111111111111111111111111111111111111111111111111111",
      "anchor": {"number": 0, "hash": "0x2222222222222222222222222222222222222222222222222222222222222222"},
      "anchor_output": "0x3333333333333333333333333333333333333333333333333333333333333333",
      "recovery_hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
      "blocks": [ {"timestamp": 2, "epoch": 1, "transactions": ["0x02"]} ],
      "private_data": "0x00",
      "l1_head": "0x4444444444444444444444444444444444444444444444444444444444444444",
      "expected_public_values": "0xPV" }"#;

    fn sample() -> String {
        SAMPLE.replace("0xPV", &format!("0x{}", "00".repeat(PUBLIC_VALUES_LEN)))
    }

    #[test]
    fn parses_v2_request() {
        let r = parse_request(sample().as_bytes()).unwrap();
        assert_eq!(r.prover, ProverKind::NativeMock);
        assert_eq!(r.anchor.hash, B256::repeat_byte(0x22));
        assert!(r.expected_digest.is_none());
        for (prover, kind) in [
            ("native", ProverKind::Native),
            ("mock", ProverKind::Mock),
            ("cpu", ProverKind::Cpu),
            ("network", ProverKind::Network),
        ] {
            let s = sample().replace("native-mock", prover);
            assert_eq!(parse_request(s.as_bytes()).unwrap().prover, kind);
        }
    }

    #[test]
    fn rejects_other_versions_and_bad_values() {
        let v1 = sample().replace("\"version\": 2", "\"version\": 1");
        assert!(parse_request(v1.as_bytes()).unwrap_err().to_string().contains("version"));
        let e = parse_request(br#"{"private_rpc": "x"}"#).unwrap_err();
        assert!(e.to_string().contains("version"));
        let short = sample().replace(&"00".repeat(PUBLIC_VALUES_LEN), "00");
        assert!(parse_request(short.as_bytes()).is_err());
        assert!(parse_request(sample().replace("native-mock", "prove").as_bytes()).is_err());
    }

    /// The batcher's golden v2 request (`op-batcher/batcher/testdata`), exactly as the client
    /// sends it.
    #[test]
    fn parses_batcher_golden_request() {
        let raw = include_bytes!(
            "../../../../../../op-batcher/batcher/testdata/publication-request-v2.json"
        );
        let r = parse_request(raw).unwrap();
        assert_eq!(r.prover, ProverKind::NativeMock);
        assert!(r.expected_digest.is_none(), "expected_digest is execution-mock only");
        assert!(!r.blocks.is_empty());
        // The relation parses the batcher's L1 config and dependency set with Kona serde. (The
        // golden rollup configs are synthetic, with `l1_chain_id: null`, which no deployed
        // rollup.json has and Kona rejects, so they are not parsed here.)
        let _: kona_genesis::L1ChainConfig = serde_json::from_slice(&r.l1_config).unwrap();
        let _: kona_interop::DependencySet = serde_json::from_slice(&r.dependency_set).unwrap();
    }

    fn info_deposit(number: u64, hash: B256, operator_fee_scalar: u32) -> OpTxEnvelope {
        let mut info =
            kona_protocol::L1BlockInfoIsthmus::new_from_number_and_block_hash(number, hash);
        info.operator_fee_scalar = operator_fee_scalar;
        OpTxEnvelope::from(TxDeposit {
            input: L1BlockInfoTx::Isthmus(info).encode_calldata(),
            ..Default::default()
        })
    }

    fn op_block(number: u64, timestamp: u64, transactions: Vec<OpTxEnvelope>) -> OpBlock {
        OpBlock {
            header: Header { number, timestamp, ..Default::default() },
            body: alloy_consensus::BlockBody { transactions, ..Default::default() },
        }
    }

    /// The host waits for `LightCL`'s private recovery blocks: a private block is their counterpart
    /// only if it carries the canonical forced inputs over the same L1 origin. The sequencer's
    /// original block at that height is not, and its witness describes another state.
    #[test]
    fn recovery_correspondence() {
        let origin = B256::repeat_byte(0x10);
        let user = OpTxEnvelope::from(TxDeposit { mint: 100, ..Default::default() });
        let post_exec = OpTxEnvelope::from(op_alloy_consensus::build_post_exec_tx(7, vec![]));
        // The projection's fee-free L1 info differs from the private one; the rest is identical.
        let public =
            op_block(7, 14, vec![info_deposit(3, origin, 0), user.clone(), post_exec.clone()]);
        let private = op_block(7, 14, vec![info_deposit(3, origin, 23), user.clone(), post_exec]);
        assert!(corresponds(&private, &public));

        let sequencer_tx = OpTxEnvelope::Eip1559(alloy_consensus::Signed::new_unhashed(
            alloy_consensus::TxEip1559 { chain_id: 901, ..Default::default() },
            alloy_primitives::Signature::test_signature(),
        ));
        let stale = [
            op_block(7, 14, vec![info_deposit(3, origin, 23), user.clone(), sequencer_tx]),
            op_block(7, 14, vec![info_deposit(3, origin, 23), user.clone()]),
            op_block(7, 14, vec![info_deposit(4, B256::repeat_byte(0x11), 23), user]),
            op_block(7, 16, private.body.transactions.clone()),
            op_block(8, 14, private.body.transactions),
            op_block(7, 14, vec![]),
        ];
        for (i, block) in stale.iter().enumerate() {
            assert!(!corresponds(block, &public), "case {i}");
        }
    }

    #[test]
    fn witness_must_describe_the_parent_state() {
        let root_node = Bytes::from_static(b"root node");
        let root = keccak256(&root_node);
        let other = Bytes::from_static(b"other");
        assert_eq!(
            witness_describes_parent(Some(root), std::slice::from_ref(&root_node), &[root]),
            Ok(())
        );
        assert_eq!(
            witness_describes_parent(None, std::slice::from_ref(&root_node), &[root]),
            Err("no parent header")
        );
        assert_eq!(
            witness_describes_parent(Some(root), std::slice::from_ref(&other), &[]),
            Err("no parent state root node")
        );
        assert_eq!(
            witness_describes_parent(Some(root), &[root_node, other.clone()], &[keccak256(&other)]),
            Err("account proof of another state")
        );
    }

    #[test]
    fn system_proofs_cover_the_pre_execution_calls() {
        let header = Header { number: 10, timestamp: 8191 + 5, ..Default::default() };
        let [passer, beacon, history] = system_proofs(&header);
        assert_eq!(passer.0, MESSAGE_PASSER.parse::<Address>().unwrap());
        assert_eq!(
            beacon,
            (BEACON_ROOTS_ADDRESS, vec![B256::from(U256::from(5)), B256::from(U256::from(8196))])
        );
        assert_eq!(history, (HISTORY_STORAGE_ADDRESS, vec![B256::from(U256::from(9))]));
    }

    #[test]
    fn parses_sp1_public_inputs() {
        let hex = "FFBn254Fr(0x00eb05d5313a68bb11f0251c0b22468e5c8de24319224bcb23e3a91418162233)";
        assert_eq!(
            B256::from(public_input_word(hex).unwrap()),
            "0x00eb05d5313a68bb11f0251c0b22468e5c8de24319224bcb23e3a91418162233"
                .parse::<B256>()
                .unwrap()
        );
        assert_eq!(public_input_word("0").unwrap(), [0u8; 32]);
        assert_eq!(B256::from(public_input_word("258").unwrap()), B256::from(U256::from(258)));
        assert!(public_input_word("x1").is_err());
    }

    #[test]
    fn word_three_is_excluded_only_under_execution_mock() {
        let pv = [7u8; PUBLIC_VALUES_LEN];
        let mut expected = pv.to_vec();
        expected[3 * 32] = 0;
        assert!(check_expected(&pv, &expected, EXECUTION_MOCK).is_ok());
        assert!(check_expected(&pv, &expected, SP1_PRIVATE_PROJECTION_V1).is_err());
        expected[19 * 32] = 0;
        assert!(check_expected(&pv, &expected, EXECUTION_MOCK).is_err());
    }
}
