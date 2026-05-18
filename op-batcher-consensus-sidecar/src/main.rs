use anyhow::{anyhow, Context, Result};
use commonware_cryptography::{secp256r1, Signer};
use k256::ecdsa::{RecoveryId, Signature, SigningKey};
use serde::{Deserialize, Serialize};
use sha3::{Digest, Keccak256};
use std::{
    env,
    io::{Read, Write},
    net::{TcpListener, TcpStream},
};

const PROVIDER: &str = "commonware-p2pft-poc-secp256r1";
const COMMONWARE_NAMESPACE: &[u8] = b"op-batcher-consensus-poc/commonware";
const EVM_PREFIX: &[u8] = b"BCSIG1";
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
                if let Err(err) = handle(stream, valid) {
                    eprintln!("request failed: {err:#}");
                }
            }
            Err(err) => eprintln!("accept failed: {err}"),
        }
    }
    Ok(())
}

fn handle(mut stream: TcpStream, valid: bool) -> Result<()> {
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
    let resp = build_response(&req, valid).context("build proof response")?;
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

fn build_response(req: &ProofRequest, valid: bool) -> Result<ProofResponse> {
    let digest = proof_digest(req)?;
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
    let signing_key = SigningKey::from_slice(&EVM_SIGNING_KEY).context("load EVM signing key")?;
    let (sig, recid): (Signature, RecoveryId) = signing_key
        .sign_prehash_recoverable(digest)
        .context("sign EVM digest")?;
    let mut out = Vec::with_capacity(EVM_PREFIX.len() + 32 + 65 + 1);
    out.extend_from_slice(EVM_PREFIX);
    out.extend_from_slice(digest);
    out.extend_from_slice(&sig.to_bytes());
    out.push(recid.to_byte());
    out.push(1);
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
