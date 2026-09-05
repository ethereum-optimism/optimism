//! Published super-range artifact loading and identity derivation.

use std::{future::Future, io::Read, sync::Arc, time::Duration};

use alloy_primitives::B256;
use anyhow::{Context, Result, anyhow, bail, ensure};
use sha2::{Digest, Sha256};
use sp1_sdk::{Elf, HashableKey, LightProver, Prover, ProvingKey};
use url::Url;

/// Immutable identity of the SP1 program used by an attempt.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct ArtifactIdentity {
    /// Derived verification-key hash of the super-range ELF.
    pub range_vkey: B256,
    /// SHA-256 digest of the decompressed super-range ELF.
    pub elf_sha256: B256,
}

/// Location and deadline for loading one published range artifact.
#[derive(Clone, Debug)]
pub struct ArtifactConfig {
    /// Direct URL of the gzip-compressed range ELF.
    pub url: Url,
    /// Whole-request deadline.
    pub fetch_timeout: Duration,
    /// Whether diagnostic `file://` loading is allowed.
    pub allow_file: bool,
}

/// Decompressed ELF bytes with a successfully derived SP1 identity.
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
    /// Returns the loaded program bytes.
    pub fn bytes(&self) -> Arc<[u8]> {
        self.bytes.clone()
    }

    /// Returns the derived artifact identity.
    pub const fn identity(&self) -> ArtifactIdentity {
        self.identity
    }
}

/// Loads the configured range artifact and derives its identity.
pub async fn load_range_artifact(config: &ArtifactConfig) -> Result<ValidatedRangeArtifact> {
    load_range_artifact_with(config, None, derive_range_vkey).await
}

/// Refetches the configured artifact without repeating SP1 setup for unchanged bytes.
pub async fn refresh_range_artifact(
    config: &ArtifactConfig,
    current: &ValidatedRangeArtifact,
) -> Result<ValidatedRangeArtifact> {
    load_range_artifact_with(config, Some(current), derive_range_vkey).await
}

async fn load_range_artifact_with<F, Fut>(
    config: &ArtifactConfig,
    current: Option<&ValidatedRangeArtifact>,
    derive_vkey: F,
) -> Result<ValidatedRangeArtifact>
where
    F: FnOnce(Arc<[u8]>) -> Fut,
    Fut: Future<Output = Result<B256>>,
{
    validate_artifact_url(&config.url, config.allow_file)?;
    ensure!(!config.fetch_timeout.is_zero(), "artifact fetch timeout must be non-zero");
    let compressed = fetch(&config.url, config.fetch_timeout, config.allow_file).await?;
    let bytes = decompress(&compressed)
        .with_context(|| format!("failed to decode artifact from {}", redacted_url(&config.url)))?;
    derive_identity_with(bytes, current, derive_vkey).await
}

async fn derive_identity_with<F, Fut>(
    bytes: Vec<u8>,
    current: Option<&ValidatedRangeArtifact>,
    derive_vkey: F,
) -> Result<ValidatedRangeArtifact>
where
    F: FnOnce(Arc<[u8]>) -> Fut,
    Fut: Future<Output = Result<B256>>,
{
    let bytes = Arc::<[u8]>::from(bytes);
    let digest_bytes: [u8; 32] = Sha256::digest(&bytes).into();
    let elf_sha256 = B256::from(digest_bytes);
    if let Some(current) = current &&
        current.identity.elf_sha256 == elf_sha256
    {
        return Ok(current.clone());
    }
    let identity = ArtifactIdentity { range_vkey: derive_vkey(bytes.clone()).await?, elf_sha256 };
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

async fn fetch(url: &Url, timeout: Duration, allow_file: bool) -> Result<Vec<u8>> {
    match url.scheme() {
        "file" if allow_file => {
            let path = url
                .to_file_path()
                .map_err(|()| anyhow!("invalid file artifact URL {}", redacted_url(url)))?;
            tokio::fs::read(&path)
                .await
                .with_context(|| format!("failed to read artifact at {}", redacted_url(url)))
        }
        "https" => {
            let client = reqwest::Client::builder()
                .timeout(timeout)
                .redirect(reqwest::redirect::Policy::none())
                .build()
                .context("failed to build artifact HTTP client")?;
            client
                .get(url.clone())
                .send()
                .await
                .with_context(|| format!("failed to fetch artifact from {}", redacted_url(url)))?
                .error_for_status()
                .with_context(|| format!("artifact fetch failed for {}", redacted_url(url)))?
                .bytes()
                .await
                .with_context(|| format!("failed to read artifact body from {}", redacted_url(url)))
                .map(|bytes| bytes.to_vec())
        }
        _ => bail!("unsupported artifact source {}", redacted_url(url)),
    }
}

fn decompress(compressed: &[u8]) -> Result<Vec<u8>> {
    let mut decoder = flate2::read::GzDecoder::new(compressed);
    let mut bytes = Vec::new();
    decoder.read_to_end(&mut bytes).context("artifact is not valid gzip")?;
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
        let path = directory.path().join("artifact.bin.gz");
        let bytes = b"synthetic SP1 ELF identity fixture".to_vec();
        let config = ArtifactConfig {
            url: Url::from_file_path(&path).unwrap(),
            fetch_timeout: Duration::from_secs(1),
            allow_file: true,
        };
        let mut encoder = GzEncoder::new(Vec::new(), Compression::default());
        encoder.write_all(&bytes).unwrap();
        std::fs::write(path, encoder.finish().unwrap()).unwrap();
        (directory, config, bytes)
    }

    #[tokio::test]
    async fn load_range_artifact_derives_identity() {
        let (_directory, config, expected_bytes) = fixture();
        let expected_vkey = B256::repeat_byte(0x22);
        let artifact =
            load_range_artifact_with(&config, None, move |_| async move { Ok(expected_vkey) })
                .await
                .unwrap();
        let expected_digest: [u8; 32] = Sha256::digest(&expected_bytes).into();
        assert_eq!(&*artifact.bytes(), expected_bytes);
        assert_eq!(artifact.identity().range_vkey, expected_vkey);
        assert_eq!(artifact.identity().elf_sha256, B256::from(expected_digest));
    }

    #[tokio::test]
    async fn load_range_artifact_rejects_invalid_or_empty_gzip() {
        let (_directory, config, _) = fixture();
        std::fs::write(config.url.to_file_path().unwrap(), b"not gzip").unwrap();
        let error = load_range_artifact_with(&config, None, |_| async { Ok(B256::ZERO) })
            .await
            .unwrap_err();
        assert!(format!("{error:#}").contains("valid gzip"));

        let mut encoder = GzEncoder::new(Vec::new(), Compression::default());
        encoder.write_all(&[]).unwrap();
        std::fs::write(config.url.to_file_path().unwrap(), encoder.finish().unwrap()).unwrap();
        let error = load_range_artifact_with(&config, None, |_| async { Ok(B256::ZERO) })
            .await
            .unwrap_err();
        assert!(format!("{error:#}").contains("empty"));
    }

    #[tokio::test]
    async fn refresh_skips_vkey_setup_for_unchanged_bytes() {
        let (_directory, config, _) = fixture();
        let current =
            load_range_artifact_with(&config, None, |_| async { Ok(B256::repeat_byte(0x22)) })
                .await
                .unwrap();

        let refreshed = load_range_artifact_with(&config, Some(&current), |_| async {
            panic!("unchanged bytes must not repeat SP1 setup")
        })
        .await
        .unwrap();

        assert_eq!(refreshed.identity(), current.identity());
    }

    #[tokio::test]
    async fn refresh_loads_changed_bytes_and_derives_a_new_identity() {
        let (_directory, config, _) = fixture();
        let current =
            load_range_artifact_with(&config, None, |_| async { Ok(B256::repeat_byte(0x22)) })
                .await
                .unwrap();
        let replacement = b"replacement SP1 ELF";
        let mut encoder = GzEncoder::new(Vec::new(), Compression::default());
        encoder.write_all(replacement).unwrap();
        std::fs::write(config.url.to_file_path().unwrap(), encoder.finish().unwrap()).unwrap();

        let refreshed = load_range_artifact_with(&config, Some(&current), |_| async {
            Ok(B256::repeat_byte(0x44))
        })
        .await
        .unwrap();

        assert_eq!(&*refreshed.bytes(), replacement);
        assert_eq!(refreshed.identity().range_vkey, B256::repeat_byte(0x44));
        assert_ne!(refreshed.identity().elf_sha256, current.identity().elf_sha256);
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
