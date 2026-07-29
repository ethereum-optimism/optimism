use std::{fs::File, io::Read, sync::Arc};
use tdx::{Tdx, device::DeviceOptions, error::TdxError};
use thiserror::Error;
use tracing::info;

#[derive(Error, Debug)]
pub enum AttestationError {
    #[error("Failed to get attestation: {0}")]
    GetAttestationFailed(TdxError),
    #[error("Failed to read mock attestation file: {0}")]
    ReadMockAttestationFailed(std::io::Error),
}

/// Configuration for attestation
#[derive(Default)]
pub struct AttestationConfig {
    /// If true, uses the mock attestation provider instead of real TDX hardware
    pub mock: bool,
    /// Path to the mock attestation file
    pub mock_attestation_path: String,
}

/// Trait for attestation providers
pub trait AttestationProvider {
    fn get_attestation(&self, report_data: [u8; 64]) -> Result<Vec<u8>, AttestationError>;
}

/// Real TDX hardware attestation provider
pub struct TdxAttestationProvider {
    tdx: Tdx,
}

impl Default for TdxAttestationProvider {
    fn default() -> Self {
        Self::new()
    }
}

impl TdxAttestationProvider {
    pub fn new() -> Self {
        Self { tdx: Tdx::new() }
    }
}

impl AttestationProvider for TdxAttestationProvider {
    fn get_attestation(&self, report_data: [u8; 64]) -> Result<Vec<u8>, AttestationError> {
        self.tdx
            .get_attestation_report_raw_with_options(DeviceOptions {
                report_data: Some(report_data),
            })
            .map(|(quote, _var_data)| quote)
            .map_err(AttestationError::GetAttestationFailed)
    }
}

/// Mock attestation provider
pub struct MockAttestationProvider {
    mock_attestation_path: String,
}

impl MockAttestationProvider {
    pub fn new(mock_attestation_path: String) -> Self {
        Self {
            mock_attestation_path,
        }
    }
}

impl AttestationProvider for MockAttestationProvider {
    fn get_attestation(&self, _report_data: [u8; 64]) -> Result<Vec<u8>, AttestationError> {
        info!(
            target: "tdx_quote_provider",
            mock_attestation_path = self.mock_attestation_path,
            "using mock attestation provider",
        );
        let mut file = File::open(self.mock_attestation_path.clone())
            .map_err(AttestationError::ReadMockAttestationFailed)?;
        let mut buffer = Vec::new();
        file.read_to_end(&mut buffer)
            .map_err(AttestationError::ReadMockAttestationFailed)?;
        Ok(buffer)
    }
}

pub fn get_attestation_provider(
    config: AttestationConfig,
) -> Arc<dyn AttestationProvider + Send + Sync> {
    if config.mock {
        Arc::new(MockAttestationProvider::new(config.mock_attestation_path))
    } else {
        Arc::new(TdxAttestationProvider::new())
    }
}
