//! Published super-range artifact loading and identity validation.

use std::{future::Future, io::Read, sync::Arc, time::Duration};

use alloy_primitives::B256;
use anyhow::{Context, Result, anyhow, bail, ensure};
use sha2::{Digest, Sha256};
use sp1_sdk::{Elf, HashableKey, LightProver, Prover, ProvingKey};
use url::Url;

const RANGE_ARTIFACT_SUFFIX: &str = ".range.bin.gz";

/// Immutable identity of the SP1 program used by an attempt.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct ArtifactIdentity {
    /// Aggregation prestate used as the published artifact key.
    pub prestate: B256,
    /// Expected verification-key hash of the super-range ELF.
    pub range_vkey: B256,
    /// SHA-256 digest of the decompressed super-range ELF.
    pub elf_sha256: B256,
}

/// Limits and expected identity for loading one published range artifact.
#[derive(Clone, Debug)]
pub struct ArtifactConfig {
    /// Immutable artifact-directory URL.
    pub base_url: Url,
    /// Expected release identity.
    pub identity: ArtifactIdentity,
    /// Maximum compressed response size.
    pub max_compressed_bytes: usize,
    /// Maximum decompressed ELF size.
    pub max_decompressed_bytes: usize,
    /// Whole-request deadline.
    pub fetch_timeout: Duration,
    /// Whether diagnostic `file://` loading is allowed.
    pub allow_file: bool,
}

/// Decompressed ELF bytes whose digest and SP1 vkey match the configured release.
#[derive(Clone)]
pub struct ValidatedRangeArtifact {
    bytes: Arc<[u8]>,
    identity: ArtifactIdentity,
}

impl std::fmt::Debug for ValidatedRangeArtifact {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("ValidatedRangeArtifact")
            .field("identity", &self.identity)
            .field("byte_len", &self.bytes.len())
            .finish()
    }
}

impl ValidatedRangeArtifact {
    /// Returns the authenticated program bytes.
    pub fn bytes(&self) -> Arc<[u8]> {
        self.bytes.clone()
    }

    /// Returns the authenticated release identity.
    pub const fn identity(&self) -> ArtifactIdentity {
        self.identity
    }
}

/// Loads and authenticates the configured published range artifact.
pub async fn load_range_artifact(config: &ArtifactConfig) -> Result<ValidatedRangeArtifact> {
    load_range_artifact_with(config, derive_range_vkey).await
}

async fn load_range_artifact_with<F, Fut>(
    config: &ArtifactConfig,
    derive_vkey: F,
) -> Result<ValidatedRangeArtifact>
where
    F: FnOnce(Arc<[u8]>) -> Fut,
    Fut: Future<Output = Result<B256>>,
{
    validate_artifact_url(&config.base_url, config.allow_file)?;
    ensure!(config.max_compressed_bytes > 0, "compressed artifact limit must be non-zero");
    ensure!(config.max_decompressed_bytes > 0, "decompressed artifact limit must be non-zero");
    ensure!(!config.fetch_timeout.is_zero(), "artifact fetch timeout must be non-zero");
    let url = range_artifact_url(&config.base_url, config.identity.prestate)?;
    let compressed =
        fetch_bounded(&url, config.max_compressed_bytes, config.fetch_timeout, config.allow_file)
            .await?;
    let bytes = decompress_bounded(&compressed, config.max_decompressed_bytes)
        .with_context(|| format!("failed to decode artifact from {}", redacted_url(&url)))?;
    validate_bytes_with(config.identity, bytes, derive_vkey).await
}

async fn validate_bytes_with<F, Fut>(
    identity: ArtifactIdentity,
    bytes: Vec<u8>,
    derive_vkey: F,
) -> Result<ValidatedRangeArtifact>
where
    F: FnOnce(Arc<[u8]>) -> Fut,
    Fut: Future<Output = Result<B256>>,
{
    let bytes = Arc::<[u8]>::from(bytes);
    let digest_bytes: [u8; 32] = Sha256::digest(&bytes).into();
    let actual_digest = B256::from(digest_bytes);
    ensure!(
        actual_digest == identity.elf_sha256,
        "range ELF SHA-256 mismatch: expected {}, got {actual_digest}",
        identity.elf_sha256,
    );
    let actual_vkey = derive_vkey(bytes.clone()).await?;
    ensure!(
        actual_vkey == identity.range_vkey,
        "range ELF vkey mismatch: expected {}, got {actual_vkey}",
        identity.range_vkey,
    );
    Ok(ValidatedRangeArtifact { bytes, identity })
}

async fn derive_range_vkey(bytes: Arc<[u8]>) -> Result<B256> {
    let prover = LightProver::new().await;
    let key = prover
        .setup(Elf::Dynamic(bytes))
        .await
        .context("range ELF setup failed while deriving vkey")?;
    Ok(B256::from(key.verifying_key().bytes32_raw()))
}

fn range_artifact_url(base: &Url, prestate: B256) -> Result<Url> {
    let mut url = base.clone();
    url.path_segments_mut()
        .map_err(|()| anyhow!("artifact URL cannot be a base: {}", redacted_url(base)))?
        .pop_if_empty()
        .push(&format!("{prestate}{RANGE_ARTIFACT_SUFFIX}"));
    Ok(url)
}

fn validate_artifact_url(url: &Url, allow_file: bool) -> Result<()> {
    match url.scheme() {
        "https" => {
            ensure!(
                url.username().is_empty() && url.password().is_none(),
                "artifact URL must not contain credentials"
            );
            ensure!(url.query().is_none(), "artifact URL must not contain a query");
        }
        "file" if allow_file => {}
        "file" => bail!("file artifact URLs are allowed only with --once"),
        scheme => bail!("unsupported artifact URL scheme {scheme}; expected https"),
    }
    Ok(())
}

async fn fetch_bounded(
    url: &Url,
    max_bytes: usize,
    timeout: Duration,
    allow_file: bool,
) -> Result<Vec<u8>> {
    match url.scheme() {
        "file" if allow_file => {
            let path = url
                .to_file_path()
                .map_err(|()| anyhow!("invalid file artifact URL {}", redacted_url(url)))?;
            let metadata = tokio::fs::metadata(&path)
                .await
                .with_context(|| format!("failed to stat artifact at {}", redacted_url(url)))?;
            ensure!(metadata.len() <= max_bytes as u64, "compressed artifact exceeds size limit");
            let bytes = tokio::fs::read(&path)
                .await
                .with_context(|| format!("failed to read artifact at {}", redacted_url(url)))?;
            ensure!(bytes.len() <= max_bytes, "compressed artifact exceeds size limit");
            Ok(bytes)
        }
        "https" => {
            let client = reqwest::Client::builder()
                .timeout(timeout)
                .redirect(reqwest::redirect::Policy::none())
                .build()
                .context("failed to build artifact HTTP client")?;
            let mut response = client
                .get(url.clone())
                .send()
                .await
                .with_context(|| format!("failed to fetch artifact from {}", redacted_url(url)))?
                .error_for_status()
                .with_context(|| format!("artifact fetch failed for {}", redacted_url(url)))?;
            if let Some(length) = response.content_length() {
                ensure!(length <= max_bytes as u64, "compressed artifact exceeds size limit");
            }
            let mut bytes = Vec::new();
            while let Some(chunk) = response.chunk().await.with_context(|| {
                format!("failed to read artifact body from {}", redacted_url(url))
            })? {
                ensure!(
                    bytes.len().saturating_add(chunk.len()) <= max_bytes,
                    "compressed artifact exceeds size limit",
                );
                bytes.extend_from_slice(&chunk);
            }
            Ok(bytes)
        }
        _ => bail!("unsupported artifact source {}", redacted_url(url)),
    }
}

fn decompress_bounded(compressed: &[u8], max_bytes: usize) -> Result<Vec<u8>> {
    let limit = u64::try_from(max_bytes).context("decompressed artifact limit is too large")?;
    let mut decoder = flate2::read::GzDecoder::new(compressed).take(limit.saturating_add(1));
    let mut bytes = Vec::new();
    decoder.read_to_end(&mut bytes).context("artifact is not valid gzip")?;
    ensure!(bytes.len() <= max_bytes, "decompressed artifact exceeds size limit");
    ensure!(!bytes.is_empty(), "decompressed artifact is empty");
    Ok(bytes)
}

/// Renders a URL without credentials, query values, or fragments.
pub fn redacted_url(url: &Url) -> String {
    let mut redacted = url.clone();
    let _ = redacted.set_username("");
    let _ = redacted.set_password(None);
    redacted.set_query(None);
    redacted.set_fragment(None);
    redacted.to_string()
}

#[cfg(test)]
mod tests {
    use std::io::Write;

    use super::*;
    use flate2::{Compression, write::GzEncoder};
    use sha2::Sha256;
    use tempfile::TempDir;

    fn fixture() -> (TempDir, ArtifactConfig, Vec<u8>) {
        let directory = tempfile::tempdir().unwrap();
        let base_url = Url::from_directory_path(directory.path()).unwrap();
        let bytes = b"synthetic SP1 ELF identity fixture".to_vec();
        let prestate = B256::repeat_byte(0x11);
        let range_vkey = B256::repeat_byte(0x22);
        let digest_bytes: [u8; 32] = Sha256::digest(&bytes).into();
        let elf_sha256 = B256::from(digest_bytes);
        let config = ArtifactConfig {
            base_url,
            identity: ArtifactIdentity { prestate, range_vkey, elf_sha256 },
            max_compressed_bytes: 1024,
            max_decompressed_bytes: 1024,
            fetch_timeout: Duration::from_secs(1),
            allow_file: true,
        };
        let path = range_artifact_url(&config.base_url, prestate).unwrap().to_file_path().unwrap();
        let expected_name = format!("{prestate}{RANGE_ARTIFACT_SUFFIX}");
        assert_eq!(path.file_name().unwrap().to_str(), Some(expected_name.as_str()));
        let mut encoder = GzEncoder::new(Vec::new(), Compression::default());
        encoder.write_all(&bytes).unwrap();
        std::fs::write(path, encoder.finish().unwrap()).unwrap();
        (directory, config, bytes)
    }

    #[tokio::test]
    async fn load_range_artifact_accepts_matching_identity() {
        let (_directory, config, expected_bytes) = fixture();
        let expected_vkey = config.identity.range_vkey;
        let artifact = load_range_artifact_with(&config, move |_| async move { Ok(expected_vkey) })
            .await
            .unwrap();
        assert_eq!(&*artifact.bytes(), expected_bytes);
        assert_eq!(artifact.identity(), config.identity);
    }

    #[tokio::test]
    async fn load_range_artifact_rejects_vkey_mismatch() {
        let (_directory, config, _) = fixture();
        let error = load_range_artifact_with(&config, |_| async { Ok(B256::repeat_byte(0xff)) })
            .await
            .unwrap_err();
        assert!(error.to_string().contains("vkey mismatch"));
    }

    #[tokio::test]
    async fn load_range_artifact_rejects_digest_mismatch_or_oversize() {
        let (_directory, mut config, _) = fixture();
        config.identity.elf_sha256 = B256::ZERO;
        let error =
            load_range_artifact_with(&config, |_| async { Ok(B256::ZERO) }).await.unwrap_err();
        assert!(error.to_string().contains("SHA-256 mismatch"));

        let (_directory, mut config, _) = fixture();
        config.max_decompressed_bytes = 4;
        let error =
            load_range_artifact_with(&config, |_| async { Ok(B256::ZERO) }).await.unwrap_err();
        assert!(format!("{error:#}").contains("size limit"));

        let (_directory, mut config, _) = fixture();
        config.max_compressed_bytes = 1;
        let error =
            load_range_artifact_with(&config, |_| async { Ok(B256::ZERO) }).await.unwrap_err();
        assert!(format!("{error:#}").contains("size limit"));
    }

    #[test]
    fn artifact_errors_redact_credentials() {
        let url =
            Url::parse("https://user:secret@example.com/prestates?token=hidden#fragment").unwrap();
        let rendered = redacted_url(&url);
        assert_eq!(rendered, "https://example.com/prestates");
        assert!(!rendered.contains("secret"));
        assert!(!rendered.contains("hidden"));
    }
}
