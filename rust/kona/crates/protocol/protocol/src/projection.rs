//! Pure, whole-range admission for the experimental public projection protocol.
//! No execution correctness is established by the explicitly insecure stub verifier.

use crate::SpanBatch;
use alloc::{vec, vec::Vec};
use alloy_consensus::{TxEnvelope, transaction::SignerRecoverable};
use alloy_eips::{Decodable2718, Encodable2718};
use alloy_primitives::{Address, B256, Bytes, U256, address, keccak256};
use alloy_sol_types::{SolCall, sol};
use kona_genesis::RollupConfig;

const REGISTRY: Address = address!("420000000000000000000000000000000000002e");
const MESSENGER: Address = address!("4200000000000000000000000000000000000023");
const INBOX: Address = address!("4200000000000000000000000000000000000022");
const REPLAYER: Address = address!("420000000000000000000000000000000000002f");
const BRIDGE: Address = address!("4200000000000000000000000000000000000024");
const MAX_MESSAGE: usize = 1024 * 1024;
const MAX_PROOF: usize = 65536;

sol! {
    /// The existing range-claim wire tuple; proof policy is owned by admission.
    #[derive(Debug, PartialEq, Eq)]
    struct RangeClaim {
        uint8 version;
        uint64 firstBlock;
        uint64 lastBlock;
        bytes32 privateTerminalBlockHash;
        bytes32 privateTerminalParentHash;
        bytes32 l1Head;
        bytes32 rollupConfigHash;
        bytes32 depSetHash;
        bytes32 privateDataHash;
        bytes proof;
    }
    #[derive(Debug)]
    struct Identifier {
        address origin;
        uint256 blockNumber;
        uint256 logIndex;
        uint256 timestamp;
        uint256 chainId;
    }
    function postClaim(RangeClaim claim);
    function replaySentMessage(uint256 destination, uint256 nonce, address sender, address target, bytes message);
    function replayEvent(bytes32[] topics, bytes data);
    function validateMessage(Identifier identifier, bytes32 payloadHash);
}

/// A public statement derived from the actual decoded span. Claim proof bytes are empty.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Statement {
    /// Full-width chain identity.
    pub chain_id: B256,
    /// Authenticated parent of the entire span, including canonical overlap.
    pub parent_hash: B256,
    /// Domain-separated commitment to ordered public records and their positions.
    pub projection_hash: B256,
    /// Operator-supplied checkpoint fields, with proof removed.
    pub claim: RangeClaim,
}

/// Deterministic proof checking with no I/O or mutable state.
pub trait ProofVerifier {
    /// Verify only the explicit statement and proof; do not consult node state.
    fn verify(&self, statement: &Statement, proof: &[u8]) -> Result<(), ProjectionError>;
}

/// Deliberately accepts dummy bytes. This is not a private-execution proof system.
#[derive(Debug, Default)]
pub struct StubVerifier;
impl ProofVerifier for StubVerifier {
    fn verify(&self, _: &Statement, _: &[u8]) -> Result<(), ProjectionError> {
        Ok(())
    }
}

/// A deterministic projection admission rejection.
#[derive(Debug, Clone, Copy, PartialEq, Eq, thiserror::Error)]
#[error("invalid projection range: {0}")]
pub struct ProjectionError(pub &'static str);

fn decode<C: SolCall>(data: &[u8]) -> Result<C, ProjectionError> {
    let call = C::abi_decode_validate(data).map_err(|_| ProjectionError("malformed call"))?;
    if call.abi_encode() != data {
        return Err(ProjectionError("noncanonical call"));
    }
    Ok(call)
}

fn put(out: &mut Vec<u8>, n: u64) {
    out.extend_from_slice(&n.to_be_bytes());
}

fn import_keys(call: &validateMessageCall) -> Result<Vec<B256>, ProjectionError> {
    let id = &call.identifier;
    let number: u64 = id.blockNumber.try_into().map_err(|_| ProjectionError("block overflow"))?;
    let timestamp: u64 =
        id.timestamp.try_into().map_err(|_| ProjectionError("timestamp overflow"))?;
    let index: u32 = id.logIndex.try_into().map_err(|_| ProjectionError("index overflow"))?;
    let chain = id.chainId.to_be_bytes::<32>();
    let mut lookup = [0u8; 32];
    lookup[0] = 1;
    lookup[4..12].copy_from_slice(&chain[24..]);
    lookup[12..20].copy_from_slice(&number.to_be_bytes());
    lookup[20..28].copy_from_slice(&timestamp.to_be_bytes());
    lookup[28..].copy_from_slice(&index.to_be_bytes());
    let mut keys = vec![B256::from(lookup)];
    if chain[..24].iter().any(|b| *b != 0) {
        let mut ext = [0u8; 32];
        ext[0] = 2;
        ext[8..].copy_from_slice(&chain[..24]);
        keys.push(B256::from(ext));
    }
    let mut log = id.origin.as_slice().to_vec();
    log.extend_from_slice(call.payloadHash.as_slice());
    let mut packed = keccak256(log).as_slice().to_vec();
    packed.extend_from_slice(&[0u8; 12]);
    put(&mut packed, number);
    put(&mut packed, timestamp);
    packed.extend_from_slice(&index.to_be_bytes());
    let mut bound = keccak256(packed).as_slice().to_vec();
    bound.extend_from_slice(&chain);
    let mut checksum = keccak256(bound);
    checksum[0] = 3;
    keys.push(checksum);
    Ok(keys)
}

/// Check every transaction and the complete range before calling the proof verifier.
/// Pure with respect to all inputs; no valid prefix is emitted on error.
pub fn validate_projection_range(
    cfg: &RollupConfig,
    parent_hash: B256,
    span: &SpanBatch,
    verifier: &impl ProofVerifier,
) -> Result<Statement, ProjectionError> {
    let mode =
        cfg.private_projection.as_ref().ok_or(ProjectionError("missing projection config"))?;
    if mode.verifier != "insecure-stub-v1" || cfg.block_time == 0 || cfg.l2_chain_id.id() == 0 {
        return Err(ProjectionError("unsupported projection config"));
    }
    let count = span.batches.len();
    if count == 0 || count > 65536 {
        return Err(ProjectionError("range length"));
    }
    let start = span.batches[0].timestamp;
    let elapsed =
        start.checked_sub(cfg.genesis.l2_time).ok_or(ProjectionError("before genesis"))?;
    if elapsed % cfg.block_time != 0 || elapsed == 0 {
        return Err(ProjectionError("unaligned range"));
    }
    let first = cfg
        .genesis
        .l2
        .number
        .checked_add(elapsed / cfg.block_time)
        .ok_or(ProjectionError("height overflow"))?;
    let last = first.checked_add((count - 1) as u64).ok_or(ProjectionError("height overflow"))?;
    start
        .checked_add(
            ((count - 1) as u64)
                .checked_mul(cfg.block_time)
                .ok_or(ProjectionError("time overflow"))?,
        )
        .ok_or(ProjectionError("time overflow"))?;
    let mut claim = None;
    let mut transcript = b"optimism.private-projection.v1\0".to_vec();
    put(&mut transcript, count as u64);
    for (i, block) in span.batches.iter().enumerate() {
        if block.timestamp != start + (i as u64) * cfg.block_time {
            return Err(ProjectionError("noncontiguous timestamps"));
        }
        put(&mut transcript, first + i as u64);
        put(&mut transcript, block.timestamp);
        put(&mut transcript, block.epoch_num);
        if i == 0 && block.transactions.is_empty() {
            return Err(ProjectionError("missing claim"));
        }
        put(&mut transcript, block.transactions.len() as u64);
        for (j, raw) in block.transactions.iter().enumerate() {
            if raw.len() > MAX_MESSAGE + 4096 {
                return Err(ProjectionError("transaction too large"));
            }
            let envelope = TxEnvelope::decode_2718_exact(raw)
                .map_err(|_| ProjectionError("transaction encoding"))?;
            if envelope.encoded_2718() != raw.as_ref() {
                return Err(ProjectionError("noncanonical transaction"));
            }
            let TxEnvelope::Eip1559(signed) = envelope else {
                return Err(ProjectionError("transaction type"));
            };
            let sender = SignerRecoverable::recover_signer(&signed)
                .map_err(|_| ProjectionError("signature"))?;
            let tx = signed.tx();
            let to = tx.to.to().ok_or(ProjectionError("creation"))?;
            if tx.chain_id != cfg.l2_chain_id.id() ||
                tx.value != U256::ZERO ||
                tx.max_fee_per_gas != 0 ||
                tx.max_priority_fee_per_gas != 0 ||
                tx.gas_limit == 0 ||
                tx.gas_limit > 16777216
            {
                return Err(ProjectionError("transaction envelope"));
            }
            let mut data = tx.input.to_vec();
            let mut keys = None;
            match *to {
                REGISTRY => {
                    if i != 0 || j != 0 {
                        return Err(ProjectionError("duplicate or misplaced claim"));
                    }
                    if data.len() > 4 + 384 + MAX_PROOF {
                        return Err(ProjectionError("claim too large"));
                    }
                    let mut decoded = decode::<postClaimCall>(&data)?;
                    let c = &decoded.claim;
                    if c.version != 1 ||
                        c.firstBlock != first ||
                        c.lastBlock != last ||
                        c.proof.len() > MAX_PROOF
                    {
                        return Err(ProjectionError("claim framing"));
                    }
                    claim = Some(decoded.claim.clone());
                    decoded.claim.proof = Bytes::new();
                    data = decoded.abi_encode();
                }
                MESSENGER => {
                    if data.len() > MAX_MESSAGE + 1024 {
                        return Err(ProjectionError("message too large"));
                    }
                    let c = decode::<replaySentMessageCall>(&data)?;
                    if c.message.len() > MAX_MESSAGE || c.sender == BRIDGE || c.target == BRIDGE {
                        return Err(ProjectionError("unsupported message"));
                    }
                }
                INBOX => {
                    if data.len() != 196 {
                        return Err(ProjectionError("import length"));
                    }
                    keys = Some(import_keys(&decode::<validateMessageCall>(&data)?)?);
                }
                REPLAYER => {
                    if !mode.allow_events || data.len() > MAX_MESSAGE + 1024 {
                        return Err(ProjectionError("event disabled or too large"));
                    }
                    let c = decode::<replayEventCall>(&data)?;
                    if c.topics.len() > 4 || c.data.len() > MAX_MESSAGE {
                        return Err(ProjectionError("event bounds"));
                    }
                }
                _ => return Err(ProjectionError("unexpected destination")),
            }
            if i == 0 && j == 0 && claim.is_none() {
                return Err(ProjectionError("opening claim"));
            }
            match keys {
                Some(expected) => {
                    if tx.access_list.0.len() != 1 ||
                        tx.access_list.0[0].address != INBOX ||
                        tx.access_list.0[0].storage_keys != expected
                    {
                        return Err(ProjectionError("import access list"));
                    }
                }
                None => {
                    if !tx.access_list.0.is_empty() {
                        return Err(ProjectionError("unexpected access list"));
                    }
                }
            }
            transcript.extend_from_slice(sender.as_slice());
            put(&mut transcript, tx.nonce);
            put(&mut transcript, tx.gas_limit);
            transcript.extend_from_slice(to.as_slice());
            put(&mut transcript, data.len() as u64);
            transcript.extend_from_slice(&data);
        }
    }
    let mut claim = claim.ok_or(ProjectionError("missing claim"))?;
    let proof = core::mem::take(&mut claim.proof);
    let statement = Statement {
        chain_id: B256::from(U256::from(cfg.l2_chain_id.id()).to_be_bytes::<32>()),
        parent_hash,
        projection_hash: keccak256(transcript),
        claim,
    };
    verifier.verify(&statement, &proof)?;
    Ok(statement)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::SpanBatchElement;
    use kona_genesis::PrivateProjectionConfig;
    use serde_json::Value;

    fn vectors() -> Vec<Value> {
        serde_json::from_str(include_str!(
            "../../../../../../op-private-interop/projection/testdata/ranges.json"
        ))
        .unwrap()
    }

    fn inputs(v: &Value) -> (RollupConfig, SpanBatch, B256) {
        let cfg = RollupConfig {
            l2_chain_id: 901.into(),
            block_time: 2,
            genesis: kona_genesis::ChainGenesis { l2_time: 1000, ..Default::default() },
            private_projection: Some(PrivateProjectionConfig {
                verifier: v["config"]["verifier"].as_str().unwrap().into(),
                allow_events: v["config"]["allow_events"].as_bool().unwrap_or(false),
            }),
            ..Default::default()
        };
        let span = SpanBatch {
            batches: v["blocks"]
                .as_array()
                .unwrap()
                .iter()
                .map(|b| SpanBatchElement {
                    timestamp: b["timestamp"].as_u64().unwrap(),
                    epoch_num: b["epoch"].as_u64().unwrap(),
                    transactions: b["transactions"]
                        .as_array()
                        .map(|xs| {
                            xs.iter()
                                .map(|tx| serde_json::from_value(tx.clone()).unwrap())
                                .collect()
                        })
                        .unwrap_or_default(),
                })
                .collect(),
            ..Default::default()
        };
        let mut parent = B256::ZERO;
        parent[0] = 1;
        (cfg, span, parent)
    }

    #[test]
    fn shared_projection_vectors_and_purity() {
        for v in vectors() {
            let (cfg, span, parent) = inputs(&v);
            let before_cfg = cfg.clone();
            let before_span = span.clone();
            let first = validate_projection_range(&cfg, parent, &span, &StubVerifier);
            assert_eq!(first.is_ok(), v["accept"].as_bool().unwrap(), "{}: {first:?}", v["name"]);
            if let Ok(statement) = &first {
                let expected: B256 = serde_json::from_value(v["digest"].clone()).unwrap();
                assert_eq!(statement.projection_hash, expected, "{}", v["name"]);
            }
            assert_eq!(first, validate_projection_range(&cfg, parent, &span, &StubVerifier));
            assert_eq!(cfg, before_cfg);
            assert_eq!(span, before_span);
        }
    }

    struct Reject;
    impl ProofVerifier for Reject {
        fn verify(&self, _: &Statement, _: &[u8]) -> Result<(), ProjectionError> {
            Err(ProjectionError("proof rejected"))
        }
    }
    struct Binding(Statement);
    impl ProofVerifier for Binding {
        fn verify(&self, statement: &Statement, proof: &[u8]) -> Result<(), ProjectionError> {
            if statement != &self.0 || proof != b"dummy" {
                return Err(ProjectionError("wrong statement or proof"));
            }
            Ok(())
        }
    }
    #[test]
    fn proof_result_gates_admission() {
        let (cfg, mut span, parent) = inputs(&vectors()[0]);
        let statement = validate_projection_range(&cfg, parent, &span, &StubVerifier).unwrap();
        assert!(statement.claim.proof.is_empty());
        assert_eq!(
            validate_projection_range(&cfg, parent, &span, &Reject),
            Err(ProjectionError("proof rejected"))
        );
        let verifier = Binding(statement);
        assert!(validate_projection_range(&cfg, parent, &span, &verifier).is_ok());
        span.batches[2].transactions.clear();
        assert_eq!(
            validate_projection_range(&cfg, parent, &span, &verifier),
            Err(ProjectionError("wrong statement or proof"))
        );
    }
    #[tokio::test]
    async fn projection_admission_preflights_late_schedule_errors() {
        use crate::{BlockInfo, L2BlockInfo, test_utils::TestBatchValidator};
        use kona_genesis::HardForkConfig;
        for scenario in ["valid", "late_origin", "late_fork", "late_drift"] {
            let (mut cfg, mut span, parent_hash) = inputs(&vectors()[0]);
            cfg.seq_window_size = 100;
            cfg.max_sequencer_drift = 600;
            cfg.hardforks = HardForkConfig {
                delta_time: Some(0),
                holocene_time: Some(0),
                ..Default::default()
            };
            let mut origins = [
                BlockInfo {
                    hash: B256::with_last_byte(5),
                    number: 5,
                    timestamp: 1000,
                    ..Default::default()
                },
                BlockInfo {
                    hash: B256::with_last_byte(6),
                    number: 6,
                    timestamp: 1012,
                    ..Default::default()
                },
            ];
            match scenario {
                "late_origin" => origins[1].timestamp = 1025,
                "late_fork" => cfg.hardforks.jovian_time = Some(1024),
                "late_drift" => {
                    cfg.genesis.l2_time += 800;
                    for block in &mut span.batches {
                        block.timestamp += 800;
                    }
                    origins[0].timestamp = 20;
                    origins[1].timestamp = 21;
                }
                _ => {}
            }
            span.parent_check = parent_hash[..20].try_into().unwrap();
            span.l1_origin_check = origins[1].hash[..20].try_into().unwrap();
            let parent = L2BlockInfo {
                block_info: BlockInfo {
                    hash: parent_hash,
                    number: 9,
                    timestamp: cfg.genesis.l2_time + 18,
                    ..Default::default()
                },
                l1_origin: origins[0].id(),
                ..Default::default()
            };
            let result = span
                .check_batch_holocene(
                    &cfg,
                    &origins,
                    parent,
                    &origins[1],
                    &mut TestBatchValidator::default(),
                )
                .await;
            let expected = match scenario {
                "late_origin" => {
                    crate::BatchValidity::Drop(crate::BatchDropReason::TimestampBeforeL1Origin)
                }
                "late_fork" => {
                    crate::BatchValidity::Drop(crate::BatchDropReason::NonEmptyTransitionBlock)
                }
                "late_drift" => crate::BatchValidity::Drop(
                    crate::BatchDropReason::SequencerDriftNotAdoptedNextOrigin,
                ),
                _ => crate::BatchValidity::Accept,
            };
            assert_eq!(result, expected, "{scenario}");
        }
    }
}
