use anyhow::{anyhow, Context, Result};
use commonware_actor::Feedback;
use commonware_codec::Encode;
use commonware_consensus::{
    simplex::{self, elector::RoundRobin, types::Activity, Engine, ForwardingPolicy, Plan},
    types::{Epoch, ViewDelta},
    Automaton, CertifiableAutomaton, Relay, Reporter,
};
use commonware_cryptography::{
    certificate::Scheme as _, ed25519, secp256r1, sha256::Digest as Sha256Digest, Hasher, Sha256,
    Signer,
};
use commonware_p2p::simulated::{Config as SimulatedNetworkConfig, Link, Network};
use commonware_parallel::Sequential;
use commonware_runtime::{
    buffer::paged::CacheRef, deterministic, Clock as _, Quota, Runner as _, Supervisor as _,
};
use commonware_utils::{
    channel::{mpsc, oneshot},
    ordered::Set,
    sync::Mutex,
    TryCollect,
};
use k256::ecdsa::{RecoveryId, Signature, SigningKey};
use serde::{Deserialize, Serialize};
use sha3::{Digest as Sha3Digest, Keccak256};
use std::{
    env,
    io::{Read, Write},
    net::{TcpListener, TcpStream},
    num::{NonZeroU16, NonZeroU32, NonZeroUsize},
    sync::Arc,
    time::Duration,
};

const PROVIDER: &str = "commonware-p2pft-poc-secp256r1";
const SIMPLEX_PROVIDER: &str = "commonware-simplex-poc-ed25519";
const COMMONWARE_NAMESPACE: &[u8] = b"op-batcher-consensus-poc/commonware";
const SIMPLEX_NAMESPACE: &[u8] = b"op-batcher-consensus-poc/simplex";
const EVM_PREFIX: &[u8] = b"BCSIG1";
const SIMPLEX_PROOF_PREFIX: &[u8] = b"CWSIMPLX1";
const EVM_SIGNING_KEY: [u8; 32] = [
    0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef,
    0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef,
];

#[derive(Debug, Deserialize)]
struct ProofRequest {
    l1_chain_id: String,
    l2_chain_id: String,
    batch_inbox: String,
    batcher: String,
    blob_versioned_hashes: Vec<String>,
}

#[derive(Debug, Serialize)]
struct ProofResponse {
    provider: String,
    certificate: String,
    calldata: String,
}

fn main() -> Result<()> {
    let mut listen = "127.0.0.1:0".to_string();
    let mut valid = true;
    let mut simplex = false;
    let args: Vec<String> = env::args().collect();
    let mut i = 1;
    while i < args.len() {
        match args[i].as_str() {
            "--listen" => {
                i += 1;
                listen = args
                    .get(i)
                    .ok_or_else(|| anyhow!("missing --listen value"))?
                    .clone();
            }
            "--invalid" => valid = false,
            "--commonware-simplex" => simplex = true,
            other => return Err(anyhow!("unknown argument {other}")),
        }
        i += 1;
    }

    let listener = TcpListener::bind(&listen).context("bind listener")?;
    eprintln!(
        "op-batcher-consensus-sidecar listening on {}",
        listener.local_addr()?
    );
    for stream in listener.incoming() {
        match stream {
            Ok(stream) => {
                if let Err(err) = handle(stream, valid, simplex) {
                    eprintln!("request failed: {err:#}");
                }
            }
            Err(err) => eprintln!("accept failed: {err}"),
        }
    }
    Ok(())
}

fn handle(mut stream: TcpStream, valid: bool, simplex: bool) -> Result<()> {
    let mut buf = vec![0u8; 64 * 1024];
    let n = stream.read(&mut buf).context("read request")?;
    let req_bytes = &buf[..n];
    let body_start = req_bytes
        .windows(4)
        .position(|w| w == b"\r\n\r\n")
        .map(|p| p + 4)
        .ok_or_else(|| anyhow!("malformed HTTP request"))?;
    let body = &req_bytes[body_start..];
    let req: ProofRequest = serde_json::from_slice(body).context("decode proof request")?;
    let resp = build_response(&req, valid, simplex).context("build proof response")?;
    let payload = serde_json::to_vec(&resp).context("encode proof response")?;
    write_response(&mut stream, 200, "OK", &payload)?;
    Ok(())
}

fn write_response(stream: &mut TcpStream, code: u16, reason: &str, body: &[u8]) -> Result<()> {
    write!(
        stream,
        "HTTP/1.1 {code} {reason}\r\ncontent-type: application/json\r\ncontent-length: {}\r\nconnection: close\r\n\r\n",
        body.len()
    )?;
    stream.write_all(body)?;
    Ok(())
}

fn build_response(req: &ProofRequest, valid: bool, simplex: bool) -> Result<ProofResponse> {
    let digest = proof_digest(req)?;
    if simplex {
        return build_simplex_response(&digest, valid);
    }
    let commonware_signer = secp256r1::recoverable::PrivateKey::from_seed(1);
    let commonware_sig = commonware_signer.sign(COMMONWARE_NAMESPACE, digest.as_slice());
    let mut certificate = Vec::new();
    certificate.extend_from_slice(b"CWPOC1");
    certificate.extend_from_slice(commonware_sig.as_ref());

    let mut calldata = evm_calldata(&digest)?;
    if !valid {
        certificate.push(0);
        let last = calldata
            .last_mut()
            .ok_or_else(|| anyhow!("empty calldata"))?;
        *last = 0;
    }

    Ok(ProofResponse {
        provider: PROVIDER.to_string(),
        certificate: format!("0x{}", hex::encode(certificate)),
        calldata: format!("0x{}", hex::encode(calldata)),
    })
}

fn build_simplex_response(digest: &[u8; 32], valid: bool) -> Result<ProofResponse> {
    let certificate = run_simplex_consensus(*digest)?;
    let mut calldata = evm_commonware_calldata(digest, &certificate)?;
    if !valid {
        let last = calldata
            .get_mut(SIMPLEX_PROOF_PREFIX.len() + 32 + 65)
            .ok_or_else(|| anyhow!("empty calldata"))?;
        *last = 0;
    }
    Ok(ProofResponse {
        provider: SIMPLEX_PROVIDER.to_string(),
        certificate: format!("0x{}", hex::encode(certificate)),
        calldata: format!("0x{}", hex::encode(calldata)),
    })
}

type SimplexScheme = simplex::scheme::ed25519::Scheme;
type SimplexFinalization = simplex::types::Finalization<SimplexScheme, Sha256Digest>;

#[derive(Clone)]
struct FixedMailbox {
    sender: mpsc::UnboundedSender<FixedMessage>,
}

enum FixedMessage {
    Genesis {
        epoch: Epoch,
        response: oneshot::Sender<Sha256Digest>,
    },
    Propose {
        response: oneshot::Sender<Sha256Digest>,
    },
    Verify {
        payload: Sha256Digest,
        response: oneshot::Sender<bool>,
    },
    Certify {
        payload: Sha256Digest,
        response: oneshot::Sender<bool>,
    },
}

struct FixedApplication<E> {
    context: E,
    target: Sha256Digest,
    mailbox: mpsc::UnboundedReceiver<FixedMessage>,
}

impl<E> FixedApplication<E>
where
    E: commonware_runtime::Spawner,
{
    fn new(context: E, target: Sha256Digest) -> (Self, FixedMailbox) {
        let (sender, mailbox) = mpsc::unbounded_channel();
        (
            Self {
                context,
                target,
                mailbox,
            },
            FixedMailbox { sender },
        )
    }

    fn start(mut self) {
        self.context.spawn(move |_| async move {
            while let Some(message) = self.mailbox.recv().await {
                match message {
                    FixedMessage::Genesis { epoch, response } => {
                        let mut hasher = Sha256::default();
                        hasher.update(b"op-batcher-consensus-simplex-genesis");
                        hasher.update(&epoch.encode());
                        let _ = response.send(hasher.finalize());
                    }
                    FixedMessage::Propose { response } => {
                        let _ = response.send(self.target);
                    }
                    FixedMessage::Verify { payload, response }
                    | FixedMessage::Certify { payload, response } => {
                        let _ = response.send(payload == self.target);
                    }
                }
            }
        });
    }
}

impl Automaton for FixedMailbox {
    type Context = simplex::types::Context<Sha256Digest, ed25519::PublicKey>;
    type Digest = Sha256Digest;

    async fn genesis(&mut self, epoch: Epoch) -> Self::Digest {
        let (response, receiver) = oneshot::channel();
        let _ = self.sender.send(FixedMessage::Genesis { epoch, response });
        receiver.await.expect("simplex genesis response")
    }

    async fn propose(&mut self, _context: Self::Context) -> oneshot::Receiver<Self::Digest> {
        let (response, receiver) = oneshot::channel();
        let _ = self.sender.send(FixedMessage::Propose { response });
        receiver
    }

    async fn verify(
        &mut self,
        _context: Self::Context,
        payload: Self::Digest,
    ) -> oneshot::Receiver<bool> {
        let (response, receiver) = oneshot::channel();
        let _ = self.sender.send(FixedMessage::Verify { payload, response });
        receiver
    }
}

impl CertifiableAutomaton for FixedMailbox {
    async fn certify(
        &mut self,
        _round: commonware_consensus::types::Round,
        payload: Self::Digest,
    ) -> oneshot::Receiver<bool> {
        let (response, receiver) = oneshot::channel();
        let _ = self
            .sender
            .send(FixedMessage::Certify { payload, response });
        receiver
    }
}

impl Relay for FixedMailbox {
    type Digest = Sha256Digest;
    type Plan = Plan<ed25519::PublicKey>;
    type PublicKey = ed25519::PublicKey;

    fn broadcast(&mut self, _payload: Self::Digest, _plan: Self::Plan) -> Feedback {
        Feedback::Ok
    }
}

#[derive(Clone)]
struct CaptureReporter {
    scheme: SimplexScheme,
    finalization: Arc<Mutex<Option<SimplexFinalization>>>,
}

impl Reporter for CaptureReporter {
    type Activity = Activity<SimplexScheme, Sha256Digest>;

    fn report(&mut self, activity: Self::Activity) -> Feedback {
        if let Activity::Finalization(finalization) = activity {
            if self
                .scheme
                .verify_certificate::<_, Sha256Digest, commonware_utils::N3f1>(
                    &mut commonware_utils::test_rng(),
                    simplex::types::Subject::Finalize {
                        proposal: &finalization.proposal,
                    },
                    &finalization.certificate,
                    &Sequential,
                )
            {
                *self.finalization.lock() = Some(finalization);
            }
        }
        Feedback::Ok
    }
}

fn run_simplex_consensus(digest: [u8; 32]) -> Result<Vec<u8>> {
    let target = Sha256Digest::from(digest);
    let runner = deterministic::Runner::default();
    runner.start(|context| async move {
        let signers = (0..4)
            .map(|i| ed25519::PrivateKey::from_seed(i + 1))
            .collect::<Vec<_>>();
        let participants: Set<_> = signers
            .iter()
            .map(|signer| signer.public_key())
            .try_collect()
            .map_err(|_| anyhow!("duplicate Commonware Simplex participants"))?;

        let schemes = signers
            .iter()
            .map(|signer| {
                SimplexScheme::signer(SIMPLEX_NAMESPACE, participants.clone(), signer.clone())
                    .ok_or_else(|| anyhow!("simplex signer not in participants"))
            })
            .collect::<Result<Vec<_>>>()?;

        let (network, oracle) = Network::new_with_peers(
            context.child("network"),
            SimulatedNetworkConfig {
                max_size: 1024 * 1024,
                disconnect_on_block: false,
                tracked_peer_sets: NonZeroUsize::new(1).unwrap(),
            },
            participants.iter().cloned().collect::<Vec<_>>(),
        )
        .await;
        network.start();

        let link = Link {
            latency: Duration::from_millis(1),
            jitter: Duration::from_millis(0),
            success_rate: 1.0,
        };
        for from in participants.iter() {
            for to in participants.iter() {
                if from == to {
                    continue;
                }
                oracle
                    .add_link(from.clone(), to.clone(), link.clone())
                    .await
                    .map_err(|err| anyhow!("link simplex peers: {err:?}"))?;
            }
        }

        let finalization = Arc::new(Mutex::new(None));
        for (idx, signer) in signers.iter().enumerate() {
            let public_key = signer.public_key();
            let control = oracle.control(public_key.clone());
            let vote = control
                .register(0, Quota::per_second(NonZeroU32::MAX))
                .await
                .map_err(|err| anyhow!("register simplex vote channel: {err:?}"))?;
            let certificate = control
                .register(1, Quota::per_second(NonZeroU32::MAX))
                .await
                .map_err(|err| anyhow!("register simplex certificate channel: {err:?}"))?;
            let resolver = control
                .register(2, Quota::per_second(NonZeroU32::MAX))
                .await
                .map_err(|err| anyhow!("register simplex resolver channel: {err:?}"))?;

            let (application, mailbox) =
                FixedApplication::new(context.child("application"), target);
            application.start();

            let scheme = schemes[idx].clone();
            let reporter = CaptureReporter {
                scheme: scheme.clone(),
                finalization: finalization.clone(),
            };
            let cfg = simplex::Config {
                scheme,
                elector: RoundRobin::<Sha256>::default(),
                blocker: oracle.control(public_key.clone()),
                automaton: mailbox.clone(),
                relay: mailbox,
                reporter,
                partition: format!("batch-consensus-{idx}"),
                mailbox_size: NonZeroUsize::new(1024).unwrap(),
                epoch: Epoch::zero(),
                replay_buffer: NonZeroUsize::new(1024 * 1024).unwrap(),
                write_buffer: NonZeroUsize::new(1024 * 1024).unwrap(),
                leader_timeout: Duration::from_millis(100),
                certification_timeout: Duration::from_millis(200),
                timeout_retry: Duration::from_secs(1),
                fetch_timeout: Duration::from_millis(100),
                activity_timeout: ViewDelta::new(10),
                skip_timeout: ViewDelta::new(5),
                fetch_concurrent: NonZeroUsize::new(4).unwrap(),
                page_cache: CacheRef::from_pooler(
                    &context,
                    NonZeroU16::new(1024).unwrap(),
                    NonZeroUsize::new(10).unwrap(),
                ),
                strategy: Sequential,
                forwarding: ForwardingPolicy::Disabled,
            };
            Engine::new(context.child("engine"), cfg).start(vote, certificate, resolver);
        }

        for _ in 0..100 {
            if let Some(finalization) = finalization.lock().clone() {
                if finalization.proposal.payload != target {
                    return Err(anyhow!("simplex finalized unexpected payload"));
                }
                eprintln!(
                    "commonware simplex finalized batch consensus proof view={} payload={:?}",
                    finalization.proposal.round.view(),
                    finalization.proposal.payload
                );
                let mut out = Vec::new();
                out.extend_from_slice(b"CWSIMPLX1");
                out.extend_from_slice(&finalization.encode());
                return Ok(out);
            }
            context.sleep(Duration::from_millis(20)).await;
        }
        Err(anyhow!(
            "timed out waiting for Commonware Simplex finalization"
        ))
    })
}

fn proof_digest(req: &ProofRequest) -> Result<[u8; 32]> {
    let mut hasher = Keccak256::new();
    hasher.update(req.l1_chain_id.as_bytes());
    hasher.update(b"|");
    hasher.update(req.l2_chain_id.as_bytes());
    hasher.update(b"|");
    hasher.update(hex_address(&req.batch_inbox)?);
    hasher.update(hex_address(&req.batcher)?);
    for h in &req.blob_versioned_hashes {
        hasher.update(hex_hash(h)?);
    }
    Ok(hasher.finalize().into())
}

fn evm_calldata(digest: &[u8; 32]) -> Result<Vec<u8>> {
    evm_calldata_with_prefix(EVM_PREFIX, digest, None)
}

fn evm_commonware_calldata(digest: &[u8; 32], certificate: &[u8]) -> Result<Vec<u8>> {
    if certificate.is_empty() {
        return Err(anyhow!("empty Commonware certificate"));
    }
    evm_calldata_with_prefix(SIMPLEX_PROOF_PREFIX, digest, Some(certificate))
}

fn evm_calldata_with_prefix(
    prefix: &[u8],
    digest: &[u8; 32],
    certificate: Option<&[u8]>,
) -> Result<Vec<u8>> {
    let signing_key = SigningKey::from_slice(&EVM_SIGNING_KEY).context("load EVM signing key")?;
    let (sig, recid): (Signature, RecoveryId) = signing_key
        .sign_prehash_recoverable(digest)
        .context("sign EVM digest")?;
    let certificate_len = certificate.map_or(0, <[u8]>::len);
    let mut out = Vec::with_capacity(prefix.len() + 32 + 65 + 1 + certificate_len);
    out.extend_from_slice(prefix);
    out.extend_from_slice(digest);
    out.extend_from_slice(&sig.to_bytes());
    out.push(recid.to_byte());
    out.push(1);
    if let Some(certificate) = certificate {
        out.extend_from_slice(certificate);
    }
    Ok(out)
}

fn hex_address(s: &str) -> Result<[u8; 20]> {
    let bytes = decode_hex(s)?;
    bytes
        .try_into()
        .map_err(|_| anyhow!("expected 20-byte address: {s}"))
}

fn hex_hash(s: &str) -> Result<[u8; 32]> {
    let bytes = decode_hex(s)?;
    bytes
        .try_into()
        .map_err(|_| anyhow!("expected 32-byte hash: {s}"))
}

fn decode_hex(s: &str) -> Result<Vec<u8>> {
    hex::decode(s.strip_prefix("0x").unwrap_or(s)).with_context(|| format!("decode hex {s}"))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn request() -> ProofRequest {
        ProofRequest {
            l1_chain_id: "900".to_string(),
            l2_chain_id: "901".to_string(),
            batch_inbox: "0x0000000000000000000000000000000000000001".to_string(),
            batcher: "0x0000000000000000000000000000000000000002".to_string(),
            blob_versioned_hashes: vec![
                "0x0100000000000000000000000000000000000000000000000000000000000000".to_string(),
            ],
        }
    }

    #[test]
    fn simplex_response_contains_commonware_certificate_and_evm_calldata() {
        let resp = build_response(&request(), true, true).unwrap();
        let certificate = decode_hex(&resp.certificate).unwrap();
        let calldata = decode_hex(&resp.calldata).unwrap();

        assert_eq!(resp.provider, SIMPLEX_PROVIDER);
        assert!(certificate.starts_with(b"CWSIMPLX1"));
        assert!(calldata.starts_with(SIMPLEX_PROOF_PREFIX));
        assert_eq!(
            calldata.len(),
            SIMPLEX_PROOF_PREFIX.len() + 32 + 65 + 1 + certificate.len()
        );
        assert_eq!(calldata[SIMPLEX_PROOF_PREFIX.len() + 32 + 65], 1);
        assert_eq!(
            &calldata[SIMPLEX_PROOF_PREFIX.len() + 32 + 65 + 1..],
            certificate
        );
    }
}
