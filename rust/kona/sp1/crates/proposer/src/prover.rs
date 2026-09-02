//! SP1 proof providers for challenged games.
//!
//! The mock pipeline computes outputs natively and submits placeholder bytes to deployments
//! whose verifier accepts arbitrary proofs. Network proving resolves [`ProofKeys`] per game
//! because games may use different prestates.

mod spn;

pub use spn::NetworkProofProvider;

use std::sync::Arc;

use alloy_primitives::B256;
use anyhow::{Context, Result, bail};
use sp1_sdk::{
    Elf, LightProver, Prover, ProvingKey, SP1ProofWithPublicValues, SP1ProvingKey, SP1Stdin,
    SP1VerifyingKey,
};
use thiserror::Error;

use crate::config::PrestatePrograms;

/// Placeholder on-chain proof bytes submitted by the mock provider. Only a
/// deployment whose game verifier accepts arbitrary bytes (devstack's
/// `ZKMockVerifier`) can resolve games proven with these.
pub const MOCK_PROOF_BYTES: &[u8] = b"kona-sp1-mock-super-aggregation-proof";

/// Identifier returned by the SP1 prover network for a proof request.
pub type ProofId = B256;

/// A terminal SP1 request outcome that permits no further polling.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ProofTerminalState {
    /// Execution cannot start for this request.
    Unexecutable,
    /// Execution completed but its proof failed validation.
    ValidationFailed,
    /// No prover can fulfill this request.
    Unfulfillable,
    /// Settlement reverted without recording a proof.
    Reverted,
    /// The request expired without a proof.
    Expired,
    /// Mainnet request details confirm cancellation.
    Cancelled,
}

/// Why polling an existing SP1 request stopped.
#[derive(Debug, Error)]
pub enum ProofWaitError {
    #[error(
        "proof request {proof_id} reached terminal state {state:?} \
         (fulfillment_status={fulfillment_status}, execution_status={execution_status})"
    )]
    /// SPN confirmed that the request cannot produce a proof.
    Terminal {
        /// Request whose polling ended.
        proof_id: ProofId,
        /// Classified terminal outcome.
        state: ProofTerminalState,
        /// Raw SPN fulfillment status for diagnostics.
        fulfillment_status: i32,
        /// Raw SPN execution status for diagnostics.
        execution_status: i32,
    },
    /// SPN may still be processing the request.
    #[error(transparent)]
    Uncertain(#[from] anyhow::Error),
}

/// Container for the proving and verifying keys of one prestate's programs.
#[derive(Clone)]
pub struct ProofKeys {
    /// Proving key for the super-range program.
    pub range_pk: Arc<SP1ProvingKey>,
    /// Verifying key for the super-range program.
    pub range_vk: Arc<SP1VerifyingKey>,
    /// Proving key for the super-aggregation program.
    pub agg_pk: Arc<SP1ProvingKey>,
    /// Verifying key for the super-aggregation program. Its hash is the
    /// on-chain `absolutePrestate()`.
    pub agg_vk: Arc<SP1VerifyingKey>,
}

impl std::fmt::Debug for ProofKeys {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("ProofKeys").finish_non_exhaustive()
    }
}

/// Runs lightweight SP1 key setup for both of a prestate's programs.
///
/// Network proving only needs each ELF and its verifying key. Using the light
/// prover avoids initializing the local CPU proving machinery and its proving
/// keys, which are not used by network proof requests.
pub async fn setup_proof_keys(programs: &PrestatePrograms) -> Result<ProofKeys> {
    let prover = LightProver::new().await;
    let range_pk = prover
        .setup(Elf::Dynamic(programs.range_elf.clone().into()))
        .await
        .context("range ELF setup failed")?;
    let range_vk = range_pk.verifying_key().clone();
    let agg_pk = prover
        .setup(Elf::Dynamic(programs.aggregation_elf.clone().into()))
        .await
        .context("aggregation ELF setup failed")?;
    let agg_vk = agg_pk.verifying_key().clone();
    Ok(ProofKeys {
        range_pk: Arc::new(range_pk),
        range_vk: Arc::new(range_vk),
        agg_pk: Arc::new(agg_pk),
        agg_vk: Arc::new(agg_vk),
    })
}

/// Proof provider abstraction for generating range and aggregation proofs.
#[derive(Clone)]
pub enum ProofProvider {
    /// Real SP1 proving via the Succinct Prover Network.
    Network(NetworkProofProvider),
    /// Native-core pipeline with placeholder on-chain bytes; produces no SP1
    /// proofs at all.
    Mock(MockProofProvider),
}

impl std::fmt::Debug for ProofProvider {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Network(provider) => f.debug_tuple("Network").field(provider).finish(),
            Self::Mock(provider) => f.debug_tuple("Mock").field(provider).finish(),
        }
    }
}

impl ProofProvider {
    /// Submits a compressed super-range or consolidation proof request.
    pub async fn request_range_proof(&self, keys: &ProofKeys, stdin: SP1Stdin) -> Result<ProofId> {
        match self {
            Self::Network(provider) => provider.request_range_proof(keys, stdin).await,
            Self::Mock(_) => bail!("mock provider produces no SP1 range proofs"),
        }
    }

    /// Submits an on-chain aggregation proof request.
    pub async fn request_aggregation_proof(
        &self,
        keys: &ProofKeys,
        stdin: SP1Stdin,
    ) -> Result<ProofId> {
        match self {
            Self::Network(provider) => provider.request_aggregation_proof(keys, stdin).await,
            Self::Mock(_) => bail!("mock provider produces no SP1 aggregation proofs"),
        }
    }

    /// Polls an existing proof request until it finishes.
    pub async fn wait_for_proof(
        &self,
        proof_id: ProofId,
    ) -> Result<SP1ProofWithPublicValues, ProofWaitError> {
        match self {
            Self::Network(provider) => provider.wait_for_proof(proof_id).await,
            Self::Mock(_) => Err(ProofWaitError::Uncertain(anyhow::anyhow!(
                "mock provider owns no SP1 proof requests"
            ))),
        }
    }

    /// Verifies a compressed range or consolidation proof locally.
    pub fn verify_range_proof(
        &self,
        keys: &ProofKeys,
        proof: &SP1ProofWithPublicValues,
    ) -> Result<()> {
        match self {
            Self::Network(provider) => provider.verify(proof, keys.range_vk.as_ref()),
            Self::Mock(_) => bail!("mock provider produces no SP1 range proofs"),
        }
    }

    /// Verifies an aggregation proof locally.
    pub fn verify_aggregation_proof(
        &self,
        keys: &ProofKeys,
        proof: &SP1ProofWithPublicValues,
    ) -> Result<()> {
        match self {
            Self::Network(provider) => provider.verify(proof, keys.agg_vk.as_ref()),
            Self::Mock(_) => bail!("mock provider produces no SP1 aggregation proofs"),
        }
    }

    /// Returns whether this provider uses the mock pipeline.
    pub const fn is_mock(&self) -> bool {
        matches!(self, Self::Mock(_))
    }
}

/// Mock proof provider: the pipeline runs the guest logic natively (witness
/// collection already computes the outputs) and submits [`MOCK_PROOF_BYTES`]
/// on-chain. Holds no SP1 client, requires no ELFs and no credentials.
#[derive(Clone, Copy, Debug, Default)]
pub struct MockProofProvider;
