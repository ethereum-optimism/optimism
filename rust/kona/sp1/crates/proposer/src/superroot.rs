//! Supernode RPC access for super-root proposal data.
//!
//! One RPC (`superroot_atTimestamp`) supplies everything the create path
//! needs: the canonical super-root proof preimage at a timestamp, and the
//! cross-chain safe/finalized timestamp views that gate proposal cadence.

use alloy_primitives::B256;
use alloy_transport_http::reqwest::Url;
use anyhow::{Context, Result, anyhow};
use futures::future::join_all;
use jsonrpsee::http_client::{HttpClient, HttpClientBuilder};
use kona_sp1_client_utils::super_root::{encode_super_root_proof, hash_super_root_proof};
pub use kona_sp1_super_range_executor::{
    SuperRootAtTimestampResponse, SuperRootResponseData, fetch_superroot_at_timestamp,
    proof_from_super_v1,
};
use std::time::Duration;

use crate::config::ProposalSafety;

/// Thin supernode client wrapper.
#[derive(Clone, Debug)]
pub struct SuperrootClient {
    clients: Vec<HttpClient>,
}

/// How responses from multiple supernodes are prioritized.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ResponseSelection {
    /// Select the response with the highest L1 view.
    HighestL1,
    /// Prefer responses containing super-root data, then select by highest L1 view.
    PreferData,
}

/// Canonical super-root material for one timestamp.
#[derive(Debug, Clone)]
pub struct SuperRootAt {
    /// Marshaled super-root proof preimage (`version || timestamp || pairs`).
    pub proof_bytes: Vec<u8>,
    /// `keccak256(proof_bytes)` - the game root claim.
    pub super_root: B256,
}

impl SuperrootClient {
    /// Builds a client for the supernode (or single-chain op-node) RPC.
    pub fn new(urls: &[Url]) -> Result<Self> {
        let clients = urls
            .iter()
            .map(|url| {
                HttpClientBuilder::default()
                    .request_timeout(Duration::from_secs(30))
                    .build(url.as_str())
                    .with_context(|| format!("failed to build super-root client for {url}"))
            })
            .collect::<Result<Vec<_>>>()?;
        Ok(Self { clients })
    }

    /// Raw `superroot_atTimestamp` response.
    pub async fn superroot_at_timestamp(
        &self,
        timestamp: u64,
        selection: ResponseSelection,
    ) -> Result<SuperRootAtTimestampResponse> {
        let results = join_all(
            self.clients.iter().map(|client| fetch_superroot_at_timestamp(client, timestamp)),
        )
        .await;

        select_highest_response(results, selection)
    }

    /// The highest timestamp the proposer may propose at under the
    /// configured safety level, read from any recent response.
    pub const fn max_proposable_timestamp(
        response: &SuperRootAtTimestampResponse,
        safety: ProposalSafety,
    ) -> u64 {
        match safety {
            ProposalSafety::Safe => response.current_safe_timestamp,
            ProposalSafety::Finalized => response.current_finalized_timestamp,
        }
    }

    /// Extracts and verifies the canonical super-root material from a
    /// response, or `None` when the timestamp is not yet safe (`data`
    /// absent). Present data must be for the requested timestamp and hash
    /// to the response's reported super root.
    pub fn super_root_at(
        response: &SuperRootAtTimestampResponse,
        expected_timestamp: u64,
    ) -> Result<Option<SuperRootAt>> {
        let Some(data) = &response.data else {
            return Ok(None);
        };
        if data.super_v1.timestamp != expected_timestamp {
            return Err(anyhow!(
                "supernode returned super root for timestamp {} but {expected_timestamp} was requested",
                data.super_v1.timestamp
            ));
        }
        let proof = proof_from_super_v1(&data.super_v1)?;
        let proof_bytes = encode_super_root_proof(&proof)
            .map_err(|err| anyhow!("failed to encode super-root proof: {err:?}"))?;
        let super_root = hash_super_root_proof(&proof)
            .map_err(|err| anyhow!("failed to hash super-root proof: {err:?}"))?;
        if B256::from(*super_root) != *data.super_root {
            return Err(anyhow!(
                "supernode super-root mismatch: computed {super_root}, response reports {}",
                data.super_root
            ));
        }
        Ok(Some(SuperRootAt { proof_bytes, super_root: B256::from(*super_root) }))
    }
}

fn select_highest_response(
    results: Vec<Result<SuperRootAtTimestampResponse>>,
    selection: ResponseSelection,
) -> Result<SuperRootAtTimestampResponse> {
    let mut errors = Vec::new();
    let mut highest_response: Option<SuperRootAtTimestampResponse> = None;
    for (idx, result) in results.into_iter().enumerate() {
        match result {
            Ok(response) => {
                let is_higher = highest_response.as_ref().is_none_or(|highest| match selection {
                    ResponseSelection::HighestL1 => {
                        response.current_l1.number > highest.current_l1.number
                    }
                    ResponseSelection::PreferData => {
                        (response.data.is_some(), response.current_l1.number) >
                            (highest.data.is_some(), highest.current_l1.number)
                    }
                });
                if is_higher {
                    highest_response = Some(response);
                }
            }
            Err(err) => {
                tracing::warn!(
                    idx,
                    error = %err,
                    "Failed to retrieve super-root"
                );
                errors.push(format!("supernode {idx}: {err:#}"));
            }
        }
    }

    highest_response
        .ok_or_else(|| anyhow!("no valid super-root response received: {}", errors.join("; ")))
}

/// Builds ZK dispute game extraData: `parentIndex (4B BE) || superRootProof`.
pub fn zk_extra_data(parent_index: u32, super_root_proof: &[u8]) -> Vec<u8> {
    let mut extra_data = Vec::with_capacity(4 + super_root_proof.len());
    extra_data.extend_from_slice(&parent_index.to_be_bytes());
    extra_data.extend_from_slice(super_root_proof);
    extra_data
}

#[cfg(test)]
mod tests {
    use alloy_primitives::U256;
    use kona_sp1_super_range_executor::{BlockId, ChainId, ChainIdAndOutput, SuperV1};

    use super::*;

    #[test]
    fn extra_data_encoding() {
        // Same vectors as Go's TestZKExtraData (#21995 lineage).
        assert_eq!(zk_extra_data(1, &[0xbe, 0xef]), vec![0x00, 0x00, 0x00, 0x01, 0xbe, 0xef]);
        assert_eq!(zk_extra_data(u32::MAX, &[0x01]), vec![0xff, 0xff, 0xff, 0xff, 0x01]);
        assert_eq!(zk_extra_data(0x01020304, &[]), vec![0x01, 0x02, 0x03, 0x04]);
    }

    /// One-character cross-wiring of the safe/finalized fields is a bug
    /// nothing in CI catches: Finalized is the production default while
    /// devstack runs Safe.
    #[test]
    fn max_proposable_selects_safety_field() {
        let response = SuperRootAtTimestampResponse {
            current_l1: BlockId::default(),
            current_safe_timestamp: 10,
            current_local_safe_timestamp: 11,
            current_finalized_timestamp: 5,
            optimistic_at_timestamp: Default::default(),
            chain_ids: vec![],
            data: None,
        };
        assert_eq!(SuperrootClient::max_proposable_timestamp(&response, ProposalSafety::Safe), 10);
        assert_eq!(
            SuperrootClient::max_proposable_timestamp(&response, ProposalSafety::Finalized),
            5
        );
    }

    /// A self-consistent response: the reported super root is computed from
    /// the proof preimage, so only the timestamp check can fail.
    fn response_at(timestamp: u64) -> SuperRootAtTimestampResponse {
        let super_v1 = SuperV1 {
            timestamp,
            chains: vec![ChainIdAndOutput {
                chain_id: ChainId(U256::from(10)),
                output: B256::repeat_byte(0xab),
            }],
        };
        let proof = proof_from_super_v1(&super_v1).unwrap();
        let super_root = B256::from(*hash_super_root_proof(&proof).unwrap());
        SuperRootAtTimestampResponse {
            current_l1: BlockId::default(),
            current_safe_timestamp: timestamp,
            current_local_safe_timestamp: timestamp,
            current_finalized_timestamp: timestamp,
            optimistic_at_timestamp: Default::default(),
            chain_ids: vec![ChainId(U256::from(10))],
            data: Some(SuperRootResponseData {
                verified_required_l1: BlockId::default(),
                super_v1,
                super_root,
            }),
        }
    }

    #[test]
    fn super_root_at_verifies_requested_timestamp() {
        let response = response_at(101);
        assert!(SuperrootClient::super_root_at(&response, 101).unwrap().is_some());
        let err = SuperrootClient::super_root_at(&response, 102).unwrap_err();
        assert!(err.to_string().contains("102 was requested"));
    }

    #[test]
    fn super_root_at_absent_data_is_none() {
        let mut response = response_at(101);
        response.data = None;
        assert!(SuperrootClient::super_root_at(&response, 101).unwrap().is_none());
    }

    #[test]
    fn response_selection_falls_over_to_a_healthy_source() {
        let expected = response_at(101);
        let response = select_highest_response(
            vec![Err(anyhow!("first source unavailable")), Ok(expected.clone())],
            ResponseSelection::HighestL1,
        )
        .unwrap();

        assert_eq!(response, expected);
    }

    #[test]
    fn response_selection_uses_the_highest_l1_view() {
        let mut lagging = response_at(101);
        lagging.current_l1.number = 10;
        let mut expected = response_at(101);
        expected.current_l1.number = 12;

        let response = select_highest_response(
            vec![Ok(lagging), Ok(expected.clone())],
            ResponseSelection::HighestL1,
        )
        .unwrap();

        assert_eq!(response, expected);
    }

    #[test]
    fn response_selection_prefers_available_data_for_exact_lookup() {
        let mut expected = response_at(101);
        expected.current_l1.number = 10;
        let mut ahead_without_data = response_at(101);
        ahead_without_data.current_l1.number = 12;
        ahead_without_data.data = None;

        let response = select_highest_response(
            vec![Ok(expected.clone()), Ok(ahead_without_data)],
            ResponseSelection::PreferData,
        )
        .unwrap();

        assert_eq!(response, expected);
    }

    #[test]
    fn response_selection_uses_highest_l1_for_sync_status_without_data() {
        let mut behind_with_data = response_at(101);
        behind_with_data.current_l1.number = 10;
        let mut expected = response_at(101);
        expected.current_l1.number = 12;
        expected.data = None;

        let response = select_highest_response(
            vec![Ok(behind_with_data), Ok(expected.clone())],
            ResponseSelection::HighestL1,
        )
        .unwrap();

        assert_eq!(response, expected);
    }

    #[test]
    fn response_selection_reports_all_source_failures() {
        let err = select_highest_response(
            vec![
                Err(anyhow!("first source unavailable")),
                Err(anyhow!("second source unavailable")),
            ],
            ResponseSelection::HighestL1,
        )
        .unwrap_err()
        .to_string();

        assert!(err.contains("supernode 0: first source unavailable"));
        assert!(err.contains("supernode 1: second source unavailable"));
    }
}
