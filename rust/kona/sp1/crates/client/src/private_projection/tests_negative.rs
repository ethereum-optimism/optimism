//! The N-series of negative tests for the `sp1-private-projection-v1` relation
//! (spec-sound-profile §D, harness owned by WP5).
//!
//! Each test mutates one input of a WP3 fixture and asserts one of two outcomes, never merely
//! "some error":
//!
//! - the relation refuses to execute, with the exact error of the check that must catch it; or
//! - the relation executes, but its public values differ from the ones **admission** computes, at
//!   exactly the named words, and the admission-side `Sp1Verifier` rejects a (consistent) mock
//!   envelope over the relation's output.
//!
//! Expected public values are never taken from `execute`. They come from the fixture's public data
//! through admission's own path: `validate_projection_range` + `public_values`, with the context a
//! verifier's `ContextCollector` would produce (`expected_public_values`, `admission_statement`).
//!
//! | ID | Mutation | Caught by |
//! |---|---|---|
//! | N1 | private rollup JSON: a non-execution field, a fee field | word 3 / execution |
//! | N2 | L1 chain config: blob schedule | word 3 |
//! | N3 | projection config `block_time` | `execute` (geometry) |
//! | N4 | dependency set with an extra chain | `execute` (dependency-set commitment) |
//! | N5 | forged `TransactionDeposited` log in an L1 receipt | `execute` (receipts trie / L1 chain) |
//! | N6 | one L1 receipt-trie node removed | `execute` (missing preimage) |
//! | N7 | an L1 header replaced (different `mix_hash`) | `execute` (L1 hash chain) |
//! | N8 | `input.l1_head != claim.l1Head` | `execute` |
//! | N9 | claim and input `l1Head` = another canonical L1 block | `execute`, and admission |
//! | N10 | recovery block L1-info deposit names another L1 block | `execute`, and word 9 |
//! | N10b | `RelationInput` JSON carries `attributes` | transport (`deny_unknown_fields`) |
//! | N11 | proof anchored at an older checkpoint than admission's | words 6–9 |
//! | N12 | `anchor_output` does not match the anchor header | `execute` |
//! | N13 | real Groth16 fixture, wrong `program_vkey` | Groth16 verifier |
//! | N14 | mock envelope, wrong vkey word | mock verifier |
//! | N15 | an export replay dropped | `execute`, and admission (words 17, 19) |
//! | N16 | an extra `validateMessage` replay | `execute`, and admission (words 17, 19) |
//! | N17 | two replays swapped | `execute`, and admission (words 17, 19) |
//! | N18 | a replay moved to the next block | `execute`, and admission (words 17, 19) |
//! | N19 | a replay with the same fields but other message bytes | `execute`, and admission (words 17, 19) |
//! | N20 | a published `recordOutput` altered | `execute`, and admission (words 17, 18, 20) |

use super::{
    tests::{
        FIXTURE_VKEY, add_l1_block, deposit_log, epoch_of, expected_public_values, fixture,
        fixture_with_sdm, l1_chain, l1_header, l1_receipts, receipt, recovery_hash_of, signed,
    },
    *,
};
use alloy_primitives::{Address, address, hex};
use kona_protocol::projection::{
    Envelope, EnvelopeKind, ProofVerifier, Sp1Verifier, encode_envelope, mock_proof,
};

const MESSENGER: Address = address!("4200000000000000000000000000000000000023");
const REGISTRY: Address = address!("420000000000000000000000000000000000002e");

// Word indices of PublicValuesV1 (§C.2).
const W_PRIVATE_CONFIG_HASH: usize = 3;
const W_ANCHOR_NUMBER: usize = 6;
const W_ANCHOR_HASH: usize = 7;
const W_ANCHOR_OUTPUT: usize = 8;
const W_RECOVERY_HASH: usize = 9;
const W_L1_HEAD: usize = 15;
const W_PROJECTION_HASH: usize = 17;
const W_OUTPUTS_ROOT: usize = 18;
const W_MESSAGES_ROOT: usize = 19;
const W_TERMINAL_OUTPUT: usize = 20;

/// The relation's error, with its full context chain.
fn relation_err(input: &RelationInput, witness: &Witness) -> String {
    format!("{:#}", execute(input, witness).expect_err("the relation must reject"))
}

/// Assert the relation rejects with an error containing `needle`.
#[track_caller]
fn rejects(id: &str, input: &RelationInput, witness: &Witness, needle: &str) {
    let e = relation_err(input, witness);
    assert!(e.contains(needle), "{id}: expected an error containing {needle:?}, got: {e}");
}

/// The words at which two public values differ.
fn differing_words(a: &PublicValues, b: &PublicValues) -> Vec<usize> {
    (0..projection::PUBLIC_VALUES_WORDS)
        .filter(|&i| public_value_word(a, i) != public_value_word(b, i))
        .collect()
}

fn projection_cfg(input: &RelationInput) -> RollupConfig {
    serde_json::from_slice(&input.projection_config).expect("fixture projection config")
}

fn span_of(blocks: &[ProjectionBlock]) -> SpanBatch {
    SpanBatch {
        batches: blocks
            .iter()
            .map(|b| SpanBatchElement {
                timestamp: b.timestamp,
                epoch_num: b.epoch,
                transactions: b.transactions.clone(),
            })
            .collect(),
        ..Default::default()
    }
}

/// Admission's statement over `blocks` under the given verifier context: the verifier's own view.
fn admission_statement(
    cfg: &RollupConfig,
    parent_hash: B256,
    l1_head: B256,
    continuation: projection::Continuation,
    blocks: &[ProjectionBlock],
) -> Result<projection::Statement, projection::ProjectionError> {
    projection::validate_projection_range(
        cfg,
        projection::ProjectionContext { parent_hash, l1_head, continuation },
        &span_of(blocks),
        &projection::StubVerifier,
    )
}

/// The canonical continuation of a fixture, as the verifier's `ContextCollector` resolves it.
fn canonical_continuation(input: &RelationInput) -> projection::Continuation {
    projection::Continuation {
        anchor: input.anchor,
        output_root: input.anchor_output,
        recovery_hash: recovery_hash_of(&input.recovery),
    }
}

/// An internally consistent SP1 mock envelope over `pv` under `vkey`.
fn mock_envelope(vkey: B256, pv: &PublicValues) -> Vec<u8> {
    encode_envelope(&Envelope {
        kind: EnvelopeKind::Mock,
        proof: mock_proof(vkey, pv),
        public_values: *pv,
    })
}

/// Admission's SP1 verifier (mock envelopes enabled, as under the test gate) over `statement`.
fn sp1_verify(
    cfg: &RollupConfig,
    statement: &projection::Statement,
    envelope: &[u8],
) -> Result<(), projection::ProjectionError> {
    let profile = cfg.private_projection.as_ref().expect("sp1 fixture profile");
    Sp1Verifier { profile, allow_mock: true }.verify(statement, envelope)
}

/// The relation executed; admission rejects its output, and the difference is exactly `words`.
#[track_caller]
fn admission_rejects_at(
    id: &str,
    cfg: &RollupConfig,
    pv: &PublicValues,
    admitted: &PublicValues,
    words: &[usize],
) {
    assert_eq!(differing_words(pv, admitted), words, "{id}: differing public-values words");
    let statement = statement_from_public_values(admitted);
    let e = sp1_verify(cfg, &statement, &mock_envelope(FIXTURE_VKEY, pv))
        .expect_err("admission must reject the relation's output");
    assert_eq!(e.0, "public values do not match the statement", "{id}");
}

/// A statement whose `public_values` are exactly `pv` (for driving the verifier from a value
/// admission computed).
fn statement_from_public_values(pv: &PublicValues) -> projection::Statement {
    let w = |i| public_value_word(pv, i);
    let n = |i| u64::try_from(U256::from_be_bytes(w(i).0)).expect("u64 word");
    let s = projection::Statement {
        chain_id: w(1),
        parent_hash: w(5),
        projection_hash: w(17),
        continuation: projection::Continuation {
            anchor: BlockNumHash { number: n(6), hash: w(7) },
            output_root: w(8),
            recovery_hash: w(9),
        },
        claim: projection::RangeClaim {
            version: 2,
            firstBlock: n(10),
            lastBlock: n(11),
            privateTerminalBlockHash: w(13),
            privateTerminalParentHash: w(14),
            anchorBlock: n(6),
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
    };
    assert_eq!(projection::public_values(&s), *pv, "statement round-trips its public values");
    s
}

/// Re-sign every transaction of a block with nonces 0.. (the admission transcript binds them).
fn resign(block: &mut ProjectionBlock) {
    for (nonce, raw) in block.transactions.iter_mut().enumerate() {
        let tx = TxEnvelope::decode_2718_exact(raw).unwrap();
        *raw = signed(tx.to().unwrap(), tx.input().clone(), nonce as u64);
    }
}

fn decode_claim(input: &RelationInput) -> projection::RangeClaim {
    let tx = TxEnvelope::decode_2718_exact(&input.blocks[0].transactions[0]).unwrap();
    projection::postClaimCall::abi_decode_validate(tx.input()).unwrap().claim
}

fn replace_claim(input: &mut RelationInput, claim: projection::RangeClaim) {
    input.blocks[0].transactions[0] =
        signed(REGISTRY, projection::postClaimCall { claim }.abi_encode().into(), 0);
}

// ---------------------------------------------------------------------------------------------
// §D.1 configs

/// N1: the private rollup config is bound byte for byte (word 3), and a fee change is also an
/// execution change.
#[test]
fn n01_private_config_bytes_and_fee_field() {
    let (input, witness, expected) = fixture(0);
    let cfg = projection_cfg(&input);

    // A field the relation does not execute on: the relation succeeds, only word 3 differs.
    let mut bad = input.clone();
    let mut private: serde_json::Value = serde_json::from_slice(&bad.private_config).unwrap();
    let window = private["seq_window_size"].as_u64().unwrap_or(0);
    private["seq_window_size"] = (window + 1).into();
    bad.private_config = serde_json::to_vec(&private).unwrap().into();
    let pv = execute(&bad, &witness).expect("a non-execution field does not change execution");
    admission_rejects_at("N1", &cfg, &pv, &expected, &[W_PRIVATE_CONFIG_HASH]);

    // A fee field: one digit of the operator fee scalar. It enters the private L1-info deposit, so
    // the executed outputs no longer match the published records.
    let mut bad = input;
    let mut private: serde_json::Value = serde_json::from_slice(&bad.private_config).unwrap();
    private["genesis"]["system_config"]["operatorFeeScalar"] = 24.into();
    bad.private_config = serde_json::to_vec(&private).unwrap().into();
    rejects("N1", &bad, &witness, "published checkpoint differs from executed output");
}

/// N2: the L1 chain config is part of `private_config_hash`.
#[test]
fn n02_l1_config_blob_schedule() {
    let (input, witness, expected) = fixture(0);
    let cfg = projection_cfg(&input);
    let mut bad = input;
    let mut l1: serde_json::Value = serde_json::from_slice(&bad.l1_config).unwrap();
    l1["blobSchedule"] = serde_json::json!({
        "cancun": { "target": 3, "max": 6, "baseFeeUpdateFraction": 3338477 },
    });
    bad.l1_config = serde_json::to_vec(&l1).unwrap().into();
    let pv = execute(&bad, &witness).expect("an inactive blob schedule does not change execution");
    admission_rejects_at("N2", &cfg, &pv, &expected, &[W_PRIVATE_CONFIG_HASH]);
}

/// N3: projection geometry is checked against the private config.
#[test]
fn n03_projection_block_time() {
    let (input, witness, _) = fixture(0);
    let mut bad = input;
    let mut projection: serde_json::Value = serde_json::from_slice(&bad.projection_config).unwrap();
    projection["block_time"] = 3.into();
    bad.projection_config = serde_json::to_vec(&projection).unwrap().into();
    rejects("N3", &bad, &witness, "private/projection geometry mismatch");
}

/// N4: the dependency set is hashed canonically and must equal the claim's `depSetHash`.
#[test]
fn n04_dependency_set_extra_chain() {
    let (input, witness, _) = fixture(0);
    let mut bad = input;
    bad.dependency_set = Bytes::from_static(br#"{"dependencies":{"901":{},"902":{},"903":{}}}"#);
    rejects("N4", &bad, &witness, "dependency-set commitment");
}

// ---------------------------------------------------------------------------------------------
// §D.2 L1 authentication and derived attributes

/// N5: a forged `TransactionDeposited` log. Under the honest header, the forged receipts do not
/// resolve the committed receipts root; behind a re-hashed header, nothing links it to `l1Head`.
#[test]
fn n05_forged_deposit_log() {
    let (input, witness, _) = fixture(0);
    let forged_log = deposit_log(
        Address::with_last_byte(0x66),
        Address::with_last_byte(0x66),
        1_000_000_000,
        0,
        100_000,
    );
    let mut forged = l1_receipts(101);
    forged.push(receipt(84_000, vec![forged_log]));
    let honest = l1_chain(epoch_of(2));

    let mut bad = witness.clone();
    let (_, honest_nodes) = ordered_list_preimages(&l1_receipts(101));
    for node in &honest_nodes {
        bad.preimages.remove(&keccak256(node));
    }
    for node in ordered_list_preimages(&forged).1 {
        bad.preimages.insert(keccak256(&node), node);
    }
    rejects("N5", &input, &bad, "receipts trie");

    let mut bad = witness;
    let header = l1_header(101, honest[&100].hash_slow(), &forged);
    add_l1_block(&header, &forged, &mut bad.preimages);
    bad.preimages.remove(&honest[&101].hash_slow());
    rejects("N5", &input, &bad, "missing witness preimage");
}

/// N6: one L1 receipts-trie node missing.
#[test]
fn n06_missing_receipt_node() {
    let (input, witness, _) = fixture(0);
    let (_, nodes) = ordered_list_preimages(&l1_receipts(101));
    let mut bad = witness;
    assert!(bad.preimages.remove(&keccak256(nodes.last().unwrap())).is_some());
    rejects("N6", &input, &bad, "L1 receipts trie");
}

/// N7: an L1 header with a different `mix_hash` (the private `prev_randao`).
#[test]
fn n07_replaced_l1_header() {
    let (input, witness, _) = fixture(0);
    let honest = l1_chain(epoch_of(2));
    let mut replaced = honest[&101].clone();
    replaced.mix_hash = B256::repeat_byte(0xee);

    // Under the honest key: the content-address check refuses it.
    let mut bad = witness.clone();
    bad.preimages.insert(honest[&101].hash_slow(), alloy_rlp::encode(&replaced).into());
    rejects("N7", &input, &bad, "incorrect witness preimage");

    // Under its own key: the hash chain from l1Head no longer reaches it.
    let mut bad = witness;
    bad.preimages.remove(&honest[&101].hash_slow());
    bad.preimages.insert(replaced.hash_slow(), alloy_rlp::encode(&replaced).into());
    rejects("N7", &input, &bad, "missing witness preimage");
}

/// N8: the relation's `l1_head` must be the claim's.
#[test]
fn n08_input_l1_head_differs_from_claim() {
    let (input, witness, _) = fixture(1);
    let mut bad = input;
    bad.l1_head = B256::repeat_byte(0x55);
    rejects("N8", &bad, &witness, "claim l1 head mismatch");
}

/// N9: claim and relation both name a different canonical L1 block. The relation rejects the
/// terminal origin, and admission rejects the claim against its own L1 view (word 15).
#[test]
fn n09_claim_l1_head_other_canonical_block() {
    let (input, witness, _) = fixture(0);
    let l1 = l1_chain(epoch_of(2));
    let other = l1[&(epoch_of(2) - 1)].hash_slow();
    assert_ne!(other, input.l1_head);
    let mut bad = input.clone();
    let mut claim = decode_claim(&bad);
    claim.l1Head = other;
    replace_claim(&mut bad, claim);
    bad.l1_head = other;
    // The prover may supply the other block's header too; the walk still cannot reach the span.
    let mut witness = witness;
    witness.preimages.insert(other, alloy_rlp::encode(&l1[&(epoch_of(2) - 1)]).into());
    // The L1 witness walk from the claimed head cannot reach the span's epochs.
    rejects("N9", &bad, &witness, "L1 head precedes the span's epochs");

    let cfg = projection_cfg(&input);
    let e = admission_statement(
        &cfg,
        input.parent_hash,
        input.l1_head,
        canonical_continuation(&input),
        &bad.blocks,
    )
    .expect_err("N9: admission compares the claim's l1Head with its own L1 window");
    assert_eq!(e.0, "claim l1 head mismatch", "N9");
}

/// Rebuild the hash chain of a recovery interval after editing one block, and let the claim
/// commit to the edited interval, so the relation reaches its per-block recovery checks.
fn rechain_recovery(bad: &mut RelationInput) -> B256 {
    for block in &mut bad.recovery {
        block.header.transactions_root = tx_root(&encode_all(block));
    }
    let mut parent = bad.anchor.hash;
    for block in &mut bad.recovery {
        block.header.parent_hash = parent;
        parent = block.header.hash_slow();
    }
    bad.parent_hash = parent;
    let forged = recovery_hash_of(&bad.recovery);
    let mut claim = decode_claim(bad);
    claim.recoveryHash = forged;
    replace_claim(bad, claim);
    forged
}

/// Patch the L1-info deposit calldata of recovery block `index`.
fn patch_recovery_l1_info(bad: &mut RelationInput, index: usize, patch: impl Fn(&mut Vec<u8>)) {
    let OpTxEnvelope::Deposit(info) = bad.recovery[index].body.transactions[0].clone() else {
        panic!("recovery block starts with the L1-info deposit");
    };
    let mut info = info.into_inner();
    let mut calldata = info.input.to_vec();
    patch(&mut calldata);
    info.input = calldata.into();
    bad.recovery[index].body.transactions[0] = OpTxEnvelope::from(info);
}

/// N10: a canonical recovery block whose L1-info deposit names another L1 block (or carries other
/// L1 fields). Its private attributes are derived in-guest from L1, so the public block no longer
/// corresponds; and word 9 differs from the recovery hash the verifier computes from the canonical
/// chain.
#[test]
fn n10_recovery_l1_info_names_other_block() {
    let (input, witness, expected) = fixture_with_sdm(3, true);
    let l1 = l1_chain(epoch_of(5));
    // Isthmus/Jovian L1-info calldata: selector(4) ‖ scalars(8) ‖ seq(8) ‖ time(8) ‖ number(8) ‖
    // basefee(32) ‖ blob basefee(32) ‖ hash(32) ‖ ...
    let current = {
        let OpTxEnvelope::Deposit(info) = &input.recovery[1].body.transactions[0] else { panic!() };
        L1BlockInfoTx::decode_calldata(&info.inner().input).unwrap().id()
    };
    let other = l1[&(current.number + 1)].clone();

    // Another canonical L1 block: an epoch change, so the derived inputs include that block's
    // forced deposit and no longer match the public block's inputs.
    let mut bad = input.clone();
    patch_recovery_l1_info(&mut bad, 1, |c| {
        c[28..36].copy_from_slice(&other.number.to_be_bytes());
        c[100..132].copy_from_slice(other.hash_slow().as_slice());
    });
    let forged = rechain_recovery(&mut bad);
    assert_ne!(forged, public_value_word(&expected, W_RECOVERY_HASH), "N10: word 9");
    rejects("N10", &bad, &witness, "recovery input count");

    // The same L1 block with another L1 base fee: same inputs, but the L1-info deposit no longer
    // corresponds to the one derived from the authenticated L1 header.
    let mut bad = input;
    patch_recovery_l1_info(&mut bad, 1, |c| c[67] ^= 1);
    let forged = rechain_recovery(&mut bad);
    assert_ne!(forged, public_value_word(&expected, W_RECOVERY_HASH), "N10: word 9");
    rejects("N10", &bad, &witness, "recovery L1 info correspondence");
}

/// N10b: there is no prover-supplied attributes path left.
#[test]
fn n10b_attributes_field_rejected() {
    let (input, witness, _) = fixture(0);
    let mut value = serde_json::to_value((&input, &witness)).unwrap();
    value[0]["attributes"] = serde_json::json!([]);
    let e = execute_encoded(&serde_json::to_vec(&value).unwrap()).unwrap_err();
    assert!(e.to_string().contains("unknown field `attributes`"), "N10b: {e}");
}

// ---------------------------------------------------------------------------------------------
// §D.3 anchor

/// N11: the relation executes from whatever consistent checkpoint it is given; it is admission's
/// own `ContextCollector` result (words 6–9) that pins the anchor. Here the prover anchors at the
/// private genesis while the verifier's canonical checkpoint is the record at block 2.
#[test]
fn n11_older_anchor_rejected_by_admission() {
    let (input, witness, expected) = fixture(3);
    let pv = execute(&input, &witness).unwrap();
    assert_eq!(pv, expected);
    assert_eq!(
        U256::from_be_bytes(public_value_word(&pv, W_ANCHOR_NUMBER).0),
        U256::from(input.anchor.number)
    );
    assert_eq!(public_value_word(&pv, W_ANCHOR_HASH), input.anchor.hash);
    assert_eq!(public_value_word(&pv, W_ANCHOR_OUTPUT), input.anchor_output);

    // The verifier's view: a newer surviving checkpoint at block 2, followed by one recovery block.
    let newer = projection::Continuation {
        anchor: BlockNumHash { number: 2, hash: input.recovery[1].header.hash_slow() },
        output_root: B256::repeat_byte(0x77),
        recovery_hash: recovery_hash_of(&input.recovery[2..]),
    };
    let cfg = projection_cfg(&input);
    let mut blocks = input.blocks.clone();
    let mut claim = decode_claim(&input);
    claim.anchorBlock = newer.anchor.number;
    claim.anchorOutputRoot = newer.output_root;
    claim.recoveryHash = newer.recovery_hash;
    blocks[0].transactions[0] =
        signed(REGISTRY, projection::postClaimCall { claim }.abi_encode().into(), 0);
    let statement = admission_statement(&cfg, input.parent_hash, input.l1_head, newer, &blocks)
        .expect("the verifier's own span is admissible");
    let admitted = projection::public_values(&statement);
    let differ = differing_words(&pv, &admitted);
    for w in [W_ANCHOR_NUMBER, W_ANCHOR_HASH, W_ANCHOR_OUTPUT, W_RECOVERY_HASH] {
        assert!(differ.contains(&w), "N11: word {w} must differ, got {differ:?}");
    }
    let e = sp1_verify(&cfg, &statement, &mock_envelope(FIXTURE_VKEY, &pv)).unwrap_err();
    assert_eq!(e.0, "public values do not match the statement", "N11");
}

/// N12: `anchor_output` must be the anchor header's `OutputV0`.
#[test]
fn n12_anchor_output_mismatch() {
    for replacements in [0, 3] {
        let (input, witness, _) = fixture(replacements);
        let mut bad = input;
        bad.anchor_output = B256::repeat_byte(0x99);
        rejects("N12", &bad, &witness, "private anchor output");
    }
}

// ---------------------------------------------------------------------------------------------
// §D.4 program identity

/// N13: the vendored real SP1 v6.0.0 Groth16 fixture verifies under its own vkey and circuit, and
/// is rejected under our `program_vkey`.
#[test]
fn n13_real_groth16_fixture_wrong_vkey() {
    use kona_protocol::projection::sp1::{Circuit, verify_groth16};
    let fixture: serde_json::Value = serde_json::from_str(include_str!(
        "../../../../../../../op-private-interop/projection/sp1groth16/testdata/groth16-fixture-v6.0.0.json"
    ))
    .unwrap();
    let vk: &[u8] = include_bytes!(
        "../../../../../../../op-private-interop/projection/sp1groth16/testdata/groth16_vk_v6.0.0.bin"
    );
    let hexed = |k: &str| hex::decode(fixture[k].as_str().unwrap()).unwrap();
    let proof = hexed("proof");
    let public_values = hexed("publicValues");
    let vkey = B256::from_slice(&hexed("vkey"));
    let vk_root: [u8; 32] =
        hex!("008cd56e10c2fe24795cff1e1d1f40d3a324528d315674da45d26afb376e8670");
    let circuit = Circuit::new(vk, vk_root);
    verify_groth16(&circuit, &proof, vkey, &public_values)
        .expect("the fixture verifies under its own vkey");
    assert_ne!(vkey, FIXTURE_VKEY);
    verify_groth16(&circuit, &proof, FIXTURE_VKEY, &public_values)
        .expect_err("N13: a proof of another program is rejected under our program_vkey");
}

/// N14: a mock envelope over the genuine public values, under the wrong vkey word.
#[test]
fn n14_mock_envelope_wrong_vkey() {
    let (input, witness, expected) = fixture(0);
    let cfg = projection_cfg(&input);
    let pv = execute(&input, &witness).unwrap();
    let statement = statement_from_public_values(&expected);
    sp1_verify(&cfg, &statement, &mock_envelope(FIXTURE_VKEY, &pv)).expect("positive control");
    let e = sp1_verify(&cfg, &statement, &mock_envelope(B256::with_last_byte(2), &pv)).unwrap_err();
    assert_eq!(e.0, "mock proof vkey", "N14");
}

// ---------------------------------------------------------------------------------------------
// §D.5 / §D.6 completeness and outputs

/// Apply `mutate` to the published span; the relation must reject with `needle`, and admission,
/// holding a genuine envelope of the UNMODIFIED span, must reject the modified span at `words`
/// (the first differing word is the one the verifier reports).
fn completeness(
    id: &str,
    needle: &str,
    words: &[usize],
    mutate: impl Fn(&mut Vec<ProjectionBlock>),
) {
    for replacements in [0, 3] {
        let (input, witness, expected) = fixture(replacements);
        let cfg = projection_cfg(&input);
        let genuine = execute(&input, &witness).unwrap();
        assert_eq!(genuine, expected);

        let mut bad = input.clone();
        mutate(&mut bad.blocks);
        rejects(id, &bad, &witness, needle);

        let statement = admission_statement(
            &cfg,
            input.parent_hash,
            input.l1_head,
            canonical_continuation(&input),
            &bad.blocks,
        )
        .unwrap_or_else(|e| {
            panic!("{id}: the mutated span stays structurally admissible: {}", e.0)
        });
        let modified = projection::public_values(&statement);
        assert_eq!(differing_words(&genuine, &modified), words, "{id}: differing words");
        let e = sp1_verify(&cfg, &statement, &mock_envelope(FIXTURE_VKEY, &genuine)).unwrap_err();
        assert_eq!(e.0, "public values do not match the statement", "{id}");
    }
}

// Block 0 of every fixture: claim, record, export replay, import replay. Block 1: record.

#[test]
fn n15_drop_export_replay() {
    completeness("N15", "projection messages differ", &[W_PROJECTION_HASH, W_MESSAGES_ROOT], |b| {
        b[0].transactions.remove(2);
    });
}

#[test]
fn n16_extra_validate_message() {
    completeness("N16", "projection messages differ", &[W_PROJECTION_HASH, W_MESSAGES_ROOT], |b| {
        let extra = b[0].transactions[3].clone();
        b[0].transactions.push(extra);
        resign(&mut b[0]);
    });
}

#[test]
fn n17_swap_two_replays() {
    completeness("N17", "projection messages differ", &[W_PROJECTION_HASH, W_MESSAGES_ROOT], |b| {
        b[0].transactions.swap(2, 3);
        resign(&mut b[0]);
    });
}

#[test]
fn n18_move_replay_to_next_block() {
    completeness("N18", "projection messages differ", &[W_PROJECTION_HASH, W_MESSAGES_ROOT], |b| {
        let moved = b[0].transactions.remove(3);
        b[1].transactions.push(moved);
        resign(&mut b[1]);
    });
}

#[test]
fn n19_same_fields_other_message_bytes() {
    completeness("N19", "projection messages differ", &[W_PROJECTION_HASH, W_MESSAGES_ROOT], |b| {
        let tx = TxEnvelope::decode_2718_exact(&b[0].transactions[2]).unwrap();
        let mut call = projection::replaySentMessageCall::abi_decode_validate(tx.input()).unwrap();
        call.message = Bytes::from_static(b"private hellO");
        b[0].transactions[2] = signed(MESSENGER, call.abi_encode().into(), 2);
    });
}

/// N20: the fixtures' spans have two blocks, so the altered record is the last one; the terminal
/// output (word 20) therefore differs as well.
#[test]
fn n20_altered_output_record() {
    completeness(
        "N20",
        "published checkpoint differs from executed output",
        &[W_PROJECTION_HASH, W_OUTPUTS_ROOT, W_TERMINAL_OUTPUT],
        |b| {
            b[1].transactions[0] = signed(
                REGISTRY,
                projection::recordOutputCall { outputRoot: B256::repeat_byte(3) }
                    .abi_encode()
                    .into(),
                1,
            );
        },
    );
}

/// The harness's own positive control: genuine public values pass the admission verifier, and
/// equal the admission-computed expected values for every fixture shape.
#[test]
fn genuine_fixtures_admitted() {
    for fixture in [fixture(0), fixture(1), fixture(3), fixture_with_sdm(3, true)] {
        let (input, witness, expected) = fixture;
        let cfg = projection_cfg(&input);
        let pv = execute(&input, &witness).unwrap();
        assert_eq!(pv, expected);
        assert_eq!(expected, expected_public_values(&input, &cfg));
        let statement = admission_statement(
            &cfg,
            input.parent_hash,
            input.l1_head,
            canonical_continuation(&input),
            &input.blocks,
        )
        .unwrap();
        sp1_verify(&cfg, &statement, &mock_envelope(FIXTURE_VKEY, &pv)).unwrap();
        assert_eq!(public_value_word(&pv, W_L1_HEAD), input.l1_head);
    }
}
