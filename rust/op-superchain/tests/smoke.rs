//! Smoke tests for the public lookup surface.

use serde_json::Value;

#[test]
fn known_chains_resolve() {
    // OP Mainnet — chain id 10, has been in the registry since day one.
    assert!(op_superchain::config_str("op", "mainnet").is_some(), "op/mainnet config missing");
    assert!(op_superchain::genesis_bytes("op", "mainnet").is_some(), "op/mainnet genesis missing");
    // Unichain — used by the existing op-reth tests.
    assert!(
        op_superchain::config_str("unichain", "mainnet").is_some(),
        "unichain/mainnet config missing"
    );
}

#[test]
fn unknown_chains_return_none() {
    assert!(op_superchain::config_str("nonexistent", "mainnet").is_none());
    assert!(op_superchain::genesis_bytes("nonexistent", "mainnet").is_none());
}

#[test]
fn config_str_is_valid_json() {
    let cfg = op_superchain::config_str("op", "mainnet").expect("op/mainnet config missing");
    let parsed: Value = serde_json::from_str(cfg).expect("parse op-mainnet config");
    assert_eq!(parsed.as_object().unwrap().get("chain_id").and_then(Value::as_u64), Some(10));
}

#[test]
fn chain_list_is_valid_json() {
    let parsed: Value =
        serde_json::from_str(op_superchain::CHAIN_LIST_JSON).expect("parse chainList.json");
    assert!(parsed.is_array() || parsed.is_object(), "chainList.json should be array or object");
}

#[test]
fn depsets_is_valid_json_array() {
    let parsed: Value =
        serde_json::from_str(op_superchain::DEPSETS_JSON).expect("parse depsets.json");
    assert!(parsed.is_array(), "depsets.json must be an array");
}

#[test]
fn superchains_aggregate_has_expected_shape() {
    let parsed: Value =
        serde_json::from_str(op_superchain::SUPERCHAINS_JSON).expect("parse superchains.json");
    let superchains = parsed
        .as_object()
        .and_then(|o| o.get("superchains"))
        .and_then(Value::as_array)
        .expect("superchains.superchains array");
    assert!(!superchains.is_empty(), "superchains array must be non-empty");
    let mainnet = superchains
        .iter()
        .find(|s| s.get("name").and_then(Value::as_str) == Some("mainnet"))
        .expect("mainnet superchain present");
    assert!(mainnet.get("config").is_some());
    assert!(mainnet.get("chains").and_then(Value::as_array).is_some_and(|v| !v.is_empty()));
}

#[test]
fn supported_chains_matches_lookup() {
    for (name, env) in op_superchain::supported_chains() {
        assert!(
            op_superchain::config_str(name, env).is_some(),
            "supported chain {name}/{env} has no config_str"
        );
        assert!(
            op_superchain::genesis_bytes(name, env).is_some(),
            "supported chain {name}/{env} has no genesis_bytes"
        );
    }
}
