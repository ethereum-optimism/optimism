use reqwest::Client;
use sha3::{Digest, Keccak256};
use tracing::info;

const DEBUG_QUOTE_SERVICE_URL: &str = "http://ns31695324.ip-141-94-163.eu:10080/attest";

// Raw TDX v4 quote structure constants
// Raw quote has a 48-byte header before the TD10ReportBody
const HEADER_LENGTH: usize = 48;
const TD_REPORT10_LENGTH: usize = 584;

// TD10ReportBody field offsets
// These offsets correspond to the Solidity parseRawReportBody implementation
const OFFSET_TD_ATTRIBUTES: usize = 120;
const OFFSET_XFAM: usize = 128;
const OFFSET_MR_TD: usize = 136;
const OFFSET_MR_CONFIG_ID: usize = 184;
const OFFSET_MR_OWNER: usize = 232;
const OFFSET_MR_OWNER_CONFIG: usize = 280;
const OFFSET_RT_MR0: usize = 328;
const OFFSET_RT_MR1: usize = 376;
const OFFSET_RT_MR2: usize = 424;
const OFFSET_RT_MR3: usize = 472;

// Field lengths
const MEASUREMENT_REGISTER_LENGTH: usize = 48;
const ATTRIBUTE_LENGTH: usize = 8;

/// Parsed TDX quote report body containing measurement registers and attributes
#[derive(Debug, Clone)]
pub struct ParsedQuote {
    pub mr_td: [u8; 48],
    pub rt_mr0: [u8; 48],
    pub rt_mr1: [u8; 48],
    pub rt_mr2: [u8; 48],
    pub rt_mr3: [u8; 48],
    pub mr_config_id: [u8; 48],
    pub mr_owner: [u8; 48],
    pub mr_owner_config: [u8; 48],
    pub xfam: u64,
    pub td_attributes: u64,
}

/// Configuration for attestation
#[derive(Default)]
pub struct AttestationConfig {
    /// If true, uses the debug HTTP service instead of real TDX hardware
    pub debug: bool,
    /// The URL of the quote provider
    pub quote_provider: Option<String>,
}
/// Remote attestation provider
#[derive(Debug, Clone)]
pub struct RemoteAttestationProvider {
    client: Client,
    service_url: String,
}

impl RemoteAttestationProvider {
    pub fn new(service_url: String) -> Self {
        let client = Client::new();
        Self {
            client,
            service_url,
        }
    }
}

impl RemoteAttestationProvider {
    pub async fn get_attestation(&self, report_data: [u8; 64]) -> eyre::Result<Vec<u8>> {
        let report_data_hex = hex::encode(report_data);
        let url = format!("{}/{}", self.service_url, report_data_hex);

        info!(target: "flashtestations", url = url, "fetching quote from remote attestation provider");

        let response = self
            .client
            .get(&url)
            .timeout(std::time::Duration::from_secs(10))
            .send()
            .await?
            .error_for_status()?;
        let body = response.bytes().await?.to_vec();

        Ok(body)
    }
}

pub fn get_attestation_provider(config: AttestationConfig) -> RemoteAttestationProvider {
    if config.debug {
        RemoteAttestationProvider::new(
            config
                .quote_provider
                .unwrap_or(DEBUG_QUOTE_SERVICE_URL.to_string()),
        )
    } else {
        RemoteAttestationProvider::new(
            config
                .quote_provider
                .expect("remote quote provider must be specified when not in debug mode"),
        )
    }
}

/// Parse the TDX report body from a raw quote
/// Extracts measurement registers and attributes according to TD10ReportBody specification
/// https://github.com/flashbots/flashtestations/tree/7cc7f68492fe672a823dd2dead649793aac1f216
pub fn parse_report_body(raw_quote: &[u8]) -> eyre::Result<ParsedQuote> {
    // Validate quote length
    if raw_quote.len() < HEADER_LENGTH + TD_REPORT10_LENGTH {
        eyre::bail!(
            "invalid quote length: {}, expected at least {}",
            raw_quote.len(),
            HEADER_LENGTH + TD_REPORT10_LENGTH
        );
    }

    // Skip the 48-byte header to get to the TD10ReportBody
    let report_body = &raw_quote[HEADER_LENGTH..];

    // Extract fields exactly as parseRawReportBody does in Solidity
    // Using named offset constants to match Solidity implementation exactly
    let mr_td: [u8; 48] = report_body[OFFSET_MR_TD..OFFSET_MR_TD + MEASUREMENT_REGISTER_LENGTH]
        .try_into()
        .map_err(|_| eyre::eyre!("failed to extract mr_td"))?;
    let rt_mr0: [u8; 48] = report_body[OFFSET_RT_MR0..OFFSET_RT_MR0 + MEASUREMENT_REGISTER_LENGTH]
        .try_into()
        .map_err(|_| eyre::eyre!("failed to extract rt_mr0"))?;
    let rt_mr1: [u8; 48] = report_body[OFFSET_RT_MR1..OFFSET_RT_MR1 + MEASUREMENT_REGISTER_LENGTH]
        .try_into()
        .map_err(|_| eyre::eyre!("failed to extract rt_mr1"))?;
    let rt_mr2: [u8; 48] = report_body[OFFSET_RT_MR2..OFFSET_RT_MR2 + MEASUREMENT_REGISTER_LENGTH]
        .try_into()
        .map_err(|_| eyre::eyre!("failed to extract rt_mr2"))?;
    let rt_mr3: [u8; 48] = report_body[OFFSET_RT_MR3..OFFSET_RT_MR3 + MEASUREMENT_REGISTER_LENGTH]
        .try_into()
        .map_err(|_| eyre::eyre!("failed to extract rt_mr3"))?;
    let mr_config_id: [u8; 48] = report_body
        [OFFSET_MR_CONFIG_ID..OFFSET_MR_CONFIG_ID + MEASUREMENT_REGISTER_LENGTH]
        .try_into()
        .map_err(|_| eyre::eyre!("failed to extract mr_config_id"))?;
    let mr_owner: [u8; 48] = report_body
        [OFFSET_MR_OWNER..OFFSET_MR_OWNER + MEASUREMENT_REGISTER_LENGTH]
        .try_into()
        .map_err(|_| eyre::eyre!("failed to extract mr_owner"))?;
    let mr_owner_config: [u8; 48] = report_body
        [OFFSET_MR_OWNER_CONFIG..OFFSET_MR_OWNER_CONFIG + MEASUREMENT_REGISTER_LENGTH]
        .try_into()
        .map_err(|_| eyre::eyre!("failed to extract mr_owner_config"))?;

    // Extract xFAM and tdAttributes (8 bytes each)
    // In Solidity, bytes8 is treated as big-endian for bitwise operations
    let xfam = u64::from_be_bytes(
        report_body[OFFSET_XFAM..OFFSET_XFAM + ATTRIBUTE_LENGTH]
            .try_into()
            .map_err(|e| eyre::eyre!("failed to parse xfam: {}", e))?,
    );
    let td_attributes = u64::from_be_bytes(
        report_body[OFFSET_TD_ATTRIBUTES..OFFSET_TD_ATTRIBUTES + ATTRIBUTE_LENGTH]
            .try_into()
            .map_err(|e| eyre::eyre!("failed to parse td_attributes: {}", e))?,
    );

    Ok(ParsedQuote {
        mr_td,
        rt_mr0,
        rt_mr1,
        rt_mr2,
        rt_mr3,
        mr_config_id,
        mr_owner,
        mr_owner_config,
        xfam,
        td_attributes,
    })
}

/// Compute workload ID from parsed quote data
/// This corresponds to QuoteParser.parseV4VerifierOutput in Solidity implementation
/// The workload ID uniquely identifies a TEE workload based on its measurement registers
pub fn compute_workload_id_from_parsed(parsed: &ParsedQuote) -> [u8; 32] {
    // Concatenate all fields
    let mut concatenated = Vec::new();
    concatenated.extend_from_slice(&parsed.mr_td);
    concatenated.extend_from_slice(&parsed.rt_mr0);
    concatenated.extend_from_slice(&parsed.rt_mr1);
    concatenated.extend_from_slice(&parsed.rt_mr2);
    concatenated.extend_from_slice(&parsed.rt_mr3);
    concatenated.extend_from_slice(&parsed.mr_config_id);
    concatenated.extend_from_slice(&parsed.xfam.to_be_bytes());
    concatenated.extend_from_slice(&parsed.td_attributes.to_be_bytes());

    // Compute keccak256 hash
    let mut hasher = Keccak256::new();
    hasher.update(&concatenated);
    let result = hasher.finalize();

    let mut workload_id = [0u8; 32];
    workload_id.copy_from_slice(&result);

    workload_id
}

/// Compute workload ID from raw quote bytes
/// This is a convenience function that combines parsing and computation
pub fn compute_workload_id(raw_quote: &[u8]) -> eyre::Result<[u8; 32]> {
    let parsed = parse_report_body(raw_quote)?;
    Ok(compute_workload_id_from_parsed(&parsed))
}

#[cfg(test)]
mod tests {
    use crate::tests::WORKLOAD_ID;

    use super::*;

    #[test]
    fn test_compute_workload_id_from_test_quote() {
        // Load the test quote output used in integration tests
        let quote_output = include_bytes!("../tests/framework/artifacts/test-quote.bin");

        // Compute the workload ID
        let workload_id = compute_workload_id(quote_output)
            .expect("failed to compute workload ID from test quote");

        assert_eq!(
            workload_id, WORKLOAD_ID,
            "workload ID mismatch for test quote"
        );
    }
}
