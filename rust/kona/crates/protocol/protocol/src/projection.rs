//! Pure, whole-range admission for the experimental public projection protocol.
//! No execution correctness is established by the explicitly insecure stub verifier.

use crate::{BatchValidationProvider, SpanBatch};
use alloc::{vec, vec::Vec};
use alloy_consensus::{TxEnvelope, transaction::SignerRecoverable};
use alloy_eips::{BlockNumHash, Decodable2718, Encodable2718};
use alloy_primitives::{Address, B256, Bytes, U256, address, keccak256};
use alloy_sol_types::{SolCall, sol};
use kona_genesis::RollupConfig;
use op_alloy_consensus::OpBlock;

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
    continuation: Continuation,
    span: &SpanBatch,
    verifier: &impl ProofVerifier,
) -> Result<Statement, ProjectionError> {
    let mode =
        cfg.private_projection.as_ref().ok_or(ProjectionError("missing projection config"))?;
    if mode.verifier != "insecure-stub-v1" ||
        mode.genesis_output_root.is_zero() ||
        cfg.block_time == 0 ||
        cfg.l2_chain_id.id() == 0
    {
        return Err(ProjectionError("unsupported projection config"));
    }
    let (first, last) = range_bounds(cfg, span)?;
    let start = span.batches[0].timestamp;
    if continuation.anchor.number >= first ||
        continuation.anchor.hash.is_zero() ||
        continuation.output_root.is_zero()
    {
        return Err(ProjectionError("invalid authenticated checkpoint"));
    }
    let mut leaves = Vec::new();
    let mut claim = None;
    let mut transcript = Vec::new();
    for (i, block) in span.batches.iter().enumerate() {
        if block.timestamp != start + (i as u64) * cfg.block_time {
            return Err(ProjectionError("noncontiguous timestamps"));
        }
        let block_start = transcript.len();
        put(&mut transcript, first + i as u64);
        put(&mut transcript, block.timestamp);
        put(&mut transcript, block.epoch_num);
        if i == 0 && block.transactions.is_empty() {
            return Err(ProjectionError("missing claim"));
        }
        put(&mut transcript, block.transactions.len() as u64);
        let mut output_seen = false;
        let output_position = usize::from(i == 0);
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
                    if output_seen ||
                        j != output_position ||
                        decode::<recordOutputCall>(&data)?.outputRoot.is_zero()
                    {
                        return Err(ProjectionError("duplicate, misplaced or empty output"));
                    }
                    output_seen = true;
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
                decode::<postClaimCall>(data)?;
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
mod tests {
    use super::*;
    use crate::SpanBatchElement;
    use kona_genesis::PrivateProjectionConfig;
    use serde_json::Value;

    #[test]
    fn shared_recovery_transcript() {
        use alloy_rlp::Decodable;
        let v: Value = serde_json::from_str(include_str!(
            "../../../../../../op-private-interop/projection/testdata/recovery.json"
        ))
        .unwrap();
        let mut hash = B256::ZERO;
        for b in v["blocks"].as_array().unwrap().iter().rev() {
            let raw: Bytes = serde_json::from_value(b["header"].clone()).unwrap();
            let header = alloy_consensus::Header::decode(&mut raw.as_ref()).unwrap();
            let txs: Vec<Bytes> = serde_json::from_value(b["transactions"].clone()).unwrap();
            assert!(canonical_output(&txs).unwrap().is_zero());
            hash = recovery_step(hash, &OpBlock { header, body: Default::default() }, &txs);
        }
        assert_eq!(hash, serde_json::from_value::<B256>(v["recovery_hash"].clone()).unwrap());
    }

    #[tokio::test]
    async fn continuation_retries_long_outage_and_reorgs() {
        use crate::test_utils::TestBatchValidator;
        let genesis = BlockNumHash { number: 0, hash: B256::repeat_byte(8) };
        let output = B256::repeat_byte(9);
        let mut blocks = Vec::new();
        let mut hash = genesis.hash;
        for n in 1..=400 {
            let block = OpBlock {
                header: alloy_consensus::Header {
                    number: n,
                    parent_hash: hash,
                    timestamp: 1000 + 2 * n,
                    ..Default::default()
                },
                body: Default::default(),
            };
            hash = block.header.hash_slow();
            blocks.push(block);
        }
        let parent = BlockNumHash { number: 400, hash };
        let mut fetcher = TestBatchValidator { op_blocks: blocks, ..Default::default() };
        let mut collector = ContextCollector::new();
        for _ in 0..3 {
            assert_eq!(
                collector.resolve(&mut fetcher, parent, genesis, output).await,
                Err(ContextError::Unavailable)
            );
        }
        let missing = fetcher.op_blocks.remove(4);
        assert_eq!(
            collector.resolve(&mut fetcher, parent, genesis, output).await,
            Err(ContextError::Unavailable)
        );
        fetcher.op_blocks.insert(4, missing);
        let got = collector.resolve(&mut fetcher, parent, genesis, output).await.unwrap();
        assert_eq!(got.anchor, genesis);
        assert_eq!(got.output_root, output);
        let mut expected = B256::ZERO;
        for block in fetcher.op_blocks.iter().rev() {
            expected = recovery_step(expected, block, &[]);
        }
        assert_eq!(got.recovery_hash, expected);
        collector.reset();
        assert_eq!(
            collector.resolve(&mut fetcher, parent, genesis, output).await,
            Err(ContextError::Unavailable)
        );
        // A mid-scan reorg invalidates the cursor even if the caller is still
        // supplying its old parent. Once updated it must start a fresh scan.
        fetcher.op_blocks[399].header.timestamp += 1;
        assert_eq!(
            collector.resolve(&mut fetcher, parent, genesis, output).await,
            Err(ContextError::Unavailable)
        );
        let new_parent =
            BlockNumHash { number: 400, hash: fetcher.op_blocks[399].header.hash_slow() };
        for _ in 0..3 {
            assert_eq!(
                collector.resolve(&mut fetcher, new_parent, genesis, output).await,
                Err(ContextError::Unavailable)
            );
        }
        assert!(collector.resolve(&mut fetcher, new_parent, genesis, output).await.is_ok());
        // Completed results are also pinned to the caller's canonical parent.
        assert_eq!(
            collector.resolve(&mut fetcher, parent, genesis, output).await,
            Err(ContextError::Unavailable)
        );
    }

    #[test]
    fn metadata_preserves_drift_scheduling() {
        use crate::{BatchDropReason, BatchValidity, BlockInfo, L2BlockInfo, SingleBatch};
        let (_, span, _) = inputs(&vectors()[0]);
        let output = span.batches[0].transactions[1].clone();
        assert!(metadata_only(core::slice::from_ref(&output)));
        assert!(canonical_output(core::slice::from_ref(&output)).is_ok());
        assert!(canonical_output(&[output.clone(), output.clone()]).is_err());
        assert!(!metadata_only(&[Bytes::from_static(&[0x7e])]));
        for scenario in [
            "late L1",
            "next origin available",
            "missing origin",
            "with replay",
            "ordinary chain",
            "activation",
        ] {
            let mut cfg = inputs(&vectors()[0]).0;
            cfg.seq_window_size = 100;
            cfg.max_sequencer_drift = 1;
            cfg.hardforks.holocene_time = Some(0);
            let mut origins = vec![
                BlockInfo {
                    hash: B256::with_last_byte(5),
                    number: 5,
                    timestamp: 1000,
                    ..Default::default()
                },
                BlockInfo { number: 6, timestamp: 3006, ..Default::default() },
            ];
            let parent = L2BlockInfo {
                block_info: BlockInfo {
                    number: 1,
                    hash: B256::with_last_byte(1),
                    timestamp: 3002,
                    ..Default::default()
                },
                l1_origin: origins[0].id(),
                ..Default::default()
            };
            let mut batch = SingleBatch {
                parent_hash: parent.block_info.hash,
                timestamp: 3004,
                epoch_num: 5,
                epoch_hash: origins[0].hash,
                transactions: vec![output.clone()],
            };
            let want = match scenario {
                "next origin available" => {
                    origins[1].timestamp = batch.timestamp;
                    BatchValidity::Drop(BatchDropReason::SequencerDriftNotAdoptedNextOrigin)
                }
                "missing origin" => {
                    origins.truncate(1);
                    BatchValidity::Undecided
                }
                "with replay" => {
                    batch.transactions.push(span.batches[2].transactions[1].clone());
                    BatchValidity::Drop(BatchDropReason::SequencerDriftExceeded)
                }
                "ordinary chain" => {
                    cfg.private_projection = None;
                    BatchValidity::Drop(BatchDropReason::SequencerDriftExceeded)
                }
                "activation" => {
                    cfg.hardforks.jovian_time = Some(3004);
                    BatchValidity::Drop(BatchDropReason::NonEmptyTransitionBlock)
                }
                _ => BatchValidity::Accept,
            };
            assert_eq!(
                batch.check_batch(
                    &cfg,
                    &origins,
                    parent,
                    &BlockInfo { number: 6, ..Default::default() }
                ),
                want,
                "{scenario}"
            );
        }
    }

    fn test_continuation(parent: B256) -> Continuation {
        let mut root = B256::ZERO;
        root[0] = 9;
        Continuation {
            anchor: BlockNumHash { number: 9, hash: parent },
            output_root: root,
            recovery_hash: B256::ZERO,
        }
    }
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
                genesis_output_root: serde_json::from_value(
                    v["config"]["genesis_output_root"].clone(),
                )
                .unwrap(),
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
            let first = validate_projection_range(
                &cfg,
                parent,
                test_continuation(parent),
                &span,
                &StubVerifier,
            );
            assert_eq!(first.is_ok(), v["accept"].as_bool().unwrap(), "{}: {first:?}", v["name"]);
            if let Ok(statement) = &first {
                let expected: B256 = serde_json::from_value(v["digest"].clone()).unwrap();
                assert_eq!(statement.projection_hash, expected, "{}", v["name"]);
            }
            assert_eq!(
                first,
                validate_projection_range(
                    &cfg,
                    parent,
                    test_continuation(parent),
                    &span,
                    &StubVerifier
                )
            );
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
        let statement = validate_projection_range(
            &cfg,
            parent,
            test_continuation(parent),
            &span,
            &StubVerifier,
        )
        .unwrap();
        assert!(statement.claim.proof.is_empty());
        assert_eq!(
            validate_projection_range(&cfg, parent, test_continuation(parent), &span, &Reject),
            Err(ProjectionError("proof rejected"))
        );
        let verifier = Binding(statement);
        assert!(
            validate_projection_range(&cfg, parent, test_continuation(parent), &span, &verifier)
                .is_ok()
        );
        span.batches[2].transactions.truncate(1);
        assert_eq!(
            validate_projection_range(&cfg, parent, test_continuation(parent), &span, &verifier),
            Err(ProjectionError("wrong statement or proof"))
        );
    }
    #[tokio::test]
    async fn projection_admission_preflights_late_schedule_errors() {
        use crate::{BlockInfo, L2BlockInfo, test_utils::TestBatchValidator};
        use kona_genesis::HardForkConfig;
        for scenario in ["valid", "late_origin", "late_fork", "late_drift"] {
            let (mut cfg, mut span, _) = inputs(&vectors()[0]);
            let mut root = B256::ZERO;
            root[0] = 9;
            let checkpoint = OpBlock {
                header: alloy_consensus::Header { number: 9, ..Default::default() },
                body: alloy_consensus::BlockBody {
                    transactions: vec![op_alloy_consensus::OpTxEnvelope::Eip1559(
                        alloy_consensus::Signed::new_unchecked(
                            alloy_consensus::TxEip1559 {
                                to: alloy_primitives::TxKind::Call(REGISTRY),
                                input: recordOutputCall { outputRoot: root }.abi_encode().into(),
                                ..Default::default()
                            },
                            alloy_primitives::Signature::test_signature(),
                            B256::ZERO,
                        ),
                    )],
                    ..Default::default()
                },
            };
            let parent_hash = checkpoint.header.hash_slow();
            let mut fetcher =
                TestBatchValidator { op_blocks: vec![checkpoint], ..Default::default() };
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
            let result =
                span.check_batch_holocene(&cfg, &origins, parent, &origins[1], &mut fetcher).await;
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
