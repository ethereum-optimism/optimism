//! Environment-driven service configuration.

use std::{
    collections::{BTreeMap, BTreeSet},
    env,
    ffi::OsStr,
    fs,
    num::{NonZeroU8, NonZeroU32, NonZeroU64, NonZeroUsize},
    path::{Path, PathBuf},
    str::FromStr,
    time::Duration,
};

use anyhow::{Context, Result, anyhow, bail, ensure};
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_sp1_host_utils::metrics::MetricsListen;
use kona_sp1_super_range_executor::HostInputs;
use url::Url;

use crate::artifact::{ArtifactConfig, ArtifactIdentity};

const ENV_PREFIX: &str = "KONA_ZKVM_CANARY_";
const MAX_SPAN_LENGTH: u8 = 16;
const MAX_CONFIGURED_CHAINS: usize = 256;
const MAX_CONFIG_FILE_BYTES: u64 = 16 * 1024 * 1024;

const DEFAULT_CADENCE_SECONDS: u64 = 300;
const DEFAULT_JITTER_SECONDS: u64 = 30;
const DEFAULT_ATTEMPT_DEADLINE_SECONDS: u64 = 3 * 60 * 60;
const DEFAULT_RPC_REQUEST_TIMEOUT_SECONDS: u64 = 30;
const DEFAULT_ARTIFACT_REQUEST_TIMEOUT_SECONDS: u64 = 60;
const DEFAULT_PARENT_RESPONSE_BYTES: u32 = 4 * 1024 * 1024;
const DEFAULT_PARENT_RESPONSE_ENTRIES: usize = 256;
const DEFAULT_ARTIFACT_COMPRESSED_BYTES: usize = 256 * 1024 * 1024;
const DEFAULT_ARTIFACT_DECOMPRESSED_BYTES: usize = 1024 * 1024 * 1024;
const DEFAULT_MEMORY_LIMIT_BYTES: u64 = 24 * 1024 * 1024 * 1024;

/// Service operation selected before environment validation.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ServiceMode {
    /// Repeated production operation.
    Loop,
    /// One diagnostic attempt.
    Once,
}

/// An explicitly chain-keyed L2 execution endpoint.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct L2Rpc {
    /// L2 chain ID expected from the endpoint.
    pub chain_id: u64,
    /// L2 execution JSON-RPC URL.
    pub url: Url,
}

/// Validated runtime configuration for one network and artifact release.
#[derive(Clone, Debug)]
pub struct CanaryConfig {
    /// Supernode endpoint serving `superroot_atTimestamp`.
    pub superroot_rpc: Url,
    /// L1 execution JSON-RPC endpoint.
    pub l1_rpc: Url,
    /// L1 beacon API endpoint.
    pub l1_beacon_rpc: Url,
    /// L2 execution endpoints sorted by their configured chain IDs.
    pub l2_rpcs: Vec<L2Rpc>,
    /// Optional rollup config overrides with exact L2 endpoint coverage.
    pub rollup_config_paths: Option<Vec<PathBuf>>,
    /// Optional L1 chain config override.
    pub l1_config_path: Option<PathBuf>,
    /// Optional dependency-set override with exact L2 endpoint coverage.
    pub dependency_set_path: Option<PathBuf>,
    /// Number of finalized timestamps selected per attempt.
    pub span_length: NonZeroU8,
    /// Wait after a completed attempt.
    pub cadence: Duration,
    /// Maximum random addition to the cadence.
    pub max_jitter: Duration,
    /// Monotonic deadline covering the whole attempt.
    pub attempt_deadline: Duration,
    /// Deadline for each parent JSON-RPC request.
    pub rpc_request_timeout: Duration,
    /// Expected release and bounds for the range ELF.
    pub artifact: ArtifactConfig,
    /// Prometheus listener selection.
    pub metrics_listen: MetricsListen,
    /// Maximum parent JSON-RPC response body size.
    pub max_parent_response_bytes: NonZeroU32,
    /// Maximum entries in each parent response collection.
    pub max_parent_response_entries: NonZeroUsize,
    /// Maximum cycles permitted for each SP1 guest execution.
    pub guest_cycle_limit: NonZeroU64,
    /// Value applied to the current SP1 process as `MEMORY_LIMIT`.
    pub memory_limit: NonZeroU64,
}

impl CanaryConfig {
    /// Parses and validates all `KONA_ZKVM_CANARY_*` variables.
    ///
    /// `KONA_ZKVM_CANARY_L2_RPCS` is a comma-separated list of
    /// `<chain-id>=<http(s)-url>` entries.
    pub fn from_env(mode: ServiceMode) -> Result<Self> {
        reject_trace_file(env::var_os("TRACE_FILE").as_deref())?;
        let values = env::vars()
            .filter_map(|(name, value)| {
                name.strip_prefix(ENV_PREFIX).map(|suffix| (suffix.to_string(), value))
            })
            .collect();
        Self::from_values(&values, mode)
    }

    /// Returns the deployment inputs accepted by the shared executor host.
    pub fn host_inputs(&self) -> HostInputs {
        HostInputs {
            l1_node_address: self.l1_rpc.to_string(),
            l1_beacon_address: self.l1_beacon_rpc.to_string(),
            l2_node_addresses: self.l2_rpcs.iter().map(|rpc| rpc.url.to_string()).collect(),
            rollup_config_paths: self.rollup_config_paths.clone(),
            l1_config_path: self.l1_config_path.clone(),
            dependency_set_path: self.dependency_set_path.clone(),
        }
    }

    /// Returns the configured chain universe.
    pub fn chain_ids(&self) -> impl ExactSizeIterator<Item = u64> + '_ {
        self.l2_rpcs.iter().map(|rpc| rpc.chain_id)
    }

    fn from_values(values: &BTreeMap<String, String>, mode: ServiceMode) -> Result<Self> {
        let superroot_rpc = parse_rpc_url(values, "SUPERROOT_RPC")?;
        let l1_rpc = parse_rpc_url(values, "L1_RPC")?;
        let l1_beacon_rpc = parse_rpc_url(values, "L1_BEACON_RPC")?;
        let l2_rpcs = parse_l2_rpcs(required(values, "L2_RPCS")?)?;
        let configured_chain_ids = l2_rpcs.iter().map(|rpc| rpc.chain_id).collect::<BTreeSet<_>>();

        let rollup_config_paths = parse_path_list(values.get("ROLLUP_CONFIG_PATHS"))?;
        validate_rollup_configs(rollup_config_paths.as_deref(), &configured_chain_ids)?;
        let l1_config_path = optional_path(values, "L1_CONFIG_PATH");
        validate_json_file::<L1ChainConfig>(l1_config_path.as_deref(), "L1 config")?;
        let dependency_set_path = optional_path(values, "DEPENDENCY_SET_PATH");
        validate_dependency_set(dependency_set_path.as_deref(), &configured_chain_ids)?;

        let span_length = parse_nonzero_or(values, "FINALIZED_SPAN", 1u8)?;
        ensure!(
            span_length.get() <= MAX_SPAN_LENGTH,
            "{} must be in 1..={MAX_SPAN_LENGTH}",
            env_name("FINALIZED_SPAN"),
        );
        let cadence_seconds = parse_nonzero_or(values, "CADENCE_SECONDS", DEFAULT_CADENCE_SECONDS)?;
        let max_jitter_seconds =
            parse_or(values, "JITTER_SECONDS", DEFAULT_JITTER_SECONDS.min(cadence_seconds.get()))?;
        ensure!(
            max_jitter_seconds <= cadence_seconds.get(),
            "{} must not exceed {}",
            env_name("JITTER_SECONDS"),
            env_name("CADENCE_SECONDS"),
        );
        let attempt_deadline_seconds =
            parse_nonzero_or(values, "ATTEMPT_DEADLINE_SECONDS", DEFAULT_ATTEMPT_DEADLINE_SECONDS)?;
        let rpc_request_timeout_seconds = parse_nonzero_or(
            values,
            "RPC_REQUEST_TIMEOUT_SECONDS",
            DEFAULT_RPC_REQUEST_TIMEOUT_SECONDS,
        )?;
        let artifact_request_timeout_seconds = parse_nonzero_or(
            values,
            "ARTIFACT_REQUEST_TIMEOUT_SECONDS",
            DEFAULT_ARTIFACT_REQUEST_TIMEOUT_SECONDS,
        )?;

        let artifact_url = parse_artifact_url(values, mode)?;
        let artifact_identity = ArtifactIdentity {
            prestate: parse_required(values, "PRESTATE")?,
            range_vkey: parse_required(values, "RANGE_VKEY")?,
            elf_sha256: parse_required(values, "ELF_SHA256")?,
        };
        let max_artifact_compressed_bytes = parse_nonzero_or(
            values,
            "MAX_ARTIFACT_COMPRESSED_BYTES",
            DEFAULT_ARTIFACT_COMPRESSED_BYTES,
        )?;
        let max_artifact_decompressed_bytes = parse_nonzero_or(
            values,
            "MAX_ARTIFACT_DECOMPRESSED_BYTES",
            DEFAULT_ARTIFACT_DECOMPRESSED_BYTES,
        )?;
        let artifact = ArtifactConfig {
            base_url: artifact_url,
            identity: artifact_identity,
            max_compressed_bytes: max_artifact_compressed_bytes.get(),
            max_decompressed_bytes: max_artifact_decompressed_bytes.get(),
            fetch_timeout: Duration::from_secs(artifact_request_timeout_seconds.get()),
            allow_file: mode == ServiceMode::Once,
        };

        let metrics_listen = parse_or(values, "METRICS_PORT", MetricsListen::default())?;
        let max_parent_response_bytes =
            parse_nonzero_or(values, "MAX_PARENT_RESPONSE_BYTES", DEFAULT_PARENT_RESPONSE_BYTES)?;
        let max_parent_response_entries = parse_nonzero_or(
            values,
            "MAX_PARENT_RESPONSE_ENTRIES",
            DEFAULT_PARENT_RESPONSE_ENTRIES,
        )?;
        ensure!(
            max_parent_response_entries.get() <= MAX_CONFIGURED_CHAINS,
            "{} must be at most {MAX_CONFIGURED_CHAINS}",
            env_name("MAX_PARENT_RESPONSE_ENTRIES"),
        );
        ensure!(
            l2_rpcs.len() <= max_parent_response_entries.get(),
            "configured L2 chain count exceeds {}",
            env_name("MAX_PARENT_RESPONSE_ENTRIES"),
        );
        let guest_cycle_limit = parse_nonzero_required::<u64>(values, "GUEST_CYCLE_LIMIT")?;
        let memory_limit = parse_nonzero_or(values, "MEMORY_LIMIT", DEFAULT_MEMORY_LIMIT_BYTES)?;

        Ok(Self {
            superroot_rpc,
            l1_rpc,
            l1_beacon_rpc,
            l2_rpcs,
            rollup_config_paths,
            l1_config_path,
            dependency_set_path,
            span_length,
            cadence: Duration::from_secs(cadence_seconds.get()),
            max_jitter: Duration::from_secs(max_jitter_seconds),
            attempt_deadline: Duration::from_secs(attempt_deadline_seconds.get()),
            rpc_request_timeout: Duration::from_secs(rpc_request_timeout_seconds.get()),
            artifact,
            metrics_listen,
            max_parent_response_bytes,
            max_parent_response_entries,
            guest_cycle_limit,
            memory_limit,
        })
    }
}

fn reject_trace_file(trace_file: Option<&OsStr>) -> Result<()> {
    ensure!(
        trace_file.is_none(),
        "TRACE_FILE must be unset because kona-zkvm-canary writes no files"
    );
    Ok(())
}

fn env_name(suffix: &str) -> String {
    format!("{ENV_PREFIX}{suffix}")
}

fn required<'a>(values: &'a BTreeMap<String, String>, suffix: &str) -> Result<&'a str> {
    values
        .get(suffix)
        .map(String::as_str)
        .filter(|value| !value.trim().is_empty())
        .ok_or_else(|| anyhow!("{} not set", env_name(suffix)))
}

fn parse_required<T>(values: &BTreeMap<String, String>, suffix: &str) -> Result<T>
where
    T: FromStr,
    T::Err: std::fmt::Display,
{
    required(values, suffix)?
        .parse()
        .map_err(|error| anyhow!("invalid {}: {error}", env_name(suffix)))
}

fn parse_or<T>(values: &BTreeMap<String, String>, suffix: &str, default: T) -> Result<T>
where
    T: FromStr,
    T::Err: std::fmt::Display,
{
    values.get(suffix).filter(|value| !value.trim().is_empty()).map_or_else(
        || Ok(default),
        |value| value.parse().map_err(|error| anyhow!("invalid {}: {error}", env_name(suffix))),
    )
}

fn parse_nonzero_or<T>(
    values: &BTreeMap<String, String>,
    suffix: &str,
    default: T,
) -> Result<T::NonZero>
where
    T: NonZeroValue + FromStr,
    T::Err: std::fmt::Display,
{
    let value = parse_or(values, suffix, default)?;
    value.into_nonzero().ok_or_else(|| anyhow!("{} must be non-zero", env_name(suffix)))
}

fn parse_nonzero_required<T>(values: &BTreeMap<String, String>, suffix: &str) -> Result<T::NonZero>
where
    T: NonZeroValue + FromStr,
    T::Err: std::fmt::Display,
{
    let value = parse_required::<T>(values, suffix)?;
    value.into_nonzero().ok_or_else(|| anyhow!("{} must be non-zero", env_name(suffix)))
}

trait NonZeroValue: Sized {
    type NonZero;

    fn into_nonzero(self) -> Option<Self::NonZero>;
}

macro_rules! impl_nonzero_value {
    ($primitive:ty, $nonzero:ty) => {
        impl NonZeroValue for $primitive {
            type NonZero = $nonzero;

            fn into_nonzero(self) -> Option<Self::NonZero> {
                <$nonzero>::new(self)
            }
        }
    };
}

impl_nonzero_value!(u8, NonZeroU8);
impl_nonzero_value!(u32, NonZeroU32);
impl_nonzero_value!(u64, NonZeroU64);
impl_nonzero_value!(usize, NonZeroUsize);

fn parse_rpc_url(values: &BTreeMap<String, String>, suffix: &str) -> Result<Url> {
    let url = parse_url(required(values, suffix)?, suffix)?;
    validate_network_url(&url, suffix)?;
    Ok(url)
}

fn parse_url(value: &str, suffix: &str) -> Result<Url> {
    Url::parse(value).map_err(|_| anyhow!("invalid {} URL", env_name(suffix)))
}

fn validate_network_url(url: &Url, suffix: &str) -> Result<()> {
    ensure!(
        matches!(url.scheme(), "http" | "https"),
        "{} must use http or https",
        env_name(suffix),
    );
    validate_secret_free_url(url, suffix)
}

fn validate_secret_free_url(url: &Url, suffix: &str) -> Result<()> {
    ensure!(
        url.username().is_empty() && url.password().is_none(),
        "{} must not contain credentials",
        env_name(suffix),
    );
    ensure!(url.query().is_none(), "{} must not contain a query", env_name(suffix));
    ensure!(url.fragment().is_none(), "{} must not contain a fragment", env_name(suffix));
    Ok(())
}

fn parse_artifact_url(values: &BTreeMap<String, String>, mode: ServiceMode) -> Result<Url> {
    let suffix = "PRESTATES_URL";
    let url = parse_url(required(values, suffix)?, suffix)?;
    match (url.scheme(), mode) {
        ("https", _) => validate_secret_free_url(&url, suffix)?,
        ("file", ServiceMode::Once) => {
            validate_secret_free_url(&url, suffix)?;
            ensure!(url.to_file_path().is_ok(), "{} is not a local file URL", env_name(suffix));
        }
        ("file", ServiceMode::Loop) => {
            bail!("{} may use file only with --once", env_name(suffix));
        }
        _ => bail!("{} must use https", env_name(suffix)),
    }
    Ok(url)
}

fn parse_l2_rpcs(value: &str) -> Result<Vec<L2Rpc>> {
    let mut endpoints = BTreeMap::new();
    for (index, entry) in value.split(',').map(str::trim).enumerate() {
        ensure!(!entry.is_empty(), "{} entry {} is empty", env_name("L2_RPCS"), index + 1);
        let (chain_id, raw_url) = entry.split_once('=').ok_or_else(|| {
            anyhow!(
                "{} entry {} must be formatted as <chain-id>=<url>",
                env_name("L2_RPCS"),
                index + 1,
            )
        })?;
        let chain_id = parse_chain_id(chain_id).with_context(|| {
            format!("invalid {} chain ID at entry {}", env_name("L2_RPCS"), index + 1)
        })?;
        let url = parse_url(raw_url, "L2_RPCS")?;
        validate_network_url(&url, "L2_RPCS")?;
        ensure!(
            endpoints.insert(chain_id, url).is_none(),
            "{} contains duplicate chain ID {chain_id}",
            env_name("L2_RPCS"),
        );
    }
    ensure!(!endpoints.is_empty(), "{} must contain at least one endpoint", env_name("L2_RPCS"));
    ensure!(
        endpoints.len() <= MAX_CONFIGURED_CHAINS,
        "{} must contain at most {MAX_CONFIGURED_CHAINS} endpoints",
        env_name("L2_RPCS"),
    );
    Ok(endpoints.into_iter().map(|(chain_id, url)| L2Rpc { chain_id, url }).collect())
}

fn parse_chain_id(value: &str) -> Result<u64> {
    let value = value.trim();
    let chain_id = if let Some(hex) = value.strip_prefix("0x") {
        u64::from_str_radix(hex, 16)?
    } else {
        value.parse()?
    };
    ensure!(chain_id != 0, "chain ID must be non-zero");
    Ok(chain_id)
}

fn optional_path(values: &BTreeMap<String, String>, suffix: &str) -> Option<PathBuf> {
    values
        .get(suffix)
        .map(String::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(PathBuf::from)
}

fn parse_path_list(value: Option<&String>) -> Result<Option<Vec<PathBuf>>> {
    let Some(value) = value.map(String::as_str).map(str::trim).filter(|value| !value.is_empty())
    else {
        return Ok(None);
    };
    let paths = value.split(',').map(str::trim).map(PathBuf::from).collect::<Vec<_>>();
    ensure!(
        paths.iter().all(|path| !path.as_os_str().is_empty()),
        "{} contains an empty path",
        env_name("ROLLUP_CONFIG_PATHS"),
    );
    ensure!(
        paths.len() <= MAX_CONFIGURED_CHAINS,
        "{} must contain at most {MAX_CONFIGURED_CHAINS} paths",
        env_name("ROLLUP_CONFIG_PATHS"),
    );
    Ok(Some(paths))
}

fn read_json_file<T>(path: &Path, label: &str) -> Result<T>
where
    T: serde::de::DeserializeOwned,
{
    let metadata = fs::metadata(path)
        .with_context(|| format!("failed to inspect {label} {}", path.display()))?;
    ensure!(
        metadata.len() <= MAX_CONFIG_FILE_BYTES,
        "{label} {} exceeds {MAX_CONFIG_FILE_BYTES} bytes",
        path.display(),
    );
    let bytes =
        fs::read(path).with_context(|| format!("failed to read {label} {}", path.display()))?;
    serde_json::from_slice(&bytes)
        .with_context(|| format!("failed to parse {label} {}", path.display()))
}

fn validate_json_file<T>(path: Option<&Path>, label: &str) -> Result<()>
where
    T: serde::de::DeserializeOwned,
{
    if let Some(path) = path {
        let _: T = read_json_file(path, label)?;
    }
    Ok(())
}

fn validate_rollup_configs(
    paths: Option<&[PathBuf]>,
    configured_chain_ids: &BTreeSet<u64>,
) -> Result<()> {
    let Some(paths) = paths else { return Ok(()) };
    let mut rollup_chain_ids = BTreeSet::new();
    for path in paths {
        let config: RollupConfig = read_json_file(path, "rollup config")?;
        let chain_id = config.l2_chain_id.id();
        ensure!(
            rollup_chain_ids.insert(chain_id),
            "rollup configs contain duplicate chain ID {chain_id}",
        );
    }
    ensure!(
        &rollup_chain_ids == configured_chain_ids,
        "rollup config chain IDs {rollup_chain_ids:?} do not match configured L2 chain IDs {configured_chain_ids:?}",
    );
    Ok(())
}

fn validate_dependency_set(
    path: Option<&Path>,
    configured_chain_ids: &BTreeSet<u64>,
) -> Result<()> {
    let Some(path) = path else { return Ok(()) };
    let dependency_set: DependencySet = read_json_file(path, "dependency set")?;
    let dependency_chain_ids = dependency_set.dependencies.keys().copied().collect::<BTreeSet<_>>();
    ensure!(
        &dependency_chain_ids == configured_chain_ids,
        "dependency-set chain IDs {dependency_chain_ids:?} do not match configured L2 chain IDs {configured_chain_ids:?}",
    );
    Ok(())
}

#[cfg(test)]
mod tests {
    use alloy_primitives::B256;

    use super::*;

    fn valid_values() -> BTreeMap<String, String> {
        BTreeMap::from([
            ("SUPERROOT_RPC".into(), "https://superroot.example".into()),
            ("L1_RPC".into(), "https://l1.example".into()),
            ("L1_BEACON_RPC".into(), "https://beacon.example".into()),
            ("L2_RPCS".into(), "10=https://op.example,8453=https://base.example".into()),
            ("PRESTATES_URL".into(), "https://artifacts.example/releases/v1/".into()),
            ("PRESTATE".into(), format!("{}", B256::repeat_byte(0x11))),
            ("RANGE_VKEY".into(), format!("{}", B256::repeat_byte(0x22))),
            ("ELF_SHA256".into(), format!("{}", B256::repeat_byte(0x33))),
            ("GUEST_CYCLE_LIMIT".into(), "1000000".into()),
        ])
    }

    fn assert_rejected(values: &BTreeMap<String, String>, needle: &str) {
        let error = CanaryConfig::from_values(values, ServiceMode::Loop).unwrap_err();
        assert!(
            format!("{error:#}").contains(needle),
            "expected {needle:?} in error, got {error:#}",
        );
    }

    #[test]
    fn config_rejects_invalid_bounds_and_endpoints_before_startup() {
        let config = CanaryConfig::from_values(&valid_values(), ServiceMode::Loop).unwrap();
        assert_eq!(config.span_length.get(), 1);
        assert_eq!(config.attempt_deadline, Duration::from_secs(3 * 60 * 60));
        assert_eq!(config.guest_cycle_limit.get(), 1_000_000);
        assert_eq!(config.memory_limit.get(), 24 * 1024 * 1024 * 1024);
        assert_eq!(config.chain_ids().collect::<Vec<_>>(), vec![10, 8453]);

        let mut short_cadence = valid_values();
        short_cadence.insert("CADENCE_SECONDS".into(), "1".into());
        assert_eq!(
            CanaryConfig::from_values(&short_cadence, ServiceMode::Loop).unwrap().max_jitter,
            Duration::from_secs(1),
        );

        for (name, value, needle) in [
            ("FINALIZED_SPAN", "0", "non-zero"),
            ("FINALIZED_SPAN", "17", "1..=16"),
            ("CADENCE_SECONDS", "0", "non-zero"),
            ("ATTEMPT_DEADLINE_SECONDS", "0", "non-zero"),
            ("RPC_REQUEST_TIMEOUT_SECONDS", "0", "non-zero"),
            ("ARTIFACT_REQUEST_TIMEOUT_SECONDS", "0", "non-zero"),
            ("MAX_PARENT_RESPONSE_BYTES", "0", "non-zero"),
            ("MAX_PARENT_RESPONSE_ENTRIES", "0", "non-zero"),
            ("MAX_ARTIFACT_COMPRESSED_BYTES", "0", "non-zero"),
            ("MAX_ARTIFACT_DECOMPRESSED_BYTES", "0", "non-zero"),
            ("GUEST_CYCLE_LIMIT", "0", "non-zero"),
            ("MEMORY_LIMIT", "0", "non-zero"),
            ("JITTER_SECONDS", "301", "must not exceed"),
        ] {
            let mut values = valid_values();
            values.insert(name.into(), value.into());
            assert_rejected(&values, needle);
        }

        for missing in [
            "SUPERROOT_RPC",
            "L1_RPC",
            "L1_BEACON_RPC",
            "L2_RPCS",
            "PRESTATES_URL",
            "PRESTATE",
            "RANGE_VKEY",
            "ELF_SHA256",
            "GUEST_CYCLE_LIMIT",
        ] {
            let mut values = valid_values();
            values.remove(missing);
            assert_rejected(&values, "not set");
        }

        let invalid_endpoints = [
            ("SUPERROOT_RPC", "ws://superroot.example", "http or https"),
            ("L1_RPC", "https://user:secret@l1.example", "credentials"),
            ("L1_BEACON_RPC", "https://beacon.example?token=secret", "query"),
            ("L2_RPCS", "10=file:///tmp/l2", "http or https"),
            ("PRESTATES_URL", "http://artifacts.example", "must use https"),
            ("PRESTATES_URL", "https://user:secret@artifacts.example", "credentials"),
        ];
        for (name, value, needle) in invalid_endpoints {
            let mut values = valid_values();
            values.insert(name.into(), value.into());
            assert_rejected(&values, needle);
        }

        let mut values = valid_values();
        values
            .insert("L2_RPCS".into(), "10=https://first.example,10=https://second.example".into());
        assert_rejected(&values, "duplicate chain ID 10");

        let mut values = valid_values();
        values.insert("PRESTATES_URL".into(), "file:///tmp/prestates".into());
        assert_rejected(&values, "only with --once");
        let once = CanaryConfig::from_values(&values, ServiceMode::Once).unwrap();
        assert!(once.artifact.allow_file);
    }

    #[test]
    fn config_validates_registry_override_chain_coverage() {
        let directory = tempfile::tempdir().unwrap();
        let rollup_path = directory.path().join("rollup.json");
        let rollup = RollupConfig { l2_chain_id: 10.into(), ..Default::default() };
        fs::write(&rollup_path, serde_json::to_vec(&rollup).unwrap()).unwrap();
        let dependency_path = directory.path().join("dependencies.json");
        fs::write(
            &dependency_path,
            br#"{"dependencies":{"10":{}},"overrideMessageExpiryWindow":null}"#,
        )
        .unwrap();

        let mut values = valid_values();
        values.insert("L2_RPCS".into(), "10=https://op.example".into());
        values.insert("ROLLUP_CONFIG_PATHS".into(), rollup_path.display().to_string());
        values.insert("DEPENDENCY_SET_PATH".into(), dependency_path.display().to_string());
        CanaryConfig::from_values(&values, ServiceMode::Loop).unwrap();

        values.insert("L2_RPCS".into(), "8453=https://base.example".into());
        assert_rejected(&values, "do not match configured L2 chain IDs");
    }

    #[test]
    fn config_errors_do_not_expose_url_secrets() {
        let mut values = valid_values();
        values.insert(
            "L2_RPCS".into(),
            "10=https://user:do-not-print@op.example?token=also-hidden".into(),
        );
        let error = CanaryConfig::from_values(&values, ServiceMode::Loop).unwrap_err();
        let rendered = format!("{error:#}");
        assert!(!rendered.contains("do-not-print"));
        assert!(!rendered.contains("also-hidden"));
    }

    #[test]
    fn config_rejects_sp1_trace_file_output() {
        let error = reject_trace_file(Some(OsStr::new("/tmp/sp1-profile.json"))).unwrap_err();
        assert_eq!(
            error.to_string(),
            "TRACE_FILE must be unset because kona-zkvm-canary writes no files"
        );
        reject_trace_file(None).unwrap();
    }
}
