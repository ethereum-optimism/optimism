//! Contains an online implementation of the `BeaconClient` trait.

#[cfg(feature = "metrics")]
use crate::Metrics;
use crate::blobs::BoxedBlobWithIndex;
use alloy_eips::eip4844::IndexedBlobHash;
use alloy_rpc_types_beacon::sidecar::{BeaconBlobBundle, GetBlobsResponse};
use async_trait::async_trait;
use reqwest::Client;
use std::{boxed::Box, format, string::String, vec::Vec};

/// The config spec engine api method.
const SPEC_METHOD: &str = "eth/v1/config/spec";

/// The beacon genesis engine api method.
const GENESIS_METHOD: &str = "eth/v1/beacon/genesis";

/// The blob sidecars engine api method prefix.
const SIDECARS_METHOD_PREFIX_DEPRECATED: &str = "eth/v1/beacon/blob_sidecars";

/// THe blobs engine api method prefix.
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
#[derive(Debug, Default, Clone, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub struct ReducedConfigData {
    /// The seconds per slot.
    #[serde(rename = "SECONDS_PER_SLOT")]
    #[serde(with = "alloy_serde::quantity")]
    pub seconds_per_slot: u64,
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
        Self { data: ReducedConfigData { seconds_per_slot } }
    }
}

impl APIGenesisResponse {
    /// Creates a new API genesis response.
    pub const fn new(genesis_time: u64) -> Self {
        Self { data: ReducedGenesisData { genesis_time } }
    }
}

/// The [BeaconClient] is a thin wrapper around the Beacon API.
#[async_trait]
pub trait BeaconClient {
    /// The error type for [BeaconClient] implementations.
    type Error: core::fmt::Display;

    /// Returns the slot interval in seconds.
    async fn slot_interval(&self) -> Result<APIConfigResponse, Self::Error>;

    /// Returns the beacon genesis time.
    async fn genesis_time(&self) -> Result<APIGenesisResponse, Self::Error>;

    /// Fetches blobs that were confirmed in the specified L1 block with the given slot.
    /// Blob data is not checked for validity.
    async fn filtered_beacon_blobs(
        &self,
        slot: u64,
        blob_hashes: &[IndexedBlobHash],
    ) -> Result<Vec<BoxedBlobWithIndex>, Self::Error>;
}

/// An online implementation of the [BeaconClient] trait.
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
    /// Creates a new [OnlineBeaconClient] from the provided base URL string.
    pub fn new_http(mut base: String) -> Self {
        // If base ends with a slash, remove it
        if base.ends_with("/") {
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

    async fn filtered_beacon_blobs(
        &self,
        slot: u64,
        blob_hashes: &[IndexedBlobHash],
    ) -> Result<Vec<BoxedBlobWithIndex>, reqwest::Error> {
        let blob_indexes = blob_hashes.iter().map(|blob| blob.index).collect::<Vec<_>>();

        Ok(
            match self
                .inner
                .get(format!("{}/{}/{}", self.base, BLOBS_METHOD_PREFIX, slot))
                .send()
                .await
            {
                Ok(response) if response.status().is_success() => {
                    let bundle = response.json::<GetBlobsResponse>().await?;

                    bundle
                        .data
                        .into_iter()
                        .enumerate()
                        .filter_map(|(index, blob)| {
                            let index = index as u64;
                            blob_indexes
                                .contains(&index)
                                .then_some(BoxedBlobWithIndex { index, blob: Box::new(blob) })
                        })
                        .collect::<Vec<_>>()
                }
                // If the blobs endpoint fails, try the deprecated sidecars endpoint. CL Clients
                // only support the blobs endpoint from Fusaka (Fulu) onwards.
                _ => self
                    .inner
                    .get(format!("{}/{}/{}", self.base, SIDECARS_METHOD_PREFIX_DEPRECATED, slot))
                    .send()
                    .await?
                    .json::<BeaconBlobBundle>()
                    .await?
                    .into_iter()
                    .filter_map(|blob| {
                        blob_indexes
                            .contains(&blob.index)
                            .then_some(BoxedBlobWithIndex { index: blob.index, blob: blob.blob })
                    })
                    .collect::<Vec<_>>(),
            },
        )
    }
}

#[async_trait]
impl BeaconClient for OnlineBeaconClient {
    type Error = reqwest::Error;

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

        result
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

        result
    }

    async fn filtered_beacon_blobs(
        &self,
        slot: u64,
        blob_hashes: &[IndexedBlobHash],
    ) -> Result<Vec<BoxedBlobWithIndex>, Self::Error> {
        kona_macros::inc!(gauge, Metrics::BEACON_CLIENT_REQUESTS, "method" => "blobs");

        // Try to get the blobs from the blobs endpoint.
        let result = self.filtered_beacon_blobs(slot, blob_hashes).await;

        if result.is_err() {
            kona_macros::inc!(gauge, Metrics::BEACON_CLIENT_ERRORS, "method" => "blobs");
        }

        result
    }
}
