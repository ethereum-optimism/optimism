//! Utilities for configuring mutual TLS clients that reload their material when it changes on disk.

use std::{env, path::PathBuf};

use alloy_transport_http::reqwest::{Url, header::HeaderMap};
use anyhow::{Context, Result, bail};
use kona_sources::{ClientCert, ReloadingRpcClient, TlsPaths};

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

    /// Builds a JSON-RPC client over mutual TLS; see [`ReloadingRpcClient`] for reloading.
    ///
    /// Only the configured CA is trusted to verify the server.
    pub fn rpc_client(&self, endpoint: Url) -> Result<ReloadingRpcClient> {
        ReloadingRpcClient::new(
            endpoint,
            TlsPaths {
                ca_cert: Some(self.ca.clone()),
                client_cert: Some(ClientCert { cert: self.cert.clone(), key: self.key.clone() }),
            },
            HeaderMap::new(),
        )
        .context("failed to build mTLS RPC client")
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
    fn rpc_client_accepts_valid_pem_material() {
        let tls = ClientTls {
            ca: fixture("ca.crt"),
            cert: fixture("client.crt"),
            key: fixture("client.key"),
        };

        tls.rpc_client("https://localhost:1".parse().unwrap()).unwrap();
    }

    #[test]
    fn rpc_client_rejects_non_pem_material() {
        let tls = ClientTls {
            ca: Path::new(env!("CARGO_MANIFEST_DIR")).join("Cargo.toml"),
            cert: fixture("client.crt"),
            key: fixture("client.key"),
        };

        assert!(tls.rpc_client("https://localhost:1".parse().unwrap()).is_err());
    }
}
