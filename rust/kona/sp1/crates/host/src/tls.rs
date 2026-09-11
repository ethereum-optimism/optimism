//! Utilities for configuring mutual TLS clients.

use std::{env, path::PathBuf, sync::Arc};

use alloy_transport_http::reqwest;
use anyhow::{Context, Result, bail, ensure};
use rustls::{
    ClientConfig, RootCertStore,
    pki_types::{CertificateDer, PrivateKeyDer, pem::PemObject},
};

use crate::prefixed_env_var;

/// PEM paths for an mTLS client identity and the CA that signs the server.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ClientTls {
    /// Path to the server CA certificate.
    pub ca: PathBuf,
    /// Path to the client certificate.
    pub cert: PathBuf,
    /// Path to the client private key.
    pub key: PathBuf,
}

impl ClientTls {
    /// Reads `<PREFIX>_<GROUP>_TLS_CA`, `_CERT`, and `_KEY`.
    ///
    /// Returns `Ok(None)` when all three are absent or empty. Setting only some of the variables is
    /// an error.
    pub fn from_env(prefix: &str, group: &str) -> Result<Option<Self>> {
        let ca_name = prefixed_env_var(prefix, &format!("{group}_TLS_CA"));
        let cert_name = prefixed_env_var(prefix, &format!("{group}_TLS_CERT"));
        let key_name = prefixed_env_var(prefix, &format!("{group}_TLS_KEY"));

        let path_from_env = |name: &str| {
            env::var(name).ok().filter(|value| !value.trim().is_empty()).map(PathBuf::from)
        };

        let (ca, cert, key) =
            (path_from_env(&ca_name), path_from_env(&cert_name), path_from_env(&key_name));

        match (ca, cert, key) {
            (None, None, None) => Ok(None),
            (Some(ca), Some(cert), Some(key)) => Ok(Some(Self { ca, cert, key })),
            _ => bail!("{ca_name}, {cert_name}, and {key_name} must all be set together"),
        }
    }

    /// Builds a rustls client configuration from the configured PEM files.
    pub fn client_config(&self) -> Result<ClientConfig> {
        let mut root_store = RootCertStore::empty();
        let ca_certs = CertificateDer::pem_file_iter(&self.ca)
            .with_context(|| format!("failed to read CA certificate from {}", self.ca.display()))?
            .collect::<Result<Vec<_>, _>>()
            .with_context(|| {
                format!("failed to parse CA certificate from {}", self.ca.display())
            })?;
        ensure!(!ca_certs.is_empty(), "no CA certificates found in {}", self.ca.display());
        for cert in ca_certs {
            root_store.add(cert).with_context(|| {
                format!("failed to add CA certificate from {}", self.ca.display())
            })?;
        }

        let certs = CertificateDer::pem_file_iter(&self.cert)
            .with_context(|| {
                format!("failed to read client certificate from {}", self.cert.display())
            })?
            .collect::<Result<Vec<_>, _>>()
            .with_context(|| {
                format!("failed to parse client certificate from {}", self.cert.display())
            })?;
        ensure!(!certs.is_empty(), "no client certificates found in {}", self.cert.display());
        let key = PrivateKeyDer::from_pem_file(&self.key)
            .with_context(|| format!("failed to read private key from {}", self.key.display()))?;

        ClientConfig::builder_with_provider(Arc::new(rustls::crypto::aws_lc_rs::default_provider()))
            .with_safe_default_protocol_versions()
            .context("failed to configure TLS protocol versions")?
            .with_root_certificates(root_store)
            .with_client_auth_cert(certs, key)
            .context("failed to configure client certificate")
    }

    /// Builds a reqwest client configured for mutual TLS.
    pub fn http_client(&self) -> Result<reqwest::Client> {
        reqwest::Client::builder()
            .tls_backend_preconfigured(self.client_config()?)
            .build()
            .context("failed to build mTLS HTTP client")
    }
}

#[cfg(test)]
mod tests {
    use std::path::Path;

    use super::*;

    fn fixture(name: &str) -> PathBuf {
        Path::new(env!("CARGO_MANIFEST_DIR")).join("testdata/mtls").join(name)
    }

    fn set_env(name: &str, value: &Path) {
        // SAFETY: Each test uses a unique environment-variable prefix, so concurrent tests do not
        // read or modify the same variables.
        unsafe { env::set_var(name, value) };
    }

    #[test]
    fn from_env_returns_material_when_all_variables_are_set() {
        let prefix = "KONA_SP1_HOST_TLS_TEST_ALL";
        let ca_name = prefixed_env_var(prefix, "SIGNER_TLS_CA");
        let cert_name = prefixed_env_var(prefix, "SIGNER_TLS_CERT");
        let key_name = prefixed_env_var(prefix, "SIGNER_TLS_KEY");
        set_env(&ca_name, &fixture("ca.crt"));
        set_env(&cert_name, &fixture("client.crt"));
        set_env(&key_name, &fixture("client.key"));

        assert_eq!(
            ClientTls::from_env(prefix, "SIGNER").unwrap(),
            Some(ClientTls {
                ca: fixture("ca.crt"),
                cert: fixture("client.crt"),
                key: fixture("client.key"),
            })
        );
    }

    #[test]
    fn from_env_returns_none_when_all_variables_are_absent() {
        assert_eq!(ClientTls::from_env("KONA_SP1_HOST_TLS_TEST_NONE", "SIGNER").unwrap(), None);
    }

    #[test]
    fn from_env_rejects_partial_configuration() {
        for (case, variables) in
            [("ONE", &["SIGNER_TLS_CA"][..]), ("TWO", &["SIGNER_TLS_CA", "SIGNER_TLS_CERT"][..])]
        {
            let prefix = format!("KONA_SP1_HOST_TLS_TEST_{case}");
            for suffix in variables {
                set_env(&prefixed_env_var(&prefix, suffix), &fixture("ca.crt"));
            }

            let error = ClientTls::from_env(&prefix, "SIGNER").unwrap_err().to_string();
            for suffix in ["SIGNER_TLS_CA", "SIGNER_TLS_CERT", "SIGNER_TLS_KEY"] {
                assert!(error.contains(&prefixed_env_var(&prefix, suffix)), "{error}");
            }
        }
    }

    #[test]
    fn client_config_accepts_valid_pem_material() {
        let tls = ClientTls {
            ca: fixture("ca.crt"),
            cert: fixture("client.crt"),
            key: fixture("client.key"),
        };

        tls.client_config().unwrap();
    }

    #[test]
    fn client_config_rejects_non_pem_material() {
        let tls = ClientTls {
            ca: Path::new(env!("CARGO_MANIFEST_DIR")).join("Cargo.toml"),
            cert: fixture("client.crt"),
            key: fixture("client.key"),
        };

        assert!(tls.client_config().is_err());
    }
}
