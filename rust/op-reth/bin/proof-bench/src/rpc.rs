use crate::utils::{CONTRACT, balance_of_slot};
use alloy_primitives::Address;
use serde::{Deserialize, Serialize};
use serde_json::json;
use std::time::Instant;

#[derive(Debug, Serialize, Deserialize)]
struct RpcResponse {
    jsonrpc: String,
    id: usize,
    result: Option<serde_json::Value>,
    error: Option<RpcError>,
}

#[derive(Debug, Serialize, Deserialize)]
struct RpcError {
    code: i32,
    message: String,
}

pub struct Sample {
    pub latency_ms: f64,
    pub success: bool,
}

pub async fn run_proof(
    client: reqwest::Client,
    url: String,
    block: u64,
    id: usize,
    addr: Address,
) -> Sample {
    let start = Instant::now();
    let slot = balance_of_slot(addr);

    // Format hash with 0x used by alloy B256 debug/display
    let params = json!([CONTRACT, [format!("{}", slot)], format!("0x{:x}", block)]);

    let body = json!({
        "jsonrpc": "2.0",
        "id": id,
        "method": "eth_getProof",
        "params": params
    });

    let resp = client.post(&url).json(&body).send().await;

    let latency = start.elapsed().as_secs_f64() * 1000.0;

    let success = match resp {
        Ok(res) => match res.json::<RpcResponse>().await {
            Ok(rpc_resp) => rpc_resp.error.is_none() && rpc_resp.result.is_some(),
            Err(_) => false,
        },
        Err(_) => false,
    };

    Sample { latency_ms: latency, success }
}
