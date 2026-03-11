//! Tests for the L1 RPC, L2 RPC, and Beacon API servers.

use derivation_tests::{config::DeterministicConfig, harness::DerivationTest};

fn build_empty_blocks_test(l2_count: usize) -> DerivationTest {
    let mut test = DerivationTest::new();
    for _ in 0..l2_count {
        test.l1.emit_empty_block();
    }
    let l1_block = test.l1.block_at(1).unwrap().clone();
    test.l2.set_epoch(&l1_block);
    for _ in 0..l2_count {
        test.l2.build_empty_block().unwrap();
    }
    test
}

#[tokio::test]
async fn test_l1_rpc_get_block_by_hash() {
    let test = build_empty_blocks_test(2);
    let servers = test.serve().await.unwrap();

    let client = reqwest::Client::new();

    let resp: serde_json::Value = client
        .post(servers.l1_rpc_url())
        .json(&serde_json::json!({
            "jsonrpc": "2.0",
            "method": "eth_getBlockByNumber",
            "params": ["0x0", false],
            "id": 1
        }))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();

    let block_by_number = resp["result"].clone();
    let hash = block_by_number["hash"].as_str().unwrap().to_string();

    let resp: serde_json::Value = client
        .post(servers.l1_rpc_url())
        .json(&serde_json::json!({
            "jsonrpc": "2.0",
            "method": "eth_getBlockByHash",
            "params": [hash, false],
            "id": 2
        }))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();

    let block_by_hash = resp["result"].clone();
    assert_eq!(
        block_by_number["hash"], block_by_hash["hash"],
        "block by number and block by hash should match"
    );
    assert_eq!(block_by_number["number"], block_by_hash["number"]);

    servers.stop();
}

#[tokio::test]
async fn test_l2_rpc_get_proof() {
    use derivation_tests::config::L2_TO_L1_MESSAGE_PASSER;

    let test = build_empty_blocks_test(1);
    let servers = test.serve().await.unwrap();

    let client = reqwest::Client::new();

    let resp: serde_json::Value = client
        .post(servers.l2_rpc_url())
        .json(&serde_json::json!({
            "jsonrpc": "2.0",
            "method": "eth_getProof",
            "params": [format!("{:?}", L2_TO_L1_MESSAGE_PASSER), [], "latest"],
            "id": 1
        }))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();

    let result = &resp["result"];
    assert!(result.is_object(), "proof response should be an object");
    assert!(result["address"].is_string(), "proof should have address field");
    assert!(result["accountProof"].is_array(), "proof should have accountProof");
    assert!(result["storageProof"].is_array(), "proof should have storageProof");

    servers.stop();
}

#[tokio::test]
async fn test_debug_db_get() {
    let test = build_empty_blocks_test(1);
    let servers = test.serve().await.unwrap();

    let client = reqwest::Client::new();

    let state_root = test.l2.head().header.inner().state_root;

    let resp: serde_json::Value = client
        .post(servers.l2_rpc_url())
        .json(&serde_json::json!({
            "jsonrpc": "2.0",
            "method": "debug_dbGet",
            "params": [format!("{state_root:?}")],
            "id": 1
        }))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();

    assert!(
        resp["result"].is_string(),
        "debug_dbGet should return data for the state root trie node, got: {resp}"
    );
    let result_hex = resp["result"].as_str().unwrap();
    assert!(result_hex.len() > 2, "result should be non-empty hex data");

    servers.stop();
}

#[tokio::test]
async fn test_beacon_blobs() {
    let config = DeterministicConfig::default();
    let mut test = DerivationTest::new();

    test.l1.emit_empty_block();
    test.l1.emit_empty_block();

    let l1_block = test.l1.block_at(1).unwrap().clone();
    test.l2.set_epoch(&l1_block);
    let block_ref = test.l2.build_empty_block().unwrap();

    let l1_origin = test.l1.block_at(1).unwrap().clone();
    let batch = test.singular_batch_calldata(&[block_ref], &l1_origin);
    test.l1.emit_block_with_batches(vec![batch]);

    let servers = test.serve().await.unwrap();

    let client = reqwest::Client::new();

    let resp: serde_json::Value = client
        .get(format!("{}/eth/v1/beacon/blobs/0", servers.beacon_url()))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();

    assert!(resp["data"].is_array(), "beacon blobs response should have data array");
    assert_eq!(resp["data"].as_array().unwrap().len(), 0);

    let resp: serde_json::Value = client
        .get(format!("{}/eth/v1/beacon/genesis", servers.beacon_url()))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();

    assert!(resp["data"]["genesis_time"].is_string());
    let genesis_time: u64 = resp["data"]["genesis_time"].as_str().unwrap().parse().unwrap();
    assert_eq!(genesis_time, config.genesis_timestamp);

    servers.stop();
}
