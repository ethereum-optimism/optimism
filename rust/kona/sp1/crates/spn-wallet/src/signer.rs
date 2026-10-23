//! The key that signs write commands: a local key from `PRIVATE_KEY`, or a GCP Cloud KMS key.
//!
//! KMS signing talks to Cloud KMS directly with Application Default Credentials, without
//! op-signer. One signer covers both the L1 transactions and the SPN EIP-191 messages.

use std::str::FromStr;

use alloy_network::EthereumWallet;
use alloy_primitives::Address;
use alloy_signer::Signer as _;
use alloy_signer_gcp::{
    GcpKeyRingRef, GcpSigner, KeySpecifier,
    gcloud_sdk::{
        GoogleApi,
        google::cloud::kms::v1::key_management_service_client::KeyManagementServiceClient,
    },
};
use alloy_signer_local::PrivateKeySigner;
use anyhow::{Context, Result, anyhow, bail};

pub(crate) const PRIVATE_KEY_ENV: &str = "PRIVATE_KEY";
const KMS_ENDPOINT: &str = "https://cloudkms.googleapis.com";

#[derive(Clone, Debug)]
pub(crate) enum Funder {
    Local(PrivateKeySigner),
    Kms(GcpSigner),
}

impl Funder {
    /// Loads the KMS key when `kms_key` is set, else the `PRIVATE_KEY` env var. Setting both is
    /// an error, so the operator always knows which account signs.
    pub(crate) async fn load(kms_key: Option<&str>) -> Result<Self> {
        let private_key = std::env::var(PRIVATE_KEY_ENV).ok().filter(|k| !k.trim().is_empty());
        match (kms_key, private_key) {
            (Some(_), Some(_)) => {
                bail!("set either {PRIVATE_KEY_ENV} or --kms-key (KMS_KEY), not both")
            }
            (Some(name), None) => Self::kms(name).await,
            (None, Some(key)) => {
                // Do not surface the parse error: it must never risk echoing key material.
                let signer = PrivateKeySigner::from_str(key.trim())
                    .map_err(|_| anyhow!("{PRIVATE_KEY_ENV} is not a valid hex private key"))?;
                Ok(Self::Local(signer))
            }
            (None, None) => {
                bail!("signing commands need {PRIVATE_KEY_ENV} or --kms-key (KMS_KEY)")
            }
        }
    }

    async fn kms(name: &str) -> Result<Self> {
        let key = parse_kms_key(name)?;
        let client = GoogleApi::from_function(KeyManagementServiceClient::new, KMS_ENDPOINT, None)
            .await
            .context(
                "creating the Cloud KMS client (run `gcloud auth application-default login`)",
            )?;
        let signer = GcpSigner::new(client, key, None)
            .await
            .with_context(|| format!("reading the public key of KMS key {name}"))?;
        Ok(Self::Kms(signer))
    }

    pub(crate) fn address(&self) -> Address {
        match self {
            Self::Local(signer) => signer.address(),
            Self::Kms(signer) => signer.address(),
        }
    }

    /// EIP-191 signature over `message`, as SPN expects for transfer and withdraw bodies.
    pub(crate) async fn sign_message(&self, message: &[u8]) -> Result<Vec<u8>> {
        let signature = match self {
            Self::Local(signer) => signer.sign_message(message).await?,
            Self::Kms(signer) => signer.sign_message(message).await?,
        };
        Ok(signature.as_bytes().to_vec())
    }

    pub(crate) fn wallet(&self) -> EthereumWallet {
        match self {
            Self::Local(signer) => EthereumWallet::from(signer.clone()),
            Self::Kms(signer) => EthereumWallet::from(signer.clone()),
        }
    }
}

/// Parses a full key-version resource name, the same string op-signer takes as `key:`:
/// `projects/P/locations/L/keyRings/R/cryptoKeys/K/cryptoKeyVersions/V`.
fn parse_kms_key(name: &str) -> Result<KeySpecifier> {
    let KmsKeyParts { ring, key, version } = parse_kms_parts(name)?;
    Ok(KeySpecifier::new(ring, key, version))
}

/// The pieces of a key-version resource name. `KeySpecifier` keeps them private.
#[derive(Debug)]
struct KmsKeyParts<'a> {
    ring: GcpKeyRingRef,
    key: &'a str,
    version: u64,
}

fn parse_kms_parts(name: &str) -> Result<KmsKeyParts<'_>> {
    let parts: Vec<&str> = name.split('/').collect();
    let [
        "projects",
        project,
        "locations",
        location,
        "keyRings",
        ring,
        "cryptoKeys",
        key,
        "cryptoKeyVersions",
        version,
    ] = parts.as_slice()
    else {
        bail!(
            "KMS key {name:?} is not of the form \
             projects/P/locations/L/keyRings/R/cryptoKeys/K/cryptoKeyVersions/V"
        );
    };
    let version = version
        .parse()
        .with_context(|| format!("KMS key {name:?} has a non-numeric version {version:?}"))?;
    Ok(KmsKeyParts { ring: GcpKeyRingRef::new(project, location, ring), key, version })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_the_op_signer_key_resource_name() {
        let parts = parse_kms_parts(
            "projects/oplabs-dev-ent-networks-us/locations/us-east1/keyRings/zk-proofs-2-0/cryptoKeys/cost-recipient/cryptoKeyVersions/1",
        )
        .unwrap();
        assert_eq!(parts.ring.google_project_id, "oplabs-dev-ent-networks-us");
        assert_eq!(parts.ring.location, "us-east1");
        assert_eq!(parts.ring.name, "zk-proofs-2-0");
        assert_eq!(parts.key, "cost-recipient");
        assert_eq!(parts.version, 1);
    }

    #[test]
    fn rejects_a_key_without_a_version() {
        assert!(parse_kms_parts("projects/p/locations/l/keyRings/r/cryptoKeys/k",).is_err());
        assert!(
            parse_kms_parts(
                "projects/p/locations/l/keyRings/r/cryptoKeys/k/cryptoKeyVersions/latest",
            )
            .is_err()
        );
    }
}
