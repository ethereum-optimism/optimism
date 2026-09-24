//! The fault-proof program admits `sp1-private-projection-v1` spans (review P1-1).
//!
//! kona-proof enables kona-protocol's `sp1-projection-verifier` itself. The only dev-dependency
//! feature added here is `private-projection-test-verifiers` (for the mock envelope), so without
//! kona-proof's own feature this test fails with "sp1 projection verifier not compiled", as every
//! sp1 span did in kona-client before.

use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, Bytes};
use kona_genesis::{ChainGenesis, PrivateProjectionConfig, RollupConfig};
use kona_protocol::{
    SpanBatch, SpanBatchElement,
    projection::{
        ConfiguredVerifier, Continuation, ProjectionContext, SP1_PROJECTION_VERIFIER_COMPILED,
        validate_projection_range,
    },
};
use serde_json::Value;

const PROOFS: &str =
    include_str!("../../../../../../op-private-interop/projection/testdata/proofs.json");

fn h(v: &Value) -> B256 {
    if v.is_null() { B256::ZERO } else { serde_json::from_value(v.clone()).unwrap() }
}

/// The admission inputs of one shared vector, as span-batch validation builds them.
fn inputs(v: &Value) -> (RollupConfig, SpanBatch, ProjectionContext) {
    let (c, x) = (&v["config"], &v["context"]);
    let cfg = RollupConfig {
        l2_chain_id: x["chain_id"].as_u64().unwrap().into(),
        block_time: x["block_time"].as_u64().unwrap(),
        genesis: ChainGenesis {
            l2_time: x["genesis_time"].as_u64().unwrap(),
            l2: BlockNumHash {
                number: x["genesis_number"].as_u64().unwrap(),
                hash: h(&x["genesis_hash"]),
            },
            ..Default::default()
        },
        private_projection: Some(PrivateProjectionConfig {
            verifier: c["verifier"].as_str().unwrap().into(),
            allow_events: c["allow_events"].as_bool().unwrap_or(false),
            genesis_output_root: h(&c["genesis_output_root"]),
            program_vkey: h(&c["program_vkey"]),
            private_config_hash: h(&c["private_config_hash"]),
            dependency_set_hash: h(&c["dependency_set_hash"]),
            mock_proofs: c["mock_proofs"].as_bool().unwrap_or(false),
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
                            .map(|t| serde_json::from_value::<Bytes>(t.clone()).unwrap())
                            .collect()
                    })
                    .unwrap_or_default(),
            })
            .collect(),
        ..Default::default()
    };
    let k = &x["continuation"];
    let ctx = ProjectionContext {
        parent_hash: h(&x["parent_hash"]),
        l1_head: h(&x["l1_head"]),
        continuation: Continuation {
            anchor: BlockNumHash {
                number: k["anchor"]["number"].as_u64().unwrap(),
                hash: h(&k["anchor"]["hash"]),
            },
            output_root: h(&k["output_root"]),
            recovery_hash: h(&k["recovery_hash"]),
        },
    };
    (cfg, span, ctx)
}

#[test]
fn fault_proof_program_admits_sp1_projection_spans() {
    assert!(
        std::hint::black_box(SP1_PROJECTION_VERIFIER_COMPILED),
        "kona-proof must enable kona-protocol/sp1-projection-verifier itself"
    );
    let cases: Vec<Value> = serde_json::from_str(PROOFS).unwrap();
    let mut checked = 0;
    for v in &cases {
        let name = v["name"].as_str().unwrap();
        if v["config"]["verifier"] != "sp1-private-projection-v1" {
            continue;
        }
        let (cfg, span, ctx) = inputs(v);
        // Exactly the verifier span-batch validation installs (`SpanBatch::check_batch_holocene`).
        let verifier = ConfiguredVerifier {
            profile: cfg.private_projection.as_ref().unwrap(),
            chain_id: cfg.l2_chain_id.id(),
        };
        let result = validate_projection_range(&cfg, ctx, &span, &verifier);
        assert_eq!(result.is_ok(), v["accept"].as_bool().unwrap(), "{name}: {result:?}");
        checked += 1;
    }
    assert!(checked > 20, "the sp1 proof vectors");
    let valid = cases.iter().find(|v| v["name"] == "sp1_mock_valid").unwrap();
    let (cfg, span, ctx) = inputs(valid);
    let verifier = ConfiguredVerifier {
        profile: cfg.private_projection.as_ref().unwrap(),
        chain_id: cfg.l2_chain_id.id(),
    };
    validate_projection_range(&cfg, ctx, &span, &verifier).expect("sp1_mock_valid is admitted");
}
