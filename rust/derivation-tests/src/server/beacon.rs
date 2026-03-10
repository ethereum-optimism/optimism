//! Beacon API server for serving blob data.
//!
//! Note: We use hand-crafted `serde_json::json!()` responses instead of
//! `alloy_rpc_types_beacon` typed responses (e.g. `BlobData`, `GetBlobsResponse`).
//! The typed responses require fields we don't have in our test context:
//! - `BlobData` requires `kzg_commitment_inclusion_proof` (Merkle proof into the beacon block)
//!   and a full `BeaconBlockHeader` with `proposer_index`, `parent_root`, `state_root`, `body_root`
//! - `GetBlobsResponse` requires `execution_optimistic` and `finalized` boolean fields
//!
//! Fabricating these would add complexity without improving correctness, since
//! the OP Stack derivation pipeline only reads `index`, `blob`, `kzg_commitment`,
//! `kzg_proof`, and `signed_block_header.message.slot` from the response.

use axum::{
    Router,
    extract::{Path, Query, State},
    response::Json,
    routing::get,
};
use serde::Deserialize;
use std::{collections::BTreeMap, sync::Arc};

use crate::{config::DeterministicConfig, l1::BlobWithCommitment};

/// Shared state for the beacon API server.
pub(crate) struct BeaconState {
    /// Configuration.
    pub(crate) config: DeterministicConfig,
    /// Blobs indexed by slot.
    pub(crate) blobs: BTreeMap<u64, Vec<BlobWithCommitment>>,
}

/// Build the beacon API router.
pub(crate) fn beacon_router(state: Arc<BeaconState>) -> Router {
    Router::new()
        .route("/eth/v1/config/spec", get(config_spec))
        .route("/eth/v1/beacon/genesis", get(beacon_genesis))
        .route("/eth/v1/beacon/blobs/{slot}", get(beacon_blobs))
        .with_state(state)
}

async fn config_spec(State(state): State<Arc<BeaconState>>) -> Json<serde_json::Value> {
    Json(serde_json::json!({
        "data": {
            "SECONDS_PER_SLOT": state.config.seconds_per_slot.to_string(),
        }
    }))
}

async fn beacon_genesis(State(state): State<Arc<BeaconState>>) -> Json<serde_json::Value> {
    Json(serde_json::json!({
        "data": {
            "genesis_time": state.config.genesis_timestamp.to_string(),
        }
    }))
}

/// Query parameters for the blobs endpoint.
#[derive(Deserialize)]
struct BlobsQuery {
    /// Comma-separated versioned hashes to filter by.
    #[serde(default)]
    versioned_hashes: Option<String>,
}

async fn beacon_blobs(
    State(state): State<Arc<BeaconState>>,
    Path(slot): Path<u64>,
    Query(query): Query<BlobsQuery>,
) -> Json<serde_json::Value> {
    let blobs = state.blobs.get(&slot);

    let data: Vec<serde_json::Value> = match blobs {
        Some(slot_blobs) => {
            let filter_hashes: Option<Vec<String>> = query
                .versioned_hashes
                .as_ref()
                .map(|h| h.split(',').map(|s| s.trim().to_string()).collect());

            slot_blobs
                .iter()
                .enumerate()
                .filter(|(_, blob)| {
                    filter_hashes
                        .as_ref()
                        .is_none_or(|hashes| hashes.contains(&format!("{:?}", blob.versioned_hash)))
                })
                .map(|(i, blob)| {
                    serde_json::json!({
                        "index": i.to_string(),
                        "blob": format!("0x{}", alloy_primitives::hex::encode(*blob.blob)),
                        "kzg_commitment": blob.commitment.commitments[0],
                        "kzg_proof": blob.commitment.proofs[0],
                        "signed_block_header": {
                            "message": {
                                "slot": slot.to_string(),
                            }
                        },
                    })
                })
                .collect()
        }
        None => vec![],
    };

    Json(serde_json::json!({
        "data": data
    }))
}
