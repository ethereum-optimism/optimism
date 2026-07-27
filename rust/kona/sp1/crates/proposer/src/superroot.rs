//! Supernode RPC access for super-root proposal data.
//!
//! One RPC (`superroot_atTimestamp`) supplies everything the create path
//! needs: the canonical super-root proof preimage at a timestamp, and the
//! cross-chain safe/finalized timestamp views that gate proposal cadence.

use alloy_primitives::B256;
use anyhow::{Context, Result, anyhow};
use jsonrpsee::http_client::{HttpClient, HttpClientBuilder};
use kona_sp1_client_utils::super_root::{encode_super_root_proof, hash_super_root_proof};
pub use kona_sp1_super_range_executor::{
    SuperRootAtTimestampResponse, SuperRootResponseData, fetch_superroot_at_timestamp,
    proof_from_super_v1,
};

use crate::config::ProposalSafety;

/// Thin supernode client wrapper.
#[derive(Clone, Debug)]
pub struct SuperrootClient {
    client: HttpClient,
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
    pub fn new(url: &str) -> Result<Self> {
        let client = HttpClientBuilder::default()
            .build(url)
            .with_context(|| format!("failed to build super-root client for {url}"))?;
        Ok(Self { client })
    }

    /// Raw `superroot_atTimestamp` response.
    pub async fn superroot_at_timestamp(
        &self,
        timestamp: u64,
    ) -> Result<SuperRootAtTimestampResponse> {
        fetch_superroot_at_timestamp(&self.client, timestamp).await
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
    /// absent).
    pub fn super_root_at(response: &SuperRootAtTimestampResponse) -> Result<Option<SuperRootAt>> {
        let Some(data) = &response.data else {
            return Ok(None);
        };
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

/// Builds ZK dispute game extraData: `parentIndex (4B BE) || superRootProof`.
pub fn zk_extra_data(parent_index: u32, super_root_proof: &[u8]) -> Vec<u8> {
    let mut extra_data = Vec::with_capacity(4 + super_root_proof.len());
    extra_data.extend_from_slice(&parent_index.to_be_bytes());
    extra_data.extend_from_slice(super_root_proof);
    extra_data
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn extra_data_encoding() {
        // Same vectors as Go's TestZKExtraData (#21995 lineage).
        assert_eq!(zk_extra_data(1, &[0xbe, 0xef]), vec![0x00, 0x00, 0x00, 0x01, 0xbe, 0xef]);
        assert_eq!(zk_extra_data(u32::MAX, &[0x01]), vec![0xff, 0xff, 0xff, 0xff, 0x01]);
        assert_eq!(zk_extra_data(0x01020304, &[]), vec![0x01, 0x02, 0x03, 0x04]);
    }
}
