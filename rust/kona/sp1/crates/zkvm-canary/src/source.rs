//! Canonical finalized snapshot selection and post-run revalidation.

use std::collections::BTreeMap;

use alloy_primitives::{B256, U256, keccak256};
use anyhow::{Result, anyhow, ensure};
use jsonrpsee_core::{client::ClientT, rpc_params};
use jsonrpsee_http_client::{HttpClient, HttpClientBuilder};
use kona_sp1_client_utils::super_root::{TimestampSpan, hash_super_root_proof};
use kona_sp1_super_range_executor::{
    BlockId, ChainId, HostInputs, OutputV0, SuperRootAtTimestampResponse, SuperRootResponseData,
    SynthesizedExecution, proof_from_super_v1, synthesize_execution,
};
use serde::{Deserialize, de::IgnoredAny};
use serde_json::{Value, value::RawValue};
use sha2::{Digest, Sha256};

use crate::{artifact::ArtifactIdentity, config::CanaryConfig};

const MAX_REVALIDATION_DETAIL_BYTES: usize = 1024;
const MAX_PARENT_REQUEST_BYTES: u32 = 64 * 1024;
const FINGERPRINT_DOMAIN: &[u8] = b"kona-zkvm-canary-snapshot-v1";

/// A canonical L1 block resolved by number through the configured execution endpoint.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct CanonicalL1Block {
    block: BlockId,
}

impl CanonicalL1Block {
    /// Returns the canonical block identity.
    pub const fn block_id(self) -> BlockId {
        self.block
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
struct PinBounds {
    l1_finalized: BlockId,
    supernode_horizon: u64,
}

/// A snapshot accepted by every parent-side trust and canonicality check.
#[derive(Clone)]
pub struct ValidatedSnapshot {
    discovery_timestamp: u64,
    span: TimestampSpan,
    pinned_l1: CanonicalL1Block,
    l1_finalized: CanonicalL1Block,
    supernode_horizon: u64,
    responses: Vec<SuperRootAtTimestampResponse>,
    chain_ids: Vec<u64>,
    canonical_l1: Vec<CanonicalL1Block>,
    artifact_identity: ArtifactIdentity,
    fingerprint: B256,
    host_inputs: HostInputs,
}

impl std::fmt::Debug for ValidatedSnapshot {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("ValidatedSnapshot")
            .field("discovery_timestamp", &self.discovery_timestamp)
            .field("span", &self.span)
            .field("pinned_l1", &self.pinned_l1)
            .field("l1_finalized", &self.l1_finalized)
            .field("supernode_horizon", &self.supernode_horizon)
            .field("response_count", &self.responses.len())
            .field("chain_ids", &self.chain_ids)
            .field("canonical_l1", &self.canonical_l1)
            .field("artifact_identity", &self.artifact_identity)
            .field("fingerprint", &self.fingerprint)
            .finish()
    }
}

impl ValidatedSnapshot {
    /// Returns the inclusive finalized timestamp span.
    pub const fn span(&self) -> TimestampSpan {
        self.span
    }

    /// Returns the earlier of the supernode horizon and the L1 finalized head.
    pub const fn pinned_l1(&self) -> CanonicalL1Block {
        self.pinned_l1
    }

    /// Returns the exact agreed-through-target responses consumed by synthesis.
    pub fn responses(&self) -> &[SuperRootAtTimestampResponse] {
        &self.responses
    }

    /// Returns the sorted chain universe covered by every response.
    pub fn chain_ids(&self) -> &[u64] {
        &self.chain_ids
    }

    /// Returns all L1 identities whose canonicality was established during selection.
    pub fn canonical_l1(&self) -> &[CanonicalL1Block] {
        &self.canonical_l1
    }

    /// Returns the release identity included in this attempt.
    pub const fn artifact_identity(&self) -> ArtifactIdentity {
        self.artifact_identity
    }

    /// Returns the deterministic identity of the artifact, pin, and response claims.
    pub const fn fingerprint(&self) -> B256 {
        self.fingerprint
    }

    /// Returns the validated deployment inputs needed for witness collection.
    pub const fn host_inputs(&self) -> &HostInputs {
        &self.host_inputs
    }

    /// Reconstructs the executor inputs from the exact validated responses.
    pub fn synthesize_execution(&self) -> Result<SynthesizedExecution> {
        let pin = self.pinned_l1.block_id();
        synthesize_execution(self.span, pin.hash, pin.number, &self.responses)
    }
}

/// Result of comparing a completed attempt with a fresh canonical refetch.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum SnapshotRevalidation {
    /// Every attempt identity is still current and canonical.
    Current,
    /// A commitment or canonical identity changed during execution.
    Stale {
        /// Bounded diagnostic detail for logs.
        reason: String,
    },
    /// The parent could not obtain a trustworthy comparison fixture.
    Unavailable {
        /// Bounded diagnostic detail for logs.
        error: String,
    },
}

/// Bounded parent clients and trust policy for one configured deployment.
pub struct SnapshotSource {
    superroot_client: HttpClient,
    l1_client: HttpClient,
    span_length: u64,
    configured_chain_ids: Vec<u64>,
    max_entries: usize,
    artifact_identity: ArtifactIdentity,
    host_inputs: HostInputs,
}

impl std::fmt::Debug for SnapshotSource {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("SnapshotSource")
            .field("span_length", &self.span_length)
            .field("configured_chain_ids", &self.configured_chain_ids)
            .field("max_entries", &self.max_entries)
            .field("artifact_identity", &self.artifact_identity)
            .finish_non_exhaustive()
    }
}

impl SnapshotSource {
    /// Builds bounded clients from a configuration already validated before RPC work.
    pub fn new(config: &CanaryConfig) -> Result<Self> {
        let max_response_size = config.max_parent_response_bytes.get();
        let client = |endpoint: &str, label: &str| {
            HttpClientBuilder::default()
                .max_request_size(MAX_PARENT_REQUEST_BYTES)
                .max_response_size(max_response_size)
                .request_timeout(config.rpc_request_timeout)
                .build(endpoint)
                .map_err(|_| anyhow!("failed to build {label} client"))
        };
        Ok(Self {
            superroot_client: client(config.superroot_rpc.as_str(), "super-root")?,
            l1_client: client(config.l1_rpc.as_str(), "L1")?,
            span_length: u64::from(config.span_length.get()),
            configured_chain_ids: config.chain_ids().collect(),
            max_entries: config.max_parent_response_entries.get(),
            artifact_identity: config.artifact.identity,
            host_inputs: config.host_inputs(),
        })
    }

    /// Selects the newest complete finalized span visible at `now`.
    pub async fn select_finalized(&self, now: u64) -> Result<ValidatedSnapshot> {
        let discovery = self.fetch_superroot(now).await.map_err(FetchFailure::into_anyhow)?;
        validate_safety_horizons(now, "discovery", &discovery)
            .map_err(BuildFailure::into_anyhow)?;

        let target = discovery.current_finalized_timestamp;
        let start = target
            .checked_sub(self.span_length - 1)
            .ok_or_else(|| anyhow!("finalized target {target} is too early for configured span"))?;
        let agreed = start
            .checked_sub(1)
            .ok_or_else(|| anyhow!("finalized span starting at 0 has no agreed timestamp"))?;
        let span = TimestampSpan::new(start, target).map_err(anyhow::Error::from)?;

        let (pin, pin_bounds) =
            self.resolve_pin(discovery.current_l1).await.map_err(BuildFailure::into_anyhow)?;
        let responses =
            self.fetch_responses(agreed, target).await.map_err(FetchFailure::into_anyhow)?;
        self.validate_snapshot(now, span, pin, responses, pin_bounds, Some(discovery.current_l1))
            .await
            .map_err(BuildFailure::into_anyhow)
    }

    /// Re-fetches every claim and canonical L1 identity after execution.
    pub async fn revalidate(&self, snapshot: &ValidatedSnapshot) -> SnapshotRevalidation {
        for canonical in &snapshot.canonical_l1 {
            let expected = canonical.block_id();
            match self.fetch_l1_block(expected.number).await {
                Ok(Some(actual)) if actual.hash == expected.hash => {}
                Ok(Some(actual)) => {
                    return stale(format!(
                        "canonical L1 block {} changed from {} to {}",
                        expected.number, expected.hash, actual.hash,
                    ));
                }
                Ok(None) => {
                    return stale(format!(
                        "canonical L1 block {} is no longer available",
                        expected.number
                    ));
                }
                Err(error) => return unavailable(error.into_anyhow()),
            }
        }

        let discovery = match self.fetch_superroot(snapshot.discovery_timestamp).await {
            Ok(discovery) => discovery,
            Err(error) => return unavailable(error.into_anyhow()),
        };
        if let Err(BuildFailure::Invalid(error)) =
            validate_safety_horizons(snapshot.discovery_timestamp, "discovery", &discovery)
        {
            return stale(error);
        }
        let (refreshed_pin, refreshed_pin_bounds) =
            match self.resolve_pin(discovery.current_l1).await {
                Ok(resolved) => resolved,
                Err(BuildFailure::Invalid(error)) => return stale(error),
                Err(BuildFailure::Unavailable(error)) => return unavailable(error.into_anyhow()),
            };
        let selected_pin = snapshot.pinned_l1.block_id();
        if refreshed_pin.number < selected_pin.number {
            return stale(format!(
                "canonical pin regressed below selected L1 {} to {} (L1 finalized head {}, supernode horizon {})",
                selected_pin.number,
                refreshed_pin.number,
                refreshed_pin_bounds.l1_finalized.number,
                refreshed_pin_bounds.supernode_horizon,
            ));
        }
        if refreshed_pin.number == selected_pin.number && refreshed_pin.hash != selected_pin.hash {
            return stale(format!(
                "canonical pin {} changed from {} to {}",
                selected_pin.number, selected_pin.hash, refreshed_pin.hash,
            ));
        }

        let agreed = snapshot.span.start - 1;
        let responses = match self.fetch_responses(agreed, snapshot.span.end).await {
            Ok(responses) => responses,
            Err(error) => return unavailable(error.into_anyhow()),
        };
        let original_claims = response_claims(agreed, &snapshot.responses);
        let refreshed_claims = response_claims(agreed, &responses);
        if original_claims != refreshed_claims {
            return stale("super-root response claims changed during execution");
        }

        let refreshed = match self
            .validate_snapshot(
                snapshot.discovery_timestamp,
                snapshot.span,
                selected_pin,
                responses,
                refreshed_pin_bounds,
                None,
            )
            .await
        {
            Ok(refreshed) => refreshed,
            Err(BuildFailure::Invalid(error)) => return stale(error),
            Err(BuildFailure::Unavailable(error)) => return unavailable(error.into_anyhow()),
        };
        if refreshed.pinned_l1 != snapshot.pinned_l1 {
            return stale("pinned L1 identity changed during execution");
        }
        if refreshed.fingerprint != snapshot.fingerprint {
            return stale("snapshot fingerprint changed during execution");
        }
        SnapshotRevalidation::Current
    }

    async fn fetch_responses(
        &self,
        first: u64,
        last: u64,
    ) -> std::result::Result<Vec<SuperRootAtTimestampResponse>, FetchFailure> {
        let count = last
            .checked_sub(first)
            .and_then(|distance| distance.checked_add(1))
            .ok_or_else(|| FetchFailure(anyhow!("invalid response range {first}..={last}")))?;
        if count > 17 {
            return Err(FetchFailure(anyhow!(
                "agreed-through-target response count {count} exceeds the protocol maximum"
            )));
        }
        let mut responses = Vec::with_capacity(count as usize);
        for timestamp in first..=last {
            responses.push(self.fetch_superroot(timestamp).await?);
        }
        Ok(responses)
    }

    async fn fetch_superroot(
        &self,
        timestamp: u64,
    ) -> std::result::Result<SuperRootAtTimestampResponse, FetchFailure> {
        let raw: Box<RawValue> = self
            .superroot_client
            .request("superroot_atTimestamp", rpc_params![format!("0x{timestamp:x}")])
            .await
            .map_err(|_| {
                FetchFailure(anyhow!("superroot_atTimestamp({timestamp}) request failed"))
            })?;
        preflight_superroot_json(raw.get(), self.max_entries).map_err(FetchFailure)?;
        let value: Value = serde_json::from_str(raw.get()).map_err(|error| {
            FetchFailure(anyhow!(
                "superroot_atTimestamp({timestamp}) returned invalid JSON: {}",
                bounded_detail(&error.to_string()),
            ))
        })?;
        enforce_response_entry_limits(&value, self.max_entries).map_err(FetchFailure)?;
        serde_json::from_value(value).map_err(|error| {
            FetchFailure(anyhow!(
                "superroot_atTimestamp({timestamp}) returned invalid JSON: {}",
                bounded_detail(&error.to_string()),
            ))
        })
    }

    async fn fetch_l1_block(
        &self,
        number: u64,
    ) -> std::result::Result<Option<BlockId>, FetchFailure> {
        let value: Value = self
            .l1_client
            .request("eth_getBlockByNumber", rpc_params![format!("0x{number:x}"), false])
            .await
            .map_err(|_| FetchFailure(anyhow!("eth_getBlockByNumber({number}) request failed")))?;
        let block: Option<RpcBlock> = serde_json::from_value(value).map_err(|error| {
            FetchFailure(anyhow!(
                "eth_getBlockByNumber({number}) returned invalid JSON: {}",
                bounded_detail(&error.to_string()),
            ))
        })?;
        let Some(block) = block else { return Ok(None) };
        if block.number != number {
            return Err(FetchFailure(anyhow!(
                "eth_getBlockByNumber({number}) returned block {}",
                block.number,
            )));
        }
        Ok(Some(BlockId { number: block.number, hash: block.hash }))
    }

    async fn fetch_l1_finalized(&self) -> std::result::Result<Option<BlockId>, FetchFailure> {
        let value: Value = self
            .l1_client
            .request("eth_getBlockByNumber", rpc_params!["finalized", false])
            .await
            .map_err(|_| FetchFailure(anyhow!("eth_getBlockByNumber(finalized) request failed")))?;
        let block: Option<RpcBlock> = serde_json::from_value(value).map_err(|error| {
            FetchFailure(anyhow!(
                "eth_getBlockByNumber(finalized) returned invalid JSON: {}",
                bounded_detail(&error.to_string()),
            ))
        })?;
        Ok(block.map(|block| BlockId { number: block.number, hash: block.hash }))
    }

    async fn resolve_pin(
        &self,
        current_l1: BlockId,
    ) -> std::result::Result<(BlockId, PinBounds), BuildFailure> {
        if current_l1.number == 0 {
            return Err(BuildFailure::Invalid(anyhow!(
                "CurrentL1 0 has no fully processed predecessor"
            )));
        }
        self.require_canonical(current_l1, "discovery CurrentL1").await?;

        let supernode_horizon = current_l1.number - 1;
        let l1_finalized = self
            .fetch_l1_finalized()
            .await
            .map_err(BuildFailure::Unavailable)?
            .ok_or_else(|| BuildFailure::Invalid(anyhow!("L1 finalized head not found")))?;
        let pin_bounds = PinBounds { l1_finalized, supernode_horizon };
        let pin_number = supernode_horizon.min(l1_finalized.number);
        let pin =
            self.fetch_l1_block(pin_number).await.map_err(BuildFailure::Unavailable)?.ok_or_else(
                || BuildFailure::Invalid(anyhow!("chosen L1 pin {pin_number} not found")),
            )?;
        if pin_number == l1_finalized.number && pin.hash != l1_finalized.hash {
            return Err(BuildFailure::Invalid(anyhow!(
                "L1 finalized head {} has hash {}, canonical block has {}",
                l1_finalized.number,
                l1_finalized.hash,
                pin.hash,
            )));
        }
        Ok((pin, pin_bounds))
    }

    async fn validate_snapshot(
        &self,
        discovery_timestamp: u64,
        span: TimestampSpan,
        pin: BlockId,
        responses: Vec<SuperRootAtTimestampResponse>,
        pin_bounds: PinBounds,
        discovery_current_l1: Option<BlockId>,
    ) -> std::result::Result<ValidatedSnapshot, BuildFailure> {
        let agreed = span.start - 1;
        let expected_count = usize::try_from(span.end - agreed + 1)
            .map_err(|error| BuildFailure::Invalid(anyhow!(error)))?;
        if responses.len() != expected_count {
            return Err(BuildFailure::Invalid(anyhow!(
                "span {}..={} requires {expected_count} responses, got {}",
                span.start,
                span.end,
                responses.len(),
            )));
        }

        let mut references = BTreeMap::new();
        insert_reference(&mut references, pin, "pinned L1")?;
        insert_reference(&mut references, pin_bounds.l1_finalized, "L1 finalized head")?;
        if let Some(current_l1) = discovery_current_l1 {
            insert_reference(&mut references, current_l1, "discovery CurrentL1")?;
        }
        for (offset, response) in responses.iter().enumerate() {
            let timestamp = agreed + offset as u64;
            self.validate_response(
                timestamp,
                span.end,
                response,
                pin.number,
                pin_bounds,
                &mut references,
            )?;
        }

        synthesize_execution(span, pin.hash, pin.number, &responses)
            .map_err(|error| BuildFailure::Invalid(error.context("snapshot synthesis failed")))?;

        let mut canonical_l1 = Vec::with_capacity(references.len());
        for (number, expected) in references {
            let actual =
                self.fetch_l1_block(number).await.map_err(BuildFailure::Unavailable)?.ok_or_else(
                    || BuildFailure::Invalid(anyhow!("referenced L1 block {number} not found")),
                )?;
            if actual.hash != expected.hash {
                return Err(BuildFailure::Invalid(anyhow!(
                    "referenced L1 block {number} has hash {}, expected {}",
                    actual.hash,
                    expected.hash,
                )));
            }
            canonical_l1.push(CanonicalL1Block { block: actual });
        }

        let chain_ids = self.configured_chain_ids.clone();
        let fingerprint = snapshot_fingerprint(self.artifact_identity, span, pin, &responses)
            .map_err(BuildFailure::Invalid)?;
        Ok(ValidatedSnapshot {
            discovery_timestamp,
            span,
            pinned_l1: CanonicalL1Block { block: pin },
            l1_finalized: CanonicalL1Block { block: pin_bounds.l1_finalized },
            supernode_horizon: pin_bounds.supernode_horizon,
            responses,
            chain_ids,
            canonical_l1,
            artifact_identity: self.artifact_identity,
            fingerprint,
            host_inputs: self.host_inputs.clone(),
        })
    }

    fn validate_response(
        &self,
        timestamp: u64,
        selected_target: u64,
        response: &SuperRootAtTimestampResponse,
        pin_number: u64,
        pin_bounds: PinBounds,
        references: &mut BTreeMap<u64, BlockId>,
    ) -> std::result::Result<(), BuildFailure> {
        validate_safety_horizons(timestamp, "response", response)?;
        if response.current_finalized_timestamp < selected_target {
            return Err(BuildFailure::Invalid(anyhow!(
                "timestamp {timestamp} finalized horizon {} precedes selected target {selected_target}",
                response.current_finalized_timestamp,
            )));
        }
        let chain_ids = response_chain_ids(response).map_err(BuildFailure::Invalid)?;
        if chain_ids != self.configured_chain_ids {
            return Err(BuildFailure::Invalid(anyhow!(
                "timestamp {timestamp} chain IDs {chain_ids:?} do not match configured chain IDs {:?}",
                self.configured_chain_ids,
            )));
        }
        let data = response.data.as_ref().ok_or_else(|| {
            BuildFailure::Invalid(anyhow!("timestamp {timestamp} response has no data"))
        })?;
        if data.super_v1.timestamp != timestamp {
            return Err(BuildFailure::Invalid(anyhow!(
                "timestamp {timestamp} response commits timestamp {}",
                data.super_v1.timestamp,
            )));
        }
        let committed_chain_ids = data
            .super_v1
            .chains
            .iter()
            .map(|chain| chain_id_u64(chain.chain_id))
            .collect::<Result<Vec<_>>>()
            .map_err(BuildFailure::Invalid)?;
        if committed_chain_ids != chain_ids {
            return Err(BuildFailure::Invalid(anyhow!(
                "timestamp {timestamp} committed chain IDs {committed_chain_ids:?} do not match response chain IDs {chain_ids:?}",
            )));
        }
        let proof = proof_from_super_v1(&data.super_v1).map_err(BuildFailure::Invalid)?;
        let computed_super_root =
            hash_super_root_proof(&proof).map_err(|error| BuildFailure::Invalid(error.into()))?;
        if computed_super_root != data.super_root {
            return Err(BuildFailure::Invalid(anyhow!(
                "timestamp {timestamp} super-root proof hashes to {computed_super_root}, expected {}",
                data.super_root,
            )));
        }
        if response.current_l1.number <= data.verified_required_l1.number {
            return Err(BuildFailure::Invalid(anyhow!(
                "timestamp {timestamp} CurrentL1 {} has not advanced beyond verified required L1 {}",
                response.current_l1.number,
                data.verified_required_l1.number,
            )));
        }
        if let Err(error) =
            ensure_required_within_pin(timestamp, "verified", data.verified_required_l1, pin_number)
        {
            log_required_l1_above_pin(
                timestamp,
                "verified",
                data.verified_required_l1,
                pin_number,
                pin_bounds,
            );
            return Err(error);
        }
        insert_reference(references, response.current_l1, "CurrentL1")?;
        insert_reference(references, data.verified_required_l1, "verified required L1")?;

        let optimistic_chain_ids = response
            .optimistic_at_timestamp
            .keys()
            .copied()
            .map(chain_id_u64)
            .collect::<Result<Vec<_>>>()
            .map_err(BuildFailure::Invalid)?;
        if optimistic_chain_ids != chain_ids {
            return Err(BuildFailure::Invalid(anyhow!(
                "timestamp {timestamp} optimistic chain IDs {optimistic_chain_ids:?} do not match response chain IDs {chain_ids:?}",
            )));
        }
        for chain_id in chain_ids {
            let output = response
                .optimistic_at_timestamp
                .get(&ChainId(U256::from(chain_id)))
                .expect("exact optimistic coverage established");
            let output_v0 = output.output.as_ref().ok_or_else(|| {
                BuildFailure::Invalid(anyhow!(
                    "timestamp {timestamp} chain {chain_id} has no optimistic output"
                ))
            })?;
            let computed_output_root = output_v0_root(output_v0);
            if computed_output_root != output.output_root {
                return Err(BuildFailure::Invalid(anyhow!(
                    "timestamp {timestamp} chain {chain_id} optimistic output hashes to {computed_output_root}, expected {}",
                    output.output_root,
                )));
            }
            let entry = format!("chain {chain_id} optimistic");
            if let Err(error) =
                ensure_required_within_pin(timestamp, &entry, output.required_l1, pin_number)
            {
                log_required_l1_above_pin(
                    timestamp,
                    &entry,
                    output.required_l1,
                    pin_number,
                    pin_bounds,
                );
                return Err(error);
            }
            insert_reference(references, output.required_l1, "optimistic required L1")?;
        }
        Ok(())
    }

    async fn require_canonical(
        &self,
        expected: BlockId,
        label: &str,
    ) -> std::result::Result<CanonicalL1Block, BuildFailure> {
        let actual = self
            .fetch_l1_block(expected.number)
            .await
            .map_err(BuildFailure::Unavailable)?
            .ok_or_else(|| {
                BuildFailure::Invalid(anyhow!("{label} {} not found", expected.number))
            })?;
        if actual.hash != expected.hash {
            return Err(BuildFailure::Invalid(anyhow!(
                "{label} {} has hash {}, expected {}",
                expected.number,
                actual.hash,
                expected.hash,
            )));
        }
        Ok(CanonicalL1Block { block: actual })
    }
}

#[derive(Debug)]
struct FetchFailure(anyhow::Error);

impl FetchFailure {
    fn into_anyhow(self) -> anyhow::Error {
        self.0
    }
}

#[derive(Debug)]
enum BuildFailure {
    Invalid(anyhow::Error),
    Unavailable(FetchFailure),
}

impl BuildFailure {
    fn into_anyhow(self) -> anyhow::Error {
        match self {
            Self::Invalid(error) => error,
            Self::Unavailable(error) => error.into_anyhow(),
        }
    }
}

impl From<anyhow::Error> for BuildFailure {
    fn from(error: anyhow::Error) -> Self {
        Self::Invalid(error)
    }
}

#[derive(Debug, Deserialize)]
struct RpcBlock {
    hash: B256,
    #[serde(deserialize_with = "deserialize_u64_or_hex")]
    number: u64,
}

#[derive(Clone, Debug, PartialEq, Eq)]
struct ResponseClaims {
    requested_timestamp: u64,
    chain_ids: Vec<ChainId>,
    optimistic_at_timestamp: BTreeMap<ChainId, kona_sp1_super_range_executor::OutputWithRequiredL1>,
    data: Option<SuperRootResponseData>,
}

#[derive(Deserialize)]
struct SuperRootPreflight {
    #[serde(default)]
    optimistic_at_timestamp: UniqueOptimisticKeys,
}

#[derive(Default)]
struct UniqueOptimisticKeys {
    count: usize,
}

impl<'de> Deserialize<'de> for UniqueOptimisticKeys {
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct UniqueOptimisticKeysVisitor;

        impl<'de> serde::de::Visitor<'de> for UniqueOptimisticKeysVisitor {
            type Value = UniqueOptimisticKeys;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("an optimistic output map with unique canonical chain IDs")
            }

            fn visit_map<A>(self, mut map: A) -> std::result::Result<Self::Value, A::Error>
            where
                A: serde::de::MapAccess<'de>,
            {
                let mut keys = BTreeMap::<ChainId, String>::new();
                let mut count = 0usize;
                while let Some(key) = map.next_key::<String>()? {
                    let chain_id = serde_json::from_value::<ChainId>(Value::String(key.clone()))
                        .map_err(<A::Error as serde::de::Error>::custom)?;
                    if let Some(previous) = keys.insert(chain_id, key.clone()) {
                        return Err(<A::Error as serde::de::Error>::custom(format!(
                            "duplicate canonical chain ID {} in optimistic_at_timestamp keys {previous:?} and {key:?}",
                            chain_id.0,
                        )));
                    }
                    count = count.saturating_add(1);
                    map.next_value::<IgnoredAny>()?;
                }
                Ok(UniqueOptimisticKeys { count })
            }
        }

        deserializer.deserialize_map(UniqueOptimisticKeysVisitor)
    }
}

fn response_claims(
    first_timestamp: u64,
    responses: &[SuperRootAtTimestampResponse],
) -> Vec<ResponseClaims> {
    responses
        .iter()
        .enumerate()
        .map(|(offset, response)| ResponseClaims {
            requested_timestamp: first_timestamp + offset as u64,
            chain_ids: response.chain_ids.clone(),
            optimistic_at_timestamp: response.optimistic_at_timestamp.clone(),
            data: response.data.clone(),
        })
        .collect()
}

fn preflight_superroot_json(raw: &str, max_entries: usize) -> Result<()> {
    let preflight: SuperRootPreflight = serde_json::from_str(raw).map_err(|error| {
        anyhow!(
            "super-root response failed duplicate-key preflight: {}",
            bounded_detail(&error.to_string()),
        )
    })?;
    ensure!(
        preflight.optimistic_at_timestamp.count <= max_entries,
        "super-root optimistic_at_timestamp has {} entries; limit is {max_entries}",
        preflight.optimistic_at_timestamp.count,
    );
    Ok(())
}

fn validate_safety_horizons(
    timestamp: u64,
    label: &str,
    response: &SuperRootAtTimestampResponse,
) -> std::result::Result<(), BuildFailure> {
    if response.current_finalized_timestamp > response.current_safe_timestamp {
        return Err(BuildFailure::Invalid(anyhow!(
            "timestamp {timestamp} {label} safe horizon {} precedes finalized horizon {}",
            response.current_safe_timestamp,
            response.current_finalized_timestamp,
        )));
    }
    if response.current_safe_timestamp > response.current_local_safe_timestamp {
        return Err(BuildFailure::Invalid(anyhow!(
            "timestamp {timestamp} {label} local-safe horizon {} precedes safe horizon {}",
            response.current_local_safe_timestamp,
            response.current_safe_timestamp,
        )));
    }
    Ok(())
}

fn enforce_response_entry_limits(value: &Value, max_entries: usize) -> Result<()> {
    let Some(response) = value.as_object() else { return Ok(()) };
    for (label, count) in [
        ("chain_ids", response.get("chain_ids").and_then(Value::as_array).map(Vec::len)),
        (
            "optimistic_at_timestamp",
            response.get("optimistic_at_timestamp").and_then(Value::as_object).map(|map| map.len()),
        ),
        (
            "data.super.chains",
            response
                .get("data")
                .and_then(|data| data.get("super"))
                .and_then(|super_v1| super_v1.get("chains"))
                .and_then(Value::as_array)
                .map(Vec::len),
        ),
    ] {
        if let Some(count) = count {
            ensure!(
                count <= max_entries,
                "super-root {label} has {count} entries; limit is {max_entries}"
            );
        }
    }
    Ok(())
}

fn response_chain_ids(response: &SuperRootAtTimestampResponse) -> Result<Vec<u64>> {
    ensure!(!response.chain_ids.is_empty(), "super-root response has no chain IDs");
    let chain_ids =
        response.chain_ids.iter().copied().map(chain_id_u64).collect::<Result<Vec<_>>>()?;
    for pair in chain_ids.windows(2) {
        ensure!(pair[0] < pair[1], "super-root chain IDs must be strictly increasing");
    }
    Ok(chain_ids)
}

fn chain_id_u64(chain_id: ChainId) -> Result<u64> {
    ensure!(chain_id.0 <= U256::from(u64::MAX), "chain ID {} does not fit in u64", chain_id.0);
    Ok(chain_id.0.saturating_to::<u64>())
}

fn ensure_required_within_pin(
    timestamp: u64,
    label: &str,
    required: BlockId,
    pin_number: u64,
) -> std::result::Result<(), BuildFailure> {
    if required.number > pin_number {
        return Err(BuildFailure::Invalid(anyhow!(
            "timestamp {timestamp} {label} required L1 {} exceeds pinned L1 {pin_number}",
            required.number,
        )));
    }
    Ok(())
}

fn log_required_l1_above_pin(
    timestamp: u64,
    entry: &str,
    required: BlockId,
    pin_number: u64,
    pin_bounds: PinBounds,
) {
    tracing::warn!(
        timestamp,
        entry,
        offending_required_l1_number = required.number,
        offending_required_l1_hash = %required.hash,
        chosen_pin_number = pin_number,
        l1_finalized_number = pin_bounds.l1_finalized.number,
        l1_finalized_hash = %pin_bounds.l1_finalized.hash,
        supernode_horizon = pin_bounds.supernode_horizon,
        "required L1 is above the finalized canary pin",
    );
}

fn insert_reference(
    references: &mut BTreeMap<u64, BlockId>,
    block: BlockId,
    label: &str,
) -> std::result::Result<(), BuildFailure> {
    if let Some(existing) = references.insert(block.number, block) &&
        existing.hash != block.hash
    {
        return Err(BuildFailure::Invalid(anyhow!(
            "{label} conflicts at L1 block {}: {} != {}",
            block.number,
            existing.hash,
            block.hash,
        )));
    }
    Ok(())
}

fn output_v0_root(output: &OutputV0) -> B256 {
    let mut preimage = [0u8; 128];
    preimage[32..64].copy_from_slice(output.state_root.as_slice());
    preimage[64..96].copy_from_slice(output.message_passer_storage_root.as_slice());
    preimage[96..128].copy_from_slice(output.block_hash.as_slice());
    keccak256(preimage)
}

pub(crate) fn snapshot_fingerprint(
    artifact: ArtifactIdentity,
    span: TimestampSpan,
    pin: BlockId,
    responses: &[SuperRootAtTimestampResponse],
) -> Result<B256> {
    span.validate().map_err(anyhow::Error::from)?;
    let agreed = span
        .start
        .checked_sub(1)
        .ok_or_else(|| anyhow!("span starting at 0 has no agreed timestamp"))?;
    let expected_count = usize::try_from(span.end - span.start + 2)?;
    ensure!(
        responses.len() == expected_count,
        "span {}..={} fingerprint requires {expected_count} responses, got {}",
        span.start,
        span.end,
        responses.len(),
    );
    let mut digest = Sha256::new();
    digest.update(FINGERPRINT_DOMAIN);
    digest.update(artifact.prestate.as_slice());
    digest.update(artifact.range_vkey.as_slice());
    digest.update(artifact.elf_sha256.as_slice());
    update_u64(&mut digest, span.start);
    update_u64(&mut digest, span.end);
    update_block(&mut digest, pin);
    for (offset, response) in responses.iter().enumerate() {
        update_u64(&mut digest, agreed + offset as u64);
        update_u64(&mut digest, response.chain_ids.len() as u64);
        for chain_id in &response.chain_ids {
            digest.update(chain_id.0.to_be_bytes::<32>());
        }
        let data = response.data.as_ref().ok_or_else(|| {
            anyhow!("fingerprint response for timestamp {} has no data", agreed + offset as u64)
        })?;
        update_block(&mut digest, data.verified_required_l1);
        digest.update(data.super_root.as_slice());
        update_u64(&mut digest, data.super_v1.timestamp);
        for chain in &data.super_v1.chains {
            digest.update(chain.chain_id.0.to_be_bytes::<32>());
            digest.update(chain.output.as_slice());
        }
        for (chain_id, output) in &response.optimistic_at_timestamp {
            digest.update(chain_id.0.to_be_bytes::<32>());
            digest.update(output.output_root.as_slice());
            update_block(&mut digest, output.required_l1);
            let output = output.output.as_ref().ok_or_else(|| {
                anyhow!(
                    "fingerprint response for timestamp {} chain {} has no optimistic output",
                    agreed + offset as u64,
                    chain_id.0,
                )
            })?;
            digest.update(output.state_root.as_slice());
            digest.update(output.message_passer_storage_root.as_slice());
            digest.update(output.block_hash.as_slice());
        }
    }
    Ok(B256::from(<[u8; 32]>::from(digest.finalize())))
}

fn update_u64(digest: &mut Sha256, value: u64) {
    digest.update(value.to_be_bytes());
}

fn update_block(digest: &mut Sha256, block: BlockId) {
    update_u64(digest, block.number);
    digest.update(block.hash.as_slice());
}

fn deserialize_u64_or_hex<'de, D>(deserializer: D) -> std::result::Result<u64, D::Error>
where
    D: serde::Deserializer<'de>,
{
    struct Visitor;

    impl serde::de::Visitor<'_> for Visitor {
        type Value = u64;

        fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
            formatter.write_str("a u64 or 0x-prefixed quantity")
        }

        fn visit_u64<E>(self, value: u64) -> std::result::Result<u64, E> {
            Ok(value)
        }

        fn visit_str<E>(self, value: &str) -> std::result::Result<u64, E>
        where
            E: serde::de::Error,
        {
            value
                .strip_prefix("0x")
                .ok_or_else(|| E::custom("quantity string must be 0x-prefixed"))
                .and_then(|hex| u64::from_str_radix(hex, 16).map_err(E::custom))
        }
    }

    deserializer.deserialize_any(Visitor)
}

fn stale(reason: impl std::fmt::Display) -> SnapshotRevalidation {
    SnapshotRevalidation::Stale { reason: bounded_detail(&reason.to_string()) }
}

fn unavailable(error: anyhow::Error) -> SnapshotRevalidation {
    SnapshotRevalidation::Unavailable { error: bounded_detail(&format!("{error:#}")) }
}

fn bounded_detail(detail: &str) -> String {
    if detail.len() <= MAX_REVALIDATION_DETAIL_BYTES {
        return detail.to_string();
    }
    let mut boundary = MAX_REVALIDATION_DETAIL_BYTES;
    while !detail.is_char_boundary(boundary) {
        boundary -= 1;
    }
    detail[..boundary].to_string()
}

#[cfg(test)]
mod tests {
    use std::{
        collections::BTreeMap,
        num::{NonZeroU8, NonZeroU32, NonZeroU64, NonZeroUsize},
        time::Duration,
    };

    use httpmock::{Mock, MockServer, prelude::POST};
    use kona_sp1_host_utils::metrics::MetricsListen;
    use kona_sp1_super_range_executor::{ChainIdAndOutput, OutputWithRequiredL1, SuperV1};
    use serde_json::json;
    use url::Url;

    use super::*;
    use crate::{
        artifact::ArtifactConfig,
        config::{CanaryConfig, L2Rpc},
    };

    const NOW: u64 = 1_000;
    const AGREED: u64 = 10;
    const TARGET: u64 = 12;
    const CHAIN_ID: u64 = 10;
    const MOCK_REQUEST_IDS: std::ops::Range<u64> = 0..32;

    fn block(number: u64) -> BlockId {
        BlockId { number, hash: B256::repeat_byte(number as u8) }
    }

    fn output(seed: u8, required_l1: BlockId) -> OutputWithRequiredL1 {
        let output = OutputV0 {
            state_root: B256::repeat_byte(seed),
            message_passer_storage_root: B256::repeat_byte(seed.wrapping_add(1)),
            block_hash: B256::repeat_byte(seed.wrapping_add(2)),
        };
        OutputWithRequiredL1 {
            output_root: output_v0_root(&output),
            output: Some(output),
            required_l1,
        }
    }

    fn response(timestamp: u64, chain_id: u64) -> SuperRootAtTimestampResponse {
        let optimistic_output = output(timestamp as u8, block(100));
        let super_v1 = SuperV1 {
            timestamp,
            chains: vec![ChainIdAndOutput {
                chain_id: ChainId(U256::from(chain_id)),
                output: optimistic_output.output_root,
            }],
        };
        let super_root = hash_super_root_proof(&proof_from_super_v1(&super_v1).unwrap()).unwrap();
        SuperRootAtTimestampResponse {
            current_l1: block(101),
            current_safe_timestamp: TARGET,
            current_local_safe_timestamp: TARGET,
            current_finalized_timestamp: TARGET,
            optimistic_at_timestamp: BTreeMap::from([(
                ChainId(U256::from(chain_id)),
                optimistic_output,
            )]),
            chain_ids: vec![ChainId(U256::from(chain_id))],
            data: Some(SuperRootResponseData {
                verified_required_l1: block(99),
                super_v1,
                super_root,
            }),
        }
    }

    fn discovery() -> SuperRootAtTimestampResponse {
        response(NOW, CHAIN_ID)
    }

    fn base_responses() -> Vec<SuperRootAtTimestampResponse> {
        (AGREED..=TARGET).map(|timestamp| response(timestamp, CHAIN_ID)).collect()
    }

    fn base_blocks() -> BTreeMap<u64, Value> {
        (99..=101).map(|number| (number, block_json(block(number)))).collect()
    }

    fn block_json(block: BlockId) -> Value {
        json!({"hash": block.hash, "number": block.number})
    }

    fn response_json(response: &SuperRootAtTimestampResponse) -> Value {
        let optimistic = response
            .optimistic_at_timestamp
            .iter()
            .map(|(chain_id, output)| {
                let output_v0 = output.output.as_ref().map(|value| {
                    json!({
                        "stateRoot": value.state_root,
                        "messagePasserStorageRoot": value.message_passer_storage_root,
                        "blockHash": value.block_hash,
                    })
                });
                (
                    chain_id.0.to_string(),
                    json!({
                        "output": output_v0,
                        "output_root": output.output_root,
                        "required_l1": block_json(output.required_l1),
                    }),
                )
            })
            .collect::<serde_json::Map<_, _>>();
        let data = response.data.as_ref().map(|data| {
            json!({
                "verified_required_l1": block_json(data.verified_required_l1),
                "super": {
                    "timestamp": data.super_v1.timestamp,
                    "chains": data.super_v1.chains.iter().map(|chain| json!({
                        "ChainID": chain.chain_id.0.to_string(),
                        "Output": chain.output,
                    })).collect::<Vec<_>>(),
                },
                "super_root": data.super_root,
            })
        });
        json!({
            "current_l1": block_json(response.current_l1),
            "safe_timestamp": response.current_safe_timestamp,
            "local_safe_timestamp": response.current_local_safe_timestamp,
            "finalized_timestamp": response.current_finalized_timestamp,
            "optimistic_at_timestamp": optimistic,
            "chain_ids": response.chain_ids.iter().map(|id| id.0.to_string()).collect::<Vec<_>>(),
            "data": data,
        })
    }

    fn config(server: &MockServer) -> CanaryConfig {
        let rpc = Url::parse(&server.base_url()).unwrap();
        CanaryConfig {
            superroot_rpc: rpc.clone(),
            l1_rpc: rpc,
            l1_beacon_rpc: Url::parse("https://beacon.example").unwrap(),
            l2_rpcs: vec![L2Rpc {
                chain_id: CHAIN_ID,
                url: Url::parse("https://l2.example").unwrap(),
            }],
            rollup_config_paths: None,
            l1_config_path: None,
            dependency_set_path: None,
            span_length: NonZeroU8::new(2).unwrap(),
            cadence: Duration::from_secs(1),
            max_jitter: Duration::ZERO,
            attempt_deadline: Duration::from_secs(60),
            rpc_request_timeout: Duration::from_secs(2),
            artifact: ArtifactConfig {
                base_url: Url::parse("https://artifacts.example").unwrap(),
                identity: ArtifactIdentity {
                    prestate: B256::repeat_byte(0xa1),
                    range_vkey: B256::repeat_byte(0xa2),
                    elf_sha256: B256::repeat_byte(0xa3),
                },
                max_compressed_bytes: 1024,
                max_decompressed_bytes: 2048,
                fetch_timeout: Duration::from_secs(2),
                allow_file: false,
            },
            metrics_listen: MetricsListen::Disabled,
            max_parent_response_bytes: NonZeroU32::new(1024 * 1024).unwrap(),
            max_parent_response_entries: NonZeroUsize::new(8).unwrap(),
            guest_cycle_limit: NonZeroU64::new(1_000_000).unwrap(),
            memory_limit: NonZeroU64::new(1024).unwrap(),
        }
    }

    fn register_superroot<'a>(
        server: &'a MockServer,
        timestamp: u64,
        response: &SuperRootAtTimestampResponse,
    ) -> Vec<Mock<'a>> {
        MOCK_REQUEST_IDS
            .map(|id| {
                let result = response_json(response);
                let body = json!({"jsonrpc": "2.0", "id": id, "result": result}).to_string();
                server.mock(move |when, then| {
                    when.method(POST).json_body_includes(
                        json!({
                            "id": id,
                            "method": "superroot_atTimestamp",
                            "params": [format!("0x{timestamp:x}")],
                        })
                        .to_string(),
                    );
                    then.status(200).header("content-type", "application/json").body(body);
                })
            })
            .collect()
    }

    fn register_raw_superroot<'a>(
        server: &'a MockServer,
        timestamp: u64,
        result: &str,
    ) -> Vec<Mock<'a>> {
        MOCK_REQUEST_IDS
            .map(|id| {
                let body = format!(r#"{{"jsonrpc":"2.0","id":{id},"result":{result}}}"#);
                server.mock(move |when, then| {
                    when.method(POST).json_body_includes(
                        json!({
                            "id": id,
                            "method": "superroot_atTimestamp",
                            "params": [format!("0x{timestamp:x}")],
                        })
                        .to_string(),
                    );
                    then.status(200).header("content-type", "application/json").body(body);
                })
            })
            .collect()
    }

    fn register_l1<'a>(server: &'a MockServer, number: u64, result: &Value) -> Vec<Mock<'a>> {
        MOCK_REQUEST_IDS
            .map(|id| {
                let body = json!({"jsonrpc": "2.0", "id": id, "result": result}).to_string();
                server.mock(move |when, then| {
                    when.method(POST).json_body_includes(
                        json!({
                            "id": id,
                            "method": "eth_getBlockByNumber",
                            "params": [format!("0x{number:x}"), false],
                        })
                        .to_string(),
                    );
                    then.status(200).header("content-type", "application/json").body(body);
                })
            })
            .collect()
    }

    fn register_l1_finalized<'a>(server: &'a MockServer, result: &Value) -> Vec<Mock<'a>> {
        MOCK_REQUEST_IDS
            .map(|id| {
                let body = json!({"jsonrpc": "2.0", "id": id, "result": result}).to_string();
                server.mock(move |when, then| {
                    when.method(POST).json_body_includes(
                        json!({
                            "id": id,
                            "method": "eth_getBlockByNumber",
                            "params": ["finalized", false],
                        })
                        .to_string(),
                    );
                    then.status(200).header("content-type", "application/json").body(body);
                })
            })
            .collect()
    }

    fn register_fixture(
        server: &MockServer,
        responses: &[SuperRootAtTimestampResponse],
        blocks: &BTreeMap<u64, Value>,
    ) {
        let _ = register_superroot(server, NOW, &discovery());
        let _ = register_l1_finalized(server, &block_json(block(101)));
        for (timestamp, response) in (AGREED..=TARGET).zip(responses) {
            let _ = register_superroot(server, timestamp, response);
        }
        for (&number, result) in blocks {
            let _ = register_l1(server, number, result);
        }
    }

    async fn assert_selection_rejected(
        mutate_responses: impl FnOnce(&mut Vec<SuperRootAtTimestampResponse>),
        mutate_blocks: impl FnOnce(&mut BTreeMap<u64, Value>),
        expected: &str,
    ) {
        let server = MockServer::start();
        let mut responses = base_responses();
        mutate_responses(&mut responses);
        let mut blocks = base_blocks();
        mutate_blocks(&mut blocks);
        register_fixture(&server, &responses, &blocks);
        let source = SnapshotSource::new(&config(&server)).unwrap();
        let error = source.select_finalized(NOW).await.unwrap_err();
        assert!(
            format!("{error:#}").contains(expected),
            "expected {expected:?} in error, got {error:#}",
        );
    }

    #[tokio::test]
    async fn selects_finalized_span_and_pins_at_min_of_horizon_and_finalized() {
        let server = MockServer::start();
        register_fixture(&server, &base_responses(), &base_blocks());
        let source = SnapshotSource::new(&config(&server)).unwrap();

        let snapshot = source.select_finalized(NOW).await.unwrap();

        assert_eq!(snapshot.span(), TimestampSpan::new(11, TARGET).unwrap());
        assert_eq!(snapshot.pinned_l1().block_id(), block(100));
        assert_eq!(snapshot.l1_finalized.block_id(), block(101));
        assert_eq!(snapshot.supernode_horizon, 100);
        assert_eq!(snapshot.responses().len(), 3);
        assert_eq!(snapshot.chain_ids(), &[CHAIN_ID]);
        assert_eq!(
            snapshot.canonical_l1().iter().map(|block| block.block_id().number).collect::<Vec<_>>(),
            vec![99, 100, 101],
        );
        assert_ne!(snapshot.fingerprint(), B256::ZERO);
        snapshot.synthesize_execution().unwrap();

        let server = MockServer::start();
        let mut current = discovery();
        current.current_l1 = block(102);
        let _ = register_superroot(&server, NOW, &current);
        let _ = register_l1_finalized(&server, &block_json(block(100)));
        for (timestamp, response) in (AGREED..=TARGET).zip(base_responses()) {
            let _ = register_superroot(&server, timestamp, &response);
        }
        let mut blocks = base_blocks();
        blocks.insert(102, block_json(block(102)));
        for (&number, result) in &blocks {
            let _ = register_l1(&server, number, result);
        }
        let source = SnapshotSource::new(&config(&server)).unwrap();

        let snapshot = source.select_finalized(NOW).await.unwrap();

        assert_eq!(snapshot.pinned_l1().block_id(), block(100));
        assert_eq!(snapshot.l1_finalized.block_id(), block(100));
        assert_eq!(snapshot.supernode_horizon, 101);
    }

    #[tokio::test]
    async fn required_l1_above_pin_is_an_input_error() {
        let server = MockServer::start();
        let _ = register_superroot(&server, NOW, &discovery());
        let _ = register_l1_finalized(&server, &block_json(block(99)));
        for (timestamp, response) in (AGREED..=TARGET).zip(base_responses()) {
            let _ = register_superroot(&server, timestamp, &response);
        }
        for (&number, result) in &base_blocks() {
            let _ = register_l1(&server, number, result);
        }
        let source = SnapshotSource::new(&config(&server)).unwrap();

        let error = source.select_finalized(NOW).await.unwrap_err();
        let detail = format!("{error:#}");
        assert!(detail.contains("timestamp 10 chain 10 optimistic required L1 100"));
        assert!(detail.contains("exceeds pinned L1 99"));
    }

    #[tokio::test]
    async fn pin_bound_regression_makes_snapshot_stale() {
        let server = MockServer::start();
        let _ = register_superroot(&server, NOW, &discovery());
        let mut finalized = register_l1_finalized(&server, &block_json(block(101)));
        for (timestamp, response) in (AGREED..=TARGET).zip(base_responses()) {
            let _ = register_superroot(&server, timestamp, &response);
        }
        for (&number, result) in &base_blocks() {
            let _ = register_l1(&server, number, result);
        }
        let source = SnapshotSource::new(&config(&server)).unwrap();
        let snapshot = source.select_finalized(NOW).await.unwrap();

        for mock in &mut finalized {
            mock.delete();
        }
        let _ = register_l1_finalized(&server, &block_json(block(99)));
        assert!(matches!(
            source.revalidate(&snapshot).await,
            SnapshotRevalidation::Stale { reason }
                if reason.contains("canonical pin regressed") &&
                    reason.contains("L1 finalized head 99")
        ));

        let server = MockServer::start();
        let mut discovery_mocks = register_superroot(&server, NOW, &discovery());
        let _ = register_l1_finalized(&server, &block_json(block(101)));
        for (timestamp, response) in (AGREED..=TARGET).zip(base_responses()) {
            let _ = register_superroot(&server, timestamp, &response);
        }
        for (&number, result) in &base_blocks() {
            let _ = register_l1(&server, number, result);
        }
        let source = SnapshotSource::new(&config(&server)).unwrap();
        let snapshot = source.select_finalized(NOW).await.unwrap();

        for mock in &mut discovery_mocks {
            mock.delete();
        }
        let mut regressed_discovery = discovery();
        regressed_discovery.current_l1 = block(100);
        let _ = register_superroot(&server, NOW, &regressed_discovery);
        assert!(matches!(
            source.revalidate(&snapshot).await,
            SnapshotRevalidation::Stale { reason }
                if reason.contains("canonical pin regressed") &&
                    reason.contains("supernode horizon 99")
        ));
    }

    #[tokio::test]
    async fn rejects_partial_untrusted_or_noncanonical_snapshots() {
        assert_selection_rejected(|responses| responses[1].data = None, |_| {}, "has no data")
            .await;
        assert_selection_rejected(
            |responses| {
                responses[1].current_l1 = block(99);
            },
            |_| {},
            "has not advanced beyond",
        )
        .await;
        assert_selection_rejected(
            |responses| responses[1] = response(11, 11),
            |_| {},
            "do not match configured chain IDs",
        )
        .await;
        assert_selection_rejected(
            |responses| responses[1].current_finalized_timestamp = TARGET - 1,
            |_| {},
            "finalized horizon",
        )
        .await;
        assert_selection_rejected(
            |responses| responses[1].current_safe_timestamp = TARGET - 1,
            |_| {},
            "safe horizon",
        )
        .await;
        assert_selection_rejected(
            |responses| responses[1].current_local_safe_timestamp = TARGET - 1,
            |_| {},
            "local-safe horizon",
        )
        .await;
        assert_selection_rejected(
            |_| {},
            |blocks| {
                blocks.insert(99, Value::Null);
            },
            "not found",
        )
        .await;
        assert_selection_rejected(
            |_| {},
            |blocks| {
                blocks.insert(99, block_json(block(98)));
            },
            "returned block 98",
        )
        .await;
        assert_selection_rejected(
            |_| {},
            |blocks| {
                let mut wrong = block(99);
                wrong.hash = B256::repeat_byte(0xff);
                blocks.insert(99, block_json(wrong));
            },
            "has hash",
        )
        .await;
    }

    #[tokio::test]
    async fn rejects_duplicate_canonical_optimistic_chain_ids() {
        let response = response_json(&response(AGREED, CHAIN_ID));
        let output = response["optimistic_at_timestamp"][CHAIN_ID.to_string()].to_string();
        let raw = response.to_string();

        for duplicate_key in [CHAIN_ID.to_string(), format!("0x{CHAIN_ID:x}")] {
            let server = MockServer::start();
            let duplicate = raw.replacen(
                &format!("\"{CHAIN_ID}\":"),
                &format!("\"{CHAIN_ID}\":{output},\"{duplicate_key}\":"),
                1,
            );
            let _ = register_raw_superroot(&server, AGREED, &duplicate);
            let source = SnapshotSource::new(&config(&server)).unwrap();

            let error = source.fetch_superroot(AGREED).await.unwrap_err().into_anyhow();
            assert!(
                format!("{error:#}").contains("duplicate canonical chain ID"),
                "unexpected error: {error:#}",
            );
        }
    }

    #[tokio::test]
    async fn discards_result_when_snapshot_changes_during_execution() {
        let server = MockServer::start();
        let _ = register_superroot(&server, NOW, &discovery());
        let _ = register_l1_finalized(&server, &block_json(block(101)));
        let responses = base_responses();
        let _ = register_superroot(&server, AGREED, &responses[0]);
        let _ = register_superroot(&server, AGREED + 1, &responses[1]);
        let mut target = register_superroot(&server, TARGET, &responses[2]);
        for (&number, result) in &base_blocks() {
            let _ = register_l1(&server, number, result);
        }
        let source = SnapshotSource::new(&config(&server)).unwrap();
        let snapshot = source.select_finalized(NOW).await.unwrap();

        for mock in &mut target {
            mock.delete();
        }
        let mut changed = response(TARGET, CHAIN_ID);
        let changed_output = output(0xee, block(100));
        changed
            .optimistic_at_timestamp
            .insert(ChainId(U256::from(CHAIN_ID)), changed_output.clone());
        let data = changed.data.as_mut().unwrap();
        data.super_v1.chains[0].output = changed_output.output_root;
        data.super_root =
            hash_super_root_proof(&proof_from_super_v1(&data.super_v1).unwrap()).unwrap();
        let mut changed_mocks = register_superroot(&server, TARGET, &changed);

        assert!(matches!(source.revalidate(&snapshot).await, SnapshotRevalidation::Stale { .. }));

        for mock in &mut changed_mocks {
            mock.delete();
        }
        server.mock(|when, then| {
            when.method(POST)
                .body_includes("\"method\":\"superroot_atTimestamp\"")
                .body_includes(format!("\"0x{TARGET:x}\""));
            then.status(503);
        });
        assert!(matches!(
            source.revalidate(&snapshot).await,
            SnapshotRevalidation::Unavailable { .. }
        ));
    }

    #[tokio::test]
    async fn finalized_horizon_regression_makes_snapshot_stale() {
        let server = MockServer::start();
        let _ = register_superroot(&server, NOW, &discovery());
        let _ = register_l1_finalized(&server, &block_json(block(101)));
        let responses = base_responses();
        let _ = register_superroot(&server, AGREED, &responses[0]);
        let _ = register_superroot(&server, AGREED + 1, &responses[1]);
        let mut target = register_superroot(&server, TARGET, &responses[2]);
        for (&number, result) in &base_blocks() {
            let _ = register_l1(&server, number, result);
        }
        let source = SnapshotSource::new(&config(&server)).unwrap();
        let snapshot = source.select_finalized(NOW).await.unwrap();

        for mock in &mut target {
            mock.delete();
        }
        let mut regressed = responses[2].clone();
        regressed.current_finalized_timestamp = TARGET - 1;
        let _ = register_superroot(&server, TARGET, &regressed);

        assert!(matches!(
            source.revalidate(&snapshot).await,
            SnapshotRevalidation::Stale { reason }
                if reason.contains("finalized horizon")
        ));
    }
}
