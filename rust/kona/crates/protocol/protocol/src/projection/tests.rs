use super::*;
use crate::SpanBatchElement;
use alloy_consensus::Transaction;
use alloy_primitives::{Log, LogData, b256};
use alloy_sol_types::SolEvent;
use kona_genesis::PrivateProjectionConfig;
use serde_json::Value;
use sha2::Digest;

const RANGES: &str =
    include_str!("../../../../../../../op-private-interop/projection/testdata/ranges.json");
const PROOFS: &str =
    include_str!("../../../../../../../op-private-interop/projection/testdata/proofs.json");
const RECOVERY: &str =
    include_str!("../../../../../../../op-private-interop/projection/testdata/recovery.json");

fn h(v: &Value) -> B256 {
    serde_json::from_value(v.clone()).unwrap()
}
fn hb(v: &Value) -> Bytes {
    serde_json::from_value(v.clone()).unwrap()
}
fn hash_or_zero(v: &Value) -> B256 {
    if v.is_null() { B256::ZERO } else { h(v) }
}

fn vectors() -> Vec<Value> {
    serde_json::from_str(RANGES).unwrap()
}

/// Rollup config, span and derivation context of a schema-v2 vector (§F.5).
fn inputs(v: &Value) -> (RollupConfig, SpanBatch, ProjectionContext) {
    let c = &v["config"];
    let x = &v["context"];
    let cfg = RollupConfig {
        l2_chain_id: x["chain_id"].as_u64().unwrap_or(901).into(),
        block_time: x["block_time"].as_u64().unwrap_or(2),
        genesis: kona_genesis::ChainGenesis {
            l2_time: x["genesis_time"].as_u64().unwrap_or(1000),
            l2: BlockNumHash {
                number: x["genesis_number"].as_u64().unwrap_or(0),
                hash: hash_or_zero(&x["genesis_hash"]),
            },
            ..Default::default()
        },
        private_projection: Some(PrivateProjectionConfig {
            verifier: c["verifier"].as_str().unwrap().into(),
            allow_events: c["allow_events"].as_bool().unwrap_or(false),
            genesis_output_root: hash_or_zero(&c["genesis_output_root"]),
            program_vkey: hash_or_zero(&c["program_vkey"]),
            private_config_hash: hash_or_zero(&c["private_config_hash"]),
            dependency_set_hash: hash_or_zero(&c["dependency_set_hash"]),
            mock_proofs: c["mock_proofs"].as_bool().unwrap_or(false),
        }),
        ..Default::default()
    };
    let span = SpanBatch {
        batches: v["blocks"]
            .as_array()
            .map(Vec::as_slice)
            .unwrap_or_default()
            .iter()
            .map(|b| SpanBatchElement {
                timestamp: b["timestamp"].as_u64().unwrap(),
                epoch_num: b["epoch"].as_u64().unwrap(),
                transactions: b["transactions"]
                    .as_array()
                    .map(|xs| xs.iter().map(hb).collect())
                    .unwrap_or_default(),
            })
            .collect(),
        ..Default::default()
    };
    let k = &x["continuation"];
    let ctx = ProjectionContext {
        parent_hash: h(&x["parent_hash"]),
        l1_head: hash_or_zero(&x["l1_head"]),
        continuation: Continuation {
            anchor: BlockNumHash {
                number: k["anchor"]["number"].as_u64().unwrap(),
                hash: h(&k["anchor"]["hash"]),
            },
            output_root: h(&k["output_root"]),
            recovery_hash: hash_or_zero(&k["recovery_hash"]),
        },
    };
    (cfg, span, ctx)
}

fn assert_statement(v: &Value, s: &Statement) {
    let name = &v["name"];
    assert_eq!(s.projection_hash, h(&v["digest"]), "{name}: records root");
    assert_eq!(s.outputs_root, h(&v["outputs_root"]), "{name}: outputs root");
    assert_eq!(s.messages_root, h(&v["messages_root"]), "{name}: messages root");
    assert_eq!(s.terminal_output, h(&v["terminal_output"]), "{name}: terminal output");
    assert_eq!(s.projection_config_hash, h(&v["config_hash"]), "{name}: config hash");
    let pv = hb(&v["public_values"]);
    let ours = public_values(s);
    for (w, (a, b)) in ours.chunks(32).zip(pv.chunks(32)).enumerate() {
        assert_eq!(a, b, "{name}: public values word {w}");
    }
    assert_eq!(ours.as_slice(), pv.as_ref(), "{name}: public values");
}

fn claim_proof(span: &SpanBatch) -> Bytes {
    let tx = TxEnvelope::decode_2718_exact(&span.batches[0].transactions[0]).unwrap();
    postClaimCall::abi_decode(tx.input()).unwrap().claim.proof
}

/// Edit the span's claim and re-sign it with the vector key.
fn set_claim(span: &mut SpanBatch, f: impl FnOnce(&mut RangeClaim)) {
    let raw = &mut span.batches[0].transactions[0];
    *raw = crate::test_utils::resign_projection_claim(
        raw,
        crate::test_utils::PROJECTION_VECTOR_KEY,
        f,
    );
}

/// Replace the claim's proof bytes. The records root is unchanged: the claim is
/// normalised without its proof and the sender is the same key.
fn set_claim_proof(span: &mut SpanBatch, proof: Vec<u8>) {
    set_claim(span, |c| c.proof = proof.into());
}

#[test]
fn shared_recovery_transcript() {
    use alloy_rlp::Decodable;
    let v: Value = serde_json::from_str(RECOVERY).unwrap();
    let mut hash = B256::ZERO;
    for b in v["blocks"].as_array().unwrap().iter().rev() {
        let raw = hb(&b["header"]);
        let header = alloy_consensus::Header::decode(&mut raw.as_ref()).unwrap();
        let txs: Vec<Bytes> = serde_json::from_value(b["transactions"].clone()).unwrap();
        assert!(canonical_output(&txs).unwrap().is_zero());
        hash = recovery_step(hash, &OpBlock { header, body: Default::default() }, &txs);
    }
    assert_eq!(hash, h(&v["recovery_hash"]));
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
    let new_parent = BlockNumHash { number: 400, hash: fetcher.op_blocks[399].header.hash_slow() };
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

#[test]
fn shared_projection_vectors_and_purity() {
    let all = vectors();
    for name in [
        "wrong_l1_head",
        "wrong_rollup_config_hash",
        "wrong_dep_set_hash",
        "recovery_accept",
        "recovery_missing_inputs",
        "recovery_wrong_anchor_root",
        "retired_stub_ungated",
        "sp1_allow_events_rejected",
        "wrong_parent_output",
        "late_gas_below_intrinsic",
        "late_gas_below_floor",
        "gas_at_minimum",
    ] {
        assert!(all.iter().any(|v| v["name"] == name), "missing range vector {name}");
    }
    // The §E.1 gas rule rejects with the same reason as Go.
    for name in ["late_gas_below_intrinsic", "late_gas_below_floor"] {
        let v = all.iter().find(|v| v["name"] == name).unwrap();
        let (cfg, span, ctx) = inputs(v);
        assert_eq!(
            validate_projection_range(&cfg, ctx, &span, &StubVerifier),
            Err(ProjectionError("transaction gas below intrinsic or calldata floor")),
            "{name}"
        );
    }
    for v in all {
        let (cfg, span, ctx) = inputs(&v);
        let (before_cfg, before_span) = (cfg.clone(), span.clone());
        let first = validate_projection_range(&cfg, ctx, &span, &StubVerifier);
        assert_eq!(first.is_ok(), v["accept"].as_bool().unwrap(), "{}: {first:?}", v["name"]);
        if let Ok(statement) = &first {
            assert_statement(&v, statement);
        }
        assert_eq!(first, validate_projection_range(&cfg, ctx, &span, &StubVerifier));
        assert_eq!(cfg, before_cfg);
        assert_eq!(span, before_span);
    }
}

#[test]
fn shared_proof_vectors() {
    let cases: Vec<Value> = serde_json::from_str(PROOFS).unwrap();
    let mut names: Vec<String> = [
        "sp1_mock_valid",
        "sp1_mock_disabled",
        "sp1_mock_ungated_chain",
        "sp1_mock_wrong_vkey",
        "sp1_mock_wrong_digest",
        "sp1_mock_nonzero_exit",
        "sp1_bad_version",
        "sp1_bad_kind",
        "sp1_trailing_byte",
        "sp1_truncated",
        "sp1_groth16_len_355",
        "execution_mock_valid",
    ]
    .map(String::from)
    .to_vec();
    names.extend((0..PUBLIC_VALUES_WORDS).map(|k| format!("sp1_pv_word_{k}_flipped")));
    for name in &names {
        assert!(cases.iter().any(|v| v["name"] == name.as_str()), "missing proof vector {name}");
    }
    for v in cases {
        let (cfg, span, ctx) = inputs(&v);
        let verifier = ConfiguredVerifier::from_config(&cfg).unwrap();
        let result = validate_projection_range(&cfg, ctx, &span, &verifier);
        assert_eq!(result.is_ok(), v["accept"].as_bool().unwrap(), "{}: {result:?}", v["name"]);
        if let Ok(statement) = &result {
            assert_statement(&v, statement);
        }
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
    let (cfg, mut span, ctx) = inputs(&vectors()[0]);
    let statement = validate_projection_range(&cfg, ctx, &span, &StubVerifier).unwrap();
    assert!(statement.claim.proof.is_empty());
    assert_eq!(
        validate_projection_range(&cfg, ctx, &span, &Reject),
        Err(ProjectionError("proof rejected"))
    );
    // The records root does not depend on the proof bytes.
    let mut reproved = span.clone();
    set_claim_proof(&mut reproved, b"dummy".to_vec());
    assert_eq!(validate_projection_range(&cfg, ctx, &reproved, &StubVerifier).unwrap(), statement);
    assert_ne!(claim_proof(&span), claim_proof(&reproved));
    span = reproved;
    let verifier = Binding(statement);
    assert!(validate_projection_range(&cfg, ctx, &span, &verifier).is_ok());
    span.batches[2].transactions.truncate(1);
    assert_eq!(
        validate_projection_range(&cfg, ctx, &span, &verifier),
        Err(ProjectionError("wrong statement or proof"))
    );
}

#[test]
fn claim_bindings_hold_in_every_mode() {
    let (cfg, span, ctx) = inputs(&vectors()[0]);
    validate_projection_range(&cfg, ctx, &span, &StubVerifier).unwrap();
    let mut wrong = ctx;
    wrong.l1_head = B256::repeat_byte(0xee);
    assert_eq!(
        validate_projection_range(&cfg, wrong, &span, &StubVerifier),
        Err(ProjectionError("claim l1 head mismatch"))
    );
    wrong.l1_head = B256::ZERO;
    assert!(validate_projection_range(&cfg, wrong, &span, &StubVerifier).is_err());
    let mut other = cfg.clone();
    other.block_time = 1;
    other.genesis.l2_time = 1000;
    assert!(validate_projection_range(&other, ctx, &span, &StubVerifier).is_err());
    let mut other = cfg.clone();
    other.genesis.l2.hash = B256::repeat_byte(0x42);
    assert_eq!(
        validate_projection_range(&other, ctx, &span, &StubVerifier),
        Err(ProjectionError("claim rollup config hash mismatch"))
    );
    let mut other = cfg;
    other.private_projection.as_mut().unwrap().dependency_set_hash = B256::repeat_byte(0x42);
    assert_eq!(
        validate_projection_range(&other, ctx, &span, &StubVerifier),
        Err(ProjectionError("claim rollup config hash mismatch"))
    );
}

#[tokio::test]
async fn projection_admission_preflights_late_schedule_errors() {
    use crate::{
        BatchDropReason, BatchValidity, BlockInfo, L2BlockInfo, test_utils::TestBatchValidator,
    };
    use kona_genesis::HardForkConfig;
    let proofs: Vec<Value> = serde_json::from_str(PROOFS).unwrap();
    let sp1 = proofs.iter().find(|v| v["name"] == "sp1_mock_valid").unwrap().clone();
    for (v, scenario) in [
        (&vectors()[0], "valid"),
        (&vectors()[0], "late_origin"),
        (&vectors()[0], "late_fork"),
        (&vectors()[0], "late_drift"),
        (&vectors()[0], "wrong_l1_head"),
        (&vectors()[0], "forged_proof"),
        (&sp1, "valid"),
        (&sp1, "wrong_l1_head"),
        (&sp1, "forged_proof"),
    ] {
        let (mut cfg, mut span, ctx) = inputs(v);
        if scenario == "late_drift" {
            // Shift the chain 800s later and re-bind the claim to the shifted config.
            cfg.genesis.l2_time += 800;
            for block in &mut span.batches {
                block.timestamp += 800;
            }
            let hash = config_hash(&cfg).unwrap();
            set_claim(&mut span, |c| c.rollupConfigHash = hash);
        }
        let checkpoint = OpBlock {
            header: alloy_consensus::Header { number: 9, ..Default::default() },
            body: alloy_consensus::BlockBody {
                transactions: vec![op_alloy_consensus::OpTxEnvelope::Eip1559(
                    alloy_consensus::Signed::new_unchecked(
                        alloy_consensus::TxEip1559 {
                            to: alloy_primitives::TxKind::Call(REGISTRY),
                            input: recordOutputCall { outputRoot: ctx.continuation.output_root }
                                .abi_encode()
                                .into(),
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
        // Prove the span for this test's derivation context (its parent is the checkpoint).
        let derived = ProjectionContext {
            parent_hash,
            l1_head: ctx.l1_head,
            continuation: Continuation {
                anchor: BlockNumHash { number: 9, hash: parent_hash },
                ..ctx.continuation
            },
        };
        let statement = validate_projection_range(&cfg, derived, &span, &StubVerifier).unwrap();
        let profile = cfg.private_projection.clone().unwrap();
        let mut proof = if profile.verifier == SP1_PRIVATE_PROJECTION_V1 {
            mock_envelope(&statement, profile.program_vkey)
        } else {
            execution_mock_proof(&statement)
        };
        if scenario == "forged_proof" {
            *proof.last_mut().unwrap() ^= 1;
        }
        set_claim_proof(&mut span, proof);
        let mut fetcher = TestBatchValidator { op_blocks: vec![checkpoint], ..Default::default() };
        cfg.seq_window_size = 100;
        cfg.max_sequencer_drift = 600;
        cfg.hardforks =
            HardForkConfig { delta_time: Some(0), holocene_time: Some(0), ..Default::default() };
        // The span's last epoch is 6; its hash is the vector's canonical `l1_head`.
        let mut origins = [
            BlockInfo {
                hash: B256::with_last_byte(5),
                number: 5,
                timestamp: 1000,
                ..Default::default()
            },
            BlockInfo { hash: ctx.l1_head, number: 6, timestamp: 1012, ..Default::default() },
        ];
        match scenario {
            "late_origin" => origins[1].timestamp = 1025,
            "late_fork" => cfg.hardforks.jovian_time = Some(1024),
            "late_drift" => {
                origins[0].timestamp = 20;
                origins[1].timestamp = 21;
            }
            "wrong_l1_head" => origins[1].hash = B256::repeat_byte(0xee),
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
            "late_origin" => BatchValidity::Drop(BatchDropReason::TimestampBeforeL1Origin),
            "late_fork" => BatchValidity::Drop(BatchDropReason::NonEmptyTransitionBlock),
            "late_drift" => {
                BatchValidity::Drop(BatchDropReason::SequencerDriftNotAdoptedNextOrigin)
            }
            "wrong_l1_head" | "forged_proof" => {
                BatchValidity::Drop(BatchDropReason::InvalidProjectionRange)
            }
            _ => BatchValidity::Accept,
        };
        assert_eq!(result, expected, "{}: {scenario}", v["name"]);
    }
}

fn leaves(n: u64) -> Vec<B256> {
    (0..n).map(|i| keccak256(i.to_be_bytes())).collect()
}

#[test]
fn commitment_tree_rules() {
    let empty = {
        let mut b = OUTPUTS_DOMAIN.to_vec();
        b.extend_from_slice(&[0u8; 40]);
        keccak256(b)
    };
    assert_eq!(commitment_root(OUTPUTS_DOMAIN, &[]), empty);
    assert_eq!(commitment_proof(&[], 0), None);
    assert!(!verify_commitment_proof(OUTPUTS_DOMAIN, 0, 0, B256::ZERO, &[], empty));
    // Same tree shape as the records root, with a parameterised domain.
    let l = leaves(5);
    let mut wrapped = b"optimism.private-projection.v2\0".to_vec();
    wrapped.extend_from_slice(&5u64.to_be_bytes());
    let top = {
        let n = |a: &B256, b: &B256| keccak256([&[1u8][..], a.as_slice(), b.as_slice()].concat());
        let (a, b, c) = (n(&l[0], &l[1]), n(&l[2], &l[3]), n(&l[4], &l[4]));
        n(&n(&a, &b), &n(&c, &c))
    };
    wrapped.extend_from_slice(top.as_slice());
    assert_eq!(records_root(l), keccak256(wrapped));
    for n in [1u64, 2, 3, 4, 5, 7, 8] {
        let l = leaves(n);
        let root = commitment_root(MESSAGES_DOMAIN, &l);
        assert_ne!(root, commitment_root(OUTPUTS_DOMAIN, &l));
        for i in 0..n {
            let proof = commitment_proof(&l, i as usize).unwrap();
            assert!(verify_commitment_proof(MESSAGES_DOMAIN, n, i, l[i as usize], &proof, root));
            assert!(!verify_commitment_proof(MESSAGES_DOMAIN, n, i, B256::ZERO, &proof, root));
            assert!(!verify_commitment_proof(
                MESSAGES_DOMAIN,
                n + 1,
                i,
                l[i as usize],
                &proof,
                root
            ));
            let mut extra = proof.clone();
            extra.push(B256::ZERO);
            assert!(!verify_commitment_proof(MESSAGES_DOMAIN, n, i, l[i as usize], &extra, root));
            if let Some((_, rest)) = proof.split_last() {
                assert!(!verify_commitment_proof(MESSAGES_DOMAIN, n, i, l[i as usize], rest, root));
            }
        }
        assert_eq!(commitment_proof(&l, n as usize), None);
    }
    // n = 3, index 2: the lone last node at level 0 emits no sibling.
    assert_eq!(commitment_proof(&leaves(3), 2).unwrap().len(), 1);
    assert_eq!(commitment_proof(&leaves(8), 5).unwrap().len(), 3);
}

#[test]
fn leaf_encodings() {
    let root = B256::repeat_byte(0xab);
    let mut want = vec![0u8];
    want.extend_from_slice(&12u64.to_be_bytes());
    want.extend_from_slice(root.as_slice());
    assert_eq!(output_leaf(12, root), keccak256(&want));
    let mut want = vec![0u8];
    want.extend_from_slice(&12u64.to_be_bytes());
    want.extend_from_slice(&3u32.to_be_bytes());
    want.push(2);
    want.extend_from_slice(root.as_slice());
    assert_eq!(message_leaf(12, 3, MessageKind::Exec, root), keccak256(&want));
    assert_eq!(PUBLIC_VALUES_MAGIC, keccak256(b"optimism.private-projection.public-values.v1"));
    assert_eq!(
        SentMessage::SIGNATURE_HASH,
        keccak256(b"SentMessage(uint256,address,uint256,address,bytes)")
    );
    assert_eq!(
        ExecutingMessage::SIGNATURE_HASH,
        keccak256(b"ExecutingMessage(bytes32,(address,uint256,uint256,uint256,uint256))")
    );
}

#[test]
fn canonical_config_hashes() {
    let pch = private_config_hash(b"{\"a\":1}", b"{\"b\":2}");
    let mut want = PRIVATE_CONFIG_DOMAIN.to_vec();
    want.extend_from_slice(keccak256(b"{\"a\":1}").as_slice());
    want.extend_from_slice(keccak256(b"{\"b\":2}").as_slice());
    assert_eq!(pch, keccak256(want));
    // Distinct, sorted chain IDs; u64 and U256 inputs agree.
    let d = dependency_set_hash([902u64, 901, 902]);
    assert_eq!(d, dependency_set_hash([U256::from(901), U256::from(902)]));
    let mut want = DEP_SET_DOMAIN.to_vec();
    want.extend_from_slice(&2u64.to_be_bytes());
    want.extend_from_slice(&U256::from(901).to_be_bytes::<32>());
    want.extend_from_slice(&U256::from(902).to_be_bytes::<32>());
    assert_eq!(d, keccak256(want));

    let (cfg, _, _) = inputs(&vectors()[0]);
    let p = cfg.private_projection.clone().unwrap();
    let mut want = CONFIG_DOMAIN.to_vec();
    want.extend_from_slice(&U256::from(901).to_be_bytes::<32>());
    want.extend_from_slice(&cfg.genesis.l2.number.to_be_bytes());
    want.extend_from_slice(cfg.genesis.l2.hash.as_slice());
    want.extend_from_slice(&1000u64.to_be_bytes());
    want.extend_from_slice(&2u64.to_be_bytes());
    want.extend_from_slice(keccak256(p.verifier.as_bytes()).as_slice());
    want.extend_from_slice(p.genesis_output_root.as_slice());
    want.extend_from_slice(p.program_vkey.as_slice());
    want.extend_from_slice(p.private_config_hash.as_slice());
    want.extend_from_slice(p.dependency_set_hash.as_slice());
    want.extend_from_slice(&[u8::from(p.allow_events), u8::from(p.mock_proofs)]);
    assert_eq!(config_hash(&cfg).unwrap(), keccak256(want));
    let mut flipped = cfg.clone();
    flipped.private_projection.as_mut().unwrap().mock_proofs ^= true;
    assert_ne!(config_hash(&flipped).unwrap(), config_hash(&cfg).unwrap());
    flipped.private_projection = None;
    assert!(config_hash(&flipped).is_err());
}

fn distinct_statement() -> Statement {
    let w = |i: u8| B256::repeat_byte(0x40 + i);
    Statement {
        chain_id: B256::from(U256::from(901)),
        parent_hash: w(5),
        projection_hash: w(17),
        continuation: Continuation {
            anchor: BlockNumHash { number: 6, hash: w(7) },
            output_root: w(8),
            recovery_hash: w(9),
        },
        claim: RangeClaim {
            version: 2,
            firstBlock: 10,
            lastBlock: 11,
            privateTerminalBlockHash: w(13),
            privateTerminalParentHash: w(14),
            anchorBlock: 6,
            anchorOutputRoot: w(8),
            recoveryHash: w(9),
            parentOutputRoot: w(12),
            l1Head: w(15),
            rollupConfigHash: w(2),
            depSetHash: w(4),
            privateDataHash: w(16),
            proof: Bytes::new(),
        },
        projection_config_hash: w(2),
        private_config_hash: w(3),
        outputs_root: w(18),
        messages_root: w(19),
        terminal_output: w(20),
    }
}

#[test]
fn public_values_layout() {
    let pv = public_values(&distinct_statement());
    assert_eq!(pv.len(), 672);
    let word = |k: usize| B256::from_slice(&pv[32 * k..32 * k + 32]);
    assert_eq!(word(0), PUBLIC_VALUES_MAGIC);
    assert_eq!(word(1), B256::from(U256::from(901)));
    for k in [2, 3, 4, 5, 7, 8, 9, 12, 13, 14, 15, 16, 17, 18, 19, 20] {
        assert_eq!(word(k), B256::repeat_byte(0x40 + k as u8), "word {k}");
    }
    assert_eq!(word(6), B256::from(U256::from(6)));
    assert_eq!(word(10), B256::from(U256::from(10)));
    assert_eq!(word(11), B256::from(U256::from(11)));
    // The §F.2 digest masks the top three bits of sha256.
    let d = public_values_digest(&pv);
    assert!(d[0] < 0x20);
    let raw: [u8; 32] = sha2::Sha256::digest(pv).into();
    assert_eq!(d[1..], raw[1..]);
    assert_eq!(d[0], raw[0] & 0x1f);
}

fn mock_envelope(s: &Statement, vkey: B256) -> Vec<u8> {
    let pv = public_values(s);
    encode_envelope(&Envelope {
        kind: EnvelopeKind::Mock,
        proof: mock_proof(vkey, &pv),
        public_values: pv,
    })
}

#[test]
fn envelope_decoding_is_strict() {
    let s = distinct_statement();
    let vkey = B256::with_last_byte(1);
    let env = mock_envelope(&s, vkey);
    assert_eq!(env.len(), 836);
    let decoded = decode_envelope(&env).unwrap();
    assert_eq!(decoded.kind, EnvelopeKind::Mock);
    assert_eq!(encode_envelope(&decoded), env);
    let groth = Envelope {
        kind: EnvelopeKind::Groth16,
        proof: vec![7; GROTH16_PROOF_LEN],
        public_values: public_values(&s),
    };
    let bytes = encode_envelope(&groth);
    assert_eq!(bytes.len(), 1032);
    assert_eq!(decode_envelope(&bytes).unwrap(), groth);
    let mutate = |f: &dyn Fn(&mut Vec<u8>)| {
        let mut b = env.clone();
        f(&mut b);
        decode_envelope(&b)
    };
    assert!(mutate(&|b| b[0] = 2).is_err());
    assert!(mutate(&|b| b[1] = 3).is_err());
    assert!(mutate(&|b| b[1] = 1).is_err()); // mock length under the Groth16 kind
    assert!(mutate(&|b| b.push(0)).is_err());
    assert!(mutate(&|b| b.truncate(b.len() - 1)).is_err());
    assert!(mutate(&|b| b.truncate(3)).is_err());
    let mut short = groth;
    short.proof.pop();
    assert!(decode_envelope(&encode_envelope(&short)).is_err());
    check_mock_proof(&decoded.proof, vkey, &decoded.public_values).unwrap();
    assert!(
        check_mock_proof(&decoded.proof, B256::with_last_byte(2), &decoded.public_values).is_err()
    );
    let mut exit = decoded.proof.clone();
    exit[64 + 31] = 1;
    assert!(check_mock_proof(&exit, vkey, &decoded.public_values).is_err());
    let mut pv = decoded.public_values;
    pv[0] ^= 1;
    assert!(check_mock_proof(&decoded.proof, vkey, &pv).is_err());
}

fn sp1_profile(mock: bool) -> PrivateProjectionConfig {
    PrivateProjectionConfig {
        verifier: SP1_PRIVATE_PROJECTION_V1.into(),
        allow_events: false,
        genesis_output_root: B256::repeat_byte(9),
        program_vkey: B256::with_last_byte(1),
        private_config_hash: B256::repeat_byte(3),
        dependency_set_hash: B256::repeat_byte(4),
        mock_proofs: mock,
    }
}

#[test]
fn sp1_mock_verification_is_gated() {
    let s = distinct_statement();
    let profile = sp1_profile(true);
    let env = mock_envelope(&s, profile.program_vkey);
    for (chain, ok) in [(901, true), (902, true), (10, false)] {
        let r = ConfiguredVerifier { profile: &profile, chain_id: chain }.verify(&s, &env);
        assert_eq!(r.is_ok(), ok, "chain {chain}: {r:?}");
    }
    let off = sp1_profile(false);
    assert_eq!(
        ConfiguredVerifier { profile: &off, chain_id: 901 }.verify(&s, &env),
        Err(ProjectionError("mock proofs disabled"))
    );
    // Even with the gate open, a mock envelope is rejected when compiled without the gate.
    assert!(Sp1Verifier { profile: &profile, allow_mock: false }.verify(&s, &env).is_err());
    // Statement binding: every public-values word is compared bytewise.
    for k in 0..PUBLIC_VALUES_WORDS {
        let mut b = env.clone();
        b[4 + MOCK_PROOF_LEN + 32 * k + 31] ^= 1;
        assert!(
            ConfiguredVerifier { profile: &profile, chain_id: 901 }.verify(&s, &b).is_err(),
            "word {k}"
        );
    }
    let wrong = mock_envelope(&s, B256::with_last_byte(2));
    assert!(ConfiguredVerifier { profile: &profile, chain_id: 901 }.verify(&s, &wrong).is_err());
}

#[test]
fn config_gate_matrix() {
    let stub = PrivateProjectionConfig {
        verifier: INSECURE_STUB.into(),
        genesis_output_root: B256::repeat_byte(9),
        dependency_set_hash: B256::repeat_byte(4),
        ..Default::default()
    };
    let exec = PrivateProjectionConfig { verifier: EXECUTION_MOCK.into(), ..stub };
    for compiled in [false, true] {
        for chain in [901u64, 902, 10] {
            for (profile, gated) in [
                (&stub, true),
                (&exec, true),
                (&sp1_profile(true), true),
                (&sp1_profile(false), false),
            ] {
                let want = !gated || (compiled && chain != 10);
                assert_eq!(
                    check_config_with_gate(profile, chain, compiled).is_ok(),
                    want,
                    "{} mock={} compiled={compiled} chain={chain}",
                    profile.verifier,
                    profile.mock_proofs
                );
            }
        }
    }
    const { assert!(TEST_VERIFIERS_COMPILED, "kona-protocol tests compile the test verifiers") };
    assert!(check_config(&stub, 901).is_ok() && check_config(&stub, 10).is_err());
    // §B.1 field rules.
    let bad = |f: &dyn Fn(&mut PrivateProjectionConfig), base: &PrivateProjectionConfig| {
        let mut p = base.clone();
        f(&mut p);
        check_profile(&p).is_err()
    };
    let sp1 = sp1_profile(false);
    assert!(check_profile(&sp1).is_ok());
    assert!(bad(&|p| p.genesis_output_root = B256::ZERO, &sp1));
    assert!(bad(&|p| p.program_vkey = B256::ZERO, &sp1));
    assert!(bad(&|p| p.program_vkey = B256::from(BN254_SCALAR_MODULUS), &sp1));
    assert!(bad(&|p| p.private_config_hash = B256::ZERO, &sp1));
    assert!(bad(&|p| p.dependency_set_hash = B256::ZERO, &sp1));
    assert!(bad(&|p| p.allow_events = true, &sp1));
    assert!(bad(&|p| p.verifier = "sp1-private-projection-v2".into(), &sp1));
    assert!(!bad(&|p| p.allow_events = true, &stub));
    assert!(bad(&|p| p.dependency_set_hash = B256::ZERO, &stub));
    assert!(bad(&|p| p.program_vkey = B256::with_last_byte(1), &exec));
    assert!(bad(&|p| p.private_config_hash = B256::with_last_byte(1), &exec));
    assert!(bad(&|p| p.mock_proofs = true, &stub));
    let mut r = B256::from(BN254_SCALAR_MODULUS);
    r[31] -= 1;
    assert!(is_canonical_scalar(&r.0) && !is_canonical_scalar(&BN254_SCALAR_MODULUS));
}

#[test]
fn canonical_output_checks_claim_version() {
    let (_, span, _) = inputs(&vectors()[0]);
    let txs = span.batches[0].transactions.clone();
    assert!(!canonical_output(&txs).unwrap().is_zero());
    let TxEnvelope::Eip1559(signed) = TxEnvelope::decode_2718_exact(&txs[0]).unwrap() else {
        panic!("claim tx")
    };
    let mut call = postClaimCall::abi_decode(signed.tx().input()).unwrap();
    call.claim.version = 1;
    let mut tx = signed.tx().clone();
    tx.input = call.abi_encode().into();
    let raw: Bytes = TxEnvelope::Eip1559(alloy_consensus::Signed::new_unchecked(
        tx,
        *signed.signature(),
        B256::ZERO,
    ))
    .encoded_2718()
    .into();
    assert_eq!(
        canonical_output(&[raw, txs[1].clone()]),
        Err(ProjectionError("canonical claim version"))
    );
}

/// Replays of the `mixed` vector (block 2: output, export, import) rendered back from
/// their private logs give the same calldata and the same leaves as admission.
#[test]
fn rendered_logs_match_published_replays() {
    let (cfg, span, ctx) = inputs(&vectors()[0]);
    let statement = validate_projection_range(&cfg, ctx, &span, &StubVerifier).unwrap();
    let first = statement.claim.firstBlock;
    let mut all_leaves = Vec::new();
    for (i, block) in span.batches.iter().enumerate() {
        let number = first + i as u64;
        let mut logs = vec![Log {
            address: Address::repeat_byte(0x77),
            data: LogData::new_unchecked(vec![SentMessage::SIGNATURE_HASH], Bytes::new()),
        }];
        let mut published = Vec::new();
        for raw in &block.transactions {
            let tx = TxEnvelope::decode_2718_exact(raw).unwrap();
            let (to, input) = (tx.to().unwrap(), tx.input().clone());
            let log = match to {
                MESSENGER => {
                    let c = replaySentMessageCall::abi_decode(&input).unwrap();
                    let (topics, data) = sent_message_log(&c);
                    Log { address: MESSENGER, data: LogData::new_unchecked(topics.to_vec(), data) }
                }
                INBOX => Log {
                    address: INBOX,
                    data: LogData::new_unchecked(
                        vec![ExecutingMessage::SIGNATURE_HASH, B256::from_slice(&input[164..196])],
                        input[4..164].to_vec().into(),
                    ),
                },
                _ => continue,
            };
            logs.push(log);
            // An unrendered private log between messages shifts nothing.
            logs.push(Log {
                address: MESSENGER,
                data: LogData::new_unchecked(vec![B256::repeat_byte(1)], Bytes::new()),
            });
            published.push((to, input));
        }
        let rendered = rendered_logs(logs.iter()).unwrap();
        assert_eq!(rendered.len(), published.len());
        for (k, rl) in rendered.iter().enumerate() {
            assert_eq!(rl.rendered_log_index as usize, k);
            assert_eq!(rl.private_log_index as usize, 1 + 2 * k);
            assert_eq!(replay_calldata(rl).unwrap(), published[k]);
        }
        all_leaves.extend(message_leaves(number, &rendered).unwrap());
    }
    assert!(!all_leaves.is_empty());
    assert_eq!(commitment_root(MESSAGES_DOMAIN, &all_leaves), statement.messages_root);
}

#[test]
fn unrenderable_logs_are_fatal() {
    let m = replaySentMessageCall {
        destination: U256::from(902),
        nonce: U256::from(1),
        sender: Address::repeat_byte(1),
        target: Address::repeat_byte(2),
        message: Bytes::from_static(b"hi"),
    };
    let render = |log: &Log| {
        let logs = [log.clone()];
        let rendered = rendered_logs(logs.iter()).unwrap();
        assert_eq!(rendered.len(), 1);
        replay_calldata(&rendered[0]).map(|_| ())
    };
    let sent = |m: &replaySentMessageCall| {
        let (topics, data) = sent_message_log(m);
        Log { address: MESSENGER, data: LogData::new_unchecked(topics.to_vec(), data) }
    };
    assert!(render(&sent(&m)).is_ok());
    assert_eq!(
        rendered_message(&rendered_logs([sent(&m)].iter()).unwrap()[0]).unwrap(),
        (MessageKind::Init, export_message_hash(&m))
    );
    for bad in [
        replaySentMessageCall { sender: BRIDGE, ..m.clone() },
        replaySentMessageCall { target: BRIDGE, ..m.clone() },
        replaySentMessageCall { message: vec![0; MAX_MESSAGE + 1].into(), ..m },
    ] {
        assert!(render(&sent(&bad)).is_err());
    }
    let mut dirty = sent(&m);
    let mut topics = dirty.topics().to_vec();
    topics[2][0] = 1;
    dirty.data = LogData::new_unchecked(topics, dirty.data.data.clone());
    assert!(render(&dirty).is_err());
    let mut trailing = sent(&m);
    let mut data = trailing.data.data.to_vec();
    data.extend_from_slice(&[0; 32]);
    trailing.data = LogData::new_unchecked(trailing.topics().to_vec(), data.into());
    assert!(render(&trailing).is_err());
    let exec = |topics: Vec<B256>, data: Vec<u8>| Log {
        address: INBOX,
        data: LogData::new_unchecked(topics, data.into()),
    };
    let t0 = ExecutingMessage::SIGNATURE_HASH;
    assert!(render(&exec(vec![t0, B256::ZERO], vec![0; 160])).is_ok());
    assert!(render(&exec(vec![t0], vec![0; 160])).is_err());
    assert!(render(&exec(vec![t0, B256::ZERO], vec![0; 159])).is_err());
    let mut padded = vec![0; 160];
    padded[64] = 1; // log index wider than u32
    assert!(render(&exec(vec![t0, B256::ZERO], padded)).is_err());
    // Other emitters or topics never render.
    assert!(!renders(&exec(vec![B256::ZERO, B256::ZERO], vec![0; 160])));
    assert!(!renders(&Log { address: REPLAYER, data: sent(&m).data }));
}

#[test]
fn shared_commitment_vectors() {
    let v: Value = serde_json::from_str(include_str!(
        "../../../../../../../op-private-interop/projection/testdata/commitments.json"
    ))
    .unwrap();
    let k = &v["constants"];
    assert_eq!(h(&k["public_values_magic"]), PUBLIC_VALUES_MAGIC);
    for (name, ours) in [
        ("config_domain", CONFIG_DOMAIN),
        ("private_config_domain", PRIVATE_CONFIG_DOMAIN),
        ("dep_set_domain", DEP_SET_DOMAIN),
        ("outputs_domain", OUTPUTS_DOMAIN),
        ("messages_domain", MESSAGES_DOMAIN),
    ] {
        assert_eq!(hb(&k[name]).as_ref(), ours, "{name}");
    }

    let trees = v["trees"].as_array().unwrap();
    assert!(trees.len() >= 8);
    for t in trees {
        let domain = hb(&t["domain"]);
        let n = t["n"].as_u64().unwrap();
        let leaves: Vec<B256> = t["leaves"].as_array().unwrap().iter().map(h).collect();
        assert_eq!(leaves.len() as u64, n);
        let root = h(&t["root"]);
        assert_eq!(commitment_root(&domain, &leaves), root, "n={n}");
        let proofs = t["proofs"].as_array().unwrap();
        assert_eq!(proofs.len() as u64, n);
        for p in proofs {
            let index = p["index"].as_u64().unwrap();
            let siblings: Vec<B256> = p["siblings"].as_array().unwrap().iter().map(h).collect();
            assert_eq!(h(&p["leaf"]), leaves[index as usize]);
            assert_eq!(commitment_proof(&leaves, index as usize).unwrap(), siblings, "n={n}");
            assert!(verify_commitment_proof(&domain, n, index, h(&p["leaf"]), &siblings, root));
        }
        let bad = t["bad_proofs"].as_array().unwrap();
        assert!(!bad.is_empty());
        for p in bad {
            let siblings: Vec<B256> = p["siblings"].as_array().unwrap().iter().map(h).collect();
            assert!(
                !verify_commitment_proof(
                    &domain,
                    n,
                    p["index"].as_u64().unwrap(),
                    h(&p["leaf"]),
                    &siblings,
                    root
                ),
                "n={n}: {}",
                p["reason"]
            );
        }
    }

    let o = &v["output_leaf"];
    assert_eq!(
        output_leaf(o["block_number"].as_u64().unwrap(), h(&o["output_root"])),
        h(&o["leaf"])
    );

    // Message leaves: calldata -> leaf (admission) and log -> leaf (relation) agree.
    let samples = v["message_leaves"].as_array().unwrap();
    assert_eq!(samples.len(), 2);
    for m in samples {
        let name = &m["name"];
        let number = m["block_number"].as_u64().unwrap();
        let index = m["rendered_index"].as_u64().unwrap() as u32;
        let kind = match m["kind"].as_u64().unwrap() {
            1 => MessageKind::Init,
            2 => MessageKind::Exec,
            k => panic!("{name}: kind {k}"),
        };
        let to: Address = serde_json::from_value(m["to"].clone()).unwrap();
        let calldata = hb(&m["calldata"]);
        let hash = match kind {
            MessageKind::Init => {
                export_message_hash(&replaySentMessageCall::abi_decode_validate(&calldata).unwrap())
            }
            MessageKind::Exec => import_message_hash(calldata[4..].try_into().unwrap()),
        };
        assert_eq!(hash, h(&m["message_hash"]), "{name}: calldata hash");
        assert_eq!(message_leaf(number, index, kind, hash), h(&m["leaf"]), "{name}: leaf");

        let l = &m["log"];
        let log = Log {
            address: serde_json::from_value(l["address"].clone()).unwrap(),
            data: LogData::new_unchecked(
                l["topics"].as_array().unwrap().iter().map(h).collect(),
                hb(&l["data"]),
            ),
        };
        // Pad the block with unrendered logs so the rendered index is the sample's.
        let filler = Log {
            address: INBOX,
            data: LogData::new_unchecked(vec![SentMessage::SIGNATURE_HASH], Bytes::new()),
        };
        let mut logs = vec![filler.clone(); 2];
        let other = sent_message_log(&replaySentMessageCall {
            destination: U256::from(1),
            nonce: U256::ZERO,
            sender: Address::ZERO,
            target: Address::ZERO,
            message: Bytes::new(),
        });
        for _ in 0..index {
            logs.push(Log {
                address: MESSENGER,
                data: LogData::new_unchecked(other.0.to_vec(), other.1.clone()),
            });
        }
        logs.push(log);
        let rendered = rendered_logs(logs.iter()).unwrap();
        let rl = rendered.last().unwrap();
        assert_eq!(rl.rendered_log_index, index);
        assert_eq!(replay_calldata(rl).unwrap(), (to, calldata), "{name}: replay calldata");
        assert_eq!(rendered_message(rl).unwrap(), (kind, hash), "{name}: log hash");
        assert_eq!(
            message_leaves(number, &rendered).unwrap().last(),
            Some(&h(&m["leaf"])),
            "{name}: log leaf"
        );
    }

    for c in v["config_hashes"].as_array().unwrap() {
        let (cfg, _, _) = inputs(c);
        assert_eq!(config_hash(&cfg).unwrap(), h(&c["hash"]), "{}", c["name"]);
    }
    let p = &v["private_config_hash"];
    assert_eq!(
        private_config_hash(&hb(&p["private_rollup_json"]), &hb(&p["l1_chain_config_json"])),
        h(&p["hash"])
    );
    for d in v["dependency_set_hashes"].as_array().unwrap() {
        let ids = d["chain_ids"]
            .as_array()
            .unwrap()
            .iter()
            .map(|x| x.as_str().unwrap().parse::<U256>().unwrap());
        assert_eq!(dependency_set_hash(ids), h(&d["hash"]), "{:?}", d["chain_ids"]);
    }

    let pv = &v["public_values"];
    let x = &pv["statement"];
    let n = |k: &str| x[k].as_u64().unwrap();
    let statement = Statement {
        chain_id: h(&x["chain_id"]),
        parent_hash: h(&x["parent_hash"]),
        projection_hash: h(&x["projection_hash"]),
        continuation: Continuation {
            anchor: BlockNumHash { number: n("anchor_number"), hash: h(&x["anchor_hash"]) },
            output_root: h(&x["anchor_output_root"]),
            recovery_hash: h(&x["recovery_hash"]),
        },
        claim: RangeClaim {
            version: 2,
            firstBlock: n("first_block"),
            lastBlock: n("last_block"),
            privateTerminalBlockHash: h(&x["private_terminal_block_hash"]),
            privateTerminalParentHash: h(&x["private_terminal_parent_hash"]),
            anchorBlock: n("anchor_number"),
            anchorOutputRoot: h(&x["anchor_output_root"]),
            recoveryHash: h(&x["recovery_hash"]),
            parentOutputRoot: h(&x["parent_output_root"]),
            l1Head: h(&x["l1_head"]),
            rollupConfigHash: h(&x["projection_config_hash"]),
            depSetHash: h(&x["dep_set_hash"]),
            privateDataHash: h(&x["private_data_hash"]),
            proof: Bytes::new(),
        },
        projection_config_hash: h(&x["projection_config_hash"]),
        private_config_hash: h(&x["private_config_hash"]),
        outputs_root: h(&x["outputs_root"]),
        messages_root: h(&x["messages_root"]),
        terminal_output: h(&x["terminal_output"]),
    };
    assert_eq!(public_values(&statement).as_slice(), hb(&pv["public_values"]).as_ref());
    if !pv["digest"].is_null() {
        assert_eq!(public_values_digest(&public_values(&statement)), h(&pv["digest"]));
    }
}

#[cfg(feature = "sp1-projection-verifier")]
mod groth16 {
    use super::*;
    use crate::projection::sp1::{Circuit, circuit_v6_1_0, verify_groth16};

    #[test]
    fn production_circuit_constants() {
        let c = circuit_v6_1_0();
        assert_eq!(c.vk().len(), 492);
        assert_eq!(
            B256::from_slice(&sha2::Sha256::digest(c.vk())),
            b256!("0x4388a21c687fdd5f218d7e3d13190cac4c5355818d3605fd5fb811df468ee696")
        );
        assert_eq!(
            B256::from(c.vk_root()),
            b256!("0x002f850ee998974d6cc00e50cd0814b098c05bfade466d28573240d057f25352")
        );
        // Go embeds the same bytes (`sp1groth16/vk/groth16_vk_v6.1.0.bin`).
        assert_eq!(
            c.vk(),
            include_bytes!(
                "../../../../../../../op-private-interop/projection/sp1groth16/vk/groth16_vk_v6.1.0.bin"
            )
        );
    }

    #[test]
    fn upstream_v6_0_0_fixture() {
        let vk = include_bytes!(
            "../../../../../../../op-private-interop/projection/sp1groth16/testdata/groth16_vk_v6.0.0.bin"
        );
        assert_eq!(
            B256::from_slice(&sha2::Sha256::digest(vk)),
            b256!("0x0e78f4db7a6771a3a6a7d9c3b0de6fe73d58781368967a7fe84d87aefffec896")
        );
        let fixture: Value = serde_json::from_str(include_str!(
            "../../../../../../../op-private-interop/projection/sp1groth16/testdata/groth16-fixture-v6.0.0.json"
        ))
        .unwrap();
        let root = b256!("0x008cd56e10c2fe24795cff1e1d1f40d3a324528d315674da45d26afb376e8670");
        let circuit = Circuit::new(vk, root.0);
        let vkey = h(&fixture["vkey"]);
        let pv = hb(&fixture["publicValues"]);
        let proof = hb(&fixture["proof"]).to_vec();
        verify_groth16(&circuit, &proof, vkey, &pv).unwrap();
        // The production circuit rejects a v6.0.0 proof (prefix and root differ).
        assert!(verify_groth16(&circuit_v6_1_0(), &proof, vkey, &pv).is_err());
        let reject = |proof: &[u8], vkey: B256, pv: &[u8]| {
            assert!(verify_groth16(&circuit, proof, vkey, pv).is_err());
        };
        let mut p = proof.clone();
        p[200] ^= 1;
        reject(&p, vkey, &pv);
        let mut bad_pv = pv.to_vec();
        bad_pv[0] ^= 1;
        reject(&proof, vkey, &bad_pv);
        let mut wrong_vkey = vkey;
        wrong_vkey[31] ^= 1;
        reject(&proof, wrong_vkey, &pv);
        let mut p = proof.clone();
        p[36 + 31] ^= 1;
        reject(&p, vkey, &pv);
        let mut p = proof.clone();
        p[4 + 31] = 1;
        reject(&p, vkey, &pv);
        let mut p = proof.clone();
        p[0] ^= 1;
        reject(&p, vkey, &pv);
        reject(&proof[..355], vkey, &pv);
        let mut p = proof.clone();
        p.push(0);
        reject(&p, vkey, &pv);
        // A noncanonical nonce (>= r) is rejected rather than reduced.
        let mut p = proof;
        p[68..100].copy_from_slice(&[0xff; 32]);
        reject(&p, vkey, &pv);
    }
}
