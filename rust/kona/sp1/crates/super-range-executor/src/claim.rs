//! A span's claim, fetched from a supernode or read from a file an earlier run saved.

use std::{fs, path::Path};

use alloy_primitives::B256;
use anyhow::{Context, Result, anyhow};
use jsonrpsee_core::{client::ClientT, rpc_params};
use jsonrpsee_http_client::HttpClient;
use kona_sp1_client_utils::super_root::TimestampSpan;
use serde::{Deserialize, Serialize};
use serde_json::Value;

use crate::SuperRootAtTimestampResponse;

/// The trusted L1 head and the `superroot_atTimestamp` results for `start_timestamp - 1` through
/// `end_timestamp`: everything a run takes from the supernode.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct Claim {
    /// Trusted L1 head hash.
    pub l1_head: B256,
    /// Number of the trusted L1 head.
    pub l1_head_number: u64,
    /// First timestamp the span proves.
    pub start_timestamp: u64,
    /// Last timestamp the span proves.
    pub end_timestamp: u64,
    /// The results as the supernode returned them, so a saved claim decodes like a live one.
    pub responses: Vec<Value>,
}

impl Claim {
    /// Fetches the span's super roots from `supernode`.
    pub async fn fetch(
        supernode: &HttpClient,
        l1_head: B256,
        l1_head_number: u64,
        span: TimestampSpan,
    ) -> Result<Self> {
        let first = span
            .start
            .checked_sub(1)
            .ok_or_else(|| anyhow!("timestamp 0 has no previous super-root timestamp"))?;
        let mut responses = Vec::new();
        for timestamp in first..=span.end {
            let response: Value = supernode
                .request("superroot_atTimestamp", rpc_params![format!("0x{timestamp:x}")])
                .await
                .with_context(|| format!("superroot_atTimestamp({timestamp}) failed"))?;
            responses.push(response);
        }
        Ok(Self {
            l1_head,
            l1_head_number,
            start_timestamp: span.start,
            end_timestamp: span.end,
            responses,
        })
    }

    /// Reads a claim that [`Self::write`] saved.
    pub fn read(path: &Path) -> Result<Self> {
        let bytes =
            fs::read(path).with_context(|| format!("failed to read claim {}", path.display()))?;
        serde_json::from_slice(&bytes)
            .with_context(|| format!("failed to decode claim {}", path.display()))
    }

    /// Saves the claim as JSON.
    pub fn write(&self, path: &Path) -> Result<()> {
        fs::write(path, serde_json::to_vec_pretty(self)?)
            .with_context(|| format!("failed to write claim {}", path.display()))
    }

    /// The timestamps the claim proves.
    pub fn span(&self) -> Result<TimestampSpan> {
        Ok(TimestampSpan::new(self.start_timestamp, self.end_timestamp)?)
    }

    /// The decoded supernode responses, oldest first.
    pub fn decoded_responses(&self) -> Result<Vec<SuperRootAtTimestampResponse>> {
        let first = self.start_timestamp.saturating_sub(1);
        self.responses
            .iter()
            .zip(first..)
            .map(|(response, timestamp)| {
                serde_json::from_value(response.clone()).with_context(|| {
                    format!("failed to decode the superroot_atTimestamp({timestamp}) response")
                })
            })
            .collect()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::tests::rpc_result;
    use httpmock::{Method::POST, MockServer};
    use serde_json::json;

    fn raw_response(timestamp: u64) -> Value {
        let block = json!({"hash": B256::repeat_byte(5), "number": 8});
        json!({
            "current_l1": block,
            "chain_ids": ["10"],
            "safe_timestamp": timestamp,
            "local_safe_timestamp": timestamp,
            "finalized_timestamp": timestamp,
            "data": {
                "verified_required_l1": block,
                "super": {
                    "timestamp": format!("0x{timestamp:x}"),
                    "chains": [{"ChainID": "10", "Output": B256::repeat_byte(4)}],
                },
                "super_root": B256::repeat_byte(6),
            },
        })
    }

    fn claim() -> Claim {
        Claim {
            l1_head: B256::repeat_byte(0xaa),
            l1_head_number: 8,
            start_timestamp: 101,
            end_timestamp: 102,
            responses: (100..=102).map(raw_response).collect(),
        }
    }

    #[test]
    fn saved_claim_reads_back_unchanged() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("claim.json");

        claim().write(&path).unwrap();

        assert_eq!(Claim::read(&path).unwrap(), claim());
    }

    #[test]
    fn decodes_responses_oldest_first() {
        let decoded = claim().decoded_responses().unwrap();

        let timestamps: Vec<_> = decoded
            .iter()
            .map(|response| response.data.as_ref().unwrap().super_v1.timestamp)
            .collect();
        assert_eq!(timestamps, [100, 101, 102]);
        assert_eq!(claim().span().unwrap(), TimestampSpan::new(101, 102).unwrap());
    }

    #[test]
    fn names_the_timestamp_of_an_undecodable_response() {
        let mut claim = claim();
        claim.responses[2] = json!({"chain_ids": ["10"]});

        let err = claim.decoded_responses().unwrap_err();

        assert_eq!(err.to_string(), "failed to decode the superroot_atTimestamp(102) response");
    }

    #[tokio::test]
    async fn fetches_the_pre_span_timestamp_through_the_span_end() {
        let server = MockServer::start();
        let mocks: Vec<_> = (100..=102)
            .map(|timestamp| {
                server.mock(|when, then| {
                    when.method(POST).json_body_includes(
                        json!({"params": [format!("0x{timestamp:x}")]}).to_string(),
                    );
                    then.respond_with(rpc_result(raw_response(timestamp)));
                })
            })
            .collect();
        let client =
            jsonrpsee_http_client::HttpClientBuilder::default().build(server.base_url()).unwrap();

        let fetched = Claim::fetch(
            &client,
            B256::repeat_byte(0xaa),
            8,
            TimestampSpan::new(101, 102).unwrap(),
        )
        .await
        .unwrap();

        assert_eq!(fetched, claim());
        for mock in mocks {
            mock.assert();
        }
    }
}
