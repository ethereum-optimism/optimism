//! Contains an online implementation of the `BeaconClient` trait.

#[cfg(feature = "metrics")]
use crate::Metrics;
use crate::blobs::BoxedBlob;
use alloy_eips::eip4844::{
    Blob as AlloyBlob, deserialize_blob, env_settings::EnvKzgSettings, kzg_to_versioned_hash,
};
use alloy_primitives::{B256, FixedBytes};
use async_trait::async_trait;
use c_kzg::Blob;
use reqwest::{self, Client};
use std::{boxed::Box, format, string::String, vec::Vec};
use thiserror::Error;

/// The config spec engine api method.
const SPEC_METHOD: &str = "eth/v1/config/spec";

/// The beacon genesis engine api method.
const GENESIS_METHOD: &str = "eth/v1/beacon/genesis";

/// The blobs engine api method prefix.
const BLOBS_METHOD_PREFIX: &str = "eth/v1/beacon/blobs";

/// A reduced genesis data.
#[derive(Debug, Default, Clone, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub struct ReducedGenesisData {
    /// The genesis time.
    #[serde(rename = "genesis_time")]
    #[serde(with = "alloy_serde::quantity")]
    pub genesis_time: u64,
}

/// An API genesis response.
#[derive(Debug, Default, Clone, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub struct APIGenesisResponse {
    /// The data.
    pub data: ReducedGenesisData,
}

/// A reduced config data.
///
/// A field missing from the spec deserializes as zero.
#[derive(Debug, Default, Clone, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub struct ReducedConfigData {
    /// The seconds per slot.
    #[serde(rename = "SECONDS_PER_SLOT", default)]
    #[serde(with = "alloy_serde::quantity")]
    pub seconds_per_slot: u64,
    /// The slot duration in milliseconds.
    #[serde(rename = "SLOT_DURATION_MS", default)]
    #[serde(with = "alloy_serde::quantity")]
    pub slot_duration_ms: u64,
}

impl ReducedConfigData {
    /// Returns the slot duration in seconds from `SLOT_DURATION_MS`, falling back to
    /// `SECONDS_PER_SLOT`. A zero value counts as absent. `SLOT_DURATION_MS` must be a whole
    /// number of seconds, because callers map L1 block timestamps, which have second
    /// granularity, to slots.
    pub const fn slot_duration_secs(&self) -> Result<u64, SlotDurationError> {
        match (self.slot_duration_ms, self.seconds_per_slot) {
            (0, 0) => Err(SlotDurationError::Missing),
            (0, secs) => Ok(secs),
            (ms, _) if !ms.is_multiple_of(1000) => Err(SlotDurationError::NotWholeSeconds(ms)),
            (ms, _) => Ok(ms / 1000),
        }
    }
}

/// An API config response.
#[derive(Debug, Default, Clone, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub struct APIConfigResponse {
    /// The data.
    pub data: ReducedConfigData,
}

impl APIConfigResponse {
    /// Creates a new API config response.
    pub const fn new(seconds_per_slot: u64) -> Self {
        Self { data: ReducedConfigData { seconds_per_slot, slot_duration_ms: 0 } }
    }
}

impl APIGenesisResponse {
    /// Creates a new API genesis response.
    pub const fn new(genesis_time: u64) -> Self {
        Self { data: ReducedGenesisData { genesis_time } }
    }
}

/// The [`BeaconClient`] is a thin wrapper around the Beacon API.
#[async_trait]
pub trait BeaconClient {
    /// The error type for [`BeaconClient`] implementations.
    type Error: core::fmt::Display;

    /// Returns the slot number if this error represents a beacon slot not found (HTTP 404).
    ///
    /// Returns `None` for all other error kinds. This allows the blob provider to distinguish
    /// permanently-unavailable slots (missed/orphaned beacon blocks) from transient errors,
    /// and trigger a pipeline reset instead of retrying indefinitely.
    fn slot_not_found(err: &Self::Error) -> Option<u64>;

    /// Returns the beacon config spec carrying the slot duration.
    async fn slot_interval(&self) -> Result<APIConfigResponse, Self::Error>;

    /// Returns the beacon genesis time.
    async fn genesis_time(&self) -> Result<APIGenesisResponse, Self::Error>;

    /// Fetches blobs that were confirmed in the specified L1 block with the given slot.
    /// Blob data is checked for validity.
    async fn filtered_beacon_blobs(
        &self,
        slot: u64,
        blob_hashes: &[B256],
    ) -> Result<Vec<BoxedBlob>, Self::Error>;
}

const BLOB_SIZE: usize = 131072;

/// A beacon API blob stored as a boxed value at the HTTP boundary.
///
/// `alloy_rpc_types_beacon::sidecar::GetBlobsResponse` stores blobs inline in a `Vec<Blob>`.
/// Deserializing the 128 KiB fixed-size values through serde can exhaust a Tokio worker's stack.
#[derive(Debug, serde::Deserialize)]
struct BoxedBeaconBlob(#[serde(deserialize_with = "deserialize_blob")] Box<AlloyBlob>);

/// The subset of the beacon API blob response used by the online provider.
#[derive(Debug, serde::Deserialize)]
struct BoxedGetBlobsResponse {
    data: Vec<BoxedBeaconBlob>,
}

/// A versioned hash paired with the boxed blob allocation that produced it.
struct HashedBlob {
    versioned_hash: B256,
    blob: Box<AlloyBlob>,
}

/// [`blob_versioned_hash`] computes the versioned hash of a blob.
fn blob_versioned_hash(blob: &FixedBytes<BLOB_SIZE>) -> Result<B256, BeaconClientError> {
    let kzg_settings = EnvKzgSettings::Default;
    let kzg_blob = Blob::new(blob.0);
    let commitment = kzg_settings.get().blob_to_kzg_commitment(&kzg_blob)?;
    Ok(kzg_to_versioned_hash(commitment.as_slice()))
}

/// An error that can occur when interacting with the beacon client.
#[derive(Error, Debug)]
pub enum BeaconClientError {
    /// HTTP request failed.
    #[error("HTTP request failed: {0}")]
    Http(#[from] reqwest::Error),

    /// The beacon node returned HTTP 404 for the requested slot. This means the slot was missed
    /// or orphaned and the blobs will never be available.
    #[error("Beacon slot not found (HTTP 404) for slot {0}")]
    SlotNotFound(u64),

    /// Blob hash not found in beacon response.
    #[error("Blob hash not found in beacon response: {0}")]
    BlobNotFound(String),

    /// KZG error.
    #[error("KZG error: {0}")]
    KZG(#[from] c_kzg::Error),
}

/// An error deriving the L1 slot duration from a beacon config spec.
#[derive(Error, Debug, Clone, Copy, PartialEq, Eq)]
pub enum SlotDurationError {
    /// The spec carries neither a non-zero `SLOT_DURATION_MS` nor `SECONDS_PER_SLOT`.
    #[error("beacon spec has neither SLOT_DURATION_MS nor SECONDS_PER_SLOT")]
    Missing,
    /// `SLOT_DURATION_MS` is not a whole number of seconds.
    #[error("beacon spec SLOT_DURATION_MS {0} is not a whole number of seconds")]
    NotWholeSeconds(u64),
}

/// An online implementation of the [`BeaconClient`] trait.
#[derive(Debug, Clone)]
pub struct OnlineBeaconClient {
    /// The base URL of the beacon API.
    pub base: String,
    /// The inner reqwest client.
    pub inner: Client,
    /// The duration in seconds of an L1 slot. This can be used to override the CL slot
    /// duration if the l1-beacon's slot configuration endpoint is not available.
    pub l1_slot_duration: Option<u64>,
}

impl OnlineBeaconClient {
    /// Creates a new [`OnlineBeaconClient`] from the provided base URL string.
    pub fn new_http(mut base: String) -> Self {
        // If base ends with a slash, remove it
        if base.ends_with('/') {
            base.remove(base.len() - 1);
        }
        Self {
            base,
            inner: Client::builder().build().expect("Failed to create beacon client"),
            l1_slot_duration: None,
        }
    }

    /// Sets the duration in seconds of an L1 slot. This can be used to override the CL slot
    /// duration if the l1-beacon's slot configuration endpoint is not available.
    pub const fn with_l1_slot_duration_override(mut self, l1_slot_duration: u64) -> Self {
        self.l1_slot_duration = Some(l1_slot_duration);
        self
    }

    /// Fetches only the blobs corresponding to the provided (versioned) blob hashes
    /// from the beacon [`BLOBS_METHOD_PREFIX`] endpoint.
    /// Blobs are validated against the supplied versioned hashes
    /// and returned in the same order as the input.
    async fn filtered_beacon_blobs(
        &self,
        slot: u64,
        blob_hashes: &[B256],
    ) -> Result<Vec<BoxedBlob>, BeaconClientError> {
        // The Beacon API defines `versioned_hashes` as a query array:
        // https://github.com/ethereum/beacon-APIs/blob/e20dfabd6230a3e0de8a8964fee7a4f276e480d6/apis/beacon/blobs/blobs.yaml#L20-L28
        // Encode its default exploded form with one query parameter per hash. Some clients reject
        // a single comma-separated value. The array also requires unique items, but a block may
        // reference the same blob more than once, so only request each distinct hash once.
        let params = blob_hashes
            .iter()
            .enumerate()
            .filter(|(index, hash)| !blob_hashes[..*index].contains(hash))
            .map(|(_, hash)| ("versioned_hashes", hash.to_string()))
            .collect::<Vec<_>>();
        let response = self
            .inner
            .get(format!("{}/{}/{}", self.base, BLOBS_METHOD_PREFIX, slot))
            .query(&params)
            .send()
            .await?;

        // A 404 means the beacon slot was missed or orphaned. Blobs for such slots will never
        // become available, so surface this as a distinct error rather than a generic HTTP error
        // so that callers can trigger a pipeline reset instead of retrying indefinitely.
        if response.status() == reqwest::StatusCode::NOT_FOUND {
            return Err(BeaconClientError::SlotNotFound(slot));
        }

        let response = response.error_for_status()?;
        let bundle = response.json::<BoxedGetBlobsResponse>().await?;

        let mut returned_blobs = bundle
            .data
            .into_iter()
            .map(|data| -> Result<_, BeaconClientError> {
                let versioned_hash = blob_versioned_hash(&data.0)?;
                Ok(HashedBlob { versioned_hash, blob: data.0 })
            })
            .collect::<Result<Vec<_>, BeaconClientError>>()?;

        // Map the input blob hashes into the output while moving each blob's existing allocation.
        // The request only names each distinct hash once, so a repeated input hash reuses a copy
        // of the blob already matched for its first occurrence.
        let mut blobs = Vec::<BoxedBlob>::with_capacity(blob_hashes.len());
        for (index, blob_hash) in blob_hashes.iter().enumerate() {
            let blob = if let Some(position) =
                returned_blobs.iter().position(|candidate| candidate.versioned_hash == *blob_hash)
            {
                returned_blobs.swap_remove(position).blob
            } else if let Some(first) = blob_hashes[..index].iter().position(|h| h == blob_hash) {
                blobs[first].blob.clone()
            } else {
                return Err(BeaconClientError::BlobNotFound(blob_hash.to_string()));
            };
            blobs.push(BoxedBlob { blob });
        }
        Ok(blobs)
    }
}

#[async_trait]
impl BeaconClient for OnlineBeaconClient {
    type Error = BeaconClientError;

    fn slot_not_found(err: &Self::Error) -> Option<u64> {
        if let BeaconClientError::SlotNotFound(slot) = err { Some(*slot) } else { None }
    }

    async fn slot_interval(&self) -> Result<APIConfigResponse, Self::Error> {
        kona_macros::inc!(gauge, Metrics::BEACON_CLIENT_REQUESTS, "method" => "spec");

        // Use the l1 slot duration if provided
        if let Some(l1_slot_duration) = self.l1_slot_duration {
            return Ok(APIConfigResponse::new(l1_slot_duration));
        }

        let result = async {
            let first = self.inner.get(format!("{}/{}", self.base, SPEC_METHOD)).send().await?;
            first.json::<APIConfigResponse>().await
        }
        .await;

        if result.is_err() {
            kona_macros::inc!(gauge, Metrics::BEACON_CLIENT_ERRORS, "method" => "spec");
        }

        Ok(result?)
    }

    async fn genesis_time(&self) -> Result<APIGenesisResponse, Self::Error> {
        kona_macros::inc!(gauge, Metrics::BEACON_CLIENT_REQUESTS, "method" => "genesis");

        let result = async {
            let first = self.inner.get(format!("{}/{}", self.base, GENESIS_METHOD)).send().await?;
            first.json::<APIGenesisResponse>().await
        }
        .await;

        if result.is_err() {
            kona_macros::inc!(gauge, Metrics::BEACON_CLIENT_ERRORS, "method" => "genesis");
        }

        Ok(result?)
    }

    async fn filtered_beacon_blobs(
        &self,
        slot: u64,
        blob_hashes: &[B256],
    ) -> Result<Vec<BoxedBlob>, BeaconClientError> {
        kona_macros::inc!(gauge, Metrics::BEACON_CLIENT_REQUESTS, "method" => "blobs");

        // Try to get the blobs from the blobs endpoint.
        let result = self.filtered_beacon_blobs(slot, blob_hashes).await;

        if result.is_err() {
            kona_macros::inc!(gauge, Metrics::BEACON_CLIENT_ERRORS, "method" => "blobs");
        }

        result
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::Blob;
    use alloy_primitives::{FixedBytes, hex::FromHex};
    use httpmock::prelude::*;
    use serde_json::json;

    const TEST_SLOT: u64 = 987654321;
    const TEST_BLOB_A: Blob = FixedBytes::repeat_byte(1);
    const TEST_BLOB_A_HASH_HEX: &str =
        "0x016c357b8b3a6b3fd82386e7bebf77143d537cdb1c856509661c412602306a04";

    struct BlobResponseTest {
        requested_blob_hashes: Vec<B256>,
        queried_blob_hashes: Vec<B256>,
        response_data: Vec<Blob>,
    }

    impl BlobResponseTest {
        async fn run(self) -> Result<Vec<BoxedBlob>, BeaconClientError> {
            let server = MockServer::start();
            let blobs_mock = server.mock(|when, then| {
                self.queried_blob_hashes.iter().fold(
                    when.method(GET)
                        .path(format!("/eth/v1/beacon/blobs/{TEST_SLOT}"))
                        .query_param_count(
                            "^versioned_hashes$",
                            ".*",
                            self.queried_blob_hashes.len(),
                        ),
                    |when, hash| when.query_param("versioned_hashes", hash.to_string()),
                );
                then.status(200).json_body(json!({
                    "execution_optimistic": false,
                    "finalized": false,
                    "data": self.response_data,
                }));
            });

            let client = OnlineBeaconClient::new_http(server.base_url());
            let response =
                client.filtered_beacon_blobs(TEST_SLOT, &self.requested_blob_hashes).await;
            blobs_mock.assert();
            response
        }
    }

    #[test]
    fn test_blob_versioned_hash() {
        let test_blob_hash: FixedBytes<32> = FixedBytes::from_hex(TEST_BLOB_A_HASH_HEX).unwrap();
        assert_eq!(test_blob_hash, blob_versioned_hash(&TEST_BLOB_A).unwrap());
    }

    #[test]
    fn test_filtered_beacon_blobs_deserializes_on_small_stack() {
        let test_blob_hash = B256::from_hex(TEST_BLOB_A_HASH_HEX).unwrap();
        let server = MockServer::start();
        let blobs_mock = server.mock(|when, then| {
            when.method(GET)
                .path(format!("/eth/v1/beacon/blobs/{TEST_SLOT}"))
                .query_param("versioned_hashes", TEST_BLOB_A_HASH_HEX);
            then.status(200).json_body(json!({
                "execution_optimistic": false,
                "finalized": false,
                "data": [TEST_BLOB_A],
            }));
        });
        let base_url = server.base_url();

        let blobs = std::thread::Builder::new()
            .stack_size(1024 * 1024)
            .spawn(move || {
                tokio::runtime::Builder::new_current_thread()
                    .enable_all()
                    .build()
                    .unwrap()
                    .block_on(async move {
                        OnlineBeaconClient::new_http(base_url)
                            .filtered_beacon_blobs(TEST_SLOT, &[test_blob_hash])
                            .await
                            .unwrap()
                    })
            })
            .unwrap()
            .join()
            .unwrap();

        blobs_mock.assert();
        assert_eq!(blobs, vec![BoxedBlob { blob: Box::new(TEST_BLOB_A) }]);
    }

    #[tokio::test]
    async fn test_filtered_beacon_blobs_matches_requested_order() {
        let blob_b = FixedBytes::repeat_byte(2);
        let extra_blob = FixedBytes::repeat_byte(3);
        let blob_a_hash = B256::from_hex(TEST_BLOB_A_HASH_HEX).unwrap();
        let blob_b_hash = blob_versioned_hash(&blob_b).unwrap();

        let blobs = BlobResponseTest {
            requested_blob_hashes: vec![blob_a_hash, blob_b_hash],
            queried_blob_hashes: vec![blob_a_hash, blob_b_hash],
            response_data: vec![blob_b, extra_blob, TEST_BLOB_A],
        }
        .run()
        .await
        .unwrap();

        assert_eq!(
            blobs,
            vec![BoxedBlob { blob: Box::new(TEST_BLOB_A) }, BoxedBlob { blob: Box::new(blob_b) },]
        );
    }

    #[tokio::test]
    async fn test_filtered_beacon_blobs_deduplicates_query_and_preserves_duplicates() {
        let blob_hash = B256::from_hex(TEST_BLOB_A_HASH_HEX).unwrap();
        let blobs = BlobResponseTest {
            requested_blob_hashes: vec![blob_hash, blob_hash],
            queried_blob_hashes: vec![blob_hash],
            response_data: vec![TEST_BLOB_A],
        }
        .run()
        .await
        .unwrap();

        assert_eq!(
            blobs,
            vec![
                BoxedBlob { blob: Box::new(TEST_BLOB_A) },
                BoxedBlob { blob: Box::new(TEST_BLOB_A) },
            ]
        );
    }

    #[tokio::test]
    async fn test_filtered_beacon_blobs_reports_missing_blob() {
        let blob_a_hash = B256::from_hex(TEST_BLOB_A_HASH_HEX).unwrap();
        let blob_b_hash = blob_versioned_hash(&FixedBytes::repeat_byte(2)).unwrap();
        let response = BlobResponseTest {
            requested_blob_hashes: vec![blob_a_hash, blob_b_hash, blob_b_hash],
            queried_blob_hashes: vec![blob_a_hash, blob_b_hash],
            response_data: vec![TEST_BLOB_A],
        }
        .run()
        .await;
        let expected_hash = blob_b_hash.to_string();

        assert!(
            matches!(
                &response,
                Err(BeaconClientError::BlobNotFound(missing_hash))
                    if missing_hash == &expected_hash
            ),
            "expected BlobNotFound({expected_hash}), got {response:?}"
        );
    }

    /// Regression test: a beacon node HTTP 404 for a given slot must return
    /// `BeaconClientError::SlotNotFound` rather than a generic `Http` error.
    /// This allows the blob provider layer to map it to `BlobProviderError::BlobNotFound`
    /// and the pipeline to issue a reset rather than retrying indefinitely.
    #[tokio::test]
    async fn test_filtered_beacon_blobs_404_returns_slot_not_found() {
        let slot = 13779552u64; // slot from the real-world missed-slot incident
        let test_blob_hash: FixedBytes<32> = FixedBytes::from_hex(TEST_BLOB_A_HASH_HEX).unwrap();
        let requested_blob_hashes: Vec<B256> = vec![test_blob_hash];

        let server = MockServer::start();
        let blobs_mock = server.mock(|when, then| {
            when.method(GET).path(format!("/eth/v1/beacon/blobs/{slot}"));
            then.status(404).body(r#"{"code":404,"message":"Block not found"}"#);
        });

        let client = OnlineBeaconClient::new_http(server.base_url());
        let response = client.filtered_beacon_blobs(slot, &requested_blob_hashes).await;
        blobs_mock.assert();

        assert!(
            matches!(response, Err(BeaconClientError::SlotNotFound(s)) if s == slot),
            "expected SlotNotFound({slot}), got {response:?}"
        );
    }
}
