//! Pure, whole-range admission for the experimental public projection protocol.
//!
//! Admission computes the public statement of a span from its own derivation state
//! and consensus config, binds the claim's `l1Head`, `rollupConfigHash` and
//! `depSetHash` in every verifier mode, and checks the proof against the 672-byte
//! `PublicValuesV1` under `sp1-private-projection-v1`. The two older verifier IDs are
//! test-gated (§B.5 of the sound-profile spec) and prove nothing about execution.

use crate::{BatchValidationProvider, SpanBatch};
use alloc::{vec, vec::Vec};
use alloy_consensus::{TxEnvelope, transaction::SignerRecoverable};
use alloy_eips::{BlockNumHash, Decodable2718, Encodable2718};
use alloy_primitives::{Address, B256, Bytes, U256, address, keccak256};
use alloy_sol_types::{SolCall, sol};
use kona_genesis::{PrivateProjectionConfig, RollupConfig};
use op_alloy_consensus::OpBlock;

mod commit;
pub use commit::{
    MESSAGES_DOMAIN, MessageKind, OUTPUTS_DOMAIN, commitment_proof, commitment_root,
    export_message_hash, import_message_hash, message_leaf, output_leaf, verify_commitment_proof,
};
mod envelope;
pub use envelope::{
    ENVELOPE_VERSION, Envelope, EnvelopeKind, GROTH16_PROOF_LEN, MOCK_PROOF_LEN, check_mock_proof,
    decode_envelope, encode_envelope, mock_proof,
};
mod render;
pub use render::{
    RenderedLog, message_leaves, rendered_logs, rendered_message, renders, replay_calldata,
    sent_message_log,
};
mod statement;
pub use statement::{
    BN254_SCALAR_MODULUS, CONFIG_DOMAIN, DEP_SET_DOMAIN, DepSetChainId, PRIVATE_CONFIG_DOMAIN,
    PUBLIC_VALUES_LEN, PUBLIC_VALUES_MAGIC, PUBLIC_VALUES_WORDS, config_hash, dependency_set_hash,
    is_canonical_scalar, private_config_hash, public_values, public_values_digest,
};
#[cfg(feature = "sp1-projection-verifier")]
pub mod sp1;

const REGISTRY: Address = address!("420000000000000000000000000000000000002e");
const MESSENGER: Address = address!("4200000000000000000000000000000000000023");
const INBOX: Address = address!("4200000000000000000000000000000000000022");
const REPLAYER: Address = address!("420000000000000000000000000000000000002f");
const BRIDGE: Address = address!("4200000000000000000000000000000000000024");
const MAX_MESSAGE: usize = 1024 * 1024;
const MAX_PROOF: usize = 65536;

/// Explicitly insecure verifier ID: accepts any proof bytes. Test-gated.
pub const INSECURE_STUB: &str = "insecure-stub-v1";
/// Forgeable execution-mock verifier ID (admission digest only). Test-gated.
pub const EXECUTION_MOCK: &str = "execution-mock-v1";
/// The production verifier ID: SP1 Groth16 over the SP1 v6.1.0 circuit.
pub const SP1_PRIVATE_PROJECTION_V1: &str = "sp1-private-projection-v1";

/// Whether this build compiles in the test verifiers (Cargo feature
/// `private-projection-test-verifiers`).
pub const TEST_VERIFIERS_COMPILED: bool = cfg!(feature = "private-projection-test-verifiers");
/// Projection chain IDs on which test-gated verifier modes may run (devstack L2 A/B).
pub const TEST_CHAIN_IDS: [u64; 2] = [901, 902];

sol! {
    /// The existing range-claim wire tuple; proof policy is owned by admission.
    #[derive(Debug, PartialEq, Eq)]
    struct RangeClaim {
        uint8 version;
        uint64 firstBlock;
        uint64 lastBlock;
        bytes32 privateTerminalBlockHash;
        bytes32 privateTerminalParentHash;
        uint64 anchorBlock;
        bytes32 anchorOutputRoot;
        bytes32 recoveryHash;
        bytes32 parentOutputRoot;
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
    function recordOutput(bytes32 outputRoot);
    function postClaim(RangeClaim claim);
    function replaySentMessage(uint256 destination, uint256 nonce, address sender, address target, bytes message);
    function replayEvent(bytes32[] topics, bytes data);
    function validateMessage(Identifier identifier, bytes32 payloadHash);
    /// `L2ToL2CrossDomainMessenger` export event.
    event SentMessage(uint256 indexed destination, address indexed target, uint256 indexed messageNonce, address sender, bytes message);
    /// `CrossL2Inbox` import event.
    event ExecutingMessage(bytes32 indexed msgHash, Identifier id);
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
    /// Independently authenticated canonical continuation.
    pub continuation: Continuation,
    /// Operator-supplied checkpoint fields, with proof removed.
    pub claim: RangeClaim,
    /// [`config_hash`] of the node's own projection config (= `claim.rollupConfigHash`).
    pub projection_config_hash: B256,
    /// Consensus constant `private_projection.private_config_hash`.
    pub private_config_hash: B256,
    /// Commitment to the published per-block private output roots.
    pub outputs_root: B256,
    /// Commitment to the published replayed messages.
    pub messages_root: B256,
    /// Root of the last `recordOutput` of the span.
    pub terminal_output: B256,
}

/// Derivation-side inputs of [`validate_projection_range`]. The chain ID and genesis
/// come from the rollup config.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct ProjectionContext {
    /// Canonical parent of the span.
    pub parent_hash: B256,
    /// Hash of the node's own L1 block at the span's last epoch number.
    pub l1_head: B256,
    /// Canonical continuation collected by [`ContextCollector`].
    pub continuation: Continuation,
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

/// `execution-mock-v1`: accepts exactly [`execution_mock_proof`] of the statement.
#[derive(Debug, Default)]
pub struct ExecutionMockVerifier;
impl ProofVerifier for ExecutionMockVerifier {
    fn verify(&self, statement: &Statement, proof: &[u8]) -> Result<(), ProjectionError> {
        if proof == execution_mock_proof(statement).as_slice() {
            Ok(())
        } else {
            Err(ProjectionError("incorrect execution-mock proof envelope"))
        }
    }
}

/// `sp1-private-projection-v1`: the envelope's public values must equal
/// [`public_values`] of the statement, bytewise; Groth16 is checked against the pinned
/// circuit and `program_vkey`; mock envelopes only when `allow_mock`. Fails closed
/// without the `sp1-projection-verifier` feature.
#[derive(Debug)]
pub struct Sp1Verifier<'a> {
    /// The consensus profile (program vkey).
    pub profile: &'a PrivateProjectionConfig,
    /// `mock_proofs` and the §B.5 gate both hold.
    pub allow_mock: bool,
}
impl ProofVerifier for Sp1Verifier<'_> {
    #[cfg(not(feature = "sp1-projection-verifier"))]
    fn verify(&self, _: &Statement, _: &[u8]) -> Result<(), ProjectionError> {
        Err(ProjectionError("sp1 projection verifier not compiled"))
    }

    #[cfg(feature = "sp1-projection-verifier")]
    fn verify(&self, statement: &Statement, proof: &[u8]) -> Result<(), ProjectionError> {
        let env = decode_envelope(proof)?;
        if env.public_values != public_values(statement) {
            return Err(ProjectionError("public values do not match the statement"));
        }
        match env.kind {
            EnvelopeKind::Groth16 => sp1::verify_groth16(
                &sp1::circuit_v6_1_0(),
                &env.proof,
                self.profile.program_vkey,
                &env.public_values,
            ),
            EnvelopeKind::Mock if self.allow_mock => {
                check_mock_proof(&env.proof, self.profile.program_vkey, &env.public_values)
            }
            EnvelopeKind::Mock => Err(ProjectionError("mock proofs disabled")),
        }
    }
}

/// Consensus-selected verifier. No operator environment or proof-supplied mode switch.
/// Every call first applies [`check_config`] for the chain.
#[derive(Debug, Clone, Copy)]
pub struct ConfiguredVerifier<'a> {
    /// The projection chain's consensus profile.
    pub profile: &'a PrivateProjectionConfig,
    /// The projection chain ID (`l2_chain_id`), for the §B.5 gate.
    pub chain_id: u64,
}
impl<'a> ConfiguredVerifier<'a> {
    /// The verifier of `cfg.private_projection`.
    pub fn from_config(cfg: &'a RollupConfig) -> Result<Self, ProjectionError> {
        let profile =
            cfg.private_projection.as_ref().ok_or(ProjectionError("missing projection config"))?;
        Ok(Self { profile, chain_id: cfg.l2_chain_id.id() })
    }
}
impl ProofVerifier for ConfiguredVerifier<'_> {
    fn verify(&self, statement: &Statement, proof: &[u8]) -> Result<(), ProjectionError> {
        check_config(self.profile, self.chain_id)?;
        match self.profile.verifier.as_str() {
            INSECURE_STUB => StubVerifier.verify(statement, proof),
            EXECUTION_MOCK => ExecutionMockVerifier.verify(statement, proof),
            SP1_PRIVATE_PROJECTION_V1 => Sp1Verifier {
                profile: self.profile,
                allow_mock: self.profile.mock_proofs &&
                    gate_allows(TEST_VERIFIERS_COMPILED, self.chain_id),
            }
            .verify(statement, proof),
            _ => Err(ProjectionError("unsupported verifier")),
        }
    }
}

/// §B.5 gate: test-gated modes run only when compiled in and on an allowlisted chain.
pub const fn gate_allows(compiled: bool, chain_id: u64) -> bool {
    compiled && (chain_id == TEST_CHAIN_IDS[0] || chain_id == TEST_CHAIN_IDS[1])
}

/// `insecure-stub-v1`, `execution-mock-v1`, or `sp1-private-projection-v1` with mock proofs.
pub fn is_test_gated(p: &PrivateProjectionConfig) -> bool {
    matches!(p.verifier.as_str(), INSECURE_STUB | EXECUTION_MOCK) ||
        (p.verifier == SP1_PRIVATE_PROJECTION_V1 && p.mock_proofs)
}

/// Chain-independent field rules of §B.1 (Go `Config.Check`).
pub fn check_profile(p: &PrivateProjectionConfig) -> Result<(), ProjectionError> {
    if p.genesis_output_root.is_zero() || p.dependency_set_hash.is_zero() {
        return Err(ProjectionError("missing genesis output or dependency set hash"));
    }
    match p.verifier.as_str() {
        SP1_PRIVATE_PROJECTION_V1 => {
            if p.program_vkey.is_zero() || !is_canonical_scalar(&p.program_vkey.0) {
                return Err(ProjectionError("invalid program vkey"));
            }
            if p.private_config_hash.is_zero() {
                return Err(ProjectionError("missing private config hash"));
            }
            if p.allow_events {
                return Err(ProjectionError("events are unsupported by the sp1 profile"));
            }
        }
        INSECURE_STUB | EXECUTION_MOCK => {
            if !p.program_vkey.is_zero() || !p.private_config_hash.is_zero() || p.mock_proofs {
                return Err(ProjectionError("sp1 fields set on an insecure verifier"));
            }
        }
        _ => return Err(ProjectionError("unsupported verifier")),
    }
    Ok(())
}

/// [`check_profile`] plus the §B.5 gate with an explicit compile-gate value.
pub fn check_config_with_gate(
    p: &PrivateProjectionConfig,
    chain_id: u64,
    compiled: bool,
) -> Result<(), ProjectionError> {
    check_profile(p)?;
    if is_test_gated(p) && !gate_allows(compiled, chain_id) {
        return Err(ProjectionError("test-gated verifier on an ungated build or chain"));
    }
    Ok(())
}

/// [`check_profile`] plus the §B.5 gate of this build (Go `Config.CheckChain`).
pub fn check_config(p: &PrivateProjectionConfig, chain_id: u64) -> Result<(), ProjectionError> {
    check_config_with_gate(p, chain_id, TEST_VERIFIERS_COMPILED)
}

/// Commitment independently reconstructed by Go and Kona admission. The records
/// root includes all normalized claim fields, outputs, messages and envelopes.
pub fn admission_digest(s: &Statement) -> B256 {
    let mut data = b"optimism.private-admission.v1\0".to_vec();
    data.extend_from_slice(s.chain_id.as_slice());
    data.extend_from_slice(s.parent_hash.as_slice());
    data.extend_from_slice(s.projection_hash.as_slice());
    put(&mut data, s.continuation.anchor.number);
    data.extend_from_slice(s.continuation.anchor.hash.as_slice());
    data.extend_from_slice(s.continuation.output_root.as_slice());
    data.extend_from_slice(s.continuation.recovery_hash.as_slice());
    keccak256(data)
}

/// Forgeable mock envelope, emitted only after native execution by the honest
/// producer. This tests publication/admission plumbing, not execution soundness.
pub fn execution_mock_proof(s: &Statement) -> Vec<u8> {
    let mut out = b"optimism.private-execution.mock.v1\0".to_vec();
    out.extend_from_slice(admission_digest(s).as_slice());
    out
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

/// Canonical inbox access-list keys for a projected executing message.
pub fn import_keys(call: &validateMessageCall) -> Result<Vec<B256>, ProjectionError> {
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
    ctx: ProjectionContext,
    span: &SpanBatch,
    verifier: &impl ProofVerifier,
) -> Result<Statement, ProjectionError> {
    let ProjectionContext { parent_hash, l1_head, continuation } = ctx;
    let mode =
        cfg.private_projection.as_ref().ok_or(ProjectionError("missing projection config"))?;
    check_config(mode, cfg.l2_chain_id.id())?;
    if cfg.block_time == 0 || cfg.l2_chain_id.id() == 0 || l1_head.is_zero() {
        return Err(ProjectionError("unsupported projection config or context"));
    }
    let projection_config_hash = config_hash(cfg)?;
    let (first, last) = range_bounds(cfg, span)?;
    let start = span.batches[0].timestamp;
    if continuation.anchor.number >= first ||
        continuation.anchor.hash.is_zero() ||
        continuation.output_root.is_zero()
    {
        return Err(ProjectionError("invalid authenticated checkpoint"));
    }
    let mut leaves = Vec::new();
    let mut output_leaves = Vec::with_capacity(span.batches.len());
    let mut message_leaves = Vec::new();
    let mut terminal_output = B256::ZERO;
    let mut claim = None;
    let mut transcript = Vec::new();
    for (i, block) in span.batches.iter().enumerate() {
        if block.timestamp != start + (i as u64) * cfg.block_time {
            return Err(ProjectionError("noncontiguous timestamps"));
        }
        let number = first + i as u64;
        let block_start = transcript.len();
        put(&mut transcript, number);
        put(&mut transcript, block.timestamp);
        put(&mut transcript, block.epoch_num);
        if i == 0 && block.transactions.is_empty() {
            return Err(ProjectionError("missing claim"));
        }
        put(&mut transcript, block.transactions.len() as u64);
        let mut output_seen = false;
        let output_position = usize::from(i == 0);
        // Rendered index of the next replay: every replay emits exactly one log (§E).
        let mut replay_ordinal: u32 = 0;
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
                REGISTRY if data.starts_with(&recordOutputCall::SELECTOR) => {
                    let root = decode::<recordOutputCall>(&data)?.outputRoot;
                    if output_seen || j != output_position || root.is_zero() {
                        return Err(ProjectionError("duplicate, misplaced or empty output"));
                    }
                    output_seen = true;
                    output_leaves.push(output_leaf(number, root));
                    terminal_output = root;
                }
                REGISTRY => {
                    if i != 0 || j != 0 {
                        return Err(ProjectionError("duplicate or misplaced claim"));
                    }
                    if data.len() > 4 + 512 + MAX_PROOF {
                        return Err(ProjectionError("claim too large"));
                    }
                    let mut decoded = decode::<postClaimCall>(&data)?;
                    let c = &decoded.claim;
                    if c.version != 2 ||
                        c.firstBlock != first ||
                        c.lastBlock != last ||
                        c.proof.len() > MAX_PROOF
                    {
                        return Err(ProjectionError("claim framing"));
                    }
                    if c.anchorBlock != continuation.anchor.number ||
                        c.anchorOutputRoot != continuation.output_root ||
                        c.recoveryHash != continuation.recovery_hash ||
                        c.parentOutputRoot.is_zero()
                    {
                        return Err(ProjectionError("claim continuation mismatch"));
                    }
                    if continuation.anchor.number == first - 1 {
                        if continuation.anchor.hash != parent_hash ||
                            !continuation.recovery_hash.is_zero() ||
                            c.parentOutputRoot != continuation.output_root
                        {
                            return Err(ProjectionError("private parent checkpoint mismatch"));
                        }
                    } else if continuation.recovery_hash.is_zero() {
                        return Err(ProjectionError("missing recovery inputs"));
                    }
                    // §C.5: bound in every verifier mode, against the node's own view.
                    if c.l1Head != l1_head {
                        return Err(ProjectionError("claim l1 head mismatch"));
                    }
                    if c.rollupConfigHash != projection_config_hash {
                        return Err(ProjectionError("claim rollup config hash mismatch"));
                    }
                    if c.depSetHash != mode.dependency_set_hash {
                        return Err(ProjectionError("claim dependency set hash mismatch"));
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
                    message_leaves.push(message_leaf(
                        number,
                        replay_ordinal,
                        MessageKind::Init,
                        export_message_hash(&c),
                    ));
                    replay_ordinal += 1;
                }
                INBOX => {
                    if data.len() != 196 {
                        return Err(ProjectionError("import length"));
                    }
                    keys = Some(import_keys(&decode::<validateMessageCall>(&data)?)?);
                    let args: &[u8; 192] = data[4..].try_into().expect("length checked above");
                    message_leaves.push(message_leaf(
                        number,
                        replay_ordinal,
                        MessageKind::Exec,
                        import_message_hash(args),
                    ));
                    replay_ordinal += 1;
                }
                REPLAYER => {
                    if !mode.allow_events || data.len() > MAX_MESSAGE + 1024 {
                        return Err(ProjectionError("event disabled or too large"));
                    }
                    let c = decode::<replayEventCall>(&data)?;
                    if c.topics.len() > 4 || c.data.len() > MAX_MESSAGE {
                        return Err(ProjectionError("event bounds"));
                    }
                    // Kind 0x03 is undefined in v1 (`allow_events` is false under sp1):
                    // an event replay takes a rendered index but has no message leaf.
                    replay_ordinal += 1;
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
        if !output_seen {
            return Err(ProjectionError("missing private output"));
        }
        let mut leaf = vec![0];
        leaf.extend_from_slice(&transcript[block_start..]);
        leaves.push(keccak256(leaf));
    }
    let mut claim = claim.ok_or(ProjectionError("missing claim"))?;
    let proof = core::mem::take(&mut claim.proof);
    let statement = Statement {
        chain_id: B256::from(U256::from(cfg.l2_chain_id.id()).to_be_bytes::<32>()),
        parent_hash,
        projection_hash: records_root(leaves),
        continuation,
        claim,
        projection_config_hash,
        private_config_hash: mode.private_config_hash,
        outputs_root: commitment_root(OUTPUTS_DOMAIN, &output_leaves),
        messages_root: commitment_root(MESSAGES_DOMAIN, &message_leaves),
        terminal_output,
    };
    verifier.verify(&statement, &proof)?;
    Ok(statement)
}

/// Canonical private checkpoint and the exact intervening public recovery inputs.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct Continuation {
    /// Public block carrying the surviving commitment.
    pub anchor: BlockNumHash,
    /// Private output-v0 commitment at the anchor.
    pub output_root: B256,
    /// Reverse-ordered, constant-space commitment to canonical recovery inputs.
    pub recovery_hash: B256,
}

/// Canonical context collection outcome; availability never means invalidity.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ContextError {
    /// Retry the retained candidate without consuming more batch data.
    Unavailable,
    /// Canonical context violates the projection protocol.
    Invalid,
}

#[derive(Debug, Clone)]
struct ContextScan {
    parent: BlockNumHash,
    genesis: BlockNumHash,
    genesis_output: B256,
    cursor: BlockNumHash,
    result: Continuation,
    complete: bool,
}

/// RPC collection, deliberately separate from pure span validation. Retains one
/// constant-size cursor, processes at most 128 blocks per attempt, and pins all
/// traversed history to the original canonical span parent. Reset on reorg.
#[derive(Debug, Clone, Default)]
pub struct ContextCollector {
    scan: Option<ContextScan>,
}

impl ContextCollector {
    /// Empty context collector.
    pub const fn new() -> Self {
        Self { scan: None }
    }

    /// Forget all history when derivation resets or flushes a candidate.
    pub const fn reset(&mut self) {
        self.scan = None;
    }

    /// Collect canonical context without imposing a maximum outage duration.
    pub async fn resolve<BV: BatchValidationProvider>(
        &mut self,
        fetcher: &mut BV,
        parent: BlockNumHash,
        genesis: BlockNumHash,
        genesis_output: B256,
    ) -> Result<Continuation, ContextError> {
        if parent.number < genesis.number || genesis_output.is_zero() {
            return Err(ContextError::Invalid);
        }
        if self.scan.as_ref().is_none_or(|s| {
            s.parent != parent || s.genesis != genesis || s.genesis_output != genesis_output
        }) {
            self.scan = Some(ContextScan {
                parent,
                genesis,
                genesis_output,
                cursor: parent,
                result: Continuation::default(),
                complete: false,
            });
        }
        if parent != genesis {
            let block = fetcher
                .block_by_number(parent.number)
                .await
                .map_err(|_| ContextError::Unavailable)?;
            if block.header.number != parent.number || block.header.hash_slow() != parent.hash {
                self.reset();
                return Err(ContextError::Unavailable);
            }
        }
        let scan = self.scan.as_mut().expect("initialized above");
        if scan.complete {
            return Ok(scan.result);
        }
        for _ in 0..128 {
            if scan.cursor.number == genesis.number {
                if scan.cursor != genesis {
                    return Err(ContextError::Invalid);
                }
                scan.result.anchor = genesis;
                scan.result.output_root = genesis_output;
                scan.complete = true;
                return Ok(scan.result);
            }
            let block = fetcher
                .block_by_number(scan.cursor.number)
                .await
                .map_err(|_| ContextError::Unavailable)?;
            if block.header.number != scan.cursor.number ||
                block.header.hash_slow() != scan.cursor.hash
            {
                self.reset();
                return Err(ContextError::Unavailable);
            }
            let txs: Vec<Bytes> =
                block.body.transactions.iter().map(|tx| tx.encoded_2718().into()).collect();
            let root = canonical_output(&txs).map_err(|_| ContextError::Invalid)?;
            if !root.is_zero() {
                scan.result.anchor = scan.cursor;
                scan.result.output_root = root;
                scan.complete = true;
                return Ok(scan.result);
            }
            scan.result.recovery_hash = recovery_step(scan.result.recovery_hash, &block, &txs);
            scan.cursor =
                BlockNumHash { hash: block.header.parent_hash, number: scan.cursor.number - 1 };
        }
        Err(ContextError::Unavailable)
    }
}

/// Interpret admitted canonical records. Deposits and `PostExec` cannot create
/// checkpoints. A nonempty sequencer block without a record is not fallback.
pub fn canonical_output(txs: &[Bytes]) -> Result<B256, ProjectionError> {
    let mut root = B256::ZERO;
    let mut position = 0;
    let mut output_position = 0;
    for raw in txs {
        let ty = raw.first().ok_or(ProjectionError("empty canonical transaction"))?;
        if *ty == 0x7e || *ty == 0x7d {
            continue;
        }
        let envelope = TxEnvelope::decode_2718_exact(raw)
            .map_err(|_| ProjectionError("canonical transaction"))?;
        use alloy_consensus::Transaction;
        if envelope.to() == Some(REGISTRY) {
            let data = envelope.input();
            if data.starts_with(&recordOutputCall::SELECTOR) {
                if position != output_position || !root.is_zero() {
                    return Err(ProjectionError("canonical output placement"));
                }
                root = decode::<recordOutputCall>(data)?.outputRoot;
                if root.is_zero() {
                    return Err(ProjectionError("empty canonical output"));
                }
            } else {
                if position != 0 {
                    return Err(ProjectionError("canonical claim placement"));
                }
                if decode::<postClaimCall>(data)?.claim.version != 2 {
                    return Err(ProjectionError("canonical claim version"));
                }
                output_position = 1;
            }
        }
        position += 1;
    }
    if position != 0 && root.is_zero() {
        return Err(ProjectionError("missing canonical output"));
    }
    Ok(root)
}

/// Commit exact fallback inputs in descending block order, matching op-node.
pub fn recovery_step(previous: B256, block: &OpBlock, txs: &[Bytes]) -> B256 {
    let mut h = alloy_primitives::Keccak256::new();
    h.update(b"optimism.private-recovery.v1\0");
    h.update(previous);
    h.update(block.header.hash_slow());
    h.update(block.header.parent_hash);
    h.update(block.header.number.to_be_bytes());
    h.update(block.header.timestamp.to_be_bytes());
    h.update((txs.len() as u64).to_be_bytes());
    for tx in txs {
        h.update((tx.len() as u64).to_be_bytes());
        h.update(tx);
    }
    h.finalize()
}

/// Checkpoint-only calls count as empty for sequencer-drift scheduling, after
/// whole-span admission validates their envelopes and placement. Replays do not.
pub fn metadata_only(txs: &[Bytes]) -> bool {
    txs.iter().all(|raw| {
        let Ok(TxEnvelope::Eip1559(signed)) = TxEnvelope::decode_2718_exact(raw) else {
            return false;
        };
        let tx = signed.tx();
        if tx.to != alloy_primitives::TxKind::Call(REGISTRY) {
            return false;
        }
        if let Ok(call) = decode::<recordOutputCall>(&tx.input) {
            return !call.outputRoot.is_zero();
        }
        decode::<postClaimCall>(&tx.input).is_ok_and(|call| {
            call.claim.version == 2 &&
                call.claim.lastBlock >= call.claim.firstBlock &&
                call.claim.proof.len() <= MAX_PROOF
        })
    })
}

/// Count-bound, domain-separated Merkle root of ordered canonical block leaves.
pub fn records_root(mut level: Vec<B256>) -> B256 {
    let count = level.len() as u64;
    if level.is_empty() {
        return B256::ZERO;
    }
    while level.len() > 1 {
        level = level
            .chunks(2)
            .map(|pair| {
                let mut node = vec![1];
                node.extend_from_slice(pair[0].as_slice());
                node.extend_from_slice(pair.get(1).unwrap_or(&pair[0]).as_slice());
                keccak256(node)
            })
            .collect();
    }
    let mut root = b"optimism.private-projection.v2\0".to_vec();
    root.extend_from_slice(&count.to_be_bytes());
    root.extend_from_slice(level[0].as_slice());
    keccak256(root)
}

/// Validate range geometry before canonical parent lookup (including underflow).
pub fn range_bounds(cfg: &RollupConfig, span: &SpanBatch) -> Result<(u64, u64), ProjectionError> {
    let count = span.batches.len();
    if count == 0 || count > 65536 || cfg.block_time == 0 {
        return Err(ProjectionError("range length or block time"));
    }
    let start = span.batches[0].timestamp;
    let elapsed =
        start.checked_sub(cfg.genesis.l2_time).ok_or(ProjectionError("before genesis"))?;
    if elapsed == 0 || elapsed % cfg.block_time != 0 {
        return Err(ProjectionError("range alignment"));
    }
    let first = cfg
        .genesis
        .l2
        .number
        .checked_add(elapsed / cfg.block_time)
        .ok_or(ProjectionError("height overflow"))?;
    let last = first.checked_add((count - 1) as u64).ok_or(ProjectionError("height overflow"))?;
    for (i, block) in span.batches.iter().enumerate() {
        let offset =
            (i as u64).checked_mul(cfg.block_time).ok_or(ProjectionError("time overflow"))?;
        if start.checked_add(offset) != Some(block.timestamp) {
            return Err(ProjectionError("timestamp alignment"));
        }
    }
    Ok((first, last))
}

#[cfg(test)]
mod tests;
